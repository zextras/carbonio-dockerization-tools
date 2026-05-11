// SPDX-FileCopyrightText: 2026 Zextras <https://www.zextras.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

type ServiceDefinition struct {
	Name         string
	DisplayName  string
	EnvVar       string
	DefaultImage string
	DefaultTag   string
	DependsOn    []string
	Available    []string
	IsRequired   bool
}

type UIImageDefinition struct {
	Name         string
	EnvVar       string
	DefaultImage string
	DefaultTag   string
	IsProxy      bool
}

type ParsedConfig struct {
	BackendServices map[string]*ServiceDefinition
	FrontendImages  map[string]*UIImageDefinition
}

type Edition string

const (
	EditionCE       Edition = "ce"
	EditionAdvanced Edition = "advanced"
)
