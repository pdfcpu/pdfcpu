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
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLoadConfigurationSchemaUpgradeContract(t *testing.T) {
	tests := []struct {
		name       string
		prepare    func(*testing.T, string) string
		wantError  bool
		cause      error
		detected   int
		current    int
		wantOutput []string
	}{
		{
			name: "legacy requires reset",
			prepare: func(t *testing.T, root string) string {
				path := filepath.Join(root, "pdfcpu", "config.yml")
				replaceConfigurationSetting(t, path, "schemaVersion: 1\n", "")
				return path
			},
			wantError: true,
			cause:     ErrConfigurationResetRequired,
			detected:  0,
			current:   1,
			wantOutput: []string{
				"configuration reset required",
				"detected schema version 0",
				"required schema version 1",
			},
		},
		{
			name: "current loads",
			prepare: func(_ *testing.T, root string) string {
				return filepath.Join(root, "pdfcpu", "config.yml")
			},
		},
		{
			name: "newer requires pdfcpu upgrade",
			prepare: func(t *testing.T, root string) string {
				path := filepath.Join(root, "pdfcpu", "config.yml")
				replaceConfigurationSetting(t, path, "schemaVersion: 1", "schemaVersion: 2")
				return path
			},
			wantError: true,
			cause:     ErrConfigurationSchemaTooNew,
			detected:  2,
			current:   1,
			wantOutput: []string{
				"configuration schema is newer",
				"detected schema version 2",
				"supported schema version 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if _, err := LoadConfiguration(ConfigurationOptions{Root: root}); err != nil {
				t.Fatalf("initialize configuration: %v", err)
			}
			path := tt.prepare(t, root)
			_, err := LoadConfiguration(ConfigurationOptions{
				Root: root,
				Mode: ConfigurationModeReadOnly,
			})
			if !tt.wantError {
				if err != nil {
					t.Fatalf("load current configuration: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("obsolete configuration unexpectedly loaded")
			}
			if !errors.Is(err, tt.cause) {
				t.Fatalf("configuration error: got %v, want %v", err, tt.cause)
			}
			var schemaErr *ConfigurationSchemaCompatibilityError
			if !errors.As(err, &schemaErr) {
				t.Fatalf("configuration schema error type: %T", err)
			}
			if schemaErr.Path != path || schemaErr.Detected != tt.detected || schemaErr.Current != tt.current {
				t.Fatalf("configuration schema error: %+v", schemaErr)
			}
			for _, want := range append([]string{strconv.Quote(path)}, tt.wantOutput...) {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("configuration error does not contain %q: %v", want, err)
				}
			}
		})
	}
}

func TestLoadConfigurationStatelessIgnoresPersistedSchema(t *testing.T) {
	root := t.TempDir()
	if _, err := LoadConfiguration(ConfigurationOptions{Root: root}); err != nil {
		t.Fatalf("initialize configuration: %v", err)
	}
	path := filepath.Join(root, "pdfcpu", "config.yml")
	replaceConfigurationSetting(t, path, "schemaVersion: 1", "schemaVersion: 2")
	t.Setenv(configurationRootEnv, root)

	if _, err := LoadConfiguration(ConfigurationOptions{Mode: ConfigurationModeStateless}); err != nil {
		t.Fatalf("load stateless configuration: %v", err)
	}
}
