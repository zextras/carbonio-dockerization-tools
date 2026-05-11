// SPDX-FileCopyrightText: 2026 Zextras <https://www.zextras.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

type DockerConfig struct {
	HiddenServices          []string
	RequiredServices        []string
	AutoIncludedServices    []string
	LockedTagValues         []string
	ForcedLockedTagServices []string
	ForcedLockedTagUIs      []string
}

func DefaultDockerConfig() *DockerConfig {
	return &DockerConfig{
		HiddenServices: []string{
			"event-listener",
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
			"event-listener",
			"consul-register",
			"carbonio-provisioner",
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
