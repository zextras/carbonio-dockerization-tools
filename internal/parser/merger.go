package parser

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// ParseAll parses all docker-compose files and Dockerfile for UI images
func ParseAll(workDir string, edition Edition) (*ParsedConfig, error) {
	log.Printf("=== ParseAll called: workDir=%s, edition=%s ===", workDir, edition)

	config := &ParsedConfig{
		BackendServices: make(map[string]*ServiceDefinition),
		FrontendImages:  make(map[string]*UIImageDefinition),
	}

	// Parse CE compose file
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

	// Add CE services to config
	for name, svc := range ceServices {
		config.BackendServices[name] = svc
	}

	// If advanced edition, parse and merge advanced compose
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

		// Merge advanced services
		for name, svc := range advServices {
			if existing, exists := config.BackendServices[name]; exists {
				// Service exists in both - merge (advanced overrides)
				existing.Available = append(existing.Available, string(EditionAdvanced))
				if svc.EnvVar != "" {
					existing.EnvVar = svc.EnvVar
				}
				if svc.DefaultImage != "" {
					existing.DefaultImage = svc.DefaultImage
					existing.DefaultTag = svc.DefaultTag
				}
			} else {
				// New service only in advanced
				config.BackendServices[name] = svc
			}
		}
	}

	log.Printf("Total backend services: %d", len(config.BackendServices))

	// Parse UI images from docker-compose build args (not Dockerfile)
	log.Println("Parsing UI images from compose build args...")
	uiImages, err := ParseUIImagesFromCompose(ceData)
	if err != nil {
		log.Printf("Warning: failed to parse UI images from compose: %v", err)
		// Try fallback to Dockerfile
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
