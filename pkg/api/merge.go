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
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func mergeSourceLabel(source string) string {
	if source == "" {
		return "source"
	}
	return fmt.Sprintf("source %s", source)
}

func wrapMergeCleanupError(context string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

func mergeFileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// appendTo appends rs to ctxDest's page tree.
func appendTo(rs io.ReadSeeker, fName string, ctxDest *model.Context, dividerPage bool) error {
	source := mergeSourceLabel(fName)
	if rs == nil {
		return fmt.Errorf("merge %s: read source: %w", source, ErrMissingPDFReadSeeker)
	}

	ctxSource, err := ReadAndValidate(rs, ctxDest.Configuration)
	if err != nil {
		return fmt.Errorf("merge %s: read and validate: %w", source, err)
	}

	if ctxDest.XRefTable.Version() < model.V20 && ctxSource.XRefTable.Version() == model.V20 {
		return fmt.Errorf("merge %s: validate version: %w", source, pdfcpu.ErrUnsupportedVersion)
	}

	// Merge source context into dest context.
	if err := pdfcpu.MergeXRefTables(fName, ctxSource, ctxDest, false, dividerPage); err != nil {
		return fmt.Errorf("merge %s: append pages: %w", source, err)
	}
	return nil
}

func appendFile(fName string, ctxDest *model.Context, dividerPage bool) (err error) {
	f, err := os.Open(fName)
	if err != nil {
		return fmt.Errorf("merge source: open %s: %w", fName, err)
	}
	defer func() {
		closeErr := f.Close()
		if err != nil {
			return
		}
		err = wrapMergeCleanupError("merge source: close input", closeErr)
	}()

	if log.CLIEnabled() {
		log.CLI.Println(fName)
	}
	return appendTo(f, filepath.Base(fName), ctxDest, dividerPage)
}

// MergeRaw merges a sequence of PDF streams and writes the result to w.
func MergeRaw(rsc []io.ReadSeeker, w io.Writer, dividerPage bool, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if len(rsc) == 0 {
		return fmt.Errorf("missing PDF inputs: %w", ErrMissingPDFInput)
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.MERGECREATE
	conf.ValidationMode = model.ValidationRelaxed
	conf.CreateBookmarks = false

	ctxDest, err := ReadAndValidate(rsc[0], conf)
	if err != nil {
		return fmt.Errorf("merge source 0: read and validate: %w", err)
	}

	ctxDest.EnsureVersionForWriting()

	for i, f := range rsc[1:] {
		if err = appendTo(f, fmt.Sprintf("%d", i+1), ctxDest, dividerPage); err != nil {
			return err
		}
	}

	if conf.OptimizeBeforeWriting {
		if err = OptimizeContext(ctxDest); err != nil {
			return fmt.Errorf("merge: optimize context: %w", err)
		}
	}

	if err = WriteContext(ctxDest, w); err != nil {
		return fmt.Errorf("merge: write output: %w", err)
	}
	return nil
}

func prepDestContext(destFile string, rs io.ReadSeeker, conf *model.Configuration) (*model.Context, error) {
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	ctxDest, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("merge destination %s: read and validate: %w", filepath.Base(destFile), err)
	}

	if conf.CreateBookmarks && conf.MergeBookmarkMode != model.MergeBookmarkModePreserve {
		if err := pdfcpu.EnsureOutlines(ctxDest, filepath.Base(destFile), conf.Cmd == model.MERGEAPPEND); err != nil {
			return nil, fmt.Errorf("merge destination %s: ensure outlines: %w", filepath.Base(destFile), err)
		}
	}

	if ctxDest.XRefTable.Version() < model.V20 {
		ctxDest.EnsureVersionForWriting()
	}
	return ctxDest, nil
}

func mergeDestFile(destFile string, inFiles []string) (string, []string, error) {
	if destFile != "" {
		return destFile, inFiles, nil
	}
	if len(inFiles) == 0 {
		return "", nil, ErrMissingPDFInput
	}
	return inFiles[0], inFiles[1:], nil
}

func mergeConfiguration(destFile string, conf *model.Configuration) *model.Configuration {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.MERGECREATE
	conf.ValidationMode = model.ValidationRelaxed
	if destFile != "" {
		conf.Cmd = model.MERGEAPPEND
	}
	return conf
}

// Merge concatenates inFiles.
// if destFile is supplied it appends the result to destfile (=MERGEAPPEND)
// if no destFile supplied it writes the result to the first entry of inFiles (=MERGECREATE).
func Merge(destFile string, inFiles []string, w io.Writer, conf *model.Configuration, dividerPage bool) (err error) {
	if w == nil {
		return ErrMissingPDFWriter
	}

	conf = mergeConfiguration(destFile, conf)
	destFile, inFiles, err = mergeDestFile(destFile, inFiles)
	if err != nil {
		return err
	}

	if conf.CreateBookmarks && log.CLIEnabled() {
		log.CLI.Println("creating bookmarks...")
	}

	f, err := os.Open(destFile)
	if err != nil {
		return fmt.Errorf("merge destination: open %s: %w", destFile, err)
	}
	defer func() {
		closeErr := f.Close()
		if err != nil {
			return
		}
		err = wrapMergeCleanupError("merge destination: close input", closeErr)
	}()

	if conf.Cmd == model.MERGECREATE {
		if log.CLIEnabled() {
			log.CLI.Println(destFile)
		}
	}

	ctxDest, err := prepDestContext(destFile, f, conf)
	if err != nil {
		return err
	}

	for _, fName := range inFiles {
		if err := appendFile(fName, ctxDest, dividerPage); err != nil {
			return err
		}
	}

	if conf.OptimizeBeforeWriting {
		if err := OptimizeContext(ctxDest); err != nil {
			return fmt.Errorf("merge: optimize context: %w", err)
		}
	}

	if err := WriteContext(ctxDest, w); err != nil {
		return fmt.Errorf("merge: write output: %w", err)
	}
	return nil
}

// MergeCreateFile merges inFiles and writes the result to outFile.
func MergeCreateFile(inFiles []string, outFile string, dividerPage bool, conf *model.Configuration) (err error) {
	staged, err := openStagedOutput(nil, "", outFile, "merge")
	if err != nil {
		return fmt.Errorf("merge: create output: %w", err)
	}
	f := staged.output.file

	defer func() {
		if err != nil {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	logWritingTo(outFile)

	if err = Merge("", inFiles, f, conf, dividerPage); err != nil {
		return err
	}
	return nil
}

// MergeAppendFile appends inFiles to outFile.
func MergeAppendFile(inFiles []string, outFile string, dividerPage bool, conf *model.Configuration) (err error) {
	var f *os.File
	ok := false

	tmpFile := outFile
	destFile := ""

	if mergeFileExists(outFile) {
		destFile = outFile
		if log.CLIEnabled() {
			log.CLI.Printf("appending to %s...\n", outFile)
		}
	} else {
		logWritingTo(outFile)
	}
	staged, err := openStagedOutput(nil, "", tmpFile, "merge")
	if err != nil {
		return fmt.Errorf("merge: create output: %w", err)
	}
	f = staged.output.file

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = Merge(destFile, inFiles, f, conf, dividerPage); err != nil {
		return err
	}

	ok = true
	return nil
}

// MergeCreateZip zips rs1 and rs2 into w.
func MergeCreateZip(rs1, rs2 io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs1 == nil {
		return fmt.Errorf("merge zip source 1: %w", ErrMissingPDFReadSeeker)
	}

	if rs2 == nil {
		return fmt.Errorf("merge zip source 2: %w", ErrMissingPDFReadSeeker)
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.MERGECREATEZIP
	conf.ValidationMode = model.ValidationRelaxed

	ctxDest, err := ReadAndValidate(rs1, conf)
	if err != nil {
		return fmt.Errorf("merge zip source 1: read and validate: %w", err)
	}
	if ctxDest.XRefTable.Version() == model.V20 {
		return fmt.Errorf("merge zip source 1: validate version: %w", pdfcpu.ErrUnsupportedVersion)
	}
	ctxDest.EnsureVersionForWriting()

	if _, err = pdfcpu.RemoveBookmarks(ctxDest); err != nil {
		return fmt.Errorf("merge zip source 1: remove bookmarks: %w", err)
	}

	ctxSrc, err := ReadAndValidate(rs2, conf)
	if err != nil {
		return fmt.Errorf("merge zip source 2: read and validate: %w", err)
	}
	if ctxSrc.XRefTable.Version() == model.V20 {
		return fmt.Errorf("merge zip source 2: validate version: %w", pdfcpu.ErrUnsupportedVersion)
	}

	if err := pdfcpu.MergeXRefTables("", ctxSrc, ctxDest, true, false); err != nil {
		return fmt.Errorf("merge zip: append pages: %w", err)
	}

	if conf.OptimizeBeforeWriting {
		if err := OptimizeContext(ctxDest); err != nil {
			return fmt.Errorf("merge zip: optimize context: %w", err)
		}
	}

	if err := WriteContext(ctxDest, w); err != nil {
		return fmt.Errorf("merge zip: write output: %w", err)
	}
	return nil
}

// MergeCreateZipFile zips inFile1 and inFile2 into outFile.
func MergeCreateZipFile(inFile1, inFile2, outFile string, conf *model.Configuration) (err error) {
	var f1, f2, f *os.File

	if f1, err = os.Open(inFile1); err != nil {
		return fmt.Errorf("merge zip source 1: open %s: %w", inFile1, err)
	}

	if f2, err = os.Open(inFile2); err != nil {
		_ = f1.Close()
		return fmt.Errorf("merge zip source 2: open %s: %w", inFile2, err)
	}

	staged, err := openStagedOutput(f2, "", outFile, "merge zip")
	if err != nil {
		_ = f1.Close()
		_ = f2.Close()
		return fmt.Errorf("merge zip: create output %s: %w", outFile, err)
	}
	f = staged.output.file
	staged = staged.withInput(f1, "merge zip source 1: close")
	staged.inputs[0].context = "merge zip source 2: close"

	defer func() {
		if err != nil {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	logWritingTo(outFile)

	if err = MergeCreateZip(f1, f2, f, conf); err != nil {
		return err
	}
	return nil
}
