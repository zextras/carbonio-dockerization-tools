package parser

type DockerConfig struct {
	HiddenServices          []string
	RequiredServices        []string
	AutoIncludedServices    []string
	RegistratorToService    map[string]string
	LockedTagValues         []string
	ForcedLockedTagServices []string
	ForcedLockedTagUIs      []string
}

func DefaultDockerConfig() *DockerConfig {
	return &DockerConfig{
		HiddenServices: []string{
			"mailbox-registrator",
			"user-management-registrator",
			"catalog-registrator",
			"storages-registrator",
			"docs-connector-registrator",
			"docs-editor-registrator",
			"preview-registrator",
			"files-registrator",
			"tasks-registrator",
			"message-dispatcher-registrator",
			"wsc-registrator",
			"advanced-registrator",
			"address-book-registrator",
			"auth-registrator",
			"consul-register",
			"carbonio-provisioner",
		},
		RequiredServices: []string{
			"carbonio-mailbox",
			"carbonio-openldap",
			"carbonio-postfix",
			"carbonio-mariadb",
			"carbonio-catalog",
			"carbonio-composed-ui",
			"consul",
			"traefik",
			"memcached",
		},
		AutoIncludedServices: []string{
			"consul-register",
			"carbonio-provisioner",
		},
		RegistratorToService: map[string]string{
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
			"advanced-registrator":           "carbonio-mailbox",
			"address-book-registrator":       "carbonio-mailbox",
			"auth-registrator":               "carbonio-mailbox",
			"consul-register":                "consul",
		},
		LockedTagValues: []string{
			"local",
		},
		ForcedLockedTagServices: []string{
			"carbonio-composed-ui",
		},
		ForcedLockedTagUIs: []string{},
	}
}

var GlobalDockerConfig = DefaultDockerConfig()

func (dc *DockerConfig) IsServiceHidden(serviceName string) bool {
	for _, hidden := range dc.HiddenServices {
		if serviceName == hidden {
			return true
		}
	}
	return false
}
func (dc *DockerConfig) IsServiceRequired(serviceName string) bool {
	for _, required := range dc.RequiredServices {
		if serviceName == required {
			return true
		}
	}
	return false
}
func (dc *DockerConfig) IsServiceAutoIncluded(serviceName string) bool {
	for _, autoIncluded := range dc.AutoIncludedServices {
		if serviceName == autoIncluded {
			return true
		}
	}
	return false
}
func (dc *DockerConfig) IsRegistrator(serviceName string) bool {
	_, exists := dc.RegistratorToService[serviceName]
	return exists
}
func (dc *DockerConfig) GetParentService(registratorName string) string {
	return dc.RegistratorToService[registratorName]
}
func (dc *DockerConfig) GetRegistratorsForService(serviceName string) []string {
	var registrators []string
	for regName, parentService := range dc.RegistratorToService {
		if parentService == serviceName {
			registrators = append(registrators, regName)
		}
	}
	return registrators
}
func (dc *DockerConfig) IsTagLocked(serviceName, tag string, isBackend bool) bool {
	if isBackend {
		for _, locked := range dc.ForcedLockedTagServices {
			if serviceName == locked {
				return true
			}
		}
	} else {
		for _, locked := range dc.ForcedLockedTagUIs {
			if serviceName == locked {
				return true
			}
		}
	}
	for _, lockedTag := range dc.LockedTagValues {
		if tag == lockedTag {
			return true
		}
	}
	return false
}
func IsServiceRequired(serviceName string) bool {
	return GlobalDockerConfig.IsServiceRequired(serviceName)
}
func IsRegistrator(serviceName string) bool {
	return GlobalDockerConfig.IsRegistrator(serviceName)
}
func GetParentServiceFromMap(registratorName string) string {
	return GlobalDockerConfig.GetParentService(registratorName)
}
func GetRegistratorsForService(serviceName string) []string {
	return GlobalDockerConfig.GetRegistratorsForService(serviceName)
}
