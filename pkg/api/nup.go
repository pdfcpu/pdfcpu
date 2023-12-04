/*
	Copyright 2020 The model Authors.

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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// PDFNUpConfig returns an NUp configuration for Nup-ing PDF files.
func PDFNUpConfig(val int, desc string, conf *model.Configuration) (*model.NUp, error) {
	return pdfcpu.PDFNUpConfig(val, desc, conf)
}

// ImageNUpConfig returns an NUp configuration for Nup-ing image files.
func ImageNUpConfig(val int, desc string, conf *model.Configuration) (*model.NUp, error) {
	return pdfcpu.ImageNUpConfig(val, desc, conf)
}

// PDFGridConfig returns a grid configuration for Grid-ing PDF files.
func PDFGridConfig(rows, cols int, desc string, conf *model.Configuration) (*model.NUp, error) {
	return pdfcpu.PDFGridConfig(rows, cols, desc, conf)
}

// ImageGridConfig returns a grid configuration for Grid-ing image files.
func ImageGridConfig(rows, cols int, desc string, conf *model.Configuration) (*model.NUp, error) {
	return pdfcpu.ImageGridConfig(rows, cols, desc, conf)
}

// PDFBookletConfig returns an NUp configuration for Booklet-ing PDF files.
func PDFBookletConfig(val int, desc string, conf *model.Configuration) (*model.NUp, error) {
	return pdfcpu.PDFBookletConfig(val, desc, conf)
}

// ImageBookletConfig returns an NUp configuration for Booklet-ing image files.
func ImageBookletConfig(val int, desc string, conf *model.Configuration) (*model.NUp, error) {
	return pdfcpu.ImageBookletConfig(val, desc, conf)
}

// NUpFromImage creates a single page n-up PDF for one image
// or a sequence of n-up pages for more than one image.
func NUpFromImage(conf *model.Configuration, imageFileNames []string, nup *model.NUp) (*model.Context, error) {
	if nup.PageDim == nil {
		// Set default paper size.
		nup.PageDim = types.PaperSize[nup.PageSize]
	}

	ctx, err := pdfcpu.CreateContextWithXRefTable(conf, nup.PageDim)
	if err != nil {
		return nil, err
	}

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		return nil, err
	}

	// This is the page tree root.
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		return nil, err
	}

	if len(imageFileNames) == 1 {
		err = pdfcpu.NUpFromOneImage(ctx, imageFileNames[0], nup, pagesDict, pagesIndRef)
	} else {
		err = pdfcpu.NUpFromMultipleImages(ctx, imageFileNames, nup, pagesDict, pagesIndRef)
	}

	return ctx, err
}

// NUp rearranges PDF pages or images into page grids and writes the result to w.
// Either rs or imgFiles will be used.
func NUp(rs io.ReadSeeker, w io.Writer, imgFiles, selectedPages []string, nup *model.NUp, conf *model.Configuration) error {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.NUP

	if log.InfoEnabled() {
		log.Info.Printf("%s", nup)
	}

	var (
		ctx *model.Context
		err error
	)

	if nup.ImgInputFile {

		if ctx, err = NUpFromImage(conf, imgFiles, nup); err != nil {
			return err
		}

	} else {

		if ctx, _, _, err = readAndValidate(rs, conf, time.Now()); err != nil {
			return err
		}

		if ctx.Version() == model.V20 {
			return pdfcpu.ErrUnsupportedVersion
		}

		if err := ctx.EnsurePageCount(); err != nil {
			return err
		}

		pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
		if err != nil {
			return err
		}

		// New pages get added to ctx while old pages get deleted.
		// This way we avoid migrating objects between contexts.
		if err = pdfcpu.NUpFromPDF(ctx, pages, nup); err != nil {
			return err
		}

	}

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	if log.StatsEnabled() {
		log.Stats.Printf("XRefTable:\n%s\n", ctx)
	}

	return nil
}

// NUpFile rearranges PDF pages or images into page grids and writes the result to outFile.
func NUpFile(inFiles []string, outFile string, selectedPages []string, nup *model.NUp, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if !nup.ImgInputFile {
		// Nup from a PDF page.
		if f1, err = os.Open(inFiles[0]); err != nil {
			return err
		}
	}

	if f2, err = os.Create(outFile); err != nil {
		if f1 != nil {
			f1.Close()
		}
		return err
	}
	logWritingTo(outFile)

	defer func() {
		if err != nil {
			f2.Close()
			if f1 != nil {
				f1.Close()
			}
			os.Remove(outFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if f1 != nil {
			err = f1.Close()
		}
	}()

	return NUp(f1, f2, inFiles, selectedPages, nup, conf)
}
