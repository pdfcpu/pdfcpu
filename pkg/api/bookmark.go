/*
	Copyright 2021 The pdfcpu Authors.

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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

var (
	ErrNoOutlines = errors.New("pdfcpu: no outlines available")
	ErrOutlines   = errors.New("pdfcpu: existing outlines")
)

// Bookmarks returns rs's bookmark hierarchy.
func Bookmarks(rs io.ReadSeeker, conf *model.Configuration) ([]pdfcpu.Bookmark, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: Bookmarks: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTBOOKMARKS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}
	return pdfcpu.Bookmarks(ctx)
}

// ExportBookmarksJSON extracts outline data from rs (originating from source) and writes the result to w.
func ExportBookmarksJSON(rs io.ReadSeeker, w io.Writer, source string, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: ExportBookmarksJSON: missing rs")
	}

	if w == nil {
		return errors.New("pdfcpu: ExportBookmarksJSON: missing w")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXPORTBOOKMARKS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	ok, err := pdfcpu.ExportBookmarksJSON(ctx, source, w)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoOutlines
	}

	return nil
}

// ExportBookmarksFile extracts outline data from inFilePDF and writes the result to outFileJSON.
func ExportBookmarksFile(inFilePDF, outFileJSON string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFilePDF); err != nil {
		return err
	}

	if f2, err = os.Create(outFileJSON); err != nil {
		f1.Close()
		return err
	}
	log.CLI.Printf("writing %s...\n", outFileJSON)

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
	}()

	return ExportBookmarksJSON(f1, f2, inFilePDF, conf)
}

// ImportBookmarks creates/replaces outlines in rs and writes the result to w.
func ImportBookmarks(rs io.ReadSeeker, rd io.Reader, w io.Writer, replace bool, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: ImportBookmarks: missing rs")
	}

	if rd == nil {
		return errors.New("pdfcpu: ImportBookmarks: missing rd")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.IMPORTBOOKMARKS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	ok, err := pdfcpu.ImportBookmarks(ctx, rd, replace)
	if err != nil {
		return err
	}
	if !ok {
		return ErrOutlines
	}

	return WriteContext(ctx, w)
}

// ImportBookmarks creates/replaces outlines in inFilePDF and writes the result to outFilePDF.
func ImportBookmarksFile(inFilePDF, inFileJSON, outFilePDF string, replace bool, conf *model.Configuration) (err error) {

	var f0, f1, f2 *os.File

	if f0, err = os.Open(inFilePDF); err != nil {
		return err
	}

	if f1, err = os.Open(inFileJSON); err != nil {
		return err
	}

	tmpFile := inFilePDF + ".tmp"
	if outFilePDF != "" && inFilePDF != outFilePDF {
		tmpFile = outFilePDF
	}
	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFilePDF == "" || inFilePDF == outFilePDF {
				os.Remove(tmpFile)
			}
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFilePDF == "" || inFilePDF == outFilePDF {
			err = os.Rename(tmpFile, inFilePDF)
		}
	}()

	return ImportBookmarks(f0, f1, f2, replace, conf)
}

// AddBookmarks adds a single bookmark outline layer to the PDF context read from rs and writes the result to w.
func AddBookmarks(rs io.ReadSeeker, w io.Writer, bms []pdfcpu.Bookmark, replace bool, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: AddBookmarks: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.ADDBOOKMARKS

	if len(bms) == 0 {
		return errors.New("pdfcpu: AddBookmarks: missing bms")
	}

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := pdfcpu.AddBookmarks(ctx, bms, replace); err != nil {
		return err
	}

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	return nil
}

// AddBookmarksFile adds outlines to the PDF context read from inFile and writes the result to outFile.
func AddBookmarksFile(inFile, outFile string, bms []pdfcpu.Bookmark, replace bool, conf *model.Configuration) (err error) {

	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
			}
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	return AddBookmarks(f1, f2, bms, replace, conf)
}

// RemoveBookmarks deletes outlines from rs and writes the result to w.
func RemoveBookmarks(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: AddBookmarks: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.REMOVEBOOKMARKS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	ok, err := pdfcpu.RemoveBookmarks(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoOutlines
	}

	return WriteContext(ctx, w)
}

// RemoveBookmarksFile deletes outlines from inFile and writes the result to outFile.
func RemoveBookmarksFile(inFile, outFile string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
			}
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	return RemoveBookmarks(f1, f2, conf)
}
