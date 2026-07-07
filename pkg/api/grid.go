/*
Copyright 2026 The pdfcpu Authors.

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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// PDFGridConfig returns a grid configuration for PDF files.
func PDFGridConfig(rows, cols int, desc string, conf *model.Configuration) (*model.NUp, error) {
	return pdfcpu.PDFGridConfig(rows, cols, desc, conf)
}

// ImageGridConfig returns a grid configuration for image files.
func ImageGridConfig(rows, cols int, desc string, conf *model.Configuration) (*model.NUp, error) {
	return pdfcpu.ImageGridConfig(rows, cols, desc, conf)
}

// ParseGridDefinition applies grid dimensions to a grid configuration.
func ParseGridDefinition(rows, cols int, nup *model.NUp) error {
	if nup == nil {
		return ErrMissingGridConfiguration
	}
	return pdfcpu.ParseNUpGridDefinition(rows, cols, nup)
}

// ParseNUpGridDefinition applies grid dimensions to a shared imposition configuration.
//
// Deprecated: use ParseGridDefinition for grid command handling.
func ParseNUpGridDefinition(rows, cols int, nup *model.NUp) error {
	return ParseGridDefinition(rows, cols, nup)
}

func prepareGridConfiguration(nup *model.NUp, imageInput bool) error {
	if !nup.PageGrid {
		return errors.New("invalid configuration: page grid disabled")
	}
	return prepareNUpConfiguration(nup, imageInput)
}

func prepareGridConfigurationForAPI(nup *model.NUp, imageInput bool) error {
	if err := prepareGridConfiguration(nup, imageInput); err != nil {
		return fmt.Errorf("grid: prepare configuration: %w", err)
	}
	return nil
}

// GridFromImage creates a page grid context for one or more images.
// On error, the returned context may be partially constructed and its PageCount remains at the pre-operation value.
// Callers must discard a non-nil context returned together with an error.
func GridFromImage(conf *model.Configuration, imageFileNames []string, nup *model.NUp) (ctx *model.Context, err error) {
	defer fault.Catch(&err)

	if nup == nil {
		return nil, ErrMissingGridConfiguration
	}
	if len(imageFileNames) == 0 {
		return nil, ErrMissingImageInput
	}
	if err := prepareGridConfigurationForAPI(nup, true); err != nil {
		return nil, err
	}
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.GRID

	ctx, err = pdfcpu.CreateContextWithXRefTable(conf, nup.PageDim)
	if err != nil {
		return nil, fmt.Errorf("grid: create image context: %w", err)
	}

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		return nil, fmt.Errorf("grid: access image page tree: %w", err)
	}

	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		return nil, fmt.Errorf("grid: dereference image page tree: %w", err)
	}

	if len(imageFileNames) == 1 {
		err = pdfcpu.GridFromOneImage(ctx, imageFileNames[0], nup, pagesDict, pagesIndRef)
	} else {
		err = pdfcpu.GridFromMultipleImages(ctx, imageFileNames, nup, pagesDict, pagesIndRef)
	}
	if err != nil {
		return ctx, fmt.Errorf("grid: impose images: %w", err)
	}
	return ctx, nil
}

// Grid rearranges PDF pages or images into page grids and writes the result to w.
// Either rs or imgFiles will be used.
func Grid(rs io.ReadSeeker, w io.Writer, imgFiles, selectedPages []string, nup *model.NUp, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if w == nil {
		return ErrMissingPDFWriter
	}
	if nup == nil {
		return ErrMissingGridConfiguration
	}
	if nup.ImgInputFile && len(imgFiles) == 0 {
		return ErrMissingImageInput
	}
	if err := prepareGridConfigurationForAPI(nup, nup.ImgInputFile); err != nil {
		return err
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.GRID

	if log.InfoEnabled() {
		log.Info.Printf("%s", nup)
	}

	var ctx *model.Context
	if nup.ImgInputFile {
		if ctx, err = GridFromImage(conf, imgFiles, nup); err != nil {
			return err
		}
	} else {
		if rs == nil {
			return ErrMissingPDFReadSeeker
		}
		if ctx, err = ReadValidateAndOptimize(rs, conf); err != nil {
			return fmt.Errorf("grid: %w", err)
		}

		pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
		if err != nil {
			return fmt.Errorf("grid: parse page selection: %w", err)
		}
		if err = pdfcpu.GridFromPDF(ctx, pages, nup); err != nil {
			return fmt.Errorf("grid: impose pages: %w", err)
		}
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("grid: write output: %w", err)
	}
	return nil
}

func rejectGridImageOutputAlias(inFiles []string, outFile string) error {
	for i, inFile := range inFiles {
		aliases, err := outputAliasesInput(inFile, outFile)
		if err != nil {
			return fmt.Errorf("grid image %d %q: check output alias: %w", i+1, inFile, err)
		}
		if aliases {
			return fmt.Errorf("grid image %d %q: output aliases input: %w", i+1, inFile, ErrGridImageOutputConflict)
		}
	}
	return nil
}

// GridFile rearranges PDF pages or images into page grids and writes the result to outFile.
func GridFile(inFiles []string, outFile string, selectedPages []string, nup *model.NUp, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if nup == nil {
		return ErrMissingGridConfiguration
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
	if err := prepareGridConfigurationForAPI(nup, nup.ImgInputFile); err != nil {
		return err
	}
	if nup.ImgInputFile {
		if err := rejectGridImageOutputAlias(inFiles, outFile); err != nil {
			return err
		}
	}

	var f1, f2 *os.File
	ok := false
	if !nup.ImgInputFile {
		if f1, err = os.Open(inFiles[0]); err != nil {
			return fmt.Errorf("grid: open input %s: %w", inFiles[0], err)
		}
	}

	staged, err := openStagedOutput(f1, inFiles[0], outFile, "grid")
	if err != nil {
		return errors.Join(
			fmt.Errorf("grid: create output: %w", err),
			closeFile(f1, "grid: close input"),
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

	if err = Grid(f1, f2, inFiles, selectedPages, nup, conf); err != nil {
		return err
	}
	ok = true
	return nil
}
