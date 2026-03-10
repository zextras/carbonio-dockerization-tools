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
	natIP           string
	backendServices map[string]*config.ImageConfig
	frontendImages  map[string]*config.ImageConfig
	parsedConfig    *parser.ParsedConfig
}

// BuildResult contains the two docker compose commands to run in sequence.
type BuildResult struct {
	EnvVars string
	PullCmd []string          // docker compose pull --ignore-pull-failures (updates remote images, ignores local-only)
	UpCmd   []string          // docker compose up --build (uses whatever is locally available)
	Images  map[string]string // serviceName → "image:tag" for pullable (non-buildable) services
}

func NewCommandBuilder(workDir string, edition parser.Edition, parsedConfig *parser.ParsedConfig, natIP string) *CommandBuilder {
	return &CommandBuilder{
		workDir:         workDir,
		edition:         edition,
		natIP:           natIP,
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
func (b *CommandBuilder) Build() (*BuildResult, error) {
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
	if b.natIP != "" {
		envVars = append(envVars, fmt.Sprintf("NAT_IP=%s", b.natIP))
	}

	for uiName, ui := range b.parsedConfig.FrontendImages {
		imgConfig, exists := b.frontendImages[uiName]
		if !exists || imgConfig == nil || imgConfig.Tag == "disabled" {
			envVars = append(envVars, fmt.Sprintf("%s=disabled", ui.EnvVar))
		} else {
			envVars = append(envVars, fmt.Sprintf("%s=%s:%s", ui.EnvVar, imgConfig.Image, imgConfig.Tag))
		}
	}

	// Build compose base args (shared by pull and up)
	var composeBase []string
	composeBase = append(composeBase, "docker", "compose")
	for _, file := range composeFiles {
		composeBase = append(composeBase, "-f", file)
	}

	// Pull command: update remote images, silently skip local/build-only ones
	pullCmd := append(append([]string{}, composeBase...), "pull", "--ignore-buildable", "--ignore-pull-failures")
	pullCmd = append(pullCmd, selectedServices...)

	// Up command: start with whatever is locally available (pull already done above)
	upCmd := append(append([]string{}, composeBase...), "up", "--build")
	upCmd = append(upCmd, selectedServices...)

	// Build image map for individual pulls (only remote/pullable images)
	images := make(map[string]string)
	for serviceName, imgConfig := range b.backendServices {
		svc := b.parsedConfig.BackendServices[serviceName]
		if svc != nil && svc.DefaultImage != "" && imgConfig != nil && imgConfig.Image != "" {
			images[serviceName] = fmt.Sprintf("%s:%s", imgConfig.Image, imgConfig.Tag)
		}
	}
	for _, autoName := range parser.GlobalDockerConfig.AutoIncludedServices {
		if _, alreadySet := images[autoName]; alreadySet {
			continue
		}
		svc := b.parsedConfig.BackendServices[autoName]
		if svc != nil && svc.DefaultImage != "" && svc.DefaultTag != "" {
			images[autoName] = fmt.Sprintf("%s:%s", svc.DefaultImage, svc.DefaultTag)
		}
	}

	return &BuildResult{
		EnvVars: strings.Join(envVars, " "),
		PullCmd: pullCmd,
		UpCmd:   upCmd,
		Images:  images,
	}, nil
}

// StartupCommand returns the full docker compose up command as a single
// copy-pasteable string with environment variables prepended.
func (r *BuildResult) StartupCommand() string {
	cmd := strings.Join(r.UpCmd, " ")
	if r.EnvVars != "" {
		return r.EnvVars + " " + cmd
	}
	return cmd
}
