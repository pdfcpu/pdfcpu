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
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func bookmarkOpError(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}

func bookmarkSourceError(op, source string, err error) error {
	if err == nil {
		return nil
	}
	if source == "" {
		return bookmarkOpError(op, err)
	}
	return fmt.Errorf("%s: %s: %w", op, source, err)
}

func closeBookmarkInput(err error, f *os.File, context string) error {
	return errors.Join(err, closeFile(f, context))
}

// Bookmarks returns rs's bookmark hierarchy and supports cancellation.
func Bookmarks(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (bms []pdfcpu.Bookmark, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.LISTBOOKMARKS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return nil, bookmarkOpError("list bookmarks", err)
	}
	bms, err = pdfcpu.Bookmarks(c, ctx)
	return bms, bookmarkOpError("list bookmarks", err)
}

// ListBookmarks returns a formatted list of rs's bookmarks and supports cancellation.
func ListBookmarks(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (ss []string, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}
	conf = operationConfiguration(conf, model.LISTBOOKMARKS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return nil, bookmarkOpError("list bookmarks", err)
	}
	ss, err = pdfcpu.BookmarkList(c, ctx)
	return ss, bookmarkOpError("list bookmarks", err)
}

// ListBookmarksFile returns a formatted list of inFile's bookmarks and supports cancellation.
func ListBookmarksFile(c context.Context, inFile string, conf *model.Configuration) (ss []string, err error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if inFile == "" {
		return nil, ErrMissingPDFInput
	}
	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list bookmarks: open input %s: %w", inFile, err)
	}
	defer func() {
		err = closeBookmarkInput(err, f, "list bookmarks: close input")
	}()
	return ListBookmarks(c, f, conf)
}

// ExportBookmarksJSON extracts bookmark data from rs, writes the result to w and supports cancellation.
func ExportBookmarksJSON(c context.Context, rs io.ReadSeeker, w io.Writer, source string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingJSONWriter
	}

	conf = operationConfiguration(conf, model.EXPORTBOOKMARKS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return bookmarkSourceError("export bookmarks", source, err)
	}

	ok, err := pdfcpu.ExportBookmarksJSON(c, ctx, source, w)
	if err != nil {
		return bookmarkSourceError("export bookmarks", source, err)
	}
	if !ok {
		return bookmarkSourceError("export bookmarks", source, ErrNoBookmarks)
	}

	return nil
}

// ExportBookmarksFile extracts bookmark data from inFilePDF, writes it to outFileJSON and supports cancellation.
func ExportBookmarksFile(c context.Context, inFilePDF, outFileJSON string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFilePDF == "" {
		return ErrMissingPDFInput
	}

	if outFileJSON == "" {
		return ErrMissingJSONOutput
	}

	if f1, err = os.Open(inFilePDF); err != nil {
		return fmt.Errorf("export bookmarks: open %s: %w", inFilePDF, err)
	}

	staged, err := openStagedOutput(f1, inFilePDF, outFileJSON, "export bookmarks")
	if err != nil {
		return errors.Join(
			fmt.Errorf("export bookmarks: create output %s: %w", outFileJSON, err),
			closeFile(f1, "export bookmarks: close input"),
		)
	}
	f2 = staged.output.file

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = ExportBookmarksJSON(c, f1, f2, inFilePDF, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

// ImportBookmarks creates or replaces bookmarks in rs, writes the result to w and supports cancellation.
func ImportBookmarks(c context.Context, rs io.ReadSeeker, rd io.Reader, w io.Writer, replace bool, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if rd == nil {
		return ErrMissingJSONReader
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	conf = operationConfiguration(conf, model.IMPORTBOOKMARKS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return bookmarkOpError("import bookmarks", err)
	}

	ok, err := pdfcpu.ImportBookmarks(c, ctx, rd, replace)
	if err != nil {
		return bookmarkOpError("import bookmarks", err)
	}
	if !ok {
		return bookmarkOpError("import bookmarks", ErrExistingBookmarks)
	}

	return bookmarkOpError("import bookmarks: write", WriteContext(c, ctx, w))
}

// ImportBookmarksFile creates or replaces bookmarks in inFilePDF, writes the result to outFilePDF and supports cancellation.
func ImportBookmarksFile(c context.Context, inFilePDF, inFileJSON, outFilePDF string, replace bool, conf *model.Configuration) (err error) {
	var f0, f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFilePDF == "" {
		return ErrMissingPDFInput
	}

	if inFileJSON == "" {
		return ErrMissingJSONInput
	}

	if f0, err = os.Open(inFilePDF); err != nil {
		return fmt.Errorf("import bookmarks: open %s: %w", inFilePDF, err)
	}

	if f1, err = os.Open(inFileJSON); err != nil {
		return errors.Join(
			fmt.Errorf("import bookmarks: open JSON %s: %w", inFileJSON, err),
			closeFile(f0, "import bookmarks: close input"),
		)
	}

	tmpFile := ""
	if outFilePDF != "" && inFilePDF != outFilePDF {
		tmpFile = outFilePDF
	}
	staged, err := openStagedOutput(f1, inFilePDF, tmpFile, "import bookmarks")
	if err != nil {
		return errors.Join(
			fmt.Errorf("import bookmarks: create output: %w", err),
			closeFile(f1, "import bookmarks: close JSON input"),
			closeFile(f0, "import bookmarks: close input"),
		)
	}
	f2 = staged.output.file
	staged.inputs[0].context = "import bookmarks: close JSON input"
	staged = staged.withInput(f0, "import bookmarks: close input")

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = ImportBookmarks(c, f0, f1, f2, replace, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

// AddBookmarks adds bookmarks to the PDF context read from rs, writes the result to w and supports cancellation.
func AddBookmarks(c context.Context, rs io.ReadSeeker, w io.Writer, bms []pdfcpu.Bookmark, replace bool, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	conf = operationConfiguration(conf, model.ADDBOOKMARKS)

	if len(bms) == 0 {
		return ErrMissingBookmarks
	}

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return bookmarkOpError("add bookmarks", err)
	}

	if err := pdfcpu.AddBookmarks(c, ctx, bms, replace); err != nil {
		return bookmarkOpError("add bookmarks", err)
	}

	return bookmarkOpError("add bookmarks: write", WriteContext(c, ctx, w))
}

// AddBookmarksFile adds bookmarks to the PDF context read from inFile, writes the result to outFile and supports cancellation.
func AddBookmarksFile(c context.Context, inFile, outFile string, bms []pdfcpu.Bookmark, replace bool, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("add bookmarks: open %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "add bookmarks")
	if err != nil {
		return errors.Join(
			fmt.Errorf("add bookmarks: create output: %w", err),
			closeFile(f1, "add bookmarks: close input"),
		)
	}
	f2 = staged.output.file

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = AddBookmarks(c, f1, f2, bms, replace, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

// RemoveBookmarks deletes bookmarks from rs, writes the result to w and supports cancellation.
func RemoveBookmarks(c context.Context, rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	conf = operationConfiguration(conf, model.REMOVEBOOKMARKS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return bookmarkOpError("remove bookmarks", err)
	}

	ok, err := pdfcpu.RemoveBookmarks(c, ctx)
	if err != nil {
		return bookmarkOpError("remove bookmarks", err)
	}
	if !ok {
		return bookmarkOpError("remove bookmarks", ErrNoBookmarks)
	}

	return bookmarkOpError("remove bookmarks: write", WriteContext(c, ctx, w))
}

// RemoveBookmarksFile deletes bookmarks from inFile, writes the result to outFile and supports cancellation.
func RemoveBookmarksFile(c context.Context, inFile, outFile string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("remove bookmarks: open %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "remove bookmarks")
	if err != nil {
		return errors.Join(
			fmt.Errorf("remove bookmarks: create output: %w", err),
			closeFile(f1, "remove bookmarks: close input"),
		)
	}
	f2 = staged.output.file

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = RemoveBookmarks(c, f1, f2, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}
