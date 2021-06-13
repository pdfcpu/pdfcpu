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
	"github.com/pkg/errors"
)

// ListAnnotations returns a list of page annotations of rs.
func ListAnnotations(rs io.ReadSeeker, selectedPages []string, conf *pdfcpu.Configuration) (int, []string, error) {
	if rs == nil {
		return 0, nil, errors.New("pdfcpu: ListAnnotations: Please provide rs")
	}
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
		conf.Cmd = pdfcpu.LISTANNOTATIONS
	}
	ctx, _, _, _, err := readValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return 0, nil, err
	}
	if err := ctx.EnsurePageCount(); err != nil {
		return 0, nil, err
	}
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, false)
	if err != nil {
		return 0, nil, err
	}

	return ctx.ListAnnotations(pages)
}

// ListAnnotationsFile returns a list of page annotations of inFile.
func ListAnnotationsFile(inFile string, selectedPages []string, conf *pdfcpu.Configuration) (int, []string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()
	return ListAnnotations(f, selectedPages, conf)
}

// AddAnnotations adds annotations for selected pages in rs and writes the result to w.
func AddAnnotations(rs io.ReadSeeker, w io.Writer, selectedPages []string, ann pdfcpu.AnnotationRenderer, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
		conf.Cmd = pdfcpu.ADDANNOTATIONS
	}

	ctx, _, _, _, err := readValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	ok, err := ctx.AddAnnotations(pages, ann, false)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no annotations added")
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdfcpu.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// AddAnnotationsAsIncrement adds annotations for selected pages in rws and writes out a PDF increment.
func AddAnnotationsAsIncrement(rws io.ReadWriteSeeker, selectedPages []string, ar pdfcpu.AnnotationRenderer, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
		conf.Cmd = pdfcpu.ADDANNOTATIONS
	}

	ctx, _, _, err := readAndValidate(rws, conf, time.Now())
	if err != nil {
		return err
	}

	if *ctx.HeaderVersion < pdfcpu.V14 {
		return errors.New("Increment writing not supported for PDF version < V1.4 (Hint: Use pdfcpu optimize then try again)")
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	ok, err := ctx.AddAnnotations(pages, ar, true)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no annotations added")
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdfcpu.ValidationNone {
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
func AddAnnotationsFile(inFile, outFile string, selectedPages []string, ar pdfcpu.AnnotationRenderer, conf *pdfcpu.Configuration, incr bool) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

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

	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return AddAnnotations(f1, f2, selectedPages, ar, conf)
}

// AddAnnotationsMap adds annotations in m to corresponding pages of rs and writes the result to w.
func AddAnnotationsMap(rs io.ReadSeeker, w io.Writer, m map[int][]pdfcpu.AnnotationRenderer, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
		conf.Cmd = pdfcpu.ADDANNOTATIONS
	}

	ctx, _, _, _, err := readValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	ok, err := ctx.AddAnnotationsMap(m, false)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no annotations added")
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdfcpu.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// AddAnnotationsMapAsIncrement adds annotations in m to corresponding pages of rws and writes out a PDF increment.
func AddAnnotationsMapAsIncrement(rws io.ReadWriteSeeker, m map[int][]pdfcpu.AnnotationRenderer, conf *pdfcpu.Configuration) error {

	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
		conf.Cmd = pdfcpu.ADDANNOTATIONS
	}

	ctx, _, _, err := readAndValidate(rws, conf, time.Now())
	if err != nil {
		return err
	}

	if *ctx.HeaderVersion < pdfcpu.V14 {
		return errors.New("Increment writing not supported for PDF version < V1.4 (Hint: Use pdfcpu optimize then try again)")
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	ok, err := ctx.AddAnnotationsMap(m, true)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no annotations added")
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdfcpu.ValidationNone {
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
func AddAnnotationsMapFile(inFile, outFile string, m map[int][]pdfcpu.AnnotationRenderer, conf *pdfcpu.Configuration, incr bool) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

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

	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return AddAnnotationsMap(f1, f2, m, conf)
}

// RemoveAnnotations removes annotations for selected pages by id and object number
// from a PDF context read from rs and writes the result to w.
func RemoveAnnotations(rs io.ReadSeeker, w io.Writer, selectedPages, ids []string, objNrs []int, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
		conf.Cmd = pdfcpu.REMOVEANNOTATIONS
	}

	ctx, _, _, _, err := readValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	ok, err := ctx.RemoveAnnotations(pages, ids, objNrs, false)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no annotation removed")
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdfcpu.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// RemoveAnnotationsAsIncrement removes annotations for selected pages by ids and object number
// from a PDF context read from rs and writes out a PDF increment.
func RemoveAnnotationsAsIncrement(rws io.ReadWriteSeeker, selectedPages, ids []string, objNrs []int, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
		conf.Cmd = pdfcpu.REMOVEANNOTATIONS
	}

	ctx, _, _, err := readAndValidate(rws, conf, time.Now())
	if err != nil {
		return err
	}

	if *ctx.HeaderVersion < pdfcpu.V14 {
		return errors.New("Increment writing unsupported for PDF version < V1.4 (Hint: Use pdfcpu optimize then try again)")
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	ok, err := ctx.RemoveAnnotations(pages, ids, objNrs, true)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no annotation removed")
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdfcpu.ValidationNone {
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
func RemoveAnnotationsFile(inFile, outFile string, selectedPages, ids []string, objNrs []int, conf *pdfcpu.Configuration, incr bool) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	//fmt.Printf("RemoveAnnotationsFile: ids:%v objNrs:%v\n", ids, objNrs)

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
			return RemoveAnnotationsAsIncrement(f, selectedPages, ids, objNrs, conf)
		}
	}

	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return RemoveAnnotations(f1, f2, selectedPages, ids, objNrs, conf)
}
