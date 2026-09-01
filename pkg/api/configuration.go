/*
Copyright 2026 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import "fmt"

// ConfigurationMode controls configuration discovery and filesystem access.
type ConfigurationMode int

const (
	// ConfigurationModeAuto discovers, reads and initializes configuration state when required.
	ConfigurationModeAuto ConfigurationMode = iota

	// ConfigurationModeReadOnly reads an existing configuration tree without modifying it.
	ConfigurationModeReadOnly

	// ConfigurationModeStateless uses built-in configuration without accessing configuration state on disk.
	ConfigurationModeStateless
)

// ConfigurationOptions controls configuration loading.
type ConfigurationOptions struct {
	// Root is the parent directory of the pdfcpu configuration directory. An empty root requests mode-specific discovery.
	Root string

	// Mode controls configuration discovery and filesystem access. Its zero value selects automatic mode.
	Mode ConfigurationMode
}

func validateConfigurationOptions(options ConfigurationOptions) error {
	switch options.Mode {
	case ConfigurationModeAuto, ConfigurationModeReadOnly, ConfigurationModeStateless:
		return nil
	}
	return fmt.Errorf("%w: %d", ErrInvalidConfigurationMode, options.Mode)
}
