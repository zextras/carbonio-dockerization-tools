package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ExportConfig exports configuration to a YAML file
func ExportConfig(filePath string, config *UserConfig) error {
	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0755); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CreateUserConfig creates a UserConfig from selections
func CreateUserConfig(edition string, backend map[string]string, frontend map[string]string) *UserConfig {
	return &UserConfig{
		Version: "1.0",
		Carbonio: CarbonioConfig{
			Edition:  edition,
			Backend:  backend,
			Frontend: frontend,
		},
	}
}
