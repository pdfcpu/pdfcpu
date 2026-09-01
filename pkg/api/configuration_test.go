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
	"testing"
)

func TestConfigurationOptionsZeroValueUsesAutoMode(t *testing.T) {
	var options ConfigurationOptions
	if options.Mode != ConfigurationModeAuto {
		t.Fatalf("zero-value mode: got %d, want %d", options.Mode, ConfigurationModeAuto)
	}
	if err := validateConfigurationOptions(options); err != nil {
		t.Fatalf("validate zero-value options: %v", err)
	}
}

func TestValidateConfigurationOptions(t *testing.T) {
	tests := []struct {
		name    string
		mode    ConfigurationMode
		wantErr bool
	}{
		{"auto", ConfigurationModeAuto, false},
		{"read only", ConfigurationModeReadOnly, false},
		{"stateless", ConfigurationModeStateless, false},
		{"negative", ConfigurationMode(-1), true},
		{"unknown", ConfigurationModeStateless + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfigurationOptions(ConfigurationOptions{Mode: tt.mode})
			if tt.wantErr && !errors.Is(err, ErrInvalidConfigurationMode) {
				t.Fatalf("validate mode %d: got %v, want %v", tt.mode, err, ErrInvalidConfigurationMode)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validate mode %d: %v", tt.mode, err)
			}
		})
	}
}
