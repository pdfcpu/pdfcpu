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
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Import parses an Import command string into an internal structure.
func Import(s string, u types.DisplayUnit) (*pdfcpu.Import, error) {
	return pdfcpu.ParseImportDetails(s, u)
}

// ImportImages appends PDF pages containing images to rs and writes the result to w.
// If rs == nil a new PDF file will be written to w.
func ImportImages(rs io.ReadSeeker, w io.Writer, imgs []io.Reader, imp *pdfcpu.Import, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.IMPORTIMAGES

	if imp == nil {
		imp = pdfcpu.DefaultImportConfig()
	}

	var ctx *model.Context

	if rs != nil {
		ctx, err = ReadAndValidate(rs, conf)
	} else {
		ctx, err = pdfcpu.CreateContextWithXRefTable(conf, imp.PageDim)
	}
	if err != nil {
		return err
	}

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		return err
	}

	// Page tree root.
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		return err
	}

	for _, r := range imgs {

		indRefs, err := pdfcpu.NewPagesForImage(ctx.XRefTable, r, pagesIndRef, imp)
		if err != nil {
			return err
		}

		for _, indRef := range indRefs {
			if err := ctx.SetValid(*indRef); err != nil {
				return err
			}
			if err = model.AppendPageTree(indRef, 1, pagesDict); err != nil {
				return err
			}
			ctx.PageCount++
		}
	}

	return Write(ctx, w, conf)
}

func fileExists(filename string) bool {
	var ret bool
	f, err := os.Open(filename)
	if err == nil {
		ret = true
	}
	defer f.Close()
	return ret

}

func prepImgFiles(imgFiles []string, f1 *os.File) ([]io.ReadCloser, []io.Reader, error) {
	rc := make([]io.ReadCloser, len(imgFiles))
	rr := make([]io.Reader, len(imgFiles))

	for i, fn := range imgFiles {
		f, err := os.Open(fn)
		if err != nil {
			if f1 != nil {
				f1.Close()
			}
			return nil, nil, err
		}
		rc[i] = f
		rr[i] = bufio.NewReader(f)
	}

	return rc, rr, nil
}

func logImportImages(s, outFile string) {
	if log.CLIEnabled() {
		log.CLI.Printf("%s to %s...\n", s, outFile)
	}
}

func importImagesInputFile(outFile string) (io.ReadSeeker, *os.File, string, error) {
	rs := io.ReadSeeker(nil)
	tmpFile := outFile

	if fileExists(outFile) {
		f, err := os.Open(outFile)
		if err != nil {
			return nil, nil, "", err
		}
		rs = f
		logImportImages("appending", outFile)
		return rs, f, tmpFile, nil
	}

	logImportImages("writing", outFile)
	return rs, nil, tmpFile, nil
}

func closeImportImageFiles(rc []io.ReadCloser) error {
	for _, f := range rc {
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

func finishImportImagesFile(ok bool, f1, f2 *os.File, rc []io.ReadCloser, tmpFile, outFile string) error {
	if !ok {
		_ = f2.Close()
		if f1 != nil {
			_ = f1.Close()
			os.Remove(tmpFile)
		}
		for _, f := range rc {
			_ = f.Close()
		}
		return nil
	}

	if err := f2.Close(); err != nil {
		return err
	}
	if f1 != nil {
		if err := f1.Close(); err != nil {
			return err
		}
		if err := os.Rename(tmpFile, outFile); err != nil {
			return err
		}
	}
	return closeImportImageFiles(rc)
}

// ImportImagesFile appends PDF pages containing images to outFile which will be created if necessary.
func ImportImagesFile(imgFiles []string, outFile string, imp *pdfcpu.Import, conf *model.Configuration) (err error) {
	ok := false

	rs, f1, tmpFile, err := importImagesInputFile(outFile)
	if err != nil {
		return err
	}

	rc, rr, err := prepImgFiles(imgFiles, f1)
	if err != nil {
		return err
	}

	inFile := ""
	if f1 != nil {
		inFile = outFile
	}
	f2, tmpFile, err := createOutputFile(inFile, tmpFile)
	if err != nil {
		if f1 != nil {
			f1.Close()
		}
		return err
	}

	defer func() {
		if e := finishImportImagesFile(ok, f1, f2, rc, tmpFile, outFile); e != nil {
			err = e
		}
	}()

	if err = ImportImages(rs, f2, rr, imp, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
