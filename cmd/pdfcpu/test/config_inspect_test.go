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
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func assertInspectionLines(t *testing.T, output []byte, lines ...string) {
	t.Helper()
	for _, line := range lines {
		if !bytes.Contains(output, []byte(line+"\n")) {
			t.Fatalf("inspection output does not contain %q:\n%s", line, output)
		}
	}
}

func expectedConfiguredInspection(root string) api.ConfigurationInspection {
	configDir := filepath.Join(root, "pdfcpu")
	path := func(value string) api.ConfigurationPathInspection {
		return api.ConfigurationPathInspection{Path: value, Available: true, Exists: true, Writable: true}
	}
	return api.ConfigurationInspection{
		Mode:         api.ConfigurationModeAuto,
		Source:       api.ConfigurationSourceFlag,
		WriteCapable: true,
		Root:         path(root),
		Paths: api.ConfigurationPathsInspection{
			Config:       path(filepath.Join(configDir, "config.yml")),
			Fonts:        path(filepath.Join(configDir, "fonts")),
			Certificates: path(filepath.Join(configDir, "certs")),
		},
		Schema: api.ConfigurationSchemaInspection{
			Detected:         1,
			MinimumSupported: 1,
			MaximumSupported: 1,
		},
		Network: api.ConfigurationNetworkInspection{
			HTTPTimeoutSeconds:         5,
			CRLTimeoutSeconds:          10,
			OCSPTimeoutSeconds:         10,
			PreferredRevocationChecker: "CRL",
			AllowedRevocationHosts:     []string{},
		},
		Limits: api.ConfigurationLimitsInspection{
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
}

func expectedInspectionJSON(t *testing.T, inspection api.ConfigurationInspection) []byte {
	t.Helper()
	bb, err := json.MarshalIndent(inspection, "", "\t")
	if err != nil {
		t.Fatal(err)
	}
	return append(bb, '\n')
}

func TestConfigInspectReportsEffectiveConfigurationWithoutWrites(t *testing.T) {
	root := t.TempDir()
	initializeConfigurationTree(t, root)
	before := configurationTreeSnapshot(t, root)

	stdout, stderr, err := runPDFCPUWithConfig(t, root, "config", "inspect")
	if err != nil {
		t.Fatalf("config inspect: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if len(stderr) != 0 {
		t.Fatalf("unexpected stderr:\n%s", stderr)
	}
	if bytes.Contains(stdout, []byte("format version:")) {
		t.Fatalf("text inspection contains JSON contract version:\n%s", stdout)
	}
	configDir := filepath.Join(root, "pdfcpu")
	assertInspectionLines(
		t,
		stdout,
		"mode: auto",
		"source: flag",
		"default: false",
		"stateless: false",
		"write capable: true",
		"    path: "+root,
		"    path: "+filepath.Join(configDir, "config.yml"),
		"    path: "+filepath.Join(configDir, "fonts"),
		"    path: "+filepath.Join(configDir, "certs"),
		"schema version:",
		"  detected: 1",
		"  minimum supported: 1",
		"  maximum supported: 1",
		"  offline: false",
		"  HTTP timeout seconds: 5",
		"  CRL timeout seconds: 10",
		"  OCSP timeout seconds: 10",
		"  preferred revocation checker: CRL",
		"  allowed revocation hosts: (none)",
		"  max stream bytes: 512 MB",
		"  max decode bytes: 512 MB",
		"  max image pixels: 100 MP",
		"  max image bytes: 512 MB",
		"  max object count: 10000000",
		"  max object stream count: 1000000",
		"  max object stream first: 16 MB",
		"  max xref entries: 10000000",
		"  max recursion depth: 100",
	)

	after := configurationTreeSnapshot(t, root)
	assertConfigurationTreeUnchanged(t, before, after)
}

func TestConfigInspectUsesEnvironmentRootWithoutWrites(t *testing.T) {
	root := t.TempDir()
	initializeConfigurationTree(t, root)
	before := configurationTreeSnapshot(t, root)
	cmd := exec.Command(pdfcpuBin, "config", "inspect")
	cmd.Env = append(environmentWithout("PDFCPU_CONFIG_ROOT"), "PDFCPU_CONFIG_ROOT="+root)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("config inspect: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.Bytes(), stderr.Bytes())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr:\n%s", stderr.Bytes())
	}
	assertInspectionLines(t, stdout.Bytes(), "source: environment", "    path: "+root)
	after := configurationTreeSnapshot(t, root)
	assertConfigurationTreeUnchanged(t, before, after)
}

func TestConfigInspectStatelessWithoutHome(t *testing.T) {
	cmd := exec.Command(pdfcpuBin, "--conf", "disable", "config", "inspect")
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
		t.Fatalf("config inspect: %v\n%s", err, out)
	}
	assertInspectionLines(
		t,
		out,
		"mode: stateless",
		"source: flag",
		"stateless: true",
		"write capable: false",
		"    path: (none)",
		"    available: false",
		"    exists: false",
		"    writable: false",
		"  max image pixels: 100 MP",
	)
}

func TestConfigInspectDoesNotCreateMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	stdout, stderr, err := runPDFCPUWithConfig(t, root, "config", "inspect")
	if err == nil {
		t.Fatalf("config inspect unexpectedly succeeded:\n%s", stdout)
	}
	if len(stdout) != 0 {
		t.Fatalf("unexpected stdout:\n%s", stdout)
	}
	if !strings.Contains(string(stderr), "load read-only configuration") {
		t.Fatalf("missing configuration error not reported:\n%s", stderr)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("configuration root was created: %v", err)
	}
}

func TestConfigInspectJSONIsExactAndDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	initializeConfigurationTree(t, root)
	before := configurationTreeSnapshot(t, root)

	stdout, stderr, err := runPDFCPUWithConfig(t, root, "config", "inspect", "--json")
	if err != nil {
		t.Fatalf("config inspect --json: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if len(stderr) != 0 {
		t.Fatalf("unexpected stderr:\n%s", stderr)
	}
	want := expectedInspectionJSON(t, expectedConfiguredInspection(root))
	if !bytes.Equal(stdout, want) {
		t.Fatalf("unexpected JSON output:\nwant:\n%s\ngot:\n%s", want, stdout)
	}
	for _, secret := range []string{"userPW", "ownerPW", "privateKeyPW"} {
		if bytes.Contains(stdout, []byte(secret)) {
			t.Fatalf("JSON output contains secret field %q:\n%s", secret, stdout)
		}
	}
	after := configurationTreeSnapshot(t, root)
	assertConfigurationTreeUnchanged(t, before, after)
}

func TestConfigInspectReportsIncompatibleSchemaWithoutWrites(t *testing.T) {
	tests := []struct {
		name        string
		old         []byte
		replacement []byte
		want        []string
	}{
		{
			"legacy",
			[]byte("schemaVersion: 1\n"),
			nil,
			[]string{
				"configuration reset required",
				"detected schema version: legacy",
				"required schema version: 1",
			},
		},
		{
			"newer",
			[]byte("schemaVersion: 1"),
			[]byte("schemaVersion: 2"),
			[]string{
				"configuration schema is newer",
				"detected schema version: 2",
				"supported schema version: 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			initializeConfigurationTree(t, root)
			rewriteCommandConfiguration(t, root, tt.old, tt.replacement)
			before := configurationTreeSnapshot(t, root)

			stdout, stderr, err := runPDFCPUWithConfig(t, root, "config", "inspect")
			if err == nil {
				t.Fatalf("config inspect unexpectedly succeeded:\n%s", stdout)
			}
			if len(stdout) != 0 {
				t.Fatalf("unexpected stdout:\n%s", stdout)
			}
			assertCommandOutputContains(t, stderr, tt.want...)
			after := configurationTreeSnapshot(t, root)
			assertConfigurationTreeUnchanged(t, before, after)
		})
	}
}
