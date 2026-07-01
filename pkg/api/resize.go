/*
Copyright 2023 The pdfcpu Authors.

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

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// Resize applies resizeConf for selected pages of rs and writes result to w.
func Resize(rs io.ReadSeeker, w io.Writer, selectedPages []string, resize *model.Resize, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: Resize: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.RESIZE

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	if err = pdfcpu.Resize(ctx, pages, resize); err != nil {
		return err
	}

	return Write(ctx, w, conf)
}

// ResizeFile applies resizeConf for selected pages of inFile and writes result to outFile.
func ResizeFile(inFile, outFile string, selectedPages []string, resize *model.Resize, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if log.CLIEnabled() {
		log.CLI.Printf("resizing %s\n", inFile)
	}

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

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.RESIZE

	if err = Resize(f1, f2, selectedPages, resize, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
