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

// Annotations returns page annotations of rs for selected pages.
func Annotations(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (map[int]model.PgAnnots, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: Annotations: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTANNOTATIONS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return nil, err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return nil, err
	}

	return pdfcpu.AnnotationsForSelectedPages(ctx, pages), nil
}

// AddAnnotations adds annotations for selected pages in rs and writes the result to w.
func AddAnnotations(rs io.ReadSeeker, w io.Writer, selectedPages []string, ann model.AnnotationRenderer, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: AddAnnotations: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDANNOTATIONS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
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

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// AddAnnotationsAsIncrement adds annotations for selected pages in rws and writes out a PDF increment.
func AddAnnotationsAsIncrement(rws io.ReadWriteSeeker, selectedPages []string, ar model.AnnotationRenderer, conf *model.Configuration) error {
	if rws == nil {
		return errors.New("pdfcpu: AddAnnotationsAsIncrement: missing rws")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDANNOTATIONS

	ctx, _, _, err := readAndValidate(rws, conf, time.Now())
	if err != nil {
		return err
	}

	if *ctx.HeaderVersion < model.V14 {
		return errors.New("Incremental writing not supported for PDF version < V1.4 (Hint: Use pdfcpu optimize then try again)")
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
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

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	if _, err = rws.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	return WriteIncrement(ctx, rws)
}

// AddAnnotationsFile adds annotations for selected pages to a PDF context read from inFile and writes the result to outFile.
func AddAnnotationsFile(inFile, outFile string, selectedPages []string, ar model.AnnotationRenderer, conf *model.Configuration, incr bool) (err error) {

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
		if incr {
			f, err := os.OpenFile(inFile, os.O_RDWR, 0644)
			if err != nil {
				return err
			}
			defer f.Close()
			return AddAnnotationsAsIncrement(f, selectedPages, ar, conf)
		}
	}

	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
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

	return AddAnnotations(f1, f2, selectedPages, ar, conf)
}

// AddAnnotationsMap adds annotations in m to corresponding pages of rs and writes the result to w.
func AddAnnotationsMap(rs io.ReadSeeker, w io.Writer, m map[int][]model.AnnotationRenderer, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: AddAnnotationsMap: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDANNOTATIONS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	ok, err := pdfcpu.AddAnnotationsMap(ctx, m, false)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("pdfcpu: AddAnnotationsMap: No annotations added")
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// AddAnnotationsMapAsIncrement adds annotations in m to corresponding pages of rws and writes out a PDF increment.
func AddAnnotationsMapAsIncrement(rws io.ReadWriteSeeker, m map[int][]model.AnnotationRenderer, conf *model.Configuration) error {
	if rws == nil {
		return errors.New("pdfcpu: AddAnnotationsMapAsIncrement: missing rws")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDANNOTATIONS

	ctx, _, _, err := readAndValidate(rws, conf, time.Now())
	if err != nil {
		return err
	}

	if *ctx.HeaderVersion < model.V14 {
		return errors.New("Increment writing not supported for PDF version < V1.4 (Hint: Use pdfcpu optimize then try again)")
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	ok, err := pdfcpu.AddAnnotationsMap(ctx, m, true)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("pdfcpu: AddAnnotationsMapAsIncrement: No annotations added")
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	if _, err = rws.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	return WriteIncrement(ctx, rws)
}

// AddAnnotationsMapFile adds annotations in m to corresponding pages of inFile and writes the result to outFile.
func AddAnnotationsMapFile(inFile, outFile string, m map[int][]model.AnnotationRenderer, conf *model.Configuration, incr bool) (err error) {
	tmpFile := inFile + ".tmp"

	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
		if incr {
			f, err := os.OpenFile(inFile, os.O_RDWR, 0644)
			if err != nil {
				return err
			}
			defer f.Close()
			return AddAnnotationsMapAsIncrement(f, m, conf)
		}
	}

	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
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

	return AddAnnotationsMap(f1, f2, m, conf)
}

// RemoveAnnotations removes annotations for selected pages by id and object number
// from a PDF context read from rs and writes the result to w.
func RemoveAnnotations(rs io.ReadSeeker, w io.Writer, selectedPages, idsAndTypes []string, objNrs []int, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: RemoveAnnotations: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEANNOTATIONS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
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

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// RemoveAnnotationsAsIncrement removes annotations for selected pages by ids and object number
// from a PDF context read from rs and writes out a PDF increment.
func RemoveAnnotationsAsIncrement(rws io.ReadWriteSeeker, selectedPages, idsAndTypes []string, objNrs []int, conf *model.Configuration) error {
	if rws == nil {
		return errors.New("pdfcpu: RemoveAnnotationsAsIncrement: missing rws")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEANNOTATIONS

	ctx, _, _, err := readAndValidate(rws, conf, time.Now())
	if err != nil {
		return err
	}

	if *ctx.HeaderVersion < model.V14 {
		return errors.New("pdfcpu: Incremental writing unsupported for PDF version < V1.4 (Hint: Use pdfcpu optimize then try again)")
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
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

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	if _, err = rws.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	return WriteIncrement(ctx, rws)
}

// RemoveAnnotationsFile removes annotations for selected pages by id and object number
// from a PDF context read from inFile and writes the result to outFile.
func RemoveAnnotationsFile(inFile, outFile string, selectedPages, idsAndTypes []string, objNrs []int, conf *model.Configuration, incr bool) (err error) {

	var f1, f2 *os.File

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
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

	return RemoveAnnotations(f1, f2, selectedPages, idsAndTypes, objNrs, conf)
}
