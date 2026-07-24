/*
Copyright 2025 The pdfcpu Authors.

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
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestListCertificates verifies list certificates.
func TestListCertificates(t *testing.T) {
	msg := "TestListCertificates"

	cmd := cli.ListCertificatesCommand(false, conf)
	if _, err := cli.Dispatch(cmd); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

// TestListCertificatesJSON verifies list certificates JSON output.
func TestListCertificatesJSON(t *testing.T) {
	msg := "TestListCertificatesJSON"
	certDir := t.TempDir()
	restoreTrustedCertDir(t, certDir)

	bb, err := os.ReadFile(filepath.Join("..", "..", "pdfcpu", "model", "resources", "certs", "uk.p7c"))
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err = os.WriteFile(filepath.Join(certDir, "uk.p7c"), bb, 0644); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	cmd := cli.ListCertificatesCommand(true, conf)
	out, err := cli.Dispatch(cmd)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if len(out) != 1 {
		t.Fatalf("%s: want 1 output string, got %d\n", msg, len(out))
	}

	var list struct {
		TrustedCertDir      string `json:"trustedCertDir"`
		TotalInstalledCerts int    `json:"totalInstalledCerts"`
		Files               []struct {
			Name         string `json:"name"`
			Certificates []struct {
				Subject struct {
					CommonName string `json:"commonName"`
				} `json:"subject"`
				SerialNumber string `json:"serialNumber"`
				NotBefore    string `json:"notBefore"`
				NotAfter     string `json:"notAfter"`
			} `json:"certificates"`
		} `json:"files"`
	}
	if err = json.Unmarshal([]byte(out[0]), &list); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if list.TrustedCertDir != certDir {
		t.Fatalf("%s: want cert dir %s, got %s\n", msg, certDir, list.TrustedCertDir)
	}
	if list.TotalInstalledCerts == 0 || len(list.Files) != 1 {
		t.Fatalf("%s: missing certificates: %v\n", msg, list)
	}
}

// TestListCertificatesErrorContext verifies CLI listing surfaces API errors unchanged.
func TestListCertificatesErrorContext(t *testing.T) {
	dir := t.TempDir()
	fileName := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(fileName, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	restoreTrustedCertDir(t, filepath.Join(fileName, "certs"))

	cmd := cli.ListCertificatesCommand(false, conf)
	_, err := cli.Dispatch(cmd)
	if err == nil {
		t.Fatal("expected directory creation error")
	}
	if !strings.Contains(err.Error(), "list certificates: create trusted certificate directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCertificateCommandErrorContext verifies CLI import and inspect preserve API causes and context.
func TestCertificateCommandErrorContext(t *testing.T) {
	missingFile := filepath.Join(t.TempDir(), "missing.pem")
	tests := []struct {
		name string
		cmd  *cli.Command
		want string
	}{
		{
			name: "import",
			cmd:  cli.ImportCertificatesCommand([]string{missingFile}, conf),
			want: "import certificates: input 1",
		},
		{
			name: "inspect",
			cmd:  cli.InspectCertificatesCommand([]string{missingFile}, conf),
			want: "inspect certificates: input 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cli.Dispatch(tt.cmd)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, err)
			}
		})
	}
}

func restoreTrustedCertDir(t *testing.T, certDir string) {
	t.Helper()
	orig := model.TrustedCertDir
	model.TrustedCertDir = certDir
	t.Cleanup(func() {
		model.TrustedCertDir = orig
	})
}
