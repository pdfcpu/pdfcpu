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

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// Optimize reads a PDF stream from rs and writes the optimized PDF stream to w.
func Optimize(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: Optimize: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	if log.StatsEnabled() {
		log.Stats.Printf("XRefTable:\n%s\n", ctx)
	}

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	// For Optimize only.
	if ctx.StatsFileName != "" {
		err = pdfcpu.AppendStatsFile(ctx)
		if err != nil {
			return errors.Wrap(err, "Write stats failed.")
		}
	}

	return nil
}

// OptimizeFile reads inFile and writes the optimized PDF to outFile.
// If outFile is not provided then inFile gets overwritten
// which leads to the same result as when inFile equals outFile.
func OptimizeFile(inFile, outFile string, conf *model.Configuration) (err error) {
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
	conf.Cmd = model.OPTIMIZE

	return Optimize(f1, f2, conf)
}
