/*
Copyright 2025 The pdfcpu Authors.

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
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// IsPDFA returns true if ctx represents a PDF/A document.
// This checks if the document claims to be PDF/A via metadata or OutputIntent.
func IsPDFA(ctx *model.Context) bool {
	if ctx == nil || ctx.XRefTable == nil {
		return false
	}
	return ctx.XRefTable.PDFA != nil && ctx.XRefTable.PDFA.ClaimsPDFA
}

// PDFAInfo returns PDF/A identification information from ctx.
// Returns nil if the document does not claim to be PDF/A.
//
// Example:
//   info := PDFAInfo(ctx)
//   if info != nil && info.ClaimsPDFA {
//       fmt.Printf("PDF/A-%d, Level %s\n", *info.Part, *info.Conformance)
//   }
func PDFAInfo(ctx *model.Context) *model.PDFAInfo {
	if ctx == nil || ctx.XRefTable == nil {
		return nil
	}
	return ctx.XRefTable.PDFA
}

// PDFAInfoFromRS returns PDF/A identification information from a ReadSeeker.
func PDFAInfoFromRS(rs io.ReadSeeker, conf *model.Configuration) (*model.PDFAInfo, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: PDFAInfoFromRS: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTINFO

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, err
	}

	return PDFAInfo(ctx), nil
}

// IsPDFAFromRS returns true if rs represents a PDF/A document.
func IsPDFAFromRS(rs io.ReadSeeker, conf *model.Configuration) (bool, error) {
	info, err := PDFAInfoFromRS(rs, conf)
	if err != nil {
		return false, err
	}
	return info != nil && info.ClaimsPDFA, nil
}

// PDFAInfoFile returns PDF/A identification information from a PDF file.
//
// Example:
//   info, err := PDFAInfoFile("document.pdf", nil)
//   if err != nil {
//       log.Fatal(err)
//   }
//   if info != nil && info.ClaimsPDFA {
//       fmt.Printf("PDF/A-%d%s\n", *info.Part, *info.Conformance)
//   }
func PDFAInfoFile(inFile string, conf *model.Configuration) (*model.PDFAInfo, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return PDFAInfoFromRS(f, conf)
}

// IsPDFAFile returns true if inFile is a PDF/A document.
//
// Example:
//   isPDFA, err := IsPDFAFile("document.pdf", nil)
//   if err != nil {
//       log.Fatal(err)
//   }
//   if isPDFA {
//       fmt.Println("This is a PDF/A document")
//   }
func IsPDFAFile(inFile string, conf *model.Configuration) (bool, error) {
	info, err := PDFAInfoFile(inFile, conf)
	if err != nil {
		return false, err
	}
	return info != nil && info.ClaimsPDFA, nil
}
