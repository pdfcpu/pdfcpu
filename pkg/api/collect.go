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
)

// Collect creates a custom PDF page sequence for selected pages of rs and writes the result to w.
func Collect(rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.COLLECT

	fromStart := time.Now()
	ctx, _, _, _, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	pages, err := PagesForPageCollection(ctx.PageCount, selectedPages)
	if err != nil {
		return err
	}

	ctxDest, err := ctx.ExtractPages(pages, true)
	if err != nil {
		return err
	}

	if conf.ValidationMode != pdfcpu.ValidationNone {
		if err = ValidateContext(ctxDest); err != nil {
			return err
		}
	}

	return WriteContext(ctxDest, w)
}

// CollectFile creates a custom PDF page sequence for inFile and writes the result to outFile.
func CollectFile(inFile, outFile string, selectedPages []string, conf *pdfcpu.Configuration) (err error) {
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

	return Collect(f1, f2, selectedPages, conf)
}
