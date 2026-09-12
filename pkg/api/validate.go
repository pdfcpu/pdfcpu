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
	"time"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func validationModeHint(mode int) string {
	if mode != model.ValidationStrict {
		return ""
	}
	return " (try --mode=relaxed)"
}

func validationError(conf *model.Configuration, err error) error {
	prefix := "validation error"
	var validationErr *model.ValidationError
	if errors.As(err, &validationErr) {
		prefix += fmt.Sprintf(" (obj#:%d)", validationErr.ObjectNumber())
	}
	return fmt.Errorf("%s%s: %w", prefix, validationModeHint(conf.ValidationMode), err)
}

func validateWithOptions(c context.Context, rs io.ReadSeeker, conf *model.Configuration, options ProgressOptions, reportReading bool) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.VALIDATE)

	from1 := time.Now()

	if reportReading {
		if err := reportProgress(options, ProgressStageReading); err != nil {
			return err
		}
	}

	ctx, err := ReadContext(c, rs, conf)
	if err != nil {
		return fmt.Errorf("read context: %w", err)
	}

	dur1 := time.Since(from1).Seconds()
	from2 := time.Now()

	if err := reportProgress(options, ProgressStageValidating); err != nil {
		return err
	}

	if err = ValidateContext(c, ctx); err != nil {
		err = validationError(conf, err)
	}

	if err == nil && conf.Optimize {
		if err := reportProgress(options, ProgressStageOptimizing); err != nil {
			return err
		}
		if err = OptimizeContext(c, ctx); err != nil {
			return err
		}
	}

	dur2 := time.Since(from2).Seconds()
	dur := time.Since(from1).Seconds()

	if log.StatsEnabled() {
		log.Stats.Printf("XRefTable:\n%s\n", ctx)
	}

	model.ValidationTimingStats(dur1, dur2, dur)

	// at this stage: no binary breakup available!
	if ctx.Read.FileSize > 0 {
		ctx.Read.LogStats(ctx.Optimized)
	}

	return err
}

// Validate validates a PDF stream, supports cancellation and reports optional semantic progress.
// A nil options pointer disables progress reporting.
func Validate(c context.Context, rs io.ReadSeeker, conf *model.Configuration, options *ProgressOptions) error {
	return validateWithOptions(c, rs, conf, progressOptionsValue(options), true)
}

// ValidateFile validates inFile, supports cancellation and reports optional semantic progress.
// A nil options pointer disables progress reporting. Supplied options are not modified.
func ValidateFile(c context.Context, inFile string, conf *model.Configuration, options *ProgressOptions) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}

	inputOptions := progressOptionsValue(options)
	inputOptions.Input = inFile
	if err := reportProgress(inputOptions, ProgressStageReading); err != nil {
		return err
	}

	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("validate: open %s: %w", inFile, err)
	}

	defer func() {
		closeErr := f.Close()
		if err != nil {
			return
		}
		if closeErr != nil {
			err = fmt.Errorf("validate: close %s: %w", inFile, closeErr)
		}
	}()

	if err = validateWithOptions(c, f, conf, inputOptions, false); err != nil {
		return fmt.Errorf("validate %s: %w", inFile, err)
	}

	return nil
}

// ValidateFiles validates inFiles, supports cancellation and reports optional semantic progress.
// A nil options pointer disables progress reporting. Supplied options are not modified.
func ValidateFiles(c context.Context, inFiles []string, conf *model.Configuration, options *ProgressOptions) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}

	var errs []error
	for i, fn := range inFiles {
		inputOptions := progressOptionsValue(options)
		inputOptions.Input = fn
		inputOptions.Item = i + 1
		inputOptions.Total = len(inFiles)
		if err := ValidateFile(c, fn, conf, &inputOptions); err != nil {
			var progressErr *ProgressError
			if errors.As(err, &progressErr) {
				return err
			}
			if len(inFiles) == 1 {
				return err
			}
			errs = append(errs, fmt.Errorf("%s: %w", fn, err))
		}
	}

	return errors.Join(errs...)
}
