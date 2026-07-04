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
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/sanitize"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func sanitizeFilenamePart(s, fallback string) string {
	return sanitize.PathOr(s, fallback)
}

// WriteImageToDisk returns a closure for writing an image to disk.
func WriteImageToDisk(outDir, fileName string) func(model.Image, bool, int) error {
	fileName = sanitizeFilenamePart(fileName, "file")

	return func(img model.Image, singleImgPerPage bool, maxPageDigits int) error {
		if img.Reader == nil {
			return nil
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
		return pdfcpu.WriteReader(outFile, img)
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
		return pdfcpu.WriteReader(outFile, font)
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
		return pdfcpu.WriteReader(outFile, md)
	}
}

// ExtractImagesRaw returns []pdfcpu.Image containing io.Readers for images contained in selectedPages.
// Note: may be memory intensive.
func ExtractImagesRaw(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (images []map[int]model.Image, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, errors.New("pdfcpu: ExtractImagesRaw: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTIMAGES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return nil, err
	}

	for i, v := range pages {
		if !v {
			continue
		}
		mm, err := pdfcpu.ExtractPageImages(ctx, i, false)
		if err != nil {
			return nil, err
		}
		images = append(images, mm)
	}

	return images, nil
}

// ExtractImages extracts and digests embedded image resources from rs for selected pages.
func ExtractImages(rs io.ReadSeeker, selectedPages []string, digestImage func(model.Image, bool, int) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: ExtractImages: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTIMAGES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	sp := sortedPages(pages)
	maxPageDigits := len(strconv.Itoa(sp[len(sp)-1]))

	for i := range sp {
		mm, err := pdfcpu.ExtractPageImages(ctx, sp[i], false)
		if err != nil {
			return err
		}
		singleImgPerPage := len(mm) == 1
		for _, img := range mm {
			if err := digestImage(img, singleImgPerPage, maxPageDigits); err != nil {
				return err
			}
		}
	}

	return nil
}

// ExtractImagesFile dumps embedded image resources from inFile into outDir for selected pages.
func ExtractImagesFile(inFile, outDir string, selectedPages []string, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if log.CLIEnabled() {
		log.CLI.Printf("extracting images from %s into %s/ ...\n", inFile, outDir)
	}
	fileName := strings.TrimSuffix(filepath.Base(inFile), ".pdf")

	return ExtractImages(f, selectedPages, WriteImageToDisk(outDir, fileName), conf)
}

func writeFonts(ff []pdfcpu.Font, digestFont func(pdfcpu.Font) error) error {
	for _, f := range ff {
		if err := digestFont(f); err != nil {
			return err
		}
	}
	return nil
}

// ExtractFonts retrieves and digests embedded fontfiles from rs for selected pages.
func ExtractFonts(rs io.ReadSeeker, selectedPages []string, digestFont func(pdfcpu.Font) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: ExtractFonts: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTFONTS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	objNrs, skipped := types.IntSet{}, types.IntSet{}

	for i, v := range pages {
		if !v {
			continue
		}

		ff, err := pdfcpu.ExtractPageFonts(ctx, i, objNrs, skipped)
		if err != nil {
			return err
		}

		if err := writeFonts(ff, digestFont); err != nil {
			return err
		}
	}

	ff, err := pdfcpu.ExtractFormFonts(ctx)
	if err != nil {
		return err
	}

	return writeFonts(ff, digestFont)
}

// ExtractFontsFile writes embedded fontfiles from inFile into outDir for selected pages.
func ExtractFontsFile(inFile, outDir string, selectedPages []string, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if log.CLIEnabled() {
		log.CLI.Printf("extracting fonts from %s into %s/ ...\n", inFile, outDir)
	}

	fnBase := strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	return ExtractFonts(f, selectedPages, WriteFontToDisk(outDir, fnBase), conf)
}

// ExtractPage extracts the page with pageNr out of ctx into an io.Reader.
func ExtractPage(ctx *model.Context, pageNr int) (io.Reader, error) {
	ctxNew, err := pdfcpu.ExtractPages(ctx, []int{pageNr}, false)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := WriteContext(ctxNew, &b); err != nil {
		return nil, err
	}

	return &b, nil
}

// ExtractPages retrieves and digests single page PDF files from rs for selected pages.
func ExtractPages(rs io.ReadSeeker, selectedPages []string, digestPage func(io.Reader, int) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: ExtractPages: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTPAGES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
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
			return err
		}
		if rd == nil {
			continue
		}

		if err := digestPage(rd, pageNr); err != nil {
			return err
		}
	}

	return nil
}

// ExtractPagesFile generates single page PDF files from inFile in outDir for selected pages.
func ExtractPagesFile(inFile, outDir string, selectedPages []string, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if log.CLIEnabled() {
		log.CLI.Printf("extracting pages from %s into %s/ ...\n", inFile, outDir)
	}

	fnBase := strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	return ExtractPages(f, selectedPages, WritePageToDisk(outDir, fnBase), conf)
}

// ExtractContent retrieves and digests "PDF sources from rs for selected pages.
func ExtractContent(rs io.ReadSeeker, selectedPages []string, digestContent func(io.Reader, int) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: ExtractContent: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTCONTENT

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	for p, v := range pages {
		if !v {
			continue
		}

		rd, err := pdfcpu.ExtractPageContent(ctx, p)
		if err != nil {
			return err
		}
		if rd == nil {
			continue
		}

		if err := digestContent(rd, p); err != nil {
			return err
		}
	}

	return nil
}

// ExtractContentFile dumps "PDF source" files from inFile into outDir for selected pages.
func ExtractContentFile(inFile, outDir string, selectedPages []string, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if log.CLIEnabled() {
		log.CLI.Printf("extracting content from %s into %s/ ...\n", inFile, outDir)
	}

	fnBase := strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	return ExtractContent(f, selectedPages, WriteContentToDisk(outDir, fnBase), conf)
}

// ExtractMetadata retrieves and digests all metadata dict entries for rs.
func ExtractMetadata(rs io.ReadSeeker, digestMetadata func(pdfcpu.Metadata) error, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: ExtractMetadata: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTMETADATA

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	mdmd, err := pdfcpu.ExtractMetadata(ctx)
	if err != nil {
		return err
	}

	for _, md := range mdmd {
		if err := digestMetadata(md); err != nil {
			return err
		}
	}

	return nil
}

// ExtractMetadataFile dumps all metadata dict entries for inFile into outDir.
func ExtractMetadataFile(inFile, outDir string, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if log.CLIEnabled() {
		log.CLI.Printf("extracting metadata from %s into %s/ ...\n", inFile, outDir)
	}

	fileNameBase := strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	return ExtractMetadata(f, WriteMetadataToDisk(outDir, fileNameBase), conf)
}
