package config

type UserConfig struct {
	Version  string         `yaml:"version"`
	Carbonio CarbonioConfig `yaml:"carbonio"`
}

type ImageConfig struct {
	Image string `yaml:"image"`
	Tag   string `yaml:"tag"`
}

type CarbonioConfig struct {
	Edition  string                  `yaml:"edition"`
	Backend  map[string]*ImageConfig `yaml:"backend"`
	Frontend map[string]*ImageConfig `yaml:"frontend"`
}
