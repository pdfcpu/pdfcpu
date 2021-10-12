/*
	Copyright 2018 The pdfcpu Authors.

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

// Package api lets you integrate pdfcpu's operations into your Go backend.
//
// There are two api layers supporting all pdfcpu operations:
//  1) The file based layer (used by pdfcpu's cli)
//  2) The io.ReadSeeker/io.Writer based layer for backend integration.
//
// For any pdfcpu command there are two functions.
//
// The file based function always calls the io.ReadSeeker/io.Writer based function:
//  func CommandFile(inFile, outFile string, conf *pdf.Configuration) error
//  func Command(rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error
//
// eg. for optimization:
//  func OptimizeFile(inFile, outFile string, conf *pdf.Configuration) error
//  func Optimize(rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error
package api

import (
	"bufio"
	"io"
	"os"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/validate"
)

// ReadContext uses an io.ReadSeeker to build an internal structure holding its cross reference table aka the Context.
func ReadContext(rs io.ReadSeeker, conf *pdfcpu.Configuration) (*pdfcpu.Context, error) {
	return pdfcpu.Read(rs, conf)
}

// ReadContextFile returns inFile's validated context.
func ReadContextFile(inFile string) (*pdfcpu.Context, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ctx, err := ReadContext(f, pdfcpu.NewDefaultConfiguration())
	if err != nil {
		return nil, err
	}
	if err = validate.XRefTable(ctx.XRefTable); err != nil {
		return nil, err
	}
	return ctx, err
}

// ValidateContext validates ctx.
func ValidateContext(ctx *pdfcpu.Context) error {
	return validate.XRefTable(ctx.XRefTable)
}

// OptimizeContext optimizes ctx.
func OptimizeContext(ctx *pdfcpu.Context) error {
	return pdfcpu.OptimizeXRefTable(ctx)
}

// WriteContext writes ctx to w.
func WriteContext(ctx *pdfcpu.Context, w io.Writer) error {
	if f, ok := w.(*os.File); ok {
		// In order to retrieve the written file size.
		ctx.Write.Fp = f
	}
	ctx.Write.Writer = bufio.NewWriter(w)
	defer ctx.Write.Flush()
	return pdfcpu.Write(ctx)
}

// WriteIncrement writes a PDF increment for ctx to w.
func WriteIncrement(ctx *pdfcpu.Context, w io.Writer) error {
	ctx.Write.Writer = bufio.NewWriter(w)
	defer ctx.Write.Flush()
	return pdfcpu.WriteIncrement(ctx)
}

// WriteContextFile writes ctx to outFile.
func WriteContextFile(ctx *pdfcpu.Context, outFile string) error {
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteContext(ctx, f)
}

func readAndValidate(rs io.ReadSeeker, conf *pdfcpu.Configuration, from1 time.Time) (ctx *pdfcpu.Context, dur1, dur2 float64, err error) {
	if ctx, err = ReadContext(rs, conf); err != nil {
		return nil, 0, 0, err
	}

	dur1 = time.Since(from1).Seconds()

	if conf.ValidationMode == pdfcpu.ValidationNone {
		// Bypass validation
		return ctx, 0, 0, nil
	}

	from2 := time.Now()

	if err = validate.XRefTable(ctx.XRefTable); err != nil {
		return nil, 0, 0, err
	}

	dur2 = time.Since(from2).Seconds()

	return ctx, dur1, dur2, nil
}

func readValidateAndOptimize(rs io.ReadSeeker, conf *pdfcpu.Configuration, from1 time.Time) (ctx *pdfcpu.Context, dur1, dur2, dur3 float64, err error) {
	ctx, dur1, dur2, err = readAndValidate(rs, conf, from1)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	from3 := time.Now()
	if err = OptimizeContext(ctx); err != nil {
		return nil, 0, 0, 0, err
	}

	dur3 = time.Since(from3).Seconds()

	return ctx, dur1, dur2, dur3, nil
}

func logOperationStats(ctx *pdfcpu.Context, op string, durRead, durVal, durOpt, durWrite, durTotal float64) {
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdfcpu.TimingStats(op, durRead, durVal, durOpt, durWrite, durTotal)
	if ctx.Read.FileSize > 0 {
		ctx.Read.LogStats(ctx.Optimized)
		ctx.Write.LogStats()
	}
}

// EnsureDefaultConfigAt switches to the pdfcpu config dir located at path.
// If path/pdfcpu is not existent, it will be created including config.yml
func EnsureDefaultConfigAt(path string) error {
	// Call if you have specific requirements regarding the location of the pdfcpu config dir.
	return pdfcpu.EnsureDefaultConfigAt(path)
}

// DisableConfigDir disables the configuration directory.
// Any needed default configuration will be loaded from configuration.go
// Since the config dir also contains the user font dir, this also limits font usage to the default core font set
// No user fonts will be available.
func DisableConfigDir() {
	// Call if you don't want to use a specific configuration
	// and also do not need to use user fonts.
	pdfcpu.ConfigPath = "disable"
}

// LoadConfiguration locates and loads the default configuration
// and also loads installed user fonts.
func LoadConfiguration() *pdfcpu.Configuration {
	// Call if you don't have a specific config dir location
	// and need to use user fonts for stamping or watermarking.
	return pdfcpu.NewDefaultConfiguration()
}
