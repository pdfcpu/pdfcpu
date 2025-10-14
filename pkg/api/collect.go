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

	"github.com/mechiko/pdfcpu/pkg/pdfcpu"
	"github.com/mechiko/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// Collect creates a custom PDF page sequence for selected pages of rs and writes the result to w.
func Collect(rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: Collect: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.COLLECT

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	pages, err := PagesForPageCollection(ctx.PageCount, selectedPages)
	if err != nil {
		return err
	}

	ctxDest, err := pdfcpu.ExtractPages(ctx, pages, false)
	if err != nil {
		return err
	}

	return Write(ctxDest, w, conf)
}

// CollectFile creates a custom PDF page sequence for inFile and writes the result to outFile.
func CollectFile(inFile, outFile string, selectedPages []string, conf *model.Configuration) (err error) {
	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}

	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
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

	return Collect(f1, f2, selectedPages, conf)
}
