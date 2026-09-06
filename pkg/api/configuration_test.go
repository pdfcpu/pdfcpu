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
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
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

func TestLoadConfigurationWithOptionsStatelessDoesNotDiscoverRoot(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing")
	t.Setenv(configurationRootEnv, missingRoot)

	conf, err := LoadConfigurationWithOptions(ConfigurationOptions{Mode: ConfigurationModeStateless})
	if err != nil {
		t.Fatalf("load stateless configuration: %v", err)
	}
	if conf.Path != "" {
		t.Fatalf("stateless configuration path: got %q, want empty", conf.Path)
	}
	if _, err := os.Stat(missingRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stateless load accessed discovered root: %v", err)
	}
}

func TestLoadConfigurationWithOptionsExplicitRoot(t *testing.T) {
	root := t.TempDir()
	discoveredRoot := filepath.Join(t.TempDir(), "discovered")
	t.Setenv(configurationRootEnv, discoveredRoot)

	conf, err := LoadConfigurationWithOptions(ConfigurationOptions{Root: root})
	if err != nil {
		t.Fatalf("load automatic configuration: %v", err)
	}
	wantPath := filepath.Join(root, "pdfcpu", "config.yml")
	if conf.Path != wantPath {
		t.Fatalf("automatic configuration path: got %q, want %q", conf.Path, wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("stat automatic configuration: %v", err)
	}
	if _, err := os.Stat(discoveredRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("explicit root did not take precedence: %v", err)
	}

	readOnly, err := LoadConfigurationWithOptions(ConfigurationOptions{
		Root: root,
		Mode: ConfigurationModeReadOnly,
	})
	if err != nil {
		t.Fatalf("load read-only configuration: %v", err)
	}
	if readOnly.Path != wantPath {
		t.Fatalf("read-only configuration path: got %q, want %q", readOnly.Path, wantPath)
	}
}

func TestLoadConfigurationWithOptionsUsesEnvironmentRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv(configurationRootEnv, root)

	conf, err := LoadConfigurationWithOptions(ConfigurationOptions{})
	if err != nil {
		t.Fatalf("load discovered configuration: %v", err)
	}
	wantPath := filepath.Join(root, "pdfcpu", "config.yml")
	if conf.Path != wantPath {
		t.Fatalf("discovered configuration path: got %q, want %q", conf.Path, wantPath)
	}
}

func TestInitializeConfigurationWithOptionsReportsCreation(t *testing.T) {
	root := t.TempDir()
	result, err := InitializeConfigurationWithOptions(ConfigurationOptions{Root: root})
	if err != nil {
		t.Fatalf("initialize new configuration: %v", err)
	}
	if !result.Created {
		t.Fatal("new configuration not reported as created")
	}
	path := filepath.Join(root, "pdfcpu", "config.yml")
	if result.Configuration.Path != path {
		t.Fatalf("configuration path: got %q, want %q", result.Configuration.Path, path)
	}
	replaceConfigurationSetting(t, path, "offline: false", "offline: true")

	result, err = InitializeConfigurationWithOptions(ConfigurationOptions{Root: root})
	if err != nil {
		t.Fatalf("initialize existing configuration: %v", err)
	}
	if result.Created {
		t.Fatal("existing configuration reported as created")
	}
	if !result.Configuration.Offline {
		t.Fatal("existing configuration was not loaded unchanged")
	}
}

func TestInitializeConfigurationWithOptionsRejectsNonAutomaticModes(t *testing.T) {
	for _, mode := range []ConfigurationMode{ConfigurationModeReadOnly, ConfigurationModeStateless} {
		_, err := InitializeConfigurationWithOptions(ConfigurationOptions{Mode: mode})
		if !errors.Is(err, ErrConfigurationNotWritable) {
			t.Fatalf("initialize mode %s: got %v, want %v", mode, err, ErrConfigurationNotWritable)
		}
	}
}

func TestLoadConfigurationWithOptionsReturnsIndependentConfigurations(t *testing.T) {
	root := t.TempDir()
	if _, err := LoadConfigurationWithOptions(ConfigurationOptions{Root: root}); err != nil {
		t.Fatalf("initialize configuration: %v", err)
	}
	path := filepath.Join(root, "pdfcpu", "config.yml")
	bb, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read configuration: %v", err)
	}
	updated := bytes.Replace(bb, []byte("allowedRevocationHosts: []"), []byte("allowedRevocationHosts: [one.example]"), 1)
	if bytes.Equal(updated, bb) {
		t.Fatal("default configuration does not contain allowedRevocationHosts")
	}
	bb = updated
	if err := os.WriteFile(path, bb, 0600); err != nil {
		t.Fatalf("write configuration: %v", err)
	}

	first, err := LoadConfigurationWithOptions(ConfigurationOptions{Root: root})
	if err != nil {
		t.Fatalf("load first configuration: %v", err)
	}
	second, err := LoadConfigurationWithOptions(ConfigurationOptions{Root: root})
	if err != nil {
		t.Fatalf("load second configuration: %v", err)
	}

	first.ValidationMode = model.ValidationStrict
	first.AllowedRevocationHosts[0] = "changed.example"
	if second.ValidationMode != model.ValidationRelaxed {
		t.Fatalf("second validation mode: got %d, want %d", second.ValidationMode, model.ValidationRelaxed)
	}
	if len(second.AllowedRevocationHosts) != 1 || second.AllowedRevocationHosts[0] != "one.example" {
		t.Fatalf("second allowed revocation hosts changed: %v", second.AllowedRevocationHosts)
	}
}

func TestLoadConfigurationWithOptionsReturnsOrdinaryErrors(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	_, err := LoadConfigurationWithOptions(ConfigurationOptions{
		Root: root,
		Mode: ConfigurationModeReadOnly,
	})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("load missing read-only configuration: got %v, want %v", err, os.ErrNotExist)
	}
}

func replaceConfigurationSetting(t *testing.T, path, old, replacement string) {
	t.Helper()
	bb, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := bytes.Replace(bb, []byte(old), []byte(replacement), 1)
	if bytes.Equal(updated, bb) {
		t.Fatalf("configuration does not contain %q", old)
	}
	if err := os.WriteFile(path, updated, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestResetConfigurationWithOptionsUsesExplicitRoot(t *testing.T) {
	root := t.TempDir()
	otherRoot := t.TempDir()
	rootConf, err := LoadConfigurationWithOptions(ConfigurationOptions{Root: root})
	if err != nil {
		t.Fatalf("initialize selected root: %v", err)
	}
	otherConf, err := LoadConfigurationWithOptions(ConfigurationOptions{Root: otherRoot})
	if err != nil {
		t.Fatalf("initialize other root: %v", err)
	}
	replaceConfigurationSetting(t, rootConf.Path, "offline: false", "offline: true")
	replaceConfigurationSetting(t, otherConf.Path, "offline: false", "offline: true")
	otherBefore, err := os.ReadFile(otherConf.Path)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(configurationRootEnv, otherRoot)

	reset, err := ResetConfigurationWithOptions(ConfigurationOptions{Root: root})
	if err != nil {
		t.Fatalf("reset selected root: %v", err)
	}
	if reset.Path != rootConf.Path || reset.Offline {
		t.Fatalf("unexpected reset configuration: %+v", reset)
	}
	otherAfter, err := os.ReadFile(otherConf.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(otherAfter, otherBefore) {
		t.Fatal("reset modified environment-selected root")
	}
}

func TestResetConfigurationWithOptionsUsesEnvironmentRoot(t *testing.T) {
	root := t.TempDir()
	conf, err := LoadConfigurationWithOptions(ConfigurationOptions{Root: root})
	if err != nil {
		t.Fatalf("initialize configuration: %v", err)
	}
	replaceConfigurationSetting(t, conf.Path, "offline: false", "offline: true")
	t.Setenv(configurationRootEnv, root)

	reset, err := ResetConfigurationWithOptions(ConfigurationOptions{})
	if err != nil {
		t.Fatalf("reset environment configuration: %v", err)
	}
	if reset.Path != conf.Path || reset.Offline {
		t.Fatalf("unexpected reset configuration: %+v", reset)
	}
}

func TestResetConfigurationWithOptionsRejectsNonWritableModes(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing")
	t.Setenv(configurationRootEnv, missingRoot)
	for _, mode := range []ConfigurationMode{ConfigurationModeReadOnly, ConfigurationModeStateless} {
		_, err := ResetConfigurationWithOptions(ConfigurationOptions{Mode: mode})
		if !errors.Is(err, ErrConfigurationNotWritable) {
			t.Fatalf("reset mode %s: got %v, want %v", mode, err, ErrConfigurationNotWritable)
		}
	}
	if _, err := os.Stat(missingRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rejected reset accessed configuration root: %v", err)
	}
}

func TestResetConfigurationWithOptionsRejectsInvalidMode(t *testing.T) {
	_, err := ResetConfigurationWithOptions(ConfigurationOptions{Mode: ConfigurationMode(-1)})
	if !errors.Is(err, ErrInvalidConfigurationMode) {
		t.Fatalf("reset invalid mode: got %v, want %v", err, ErrInvalidConfigurationMode)
	}
}
