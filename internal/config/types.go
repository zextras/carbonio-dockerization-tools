package config

type UserConfig struct {
	Version  string         `yaml:"version"`
	Carbonio CarbonioConfig `yaml:"carbonio"`
}
type CarbonioConfig struct {
	Edition  string            `yaml:"edition"`
	Backend  map[string]string `yaml:"backend"`
	Frontend map[string]string `yaml:"frontend"`
}
