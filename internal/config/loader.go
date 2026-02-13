package config

import (
	"carbonio-dockerization-tools/internal/parser"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

func LoadConfigEdition(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}
	var config UserConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}
	if config.Carbonio.Edition != "ce" && config.Carbonio.Edition != "advanced" {
		return "", fmt.Errorf("invalid edition: %s (must be 'ce' or 'advanced')", config.Carbonio.Edition)
	}
	return config.Carbonio.Edition, nil
}

func LoadConfig(filePath string, parsedConfig *parser.ParsedConfig) (*UserConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var config UserConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	if config.Carbonio.Edition != "ce" && config.Carbonio.Edition != "advanced" {
		return nil, fmt.Errorf("invalid edition: %s (must be 'ce' or 'advanced')", config.Carbonio.Edition)
	}
	for serviceName, imgConfig := range config.Carbonio.Backend {
		if _, exists := parsedConfig.BackendServices[serviceName]; !exists {
			return nil, fmt.Errorf("backend service '%s' in config file not found in docker-compose (config may be outdated)", serviceName)
		}
		if imgConfig == nil || imgConfig.Image == "" {
			return nil, fmt.Errorf("backend service '%s' has empty image", serviceName)
		}
		if imgConfig.Tag == "" {
			return nil, fmt.Errorf("backend service '%s' has empty tag", serviceName)
		}
	}
	for uiName, imgConfig := range config.Carbonio.Frontend {
		if _, exists := parsedConfig.FrontendImages[uiName]; !exists {
			return nil, fmt.Errorf("frontend UI '%s' in config file not found in Dockerfile (config may be outdated)", uiName)
		}
		if imgConfig == nil || imgConfig.Image == "" {
			return nil, fmt.Errorf("frontend UI '%s' has empty image", uiName)
		}
		if imgConfig.Tag == "" {
			return nil, fmt.Errorf("frontend UI '%s' has empty tag", uiName)
		}
	}
	for serviceName, svc := range parsedConfig.BackendServices {
		if svc.IsRequired {
			if _, exists := config.Carbonio.Backend[serviceName]; !exists {
				return nil, fmt.Errorf("required service '%s' is missing from config file", serviceName)
			}
		}
	}
	for uiName, ui := range parsedConfig.FrontendImages {
		if ui.IsProxy {
			imgConfig, exists := config.Carbonio.Frontend[uiName]
			if !exists {
				return nil, fmt.Errorf("proxy UI '%s' is missing from config file", uiName)
			}
			if imgConfig != nil && imgConfig.Tag == "disabled" {
				return nil, fmt.Errorf("proxy UI '%s' cannot be disabled", uiName)
			}
		}
	}
	return &config, nil
}
