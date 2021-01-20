/*
	Copyright 2021 The pdfcpu Authors.

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
)

// PDFBooklet rearranges PDF pages or images into a booklet layout and writes the result to outFile.
func PDFBooklet(inFiles []string, outFile string, selectedPages []string, booklet *pdfcpu.Booklet, conf *pdfcpu.Configuration) (err error) {
	var f1, f2 *os.File

	// booklet from a PDF
	if f1, err = os.Open(inFiles[0]); err != nil {
		return err
	}

	if f2, err = os.Create(outFile); err != nil {
		return err
	}
	log.CLI.Printf("writing %s...\n", outFile)

	defer func() {
		if err != nil {
			if f1 != nil {
				f1.Close()
			}
			f2.Close()
			return
		}
		if f1 != nil {
			if err = f1.Close(); err != nil {
				return
			}
		}
		err = f2.Close()
		return

	}()

	return Booklet(f1, f2, selectedPages, booklet, conf)
}

// Booklet arranges PDF pages on larger sheets of paper and writes the result to w.
func Booklet(rs io.ReadSeeker, w io.Writer, selectedPages []string, booklet *pdfcpu.Booklet, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.BOOKLET

	log.Info.Printf("%s", booklet)

	// below is very similar to api.NUp
	var (
		ctx *pdfcpu.Context
		err error
	)

	if ctx, _, _, err = readAndValidate(rs, conf, time.Now()); err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	if err = ctx.BookletFromPDF(pages, booklet); err != nil {
		return err
	}

	if conf.ValidationMode != pdfcpu.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	return nil
}
