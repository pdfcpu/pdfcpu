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
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Validate validates a PDF stream read from rs.
func Validate(rs io.ReadSeeker, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.VALIDATE

	from1 := time.Now()

	ctx, err := ReadContext(rs, conf)
	if err != nil {
		return err
	}

	dur1 := time.Since(from1).Seconds()
	from2 := time.Now()

	if err = ValidateContext(ctx); err != nil {
		s := ""
		if conf.ValidationMode == model.ValidationStrict {
			s = " (try --mode=relaxed)"
		}
		err = fmt.Errorf("%s: %w", fmt.Sprintf("validation error (obj#:%d)%s", ctx.CurObj, s), err)
	}

	if err == nil {
		if conf.Optimize {
			if log.CLIEnabled() {
				log.CLI.Println("optimizing...")
			}
			err = pdfcpu.OptimizeXRefTable(ctx)
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

// ValidateFile validates inFile.
func ValidateFile(inFile string, conf *model.Configuration) error {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}

	if log.CLIEnabled() {
		log.CLI.Printf("validating(mode=%s) %s ...\n", conf.ValidationModeString(), inFile)
	}

	f, err := os.Open(inFile)
	if err != nil {
		return err
	}

	defer f.Close()

	if err = Validate(f, conf); err != nil {
		return err
	}

	if log.CLIEnabled() {
		log.CLI.Println("validation ok")
	}

	return nil
}

// ValidateFiles validates inFiles.
func ValidateFiles(inFiles []string, conf *model.Configuration) error {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}

	var errs []error
	for i, fn := range inFiles {
		if i > 0 && log.CLIEnabled() {
			log.CLI.Println()
		}
		if err := ValidateFile(fn, conf); err != nil {
			if len(inFiles) == 1 {
				return err
			}
			errs = append(errs, fmt.Errorf("%s: %w", fn, err))
		}
	}

	return errors.Join(errs...)
}

// DumpObject writes an object from rs to stdout.
func DumpObject(rs io.ReadSeeker, mode, objNr int, conf *model.Configuration) error {
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.DUMP

	ctx, err := ReadContext(rs, conf)
	if err != nil {
		return err
	}

	if err = ValidateContext(ctx); err != nil {
		s := ""
		if conf.ValidationMode == model.ValidationStrict {
			s = " (try --mode=relaxed)"
		}
		return fmt.Errorf("%s: %w", fmt.Sprintf("validation error (obj#:%d)%s", ctx.CurObj, s), err)
	}

	ctx.DumpObject(objNr, mode)

	return err
}

// DumpObjectFile writes an object from rs to stdout.
func DumpObjectFile(inFile string, mode, objNr int, conf *model.Configuration) error {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}

	f, err := os.Open(inFile)
	if err != nil {
		return err
	}

	defer f.Close()

	return DumpObject(f, mode, objNr, conf)
}
