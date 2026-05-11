// SPDX-FileCopyrightText: 2026 Zextras <https://www.zextras.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

func ExportConfig(filePath string, config *UserConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0755); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}
func CreateUserConfig(edition string, backend map[string]*ImageConfig, frontend map[string]*ImageConfig, appVersion string) *UserConfig {
	return &UserConfig{
		AppVersion: appVersion,
		Carbonio: CarbonioConfig{
			Edition:  edition,
			Backend:  backend,
			Frontend: frontend,
		},
	}
}
