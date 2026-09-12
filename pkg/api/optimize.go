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
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func optimize(c context.Context, rs io.ReadSeeker, w io.Writer, conf *model.Configuration, options ProgressOptions) error {
	ctx, err := ReadValidateAndOptimize(c, rs, conf, &options)
	if err != nil {
		return err
	}

	if log.StatsEnabled() {
		log.Stats.Printf("XRefTable:\n%s\n", ctx)
	}

	if err := reportProgress(options, ProgressStageWriting); err != nil {
		return err
	}
	if err := contextutil.Check(c); err != nil {
		return err
	}

	if err := WriteContext(c, ctx, w); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if err := contextutil.Check(c); err != nil {
		return err
	}

	if ctx.StatsFileName != "" {
		if err := pdfcpu.AppendStatsFile(ctx); err != nil {
			return fmt.Errorf("write stats: %w", err)
		}
	}

	return nil
}

// Optimize reads and optimizes a PDF stream, supports cancellation and reports optional semantic progress.
// A nil options pointer disables progress reporting. Supplied options are not modified.
func Optimize(c context.Context, rs io.ReadSeeker, w io.Writer, conf *model.Configuration, options *ProgressOptions) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	conf = operationConfiguration(conf, model.OPTIMIZE)

	if err := optimize(c, rs, w, conf, progressOptionsValue(options)); err != nil {
		return fmt.Errorf("optimize: %w", err)
	}
	return nil
}

// OptimizeFile reads inFile, writes the optimized PDF to outFile and supports cancellation and optional progress reporting.
// An empty outFile or one equal to inFile replaces inFile.
// A nil options pointer disables progress reporting. Supplied options are not modified.
func OptimizeFile(c context.Context, inFile, outFile string, conf *model.Configuration, options *ProgressOptions) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("optimize: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}

	staged, err := openStagedOutput(f1, inFile, tmpFile, "optimize")
	if err != nil {
		return errors.Join(
			fmt.Errorf("optimize: create output: %w", err),
			closeFile(f1, "optimize: close input"),
		)
	}
	f2 = staged.output.file
	inputOptions := progressOptionsValue(options)
	inputOptions.Input = inFile

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = Optimize(c, f1, f2, conf, &inputOptions); err != nil {
		return err
	}

	if err = reportProgress(inputOptions, ProgressStageCommitting); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}
