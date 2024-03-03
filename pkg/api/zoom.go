/*
Copyright 2024 The pdfcpu Authors.

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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// Zoom applies resizeConf for selected pages of rs and writes result to w.
func Zoom(rs io.ReadSeeker, w io.Writer, selectedPages []string, zoom *model.Zoom, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: Zoom: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ZOOM

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	if err = pdfcpu.Zoom(ctx, pages, zoom); err != nil {
		return err
	}

	return Write(ctx, w, conf)
}

// ZoomFile applies zoomConf for selected pages of inFile and writes result to outFile.
func ZoomFile(inFile, outFile string, selectedPages []string, zoom *model.Zoom, conf *model.Configuration) (err error) {
	if log.CLIEnabled() {
		log.CLI.Printf("zooming %s\n", inFile)
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}

	var (
		f1, f2 *os.File
	)

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

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ZOOM

	return Zoom(f1, f2, selectedPages, zoom, conf)
}
