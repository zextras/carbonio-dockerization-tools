package config

import (
	"carbonio-docker-cli/internal/parser"
	"fmt"
	"os"

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

	// VALIDAZIONE COMPLETA: Il YAML deve contenere ESATTAMENTE gli stessi servizi del compose

	// 1. Controlla che ogni servizio nel YAML esista nel compose
	for serviceName := range config.Carbonio.Backend {
		if _, exists := parsedConfig.BackendServices[serviceName]; !exists {
			return nil, fmt.Errorf("backend service '%s' in config file not found in docker-compose (config may be outdated)", serviceName)
		}
	}

	// 2. Controlla che ogni UI nel YAML esista nel Dockerfile
	for uiName, tag := range config.Carbonio.Frontend {
		if _, exists := parsedConfig.FrontendImages[uiName]; !exists {
			return nil, fmt.Errorf("frontend UI '%s' in config file not found in Dockerfile (config may be outdated)", uiName)
		}

		// Valida che il tag sia valido (non vuoto, tranne "disabled")
		if tag == "" {
			return nil, fmt.Errorf("frontend UI '%s' has empty tag", uiName)
		}
	}

	// 3. Controlla che il compose non abbia servizi non presenti nel YAML
	// Questo serve per assicurarci che il YAML sia completo
	for serviceName := range parsedConfig.BackendServices {
		if _, exists := config.Carbonio.Backend[serviceName]; !exists {
			return nil, fmt.Errorf("backend service '%s' found in docker-compose but missing from config file (config is incomplete)", serviceName)
		}
	}

	// 4. Controlla che il Dockerfile non abbia UI non presenti nel YAML
	for uiName := range parsedConfig.FrontendImages {
		if _, exists := config.Carbonio.Frontend[uiName]; !exists {
			return nil, fmt.Errorf("frontend UI '%s' found in Dockerfile but missing from config file (config is incomplete)", uiName)
		}
	}

	// 5. Verifica che i servizi required siano presenti e non disabilitati
	for _, serviceName := range parser.RequiredServices {
		// Controlla se il servizio esiste nel parsed config (potrebbe non essere in tutti gli edition)
		if _, exists := parsedConfig.BackendServices[serviceName]; exists {
			if _, configured := config.Carbonio.Backend[serviceName]; !configured {
				return nil, fmt.Errorf("required service '%s' is missing from config file", serviceName)
			}
		}
	}

	// 6. Verifica che il proxy sia presente e non disabilitato
	for uiName, ui := range parsedConfig.FrontendImages {
		if ui.IsProxy {
			if tag, exists := config.Carbonio.Frontend[uiName]; !exists {
				return nil, fmt.Errorf("proxy UI '%s' is missing from config file", uiName)
			} else if tag == "disabled" {
				return nil, fmt.Errorf("proxy UI '%s' cannot be disabled", uiName)
			}
		}
	}

	return &config, nil
}
