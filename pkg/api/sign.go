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
	"errors"
	"fmt"
	"io"
	"os"

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
	io.ReaderAt,
	*model.Context,
	bool,
) ([]*model.SignatureValidationResult, error)

func signatureStats(signValidResults []*model.SignatureValidationResult) model.SignatureStats {
	sigStats := model.SignatureStats{Total: len(signValidResults)}
	for _, svr := range signValidResults {
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
	return sigStats
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

func digest(signValidResults []*model.SignatureValidationResult, full bool) []string {
	var ss []string

	if full {
		ss = append(ss, "")
		for i, r := range signValidResults {
			//ss = append(ss, fmt.Sprintf("%d. Sisgnature:\n", i+1))
			ss = append(ss, fmt.Sprintf("%d:", i+1))
			ss = append(ss, r.String()+"\n")
		}
		return ss
	}

	if len(signValidResults) == 1 {
		svr := signValidResults[0]
		ss = append(ss, "")
		ss = append(ss, fmt.Sprintf("1 %s", svr.Signature.String(svr.Status)))
		ss = append(ss, fmt.Sprintf("   Status: %s", svr.Status))
		s := compactSignatureReason(svr)
		ss = append(ss, fmt.Sprintf("   Reason: %s", s))
		ss = append(ss, fmt.Sprintf("   Signed: %s", svr.SigningTime()))
		return ss
	}

	stats := signatureStats(signValidResults)

	ss = append(ss, "")
	ss = append(ss, fmt.Sprintf("%d signatures present:", stats.Total))

	statsCounter(stats, &ss)

	for i, svr := range signValidResults {
		ss = append(ss, fmt.Sprintf("\n%d:", i+1))
		ss = append(ss, fmt.Sprintf("     Type: %s", svr.Signature.String(svr.Status)))
		ss = append(ss, fmt.Sprintf("   Status: %s", svr.Status.String()))
		s := compactSignatureReason(svr)
		ss = append(ss, fmt.Sprintf("   Reason: %s", s))
		ss = append(ss, fmt.Sprintf("   Signed: %s", svr.SigningTime()))
	}

	return ss
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

// ValidateSignatures validates signature integrity, reports available trust evidence and performs a best-effort local
// assessment.
func ValidateSignatures(inFile string, all bool, conf *model.Configuration) (results []*model.SignatureValidationResult, err error) {
	defer fault.Catch(&err)
	return validateSignaturesFile(inFile, all, conf, pdfcpu.ValidateSignatures)
}

func validateSignaturesFile(
	inFile string,
	all bool,
	conf *model.Configuration,
	operation signatureValidationOperation,
) (results []*model.SignatureValidationResult, err error) {
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

	return validateSignaturesRaw(f, all, conf, operation)
}

// ValidateSignaturesRaw validates signature integrity, reports available trust evidence and performs a best-effort
// local assessment.
func ValidateSignaturesRaw(
	rs ReadSeekerAt,
	all bool,
	conf *model.Configuration,
) (results []*model.SignatureValidationResult, err error) {
	defer fault.Catch(&err)

	return validateSignaturesRaw(rs, all, conf, pdfcpu.ValidateSignatures)
}

func validateSignaturesRaw(
	rs ReadSeekerAt,
	all bool,
	conf *model.Configuration,
	operation signatureValidationOperation,
) (results []*model.SignatureValidationResult, err error) {
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.VALIDATESIGNATURES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("validate signatures: %w", err)
	}

	if len(ctx.Signatures) == 0 &&
		!ctx.SignatureExist &&
		!ctx.AppendOnly {
		return nil, fmt.Errorf("validate signatures: %w", ErrNoSignatures)
	}

	if err := pdfcpu.LoadCertificates(); err != nil {
		return nil, fmt.Errorf(
			"validate signatures: load trust pool: %w",
			err,
		)
	}

	results, err = operation(rs, ctx, all)
	if err != nil {
		return nil, fmt.Errorf(
			"validate signatures: verify signatures: %w",
			err,
		)
	}
	return results, nil
}

// ValidateSignaturesFile presents observed signature, certificate, timestamp
// and revocation evidence together with a local assessment.
// all: processes all signatures meaning not only the authoritative/certified signature..
// full: detailed output including certificate paths, observed evidence and problems encountered.
func ValidateSignaturesFile(inFile string, all, full bool, conf *model.Configuration) ([]string, error) {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}

	signValidResults, err := ValidateSignatures(inFile, all, conf)
	if err != nil {
		return nil, err
	}

	return digest(signValidResults, full), nil
}

// RemoveSignatures removes all digital signatures from rs and writes to w.
func RemoveSignatures(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVESIGNATURES

	if err := optimize(rs, w, conf); err != nil {
		return fmt.Errorf("remove signatures: %w", err)
	}
	return nil
}

// RemoveSignaturesFile removes all digital signatures from inFile and writes to outFile if provided else overwrites
// inFile.
func RemoveSignaturesFile(inFile, outFile string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("remove signatures: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "remove signatures")
	if err != nil {
		return errors.Join(
			fmt.Errorf("remove signatures: create output: %w", err),
			closeFile(f1, "remove signatures: close input"),
		)
	}
	f2 = staged.output.file

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = RemoveSignatures(f1, f2, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
