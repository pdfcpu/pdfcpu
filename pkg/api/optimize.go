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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func optimize(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) error {
	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	if log.StatsEnabled() {
		log.Stats.Printf("XRefTable:\n%s\n", ctx)
	}

	if err := WriteContext(ctx, w); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	if ctx.StatsFileName != "" {
		if err := pdfcpu.AppendStatsFile(ctx); err != nil {
			return fmt.Errorf("write stats: %w", err)
		}
	}

	return nil
}

// Optimize reads a PDF stream from rs and writes the optimized PDF stream to w.
// noEncryption ensures w writes without encryption.
func Optimize(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.OPTIMIZE

	if err := optimize(rs, w, conf); err != nil {
		return fmt.Errorf("optimize: %w", err)
	}
	return nil
}

// OptimizeFile reads inFile and writes the optimized PDF to outFile.
// If outFile is not provided then inFile gets overwritten
// which leads to the same result as when inFile equals outFile.
// noEncryption ensures outFile is not encrypted.
func OptimizeFile(inFile, outFile string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("optimize: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}

	staged, err := openStagedOutput(f1, inFile, tmpFile, "optimize")
	if err != nil {
		return errors.Join(
			fmt.Errorf("optimize: create output: %w", err),
			closeFile(f1, "optimize: close input"),
		)
	}
	f2 = staged.output.file

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.OPTIMIZE

	if err = Optimize(f1, f2, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
