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
)

// CreatePDFFile creates a PDF file for an xRefTable and writes it to outFile.
func CreatePDFFile(xRefTable *pdfcpu.XRefTable, outFile string, conf *pdfcpu.Configuration) error {
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()
	ctx := pdfcpu.CreateContext(xRefTable, conf)
	return WriteContext(ctx, f)
}

// CreateFromJSONFile renders the PDF structure corresponding to rd to w.
// If rs == nil a new PDF file will be written to w.
// If infile exists it appends page content for existing pages and
// appends new pages including any empty pages needed in between.
func CreateFromJSON(rd io.Reader, rs io.ReadSeeker, w io.Writer, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.CREATE

	var (
		ctx *pdfcpu.Context
		err error
	)

	if rs != nil {
		ctx, _, _, _, err = readValidateAndOptimize(rs, conf, time.Now())
	} else {
		ctx, err = pdfcpu.CreateContextWithXRefTable(conf, pdfcpu.PaperSize["A4"])
	}
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	if err := create.FromJSON(rd, ctx); err != nil {
		return err
	}

	if conf.ValidationMode != pdfcpu.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// CreateFromJSONFile renders the PDF structure corresponding to jsonFile to outFile.
// If inFile does not exist it creates a new PDF Context.
// If infile exists it appends page content for existing pages and
// appends new pages including any empty pages needed.
func CreateFromJSONFile(jsonFile, inFile, outFile string, conf *pdfcpu.Configuration) (err error) {

	var f0, f1, f2 *os.File

	if f0, err = os.Open(jsonFile); err != nil {
		return err
	}

	rs := io.ReadSeeker(nil)
	f1 = nil
	if fileExists(inFile) {
		if f1, err = os.Open(inFile); err != nil {
			return err
		}
		rs = f1
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
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
			os.Remove(tmpFile)
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
		s := outFile
		if outFile == "" || inFile == outFile {
			s = inFile
		}
		if err = os.Rename(tmpFile, s); err != nil {
			return
		}
	}()

	return CreateFromJSON(f0, rs, f2, conf)
}
