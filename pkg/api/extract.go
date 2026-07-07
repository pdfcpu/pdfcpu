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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
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

func digestImages(mm map[int]model.Image, singleImgPerPage bool, maxPageDigits int, digestImage func(model.Image, bool, int) error) (int, error) {
	objNrs := make([]int, 0, len(mm))
	for objNr := range mm {
		objNrs = append(objNrs, objNr)
	}
	sort.Ints(objNrs)
	for _, objNr := range objNrs {
		img := mm[objNr]
		if err := digestImage(img, singleImgPerPage, maxPageDigits); err != nil {
			return img.ObjNr, err
		}
	}
	return 0, nil
}

func sanitizeFilenamePart(s, fallback string) string {
	return sanitize.PathOr(s, fallback)
}

// WriteImageToDisk returns a closure for writing an image to disk.
func WriteImageToDisk(outDir, fileName string) func(model.Image, bool, int) error {
	fileName = sanitizeFilenamePart(fileName, "file")

	return func(img model.Image, singleImgPerPage bool, maxPageDigits int) error {
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
		logWritingTo(outFile)
		if err := pdfcpu.WriteReader(outFile, img.Reader); err != nil {
			if errors.Is(err, pdfcpu.ErrMissingReader) {
				return fmt.Errorf("image obj#%d: %w", img.ObjNr, ErrMissingImageReader)
			}
			return err
		}
		return nil
	}
}

// WriteFontToDisk returns a closure for writing a font file to disk.
func WriteFontToDisk(outDir, fnBase string) func(pdfcpu.Font) error {
	fnBase = sanitizeFilenamePart(fnBase, "file")

	return func(font pdfcpu.Font) error {
		fontName := sanitizeFilenamePart(font.Name, "fontName")
		fontType := sanitizeFilenamePart(font.Type, "fontType")
		outFile := filepath.Join(outDir, fmt.Sprintf("%s_%s.%s", fnBase, fontName, fontType))
		logWritingTo(outFile)
		return pdfcpu.WriteReader(outFile, font.Reader)
	}
}

// WritePageToDisk returns a closure for writing a single page PDF to disk.
func WritePageToDisk(outDir, fnBase string) func(io.Reader, int) error {
	fnBase = sanitizeFilenamePart(fnBase, "file")

	return func(rd io.Reader, pageNr int) error {
		outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fnBase, pageNr))
		logWritingTo(outFile)
		return pdfcpu.WriteReader(outFile, rd)
	}
}

// WriteContentToDisk returns a closure for writing content to disk.
func WriteContentToDisk(outDir, fnBase string) func(io.Reader, int) error {
	fnBase = sanitizeFilenamePart(fnBase, "file")

	return func(rd io.Reader, pageNr int) error {
		outFile := filepath.Join(outDir, fmt.Sprintf("%s_Content_page_%d.txt", fnBase, pageNr))
		logWritingTo(outFile)
		return pdfcpu.WriteReader(outFile, rd)
	}
}

// WriteMetadataToDisk returns a closure for writing metadata to disk.
func WriteMetadataToDisk(outDir, fnBase string) func(pdfcpu.Metadata) error {
	fnBase = sanitizeFilenamePart(fnBase, "file")

	return func(md pdfcpu.Metadata) error {
		parentType := sanitizeFilenamePart(md.ParentType, "metadata")
		outFile := filepath.Join(outDir, fmt.Sprintf("%s_Metadata_%s_%d_%d.txt", fnBase, parentType, md.ParentObjNr, md.ObjNr))
		logWritingTo(outFile)
		return pdfcpu.WriteReader(outFile, md.Reader)
	}
}

// ExtractImagesRaw returns image maps containing readers for images on selectedPages.
// Note: may be memory intensive.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError and images contains all
// successfully extracted images.
func ExtractImagesRaw(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (images []map[int]model.Image, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTIMAGES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("extract images: %w", err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return nil, fmt.Errorf("extract images: parse page selection: %w", err)
	}

	var skipErr error
	for _, pageNr := range sortedPages(pages) {
		mm, err := pdfcpu.ExtractPageImages(ctx, pageNr, false)
		if err != nil {
			if !skipUnsupportedResource(err, conf) {
				return nil, fmt.Errorf("extract images: %w", err)
			}
			skipErr = errors.Join(skipErr, fmt.Errorf("extract images: %w", err))
		}
		images = append(images, mm)
	}

	return images, unsupportedResourceError(skipErr)
}

// ExtractImages extracts and digests embedded image resources from rs for selected pages.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError after all supported
// images have been digested.
func ExtractImages(rs io.ReadSeeker, selectedPages []string, digestImage func(model.Image, bool, int) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if digestImage == nil {
		return fmt.Errorf("extract images: %w", ErrMissingDigestFunction)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTIMAGES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("extract images: %w", err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return fmt.Errorf("extract images: parse page selection: %w", err)
	}

	sp := sortedPages(pages)
	if len(sp) == 0 {
		if log.CLIEnabled() {
			log.CLI.Println("aborted: missing page numbers!")
		}
		return nil
	}

	maxPageDigits := len(strconv.Itoa(sp[len(sp)-1]))

	var skipErr error
	for i := range sp {
		pageNr := sp[i]
		mm, err := pdfcpu.ExtractPageImages(ctx, pageNr, false)
		if err != nil {
			if !skipUnsupportedResource(err, conf) {
				return fmt.Errorf("extract images: %w", err)
			}
			skipErr = errors.Join(skipErr, fmt.Errorf("extract images: %w", err))
		}
		singleImgPerPage := len(mm) == 1
		objNr, err := digestImages(mm, singleImgPerPage, maxPageDigits, digestImage)
		if err != nil {
			return fmt.Errorf("extract images: page %d image obj#%d: digest: %w", pageNr, objNr, err)
		}
	}

	return unsupportedResourceError(skipErr)
}

// ExtractImagesFile dumps embedded image resources from inFile into outDir for selected pages.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError after all supported
// images have been written.
func ExtractImagesFile(inFile, outDir string, selectedPages []string, conf *model.Configuration) (err error) {
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

	if log.CLIEnabled() {
		log.CLI.Printf("extracting images from %s into %s/ ...\n", inFile, outDir)
	}
	fileName := strings.TrimSuffix(filepath.Base(inFile), ".pdf")

	if err := ExtractImages(f, selectedPages, WriteImageToDisk(outDir, fileName), conf); err != nil {
		return fmt.Errorf("extract images %s: %w", inFile, err)
	}
	return nil
}

func writeFonts(ff []pdfcpu.Font, digestFont func(pdfcpu.Font) error) error {
	for _, f := range ff {
		if err := digestFont(f); err != nil {
			return fmt.Errorf("font %q obj#%d: digest: %w", f.Name, f.ObjNr, err)
		}
	}
	return nil
}

// ExtractFonts retrieves and digests embedded fontfiles from rs for selected pages.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError after all supported
// fonts have been digested.
func ExtractFonts(rs io.ReadSeeker, selectedPages []string, digestFont func(pdfcpu.Font) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if digestFont == nil {
		return fmt.Errorf("extract fonts: %w", ErrMissingDigestFunction)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTFONTS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("extract fonts: %w", err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return fmt.Errorf("extract fonts: parse page selection: %w", err)
	}

	objNrs, skipped := types.IntSet{}, types.IntSet{}
	var skipErr error

	for _, pageNr := range sortedPages(pages) {
		ff, err := pdfcpu.ExtractPageFonts(ctx, pageNr, objNrs, skipped)
		if err != nil {
			if !skipUnsupportedResource(err, conf) {
				return fmt.Errorf("extract fonts: %w", err)
			}
			skipErr = errors.Join(skipErr, fmt.Errorf("extract fonts: %w", err))
		}

		if err := writeFonts(ff, digestFont); err != nil {
			return fmt.Errorf("extract fonts: page %d: %w", pageNr, err)
		}
	}

	ff, err := pdfcpu.ExtractFormFonts(ctx)
	if err != nil {
		if !skipUnsupportedResource(err, conf) {
			return fmt.Errorf("extract fonts: %w", err)
		}
		skipErr = errors.Join(skipErr, fmt.Errorf("extract fonts: %w", err))
	}

	if err := writeFonts(ff, digestFont); err != nil {
		return fmt.Errorf("extract fonts: form: %w", err)
	}
	return unsupportedResourceError(skipErr)
}

// ExtractFontsFile writes embedded fontfiles from inFile into outDir for selected pages.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError after all supported
// fonts have been written.
func ExtractFontsFile(inFile, outDir string, selectedPages []string, conf *model.Configuration) (err error) {
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

	if log.CLIEnabled() {
		log.CLI.Printf("extracting fonts from %s into %s/ ...\n", inFile, outDir)
	}

	fnBase := strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	if err := ExtractFonts(f, selectedPages, WriteFontToDisk(outDir, fnBase), conf); err != nil {
		return fmt.Errorf("extract fonts %s: %w", inFile, err)
	}
	return nil
}

// ExtractPage extracts the page with pageNr out of ctx into an io.Reader.
func ExtractPage(ctx *model.Context, pageNr int) (io.Reader, error) {
	if ctx == nil {
		return nil, ErrMissingPDFContext
	}

	ctxNew, err := pdfcpu.ExtractPages(ctx, []int{pageNr}, false)
	if err != nil {
		return nil, fmt.Errorf("extract page %d: %w", pageNr, err)
	}

	var b bytes.Buffer
	if err := WriteContext(ctxNew, &b); err != nil {
		return nil, fmt.Errorf("extract page %d: write output: %w", pageNr, err)
	}

	return &b, nil
}

// ExtractPages retrieves and digests single page PDF files from rs for selected pages.
func ExtractPages(rs io.ReadSeeker, selectedPages []string, digestPage func(io.Reader, int) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if digestPage == nil {
		return fmt.Errorf("extract pages: %w", ErrMissingDigestFunction)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTPAGES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("extract pages: %w", err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return fmt.Errorf("extract pages: parse page selection: %w", err)
	}

	if len(pages) == 0 {
		if log.CLIEnabled() {
			log.CLI.Println("aborted: missing page numbers!")
		}
		return nil
	}

	sp := sortedPages(pages)

	for i := range sp {
		pageNr := sp[i]
		rd, err := ExtractPage(ctx, pageNr)
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

	return nil
}

// ExtractPagesFile generates single page PDF files from inFile in outDir for selected pages.
func ExtractPagesFile(inFile, outDir string, selectedPages []string, conf *model.Configuration) (err error) {
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

	if log.CLIEnabled() {
		log.CLI.Printf("extracting pages from %s into %s/ ...\n", inFile, outDir)
	}

	fnBase := strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	if err := ExtractPages(f, selectedPages, WritePageToDisk(outDir, fnBase), conf); err != nil {
		return fmt.Errorf("extract pages %s: %w", inFile, err)
	}
	return nil
}

// ExtractContent retrieves and digests PDF sources from rs for selected pages.
func ExtractContent(rs io.ReadSeeker, selectedPages []string, digestContent func(io.Reader, int) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if digestContent == nil {
		return fmt.Errorf("extract content: %w", ErrMissingDigestFunction)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTCONTENT

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("extract content: %w", err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return fmt.Errorf("extract content: parse page selection: %w", err)
	}

	for _, pageNr := range sortedPages(pages) {
		rd, err := pdfcpu.ExtractPageContent(ctx, pageNr)
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

	return nil
}

// ExtractContentFile dumps "PDF source" files from inFile into outDir for selected pages.
func ExtractContentFile(inFile, outDir string, selectedPages []string, conf *model.Configuration) (err error) {
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

	if log.CLIEnabled() {
		log.CLI.Printf("extracting content from %s into %s/ ...\n", inFile, outDir)
	}

	fnBase := strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	if err := ExtractContent(f, selectedPages, WriteContentToDisk(outDir, fnBase), conf); err != nil {
		return fmt.Errorf("extract content %s: %w", inFile, err)
	}
	return nil
}

// ExtractMetadata retrieves and digests all metadata dict entries for rs.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError after all supported
// metadata has been digested.
func ExtractMetadata(rs io.ReadSeeker, digestMetadata func(pdfcpu.Metadata) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if digestMetadata == nil {
		return fmt.Errorf("extract metadata: %w", ErrMissingDigestFunction)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTMETADATA

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("extract metadata: %w", err)
	}

	mdmd, err := pdfcpu.ExtractMetadata(ctx)
	if err != nil {
		if !skipUnsupportedResource(err, conf) {
			return fmt.Errorf("extract metadata: collect entries: %w", err)
		}
	}

	for _, md := range mdmd {
		if err := digestMetadata(md); err != nil {
			return fmt.Errorf("extract metadata: parent obj#%d metadata obj#%d: digest: %w", md.ParentObjNr, md.ObjNr, err)
		}
	}

	if err != nil {
		return unsupportedResourceError(fmt.Errorf("extract metadata: collect entries: %w", err))
	}
	return nil
}

// ExtractMetadataFile dumps all metadata dict entries for inFile into outDir.
// Unsupported resources are handled according to conf.UnsupportedResourcePolicy.
// In skip mode err contains an *UnsupportedResourceError after all supported
// metadata has been written.
func ExtractMetadataFile(inFile, outDir string, conf *model.Configuration) (err error) {
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

	if log.CLIEnabled() {
		log.CLI.Printf("extracting metadata from %s into %s/ ...\n", inFile, outDir)
	}

	fileNameBase := strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	if err := ExtractMetadata(f, WriteMetadataToDisk(outDir, fileNameBase), conf); err != nil {
		return fmt.Errorf("extract metadata %s: %w", inFile, err)
	}
	return nil
}
