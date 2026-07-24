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
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type usageRightsSignatureFixture struct {
	data    []byte
	sigDict types.Dict
	roots   *x509.CertPool
}

type usageRightsReadError struct {
	err error
}

func (r usageRightsReadError) ReadAt([]byte, int64) (int, error) {
	return 0, r.err
}

func newUsageRightsSignatureFixture(t *testing.T) usageRightsSignatureFixture {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Usage Rights Signer"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("usage-rights signed data")
	digest := sha1.Sum(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA1, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	contents, err := asn1.Marshal(signature)
	if err != nil {
		t.Fatal(err)
	}
	contentsLiteral := types.HexLiteral(hex.EncodeToString(contents))
	file := append(append([]byte(nil), data...), []byte(contentsLiteral.String())...)
	roots := x509.NewCertPool()
	roots.AddCert(cert)

	return usageRightsSignatureFixture{
		data: file,
		sigDict: types.Dict{
			"SubFilter": types.Name("adbe.x509.rsa_sha1"),
			"ByteRange": types.Array{
				types.Integer(0),
				types.Integer(len(data)),
				types.Integer(len(file)),
				types.Integer(0),
			},
			"Cert":     types.Array{types.HexLiteral(hex.EncodeToString(der))},
			"Contents": contentsLiteral,
		},
		roots: roots,
	}
}

func useTestCertificatePool(t *testing.T, pool *x509.CertPool) {
	t.Helper()

	trustedCertificatePool.Lock()
	oldPool := trustedCertificatePool.pool
	trustedCertificatePool.pool = pool
	trustedCertificatePool.Unlock()
	t.Cleanup(func() {
		trustedCertificatePool.Lock()
		trustedCertificatePool.pool = oldPool
		trustedCertificatePool.Unlock()
	})
}

// pdfcpu reports observed signature, certificate, timestamp and revocation
// evidence together with a configuration-dependent local assessment, using
// this fatal-versus-evidence contract:
//   - A construction failure that prevents inspecting a signature is fatal and
//     returns a wrapped error with signature object, dictionary, or entry context.
//   - A completed inspection that finds an invalid signature, certificate,
//     digest, timestamp, trust chain, or revocation status is evidence. It returns
//     a SignatureValidationResult with Problems instead of a fatal error.
//
// Fatal errors stop the API operation. Evidence remains available to callers for
// reporting and must never be promoted to a panic or an opaque fatal error.

// TestSignatureValidationFatalVersusEvidenceContract locks down
// evidence-reporting and local-assessment contract.
func TestSignatureValidationFatalVersusEvidenceContract(t *testing.T) {
	t.Run("ConstructionFailureIsFatal", func(t *testing.T) {
		cause := errors.New("decode signature field")
		osd := &types.ObjectStreamDict{
			StreamDict: types.StreamDict{Content: []byte("signature field")},
		}
		lazy := types.NewLazyObjectStreamObject(osd, 0, -1, func(context.Context, string) (types.Object, error) {
			return nil, cause
		})
		ctx := &model.Context{
			XRefTable: &model.XRefTable{
				Table: map[int]*model.XRefTableEntry{
					7: model.NewXRefTableEntryGen0(lazy),
				},
				Signatures: map[int]map[int]model.Signature{
					0: {
						7: {ObjNr: 7},
					},
				},
			},
		}

		results, err := ValidateSignatures(bytes.NewReader(nil), ctx, false)
		if !errors.Is(err, cause) {
			t.Fatalf("expected fatal construction cause %v, got results %v and error %v", cause, results, err)
		}
		for _, want := range []string{"signature obj#7", "signature field dict", "dereference"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("expected fatal context %q, got %q", want, err)
			}
		}
	})

	t.Run("InspectableFailureIsEvidence", func(t *testing.T) {
		ctx := &model.Context{
			XRefTable: &model.XRefTable{
				Table: map[int]*model.XRefTableEntry{
					7: model.NewXRefTableEntryGen0(types.Dict{
						"V": *types.NewIndirectRef(9, 0),
					}),
					9: model.NewXRefTableEntryGen0(types.Dict{}),
				},
				Signatures: map[int]map[int]model.Signature{
					0: {
						7: {ObjNr: 7},
					},
				},
			},
		}

		results, err := ValidateSignatures(bytes.NewReader(nil), ctx, false)
		if err != nil {
			t.Fatalf("expected reportable evidence, got fatal error %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("got %d results, want 1", len(results))
		}
		if got := strings.Join(results[0].Problems, "\n"); !strings.Contains(got, "signature dict entry SubFilter: malformed: missing") {
			t.Fatalf("expected malformed signature evidence, got %q", got)
		}
	})
}

// TestValidateSignaturesOrdersSameIncrementByObjectNumber verifies map
// iteration cannot change result order for signatures in one increment.
func TestValidateSignaturesOrdersSameIncrementByObjectNumber(t *testing.T) {
	ctx := sameIncrementSignatureContext()
	for i := 0; i < 32; i++ {
		results, err := ValidateSignatures(bytes.NewReader(nil), ctx, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 2 || results[0].ObjNr != 5 || results[1].ObjNr != 7 {
			t.Fatalf("run %d: got signature order %v, want object numbers 5, 7", i+1, signatureObjectNumbers(results))
		}
	}
}

// TestValidateSignaturesSelectsDeterministicAuthoritativeSignature verifies the
// lowest object number is consistently selected first within one increment.
func TestValidateSignaturesSelectsDeterministicAuthoritativeSignature(t *testing.T) {
	results, err := ValidateSignatures(bytes.NewReader(nil), sameIncrementSignatureContext(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 authoritative result", len(results))
	}
	if results[0].ObjNr != 5 || !results[0].Authoritative {
		t.Fatalf("got signature obj#%d authoritative=%t, want obj#5 authoritative", results[0].ObjNr, results[0].Authoritative)
	}
}

func sameIncrementSignatureContext() *model.Context {
	sigDict := func() types.Dict {
		return types.Dict{"SubFilter": types.Name("unsupported")}
	}
	return &model.Context{
		XRefTable: &model.XRefTable{
			Table: map[int]*model.XRefTableEntry{
				5:  model.NewXRefTableEntryGen0(types.Dict{"V": *types.NewIndirectRef(15, 0)}),
				7:  model.NewXRefTableEntryGen0(types.Dict{"V": *types.NewIndirectRef(17, 0)}),
				15: model.NewXRefTableEntryGen0(sigDict()),
				17: model.NewXRefTableEntryGen0(sigDict()),
			},
			Signatures: map[int]map[int]model.Signature{
				0: {
					7: {Type: model.SigTypeForm, ObjNr: 7, Signed: true},
					5: {Type: model.SigTypeForm, ObjNr: 5, Signed: true},
				},
			},
		},
	}
}

func signatureObjectNumbers(results []*model.SignatureValidationResult) []int {
	objNrs := make([]int, 0, len(results))
	for _, result := range results {
		objNrs = append(objNrs, result.ObjNr)
	}
	return objNrs
}

type signatureResultCharacterization struct {
	Type           int
	Status         model.SignatureStatus
	Reason         model.SignatureReason
	DocModified    int
	SubFilter      string
	SignerIdentity string
	Problems       []string
}

func characterizeSignatureResult(result *model.SignatureValidationResult) signatureResultCharacterization {
	return signatureResultCharacterization{
		Type:           result.Type,
		Status:         result.Status,
		Reason:         result.Reason,
		DocModified:    result.DocModified,
		SubFilter:      result.Details.SubFilter,
		SignerIdentity: result.Details.SignerIdentity,
		Problems:       append([]string(nil), result.Problems...),
	}
}

// TestSignatureResultFieldsAndKindsCharacterization locks the result fields
// shared by usage-rights and ordinary signatures for the same inspectable
// malformed signature dictionary. Both paths report evidence without a fatal
// error; only the signature kind differs.
func TestSignatureResultFieldsAndKindsCharacterization(t *testing.T) {
	const problem = "signature dict entry SubFilter: malformed: missing"
	ctx := &model.Context{
		XRefTable: &model.XRefTable{
			Table: map[int]*model.XRefTableEntry{
				7: model.NewXRefTableEntryGen0(types.Dict{
					"V": *types.NewIndirectRef(9, 0),
				}),
				9: model.NewXRefTableEntryGen0(types.Dict{}),
			},
		},
	}
	ordinary, err := validateSignature(
		model.Signature{
			Type:    model.SigTypeForm,
			Visible: true,
			Signed:  true,
			ObjNr:   7,
			PageNr:  1,
		},
		ctx,
		bytes.NewReader(nil),
		true,
		false,
		0,
	)
	if err != nil {
		t.Fatalf("ordinary signature: %v", err)
	}
	usageRights, err := validateURSignature(
		types.Dict{},
		0,
		ctx,
		bytes.NewReader(nil),
	)
	if err != nil {
		t.Fatalf("usage-rights signature: %v", err)
	}

	tests := []struct {
		name   string
		result *model.SignatureValidationResult
		want   signatureResultCharacterization
	}{
		{
			name:   "Ordinary",
			result: ordinary,
			want: signatureResultCharacterization{
				Type:           model.SigTypeForm,
				Status:         model.SignatureStatusUnknown,
				Reason:         model.SignatureReasonMalformed,
				DocModified:    model.Unknown,
				SignerIdentity: "Unknown",
				Problems:       []string{problem},
			},
		},
		{
			name:   "UsageRights",
			result: usageRights,
			want: signatureResultCharacterization{
				Type:           model.SigTypeUR,
				Status:         model.SignatureStatusUnknown,
				Reason:         model.SignatureReasonMalformed,
				DocModified:    model.Unknown,
				SignerIdentity: "Unknown",
				Problems:       []string{problem},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := characterizeSignatureResult(tt.result); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("signature result changed:\ngot:  %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}

// TestUsageRightsUsesDirectSignatureEvidencePath verifies usage-rights and
// ordinary signatures report the same observed evidence and
// configuration-dependent local assessment through direct helpers.
func TestUsageRightsUsesDirectSignatureEvidencePath(t *testing.T) {
	fixture := newUsageRightsSignatureFixture(t)
	useTestCertificatePool(t, fixture.roots)
	ctx := &model.Context{
		Configuration: model.NewDefaultConfiguration(),
		XRefTable:     &model.XRefTable{},
	}

	usageRights, err := validateURSignature(fixture.sigDict, 0, ctx, bytes.NewReader(fixture.data))
	if err != nil {
		t.Fatalf("usage-rights signature: %v", err)
	}

	ordinary := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}
	handler := sigHandler("adbe.x509.rsa_sha1")
	if err := handler(
		bytes.NewReader(fixture.data),
		fixture.sigDict,
		false,
		false,
		true,
		0,
		fixture.roots,
		ordinary,
		ctx,
	); err != nil {
		t.Fatalf("ordinary signature handler: %v", err)
	}

	if usageRights.Status != ordinary.Status ||
		usageRights.Reason != ordinary.Reason ||
		usageRights.DocModified != ordinary.DocModified {
		t.Fatalf(
			"usage-rights assessment differs: got status=%s reason=%s modified=%d; want status=%s reason=%s modified=%d",
			usageRights.Status,
			usageRights.Reason,
			usageRights.DocModified,
			ordinary.Status,
			ordinary.Reason,
			ordinary.DocModified,
		)
	}
	assertDirectCertificateEvidence(t, usageRights)
	assertDirectCertificateEvidence(t, ordinary)
}

// TestUsageRightsRevisionReportingUsesCachedIncrement verifies the top-level
// dispatcher applies historical reporting to usage-rights signatures.
func TestUsageRightsRevisionReportingUsesCachedIncrement(t *testing.T) {
	fixture := newUsageRightsSignatureFixture(t)
	useTestCertificatePool(t, fixture.roots)
	file := append(append([]byte(nil), fixture.data...), []byte("later increment")...)
	ctx := &model.Context{
		Configuration: model.NewDefaultConfiguration(),
		Read:          &model.ReadContext{FileSize: int64(len(file))},
		XRefTable:     &model.XRefTable{},
	}
	ctx.URSignature = fixture.sigDict
	ctx.URSignatureIncrement = 1

	results, err := ValidateSignatures(bytes.NewReader(file), ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want one usage-rights signature", len(results))
	}
	result := results[0]
	if result.Type != model.SigTypeUR ||
		result.Status != model.SignatureStatusUnknown ||
		result.Reason != model.SignatureReasonUnknown ||
		result.DocModified != model.Unknown {
		t.Fatalf("unexpected historical usage-rights result: %+v", result)
	}
}

// TestUsageRightsCurrentRevisionRejectsUnsignedSuffix verifies usage-rights
// signatures use the same current-revision boundary check as ordinary ones.
func TestUsageRightsCurrentRevisionRejectsUnsignedSuffix(t *testing.T) {
	fixture := newUsageRightsSignatureFixture(t)
	file := append(append([]byte(nil), fixture.data...), []byte("unsigned suffix")...)
	ctx := &model.Context{
		Configuration: model.NewDefaultConfiguration(),
		Read:          &model.ReadContext{FileSize: int64(len(file))},
		XRefTable:     &model.XRefTable{},
	}

	result, err := validateURSignature(
		fixture.sigDict,
		0,
		ctx,
		bytes.NewReader(file),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != model.SignatureReasonMalformed ||
		!strings.Contains(strings.Join(result.Problems, "\n"), "signed revision boundary mismatch") {
		t.Fatalf("unsigned usage-rights suffix was not rejected: %+v", result)
	}
}

// TestUsageRightsContentsGapIsEvidence verifies malformed usage-rights
// Contents exclusion remains reportable rather than fatal.
func TestUsageRightsContentsGapIsEvidence(t *testing.T) {
	fixture := newUsageRightsSignatureFixture(t)
	file := append([]byte(nil), fixture.data...)
	byteRange := fixture.sigDict.ArrayEntry("ByteRange")
	gapStart := int(byteRange[1].(types.Integer))
	file[gapStart] = '['
	ctx := &model.Context{
		Configuration: model.NewDefaultConfiguration(),
		Read:          &model.ReadContext{FileSize: int64(len(file))},
		XRefTable:     &model.XRefTable{},
	}

	result, err := validateURSignature(
		fixture.sigDict,
		0,
		ctx,
		bytes.NewReader(file),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(
		strings.Join(result.Problems, "\n"),
		"excluded gap does not match signature dict entry Contents",
	) {
		t.Fatalf("malformed usage-rights gap was not evidence: %+v", result)
	}
}

// TestUsageRightsByteRangeParsingIsEvidence verifies malformed usage-rights
// ByteRange arrays use the ordinary signature evidence path.
func TestUsageRightsByteRangeParsingIsEvidence(t *testing.T) {
	fixture := newUsageRightsSignatureFixture(t)
	sigDict := fixture.sigDict.Clone().(types.Dict)
	sigDict["ByteRange"] = types.Array{
		types.Integer(0),
		types.Integer(1),
		types.Integer(2),
	}
	ctx := &model.Context{
		Configuration: model.NewDefaultConfiguration(),
		Read:          &model.ReadContext{FileSize: int64(len(fixture.data))},
		XRefTable:     &model.XRefTable{},
	}

	result, err := validateURSignature(
		sigDict,
		0,
		ctx,
		bytes.NewReader(fixture.data),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(
		strings.Join(result.Problems, "\n"),
		"signature dict entry ByteRange: missing or invalid length",
	) {
		t.Fatalf("malformed usage-rights ByteRange was not evidence: %+v", result)
	}
}

// TestUsageRightsFatalPositionalReadPreservesCause verifies usage-rights
// positional I/O failures remain fatal and discoverable.
func TestUsageRightsFatalPositionalReadPreservesCause(t *testing.T) {
	fixture := newUsageRightsSignatureFixture(t)
	cause := errors.New("usage-rights positional read failed")
	ctx := &model.Context{
		Configuration: model.NewDefaultConfiguration(),
		Read:          &model.ReadContext{FileSize: int64(len(fixture.data))},
		XRefTable:     &model.XRefTable{},
	}

	_, err := validateURSignature(
		fixture.sigDict,
		0,
		ctx,
		usageRightsReadError{err: cause},
	)
	if !errors.Is(err, cause) {
		t.Fatalf("got %v, want positional cause %v", err, cause)
	}
	for _, want := range []string{
		"signature dict entry SubFilter adbe.x509.rsa_sha1",
		"read signed data",
		"signature dict entry ByteRange",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("fatal usage-rights error missing %q: %v", want, err)
		}
	}
}

func assertDirectCertificateEvidence(t *testing.T, result *model.SignatureValidationResult) {
	t.Helper()

	if len(result.Details.Signers) != 1 {
		t.Fatalf("got %d signers, want 1", len(result.Details.Signers))
	}
	certificate := result.Details.Signers[0].Certificate
	if certificate == nil {
		t.Fatal("missing certificate evidence")
	}
	if certificate.Subject != "Usage Rights Signer" {
		t.Fatalf("got certificate subject %q", certificate.Subject)
	}
	if certificate.PathEvidence.AssessmentScope != model.AssessmentScopeLocal {
		t.Fatalf("got assessment scope %d, want local", certificate.PathEvidence.AssessmentScope)
	}
}

// TestSignedRevisionBoundaryEvidence distinguishes a current unsigned suffix
// from a later incremental update and requires document timestamps to end at
// the current file boundary.
func TestSignedRevisionBoundaryEvidence(t *testing.T) {
	const file = "HEAD<aa>TAIL"
	tests := []struct {
		name      string
		sigType   int
		increment int
		subFilter string
		want      string
		notWant   string
	}{
		{
			name:      "UnsignedSuffixInCurrentRevision",
			sigType:   model.SigTypeForm,
			subFilter: "adbe.pkcs7.detached",
			increment: 0,
			want:      "signed revision boundary mismatch",
		},
		{
			name:      "LaterIncrementSuffix",
			sigType:   model.SigTypeForm,
			subFilter: "unsupported",
			increment: 1,
			want:      "unsupported: value unsupported",
			notWant:   "signed revision boundary mismatch",
		},
		{
			name:      "DocumentTimestampMustEndAtCurrentFileEnd",
			sigType:   model.SigTypeDTS,
			subFilter: "ETSI.RFC3161",
			increment: 1,
			want:      "SubFilter ETSI.RFC3161: read signed data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := signedRevisionBoundaryContext(int64(len(file)), tt.subFilter)
			result, err := validateSignature(
				model.Signature{Type: tt.sigType, ObjNr: 7},
				ctx,
				bytes.NewReader([]byte(file)),
				true,
				false,
				tt.increment,
			)
			if err != nil {
				t.Fatalf("expected reportable boundary evidence, got %v", err)
			}
			problems := strings.Join(result.Problems, "\n")
			if !strings.Contains(problems, tt.want) {
				t.Fatalf("got problems %q, want %q", problems, tt.want)
			}
			if tt.notWant != "" && strings.Contains(problems, tt.notWant) {
				t.Fatalf("got problems %q, do not want %q", problems, tt.notWant)
			}
		})
	}
}

// TestHistoricalSignatureReportingPreservesCryptographicEvidence verifies a
// signature followed by later increments retains successful cryptographic and
// local certificate evidence without claiming local validity or that the
// current document is unmodified.
func TestHistoricalSignatureReportingPreservesCryptographicEvidence(t *testing.T) {
	fixture := newUsageRightsSignatureFixture(t)
	useTestCertificatePool(t, fixture.roots)

	tests := []struct {
		name          string
		increment     int
		wantReason    model.SignatureReason
		wantModified  int
		wantPathState int
	}{
		{
			name:          "CurrentRevision",
			wantReason:    model.SignatureReasonUnknown,
			wantModified:  model.False,
			wantPathState: model.True,
		},
		{
			name:          "HistoricalRevision",
			increment:     1,
			wantReason:    model.SignatureReasonUnknown,
			wantModified:  model.Unknown,
			wantPathState: model.True,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := historicalSignatureContext(fixture)
			result, err := validateSignature(
				model.Signature{
					Type:   model.SigTypeForm,
					ObjNr:  7,
					Signed: true,
				},
				ctx,
				bytes.NewReader(fixture.data),
				true,
				true,
				tt.increment,
			)
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != model.SignatureStatusUnknown ||
				result.Reason != tt.wantReason ||
				result.DocModified != tt.wantModified {
				t.Fatalf(
					"got status=%s reason=%s modified=%d, want unknown, %s, %d",
					result.Status,
					result.Reason,
					result.DocModified,
					tt.wantReason,
					tt.wantModified,
				)
			}
			if len(result.Problems) != 0 ||
				len(result.Details.Signers) != 1 ||
				result.Details.Signers[0].CertificatePathStatus != tt.wantPathState {
				t.Fatalf("signed-byte or certificate evidence was lost: %+v", result)
			}
		})
	}
}

// TestHistoricalRevisionReportingRetainsFailureEvidence verifies normalizing
// the whole-document state does not erase a cryptographic mismatch.
func TestHistoricalRevisionReportingRetainsFailureEvidence(t *testing.T) {
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusInvalid,
		Reason:      model.SignatureReasonSignatureForged,
		DocModified: model.True,
		Problems:    []string{"cryptographic signature mismatch"},
	}

	applyHistoricalRevisionReporting(1, model.SigTypeForm, result)

	if result.Status != model.SignatureStatusInvalid ||
		result.Reason != model.SignatureReasonSignatureForged ||
		result.DocModified != model.True ||
		len(result.Problems) != 1 {
		t.Fatalf("historical mismatch evidence changed: %+v", result)
	}
}

// TestHistoricalRevisionReportingPreservesDetectedModification verifies an
// older ordinary or usage-rights signature retains an established document
// modification conclusion.
func TestHistoricalRevisionReportingPreservesDetectedModification(t *testing.T) {
	for _, signatureType := range []int{model.SigTypeForm, model.SigTypeUR} {
		result := &model.SignatureValidationResult{
			Status:      model.SignatureStatusInvalid,
			Reason:      model.SignatureReasonDocModified,
			DocModified: model.True,
		}

		applyHistoricalRevisionReporting(1, signatureType, result)

		if result.DocModified != model.True ||
			result.Reason != model.SignatureReasonDocModified {
			t.Fatalf("signature type %d lost modification evidence: %+v", signatureType, result)
		}
	}
}

// TestDTSRevisionReportingRemainsStrict verifies DTS ByteRanges retain their
// current-document conclusion regardless of increment metadata.
func TestDTSRevisionReportingRemainsStrict(t *testing.T) {
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusValid,
		Reason:      model.SignatureReasonDocNotModified,
		DocModified: model.False,
	}

	applyHistoricalRevisionReporting(1, model.SigTypeDTS, result)

	if result.Status != model.SignatureStatusValid ||
		result.Reason != model.SignatureReasonDocNotModified ||
		result.DocModified != model.False {
		t.Fatalf("DTS strict conclusion changed: %+v", result)
	}
}

func historicalSignatureContext(fixture usageRightsSignatureFixture) *model.Context {
	return &model.Context{
		Configuration: model.NewDefaultConfiguration(),
		Read:          &model.ReadContext{FileSize: int64(len(fixture.data))},
		XRefTable: &model.XRefTable{
			Table: map[int]*model.XRefTableEntry{
				7: model.NewXRefTableEntryGen0(types.Dict{
					"V": *types.NewIndirectRef(9, 0),
				}),
				9: model.NewXRefTableEntryGen0(fixture.sigDict),
			},
		},
	}
}

func signedRevisionBoundaryContext(fileSize int64, subFilter string) *model.Context {
	sigDict := types.Dict{
		"SubFilter": types.Name(subFilter),
		"ByteRange": types.Array{
			types.Integer(0),
			types.Integer(4),
			types.Integer(8),
			types.Integer(3),
		},
		"Contents": types.HexLiteral("aa"),
	}
	return &model.Context{
		Read: &model.ReadContext{FileSize: fileSize},
		XRefTable: &model.XRefTable{
			Table: map[int]*model.XRefTableEntry{
				7: model.NewXRefTableEntryGen0(types.Dict{
					"V": *types.NewIndirectRef(9, 0),
				}),
				9: model.NewXRefTableEntryGen0(sigDict),
			},
		},
	}
}

// TestValidateSignaturesAddsSignatureObjectContext verifies fatal field errors identify the signature object.
func TestValidateSignaturesAddsSignatureObjectContext(t *testing.T) {
	cause := errors.New("decode signature field")
	osd := &types.ObjectStreamDict{
		StreamDict: types.StreamDict{Content: []byte("signature field")},
	}
	lazy := types.NewLazyObjectStreamObject(osd, 0, -1, func(context.Context, string) (types.Object, error) {
		return nil, cause
	})
	ctx := &model.Context{
		XRefTable: &model.XRefTable{
			Table: map[int]*model.XRefTableEntry{
				7: model.NewXRefTableEntryGen0(lazy),
			},
			Signatures: map[int]map[int]model.Signature{
				0: {
					7: {ObjNr: 7},
				},
			},
		},
	}

	_, err := ValidateSignatures(bytes.NewReader(nil), ctx, false)
	if !errors.Is(err, cause) {
		t.Fatalf("expected %v, got %v", cause, err)
	}
	for _, want := range []string{"signature obj#7", "signature field dict", "dereference"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q, got %q", want, err)
		}
	}
}

// TestValidateSignaturesAddsSignatureDictionaryObjectContext verifies fatal dictionary dereferences identify both objects.
func TestValidateSignaturesAddsSignatureDictionaryObjectContext(t *testing.T) {
	cause := errors.New("decode signature dictionary")
	osd := &types.ObjectStreamDict{
		StreamDict: types.StreamDict{Content: []byte("signature dictionary")},
	}
	lazy := types.NewLazyObjectStreamObject(osd, 0, -1, func(context.Context, string) (types.Object, error) {
		return nil, cause
	})
	ctx := &model.Context{
		XRefTable: &model.XRefTable{
			Table: map[int]*model.XRefTableEntry{
				7: model.NewXRefTableEntryGen0(types.Dict{
					"V": *types.NewIndirectRef(9, 0),
				}),
				9: model.NewXRefTableEntryGen0(lazy),
			},
			Signatures: map[int]map[int]model.Signature{
				0: {
					7: {ObjNr: 7},
				},
			},
		},
	}

	_, err := ValidateSignatures(bytes.NewReader(nil), ctx, false)
	if !errors.Is(err, cause) {
		t.Fatalf("expected %v, got %v", cause, err)
	}
	for _, want := range []string{"signature obj#7", "signature dict obj#9", "dereference"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q, got %q", want, err)
		}
	}
}

// TestValidateSignaturesReportsUsageRightsDetailEvidence verifies optional signature details remain reportable evidence.
func TestValidateSignaturesReportsUsageRightsDetailEvidence(t *testing.T) {
	ctx := &model.Context{
		XRefTable: &model.XRefTable{
			URSignature: types.Dict{
				"M": types.Integer(1),
			},
		},
	}

	results, err := ValidateSignatures(bytes.NewReader(nil), ctx, false)
	if err != nil {
		t.Fatalf("expected signature detail evidence, got fatal error %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	problems := strings.Join(results[0].Problems, "\n")
	for _, want := range []string{"signature dict entry M", "signature dict entry SubFilter: malformed: missing"} {
		if !strings.Contains(problems, want) {
			t.Errorf("expected problem %q, got %q", want, problems)
		}
	}
}

// TestValidateURSignatureNormalizesSubFilterProblems verifies PDF entry wording is stable.
func TestValidateURSignatureNormalizesSubFilterProblems(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{}}
	tests := []struct {
		name       string
		sigDict    types.Dict
		want       string
		wantReason model.SignatureReason
	}{
		{
			name:       "missing",
			sigDict:    types.Dict{},
			want:       "signature dict entry SubFilter: malformed: missing",
			wantReason: model.SignatureReasonMalformed,
		},
		{
			name: "unsupported",
			sigDict: types.Dict{
				"SubFilter": types.Name("unsupported"),
			},
			want:       "signature dict entry SubFilter: unsupported: value unsupported",
			wantReason: model.SignatureReasonUnsupported,
		},
		{
			name: "unsupported usage-rights timestamp",
			sigDict: types.Dict{
				"SubFilter": types.Name("ETSI.RFC3161"),
			},
			want:       "signature dict entry SubFilter: unsupported: value ETSI.RFC3161",
			wantReason: model.SignatureReasonUnsupported,
		},
	}

	for _, tt := range tests {
		result, err := validateURSignature(
			tt.sigDict,
			0,
			ctx,
			bytes.NewReader(nil),
		)
		if err != nil {
			t.Errorf("%s: expected evidence, got fatal error %v", tt.name, err)
			continue
		}
		if len(result.Problems) != 1 || result.Problems[0] != tt.want {
			t.Errorf("%s: got problems %v, want %q", tt.name, result.Problems, tt.want)
		}
		if result.Status != model.SignatureStatusUnknown ||
			result.Reason != tt.wantReason ||
			result.DocModified != model.Unknown {
			t.Errorf(
				"%s: got status=%s reason=%s modified=%d, want unknown/%s/unknown",
				tt.name,
				result.Status,
				result.Reason,
				result.DocModified,
				tt.wantReason,
			)
		}
	}
}

// TestSubFilterEvidenceIsConsistentAcrossSignatureKinds verifies missing,
// malformed and unsupported SubFilter values remain reportable evidence for
// ordinary and usage-rights signatures without attempting positional reads.
func TestSubFilterEvidenceIsConsistentAcrossSignatureKinds(t *testing.T) {
	tests := []struct {
		name       string
		sigDict    types.Dict
		want       string
		wantReason model.SignatureReason
	}{
		{
			name:       "Missing",
			sigDict:    types.Dict{},
			want:       "signature dict entry SubFilter: malformed: missing",
			wantReason: model.SignatureReasonMalformed,
		},
		{
			name: "Malformed",
			sigDict: types.Dict{
				"SubFilter": types.Integer(1),
			},
			want:       "signature dict entry SubFilter: malformed: expected name, got types.Integer",
			wantReason: model.SignatureReasonMalformed,
		},
		{
			name: "Unsupported",
			sigDict: types.Dict{
				"SubFilter": types.Name("unsupported"),
			},
			want:       "signature dict entry SubFilter: unsupported: value unsupported",
			wantReason: model.SignatureReasonUnsupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cause := errors.New("SubFilter evidence attempted a positional read")
			ordinaryCtx := signatureSubFilterContext(tt.sigDict)
			ordinary, err := validateSignature(
				model.Signature{
					Type:   model.SigTypeForm,
					Signed: true,
					ObjNr:  7,
				},
				ordinaryCtx,
				usageRightsReadError{err: cause},
				true,
				false,
				0,
			)
			if err != nil {
				t.Fatalf("ordinary signature: %v", err)
			}

			usageRights, err := validateURSignature(
				tt.sigDict,
				0,
				&model.Context{XRefTable: &model.XRefTable{}},
				usageRightsReadError{err: cause},
			)
			if err != nil {
				t.Fatalf("usage-rights signature: %v", err)
			}

			for _, result := range []*model.SignatureValidationResult{ordinary, usageRights} {
				if result.Status != model.SignatureStatusUnknown ||
					result.Reason != tt.wantReason ||
					result.DocModified != model.Unknown {
					t.Fatalf("unexpected SubFilter conclusion: %+v", result)
				}
				if len(result.Problems) != 1 || result.Problems[0] != tt.want {
					t.Fatalf("got problems %v, want %q", result.Problems, tt.want)
				}
				if !strings.Contains(result.Problems[0], "signature dict entry SubFilter") {
					t.Fatalf("missing dictionary and entry context: %v", result.Problems)
				}
			}
		})
	}
}

func signatureSubFilterContext(sigDict types.Dict) *model.Context {
	return &model.Context{
		XRefTable: &model.XRefTable{
			Table: map[int]*model.XRefTableEntry{
				7: model.NewXRefTableEntryGen0(types.Dict{
					"V": *types.NewIndirectRef(9, 0),
				}),
				9: model.NewXRefTableEntryGen0(sigDict),
			},
		},
	}
}

// TestFieldDetailsReportsEvidence verifies malformed optional field metadata does not abort signature inspection.
func TestFieldDetailsReportsEvidence(t *testing.T) {
	result := model.SignatureValidationResult{}
	fieldDetails(
		types.Dict{"T": types.StringLiteral(string([]byte{0xFE, 0xFF, 0xD8, 0x00}))},
		&result,
	)
	if len(result.Problems) != 1 {
		t.Fatalf("got %d problems, want 1", len(result.Problems))
	}
	if want := "signature field dict entry T"; !strings.Contains(result.Problems[0], want) {
		t.Errorf("expected %q, got %q", want, result.Problems[0])
	}
}

// TestSignatureDetailsReportsEvidence verifies malformed optional signature metadata remains reportable.
func TestSignatureDetailsReportsEvidence(t *testing.T) {
	result := model.SignatureValidationResult{}
	sigDict := types.Dict{
		"Name": types.StringLiteral(string([]byte{0xFE, 0xFF, 0xD8, 0x00})),
		"M":    types.Integer(1),
	}
	signatureDetails(sigDict, &model.Context{XRefTable: &model.XRefTable{}}, &result)
	problems := strings.Join(result.Problems, "\n")
	for _, want := range []string{"signature dict entry Name", "signature dict entry M"} {
		if !strings.Contains(problems, want) {
			t.Errorf("expected %q, got %q", want, problems)
		}
	}
}

// TestDetectPermissionsReportsReferenceEntryEvidence verifies malformed Reference containers remain nonfatal.
func TestDetectPermissionsReportsReferenceEntryEvidence(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{}}
	tests := []struct {
		name    string
		sigDict types.Dict
		want    string
	}{
		{
			name: "reference entry",
			sigDict: types.Dict{
				"Reference": types.Integer(1),
			},
			want: "signature dict entry Reference",
		},
		{
			name: "empty reference array",
			sigDict: types.Dict{
				"Reference": types.Array{},
			},
			want: "signature dict entry Reference: empty array",
		},
	}

	for _, tt := range tests {
		result := &model.SignatureValidationResult{}
		_, err := detectPermissions(tt.sigDict, ctx, result)
		if err != nil {
			t.Errorf("%s: expected evidence, got fatal error %v", tt.name, err)
		}
		if got := strings.Join(result.Problems, "\n"); !strings.Contains(got, tt.want) {
			t.Errorf("%s: expected %q, got %q", tt.name, tt.want, got)
		}
	}
}

// TestDetectPermissionsReportsReferenceDictionaryEvidence verifies malformed entries retain one-based context.
func TestDetectPermissionsReportsReferenceDictionaryEvidence(t *testing.T) {
	cause := errors.New("decode signature reference")
	osd := &types.ObjectStreamDict{
		StreamDict: types.StreamDict{Content: []byte("signature reference")},
	}
	lazy := types.NewLazyObjectStreamObject(osd, 0, -1, func(context.Context, string) (types.Object, error) {
		return nil, cause
	})
	ctx := &model.Context{
		XRefTable: &model.XRefTable{
			Table: map[int]*model.XRefTableEntry{
				11: model.NewXRefTableEntryGen0(lazy),
			},
		},
	}
	valid := types.Dict{
		"TransformMethod": types.Name("DocMDP"),
		"TransformParams": types.Dict{
			"Type": types.Name("TransformParams"),
			"P":    types.Integer(1),
		},
	}
	sigDict := types.Dict{
		"Reference": types.Array{
			*types.NewIndirectRef(11, 0),
			nil,
			valid,
		},
	}
	result := &model.SignatureValidationResult{}

	perms, err := detectPermissions(sigDict, ctx, result)
	if err != nil {
		t.Fatalf("expected reference evidence, got fatal error %v", err)
	}
	if perms != 1 {
		t.Fatalf("got permission %d, want 1 from later valid reference", perms)
	}
	problems := strings.Join(result.Problems, "\n")
	for _, want := range []string{
		"signature dict entry Reference, reference index 1: dereference dictionary",
		cause.Error(),
		"signature dict entry Reference, reference index 2: missing dictionary",
	} {
		if !strings.Contains(problems, want) {
			t.Errorf("expected %q, got %q", want, problems)
		}
	}
}

// TestValidateSignatureContinuesAfterReferenceEvidence verifies permission evidence does not stop SubFilter inspection.
func TestValidateSignatureContinuesAfterReferenceEvidence(t *testing.T) {
	ctx := &model.Context{
		XRefTable: &model.XRefTable{
			Table: map[int]*model.XRefTableEntry{
				7: model.NewXRefTableEntryGen0(types.Dict{
					"V": *types.NewIndirectRef(9, 0),
				}),
				9: model.NewXRefTableEntryGen0(types.Dict{
					"Reference": types.Integer(1),
					"SubFilter": types.Name("unsupported"),
				}),
			},
		},
	}

	result, err := validateSignature(model.Signature{ObjNr: 7}, ctx, bytes.NewReader(nil), true, false, 0)
	if err != nil {
		t.Fatalf("expected evidence, got fatal error %v", err)
	}
	problems := strings.Join(result.Problems, "\n")
	for _, want := range []string{
		"signature dict entry Reference: dereference array",
		"signature dict entry SubFilter: unsupported: value unsupported",
	} {
		if !strings.Contains(problems, want) {
			t.Errorf("expected %q, got %q", want, problems)
		}
	}
}

// TestDetectPermissionsReportsInvalidPermissionEvidence verifies semantic permission findings remain reportable.
func TestDetectPermissionsReportsInvalidPermissionEvidence(t *testing.T) {
	sigDict := types.Dict{
		"Reference": types.Array{
			types.Dict{
				"TransformMethod": types.Name("DocMDP"),
				"TransformParams": types.Dict{
					"Type": types.Name("TransformParams"),
					"P":    types.Integer(4),
				},
			},
		},
	}
	result := model.SignatureValidationResult{}
	_, err := detectPermissions(sigDict, &model.Context{XRefTable: &model.XRefTable{}}, &result)
	if err != nil {
		t.Fatalf("expected permission evidence, got fatal error %v", err)
	}
	if len(result.Problems) != 1 {
		t.Fatalf("got %d problems, want 1", len(result.Problems))
	}
	for _, want := range []string{
		"signature dict entry Reference",
		"reference index 1",
		"TransformParams dict entry P",
		"invalid DocMDP permission: 4",
	} {
		if !strings.Contains(result.Problems[0], want) {
			t.Errorf("expected %q, got %q", want, result.Problems[0])
		}
	}
}

// TestDetectPermissionsReportsTransformParamsEvidence verifies malformed parameter dictionaries do not stop later references.
func TestDetectPermissionsReportsTransformParamsEvidence(t *testing.T) {
	cause := errors.New("decode transform parameters")
	osd := &types.ObjectStreamDict{
		StreamDict: types.StreamDict{Content: []byte("transform parameters")},
	}
	lazy := types.NewLazyObjectStreamObject(osd, 0, -1, func(context.Context, string) (types.Object, error) {
		return nil, cause
	})
	ctx := &model.Context{
		XRefTable: &model.XRefTable{
			Table: map[int]*model.XRefTableEntry{
				21: model.NewXRefTableEntryGen0(lazy),
				22: model.NewXRefTableEntryGen0(types.Dict{
					"Type": types.Name("TransformParams"),
				}),
			},
		},
	}
	sigDict := types.Dict{
		"Reference": types.Array{
			types.Dict{
				"TransformMethod": types.Name("DocMDP"),
			},
			types.Dict{
				"TransformMethod": types.Name("DocMDP"),
				"TransformParams": *types.NewIndirectRef(21, 0),
			},
			types.Dict{
				"TransformMethod": types.Name("DocMDP"),
				"TransformParams": types.Dict{
					"Type": types.Name("WrongType"),
				},
			},
			types.Dict{
				"TransformMethod": types.Name("DocMDP"),
				"TransformParams": *types.NewIndirectRef(22, 0),
			},
		},
	}
	result := &model.SignatureValidationResult{}

	perms, err := detectPermissions(sigDict, ctx, result)
	if err != nil {
		t.Fatalf("expected TransformParams evidence, got fatal error %v", err)
	}
	if perms != 2 {
		t.Fatalf("got permission %d, want default 2 from later valid reference", perms)
	}
	problems := strings.Join(result.Problems, "\n")
	for _, want := range []string{
		"reference index 1, signature reference dict entry TransformParams: missing",
		"reference index 2, signature reference dict entry TransformParams: dereference dictionary",
		cause.Error(),
		"reference index 3, TransformParams dict entry Type: expected TransformParams",
	} {
		if !strings.Contains(problems, want) {
			t.Errorf("expected %q, got %q", want, problems)
		}
	}
}

// TestDetectPermissionsReportsPermissionEntryEvidence verifies malformed P entries retain context and allow recovery.
func TestDetectPermissionsReportsPermissionEntryEvidence(t *testing.T) {
	cause := errors.New("decode permission")
	osd := &types.ObjectStreamDict{
		StreamDict: types.StreamDict{Content: []byte("permission")},
	}
	lazy := types.NewLazyObjectStreamObject(osd, 0, -1, func(context.Context, string) (types.Object, error) {
		return nil, cause
	})
	ctx := &model.Context{
		XRefTable: &model.XRefTable{
			Table: map[int]*model.XRefTableEntry{
				31: model.NewXRefTableEntryGen0(lazy),
				32: model.NewXRefTableEntryGen0(types.Integer(1)),
			},
		},
	}
	params := func(p types.Object) types.Dict {
		return types.Dict{
			"Type": types.Name("TransformParams"),
			"P":    p,
		}
	}
	ref := func(p types.Object) types.Dict {
		return types.Dict{
			"TransformMethod": types.Name("DocMDP"),
			"TransformParams": params(p),
		}
	}
	sigDict := types.Dict{
		"Reference": types.Array{
			ref(types.StringLiteral("one")),
			ref(*types.NewIndirectRef(31, 0)),
			ref(nil),
			ref(types.Integer(4)),
			ref(*types.NewIndirectRef(32, 0)),
		},
	}
	result := &model.SignatureValidationResult{}

	perms, err := detectPermissions(sigDict, ctx, result)
	if err != nil {
		t.Fatalf("expected permission evidence, got fatal error %v", err)
	}
	if perms != 1 {
		t.Fatalf("got permission %d, want 1 from later valid reference", perms)
	}
	problems := strings.Join(result.Problems, "\n")
	for _, want := range []string{
		"reference index 1, TransformParams dict entry P: dereference integer",
		"reference index 2, TransformParams dict entry P: dereference integer",
		cause.Error(),
		"reference index 3, TransformParams dict entry P: missing integer",
		"reference index 4, TransformParams dict entry P: invalid DocMDP permission: 4",
	} {
		if !strings.Contains(problems, want) {
			t.Errorf("expected %q, got %q", want, problems)
		}
	}
}
