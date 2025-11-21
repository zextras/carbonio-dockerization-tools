package parser

// ServiceDefinition represents a backend service from docker-compose
type ServiceDefinition struct {
	Name          string   // Service name (e.g., "carbonio-mailbox")
	DisplayName   string   // Display name from image (e.g., "carbonio-files-ce")
	EnvVar        string   // Environment variable for image (e.g., "CARBONIO_MAILBOX_IMAGE")
	DefaultImage  string   // Default image URL
	DefaultTag    string   // Extracted default tag
	DependsOn     []string // List of service dependencies
	Available     []string // Available in editions: ["ce", "advanced"]
	IsRequired    bool     // True if this is a required service (mailbox + deps)
	IsRegistrator bool     // True if this is a registrator service
	ParentService string   // For registrators: the service they register
}

// UIImageDefinition represents a frontend UI image from Dockerfile args
type UIImageDefinition struct {
	Name         string // Friendly name (e.g., "carbonio-shell-ui")
	EnvVar       string // Environment variable (e.g., "CARBONIO_SHELL_UI_IMAGE")
	DefaultImage string // Default image URL
	DefaultTag   string // Extracted default tag
	IsProxy      bool   // True if this is the proxy (carbonio-proxy-image)
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
