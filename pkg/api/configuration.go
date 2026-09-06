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

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const configurationRootEnv = "PDFCPU_CONFIG_ROOT"

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

// String returns the stable name for mode.
func (mode ConfigurationMode) String() string {
	switch mode {
	case ConfigurationModeAuto:
		return "auto"
	case ConfigurationModeReadOnly:
		return "read-only"
	case ConfigurationModeStateless:
		return "stateless"
	}
	return "unknown"
}

// MarshalText returns the stable text representation for mode.
func (mode ConfigurationMode) MarshalText() ([]byte, error) {
	if err := validateConfigurationOptions(ConfigurationOptions{Mode: mode}); err != nil {
		return nil, err
	}
	return []byte(mode.String()), nil
}

// ConfigurationOptions controls configuration loading.
type ConfigurationOptions struct {
	// Root is the parent directory of the pdfcpu configuration directory. An empty root requests mode-specific discovery.
	Root string

	// Mode controls configuration discovery and filesystem access. Its zero value selects automatic mode.
	Mode ConfigurationMode
}

// ConfigurationInitialization reports the result of initializing a configuration tree.
type ConfigurationInitialization struct {
	// Configuration is the loaded effective configuration.
	Configuration *model.Configuration

	// Created is true when config.yml was created by this initialization.
	Created bool
}

func validateConfigurationOptions(options ConfigurationOptions) error {
	switch options.Mode {
	case ConfigurationModeAuto, ConfigurationModeReadOnly, ConfigurationModeStateless:
		return nil
	}
	return fmt.Errorf("%w: %d", ErrInvalidConfigurationMode, options.Mode)
}

func resolveConfigurationRoot(root string) (string, error) {
	root, _, err := resolveConfigurationSelection(root)
	return root, err
}

func resolveConfigurationSelection(root string) (string, ConfigurationSource, error) {
	if root != "" {
		return root, ConfigurationSourceFlag, nil
	}
	if root = os.Getenv(configurationRootEnv); root != "" {
		return root, ConfigurationSourceEnvironment, nil
	}
	root, err := os.UserConfigDir()
	if err != nil {
		return "", "", fmt.Errorf("discover configuration root: %w", err)
	}
	return root, ConfigurationSourceOSDefault, nil
}

// LoadConfigurationWithOptions loads an independent configuration using options.
func LoadConfigurationWithOptions(options ConfigurationOptions) (*model.Configuration, error) {
	if err := validateConfigurationOptions(options); err != nil {
		return nil, err
	}
	if options.Mode == ConfigurationModeStateless {
		return model.NewStatelessConfiguration(), nil
	}

	root, err := resolveConfigurationRoot(options.Root)
	if err != nil {
		return nil, err
	}
	if options.Mode == ConfigurationModeReadOnly {
		conf, err := model.LoadConfigurationReadOnly(root)
		if err != nil {
			return nil, fmt.Errorf("load read-only configuration at %q: %w", root, err)
		}
		return conf, nil
	}

	conf, err := model.LoadConfiguration(root)
	if err != nil {
		return nil, fmt.Errorf("load automatic configuration at %q: %w", root, err)
	}
	return conf, nil
}

// InitializeConfigurationWithOptions loads or initializes the selected configuration tree and reports whether
// config.yml was created.
func InitializeConfigurationWithOptions(options ConfigurationOptions) (*ConfigurationInitialization, error) {
	if err := validateConfigurationOptions(options); err != nil {
		return nil, err
	}
	if options.Mode != ConfigurationModeAuto {
		return nil, fmt.Errorf("%w: mode %s", ErrConfigurationNotWritable, options.Mode)
	}
	root, err := resolveConfigurationRoot(options.Root)
	if err != nil {
		return nil, err
	}
	conf, created, err := model.InitializeConfiguration(root)
	if err != nil {
		return nil, fmt.Errorf("initialize configuration at %q: %w", root, err)
	}
	return &ConfigurationInitialization{Configuration: conf, Created: created}, nil
}

// ResetConfigurationWithOptions replaces config.yml at the selected root with the built-in configuration.
func ResetConfigurationWithOptions(options ConfigurationOptions) (*model.Configuration, error) {
	if err := validateConfigurationOptions(options); err != nil {
		return nil, err
	}
	if options.Mode != ConfigurationModeAuto {
		return nil, fmt.Errorf("%w: mode %s", ErrConfigurationNotWritable, options.Mode)
	}
	root, err := resolveConfigurationRoot(options.Root)
	if err != nil {
		return nil, err
	}
	conf, err := model.ResetConfiguration(root)
	if err != nil {
		return nil, fmt.Errorf("reset configuration at %q: %w", root, err)
	}
	return conf, nil
}
