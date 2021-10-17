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
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// Validate validates a PDF stream read from rs.
func Validate(rs io.ReadSeeker, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.VALIDATE

	if conf.ValidationMode == pdfcpu.ValidationNone {
		return errors.New("pdfcpu: validate: mode ValidationNone not allowed")
	}

	from1 := time.Now()

	ctx, err := ReadContext(rs, conf)
	if err != nil {
		return err
	}

	dur1 := time.Since(from1).Seconds()
	from2 := time.Now()

	if err = ValidateContext(ctx); err != nil {
		s := ""
		if conf.ValidationMode == pdfcpu.ValidationStrict {
			s = " (try -mode=relaxed)"
		}
		err = errors.Wrap(err, fmt.Sprintf("validation error (obj#:%d)%s", ctx.CurObj, s))
	}

	dur2 := time.Since(from2).Seconds()
	dur := time.Since(from1).Seconds()

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdfcpu.ValidationTimingStats(dur1, dur2, dur)

	// at this stage: no binary breakup available!
	if ctx.Read.FileSize > 0 {
		ctx.Read.LogStats(ctx.Optimized)
	}

	return err
}

// ValidateFile validates inFile.
func ValidateFile(inFile string, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}

	if conf != nil && conf.ValidationMode == pdfcpu.ValidationNone {
		return nil
	}

	log.CLI.Printf("validating(mode=%s) %s ...\n", conf.ValidationModeString(), inFile)

	f, err := os.Open(inFile)
	if err != nil {
		return err
	}

	defer f.Close()

	if err = Validate(f, conf); err != nil {
		return err
	}

	log.CLI.Println("validation ok")

	return nil
}

// ValidateFiles validates inFiles.
func ValidateFiles(inFiles []string, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}

	if conf != nil && conf.ValidationMode == pdfcpu.ValidationNone {
		return nil
	}

	for i, fn := range inFiles {
		if i > 0 {
			log.CLI.Println()
		}
		if err := ValidateFile(fn, conf); err != nil {
			if len(inFiles) == 1 {
				return err
			}
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}

	return nil
}
