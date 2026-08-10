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

package sign

import (
	"bytes"
	"crypto"
	"crypto/dsa"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/pkcs7"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"golang.org/x/crypto/ocsp"
)

type malformedP7ContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

type malformedP7RawCertificates struct {
	Raw asn1.RawContent
}

type malformedP7SignedData struct {
	Version                    int                        `asn1:"default:1"`
	DigestAlgorithmIdentifiers []pkix.AlgorithmIdentifier `asn1:"set"`
	ContentInfo                malformedP7ContentInfo
	Certificates               malformedP7RawCertificates `asn1:"optional,tag:0"`
	SignerInfos                []asn1.RawValue            `asn1:"set"`
}

type signReadCloser struct {
	io.Reader
	closeErr error
	closed   bool
}

func (rc *signReadCloser) Close() error {
	rc.closed = true
	return rc.closeErr
}

type signErrorReader struct {
	err error
}

// TestSignatureResultStateTransitions verifies evidence categories have deterministic precedence.
func TestSignatureResultStateTransitions(t *testing.T) {
	t.Run("ReasonOnlyFillsUnknown", func(t *testing.T) {
		result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
		setResultReason(result, model.SignatureReasonInternal)
		setResultReason(result, model.SignatureReasonCertInvalid)
		if result.Reason != model.SignatureReasonInternal {
			t.Fatalf("got reason %s, want first evidence %s", result.Reason, model.SignatureReasonInternal)
		}
	})

	t.Run("DocumentUnmodifiedDoesNotOverwriteModification", func(t *testing.T) {
		result := &model.SignatureValidationResult{DocModified: model.Unknown}
		markDocumentUnmodified(result)
		if result.DocModified != model.False {
			t.Fatalf("got document state %d, want false", result.DocModified)
		}
		result.DocModified = model.True
		markDocumentUnmodified(result)
		if result.DocModified != model.True {
			t.Fatalf("document modification evidence was overwritten: %d", result.DocModified)
		}
	})

	tests := []struct {
		name          string
		initialStatus model.SignatureStatus
		initialReason model.SignatureReason
		initialDoc    int
		reason        model.SignatureReason
		docModified   int
		wantReason    model.SignatureReason
		wantDoc       int
	}{
		{
			name:          "MismatchOverridesNonInvalidEvidence",
			initialStatus: model.SignatureStatusUnknown,
			initialReason: model.SignatureReasonInternal,
			initialDoc:    model.Unknown,
			reason:        model.SignatureReasonDocModified,
			docModified:   model.True,
			wantReason:    model.SignatureReasonDocModified,
			wantDoc:       model.True,
		},
		{
			name:          "FirstInvalidReasonWins",
			initialStatus: model.SignatureStatusInvalid,
			initialReason: model.SignatureReasonSignatureForged,
			initialDoc:    model.Unknown,
			reason:        model.SignatureReasonDocModified,
			docModified:   model.True,
			wantReason:    model.SignatureReasonSignatureForged,
			wantDoc:       model.True,
		},
		{
			name:          "LaterUnknownDocumentStateDoesNotEraseFalse",
			initialStatus: model.SignatureStatusInvalid,
			initialReason: model.SignatureReasonDocModified,
			initialDoc:    model.False,
			reason:        model.SignatureReasonSignatureForged,
			docModified:   model.Unknown,
			wantReason:    model.SignatureReasonDocModified,
			wantDoc:       model.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &model.SignatureValidationResult{
				Status:      tt.initialStatus,
				Reason:      tt.initialReason,
				DocModified: tt.initialDoc,
			}
			markInvalidEvidence(result, tt.reason, tt.docModified)
			if result.Status != model.SignatureStatusInvalid {
				t.Fatalf("got status %s, want invalid", result.Status)
			}
			if result.Reason != tt.wantReason || result.DocModified != tt.wantDoc {
				t.Fatalf(
					"got reason %s and document state %d, want %s and %d",
					result.Reason,
					result.DocModified,
					tt.wantReason,
					tt.wantDoc,
				)
			}
		})
	}
}

// TestSignatureResultEvidenceCategoriesRemainDistinct verifies non-cryptographic findings cannot downgrade invalid evidence.
func TestSignatureResultEvidenceCategoriesRemainDistinct(t *testing.T) {
	unsupported := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}
	markUnsupportedEvidence(unsupported)
	if unsupported.Status != model.SignatureStatusUnknown ||
		unsupported.Reason != model.SignatureReasonUnsupported {
		t.Fatalf("unexpected unsupported state: status=%s reason=%s", unsupported.Status, unsupported.Reason)
	}

	malformed := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}
	markMalformedEvidence(malformed)
	if malformed.Status != model.SignatureStatusUnknown ||
		malformed.Reason != model.SignatureReasonMalformed {
		t.Fatalf("unexpected malformed state: status=%s reason=%s", malformed.Status, malformed.Reason)
	}

	certificate := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}
	markCertificateInvalidEvidence(certificate)
	if certificate.Status != model.SignatureStatusUnknown || certificate.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("unexpected certificate state: status=%s reason=%s", certificate.Status, certificate.Reason)
	}

	invalid := &model.SignatureValidationResult{
		Status:      model.SignatureStatusInvalid,
		Reason:      model.SignatureReasonSignatureForged,
		DocModified: model.False,
	}
	markUnsupportedEvidence(invalid)
	markCertificateInvalidEvidence(invalid)
	if invalid.Status != model.SignatureStatusInvalid ||
		invalid.Reason != model.SignatureReasonSignatureForged ||
		invalid.DocModified != model.False {
		t.Fatalf("invalid evidence was downgraded: status=%s reason=%s document=%d", invalid.Status, invalid.Reason, invalid.DocModified)
	}
}

// TestCertificatePathConclusionPreservesTrustAndRecordsMethod verifies
// structured path evidence mirrors the legacy Trust conclusion.
func TestCertificatePathConclusionPreservesTrustAndRecordsMethod(t *testing.T) {
	certDetails := &model.CertificateDetails{}
	setCertificatePathConclusion(
		certDetails,
		model.True,
		"certificate path resolved using the configured local certificate store",
		model.CertificatePathMethodLocalTrustStore,
	)

	if certDetails.Trust.Status != model.True ||
		certDetails.Trust.Reason != "certificate path resolved using the configured local certificate store" {
		t.Fatalf("unexpected legacy trust details: %+v", certDetails.Trust)
	}
	evidence := certDetails.PathEvidence
	if evidence.AssessmentScope != model.AssessmentScopeLocal ||
		evidence.Method != model.CertificatePathMethodLocalTrustStore ||
		evidence.Status != certDetails.Trust.Status ||
		evidence.Reason != certDetails.Trust.Reason {
		t.Fatalf("unexpected certificate path evidence: %+v", evidence)
	}
}

// TestRecognizedQualifiedCertificatePolicyEvidence verifies qualified evidence
// is limited to recognized certificate-policy identifiers.
func TestRecognizedQualifiedCertificatePolicyEvidence(t *testing.T) {
	recognized := []asn1.ObjectIdentifier{
		oidQCESign,
		oidQCESeal,
		oidQWebAuthCert,
		oidETSIQCPublicWithSSCD,
	}
	for _, policy := range recognized {
		cert := &x509.Certificate{PolicyIdentifiers: []asn1.ObjectIdentifier{policy}}
		if !hasRecognizedQualifiedCertificatePolicy(cert) {
			t.Errorf("recognized certificate policy %s was not reported", policy)
		}
	}

	cert := &x509.Certificate{PolicyIdentifiers: []asn1.ObjectIdentifier{{1, 2, 3, 4}}}
	if hasRecognizedQualifiedCertificatePolicy(cert) {
		t.Fatal("unrecognized certificate policy was reported as qualified evidence")
	}
}

// TestCertificateAuthorityObservationsDoNotResolvePath verifies CA and
// self-signature observations remain independent of path resolution.
func TestCertificateAuthorityObservationsDoNotResolvePath(t *testing.T) {
	root, _ := testCertChain(t, "Observed Root", "Observed Leaf")
	signer := &model.Signer{}
	result := unknownSignatureResult()

	validateCertChains(
		[][]*x509.Certificate{{root}},
		false,
		x509.NewCertPool(),
		signer,
		nil,
		nil,
		result,
		model.NewDefaultConfiguration(),
	)

	certificate := signer.Certificate
	if certificate == nil || !certificate.CA || !certificate.SelfSigned {
		t.Fatalf("CA or self-signature observation missing: %+v", certificate)
	}
	if certificate.Trust.Status != model.Unknown ||
		certificate.PathEvidence.Status != model.Unknown ||
		signer.CertificatePathStatus != model.Unknown {
		t.Fatalf("observations resolved fallback path: %+v", certificate)
	}
}

// TestFallbackCAEntriesDoNotOverwritePathFailure verifies fallback-chain CA
// observations cannot turn the unresolved overall path into a success.
func TestFallbackCAEntriesDoNotOverwritePathFailure(t *testing.T) {
	root, leaf := testCertChain(t, "Fallback Root", "Fallback Leaf")
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonCertNotTrusted}

	validateCertChains(
		[][]*x509.Certificate{{leaf, root}},
		false,
		x509.NewCertPool(),
		signer,
		nil,
		nil,
		result,
		model.NewDefaultConfiguration(),
	)

	leafDetails := signer.Certificate
	if leafDetails == nil || leafDetails.IssuerCertificate == nil {
		t.Fatalf("fallback observations missing: %+v", signer)
	}
	rootDetails := leafDetails.IssuerCertificate
	if !rootDetails.CA || !rootDetails.SelfSigned {
		t.Fatalf("root observations missing: %+v", rootDetails)
	}
	if leafDetails.PathEvidence.Status != model.Unknown ||
		rootDetails.PathEvidence.Status != model.Unknown ||
		signer.CertificatePathStatus != model.Unknown {
		t.Fatalf("fallback CA overwrote overall path failure: %+v", signer)
	}
	if result.Reason != model.SignatureReasonCertNotTrusted {
		t.Fatalf("CA entry overwrote path failure reason: %+v", result)
	}
}

type signRoundTripFunc func(*http.Request) (*http.Response, error)

func (f signRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func requireTimestampEvidence(
	t *testing.T,
	evidence timestampEvidence,
	kind timestampKind,
	signingTime time.Time,
	present bool,
) {
	t.Helper()
	if evidence.Err != nil {
		t.Fatal(evidence.Err)
	}
	if evidence.Kind != kind {
		t.Errorf("got timestamp kind %d, want %d", evidence.Kind, kind)
	}
	if evidence.Present != present {
		t.Errorf("got timestamp presence %t, want %t", evidence.Present, present)
	}
	if !evidence.SigningTime.Equal(signingTime) {
		t.Errorf("got timestamp %v, want %v", evidence.SigningTime, signingTime)
	}
	if evidence.AssessmentScope != model.AssessmentScopeLocal {
		t.Errorf("got assessment scope %d, want local", evidence.AssessmentScope)
	}
}

type exportedValidationFunc func(
	io.ReaderAt,
	types.Dict,
	bool,
	bool,
	bool,
	int,
	*x509.CertPool,
	*model.SignatureValidationResult,
	*model.Context,
) error

var (
	_ exportedValidationFunc = ValidatePKCS7Signatures
	_ exportedValidationFunc = ValidateDTS
	_ exportedValidationFunc = ValidateX509RSASHA1Signature
)

// TestExportedValidatorSignaturesRemainCompatible locks down the service-free
// PKCS#7, DTS, and legacy PKCS#1 entry points.
func TestExportedValidatorSignaturesRemainCompatible(t *testing.T) {
	validators := []exportedValidationFunc{
		ValidatePKCS7Signatures,
		ValidateDTS,
		ValidateX509RSASHA1Signature,
	}
	for _, validate := range validators {
		if validate != nil {
			continue
		}
		t.Fatal("missing exported signature validator")
	}
}

func (r signErrorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func (r signErrorReader) ReadAt([]byte, int64) (int, error) {
	return 0, r.err
}

func byteRange(values ...types.Object) types.Array {
	return types.Array(values)
}

func malformedEmbeddedCertificatePKCS7(t *testing.T) []byte {
	t.Helper()
	// The context-specific certificate set contains a DER SEQUENCE with an
	// INTEGER instead of an X.509 Certificate.
	return pkcs7WithRawCertificates(t, []byte{0xA0, 0x05, 0x30, 0x03, 0x02, 0x01, 0x01})
}

func pkcs7WithRawCertificates(t *testing.T, rawCerts []byte) []byte {
	t.Helper()
	signedData, err := asn1.Marshal(malformedP7SignedData{
		Version:     1,
		ContentInfo: malformedP7ContentInfo{ContentType: pkcs7.OIDData},
		Certificates: malformedP7RawCertificates{
			Raw: rawCerts,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := asn1.Marshal(malformedP7ContentInfo{
		ContentType: pkcs7.OIDSignedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      signedData,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return fixture
}

func zeroSignerPKCS7(t *testing.T) []byte {
	t.Helper()
	signedData, err := asn1.Marshal(malformedP7SignedData{
		Version:     1,
		ContentInfo: malformedP7ContentInfo{ContentType: pkcs7.OIDData},
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := asn1.Marshal(malformedP7ContentInfo{
		ContentType: pkcs7.OIDSignedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      signedData,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return fixture
}

func signerPKCS7(t *testing.T, signerCount int) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	template := testCertTemplate("Multiple Signer", false)
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	signedData, err := pkcs7.NewSignedData()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("digest"))
	for range signerCount {
		if err := signedData.AddSigner(
			cert,
			key,
			digest[:],
			pkcs7.OIDDigestAlgorithmSHA256,
			pkcs7.SignerInfoConfig{},
		); err != nil {
			t.Fatal(err)
		}
	}
	fixture, err := signedData.Finish()
	if err != nil {
		t.Fatal(err)
	}
	return fixture
}

func multipleSignerPKCS7(t *testing.T) []byte {
	t.Helper()
	return signerPKCS7(t, 2)
}

type detachedP7SignerFixture struct {
	signer  pkcs7.SignerInfo
	certs   []*x509.Certificate
	cert    *x509.Certificate
	content []byte
	raw     []byte
	roots   *x509.CertPool
	ctx     *model.Context
}

func newDetachedP7SignerFixture(
	t *testing.T,
	signedAttributes, unsignedAttributes []pkcs7.Attribute,
) detachedP7SignerFixture {
	t.Helper()
	return newDetachedP7SignerFixtureWithAttributes(
		t,
		func(*x509.Certificate) []pkcs7.Attribute {
			return signedAttributes
		},
		unsignedAttributes,
	)
}

func newDetachedP7SignerFixtureWithAttributes(
	t *testing.T,
	signedAttributes func(*x509.Certificate) []pkcs7.Attribute,
	unsignedAttributes []pkcs7.Attribute,
) detachedP7SignerFixture {
	t.Helper()
	key := testRSAKey(t)
	template := testCertTemplate("PKCS7 Signer", true)
	cert := testCertificate(t, template, template, &key.PublicKey, key)
	content := []byte("signed content")
	digest := sha256.Sum256(content)
	signedData, err := pkcs7.NewSignedData()
	if err != nil {
		t.Fatal(err)
	}
	if err := signedData.AddSigner(
		cert,
		key,
		digest[:],
		pkcs7.OIDDigestAlgorithmSHA256,
		pkcs7.SignerInfoConfig{
			ExtraSignedAttributes:   signedAttributes(cert),
			ExtraUnsignedAttributes: unsignedAttributes,
		},
	); err != nil {
		t.Fatal(err)
	}
	fixture, err := signedData.Finish()
	if err != nil {
		t.Fatal(err)
	}
	p7, err := pkcs7.Parse(fixture)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	return detachedP7SignerFixture{
		signer:  p7.Signers[0],
		certs:   p7.Certificates,
		cert:    cert,
		content: content,
		raw:     fixture,
		roots:   roots,
		ctx: &model.Context{
			Configuration: model.NewDefaultConfiguration(),
			XRefTable:     &model.XRefTable{},
		},
	}
}

// TestValidateP7ClassifiesMalformedEmbeddedCertificate verifies malformed embedded certificates are certificate evidence.
func TestValidateP7ClassifiesMalformedEmbeddedCertificate(t *testing.T) {
	fixture := malformedEmbeddedCertificatePKCS7(t)
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	sigDict := types.Dict{"Contents": types.HexLiteral(hex.EncodeToString(fixture))}

	_, err := pkcs7.Parse(fixture)
	if !errors.Is(err, pkcs7.ErrCertificateParse) {
		t.Fatalf("expected dependency certificate parse classification, got %v", err)
	}

	_, err = p7(sigDict)
	if !errors.Is(err, pkcs7.ErrCertificateParse) {
		t.Fatalf("expected wrapped certificate parse classification, got %v", err)
	}
	var parseErr *pkcs7.CertificateParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected typed certificate parse error, got %T", err)
	}
	if want := "PKCS#7 embedded certificates: parse certificate"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}

	if got := validateP7(sigDict, result); got != nil {
		t.Fatal("expected malformed PKCS#7 parsing to stop")
	}
	if result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("got reason %s, want %s; problems: %v", result.Reason, model.SignatureReasonCertInvalid, result.Problems)
	}
	problems := strings.Join(result.Problems, "\n")
	if !strings.Contains(problems, "certificate data is malformed") {
		t.Fatalf("expected certificate-invalid evidence, got %q", problems)
	}
	if strings.Contains(problems, "pkcs7: message without signers") {
		t.Fatalf("got generic internal evidence: %q", problems)
	}
}

// TestValidateP7ReportsZeroSigners verifies signerless PKCS#7 data is reportable and does not panic.
func TestValidateP7ReportsZeroSigners(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("zero-signer PKCS#7 validation panicked: %v", recovered)
		}
	}()

	fixture := zeroSignerPKCS7(t)
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	sigDict := types.Dict{"Contents": types.HexLiteral(hex.EncodeToString(fixture))}

	if got := validateP7(sigDict, result); got != nil {
		t.Fatal("expected zero-signer PKCS#7 parsing to stop")
	}
	if result.Reason != model.SignatureReasonMalformed {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonMalformed)
	}
	if len(result.Problems) != 1 || result.Problems[0] != "pkcs7: message without signers" {
		t.Fatalf("got problems %v, want %q", result.Problems, "pkcs7: message without signers")
	}
}

// TestValidateP7RejectsMultipleCAdESSigners verifies ETSI.CAdES.detached requires exactly one signer.
func TestValidateP7RejectsMultipleCAdESSigners(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("multiple-signer PKCS#7 validation panicked: %v", recovered)
		}
	}()

	fixture := multipleSignerPKCS7(t)
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	result.Details.SubFilter = "ETSI.CAdES.detached"
	sigDict := types.Dict{"Contents": types.HexLiteral(hex.EncodeToString(fixture))}

	if got := validateP7(sigDict, result); got != nil {
		t.Fatal("expected multiple-signer CAdES parsing to stop")
	}
	if result.Reason != model.SignatureReasonMalformed {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonMalformed)
	}
	const want = "pkcs7: \"ETSI.CAdES.detached\" requires a single signer"
	if len(result.Problems) != 1 || result.Problems[0] != want {
		t.Fatalf("got problems %v, want %q", result.Problems, want)
	}
}

// TestFinalizePKCS7ResultBranches verifies successful finalization and invalid-evidence presentation.
func TestFinalizePKCS7ResultBranches(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		result := &model.SignatureValidationResult{
			Status: model.SignatureStatusUnknown,
			Reason: model.SignatureReasonUnknown,
		}

		finalizePKCS7Result(result, completedLocalSignatureAssessment())

		if result.Status != model.SignatureStatusValid ||
			result.Reason != model.SignatureReasonDocNotModified ||
			result.DocModified != model.False {
			t.Fatalf("got status %s, reason %s and modified %d", result.Status, result.Reason, result.DocModified)
		}
	})

	t.Run("NoSigners", func(t *testing.T) {
		result := unknownSignatureResult()

		finalizePKCS7Result(result, localSignatureAssessment{})

		if result.Status != model.SignatureStatusUnknown ||
			result.Reason != model.SignatureReasonUnknown ||
			result.DocModified != model.Unknown {
			t.Fatalf("zero-signer assessment finalized validity: %+v", result)
		}
	})

	t.Run("InvalidClearsPAdES", func(t *testing.T) {
		result := &model.SignatureValidationResult{
			Status: model.SignatureStatusInvalid,
			Reason: model.SignatureReasonSignatureForged,
		}
		result.Details.AddSigner(&model.Signer{PAdES: "B-B"})
		result.Details.AddSigner(&model.Signer{PAdES: "B-T"})

		finalizePKCS7Result(result, completedLocalSignatureAssessment())

		for i, signer := range result.Details.Signers {
			if signer.PAdES != "" {
				t.Fatalf("signer %d: got PAdES level %q, want empty", i+1, signer.PAdES)
			}
		}
	})
}

// TestLocalSignatureAssessmentMergeSelfInitializes verifies aggregate
// construction is safe from the zero value and ignores empty observations.
func TestLocalSignatureAssessmentMergeSelfInitializes(t *testing.T) {
	first := completedLocalSignatureAssessment()
	var aggregate localSignatureAssessment

	aggregate.merge(localSignatureAssessment{})
	if aggregate != (localSignatureAssessment{}) {
		t.Fatalf("empty signer assessment changed aggregate: %+v", aggregate)
	}

	aggregate.merge(first)
	if aggregate != first {
		t.Fatalf("first signer did not initialize aggregate: got %+v, want %+v", aggregate, first)
	}

	second := completedLocalSignatureAssessment()
	second.SignatureAuthenticated = false
	second.ProfileValidated = false
	aggregate.merge(second)
	if aggregate.SignersProcessed != 2 {
		t.Fatalf("got %d processed signers, want 2", aggregate.SignersProcessed)
	}
	if aggregate.SignatureAuthenticated || aggregate.ProfileValidated {
		t.Fatalf("positive signer gates were not AND-merged: %+v", aggregate)
	}
	if !aggregate.DigestVerified ||
		!aggregate.CertificateIdentified ||
		!aggregate.PathValidated ||
		!aggregate.RevocationGood {
		t.Fatalf("successful signer gates were lost during merge: %+v", aggregate)
	}
}

// TestFinalizeLocalSignatureResultRequiresEveryGate verifies local validity
// cannot be inferred from the absence of a reported problem.
func TestFinalizeLocalSignatureResultRequiresEveryGate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*localSignatureAssessment)
	}{
		{"SignatureNotAuthenticated", func(a *localSignatureAssessment) { a.SignatureAuthenticated = false }},
		{"DigestNotVerified", func(a *localSignatureAssessment) { a.DigestVerified = false }},
		{"ProfileNotValidated", func(a *localSignatureAssessment) { a.ProfileValidated = false }},
		{"CertificateNotIdentified", func(a *localSignatureAssessment) { a.CertificateIdentified = false }},
		{"PathNotValidated", func(a *localSignatureAssessment) { a.PathValidated = false }},
		{"RevocationNotGood", func(a *localSignatureAssessment) { a.RevocationGood = false }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment := completedLocalSignatureAssessment()
			tt.mutate(&assessment)
			result := unknownSignatureResult()

			if finalizeLocalSignatureResult(result, assessment) {
				t.Fatal("incomplete assessment reported successful finalization")
			}
			if result.Status != model.SignatureStatusUnknown ||
				result.Reason != model.SignatureReasonUnknown ||
				result.DocModified != model.Unknown {
				t.Fatalf("incomplete assessment produced validity: %+v", result)
			}
		})
	}
}

// TestFinalizeLocalSignatureResultRequiresGoodRevocation verifies the positive
// validity gate does not depend on separate revoked or unknown reason handling.
func TestFinalizeLocalSignatureResultRequiresGoodRevocation(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		wantComplete bool
	}{
		{"Unknown", model.Unknown, false},
		{"Revoked", model.False, false},
		{"Good", model.True, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment := completedLocalSignatureAssessment()
			assessment.applyCertificateAssessment(certificateAssessment{
				Certificate: &model.CertificateDetails{
					Revocation: model.RevocationDetails{Status: tt.status},
				},
				CertificatePathStatus: model.True,
			})
			if assessment.RevocationGood != tt.wantComplete {
				t.Fatalf(
					"got RevocationGood=%t, want %t",
					assessment.RevocationGood,
					tt.wantComplete,
				)
			}

			result := unknownSignatureResult()
			if got := finalizeLocalSignatureResult(result, assessment); got != tt.wantComplete {
				t.Fatalf("got finalized=%t, want %t", got, tt.wantComplete)
			}
			if tt.wantComplete {
				if result.Status != model.SignatureStatusValid {
					t.Fatalf("good revocation evidence did not finalize valid: %+v", result)
				}
				return
			}
			if result.Status != model.SignatureStatusUnknown ||
				result.Reason != model.SignatureReasonUnknown ||
				result.DocModified != model.Unknown {
				t.Fatalf("non-good revocation evidence finalized validity: %+v", result)
			}
		})
	}
}

// TestVerifyP7SignerSignatureProblemHasNoTrailingNewline verifies signature evidence is presentation neutral.
func TestVerifyP7SignerSignatureProblemHasNoTrailingNewline(t *testing.T) {
	key := testRSAKey(t)
	template := testCertTemplate("PKCS7 Signer", false)
	cert := testCertificate(t, template, template, &key.PublicKey, key)
	content := []byte("signed content")
	digest := sha256.Sum256(content)
	signedData, err := pkcs7.NewSignedData()
	if err != nil {
		t.Fatal(err)
	}
	if err := signedData.AddSigner(
		cert,
		key,
		digest[:],
		pkcs7.OIDDigestAlgorithmSHA256,
		pkcs7.SignerInfoConfig{},
	); err != nil {
		t.Fatal(err)
	}
	fixture, err := signedData.Finish()
	if err != nil {
		t.Fatal(err)
	}
	p7, err := pkcs7.Parse(fixture)
	if err != nil {
		t.Fatal(err)
	}
	p7Signer := p7.Signers[0]
	p7Signer.EncryptedDigest[0] ^= 0xff
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}
	ctx := &model.Context{Configuration: model.NewDefaultConfiguration()}

	verifyP7Signer(
		p7Signer,
		p7.Certificates,
		x509.NewCertPool(),
		content,
		content,
		true,
		false,
		false,
		0,
		0,
		result,
		ctx,
	)

	if len(result.Details.Signers) != 1 {
		t.Fatalf("got %d signers, want 1", len(result.Details.Signers))
	}
	problems := result.Details.Signers[0].Problems
	if len(problems) != 1 || !strings.Contains(problems[0], "pkcs7: verify signature failure") {
		t.Fatalf("got problems %v, want verify-signature evidence", problems)
	}
	if strings.HasSuffix(problems[0], "\n") {
		t.Fatalf("unexpected trailing newline in %q", problems[0])
	}
}

// TestVerifyP7SignerValidBranch verifies a successful generic detached signer
// does not imply an ETSI baseline profile.
func TestVerifyP7SignerValidBranch(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}

	verifyP7Signer(
		fixture.signer,
		fixture.certs,
		fixture.roots,
		fixture.content,
		fixture.content,
		true,
		false,
		false,
		0,
		0,
		result,
		fixture.ctx,
	)

	if result.Reason != model.SignatureReasonUnknown {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonUnknown)
	}
	if result.DocModified != model.False {
		t.Fatalf("got document modified %d, want %d", result.DocModified, model.False)
	}
	if len(result.Details.Signers) != 1 {
		t.Fatalf("got %d signers, want 1", len(result.Details.Signers))
	}
	signer := result.Details.Signers[0]
	if signer.PAdES != "" {
		t.Fatalf("got PAdES level %q, want none", signer.PAdES)
	}
	if len(signer.Problems) != 0 {
		t.Fatalf("unexpected signer problems %v", signer.Problems)
	}
	if signer.Certificate == nil {
		t.Fatal("expected certificate details")
	}
}

// TestPAdESBaselineBClassification verifies B-B is limited to a complete,
// explicitly supported ETSI baseline signature profile.
func TestPAdESBaselineBClassification(t *testing.T) {
	tests := []struct {
		name                 string
		subFilter            string
		attributes           func(*testing.T) func(*x509.Certificate) []pkcs7.Attribute
		want                 string
		wantProfileValidated bool
		wantProfileProblem   bool
	}{
		{
			name:                 "AdobeDetachedWithESS",
			subFilter:            "adbe.pkcs7.detached",
			wantProfileValidated: true,
			attributes: func(t *testing.T) func(*x509.Certificate) []pkcs7.Attribute {
				return validBaselineAttributes(t)
			},
		},
		{
			name:                 "AdobeMalformedESS",
			subFilter:            "adbe.pkcs7.detached",
			wantProfileValidated: true,
			attributes: func(*testing.T) func(*x509.Certificate) []pkcs7.Attribute {
				return func(*x509.Certificate) []pkcs7.Attribute {
					return []pkcs7.Attribute{{
						Type:  oidSigningCertificateV2,
						Value: struct{}{},
					}}
				}
			},
		},
		{
			name:               "ETSIIncomplete",
			subFilter:          "ETSI.CAdES.detached",
			wantProfileProblem: true,
			attributes: func(*testing.T) func(*x509.Certificate) []pkcs7.Attribute {
				return func(*x509.Certificate) []pkcs7.Attribute {
					return nil
				}
			},
		},
		{
			name:               "ETSIMalformed",
			subFilter:          "ETSI.CAdES.detached",
			wantProfileProblem: true,
			attributes: func(*testing.T) func(*x509.Certificate) []pkcs7.Attribute {
				return func(*x509.Certificate) []pkcs7.Attribute {
					return []pkcs7.Attribute{{
						Type:  oidSigningCertificateV2,
						Value: struct{}{},
					}}
				}
			},
		},
		{
			name:                 "ETSISupportedBaseline",
			subFilter:            "ETSI.CAdES.detached",
			wantProfileValidated: true,
			attributes: func(t *testing.T) func(*x509.Certificate) []pkcs7.Attribute {
				return validBaselineAttributes(t)
			},
			want: "B-B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newDetachedP7SignerFixtureWithAttributes(
				t,
				tt.attributes(t),
				nil,
			)
			result := unknownSignatureResult()
			result.Details.SubFilter = tt.subFilter
			assessment := localSignatureAssessment{}

			if err := verifyP7SignerWithContentType(
				fixture.signer,
				fixture.certs,
				fixture.roots,
				fixture.content,
				fixture.content,
				true,
				false,
				false,
				0,
				0,
				result,
				fixture.ctx,
				pkcs7.OIDData,
				&assessment,
			); err != nil {
				t.Fatal(err)
			}

			signer := result.Details.Signers[0]
			if signer.PAdES != tt.want {
				t.Fatalf("got PAdES level %q, want %q", signer.PAdES, tt.want)
			}
			if assessment.ProfileValidated != tt.wantProfileValidated {
				t.Fatalf(
					"got ProfileValidated=%t, want %t",
					assessment.ProfileValidated,
					tt.wantProfileValidated,
				)
			}
			gotProfileProblem := false
			for _, problem := range signer.Problems {
				if strings.HasPrefix(
					problem,
					"SubFilter ETSI.CAdES.detached: validate baseline B profile:",
				) {
					gotProfileProblem = true
					break
				}
			}
			if gotProfileProblem != tt.wantProfileProblem {
				t.Fatalf(
					"got profile problem=%t, want %t; Problems=%v",
					gotProfileProblem,
					tt.wantProfileProblem,
					signer.Problems,
				)
			}
		})
	}
}

// TestAdobePKCS7ProfileValidation verifies the supported Adobe subfilters
// agree with the CMS content layout and content type.
func TestAdobePKCS7ProfileValidation(t *testing.T) {
	tests := []struct {
		name        string
		subFilter   string
		detached    bool
		contentType asn1.ObjectIdentifier
		ok          bool
	}{
		{"Detached", "adbe.pkcs7.detached", true, pkcs7.OIDData, true},
		{"SHA1", "adbe.pkcs7.sha1", false, pkcs7.OIDData, true},
		{"DetachedWithEmbeddedContent", "adbe.pkcs7.detached", false, pkcs7.OIDData, false},
		{"SHA1WithDetachedContent", "adbe.pkcs7.sha1", true, pkcs7.OIDData, false},
		{"DetachedWrongContentType", "adbe.pkcs7.detached", true, pkcs7.OIDSignedData, false},
		{"SHA1WrongContentType", "adbe.pkcs7.sha1", false, pkcs7.OIDSignedData, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAdobePKCS7Profile(tt.subFilter, tt.detached, tt.contentType)
			if tt.ok && err != nil {
				t.Fatalf("supported Adobe profile rejected: %v", err)
			}
			if !tt.ok && !errors.Is(err, errMalformedAdobePKCS7Profile) {
				t.Fatalf("got %v, want malformed Adobe profile", err)
			}
		})
	}
}

// TestCAdESBaselineBProfileValidationMatrix locks every supported B-B profile
// condition independently from signature and revocation conclusions.
func TestCAdESBaselineBProfileValidationMatrix(t *testing.T) {
	v2 := newDetachedP7SignerFixtureWithAttributes(t, validBaselineAttributes(t), nil)
	v1 := newDetachedP7SignerFixtureWithAttributes(
		t,
		func(cert *x509.Certificate) []pkcs7.Attribute {
			return []pkcs7.Attribute{timestampSigningCertificateAttributeV1(t, cert, true)}
		},
		nil,
	)
	missing := newDetachedP7SignerFixture(t, nil, nil)
	malformed := newDetachedP7SignerFixtureWithAttributes(
		t,
		func(*x509.Certificate) []pkcs7.Attribute {
			return []pkcs7.Attribute{{
				Type:  oidSigningCertificateV2,
				Value: struct{}{},
			}}
		},
		nil,
	)
	duplicateSigner := v2.signer
	for _, attr := range v2.signer.AuthenticatedAttributes {
		if attr.Type.Equal(oidSigningCertificateV2) {
			duplicateSigner.AuthenticatedAttributes = append(
				duplicateSigner.AuthenticatedAttributes,
				attr,
			)
			break
		}
	}
	other := newDetachedP7SignerFixture(t, nil, nil)
	disallowedKeyUsage := *v2.cert
	disallowedKeyUsage.KeyUsage = x509.KeyUsageCertSign

	tests := []struct {
		name        string
		detached    bool
		contentType asn1.ObjectIdentifier
		signer      pkcs7.SignerInfo
		cert        *x509.Certificate
		class       error
		cause       error
	}{
		{"SigningCertificateV2", true, pkcs7.OIDData, v2.signer, v2.cert, nil, nil},
		{"SigningCertificateV1", true, pkcs7.OIDData, v1.signer, v1.cert, nil, nil},
		{
			"AttachedContent",
			false,
			pkcs7.OIDData,
			v2.signer,
			v2.cert,
			errMalformedCAdESBaselineBProfile,
			nil,
		},
		{
			"WrongContentType",
			true,
			pkcs7.OIDSignedData,
			v2.signer,
			v2.cert,
			errUnsupportedCAdESBaselineBProfile,
			nil,
		},
		{
			"MissingESSBinding",
			true,
			pkcs7.OIDData,
			missing.signer,
			missing.cert,
			errMalformedCAdESBaselineBProfile,
			pkcs7.ErrMalformedAttribute,
		},
		{
			"MalformedESSBinding",
			true,
			pkcs7.OIDData,
			malformed.signer,
			malformed.cert,
			errMalformedCAdESBaselineBProfile,
			pkcs7.ErrMalformedAttribute,
		},
		{
			"DuplicateESSBinding",
			true,
			pkcs7.OIDData,
			duplicateSigner,
			v2.cert,
			errMalformedCAdESBaselineBProfile,
			pkcs7.ErrMalformedAttribute,
		},
		{
			"CertificateMismatch",
			true,
			pkcs7.OIDData,
			v2.signer,
			other.cert,
			errCAdESCertificateBindingMismatch,
			errESSCertificateMismatch,
		},
		{
			"DisallowedKeyUsage",
			true,
			pkcs7.OIDData,
			v2.signer,
			&disallowedKeyUsage,
			errUnsupportedCAdESBaselineBProfile,
			errUnsupportedESSCertificateProfile,
		},
		{
			"MissingCertificate",
			true,
			pkcs7.OIDData,
			v2.signer,
			nil,
			errMalformedCAdESBaselineBProfile,
			pkcs7.ErrMalformedAttribute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCAdESBaselineBProfile(
				tt.detached,
				tt.contentType,
				tt.signer,
				tt.cert,
			)
			if !errors.Is(err, tt.class) {
				t.Fatalf("got %v, want classification %v", err, tt.class)
			}
			if tt.cause != nil && !errors.Is(err, tt.cause) {
				t.Fatalf("got %v, want wrapped cause %v", err, tt.cause)
			}
		})
	}
}

// TestCAdESBaselineBProfileProblemClassification verifies profile failures are
// deliberately converted into evidence only after their causes are classified.
func TestCAdESBaselineBProfileProblemClassification(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus model.SignatureStatus
		wantReason model.SignatureReason
	}{
		{
			"Malformed",
			fmt.Errorf("%w: %w", errMalformedCAdESBaselineBProfile, pkcs7.ErrMalformedAttribute),
			model.SignatureStatusUnknown,
			model.SignatureReasonMalformed,
		},
		{
			"Unsupported",
			fmt.Errorf("%w: %w", errUnsupportedCAdESBaselineBProfile, pkcs7.ErrUnsupportedAlgorithm),
			model.SignatureStatusUnknown,
			model.SignatureReasonUnsupported,
		},
		{
			"CertificateBindingMismatch",
			fmt.Errorf("%w: %w", errCAdESCertificateBindingMismatch, errESSCertificateMismatch),
			model.SignatureStatusInvalid,
			model.SignatureReasonCertInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := &model.Signer{}
			result := unknownSignatureResult()
			reportCAdESBaselineBProfileError(tt.err, signer, result)
			if result.Status != tt.wantStatus || result.Reason != tt.wantReason {
				t.Fatalf("got status=%s reason=%s, want status=%s reason=%s",
					result.Status,
					result.Reason,
					tt.wantStatus,
					tt.wantReason,
				)
			}
			if len(signer.Problems) != 1 ||
				!strings.Contains(signer.Problems[0], tt.err.Error()) {
				t.Fatalf("got Problems %v, want classified error evidence", signer.Problems)
			}
		})
	}
}

func validBaselineAttributes(t *testing.T) func(*x509.Certificate) []pkcs7.Attribute {
	t.Helper()
	return func(cert *x509.Certificate) []pkcs7.Attribute {
		return []pkcs7.Attribute{
			timestampSigningCertificateAttributeV2(
				t,
				cert,
				crypto.SHA256,
				pkcs7.OIDDigestAlgorithmSHA256,
				true,
			),
		}
	}
}

// TestVerifyP7SignerStopsAfterDigestFailure verifies digest evidence terminates signer processing before certificate work.
func TestVerifyP7SignerStopsAfterDigestFailure(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}

	verifyP7Signer(
		fixture.signer,
		fixture.certs,
		fixture.roots,
		[]byte("modified"),
		fixture.content,
		true,
		false,
		false,
		0,
		0,
		result,
		fixture.ctx,
	)

	if result.Status != model.SignatureStatusInvalid ||
		result.Reason != model.SignatureReasonDocModified ||
		result.DocModified != model.True {
		t.Fatalf(
			"got status %s, reason %s and modified %d",
			result.Status,
			result.Reason,
			result.DocModified,
		)
	}
	signer := result.Details.Signers[0]
	if signer.Certificate != nil {
		t.Fatal("digest failure unexpectedly reached certificate processing")
	}
	if len(signer.Problems) != 1 {
		t.Fatalf("got problems %v, want one digest problem", signer.Problems)
	}
}

// TestVerifyP7SignerUsesFallbackChainAsEvidence verifies trust failure remains reportable while certificate details are built.
func TestVerifyP7SignerUsesFallbackChainAsEvidence(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}

	verifyP7Signer(
		fixture.signer,
		fixture.certs,
		x509.NewCertPool(),
		fixture.content,
		fixture.content,
		true,
		false,
		false,
		0,
		0,
		result,
		fixture.ctx,
	)

	if result.Reason != model.SignatureReasonCertNotTrusted {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertNotTrusted)
	}
	signer := result.Details.Signers[0]
	if signer.Certificate == nil {
		t.Fatal("expected fallback certificate details")
	}
	problems := strings.Join(signer.Problems, "\n")
	if !strings.Contains(problems, "certificate path was not resolved using the configured local certificate store") {
		t.Fatalf("got problems %q, want local certificate-path evidence", problems)
	}
}

// TestVerifyP7SignerReportsClaimedTimeAfterDTS verifies timestamp precedence remains reportable signer evidence.
func TestVerifyP7SignerReportsClaimedTimeAfterDTS(t *testing.T) {
	documentTimestamp := time.Now().UTC().Truncate(time.Second)
	claimedTime := documentTimestamp.Add(30 * time.Minute)
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidSigningTime, Value: claimedTime}},
		nil,
	)
	fixture.ctx.DTS = documentTimestamp
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}

	verifyP7Signer(
		fixture.signer,
		fixture.certs,
		fixture.roots,
		fixture.content,
		fixture.content,
		true,
		false,
		false,
		0,
		0,
		result,
		fixture.ctx,
	)

	if !result.Details.SigningTime.Equal(claimedTime) {
		t.Fatalf("got claimed signing time %v, want %v", result.Details.SigningTime, claimedTime)
	}
	signer := result.Details.Signers[0]
	if signer.HasTimestamp || !signer.Timestamp.IsZero() || signer.PAdES != "" {
		t.Fatalf("got signer timestamp %v and PAdES %q", signer.Timestamp, signer.PAdES)
	}
	if signer.Certificate == nil || !signer.Certificate.ValidationTime.Time.IsZero() {
		t.Fatalf("locally validated DTS selected certificate assessment time: %+v", signer.Certificate)
	}
	problems := strings.Join(signer.Problems, "\n")
	if !strings.Contains(problems, "Claimed signing time") ||
		!strings.Contains(problems, "is not before document timestamp") {
		t.Fatalf("got problems %q, want claimed-time precedence evidence", problems)
	}
}

// TestVerifyP7SignerRetainsMultipleProblems verifies later DSS and chain findings preserve earlier signer evidence.
func TestVerifyP7SignerRetainsMultipleProblems(t *testing.T) {
	documentTimestamp := time.Now().UTC().Truncate(time.Second)
	claimedTime := documentTimestamp.Add(time.Minute)
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidSigningTime, Value: claimedTime}},
		nil,
	)
	fixture.ctx.DTS = documentTimestamp
	fixture.ctx.DSS = types.Dict{"VRI": types.Dict{}}
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}

	verifyP7Signer(
		fixture.signer,
		fixture.certs,
		x509.NewCertPool(),
		fixture.content,
		fixture.content,
		true,
		false,
		false,
		0,
		0,
		result,
		fixture.ctx,
	)
	finalizePKCS7Result(result, completedLocalSignatureAssessment())

	if result.Status != model.SignatureStatusUnknown ||
		result.Reason != model.SignatureReasonUnsupported ||
		result.DocModified != model.False {
		t.Fatalf(
			"got status=%s reason=%s modified=%d, want unknown, unsupported, false",
			result.Status,
			result.Reason,
			result.DocModified,
		)
	}
	if len(result.Details.Signers) != 1 {
		t.Fatalf("got %d signers, want 1", len(result.Details.Signers))
	}
	signer := result.Details.Signers[0]
	if signer.PAdES != "" {
		t.Fatalf("got PAdES level %q after evidence finalization, want empty", signer.PAdES)
	}
	problems := strings.Join(signer.Problems, "\n")
	for _, want := range []string{
		"Claimed signing time",
		"DSS dict entry VRI: unsupported",
		"certificate path was not resolved using the configured local certificate store",
	} {
		if !strings.Contains(problems, want) {
			t.Errorf("expected %q, got %q", want, problems)
		}
	}
}

// TestVerifyP7SignerMissingCertificateIsCertificateEvidence verifies signer lookup failures are not generic internal evidence.
func TestVerifyP7SignerMissingCertificateIsCertificateEvidence(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}

	verifyP7Signer(
		fixture.signer,
		nil,
		fixture.roots,
		fixture.content,
		fixture.content,
		true,
		false,
		false,
		0,
		1,
		result,
		fixture.ctx,
	)

	if result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertInvalid)
	}
	problems := result.Details.Signers[0].Problems
	if len(problems) != 1 || !strings.Contains(problems[0], "missing certificate for signer 2") {
		t.Fatalf("got problems %v, want missing signer certificate evidence", problems)
	}
}

// TestVerifyP7SignerLaterCertificateEvidencePreservesInvalidStatus verifies later signers do not mask established forgery evidence.
func TestVerifyP7SignerLaterCertificateEvidencePreservesInvalidStatus(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	forged := fixture.signer
	forged.EncryptedDigest[0] ^= 0xff
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}

	verifyP7Signer(
		forged,
		fixture.certs,
		fixture.roots,
		fixture.content,
		fixture.content,
		true,
		false,
		false,
		0,
		0,
		result,
		fixture.ctx,
	)
	verifyP7Signer(
		fixture.signer,
		nil,
		fixture.roots,
		fixture.content,
		fixture.content,
		true,
		false,
		false,
		0,
		1,
		result,
		fixture.ctx,
	)

	if result.Status != model.SignatureStatusInvalid {
		t.Fatalf("got status %s, want %s", result.Status, model.SignatureStatusInvalid)
	}
	if result.Reason != model.SignatureReasonSignatureForged {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonSignatureForged)
	}
	if len(result.Details.Signers) != 2 {
		t.Fatalf("got %d signers, want 2", len(result.Details.Signers))
	}
	if problems := result.Details.Signers[1].Problems; len(problems) != 1 ||
		!strings.Contains(problems[0], "missing certificate for signer 2") {
		t.Fatalf("got second signer problems %v, want missing certificate evidence", problems)
	}
}

// TestVerifyP7SignerMultipleEvidencePrecedence verifies later certificate findings preserve decisive signer failures.
func TestVerifyP7SignerMultipleEvidencePrecedence(t *testing.T) {
	tests := []struct {
		name       string
		forge      bool
		content    []byte
		wantReason model.SignatureReason
		wantDoc    int
	}{
		{
			name:       "DigestThenCertificate",
			content:    []byte("modified"),
			wantReason: model.SignatureReasonDocModified,
			wantDoc:    model.True,
		},
		{
			name:       "ForgeryThenCertificate",
			forge:      true,
			wantReason: model.SignatureReasonSignatureForged,
			wantDoc:    model.Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newDetachedP7SignerFixture(t, nil, nil)
			first := fixture.signer
			if tt.forge {
				first.EncryptedDigest[0] ^= 0xff
			}
			content := tt.content
			if content == nil {
				content = fixture.content
			}
			result := &model.SignatureValidationResult{
				Status:      model.SignatureStatusUnknown,
				Reason:      model.SignatureReasonUnknown,
				DocModified: model.Unknown,
			}

			verifyP7Signer(
				first,
				fixture.certs,
				fixture.roots,
				content,
				fixture.content,
				true,
				false,
				false,
				0,
				0,
				result,
				fixture.ctx,
			)
			verifyP7Signer(
				fixture.signer,
				nil,
				fixture.roots,
				fixture.content,
				fixture.content,
				true,
				false,
				false,
				0,
				1,
				result,
				fixture.ctx,
			)

			if result.Status != model.SignatureStatusInvalid ||
				result.Reason != tt.wantReason ||
				result.DocModified != tt.wantDoc {
				t.Fatalf(
					"got status=%s reason=%s modified=%d, want invalid, %s, %d",
					result.Status,
					result.Reason,
					result.DocModified,
					tt.wantReason,
					tt.wantDoc,
				)
			}
			if len(result.Details.Signers) != 2 {
				t.Fatalf("got %d signers, want 2", len(result.Details.Signers))
			}
			if problems := strings.Join(result.Details.Signers[1].Problems, "\n"); !strings.Contains(problems, "missing certificate for signer 2") {
				t.Fatalf("got second signer problems %q, want certificate evidence", problems)
			}
		})
	}
}

// TestVerifyP7SignerUnsupportedPublicKeyIsUnsupportedEvidence verifies unsupported verification is not classified as forgery.
func TestVerifyP7SignerUnsupportedPublicKeyIsUnsupportedEvidence(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	unsupported := *fixture.cert
	unsupported.PublicKey = nil
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}

	verifyP7Signer(
		fixture.signer,
		[]*x509.Certificate{&unsupported},
		fixture.roots,
		fixture.content,
		fixture.content,
		true,
		false,
		false,
		0,
		0,
		result,
		fixture.ctx,
	)

	if result.Status != model.SignatureStatusUnknown {
		t.Fatalf("got status %s, want %s", result.Status, model.SignatureStatusUnknown)
	}
	if result.Reason != model.SignatureReasonUnsupported {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonUnsupported)
	}
	problems := result.Details.Signers[0].Problems
	if len(problems) != 1 || !strings.Contains(problems[0], "verify signature unsupported") {
		t.Fatalf("got problems %v, want unsupported signature evidence", problems)
	}
}

// TestVerifyP7SignerUnsupportedSignatureAlgorithmIsUnsupportedEvidence verifies unrecognized PKCS#7 algorithms are not forgery.
func TestVerifyP7SignerUnsupportedSignatureAlgorithmIsUnsupportedEvidence(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	signerInfo := fixture.signer
	signerInfo.DigestEncryptionAlgorithm.Algorithm = asn1.ObjectIdentifier{1, 2, 3}
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}

	verifyP7Signer(
		signerInfo,
		fixture.certs,
		fixture.roots,
		fixture.content,
		fixture.content,
		true,
		false,
		false,
		0,
		0,
		result,
		fixture.ctx,
	)

	if result.Status != model.SignatureStatusUnknown {
		t.Fatalf("got status %s, want %s", result.Status, model.SignatureStatusUnknown)
	}
	if result.Reason != model.SignatureReasonUnsupported {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonUnsupported)
	}
	problems := result.Details.Signers[0].Problems
	if len(problems) != 1 || !strings.Contains(problems[0], "verify signature unsupported") {
		t.Fatalf("got problems %v, want unsupported signature evidence", problems)
	}
}

func TestVerifyP7SignerReportsMissingSignerIdentifierEvidence(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	p7Signer := fixture.signer
	p7Signer.IssuerAndSerialNumber.IssuerName = asn1.RawValue{}
	p7Signer.IssuerAndSerialNumber.SerialNumber = nil
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}

	verifyP7Signer(
		p7Signer,
		fixture.certs,
		fixture.roots,
		fixture.content,
		fixture.content,
		true,
		false,
		false,
		0,
		0,
		result,
		fixture.ctx,
	)

	if result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertInvalid)
	}
	problems := strings.Join(result.Details.Signers[0].Problems, "\n")
	for _, want := range []string{"signer 1 identifier", "missing signer identifier"} {
		if !strings.Contains(problems, want) {
			t.Errorf("got problems %q, want %q", problems, want)
		}
	}
}

// TestP7SignatureErrorClassification verifies malformed encodings remain
// distinct from genuinely unsupported verification mechanisms.
func TestP7SignatureErrorClassification(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		wantUnsupported bool
		wantMalformed   bool
	}{
		{"PKCS7Unsupported", fmt.Errorf("select algorithm: %w", pkcs7.ErrUnsupportedAlgorithm), true, false},
		{"X509Unsupported", x509.ErrUnsupportedAlgorithm, true, false},
		{"Insecure", x509.InsecureAlgorithmError(x509.SHA1WithRSA), true, false},
		{"Syntax", asn1.SyntaxError{Msg: "malformed"}, false, true},
		{"Structural", asn1.StructuralError{Msg: "malformed"}, false, true},
		{"PlainTextUnsupported", errors.New("unsupported signature algorithm"), false, false},
		{"Verification", rsa.ErrVerification, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUnsupportedP7SignatureError(tt.err); got != tt.wantUnsupported {
				t.Errorf("got unsupported=%t, want %t", got, tt.wantUnsupported)
			}
			if got := isMalformedP7SignatureError(tt.err); got != tt.wantMalformed {
				t.Errorf("got malformed=%t, want %t", got, tt.wantMalformed)
			}
		})
	}
}

// TestVerifyP7SignaturePreservesVerificationCause verifies the PKCS#7 call
// path retains rsa.ErrVerification.
func TestVerifyP7SignaturePreservesVerificationCause(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	signer := fixture.signer
	signer.EncryptedDigest[0] ^= 0xff

	tests := []struct {
		name   string
		signer pkcs7.SignerInfo
	}{
		{"Detached", signer},
		{"Embedded", fixture.signer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyP7Signature(tt.signer, fixture.cert, fixture.content)
			if !errors.Is(err, rsa.ErrVerification) {
				t.Fatalf("expected rsa.ErrVerification, got %v", err)
			}
			if want := "verify cryptographic signature"; !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %q", want, err)
			}
		})
	}
}

// TestP7DigestEvidenceIsAppliedAfterObservation verifies calculating detached
// digest evidence does not mutate document-integrity conclusions.
func TestP7DigestEvidenceIsAppliedAfterObservation(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	tests := []struct {
		name    string
		signer  pkcs7.SignerInfo
		content []byte
		reason  model.SignatureReason
		status  model.SignatureStatus
	}{
		{
			name:    "MissingAttributes",
			signer:  pkcs7.SignerInfo{},
			content: fixture.content,
			reason:  model.SignatureReasonMalformed,
			status:  model.SignatureStatusUnknown,
		},
		{
			name:    "DigestMismatch",
			signer:  fixture.signer,
			content: []byte("modified"),
			reason:  model.SignatureReasonDocModified,
			status:  model.SignatureStatusInvalid,
		},
		{
			name: "UnsupportedDigest",
			signer: func() pkcs7.SignerInfo {
				signer := fixture.signer
				signer.DigestAlgorithm.Algorithm = asn1.ObjectIdentifier{1, 2, 3}
				return signer
			}(),
			content: fixture.content,
			reason:  model.SignatureReasonUnsupported,
			status:  model.SignatureStatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := &model.Signer{}
			result := &model.SignatureValidationResult{
				Status:      model.SignatureStatusUnknown,
				Reason:      model.SignatureReasonUnknown,
				DocModified: model.Unknown,
			}

			reason, err := verifyP7Digest(tt.signer, tt.content, nil, true)
			if err == nil {
				t.Fatal("expected digest evidence error")
			}
			if result.DocModified != model.Unknown ||
				result.Status != model.SignatureStatusUnknown ||
				result.Reason != model.SignatureReasonUnknown ||
				len(signer.Problems) != 0 {
				t.Fatalf("digest observation mutated result: signer=%+v result=%+v", signer, result)
			}
			if applyP7DigestEvidence(reason, err, signer, result) {
				t.Fatal("expected digest evidence rejection")
			}
			if result.Reason != tt.reason || result.Status != tt.status {
				t.Fatalf(
					"got status %s and reason %s, want status %s and reason %s",
					result.Status,
					result.Reason,
					tt.status,
					tt.reason,
				)
			}
			if len(signer.Problems) != 1 {
				t.Fatalf("got problems %v, want one digest problem", signer.Problems)
			}
		})
	}
}

// TestP7AuthenticationFailureRetainsUnknownDocumentIntegrity verifies digest
// success does not become DocModified=False before CMS authentication.
func TestP7AuthenticationFailureRetainsUnknownDocumentIntegrity(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	tests := []struct {
		name   string
		mutate func(*pkcs7.SignerInfo)
	}{
		{
			name: "Forged",
			mutate: func(signer *pkcs7.SignerInfo) {
				signer.EncryptedDigest = append([]byte(nil), signer.EncryptedDigest...)
				signer.EncryptedDigest[0] ^= 0xff
			},
		},
		{
			name: "Malformed",
			mutate: func(signer *pkcs7.SignerInfo) {
				contentType := signer.AuthenticatedAttributes[0]
				signer.AuthenticatedAttributes = append(
					signer.AuthenticatedAttributes,
					contentType,
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p7Signer := fixture.signer
			tt.mutate(&p7Signer)
			result := &model.SignatureValidationResult{
				Status:      model.SignatureStatusUnknown,
				Reason:      model.SignatureReasonUnknown,
				DocModified: model.Unknown,
			}

			if err := verifyP7Signer(
				p7Signer,
				fixture.certs,
				fixture.roots,
				fixture.content,
				fixture.content,
				true,
				false,
				false,
				model.CertifiedSigPermNone,
				0,
				result,
				fixture.ctx,
			); err != nil {
				t.Fatal(err)
			}
			if result.DocModified != model.Unknown {
				t.Fatalf("authentication failure concluded document integrity: %+v", result)
			}
		})
	}
}

// TestVerifyP7DigestEmbeddedBranches verifies embedded SHA-1 digests distinguish success from modification.
func TestVerifyP7DigestEmbeddedBranches(t *testing.T) {
	data := []byte("embedded content")
	digest := sha1.Sum(data)

	reason, err := verifyP7Digest(pkcs7.SignerInfo{}, digest[:], data, false)
	if err != nil {
		t.Fatalf("unexpected embedded digest failure: %v", err)
	}
	if reason != model.SignatureReasonDocNotModified {
		t.Fatalf("got reason %s, want %s", reason, model.SignatureReasonDocNotModified)
	}

	reason, err = verifyP7Digest(pkcs7.SignerInfo{}, digest[:], []byte("modified"), false)
	if err == nil {
		t.Fatal("expected embedded digest mismatch")
	}
	var mismatch *pkcs7.MessageDigestMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("expected MessageDigestMismatchError, got %v", err)
	}
	if reason != model.SignatureReasonDocModified {
		t.Fatalf("got reason %s, want %s", reason, model.SignatureReasonDocModified)
	}
}

// TestCheckPermsReportsEvidence verifies unsupported certified permissions remain signer evidence.
func TestCheckPermsReportsEvidence(t *testing.T) {
	signer := &model.Signer{
		Certified:   true,
		Permissions: model.CertifiedSigPermFillingAndSigningOK,
	}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	checkPerms(signer, result)

	if result.Reason != model.SignatureReasonUnsupported {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonUnsupported)
	}
	if len(signer.Problems) != 1 || signer.Problems[0] != CertifiedSigPermsNotSupported {
		t.Fatalf("got problems %v, want unsupported permissions evidence", signer.Problems)
	}
}

// TestValidateDTSReportsMissingTimestampInfo verifies non-RFC3161 content remains timestamp evidence.
func TestValidateDTSReportsMissingTimestampInfo(t *testing.T) {
	fixture := signerPKCS7(t, 1)
	result := &model.SignatureValidationResult{
		Reason:      model.SignatureReasonUnknown,
		Status:      model.SignatureStatusUnknown,
		DocModified: model.Unknown,
	}
	ctx := &model.Context{
		Configuration: model.NewDefaultConfiguration(),
		XRefTable:     &model.XRefTable{},
	}
	sigDict := types.Dict{"Contents": types.HexLiteral(hex.EncodeToString(fixture))}

	err := ValidateDTS(
		bytes.NewReader(nil),
		sigDict,
		false,
		false,
		true,
		0,
		x509.NewCertPool(),
		result,
		ctx,
	)
	if err != nil {
		t.Fatalf("expected timestamp evidence, got fatal error %v", err)
	}
	if result.Reason != model.SignatureReasonTimestampTokenInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonTimestampTokenInvalid)
	}
	if len(result.Details.Signers) != 1 {
		t.Fatalf("got %d signers, want 1", len(result.Details.Signers))
	}
	const want = "SubFilter ETSI.RFC3161: missing timestamp info"
	problems := result.Details.Signers[0].Problems
	if len(problems) != 1 || problems[0] != want {
		t.Fatalf("got problems %v, want %q", problems, want)
	}
}

// TestValidateDTSClassifiesMalformedPKCS7 verifies malformed timestamp containers remain timestamp evidence.
func TestValidateDTSClassifiesMalformedPKCS7(t *testing.T) {
	result := &model.SignatureValidationResult{
		Status: model.SignatureStatusUnknown,
		Reason: model.SignatureReasonUnknown,
	}
	ctx := &model.Context{
		Configuration: model.NewDefaultConfiguration(),
		XRefTable:     &model.XRefTable{},
	}
	sigDict := types.Dict{"Contents": types.HexLiteral("01")}

	err := ValidateDTS(
		bytes.NewReader(nil),
		sigDict,
		false,
		false,
		true,
		0,
		x509.NewCertPool(),
		result,
		ctx,
	)
	if err != nil {
		t.Fatalf("expected timestamp evidence, got fatal error %v", err)
	}
	if result.Reason != model.SignatureReasonMalformed {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonMalformed)
	}
	if problems := strings.Join(result.Problems, "\n"); !strings.Contains(problems, "pkcs7:") {
		t.Fatalf("unexpected problems %q", problems)
	}
}

// TestValidateDTSRejectsMultipleSigners verifies RFC 3161 timestamp tokens require one signer.
func TestValidateDTSRejectsMultipleSigners(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("multiple-signer RFC 3161 validation panicked: %v", recovered)
		}
	}()

	fixture := multipleSignerPKCS7(t)
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	ctx := &model.Context{
		Configuration: model.NewDefaultConfiguration(),
		XRefTable:     &model.XRefTable{},
	}
	sigDict := types.Dict{"Contents": types.HexLiteral(hex.EncodeToString(fixture))}

	err := ValidateDTS(
		bytes.NewReader(nil),
		sigDict,
		false,
		false,
		true,
		0,
		x509.NewCertPool(),
		result,
		ctx,
	)
	if err != nil {
		t.Fatalf("expected timestamp evidence, got fatal error %v", err)
	}
	if result.Reason != model.SignatureReasonTimestampTokenInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonTimestampTokenInvalid)
	}
	const want = "SubFilter ETSI.RFC3161: requires a single signer"
	if len(result.Problems) != 1 || result.Problems[0] != want {
		t.Fatalf("got problems %v, want %q", result.Problems, want)
	}
}

// TestDTSSignerCertificateReportsMissingEvidence verifies absent signer certificates do not panic.
func TestDTSSignerCertificateReportsMissingEvidence(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.False,
	}

	if cert := dtsSignerCertificate(nil, fixture.signer, signer, result); cert != nil {
		t.Fatalf("got signer certificate %v, want nil", cert)
	}
	if result.Status != model.SignatureStatusUnknown ||
		result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("got status=%s reason=%s, want unknown and certificate invalid", result.Status, result.Reason)
	}
	if problems := strings.Join(signer.Problems, "\n"); !strings.Contains(problems, "SubFilter ETSI.RFC3161: missing certificate for signer") {
		t.Fatalf("unexpected problems %q", problems)
	}
}

func TestDTSSignerCertificateReportsMissingIdentifierEvidence(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	p7Signer := fixture.signer
	p7Signer.IssuerAndSerialNumber.IssuerName = asn1.RawValue{}
	p7Signer.IssuerAndSerialNumber.SerialNumber = nil
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.False,
	}

	if cert := dtsSignerCertificate(fixture.certs, p7Signer, signer, result); cert != nil {
		t.Fatalf("got signer certificate %v, want nil", cert)
	}
	if result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertInvalid)
	}
	problems := strings.Join(signer.Problems, "\n")
	for _, want := range []string{"signer identifier", "missing signer identifier"} {
		if !strings.Contains(problems, want) {
			t.Errorf("got problems %q, want %q", problems, want)
		}
	}
}

// TestDTSSignatureErrorClassification verifies unsupported encodings remain distinct from cryptographic mismatch.
func TestDTSSignatureErrorClassification(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus model.SignatureStatus
		wantReason model.SignatureReason
		wantPhase  string
	}{
		{
			name:       "UnsupportedAlgorithm",
			err:        fmt.Errorf("select algorithm: %w", pkcs7.ErrUnsupportedAlgorithm),
			wantStatus: model.SignatureStatusUnknown,
			wantReason: model.SignatureReasonUnsupported,
			wantPhase:  "verify signature unsupported",
		},
		{
			name:       "AlgorithmMismatch",
			err:        fmt.Errorf("select algorithm: %w", pkcs7.ErrAlgorithmMismatch),
			wantStatus: model.SignatureStatusUnknown,
			wantReason: model.SignatureReasonUnsupported,
			wantPhase:  "verify signature unsupported",
		},
		{
			name:       "InvalidPSSParameters",
			err:        fmt.Errorf("select algorithm: %w", pkcs7.ErrInvalidPSSParameters),
			wantStatus: model.SignatureStatusUnknown,
			wantReason: model.SignatureReasonMalformed,
			wantPhase:  "verify signature malformed",
		},
		{
			name:       "MalformedSignedAttributes",
			err:        fmt.Errorf("signed attributes: %w", pkcs7.ErrMalformedAttribute),
			wantStatus: model.SignatureStatusUnknown,
			wantReason: model.SignatureReasonMalformed,
			wantPhase:  "verify signature malformed",
		},
		{
			name:       "MalformedASN1",
			err:        asn1.StructuralError{Msg: "malformed signature"},
			wantStatus: model.SignatureStatusUnknown,
			wantReason: model.SignatureReasonMalformed,
			wantPhase:  "verify signature malformed",
		},
		{
			name: "ContentDigestMismatch",
			err: &pkcs7.MessageDigestMismatchError{
				ExpectedDigest: []byte{1},
				ActualDigest:   []byte{2},
			},
			wantStatus: model.SignatureStatusInvalid,
			wantReason: model.SignatureReasonTimestampTokenInvalid,
			wantPhase:  "verify signature content mismatch",
		},
		{
			name:       "CryptographicMismatch",
			err:        fmt.Errorf("%w: %w", pkcs7.ErrSignatureMismatch, rsa.ErrVerification),
			wantStatus: model.SignatureStatusInvalid,
			wantReason: model.SignatureReasonSignatureForged,
			wantPhase:  "verify signature failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := &model.Signer{}
			result := &model.SignatureValidationResult{
				Status:      model.SignatureStatusUnknown,
				Reason:      model.SignatureReasonUnknown,
				DocModified: model.Unknown,
			}

			reportDTSSignatureError(tt.err, signer, result)

			if result.Status != tt.wantStatus || result.Reason != tt.wantReason {
				t.Fatalf(
					"got status=%s reason=%s, want %s and %s",
					result.Status,
					result.Reason,
					tt.wantStatus,
					tt.wantReason,
				)
			}
			if result.DocModified != model.Unknown {
				t.Fatalf("CMS failure concluded document integrity: %+v", result)
			}
			problems := strings.Join(signer.Problems, "\n")
			for _, want := range []string{"SubFilter ETSI.RFC3161", tt.wantPhase, tt.err.Error()} {
				if !strings.Contains(problems, want) {
					t.Errorf("expected %q, got %q", want, problems)
				}
			}
		})
	}
}

func TestBuildP7CertChainsReportsNilIntermediateWithoutPanic(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	chains := buildP7CertChains(
		false,
		fixture.cert,
		[]*x509.Certificate{nil, fixture.cert},
		fixture.roots,
		signer,
		result,
	)
	if len(chains) != 0 {
		t.Fatalf("got %d chains, want none", len(chains))
	}
	if result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertInvalid)
	}
	problems := strings.Join(signer.Problems, "\n")
	if !strings.Contains(problems, "intermediate certificate index 1 is missing") {
		t.Fatalf("got problems %q, want indexed missing-intermediate evidence", problems)
	}
}

// TestParseTSTInfoPreservesASN1Cause verifies timestamp construction errors retain their lower-level cause.
func TestParseTSTInfoPreservesASN1Cause(t *testing.T) {
	_, err := parseTSTInfo([]byte{1})
	requireErrorsIsWrappedCause(t, err)
}

// TestDTSDigestMismatchIsAppliedAfterObservation verifies timestamp digest
// calculation does not mutate document-integrity conclusions.
func TestDTSDigestMismatchIsAppliedAfterObservation(t *testing.T) {
	var tstInfo TSTInfo
	tstInfo.MessageImprint.HashAlgorithm.Algorithm = pkcs7.OIDDigestAlgorithmSHA256
	tstInfo.MessageImprint.HashedMessage = []byte("wrong digest")
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{
		Reason:      model.SignatureReasonUnknown,
		Status:      model.SignatureStatusUnknown,
		DocModified: model.Unknown,
	}

	err := verifyDTSDigest(&tstInfo, []byte("signed data"))
	if err == nil {
		t.Fatal("expected timestamp digest mismatch")
	}
	if result.DocModified != model.Unknown ||
		result.Status != model.SignatureStatusUnknown ||
		result.Reason != model.SignatureReasonUnknown ||
		len(signer.Problems) != 0 {
		t.Fatalf("digest observation mutated result: signer=%+v result=%+v", signer, result)
	}
	if applyDTSDigestEvidence(err, signer, result) {
		t.Fatal("expected timestamp digest rejection")
	}
	if result.Status != model.SignatureStatusInvalid ||
		result.Reason != model.SignatureReasonDocModified ||
		result.DocModified != model.True {
		t.Fatalf("unexpected result state: status=%s reason=%s modified=%v", result.Status, result.Reason, result.DocModified)
	}
	if problems := strings.Join(signer.Problems, "\n"); !strings.Contains(problems, "SubFilter ETSI.RFC3161: message digest mismatch") {
		t.Fatalf("unexpected problems %q", problems)
	}
}

// TestDTSDigestUnsupportedAlgorithmIsAppliedAfterObservation verifies
// unsupported timestamp digest evidence remains a deferred conclusion.
func TestDTSDigestUnsupportedAlgorithmIsAppliedAfterObservation(t *testing.T) {
	var tstInfo TSTInfo
	tstInfo.MessageImprint.HashAlgorithm.Algorithm = asn1.ObjectIdentifier{1, 2, 3}
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	err := verifyDTSDigest(&tstInfo, []byte("signed data"))
	if err == nil {
		t.Fatal("expected unsupported timestamp digest algorithm")
	}
	if result.Reason != model.SignatureReasonUnknown || len(signer.Problems) != 0 {
		t.Fatalf("digest observation mutated result: signer=%+v result=%+v", signer, result)
	}
	if applyDTSDigestEvidence(err, signer, result) {
		t.Fatal("expected unsupported timestamp digest rejection")
	}
	if result.Reason != model.SignatureReasonUnsupported {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonUnsupported)
	}
	if problems := strings.Join(signer.Problems, "\n"); !strings.Contains(problems, "SubFilter ETSI.RFC3161: verify message digest") {
		t.Fatalf("unexpected problems %q", problems)
	}
}

// TestValidateDTSCertRejectsExpiredCertificate verifies invalid TSA evidence clears stale timestamp state.
func TestValidateDTSCertRejectsExpiredCertificate(t *testing.T) {
	signingTime := time.Now()
	key := testRSAKey(t)
	template := testCertTemplate("Expired DTS TSA", true)
	template.NotBefore = signingTime.Add(-2 * time.Hour)
	template.NotAfter = signingTime.Add(-time.Hour)
	cert := testCertificate(t, template, template, &key.PublicKey, key)
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	ctx := &model.Context{
		Configuration: model.NewDefaultConfiguration(),
		XRefTable: &model.XRefTable{
			DTS: signingTime.Add(-24 * time.Hour),
		},
	}

	pathValidated, err := validateDTSCert(
		cert,
		[]*x509.Certificate{cert},
		roots,
		nil,
		nil,
		signer,
		result,
		ctx,
	)
	if err != nil {
		t.Fatal(err)
	}
	if pathValidated {
		t.Fatal("expired TSA path was locally validated")
	}
	finalizeDTSValidationResult(result, ctx, timestampEvidence{
		Kind:              timestampKindDocument,
		SigningTime:       signingTime,
		Present:           true,
		DigestVerified:    true,
		SignatureVerified: true,
		CorrectProfile:    true,
	}, completedLocalSignatureAssessment())
	if result.Reason != model.SignatureReasonCertExpired {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertExpired)
	}
	if !ctx.DTS.IsZero() {
		t.Fatalf("stale document timestamp retained: %s", ctx.DTS)
	}
	if problems := strings.Join(signer.Problems, "\n"); !strings.Contains(problems, "certificate verification failed") ||
		!strings.Contains(problems, "certificate has expired") {
		t.Fatalf("unexpected problems %q", problems)
	}
}

// TestReadAndCloseResponsePreservesCloseFailure verifies successful reads still report cleanup failures promptly.
func TestReadAndCloseResponsePreservesCloseFailure(t *testing.T) {
	closeCause := errors.New("close response")
	body := &signReadCloser{Reader: strings.NewReader("response"), closeErr: closeCause}
	resp := &http.Response{StatusCode: http.StatusOK, Body: body}

	_, err := readAndCloseResponse(resp)
	if !body.closed {
		t.Fatal("response body was not closed before return")
	}
	if !errors.Is(err, closeCause) {
		t.Fatalf("expected close cause, got %v", err)
	}
}

// TestReadAndCloseResponsePreservesReadAndCloseFailures verifies joined response failures remain discoverable.
func TestReadAndCloseResponsePreservesReadAndCloseFailures(t *testing.T) {
	readCause := errors.New("read response")
	closeCause := errors.New("close response")
	body := &signReadCloser{Reader: signErrorReader{err: readCause}, closeErr: closeCause}
	resp := &http.Response{StatusCode: http.StatusOK, Body: body}

	_, err := readAndCloseResponse(resp)
	if !body.closed {
		t.Fatal("response body was not closed before return")
	}
	for _, cause := range []error{readCause, closeCause} {
		if !errors.Is(err, cause) {
			t.Errorf("expected cause %v, got %v", cause, err)
		}
	}
}

// TestReadAndCloseResponsePreservesStatusAndCloseFailures verifies rejected responses are also closed.
func TestReadAndCloseResponsePreservesStatusAndCloseFailures(t *testing.T) {
	closeCause := errors.New("close response")
	body := &signReadCloser{Reader: strings.NewReader("response"), closeErr: closeCause}
	resp := &http.Response{StatusCode: http.StatusBadGateway, Body: body}

	_, err := readAndCloseResponse(resp)
	if !body.closed {
		t.Fatal("response body was not closed before return")
	}
	if !errors.Is(err, closeCause) {
		t.Fatalf("expected close cause, got %v", err)
	}
	if want := "unexpected HTTP status: 502"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

// TestReadAndCloseResponseAcceptsBodyAtLimit verifies that the response limit is inclusive.
func TestReadAndCloseResponseAcceptsBodyAtLimit(t *testing.T) {
	const bodyContent = "response"
	body := &signReadCloser{Reader: strings.NewReader(bodyContent)}
	resp := &http.Response{StatusCode: http.StatusOK, Body: body}

	bb, err := readAndCloseResponseWithLimit(resp, int64(len(bodyContent)))
	if err != nil {
		t.Fatal(err)
	}
	if !body.closed {
		t.Fatal("response body was not closed before return")
	}
	if string(bb) != bodyContent {
		t.Fatalf("response body: got %q, want %q", bb, bodyContent)
	}
}

// TestReadAndCloseResponseRejectsOversizedBody verifies that chunked responses cannot bypass the response limit.
func TestReadAndCloseResponseRejectsOversizedBody(t *testing.T) {
	body := &signReadCloser{Reader: strings.NewReader("response")}
	resp := &http.Response{StatusCode: http.StatusOK, Body: body, ContentLength: -1}

	bb, err := readAndCloseResponseWithLimit(resp, 4)
	if !body.closed {
		t.Fatal("response body was not closed before return")
	}
	if bb != nil {
		t.Fatalf("expected no response body, got %q", bb)
	}
	if !errors.Is(err, errRevocationResponseTooLarge) {
		t.Fatalf("expected oversized response error, got %v", err)
	}
}

// TestReadAndCloseResponseRejectsOversizedContentLength verifies known oversized responses fail before reading.
func TestReadAndCloseResponseRejectsOversizedContentLength(t *testing.T) {
	body := &signReadCloser{Reader: signErrorReader{err: errors.New("body must not be read")}}
	resp := &http.Response{StatusCode: http.StatusOK, Body: body, ContentLength: 5}

	_, err := readAndCloseResponseWithLimit(resp, 4)
	if !body.closed {
		t.Fatal("response body was not closed before return")
	}
	if !errors.Is(err, errRevocationResponseTooLarge) {
		t.Fatalf("expected oversized response error, got %v", err)
	}
}

// TestReadDTSSignedDataReportsMalformedByteRange verifies document timestamp
// ByteRange defects remain reportable evidence.
func TestReadDTSSignedDataReportsMalformedByteRange(t *testing.T) {
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	data, ok, err := readDTSSignedData(bytes.NewReader(nil), types.Dict{}, result)
	if err != nil {
		t.Fatalf("expected reportable ByteRange evidence, got %v", err)
	}
	if ok || data != nil {
		t.Fatalf("got data %v and ok=%t for malformed ByteRange", data, ok)
	}
	if result.Reason != model.SignatureReasonMalformed {
		t.Fatalf("got reason %s, want malformed", result.Reason)
	}
	if len(result.Problems) != 1 ||
		!strings.Contains(result.Problems[0], "signature dict entry ByteRange") {
		t.Fatalf("unexpected DTS Problems: %v", result.Problems)
	}
}

// TestReadDTSSignedDataPreservesFatalIO verifies document timestamp positional
// read failures remain fatal and retain their cause.
func TestReadDTSSignedDataPreservesFatalIO(t *testing.T) {
	cause := errors.New("storage unavailable")
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	sigDict := types.Dict{
		"ByteRange": byteRange(
			types.Integer(0),
			types.Integer(1),
			types.Integer(5),
			types.Integer(0),
		),
		"Contents": types.HexLiteral("aa"),
	}

	data, ok, err := readDTSSignedData(signErrorReader{err: cause}, sigDict, result)
	if err == nil {
		t.Fatal("expected fatal DTS signed-data I/O error")
	}
	if ok || data != nil {
		t.Fatalf("got data %v and ok=%t for fatal I/O", data, ok)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, %v) = false", err, cause)
	}
	if len(result.Problems) != 0 {
		t.Fatalf("fatal I/O error was converted into Problems: %v", result.Problems)
	}
}

func requireErrorsIsWrappedCause(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	cause := errors.Unwrap(err)
	if cause == nil {
		t.Fatalf("expected wrapped cause, got %v", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, %v) = false", err, cause)
	}
}

func requireP1ValidationProblem(t *testing.T, sigDict types.Dict, want string) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("legacy signature validation panicked: %v", recovered)
		}
	}()

	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	ctx := &model.Context{Configuration: model.NewDefaultConfiguration()}
	err := ValidateX509RSASHA1Signature(
		bytes.NewReader(nil),
		sigDict,
		false,
		false,
		true,
		0,
		x509.NewCertPool(),
		result,
		ctx,
	)
	if err != nil {
		t.Fatalf("expected reportable validation problem, got %v", err)
	}
	if result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertInvalid)
	}
	if problems := strings.Join(result.Problems, "\n"); !strings.Contains(problems, want) {
		t.Fatalf("expected problem containing %q, got %q", want, problems)
	}
}

// TestValidateX509RSASHA1SignatureRequiresRevocationConclusion verifies
// cryptographic success alone does not establish local validity.
func TestValidateX509RSASHA1SignatureRequiresRevocationConclusion(t *testing.T) {
	key := testRSAKey(t)
	template := testCertTemplate("Legacy RSA Signer", true)
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("legacy signed data")
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
	sigDict := types.Dict{
		"ByteRange": byteRange(
			types.Integer(0),
			types.Integer(len(data)),
			types.Integer(len(file)),
			types.Integer(0),
		),
		"Cert":     types.Array{types.HexLiteral(hex.EncodeToString(der))},
		"Contents": contentsLiteral,
	}
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	result := &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}
	ctx := &model.Context{Configuration: model.NewDefaultConfiguration()}

	err = ValidateX509RSASHA1Signature(
		bytes.NewReader(file),
		sigDict,
		false,
		false,
		true,
		0,
		roots,
		result,
		ctx,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != model.SignatureStatusUnknown ||
		result.Reason != model.SignatureReasonUnknown ||
		result.DocModified != model.False {
		t.Fatalf(
			"got status=%s reason=%s modified=%d, want unknown, no reason, false",
			result.Status,
			result.Reason,
			result.DocModified,
		)
	}
}

// TestValidateX509RSASHA1SignatureRejectsEmptyCertArray verifies empty legacy certificate arrays do not panic.
func TestValidateX509RSASHA1SignatureRejectsEmptyCertArray(t *testing.T) {
	requireP1ValidationProblem(t, types.Dict{"Cert": types.Array{}}, "legacy certificate: signature dict entry Cert: empty array")
}

// TestValidateX509RSASHA1SignatureRejectsMalformedCertificateContainers verifies malformed Cert evidence does not panic.
func TestValidateX509RSASHA1SignatureRejectsMalformedCertificateContainers(t *testing.T) {
	tests := []struct {
		name    string
		sigDict types.Dict
		want    string
	}{
		{
			name:    "Missing",
			sigDict: types.Dict{},
			want:    "legacy certificate: signature dict entry Cert: missing",
		},
		{
			name: "NilArrayEntry",
			sigDict: types.Dict{
				"Cert": types.Array{nil},
			},
			want: "legacy certificate: signature dict entry Cert, array index 1: parse certificate: unsupported certificate object type <nil>",
		},
		{
			name: "MalformedDER",
			sigDict: types.Dict{
				"Cert": types.Array{types.HexLiteral("00")},
			},
			want: "legacy certificate: signature dict entry Cert, array index 1: parse certificate: certificate parse error",
		},
		{
			name: "UnsupportedTopLevelType",
			sigDict: types.Dict{
				"Cert": types.Integer(1),
			},
			want: "legacy certificate: signature dict entry Cert: unsupported type types.Integer",
		},
		{
			name: "UnsupportedArrayEntryType",
			sigDict: types.Dict{
				"Cert": types.Array{types.Integer(1)},
			},
			want: "legacy certificate: signature dict entry Cert, array index 1: parse certificate: unsupported certificate object type types.Integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireP1ValidationProblem(t, tt.sigDict, tt.want)
		})
	}
}

// TestValidateX509RSASHA1SignatureRejectsNonRSACertificate verifies non-RSA legacy certificates do not panic.
func TestValidateX509RSASHA1SignatureRejectsNonRSACertificate(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := testCertTemplate("ECDSA Signer", false)
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert := types.HexLiteral(hex.EncodeToString(der))
	requireP1ValidationProblem(
		t,
		types.Dict{"Cert": types.Array{cert}},
		"legacy certificate: signature dict entry Cert, array index 1: SubFilter adbe.x509.rsa_sha1 requires an RSA public key",
	)
}

// TestP1SigningCertificateRejectsInvalidRSAKeys verifies malformed public keys return contextual evidence.
func TestP1SigningCertificateRejectsInvalidRSAKeys(t *testing.T) {
	tests := []struct {
		name string
		cert *x509.Certificate
		want string
	}{
		{
			name: "MissingCertificate",
			want: "signature dict entry Cert, array index 1: missing certificate",
		},
		{
			name: "TypedNilKey",
			cert: &x509.Certificate{PublicKey: (*rsa.PublicKey)(nil)},
			want: "signature dict entry Cert, array index 1: invalid RSA public key: missing key",
		},
		{
			name: "MissingModulus",
			cert: &x509.Certificate{PublicKey: &rsa.PublicKey{E: 65537}},
			want: "invalid RSA public key: missing modulus",
		},
		{
			name: "NonPositiveModulus",
			cert: &x509.Certificate{PublicKey: &rsa.PublicKey{N: big.NewInt(0), E: 65537}},
			want: "invalid RSA public key: modulus must be positive",
		},
		{
			name: "EvenModulus",
			cert: &x509.Certificate{PublicKey: &rsa.PublicKey{N: big.NewInt(4), E: 3}},
			want: "invalid RSA public key: modulus must be odd",
		},
		{
			name: "SmallExponent",
			cert: &x509.Certificate{PublicKey: &rsa.PublicKey{N: big.NewInt(3), E: 1}},
			want: "invalid RSA public key: invalid public exponent 1",
		},
		{
			name: "EvenExponent",
			cert: &x509.Certificate{PublicKey: &rsa.PublicKey{N: big.NewInt(3), E: 4}},
			want: "invalid RSA public key: invalid public exponent 4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("p1SigningCertificate panicked: %v", recovered)
				}
			}()

			_, _, err := p1SigningCertificate([]*x509.Certificate{tt.cert})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
		})
	}
}

// TestNormalizedLegacyCertificateAndOCSPWording verifies stable signing evidence terminology.
func TestNormalizedLegacyCertificateAndOCSPWording(t *testing.T) {
	_, err := parseP1Certificates(types.Dict{})
	if want := "signature dict entry Cert: missing"; err == nil || err.Error() != want {
		t.Errorf("legacy certificate: got %v, want %q", err, want)
	}

	_, err = checkCertViaOCSP(
		&x509.Certificate{},
		&x509.Certificate{},
		x509.NewCertPool(),
		nil,
		&model.Configuration{Offline: true},
	)
	if want := "OCSP: offline"; err == nil || err.Error() != want {
		t.Errorf("offline OCSP: got %v, want %q", err, want)
	}

	_, err = checkCertViaOCSP(
		&x509.Certificate{},
		&x509.Certificate{},
		x509.NewCertPool(),
		nil,
		&model.Configuration{},
	)
	if want := "OCSP: certificate has no responder URL"; err == nil || err.Error() != want {
		t.Errorf("missing OCSP responder: got %v, want %q", err, want)
	}
}

// TestCheckResponderCertAcceptsIssuerSignedResponseWithoutEmbeddedCertificate verifies issuer-signed OCSP responses do not require a delegated responder certificate.
func TestCheckResponderCertAcceptsIssuerSignedResponseWithoutEmbeddedCertificate(t *testing.T) {
	issuerKey := testRSAKey(t)
	issuerTemplate := testCertTemplate("OCSP Issuer", true)
	issuer := testCertificate(t, issuerTemplate, issuerTemplate, &issuerKey.PublicKey, issuerKey)
	resp := testIssuerOCSPResponse(t, issuer, issuerKey)
	if resp.Certificate != nil {
		t.Fatal("expected OCSP response without an embedded responder certificate")
	}

	if err := checkResponderCert(resp, issuer, time.Now()); err != nil {
		t.Fatalf("issuer-signed OCSP response: %v", err)
	}
}

// TestCheckResponderCertAcceptsEmbeddedIssuerCertificate verifies an embedded issuer remains a direct CA signer.
func TestCheckResponderCertAcceptsEmbeddedIssuerCertificate(t *testing.T) {
	issuerKey := testRSAKey(t)
	issuerTemplate := testCertTemplate("OCSP Issuer", true)
	issuer := testCertificate(t, issuerTemplate, issuerTemplate, &issuerKey.PublicKey, issuerKey)
	resp := testEmbeddedOCSPResponse(t, issuer, issuer, issuerKey)
	if resp.Certificate == nil || !resp.Certificate.Equal(issuer) {
		t.Fatal("expected embedded issuer certificate")
	}

	if err := checkResponderCert(resp, issuer, time.Now()); err != nil {
		t.Fatalf("embedded issuer-signed OCSP response: %v", err)
	}
}

// TestCheckResponderCertAcceptsValidDelegatedResponder verifies an issuer-authorized responder with the OCSP signing EKU.
func TestCheckResponderCertAcceptsValidDelegatedResponder(t *testing.T) {
	issuerKey := testRSAKey(t)
	issuerTemplate := testCertTemplate("OCSP Issuer", true)
	issuer := testCertificate(t, issuerTemplate, issuerTemplate, &issuerKey.PublicKey, issuerKey)
	responderKey := testRSAKey(t)
	responderTemplate := testCertTemplate("Delegated OCSP Responder", false)
	responderTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning}
	responder := testCertificate(t, responderTemplate, issuer, &responderKey.PublicKey, issuerKey)
	resp := testEmbeddedOCSPResponse(t, issuer, responder, responderKey)

	if err := checkResponderCert(resp, issuer, time.Now()); err != nil {
		t.Fatalf("valid delegated responder: %v", err)
	}
}

// TestCheckResponderCertRejectsDelegatedResponderWithoutOCSPSigningEKU verifies delegated responders require explicit authorization.
func TestCheckResponderCertRejectsDelegatedResponderWithoutOCSPSigningEKU(t *testing.T) {
	issuerKey := testRSAKey(t)
	issuerTemplate := testCertTemplate("OCSP Issuer", true)
	issuer := testCertificate(t, issuerTemplate, issuerTemplate, &issuerKey.PublicKey, issuerKey)
	responderKey := testRSAKey(t)
	responderTemplate := testCertTemplate("Delegated OCSP Responder", false)
	responder := testCertificate(t, responderTemplate, issuer, &responderKey.PublicKey, issuerKey)
	resp := testEmbeddedOCSPResponse(t, issuer, responder, responderKey)

	err := checkResponderCert(resp, issuer, time.Now())
	if want := "OCSP: responder certificate missing OCSP signing EKU"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("delegated responder without OCSP signing EKU: got %v, want context %q", err, want)
	}
}

// TestCheckResponderCertRejectsInvalidResponseSignature verifies response-signature failures remain reportable.
func TestCheckResponderCertRejectsInvalidResponseSignature(t *testing.T) {
	issuerKey := testRSAKey(t)
	issuerTemplate := testCertTemplate("OCSP Issuer", true)
	issuer := testCertificate(t, issuerTemplate, issuerTemplate, &issuerKey.PublicKey, issuerKey)
	resp := testIssuerOCSPResponse(t, issuer, issuerKey)
	resp.Signature[0] ^= 0xff

	err := checkResponderCert(resp, issuer, time.Now())
	if err == nil || !strings.Contains(err.Error(), "OCSP: verify response signature") {
		t.Fatalf("invalid OCSP response signature: got %v", err)
	}
	if !errors.Is(err, rsa.ErrVerification) {
		t.Fatalf("expected RSA verification cause, got %v", err)
	}
}

// TestCheckResponderCertRejectsMissingResponderAndIssuer verifies signer resolution fails cleanly.
func TestCheckResponderCertRejectsMissingResponderAndIssuer(t *testing.T) {
	err := checkResponderCert(&ocsp.Response{}, nil, time.Now())
	if want := "OCSP: responder certificate unavailable"; err == nil || err.Error() != want {
		t.Fatalf("missing responder and issuer: got %v, want %q", err, want)
	}
}

// TestCheckResponderCertDoesNotPanicWithoutEmbeddedCertificate verifies nil embedded responder handling.
func TestCheckResponderCertDoesNotPanicWithoutEmbeddedCertificate(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("checkResponderCert panicked: %v", recovered)
		}
	}()

	issuerKey := testRSAKey(t)
	issuerTemplate := testCertTemplate("OCSP Issuer", true)
	issuer := testCertificate(t, issuerTemplate, issuerTemplate, &issuerKey.PublicKey, issuerKey)
	resp := testIssuerOCSPResponse(t, issuer, issuerKey)
	if resp.Certificate != nil {
		t.Fatal("expected OCSP response without an embedded responder certificate")
	}
	if err := checkResponderCert(resp, issuer, time.Now()); err != nil {
		t.Fatalf("issuer-signed OCSP response: %v", err)
	}
}

// TestCheckResponderCertAcceptsAuthorizedNoCheckResponder verifies OCSP No Check suppresses revocation checking without invalidating an authorized responder.
func TestCheckResponderCertAcceptsAuthorizedNoCheckResponder(t *testing.T) {
	issuerKey := testRSAKey(t)
	issuerTemplate := testCertTemplate("OCSP Issuer", true)
	issuer := testCertificate(t, issuerTemplate, issuerTemplate, &issuerKey.PublicKey, issuerKey)

	responderKey := testRSAKey(t)
	responderTemplate := testCertTemplate("Delegated OCSP Responder", false)
	responderTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning}
	responderTemplate.ExtraExtensions = []pkix.Extension{{
		Id:    oidOCSPNoCheck,
		Value: []byte{0x05, 0x00},
	}}
	responder := testCertificate(t, responderTemplate, issuer, &responderKey.PublicKey, issuerKey)
	resp := testEmbeddedOCSPResponse(t, issuer, responder, responderKey)

	if err := checkResponderCert(resp, issuer, time.Now()); err != nil {
		t.Fatalf("authorized OCSP No Check responder: %v", err)
	}
}

// TestCheckResponderCertRejectsUnrelatedDelegatedResponder verifies a delegated responder must be issued directly by the certificate issuer.
func TestCheckResponderCertRejectsUnrelatedDelegatedResponder(t *testing.T) {
	issuerKey := testRSAKey(t)
	issuerTemplate := testCertTemplate("Certificate Issuer", true)
	issuer := testCertificate(t, issuerTemplate, issuerTemplate, &issuerKey.PublicKey, issuerKey)

	responderKey := testRSAKey(t)
	responderTemplate := testCertTemplate("Unrelated OCSP Responder", true)
	responderTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning}
	responder := testCertificate(
		t,
		responderTemplate,
		responderTemplate,
		&responderKey.PublicKey,
		responderKey,
	)
	resp := testEmbeddedOCSPResponse(t, issuer, responder, responderKey)

	err := checkResponderCert(resp, issuer, time.Now())
	if err == nil || !strings.Contains(err.Error(), "OCSP: authorize delegated responder") {
		t.Fatalf("unrelated delegated responder: got %v", err)
	}
	var unknownAuthorityErr x509.UnknownAuthorityError
	if !errors.As(err, &unknownAuthorityErr) {
		t.Fatalf("expected unknown-authority cause, got %v", err)
	}
}

// TestCheckResponderCertRejectsTrustedUnrelatedDelegatedResponder verifies general trust does not authorize an OCSP responder.
func TestCheckResponderCertRejectsTrustedUnrelatedDelegatedResponder(t *testing.T) {
	issuerKey := testRSAKey(t)
	issuerTemplate := testCertTemplate("Certificate Issuer", true)
	issuer := testCertificate(t, issuerTemplate, issuerTemplate, &issuerKey.PublicKey, issuerKey)

	rootKey := testRSAKey(t)
	rootTemplate := testCertTemplate("Trusted Responder Root", true)
	root := testCertificate(t, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	responderKey := testRSAKey(t)
	responderTemplate := testCertTemplate("Trusted OCSP Responder", false)
	responderTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning}
	responder := testCertificate(t, responderTemplate, root, &responderKey.PublicKey, rootKey)
	resp := testEmbeddedOCSPResponse(t, issuer, responder, responderKey)

	roots := x509.NewCertPool()
	roots.AddCert(root)
	if _, err := responder.Verify(x509.VerifyOptions{
		Roots:       roots,
		CurrentTime: time.Now(),
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning},
	}); err != nil {
		t.Fatalf("responder does not chain to general trust pool: %v", err)
	}
	err := checkResponderCert(resp, issuer, time.Now())
	if err == nil || !strings.Contains(err.Error(), "OCSP: authorize delegated responder") {
		t.Fatalf("trusted but unrelated delegated responder: got %v", err)
	}
	var unknownAuthorityErr x509.UnknownAuthorityError
	if !errors.As(err, &unknownAuthorityErr) {
		t.Fatalf("expected unknown-authority cause, got %v", err)
	}
}

// TestArchivedOCSPDelegatedResponderValidityRemainsUnknown verifies an
// archived delegated responder is not validated at its asserted ProducedAt.
func TestArchivedOCSPDelegatedResponderValidityRemainsUnknown(t *testing.T) {
	now := time.Now().UTC()
	signingTime := now.Add(time.Hour)
	cert, issuer, responder, bb := testArchivedOCSPFixture(
		t,
		now.Add(-time.Hour),
		now.Add(-time.Nanosecond),
		signingTime.Add(time.Hour),
	)
	resp, err := ocsp.ParseResponse(bb, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !time.Now().After(responder.NotAfter) {
		t.Fatalf("responder certificate is not expired now: NotAfter=%v", responder.NotAfter)
	}
	if resp.ProducedAt.Before(responder.NotBefore) || resp.ProducedAt.After(responder.NotAfter) {
		t.Fatalf("responder certificate is not valid at ProducedAt %v", resp.ProducedAt)
	}

	details, err := processArchivedOCSPResponses(
		cert,
		issuer,
		[][]byte{bb},
	)
	if err != nil {
		t.Fatalf("archived OCSP response: %v", err)
	}
	if !isUnknownArchivedDelegatedResponderEvidence(details) {
		t.Fatalf("delegated responder produced a historical conclusion: %+v", details)
	}
}

func isUnknownArchivedDelegatedResponderEvidence(details *model.RevocationDetails) bool {
	if details == nil ||
		details.Status != model.Unknown ||
		details.Reason != "OCSP: archived response applicability unavailable" {
		return false
	}
	ocspEvidence := details.OCSP
	return ocspEvidence != nil &&
		ocspEvidence.Responder == model.OCSPResponderDelegated &&
		ocspEvidence.Authenticated == model.Unknown &&
		ocspEvidence.Applicable == model.Unknown &&
		ocspEvidence.ResponseSignatureValid == model.True &&
		ocspEvidence.ResponderCertificateIssuedByIssuer == model.True &&
		ocspEvidence.ResponderCertificateOCSPSigningValid == model.True &&
		ocspEvidence.ResponderRevocation == model.Unknown
}

// TestArchivedOCSPNoCheckResponderRemainsInconclusive verifies OCSP No Check
// does not supply historical applicability or certificate validity.
func TestArchivedOCSPNoCheckResponderRemainsInconclusive(t *testing.T) {
	issuer, issuerKey, cert := testOCSPIssuerAndCertificate(t, "Certificate")
	responderKey := testRSAKey(t)
	responderTemplate := testCertTemplate("No Check OCSP Responder", false)
	responderTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning}
	responderTemplate.ExtraExtensions = []pkix.Extension{{
		Id:    oidOCSPNoCheck,
		Value: []byte{5, 0},
	}}
	responder := testCertificate(t, responderTemplate, issuer, &responderKey.PublicKey, issuerKey)
	signingTime := time.Now().Add(time.Minute)
	response := testOCSPResponseBytes(
		t,
		issuer,
		responder,
		responderKey,
		cert.SerialNumber,
		responder,
		signingTime.Add(-5*time.Minute),
		signingTime.Add(time.Hour),
	)

	details, err := processArchivedOCSPResponses(cert, issuer, [][]byte{response})
	if err != nil {
		t.Fatal(err)
	}
	if details.Status != model.Unknown ||
		details.OCSP == nil ||
		details.OCSP.Authenticated != model.Unknown ||
		details.OCSP.Applicable != model.Unknown ||
		details.OCSP.ResponseSignatureValid != model.True ||
		details.OCSP.ResponderCertificateIssuedByIssuer != model.True ||
		details.OCSP.ResponderCertificateOCSPSigningValid != model.True ||
		details.OCSP.ResponderRevocation != model.True {
		t.Fatalf("OCSP No Check observation: %+v", details)
	}
}

// TestArchivedOCSPDoesNotAssessResponderAtProducedAt verifies archived
// responder certificate validity remains unknown without an assessment time.
func TestArchivedOCSPDoesNotAssessResponderAtProducedAt(t *testing.T) {
	now := time.Now().UTC()
	signingTime := now.Add(2 * time.Hour)
	cert, issuer, responder, bb := testArchivedOCSPFixture(
		t,
		now.Add(time.Hour),
		now.Add(3*time.Hour),
		signingTime.Add(time.Hour),
	)
	resp, err := ocsp.ParseResponse(bb, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ProducedAt.Before(responder.NotBefore) {
		t.Fatalf("responder certificate is valid at ProducedAt %v", resp.ProducedAt)
	}
	if signingTime.Before(responder.NotBefore) || signingTime.After(responder.NotAfter) {
		t.Fatalf("responder certificate is not valid at signing time %v", signingTime)
	}

	details, err := processArchivedOCSPResponses(
		cert,
		issuer,
		[][]byte{bb},
	)
	if err != nil {
		t.Fatal(err)
	}
	if details == nil ||
		details.Status != model.Unknown ||
		details.OCSP == nil ||
		details.OCSP.Authenticated != model.Unknown ||
		details.OCSP.Applicable != model.Unknown {
		t.Fatalf("responder validity was inferred from ProducedAt: %+v", details)
	}
}

// TestCurrentOCSPRejectsResponseForDifferentCertificate verifies live responses are bound to the requested certificate.
func TestCurrentOCSPRejectsResponseForDifferentCertificate(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("processCurrentOCSPResponses panicked: %v", recovered)
		}
	}()

	issuer, issuerKey, certA := testOCSPIssuerAndCertificate(t, "Certificate A")
	certBKey := testRSAKey(t)
	certBTemplate := testCertTemplate("Certificate B", false)
	certB := testCertificate(t, certBTemplate, issuer, &certBKey.PublicKey, issuerKey)
	now := time.Now().UTC()
	bb := testOCSPResponseBytes(
		t,
		issuer,
		issuer,
		issuerKey,
		certB.SerialNumber,
		nil,
		now.Add(-time.Minute),
		now.Add(time.Hour),
	)
	certA.OCSPServer = []string{"https://ocsp.test"}

	_, err := processCurrentOCSPResponses(certA, issuer, testOCSPHTTPClient(bb))
	if err == nil || !strings.Contains(err.Error(), "OCSP: parse response for certificate") {
		t.Fatalf("certificate-mismatched OCSP response: got %v", err)
	}
	var parseErr ocsp.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected OCSP parse cause, got %v", err)
	}
}

// TestCurrentOCSPRecordsIssuerAuthenticationEvidence verifies live issuer-signed
// responses retain their source and responder authentication provenance.
func TestCurrentOCSPRecordsIssuerAuthenticationEvidence(t *testing.T) {
	issuer, issuerKey, cert := testOCSPIssuerAndCertificate(t, "Certificate")
	now := time.Now().UTC()
	bb := testOCSPResponseBytes(
		t,
		issuer,
		issuer,
		issuerKey,
		cert.SerialNumber,
		nil,
		now.Add(-time.Minute),
		now.Add(time.Hour),
	)
	cert.OCSPServer = []string{"https://ocsp.test"}

	details, err := processCurrentOCSPResponses(cert, issuer, testOCSPHTTPClient(bb))
	if err != nil {
		t.Fatalf("current OCSP response: %v", err)
	}
	if details == nil || details.Status != model.True {
		t.Fatalf("current OCSP status: %+v", details)
	}
	requireOCSPEvidence(
		t,
		details,
		model.RevocationEvidenceSourceOnline,
		model.OCSPResponderIssuer,
	)
}

// TestCheckCertViaOCSPRetainsInconclusiveArchivedEvidenceOffline verifies offline mode preserves observations.
func TestCheckCertViaOCSPRetainsInconclusiveArchivedEvidenceOffline(t *testing.T) {
	now := time.Now().UTC()
	signingTime := now.Add(time.Hour)
	cert, issuer, _, bb := testArchivedOCSPFixture(
		t,
		now.Add(-time.Hour),
		now.Add(2*time.Hour),
		signingTime.Add(time.Hour),
	)

	details, err := checkCertViaOCSP(
		cert,
		issuer,
		x509.NewCertPool(),
		[][]byte{bb},
		&model.Configuration{Offline: true},
	)
	if err == nil || !strings.Contains(err.Error(), "OCSP: offline") {
		t.Fatalf("offline archived OCSP response: %v", err)
	}
	if details == nil ||
		details.Status != model.Unknown ||
		details.Reason != "OCSP: archived response applicability unavailable" {
		t.Fatalf("offline archived OCSP status: %+v", details)
	}
}

// TestCheckCertViaOCSPOfflineRetainsArchivedFailure verifies unusable embedded evidence does not trigger network access.
func TestCheckCertViaOCSPOfflineRetainsArchivedFailure(t *testing.T) {
	issuer, _, cert := testOCSPIssuerAndCertificate(t, "Certificate")
	_, err := checkCertViaOCSP(
		cert,
		issuer,
		x509.NewCertPool(),
		[][]byte{{1}},
		&model.Configuration{Offline: true},
	)
	for _, want := range []string{"OCSP: offline", "OCSP: no valid archived response", "archived response 1"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q, got %v", want, err)
		}
	}
}

// TestArchivedOCSPSelectsValidResponseAfterMalformedCandidate verifies malformed evidence does not mask a later response.
func TestArchivedOCSPSelectsValidResponseAfterMalformedCandidate(t *testing.T) {
	issuer, issuerKey, cert := testOCSPIssuerAndCertificate(t, "Certificate")
	signingTime := time.Now().Add(time.Minute)
	valid := testOCSPResponseBytes(
		t,
		issuer,
		issuer,
		issuerKey,
		cert.SerialNumber,
		nil,
		signingTime.Add(-5*time.Minute),
		signingTime.Add(time.Hour),
	)

	details, err := processArchivedOCSPResponses(cert, issuer, [][]byte{{1}, valid})
	if err == nil || !strings.Contains(err.Error(), "archived response 1") {
		t.Fatalf("missing malformed archived response: %v", err)
	}
	if details == nil || details.Status != model.Unknown {
		t.Fatalf("archived OCSP status: %+v", details)
	}
	requireOCSPEvidence(
		t,
		details,
		model.RevocationEvidenceSourceArchived,
		model.OCSPResponderIssuer,
	)
	if len(details.OCSPs) != 2 ||
		details.OCSPs[0].Index != 1 ||
		details.OCSPs[0].Error == "" ||
		details.OCSPs[1].Index != 2 {
		t.Fatalf("indexed archived OCSP observations: %+v", details.OCSPs)
	}
}

// TestArchivedOCSPSelectsValidResponseAfterTemporalFailure verifies stale evidence does not mask a later response.
func TestArchivedOCSPSelectsValidResponseAfterTemporalFailure(t *testing.T) {
	issuer, issuerKey, cert := testOCSPIssuerAndCertificate(t, "Certificate")
	signingTime := time.Now().Add(time.Minute)
	expired := testOCSPResponseBytes(
		t,
		issuer,
		issuer,
		issuerKey,
		cert.SerialNumber,
		nil,
		signingTime.Add(-time.Hour),
		signingTime.Add(-5*time.Minute),
	)
	valid := testOCSPResponseBytes(
		t,
		issuer,
		issuer,
		issuerKey,
		cert.SerialNumber,
		nil,
		signingTime.Add(-5*time.Minute),
		signingTime.Add(time.Hour),
	)

	details, err := processArchivedOCSPResponses(cert, issuer, [][]byte{expired, valid})
	if err != nil {
		t.Fatal(err)
	}
	if details == nil || details.Status != model.Unknown || len(details.OCSPs) != 2 {
		t.Fatalf("archived OCSP status: %+v", details)
	}
}

// TestArchivedOCSPSelectsValidResponseAfterUnauthorizedResponder verifies authorization failure does not mask later evidence.
func TestArchivedOCSPSelectsValidResponseAfterUnauthorizedResponder(t *testing.T) {
	issuer, issuerKey, cert := testOCSPIssuerAndCertificate(t, "Certificate")
	signingTime := time.Now().Add(time.Minute)
	responderKey := testRSAKey(t)
	responderTemplate := testCertTemplate("Unauthorized OCSP Responder", false)
	responder := testCertificate(t, responderTemplate, issuer, &responderKey.PublicKey, issuerKey)
	unauthorized := testOCSPResponseBytes(
		t,
		issuer,
		responder,
		responderKey,
		cert.SerialNumber,
		responder,
		signingTime.Add(-5*time.Minute),
		signingTime.Add(time.Hour),
	)
	valid := testOCSPResponseBytes(
		t,
		issuer,
		issuer,
		issuerKey,
		cert.SerialNumber,
		nil,
		signingTime.Add(-5*time.Minute),
		signingTime.Add(time.Hour),
	)

	details, err := processArchivedOCSPResponses(cert, issuer, [][]byte{unauthorized, valid})
	if err == nil || !strings.Contains(err.Error(), "responder certificate missing OCSP signing EKU") {
		t.Fatalf("missing delegated responder evidence: %v", err)
	}
	if details == nil || details.Status != model.Unknown || len(details.OCSPs) != 2 {
		t.Fatalf("archived OCSP status: %+v", details)
	}
}

// TestArchivedOCSPContinuesPastUnknownResponse verifies an authenticated
// Unknown response does not mask a later conclusive response.
func TestArchivedOCSPContinuesPastUnknownResponse(t *testing.T) {
	issuer, issuerKey, cert := testOCSPIssuerAndCertificate(t, "Certificate")
	signingTime := time.Now().Add(time.Minute)
	thisUpdate := signingTime.Add(-5 * time.Minute)
	nextUpdate := signingTime.Add(time.Hour)
	unknown := testOCSPResponseBytesWithStatus(
		t,
		issuer,
		issuer,
		issuerKey,
		cert.SerialNumber,
		nil,
		thisUpdate,
		nextUpdate,
		ocsp.Unknown,
	)
	good := testOCSPResponseBytes(
		t,
		issuer,
		issuer,
		issuerKey,
		cert.SerialNumber,
		nil,
		thisUpdate,
		nextUpdate,
	)

	details, err := processArchivedOCSPResponses(cert, issuer, [][]byte{unknown, good})
	if err != nil {
		t.Fatal(err)
	}
	if details.Status != model.Unknown || len(details.OCSPs) != 2 {
		t.Fatalf("archived OCSP observations: %+v", details)
	}
	if details.OCSPs[0].CertificateStatus != model.Unknown ||
		details.OCSPs[0].Index != 1 ||
		details.OCSPs[0].Location != "archived response 1" ||
		details.OCSPs[1].CertificateStatus != model.True ||
		details.OCSPs[1].Index != 2 ||
		details.OCSPs[1].Location != "archived response 2" {
		t.Fatalf("indexed archived OCSP observations: %+v", details.OCSPs)
	}
}

// TestArchivedOCSPReportsAllMalformedCandidates verifies indexed parse causes survive total failure.
func TestArchivedOCSPReportsAllMalformedCandidates(t *testing.T) {
	issuer, _, cert := testOCSPIssuerAndCertificate(t, "Certificate")
	details, err := processArchivedOCSPResponses(cert, issuer, [][]byte{{1}, {2}})
	if err == nil {
		t.Fatal("expected archived OCSP failure")
	}
	for _, want := range []string{"OCSP: no valid archived response", "archived response 1", "archived response 2"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q, got %q", want, err)
		}
	}
	var syntaxErr asn1.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("expected ASN.1 cause, got %v", err)
	}
	if details == nil || details.Status != model.Unknown || len(details.OCSPs) != 2 {
		t.Fatalf("archived OCSP failure observations: %+v", details)
	}
	for i, evidence := range details.OCSPs {
		if evidence.Index != i+1 ||
			evidence.Location != fmt.Sprintf("archived response %d", i+1) ||
			evidence.Error == "" {
			t.Fatalf("archived OCSP observation %d: %+v", i+1, evidence)
		}
	}
}

// TestArchivedOCSPReportsDifferentCandidateFailures verifies independent failure causes retain candidate context.
func TestArchivedOCSPReportsDifferentCandidateFailures(t *testing.T) {
	issuer, issuerKey, cert := testOCSPIssuerAndCertificate(t, "Certificate")
	signingTime := time.Now().Add(time.Minute)
	expired := testOCSPResponseBytes(
		t,
		issuer,
		issuer,
		issuerKey,
		cert.SerialNumber,
		nil,
		signingTime.Add(-time.Hour),
		signingTime.Add(-5*time.Minute),
	)
	responderKey := testRSAKey(t)
	responderTemplate := testCertTemplate("Unauthorized OCSP Responder", false)
	responder := testCertificate(t, responderTemplate, issuer, &responderKey.PublicKey, issuerKey)
	unauthorized := testOCSPResponseBytes(
		t,
		issuer,
		responder,
		responderKey,
		cert.SerialNumber,
		responder,
		signingTime.Add(-5*time.Minute),
		signingTime.Add(time.Hour),
	)

	details, err := processArchivedOCSPResponses(cert, issuer, [][]byte{expired, unauthorized})
	if err == nil {
		t.Fatal("expected archived OCSP failure")
	}
	for _, want := range []string{
		"OCSP: no valid archived response",
		"archived response 2",
		"OCSP: responder certificate missing OCSP signing EKU",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q, got %q", want, err)
		}
	}
	if details == nil || details.Status != model.Unknown || len(details.OCSPs) != 2 {
		t.Fatalf("archived OCSP observations: %+v", details)
	}
}

// TestCurrentOCSPExhaustsResponderURLs verifies one unusable endpoint does not
// prevent a later responder from establishing a conclusion.
func TestCurrentOCSPExhaustsResponderURLs(t *testing.T) {
	issuer, issuerKey, cert := testOCSPIssuerAndCertificate(t, "Certificate")
	firstURL := "https://ocsp.test/first"
	secondURL := "https://ocsp.test/second"
	cert.OCSPServer = []string{firstURL, secondURL}
	now := time.Now()
	good := testOCSPResponseBytes(
		t,
		issuer,
		issuer,
		issuerKey,
		cert.SerialNumber,
		nil,
		now.Add(-time.Minute),
		now.Add(time.Hour),
	)

	details, err := processCurrentOCSPResponses(
		cert,
		issuer,
		testCurrentCRLClient(map[string][]byte{
			firstURL:  {1},
			secondURL: good,
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if details.Status != model.True || len(details.OCSPs) != 2 {
		t.Fatalf("current OCSP observations: %+v", details)
	}
	if details.OCSPs[0].Location != firstURL ||
		details.OCSPs[0].Index != 1 ||
		details.OCSPs[0].Error == "" ||
		details.OCSPs[1].Location != secondURL ||
		details.OCSPs[1].Index != 2 ||
		details.OCSPs[1].CertificateStatus != model.True {
		t.Fatalf("indexed current OCSP observations: %+v", details.OCSPs)
	}
}

// TestCheckCurrentOCSPResponseFreshness verifies deterministic local freshness policy.
func TestCheckCurrentOCSPResponseFreshness(t *testing.T) {
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		resp ocsp.Response
		want string
	}{
		{"RecentWithoutNextUpdate", ocsp.Response{ThisUpdate: now.Add(-time.Hour)}, ""},
		{"StaleWithoutNextUpdate", ocsp.Response{ThisUpdate: now.Add(-defaultOCSPResponseMaxAge - time.Minute)}, "OCSP: ThisUpdate exceeds maximum response age"},
		{"ZeroThisUpdate", ocsp.Response{}, "OCSP: ThisUpdate is missing"},
		{"FutureThisUpdateWithinSkew", ocsp.Response{ThisUpdate: now.Add(4 * time.Minute)}, ""},
		{"FutureThisUpdateBeyondSkew", ocsp.Response{ThisUpdate: now.Add(6 * time.Minute)}, "OCSP: ThisUpdate is in the future"},
		{"ExpiredNextUpdate", ocsp.Response{ThisUpdate: now.Add(-time.Hour), NextUpdate: now.Add(-time.Minute)}, "OCSP: NextUpdate precedes current time"},
		{"ValidNextUpdate", ocsp.Response{ThisUpdate: now.Add(-time.Hour), NextUpdate: now.Add(time.Hour)}, ""},
		{"FutureProducedAtBeyondSkew", ocsp.Response{ProducedAt: now.Add(6 * time.Minute), ThisUpdate: now}, "OCSP: ProducedAt is in the future"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkCurrentOCSPResponse(&tt.resp, now)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("unexpected freshness failure: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func testOCSPIssuerAndCertificate(
	t *testing.T,
	certificateName string,
) (*x509.Certificate, *rsa.PrivateKey, *x509.Certificate) {
	t.Helper()
	issuerKey := testRSAKey(t)
	issuerTemplate := testCertTemplate("OCSP Issuer", true)
	issuer := testCertificate(t, issuerTemplate, issuerTemplate, &issuerKey.PublicKey, issuerKey)
	certKey := testRSAKey(t)
	certTemplate := testCertTemplate(certificateName, false)
	cert := testCertificate(t, certTemplate, issuer, &certKey.PublicKey, issuerKey)
	return issuer, issuerKey, cert
}

func requireOCSPEvidence(
	t *testing.T,
	details *model.RevocationDetails,
	source model.RevocationEvidenceSource,
	responder model.OCSPResponder,
) {
	t.Helper()
	if details.OCSP == nil {
		t.Fatal("missing OCSP evidence")
	}
	evidence := details.OCSP
	if evidence.AssessmentScope != model.AssessmentScopeLocal ||
		evidence.Source != source ||
		evidence.Responder != responder ||
		evidence.Authenticated != model.True {
		t.Fatalf("unexpected OCSP evidence: %+v", evidence)
	}
}

func requireCRLEvidence(
	t *testing.T,
	details *model.RevocationDetails,
	status int,
	issuerMatched, signatureValid, applicable int,
) {
	t.Helper()
	if details == nil {
		t.Fatal("missing CRL revocation details")
	}
	if details.Status != status {
		t.Fatalf("got CRL status %d, want %d: %+v", details.Status, status, details)
	}
	if details.CRL == nil {
		t.Fatal("missing CRL evidence")
	}
	evidence := details.CRL
	if evidence.AssessmentScope != model.AssessmentScopeLocal ||
		evidence.Source != model.RevocationEvidenceSourceArchived ||
		evidence.IssuerMatched != issuerMatched ||
		evidence.SignatureValid != signatureValid ||
		evidence.Applicable != applicable {
		t.Fatalf("unexpected CRL evidence: %+v", evidence)
	}
}

func testCurrentCRLChain(
	t *testing.T,
	issuerName string,
) (*x509.Certificate, *rsa.PrivateKey, *x509.Certificate) {
	t.Helper()
	issuerKey := testRSAKey(t)
	issuerTemplate := testCertTemplate(issuerName, true)
	issuerTemplate.KeyUsage |= x509.KeyUsageCRLSign
	issuer := testCertificate(t, issuerTemplate, issuerTemplate, &issuerKey.PublicKey, issuerKey)
	certKey := testRSAKey(t)
	certTemplate := testCertTemplate("CRL Target", false)
	cert := testCertificate(t, certTemplate, issuer, &certKey.PublicKey, issuerKey)
	return issuer, issuerKey, cert
}

func testCurrentCRL(
	t *testing.T,
	issuer *x509.Certificate,
	issuerKey *rsa.PrivateKey,
	thisUpdate, nextUpdate time.Time,
	entries []x509.RevocationListEntry,
) []byte {
	t.Helper()
	bb, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		SignatureAlgorithm:        issuer.SignatureAlgorithm,
		RevokedCertificateEntries: entries,
		Number:                    big.NewInt(1),
		ThisUpdate:                thisUpdate,
		NextUpdate:                nextUpdate,
	}, issuer, issuerKey)
	if err != nil {
		t.Fatal(err)
	}
	return bb
}

func testCurrentCRLClient(responses map[string][]byte) *http.Client {
	return &http.Client{Transport: signRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		bb, ok := responses[req.URL.String()]
		if !ok {
			return nil, fmt.Errorf("%s unavailable", req.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(bb)),
		}, nil
	})}
}

// TestCurrentCRLNoApplicableEvidenceIsUnknown verifies stale observations do
// not establish a good certificate status.
func TestCurrentCRLNoApplicableEvidenceIsUnknown(t *testing.T) {
	issuer, issuerKey, cert := testCurrentCRLChain(t, "Current CRL Issuer")
	url := "https://crl.test/stale"
	cert.CRLDistributionPoints = []string{url}
	now := time.Now()
	bb := testCurrentCRL(t, issuer, issuerKey, now.Add(-2*time.Hour), now.Add(-time.Hour), nil)

	details, err := processCurrentCRLs(cert, issuer, testCurrentCRLClient(map[string][]byte{url: bb}))
	if err != nil {
		t.Fatal(err)
	}
	if details.Status != model.Unknown ||
		details.Reason != "CRL: no applicable authenticated revocation list" ||
		details.CRL == nil ||
		details.CRL.Applicable != model.False {
		t.Fatalf("stale CRL produced a conclusion: %+v", details)
	}
}

// TestCurrentCRLIssuerMismatchRecordsEntriesWithoutConclusion verifies
// unauthenticated revocation observations remain visible but non-authoritative.
func TestCurrentCRLIssuerMismatchRecordsEntriesWithoutConclusion(t *testing.T) {
	issuer, _, cert := testCurrentCRLChain(t, "Certificate Issuer")
	otherIssuer, otherKey, _ := testCurrentCRLChain(t, "Other CRL Issuer")
	url := "https://crl.test/mismatched"
	cert.CRLDistributionPoints = []string{url}
	now := time.Now()
	entry := x509.RevocationListEntry{SerialNumber: cert.SerialNumber, RevocationTime: now.Add(-time.Minute)}
	bb := testCurrentCRL(t, otherIssuer, otherKey, now.Add(-time.Hour), now.Add(time.Hour), []x509.RevocationListEntry{entry})

	details, err := processCurrentCRLs(cert, issuer, testCurrentCRLClient(map[string][]byte{url: bb}))
	if err != nil {
		t.Fatal(err)
	}
	if details.Status != model.Unknown ||
		details.CRL.IssuerMatched != model.False ||
		details.CRL.SignatureValid != model.False ||
		len(details.CRL.Entries) != 1 ||
		details.CRL.Entries[0].SerialNumber != cert.SerialNumber.Text(16) {
		t.Fatalf("issuer-mismatched CRL evidence: %+v", details)
	}
}

// TestCurrentCRLBadSignatureRecordsEntriesWithoutConclusion verifies matching
// issuer names do not authenticate a CRL signed by a different key.
func TestCurrentCRLBadSignatureRecordsEntriesWithoutConclusion(t *testing.T) {
	issuer, _, cert := testCurrentCRLChain(t, "Shared CRL Issuer")
	impostor, impostorKey, _ := testCurrentCRLChain(t, "Shared CRL Issuer")
	url := "https://crl.test/bad-signature"
	cert.CRLDistributionPoints = []string{url}
	now := time.Now()
	entry := x509.RevocationListEntry{SerialNumber: cert.SerialNumber, RevocationTime: now.Add(-time.Minute)}
	bb := testCurrentCRL(t, impostor, impostorKey, now.Add(-time.Hour), now.Add(time.Hour), []x509.RevocationListEntry{entry})

	details, err := processCurrentCRLs(cert, issuer, testCurrentCRLClient(map[string][]byte{url: bb}))
	if err != nil {
		t.Fatal(err)
	}
	if details.Status != model.Unknown ||
		details.CRL.IssuerMatched != model.True ||
		details.CRL.SignatureValid != model.False ||
		len(details.CRL.Entries) != 1 {
		t.Fatalf("bad-signature CRL evidence: %+v", details)
	}
}

// TestCurrentCRLAuthenticatedRevocationConcludes verifies authenticated,
// applicable online evidence may establish revoked status.
func TestCurrentCRLAuthenticatedRevocationConcludes(t *testing.T) {
	issuer, issuerKey, cert := testCurrentCRLChain(t, "Current CRL Issuer")
	url := "https://crl.test/revoked"
	cert.CRLDistributionPoints = []string{url}
	now := time.Now()
	entry := x509.RevocationListEntry{SerialNumber: cert.SerialNumber, RevocationTime: now.Add(-time.Minute)}
	bb := testCurrentCRL(t, issuer, issuerKey, now.Add(-time.Hour), now.Add(time.Hour), []x509.RevocationListEntry{entry})

	details, err := processCurrentCRLs(cert, issuer, testCurrentCRLClient(map[string][]byte{url: bb}))
	if err != nil {
		t.Fatal(err)
	}
	if details.Status != model.False ||
		details.CRL == nil ||
		!authenticatedApplicableCRL(details.CRL) ||
		len(details.CRL.Entries) != 1 {
		t.Fatalf("authenticated revocation did not conclude: %+v", details)
	}
}

// TestCurrentCRLNilEvidenceIsUnknown verifies an empty observation set cannot
// produce a good status.
func TestCurrentCRLNilEvidenceIsUnknown(t *testing.T) {
	issuer, _, cert := testCurrentCRLChain(t, "Current CRL Issuer")
	details, err := processCurrentCRLs(cert, issuer, testCurrentCRLClient(nil))
	if err != nil {
		t.Fatal(err)
	}
	if details == nil ||
		details.Status != model.Unknown ||
		details.CRL != nil ||
		len(details.CRLs) != 0 {
		t.Fatalf("nil CRL evidence produced a conclusion: %+v", details)
	}
}

// TestAssessCRLNilInputDoesNotPanic verifies an absent parsed CRL remains an
// unknown observation.
func TestAssessCRLNilInputDoesNotPanic(t *testing.T) {
	evidence := assessCRL(nil, nil, model.RevocationEvidenceSourceOnline, true)
	if evidence == nil ||
		evidence.IssuerMatched != model.Unknown ||
		evidence.SignatureValid != model.Unknown ||
		evidence.Applicable != model.Unknown ||
		len(evidence.Entries) != 0 {
		t.Fatalf("nil CRL evidence: %+v", evidence)
	}
}

// TestCurrentCRLRetainsMultipleDistributionPointFailures verifies endpoint
// failures are accumulated and never converted into good status.
func TestCurrentCRLRetainsMultipleDistributionPointFailures(t *testing.T) {
	issuer, _, cert := testCurrentCRLChain(t, "Current CRL Issuer")
	cert.CRLDistributionPoints = []string{
		"https://crl.test/first",
		"https://crl.test/second",
	}

	details, err := processCurrentCRLs(cert, issuer, testCurrentCRLClient(nil))
	if err == nil {
		t.Fatal("expected joined distribution-point failures")
	}
	for _, url := range cert.CRLDistributionPoints {
		if !strings.Contains(err.Error(), url) {
			t.Fatalf("missing distribution-point failure %q in %v", url, err)
		}
	}
	if details == nil || details.Status != model.Unknown || len(details.CRLs) != 2 {
		t.Fatalf("multiple CRL failures produced a conclusion: %+v", details)
	}
	for i, evidence := range details.CRLs {
		if evidence.Error == "" || evidence.Applicable != model.Unknown {
			t.Fatalf("incomplete failed CRL observation: %+v", evidence)
		}
		if evidence.Index != i+1 {
			t.Fatalf("CRL observation index: got %d, want %d", evidence.Index, i+1)
		}
	}
}

func testOCSPResponseBytes(
	t *testing.T,
	issuer, responder *x509.Certificate,
	responderKey *rsa.PrivateKey,
	serialNumber *big.Int,
	embeddedCertificate *x509.Certificate,
	thisUpdate, nextUpdate time.Time,
) []byte {
	t.Helper()
	return testOCSPResponseBytesWithStatus(
		t,
		issuer,
		responder,
		responderKey,
		serialNumber,
		embeddedCertificate,
		thisUpdate,
		nextUpdate,
		ocsp.Good,
	)
}

func testOCSPResponseBytesWithStatus(
	t *testing.T,
	issuer, responder *x509.Certificate,
	responderKey *rsa.PrivateKey,
	serialNumber *big.Int,
	embeddedCertificate *x509.Certificate,
	thisUpdate, nextUpdate time.Time,
	status int,
) []byte {
	t.Helper()
	bb, err := ocsp.CreateResponse(issuer, responder, ocsp.Response{
		Status:       status,
		SerialNumber: serialNumber,
		ThisUpdate:   thisUpdate,
		NextUpdate:   nextUpdate,
		Certificate:  embeddedCertificate,
	}, responderKey)
	if err != nil {
		t.Fatal(err)
	}
	return bb
}

func testOCSPHTTPClient(bb []byte) *http.Client {
	return &http.Client{
		Transport: signRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(bb)),
			}, nil
		}),
	}
}

func testArchivedOCSPFixture(
	t *testing.T,
	responderNotBefore, responderNotAfter, nextUpdate time.Time,
) (*x509.Certificate, *x509.Certificate, *x509.Certificate, []byte) {
	t.Helper()
	issuerKey := testRSAKey(t)
	issuerTemplate := testCertTemplate("OCSP Issuer", true)
	issuer := testCertificate(t, issuerTemplate, issuerTemplate, &issuerKey.PublicKey, issuerKey)

	certKey := testRSAKey(t)
	certTemplate := testCertTemplate("Certificate", false)
	cert := testCertificate(t, certTemplate, issuer, &certKey.PublicKey, issuerKey)

	responderKey := testRSAKey(t)
	responderTemplate := testCertTemplate("Archived OCSP Responder", false)
	responderTemplate.NotBefore = responderNotBefore
	responderTemplate.NotAfter = responderNotAfter
	responderTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning}
	responder := testCertificate(t, responderTemplate, issuer, &responderKey.PublicKey, issuerKey)

	bb, err := ocsp.CreateResponse(issuer, responder, ocsp.Response{
		Status:       ocsp.Good,
		SerialNumber: cert.SerialNumber,
		ThisUpdate:   time.Now().Add(-time.Minute),
		NextUpdate:   nextUpdate,
		Certificate:  responder,
	}, responderKey)
	if err != nil {
		t.Fatal(err)
	}
	return cert, issuer, responder, bb
}

func testIssuerOCSPResponse(
	t *testing.T,
	issuer *x509.Certificate,
	issuerKey *rsa.PrivateKey,
) *ocsp.Response {
	t.Helper()
	now := time.Now().UTC()
	bb, err := ocsp.CreateResponse(issuer, issuer, ocsp.Response{
		Status:       ocsp.Good,
		SerialNumber: big.NewInt(1),
		ThisUpdate:   now.Add(-time.Minute),
		NextUpdate:   now.Add(time.Hour),
	}, issuerKey)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := ocsp.ParseResponse(bb, nil)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func testEmbeddedOCSPResponse(
	t *testing.T,
	issuer, responder *x509.Certificate,
	responderKey *rsa.PrivateKey,
) *ocsp.Response {
	t.Helper()
	now := time.Now().UTC()
	bb, err := ocsp.CreateResponse(issuer, responder, ocsp.Response{
		Status:       ocsp.Good,
		SerialNumber: big.NewInt(1),
		ThisUpdate:   now.Add(-time.Minute),
		NextUpdate:   now.Add(time.Hour),
		Certificate:  responder,
	}, responderKey)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := ocsp.ParseResponse(bb, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Certificate == nil {
		t.Fatal("expected embedded responder certificate")
	}
	return resp
}

// TestCertificateParseErrorIsDiscoverable verifies certificate parsing no longer relies on error text.
func TestCertificateParseErrorIsDiscoverable(t *testing.T) {
	_, err := certFromHexLiteral(types.HexLiteral("00"))
	if !errors.Is(err, errCertificateParse) {
		t.Fatalf("expected certificate parse sentinel, got %v", err)
	}
	if errors.Is(errors.New("x509: malformed certificate"), errCertificateParse) {
		t.Fatal("plain x509-looking text must not classify as a certificate parse error")
	}
}

// TestP1CertificateLiteralDecodePreservesCauseAndContext verifies Cert array decoding wraps its lower cause.
func TestP1CertificateLiteralDecodePreservesCauseAndContext(t *testing.T) {
	_, err := parseP1Certificates(types.Dict{
		"Cert": types.Array{types.HexLiteral("0")},
	})
	if !errors.Is(err, hex.ErrLength) {
		t.Fatalf("expected hex.ErrLength, got %v", err)
	}
	for _, want := range []string{
		"signature dict entry Cert",
		"array index 1",
		"parse certificate",
		"decode hex literal",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q, got %q", want, err)
		}
	}
}

// TestASN1CauseIsPreserved verifies legacy signature ASN.1 failures retain their cause.
func TestASN1CauseIsPreserved(t *testing.T) {
	_, err := verifyRSASHA1Signature(
		bytes.NewReader(nil),
		types.Dict{"Contents": types.HexLiteral("01")},
		&rsa.PublicKey{},
	)
	requireErrorsIsWrappedCause(t, err)
}

// TestPKCS1ContentsRejectsTrailingASN1Data verifies the legacy signature
// decoder requires the Contents ASN.1 value to consume the complete input.
func TestPKCS1ContentsRejectsTrailingASN1Data(t *testing.T) {
	contents, err := asn1.Marshal([]byte{1})
	if err != nil {
		t.Fatal(err)
	}
	contents = append(contents, 0)

	_, err = verifyRSASHA1Signature(
		bytes.NewReader(nil),
		types.Dict{"Contents": types.HexLiteral(hex.EncodeToString(contents))},
		&rsa.PublicKey{},
	)
	if err == nil {
		t.Fatal("expected trailing ASN.1 data failure")
	}
	var syntaxErr asn1.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("expected ASN.1 syntax cause, got %v", err)
	}
	for _, want := range []string{"signature dict entry Contents", "unmarshal ASN.1", "trailing data"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q, got %v", want, err)
		}
	}
}

// TestValidatePKCS1ReportsTrailingASN1Data verifies trailing signature bytes
// remain reportable evidence at the exported validation entry point.
func TestValidatePKCS1ReportsTrailingASN1Data(t *testing.T) {
	key := testRSAKey(t)
	template := testCertTemplate("Legacy RSA Signer", true)
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := asn1.Marshal([]byte{1})
	if err != nil {
		t.Fatal(err)
	}
	contents = append(contents, 0)
	result := &model.SignatureValidationResult{
		Status: model.SignatureStatusUnknown,
		Reason: model.SignatureReasonUnknown,
	}

	err = ValidateX509RSASHA1Signature(
		bytes.NewReader(nil),
		types.Dict{
			"Cert":     types.Array{types.HexLiteral(hex.EncodeToString(der))},
			"Contents": types.HexLiteral(hex.EncodeToString(contents)),
		},
		false,
		false,
		true,
		0,
		x509.NewCertPool(),
		result,
		&model.Context{Configuration: model.NewDefaultConfiguration()},
	)
	if err != nil {
		t.Fatalf("expected reportable signature evidence, got %v", err)
	}
	if result.Status != model.SignatureStatusUnknown ||
		result.Reason != model.SignatureReasonMalformed {
		t.Fatalf("got status=%s reason=%s, want unknown and malformed", result.Status, result.Reason)
	}
	if problems := strings.Join(result.Problems, "\n"); !strings.Contains(problems, "trailing data") {
		t.Fatalf("expected trailing ASN.1 evidence, got %q", problems)
	}
}

// TestDigestCauseIsPreserved verifies signature digest failures retain rsa.ErrVerification.
func TestDigestCauseIsPreserved(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := asn1.Marshal([]byte{1})
	if err != nil {
		t.Fatal(err)
	}
	contentsLiteral := types.HexLiteral(hex.EncodeToString(contents))
	file := append([]byte("data"), []byte(contentsLiteral.String())...)
	sigDict := types.Dict{
		"Contents":  contentsLiteral,
		"ByteRange": byteRange(types.Integer(0), types.Integer(4), types.Integer(len(file)), types.Integer(0)),
	}
	_, err = verifyRSASHA1Signature(bytes.NewReader(file), sigDict, &key.PublicKey)
	if !errors.Is(err, rsa.ErrVerification) {
		t.Fatalf("expected rsa.ErrVerification, got %v", err)
	}
	if want := "SubFilter adbe.x509.rsa_sha1: verify signature"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

// TestTimestampCauseIsPreserved verifies timestamp parsing failures retain their lower-level cause.
func TestTimestampCauseIsPreserved(t *testing.T) {
	_, err := extractTimestampTokenTime([]byte{1})
	requireErrorsIsWrappedCause(t, err)
}

// TestTimestampSigningTimeParsePreservesCause verifies timestamp time parsing retains its semantic phase and typed cause.
func TestTimestampSigningTimeParsePreservesCause(t *testing.T) {
	tests := []struct {
		name string
		tag  int
		want string
	}{
		{"UTCTime", asn1.TagUTCTime, "parse UTC signing time"},
		{"GeneralizedTime", asn1.TagGeneralizedTime, "parse generalized signing time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTimestampSigningTime(asn1.RawValue{Tag: tt.tag, Bytes: []byte("invalid")})
			var parseErr *time.ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("expected time.ParseError, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, err)
			}
		})
	}
}

// TestTimestampSigningTimeRejectsUnexpectedTag verifies unsupported ASN.1 time encodings are contextual.
func TestTimestampSigningTimeRejectsUnexpectedTag(t *testing.T) {
	_, err := parseTimestampSigningTime(asn1.RawValue{Tag: asn1.TagInteger})
	if err == nil || !strings.Contains(err.Error(), "unexpected tag for signing time") {
		t.Fatalf("got %v, want unexpected signing-time tag", err)
	}
}

// TestParseTimestampTokenSigningTimeRejectsMalformedValues verifies complete ASN.1 consumption and typed causes.
func TestParseTimestampTokenSigningTimeRejectsMalformedValues(t *testing.T) {
	valid, err := asn1.Marshal(time.Now().UTC().Truncate(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	integer, err := asn1.Marshal(1)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"Malformed", []byte{1}, "unmarshal"},
		{"Trailing", append(valid, 0), "trailing data"},
		{"WrongTag", integer, "unexpected tag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTimestampTokenSigningTime(tt.data)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
			if tt.name == "Malformed" {
				var syntaxErr asn1.SyntaxError
				if !errors.As(err, &syntaxErr) {
					t.Fatalf("expected asn1.SyntaxError, got %v", err)
				}
			}
		})
	}
}

// TestExtractTimestampTokenTimeRejectsSignerCardinalityAndMissingTime verifies complete token evidence is required.
func TestExtractTimestampTokenTimeRejectsSignerCardinalityAndMissingTime(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"NoSigners", zeroSignerPKCS7(t), "expected one signer, got 0"},
		{"MissingSigningTime", signerPKCS7(t, 1), "signing time unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := extractTimestampTokenTime(tt.data); err == nil ||
				!strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
		})
	}
}

// TestParseClaimedSigningTimeRejectsTrailingData verifies signed signing-time attributes are consumed completely.
func TestParseClaimedSigningTimeRejectsTrailingData(t *testing.T) {
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidSigningTime, Value: time.Now().UTC().Truncate(time.Second)}},
		nil,
	)
	signer := fixture.signer
	found := false
	for i := range signer.AuthenticatedAttributes {
		if signer.AuthenticatedAttributes[i].Type.Equal(oidSigningTime) {
			signer.AuthenticatedAttributes[i].Value.Bytes = append(
				signer.AuthenticatedAttributes[i].Value.Bytes,
				0,
			)
			found = true
			break
		}
	}
	if !found {
		t.Fatal("missing signing-time test attribute")
	}

	_, err := parseClaimedSigningTime(signer)
	if err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Fatalf("expected trailing signing-time data failure, got %v", err)
	}
}

// TestParseClaimedSigningTimeRejectsDuplicates verifies ambiguous signed time evidence is rejected.
func TestParseClaimedSigningTimeRejectsDuplicates(t *testing.T) {
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidSigningTime, Value: time.Now().UTC().Truncate(time.Second)}},
		nil,
	)
	signerInfo := fixture.signer
	for _, attr := range signerInfo.AuthenticatedAttributes {
		if attr.Type.Equal(oidSigningTime) {
			signerInfo.AuthenticatedAttributes = append(signerInfo.AuthenticatedAttributes, attr)
			break
		}
	}

	signer := &model.Signer{}
	result := &model.SignatureValidationResult{
		Status: model.SignatureStatusUnknown,
		Reason: model.SignatureReasonUnknown,
	}
	handleClaimedSigningTime(signerInfo, signer, result)
	if result.Reason != model.SignatureReasonSigningTimeInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonSigningTimeInvalid)
	}
	if problems := strings.Join(signer.Problems, "\n"); !strings.Contains(problems, "signing time attributes") ||
		!strings.Contains(problems, "duplicate") {
		t.Fatalf("got problems %v, want indexed duplicate evidence", signer.Problems)
	}
}

// TestRevocationInfoArchivalPreservesASN1Cause verifies malformed attribute decoding retains the typed cause.
func TestRevocationInfoArchivalPreservesASN1Cause(t *testing.T) {
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidRevocationInfoArchival, Value: RevocationInfoArchival{}}},
		nil,
	)
	signerInfo := fixture.signer
	for i := range signerInfo.AuthenticatedAttributes {
		if signerInfo.AuthenticatedAttributes[i].Type.Equal(oidRevocationInfoArchival) {
			signerInfo.AuthenticatedAttributes[i].Value.Bytes = []byte{1}
			break
		}
	}

	_, err := revocationInfoArchival(signerInfo)
	var syntaxErr asn1.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("expected asn1.SyntaxError, got %v", err)
	}
	if !strings.Contains(err.Error(), "pkcs7: revocation info archival attribute 1: unmarshal") {
		t.Fatalf("missing revocation-info context: %v", err)
	}
}

// TestHandleClaimedSigningTimeReportsParseEvidence verifies malformed signed attributes remain nonfatal evidence.
func TestHandleClaimedSigningTimeReportsParseEvidence(t *testing.T) {
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidSigningTime, Value: time.Now().UTC().Truncate(time.Second)}},
		nil,
	)
	signerInfo := fixture.signer
	for i := range signerInfo.AuthenticatedAttributes {
		if signerInfo.AuthenticatedAttributes[i].Type.Equal(oidSigningTime) {
			signerInfo.AuthenticatedAttributes[i].Value.Bytes = []byte{1}
			break
		}
	}
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{
		Status: model.SignatureStatusUnknown,
		Reason: model.SignatureReasonUnknown,
	}

	handleClaimedSigningTime(signerInfo, signer, result)
	if result.Reason != model.SignatureReasonSigningTimeInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonSigningTimeInvalid)
	}
	if len(signer.Problems) != 1 || !strings.Contains(signer.Problems[0], "pkcs7: signing time attribute") {
		t.Fatalf("got problems %v, want signing-time evidence", signer.Problems)
	}
}

// TestHandleClaimedSigningTimePreservesEarlierSignerState verifies later signer attributes cannot replace prior evidence.
func TestHandleClaimedSigningTimePreservesEarlierSignerState(t *testing.T) {
	firstTime := time.Now().UTC().Truncate(time.Second)
	secondTime := firstTime.Add(time.Minute)
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidSigningTime, Value: secondTime}},
		nil,
	)
	result := &model.SignatureValidationResult{
		Status: model.SignatureStatusUnknown,
		Reason: model.SignatureReasonCertInvalid,
	}
	result.Details.SigningTime = firstTime
	signer := &model.Signer{}

	handleClaimedSigningTime(fixture.signer, signer, result)
	if !result.Details.SigningTime.Equal(firstTime) {
		t.Fatalf("later signer replaced first signing time: got %v, want %v", result.Details.SigningTime, firstTime)
	}

	malformed := fixture.signer
	for i := range malformed.AuthenticatedAttributes {
		if malformed.AuthenticatedAttributes[i].Type.Equal(oidSigningTime) {
			malformed.AuthenticatedAttributes[i].Value.Bytes = []byte{1}
			break
		}
	}
	handleClaimedSigningTime(malformed, signer, result)
	if result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("later signing-time evidence replaced reason: %s", result.Reason)
	}
	if problems := strings.Join(signer.Problems, "\n"); !strings.Contains(problems, "pkcs7: signing time attribute") {
		t.Fatalf("missing later signer problem: %q", problems)
	}
}

// TestCheckTimestampTokenDoesNotAuthenticateContextTime verifies ctx.DTS alone
// cannot manufacture timestamp evidence for an ordinary signature.
func TestCheckTimestampTokenDoesNotAuthenticateContextTime(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	documentTimestamp := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	fixture.ctx.DTS = documentTimestamp
	signer := &model.Signer{PAdES: "B-B"}
	result := &model.SignatureValidationResult{
		Status: model.SignatureStatusUnknown,
		Reason: model.SignatureReasonUnknown,
	}

	checkTimestampToken(fixture.signer, fixture.ctx, signer, result)
	if signer.HasTimestamp || !signer.Timestamp.IsZero() {
		t.Fatalf("standalone ctx.DTS manufactured timestamp evidence: %+v", signer)
	}
	if signer.PAdES != "B-B" {
		t.Fatalf("got PAdES level %q, want B-B", signer.PAdES)
	}
}

// TestLocalTimestampPreparationReproducesCurrentBehavior states local
// preparation preserves the existing absent, embedded, and document evidence.
func TestLocalTimestampPreparationReproducesCurrentBehavior(t *testing.T) {
	tokenTime := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	token := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidSigningTime, Value: tokenTime}},
		nil,
	)
	embedded := newDetachedP7SignerFixture(
		t,
		nil,
		[]pkcs7.Attribute{{
			Type:  oidTimestampToken,
			Value: asn1.RawValue{FullBytes: token.raw},
		}},
	)
	absent := newDetachedP7SignerFixture(t, nil, nil)

	got := embeddedSignatureTimestampEvidence(absent.signer)
	requireTimestampEvidence(t, got, timestampKindSignature, time.Time{}, false)
	got = embeddedSignatureTimestampEvidence(embedded.signer)
	if !got.Present || got.Err == nil ||
		!strings.Contains(got.Err.Error(), "timestamp token: missing timestamp info") ||
		!got.SigningTime.IsZero() {
		t.Fatalf("non-TSTInfo token was accepted as timestamp evidence: %+v", got)
	}

	got = documentTimestampEvidence(tokenTime)
	requireTimestampEvidence(t, got, timestampKindDocument, tokenTime, true)
}

// TestPopulateEmbeddedTimestampInfoUsesTSTInfoGenTime verifies observed
// embedded timestamp time comes from TSTInfo rather than CMS signer metadata.
func TestPopulateEmbeddedTimestampInfoUsesTSTInfoGenTime(t *testing.T) {
	genTime := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	var tstInfo TSTInfo
	tstInfo.Version = 1
	tstInfo.Policy = asn1.ObjectIdentifier{1, 2, 3}
	tstInfo.MessageImprint.HashAlgorithm.Algorithm = pkcs7.OIDDigestAlgorithmSHA256
	tstInfo.MessageImprint.HashedMessage = []byte{1, 2, 3}
	tstInfo.SerialNumber = asn1.RawValue{Tag: asn1.TagInteger, Bytes: []byte{1}}
	tstInfo.GenTime = genTime
	content, err := asn1.Marshal(tstInfo)
	if err != nil {
		t.Fatal(err)
	}
	evidence := timestampEvidence{
		Kind:    timestampKindSignature,
		Present: true,
		CMS: &pkcs7.PKCS7{
			ContentType: oidTSTInfo,
			Content:     content,
		},
	}

	if err := populateEmbeddedTimestampInfo(&evidence); err != nil {
		t.Fatal(err)
	}
	if !evidence.SigningTime.Equal(genTime) ||
		evidence.TokenInfo == nil ||
		!evidence.TokenInfo.GeneratedAt.Equal(genTime) {
		t.Fatalf("TSTInfo genTime was not retained: %+v", evidence)
	}
}

// TestEmbeddedTimestampEvidenceReportsMalformedPresence verifies malformed
// embedded tokens remain present evidence with their parsing failure attached.
func TestEmbeddedTimestampEvidenceReportsMalformedPresence(t *testing.T) {
	fixture := newDetachedP7SignerFixture(
		t,
		nil,
		[]pkcs7.Attribute{{Type: oidTimestampToken, Value: struct{}{}}},
	)
	signerInfo := fixture.signer
	for i := range signerInfo.UnauthenticatedAttributes {
		if signerInfo.UnauthenticatedAttributes[i].Type.Equal(oidTimestampToken) {
			signerInfo.UnauthenticatedAttributes[i].Value.Bytes = []byte{1}
			break
		}
	}

	evidence := embeddedSignatureTimestampEvidence(signerInfo)
	if !evidence.Present {
		t.Fatal("malformed embedded timestamp was reported as absent")
	}
	if evidence.Err == nil {
		t.Fatal("malformed embedded timestamp has no error evidence")
	}
	if !evidence.SigningTime.IsZero() {
		t.Fatalf("malformed embedded timestamp supplied time %v", evidence.SigningTime)
	}
	if evidence.Kind != timestampKindSignature {
		t.Fatalf("got timestamp kind %d, want signature timestamp", evidence.Kind)
	}
	if evidence.AssessmentScope != model.AssessmentScopeLocal {
		t.Fatalf("got assessment scope %d, want local", evidence.AssessmentScope)
	}
}

// TestDocumentTimestampEvidencePopulatesCommonStructure verifies document
// timestamps use the same structured evidence representation.
func TestDocumentTimestampEvidencePopulatesCommonStructure(t *testing.T) {
	signingTime := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	evidence := documentTimestampEvidence(signingTime)

	if !evidence.Present || evidence.Kind != timestampKindDocument ||
		evidence.Err != nil || !evidence.SigningTime.Equal(signingTime) ||
		evidence.AssessmentScope != model.AssessmentScopeLocal {
		t.Fatalf("unexpected document timestamp evidence: %+v", evidence)
	}
}

// TestClaimedSigningTimeIsDisplayOnly verifies a CMS signing-time claim
// remains available for compatible presentation without becoming validation
// time evidence.
func TestClaimedSigningTimeIsDisplayOnly(t *testing.T) {
	signingTime := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidSigningTime, Value: signingTime}},
		nil,
	)
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	handleClaimedSigningTime(fixture.signer, &model.Signer{}, result)

	if !result.Details.SigningTime.Equal(signingTime) {
		t.Fatalf("got displayed signing time %v, want %v", result.Details.SigningTime, signingTime)
	}
	if result.Details.Signers != nil {
		t.Fatalf("claimed time unexpectedly produced signer assessment: %+v", result.Details.Signers)
	}
}

// TestPreparedDocumentTimestampEvidenceAvoidsReparsing verifies preparation
// retains the decoded CMS token and already-read positional PDF data.
func TestPreparedDocumentTimestampEvidenceAvoidsReparsing(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	cms, err := pkcs7.Parse(fixture.raw)
	if err != nil {
		t.Fatal(err)
	}
	signingTime := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	tstInfo := &TSTInfo{Version: 1, GenTime: signingTime}
	tstInfo.Policy = asn1.ObjectIdentifier{1, 2, 3}
	tstInfo.MessageImprint.HashAlgorithm.Algorithm = pkcs7.OIDDigestAlgorithmSHA256
	tstInfo.MessageImprint.HashedMessage = []byte{1, 2, 3}
	evidence := preparedDocumentTimestampEvidence(
		tstInfo,
		fixture.raw,
		cms,
		fixture.signer,
		fixture.content,
	)

	if !bytes.Equal(evidence.RawToken, fixture.raw) ||
		evidence.CMS != cms ||
		!bytes.Equal(evidence.SourceSigner.EncryptedDigest, fixture.signer.EncryptedDigest) ||
		evidence.TokenInfo == nil ||
		!evidence.TokenInfo.GeneratedAt.Equal(signingTime) ||
		!evidence.TokenInfo.Policy.Equal(tstInfo.Policy) ||
		!evidence.TokenInfo.MessageImprintAlgorithm.Equal(pkcs7.OIDDigestAlgorithmSHA256) ||
		!bytes.Equal(evidence.TokenInfo.MessageImprint, tstInfo.MessageImprint.HashedMessage) ||
		!bytes.Equal(evidence.SignedData, fixture.content) ||
		!evidence.DigestVerified ||
		!evidence.SignatureVerified ||
		evidence.CorrectProfile ||
		evidence.LocalTSAPathValidated {
		t.Fatalf("incomplete prepared document timestamp evidence: %+v", evidence)
	}
}

// TestTimestampPreparationIsPureAndEvidenceAppliedCentrally verifies preparation
// does not mutate validation state and only locally validated evidence produces
// document-timestamp conclusions.
func TestTimestampPreparationIsPureAndEvidenceAppliedCentrally(t *testing.T) {
	signingTime := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	signer := &model.Signer{PAdES: "B-B"}
	result := &model.SignatureValidationResult{}

	evidence := documentTimestampEvidence(signingTime)

	if evidence.Err != nil || !evidence.SigningTime.Equal(signingTime) {
		t.Fatalf("unexpected document timestamp evidence: %+v", evidence)
	}
	if signer.HasTimestamp || !signer.Timestamp.IsZero() || !result.Details.SigningTime.IsZero() {
		t.Fatalf("timestamp preparation mutated signer or result: signer=%+v result=%+v", signer, result)
	}

	applyTimestampEvidence(evidence, timestampApplication{
		signer:               signer,
		result:               result,
		setResultSigningTime: true,
		problemPrefix:        "SubFilter ETSI.RFC3161: evaluate timestamp",
	})
	if !signer.HasTimestamp || !signer.Timestamp.Equal(signingTime) || signer.PAdES != "B-B" {
		t.Fatalf("unexpected signer state: %+v", signer)
	}
	if !result.Details.SigningTime.IsZero() {
		t.Fatalf("timestamp without local validation selected signing time %v", result.Details.SigningTime)
	}

	evidence.DigestVerified = true
	evidence.SignatureVerified = true
	evidence.CorrectProfile = true
	evidence.LocalTSAPathValidated = true
	applyTimestampEvidence(evidence, timestampApplication{
		signer:               signer,
		result:               result,
		setResultSigningTime: true,
		problemPrefix:        "SubFilter ETSI.RFC3161: evaluate timestamp",
	})
	if !result.Details.SigningTime.Equal(signingTime) {
		t.Fatalf("locally validated timestamp did not select signing time: %v", result.Details.SigningTime)
	}
}

// TestEmbeddedTimestampEvidenceIsContained verifies observed embedded
// timestamp data cannot promote PAdES while archived evidence remains
// collectable.
func TestEmbeddedTimestampEvidenceIsContained(t *testing.T) {
	tokenTime := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{
			Type: oidRevocationInfoArchival,
			Value: RevocationInfoArchival{
				CRLs: []asn1.RawValue{{FullBytes: []byte{0x30, 0x00}}},
			},
		}},
		nil,
	)
	signer := &model.Signer{PAdES: "B-B"}
	result := &model.SignatureValidationResult{
		Status: model.SignatureStatusUnknown,
		Reason: model.SignatureReasonUnknown,
	}
	evidence := timestampEvidence{
		Kind:        timestampKindSignature,
		Present:     true,
		SigningTime: tokenTime,
		TokenInfo:   &timestampTokenInfo{GeneratedAt: tokenTime},
	}
	applyTimestampEvidence(evidence, timestampApplication{
		signer:        signer,
		result:        result,
		problemPrefix: "pkcs7",
	})
	crls, ocsps := handleArchivedRevocationInfo(fixture.signer, signer)

	if !signer.HasTimestamp ||
		!signer.Timestamp.Equal(tokenTime) ||
		signer.PAdES != "B-B" ||
		signer.LTVEnabled ||
		len(crls) != 1 ||
		len(ocsps) != 0 ||
		result.Reason != model.SignatureReasonUnknown {
		t.Fatalf("embedded timestamp escaped containment: signer=%+v CRLs=%d OCSPs=%d", signer, len(crls), len(ocsps))
	}
	if len(signer.Problems) != 1 ||
		!strings.Contains(signer.Problems[0], embeddedTimestampNotAuthenticated) {
		t.Fatalf("missing unauthenticated-token problem: %v", signer.Problems)
	}
}

// TestLocalTimestampEvidenceBoundary documents reported observed
// timestamp evidence together with a configuration-dependent local assessment.
func TestLocalTimestampEvidenceBoundary(t *testing.T) {
	cause := errors.New("malformed timestamp token")
	prepared := timestampEvidence{
		Kind:    timestampKindSignature,
		Present: true,
		Err:     fmt.Errorf("timestamp token: parse: %w", cause),
	}
	signer := &model.Signer{}
	result := unknownSignatureResult()

	applyTimestampEvidence(prepared, timestampApplication{
		signer:        signer,
		result:        result,
		problemPrefix: "pkcs7",
	})

	if !signer.HasTimestamp ||
		!signer.Timestamp.IsZero() ||
		result.Reason != model.SignatureReasonTimestampTokenInvalid ||
		len(signer.Problems) != 1 ||
		!strings.Contains(signer.Problems[0], cause.Error()) {
		t.Fatalf("malformed timestamp application changed: signer=%+v result=%+v", signer, result)
	}
}

// TestDocumentTimestampTimeValueIsObservedOnly verifies a time value cannot
// reconstruct the authentication checks that produced it.
func TestDocumentTimestampTimeValueIsObservedOnly(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	documentTimestamp := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	fixture.ctx.DTS = documentTimestamp
	evidence := documentTimestampEvidence(documentTimestamp)
	if evidence.Kind != timestampKindDocument ||
		evidence.DigestVerified ||
		evidence.SignatureVerified ||
		evidence.CorrectProfile ||
		evidence.LocalTSAPathValidated ||
		isCryptographicallyAuthenticatedDocumentTimestampEvidence(evidence) ||
		isLocallyValidatedDocumentTimestampEvidence(evidence) {
		t.Fatalf("standalone time manufactured authentication evidence: %+v", evidence)
	}

	checkTimestampToken(
		fixture.signer,
		fixture.ctx,
		&model.Signer{},
		&model.SignatureValidationResult{},
	)
}

// TestUnauthenticatedEmbeddedTimestampIsNotReplacedByContextTime verifies
// malformed embedded evidence remains reported without a ctx.DTS fallback.
func TestUnauthenticatedEmbeddedTimestampIsNotReplacedByContextTime(t *testing.T) {
	tokenTime := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	token := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidSigningTime, Value: tokenTime}},
		nil,
	)
	fixture := newDetachedP7SignerFixture(
		t,
		nil,
		[]pkcs7.Attribute{{
			Type:  oidTimestampToken,
			Value: asn1.RawValue{FullBytes: token.raw},
		}},
	)
	fixture.ctx.DTS = tokenTime.Add(time.Hour)
	signer := &model.Signer{PAdES: "B-B"}
	result := &model.SignatureValidationResult{
		Status: model.SignatureStatusUnknown,
		Reason: model.SignatureReasonUnknown,
	}

	checkTimestampToken(fixture.signer, fixture.ctx, signer, result)
	if !signer.HasTimestamp || !signer.Timestamp.IsZero() {
		t.Fatalf("ctx.DTS replaced unauthenticated embedded evidence: %+v", signer)
	}
	if signer.PAdES != "B-B" {
		t.Fatalf("context time changed PAdES %q", signer.PAdES)
	}
	if result.Reason != model.SignatureReasonTimestampTokenInvalid {
		t.Fatalf("embedded evidence classification changed: %+v", result)
	}
	if problems := strings.Join(signer.Problems, "\n"); !strings.Contains(problems, "missing timestamp info") {
		t.Fatalf("unauthenticated embedded timestamp was not reported: %q", problems)
	}
}

// TestDocumentTimestampDoesNotConcludeArchivedRevocation verifies a locally
// validated DTS does not turn archived evidence collection into a conclusion.
func TestDocumentTimestampDoesNotConcludeArchivedRevocation(t *testing.T) {
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{
			Type: oidRevocationInfoArchival,
			Value: RevocationInfoArchival{
				CRLs:  []asn1.RawValue{{FullBytes: []byte{0x30, 0x00}}},
				OCSPs: []asn1.RawValue{{FullBytes: []byte{0x30, 0x00}}},
			},
		}},
		nil,
	)
	fixture.ctx.DTS = time.Now()
	signer := &model.Signer{PAdES: "B-B"}

	checkTimestampToken(
		fixture.signer,
		fixture.ctx,
		signer,
		&model.SignatureValidationResult{},
	)
	crls, ocsps := handleArchivedRevocationInfo(fixture.signer, signer)

	if signer.HasTimestamp ||
		!signer.Timestamp.IsZero() ||
		signer.PAdES != "B-B" ||
		signer.LTVEnabled ||
		len(crls) != 1 ||
		len(ocsps) != 1 {
		t.Fatalf(
			"document timestamp changed archived evidence conclusion: signer=%+v CRLs=%d OCSPs=%d",
			signer,
			len(crls),
			len(ocsps),
		)
	}
}

// TestHandleTimestampTokenReportsMalformedEvidence verifies malformed unsigned token attributes remain nonfatal evidence.
func TestHandleTimestampTokenReportsMalformedEvidence(t *testing.T) {
	fixture := newDetachedP7SignerFixture(
		t,
		nil,
		[]pkcs7.Attribute{{
			Type:  oidTimestampToken,
			Value: struct{}{},
		}},
	)
	signerInfo := fixture.signer
	for i := range signerInfo.UnauthenticatedAttributes {
		if signerInfo.UnauthenticatedAttributes[i].Type.Equal(oidTimestampToken) {
			signerInfo.UnauthenticatedAttributes[i].Value.Bytes = []byte{1}
			break
		}
	}
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{
		Status: model.SignatureStatusUnknown,
		Reason: model.SignatureReasonUnknown,
	}

	handleTimestampToken(signerInfo, signer, result)
	if result.Reason != model.SignatureReasonTimestampTokenInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonTimestampTokenInvalid)
	}
	if !signer.HasTimestamp {
		t.Fatal("expected detected timestamp evidence")
	}
	if !signer.Timestamp.IsZero() {
		t.Fatalf("got invalid timestamp value %v, want zero", signer.Timestamp)
	}
	if len(signer.Problems) != 1 || !strings.Contains(signer.Problems[0], "pkcs7: timestamp token") {
		t.Fatalf("got problems %v, want timestamp-token evidence", signer.Problems)
	}
}

// TestHandleTimestampTokenRejectsDuplicates verifies ambiguous unsigned timestamp evidence is rejected.
func TestHandleTimestampTokenRejectsDuplicates(t *testing.T) {
	fixture := newDetachedP7SignerFixture(
		t,
		nil,
		[]pkcs7.Attribute{{Type: oidTimestampToken, Value: struct{}{}}},
	)
	signerInfo := fixture.signer
	for _, attr := range signerInfo.UnauthenticatedAttributes {
		if attr.Type.Equal(oidTimestampToken) {
			signerInfo.UnauthenticatedAttributes = append(signerInfo.UnauthenticatedAttributes, attr)
			break
		}
	}
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{
		Status: model.SignatureStatusUnknown,
		Reason: model.SignatureReasonUnknown,
	}

	handleTimestampToken(signerInfo, signer, result)
	if result.Reason != model.SignatureReasonTimestampTokenInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonTimestampTokenInvalid)
	}
	if problems := strings.Join(signer.Problems, "\n"); !strings.Contains(problems, "timestamp token attributes") ||
		!strings.Contains(problems, "duplicate") {
		t.Fatalf("got problems %v, want indexed duplicate evidence", signer.Problems)
	}
}

// TestTimestampTokenSigningTimeRejectsDuplicates verifies ambiguous token signing times are rejected.
func TestTimestampTokenSigningTimeRejectsDuplicates(t *testing.T) {
	tokenTime := time.Now().UTC().Truncate(time.Second)
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidSigningTime, Value: tokenTime}},
		nil,
	)
	signerInfo := fixture.signer
	for _, attr := range signerInfo.AuthenticatedAttributes {
		if attr.Type.Equal(oidSigningTime) {
			signerInfo.AuthenticatedAttributes = append(signerInfo.AuthenticatedAttributes, attr)
			break
		}
	}

	_, err := timestampTokenSigningTime(signerInfo)
	if err == nil || !strings.Contains(err.Error(), "timestamp token: signing time attributes") ||
		!strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("got %v, want indexed duplicate signing-time failure", err)
	}
}

// TestInvalidTimestampDoesNotConcludeArchivedRevocation verifies detected but
// invalid time evidence cannot turn collected evidence into an LTV conclusion.
func TestInvalidTimestampDoesNotConcludeArchivedRevocation(t *testing.T) {
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{
			Type: oidRevocationInfoArchival,
			Value: RevocationInfoArchival{
				CRLs: []asn1.RawValue{{FullBytes: []byte{0x30, 0x00}}},
			},
		}},
		[]pkcs7.Attribute{{Type: oidTimestampToken, Value: struct{}{}}},
	)
	signerInfo := fixture.signer
	for i := range signerInfo.UnauthenticatedAttributes {
		if signerInfo.UnauthenticatedAttributes[i].Type.Equal(oidTimestampToken) {
			signerInfo.UnauthenticatedAttributes[i].Value.Bytes = []byte{1}
			break
		}
	}
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{
		Status: model.SignatureStatusUnknown,
		Reason: model.SignatureReasonUnknown,
	}

	handleTimestampToken(signerInfo, signer, result)
	crls, ocsps := handleArchivedRevocationInfo(signerInfo, signer)

	if len(crls) != 1 || len(ocsps) != 0 {
		t.Fatalf("invalid timestamp suppressed archived evidence: CRLs=%d OCSPs=%d", len(crls), len(ocsps))
	}
	if signer.LTVEnabled {
		t.Fatal("invalid timestamp unexpectedly enabled LTV")
	}
}

// TestHandleArchivedRevocationInfoRecordsEvidenceWithoutLTVConclusion verifies archived evidence does not imply LTV.
func TestHandleArchivedRevocationInfoRecordsEvidenceWithoutLTVConclusion(t *testing.T) {
	crl := []byte{0x30, 0x00}
	ocspResponse := []byte{0x30, 0x00}
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{
			Type: oidRevocationInfoArchival,
			Value: RevocationInfoArchival{
				CRLs:  []asn1.RawValue{{FullBytes: crl}},
				OCSPs: []asn1.RawValue{{FullBytes: ocspResponse}},
			},
		}},
		nil,
	)
	signer := &model.Signer{PAdES: "B-B"}

	crls, ocsps := handleArchivedRevocationInfo(fixture.signer, signer)

	if len(crls) != 1 || !bytes.Equal(crls[0], crl) {
		t.Fatalf("got CRLs %x, want %x", crls, crl)
	}
	if len(ocsps) != 1 || !bytes.Equal(ocsps[0], ocspResponse) {
		t.Fatalf("got OCSP responses %x, want %x", ocsps, ocspResponse)
	}
	if signer.LTVEnabled {
		t.Fatal("archived revocation evidence unexpectedly enabled LTV")
	}
	if signer.PAdES != "B-B" {
		t.Fatalf("got PAdES %q, want unchanged B-B", signer.PAdES)
	}
	if len(signer.Problems) != 0 {
		t.Fatalf("unexpected problems %v", signer.Problems)
	}
}

// TestRevocationInfoArchivalRejectsTrailingData verifies malformed attributes cannot supply partial revocation evidence.
func TestRevocationInfoArchivalRejectsTrailingData(t *testing.T) {
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidRevocationInfoArchival, Value: RevocationInfoArchival{}}},
		nil,
	)
	signerInfo := fixture.signer
	found := false
	for i := range signerInfo.AuthenticatedAttributes {
		if signerInfo.AuthenticatedAttributes[i].Type.Equal(oidRevocationInfoArchival) {
			signerInfo.AuthenticatedAttributes[i].Value.Bytes = append(
				signerInfo.AuthenticatedAttributes[i].Value.Bytes,
				0,
			)
			found = true
			break
		}
	}
	if !found {
		t.Fatal("missing revocation-info test attribute")
	}

	ria, err := revocationInfoArchival(signerInfo)
	if err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Fatalf("expected trailing revocation-info data failure, got %v", err)
	}
	if ria != nil {
		t.Fatalf("got partial revocation info %+v, want nil", ria)
	}

	signer := &model.Signer{HasTimestamp: true, Timestamp: time.Now(), PAdES: "B-T"}
	crls, ocsps := handleArchivedRevocationInfo(signerInfo, signer)
	if len(crls) != 0 || len(ocsps) != 0 {
		t.Fatalf("got partial revocation evidence: CRLs=%d OCSPs=%d", len(crls), len(ocsps))
	}
	if signer.LTVEnabled {
		t.Fatal("malformed revocation evidence unexpectedly enabled LTV")
	}
	if len(signer.Problems) != 1 || !strings.Contains(signer.Problems[0], "pkcs7: revocation info archival") {
		t.Fatalf("got problems %v, want revocation-info evidence", signer.Problems)
	}
}

// TestRevocationInfoArchivalRejectsDuplicates verifies ambiguous archived revocation evidence is rejected.
func TestRevocationInfoArchivalRejectsDuplicates(t *testing.T) {
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidRevocationInfoArchival, Value: RevocationInfoArchival{}}},
		nil,
	)
	signerInfo := fixture.signer
	for _, attr := range signerInfo.AuthenticatedAttributes {
		if attr.Type.Equal(oidRevocationInfoArchival) {
			signerInfo.AuthenticatedAttributes = append(signerInfo.AuthenticatedAttributes, attr)
			break
		}
	}

	signer := &model.Signer{HasTimestamp: true, Timestamp: time.Now(), PAdES: "B-T"}
	crls, ocsps := handleArchivedRevocationInfo(signerInfo, signer)
	if len(crls) != 0 || len(ocsps) != 0 || signer.LTVEnabled {
		t.Fatalf("duplicate attributes supplied revocation evidence: signer=%+v", signer)
	}
	if problems := strings.Join(signer.Problems, "\n"); !strings.Contains(problems, "revocation info archival attributes") ||
		!strings.Contains(problems, "duplicate") {
		t.Fatalf("got problems %v, want indexed duplicate evidence", signer.Problems)
	}
}

// TestRevocationCauseIsPreserved verifies archived CRL parsing failures retain their lower-level cause.
func TestRevocationCauseIsPreserved(t *testing.T) {
	_, err := processArchivedCRLs(
		&x509.Certificate{SerialNumber: big.NewInt(1)},
		nil,
		[][]byte{{1}},
	)
	requireErrorsIsWrappedCause(t, err)
}

// TestArchivedCRLReportsAllMalformedCandidates verifies indexed parse failures
// and observations survive when no archived CRL is usable.
func TestArchivedCRLReportsAllMalformedCandidates(t *testing.T) {
	details, err := processArchivedCRLs(
		&x509.Certificate{SerialNumber: big.NewInt(1)},
		nil,
		[][]byte{{1}, {2}},
	)
	if err == nil {
		t.Fatal("expected archived CRL failures")
	}
	for _, want := range []string{"archived CRL 1", "archived CRL 2"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q, got %v", want, err)
		}
	}
	if details == nil || details.Status != model.Unknown || len(details.CRLs) != 2 {
		t.Fatalf("archived CRL failure observations: %+v", details)
	}
	for i, evidence := range details.CRLs {
		if evidence.Source != model.RevocationEvidenceSourceArchived ||
			evidence.Index != i+1 ||
			evidence.Location != fmt.Sprintf("archived CRL %d", i+1) ||
			evidence.Error == "" {
			t.Fatalf("archived CRL observation %d: %+v", i+1, evidence)
		}
	}
}

// TestArchivedCRLRecordsIssuerAndSignatureEvidence verifies archived CRL
// authentication observations do not establish historical applicability.
func TestArchivedCRLRecordsIssuerAndSignatureEvidence(t *testing.T) {
	signingTime := time.Now().UTC().Truncate(time.Second)
	issuerKey := testRSAKey(t)
	issuerTemplate := testCertTemplate("CRL Issuer", true)
	issuerTemplate.KeyUsage |= x509.KeyUsageCRLSign
	issuerTemplate.SubjectKeyId = []byte{1, 2, 3, 4}
	issuer := testCertificate(t, issuerTemplate, issuerTemplate, &issuerKey.PublicKey, issuerKey)
	certKey := testRSAKey(t)
	certTemplate := testCertTemplate("Certificate", false)
	cert := testCertificate(t, certTemplate, issuer, &certKey.PublicKey, issuerKey)
	crlDER, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		SignatureAlgorithm: issuer.SignatureAlgorithm,
		RevokedCertificateEntries: []x509.RevocationListEntry{{
			SerialNumber:   cert.SerialNumber,
			RevocationTime: signingTime.Add(-time.Minute),
		}},
		Number:     big.NewInt(1),
		ThisUpdate: signingTime.Add(-time.Hour),
		NextUpdate: signingTime.Add(time.Hour),
	}, issuer, issuerKey)
	if err != nil {
		t.Fatal(err)
	}

	details, err := processArchivedCRLs(cert, issuer, [][]byte{crlDER})
	if err != nil {
		t.Fatalf("archived CRL: %v", err)
	}
	requireCRLEvidence(t, details, model.Unknown, model.True, model.True, model.Unknown)

	otherKey := testRSAKey(t)
	otherTemplate := testCertTemplate("Other CRL Issuer", true)
	otherTemplate.KeyUsage |= x509.KeyUsageCRLSign
	otherTemplate.SubjectKeyId = []byte{5, 6, 7, 8}
	otherIssuer := testCertificate(t, otherTemplate, otherTemplate, &otherKey.PublicKey, otherKey)
	details, err = processArchivedCRLs(cert, otherIssuer, [][]byte{crlDER})
	if err != nil {
		t.Fatalf("archived CRL with unrelated issuer: %v", err)
	}
	requireCRLEvidence(t, details, model.Unknown, model.False, model.False, model.Unknown)
	if details.CRL == nil ||
		len(details.CRL.Entries) != 1 ||
		details.CRL.Entries[0].SerialNumber != cert.SerialNumber.Text(16) {
		t.Fatalf("unauthenticated archived CRL lost observed revocation entries: %+v", details)
	}
	if details.Status != model.Unknown {
		t.Fatalf("unauthenticated archived CRL produced status %d", details.Status)
	}
}

// TestArchivedCRLContinuesAfterMalformedCandidate verifies malformed archived
// CRLs remain indexed observations and do not mask later authenticated evidence.
func TestArchivedCRLContinuesAfterMalformedCandidate(t *testing.T) {
	signingTime := time.Now().UTC().Truncate(time.Second)
	issuer, issuerKey, cert := testCurrentCRLChain(t, "Archived CRL Issuer")
	valid := testCurrentCRL(
		t,
		issuer,
		issuerKey,
		signingTime.Add(-time.Hour),
		signingTime.Add(time.Hour),
		nil,
	)

	details, err := processArchivedCRLs(cert, issuer, [][]byte{{1}, valid})
	if err == nil || !strings.Contains(err.Error(), "archived CRL 1") {
		t.Fatalf("missing malformed archived CRL evidence: %v", err)
	}
	if details.Status != model.Unknown || len(details.CRLs) != 2 {
		t.Fatalf("archived CRL observations: %+v", details)
	}
	if details.CRLs[0].Location != "archived CRL 1" ||
		details.CRLs[0].Index != 1 ||
		details.CRLs[0].Error == "" ||
		details.CRLs[1].Location != "archived CRL 2" ||
		details.CRLs[1].Index != 2 ||
		details.CRLs[1].IssuerMatched != model.True ||
		details.CRLs[1].SignatureValid != model.True ||
		details.CRLs[1].Applicable != model.Unknown {
		t.Fatalf("indexed archived CRL observations: %+v", details.CRLs)
	}
}

// TestUnsupportedCriticalCRLProfileIsUnknown verifies unsupported CRL profile
// extensions cannot establish a good certificate status.
func TestUnsupportedCriticalCRLProfileIsUnknown(t *testing.T) {
	signingTime := time.Now().UTC().Truncate(time.Second)
	issuer, issuerKey, cert := testCurrentCRLChain(t, "Unsupported CRL Issuer")
	bb, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		SignatureAlgorithm: issuer.SignatureAlgorithm,
		Number:             big.NewInt(1),
		ThisUpdate:         signingTime.Add(-time.Hour),
		NextUpdate:         signingTime.Add(time.Hour),
		ExtraExtensions: []pkix.Extension{{
			Id:       oidIssuingDistributionPoint,
			Critical: true,
			Value:    []byte{0x30, 0},
		}},
	}, issuer, issuerKey)
	if err != nil {
		t.Fatal(err)
	}

	details, err := processArchivedCRLs(cert, issuer, [][]byte{bb})
	if err != nil {
		t.Fatal(err)
	}
	if details.Status != model.Unknown ||
		details.CRL == nil ||
		details.CRL.Applicable != model.Unknown ||
		!strings.Contains(details.CRL.Error, "unsupported CRL extensions") {
		t.Fatalf("unsupported CRL profile produced a conclusion: %+v", details)
	}
}

// TestPKCS1SignedDataFailureUsesReadPhase verifies ByteRange failures are classified as signed-data reads.
func TestPKCS1SignedDataFailureUsesReadPhase(t *testing.T) {
	contents, err := asn1.Marshal([]byte{1})
	if err != nil {
		t.Fatal(err)
	}
	sigDict := types.Dict{
		"Contents": types.HexLiteral(hex.EncodeToString(contents)),
	}
	_, err = verifyRSASHA1Signature(bytes.NewReader(nil), sigDict, &rsa.PublicKey{})
	if err == nil {
		t.Fatal("expected signed-data read failure")
	}
	for _, want := range []string{"read signed data", "signature dict entry ByteRange"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q, got %q", want, err)
		}
	}
}

// TestPKCS1SignedDataIOFailureIsFatal verifies legacy validation does not turn
// lower ReaderAt failures into signature evidence.
func TestPKCS1SignedDataIOFailureIsFatal(t *testing.T) {
	cause := errors.New("storage unavailable")
	key := testRSAKey(t)
	template := testCertTemplate("Legacy RSA Signer", true)
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := asn1.Marshal([]byte{1})
	if err != nil {
		t.Fatal(err)
	}
	contentsLiteral := types.HexLiteral(hex.EncodeToString(contents))
	sigDict := types.Dict{
		"ByteRange": byteRange(
			types.Integer(0),
			types.Integer(1),
			types.Integer(1+len(contentsLiteral.String())),
			types.Integer(0),
		),
		"Cert":     types.Array{types.HexLiteral(hex.EncodeToString(der))},
		"Contents": contentsLiteral,
	}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	err = ValidateX509RSASHA1Signature(
		signErrorReader{err: cause},
		sigDict,
		false,
		false,
		true,
		0,
		x509.NewCertPool(),
		result,
		&model.Context{Configuration: model.NewDefaultConfiguration()},
	)

	if err == nil {
		t.Fatal("expected fatal signed-data I/O error")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, %v) = false", err, cause)
	}
	if !strings.Contains(err.Error(), "read signed data") {
		t.Fatalf("missing signed-data phase: %v", err)
	}
	if len(result.Problems) != 0 {
		t.Fatalf("fatal I/O error was converted into Problems: %v", result.Problems)
	}
}

// TestPKCS7SignedDataFailureUsesReadPhase verifies PKCS#7 evidence uses the same signed-data phase.
func TestPKCS7SignedDataFailureUsesReadPhase(t *testing.T) {
	fixture := signerPKCS7(t, 1)
	sigDict := types.Dict{
		"Contents": types.HexLiteral(hex.EncodeToString(fixture)),
	}
	result := &model.SignatureValidationResult{}
	err := ValidatePKCS7Signatures(
		bytes.NewReader(nil),
		sigDict,
		false,
		false,
		true,
		0,
		x509.NewCertPool(),
		result,
		&model.Context{Configuration: model.NewDefaultConfiguration()},
	)
	if err != nil {
		t.Fatalf("expected reportable evidence, got %v", err)
	}
	problems := strings.Join(result.Problems, "\n")
	for _, want := range []string{"read signed data", "signature dict entry ByteRange"} {
		if !strings.Contains(problems, want) {
			t.Errorf("expected %q, got %q", want, problems)
		}
	}
}

// TestPKCS7SignedDataIOFailureIsFatal verifies lower ReaderAt failures remain
// fatal and discoverable instead of becoming signature Problems.
func TestPKCS7SignedDataIOFailureIsFatal(t *testing.T) {
	cause := errors.New("storage unavailable")
	fixture := signerPKCS7(t, 1)
	contentsLiteral := types.HexLiteral(hex.EncodeToString(fixture))
	sigDict := types.Dict{
		"ByteRange": byteRange(
			types.Integer(0),
			types.Integer(1),
			types.Integer(1+len(contentsLiteral.String())),
			types.Integer(0),
		),
		"Contents": contentsLiteral,
	}
	result := &model.SignatureValidationResult{}

	err := ValidatePKCS7Signatures(
		signErrorReader{err: cause},
		sigDict,
		false,
		false,
		true,
		0,
		x509.NewCertPool(),
		result,
		&model.Context{Configuration: model.NewDefaultConfiguration()},
	)

	if err == nil {
		t.Fatal("expected fatal signed-data I/O error")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, %v) = false", err, cause)
	}
	if !strings.Contains(err.Error(), "read signed data") {
		t.Fatalf("missing signed-data phase: %v", err)
	}
	if len(result.Problems) != 0 {
		t.Fatalf("fatal I/O error was converted into Problems: %v", result.Problems)
	}
}

// TestDSSCRLErrorRetainsEntryContext verifies the PDF-defined CRLs entry and array index are reported.
func TestDSSCRLErrorRetainsEntryContext(t *testing.T) {
	ctx := &model.Context{
		XRefTable: &model.XRefTable{
			DSS: types.Dict{
				"CRLs": types.Array{types.Integer(1)},
			},
		},
	}
	_, err := extractCRLsFromDSS(ctx)
	if err == nil {
		t.Fatal("expected malformed DSS CRLs failure")
	}
	for _, want := range []string{"DSS dict entry CRLs", "array index 1", "stream dictionary"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q, got %q", want, err)
		}
	}
}

// TestHandleDSSRecordsAvailableEvidenceWithoutPromotion verifies DSS observations do not imply LTV or a PAdES level.
func TestHandleDSSRecordsAvailableEvidenceWithoutPromotion(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	_, additionalCert := testCertChain(t, "DSS Root", "DSS Certificate")
	dssCRL := []byte{0x30, 0x00}
	dssOCSP := []byte{0x30, 0x00}
	fixture.ctx.DSS = types.Dict{
		"Certs": types.Array{types.StreamDict{Raw: additionalCert.Raw}},
		"CRLs":  types.Array{types.StreamDict{Raw: dssCRL}},
		"OCSPs": types.Array{types.StreamDict{Raw: dssOCSP}},
	}
	fixture.ctx.DTS = time.Now()
	certs := append([]*x509.Certificate(nil), fixture.certs...)
	crls := [][]byte{{1}}
	ocsps := [][]byte{{2}}
	signer := &model.Signer{PAdES: "B-T"}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	handleDSS(&certs, &crls, &ocsps, fixture.ctx, signer, result, true)

	if len(certs) != len(fixture.certs)+1 {
		t.Fatalf("got %d certificates, want %d", len(certs), len(fixture.certs)+1)
	}
	if len(crls) != 2 || !bytes.Equal(crls[0], []byte{1}) || !bytes.Equal(crls[1], dssCRL) {
		t.Fatalf("got CRLs %x, want existing and DSS evidence", crls)
	}
	if len(ocsps) != 2 || !bytes.Equal(ocsps[0], []byte{2}) || !bytes.Equal(ocsps[1], dssOCSP) {
		t.Fatalf("got OCSP responses %x, want existing and DSS evidence", ocsps)
	}
	if signer.LTVEnabled || signer.PAdES != "B-T" {
		t.Fatalf("got LTV=%t and PAdES=%q, want false and unchanged B-T", signer.LTVEnabled, signer.PAdES)
	}
	if len(signer.Problems) != 0 {
		t.Fatalf("unexpected problems %v", signer.Problems)
	}
	if result.Reason != model.SignatureReasonUnknown {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonUnknown)
	}
}

// TestHandleDSSRejectsPartialEvidence verifies malformed DSS components are reported without unsupported conclusions.
func TestHandleDSSRejectsPartialEvidence(t *testing.T) {
	tests := []struct {
		name string
		dss  types.Dict
		want string
	}{
		{"Certificates", types.Dict{"Certs": types.Array{types.Integer(1)}}, "DSS dict entry Certs"},
		{"CRLs", types.Dict{"CRLs": types.Array{types.Integer(1)}}, "DSS dict entry CRLs"},
		{"OCSPs", types.Dict{"OCSPs": types.Array{types.Integer(1)}}, "DSS dict entry OCSPs"},
		{"VRI", types.Dict{"VRI": types.Dict{}}, "DSS dict entry VRI: unsupported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newDetachedP7SignerFixture(t, nil, nil)
			fixture.ctx.DSS = tt.dss
			certs := append([]*x509.Certificate(nil), fixture.certs...)
			crls := [][]byte{{1}}
			ocsps := [][]byte{{2}}
			signer := &model.Signer{PAdES: "B-T"}
			result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

			handleDSS(&certs, &crls, &ocsps, fixture.ctx, signer, result, true)

			if len(certs) != len(fixture.certs) || !bytes.Equal(crls[0], []byte{1}) ||
				!bytes.Equal(ocsps[0], []byte{2}) {
				t.Fatal("malformed DSS changed partial evidence")
			}
			if signer.LTVEnabled || signer.PAdES != "B-T" {
				t.Fatalf("got LTV=%t and PAdES=%q, want false and B-T", signer.LTVEnabled, signer.PAdES)
			}
			if len(signer.Problems) != 1 || !strings.Contains(signer.Problems[0], tt.want) {
				t.Fatalf("got problems %v, want %q", signer.Problems, tt.want)
			}
			if result.Reason != model.SignatureReasonUnsupported {
				t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonUnsupported)
			}
		})
	}
}

// TestHandleDSSRetainsIndependentObservations verifies one malformed DSS component does not discard valid evidence.
func TestHandleDSSRetainsIndependentObservations(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	_, additionalCert := testCertChain(t, "DSS Root", "DSS Certificate")
	dssCRL := []byte{0x30, 0x00}
	fixture.ctx.DSS = types.Dict{
		"Certs": types.Array{types.StreamDict{Raw: additionalCert.Raw}},
		"CRLs":  types.Array{types.StreamDict{Raw: dssCRL}},
		"OCSPs": types.Array{types.Integer(1)},
	}
	certs := append([]*x509.Certificate(nil), fixture.certs...)
	var crls, ocsps [][]byte
	signer := &model.Signer{PAdES: "B-B"}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	handleDSS(&certs, &crls, &ocsps, fixture.ctx, signer, result, true)

	if len(certs) != len(fixture.certs)+1 {
		t.Fatalf("got %d certificates, want %d", len(certs), len(fixture.certs)+1)
	}
	if len(crls) != 1 || !bytes.Equal(crls[0], dssCRL) {
		t.Fatalf("got CRLs %x, want %x", crls, dssCRL)
	}
	if len(ocsps) != 0 {
		t.Fatalf("got malformed OCSP evidence %x, want none", ocsps)
	}
	if signer.LTVEnabled || signer.PAdES != "B-B" {
		t.Fatalf("got LTV=%t and PAdES=%q, want false and unchanged B-B", signer.LTVEnabled, signer.PAdES)
	}
	if len(signer.Problems) != 1 || !strings.Contains(signer.Problems[0], "DSS dict entry OCSPs") {
		t.Fatalf("got problems %v, want malformed OCSP evidence", signer.Problems)
	}
	if result.Reason != model.SignatureReasonUnsupported {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonUnsupported)
	}
}

// TestNormalizedCRLWording verifies stable CRL phase terminology.
func TestNormalizedCRLWording(t *testing.T) {
	_, err := checkCertAgainstCRL(
		&x509.Certificate{},
		nil,
		x509.NewCertPool(),
		nil,
		&model.Configuration{Offline: true},
	)
	if want := "CRL: offline"; err == nil || err.Error() != want {
		t.Errorf("offline CRL: got %v, want %q", err, want)
	}

	_, err = checkCertAgainstCRL(
		&x509.Certificate{},
		nil,
		x509.NewCertPool(),
		nil,
		&model.Configuration{},
	)
	if want := "CRL: certificate has no distribution point"; err == nil || err.Error() != want {
		t.Errorf("missing CRL distribution point: got %v, want %q", err, want)
	}
}

// TestCertPoolSubjectsCannotRecoverIssuerCertificate proves CertPool.Subjects does not expose complete certificates.
func TestCertPoolSubjectsCannotRecoverIssuerCertificate(t *testing.T) {
	issuer, leaf := testCertChain(t, "Issuer CA", "Leaf")
	roots := x509.NewCertPool()
	roots.AddCert(issuer)

	subjects := roots.Subjects()
	if len(subjects) != 1 {
		t.Fatalf("got %d subjects, want 1", len(subjects))
	}
	if _, err := x509.ParseCertificate(subjects[0]); err == nil {
		t.Fatal("CertPool.Subjects unexpectedly exposed a complete certificate")
	}

	_, err := checkCertViaOCSP(leaf, nil, roots, nil, &model.Configuration{})
	if want := "OCSP: certificate issuer unavailable"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("missing chain issuer: got %v, want %q", err, want)
	}
}

// TestCheckCertViaOCSPMissingIssuerDoesNotPanic verifies missing chain evidence returns a reportable error.
func TestCheckCertViaOCSPMissingIssuerDoesNotPanic(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("checkCertViaOCSP panicked: %v", recovered)
		}
	}()

	_, err := checkCertViaOCSP(
		&x509.Certificate{},
		nil,
		x509.NewCertPool(),
		nil,
		&model.Configuration{},
	)
	if want := "OCSP: certificate issuer unavailable"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("missing issuer: got %v, want %q", err, want)
	}
}

// TestMissingOCSPIssuerFallsBackToCRL verifies unavailable chain evidence does not stop the configured fallback.
func TestMissingOCSPIssuerFallsBackToCRL(t *testing.T) {
	signer := &model.Signer{}
	_, err := checkCertificateRevocation(
		&x509.Certificate{},
		nil,
		x509.NewCertPool(),
		signer,
		nil,
		nil,
		&model.Configuration{PreferredCertRevocationChecker: model.OCSP},
	)
	for _, want := range []string{
		"CRL: certificate has no distribution point",
		"OCSP: certificate issuer unavailable",
	} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("CRL fallback: got %v, want %q", err, want)
		}
	}
	problems := strings.Join(signer.Problems, "\n")
	if want := "OCSP: certificate issuer unavailable"; !strings.Contains(problems, want) {
		t.Fatalf("expected fallback evidence %q, got %q", want, problems)
	}
}

// TestBytesForByteRange verifies safe extraction of signature byte ranges.
func TestBytesForByteRange(t *testing.T) {
	ra := bytes.NewReader([]byte("0123456789"))
	arr := byteRange(types.Integer(0), types.Integer(3), types.Integer(7), types.Integer(3))

	got, err := bytesForByteRange(ra, arr)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte("012789"); !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestSignedDataRejectsAdversarialByteRanges verifies signed data starts at
// offset zero and excludes exactly the signature Contents hex literal.
func TestSignedDataRejectsAdversarialByteRanges(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		byteRange types.Array
		want      string
	}{
		{
			name:      "NonZeroStart",
			file:      "xAB<aa>CD",
			byteRange: byteRange(types.Integer(1), types.Integer(2), types.Integer(7), types.Integer(2)),
			want:      "first range must begin at offset 0",
		},
		{
			name:      "UnsignedPrefix",
			file:      "HEAD<aa>TAIL",
			byteRange: byteRange(types.Integer(0), types.Integer(0), types.Integer(4), types.Integer(8)),
			want:      "excluded gap does not match signature dict entry Contents",
		},
		{
			name:      "WrongContentsGap",
			file:      "HEAD<bb>TAIL",
			byteRange: byteRange(types.Integer(0), types.Integer(4), types.Integer(8), types.Integer(4)),
			want:      "excluded gap does not match signature dict entry Contents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sigDict := types.Dict{
				"ByteRange": tt.byteRange,
				"Contents":  types.HexLiteral("aa"),
			}
			_, err := signedData(bytes.NewReader([]byte(tt.file)), sigDict)
			if !errors.Is(err, errMalformedByteRange) {
				t.Fatalf("got %v, want malformed ByteRange evidence", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %q, want %q", err, tt.want)
			}
		})
	}
}

// TestBytesForByteRangeRejectsInvalidRanges verifies malformed range handling.
func TestBytesForByteRangeRejectsInvalidRanges(t *testing.T) {
	ra := bytes.NewReader([]byte("0123456789"))
	tests := []struct {
		name string
		arr  types.Array
		want string
	}{
		{"Length", byteRange(types.Integer(0), types.Integer(3)), "signature dict entry ByteRange: invalid length"},
		{"Type", byteRange(types.Integer(0), types.Name("x"), types.Integer(7), types.Integer(3)), "array index 2: expected integer"},
		{"Negative", byteRange(types.Integer(0), types.Integer(-1), types.Integer(7), types.Integer(3)), "array index 2: negative value"},
		{"Overlap", byteRange(types.Integer(0), types.Integer(5), types.Integer(4), types.Integer(3)), "signature dict entry ByteRange: overlapping ranges"},
		{"OffsetOverflow", byteRange(types.Integer(0), types.Integer(0), types.Integer(math.MaxInt), types.Integer(1)), "signature dict entry ByteRange: offset and length overflow"},
		{"OutOfBounds", byteRange(types.Integer(0), types.Integer(3), types.Integer(7), types.Integer(100)), "signature dict entry ByteRange, offset 7, length 100: read"},
		{"HugeOutOfBounds", byteRange(types.Integer(0), types.Integer(0), types.Integer(0), types.Integer(math.MaxInt)), "signature dict entry ByteRange, offset 0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := bytesForByteRange(ra, tt.arr)
			if err == nil {
				t.Fatal("expected invalid ByteRange error")
			}
			if !errors.Is(err, errMalformedByteRange) {
				t.Fatalf("expected malformed ByteRange classification, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, err)
			}
		})
	}
}

// TestBytesForByteRangePreservesFatalReadCause verifies unrelated ReaderAt
// failures are not classified as malformed PDF evidence.
func TestBytesForByteRangePreservesFatalReadCause(t *testing.T) {
	arr := byteRange(types.Integer(0), types.Integer(1), types.Integer(1), types.Integer(0))
	tests := []struct {
		name  string
		cause error
	}{
		{"Storage", errors.New("storage unavailable")},
		{"WrappedEOF", fmt.Errorf("device read: %w", io.EOF)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := bytesForByteRange(signErrorReader{err: tt.cause}, arr)
			if err == nil {
				t.Fatal("expected fatal ByteRange read error")
			}
			if errors.Is(err, errMalformedByteRange) {
				t.Fatalf("fatal read was classified as malformed ByteRange evidence: %v", err)
			}
			if !errors.Is(err, tt.cause) {
				t.Fatalf("errors.Is(%v, %v) = false", err, tt.cause)
			}
			if !isFatalByteRangeRead(err) {
				t.Fatalf("missing fatal ByteRange read classification: %v", err)
			}
		})
	}
}

// TestHandleCertVerifyErrReportsUnknownAuthority verifies trust-store failures are user actionable.
func TestHandleCertVerifyErrReportsUnknownAuthority(t *testing.T) {
	cert := &x509.Certificate{SerialNumber: big.NewInt(42)}
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	err := fmt.Errorf("wrapped: %w", x509.UnknownAuthorityError{Cert: cert})

	handleCertVerifyErr(err, cert, signer, result)

	if result.Reason != model.SignatureReasonCertNotTrusted {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertNotTrusted)
	}
	if len(signer.Problems) != 2 {
		t.Fatalf("got %d problems, want 2: %v", len(signer.Problems), signer.Problems)
	}
	if !strings.Contains(
		signer.Problems[0],
		"certificate path was not resolved using the configured local certificate store",
	) {
		t.Fatalf("missing local certificate-path problem: %q", signer.Problems[0])
	}
	if signer.Problems[1] != certImportHint {
		t.Fatalf("got hint %q, want %q", signer.Problems[1], certImportHint)
	}
}

// TestHandleCertParseErrReportsMalformedCertificate verifies x509 parse errors are classified as certificate problems.
func TestHandleCertParseErrReportsMalformedCertificate(t *testing.T) {
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	err := errors.New("x509: SAN rfc822Name is malformed")

	handleCertParseErr(err, result)

	if result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertInvalid)
	}
	if len(result.Problems) != 1 {
		t.Fatalf("got %d problems, want 1: %v", len(result.Problems), result.Problems)
	}
	if !strings.Contains(result.Problems[0], "certificate data is malformed") {
		t.Fatalf("missing malformed certificate problem: %q", result.Problems[0])
	}
}

// TestBuildP7CertChainsSetsPrimarySignerIdentity verifies signer identity reporting.
func TestBuildP7CertChainsSetsPrimarySignerIdentity(t *testing.T) {
	root, leaf := testCertChain(t, "Root CA", "Alice Signer")
	roots := x509.NewCertPool()
	roots.AddCert(root)
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	chains := buildP7CertChains(true, leaf, []*x509.Certificate{root, leaf}, roots, &model.Signer{}, result)

	if len(chains) == 0 {
		t.Fatal("expected certificate path resolved using the configured local certificate store")
	}
	if got := result.Details.SignerIdentity; got != "Alice Signer" {
		t.Fatalf("got signer identity %q, want Alice Signer", got)
	}
}

// TestBuildP7CertChainsDoesNotSetIdentityForUntrustedCert verifies issue 1271 behavior.
func TestBuildP7CertChainsDoesNotSetIdentityForUntrustedCert(t *testing.T) {
	root, leaf := testCertChain(t, "Root CA", "Alice Signer")
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	chains := buildP7CertChains(true, leaf, []*x509.Certificate{root, leaf}, x509.NewCertPool(), &model.Signer{}, result)

	if len(chains) != 0 {
		t.Fatal("expected unresolved certificate path")
	}
	if got := result.Details.SignerIdentity; got != "" {
		t.Fatalf("got signer identity %q, want empty", got)
	}
}

// TestBuildP7CertChainsDoesNotOverwriteSignerIdentity verifies secondary signer handling.
func TestBuildP7CertChainsDoesNotOverwriteSignerIdentity(t *testing.T) {
	root, leaf := testCertChain(t, "Root CA", "Alice Signer")
	roots := x509.NewCertPool()
	roots.AddCert(root)
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	result.Details.SignerIdentity = "Primary Signer"

	chains := buildP7CertChains(false, leaf, []*x509.Certificate{root, leaf}, roots, &model.Signer{}, result)

	if len(chains) == 0 {
		t.Fatal("expected certificate path resolved using the configured local certificate store")
	}
	if got := result.Details.SignerIdentity; got != "Primary Signer" {
		t.Fatalf("got signer identity %q, want Primary Signer", got)
	}
}

// TestCertChainIgnoresNilCertificates verifies malformed candidate
// collections do not panic or introduce gaps into fallback chain ordering.
func TestCertChainIgnoresNilCertificates(t *testing.T) {
	root, leaf := testCertChain(t, "Root CA", "Alice Signer")

	got := certChain(leaf, []*x509.Certificate{nil, root, leaf, nil})
	if len(got) != 2 || got[0] != leaf || got[1] != root {
		t.Fatalf("got fallback chain %v, want leaf followed by root", got)
	}
	if got := certChain(nil, []*x509.Certificate{nil, root}); got != nil {
		t.Fatalf("got fallback chain %v for missing starting certificate, want nil", got)
	}
}

// TestMergeCertsIgnoresNilCertificates verifies merged DSS and PKCS#7
// collections retain valid order and deduplication without nil entries.
func TestMergeCertsIgnoresNilCertificates(t *testing.T) {
	root, leaf := testCertChain(t, "Root CA", "Alice Signer")

	got := mergeCerts(
		[]*x509.Certificate{nil, leaf},
		[]*x509.Certificate{leaf, nil, root},
		[]*x509.Certificate{nil},
	)
	if len(got) != 2 || got[0] != leaf || got[1] != root {
		t.Fatalf("got merged certificates %v, want leaf followed by root", got)
	}
	if got := mergeCerts([]*x509.Certificate{nil}); len(got) != 0 {
		t.Fatalf("got merged certificates %v for nil-only input, want empty", got)
	}
}

// TestLocalCertificateAssessmentReproducesCurrentBehavior states that observed
// certificate evidence and its configuration-dependent local assessment
// preserve existing certificate-path behavior.
func TestLocalCertificateAssessmentReproducesCurrentBehavior(t *testing.T) {
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	assessment, err := assessCertificateEvidence(
		nil,
		false,
		x509.NewCertPool(),
		nil,
		nil,
		result.Reason,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if signer.Certificate != nil || len(signer.Problems) != 0 ||
		result.Reason != model.SignatureReasonUnknown {
		t.Fatalf("certificate evaluation mutated validation state: signer=%+v result=%+v", signer, result)
	}
	if assessment.Certificate != nil ||
		assessment.Reason != model.SignatureReasonCertInvalid ||
		len(assessment.Problems) != 1 ||
		assessment.Problems[0] != "certificate chain: missing" {
		t.Fatalf("unexpected certificate assessment: %+v", assessment)
	}

	applyCertificateAssessment(assessment, signer, result)

	if result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertInvalid)
	}
	if len(signer.Problems) != 1 || signer.Problems[0] != "certificate chain: missing" {
		t.Fatalf("got problems %v, want missing certificate chain evidence", signer.Problems)
	}
}

// TestLocalCertificateAssessmentDoesNotRecordHistoricalValidationTime verifies
// local certificate assessment does not populate compatibility fields for an
// applied historical assessment.
func TestLocalCertificateAssessmentDoesNotRecordHistoricalValidationTime(t *testing.T) {
	root, _ := testCertChain(t, "Root CA", "Alice Signer")

	assessment, err := assessCertificateEvidence(
		[][]*x509.Certificate{{root}},
		false,
		x509.NewCertPool(),
		nil,
		nil,
		model.SignatureReasonUnknown,
		model.NewDefaultConfiguration(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Certificate == nil {
		t.Fatal("missing certificate assessment")
	}
	if got := assessment.Certificate.ValidationTime; !got.Time.IsZero() {
		t.Fatalf("got applied historical validation time %+v, want zero value", got)
	}
}

// TestLocalCertificatePathBehaviorCharacterization locks the observed
// certificate-path evidence and configuration-dependent local assessment for a
// self-signed path.
func TestLocalCertificatePathBehaviorCharacterization(t *testing.T) {
	root, _ := testCertChain(t, "Root CA", "Alice Signer")
	roots := x509.NewCertPool()
	roots.AddCert(root)

	chains, err := root.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := assessCertificateEvidence(
		chains,
		true,
		roots,
		nil,
		nil,
		model.SignatureReasonUnknown,
		model.NewDefaultConfiguration(),
	)
	if err != nil {
		t.Fatal(err)
	}
	certificate := assessment.Certificate
	if certificate == nil ||
		certificate.Trust.Status != model.True ||
		certificate.Trust.Reason != "certificate path resolved using the configured local certificate store" ||
		certificate.PathEvidence.Method != model.CertificatePathMethodLocalTrustStore ||
		!certificate.CA ||
		!certificate.SelfSigned ||
		assessment.CertificatePathStatus != model.True ||
		certificate.PathEvidence.AssessmentScope != model.AssessmentScopeLocal {
		t.Fatalf("local certificate-path behavior changed: %+v", assessment)
	}
}

// TestLocalCRLBehaviorCharacterization locks the observed revocation evidence
// and configuration-dependent local assessment for an unauthenticated CRL.
func TestLocalCRLBehaviorCharacterization(t *testing.T) {
	issuer, _, cert := testCurrentCRLChain(t, "Certificate Issuer")
	otherIssuer, otherKey, _ := testCurrentCRLChain(t, "Other CRL Issuer")
	url := "https://crl.test/characterization"
	cert.CRLDistributionPoints = []string{url}
	now := time.Now()
	entry := x509.RevocationListEntry{
		SerialNumber:   cert.SerialNumber,
		RevocationTime: now.Add(-time.Minute),
	}
	bb := testCurrentCRL(
		t,
		otherIssuer,
		otherKey,
		now.Add(-time.Hour),
		now.Add(time.Hour),
		[]x509.RevocationListEntry{entry},
	)

	details, err := processCurrentCRLs(
		cert,
		issuer,
		testCurrentCRLClient(map[string][]byte{url: bb}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if details.Status != model.Unknown ||
		details.CRL == nil ||
		details.CRL.IssuerMatched != model.False ||
		details.CRL.SignatureValid != model.False ||
		len(details.CRL.Entries) != 1 {
		t.Fatalf("local CRL behavior changed: %+v", details)
	}
}

// TestLocalOCSPBehaviorCharacterization locks the observed revocation evidence
// and configuration-dependent local assessment for an archived delegated
// responder.
func TestLocalOCSPBehaviorCharacterization(t *testing.T) {
	now := time.Now().UTC()
	cert, issuer, _, bb := testArchivedOCSPFixture(
		t,
		now.Add(-time.Hour),
		now.Add(time.Hour),
		now.Add(time.Hour),
	)

	details, err := checkCertViaOCSP(
		cert,
		issuer,
		x509.NewCertPool(),
		[][]byte{bb},
		&model.Configuration{Offline: true},
	)
	if err == nil || !strings.Contains(err.Error(), "OCSP: offline") {
		t.Fatalf("got %v, want offline fallback failure", err)
	}
	if details == nil ||
		details.Status != model.Unknown ||
		details.OCSP == nil ||
		details.OCSP.Source != model.RevocationEvidenceSourceArchived ||
		details.OCSP.Responder != model.OCSPResponderDelegated ||
		details.OCSP.Authenticated != model.Unknown ||
		details.OCSP.Applicable != model.Unknown ||
		details.OCSP.ResponseSignatureValid != model.True ||
		details.OCSP.ResponderCertificateIssuedByIssuer != model.True ||
		details.OCSP.ResponderCertificateOCSPSigningValid != model.True ||
		details.OCSP.ResponderRevocation != model.Unknown {
		t.Fatalf("local OCSP behavior changed: %+v", details)
	}
}

// TestLocalTimestampBehaviorCharacterization locks standalone ctx.DTS
// containment without reconstructing validation evidence.
func TestLocalTimestampBehaviorCharacterization(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	documentTimestamp := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	fixture.ctx.DTS = documentTimestamp
	signer := &model.Signer{PAdES: "B-B"}
	result := &model.SignatureValidationResult{
		Status: model.SignatureStatusUnknown,
		Reason: model.SignatureReasonUnknown,
	}

	checkTimestampToken(
		fixture.signer,
		fixture.ctx,
		signer,
		result,
	)
	if signer.HasTimestamp ||
		!signer.Timestamp.IsZero() ||
		signer.PAdES != "B-B" ||
		result.Reason != model.SignatureReasonUnknown {
		t.Fatalf(
			"local timestamp behavior changed: signer=%+v result=%+v",
			signer,
			result,
		)
	}
}

// TestValidateCertChainsReportsMissingChain verifies missing chain evidence is reportable and does not panic.
func TestValidateCertChainsReportsMissingChain(t *testing.T) {
	tests := []struct {
		name   string
		chains [][]*x509.Certificate
	}{
		{"NoChains", nil},
		{"EmptyChain", [][]*x509.Certificate{{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := &model.Signer{}
			result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

			validateCertChains(tt.chains, false, x509.NewCertPool(), signer, nil, nil, result, nil)

			if result.Reason != model.SignatureReasonCertInvalid {
				t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertInvalid)
			}
			if len(signer.Problems) != 1 || signer.Problems[0] != "certificate chain: missing" {
				t.Fatalf("got problems %v, want missing certificate chain evidence", signer.Problems)
			}
		})
	}
}

// TestValidateCertChainsReportsNilCertificate verifies malformed chain entries are reportable and do not panic.
func TestValidateCertChainsReportsNilCertificate(t *testing.T) {
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	validateCertChains(
		[][]*x509.Certificate{{nil}},
		false,
		x509.NewCertPool(),
		signer,
		nil,
		nil,
		result,
		nil,
	)

	if result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertInvalid)
	}
	if len(signer.Problems) != 1 ||
		signer.Problems[0] != "certificate chain, certificate index 1: missing certificate" {
		t.Fatalf("got problems %v, want missing certificate evidence", signer.Problems)
	}
	if signer.Certificate == nil || !signer.Certificate.Leaf {
		t.Fatal("expected leaf certificate evidence")
	}
}

// TestValidateCertChainsOwnsCertificateDetails verifies stale signer details do not corrupt chain construction.
func TestValidateCertChainsOwnsCertificateDetails(t *testing.T) {
	root, _ := testCertChain(t, "Root CA", "Alice Signer")
	stale := &model.CertificateDetails{Subject: "stale"}
	signer := &model.Signer{Certificate: stale}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	validateCertChains(
		[][]*x509.Certificate{{root}},
		false,
		x509.NewCertPool(),
		signer,
		nil,
		nil,
		result,
		nil,
	)

	if signer.Certificate == stale {
		t.Fatal("expected certificate chain details to replace stale signer details")
	}
	if got := signer.Certificate.Subject; got != "Root CA" {
		t.Fatalf("got subject %q, want Root CA", got)
	}
}

// TestValidateCertChainsReportsPublicKeyContext verifies unsupported keys retain chain and certificate identity.
func TestValidateCertChainsReportsPublicKeyContext(t *testing.T) {
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject:      pkix.Name{CommonName: "Unsupported Key"},
		Issuer:       pkix.Name{CommonName: "Issuer"},
		PublicKey:    privateKey.PublicKey(),
	}
	signer := &model.Signer{}
	result := unknownSignatureResult()

	err = validateCertChains(
		[][]*x509.Certificate{{cert}},
		false,
		x509.NewCertPool(),
		signer,
		nil,
		nil,
		result,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Reason != model.SignatureReasonUnsupported {
		t.Fatalf("got reason %s, want unsupported evidence", result.Reason)
	}
	if result.Status != model.SignatureStatusUnknown {
		t.Fatalf("got status %s, want unknown for unsupported key", result.Status)
	}
	if result.Reason == model.SignatureReasonCertNotTrusted {
		t.Fatal("unsupported public key was classified as an unresolved local certificate path")
	}
	problems := strings.Join(signer.Problems, "\n")
	for _, want := range []string{
		"certificate chain, certificate index 1",
		`serial="2a"`,
		"inspect public key",
		"unsupported public key algorithm",
	} {
		if !strings.Contains(problems, want) {
			t.Errorf("expected %q, got %q", want, problems)
		}
	}
}

// TestPublicKeySizeRejectsStructurallyInvalidKeys verifies malformed key values return evidence instead of panicking.
func TestPublicKeySizeRejectsStructurallyInvalidKeys(t *testing.T) {
	tests := []struct {
		name string
		key  any
		want string
	}{
		{"RSA", (*rsa.PublicKey)(nil), "invalid RSA public key"},
		{"ECDSA", (*ecdsa.PublicKey)(nil), "invalid ECDSA public key"},
		{"Ed25519", ed25519.PublicKey{1}, "invalid Ed25519 public key"},
		{"DSA", (*dsa.PublicKey)(nil), "invalid DSA public key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := publicKeySize(&x509.Certificate{PublicKey: tt.key})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
			if !errors.Is(err, errMalformedPublicKey) {
				t.Fatalf("got %v, want malformed public-key classification", err)
			}
		})
	}
}

// TestValidateCertChainsClassifiesMalformedSupportedPublicKey verifies a
// malformed supported key is certificate-invalid evidence, not a path failure.
func TestValidateCertChainsClassifiesMalformedSupportedPublicKey(t *testing.T) {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(43),
		Subject:      pkix.Name{CommonName: "Malformed RSA Key"},
		Issuer:       pkix.Name{CommonName: "Issuer"},
		PublicKey:    (*rsa.PublicKey)(nil),
	}
	signer := &model.Signer{}
	result := unknownSignatureResult()

	if err := validateCertChains(
		[][]*x509.Certificate{{cert}},
		false,
		x509.NewCertPool(),
		signer,
		nil,
		nil,
		result,
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertInvalid)
	}
	if result.Reason == model.SignatureReasonCertNotTrusted {
		t.Fatal("malformed public key was classified as an unresolved local certificate path")
	}
	if problems := strings.Join(signer.Problems, "\n"); !strings.Contains(problems, "malformed public key") {
		t.Fatalf("missing malformed public-key evidence: %q", problems)
	}
}

// TestGetKeySizePropagatesOperationalFailure verifies an actual helper
// operation failure remains fatal and discoverable without becoming evidence.
func TestGetKeySizePropagatesOperationalFailure(t *testing.T) {
	cause := errors.New("key service unavailable")
	signer := &model.Signer{}
	result := unknownSignatureResult()
	cert := &x509.Certificate{SerialNumber: big.NewInt(44)}

	_, ok, err := getKeySizeWith(
		cert,
		signer,
		&model.CertificateDetails{},
		result,
		0,
		func(*x509.Certificate) (int, error) {
			return 0, cause
		},
	)
	if err == nil || ok {
		t.Fatalf("got ok=%t err=%v, want fatal helper failure", ok, err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, cause) = false", err)
	}
	if len(signer.Problems) != 0 ||
		result.Reason != model.SignatureReasonUnknown {
		t.Fatalf("operational failure became evidence: signer=%+v result=%+v", signer, result)
	}
}

// TestValidateCertChainsReportsSelfSignedVerificationEvidence verifies forged self-issued certificates do not panic.
func TestValidateCertChainsReportsSelfSignedVerificationEvidence(t *testing.T) {
	root, _ := testCertChain(t, "Root CA", "Alice Signer")
	cert := *root
	cert.Signature = append([]byte(nil), root.Signature...)
	cert.Signature[0] ^= 0xff
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	validateCertChains(
		[][]*x509.Certificate{{&cert}},
		false,
		x509.NewCertPool(),
		signer,
		nil,
		nil,
		result,
		nil,
	)

	if result.Reason != model.SignatureReasonSelfSignedCertErr {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonSelfSignedCertErr)
	}
	if len(signer.Problems) != 1 {
		t.Fatalf("got problems %v, want one verification problem", signer.Problems)
	}
	for _, want := range []string{"certificate chain, certificate index 1", "verify self-signed certificate"} {
		if !strings.Contains(signer.Problems[0], want) {
			t.Errorf("expected %q, got %q", want, signer.Problems[0])
		}
	}
	if strings.HasSuffix(signer.Problems[0], "\n") {
		t.Fatalf("unexpected trailing newline in %q", signer.Problems[0])
	}
}

// TestValidateCertChainsReportsRevocationCertificateContext verifies terminal revocation failures identify the certificate.
func TestValidateCertChainsReportsRevocationCertificateContext(t *testing.T) {
	root, leaf := testCertChain(t, "Root CA", "Alice Signer")
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	conf := model.NewDefaultConfiguration()
	conf.Offline = true

	validateCertChains(
		[][]*x509.Certificate{{leaf, root}},
		false,
		x509.NewCertPool(),
		signer,
		nil,
		nil,
		result,
		conf,
	)

	if result.Reason != model.SignatureReasonCertRevocationUnknown {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertRevocationUnknown)
	}
	problems := strings.Join(signer.Problems, "\n")
	for _, want := range []string{
		"certificate revocation check",
		certInfo(leaf),
		"CRL: offline",
		"OCSP: offline",
	} {
		if !strings.Contains(problems, want) {
			t.Errorf("expected %q, got %q", want, problems)
		}
	}
}

// TestCertificateIssuerUsesAdjacentChainEntry verifies issuer resolution follows the selected chain path.
func TestCertificateIssuerUsesAdjacentChainEntry(t *testing.T) {
	leaf := &x509.Certificate{}
	intermediate := &x509.Certificate{}
	root := &x509.Certificate{}
	chain := []*x509.Certificate{leaf, intermediate, root}

	if got := certificateIssuer(chain, 0); got != intermediate {
		t.Fatalf("got leaf issuer %p, want intermediate %p", got, intermediate)
	}
	if got := certificateIssuer(chain, 1); got != root {
		t.Fatalf("got intermediate issuer %p, want root %p", got, root)
	}
	if got := certificateIssuer(chain, 2); got != nil {
		t.Fatalf("got root issuer %p, want nil", got)
	}
	if got := certificateIssuer(chain, -1); got != nil {
		t.Fatalf("got negative-index issuer %p, want nil", got)
	}
}

func testCertChain(t *testing.T, rootCN, leafCN string) (*x509.Certificate, *x509.Certificate) {
	t.Helper()

	rootKey := testRSAKey(t)
	rootTemplate := testCertTemplate(rootCN, true)
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}

	leafKey := testRSAKey(t)
	leafTemplate := testCertTemplate(leafCN, false)
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, root, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatal(err)
	}
	return root, leaf
}

func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func testCertificate(
	t *testing.T,
	template, parent *x509.Certificate,
	publicKey, privateKey any,
) *x509.Certificate {
	t.Helper()
	der, err := x509.CreateCertificate(rand.Reader, template, parent, publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func testCertTemplate(commonName string, ca bool) *x509.Certificate {
	keyUsage := x509.KeyUsageDigitalSignature
	if ca {
		keyUsage |= x509.KeyUsageCertSign
	}
	return &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              keyUsage,
		BasicConstraintsValid: true,
		IsCA:                  ca,
	}
}
