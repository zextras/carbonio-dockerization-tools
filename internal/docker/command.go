package docker

import (
	"carbonio-docker-cli/internal/parser"
	"fmt"
	"strings"
)

type CommandBuilder struct {
	workDir         string
	edition         parser.Edition
	backendServices map[string]string
	frontendImages  map[string]string
	parsedConfig    *parser.ParsedConfig
}

func NewCommandBuilder(workDir string, edition parser.Edition, parsedConfig *parser.ParsedConfig) *CommandBuilder {
	return &CommandBuilder{
		workDir:         workDir,
		edition:         edition,
		backendServices: make(map[string]string),
		frontendImages:  make(map[string]string),
		parsedConfig:    parsedConfig,
	}
}
func (b *CommandBuilder) SetBackendService(serviceName, tag string) {
	b.backendServices[serviceName] = tag
}
func (b *CommandBuilder) SetFrontendImage(uiName, tag string) {
	b.frontendImages[uiName] = tag
}
func (b *CommandBuilder) Build() (string, []string, error) {
	var envVars []string
	var composeFiles []string
	var selectedServices []string
	composeFiles = append(composeFiles, "docker-compose.yaml")
	if b.edition == parser.EditionAdvanced {
		composeFiles = append(composeFiles, "docker-compose-advanced.yaml")
	}
	for serviceName, svc := range b.parsedConfig.BackendServices {
		if tag, selected := b.backendServices[serviceName]; selected {
			if svc.EnvVar != "" {
				registry := extractRegistry(svc.DefaultImage)
				envVars = append(envVars, fmt.Sprintf("%s=%s:%s", svc.EnvVar, registry, tag))
			}
			selectedServices = append(selectedServices, serviceName)
		}
	}
	for uiName, ui := range b.parsedConfig.FrontendImages {
		tag, exists := b.frontendImages[uiName]
		if !exists || tag == "disabled" {
			envVars = append(envVars, fmt.Sprintf("%s=disabled", ui.EnvVar))
		} else {
			registry := extractRegistry(ui.DefaultImage)
			envVars = append(envVars, fmt.Sprintf("%s=%s:%s", ui.EnvVar, registry, tag))
		}
	}
	cmdParts := []string{"docker", "compose"}
	for _, file := range composeFiles {
		cmdParts = append(cmdParts, "-f", file)
	}
	cmdParts = append(cmdParts, "up", "--build")
	cmdParts = append(cmdParts, selectedServices...)
	return strings.Join(envVars, " "), cmdParts, nil
}
func extractRegistry(imageURL string) string {
	lastColon := strings.LastIndex(imageURL, ":")
	lastSlash := strings.LastIndex(imageURL, "/")
	if lastColon > lastSlash && lastColon != -1 {
		return imageURL[:lastColon]
	}
	return imageURL
}
