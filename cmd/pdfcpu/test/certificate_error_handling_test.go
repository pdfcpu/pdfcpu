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
	"path/filepath"
	"strings"
	"testing"
)

// TestListCertificatesPrintsPartialOutputOnError verifies mixed listings print and fail.
func TestListCertificatesPrintsPartialOutputOnError(t *testing.T) {
	configDir := t.TempDir()
	if _, stderr, err := runPDFCPUWithConfig(t, configDir, "version"); err != nil {
		t.Fatalf("initialize config: %v\nstderr:\n%s", err, stderr)
	}

	certDir := filepath.Join(configDir, "pdfcpu", "certs")
	valid, err := os.ReadFile(repoFile(t, "pkg", "pdfcpu", "model", "resources", "certs", "uk.p7c"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "valid.p7c"), valid, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "empty.pem"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runPDFCPUWithConfig(t, configDir, "certificates", "list")
	if err == nil {
		t.Fatal("expected nonzero certificate listing exit")
	}
	for _, want := range []string{"valid.p7c:", "empty.pem:", "total installed certs:"} {
		if !strings.Contains(string(stdout), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if !strings.Contains(string(stderr), "no certificates found") {
		t.Fatalf("stderr missing certificate error:\n%s", stderr)
	}
}
