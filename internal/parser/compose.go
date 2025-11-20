package parser

import (
	"fmt"
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

	services := make(map[string]*ServiceDefinition)

	for name, svc := range compose.Services {
		// Skip services with restart: "no" (provisioner, etc.)
		if svc.Restart == "no" {
			continue
		}

		// Skip if no image and no build (shouldn't happen but be safe)
		if svc.Image == "" && svc.Build == nil {
			continue
		}

		def := &ServiceDefinition{
			Name:       name,
			Available:  []string{string(edition)},
			IsRequired: IsServiceRequired(name), // Marca se obbligatorio
		}

		// Extract ENV var and default image from ${ENV:-default} or ${ENV-default} syntax
		if strings.Contains(svc.Image, "${") {
			envVar, defaultImg := extractEnvVar(svc.Image)
			def.EnvVar = envVar
			def.DefaultImage = defaultImg
			def.DefaultTag = extractTag(defaultImg)
		} else if svc.Image != "" {
			// Direct image reference (no env var)
			def.DefaultImage = svc.Image
			def.DefaultTag = extractTag(svc.Image)
		}

		// Extract dependencies
		def.DependsOn = extractDependencies(svc.DependsOn)

		services[name] = def
	}

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
		return imageURL[lastColon+1:]
	}

	return "latest"
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
