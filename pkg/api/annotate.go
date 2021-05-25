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

	if err = ctx.AddAnnotations(pages, ann, false); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdfcpu.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

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

	if err = ctx.AddAnnotations(pages, ar, true); err != nil {
		return err
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

func RemoveAnnotations(rs io.ReadSeeker, w io.Writer, selectedPages []string, id string, conf *pdfcpu.Configuration) error {
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

	ok, err := ctx.RemoveAnnotations(pages, id, false)
	if err != nil {
		return err
	}
	if !ok {
		// None removed.
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdfcpu.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

func RemoveAnnotationsAsIncrement(rws io.ReadWriteSeeker, selectedPages []string, id string, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
		conf.Cmd = pdfcpu.REMOVEANNOTATIONS
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

	ok, err := ctx.RemoveAnnotations(pages, id, true)
	if err != nil {
		return err
	}
	if !ok {
		// None removed.
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

// child annotations get removed automatically.
func RemoveAnnotationsFile(inFile, outFile string, selectedPages []string, id string, conf *pdfcpu.Configuration, incr bool) (err error) {
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
			return RemoveAnnotationsAsIncrement(f, selectedPages, id, conf)
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

	return RemoveAnnotations(f1, f2, selectedPages, id, conf)
}

// List annotation only if V >= 1.4
