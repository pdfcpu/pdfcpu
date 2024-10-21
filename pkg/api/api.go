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
//  1. The file based layer (used by pdfcpu's cli)
//  2. The io.ReadSeeker/io.Writer based layer for backend integration.
//
// For any pdfcpu command there are two functions.
//
// The file based function always calls the io.ReadSeeker/io.Writer based function:
//
//	func CommandFile(inFile, outFile string, conf *pdf.Configuration) error
//	func Command(rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error
//
// eg. for optimization:
//
//	func OptimizeFile(inFile, outFile string, conf *pdf.Configuration) error
//	func Optimize(rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error
package api

import (
	"bufio"
	"io"
	"os"
	"sync"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/validate"
	"github.com/pkg/errors"
)

func logDisclaimerPDF20() {
	disclaimer := `
***************************** Disclaimer ****************************
* PDF 2.0 features are supported on a need basis.                   *
* (See ISO 32000:2 6.3.2 Conformance of PDF processors)             *
* At the moment pdfcpu comes with basic PDF 2.0 support.            *
* Please let us know which feature you would like to see supported, *
* provide a sample PDF file and create an issue:                    *
* https://github.com/pdfcpu/pdfcpu/issues/new/choose                *
* Thank you for using pdfcpu <3                                     *
*********************************************************************`

	if log.ValidateEnabled() {
		log.Validate.Println(disclaimer)
	}
	if log.CLIEnabled() {
		log.CLI.Println(disclaimer)
	}
}

// ReadContext uses an io.ReadSeeker to build an internal structure holding its cross reference table aka the Context.
func ReadContext(rs io.ReadSeeker, conf *model.Configuration) (*model.Context, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: ReadContext: missing rs")
	}
	return pdfcpu.Read(rs, conf)
}

// ReadContextFile returns inFile's validated context.
func ReadContextFile(inFile string) (*model.Context, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ctx, err := ReadContext(f, model.NewDefaultConfiguration())
	if err != nil {
		return nil, err
	}

	if ctx.Conf.Version != model.VersionStr {
		model.CheckConfigVersion(ctx.Conf.Version)
	}

	if ctx.XRefTable.Version() == model.V20 {
		logDisclaimerPDF20()
	}

	if err = validate.XRefTable(ctx); err != nil {
		return nil, err
	}

	return ctx, err
}

// ValidateContext validates ctx.
func ValidateContext(ctx *model.Context) error {
	if ctx.XRefTable.Version() == model.V20 {
		logDisclaimerPDF20()
	}
	return validate.XRefTable(ctx)
}

// OptimizeContext optimizes ctx.
func OptimizeContext(ctx *model.Context) error {
	if log.CLIEnabled() {
		log.CLI.Println("optimizing...")
	}
	return pdfcpu.OptimizeXRefTable(ctx)
}

// WriteContext writes ctx to w.
func WriteContext(ctx *model.Context, w io.Writer) error {
	if f, ok := w.(*os.File); ok {
		// In order to retrieve the written file size.
		ctx.Write.Fp = f
	}
	ctx.Write.Writer = bufio.NewWriter(w)
	defer ctx.Write.Flush()
	return pdfcpu.Write(ctx)
}

// WriteIncrement writes a PDF increment for ctx to w.
func WriteIncrement(ctx *model.Context, w io.Writer) error {
	ctx.Write.Writer = bufio.NewWriter(w)
	defer ctx.Write.Flush()
	return pdfcpu.WriteIncrement(ctx)
}

// WriteContextFile writes ctx to outFile.
func WriteContextFile(ctx *model.Context, outFile string) error {
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteContext(ctx, f)
}

// ReadAndValidate returns a model.Context of rs ready for processing.
func ReadAndValidate(rs io.ReadSeeker, conf *model.Configuration) (ctx *model.Context, err error) {
	if ctx, err = ReadContext(rs, conf); err != nil {
		return nil, err
	}

	if err := ValidateContext(ctx); err != nil {
		return nil, err
	}

	return ctx, nil
}

func cmdAssumingOptimization(cmd model.CommandMode) bool {
	return cmd == model.OPTIMIZE ||
		cmd == model.FILLFORMFIELDS ||
		cmd == model.RESETFORMFIELDS ||
		cmd == model.LISTIMAGES ||
		cmd == model.EXTRACTIMAGES ||
		cmd == model.EXTRACTFONTS
}

// ReadValidateAndOptimize returns an optimized model.Context of rs ready for processing a specific command.
// conf.Cmd is expected to be configured properly.
func ReadValidateAndOptimize(rs io.ReadSeeker, conf *model.Configuration) (ctx *model.Context, err error) {
	if conf == nil {
		return nil, errors.New("pdfcpu: ReadValidateAndOptimize: missing conf")
	}

	ctx, err = ReadAndValidate(rs, conf)
	if err != nil {
		return nil, err
	}

	// With the exception of commands utilizing structs provided the Optimize step
	// command optimization of the cross reference table is optional but usually recommended.
	// For large or complex files it may make sense to skip optimization and set conf.Optimize = false.
	if cmdAssumingOptimization(conf.Cmd) || conf.Optimize {
		if err = OptimizeContext(ctx); err != nil {
			return nil, err
		}
	}

	// TODO move to form related commands.
	if err := pdfcpu.CacheFormFonts(ctx); err != nil {
		return nil, err
	}

	return ctx, nil
}

func logWritingTo(s string) {
	if log.CLIEnabled() {
		log.CLI.Printf("writing %s...\n", s)
	}
}

func Write(ctx *model.Context, w io.Writer, conf *model.Configuration) error {
	if log.StatsEnabled() {
		log.Stats.Printf("XRefTable:\n%s\n", ctx)
	}

	// Note side effects of validation before writing!
	// if conf.PostProcessValidate {
	// 	if err := ValidateContext(ctx); err != nil {
	// 		return err
	// 	}
	// }

	return WriteContext(ctx, w)
}

func WriteIncr(ctx *model.Context, rws io.ReadWriteSeeker, conf *model.Configuration) error {
	if log.StatsEnabled() {
		log.Stats.Printf("XRefTable:\n%s\n", ctx)
	}

	if conf.PostProcessValidate {
		if err := ValidateContext(ctx); err != nil {
			return err
		}
	}

	if _, err := rws.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	return WriteIncrement(ctx, rws)
}

// EnsureDefaultConfigAt switches to the pdfcpu config dir located at path.
// If path/pdfcpu is not existent, it will be created including config.yml
func EnsureDefaultConfigAt(path string) error {
	// Call if you have specific requirements regarding the location of the pdfcpu config dir.
	return model.EnsureDefaultConfigAt(path, false)
}

var (
	// mutexDisableConfigDir protects DisableConfigDir from concurrent access.
	// NOTE Not a guard for model.ConfigPath!
	mutexDisableConfigDir sync.Mutex
)

// DisableConfigDir disables the configuration directory.
// Any needed default configuration will be loaded from configuration.go
// Since the config dir also contains the user font dir, this also limits font usage to the default core font set
// No user fonts will be available.
func DisableConfigDir() {
	mutexDisableConfigDir.Lock()
	defer mutexDisableConfigDir.Unlock()
	// Call if you don't want to use a specific configuration
	// and also do not need to use user fonts.
	model.ConfigPath = "disable"
}

// LoadConfiguration locates and loads the default configuration
// and also loads installed user fonts.
func LoadConfiguration() *model.Configuration {
	// Call if you don't have a specific config dir location
	// and need to use user fonts for stamping or watermarking.
	return model.NewDefaultConfiguration()
}
