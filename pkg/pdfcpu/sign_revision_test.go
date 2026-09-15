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

package pdfcpu

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestSignatureReaderRevisionReporting verifies production ordinals preserve current and historical evidence.
func TestSignatureReaderRevisionReporting(t *testing.T) {
	fixture := newUsageRightsSignatureFixture(t)
	useTestCertificatePool(t, fixture.roots)
	for _, ordinal := range []int{1, 2, 4} {
		for _, usageRights := range []bool{false, true} {
			ctx := signedRevisionBoundaryContext(int64(len(fixture.data)), "adbe.x509.rsa_sha1")
			ctx.Configuration = model.NewDefaultConfiguration()
			ctx.Table[9] = model.NewXRefTableEntryGen0(fixture.sigDict)
			if usageRights {
				ctx.URSignature = fixture.sigDict
				ctx.URSignatureIncrement = ordinal
			} else {
				ctx.Signatures = map[int]map[int]model.Signature{
					ordinal: {7: {ObjNr: 7, Type: model.SigTypeForm, Signed: true}},
				}
			}
			results, err := ValidateSignatures(t.Context(), bytes.NewReader(fixture.data), ctx, true)
			if err != nil || len(results) != 1 {
				t.Fatalf("ordinal=%d usageRights=%t: results=%v error=%v", ordinal, usageRights, results, err)
			}
			want := model.Unknown
			if ordinal == 1 {
				want = model.False
			}
			if results[0].DocModified != want {
				t.Fatalf("ordinal=%d usageRights=%t: DocModified=%d, want %d", ordinal, usageRights, results[0].DocModified, want)
			}
		}
	}
}

// TestSignatureReaderCurrentBoundary verifies the dispatcher enforces coverage for reader ordinal 1.
func TestSignatureReaderCurrentBoundary(t *testing.T) {
	ctx := signedRevisionBoundaryContext(12, "adbe.pkcs7.detached")
	ctx.Signatures = map[int]map[int]model.Signature{
		1: {7: {ObjNr: 7, Type: model.SigTypeForm, Signed: true}},
	}
	results, err := ValidateSignatures(t.Context(), bytes.NewReader(nil), ctx, true)
	if err != nil || len(results) != 1 {
		t.Fatalf("results=%v error=%v", results, err)
	}
	if !strings.Contains(strings.Join(results[0].Problems, "\n"), "signed revision boundary mismatch") {
		t.Fatalf("missing current boundary evidence: %+v", results[0])
	}
}
