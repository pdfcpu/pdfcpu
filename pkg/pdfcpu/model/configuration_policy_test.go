//go:build !js

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
	"testing"

	"go.yaml.in/yaml/v3"
)

// TestBuiltInRevocationTimeoutsMatchEmbeddedConfig verifies that disabling config directory usage preserves safe
// revocation timeouts.
func TestBuiltInRevocationTimeoutsMatchEmbeddedConfig(t *testing.T) {
	var configured configuration
	if err := yaml.Unmarshal(configFileBytes, &configured); err != nil {
		t.Fatalf("parse embedded configuration: %v", err)
	}

	conf := newDefaultConfiguration()
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"CRL", conf.TimeoutCRL, configured.TimeoutCRL},
		{"OCSP", conf.TimeoutOCSP, configured.TimeoutOCSP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want <= 0 {
				t.Fatalf("embedded timeout must be positive: %d", tt.want)
			}
			if tt.got != tt.want {
				t.Fatalf("built-in timeout: got %d, want %d", tt.got, tt.want)
			}
		})
	}
}

// TestAllowedRevocationHostsConfiguration verifies private PKI exceptions survive YAML loading without aliasing input.
func TestAllowedRevocationHostsConfiguration(t *testing.T) {
	var configured configuration
	if err := yaml.Unmarshal([]byte("allowedRevocationHosts: [ocsp.example.corp, crl.example.corp]"), &configured); err != nil {
		t.Fatal(err)
	}

	conf := loadedConfig(configured, "")
	if len(conf.AllowedRevocationHosts) != 2 ||
		conf.AllowedRevocationHosts[0] != "ocsp.example.corp" ||
		conf.AllowedRevocationHosts[1] != "crl.example.corp" {
		t.Fatalf("allowed revocation hosts: %v", conf.AllowedRevocationHosts)
	}

	configured.AllowedRevocationHosts[0] = "changed.example.corp"
	if conf.AllowedRevocationHosts[0] != "ocsp.example.corp" {
		t.Fatalf("loaded configuration aliases parser input: %v", conf.AllowedRevocationHosts)
	}
}

// TestUnsupportedResourcePolicyDefaultsToSkip verifies the operational default independently of validation mode.
func TestUnsupportedResourcePolicyDefaultsToSkip(t *testing.T) {
	tests := []struct {
		name string
		conf *Configuration
		mode int
	}{
		{"default", newDefaultConfiguration(), ValidationRelaxed},
		{"strict YAML", loadedConfig(configuration{ValidationMode: "ValidationStrict"}, ""), ValidationStrict},
		{"relaxed YAML", loadedConfig(configuration{ValidationMode: "ValidationRelaxed"}, ""), ValidationRelaxed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.conf.ValidationMode != tt.mode {
				t.Fatalf("validation mode: got %d, want %d", tt.conf.ValidationMode, tt.mode)
			}
			if tt.conf.UnsupportedResourcePolicy != UnsupportedResourceSkip {
				t.Fatalf("unsupported resource policy: got %d, want %d", tt.conf.UnsupportedResourcePolicy, UnsupportedResourceSkip)
			}
		})
	}
}

// TestLoadValidationModePreservesUnsupportedResourcePolicy verifies that validation parsing does not overwrite an operational policy.
func TestLoadValidationModePreservesUnsupportedResourcePolicy(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{"strict", "ValidationStrict"},
		{"relaxed", "ValidationRelaxed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := Configuration{UnsupportedResourcePolicy: UnsupportedResourceFail}
			loadValidationMode(configuration{ValidationMode: tt.mode}, &conf)
			if conf.UnsupportedResourcePolicy != UnsupportedResourceFail {
				t.Fatalf("unsupported resource policy: got %d, want %d", conf.UnsupportedResourcePolicy, UnsupportedResourceFail)
			}
		})
	}
}
