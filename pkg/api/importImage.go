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
	"bufio"
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

// Import parses an Import command string into an internal structure.
func Import(s string, u types.DisplayUnit) (*pdfcpu.Import, error) {
	imp, err := pdfcpu.ParseImportDetails(s, u)
	if err != nil {
		return nil, fmt.Errorf("import images: parse configuration: %w", err)
	}
	return imp, nil
}

// DefaultImportConfig returns the default image import configuration.
func DefaultImportConfig() *pdfcpu.Import {
	return pdfcpu.DefaultImportConfig()
}

func validateImportConfiguration(imp *pdfcpu.Import) error {
	if imp.PageDim == nil {
		return fmt.Errorf("missing page dimensions: %w", ErrInvalidImportConfiguration)
	}
	if invalidImportNumber(imp.PageDim.Width) || invalidImportNumber(imp.PageDim.Height) {
		return fmt.Errorf("page dimensions: %w", ErrInvalidImportConfiguration)
	}
	if invalidImportScale(imp.Scale, imp.ScaleAbs) {
		return fmt.Errorf("scale factor: %w", ErrInvalidImportConfiguration)
	}
	if imp.DPI < 0 {
		return fmt.Errorf("DPI: %w", ErrInvalidImportConfiguration)
	}
	if math.IsNaN(imp.Dx) || math.IsInf(imp.Dx, 0) || math.IsNaN(imp.Dy) || math.IsInf(imp.Dy, 0) {
		return fmt.Errorf("offset: %w", ErrInvalidImportConfiguration)
	}
	if imp.Pos < types.TopLeft || imp.Pos > types.Full {
		return fmt.Errorf("position: %w", ErrInvalidImportConfiguration)
	}
	return nil
}

func invalidImportNumber(v float64) bool {
	return v <= 0 || math.IsNaN(v) || math.IsInf(v, 0)
}

func invalidImportScale(scale float64, absolute bool) bool {
	if invalidImportNumber(scale) {
		return true
	}
	return !absolute && scale > 1
}

// PrepareImportConfiguration applies image import defaults and validates the result without performing I/O.
func PrepareImportConfiguration(imp *pdfcpu.Import) (*pdfcpu.Import, error) {
	if imp == nil {
		imp = pdfcpu.DefaultImportConfig()
	}
	if err := validateImportConfiguration(imp); err != nil {
		return nil, fmt.Errorf("import images: validate configuration: %w", err)
	}
	return imp, nil
}

func validateImportImageReaders(imgs []io.Reader) error {
	if len(imgs) == 0 {
		return ErrMissingImageInput
	}
	for i, r := range imgs {
		if r == nil {
			return fmt.Errorf("import images: image %d: %w", i+1, ErrMissingImageReader)
		}
	}
	return nil
}

func validateImportImageFiles(imgFiles []string) error {
	if len(imgFiles) == 0 {
		return ErrMissingImageInput
	}
	for i, fileName := range imgFiles {
		if fileName == "" {
			return fmt.Errorf("import images: image %d: %w", i+1, ErrMissingImageInput)
		}
	}
	return nil
}

func validateImportImagesOutput(imgFiles []string, outFile string, skipStdinMarker bool) error {
	if err := validateImportImageFiles(imgFiles); err != nil {
		return err
	}
	if outFile == "" {
		return ErrMissingPDFOutput
	}
	for i, imageFile := range imgFiles {
		if skipStdinMarker && imageFile == "-" {
			continue
		}
		aliases, err := outputAliasesInput(imageFile, outFile)
		if err != nil {
			return fmt.Errorf("import images: image %d %q: check output alias: %w", i+1, imageFile, err)
		}
		if aliases {
			return fmt.Errorf(
				"import images: image %d %q: output aliases input: %w",
				i+1,
				imageFile,
				ErrImportImagesOutputConflict,
			)
		}
	}
	return nil
}

// ValidateImportImagesOutput validates that outFile does not alias a file-based image input.
// A "-" image input denotes stdin and is ignored.
func ValidateImportImagesOutput(imgFiles []string, outFile string) error {
	return validateImportImagesOutput(imgFiles, outFile, true)
}

func importImagesContext(rs io.ReadSeeker, imp *pdfcpu.Import, conf *model.Configuration) (*model.Context, error) {
	if rs != nil {
		ctx, err := ReadAndValidate(rs, conf)
		if err != nil {
			return nil, fmt.Errorf("import images: prepare PDF context: %w", err)
		}
		return ctx, nil
	}
	ctx, err := pdfcpu.CreateContextWithXRefTable(conf, imp.PageDim)
	if err != nil {
		return nil, fmt.Errorf("import images: create PDF context: %w", err)
	}
	return ctx, nil
}

func importImagesPageTree(ctx *model.Context) (*types.IndirectRef, types.Dict, error) {
	pagesIndRef, err := ctx.Pages()
	if err != nil {
		return nil, nil, fmt.Errorf("import images: access page tree: %w", err)
	}
	if pagesIndRef == nil {
		return nil, nil, errors.New("import images: access page tree: missing page tree reference")
	}
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		return nil, nil, fmt.Errorf("import images: dereference page tree: %w", err)
	}
	if pagesDict == nil {
		return nil, nil, errors.New("import images: dereference page tree: missing page tree dictionary")
	}
	return pagesIndRef, pagesDict, nil
}

func appendImportedImagePages(
	ctx *model.Context,
	indRefs []*types.IndirectRef,
	imageIndex int,
	pagesDict types.Dict,
) error {
	for pageIndex, indRef := range indRefs {
		if indRef == nil {
			return fmt.Errorf("import images: image %d page %d: missing page reference", imageIndex, pageIndex+1)
		}
		if err := ctx.SetValid(*indRef); err != nil {
			return fmt.Errorf("import images: image %d page %d: mark valid: %w", imageIndex, pageIndex+1, err)
		}
		if err := model.AppendPageTree(indRef, 1, pagesDict); err != nil {
			return fmt.Errorf("import images: image %d page %d: append page tree: %w", imageIndex, pageIndex+1, err)
		}
		ctx.PageCount++
	}
	return nil
}

func appendImportedImages(
	ctx *model.Context,
	imgs []io.Reader,
	pagesIndRef *types.IndirectRef,
	pagesDict types.Dict,
	imp *pdfcpu.Import,
) error {
	for imageIndex, r := range imgs {
		indRefs, err := pdfcpu.NewPagesForImage(ctx.XRefTable, r, pagesIndRef, imp)
		if err != nil {
			return fmt.Errorf("import images: image %d: create pages: %w", imageIndex+1, err)
		}
		if err := appendImportedImagePages(ctx, indRefs, imageIndex+1, pagesDict); err != nil {
			return err
		}
	}
	return nil
}

// ImportImages appends PDF pages containing images to rs and writes the result to w.
// If rs == nil a new PDF file will be written to w.
func ImportImages(rs io.ReadSeeker, w io.Writer, imgs []io.Reader, imp *pdfcpu.Import, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if w == nil {
		return ErrMissingPDFWriter
	}
	if err := validateImportImageReaders(imgs); err != nil {
		return err
	}

	imp, err = PrepareImportConfiguration(imp)
	if err != nil {
		return err
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.IMPORTIMAGES

	ctx, err := importImagesContext(rs, imp, conf)
	if err != nil {
		return err
	}
	pagesIndRef, pagesDict, err := importImagesPageTree(ctx)
	if err != nil {
		return err
	}
	if err := appendImportedImages(ctx, imgs, pagesIndRef, pagesDict, imp); err != nil {
		return err
	}
	if err := Write(ctx, w, conf); err != nil {
		return fmt.Errorf("import images: write output: %w", err)
	}
	return nil
}

type importImageFileCloser struct {
	closer     io.Closer
	imageIndex int
	fileName   string
}

func prepImgFiles(imgFiles []string) ([]importImageFileCloser, []io.Reader, error) {
	rc := make([]importImageFileCloser, 0, len(imgFiles))
	rr := make([]io.Reader, 0, len(imgFiles))
	for i, fn := range imgFiles {
		f, err := os.Open(fn)
		if err != nil {
			return nil, nil, errors.Join(
				fmt.Errorf("import images: image %d %q: open: %w", i+1, fn, err),
				closeImportImageInputs(rc),
			)
		}
		rc = append(rc, importImageFileCloser{
			closer:     f,
			imageIndex: i + 1,
			fileName:   fn,
		})
		rr = append(rr, bufio.NewReader(f))
	}
	return rc, rr, nil
}

func logImportImages(s, outFile string) {
	if log.CLIEnabled() {
		log.CLI.Printf("%s to %s...\n", s, outFile)
	}
}

func importImagesInputFile(outFile string) (io.ReadSeeker, *os.File, error) {
	f, err := os.Open(outFile)
	if err == nil {
		logImportImages("appending", outFile)
		return f, f, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, nil, fmt.Errorf("import images: inspect output %s: %w", outFile, err)
	}
	logImportImages("writing", outFile)
	return nil, nil, nil
}

func closeImportImageInputs(rc []importImageFileCloser) error {
	errs := make([]error, 0, len(rc))
	for _, c := range rc {
		if c.closer == nil {
			continue
		}
		if err := c.closer.Close(); err != nil {
			errs = append(
				errs,
				fmt.Errorf("import images: image %d %q: close: %w", c.imageIndex, c.fileName, err),
			)
		}
	}
	return errors.Join(errs...)
}

// ImportImagesFile appends PDF pages containing images to outFile which will be created if necessary.
func ImportImagesFile(imgFiles []string, outFile string, imp *pdfcpu.Import, conf *model.Configuration) (err error) {
	if err := validateImportImageFiles(imgFiles); err != nil {
		return err
	}
	if outFile == "" {
		return ErrMissingPDFOutput
	}
	imp, err = PrepareImportConfiguration(imp)
	if err != nil {
		return err
	}
	if err := validateImportImagesOutput(imgFiles, outFile, false); err != nil {
		return err
	}

	ok := false

	rs, f1, err := importImagesInputFile(outFile)
	if err != nil {
		return err
	}

	rc, rr, err := prepImgFiles(imgFiles)
	if err != nil {
		return errors.Join(err, closeFile(f1, "import images: close input"))
	}

	inFile := ""
	if f1 != nil {
		inFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, outFile, "import images")
	if err != nil {
		return errors.Join(
			fmt.Errorf("import images: create output: %w", err),
			closeFile(f1, "import images: close input"),
			closeImportImageInputs(rc),
		)
	}
	f2 := staged.output.file
	staged = staged.withCloser(func() error { return closeImportImageInputs(rc) })
	staged.replaceContext = "import images: replace output"

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = ImportImages(rs, f2, rr, imp, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
