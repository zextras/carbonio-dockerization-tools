package docker

import (
	"carbonio-dockerization-tools/internal/config"
	"carbonio-dockerization-tools/internal/parser"
	"fmt"
	"strings"
)

type CommandBuilder struct {
	workDir         string
	edition         parser.Edition
	backendServices map[string]*config.ImageConfig
	frontendImages  map[string]*config.ImageConfig
	parsedConfig    *parser.ParsedConfig
}

func NewCommandBuilder(workDir string, edition parser.Edition, parsedConfig *parser.ParsedConfig) *CommandBuilder {
	return &CommandBuilder{
		workDir:         workDir,
		edition:         edition,
		backendServices: make(map[string]*config.ImageConfig),
		frontendImages:  make(map[string]*config.ImageConfig),
		parsedConfig:    parsedConfig,
	}
}
func (b *CommandBuilder) SetBackendService(serviceName string, imgConfig *config.ImageConfig) {
	b.backendServices[serviceName] = imgConfig
}
func (b *CommandBuilder) SetFrontendImage(uiName string, imgConfig *config.ImageConfig) {
	b.frontendImages[uiName] = imgConfig
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
		if imgConfig, selected := b.backendServices[serviceName]; selected {
			if svc.EnvVar != "" {
				envVars = append(envVars, fmt.Sprintf("%s=%s:%s", svc.EnvVar, imgConfig.Image, imgConfig.Tag))
			}
			selectedServices = append(selectedServices, serviceName)
		}
	}
	for _, autoIncludedName := range parser.GlobalDockerConfig.AutoIncludedServices {
		if _, exists := b.parsedConfig.BackendServices[autoIncludedName]; exists {
			selectedServices = append(selectedServices, autoIncludedName)
		}
	}
	for uiName, ui := range b.parsedConfig.FrontendImages {
		imgConfig, exists := b.frontendImages[uiName]
		if !exists || imgConfig == nil || imgConfig.Tag == "disabled" {
			envVars = append(envVars, fmt.Sprintf("%s=disabled", ui.EnvVar))
		} else {
			envVars = append(envVars, fmt.Sprintf("%s=%s:%s", ui.EnvVar, imgConfig.Image, imgConfig.Tag))
		}
	}
	cmdParts := []string{"docker", "compose"}
	for _, file := range composeFiles {
		cmdParts = append(cmdParts, "-f", file)
	}
	cmdParts = append(cmdParts, "up", "--build", "--pull", "missing")
	cmdParts = append(cmdParts, selectedServices...)
	return strings.Join(envVars, " "), cmdParts, nil
}
