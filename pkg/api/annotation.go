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
	"sort"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func validateAnnotationRenderer(ar model.AnnotationRenderer) error {
	if ar == nil {
		return ErrMissingAnnotation
	}
	return nil
}

func validateAnnotationRendererMap(m map[int][]model.AnnotationRenderer) error {
	pageNrs := make([]int, 0, len(m))
	for pageNr := range m {
		pageNrs = append(pageNrs, pageNr)
	}
	sort.Ints(pageNrs)
	for _, pageNr := range pageNrs {
		annots := m[pageNr]
		for i, ar := range annots {
			if ar == nil {
				return fmt.Errorf("page %d annotation %d: %w", pageNr, i+1, ErrMissingAnnotation)
			}
		}
	}
	return nil
}

// Annotations returns page annotations of rs for selected pages and supports cancellation.
func Annotations(c context.Context, rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (m map[int]model.PgAnnots, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.LISTANNOTATIONS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return nil, fmt.Errorf("list annotations: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return nil, fmt.Errorf("list annotations: parse page selection: %w", err)
	}

	return pdfcpu.AnnotationsForSelectedPages(c, ctx, pages)
}

// AddAnnotations adds annotations for selected pages in rs, writes the result to w and supports cancellation.
func AddAnnotations(c context.Context, rs io.ReadSeeker, w io.Writer, selectedPages []string, ann model.AnnotationRenderer, conf *model.Configuration) (err error) {
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

	if err := validateAnnotationRenderer(ann); err != nil {
		return err
	}

	conf = operationConfiguration(conf, model.ADDANNOTATIONS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("add annotations: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("add annotations: parse page selection: %w", err)
	}

	ok, err := pdfcpu.AddAnnotations(c, ctx, pages, ann, false)
	if err != nil {
		return fmt.Errorf("add annotations: add: %w", err)
	}
	if !ok {
		return errors.New("no annotations added")
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("add annotations: write output: %w", err)
	}
	return nil
}

// AddAnnotationsAsIncrement adds annotations for selected pages in rws, writes a PDF increment and supports cancellation.
func AddAnnotationsAsIncrement(c context.Context, rws io.ReadWriteSeeker, selectedPages []string, ar model.AnnotationRenderer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rws == nil {
		return ErrMissingPDFReadWriteSeeker
	}

	if err := validateAnnotationRenderer(ar); err != nil {
		return err
	}

	conf = operationConfiguration(conf, model.ADDANNOTATIONS)

	ctx, err := ReadAndValidate(c, rws, conf)
	if err != nil {
		return fmt.Errorf("add annotations: prepare PDF context: %w", err)
	}

	if *ctx.HeaderVersion < model.V14 {
		return errors.New("incremental writing not supported for PDF version < V1.4")
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("add annotations: parse page selection: %w", err)
	}

	ok, err := pdfcpu.AddAnnotations(c, ctx, pages, ar, true)
	if err != nil {
		return fmt.Errorf("add annotations: add: %w", err)
	}
	if !ok {
		return errors.New("no annotations added")
	}

	if err = WriteIncr(c, ctx, rws, conf); err != nil {
		return fmt.Errorf("add annotations: write increment: %w", err)
	}
	return nil
}

// AddAnnotationsFile adds annotations for selected pages to inFile, writes the result to outFile and supports cancellation.
func AddAnnotationsFile(c context.Context, inFile, outFile string, selectedPages []string, ar model.AnnotationRenderer, conf *model.Configuration, incr bool) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if err := validateAnnotationRenderer(ar); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	} else {
		if incr {
			return updateFileTransaction(c, inFile, "add annotations", func(c context.Context, f *os.File) error {
				return AddAnnotationsAsIncrement(c, f, selectedPages, ar, conf)
			})
		}
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("add annotations: open input %s: %w", inFile, err)
	}

	staged, err := openStagedOutput(f1, inFile, tmpFile, "add annotations")
	if err != nil {
		return errors.Join(
			fmt.Errorf("add annotations: create output: %w", err),
			closeFile(f1, "add annotations: close input"),
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

	if err = AddAnnotations(c, f1, f2, selectedPages, ar, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// AddAnnotationsMap adds annotations in m to corresponding pages of rs, writes the result to w and supports cancellation.
func AddAnnotationsMap(c context.Context, rs io.ReadSeeker, w io.Writer, m map[int][]model.AnnotationRenderer, conf *model.Configuration) (err error) {
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

	if err := validateAnnotationRendererMap(m); err != nil {
		return err
	}

	conf = operationConfiguration(conf, model.ADDANNOTATIONS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("add annotations: %w", err)
	}

	ok, err := pdfcpu.AddAnnotationsMap(c, ctx, m, false)
	if err != nil {
		return fmt.Errorf("add annotations: add: %w", err)
	}
	if !ok {
		return errors.New("no annotations added")
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("add annotations: write output: %w", err)
	}
	return nil
}

// AddAnnotationsMapAsIncrement adds annotations in m to corresponding pages of rws, writes a PDF increment and supports cancellation.
func AddAnnotationsMapAsIncrement(c context.Context, rws io.ReadWriteSeeker, m map[int][]model.AnnotationRenderer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rws == nil {
		return ErrMissingPDFReadWriteSeeker
	}

	if err := validateAnnotationRendererMap(m); err != nil {
		return err
	}

	conf = operationConfiguration(conf, model.ADDANNOTATIONS)

	ctx, err := ReadAndValidate(c, rws, conf)
	if err != nil {
		return fmt.Errorf("add annotations: prepare PDF context: %w", err)
	}

	if *ctx.HeaderVersion < model.V14 {
		return errors.New("incremental writing not supported for PDF version < V1.4")
	}

	ok, err := pdfcpu.AddAnnotationsMap(c, ctx, m, true)
	if err != nil {
		return fmt.Errorf("add annotations: add: %w", err)
	}
	if !ok {
		return errors.New("no annotations added")
	}

	if err = WriteIncr(c, ctx, rws, conf); err != nil {
		return fmt.Errorf("add annotations: write increment: %w", err)
	}
	return nil
}

// AddAnnotationsMapFile adds annotations in m to corresponding pages of inFile, writes the result to outFile and supports cancellation.
func AddAnnotationsMapFile(c context.Context, inFile, outFile string, m map[int][]model.AnnotationRenderer, conf *model.Configuration, incr bool) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if err := validateAnnotationRendererMap(m); err != nil {
		return err
	}

	tmpFile := ""

	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	} else {
		if incr {
			return updateFileTransaction(c, inFile, "add annotations", func(c context.Context, f *os.File) error {
				return AddAnnotationsMapAsIncrement(c, f, m, conf)
			})
		}
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("add annotations: open input %s: %w", inFile, err)
	}

	staged, err := openStagedOutput(f1, inFile, tmpFile, "add annotations")
	if err != nil {
		return errors.Join(
			fmt.Errorf("add annotations: create output: %w", err),
			closeFile(f1, "add annotations: close input"),
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

	if err = AddAnnotationsMap(c, f1, f2, m, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// RemoveAnnotations removes annotations for selected pages by ID and object number from a PDF context
// read from rs, writes the result to w and supports cancellation.
func RemoveAnnotations(c context.Context, rs io.ReadSeeker, w io.Writer, selectedPages, idsAndTypes []string, objNrs []int, conf *model.Configuration) (err error) {
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

	if err := validateNoEmptyStrings(idsAndTypes, "annotation ID or type"); err != nil {
		return err
	}

	conf = operationConfiguration(conf, model.REMOVEANNOTATIONS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("remove annotations: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("remove annotations: parse page selection: %w", err)
	}

	ok, err := pdfcpu.RemoveAnnotations(c, ctx, pages, idsAndTypes, objNrs, false)
	if err != nil {
		return fmt.Errorf("remove annotations: remove: %w", err)
	}
	if !ok {
		return errors.New("no annotation removed")
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("remove annotations: write output: %w", err)
	}
	return nil
}

// RemoveAnnotationsAsIncrement removes annotations for selected pages by IDs and object number from a PDF context
// read from rws, writes out a PDF increment and supports cancellation.
func RemoveAnnotationsAsIncrement(c context.Context, rws io.ReadWriteSeeker, selectedPages, idsAndTypes []string, objNrs []int, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rws == nil {
		return ErrMissingPDFReadWriteSeeker
	}

	if err := validateNoEmptyStrings(idsAndTypes, "annotation ID or type"); err != nil {
		return err
	}

	conf = operationConfiguration(conf, model.REMOVEANNOTATIONS)

	ctx, err := ReadAndValidate(c, rws, conf)
	if err != nil {
		return fmt.Errorf("remove annotations: prepare PDF context: %w", err)
	}

	if *ctx.HeaderVersion < model.V14 {
		return errors.New("incremental writing not supported for PDF version < V1.4")
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("remove annotations: parse page selection: %w", err)
	}

	ok, err := pdfcpu.RemoveAnnotations(c, ctx, pages, idsAndTypes, objNrs, true)
	if err != nil {
		return fmt.Errorf("remove annotations: remove: %w", err)
	}
	if !ok {
		return errors.New("no annotation removed")
	}

	if err = WriteIncr(c, ctx, rws, conf); err != nil {
		return fmt.Errorf("remove annotations: write increment: %w", err)
	}
	return nil
}

// RemoveAnnotationsFile removes annotations for selected pages by ID and object number
// from a PDF context read from inFile, writes the result to outFile and supports cancellation.
func RemoveAnnotationsFile(c context.Context, inFile, outFile string, selectedPages, idsAndTypes []string, objNrs []int, conf *model.Configuration, incr bool) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if err := validateNoEmptyStrings(idsAndTypes, "annotation ID or type"); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	} else {
		if incr {
			return updateFileTransaction(c, inFile, "remove annotations", func(c context.Context, f *os.File) error {
				return RemoveAnnotationsAsIncrement(c, f, selectedPages, idsAndTypes, objNrs, conf)
			})
		}
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("remove annotations: open input %s: %w", inFile, err)
	}

	staged, err := openStagedOutput(f1, inFile, tmpFile, "remove annotations")
	if err != nil {
		return errors.Join(
			fmt.Errorf("remove annotations: create output: %w", err),
			closeFile(f1, "remove annotations: close input"),
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

	if err = RemoveAnnotations(c, f1, f2, selectedPages, idsAndTypes, objNrs, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}
