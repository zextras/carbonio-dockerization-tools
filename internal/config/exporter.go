package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

func ExportConfig(filePath string, config *UserConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0755); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}
func CreateUserConfig(edition string, backend map[string]*ImageConfig, frontend map[string]*ImageConfig) *UserConfig {
	return &UserConfig{
		Version: "1.0",
		Carbonio: CarbonioConfig{
			Edition:  edition,
			Backend:  backend,
			Frontend: frontend,
		},
	}
}
