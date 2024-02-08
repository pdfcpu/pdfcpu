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
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// Trim generates a trimmed version of rs
// containing all selected pages and writes the result to w.
func Trim(rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: Trim: missing rs")
	}

	var err error
	if conf == nil {
		conf, err = model.NewDefaultConfiguration()
		if err != nil {
			return err
		}
	}
	conf.Cmd = model.TRIM

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := ReadValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	fromWrite := time.Now()

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, false, true)
	if err != nil {
		return err
	}

	// No special context processing required.
	// WriteContext decides which pages get written by checking conf.Cmd

	ctx.Write.SelectedPages = pages
	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "trim, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// TrimFile generates a trimmed version of inFile
// containing all selected pages and writes the result to outFile.
func TrimFile(inFile, outFile string, selectedPages []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
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

	return Trim(f1, f2, selectedPages, conf)
}
