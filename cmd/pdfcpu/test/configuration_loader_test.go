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
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalCommandUsesEnvironmentConfigurationRoot(t *testing.T) {
	root := t.TempDir()
	initializeConfigurationRoot(t, root)
	replaceConfigurationSchemaVersion(t, root)

	out, err := runNormalCommand(t, isolatedCommandEnvironment(root, t.TempDir()), "paper")
	if err == nil {
		t.Fatalf("normal command ignored invalid environment-selected configuration:\n%s", out)
	}
	if !strings.Contains(string(out), "configuration schema is newer") {
		t.Fatalf("environment-selected schema error not reported:\n%s", out)
	}
}

func TestNormalCommandReturnsConfigurationDiscoveryError(t *testing.T) {
	out, err := runNormalCommand(t, isolatedCommandEnvironment("", ""), "paper")
	if err == nil {
		t.Fatalf("normal command unexpectedly succeeded without configuration discovery:\n%s", out)
	}
	if !strings.Contains(string(out), "discover configuration root") {
		t.Fatalf("configuration discovery error not reported:\n%s", out)
	}
}

func TestNormalCommandExplicitRootOverridesEnvironment(t *testing.T) {
	selectedRoot := t.TempDir()
	environmentRoot := t.TempDir()
	initializeConfigurationRoot(t, selectedRoot)
	initializeConfigurationRoot(t, environmentRoot)
	replaceConfigurationSchemaVersion(t, environmentRoot)

	out, err := runNormalCommand(
		t,
		isolatedCommandEnvironment(environmentRoot, t.TempDir()),
		"--conf",
		selectedRoot,
		"paper",
	)
	if err != nil {
		t.Fatalf("normal command did not prefer explicit root: %v\n%s", err, out)
	}
}

func TestNormalCommandStatelessDoesNotDiscoverConfiguration(t *testing.T) {
	out, err := runNormalCommand(t, isolatedCommandEnvironment("", ""), "--conf", "disable", "paper")
	if err != nil {
		t.Fatalf("stateless normal command: %v\n%s", err, out)
	}
}

func TestNormalCommandActivatesEnvironmentResourceDirectories(t *testing.T) {
	root := t.TempDir()
	initializeConfigurationRoot(t, root)
	env := isolatedCommandEnvironment(root, t.TempDir())

	out, err := runNormalCommand(t, env, "fonts", "list")
	if err != nil {
		t.Fatalf("list fonts: %v\n%s", err, out)
	}
	wantFontDir := "Userfonts(" + filepath.Join(root, "pdfcpu", "fonts") + "):"
	if !strings.Contains(string(out), wantFontDir) {
		t.Fatalf("selected font directory not active, want %q:\n%s", wantFontDir, out)
	}

	out, err = runNormalCommand(t, env, "certificates", "list")
	if err != nil {
		t.Fatalf("list certificates: %v\n%s", err, out)
	}
	wantCertificateDir := "trustedCertDir: " + filepath.Join(root, "pdfcpu", "certs")
	if !strings.Contains(string(out), wantCertificateDir) {
		t.Fatalf("selected certificate directory not active, want %q:\n%s", wantCertificateDir, out)
	}
}
