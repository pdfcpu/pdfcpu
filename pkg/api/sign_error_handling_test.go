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

package api

import (
	"bytes"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/pkcs7"
)

type signErrorWriter struct {
	err error
}

type signPanicReadSeekerAt struct{}

type signErrorReadSeekerAt struct {
	err error
}

type signReadSeekerAtFailure struct {
	*bytes.Reader
	err error
}

// TestSignatureStatsRetainsMultiSignatureIncrements verifies counters mutate
// the returned aggregate instead of a value-receiver copy.
func TestSignatureStatsRetainsMultiSignatureIncrements(t *testing.T) {
	results := []*model.SignatureValidationResult{
		{Signature: model.Signature{Type: model.SigTypeForm, Signed: true, Visible: true}},
		{Signature: model.Signature{Type: model.SigTypeForm, Signed: true}},
		{Signature: model.Signature{Type: model.SigTypePage}},
		{Signature: model.Signature{Type: model.SigTypeUR, Signed: true}},
		{Signature: model.Signature{Type: model.SigTypeDTS, Signed: true, Visible: true}},
	}
	got := signatureStats(results)
	want := model.SignatureStats{
		FormSigned:        2,
		FormSignedVisible: 1,
		PageUnsigned:      1,
		URSigned:          1,
		DTSSigned:         1,
		DTSSignedVisible:  1,
		Total:             len(results),
	}
	if got != want {
		t.Fatalf("signature counters lost increments:\ngot:  %+v\nwant: %+v", got, want)
	}
}

// TestDigestAndModelUseStoredSignatureText verifies concise API output and
// model output use the same source reason and do not rewrite stored Problems.
func TestDigestAndModelUseStoredSignatureText(t *testing.T) {
	tests := []struct {
		name       string
		sigType    int
		typeString string
	}{
		{"Ordinary", model.SigTypeForm, "form signature (invisible, signed)"},
		{"DTS", model.SigTypeDTS, "document timestamp (not locally validated, invisible, signed)"},
		{"UsageRights", model.SigTypeUR, "usage rights signature (invisible, signed)"},
	}
	reason := model.SignatureReasonCertNotTrusted.String()
	const problem = "problem text: KEEP /Name <Value> unchanged"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &model.SignatureValidationResult{
				Signature: model.Signature{Type: tt.sigType, Signed: true},
				Status:    model.SignatureStatusUnknown,
				Reason:    model.SignatureReasonCertNotTrusted,
			}
			apiOutput := strings.Join(digest([]*model.SignatureValidationResult{result}, false), "\n")
			modelOutput := result.String()
			for name, output := range map[string]string{
				"API":   apiOutput,
				"model": modelOutput,
			} {
				if !strings.Contains(output, tt.typeString) ||
					!strings.Contains(output, "Reason: "+reason) {
					t.Errorf("%s output changed structured conclusion:\n%s", name, output)
				}
			}

			result.Reason = model.SignatureReasonInternal
			result.Problems = []string{problem}
			apiOutput = strings.Join(digest([]*model.SignatureValidationResult{result}, false), "\n")
			if !strings.Contains(apiOutput, "Reason: "+problem) {
				t.Fatalf("API digest rewrote stored Problem:\n%s", apiOutput)
			}
			if modelOutput = result.String(); !strings.Contains(modelOutput, "Problems: "+problem) {
				t.Fatalf("model output rewrote stored Problem:\n%s", modelOutput)
			}
		})
	}
}

// TestCompactDigestPreservesDetailedEvidenceText locks the deliberate compact
// presentation of the first Problem for nonfatal implementation, malformed and
// unsupported evidence. Full output retains the structured reason separately.
func TestCompactDigestPreservesDetailedEvidenceText(t *testing.T) {
	tests := []struct {
		reason  model.SignatureReason
		problem string
	}{
		{model.SignatureReasonInternal, "implementation invariant detail"},
		{model.SignatureReasonMalformed, "signature dict entry ByteRange: malformed detail"},
		{model.SignatureReasonUnsupported, "SubFilter Example.Filter: unsupported detail"},
	}
	for _, tt := range tests {
		result := &model.SignatureValidationResult{
			Signature: model.Signature{Type: model.SigTypeForm, Signed: true},
			Status:    model.SignatureStatusUnknown,
			Reason:    tt.reason,
			Problems:  []string{tt.problem},
		}
		got := strings.Join(digest([]*model.SignatureValidationResult{result}, false), "\n")
		want := "\n" +
			"1 form signature (invisible, signed)\n" +
			"   Status: validity of the signature is unknown\n" +
			"   Reason: " + tt.problem + "\n" +
			"   Signed: not available"
		if got != want {
			t.Errorf("compact output:\ngot:\n%q\nwant:\n%q", got, want)
		}

		full := strings.Join(digest([]*model.SignatureValidationResult{result}, true), "\n")
		if !strings.Contains(full, "Reason: "+tt.reason.String()) ||
			!strings.Contains(full, "Problems: "+tt.problem) {
			t.Errorf("full output did not separate structured reason and Problem:\n%s", full)
		}
	}
}

// Write returns the configured test error.
func (w signErrorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func (r signErrorReadSeekerAt) Read([]byte) (int, error) {
	return 0, r.err
}

func (r signErrorReadSeekerAt) ReadAt([]byte, int64) (int, error) {
	return 0, r.err
}

func (r signErrorReadSeekerAt) Seek(int64, int) (int64, error) {
	return 0, r.err
}

func (r signReadSeekerAtFailure) ReadAt([]byte, int64) (int, error) {
	return 0, r.err
}

func (signPanicReadSeekerAt) Read([]byte) (int, error) {
	fault.Fail("read fault")
	return 0, nil
}

func (signPanicReadSeekerAt) ReadAt([]byte, int64) (int, error) {
	fault.Fail("read-at fault")
	return 0, nil
}

func (signPanicReadSeekerAt) Seek(int64, int) (int64, error) {
	fault.Fail("seek fault")
	return 0, nil
}

func signedPDFBytes(t *testing.T) []byte {
	t.Helper()
	inFile := filepath.Join("..", "samples", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf")
	bb, err := os.ReadFile(inFile)
	if err != nil {
		t.Fatal(err)
	}
	return bb
}

func signatureContents(t *testing.T, pdf []byte) ([]byte, int, int) {
	t.Helper()
	marker := []byte("/Contents <")
	start := bytes.Index(pdf, marker)
	if start < 0 {
		t.Fatal("missing signature dict entry Contents")
	}
	start += len(marker)
	endOffset := bytes.IndexByte(pdf[start:], '>')
	if endOffset < 0 {
		t.Fatal("unterminated signature dict entry Contents")
	}
	end := start + endOffset
	contents := make([]byte, hex.DecodedLen(end-start))
	n, err := hex.Decode(contents, pdf[start:end])
	if err != nil {
		t.Fatal(err)
	}
	return contents[:n], start, end
}

func mutateSignatureContents(t *testing.T, pdf []byte, mutate func([]byte)) []byte {
	t.Helper()
	contents, start, end := signatureContents(t, pdf)
	mutate(contents)
	if hex.EncodedLen(len(contents)) != end-start {
		t.Fatal("signature Contents mutation changed encoded length")
	}
	result := append([]byte(nil), pdf...)
	hex.Encode(result[start:end], contents)
	return result
}

func rawSignatureEvidence(t *testing.T, pdf []byte) (*model.SignatureValidationResult, string) {
	t.Helper()
	conf := model.NewDefaultConfiguration()
	conf.Offline = true
	results, err := ValidateSignaturesRaw(bytes.NewReader(pdf), false, conf)
	if err != nil {
		t.Fatalf("expected reportable signature evidence, got fatal error %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected signature validation result")
	}
	var problems []string
	for _, result := range results {
		problems = append(problems, result.Problems...)
		for _, signer := range result.Details.Signers {
			problems = append(problems, signer.Problems...)
		}
	}
	return results[0], strings.Join(problems, "\n")
}

// TestSignAPIMissingArgumentsPreserveSentinels verifies public sign API argument guards.
func TestSignAPIMissingArgumentsPreserveSentinels(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "validate signatures input",
			err: func() error {
				_, err := ValidateSignatures("", false, nil)
				return err
			}(),
			want: ErrMissingPDFInput,
		},
		{
			name: "validate signatures file input",
			err: func() error {
				_, err := ValidateSignaturesFile("", false, false, nil)
				return err
			}(),
			want: ErrMissingPDFInput,
		},
		{
			name: "remove signatures reader",
			err:  RemoveSignatures(nil, io.Discard, nil),
			want: ErrMissingPDFReadSeeker,
		},
		{
			name: "remove signatures writer",
			err:  RemoveSignatures(bytes.NewReader(nil), nil, nil),
			want: ErrMissingPDFWriter,
		},
		{
			name: "remove signatures file input",
			err:  RemoveSignaturesFile("", "", nil),
			want: ErrMissingPDFInput,
		},
	}

	for _, tt := range tests {
		if !errors.Is(tt.err, tt.want) {
			t.Errorf("%s: expected %v, got %v", tt.name, tt.want, tt.err)
		}
	}
}

// TestValidateSignaturesRawRejectsMissingInput verifies the stream API preserves its missing-reader sentinel.
func TestValidateSignaturesRawRejectsMissingInput(t *testing.T) {
	_, err := ValidateSignaturesRaw(nil, false, nil)
	if !errors.Is(err, ErrMissingPDFReadSeeker) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFReadSeeker, err)
	}
}

// TestValidateSignaturesRawRejectsMalformedInput verifies every stream preparation phase is retained.
func TestValidateSignaturesRawRejectsMalformedInput(t *testing.T) {
	_, err := ValidateSignaturesRaw(bytes.NewReader([]byte("not a PDF")), false, nil)
	if err == nil {
		t.Fatal("expected malformed stream failure")
	}
	for _, want := range []string{"validate signatures", "prepare PDF context", "read context"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected phase %q, got %q", want, err)
		}
	}
}

// TestValidateSignaturesRawStructuralFailurePreservesCause verifies PDF construction failures remain fatal and wrapped.
func TestValidateSignaturesRawStructuralFailurePreservesCause(t *testing.T) {
	cause := errors.New("structural read failure")

	results, err := ValidateSignaturesRaw(signErrorReadSeekerAt{err: cause}, false, nil)
	if results != nil {
		t.Fatalf("got results %v, want nil", results)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("expected structural cause %v, got %v", cause, err)
	}
	for _, want := range []string{"validate signatures", "prepare PDF context", "read context"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected phase %q, got %q", want, err)
		}
	}
}

// TestValidateSignaturesRawUnsignedPDFPrecedesTrustPool verifies semantic input evidence wins over trust setup.
func TestValidateSignaturesRawUnsignedPDFPrecedesTrustPool(t *testing.T) {
	inFile := filepath.Join("..", "samples", "bookmarks", "bookmarkSimple.pdf")
	bb, err := os.ReadFile(inFile)
	if err != nil {
		t.Fatal(err)
	}

	oldDir := model.TrustedCertDir
	model.TrustedCertDir = filepath.Join(t.TempDir(), "missing")
	pdfcpu.InvalidateCertificatePool()
	t.Cleanup(func() {
		model.TrustedCertDir = oldDir
		pdfcpu.InvalidateCertificatePool()
	})

	_, err = ValidateSignaturesRaw(bytes.NewReader(bb), false, nil)
	if !errors.Is(err, ErrNoSignatures) {
		t.Fatalf("expected %v, got %v", ErrNoSignatures, err)
	}
	if strings.Contains(err.Error(), "load trust pool") {
		t.Fatalf("trust-pool failure took precedence: %q", err)
	}
}

// TestValidateSignaturesRawUnsignedPDFSkipsDomainOperation verifies unsigned input stops before trust and verification.
func TestValidateSignaturesRawUnsignedPDFSkipsDomainOperation(t *testing.T) {
	inFile := filepath.Join("..", "samples", "bookmarks", "bookmarkSimple.pdf")
	bb, err := os.ReadFile(inFile)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	operation := func(io.ReaderAt, *model.Context, bool) ([]*model.SignatureValidationResult, error) {
		called = true
		return nil, errors.New("unexpected domain operation")
	}

	results, err := validateSignaturesRaw(bytes.NewReader(bb), false, nil, operation)
	if results != nil {
		t.Fatalf("got results %v, want nil", results)
	}
	if !errors.Is(err, ErrNoSignatures) {
		t.Fatalf("expected %v, got %v", ErrNoSignatures, err)
	}
	if called {
		t.Fatal("domain signature operation was invoked for unsigned input")
	}
	if strings.Contains(err.Error(), "load trust pool") ||
		strings.Contains(err.Error(), "verify signatures") {
		t.Fatalf("later phase took precedence: %q", err)
	}
}

// TestValidateSignaturesRawRecoversPanics verifies the stream API returns recovered boundary failures.
func TestValidateSignaturesRawRecoversPanics(t *testing.T) {
	_, err := ValidateSignaturesRaw(signPanicReadSeekerAt{}, false, nil)
	if err == nil {
		t.Fatal("expected recovered stream failure")
	}
	if want := "seek fault"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

// TestValidateSignaturesRawReadsSignedBytesReader verifies the raw operation does not depend on *os.File.
func TestValidateSignaturesRawReadsSignedBytesReader(t *testing.T) {
	rs := bytes.NewReader(signedPDFBytes(t))
	results, err := ValidateSignaturesRaw(rs, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected signature validation result")
	}
}

// TestValidateSignaturesRawPreservesSignedDataReadFailure verifies positional
// signature I/O failures remain fatal across the API boundary.
func TestValidateSignaturesRawPreservesSignedDataReadFailure(t *testing.T) {
	cause := errors.New("storage unavailable")
	rs := signReadSeekerAtFailure{
		Reader: bytes.NewReader(signedPDFBytes(t)),
		err:    cause,
	}

	_, err := ValidateSignaturesRaw(rs, false, nil)
	if err == nil {
		t.Fatal("expected fatal signed-data read error")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, %v) = false", err, cause)
	}
	for _, want := range []string{"validate signatures: verify signatures", "read signed data", "signature dict entry ByteRange"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q, got %v", want, err)
		}
	}
}

// TestValidateSignaturesRawInitializesOperationState verifies default configuration and flags reach the domain boundary.
func TestValidateSignaturesRawInitializesOperationState(t *testing.T) {
	inFile := filepath.Join("..", "samples", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf")
	bb, err := os.ReadFile(inFile)
	if err != nil {
		t.Fatal(err)
	}

	called := false
	operation := func(_ io.ReaderAt, ctx *model.Context, all bool) ([]*model.SignatureValidationResult, error) {
		called = true
		if ctx.Configuration == nil {
			t.Fatal("expected initialized configuration")
		}
		if ctx.Configuration.Cmd != model.VALIDATESIGNATURES {
			t.Fatalf("got command %v, want %v", ctx.Configuration.Cmd, model.VALIDATESIGNATURES)
		}
		if !all {
			t.Fatal("expected all-signatures flag at domain boundary")
		}
		return []*model.SignatureValidationResult{{}}, nil
	}

	results, err := validateSignaturesRaw(bytes.NewReader(bb), true, nil, operation)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected domain operation")
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

// TestValidateSignaturesRawReportsMalformedSignatureEvidence verifies malformed signature metadata is nonfatal.
func TestValidateSignaturesRawReportsMalformedSignatureEvidence(t *testing.T) {
	pdf := signedPDFBytes(t)
	oldValue := []byte("ETSI.CAdES.detached")
	newValue := []byte("Unknown.Filter.Test")
	if len(oldValue) != len(newValue) {
		t.Fatal("SubFilter mutation must preserve PDF offsets")
	}
	index := bytes.Index(pdf, oldValue)
	if index < 0 {
		t.Fatal("missing SubFilter fixture value")
	}
	pdf = append([]byte(nil), pdf...)
	copy(pdf[index:index+len(oldValue)], newValue)

	result, problems := rawSignatureEvidence(t, pdf)
	if result.Status != model.SignatureStatusUnknown {
		t.Fatalf("got status %s, want unknown", result.Status)
	}
	if want := "signature dict entry SubFilter: unsupported: value Unknown.Filter.Test"; !strings.Contains(problems, want) {
		t.Fatalf("expected %q, got %q", want, problems)
	}
}

// TestValidateSignaturesRawReportsUnsupportedAlgorithmEvidence verifies unknown PKCS#7 algorithms do not panic.
func TestValidateSignaturesRawReportsUnsupportedAlgorithmEvidence(t *testing.T) {
	pdf := mutateSignatureContents(t, signedPDFBytes(t), func(contents []byte) {
		oid, err := asn1.Marshal(pkcs7.OIDEncryptionAlgorithmRSASHA256)
		if err != nil {
			t.Fatal(err)
		}
		index := bytes.LastIndex(contents, oid)
		if index < 0 {
			t.Fatal("missing PKCS#7 signature algorithm")
		}
		contents[index+len(oid)-1] = 0x7f
	})

	result, problems := rawSignatureEvidence(t, pdf)
	if result.Status != model.SignatureStatusUnknown ||
		result.Reason != model.SignatureReasonUnsupported {
		t.Fatalf("got status=%s reason=%s, want unknown and unsupported", result.Status, result.Reason)
	}
	if want := "pkcs7: verify signature unsupported"; !strings.Contains(problems, want) {
		t.Fatalf("expected %q, got %q", want, problems)
	}
}

// TestValidateSignaturesRawReportsCertificateParseEvidence verifies malformed embedded certificates remain evidence.
func TestValidateSignaturesRawReportsCertificateParseEvidence(t *testing.T) {
	pdf := mutateSignatureContents(t, signedPDFBytes(t), func(contents []byte) {
		p7, err := pkcs7.Parse(contents)
		if err != nil {
			t.Fatal(err)
		}
		if len(p7.Certificates) == 0 {
			t.Fatal("missing embedded certificate")
		}
		raw := p7.Certificates[0].Raw
		certOffset := bytes.Index(contents, raw)
		if certOffset < 0 {
			t.Fatal("embedded certificate not found in PKCS#7 data")
		}
		var envelope asn1.RawValue
		rest, err := asn1.Unmarshal(raw, &envelope)
		if err != nil || len(rest) > 0 {
			t.Fatalf("parse embedded certificate envelope: %v", err)
		}
		headerLen := len(raw) - len(envelope.Bytes)
		contents[certOffset+headerLen] = byte(asn1.TagSet)
	})

	result, problems := rawSignatureEvidence(t, pdf)
	if result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("got reason %s, want certificate invalid", result.Reason)
	}
	for _, want := range []string{"PKCS#7 embedded certificates", "parse certificate", "certificate data is malformed"} {
		if !strings.Contains(problems, want) {
			t.Errorf("expected %q, got %q", want, problems)
		}
	}
}

// TestValidateSignaturesRawReportsDigestMismatchEvidence verifies historical
// signed-byte tampering remains nonfatal modification evidence.
func TestValidateSignaturesRawReportsDigestMismatchEvidence(t *testing.T) {
	pdf := signedPDFBytes(t)
	_, contentsStart, _ := signatureContents(t, pdf)
	marker := []byte("/M (D:")
	index := bytes.LastIndex(pdf[:contentsStart], marker)
	if index < 0 {
		t.Fatal("missing signed signature date")
	}
	index += len(marker) + 13
	pdf = append([]byte(nil), pdf...)
	if pdf[index] == '9' {
		pdf[index] = '8'
	} else {
		pdf[index]++
	}

	result, problems := rawSignatureEvidence(t, pdf)
	if result.Status != model.SignatureStatusInvalid ||
		result.Reason != model.SignatureReasonDocModified ||
		result.DocModified != model.True {
		t.Fatalf(
			"got status=%s reason=%s modified=%d, want invalid and modified",
			result.Status,
			result.Reason,
			result.DocModified,
		)
	}
	if want := "pkcs7: verify signature content mismatch"; !strings.Contains(problems, want) {
		t.Fatalf("expected %q, got %q", want, problems)
	}
}

// TestValidateSignaturesRawDomainErrorPreservesCauseAndPhase verifies API-domain boundary wrapping.
func TestValidateSignaturesRawDomainErrorPreservesCauseAndPhase(t *testing.T) {
	cause := errors.New("domain signature failure")
	operation := func(io.ReaderAt, *model.Context, bool) ([]*model.SignatureValidationResult, error) {
		return nil, cause
	}

	inFile := filepath.Join("..", "samples", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf")
	bb, err := os.ReadFile(inFile)
	if err != nil {
		t.Fatal(err)
	}

	_, err = validateSignaturesRaw(bytes.NewReader(bb), false, nil, operation)
	if !errors.Is(err, cause) {
		t.Fatalf("expected %v, got %v", cause, err)
	}
	if want := "validate signatures: verify signatures"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

// TestValidateSignaturesFileLifecycleJoinsOperationAndCloseErrors verifies production cleanup preserves both failures.
func TestValidateSignaturesFileLifecycleJoinsOperationAndCloseErrors(t *testing.T) {
	cause := errors.New("domain signature failure")
	operation := func(ra io.ReaderAt, _ *model.Context, _ bool) ([]*model.SignatureValidationResult, error) {
		f, ok := ra.(*os.File)
		if !ok {
			return nil, errors.New("domain input is not *os.File")
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
		return nil, cause
	}

	inFile := filepath.Join("..", "samples", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf")
	_, err := validateSignaturesFile(inFile, false, nil, operation)
	for _, want := range []error{cause, fs.ErrClosed} {
		if !errors.Is(err, want) {
			t.Errorf("expected joined cause %v, got %v", want, err)
		}
	}
	for _, want := range []string{"validate signatures: verify signatures", "validate signatures: close input"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q, got %q", want, err)
		}
	}
}

// TestValidateSignaturesFileLifecycleReturnsCloseErrorAfterSuccess verifies cleanup failure changes a successful result.
func TestValidateSignaturesFileLifecycleReturnsCloseErrorAfterSuccess(t *testing.T) {
	wantResults := []*model.SignatureValidationResult{{}}
	operation := func(ra io.ReaderAt, _ *model.Context, _ bool) ([]*model.SignatureValidationResult, error) {
		f, ok := ra.(*os.File)
		if !ok {
			return nil, errors.New("domain input is not *os.File")
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
		return wantResults, nil
	}

	inFile := filepath.Join("..", "samples", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf")
	results, err := validateSignaturesFile(inFile, false, nil, operation)
	if !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("expected close failure %v, got results %v and error %v", fs.ErrClosed, results, err)
	}
	if want := "validate signatures: close input"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

// TestValidateSignaturesPDFErrorsPrecedeTrustPoolError verifies trust loading does not mask input or semantic errors.
func TestValidateSignaturesPDFErrorsPrecedeTrustPoolError(t *testing.T) {
	oldDir := model.TrustedCertDir
	model.TrustedCertDir = filepath.Join(t.TempDir(), "missing")
	pdfcpu.InvalidateCertificatePool()
	t.Cleanup(func() {
		model.TrustedCertDir = oldDir
		pdfcpu.InvalidateCertificatePool()
	})

	tests := []struct {
		name   string
		inFile string
		want   error
		phase  string
	}{
		{
			name:   "open input",
			inFile: filepath.Join(t.TempDir(), "missing.pdf"),
			want:   os.ErrNotExist,
			phase:  "validate signatures: open input",
		},
		{
			name:   "no signatures",
			inFile: filepath.Join("..", "samples", "bookmarks", "bookmarkSimple.pdf"),
			want:   ErrNoSignatures,
			phase:  "validate signatures",
		},
	}

	for _, tt := range tests {
		_, err := ValidateSignatures(tt.inFile, false, nil)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: expected %v, got %v", tt.name, tt.want, err)
			continue
		}
		if !strings.Contains(err.Error(), tt.phase) {
			t.Errorf("%s: expected %q, got %q", tt.name, tt.phase, err)
		}
		if strings.Contains(err.Error(), "load trust pool") {
			t.Errorf("%s: trust-pool failure took precedence: %q", tt.name, err)
		}
	}
}

// TestValidateSignaturesTrustPoolErrorContext verifies trust failures surface after successful signed-PDF parsing.
func TestValidateSignaturesTrustPoolErrorContext(t *testing.T) {
	oldDir := model.TrustedCertDir
	model.TrustedCertDir = filepath.Join(t.TempDir(), "missing")
	pdfcpu.InvalidateCertificatePool()
	t.Cleanup(func() {
		model.TrustedCertDir = oldDir
		pdfcpu.InvalidateCertificatePool()
	})

	inFile := filepath.Join("..", "samples", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf")
	_, err := ValidateSignatures(inFile, false, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if want := "validate signatures: load trust pool"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

// TestSignAPIOpenErrorsPreserveCauseAndContext verifies file-open phase reporting.
func TestSignAPIOpenErrorsPreserveCauseAndContext(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.pdf")
	tests := []struct {
		name string
		call func() error
		want string
	}{
		{
			name: "validate signatures",
			call: func() error {
				_, err := ValidateSignatures(missing, false, nil)
				return err
			},
			want: "validate signatures: open input " + missing,
		},
		{
			name: "remove signatures",
			call: func() error {
				return RemoveSignaturesFile(missing, "", nil)
			},
			want: "remove signatures: open input " + missing,
		},
	}

	for _, tt := range tests {
		err := tt.call()
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s: expected %v, got %v", tt.name, os.ErrNotExist, err)
			continue
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: expected %q, got %q", tt.name, tt.want, err)
		}
	}
}

// TestSignAPIMalformedPDFErrorsIncludeOperationAndReadPhase verifies read failures retain operation identity.
func TestSignAPIMalformedPDFErrorsIncludeOperationAndReadPhase(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "invalid.pdf")
	if err := os.WriteFile(inFile, []byte("not a PDF"), 0600); err != nil {
		t.Fatal(err)
	}

	_, validateErr := ValidateSignatures(inFile, false, nil)
	if validateErr == nil {
		t.Fatal("expected signature-validation failure")
	}
	if want := "validate signatures: prepare PDF context: read context"; !strings.Contains(validateErr.Error(), want) {
		t.Errorf("validate signatures: expected %q, got %q", want, validateErr)
	}

	_, validateFileErr := ValidateSignaturesFile(inFile, false, false, nil)
	if validateFileErr == nil {
		t.Fatal("expected signature-validation file failure")
	}
	if want := "validate signatures: prepare PDF context: read context"; !strings.Contains(validateFileErr.Error(), want) {
		t.Errorf("validate signatures file: expected %q, got %q", want, validateFileErr)
	}

	removeErr := RemoveSignatures(bytes.NewReader([]byte("not a PDF")), io.Discard, nil)
	if removeErr == nil {
		t.Fatal("expected signature-removal failure")
	}
	if want := "remove signatures: prepare PDF context: read context"; !strings.Contains(removeErr.Error(), want) {
		t.Errorf("remove signatures: expected %q, got %q", want, removeErr)
	}
	if strings.Contains(removeErr.Error(), "optimize:") {
		t.Errorf("remove signatures: leaked optimize operation context: %q", removeErr)
	}
}

// TestRemoveSignaturesWriteErrorPreservesCauseAndContext verifies output failures retain their operation phase.
func TestRemoveSignaturesWriteErrorPreservesCauseAndContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf")
	f, err := os.Open(inFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("close input: %v", err)
		}
	}()

	cause := errors.New("write failed")
	err = RemoveSignatures(f, signErrorWriter{err: cause}, nil)
	if !errors.Is(err, cause) {
		t.Fatalf("expected %v, got %v", cause, err)
	}
	for _, want := range []string{"remove signatures", "write output"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q, got %q", want, err)
		}
	}
}

// TestRemoveSignaturesFileCreateOutputErrorPreservesCauseAndContext verifies staging failures.
func TestRemoveSignaturesFileCreateOutputErrorPreservesCauseAndContext(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "input.pdf")
	if err := os.WriteFile(inFile, []byte("not a PDF"), 0600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(t.TempDir(), "missing", "out.pdf")

	err := RemoveSignaturesFile(inFile, outFile, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if want := "remove signatures: create output"; !strings.Contains(err.Error(), want) {
		t.Errorf("expected %q, got %q", want, err)
	}
}

// TestSignAPINoSignaturesPreservesSentinelAndOperation verifies semantic errors retain their identity.
func TestSignAPINoSignaturesPreservesSentinelAndOperation(t *testing.T) {
	inFile := filepath.Join("..", "samples", "bookmarks", "bookmarkSimple.pdf")
	tests := []struct {
		name string
		call func() error
		want string
	}{
		{
			name: "validate signatures",
			call: func() error {
				_, err := ValidateSignatures(inFile, false, nil)
				return err
			},
			want: "validate signatures",
		},
		{
			name: "remove signatures",
			call: func() error {
				return RemoveSignaturesFile(inFile, filepath.Join(t.TempDir(), "out.pdf"), nil)
			},
			want: "remove signatures",
		},
	}

	for _, tt := range tests {
		err := tt.call()
		if !errors.Is(err, ErrNoSignatures) {
			t.Errorf("%s: expected %v, got %v", tt.name, ErrNoSignatures, err)
			continue
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: expected %q, got %q", tt.name, tt.want, err)
		}
	}
}

// TestRemoveSignaturesFileFailurePreservesExistingOutput verifies staged publication.
func TestRemoveSignaturesFileFailurePreservesExistingOutput(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "invalid.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(inFile, []byte("not a PDF"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outFile, original, 0640); err != nil {
		t.Fatal(err)
	}

	err := RemoveSignaturesFile(inFile, outFile, nil)
	if err == nil {
		t.Fatal("expected signature-removal failure")
	}
	for _, want := range []string{"remove signatures", "read context"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q, got %q", want, err)
		}
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bb, original) {
		t.Fatalf("existing output changed: got %q, want %q", bb, original)
	}
}
