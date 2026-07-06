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
	"path/filepath"
	"strconv"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// appendTo appends rs to ctxDest's page tree.
func appendTo(rs io.ReadSeeker, fName string, ctxDest *model.Context, dividerPage bool) error {
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	ctxSource, err := ReadAndValidate(rs, ctxDest.Configuration)
	if err != nil {
		return err
	}

	if ctxDest.XRefTable.Version() < model.V20 && ctxSource.XRefTable.Version() == model.V20 {
		return pdfcpu.ErrUnsupportedVersion
	}

	// Merge source context into dest context.
	return pdfcpu.MergeXRefTables(fName, ctxSource, ctxDest, false, dividerPage)
}

func appendFile(fName string, ctxDest *model.Context, dividerPage bool) error {
	f, err := os.Open(fName)
	if err != nil {
		return err
	}
	defer f.Close()

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
		return err
	}

	ctxDest.EnsureVersionForWriting()

	for i, f := range rsc[1:] {
		if err = appendTo(f, strconv.Itoa(i), ctxDest, dividerPage); err != nil {
			return err
		}
	}

	if conf.OptimizeBeforeWriting {
		if err = OptimizeContext(ctxDest); err != nil {
			return err
		}
	}

	return WriteContext(ctxDest, w)
}

func prepDestContext(destFile string, rs io.ReadSeeker, conf *model.Configuration) (*model.Context, error) {
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	ctxDest, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, err
	}

	if conf.CreateBookmarks && conf.MergeBookmarkMode != model.MergeBookmarkModePreserve {
		if err := pdfcpu.EnsureOutlines(ctxDest, filepath.Base(destFile), conf.Cmd == model.MERGEAPPEND); err != nil {
			return nil, err
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

// Merge concatenates inFiles.
// if destFile is supplied it appends the result to destfile (=MERGEAPPEND)
// if no destFile supplied it writes the result to the first entry of inFiles (=MERGECREATE).
func Merge(destFile string, inFiles []string, w io.Writer, conf *model.Configuration, dividerPage bool) error {
	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.MERGECREATE
	conf.ValidationMode = model.ValidationRelaxed

	if destFile != "" {
		conf.Cmd = model.MERGEAPPEND
	}
	var err error
	destFile, inFiles, err = mergeDestFile(destFile, inFiles)
	if err != nil {
		return err
	}

	if conf.CreateBookmarks && log.CLIEnabled() {
		log.CLI.Println("creating bookmarks...")
	}

	f, err := os.Open(destFile)
	if err != nil {
		return err
	}
	defer f.Close()

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
			return err
		}
	}

	return WriteContext(ctxDest, w)
}

// MergeCreateFile merges inFiles and writes the result to outFile.
func MergeCreateFile(inFiles []string, outFile string, dividerPage bool, conf *model.Configuration) (err error) {
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if err1 := f.Close(); err1 != nil {
				return
			}
			os.Remove(outFile)
			return
		}
		if err = f.Close(); err != nil {
			return
		}
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
	overWrite := false
	destFile := ""

	if fileExists(outFile) {
		overWrite = true
		destFile = outFile
		if log.CLIEnabled() {
			log.CLI.Printf("appending to %s...\n", outFile)
		}
	} else {
		logWritingTo(outFile)
	}
	inFile := ""
	if overWrite {
		inFile = outFile
	}
	if f, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		return err
	}

	defer func() {
		if !ok {
			_ = f.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f.Close(); err != nil {
			return
		}
		if overWrite {
			err = os.Rename(tmpFile, outFile)
		}
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
		return errors.New("missing first PDF read seeker")
	}

	if rs2 == nil {
		return errors.New("missing second PDF read seeker")
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
		return err
	}
	if ctxDest.XRefTable.Version() == model.V20 {
		return pdfcpu.ErrUnsupportedVersion
	}
	ctxDest.EnsureVersionForWriting()

	if _, err = pdfcpu.RemoveBookmarks(ctxDest); err != nil {
		return err
	}

	ctxSrc, err := ReadAndValidate(rs2, conf)
	if err != nil {
		return err
	}
	if ctxSrc.XRefTable.Version() == model.V20 {
		return pdfcpu.ErrUnsupportedVersion
	}

	if err := pdfcpu.MergeXRefTables("", ctxSrc, ctxDest, true, false); err != nil {
		return err
	}

	if conf.OptimizeBeforeWriting {
		if err := OptimizeContext(ctxDest); err != nil {
			return err
		}
	}

	return WriteContext(ctxDest, w)
}

// MergeCreateZipFile zips inFile1 and inFile2 into outFile.
func MergeCreateZipFile(inFile1, inFile2, outFile string, conf *model.Configuration) (err error) {
	var f1, f2, f *os.File

	if f1, err = os.Open(inFile1); err != nil {
		return err
	}

	if f2, err = os.Open(inFile2); err != nil {
		_ = f1.Close()
		return err
	}

	if f, err = os.Create(outFile); err != nil {
		_ = f1.Close()
		_ = f2.Close()
		return err
	}

	defer func() {
		if err = f.Close(); err != nil {
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
	}()

	logWritingTo(outFile)

	if err = MergeCreateZip(f1, f2, f, conf); err != nil {
		return err
	}

	return nil
}
