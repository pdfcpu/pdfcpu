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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalCommandConfigurationSchemaUpgradeContract(t *testing.T) {
	tests := []struct {
		name       string
		prepare    func(*testing.T, string)
		wantError  bool
		wantOutput []string
	}{
		{
			name: "legacy requires reset",
			prepare: func(t *testing.T, root string) {
				rewriteCommandConfiguration(t, root, []byte("schemaVersion: 1\n"), nil)
			},
			wantError: true,
			wantOutput: []string{
				"configuration reset required",
				"detected schema version: legacy",
				"required schema version: 1",
				"pdfcpu config reset",
			},
		},
		{
			name:    "current runs",
			prepare: func(*testing.T, string) {},
		},
		{
			name: "newer requires pdfcpu upgrade",
			prepare: func(t *testing.T, root string) {
				rewriteCommandConfiguration(t, root, []byte("schemaVersion: 1"), []byte("schemaVersion: 2"))
			},
			wantError: true,
			wantOutput: []string{
				"configuration schema is newer",
				"detected schema version: 2",
				"supported schema version: 1",
				"upgrade pdfcpu",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			initializeConfigurationRoot(t, root)
			tt.prepare(t, root)

			out, err := runNormalCommand(t, isolatedCommandEnvironment(root, t.TempDir()), "paper")
			if !tt.wantError {
				if err != nil {
					t.Fatalf("run with current configuration: %v\n%s", err, out)
				}
				return
			}
			if err == nil {
				t.Fatalf("command unexpectedly used incompatible configuration:\n%s", out)
			}
			assertCommandOutputContains(t, out, tt.wantOutput...)
		})
	}
}

func TestLegacySchemaGuidancePreservesExplicitRoot(t *testing.T) {
	root := t.TempDir()
	initializeConfigurationRoot(t, root)
	rewriteCommandConfiguration(t, root, []byte("schemaVersion: 1\n"), nil)

	out, err := runNormalCommand(
		t,
		isolatedCommandEnvironment("", t.TempDir()),
		"--conf",
		root,
		"paper",
	)
	if err == nil {
		t.Fatalf("command unexpectedly used a legacy configuration:\n%s", out)
	}
	want := fmt.Sprintf("pdfcpu --conf %q config reset", root)
	assertCommandOutputContains(t, out, want)
}

func TestConfigListReportsLegacySchemaUpgrade(t *testing.T) {
	root := t.TempDir()
	initializeConfigurationRoot(t, root)
	rewriteCommandConfiguration(t, root, []byte("schemaVersion: 1\n"), nil)

	out, err := runNormalCommand(t, isolatedCommandEnvironment(root, t.TempDir()), "config", "list")
	if err == nil {
		t.Fatalf("config list unexpectedly used a legacy configuration:\n%s", out)
	}
	assertCommandOutputContains(
		t,
		out,
		"configuration reset required",
		"detected schema version: legacy",
		"required schema version: 1",
		"pdfcpu config reset",
	)
}

func TestCurrentSchemaDoesNotGenerateConfigurationProductVersion(t *testing.T) {
	root := t.TempDir()
	initializeConfigurationRoot(t, root)
	path := filepath.Join(root, "pdfcpu", "config.yml")
	bb, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bb, []byte("\nversion:")) {
		t.Fatalf("generated configuration contains product version:\n%s", bb)
	}

	out, err := runNormalCommand(t, isolatedCommandEnvironment(root, t.TempDir()), "paper")
	if err != nil {
		t.Fatalf("run with current schema: %v\n%s", err, out)
	}
}

func TestV015LegacyConfigurationUpgradeProcedure(t *testing.T) {
	root := t.TempDir()
	initializeConfigurationRoot(t, root)
	rewriteCommandConfiguration(
		t,
		root,
		[]byte("schemaVersion: 1\n"),
		[]byte("version: v0.15.0\n"),
	)
	env := isolatedCommandEnvironment(root, t.TempDir())

	out, err := runNormalCommand(t, env, "paper")
	if err == nil {
		t.Fatalf("command unexpectedly used v0.15 configuration:\n%s", out)
	}
	assertCommandOutputContains(
		t,
		out,
		"configuration reset required",
		"detected schema version: legacy",
		"required schema version: 1",
		"pdfcpu config reset",
	)

	stdout, stderr, err := runConfigReset(t, root, "", "--force")
	if err != nil {
		t.Fatalf("reset v0.15 configuration: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	path := filepath.Join(root, "pdfcpu", "config.yml")
	bb, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(bb, []byte("schemaVersion: 1")) || bytes.Contains(bb, []byte("\nversion:")) {
		t.Fatalf("reset did not install schema-1 configuration:\n%s", bb)
	}

	out, err = runNormalCommand(t, env, "paper")
	if err != nil {
		t.Fatalf("command after configuration reset: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "configuration reset required") {
		t.Fatalf("reset guidance repeated after successful reset:\n%s", out)
	}
}
