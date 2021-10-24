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
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// Import parses an Import command string into an internal structure.
func Import(s string, u pdfcpu.DisplayUnit) (*pdfcpu.Import, error) {
	return pdfcpu.ParseImportDetails(s, u)
}

// ImportImages appends PDF pages containing images to rs and writes the result to w.
// If rs == nil a new PDF file will be written to w.
func ImportImages(rs io.ReadSeeker, w io.Writer, imgs []io.Reader, imp *pdfcpu.Import, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.IMPORTIMAGES

	if imp == nil {
		imp = pdfcpu.DefaultImportConfig()
	}

	var (
		ctx *pdfcpu.Context
		err error
	)

	if rs != nil {
		ctx, _, _, err = readAndValidate(rs, conf, time.Now())
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

	// This is the page tree root.
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		return err
	}

	for _, r := range imgs {

		indRef, err := pdfcpu.NewPageForImage(ctx.XRefTable, r, pagesIndRef, imp)
		if err != nil {
			return err
		}

		if err = pdfcpu.AppendPageTree(indRef, 1, pagesDict); err != nil {
			return err
		}

		ctx.PageCount++
	}

	if conf.ValidationMode != pdfcpu.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	return nil
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

// ImportImagesFile appends PDF pages containing images to outFile which will be created if necessary.
func ImportImagesFile(imgFiles []string, outFile string, imp *pdfcpu.Import, conf *pdfcpu.Configuration) (err error) {
	var f1, f2 *os.File

	rs := io.ReadSeeker(nil)
	f1 = nil
	tmpFile := outFile
	if fileExists(outFile) {
		if f1, err = os.Open(outFile); err != nil {
			return err
		}
		rs = f1
		tmpFile += ".tmp"
		log.CLI.Printf("appending to %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", outFile)
	}

	rc := make([]io.ReadCloser, len(imgFiles))
	rr := make([]io.Reader, len(imgFiles))
	for i, fn := range imgFiles {
		f, err := os.Open(fn)
		if err != nil {
			return err
		}
		rc[i] = f
		rr[i] = bufio.NewReader(f)
	}

	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			if f1 != nil {
				f1.Close()
				os.Remove(tmpFile)
			}
			for _, f := range rc {
				f.Close()
			}
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if f1 != nil {
			if err = f1.Close(); err != nil {
				return
			}
			if err = os.Rename(tmpFile, outFile); err != nil {
				return
			}
		}
		for _, f := range rc {
			if err := f.Close(); err != nil {
				return
			}
		}
	}()

	return ImportImages(rs, f2, rr, imp, conf)
}
