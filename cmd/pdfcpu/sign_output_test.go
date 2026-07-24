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
	"testing"

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

// TestValidateSignaturesCLIOutput locks down CLI presentation of observed
// signature, certificate, timestamp and revocation evidence together with the
// configuration-dependent local assessment.
func TestValidateSignaturesCLIOutput(t *testing.T) {
	useEmptyCLISignatureTrustStore(t)
	tests := []struct {
		name string
		file string
		want string
	}{
		{
			name: "ETSI.CAdES.detached",
			file: filepath.Join("..", "..", "pkg", "samples", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf"),
			want: `
1 form signature (authoritative, visible, signed) on page 1
   Status: validity of the signature is unknown
   Reason: signer's certificate or one of its parent certificates has expired
   Signed: 2024-03-04 14:25:54 +0200
`,
		},
		{
			name: "adbe.pkcs7.detached",
			file: filepath.Join("..", "..", "pkg", "samples", "signatures", "adbe.pkcs7.detached", "sample1.pdf"),
			want: `
1 form signature (authoritative, visible, signed) on page 1
   Status: validity of the signature is unknown
   Reason: signer's certificate or one of its parent certificates has expired
   Signed: 2009-07-16 10:47:47 -0400
`,
		},
		{
			name: "adbe.x509.rsa_sha1",
			file: filepath.Join("..", "..", "pkg", "samples", "signatures", "adbe.x509.rsa_sha1", "sample01.pdf"),
			want: `
1 form signature (authoritative, visible, signed) on page 1
   Status: validity of the signature is unknown
   Reason: signer's certificate is invalid
   Signed: 2009-10-02 00:11:31 +0500
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := model.NewDefaultConfiguration()
			conf.Offline = true
			cmd := cli.ValidateSignaturesCommand(tt.file, false, false, conf)
			var out bytes.Buffer
			if err := runCommandWithOutput(cmd, &out, cli.Dispatch, false); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tt.want {
				t.Fatalf("unexpected CLI output:\ngot:\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}
