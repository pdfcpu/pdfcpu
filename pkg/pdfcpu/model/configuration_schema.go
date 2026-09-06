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

package model

import (
	"bytes"
	"errors"
	"fmt"
	"os"
)

var (
	// ErrConfigurationResetRequired signals a configuration schema older than the schema required by this build.
	ErrConfigurationResetRequired = errors.New("configuration reset required")

	// ErrConfigurationSchemaTooNew signals a configuration schema newer than the schemas supported by this build.
	ErrConfigurationSchemaTooNew = errors.New("configuration schema is newer")
)

// ConfigurationSchemaCompatibilityError reports an incompatible persisted configuration schema.
type ConfigurationSchemaCompatibilityError struct {
	Path     string
	Detected int
	Current  int
	cause    error
}

// Error returns the configuration schema compatibility failure.
func (e *ConfigurationSchemaCompatibilityError) Error() string {
	if errors.Is(e.cause, ErrConfigurationResetRequired) {
		return fmt.Sprintf(
			"%v at %q: detected schema version %d, required schema version %d",
			e.cause,
			e.Path,
			e.Detected,
			e.Current,
		)
	}
	return fmt.Sprintf(
		"%v at %q: detected schema version %d, supported schema version %d",
		e.cause,
		e.Path,
		e.Detected,
		e.Current,
	)
}

// Unwrap returns the configuration schema compatibility cause.
func (e *ConfigurationSchemaCompatibilityError) Unwrap() error {
	return e.cause
}

func checkConfigurationSchemaCompatibility(path string, detected, current int) error {
	if detected < current {
		return &ConfigurationSchemaCompatibilityError{
			Path:     path,
			Detected: detected,
			Current:  current,
			cause:    ErrConfigurationResetRequired,
		}
	}
	if detected > current {
		return &ConfigurationSchemaCompatibilityError{
			Path:     path,
			Detected: detected,
			Current:  current,
			cause:    ErrConfigurationSchemaTooNew,
		}
	}
	return nil
}

func preflightConfigurationSchema(path string, current int) (int, error) {
	bb, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	version, err := readConfigurationSchemaVersion(bytes.NewReader(bb))
	if err != nil {
		return 0, fmt.Errorf("read configuration schema at %q: %w", path, err)
	}
	return version, checkConfigurationSchemaCompatibility(path, version, current)
}
