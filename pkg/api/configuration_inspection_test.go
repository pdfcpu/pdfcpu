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
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type inspectionTreeEntry struct {
	mode    fs.FileMode
	modTime time.Time
	digest  [sha256.Size]byte
}

func inspectionTreeSnapshot(t *testing.T, root string) map[string]inspectionTreeEntry {
	t.Helper()
	entries := map[string]inspectionTreeEntry{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		treeEntry := inspectionTreeEntry{mode: info.Mode(), modTime: info.ModTime()}
		if info.Mode().IsRegular() {
			bb, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			treeEntry.digest = sha256.Sum256(bb)
		}
		entries[rel] = treeEntry
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func initializeInspectionRoot(t *testing.T, root string) string {
	t.Helper()
	conf, err := LoadConfiguration(ConfigurationOptions{Root: root})
	if err != nil {
		t.Fatalf("initialize configuration: %v", err)
	}
	return conf.Path
}

func setInspectionTreeMode(t *testing.T, root string, dirMode, fileMode fs.FileMode) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return os.Chmod(path, dirMode)
		}
		return os.Chmod(path, fileMode)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestConfigurationModeText(t *testing.T) {
	tests := []struct {
		mode ConfigurationMode
		want string
	}{
		{ConfigurationModeAuto, "auto"},
		{ConfigurationModeReadOnly, "read-only"},
		{ConfigurationModeStateless, "stateless"},
	}

	for _, tt := range tests {
		bb, err := tt.mode.MarshalText()
		if err != nil {
			t.Fatalf("marshal %d: %v", tt.mode, err)
		}
		if got := string(bb); got != tt.want {
			t.Fatalf("marshal %d: got %q, want %q", tt.mode, got, tt.want)
		}
	}

	if _, err := (ConfigurationMode(-1)).MarshalText(); !errors.Is(err, ErrInvalidConfigurationMode) {
		t.Fatalf("marshal invalid mode: got %v, want %v", err, ErrInvalidConfigurationMode)
	}
}

func TestConfigurationInspectionJSONContract(t *testing.T) {
	inspection := ConfigurationInspection{
		Mode:         ConfigurationModeReadOnly,
		Source:       ConfigurationSourceFlag,
		Default:      false,
		Stateless:    false,
		WriteCapable: false,
		Root: ConfigurationPathInspection{
			Path:      "/srv/pdfcpu",
			Available: true,
			Exists:    true,
			Writable:  false,
		},
		Paths: ConfigurationPathsInspection{
			Config: ConfigurationPathInspection{
				Path:      "/srv/pdfcpu/pdfcpu/config.yml",
				Available: true,
				Exists:    true,
				Writable:  false,
			},
			Fonts: ConfigurationPathInspection{
				Path:      "/srv/pdfcpu/pdfcpu/fonts",
				Available: true,
				Exists:    true,
				Writable:  false,
			},
			Certificates: ConfigurationPathInspection{
				Path:      "/srv/pdfcpu/pdfcpu/certs",
				Available: true,
				Exists:    true,
				Writable:  false,
			},
		},
		Schema: ConfigurationSchemaInspection{
			Detected:         1,
			MinimumSupported: 1,
			MaximumSupported: 1,
		},
		Network: ConfigurationNetworkInspection{
			Offline:                    true,
			HTTPTimeoutSeconds:         5,
			CRLTimeoutSeconds:          10,
			OCSPTimeoutSeconds:         10,
			PreferredRevocationChecker: "CRL",
			AllowedRevocationHosts:     []string{"pki.example"},
		},
		Limits: ConfigurationLimitsInspection{
			MaxStreamBytes:       536870912,
			MaxDecodeBytes:       536870912,
			MaxImagePixels:       100000000,
			MaxImageBytes:        536870912,
			MaxObjectCount:       10000000,
			MaxObjectStreamCount: 1000000,
			MaxObjectStreamFirst: 16777216,
			MaxXRefEntries:       10000000,
			MaxRecursionDepth:    100,
		},
	}

	bb, err := json.Marshal(inspection)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"mode":"read-only","source":"flag","default":false,"stateless":false,` +
		`"writeCapable":false,"root":{"path":"/srv/pdfcpu","available":true,"exists":true,"writable":false},` +
		`"paths":{"config":{"path":"/srv/pdfcpu/pdfcpu/config.yml","available":true,"exists":true,` +
		`"writable":false},"fonts":{"path":"/srv/pdfcpu/pdfcpu/fonts","available":true,"exists":true,` +
		`"writable":false},"certificates":{"path":"/srv/pdfcpu/pdfcpu/certs","available":true,"exists":true,` +
		`"writable":false}},"schema":{"detected":1,"minimumSupported":1,"maximumSupported":1},` +
		`"network":{"offline":true,"httpTimeoutSeconds":5,"crlTimeoutSeconds":10,"ocspTimeoutSeconds":10,` +
		`"preferredRevocationChecker":"CRL","allowedRevocationHosts":["pki.example"]},` +
		`"limits":{"maxStreamBytes":536870912,"maxDecodeBytes":536870912,"maxImagePixels":100000000,` +
		`"maxImageBytes":536870912,"maxObjectCount":10000000,"maxObjectStreamCount":1000000,` +
		`"maxObjectStreamFirst":16777216,"maxXRefEntries":10000000,"maxRecursionDepth":100}}`
	if got := string(bb); got != want {
		t.Fatalf("inspection JSON changed:\nwant: %s\ngot:  %s", want, got)
	}
}

func assertExplicitInspectionIdentity(t *testing.T, inspection *ConfigurationInspection) {
	t.Helper()
	if inspection.Mode != ConfigurationModeAuto {
		t.Fatalf("unexpected inspection identity: %+v", inspection)
	}
	if inspection.Source != ConfigurationSourceFlag || inspection.Default || inspection.Stateless {
		t.Fatalf("unexpected selection: %+v", inspection)
	}
	if !inspection.WriteCapable {
		t.Fatal("automatic mode not reported as write capable")
	}
	if inspection.Schema.Detected != 1 || inspection.Schema.MinimumSupported != 1 ||
		inspection.Schema.MaximumSupported != 1 {
		t.Fatalf("unexpected schema inspection: %+v", inspection.Schema)
	}
}

func assertExplicitInspectionPaths(t *testing.T, inspection *ConfigurationInspection, root, configPath string) {
	t.Helper()
	if inspection.Root.Path != root || !inspection.Root.Available || !inspection.Root.Exists ||
		!inspection.Root.Writable {
		t.Fatalf("unexpected root inspection: %+v", inspection.Root)
	}
	if inspection.Paths.Config.Path != configPath || !inspection.Paths.Config.Exists ||
		!inspection.Paths.Config.Writable {
		t.Fatalf("unexpected config inspection: %+v", inspection.Paths.Config)
	}
	fontPath := filepath.Join(root, "pdfcpu", "fonts")
	if inspection.Paths.Fonts.Path != fontPath || !inspection.Paths.Fonts.Exists || !inspection.Paths.Fonts.Writable {
		t.Fatalf("unexpected font inspection: %+v", inspection.Paths.Fonts)
	}
	certificatePath := filepath.Join(root, "pdfcpu", "certs")
	if inspection.Paths.Certificates.Path != certificatePath || !inspection.Paths.Certificates.Available ||
		inspection.Paths.Certificates.Exists || inspection.Paths.Certificates.Writable {
		t.Fatalf("unexpected certificate inspection: %+v", inspection.Paths.Certificates)
	}
}

func assertExplicitInspectionSettings(t *testing.T, inspection *ConfigurationInspection) {
	t.Helper()
	if !inspection.Network.Offline || inspection.Network.HTTPTimeoutSeconds != 7 ||
		inspection.Network.PreferredRevocationChecker != "OCSP" {
		t.Fatalf("unexpected network inspection: %+v", inspection.Network)
	}
	if len(inspection.Network.AllowedRevocationHosts) != 1 ||
		inspection.Network.AllowedRevocationHosts[0] != "pki.example" {
		t.Fatalf("unexpected allowed revocation hosts: %v", inspection.Network.AllowedRevocationHosts)
	}
	wantLimits := ConfigurationLimitsInspection{
		MaxStreamBytes:       536870912,
		MaxDecodeBytes:       536870912,
		MaxImagePixels:       100000000,
		MaxImageBytes:        536870912,
		MaxObjectCount:       10000000,
		MaxObjectStreamCount: 1000000,
		MaxObjectStreamFirst: 16777216,
		MaxXRefEntries:       10000000,
		MaxRecursionDepth:    100,
	}
	if inspection.Limits != wantLimits {
		t.Fatalf("unexpected limits inspection: %+v", inspection.Limits)
	}
}

func TestInspectConfigurationReportsExplicitSelectionWithoutWrites(t *testing.T) {
	root := t.TempDir()
	configPath := initializeInspectionRoot(t, root)
	missingEnvironmentRoot := filepath.Join(t.TempDir(), "environment")
	t.Setenv(configurationRootEnv, missingEnvironmentRoot)
	replaceConfigurationSetting(t, configPath, "offline: false", "offline: true")
	replaceConfigurationSetting(t, configPath, "timeout: 5", "timeout: 7")
	replaceConfigurationSetting(t, configPath, "allowedRevocationHosts: []", "allowedRevocationHosts: [pki.example]")
	replaceConfigurationSetting(t, configPath, "preferredCertRevocationChecker: crl", "preferredCertRevocationChecker: ocsp")
	certificatePath := filepath.Join(root, "pdfcpu", "certs")
	if err := os.RemoveAll(certificatePath); err != nil {
		t.Fatal(err)
	}
	before := inspectionTreeSnapshot(t, root)

	inspection, err := InspectConfiguration(ConfigurationOptions{Root: root})
	if err != nil {
		t.Fatalf("inspect configuration: %v", err)
	}
	assertExplicitInspectionIdentity(t, inspection)
	assertExplicitInspectionPaths(t, inspection, root, configPath)
	assertExplicitInspectionSettings(t, inspection)

	after := inspectionTreeSnapshot(t, root)
	if !maps.Equal(before, after) {
		t.Fatal("configuration inspection modified the configuration tree")
	}
	if _, err := os.Stat(missingEnvironmentRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("explicit selection accessed environment root: %v", err)
	}
}

func TestInspectConfigurationReportsEnvironmentSelection(t *testing.T) {
	root := t.TempDir()
	initializeInspectionRoot(t, root)
	t.Setenv(configurationRootEnv, root)

	inspection, err := InspectConfiguration(ConfigurationOptions{Mode: ConfigurationModeReadOnly})
	if err != nil {
		t.Fatalf("inspect configuration: %v", err)
	}
	if inspection.Source != ConfigurationSourceEnvironment || inspection.Default {
		t.Fatalf("unexpected selection: %+v", inspection)
	}
	if inspection.Mode != ConfigurationModeReadOnly || inspection.WriteCapable {
		t.Fatalf("unexpected mode: %+v", inspection)
	}
}

func TestResolveConfigurationSelectionUsesOSDefault(t *testing.T) {
	t.Setenv(configurationRootEnv, "")
	want, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	root, source, err := resolveConfigurationSelection("")
	if err != nil {
		t.Fatal(err)
	}
	if root != want || source != ConfigurationSourceOSDefault {
		t.Fatalf("OS default selection: got (%q, %q), want (%q, %q)", root, source, want, ConfigurationSourceOSDefault)
	}
}

func TestInspectConfigurationStatelessDoesNotDiscoverRoot(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing")
	t.Setenv(configurationRootEnv, missingRoot)

	inspection, err := InspectConfiguration(ConfigurationOptions{Mode: ConfigurationModeStateless})
	if err != nil {
		t.Fatalf("inspect stateless configuration: %v", err)
	}
	if inspection.Mode != ConfigurationModeStateless || inspection.Source != ConfigurationSourceFlag ||
		!inspection.Stateless || inspection.Default || inspection.WriteCapable {
		t.Fatalf("unexpected stateless selection: %+v", inspection)
	}
	if inspection.Root.Available || inspection.Paths.Config.Available || inspection.Paths.Fonts.Available ||
		inspection.Paths.Certificates.Available {
		t.Fatalf("stateless filesystem resources reported as available: %+v", inspection)
	}
	if inspection.Network.AllowedRevocationHosts == nil || len(inspection.Network.AllowedRevocationHosts) != 0 {
		t.Fatalf("unexpected stateless allowed hosts: %v", inspection.Network.AllowedRevocationHosts)
	}
	if _, err := os.Stat(missingRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stateless inspection accessed discovered root: %v", err)
	}
}

func TestInspectConfigurationReportsReadOnlyPaths(t *testing.T) {
	root := t.TempDir()
	initializeInspectionRoot(t, root)
	setInspectionTreeMode(t, root, 0555, 0444)
	t.Cleanup(func() {
		setInspectionTreeMode(t, root, 0755, 0644)
	})
	before := inspectionTreeSnapshot(t, root)

	inspection, err := InspectConfiguration(ConfigurationOptions{Root: root, Mode: ConfigurationModeReadOnly})
	if err != nil {
		t.Fatalf("inspect read-only configuration: %v", err)
	}
	paths := []ConfigurationPathInspection{
		inspection.Root,
		inspection.Paths.Config,
		inspection.Paths.Fonts,
		inspection.Paths.Certificates,
	}
	for _, path := range paths {
		if !path.Available || !path.Exists || path.Writable {
			t.Fatalf("unexpected read-only path inspection: %+v", path)
		}
	}
	if after := inspectionTreeSnapshot(t, root); !maps.Equal(before, after) {
		t.Fatal("read-only inspection modified the configuration tree")
	}
}

func TestInspectConfigurationDoesNotCreateMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	_, err := InspectConfiguration(ConfigurationOptions{Root: root})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("inspect missing root: got %v, want %v", err, os.ErrNotExist)
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("configuration root was created: %v", err)
	}
}

func TestInspectConfigurationRejectsInvalidMode(t *testing.T) {
	_, err := InspectConfiguration(ConfigurationOptions{Mode: ConfigurationMode(-1)})
	if !errors.Is(err, ErrInvalidConfigurationMode) {
		t.Fatalf("inspect invalid mode: got %v, want %v", err, ErrInvalidConfigurationMode)
	}
}

func TestInspectConfigurationReportsIncompatibleSchemaWithoutWrites(t *testing.T) {
	tests := []struct {
		name        string
		old         string
		replacement string
		cause       error
		detected    int
	}{
		{"legacy", "schemaVersion: 1\n", "", ErrConfigurationResetRequired, 0},
		{"newer", "schemaVersion: 1", "schemaVersion: 2", ErrConfigurationSchemaTooNew, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			path := initializeInspectionRoot(t, root)
			replaceConfigurationSetting(t, path, tt.old, tt.replacement)
			before := inspectionTreeSnapshot(t, root)

			_, err := InspectConfiguration(ConfigurationOptions{Root: root})
			if !errors.Is(err, tt.cause) {
				t.Fatalf("inspect incompatible schema: got %v, want %v", err, tt.cause)
			}
			var schemaErr *ConfigurationSchemaCompatibilityError
			if !errors.As(err, &schemaErr) {
				t.Fatalf("inspection schema error type: %T", err)
			}
			if schemaErr.Path != path || schemaErr.Detected != tt.detected || schemaErr.Current != 1 {
				t.Fatalf("inspection schema error: %+v", schemaErr)
			}
			if after := inspectionTreeSnapshot(t, root); !maps.Equal(before, after) {
				t.Fatal("incompatible schema inspection modified the configuration tree")
			}
		})
	}
}
