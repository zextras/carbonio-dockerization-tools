package parser

import (
	"fmt"
	"log"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents a simplified docker-compose.yaml structure
type ComposeFile struct {
	Services map[string]Service `yaml:"services"`
}

// Service represents a service in docker-compose
type Service struct {
	Image     string       `yaml:"image"`
	Restart   string       `yaml:"restart"`
	DependsOn interface{}  `yaml:"depends_on"` // Can be []string or map[string]Condition
	Build     *BuildConfig `yaml:"build"`
}

// BuildConfig represents build configuration
type BuildConfig struct {
	Context    string            `yaml:"context"`
	Dockerfile string            `yaml:"dockerfile"`
	Args       map[string]string `yaml:"args"`
}

// ParseComposeFile parses a docker-compose YAML file
func ParseComposeFile(data []byte, edition Edition) (map[string]*ServiceDefinition, error) {
	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	log.Printf("Parsing compose for edition %s, found %d services", edition, len(compose.Services))

	services := make(map[string]*ServiceDefinition)

	for name, svc := range compose.Services {
		// Skip if no image and no build (shouldn't happen but be safe)
		if svc.Image == "" && svc.Build == nil {
			log.Printf("Skipping service %s (no image or build)", name)
			continue
		}

		def := &ServiceDefinition{
			Name:          name,
			Available:     []string{string(edition)},
			IsRequired:    IsServiceRequired(name),
			IsRegistrator: IsRegistrator(name),
			ParentService: GetParentServiceFromMap(name), // Usa la mappa statica
		}

		// Extract ENV var and default image from ${ENV:-default} or ${ENV-default} syntax
		if strings.Contains(svc.Image, "${") {
			envVar, defaultImg := extractEnvVar(svc.Image)
			def.EnvVar = envVar
			def.DefaultImage = defaultImg
			def.DefaultTag = extractTag(defaultImg)
			def.DisplayName = extractImageName(defaultImg)
			log.Printf("Service %s: env=%s, image=%s, tag=%s, display=%s, restart=%s",
				name, envVar, defaultImg, def.DefaultTag, def.DisplayName, svc.Restart)
		} else if svc.Image != "" {
			// Direct image reference (no env var)
			def.DefaultImage = svc.Image
			def.DefaultTag = extractTag(svc.Image)
			def.DisplayName = extractImageName(svc.Image)
			log.Printf("Service %s: direct image=%s, tag=%s, display=%s, restart=%s",
				name, svc.Image, def.DefaultTag, def.DisplayName, svc.Restart)
		} else {
			// Build-only service (like carbonio-docs-editor)
			def.DefaultTag = "local"
			def.DisplayName = name
			log.Printf("Service %s: build-only, tag=local, display=%s, restart=%s",
				name, def.DisplayName, svc.Restart)
		}

		// Extract dependencies
		def.DependsOn = extractDependencies(svc.DependsOn)

		services[name] = def
	}

	log.Printf("Parsed %d services from compose", len(services))

	return services, nil
}

// extractEnvVar extracts environment variable and default from ${VAR:-default} or ${VAR-default}
func extractEnvVar(imageStr string) (envVar, defaultImage string) {
	// Remove leading ${ and trailing }
	imageStr = strings.TrimPrefix(imageStr, "${")
	imageStr = strings.TrimSuffix(imageStr, "}")

	// Split on :- or -
	var parts []string
	if strings.Contains(imageStr, ":-") {
		parts = strings.SplitN(imageStr, ":-", 2)
	} else if strings.Contains(imageStr, "-") {
		parts = strings.SplitN(imageStr, "-", 2)
	} else {
		// Just ${VAR} without default
		return imageStr, ""
	}

	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return imageStr, ""
}

// extractTag extracts tag from image URL (e.g., "registry.com/image:tag" -> "tag")
func extractTag(imageURL string) string {
	if imageURL == "" {
		return "latest"
	}

	// Find last : after last /
	lastSlash := strings.LastIndex(imageURL, "/")
	lastColon := strings.LastIndex(imageURL, ":")

	if lastColon > lastSlash && lastColon != -1 {
		tag := imageURL[lastColon+1:]
		log.Printf("Extracted tag '%s' from image '%s'", tag, imageURL)
		return tag
	}

	log.Printf("No tag found in image '%s', defaulting to 'latest'", imageURL)
	return "latest"
}

// extractImageName extracts the image name from full URL
// e.g., "registry.dev.zextras.com/dev/carbonio-files-ce:devel" -> "carbonio-files-ce"
func extractImageName(imageURL string) string {
	if imageURL == "" {
		return ""
	}

	// Remove tag first (everything after last :)
	lastColon := strings.LastIndex(imageURL, ":")
	lastSlash := strings.LastIndex(imageURL, "/")

	imageWithoutTag := imageURL
	if lastColon > lastSlash && lastColon != -1 {
		imageWithoutTag = imageURL[:lastColon]
	}

	// Get everything after last /
	parts := strings.Split(imageWithoutTag, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return imageURL
}

// extractDependencies handles both array and map syntax for depends_on
func extractDependencies(dependsOn interface{}) []string {
	if dependsOn == nil {
		return []string{}
	}

	switch v := dependsOn.(type) {
	case []interface{}:
		// Array syntax: ["service1", "service2"]
		deps := make([]string, 0, len(v))
		for _, dep := range v {
			if s, ok := dep.(string); ok {
				deps = append(deps, s)
			}
		}
		return deps

	case map[string]interface{}:
		// Map syntax: {service1: {condition: ...}, service2: ...}
		deps := make([]string, 0, len(v))
		for serviceName := range v {
			deps = append(deps, serviceName)
		}
		return deps

	default:
		return []string{}
	}
}

// ParseUIImagesFromCompose parses UI images from docker-compose build args
func ParseUIImagesFromCompose(data []byte) (map[string]*UIImageDefinition, error) {
	log.Println("Parsing UI images from compose build args...")

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	uiImages := make(map[string]*UIImageDefinition)

	// Cerca il servizio carbonio-composed-ui
	composedUI, exists := compose.Services["carbonio-composed-ui"]
	if !exists || composedUI.Build == nil || composedUI.Build.Args == nil {
		log.Println("No build args found in carbonio-composed-ui")
		return uiImages, nil
	}

	log.Printf("Found carbonio-composed-ui with %d build args", len(composedUI.Build.Args))

	// Processa ogni build arg
	for envVar, argValue := range composedUI.Build.Args {
		log.Printf("Build arg: %s = %s", envVar, argValue)

		// Estrai il valore reale dalla sintassi ${VAR:-default} se presente
		defaultImage := argValue
		if strings.Contains(argValue, "${") {
			// Ha sintassi ${VAR:-default}, estraiamo il default
			_, extracted := extractEnvVar(argValue)
			if extracted != "" {
				defaultImage = extracted
			}
		}

		// Pulisci spazi
		defaultImage = strings.TrimSpace(defaultImage)

		log.Printf("  -> Extracted default image: %s", defaultImage)

		// Process both UI images and PROXY image
		if strings.HasSuffix(envVar, "_UI_IMAGE") || envVar == "CARBONIO_PROXY_IMAGE" {
			// Extract friendly name from env var
			friendlyName := extractUIName(envVar)

			// Check if it's the proxy
			isProxy := envVar == "CARBONIO_PROXY_IMAGE"

			log.Printf("  -> UI: %s (proxy=%v)", friendlyName, isProxy)

			uiImages[friendlyName] = &UIImageDefinition{
				Name:         friendlyName,
				EnvVar:       envVar,
				DefaultImage: defaultImage,
				DefaultTag:   extractTag(defaultImage),
				IsProxy:      isProxy,
			}
		}
	}

	log.Printf("Parsed %d UI images from compose build args", len(uiImages))
	for name, ui := range uiImages {
		log.Printf("  - %s: tag=%s, proxy=%v, image=%s", name, ui.DefaultTag, ui.IsProxy, ui.DefaultImage)
	}

	return uiImages, nil
}

// extractUIName converts CARBONIO_SHELL_UI_IMAGE to carbonio-shell-ui
// or CARBONIO_PROXY_IMAGE to carbonio-proxy
func extractUIName(envVar string) string {
	// Remove _IMAGE suffix
	name := strings.TrimSuffix(envVar, "_IMAGE")
	// Convert to lowercase and replace _ with -
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	return name
}
