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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/create"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// CreatePDFFile creates a PDF file for an xRefTable and writes it to outFile.
func CreatePDFFile(xRefTable *model.XRefTable, outFile string, conf *model.Configuration) error {
	if xRefTable == nil {
		return ErrMissingXRefTable
	}

	staged, err := openStagedOutput(nil, "", outFile, "create")
	if err != nil {
		return fmt.Errorf("create: create output %s: %w", outFile, err)
	}
	f := staged.output.file
	ctx := pdfcpu.CreateContext(xRefTable, conf)
	if err := WriteContext(ctx, f); err != nil {
		return staged.cleanup(fmt.Errorf("create: write output: %w", err))
	}
	return staged.commit()
}

// Create renders the PDF structure represented by rs into w.
// If rs is present, new PDF content will be appended including any empty pages needed.
// rd is a JSON representation of PDF page content which may include form data.
func Create(rs io.ReadSeeker, rd io.Reader, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rd == nil {
		return ErrMissingJSONInput
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CREATE

	var ctx *model.Context

	if rs != nil {
		ctx, err = ReadValidateAndOptimize(rs, conf)
		if err != nil {
			return fmt.Errorf("create: %w", err)
		}
	} else {
		ctx, err = pdfcpu.CreateContextWithXRefTable(conf, types.PaperSize["A4"])
		if err != nil {
			return fmt.Errorf("create: create PDF context: %w", err)
		}
	}

	if err := create.FromJSON(ctx, rd); err != nil {
		return fmt.Errorf("create: import JSON: %w", err)
	}

	if conf.PostProcessValidate {
		if err = ValidateContext(ctx); err != nil {
			return fmt.Errorf("create: validate output: %w", err)
		}
	}

	if err = WriteContext(ctx, w); err != nil {
		return fmt.Errorf("create: write output: %w", err)
	}
	return nil
}

func handleOutFilePDF(inFilePDF, outFilePDF string, tmpFile *string) {
	if outFilePDF != "" && inFilePDF != outFilePDF {
		*tmpFile = outFilePDF
		logWritingTo(outFilePDF)
	} else {
		logWritingTo(inFilePDF)
	}
}

// CreateFile renders the PDF structure represented by inFileJSON into outFilePDF.
// If inFilePDF is present, new PDF content will be appended including any empty pages needed.
// inFileJSON represents PDF page content which may include form data.
func CreateFile(inFilePDF, inFileJSON, outFilePDF string, conf *model.Configuration) (err error) {
	var f0, f1, f2 *os.File
	ok := false

	if inFileJSON == "" {
		return ErrMissingJSONInput
	}

	if inFilePDF == "" && outFilePDF == "" {
		return fmt.Errorf("create: missing input or output: %w", ErrMissingPDFInput)
	}

	if f0, err = os.Open(inFileJSON); err != nil {
		return fmt.Errorf("create: open JSON input %s: %w", inFileJSON, err)
	}

	rs := io.ReadSeeker(nil)
	f1 = nil
	if inFilePDF != "" {
		f1, err = os.Open(inFilePDF)
		if err != nil {
			return errors.Join(
				fmt.Errorf("create: open input %s: %w", inFilePDF, err),
				closeFile(f0, "create: close JSON input"),
			)
		}
		if log.CLIEnabled() {
			log.CLI.Printf("reading %s...\n", inFilePDF)
		}
		rs = f1
	}

	tmpFile := ""
	handleOutFilePDF(inFilePDF, outFilePDF, &tmpFile)

	staged, err := openStagedOutput(f1, inFilePDF, tmpFile, "create")
	if err != nil {
		return errors.Join(
			fmt.Errorf("create: create output: %w", err),
			closeFile(f1, "create: close input"),
			closeFile(f0, "create: close JSON input"),
		)
	}
	f2 = staged.output.file
	staged = staged.withInput(f0, "create: close JSON input")

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = Create(rs, f0, f2, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
