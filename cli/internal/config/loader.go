package config

import (
	"fmt"
	"os"

	"github.com/zextras/carbonio-base-dockerization/cli/internal/parser"
	"gopkg.in/yaml.v3"
)

// LoadConfig loads and validates a configuration file
func LoadConfig(filePath string, parsedConfig *parser.ParsedConfig) (*UserConfig, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config UserConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate edition
	if config.Carbonio.Edition != "ce" && config.Carbonio.Edition != "advanced" {
		return nil, fmt.Errorf("invalid edition: %s (must be 'ce' or 'advanced')", config.Carbonio.Edition)
	}

	// Validate backend services exist
	for serviceName := range config.Carbonio.Backend {
		if _, exists := parsedConfig.BackendServices[serviceName]; !exists {
			return nil, fmt.Errorf("backend service '%s' not found in docker-compose files (config may be outdated)", serviceName)
		}
	}

	// Validate frontend UI images exist
	for uiName := range config.Carbonio.Frontend {
		if _, exists := parsedConfig.FrontendImages[uiName]; !exists {
			return nil, fmt.Errorf("frontend UI '%s' not found in Dockerfile (config may be outdated)", uiName)
		}
	}

	return &config, nil
}
