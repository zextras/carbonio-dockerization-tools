// SPDX-FileCopyrightText: 2026 Zextras <https://www.zextras.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"strings"
)

type ComposeFile struct {
	Services map[string]Service `yaml:"services"`
}

type Service struct {
	Image     string       `yaml:"image"`
	Restart   string       `yaml:"restart"`
	DependsOn interface{}  `yaml:"depends_on"`
	Build     *BuildConfig `yaml:"build"`
}

type BuildConfig struct {
	Context    string            `yaml:"context"`
	Dockerfile string            `yaml:"dockerfile"`
	Args       map[string]string `yaml:"args"`
}

func ParseComposeFile(data []byte, edition Edition) (map[string]*ServiceDefinition, error) {
	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}
	log.Printf("Parsing compose for edition %s, found %d services", edition, len(compose.Services))
	services := make(map[string]*ServiceDefinition)
	for name, svc := range compose.Services {
		if svc.Image == "" && svc.Build == nil {
			log.Printf("Skipping service %s (no image or build)", name)
			continue
		}
		def := &ServiceDefinition{
			Name:       name,
			Available:  []string{string(edition)},
			IsRequired: IsServiceRequired(name),
		}
		if strings.Contains(svc.Image, "${") {
			envVar, defaultImg := extractEnvVar(svc.Image)
			def.EnvVar = envVar
			def.DefaultImage = defaultImg
			def.DefaultTag = extractTag(defaultImg)
			def.DisplayName = extractImageName(defaultImg)
			log.Printf("Service %s: env=%s, image=%s, tag=%s, display=%s, restart=%s",
				name, envVar, defaultImg, def.DefaultTag, def.DisplayName, svc.Restart)
		} else if svc.Image != "" {
			def.DefaultImage = svc.Image
			def.DefaultTag = extractTag(svc.Image)
			def.DisplayName = extractImageName(svc.Image)
			log.Printf("Service %s: direct image=%s, tag=%s, display=%s, restart=%s",
				name, svc.Image, def.DefaultTag, def.DisplayName, svc.Restart)
		} else {
			def.DefaultTag = "local"
			def.DisplayName = name
			log.Printf("Service %s: build-only, tag=local, display=%s, restart=%s",
				name, def.DisplayName, svc.Restart)
		}
		def.DependsOn = extractDependencies(svc.DependsOn)
		services[name] = def
	}
	log.Printf("Parsed %d services from compose", len(services))
	return services, nil
}

func extractEnvVar(imageStr string) (envVar, defaultImage string) {
	if !strings.HasPrefix(imageStr, "${") {
		return imageStr, ""
	}
	content := strings.TrimPrefix(imageStr, "${")
	if strings.HasSuffix(content, "}") {
		content = content[:len(content)-1]
	}
	if idx := strings.Index(content, ":-"); idx != -1 {
		envVar = content[:idx]
		defaultImage = content[idx+2:]
		defaultImage = resolveNestedEnvVars(defaultImage)
		return envVar, defaultImage
	}
	return content, ""
}

func resolveNestedEnvVars(s string) string {
	result := s
	for {
		start := strings.Index(result, "${")
		if start == -1 {
			break
		}
		depth := 1
		end := -1
		for i := start + 2; i < len(result); i++ {
			if i+1 < len(result) && result[i] == '$' && result[i+1] == '{' {
				depth++
				i++
			} else if result[i] == '}' {
				depth--
				if depth == 0 {
					end = i
					break
				}
			}
		}
		var defaultVal string
		if end != -1 {
			varContent := result[start+2 : end]
			if idx := strings.Index(varContent, ":-"); idx != -1 {
				defaultVal = varContent[idx+2:]
			}
			result = result[:start] + defaultVal + result[end+1:]
		} else {
			varContent := result[start+2:]
			if idx := strings.Index(varContent, ":-"); idx != -1 {
				defaultVal = varContent[idx+2:]
			}
			result = result[:start] + defaultVal
			break
		}
	}
	return result
}

func extractTag(imageURL string) string {
	if imageURL == "" {
		return "latest"
	}
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

func extractImageName(imageURL string) string {
	if imageURL == "" {
		return ""
	}
	lastColon := strings.LastIndex(imageURL, ":")
	lastSlash := strings.LastIndex(imageURL, "/")
	imageWithoutTag := imageURL
	if lastColon > lastSlash && lastColon != -1 {
		imageWithoutTag = imageURL[:lastColon]
	}
	parts := strings.Split(imageWithoutTag, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return imageURL
}

func extractDependencies(dependsOn interface{}) []string {
	if dependsOn == nil {
		return []string{}
	}
	switch v := dependsOn.(type) {
	case []interface{}:
		deps := make([]string, 0, len(v))
		for _, dep := range v {
			if s, ok := dep.(string); ok {
				deps = append(deps, s)
			}
		}
		return deps
	case map[string]interface{}:
		deps := make([]string, 0, len(v))
		for serviceName := range v {
			deps = append(deps, serviceName)
		}
		return deps
	default:
		return []string{}
	}
}

func ParseUIImagesFromCompose(data []byte) (map[string]*UIImageDefinition, error) {
	log.Println("Parsing UI images from compose build args...")
	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}
	uiImages := make(map[string]*UIImageDefinition)
	composedUI, exists := compose.Services["carbonio-composed-ui"]
	if !exists || composedUI.Build == nil || composedUI.Build.Args == nil {
		log.Println("No build args found in carbonio-composed-ui")
		return uiImages, nil
	}
	log.Printf("Found carbonio-composed-ui with %d build args", len(composedUI.Build.Args))
	for envVar, argValue := range composedUI.Build.Args {
		log.Printf("Build arg: %s = %s", envVar, argValue)
		defaultImage := argValue
		if strings.Contains(argValue, "${") {
			_, extracted := extractEnvVar(argValue)
			if extracted != "" {
				defaultImage = extracted
			}
		}
		defaultImage = strings.TrimSpace(defaultImage)
		log.Printf("  -> Extracted default image: %s", defaultImage)
		if strings.HasSuffix(envVar, "_UI_IMAGE") || envVar == "CARBONIO_PROXY_IMAGE" {
			friendlyName := extractUIName(envVar)
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

func extractUIName(envVar string) string {
	name := strings.TrimSuffix(envVar, "_IMAGE")
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	return name
}
