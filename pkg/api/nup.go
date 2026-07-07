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
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validNUpDimension(v float64) bool {
	return v > 0 && !math.IsInf(v, 0) && !math.IsNaN(v)
}

func validateNUpGrid(nup *model.NUp) error {
	if nup.Grid == nil {
		return errors.New("invalid configuration: missing page grid")
	}
	cellCount := nup.Grid.Width * nup.Grid.Height
	maxInt := float64(int(^uint(0) >> 1))
	if !validNUpDimension(nup.Grid.Width) || !validNUpDimension(nup.Grid.Height) || !validNUpDimension(cellCount) ||
		cellCount > maxInt ||
		math.Trunc(nup.Grid.Width) != nup.Grid.Width || math.Trunc(nup.Grid.Height) != nup.Grid.Height {
		return errors.New("invalid configuration: invalid page grid")
	}
	return nil
}

func resolveNUpPageDimension(nup *model.NUp) error {
	if nup.PageDim == nil {
		dim := types.PaperSize[nup.PageSize]
		if dim == nil {
			return fmt.Errorf("invalid configuration: unknown page size %q", nup.PageSize)
		}
		pageDim := *dim
		nup.PageDim = &pageDim
	}
	if !validNUpDimension(nup.PageDim.Width) || !validNUpDimension(nup.PageDim.Height) {
		return errors.New("invalid configuration: invalid page dimensions")
	}
	return nil
}

func prepareNUpConfiguration(nup *model.NUp, imageInput bool) error {
	if err := validateNUpGrid(nup); err != nil {
		return err
	}
	if nup.PageDim == nil && !imageInput {
		return nil
	}
	return resolveNUpPageDimension(nup)
}

func prepareNUpConfigurationForAPI(nup *model.NUp, imageInput bool) error {
	if err := prepareNUpConfiguration(nup, imageInput); err != nil {
		return fmt.Errorf("n-up: prepare configuration: %w", err)
	}
	return nil
}

// NUpValuesForBooklets returns the supported booklet page counts per sheet.
func NUpValuesForBooklets() []int {
	return pdfcpu.NUpValuesForBooklets()
}

// NUpValues returns the supported n-up page counts per sheet.
func NUpValues() []int {
	return append([]int(nil), pdfcpu.NUpValues...)
}

// DefaultBookletConfig returns the default configuration for a booklet.
func DefaultBookletConfig() *model.NUp {
	return pdfcpu.DefaultBookletConfig()
}

// PDFNUpConfig returns an NUp configuration for Nup-ing PDF files.
func PDFNUpConfig(val int, desc string, conf *model.Configuration) (*model.NUp, error) {
	return pdfcpu.PDFNUpConfig(val, desc, conf)
}

// ImageNUpConfig returns an NUp configuration for Nup-ing image files.
func ImageNUpConfig(val int, desc string, conf *model.Configuration) (*model.NUp, error) {
	return pdfcpu.ImageNUpConfig(val, desc, conf)
}

// ParseNUpDetails parses an n-up command string into nup.
func ParseNUpDetails(s string, nup *model.NUp) error {
	if nup == nil {
		return ErrMissingNUpConfiguration
	}
	return pdfcpu.ParseNUpDetails(s, nup)
}

// ParseNUpValue applies an n-up page count to nup.
func ParseNUpValue(n int, nup *model.NUp) error {
	if nup == nil {
		return ErrMissingNUpConfiguration
	}
	return pdfcpu.ParseNUpValue(n, nup)
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
// On error, the returned context may be partially constructed and its PageCount remains at the pre-operation value.
// Callers must discard a non-nil context returned together with an error.
func NUpFromImage(conf *model.Configuration, imageFileNames []string, nup *model.NUp) (ctx *model.Context, err error) {
	defer fault.Catch(&err)

	if nup == nil {
		return nil, ErrMissingNUpConfiguration
	}
	if len(imageFileNames) == 0 {
		return nil, ErrMissingImageInput
	}
	if err := prepareNUpConfigurationForAPI(nup, true); err != nil {
		return nil, err
	}
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.NUP

	ctx, err = pdfcpu.CreateContextWithXRefTable(conf, nup.PageDim)
	if err != nil {
		return nil, fmt.Errorf("n-up: create image context: %w", err)
	}

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		return nil, fmt.Errorf("n-up: access image page tree: %w", err)
	}

	// This is the page tree root.
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		return nil, fmt.Errorf("n-up: dereference image page tree: %w", err)
	}

	if len(imageFileNames) == 1 {
		err = pdfcpu.NUpFromOneImage(ctx, imageFileNames[0], nup, pagesDict, pagesIndRef)
	} else {
		err = pdfcpu.NUpFromMultipleImages(ctx, imageFileNames, nup, pagesDict, pagesIndRef)
	}

	if err != nil {
		return ctx, fmt.Errorf("n-up: impose images: %w", err)
	}
	return ctx, nil
}

// NUp rearranges PDF pages or images into page grids and writes the result to w.
// Either rs or imgFiles will be used.
func NUp(rs io.ReadSeeker, w io.Writer, imgFiles, selectedPages []string, nup *model.NUp, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if w == nil {
		return ErrMissingPDFWriter
	}

	if nup == nil {
		return ErrMissingNUpConfiguration
	}
	if nup.ImgInputFile && len(imgFiles) == 0 {
		return ErrMissingImageInput
	}
	if err := prepareNUpConfigurationForAPI(nup, nup.ImgInputFile); err != nil {
		return err
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.NUP

	if log.InfoEnabled() {
		log.Info.Printf("%s", nup)
	}

	var ctx *model.Context

	if nup.ImgInputFile {

		if ctx, err = NUpFromImage(conf, imgFiles, nup); err != nil {
			return err
		}

	} else {

		if rs == nil {
			return ErrMissingPDFReadSeeker
		}

		if ctx, err = ReadAndValidate(rs, conf); err != nil {
			return fmt.Errorf("n-up: read and validate: %w", err)
		}

		pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
		if err != nil {
			return fmt.Errorf("n-up: parse page selection: %w", err)
		}

		// New pages get added to ctx while old pages get deleted.
		// This way we avoid migrating objects between contexts.
		if err = pdfcpu.NUpFromPDF(ctx, pages, nup); err != nil {
			return fmt.Errorf("n-up: impose pages: %w", err)
		}

	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("n-up: write output: %w", err)
	}
	return nil
}

func nUpImageOutputAliasesInput(inFile, outFile string) (bool, error) {
	return outputAliasesInput(inFile, outFile)
}

func rejectNUpImageOutputAlias(inFiles []string, outFile string) error {
	for i, inFile := range inFiles {
		aliases, err := nUpImageOutputAliasesInput(inFile, outFile)
		if err != nil {
			return fmt.Errorf("n-up image %d %q: check output alias: %w", i+1, inFile, err)
		}
		if aliases {
			return fmt.Errorf("n-up image %d %q: output aliases input: %w", i+1, inFile, ErrNUpImageOutputConflict)
		}
	}
	return nil
}

// NUpFile rearranges PDF pages or images into page grids and writes the result to outFile.
func NUpFile(inFiles []string, outFile string, selectedPages []string, nup *model.NUp, conf *model.Configuration) (err error) {
	if nup == nil {
		return ErrMissingNUpConfiguration
	}
	if len(inFiles) == 0 {
		if nup.ImgInputFile {
			return ErrMissingImageInput
		}
		return ErrMissingPDFInput
	}
	if outFile == "" {
		return ErrMissingPDFOutput
	}
	if err := prepareNUpConfigurationForAPI(nup, nup.ImgInputFile); err != nil {
		return err
	}
	if nup.ImgInputFile {
		if err := rejectNUpImageOutputAlias(inFiles, outFile); err != nil {
			return err
		}
	}

	var f1, f2 *os.File
	ok := false

	if !nup.ImgInputFile {
		// Nup from a PDF page.
		if f1, err = os.Open(inFiles[0]); err != nil {
			return fmt.Errorf("n-up: open input %s: %w", inFiles[0], err)
		}
	}

	staged, err := openStagedOutput(f1, inFiles[0], outFile, "n-up")
	if err != nil {
		return errors.Join(
			fmt.Errorf("n-up: create output: %w", err),
			closeFile(f1, "n-up: close input"),
		)
	}
	f2 = staged.output.file
	logWritingTo(outFile)

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = NUp(f1, f2, inFiles, selectedPages, nup, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
