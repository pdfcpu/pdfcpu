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
//	func CommandFile(c context.Context, inFile, outFile string, conf *pdf.Configuration) error
//	func Command(c context.Context, rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error
//
// eg. for optimization:
//
//	func OptimizeFile(c context.Context, inFile, outFile string, conf *pdf.Configuration) error
//	func Optimize(c context.Context, rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error
package api

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/validate"
)

func logDisclaimerPDF20() {
	disclaimer := `
***************************** Disclaimer ****************************
* PDF 2.0 features are supported on a need basis.                   *
* (See ISO 32000:2 6.3.2 Conformance of PDF processors)             *
* At the moment pdfcpu ships with basic PDF 2.0 support.            *
* Please let us know which feature you would like to see supported, *
* provide a sample PDF file and create an issue:                    *
* https://github.com/pdfcpu/pdfcpu/issues/new/choose                *
* Thank you for using pdfcpu <3                                     *
*********************************************************************`

	if log.ValidateEnabled() {
		log.Validate.Println(disclaimer)
	}
}

func operationConfiguration(conf *model.Configuration, cmd model.CommandMode) *model.Configuration {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf = conf.Clone()
	}
	conf.Cmd = cmd
	return conf
}

// ReadContext uses an io.ReadSeeker to build an internal PDF context and supports cancellation.
func ReadContext(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (ctx *model.Context, err error) {
	defer fault.Catch(&err)
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}
	return pdfcpu.Read(c, rs, conf)
}

// ReadContextFile returns inFile's validated context and supports cancellation.
func ReadContextFile(c context.Context, inFile string) (*model.Context, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ctx, err := ReadContext(c, f, model.NewDefaultConfiguration())
	if err != nil {
		return nil, err
	}

	if err = ValidateContext(c, ctx); err != nil {
		return nil, err
	}

	return ctx, nil
}

// ValidateContext validates ctx and supports cancellation.
func ValidateContext(c context.Context, ctx *model.Context) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if ctx == nil {
		return ErrMissingPDFContext
	}

	if ctx.XRefTable == nil {
		return ErrMissingXRefTable
	}

	if ctx.XRefTable.Version() == model.V20 {
		logDisclaimerPDF20()
	}
	return validate.XRefTable(c, ctx)
}

// OptimizeContext optimizes ctx and supports cancellation.
func OptimizeContext(c context.Context, ctx *model.Context) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if ctx == nil {
		return ErrMissingPDFContext
	}

	if err := pdfcpu.OptimizeXRefTable(c, ctx); err != nil {
		return fmt.Errorf("optimize context: %w", err)
	}
	return nil
}

// PatchFile writes bb at offset in a staged copy, replaces fileName after the update succeeds
// and supports cancellation.
func PatchFile(c context.Context, fileName string, bb []byte, offset int64) error {
	return updateFileTransaction(c, fileName, "patch", func(c context.Context, f *os.File) error {
		_, err := f.WriteAt(bb, offset)
		return err
	})
}

// WriteContext writes ctx to w and supports cancellation.
func WriteContext(c context.Context, ctx *model.Context, w io.Writer) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if ctx == nil {
		return ErrMissingPDFContext
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if f, ok := w.(*os.File); ok {
		// In order to retrieve the written file size.
		ctx.Write.Fp = f
	}
	ctx.Write.Writer = bufio.NewWriter(w)
	defer func() {
		if cancelErr := contextutil.Check(c); cancelErr != nil {
			err = errors.Join(err, cancelErr)
			return
		}
		err = errors.Join(err, ctx.Write.Flush())
	}()
	return pdfcpu.WriteContext(c, ctx)
}

// WriteIncrement writes a PDF increment for ctx to w and supports cancellation.
func WriteIncrement(c context.Context, ctx *model.Context, w io.Writer) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if ctx == nil {
		return ErrMissingPDFContext
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	ctx.Write.Writer = bufio.NewWriter(w)
	defer func() {
		if cancelErr := contextutil.Check(c); cancelErr != nil {
			err = errors.Join(err, cancelErr)
			return
		}
		err = errors.Join(err, ctx.Write.Flush())
	}()
	return pdfcpu.WriteIncrement(c, ctx)
}

// WriteContextFile writes ctx to outFile and supports cancellation.
func WriteContextFile(c context.Context, ctx *model.Context, outFile string) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	staged, err := openStagedOutput(nil, "", outFile, "write context")
	if err != nil {
		return err
	}
	f := staged.output.file
	if err := WriteContext(c, ctx, f); err != nil {
		return staged.cleanup(err)
	}
	return staged.commit()
}

func readAndValidate(c context.Context, rs io.ReadSeeker, conf *model.Configuration, options ProgressOptions) (ctx *model.Context, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if err := reportProgress(options, ProgressStageReading); err != nil {
		return nil, err
	}

	if ctx, err = ReadContext(c, rs, conf); err != nil {
		return nil, fmt.Errorf("read context: %w", err)
	}
	if conf == nil {
		conf = ctx.Configuration
	}

	if err := reportProgress(options, ProgressStageValidating); err != nil {
		return nil, err
	}

	if err := ValidateContext(c, ctx); err != nil {
		return nil, validationError(conf, err)
	}
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}

	if conf.Cmd == model.REMOVESIGNATURES || ctx.RemoveSignatures && conf.Cmd.AllowRemoveSignatures() {

		if len(ctx.Signatures) == 0 {
			if conf.Cmd == model.REMOVESIGNATURES {
				return nil, ErrNoSignatures
			}
			return ctx, nil
		}

		if err := ctx.RemoveAllSignatures(); err != nil {
			return nil, fmt.Errorf("remove signatures: %w", err)
		}
	}

	return ctx, nil
}

// ReadAndValidate returns a validated model.Context and supports cancellation.
func ReadAndValidate(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (*model.Context, error) {
	return readAndValidate(c, rs, conf, ProgressOptions{})
}

func cmdAssumingOptimization(cmd model.CommandMode) bool {
	return cmd == model.OPTIMIZE ||
		cmd == model.FILLFORMFIELDS ||
		cmd == model.RESETFORMFIELDS ||
		cmd == model.LISTIMAGES ||
		cmd == model.UPDATEIMAGES ||
		cmd == model.EXTRACTIMAGES ||
		cmd == model.EXTRACTFONTS ||
		cmd == model.REMOVESIGNATURES
}

// ReadValidateAndOptimize returns an optimized model.Context, supports cancellation and reports optional
// semantic progress. conf.Cmd is expected to be configured properly.
// A nil options pointer disables progress reporting. Supplied options are not modified.
func ReadValidateAndOptimize(c context.Context, rs io.ReadSeeker, conf *model.Configuration, options *ProgressOptions) (ctx *model.Context, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		return nil, ErrMissingConfiguration
	}

	progress := progressOptionsValue(options)
	ctx, err = readAndValidate(c, rs, conf, progress)
	if err != nil {
		return nil, fmt.Errorf("prepare PDF context: %w", err)
	}

	// With the exception of commands utilizing structs provided the Optimize step
	// command optimization of the cross reference table is optional but usually recommended.
	// For large or complex files it may make sense to skip optimization and set conf.Optimize = false.
	if cmdAssumingOptimization(conf.Cmd) || conf.Optimize {
		if err := reportProgress(progress, ProgressStageOptimizing); err != nil {
			return nil, err
		}
		if err = OptimizeContext(c, ctx); err != nil {
			return nil, err
		}
	}
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}

	// TODO move to form related commands.
	if err := pdfcpu.CacheFormFonts(ctx); err != nil {
		return nil, fmt.Errorf("cache form fonts: %w", err)
	}

	return ctx, nil
}

// Write writes ctx using w and supports cancellation.
func Write(c context.Context, ctx *model.Context, w io.Writer, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if ctx == nil {
		return ErrMissingPDFContext
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if log.StatsEnabled() {
		log.Stats.Printf("XRefTable:\n%s\n", ctx)
	}

	return WriteContext(c, ctx, w)
}

// WriteIncr writes ctx as increment using rws and supports cancellation.
func WriteIncr(c context.Context, ctx *model.Context, rws io.ReadWriteSeeker, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if ctx == nil {
		return ErrMissingPDFContext
	}

	if rws == nil {
		return ErrMissingPDFReadWriteSeeker
	}

	if conf == nil {
		return ErrMissingConfiguration
	}

	if log.StatsEnabled() {
		log.Stats.Printf("XRefTable:\n%s\n", ctx)
	}

	if conf.PostProcessValidate {
		if err := ValidateContext(c, ctx); err != nil {
			return err
		}
	}

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if _, err := rws.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	return WriteIncrement(c, ctx, rws)
}

// EnsureDefaultConfigAt switches to the pdfcpu config dir located at path.
// If path/pdfcpu is not existent, it will be created including config.yml
//
// Deprecated: use LoadConfiguration with ConfigurationOptions.Root.
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
//
// Deprecated: use LoadConfiguration with ConfigurationModeStateless.
func DisableConfigDir() {
	mutexDisableConfigDir.Lock()
	defer mutexDisableConfigDir.Unlock()
	// Call if you don't want to use a specific configuration
	// and also do not need to use user fonts.
	model.ConfigPath = "disable"
}
