package parser

// ServiceDefinition represents a backend service from docker-compose
type ServiceDefinition struct {
	Name         string   // Service name (e.g., "carbonio-mailbox")
	EnvVar       string   // Environment variable for image (e.g., "CARBONIO_MAILBOX_IMAGE")
	DefaultImage string   // Default image URL
	DefaultTag   string   // Extracted default tag
	DependsOn    []string // List of service dependencies
	Available    []string // Available in editions: ["ce", "advanced"]
}

// UIImageDefinition represents a frontend UI image from Dockerfile args
type UIImageDefinition struct {
	Name         string // Friendly name (e.g., "carbonio-shell-ui")
	EnvVar       string // Environment variable (e.g., "CARBONIO_SHELL_UI_IMAGE")
	DefaultImage string // Default image URL
	DefaultTag   string // Extracted default tag
}

// ParsedConfig contains all parsed services and UI images
type ParsedConfig struct {
	BackendServices map[string]*ServiceDefinition // Key: service name
	FrontendImages  map[string]*UIImageDefinition // Key: UI name
}

// Edition type
type Edition string

const (
	EditionCE       Edition = "ce"
	EditionAdvanced Edition = "advanced"
)
