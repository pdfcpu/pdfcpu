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

package test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func customizeResetConfiguration(t *testing.T, root string) string {
	t.Helper()
	configPath := filepath.Join(root, "pdfcpu", "config.yml")
	bb, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := bytes.Replace(bb, []byte("offline: false"), []byte("offline: true"), 1)
	if bytes.Equal(updated, bb) {
		t.Fatal("configuration does not contain offline setting")
	}
	if err := os.WriteFile(configPath, updated, 0600); err != nil {
		t.Fatal(err)
	}
	return configPath
}

func TestConfigResetForceUsesSelectedRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "selected")
	otherRoot := filepath.Join(t.TempDir(), "environment")
	for _, selected := range []string{root, otherRoot} {
		if _, stderr, err := runPDFCPUWithConfig(t, selected, "config", "init"); err != nil {
			t.Fatalf("config init %q: %v\n%s", selected, err, stderr)
		}
		customizeResetConfiguration(t, selected)
	}
	t.Setenv("PDFCPU_CONFIG_ROOT", otherRoot)
	otherBefore := configurationTreeSnapshot(t, otherRoot)

	stdout, stderr, err := runConfigReset(t, root, "", "--force")
	if err != nil {
		t.Fatalf("config reset --force: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if len(stderr) != 0 {
		t.Fatalf("unexpected stderr:\n%s", stderr)
	}
	configPath := filepath.Join(root, "pdfcpu", "config.yml")
	want := "configuration reset\nconfig: " + configPath + "\nschema version: 1\n"
	if string(stdout) != want {
		t.Fatalf("unexpected stdout:\nwant:\n%s\ngot:\n%s", want, stdout)
	}
	bb, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bb, []byte("offline: true")) {
		t.Fatal("selected configuration was not reset")
	}
	otherAfter := configurationTreeSnapshot(t, otherRoot)
	assertConfigurationTreeUnchanged(t, otherBefore, otherAfter)
}

func TestConfigResetForceUsesEnvironmentRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "environment")
	if _, stderr, err := runPDFCPUWithConfig(t, root, "config", "init"); err != nil {
		t.Fatalf("config init: %v\n%s", err, stderr)
	}
	configPath := customizeResetConfiguration(t, root)
	cmd := exec.Command(pdfcpuBin, "config", "reset", "--force")
	cmd.Env = append(environmentWithout("PDFCPU_CONFIG_ROOT"), "PDFCPU_CONFIG_ROOT="+root)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config reset --force: %v\n%s", err, out)
	}
	bb, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bb, []byte("offline: true")) {
		t.Fatal("environment configuration was not reset")
	}
}

func TestConfigResetCancellationDoesNotModifyConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantError  bool
		wantStdout string
		wantStderr string
	}{
		{
			name:       "EOF requires explicit confirmation",
			wantError:  true,
			wantStdout: "Reset the selected configuration to built-in defaults? (yes/no): ",
			wantStderr: "confirmation required; use --force",
		},
		{
			name:       "no cancels",
			input:      "no\n",
			wantStdout: "Reset the selected configuration to built-in defaults? (yes/no): configuration reset canceled\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "selected")
			if _, stderr, err := runPDFCPUWithConfig(t, root, "config", "init"); err != nil {
				t.Fatalf("config init: %v\n%s", err, stderr)
			}
			customizeResetConfiguration(t, root)
			before := configurationTreeSnapshot(t, root)

			stdout, stderr, err := runConfigReset(t, root, tt.input)
			if tt.wantError && err == nil {
				t.Fatalf("config reset unexpectedly succeeded:\n%s", stdout)
			}
			if !tt.wantError && err != nil {
				t.Fatalf("config reset: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
			}
			if string(stdout) != tt.wantStdout {
				t.Fatalf("unexpected stdout:\nwant:\n%s\ngot:\n%s", tt.wantStdout, stdout)
			}
			if tt.wantStderr != "" && !strings.Contains(string(stderr), tt.wantStderr) {
				t.Fatalf("stderr does not contain %q:\n%s", tt.wantStderr, stderr)
			}
			if tt.wantStderr == "" && len(stderr) != 0 {
				t.Fatalf("unexpected stderr:\n%s", stderr)
			}

			after := configurationTreeSnapshot(t, root)
			assertConfigurationTreeUnchanged(t, before, after)
		})
	}
}

func TestConfigResetRejectsStatelessMode(t *testing.T) {
	stdout, stderr, err := runConfigReset(t, "disable", "", "--force")
	if err == nil {
		t.Fatalf("config reset unexpectedly succeeded:\n%s", stdout)
	}
	if len(stdout) != 0 {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
	if !strings.Contains(string(stderr), "configuration is not writable: mode stateless") {
		t.Fatalf("stateless error not reported:\n%s", stderr)
	}
}

func TestConfigResetRecoversIncompatibleSchemaAndPreservesResources(t *testing.T) {
	tests := []struct {
		name        string
		old         []byte
		replacement []byte
	}{
		{"legacy", []byte("schemaVersion: 1\n"), nil},
		{"newer", []byte("schemaVersion: 1"), []byte("schemaVersion: 2")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "selected")
			if _, stderr, err := runPDFCPUWithConfig(t, root, "config", "init"); err != nil {
				t.Fatalf("config init: %v\n%s", err, stderr)
			}
			configDir := filepath.Join(root, "pdfcpu")
			fontPath := filepath.Join(configDir, "fonts", "custom-font.marker")
			certificatePath := filepath.Join(configDir, "certs", "custom-certificate.marker")
			resources := []struct {
				path    string
				content string
			}{
				{fontPath, "font"},
				{certificatePath, "cert"},
			}
			for _, resource := range resources {
				if err := os.WriteFile(resource.path, []byte(resource.content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			rewriteCommandConfiguration(t, root, tt.old, tt.replacement)

			if stdout, stderr, err := runConfigReset(t, root, "", "--force"); err != nil {
				t.Fatalf("config reset: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
			}
			configBytes, err := os.ReadFile(filepath.Join(configDir, "config.yml"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(configBytes, []byte("schemaVersion: 1")) {
				t.Fatalf("reset configuration does not contain current schema:\n%s", configBytes)
			}
			for _, resource := range resources {
				got, err := os.ReadFile(resource.path)
				if err != nil {
					t.Fatalf("read preserved resource %q: %v", resource.path, err)
				}
				if string(got) != resource.content {
					t.Fatalf(
						"preserved resource %q: got %q, want %q",
						resource.path,
						got,
						resource.content,
					)
				}
			}
		})
	}
}
