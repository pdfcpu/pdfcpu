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

func validBookletDimension(v float64) bool {
	return v > 0 && !math.IsInf(v, 0) && !math.IsNaN(v)
}

func validateBookletGrid(nup *model.NUp) error {
	if nup.Grid == nil {
		return errors.New("invalid configuration: missing page grid")
	}
	if !validBookletDimension(nup.Grid.Width) || !validBookletDimension(nup.Grid.Height) ||
		math.Trunc(nup.Grid.Width) != nup.Grid.Width || math.Trunc(nup.Grid.Height) != nup.Grid.Height {
		return errors.New("invalid configuration: invalid page grid")
	}
	return nil
}

func resolveBookletPageDimension(nup *model.NUp) error {
	if nup.PageDim == nil {
		dim := types.PaperSize[nup.PageSize]
		if dim == nil {
			return fmt.Errorf("invalid configuration: unknown page size %q", nup.PageSize)
		}
		pageDim := *dim
		nup.PageDim = &pageDim
	}
	if !validBookletDimension(nup.PageDim.Width) || !validBookletDimension(nup.PageDim.Height) {
		return errors.New("invalid configuration: invalid page dimensions")
	}
	return nil
}

func validateBookletLayout(nup *model.NUp) error {
	n := nup.N()
	if n != 2 && n != 4 && n != 6 && n != 8 {
		return fmt.Errorf("invalid configuration: unsupported pages per sheet: %d", n)
	}
	if nup.BookletType < model.Booklet || nup.BookletType > model.BookletPerfectBound {
		return errors.New("invalid configuration: unsupported booklet type")
	}
	if nup.MultiFolio && nup.FolioSize <= 0 {
		return errors.New("invalid configuration: folio size must be positive")
	}
	return nil
}

func prepareBookletConfiguration(nup *model.NUp) error {
	if err := validateBookletGrid(nup); err != nil {
		return err
	}
	if err := resolveBookletPageDimension(nup); err != nil {
		return err
	}
	return validateBookletLayout(nup)
}

func wrapBookletConfigurationError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("booklet: prepare configuration: %w", err)
}

func prepareBookletConfigurationForAPI(nup *model.NUp) error {
	return wrapBookletConfigurationError(prepareBookletConfiguration(nup))
}

// BookletFromImages creates a booklet from images.
func BookletFromImages(conf *model.Configuration, imageFileNames []string, nup *model.NUp) (ctx *model.Context, err error) {
	defer fault.Catch(&err)

	if nup == nil {
		return nil, ErrMissingBookletConfiguration
	}
	if len(imageFileNames) == 0 {
		return nil, ErrMissingImageInput
	}
	if err := prepareBookletConfigurationForAPI(nup); err != nil {
		return nil, err
	}
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.BOOKLET

	ctx, err = pdfcpu.CreateContextWithXRefTable(conf, nup.PageDim)
	if err != nil {
		return nil, fmt.Errorf("booklet: create image context: %w", err)
	}

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		return nil, fmt.Errorf("booklet: access image page tree: %w", err)
	}

	// This is the page tree root.
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		return nil, fmt.Errorf("booklet: dereference image page tree: %w", err)
	}

	if err = pdfcpu.BookletFromImages(ctx, imageFileNames, nup, pagesDict, pagesIndRef); err != nil {
		return ctx, fmt.Errorf("booklet: impose images: %w", err)
	}
	return ctx, nil
}

// Booklet arranges PDF pages on larger sheets of paper and writes the result to w.
func Booklet(rs io.ReadSeeker, w io.Writer, imgFiles, selectedPages []string, nup *model.NUp, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if w == nil {
		return ErrMissingPDFWriter
	}

	if nup == nil {
		return ErrMissingBookletConfiguration
	}
	if nup.ImgInputFile && len(imgFiles) == 0 {
		return ErrMissingImageInput
	}
	if err := prepareBookletConfigurationForAPI(nup); err != nil {
		return err
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.BOOKLET

	if log.InfoEnabled() {
		log.Info.Printf("%s", nup)
	}

	var ctx *model.Context

	if nup.ImgInputFile {

		if ctx, err = BookletFromImages(conf, imgFiles, nup); err != nil {
			return err
		}

	} else {
		if rs == nil {
			return ErrMissingPDFReadSeeker
		}

		if ctx, err = ReadAndValidate(rs, conf); err != nil {
			return fmt.Errorf("booklet: read and validate: %w", err)
		}

		pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
		if err != nil {
			return fmt.Errorf("booklet: parse page selection: %w", err)
		}

		if err = pdfcpu.BookletFromPDF(ctx, pages, nup); err != nil {
			return fmt.Errorf("booklet: impose pages: %w", err)
		}
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("booklet: write output: %w", err)
	}
	return nil
}

func bookletImageOutputAliasesInput(inFile, outFile string) (bool, error) {
	return outputAliasesInput(inFile, outFile)
}

func bookletImageOutputAliasesInputWith(
	inFile, outFile string,
	abs func(string) (string, error),
	stat func(string) (os.FileInfo, error),
) (bool, error) {
	return outputAliasesInputWith(inFile, outFile, abs, stat)
}

func rejectBookletImageOutputAlias(inFiles []string, outFile string) error {
	for i, inFile := range inFiles {
		aliases, err := bookletImageOutputAliasesInput(inFile, outFile)
		if err != nil {
			return fmt.Errorf("booklet image %d %q: check output alias: %w", i+1, inFile, err)
		}
		if aliases {
			return fmt.Errorf("booklet image %d %q: output aliases input: %w", i+1, inFile, ErrBookletImageOutputConflict)
		}
	}
	return nil
}

// BookletFile rearranges PDF pages or images into a booklet layout and writes the result to outFile.
func BookletFile(inFiles []string, outFile string, selectedPages []string, nup *model.NUp, conf *model.Configuration) (err error) {
	if len(inFiles) == 0 {
		if nup != nil && nup.ImgInputFile {
			return ErrMissingImageInput
		}
		return ErrMissingPDFInput
	}
	if outFile == "" {
		return ErrMissingPDFOutput
	}
	if nup == nil {
		return ErrMissingBookletConfiguration
	}
	if err := prepareBookletConfigurationForAPI(nup); err != nil {
		return err
	}
	if nup.ImgInputFile {
		if err := rejectBookletImageOutputAlias(inFiles, outFile); err != nil {
			return err
		}
	}

	var f1, f2 *os.File
	ok := false

	if !nup.ImgInputFile {
		if f1, err = os.Open(inFiles[0]); err != nil {
			return fmt.Errorf("booklet: open input %s: %w", inFiles[0], err)
		}
	}

	staged, err := openStagedOutput(f1, inFiles[0], outFile, "booklet")
	if err != nil {
		return errors.Join(
			fmt.Errorf("booklet: create output: %w", err),
			closeFile(f1, "booklet: close input"),
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

	if err = Booklet(f1, f2, inFiles, selectedPages, nup, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
