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
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

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
		s := svr.Reason.String()
		if svr.Reason == model.SignatureReasonInternal {
			if len(svr.Problems) > 0 {
				s = svr.Problems[0]
			}
		}
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
		s := svr.Reason.String()
		if svr.Reason == model.SignatureReasonInternal {
			if len(svr.Problems) > 0 {
				s = svr.Problems[0]
			}
		}
		ss = append(ss, fmt.Sprintf("   Reason: %s", s))
		ss = append(ss, fmt.Sprintf("   Signed: %s", svr.SigningTime()))
	}

	return ss
}

// ValidateSignatures validates signatures of inFile and returns the signature validation results.
func ValidateSignatures(inFile string, all bool, conf *model.Configuration) ([]*model.SignatureValidationResult, error) {

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.VALIDATESIGNATURE

	if _, err := LoadCertificates(); err != nil {
		return nil, err
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}

	ctx, err := ReadValidateAndOptimize(f, conf)
	if err != nil {
		return nil, err
	}

	if len(ctx.Signatures) == 0 && !ctx.SignatureExist && !ctx.AppendOnly {
		return nil, errors.New("pdfcpu: No signatures present.")
	}

	return pdfcpu.ValidateSignatures(f, ctx, all)
}

// ValidateSignaturesFile validates signatures of inFile.
// all: processes all signatures meaning not only the authoritative/certified signature..
// full: verbose output including cert chain and problems encountered.
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
