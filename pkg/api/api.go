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
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/validate"
)

var (
	// ErrMissingBookletConfiguration signals a missing booklet configuration.
	ErrMissingBookletConfiguration = errors.New("missing booklet configuration")

	// ErrBookletImageOutputConflict signals that booklet output aliases an image input.
	ErrBookletImageOutputConflict = errors.New("booklet image output aliases input")

	// ErrMissingGridConfiguration signals a missing grid configuration.
	ErrMissingGridConfiguration = errors.New("missing grid configuration")

	// ErrGridImageOutputConflict signals that grid output aliases an image input.
	ErrGridImageOutputConflict = errors.New("grid image output aliases input")

	// ErrMissingNUpConfiguration signals a missing n-up configuration.
	ErrMissingNUpConfiguration = errors.New("missing n-up configuration")

	// ErrMissingCutConfiguration signals a missing cut configuration.
	ErrMissingCutConfiguration = errors.New("missing cut configuration")

	// ErrInvalidCutConfiguration signals an invalid cut configuration.
	ErrInvalidCutConfiguration = errors.New("invalid cut configuration")

	// ErrMissingResizeConfiguration signals a missing resize configuration.
	ErrMissingResizeConfiguration = errors.New("missing resize configuration")

	// ErrInvalidResizeConfiguration signals an invalid resize configuration.
	ErrInvalidResizeConfiguration = errors.New("invalid resize configuration")

	// ErrInvalidPageConfiguration signals an invalid page configuration.
	ErrInvalidPageConfiguration = errors.New("invalid page configuration")

	// ErrMissingZoomConfiguration signals a missing zoom configuration.
	ErrMissingZoomConfiguration = errors.New("missing zoom configuration")

	// ErrInvalidZoomConfiguration signals an invalid zoom configuration.
	ErrInvalidZoomConfiguration = errors.New("invalid zoom configuration")

	// ErrNUpImageOutputConflict signals that n-up output aliases an image input.
	ErrNUpImageOutputConflict = errors.New("n-up image output aliases input")

	// ErrMissingPageBoundaries signals missing required page boundaries.
	ErrMissingPageBoundaries = errors.New("missing page boundaries")

	// ErrInvalidPageBoundaries signals invalid page boundaries for an operation.
	ErrInvalidPageBoundaries = errors.New("invalid page boundaries")

	// ErrMissingBoxConfiguration signals a missing box configuration.
	ErrMissingBoxConfiguration = errors.New("missing box configuration")

	// ErrMissingConfiguration signals a missing pdfcpu configuration.
	ErrMissingConfiguration = errors.New("missing configuration")

	// ErrMissingDigestFunction signals a missing required digest function.
	ErrMissingDigestFunction = errors.New("missing digest function")

	// ErrMissingImageInput signals a missing required image input.
	ErrMissingImageInput = errors.New("missing image input")

	// ErrMissingFontInput signals a missing required font input file.
	ErrMissingFontInput = errors.New("missing font input")

	// ErrUnsupportedFontFile signals an unsupported font input file.
	ErrUnsupportedFontFile = errors.New("unsupported font file")

	// ErrInvalidUnicodePlane signals an invalid Unicode plane.
	ErrInvalidUnicodePlane = errors.New("invalid Unicode plane")

	// ErrImportImagesOutputConflict signals that import-images output aliases an image input.
	ErrImportImagesOutputConflict = errors.New("import images output aliases image input")

	// ErrInvalidImportConfiguration signals an invalid image import configuration.
	ErrInvalidImportConfiguration = errors.New("invalid import configuration")

	// ErrInvalidImageSelection signals an invalid image object or page resource selection.
	ErrInvalidImageSelection = errors.New("invalid image selection")

	// ErrUpdateImagesOutputConflict signals that update-images output aliases the image input.
	ErrUpdateImagesOutputConflict = errors.New("update images output aliases image input")

	// ErrMissingImageReader signals a missing required image reader.
	ErrMissingImageReader = pdfcpu.ErrMissingImageReader

	// ErrMissingJSONInput signals a missing required JSON input file.
	ErrMissingJSONInput = errors.New("missing JSON input")

	// ErrMissingJSONOutput signals a missing required JSON output file.
	ErrMissingJSONOutput = errors.New("missing JSON output")

	// ErrMissingPDFContext signals a missing required PDF context.
	ErrMissingPDFContext = pdfcpu.ErrMissingPDFContext

	// ErrMissingXRefTable signals a missing required PDF cross-reference table.
	ErrMissingXRefTable = pdfcpu.ErrMissingXRefTable

	// ErrMissingPDFInput signals a missing required PDF input file.
	ErrMissingPDFInput = errors.New("missing PDF input")

	// ErrMissingPDFOutput signals a missing required PDF output file.
	ErrMissingPDFOutput = errors.New("missing PDF output")

	// ErrMissingPDFReadSeeker signals a missing required PDF input reader.
	ErrMissingPDFReadSeeker = errors.New("missing PDF read seeker")

	// ErrMissingPDFReadWriteSeeker signals a missing required PDF input/output seeker.
	ErrMissingPDFReadWriteSeeker = errors.New("missing PDF read write seeker")

	// ErrMissingPDFWriter signals a missing required PDF output writer.
	ErrMissingPDFWriter = errors.New("missing PDF writer")

	// ErrMissingReader signals a missing required reader.
	ErrMissingReader = pdfcpu.ErrMissingReader

	// ErrMissingWatermarkConfiguration signals a missing required watermark configuration.
	ErrMissingWatermarkConfiguration = pdfcpu.ErrMissingWatermarkConfiguration

	// ErrMissingWatermarks signals missing required watermarks.
	ErrMissingWatermarks = pdfcpu.ErrMissingWatermarks

	// ErrNoSignatures signals that a PDF has no signatures to process.
	ErrNoSignatures = pdfcpu.ErrNoSignatures
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
	if log.CLIEnabled() {
		log.CLI.Println(disclaimer)
	}
}

// ReadContext uses an io.ReadSeeker to build an internal structure holding its cross reference table aka the Context.
func ReadContext(rs io.ReadSeeker, conf *model.Configuration) (ctx *model.Context, err error) {
	defer fault.Catch(&err)
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
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
	if ctx == nil {
		return ErrMissingPDFContext
	}

	if ctx.XRefTable == nil {
		return ErrMissingXRefTable
	}

	if ctx.XRefTable.Version() == model.V20 {
		logDisclaimerPDF20()
	}
	return validate.XRefTable(ctx)
}

// OptimizeContext optimizes ctx.
func OptimizeContext(ctx *model.Context) error {
	if ctx == nil {
		return ErrMissingPDFContext
	}

	if log.CLIEnabled() {
		log.CLI.Println("optimizing...")
	}
	if err := pdfcpu.OptimizeXRefTable(ctx); err != nil {
		return fmt.Errorf("optimize context: %w", err)
	}
	return nil
}

// PatchFile writes bb at offset.
func PatchFile(fileName string, bb []byte, offset int64) error {
	f, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	if _, err := f.WriteAt(bb, offset); err != nil {
		return err
	}

	return nil
}

// WriteContext writes ctx to w.
func WriteContext(ctx *model.Context, w io.Writer) (err error) {
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
		err = errors.Join(err, ctx.Write.Flush())
	}()
	return pdfcpu.WriteContext(ctx)
}

// WriteIncrement writes a PDF increment for ctx to w.
func WriteIncrement(ctx *model.Context, w io.Writer) (err error) {
	if ctx == nil {
		return ErrMissingPDFContext
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	ctx.Write.Writer = bufio.NewWriter(w)
	defer func() {
		err = errors.Join(err, ctx.Write.Flush())
	}()
	return pdfcpu.WriteIncrement(ctx)
}

// WriteContextFile writes ctx to outFile.
func WriteContextFile(ctx *model.Context, outFile string) (err error) {
	staged, err := openStagedOutput(nil, "", outFile, "write context")
	if err != nil {
		return err
	}
	f := staged.output.file
	if err := WriteContext(ctx, f); err != nil {
		return staged.cleanup(err)
	}
	return staged.commit()
}

// ReadAndValidate returns a model.Context of rs ready for processing.
func ReadAndValidate(rs io.ReadSeeker, conf *model.Configuration) (ctx *model.Context, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if ctx, err = ReadContext(rs, conf); err != nil {
		return nil, fmt.Errorf("read context: %w", err)
	}
	if conf == nil {
		conf = ctx.Configuration
	}

	if err := ValidateContext(ctx); err != nil {
		return nil, validationError(ctx, conf, err)
	}

	if conf.Cmd == model.REMOVESIGNATURES || ctx.RemoveSignatures && conf.Cmd.AllowRemoveSignatures() {

		if len(ctx.Signatures) == 0 {
			if conf.Cmd == model.REMOVESIGNATURES {
				return nil, ErrNoSignatures
			}
			if log.CLIEnabled() {
				log.CLI.Println("no signatures to remove...")
			}
			return ctx, nil
		}

		if log.CLIEnabled() {
			log.CLI.Println("removing signatures...")
		}
		if err := ctx.RemoveAllSignatures(); err != nil {
			return nil, fmt.Errorf("remove signatures: %w", err)
		}
	}

	return ctx, nil
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

// ReadValidateAndOptimize returns an optimized model.Context of rs ready for processing a specific command.
// conf.Cmd is expected to be configured properly.
func ReadValidateAndOptimize(rs io.ReadSeeker, conf *model.Configuration) (ctx *model.Context, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		return nil, ErrMissingConfiguration
	}

	ctx, err = ReadAndValidate(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("prepare PDF context: %w", err)
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
		return nil, fmt.Errorf("cache form fonts: %w", err)
	}

	return ctx, nil
}

func logWritingTo(s string) {
	if log.CLIEnabled() {
		log.CLI.Printf("writing %s...\n", s)
	}
}

// Write writes ctx using w.
func Write(ctx *model.Context, w io.Writer, conf *model.Configuration) error {
	if ctx == nil {
		return ErrMissingPDFContext
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if log.StatsEnabled() {
		log.Stats.Printf("XRefTable:\n%s\n", ctx)
	}

	return WriteContext(ctx, w)
}

// WriteIncr writes ctx as increment using rws.
func WriteIncr(ctx *model.Context, rws io.ReadWriteSeeker, conf *model.Configuration) error {
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
