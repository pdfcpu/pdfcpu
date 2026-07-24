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

package pkcs7_test

import (
	"bytes"
	"crypto/x509"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/pkcs7"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/sign"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestParserFailureRemainsError locks the package boundary: malformed PKCS#7
// input is returned to the domain caller as an error.
func TestParserFailureRemainsError(t *testing.T) {
	if _, err := pkcs7.Parse([]byte{0x30, 0x01}); err == nil {
		t.Fatal("expected malformed PKCS#7 error")
	}
}

// TestConstructionFailureRemainsError locks the package boundary: PKCS#7
// construction failures are operational errors.
func TestConstructionFailureRemainsError(t *testing.T) {
	sd, err := pkcs7.NewSignedData()
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1)}
	err = sd.AddSigner(cert, struct{}{}, nil, pkcs7.OIDDigestAlgorithmSHA256, pkcs7.SignerInfoConfig{})
	if err == nil {
		t.Fatal("expected unsupported signing-key error")
	}
}

// TestSignatureValidationConvertsParserFailureToEvidence locks the signature
// contract: malformed signature contents are Problems, not a fatal domain error.
func TestSignatureValidationConvertsParserFailureToEvidence(t *testing.T) {
	malformed := []byte{0x30, 0x01}
	sigDict := types.Dict{
		"Contents": types.HexLiteral(hex.EncodeToString(malformed)),
	}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	ctx := &model.Context{Configuration: model.NewDefaultConfiguration()}

	err := sign.ValidatePKCS7Signatures(
		bytes.NewReader(nil),
		sigDict,
		false,
		false,
		false,
		0,
		x509.NewCertPool(),
		result,
		ctx,
	)
	if err != nil {
		t.Fatalf("expected reportable evidence, got fatal error %v", err)
	}
	if len(result.Problems) == 0 {
		t.Fatal("expected malformed PKCS#7 problem")
	}
	if problems := strings.Join(result.Problems, "\n"); !strings.Contains(problems, "parse PKCS#7") {
		t.Fatalf("got problems %q, want PKCS#7 parse evidence", problems)
	}
}
