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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// Annotations returns page annotations of rs for selected pages.
func Annotations(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (m map[int]model.PgAnnots, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, errors.New("pdfcpu: Annotations: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTANNOTATIONS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return nil, err
	}

	return pdfcpu.AnnotationsForSelectedPages(ctx, pages), nil
}

// AddAnnotations adds annotations for selected pages in rs and writes the result to w.
func AddAnnotations(rs io.ReadSeeker, w io.Writer, selectedPages []string, ann model.AnnotationRenderer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: AddAnnotations: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDANNOTATIONS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	ok, err := pdfcpu.AddAnnotations(ctx, pages, ann, false)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("pdfcpu: AddAnnotations: No annotations added")
	}

	return Write(ctx, w, conf)
}

// AddAnnotationsAsIncrement adds annotations for selected pages in rws and writes out a PDF increment.
func AddAnnotationsAsIncrement(rws io.ReadWriteSeeker, selectedPages []string, ar model.AnnotationRenderer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rws == nil {
		return errors.New("pdfcpu: AddAnnotationsAsIncrement: missing rws")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDANNOTATIONS

	ctx, err := ReadAndValidate(rws, conf)
	if err != nil {
		return err
	}

	if *ctx.HeaderVersion < model.V14 {
		return errors.New("Incremental writing not supported for PDF version < V1.4 (Hint: Use pdfcpu optimize then try again)")
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	ok, err := pdfcpu.AddAnnotations(ctx, pages, ar, true)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("pdfcpu: AddAnnotationsAsIncrement: No annotations added")
	}

	return WriteIncr(ctx, rws, conf)
}

// AddAnnotationsFile adds annotations for selected pages to a PDF context read from inFile and writes the result to outFile.
func AddAnnotationsFile(inFile, outFile string, selectedPages []string, ar model.AnnotationRenderer, conf *model.Configuration, incr bool) (err error) {
	var f1, f2 *os.File
	ok := false

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
		if incr {
			f, err := os.OpenFile(inFile, os.O_RDWR, 0644)
			if err != nil {
				return err
			}
			defer f.Close()
			return AddAnnotationsAsIncrement(f, selectedPages, ar, conf)
		}
	}

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			_ = os.Remove(tmpFile)
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

	if err = AddAnnotations(f1, f2, selectedPages, ar, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// AddAnnotationsMap adds annotations in m to corresponding pages of rs and writes the result to w.
func AddAnnotationsMap(rs io.ReadSeeker, w io.Writer, m map[int][]model.AnnotationRenderer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: AddAnnotationsMap: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDANNOTATIONS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	ok, err := pdfcpu.AddAnnotationsMap(ctx, m, false)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("pdfcpu: AddAnnotationsMap: No annotations added")
	}

	return Write(ctx, w, conf)
}

// AddAnnotationsMapAsIncrement adds annotations in m to corresponding pages of rws and writes out a PDF increment.
func AddAnnotationsMapAsIncrement(rws io.ReadWriteSeeker, m map[int][]model.AnnotationRenderer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rws == nil {
		return errors.New("pdfcpu: AddAnnotationsMapAsIncrement: missing rws")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDANNOTATIONS

	ctx, err := ReadAndValidate(rws, conf)
	if err != nil {
		return err
	}

	if *ctx.HeaderVersion < model.V14 {
		return errors.New("Increment writing not supported for PDF version < V1.4 (Hint: Use pdfcpu optimize then try again)")
	}

	ok, err := pdfcpu.AddAnnotationsMap(ctx, m, true)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("pdfcpu: AddAnnotationsMapAsIncrement: No annotations added")
	}

	return WriteIncr(ctx, rws, conf)
}

// AddAnnotationsMapFile adds annotations in m to corresponding pages of inFile and writes the result to outFile.
func AddAnnotationsMapFile(inFile, outFile string, m map[int][]model.AnnotationRenderer, conf *model.Configuration, incr bool) (err error) {
	var f1, f2 *os.File
	ok := false

	tmpFile := ""

	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
		if incr {
			f, err := os.OpenFile(inFile, os.O_RDWR, 0644)
			if err != nil {
				return err
			}
			defer f.Close()
			return AddAnnotationsMapAsIncrement(f, m, conf)
		}
	}

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			_ = os.Remove(tmpFile)
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

	if err = AddAnnotationsMap(f1, f2, m, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// RemoveAnnotations removes annotations for selected pages by id and object number
// from a PDF context read from rs and writes the result to w.
func RemoveAnnotations(rs io.ReadSeeker, w io.Writer, selectedPages, idsAndTypes []string, objNrs []int, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: RemoveAnnotations: missing rs")
	}
	if err := validateNoEmptyStrings(idsAndTypes, "annotation ID or type"); err != nil {
		return err
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEANNOTATIONS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	ok, err := pdfcpu.RemoveAnnotations(ctx, pages, idsAndTypes, objNrs, false)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("pdfcpu: RemoveAnnotations: No annotation removed")
	}

	return Write(ctx, w, conf)
}

// RemoveAnnotationsAsIncrement removes annotations for selected pages by ids and object number
// from a PDF context read from rs and writes out a PDF increment.
func RemoveAnnotationsAsIncrement(rws io.ReadWriteSeeker, selectedPages, idsAndTypes []string, objNrs []int, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rws == nil {
		return errors.New("pdfcpu: RemoveAnnotationsAsIncrement: missing rws")
	}
	if err := validateNoEmptyStrings(idsAndTypes, "annotation ID or type"); err != nil {
		return err
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEANNOTATIONS

	ctx, err := ReadAndValidate(rws, conf)
	if err != nil {
		return err
	}

	if *ctx.HeaderVersion < model.V14 {
		return errors.New("pdfcpu: Incremental writing unsupported for PDF version < V1.4 (Hint: Use pdfcpu optimize then try again)")
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	ok, err := pdfcpu.RemoveAnnotations(ctx, pages, idsAndTypes, objNrs, true)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("pdfcpu: RemoveAnnotationsAsIncrement: No annotation removed")
	}

	return WriteIncr(ctx, rws, conf)
}

// RemoveAnnotationsFile removes annotations for selected pages by id and object number
// from a PDF context read from inFile and writes the result to outFile.
func RemoveAnnotationsFile(inFile, outFile string, selectedPages, idsAndTypes []string, objNrs []int, conf *model.Configuration, incr bool) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := validateNoEmptyStrings(idsAndTypes, "annotation ID or type"); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
		if incr {
			if f1, err = os.OpenFile(inFile, os.O_RDWR, 0644); err != nil {
				return err
			}
			defer func() {
				cerr := f1.Close()
				if err == nil {
					err = cerr
				}
			}()
			return RemoveAnnotationsAsIncrement(f1, selectedPages, idsAndTypes, objNrs, conf)
		}
	}

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			_ = os.Remove(tmpFile)
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

	if err = RemoveAnnotations(f1, f2, selectedPages, idsAndTypes, objNrs, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
