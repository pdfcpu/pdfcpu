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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigValidateDoesNotModifyValidConfiguration(t *testing.T) {
	root := t.TempDir()
	initializeConfigurationTree(t, root)
	before := configurationTreeSnapshot(t, root)

	stdout, stderr, err := runPDFCPUWithConfig(t, root, "config", "validate")
	if err != nil {
		t.Fatalf("config validate: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	want := "configuration valid\nconfig: " + filepath.Join(root, "pdfcpu", "config.yml") +
		"\nschema version: 1\n"
	if string(stdout) != want {
		t.Fatalf("unexpected stdout:\nwant:\n%s\ngot:\n%s", want, stdout)
	}
	if len(stderr) != 0 {
		t.Fatalf("unexpected stderr:\n%s", stderr)
	}

	after := configurationTreeSnapshot(t, root)
	assertConfigurationTreeUnchanged(t, before, after)
}

func TestConfigValidateDoesNotModifyInvalidConfiguration(t *testing.T) {
	root := t.TempDir()
	initializeConfigurationTree(t, root)
	configFile := filepath.Join(root, "pdfcpu", "config.yml")
	bb, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	bb = []byte(strings.Replace(string(bb), "schemaVersion: 1", "schemaVersion: 2", 1))
	if err := os.WriteFile(configFile, bb, 0600); err != nil {
		t.Fatal(err)
	}
	before := configurationTreeSnapshot(t, root)

	stdout, stderr, err := runPDFCPUWithConfig(t, root, "config", "validate")
	if err == nil {
		t.Fatalf("config validate unexpectedly succeeded:\n%s", stdout)
	}
	if len(stdout) != 0 {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
	if !strings.Contains(string(stderr), "configuration schema is newer") {
		t.Fatalf("schema error not reported:\n%s", stderr)
	}

	after := configurationTreeSnapshot(t, root)
	assertConfigurationTreeUnchanged(t, before, after)
}

func TestConfigValidateDoesNotCreateMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	_, stderr, err := runPDFCPUWithConfig(t, root, "config", "validate")
	if err == nil {
		t.Fatal("config validate unexpectedly succeeded")
	}
	if !strings.Contains(string(stderr), "load read-only configuration") {
		t.Fatalf("missing configuration error not reported:\n%s", stderr)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("configuration root was created: %v", err)
	}
}

func TestConfigValidateStateless(t *testing.T) {
	cmd := exec.Command(pdfcpuBin, "--conf", "disable", "config", "validate")
	cmd.Env = environmentWithout(
		"APPDATA",
		"HOME",
		"HOMEDRIVE",
		"HOMEPATH",
		"PDFCPU_CONFIG_ROOT",
		"USERPROFILE",
		"XDG_CONFIG_HOME",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config validate: %v\n%s", err, out)
	}
	want := "configuration valid\nconfig: built-in defaults\nschema version: 1\n"
	if string(out) != want {
		t.Fatalf("unexpected output:\nwant:\n%s\ngot:\n%s", want, out)
	}
}

func TestConfigValidateReportsLegacySchemaWithoutWrites(t *testing.T) {
	root := t.TempDir()
	initializeConfigurationTree(t, root)
	rewriteCommandConfiguration(t, root, []byte("schemaVersion: 1\n"), nil)
	before := configurationTreeSnapshot(t, root)

	stdout, stderr, err := runPDFCPUWithConfig(t, root, "config", "validate")
	if err == nil {
		t.Fatalf("config validate unexpectedly succeeded:\n%s", stdout)
	}
	if len(stdout) != 0 {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
	assertCommandOutputContains(
		t,
		stderr,
		"configuration reset required",
		"detected schema version: legacy",
		"required schema version: 1",
	)
	after := configurationTreeSnapshot(t, root)
	assertConfigurationTreeUnchanged(t, before, after)
}
