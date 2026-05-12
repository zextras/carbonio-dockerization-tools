// SPDX-FileCopyrightText: 2026 Zextras <https://www.zextras.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

package docker

import (
	"carbonio-dockerization-tools/internal/config"
	"carbonio-dockerization-tools/internal/parser"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
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

// BuildResult contains the docker compose commands to run in sequence.
type BuildResult struct {
	EnvVars         string
	FrontendEnvVars string            // only used by StartupCommand for the copy-pasteable command
	PullCmd         []string          // docker compose pull --ignore-pull-failures (updates remote images, ignores local-only)
	BuildCmd        []string          // docker compose build --pull (refreshes FROM base images in Dockerfiles)
	UpCmd           []string          // docker compose up (starts services using pre-built images)
	Images          map[string]string // serviceName → "image:tag" for pullable (non-buildable) services
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

	// Frontend build args are written to a compose override file instead of env
	// vars. Docker Compose v5.x has a bug where env-var-interpolated build args
	// get the KEY= prefix baked into the value when generating the bake JSON,
	// producing invalid image references like "VAR_NAME=registry/image:tag".
	overridePath, err := b.writeUIArgsOverride()
	if err != nil {
		return nil, fmt.Errorf("failed to write UI args override: %w", err)
	}
	if overridePath != "" {
		composeFiles = append(composeFiles, overridePath)
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

	// Build command: --pull refreshes base images in FROM stages (e.g. carbonio-proxy:devel).
	// No service filter: builds all services with a build: section.
	buildCmd := append(append([]string{}, composeBase...), "build", "--pull")

	// Up command: images already built by buildCmd, no need for --build
	upCmd := append(append([]string{}, composeBase...), "up")
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
			images[autoName] = imageWithTag(svc.DefaultImage, svc.DefaultTag)
		}
	}

	// Build frontend env vars string for the copy-pasteable StartupCommand only.
	var feEnvVars []string
	for uiName, ui := range b.parsedConfig.FrontendImages {
		imgConfig, exists := b.frontendImages[uiName]
		if !exists || imgConfig == nil || imgConfig.Tag == "disabled" {
			feEnvVars = append(feEnvVars, fmt.Sprintf("%s=disabled", ui.EnvVar))
		} else {
			feEnvVars = append(feEnvVars, fmt.Sprintf("%s=%s", ui.EnvVar, imageWithTag(imgConfig.Image, imgConfig.Tag)))
		}
	}

	return &BuildResult{
		EnvVars:         strings.Join(envVars, " "),
		FrontendEnvVars: strings.Join(feEnvVars, " "),
		PullCmd:         pullCmd,
		BuildCmd:        buildCmd,
		UpCmd:           upCmd,
		Images:          images,
	}, nil
}

// StartupCommand returns the full docker compose up command as a single
// copy-pasteable string with environment variables prepended.
// FrontendEnvVars are included here for the copy-pasteable command even though
// the actual execution uses the override YAML file.
func (r *BuildResult) StartupCommand() string {
	cmd := strings.Join(r.UpCmd, " ")
	allEnv := r.EnvVars
	if r.FrontendEnvVars != "" {
		if allEnv != "" {
			allEnv += " " + r.FrontendEnvVars
		} else {
			allEnv = r.FrontendEnvVars
		}
	}
	if allEnv != "" {
		return allEnv + " " + cmd
	}
	return cmd
}

const uiArgsOverrideFile = "docker-compose.ui-args.yaml"

// writeUIArgsOverride generates a compose override file that sets build args for
// carbonio-composed-ui with hardcoded values, bypassing Docker Compose env var
// interpolation entirely.
func (b *CommandBuilder) writeUIArgsOverride() (string, error) {
	if len(b.parsedConfig.FrontendImages) == 0 {
		return "", nil
	}

	// Collect env var names in sorted order for deterministic output
	type argEntry struct {
		envVar string
		value  string
	}
	var args []argEntry
	for uiName, ui := range b.parsedConfig.FrontendImages {
		imgConfig, exists := b.frontendImages[uiName]
		if !exists || imgConfig == nil || imgConfig.Tag == "disabled" {
			args = append(args, argEntry{ui.EnvVar, "disabled"})
		} else {
			args = append(args, argEntry{ui.EnvVar, imageWithTag(imgConfig.Image, imgConfig.Tag)})
		}
	}
	sort.Slice(args, func(i, j int) bool { return args[i].envVar < args[j].envVar })

	var buf strings.Builder
	buf.WriteString("services:\n")
	buf.WriteString("  carbonio-composed-ui:\n")
	buf.WriteString("    build:\n")
	buf.WriteString("      args:\n")
	for _, a := range args {
		fmt.Fprintf(&buf, "        %s: \"%s\"\n", a.envVar, a.value)
	}

	outPath := filepath.Join(b.workDir, uiArgsOverrideFile)
	if err := os.WriteFile(outPath, []byte(buf.String()), 0644); err != nil {
		return "", err
	}
	log.Printf("Wrote UI build args override: %s", outPath)
	return uiArgsOverrideFile, nil
}

// imageWithTag builds an "image:tag" reference. If image already contains a tag
// (e.g. "alpinelinux/docker-cli:latest"), the existing tag is stripped before
// appending the provided tag, preventing duplicates like "image:latest:latest".
func imageWithTag(image, tag string) string {
	lastSlash := strings.LastIndex(image, "/")
	lastColon := strings.LastIndex(image, ":")
	if lastColon > lastSlash {
		image = image[:lastColon]
	}
	return image + ":" + tag
}
