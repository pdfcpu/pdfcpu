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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// ExtractImagesRaw returns []pdfcpu.Image containing io.Readers for images contained in selectedPages.
// Beware of memory intensive returned slice.
func ExtractImagesRaw(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) ([]map[int]model.Image, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: ExtractImages: missing rs")
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

	var images []map[int]model.Image
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
func ExtractImages(rs io.ReadSeeker, selectedPages []string, digestImage func(model.Image, bool, int) error, conf *model.Configuration) error {
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

	pageNrs := []int{}
	for k, v := range pages {
		if !v {
			continue
		}
		pageNrs = append(pageNrs, k)
	}

	sort.Ints(pageNrs)
	maxPageDigits := len(strconv.Itoa(pageNrs[len(pageNrs)-1]))

	for _, i := range pageNrs {
		mm, err := pdfcpu.ExtractPageImages(ctx, i, false)
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

	return ExtractImages(f, selectedPages, pdfcpu.WriteImageToDisk(outDir, fileName), conf)
}

func writeFonts(ff []pdfcpu.Font, outDir, fileName string) error {
	for _, f := range ff {
		outFile := filepath.Join(outDir, fmt.Sprintf("%s_%s.%s", fileName, f.Name, f.Type))
		logWritingTo(outFile)
		w, err := os.Create(outFile)
		if err != nil {
			return err
		}
		if _, err = io.Copy(w, f); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
	}

	return nil
}

// ExtractFonts dumps embedded fontfiles from rs into outDir for selected pages.
func ExtractFonts(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, conf *model.Configuration) error {
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

	fileName = strings.TrimSuffix(filepath.Base(fileName), ".pdf")

	objNrs, skipped := types.IntSet{}, types.IntSet{}

	for i, v := range pages {
		if !v {
			continue
		}
		ff, err := pdfcpu.ExtractPageFonts(ctx, i, objNrs, skipped)
		if err != nil {
			return err
		}
		if err := writeFonts(ff, outDir, fileName); err != nil {
			return err
		}
	}

	ff, err := pdfcpu.ExtractFormFonts(ctx)
	if err != nil {
		return err
	}

	return writeFonts(ff, outDir, fileName)
}

// ExtractFontsFile dumps embedded fontfiles from inFile into outDir for selected pages.
func ExtractFontsFile(inFile, outDir string, selectedPages []string, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if log.CLIEnabled() {
		log.CLI.Printf("extracting fonts from %s into %s/ ...\n", inFile, outDir)
	}

	return ExtractFonts(f, outDir, filepath.Base(inFile), selectedPages, conf)
}

// WritePage consumes an io.Reader containing some PDF bytes and writes to outDir/fileName.
func WritePage(r io.Reader, outDir, fileName string, pageNr int) error {
	outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fileName, pageNr))
	logWritingTo(outFile)
	w, err := os.Create(outFile)
	if err != nil {
		return err
	}
	if _, err = io.Copy(w, r); err != nil {
		return err
	}
	return w.Close()
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

// ExtractPages generates single page PDF files from rs in outDir for selected pages.
func ExtractPages(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, conf *model.Configuration) error {
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

	fileName = strings.TrimSuffix(filepath.Base(fileName), ".pdf")

	for _, i := range sortedPages(pages) {
		r, err := ExtractPage(ctx, i)
		if err != nil {
			return err
		}
		if err := WritePage(r, outDir, fileName, i); err != nil {
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

	return ExtractPages(f, outDir, filepath.Base(inFile), selectedPages, conf)
}

// ExtractContent dumps "PDF source" files from rs into outDir for selected pages.
func ExtractContent(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, conf *model.Configuration) error {
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

	fileName = strings.TrimSuffix(filepath.Base(fileName), ".pdf")

	for p, v := range pages {
		if !v {
			continue
		}

		r, err := pdfcpu.ExtractPageContent(ctx, p)
		if err != nil {
			return err
		}
		if r == nil {
			continue
		}

		outFile := filepath.Join(outDir, fmt.Sprintf("%s_Content_page_%d.txt", fileName, p))
		logWritingTo(outFile)
		f, err := os.Create(outFile)
		if err != nil {
			return err
		}

		if _, err = io.Copy(f, r); err != nil {
			return err
		}

		if err := f.Close(); err != nil {
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

	return ExtractContent(f, outDir, inFile, selectedPages, conf)
}

// ExtractMetadata dumps all metadata dict entries for rs into outDir.
func ExtractMetadata(rs io.ReadSeeker, outDir, fileName string, conf *model.Configuration) error {
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

	mm, err := pdfcpu.ExtractMetadata(ctx)
	if err != nil {
		return err
	}

	if len(mm) > 0 {
		fileName = strings.TrimSuffix(filepath.Base(fileName), ".pdf")
		for _, m := range mm {
			outFile := filepath.Join(outDir, fmt.Sprintf("%s_Metadata_%s_%d_%d.txt", fileName, m.ParentType, m.ParentObjNr, m.ObjNr))
			logWritingTo(outFile)
			f, err := os.Create(outFile)
			if err != nil {
				return err
			}
			if _, err = io.Copy(f, m); err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
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

	return ExtractMetadata(f, outDir, filepath.Base(inFile), conf)
}
