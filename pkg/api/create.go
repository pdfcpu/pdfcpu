/*
	Copyright 2019 The pdfcpu Authors.

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
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/create"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// CreatePDFFile creates a PDF file for an xRefTable and writes it to outFile.
func CreatePDFFile(xRefTable *model.XRefTable, outFile string, conf *model.Configuration) error {
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()
	ctx := pdfcpu.CreateContext(xRefTable, conf)
	return WriteContext(ctx, f)
}

// Create renders the PDF structure represented by rs into w.
// If rs is present, new PDF content will be appended including any empty pages needed.
// rd is a JSON representation of PDF page content which may include form data.
func Create(rs io.ReadSeeker, rd io.Reader, w io.Writer, conf *model.Configuration) error {
	if rd == nil {
		return errors.New("pdfcpu: Create: missing rd")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CREATE

	var (
		ctx *model.Context
		err error
	)

	if rs != nil {
		ctx, _, _, _, err = ReadValidateAndOptimize(rs, conf, time.Now())
	} else {
		ctx, err = pdfcpu.CreateContextWithXRefTable(conf, types.PaperSize["A4"])
	}
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	if err := create.FromJSON(ctx, rd); err != nil {
		return err
	}

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

func handleOutFilePDF(inFilePDF, outFilePDF string, tmpFile *string) {
	if outFilePDF != "" && inFilePDF != outFilePDF {
		*tmpFile = outFilePDF
		log.CLI.Printf("writing %s...\n", outFilePDF)
	} else {
		log.CLI.Printf("writing %s...\n", inFilePDF)
	}
}

// CreateFile renders the PDF structure represented by inFileJSON into outFilePDF.
// If inFilePDF is present, new PDF content will be appended including any empty pages needed.
// inFileJSON represents PDF page content which may include form data.
func CreateFile(inFilePDF, inFileJSON, outFilePDF string, conf *model.Configuration) (err error) {

	var f0, f1, f2 *os.File

	if f0, err = os.Open(inFileJSON); err != nil {
		return err
	}

	rs := io.ReadSeeker(nil)
	f1 = nil
	if fileExists(inFilePDF) {
		if f1, err = os.Open(inFilePDF); err != nil {
			return err
		}
		log.CLI.Printf("reading %s...\n", inFilePDF)
		rs = f1
	}

	tmpFile := inFilePDF + ".tmp"
	handleOutFilePDF(inFilePDF, outFilePDF, &tmpFile)

	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			if f1 != nil {
				f1.Close()
			}
			f0.Close()
			if outFilePDF == "" || inFilePDF == outFilePDF {
				os.Remove(tmpFile)
			}
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if f1 != nil {
			if err = f1.Close(); err != nil {
				return
			}
		}
		if err = f0.Close(); err != nil {
			return
		}
		if outFilePDF == "" || inFilePDF == outFilePDF {
			err = os.Rename(tmpFile, inFilePDF)
		}
	}()

	return Create(rs, f0, f2, conf)
}
