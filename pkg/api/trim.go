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
	"io"
	"os"
	"sort"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// Trim generates a trimmed version of rs
// containing all selected pages and writes the result to w.
func Trim(rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: Trim: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.TRIM

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, false, true)
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		if log.CLIEnabled() {
			log.CLI.Println("aborted: missing page numbers!")
		}
		return nil
	}

	var pageNrs []int
	for k, v := range pages {
		if v {
			pageNrs = append(pageNrs, k)
		}
	}
	sort.Ints(pageNrs)

	ctxDest, err := pdfcpu.ExtractPages(ctx, pageNrs, false)
	if err != nil {
		return err
	}

	if conf.PostProcessValidate {
		if err = ValidateContext(ctxDest); err != nil {
			return err
		}
	}

	return WriteContext(ctxDest, w)
}

// TrimFile generates a trimmed version of inFile
// containing all selected pages and writes the result to outFile.
func TrimFile(inFile, outFile string, selectedPages []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	if err = Trim(f1, f2, selectedPages, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
