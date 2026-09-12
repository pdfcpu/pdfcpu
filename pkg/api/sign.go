/*
Copyright 2025 The pdf Authors.

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
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ReadSeekerAt supports PDF parsing and positional signature verification.
type ReadSeekerAt interface {
	io.ReadSeeker
	io.ReaderAt
}

type signatureValidationOperation func(
	context.Context,
	io.ReaderAt,
	*model.Context,
	bool,
	*x509.CertPool,
) ([]*model.SignatureValidationResult, error)

type contextReadSeekerAt struct {
	context.Context
	ReadSeekerAt
}

func (r contextReadSeekerAt) Read(p []byte) (int, error) {
	if err := r.Err(); err != nil {
		return 0, err
	}
	n, err := r.ReadSeekerAt.Read(p)
	if contextErr := r.Err(); contextErr != nil {
		return n, contextErr
	}
	return n, err
}

func (r contextReadSeekerAt) ReadAt(p []byte, off int64) (int, error) {
	if err := r.Err(); err != nil {
		return 0, err
	}
	n, err := r.ReadSeekerAt.ReadAt(p, off)
	if contextErr := r.Err(); contextErr != nil {
		return n, contextErr
	}
	return n, err
}

func (r contextReadSeekerAt) Seek(offset int64, whence int) (int64, error) {
	if err := r.Err(); err != nil {
		return 0, err
	}
	off, err := r.ReadSeekerAt.Seek(offset, whence)
	if contextErr := r.Err(); contextErr != nil {
		return off, contextErr
	}
	return off, err
}

func signatureStats(c context.Context, signValidResults []*model.SignatureValidationResult) (model.SignatureStats, error) {
	sigStats := model.SignatureStats{Total: len(signValidResults)}
	for _, svr := range signValidResults {
		if err := contextutil.Check(c); err != nil {
			return model.SignatureStats{}, err
		}
		signed, signedVisible, unsigned, unsignedVisible := sigStats.Counter(svr)
		if svr.Signed {
			*signed++
			if svr.Visible {
				*signedVisible++
			}
			continue
		}
		*unsigned++
		if svr.Visible {
			*unsignedVisible++
		}
	}
	return sigStats, contextutil.Check(c)
}

func statsCounter(stats model.SignatureStats, ss *[]string) {
	plural := func(count int) string {
		if count == 1 {
			return ""
		}
		return "s"
	}

	if stats.FormSigned > 0 {
		*ss = append(*ss, fmt.Sprintf("%d signed form signature%s (%d visible)", stats.FormSigned, plural(stats.FormSigned), stats.FormSignedVisible))
	}
	if stats.FormUnsigned > 0 {
		*ss = append(*ss, fmt.Sprintf("%d unsigned form signature%s (%d visible)", stats.FormUnsigned, plural(stats.FormUnsigned), stats.FormUnsignedVisible))
	}

	if stats.PageSigned > 0 {
		*ss = append(*ss, fmt.Sprintf("%d signed page signature%s (%d visible)", stats.PageSigned, plural(stats.PageSigned), stats.PageSignedVisible))
	}
	if stats.PageUnsigned > 0 {
		*ss = append(*ss, fmt.Sprintf("%d unsigned page signature%s (%d visible)", stats.PageUnsigned, plural(stats.PageUnsigned), stats.PageUnsignedVisible))
	}

	if stats.URSigned > 0 {
		*ss = append(*ss, fmt.Sprintf("%d signed usage rights signature%s (%d visible)", stats.URSigned, plural(stats.URSigned), stats.URSignedVisible))
	}
	if stats.URUnsigned > 0 {
		*ss = append(*ss, fmt.Sprintf("%d unsigned usage rights signature%s (%d visible)", stats.URUnsigned, plural(stats.URUnsigned), stats.URUnsignedVisible))
	}

	if stats.DTSSigned > 0 {
		*ss = append(*ss, fmt.Sprintf("%d signed doc timestamp signature%s (%d visible)", stats.DTSSigned, plural(stats.DTSSigned), stats.DTSSignedVisible))
	}
	if stats.DTSUnsigned > 0 {
		*ss = append(*ss, fmt.Sprintf("%d unsigned doc timestamp signature%s (%d visible)", stats.DTSUnsigned, plural(stats.DTSUnsigned), stats.DTSUnsignedVisible))
	}
}

func digest(c context.Context, signValidResults []*model.SignatureValidationResult, full bool) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	var ss []string

	if full {
		ss = append(ss, "")
		for i, r := range signValidResults {
			if err := contextutil.Check(c); err != nil {
				return nil, err
			}
			//ss = append(ss, fmt.Sprintf("%d. Sisgnature:\n", i+1))
			ss = append(ss, fmt.Sprintf("%d:", i+1))
			ss = append(ss, r.String()+"\n")
		}
		return ss, contextutil.Check(c)
	}

	if len(signValidResults) == 1 {
		svr := signValidResults[0]
		ss = append(ss, "")
		ss = append(ss, fmt.Sprintf("1 %s", svr.Signature.String(svr.Status)))
		ss = append(ss, fmt.Sprintf("   Status: %s", svr.Status))
		s := compactSignatureReason(svr)
		ss = append(ss, fmt.Sprintf("   Reason: %s", s))
		ss = append(ss, fmt.Sprintf("   Signed: %s", svr.SigningTime()))
		return ss, contextutil.Check(c)
	}

	stats, err := signatureStats(c, signValidResults)
	if err != nil {
		return nil, err
	}

	ss = append(ss, "")
	ss = append(ss, fmt.Sprintf("%d signatures present:", stats.Total))

	statsCounter(stats, &ss)

	for i, svr := range signValidResults {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		ss = append(ss, fmt.Sprintf("\n%d:", i+1))
		ss = append(ss, fmt.Sprintf("     Type: %s", svr.Signature.String(svr.Status)))
		ss = append(ss, fmt.Sprintf("   Status: %s", svr.Status.String()))
		s := compactSignatureReason(svr)
		ss = append(ss, fmt.Sprintf("   Reason: %s", s))
		ss = append(ss, fmt.Sprintf("   Signed: %s", svr.SigningTime()))
	}

	return ss, contextutil.Check(c)
}

func compactSignatureReason(svr *model.SignatureValidationResult) string {
	s := svr.Reason.String()
	switch svr.Reason {
	case model.SignatureReasonInternal,
		model.SignatureReasonMalformed,
		model.SignatureReasonUnsupported:
		if len(svr.Problems) > 0 {
			s = svr.Problems[0]
		}
	}
	return s
}

// ValidateSignatures validates signature integrity, reports available trust evidence, performs a best-effort local
// assessment and supports cancellation.
func ValidateSignatures(c context.Context, inFile string, all bool, conf *model.Configuration) (results []*model.SignatureValidationResult, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if inFile == "" {
		return nil, ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf(
			"validate signatures: open input %s: %w",
			inFile,
			err,
		)
	}
	defer func() {
		err = errors.Join(
			err,
			closeFile(f, "validate signatures: close input"),
		)
	}()

	return ValidateSignaturesRaw(c, f, all, conf)
}

// ValidateSignaturesRaw validates signature integrity, reports available trust evidence, performs a best-effort local
// assessment and supports cancellation.
func ValidateSignaturesRaw(c context.Context, rs ReadSeekerAt, all bool, conf *model.Configuration) (results []*model.SignatureValidationResult, err error) {
	defer fault.Catch(&err)
	return validateSignaturesRawUsing(
		c, rs, all, conf, pdfcpu.ValidateSignaturesWithCertificatePool,
	)
}

func validateSignaturesRawUsing(c context.Context, rs ReadSeekerAt, all bool, conf *model.Configuration, operation signatureValidationOperation) (results []*model.SignatureValidationResult, err error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.VALIDATESIGNATURES)
	contextReader := contextReadSeekerAt{Context: c, ReadSeekerAt: rs}

	ctx, err := ReadValidateAndOptimize(c, contextReader, conf, nil)
	if err != nil {
		return nil, fmt.Errorf("validate signatures: %w", err)
	}

	if len(ctx.Signatures) == 0 &&
		!ctx.SignatureExist &&
		!ctx.AppendOnly {
		return nil, fmt.Errorf("validate signatures: %w", ErrNoSignatures)
	}

	certPool, err := pdfcpu.CertificatePoolForConfiguration(c, conf)
	if err != nil {
		return nil, fmt.Errorf(
			"validate signatures: load trust pool: %w",
			err,
		)
	}
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}

	results, err = operation(c, contextReader, ctx, all, certPool)
	if err != nil {
		return nil, fmt.Errorf(
			"validate signatures: verify signatures: %w",
			err,
		)
	}
	return results, contextutil.Check(c)
}

// ValidateSignaturesFile presents observed signature, certificate, timestamp and revocation evidence together with a
// local assessment and supports cancellation.
// all: processes all signatures meaning not only the authoritative/certified signature..
// full: detailed output including certificate paths, observed evidence and problems encountered.
func ValidateSignaturesFile(c context.Context, inFile string, all, full bool, conf *model.Configuration) ([]string, error) {
	signValidResults, err := ValidateSignatures(c, inFile, all, conf)
	if err != nil {
		return nil, err
	}
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	return digest(c, signValidResults, full)
}

// RemoveSignatures removes all digital signatures from rs, writes to w and supports cancellation.
func RemoveSignatures(c context.Context, rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if w == nil {
		return ErrMissingPDFWriter
	}

	conf = operationConfiguration(conf, model.REMOVESIGNATURES)

	if err := optimize(c, rs, w, conf, ProgressOptions{}); err != nil {
		return fmt.Errorf("remove signatures: %w", err)
	}
	return nil
}

// RemoveSignaturesFile removes all digital signatures from inFile, writes to outFile if provided, otherwise overwrites
// inFile, and supports cancellation.
func RemoveSignaturesFile(c context.Context, inFile, outFile string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	f1, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("remove signatures: open input %s: %w", inFile, err)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "remove signatures")
	if err != nil {
		return errors.Join(
			fmt.Errorf("remove signatures: create output: %w", err),
			closeFile(f1, "remove signatures: close input"),
		)
	}
	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = RemoveSignatures(c, f1, staged.output.file, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}
