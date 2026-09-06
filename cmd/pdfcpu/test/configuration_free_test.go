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
	"runtime"
	"strings"
	"testing"
)

func runConfigurationFreeCommand(t *testing.T, configRoot string, args ...string) []byte {
	t.Helper()
	commandArgs := append([]string{"--conf", configRoot}, args...)
	cmd := exec.Command(pdfcpuBin, commandArgs...)
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
		t.Fatalf("%v failed: %v\n%s", args, err, out)
	}
	return out
}

func TestConfigurationFreeCommandsDoNotAccessConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		notWant string
	}{
		{"help", []string{"help"}, "", ""},
		{"version", []string{"version"}, "version:", "config:"},
		{"completion", []string{"completion", "bash"}, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent := t.TempDir()
			if runtime.GOOS != "windows" {
				if err := os.Chmod(parent, 0555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(parent, 0755) })
			}
			configRoot := filepath.Join(parent, "missing-config-root")
			out := runConfigurationFreeCommand(t, configRoot, tt.args...)
			if tt.want != "" && !strings.Contains(string(out), tt.want) {
				t.Fatalf("output does not contain %q:\n%s", tt.want, out)
			}
			if tt.notWant != "" && strings.Contains(string(out), tt.notWant) {
				t.Fatalf("output contains %q:\n%s", tt.notWant, out)
			}
			if _, err := os.Stat(configRoot); !os.IsNotExist(err) {
				t.Fatalf("configuration root was accessed or created: %v", err)
			}
		})
	}
}

func TestConfigurationDependentCommandStillValidatesConfiguration(t *testing.T) {
	configRoot := filepath.Join(t.TempDir(), "missing-config-root")
	cmd := exec.Command(pdfcpuBin, "--conf", configRoot, "paper")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("paper unexpectedly ignored missing configuration root:\n%s", out)
	}
	if !strings.Contains(string(out), "does not exist") {
		t.Fatalf("missing configuration error not reported:\n%s", out)
	}
}
