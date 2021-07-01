/*
	Copyright 2021 The pdfcpu Authors.

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

	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// AddBookmarks adds a single bookmark outline layer to the PDF context read from rs and writes the result to w.
func AddBookmarks(rs io.ReadSeeker, w io.Writer, bms []pdf.Bookmark, conf *pdf.Configuration) error {

	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = pdf.ValidationRelaxed
	}
	conf.Cmd = pdf.ADDBOOKMARKS

	if len(bms) == 0 {
		return errors.New("pdfcpu: AddBookmarks: Please supply m")
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	from := time.Now()

	if err := ctx.AddBookmarks(bms); err != nil {
		return err
	}

	durAdd := time.Since(from).Seconds()
	fromWrite := time.Now()

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := durAdd + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "add bookmarks, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// AddBookmarksFile adds a single bookmark outline layer to the PDF context read from inFile and writes the result to outFile.
func AddBookmarksFile(inFile, outFile string, bms []pdf.Bookmark, conf *pdf.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	if f2, err = os.Create(tmpFile); err != nil {
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
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return AddBookmarks(f1, f2, bms, conf)
}
