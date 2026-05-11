// SPDX-FileCopyrightText: 2026 Zextras <https://www.zextras.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func ParseAll(workDir string, edition Edition) (*ParsedConfig, error) {
	log.Printf("=== ParseAll called: workDir=%s, edition=%s ===", workDir, edition)
	config := &ParsedConfig{
		BackendServices: make(map[string]*ServiceDefinition),
		FrontendImages:  make(map[string]*UIImageDefinition),
	}
	ceComposePath := filepath.Join(workDir, "docker-compose.yaml")
	log.Printf("Reading CE compose: %s", ceComposePath)
	ceData, err := os.ReadFile(ceComposePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read docker-compose.yaml: %w", err)
	}
	log.Printf("CE compose size: %d bytes", len(ceData))
	ceServices, err := ParseComposeFile(ceData, EditionCE)
	if err != nil {
		return nil, fmt.Errorf("failed to parse docker-compose.yaml: %w", err)
	}
	for name, svc := range ceServices {
		config.BackendServices[name] = svc
	}
	if edition == EditionAdvanced {
		log.Println("Parsing advanced compose...")
		advComposePath := filepath.Join(workDir, "docker-compose-advanced.yaml")
		advData, err := os.ReadFile(advComposePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read docker-compose-advanced.yaml: %w", err)
		}
		advServices, err := ParseComposeFile(advData, EditionAdvanced)
		if err != nil {
			return nil, fmt.Errorf("failed to parse docker-compose-advanced.yaml: %w", err)
		}
		for name, svc := range advServices {
			if existing, exists := config.BackendServices[name]; exists {
				existing.Available = append(existing.Available, string(EditionAdvanced))
				if svc.EnvVar != "" {
					existing.EnvVar = svc.EnvVar
				}
				if svc.DefaultImage != "" {
					existing.DefaultImage = svc.DefaultImage
					existing.DefaultTag = svc.DefaultTag
				}
				if len(svc.DependsOn) > 0 {
					existing.DependsOn = svc.DependsOn
				}
			} else {
				config.BackendServices[name] = svc
			}
		}
	}
	log.Printf("Total backend services: %d", len(config.BackendServices))
	log.Println("Parsing UI images from compose build args...")
	uiImages, err := ParseUIImagesFromCompose(ceData)
	if err != nil {
		log.Printf("Warning: failed to parse UI images from compose: %v", err)
		dockerfilePath := filepath.Join(workDir, "composed-ui", "Dockerfile")
		log.Printf("Trying fallback: Reading Dockerfile: %s", dockerfilePath)
		dockerfileData, err := os.ReadFile(dockerfilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read composed-ui/Dockerfile: %w", err)
		}
		log.Printf("Dockerfile size: %d bytes", len(dockerfileData))
		uiImages, err = ParseDockerfileUIArgs(dockerfileData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UI images: %w", err)
		}
	}
	config.FrontendImages = uiImages
	log.Printf("=== Parse complete: %d backend, %d frontend ===",
		len(config.BackendServices),
		len(config.FrontendImages))
	return config, nil
}
