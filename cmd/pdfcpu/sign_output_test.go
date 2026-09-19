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

package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func useEmptyCLISignatureTrustStore(t *testing.T) {
	t.Helper()
	oldDir := model.TrustedCertDir
	model.TrustedCertDir = t.TempDir()
	pdfcpu.InvalidateCertificatePool()
	t.Cleanup(func() {
		model.TrustedCertDir = oldDir
		pdfcpu.InvalidateCertificatePool()
	})
}

func signatureOutputSummary(s string) string {
	const details = "\nDetails:"
	if i := strings.Index(s, details); i >= 0 {
		return s[:i]
	}
	return s
}

// TestValidateSignaturesCLIUsesAPIOutput verifies compact and full CLI output
// preserve the API presentation without owning a duplicate formatting contract.
func TestValidateSignaturesCLIUsesAPIOutput(t *testing.T) {
	useEmptyCLISignatureTrustStore(t)
	tests := []struct {
		name string
		full bool
	}{
		{name: "compact"},
		{name: "full", full: true},
	}

	inFile := filepath.Join("..", "..", "pkg", "testdata", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := model.NewDefaultConfiguration()
			conf.Offline = true
			lines, err := api.ValidateSignaturesFile(t.Context(), inFile, false, tt.full, conf)
			if err != nil {
				t.Fatal(err)
			}
			want := strings.Join(lines, "\n") + "\n"
			cmd := cli.ValidateSignaturesCommand(inFile, false, tt.full, conf)
			var out bytes.Buffer
			if err := runCommandWithOutput(t.Context(), cmd, &out, cli.Dispatch, false); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); signatureOutputSummary(got) != signatureOutputSummary(want) {
				t.Fatalf("CLI output diverged from API output:\ngot:\n%q\nwant:\n%q", got, want)
			}
		})
	}
}
