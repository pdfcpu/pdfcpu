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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/sanitize"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// UnsupportedResourceError aggregates resources skipped during an otherwise
// successful extraction.
type UnsupportedResourceError struct {
	// Err contains the contextual errors for all skipped resources.
	Err error
}

// Error returns the aggregated unsupported-resource error text.
func (e *UnsupportedResourceError) Error() string {
	if e == nil || e.Err == nil {
		return pdfcpu.ErrUnsupportedResource.Error()
	}
	return e.Err.Error()
}

// Unwrap returns the aggregated unsupported-resource causes.
func (e *UnsupportedResourceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func unsupportedResourceError(err error) error {
	if err == nil {
		return nil
	}
	return &UnsupportedResourceError{Err: err}
}

func skipUnsupportedResource(err error, conf *model.Configuration) bool {
	return errors.Is(err, pdfcpu.ErrUnsupportedResource) &&
		conf.UnsupportedResourcePolicy == model.UnsupportedResourceSkip
}

type extractionErrorWithoutSkipMarker struct {
	message string
	cause   error
}

func (e extractionErrorWithoutSkipMarker) Error() string {
	return e.message
}

func (e extractionErrorWithoutSkipMarker) Unwrap() error {
	return e.cause
}

func joinExtractionCleanupError(err, cleanupErr error) error {
	if cleanupErr == nil {
		return err
	}
	var unsupportedErr *UnsupportedResourceError
	if errors.As(err, &unsupportedErr) {
		cause := unsupportedErr.Err
		if cause == nil {
			cause = pdfcpu.ErrUnsupportedResource
		}
		err = extractionErrorWithoutSkipMarker{message: err.Error(), cause: cause}
	}
	return errors.Join(err, cleanupErr)
}

func digestImages(c context.Context, mm map[int]model.Image, singleImgPerPage bool, maxPageDigits int, digestImage func(model.Image, bool, int) error) (int, error) {
	objNrs := make([]int, 0, len(mm))
	for objNr := range mm {
		if err := contextutil.Check(c); err != nil {
			return 0, err
		}
		objNrs = append(objNrs, objNr)
	}
	sort.Ints(objNrs)
	for _, objNr := range objNrs {
		if err := contextutil.Check(c); err != nil {
			return 0, err
		}
		img := mm[objNr]
		if err := digestImage(img, singleImgPerPage, maxPageDigits); err != nil {
			return img.ObjNr, err
		}
	}
	return 0, contextutil.Check(c)
}

func sanitizeFilenamePart(s, fallback string) string {
	return sanitize.PathOr(s, fallback)
}

// WriteImageToDisk returns a closure for writing an image transactionally to disk with cancellation.
func WriteImageToDisk(c context.Context, outDir, fileName string) func(model.Image, bool, int) error {
	fileName = sanitizeFilenamePart(fileName, "file")

	return func(img model.Image, singleImgPerPage bool, maxPageDigits int) error {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if img.Reader == nil {
			return fmt.Errorf("image obj#%d: %w", img.ObjNr, ErrMissingImageReader)
		}
		s := "%s_%" + fmt.Sprintf("0%dd", maxPageDigits)
		qual := img.Name
		if img.Thumb {
			qual = "thumb"
		}
		qual = sanitizeFilenamePart(qual, "image")
		fileType := sanitizeFilenamePart(img.FileType, "img")
		f := fmt.Sprintf(s+"_%s.%s", fileName, img.PageNr, qual, fileType)
		outFile := filepath.Join(outDir, f)
		if err := pdfcpu.WriteReader(c, outFile, img.Reader); err != nil {
			if errors.Is(err, pdfcpu.ErrMissingReader) {
				return fmt.Errorf("image obj#%d: %w", img.ObjNr, ErrMissingImageReader)
			}
			return err
		}
		return nil
	}
}

// WriteFontToDisk returns a closure for writing a font file transactionally to disk with cancellation.
func WriteFontToDisk(c context.Context, outDir, fnBase string) func(pdfcpu.Font) error {
	fnBase = sanitizeFilenamePart(fnBase, "file")

	return func(font pdfcpu.Font) error {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		fontName := sanitizeFilenamePart(font.Name, "fontName")
		fontType := sanitizeFilenamePart(font.Type, "fontType")
		outFile := filepath.Join(outDir, fmt.Sprintf("%s_%s.%s", fnBase, fontName, fontType))
		return pdfcpu.WriteReader(c, outFile, font.Reader)
	}
}

// WritePageToDisk returns a closure for writing a single page PDF transactionally with cancellation.
func WritePageToDisk(c context.Context, outDir, fnBase string) func(io.Reader, int) error {
	fnBase = sanitizeFilenamePart(fnBase, "file")

	return func(rd io.Reader, pageNr int) error {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fnBase, pageNr))
		return pdfcpu.WriteReader(c, outFile, rd)
	}
}

// WriteContentToDisk returns a closure for writing content transactionally to disk with cancellation.
func WriteContentToDisk(c context.Context, outDir, fnBase string) func(io.Reader, int) error {
	fnBase = sanitizeFilenamePart(fnBase, "file")

	return func(rd io.Reader, pageNr int) error {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		outFile := filepath.Join(outDir, fmt.Sprintf("%s_Content_page_%d.txt", fnBase, pageNr))
		return pdfcpu.WriteReader(c, outFile, rd)
	}
}

// WriteMetadataToDisk returns a closure for writing metadata transactionally to disk with cancellation.
func WriteMetadataToDisk(c context.Context, outDir, fnBase string) func(pdfcpu.Metadata) error {
	fnBase = sanitizeFilenamePart(fnBase, "file")

	return func(md pdfcpu.Metadata) error {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		parentType := sanitizeFilenamePart(md.ParentType, "metadata")
		outFile := filepath.Join(outDir, fmt.Sprintf("%s_Metadata_%s_%d_%d.txt", fnBase, parentType, md.ParentObjNr, md.ObjNr))
		return pdfcpu.WriteReader(c, outFile, md.Reader)
	}
}

// ExtractImagesRaw returns image maps containing readers for images on selectedPages.
// Note: may be memory intensive.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError and images contains all successfully extracted images.
// Extraction supports cancellation and may be memory intensive.
func ExtractImagesRaw(c context.Context, rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (images []map[int]model.Image, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.EXTRACTIMAGES)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return nil, fmt.Errorf("extract images: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return nil, fmt.Errorf("extract images: parse page selection: %w", err)
	}

	var skipErr error
	for _, pageNr := range sortedPages(pages) {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		mm, err := pdfcpu.ExtractPageImages(c, ctx, pageNr, false)
		if err != nil {
			if !skipUnsupportedResource(err, conf) {
				return nil, fmt.Errorf("extract images: %w", err)
			}
			skipErr = errors.Join(skipErr, fmt.Errorf("extract images: %w", err))
		}
		images = append(images, mm)
	}

	return images, errors.Join(unsupportedResourceError(skipErr), contextutil.Check(c))
}

// ExtractImages extracts and digests embedded image resources from rs for selected pages.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError after all supported images have been digested.
// Extraction supports cancellation.
func ExtractImages(c context.Context, rs io.ReadSeeker, selectedPages []string, digestImage func(model.Image, bool, int) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if digestImage == nil {
		return fmt.Errorf("extract images: %w", ErrMissingDigestFunction)
	}

	conf = operationConfiguration(conf, model.EXTRACTIMAGES)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("extract images: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("extract images: parse page selection: %w", err)
	}

	sp := sortedPages(pages)
	if len(sp) == 0 {
		return nil
	}

	maxPageDigits := len(strconv.Itoa(sp[len(sp)-1]))

	var skipErr error
	for i := range sp {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		pageNr := sp[i]
		mm, err := pdfcpu.ExtractPageImages(c, ctx, pageNr, false)
		if err != nil {
			if !skipUnsupportedResource(err, conf) {
				return fmt.Errorf("extract images: %w", err)
			}
			skipErr = errors.Join(skipErr, fmt.Errorf("extract images: %w", err))
		}
		singleImgPerPage := len(mm) == 1
		objNr, err := digestImages(c, mm, singleImgPerPage, maxPageDigits, digestImage)
		if err != nil {
			return fmt.Errorf("extract images: page %d image obj#%d: digest: %w", pageNr, objNr, err)
		}
	}

	return errors.Join(unsupportedResourceError(skipErr), contextutil.Check(c))
}

// ExtractImagesFile dumps embedded image resources from inFile into outDir for selected pages.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError after all supported images have been written.
// Extraction supports cancellation.
func ExtractImagesFile(c context.Context, inFile, outDir string, selectedPages []string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("extract images: open input %s: %w", inFile, err)
	}
	defer func() {
		err = joinExtractionCleanupError(err, closeFile(f, "extract images: close input"))
	}()

	fileName := strings.TrimSuffix(filepath.Base(inFile), ".pdf")

	if err := ExtractImages(
		c, f, selectedPages, WriteImageToDisk(c, outDir, fileName), conf,
	); err != nil {
		return fmt.Errorf("extract images %s: %w", inFile, err)
	}
	return nil
}

func writeFonts(c context.Context, ff []pdfcpu.Font, digestFont func(pdfcpu.Font) error) error {
	for _, f := range ff {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if err := digestFont(f); err != nil {
			return fmt.Errorf("font %q obj#%d: digest: %w", f.Name, f.ObjNr, err)
		}
	}
	return contextutil.Check(c)
}

func extractPageFonts(c context.Context, ctx *model.Context, pageNr int, objNrs, skipped types.IntSet, digestFont func(pdfcpu.Font) error, conf *model.Configuration) (error, error) {
	ff, err := pdfcpu.ExtractPageFonts(c, ctx, pageNr, objNrs, skipped)
	if ctxErr := contextutil.Check(c); ctxErr != nil {
		return nil, ctxErr
	}
	var skipErr error
	if err != nil {
		if !skipUnsupportedResource(err, conf) {
			return nil, err
		}
		skipErr = fmt.Errorf("extract fonts: %w", err)
	}
	if err := writeFonts(c, ff, digestFont); err != nil {
		return nil, fmt.Errorf("page %d: %w", pageNr, err)
	}
	return skipErr, nil
}

// ExtractFonts retrieves and digests embedded fontfiles from rs for selected pages.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError after all supported fonts have been digested.
// Extraction supports cancellation.
func ExtractFonts(c context.Context, rs io.ReadSeeker, selectedPages []string, digestFont func(pdfcpu.Font) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if digestFont == nil {
		return fmt.Errorf("extract fonts: %w", ErrMissingDigestFunction)
	}

	conf = operationConfiguration(conf, model.EXTRACTFONTS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("extract fonts: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("extract fonts: parse page selection: %w", err)
	}

	objNrs, skipped := types.IntSet{}, types.IntSet{}
	var skipErr error

	for _, pageNr := range sortedPages(pages) {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		pageSkipErr, err := extractPageFonts(c, ctx, pageNr, objNrs, skipped, digestFont, conf)
		if err != nil {
			return fmt.Errorf("extract fonts: %w", err)
		}
		skipErr = errors.Join(skipErr, pageSkipErr)
	}

	ff, err := pdfcpu.ExtractFormFonts(c, ctx)
	if ctxErr := contextutil.Check(c); ctxErr != nil {
		return ctxErr
	}
	if err != nil {
		if !skipUnsupportedResource(err, conf) {
			return fmt.Errorf("extract fonts: %w", err)
		}
		skipErr = errors.Join(skipErr, fmt.Errorf("extract fonts: %w", err))
	}

	if err := writeFonts(c, ff, digestFont); err != nil {
		return fmt.Errorf("extract fonts: form: %w", err)
	}
	return errors.Join(unsupportedResourceError(skipErr), contextutil.Check(c))
}

// ExtractFontsFile writes embedded fontfiles from inFile into outDir for selected pages.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError after all supported fonts have been written.
// Extraction supports cancellation.
func ExtractFontsFile(c context.Context, inFile, outDir string, selectedPages []string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("extract fonts: open input %s: %w", inFile, err)
	}
	defer func() {
		err = joinExtractionCleanupError(err, closeFile(f, "extract fonts: close input"))
	}()

	fnBase := strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	if err := ExtractFonts(
		c, f, selectedPages, WriteFontToDisk(c, outDir, fnBase), conf,
	); err != nil {
		return fmt.Errorf("extract fonts %s: %w", inFile, err)
	}
	return nil
}

// ExtractPage extracts pageNr out of ctx into an io.Reader and supports cancellation.
func ExtractPage(c context.Context, ctx *model.Context, pageNr int) (io.Reader, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if ctx == nil {
		return nil, ErrMissingPDFContext
	}

	ctxNew, err := pdfcpu.ExtractPages(c, ctx, []int{pageNr}, false)
	if err != nil {
		return nil, fmt.Errorf("extract page %d: %w", pageNr, err)
	}

	var b bytes.Buffer
	if err := WriteContext(c, ctxNew, &b); err != nil {
		return nil, fmt.Errorf("extract page %d: write output: %w", pageNr, err)
	}

	return &b, nil
}

// ExtractPages retrieves and digests single page PDF files for selected pages and supports cancellation.
func ExtractPages(c context.Context, rs io.ReadSeeker, selectedPages []string, digestPage func(io.Reader, int) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if digestPage == nil {
		return fmt.Errorf("extract pages: %w", ErrMissingDigestFunction)
	}

	conf = operationConfiguration(conf, model.EXTRACTPAGES)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("extract pages: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("extract pages: parse page selection: %w", err)
	}

	if len(pages) == 0 {
		return nil
	}

	sp := sortedPages(pages)

	for i := range sp {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		pageNr := sp[i]
		rd, err := ExtractPage(c, ctx, pageNr)
		if err != nil {
			return fmt.Errorf("extract pages: %w", err)
		}
		if rd == nil {
			continue
		}

		if err := digestPage(rd, pageNr); err != nil {
			return fmt.Errorf("extract pages: page %d: digest: %w", pageNr, err)
		}
	}

	return contextutil.Check(c)
}

// ExtractPagesFile generates single page PDF files for selected pages and supports cancellation.
func ExtractPagesFile(c context.Context, inFile, outDir string, selectedPages []string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("extract pages: open input %s: %w", inFile, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(f, "extract pages: close input"))
	}()

	fnBase := strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	if err := ExtractPages(
		c, f, selectedPages, WritePageToDisk(c, outDir, fnBase), conf,
	); err != nil {
		return fmt.Errorf("extract pages %s: %w", inFile, err)
	}
	return nil
}

// ExtractContent retrieves and digests PDF sources for selected pages and supports cancellation.
func ExtractContent(c context.Context, rs io.ReadSeeker, selectedPages []string, digestContent func(io.Reader, int) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if digestContent == nil {
		return fmt.Errorf("extract content: %w", ErrMissingDigestFunction)
	}

	conf = operationConfiguration(conf, model.EXTRACTCONTENT)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("extract content: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("extract content: parse page selection: %w", err)
	}

	for _, pageNr := range sortedPages(pages) {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		rd, err := pdfcpu.ExtractPageContent(c, ctx, pageNr)
		if err != nil {
			return fmt.Errorf("extract content: %w", err)
		}
		if rd == nil {
			continue
		}

		if err := digestContent(rd, pageNr); err != nil {
			return fmt.Errorf("extract content: page %d: digest content: %w", pageNr, err)
		}
	}

	return contextutil.Check(c)
}

// ExtractContentFile dumps PDF sources into outDir for selected pages and supports cancellation.
func ExtractContentFile(c context.Context, inFile, outDir string, selectedPages []string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("extract content: open input %s: %w", inFile, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(f, "extract content: close input"))
	}()

	fnBase := strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	if err := ExtractContent(
		c, f, selectedPages, WriteContentToDisk(c, outDir, fnBase), conf,
	); err != nil {
		return fmt.Errorf("extract content %s: %w", inFile, err)
	}
	return nil
}

// ExtractMetadata retrieves and digests all metadata dict entries for rs.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError after all supported metadata has been digested.
// Extraction supports cancellation.
func ExtractMetadata(c context.Context, rs io.ReadSeeker, digestMetadata func(pdfcpu.Metadata) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if digestMetadata == nil {
		return fmt.Errorf("extract metadata: %w", ErrMissingDigestFunction)
	}

	conf = operationConfiguration(conf, model.EXTRACTMETADATA)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("extract metadata: %w", err)
	}

	mdmd, err := pdfcpu.ExtractMetadata(c, ctx)
	if ctxErr := contextutil.Check(c); ctxErr != nil {
		return ctxErr
	}
	if err != nil {
		if !skipUnsupportedResource(err, conf) {
			return fmt.Errorf("extract metadata: collect entries: %w", err)
		}
	}

	for _, md := range mdmd {
		if ctxErr := contextutil.Check(c); ctxErr != nil {
			return ctxErr
		}
		if err := digestMetadata(md); err != nil {
			return fmt.Errorf("extract metadata: parent obj#%d metadata obj#%d: digest: %w", md.ParentObjNr, md.ObjNr, err)
		}
	}

	if err != nil {
		return unsupportedResourceError(fmt.Errorf("extract metadata: collect entries: %w", err))
	}
	return contextutil.Check(c)
}

// ExtractMetadataFile dumps all metadata dict entries for inFile into outDir.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError after all supported metadata has been written.
// Extraction supports cancellation.
func ExtractMetadataFile(c context.Context, inFile, outDir string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("extract metadata: open input %s: %w", inFile, err)
	}
	defer func() {
		err = joinExtractionCleanupError(err, closeFile(f, "extract metadata: close input"))
	}()

	fileNameBase := strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	if err := ExtractMetadata(
		c, f, WriteMetadataToDisk(c, outDir, fileNameBase), conf,
	); err != nil {
		return fmt.Errorf("extract metadata %s: %w", inFile, err)
	}
	return nil
}
