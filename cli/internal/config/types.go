package config

// UserConfig represents the YAML configuration file structure
type UserConfig struct {
	Version  string         `yaml:"version"`
	Carbonio CarbonioConfig `yaml:"carbonio"`
}

// CarbonioConfig contains Carbonio-specific configuration
type CarbonioConfig struct {
	Edition  string            `yaml:"edition"`  // "ce" or "advanced"
	Backend  map[string]string `yaml:"backend"`  // service_name: image_tag
	Frontend map[string]string `yaml:"frontend"` // ui_name: image_tag
}
