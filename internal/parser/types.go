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

// RequiredServices lista dei servizi backend obbligatori
// Questi sono sempre necessari per far funzionare Carbonio
var RequiredServices = []string{
	"carbonio-mailbox",
	"carbonio-openldap",
	"carbonio-postfix",
	"carbonio-mariadb",
	"consul",
	"consul-register",
	"carbonio-composed-ui", // Il proxy è sempre necessario
	"traefik",
	"memcached",
}

// RegistratorToService mappa ogni registrator al suo servizio parent
var RegistratorToService = map[string]string{
	// CE registrators
	"mailbox-registrator":            "carbonio-mailbox",
	"user-management-registrator":    "carbonio-user-management",
	"catalog-registrator":            "carbonio-catalog",
	"storages-registrator":           "carbonio-storages",
	"docs-connector-registrator":     "carbonio-docs-connector",
	"docs-editor-registrator":        "carbonio-docs-editor",
	"preview-registrator":            "carbonio-preview",
	"files-registrator":              "carbonio-files",
	"tasks-registrator":              "carbonio-tasks",
	"message-dispatcher-registrator": "carbonio-message-dispatcher",
	"wsc-registrator":                "carbonio-ws-collaboration",

	// Advanced registrators
	"advanced-registrator":     "carbonio-mailbox",
	"address-book-registrator": "carbonio-mailbox",
	"auth-registrator":         "carbonio-mailbox",
}

// IsServiceRequired controlla se un servizio è obbligatorio
func IsServiceRequired(serviceName string) bool {
	for _, required := range RequiredServices {
		if serviceName == required {
			return true
		}
	}
	return false
}

// IsRegistrator controlla se un servizio è un registrator
func IsRegistrator(serviceName string) bool {
	_, exists := RegistratorToService[serviceName]
	return exists
}

// GetParentServiceFromMap restituisce il servizio parent di un registrator
func GetParentServiceFromMap(registratorName string) string {
	return RegistratorToService[registratorName]
}

// GetRegistratorsForService restituisce tutti i registrator per un dato servizio
func GetRegistratorsForService(serviceName string) []string {
	var registrators []string
	for regName, parentService := range RegistratorToService {
		if parentService == serviceName {
			registrators = append(registrators, regName)
		}
	}
	return registrators
}
