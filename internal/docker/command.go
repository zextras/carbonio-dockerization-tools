package docker

import (
	"carbonio-docker-cli/internal/parser"
	"fmt"
	"strings"
)

// CommandBuilder builds docker compose commands
type CommandBuilder struct {
	workDir         string
	edition         parser.Edition
	backendServices map[string]string // service_name: tag
	frontendImages  map[string]string // ui_name: tag (or "disabled")
	parsedConfig    *parser.ParsedConfig
}

// NewCommandBuilder creates a new command builder
func NewCommandBuilder(workDir string, edition parser.Edition, parsedConfig *parser.ParsedConfig) *CommandBuilder {
	return &CommandBuilder{
		workDir:         workDir,
		edition:         edition,
		backendServices: make(map[string]string),
		frontendImages:  make(map[string]string),
		parsedConfig:    parsedConfig,
	}
}

// SetBackendService sets a backend service with its tag
func (b *CommandBuilder) SetBackendService(serviceName, tag string) {
	b.backendServices[serviceName] = tag
}

// SetFrontendImage sets a frontend UI image with its tag or "disabled"
func (b *CommandBuilder) SetFrontendImage(uiName, tag string) {
	b.frontendImages[uiName] = tag
}

// Build generates the docker compose command
func (b *CommandBuilder) Build() (string, []string, error) {
	var envVars []string
	var composeFiles []string
	var selectedServices []string

	// Always include base compose file
	composeFiles = append(composeFiles, "docker-compose.yaml")

	// Add advanced compose if needed
	if b.edition == parser.EditionAdvanced {
		composeFiles = append(composeFiles, "docker-compose-advanced.yaml")
	}

	// Build environment variables for backend services
	for serviceName, svc := range b.parsedConfig.BackendServices {
		if tag, selected := b.backendServices[serviceName]; selected {
			// Service is selected - set its image tag
			if svc.EnvVar != "" {
				registry := extractRegistry(svc.DefaultImage)
				envVars = append(envVars, fmt.Sprintf("%s=%s:%s", svc.EnvVar, registry, tag))
			}
			selectedServices = append(selectedServices, serviceName)
		}
		// If not selected, simply don't include it
	}

	// Build environment variables for frontend UI images
	for uiName, ui := range b.parsedConfig.FrontendImages {
		tag, exists := b.frontendImages[uiName]

		if !exists || tag == "disabled" {
			// UI not selected or explicitly disabled - set to "disabled"
			envVars = append(envVars, fmt.Sprintf("%s=disabled", ui.EnvVar))
		} else {
			// UI is selected - set its image tag
			registry := extractRegistry(ui.DefaultImage)
			envVars = append(envVars, fmt.Sprintf("%s=%s:%s", ui.EnvVar, registry, tag))
		}
	}

	// Always include carbonio-composed-ui in selected services (it provides proxy)
	if !contains(selectedServices, "carbonio-composed-ui") {
		selectedServices = append(selectedServices, "carbonio-composed-ui")
	}

	// Build command
	cmdParts := []string{"docker", "compose"}

	// Add compose files
	for _, file := range composeFiles {
		cmdParts = append(cmdParts, "-f", file)
	}

	// Add up command with --build for composed-ui
	cmdParts = append(cmdParts, "up", "--build")

	// Add selected services
	cmdParts = append(cmdParts, selectedServices...)

	return strings.Join(envVars, " "), cmdParts, nil
}

// extractRegistry extracts registry+image path without tag
// e.g., "registry.dev.zextras.com/dev/carbonio-mailbox:latest" -> "registry.dev.zextras.com/dev/carbonio-mailbox"
func extractRegistry(imageURL string) string {
	// Find last :
	lastColon := strings.LastIndex(imageURL, ":")
	lastSlash := strings.LastIndex(imageURL, "/")

	if lastColon > lastSlash && lastColon != -1 {
		return imageURL[:lastColon]
	}

	return imageURL
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
