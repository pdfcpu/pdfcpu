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

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// Rotate rotates selected pages of rs clockwise by rotation degrees and writes the result to w.
func Rotate(rs io.ReadSeeker, w io.Writer, rotation int, selectedPages []string, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: Rotate: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ROTATE

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := ReadValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	from := time.Now()
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	if err = pdfcpu.RotatePages(ctx, pages, rotation); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	durStamp := time.Since(from).Seconds()
	fromWrite := time.Now()

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := durStamp + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "rotate, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// RotateFile rotates selected pages of inFile clockwise by rotation degrees and writes the result to outFile.
func RotateFile(inFile, outFile string, rotation int, selectedPages []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
			}
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

	return Rotate(f1, f2, rotation, selectedPages, conf)
}
