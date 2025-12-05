package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"strings"
)

func ParseDockerfileUIArgs(data []byte) (map[string]*UIImageDefinition, error) {
	log.Printf("Parsing Dockerfile (%d bytes)", len(data))
	scanner := bufio.NewScanner(bytes.NewReader(data))
	uiImages := make(map[string]*UIImageDefinition)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "ARG ") {
			continue
		}
		line = strings.TrimPrefix(line, "ARG ")
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			log.Printf("Line %d: ARG without default value, skipping: %s", lineNum, line)
			continue
		}
		envVar := strings.TrimSpace(parts[0])
		defaultImage := strings.TrimSpace(parts[1])
		if strings.Contains(defaultImage, "${") {
			defaultImage = resolveNestedEnvVars(defaultImage)
		}
		log.Printf("Line %d: Found ARG %s=%s", lineNum, envVar, defaultImage)
		if strings.HasSuffix(envVar, "_UI_IMAGE") || envVar == "CARBONIO_PROXY_IMAGE" {
			friendlyName := extractUIName(envVar)
			isProxy := envVar == "CARBONIO_PROXY_IMAGE"
			log.Printf("  -> UI: %s (proxy=%v)", friendlyName, isProxy)
			uiImages[friendlyName] = &UIImageDefinition{
				Name:         friendlyName,
				EnvVar:       envVar,
				DefaultImage: defaultImage,
				DefaultTag:   extractTag(defaultImage),
				IsProxy:      isProxy,
			}
		} else {
			log.Printf("  -> Skipped (not a UI image)")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading Dockerfile: %w", err)
	}
	log.Printf("Parsed %d UI images from Dockerfile", len(uiImages))
	for name, ui := range uiImages {
		log.Printf("  - %s: tag=%s, proxy=%v", name, ui.DefaultTag, ui.IsProxy)
	}
	return uiImages, nil
}
