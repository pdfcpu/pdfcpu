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
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigurationSchemaCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		detected int
		current  int
		cause    error
	}{
		{"legacy to schema 1", 0, 1, ErrConfigurationResetRequired},
		{"schema 2 to schema 3", 2, 3, ErrConfigurationResetRequired},
		{"current schema", 3, 3, nil},
		{"schema 4 with schema 3 binary", 4, 3, ErrConfigurationSchemaTooNew},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const path = "/configuration/config.yml"
			err := checkConfigurationSchemaCompatibility(path, tt.detected, tt.current)
			if tt.cause == nil {
				if err != nil {
					t.Fatalf("check current schema: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.cause) {
				t.Fatalf("schema compatibility: got %v, want %v", err, tt.cause)
			}
			var schemaErr *ConfigurationSchemaCompatibilityError
			if !errors.As(err, &schemaErr) {
				t.Fatalf("schema compatibility error type: %T", err)
			}
			if schemaErr.Path != path || schemaErr.Detected != tt.detected || schemaErr.Current != tt.current {
				t.Fatalf("schema compatibility error: %+v", schemaErr)
			}
		})
	}
}

func TestPreflightConfigurationSchema(t *testing.T) {
	tests := []struct {
		name    string
		content string
		current int
		want    int
		cause   error
	}{
		{"legacy", "offline: false\n", 1, 0, ErrConfigurationResetRequired},
		{"older", "schemaVersion: 2\n", 3, 2, ErrConfigurationResetRequired},
		{"current", "schemaVersion: 3\n", 3, 3, nil},
		{"newer", "schemaVersion: 4\n", 3, 4, ErrConfigurationSchemaTooNew},
		{"invalid", "schemaVersion: zero\n", 3, 0, ErrInvalidConfigurationSchema},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yml")
			if err := os.WriteFile(path, []byte(tt.content), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := preflightConfigurationSchema(path, tt.current)
			if got != tt.want {
				t.Fatalf("detected schema version: got %d, want %d", got, tt.want)
			}
			if !errors.Is(err, tt.cause) {
				t.Fatalf("schema preflight: got %v, want %v", err, tt.cause)
			}
		})
	}
}

func TestSchema2ToSchema3UpgradeProcedure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("schemaVersion: 2\n"), 0600); err != nil {
		t.Fatal(err)
	}

	version, err := preflightConfigurationSchema(path, 3)
	if version != 2 || !errors.Is(err, ErrConfigurationResetRequired) {
		t.Fatalf("schema-2 preflight: version=%d error=%v", version, err)
	}
	var schemaErr *ConfigurationSchemaCompatibilityError
	if !errors.As(err, &schemaErr) {
		t.Fatalf("schema-2 compatibility error type: %T", err)
	}
	if schemaErr.Path != path || schemaErr.Detected != 2 || schemaErr.Current != 3 {
		t.Fatalf("schema-2 compatibility error: %+v", schemaErr)
	}

	if err := os.WriteFile(path, []byte("schemaVersion: 3\n"), 0600); err != nil {
		t.Fatal(err)
	}
	version, err = preflightConfigurationSchema(path, 3)
	if err != nil || version != 3 {
		t.Fatalf("schema-3 preflight after reset: version=%d error=%v", version, err)
	}
}
