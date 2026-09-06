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
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigInitCreatesSelectedRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "selected")
	environmentRoot := filepath.Join(t.TempDir(), "environment")
	t.Setenv("PDFCPU_CONFIG_ROOT", environmentRoot)

	stdout, stderr, err := runPDFCPUWithConfig(t, root, "config", "init")
	if err != nil {
		t.Fatalf("config init: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if len(stderr) != 0 {
		t.Fatalf("unexpected stderr:\n%s", stderr)
	}
	configDir := filepath.Join(root, "pdfcpu")
	want := "configuration initialized\n" +
		"root: " + root + "\n" +
		"config: " + filepath.Join(configDir, "config.yml") + "\n" +
		"fonts: " + filepath.Join(configDir, "fonts") + "\n" +
		"certificates: " + filepath.Join(configDir, "certs") + "\n" +
		"schema version: 1\n"
	if string(stdout) != want {
		t.Fatalf("unexpected stdout:\nwant:\n%s\ngot:\n%s", want, stdout)
	}
	for _, path := range []string{
		filepath.Join(configDir, "config.yml"),
		filepath.Join(configDir, "fonts"),
		filepath.Join(configDir, "certs"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("initialized path %q: %v", path, err)
		}
	}
	if _, err := os.Stat(environmentRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("explicit root did not take precedence: %v", err)
	}
}

func TestConfigInitUsesEnvironmentRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "environment")
	cmd := exec.Command(pdfcpuBin, "config", "init")
	cmd.Env = append(environmentWithout("PDFCPU_CONFIG_ROOT"), "PDFCPU_CONFIG_ROOT="+root)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("config init: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.Bytes(), stderr.Bytes())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr:\n%s", stderr.Bytes())
	}
	if !strings.Contains(stdout.String(), "root: "+root+"\n") {
		t.Fatalf("environment root not reported:\n%s", stdout.Bytes())
	}
	if _, err := os.Stat(filepath.Join(root, "pdfcpu", "config.yml")); err != nil {
		t.Fatalf("environment configuration not initialized: %v", err)
	}
}

func TestConfigInitDoesNotOverwriteValidTree(t *testing.T) {
	root := filepath.Join(t.TempDir(), "selected")
	if _, stderr, err := runPDFCPUWithConfig(t, root, "config", "init"); err != nil {
		t.Fatalf("first config init: %v\n%s", err, stderr)
	}
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
	before := configurationTreeSnapshot(t, root)

	stdout, stderr, err := runPDFCPUWithConfig(t, root, "config", "init")
	if err != nil {
		t.Fatalf("second config init: %v\n%s", err, stderr)
	}
	if !strings.HasPrefix(string(stdout), "configuration already initialized\n") {
		t.Fatalf("existing configuration status not reported:\n%s", stdout)
	}
	after := configurationTreeSnapshot(t, root)
	assertConfigurationTreeUnchanged(t, before, after)
}

func TestConfigInitDoesNotOverwriteMalformedConfiguration(t *testing.T) {
	root := filepath.Join(t.TempDir(), "selected")
	if _, stderr, err := runPDFCPUWithConfig(t, root, "config", "init"); err != nil {
		t.Fatalf("first config init: %v\n%s", err, stderr)
	}
	configPath := filepath.Join(root, "pdfcpu", "config.yml")
	bb, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := bytes.Replace(bb, []byte("schemaVersion: 1"), []byte("schemaVersion: 2"), 1)
	if bytes.Equal(updated, bb) {
		t.Fatal("configuration does not contain schema version")
	}
	if err := os.WriteFile(configPath, updated, 0600); err != nil {
		t.Fatal(err)
	}
	before := configurationTreeSnapshot(t, root)

	stdout, stderr, err := runPDFCPUWithConfig(t, root, "config", "init")
	if err == nil {
		t.Fatalf("config init unexpectedly succeeded:\n%s", stdout)
	}
	if !strings.Contains(string(stderr), "configuration schema is newer") {
		t.Fatalf("schema error not reported:\n%s", stderr)
	}
	after := configurationTreeSnapshot(t, root)
	assertConfigurationTreeUnchanged(t, before, after)
}

func TestConfigInitRejectsStatelessMode(t *testing.T) {
	cmd := exec.Command(pdfcpuBin, "--conf", "disable", "config", "init")
	cmd.Env = environmentWithout(
		"APPDATA",
		"HOME",
		"HOMEDRIVE",
		"HOMEPATH",
		"PDFCPU_CONFIG_ROOT",
		"USERPROFILE",
		"XDG_CONFIG_HOME",
	)
	stdout, stderr := bytes.Buffer{}, bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatalf("config init unexpectedly succeeded:\n%s", stdout.Bytes())
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected stdout:\n%s", stdout.Bytes())
	}
	if !strings.Contains(stderr.String(), "--conf disable selects stateless mode") {
		t.Fatalf("stateless error not reported:\n%s", stderr.Bytes())
	}
}

func TestConfigInitReportsLegacySchemaWithoutWrites(t *testing.T) {
	root := filepath.Join(t.TempDir(), "selected")
	if _, stderr, err := runPDFCPUWithConfig(t, root, "config", "init"); err != nil {
		t.Fatalf("first config init: %v\n%s", err, stderr)
	}
	rewriteCommandConfiguration(t, root, []byte("schemaVersion: 1\n"), nil)
	before := configurationTreeSnapshot(t, root)

	stdout, stderr, err := runPDFCPUWithConfig(t, root, "config", "init")
	if err == nil {
		t.Fatalf("config init unexpectedly succeeded:\n%s", stdout)
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
