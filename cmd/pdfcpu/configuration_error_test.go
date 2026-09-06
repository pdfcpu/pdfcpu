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

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func TestConfigurationSchemaCommandErrorPreservesCause(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "pdfcpu")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, "config.yml")
	if err := os.WriteFile(path, []byte("offline: false\n"), 0600); err != nil {
		t.Fatal(err)
	}

	_, loadErr := api.LoadConfigurationWithOptions(api.ConfigurationOptions{
		Root: root,
		Mode: api.ConfigurationModeReadOnly,
	})
	err := commandError(fmt.Errorf("pdfcpu: %w", loadErr))
	if !errors.Is(err, api.ErrConfigurationResetRequired) {
		t.Fatalf("command error cause: got %v, want %v", err, api.ErrConfigurationResetRequired)
	}
	var schemaErr *api.ConfigurationSchemaCompatibilityError
	if !errors.As(err, &schemaErr) {
		t.Fatalf("command schema error type: %T", err)
	}
	for _, want := range []string{path, "detected schema version: legacy", "required schema version: 1"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("command error does not contain %q: %v", want, err)
		}
	}
}
