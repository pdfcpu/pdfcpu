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
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/font"
)

func TestReadConfigurationFileDoesNotWriteOrCache(t *testing.T) {
	preserveConfigurationGlobals(t)

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := initializeConfigurationFile(path); err != nil {
		t.Fatalf("initialize configuration: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read configuration before test: %v", err)
	}
	beforeInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat configuration before test: %v", err)
	}

	cached := &Configuration{Path: "cached.yml"}
	loadedDefaultConfig = cached
	conf, err := readConfigurationFile(path)
	if err != nil {
		t.Fatalf("read configuration file: %v", err)
	}
	if conf.Path != path {
		t.Fatalf("configuration path: got %q, want %q", conf.Path, path)
	}
	if loadedDefaultConfig != cached {
		t.Fatal("configuration file read replaced cached configuration")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read configuration after test: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("configuration file read changed file content")
	}
	afterInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat configuration after test: %v", err)
	}
	if afterInfo.Mode() != beforeInfo.Mode() {
		t.Fatalf("configuration mode changed: got %v, want %v", afterInfo.Mode(), beforeInfo.Mode())
	}
	if !afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
		t.Fatalf("configuration modification time changed: got %v, want %v", afterInfo.ModTime(), beforeInfo.ModTime())
	}
}

func TestReadConfigurationFileDoesNotCreateMissingState(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "missing")
	path := filepath.Join(configDir, "config.yml")

	if _, err := readConfigurationFile(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read missing configuration: got %v, want %v", err, os.ErrNotExist)
	}
	if _, err := os.Stat(configDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing configuration read created state: %v", err)
	}
}

func requireReadOnlyConfigurationResources(t *testing.T, conf *Configuration, configPath, configDir string) {
	t.Helper()
	if conf.Path != configPath {
		t.Fatalf("configuration path: got %q, want %q", conf.Path, configPath)
	}
	want := resourcesForConfigurationDir(configurationResourceModeReadOnly, configDir)
	if conf.resources != want {
		t.Fatalf("configuration resources: got %+v, want %+v", conf.resources, want)
	}
	trustedCertDir, available := conf.TrustedCertificateStore()
	if !available || trustedCertDir != want.trustedCertDir {
		t.Fatalf(
			"trusted certificate store: got %q, available=%t; want %q, available=true",
			trustedCertDir,
			available,
			want.trustedCertDir,
		)
	}
	userFontDir, available := conf.UserFontStore()
	if !available || userFontDir != want.userFontDir {
		t.Fatalf(
			"user font store: got %q, available=%t; want %q, available=true",
			userFontDir,
			available,
			want.userFontDir,
		)
	}
	if got := (&XRefTable{Conf: conf}).FontRepository(); got != font.RepositoryForDir(userFontDir) {
		t.Fatal("cross-reference table selected a different font repository")
	}
}

func requireUnchangedConfigurationGlobals(t *testing.T, cached *Configuration) {
	t.Helper()
	if loadedDefaultConfig != cached {
		t.Fatal("read-only configuration load replaced cached configuration")
	}
	if ConfigPath != "cached-root" {
		t.Fatalf("read-only configuration load changed config root: %q", ConfigPath)
	}
	if font.UserFontDir != "cached-fonts" {
		t.Fatalf("read-only configuration load changed user font directory: %q", font.UserFontDir)
	}
	if TrustedCertDir != "cached-certs" {
		t.Fatalf("read-only configuration load changed trusted certificate directory: %q", TrustedCertDir)
	}
}

func requireMissingResourceDirectories(t *testing.T, configDir string) {
	t.Helper()
	for _, name := range []string{"fonts", "certs"} {
		path := filepath.Join(configDir, name)
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("read-only configuration load created %s: %v", name, err)
		}
	}
}

func TestReadConfigurationAtDoesNotInitializeTreeOrGlobals(t *testing.T) {
	preserveConfigurationGlobals(t)

	root := t.TempDir()
	configDir := filepath.Join(root, "pdfcpu")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("create configuration directory: %v", err)
	}
	configPath := filepath.Join(configDir, "config.yml")
	if err := initializeConfigurationFile(configPath); err != nil {
		t.Fatalf("initialize configuration: %v", err)
	}

	cached := &Configuration{Path: "cached.yml"}
	loadedDefaultConfig = cached
	ConfigPath = "cached-root"
	font.UserFontDir = "cached-fonts"
	TrustedCertDir = "cached-certs"

	conf, err := readConfigurationAt(root)
	if err != nil {
		t.Fatalf("read configuration tree: %v", err)
	}
	requireReadOnlyConfigurationResources(t, conf, configPath, configDir)
	requireUnchangedConfigurationGlobals(t, cached)
	requireMissingResourceDirectories(t, configDir)
}

func TestEnsureConfigFileAtRecordsAutomaticResources(t *testing.T) {
	preserveConfigurationGlobals(t)

	TrustedCertDir = "legacy-certs"
	font.UserFontDir = "legacy-fonts"
	configDir := filepath.Join(t.TempDir(), "pdfcpu")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("create configuration directory: %v", err)
	}
	if err := ensureConfigFileAt(filepath.Join(configDir, "config.yml"), false); err != nil {
		t.Fatalf("ensure configuration file: %v", err)
	}

	want := resourcesForConfigurationDir(configurationResourceModeAuto, configDir)
	if loadedDefaultConfig.resources != want {
		t.Fatalf("configuration resources: got %+v, want %+v", loadedDefaultConfig.resources, want)
	}
	trustedCertDir, available := loadedDefaultConfig.TrustedCertificateStore()
	if !available || trustedCertDir != TrustedCertDir {
		t.Fatalf(
			"trusted certificate store: got %q, available=%t; want %q, available=true",
			trustedCertDir,
			available,
			TrustedCertDir,
		)
	}
	userFontDir, available := loadedDefaultConfig.UserFontStore()
	if !available || userFontDir != font.UserFontDir {
		t.Fatalf(
			"user font store: got %q, available=%t; want %q, available=true",
			userFontDir,
			available,
			font.UserFontDir,
		)
	}
}

func TestReadConfigurationAtRequiresExistingExplicitRoot(t *testing.T) {
	if _, err := readConfigurationAt(""); err == nil {
		t.Fatal("expected missing configuration root error")
	}

	root := filepath.Join(t.TempDir(), "missing")
	if _, err := readConfigurationAt(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read missing configuration root: got %v, want %v", err, os.ErrNotExist)
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read-only configuration load created missing root: %v", err)
	}
}

func TestLoadConfigurationInitializesSelectedRootWithoutChangingCompatibilityGlobals(t *testing.T) {
	preserveConfigurationGlobals(t)

	cached := &Configuration{Path: "cached.yml"}
	loadedDefaultConfig = cached
	ConfigPath = "cached-root"
	font.UserFontDir = "cached-fonts"
	TrustedCertDir = "cached-certs"

	root := t.TempDir()
	conf, err := LoadConfiguration(root)
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}
	configDir := filepath.Join(root, "pdfcpu")
	if conf.Path != filepath.Join(configDir, "config.yml") {
		t.Fatalf("configuration path: got %q, want root %q", conf.Path, configDir)
	}
	userFontDir, available := conf.UserFontStore()
	if !available || userFontDir != filepath.Join(configDir, "fonts") {
		t.Fatalf("user font store: got %q, available=%t", userFontDir, available)
	}
	trustedCertDir, available := conf.TrustedCertificateStore()
	if !available || trustedCertDir != filepath.Join(configDir, "certs") {
		t.Fatalf("trusted certificate store: got %q, available=%t", trustedCertDir, available)
	}
	requireUnchangedConfigurationGlobals(t, cached)
}

func requireUnchangedConfigurationFile(t *testing.T, path string, before os.FileInfo, want []byte) {
	t.Helper()

	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if after.Mode() != before.Mode() || !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("configuration metadata changed during loading")
	}
	bb, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bb, want) {
		t.Fatal("configuration content changed during loading")
	}
}

func TestConfigurationLoadRejectsLegacySchemaWithoutRewriting(t *testing.T) {
	tests := []struct {
		name string
		load func(string) (*Configuration, error)
	}{
		{"automatic", LoadConfiguration},
		{"read only", LoadConfigurationReadOnly},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			configDir := filepath.Join(root, "pdfcpu")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(configDir, "config.yml")
			legacy := configurationWithSchemaLine(t, "")
			if err := os.WriteFile(path, legacy, 0600); err != nil {
				t.Fatal(err)
			}
			before, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}

			_, err = tt.load(root)
			if !errors.Is(err, ErrConfigurationResetRequired) {
				t.Fatalf("load legacy configuration: got %v, want %v", err, ErrConfigurationResetRequired)
			}
			var schemaErr *ConfigurationSchemaCompatibilityError
			if !errors.As(err, &schemaErr) {
				t.Fatalf("legacy schema error type: %T", err)
			}
			if schemaErr.Path != path || schemaErr.Detected != 0 || schemaErr.Current != 1 {
				t.Fatalf("legacy schema error: %+v", schemaErr)
			}
			requireUnchangedConfigurationFile(t, path, before, legacy)
		})
	}
}

func TestEnsureConfigFileAtDoesNotReplaceInvalidExistingFile(t *testing.T) {
	preserveConfigurationGlobals(t)

	path := filepath.Join(t.TempDir(), "config.yml")
	invalid := []byte("invalid configuration")
	if err := os.WriteFile(path, invalid, 0600); err != nil {
		t.Fatalf("write invalid configuration: %v", err)
	}
	cached := &Configuration{Path: "cached.yml"}
	loadedDefaultConfig = cached

	if err := ensureConfigFileAt(path, false); err == nil {
		t.Fatal("expected invalid configuration error")
	}
	if loadedDefaultConfig != cached {
		t.Fatal("invalid existing configuration replaced cached configuration")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read invalid configuration: %v", err)
	}
	if !bytes.Equal(after, invalid) {
		t.Fatal("invalid existing configuration was replaced")
	}
}

func TestReadConfigurationDoesNotMutateLoaderState(t *testing.T) {
	preserveConfigurationGlobals(t)

	cached := &Configuration{Path: "cached.yml", ValidationMode: ValidationStrict}
	loadedDefaultConfig = cached

	conf, err := readConfiguration(bytes.NewReader(configFileBytes), "read.yml")
	if err != nil {
		t.Fatalf("read configuration: %v", err)
	}
	if conf == cached {
		t.Fatal("read configuration aliases cached configuration")
	}
	if conf.Path != "read.yml" {
		t.Fatalf("read configuration path: got %q, want %q", conf.Path, "read.yml")
	}
	if loadedDefaultConfig != cached {
		t.Fatal("read configuration replaced cached configuration")
	}
}

func TestReadConfigurationFailureDoesNotMutateLoaderState(t *testing.T) {
	preserveConfigurationGlobals(t)

	cached := &Configuration{Path: "cached.yml"}
	loadedDefaultConfig = cached
	invalid := bytes.Replace(
		configFileBytes,
		[]byte("validationMode: ValidationRelaxed"),
		[]byte("validationMode: invalid"),
		1,
	)

	if _, err := readConfiguration(bytes.NewReader(invalid), "invalid.yml"); err == nil {
		t.Fatal("expected invalid configuration error")
	}
	if loadedDefaultConfig != cached {
		t.Fatal("failed configuration read replaced cached configuration")
	}
}

func TestParseConfigFileRetainsCompatibilityAssignment(t *testing.T) {
	preserveConfigurationGlobals(t)

	cached := &Configuration{Path: "cached.yml"}
	loadedDefaultConfig = cached
	if err := parseConfigFile(bytes.NewReader(configFileBytes), "compatibility.yml"); err != nil {
		t.Fatalf("parse configuration: %v", err)
	}
	if loadedDefaultConfig == cached {
		t.Fatal("compatibility parser did not replace cached configuration")
	}
	if loadedDefaultConfig.Path != "compatibility.yml" {
		t.Fatalf("cached configuration path: got %q, want %q", loadedDefaultConfig.Path, "compatibility.yml")
	}
}

func preserveConfigurationGlobals(t *testing.T) {
	t.Helper()
	configPath := ConfigPath
	loadedConfig := loadedDefaultConfig
	userFontDir := font.UserFontDir
	trustedCertDir := TrustedCertDir
	t.Cleanup(func() {
		ConfigPath = configPath
		loadedDefaultConfig = loadedConfig
		font.UserFontDir = userFontDir
		TrustedCertDir = trustedCertDir
	})
}

func TestNewStatelessConfigurationIgnoresLoaderState(t *testing.T) {
	preserveConfigurationGlobals(t)

	stateRoot := t.TempDir()
	discoveredRoot := filepath.Join(stateRoot, "discovered")
	t.Setenv("XDG_CONFIG_HOME", discoveredRoot)
	t.Setenv("HOME", filepath.Join(stateRoot, "home"))

	ConfigPath = filepath.Join(stateRoot, "configured")
	loadedDefaultConfig = &Configuration{
		Path:           filepath.Join(stateRoot, "cached.yml"),
		ValidationMode: ValidationStrict,
	}
	font.UserFontDir = filepath.Join(stateRoot, "cached-fonts")
	TrustedCertDir = filepath.Join(stateRoot, "cached-certs")

	conf := NewStatelessConfiguration()
	if conf.Path != "" {
		t.Fatalf("stateless configuration path: got %q, want empty", conf.Path)
	}
	if conf.ValidationMode != ValidationRelaxed {
		t.Fatalf("stateless validation mode: got %d, want %d", conf.ValidationMode, ValidationRelaxed)
	}
	wantResources := configurationResources{mode: configurationResourceModeStateless}
	if conf.resources != wantResources {
		t.Fatalf("stateless configuration resources: got %+v, want %+v", conf.resources, wantResources)
	}
	if trustedCertDir, available := conf.TrustedCertificateStore(); available || trustedCertDir != "" {
		t.Fatalf(
			"stateless trusted certificate store: got %q, available=%t; want empty, available=false",
			trustedCertDir,
			available,
		)
	}
	if userFontDir, available := conf.UserFontStore(); available || userFontDir != "" {
		t.Fatalf(
			"stateless user font store: got %q, available=%t; want empty, available=false",
			userFontDir,
			available,
		)
	}
	if got := (&XRefTable{Conf: conf}).FontRepository(); got != font.RepositoryForDir("") {
		t.Fatal("stateless cross-reference table selected a filesystem font repository")
	}
	if _, err := os.Stat(discoveredRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stateless construction accessed discovered root: %v", err)
	}
	if font.UserFontDir != filepath.Join(stateRoot, "cached-fonts") {
		t.Fatalf("stateless construction changed user font directory: %q", font.UserFontDir)
	}
	if TrustedCertDir != filepath.Join(stateRoot, "cached-certs") {
		t.Fatalf("stateless construction changed trusted certificate directory: %q", TrustedCertDir)
	}
}

func TestNewStatelessConfigurationReturnsIndependentState(t *testing.T) {
	conf1 := NewStatelessConfiguration()
	conf2 := NewStatelessConfiguration()
	if conf1 == conf2 {
		t.Fatal("stateless configuration instances alias")
	}

	conf1.ValidationMode = ValidationStrict
	conf1.AllowedRevocationHosts = append(conf1.AllowedRevocationHosts, "changed.example")
	if conf2.ValidationMode != ValidationRelaxed {
		t.Fatalf("second validation mode: got %d, want %d", conf2.ValidationMode, ValidationRelaxed)
	}
	if len(conf2.AllowedRevocationHosts) != 0 {
		t.Fatalf("second allowed revocation hosts changed: %v", conf2.AllowedRevocationHosts)
	}
}
