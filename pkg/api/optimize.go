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
	"github.com/pkg/errors"
)

// Optimize reads a PDF stream from rs and writes the optimized PDF stream to w.
func Optimize(rs io.ReadSeeker, w io.Writer, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
		conf.Cmd = pdfcpu.OPTIMIZE
	}

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	fromWrite := time.Now()

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "write", durRead, durVal, durOpt, durWrite, durTotal)

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
func OptimizeFile(inFile, outFile string, conf *pdfcpu.Configuration) (err error) {
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

	return Optimize(f1, f2, conf)
}
