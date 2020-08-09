/*
	Copyright 2020 The pdfcpu Authors.

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

// NUp rearranges PDF pages or images into page grids and writes the result to w.
// Either rs or imgFiles will be used.
func NUp(rs io.ReadSeeker, w io.Writer, imgFiles, selectedPages []string, nup *pdfcpu.NUp, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.NUP

	log.Info.Printf("%s", nup)

	var (
		ctx *pdfcpu.Context
		err error
	)

	if nup.ImgInputFile {

		if ctx, err = pdfcpu.NUpFromImage(conf, imgFiles, nup); err != nil {
			return err
		}

	} else {

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

		// New pages get added to ctx while old pages get deleted.
		// This way we avoid migrating objects between contexts.
		if err = pdfcpu.NUpFromPDF(ctx, pages, nup); err != nil {
			return err
		}

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

// NUpFile rearranges PDF pages or images into page grids and writes the result to outFile.
func NUpFile(inFiles []string, outFile string, selectedPages []string, nup *pdfcpu.NUp, conf *pdfcpu.Configuration) (err error) {
	var f1, f2 *os.File

	if !nup.ImgInputFile {
		// Nup from a PDF page.
		if f1, err = os.Open(inFiles[0]); err != nil {
			return err
		}
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

	return NUp(f1, f2, inFiles, selectedPages, nup, conf)
}
