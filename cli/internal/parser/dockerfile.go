package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// ParseDockerfileUIArgs parses Dockerfile to extract UI image ARGs
func ParseDockerfileUIArgs(data []byte) (map[string]*UIImageDefinition, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	uiImages := make(map[string]*UIImageDefinition)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for ARG lines like: ARG CARBONIO_SHELL_UI_IMAGE=registry...
		if !strings.HasPrefix(line, "ARG ") {
			continue
		}

		// Remove "ARG " prefix
		line = strings.TrimPrefix(line, "ARG ")

		// Split on = to get var name and default value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // ARG without default, skip
		}

		envVar := strings.TrimSpace(parts[0])
		defaultImage := strings.TrimSpace(parts[1])

		// Only process UI image args (skip CARBONIO_PROXY_IMAGE)
		if !strings.HasSuffix(envVar, "_UI_IMAGE") {
			continue
		}

		// Extract friendly name from env var
		// CARBONIO_SHELL_UI_IMAGE -> carbonio-shell-ui
		friendlyName := extractUIName(envVar)

		uiImages[friendlyName] = &UIImageDefinition{
			Name:         friendlyName,
			EnvVar:       envVar,
			DefaultImage: defaultImage,
			DefaultTag:   extractTag(defaultImage),
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading Dockerfile: %w", err)
	}

	return uiImages, nil
}

// extractUIName converts CARBONIO_SHELL_UI_IMAGE to carbonio-shell-ui
func extractUIName(envVar string) string {
	// Remove _IMAGE suffix
	name := strings.TrimSuffix(envVar, "_IMAGE")
	// Convert to lowercase and replace _ with -
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	return name
}
