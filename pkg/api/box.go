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
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// PageBoundariesFromBoxList parses a list of box types.
func PageBoundariesFromBoxList(s string) (*model.PageBoundaries, error) {
	pb, err := model.ParseBoxList(s)
	if err != nil {
		return nil, fmt.Errorf("parse box list: %w", err)
	}
	return pb, nil
}

// PageBoundaries parses a list of box definitions and assignments.
func PageBoundaries(s string, unit types.DisplayUnit) (*model.PageBoundaries, error) {
	pb, err := model.ParsePageBoundaries(s, unit)
	if err != nil {
		return nil, fmt.Errorf("parse page boundaries: %w", err)
	}
	return pb, nil
}

// Box parses a box definition.
func Box(s string, u types.DisplayUnit) (*model.Box, error) {
	b, err := model.ParseBox(s, u)
	if err != nil {
		return nil, fmt.Errorf("parse box: %w", err)
	}
	return b, nil
}

func prepareBoxListing(c context.Context, rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (*model.Context, types.IntSet, error) {
	conf = operationConfiguration(conf, model.LISTBOXES)
	ctx, err := ReadAndValidate(c, rs, conf)
	if err != nil {
		return nil, nil, fmt.Errorf("list boxes: prepare PDF context: %w", err)
	}
	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return nil, nil, fmt.Errorf("list boxes: parse page selection: %w", err)
	}
	return ctx, pages, nil
}

// Boxes returns rs's page boundaries for selected pages and supports cancellation.
func Boxes(c context.Context, rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (pb []model.PageBoundaries, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	ctx, pages, err := prepareBoxListing(c, rs, selectedPages, conf)
	if err != nil {
		return nil, err
	}
	pb, err = ctx.PageBoundaries(c, pages)
	if err != nil {
		return nil, fmt.Errorf("list boxes: inspect page boundaries: %w", err)
	}
	return pb, nil
}

// ListBoxes returns formatted page boundaries for selected pages of rs and supports cancellation.
func ListBoxes(c context.Context, rs io.ReadSeeker, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (ss []string, err error) {
	defer fault.Catch(&err)
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}
	if pb == nil {
		pb = &model.PageBoundaries{}
		pb.SelectAll()
	}
	ctx, pages, err := prepareBoxListing(c, rs, selectedPages, conf)
	if err != nil {
		return nil, err
	}
	ss, err = ctx.ListPageBoundaries(c, pages, pb)
	if err != nil {
		return nil, fmt.Errorf("list boxes: format page boundaries: %w", err)
	}
	return ss, nil
}

func closeBoxInput(err error, f *os.File, context string) error {
	return errors.Join(err, closeFile(f, context))
}

// ListBoxesFile returns formatted page boundaries for selected pages of inFile and supports cancellation.
func ListBoxesFile(c context.Context, inFile string, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (ss []string, err error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if inFile == "" {
		return nil, ErrMissingPDFInput
	}
	if pb == nil {
		pb = &model.PageBoundaries{}
		pb.SelectAll()
	}
	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list boxes: open input %s: %w", inFile, err)
	}
	defer func() {
		err = closeBoxInput(err, f, "list boxes: close input")
	}()
	return ListBoxes(c, f, selectedPages, pb, conf)
}

func validatePageBoundariesRequest(pb *model.PageBoundaries, remove bool) error {
	if pb == nil {
		return ErrMissingPageBoundaries
	}
	if pb.Media == nil && pb.Crop == nil && pb.Trim == nil && pb.Bleed == nil && pb.Art == nil {
		return fmt.Errorf("empty request: %w", ErrInvalidPageBoundaries)
	}
	if remove && pb.Media != nil {
		return fmt.Errorf("MediaBox removal: %w", ErrInvalidPageBoundaries)
	}
	return nil
}

// AddBoxes adds page boundaries for selected pages of rs, writes the result to w and supports cancellation.
func AddBoxes(c context.Context, rs io.ReadSeeker, w io.Writer, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (err error) {
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
	if err := validatePageBoundariesRequest(pb, false); err != nil {
		return fmt.Errorf("add boxes: validate page boundaries: %w", err)
	}

	conf = operationConfiguration(conf, model.ADDBOXES)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("add boxes: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("add boxes: parse page selection: %w", err)
	}

	if err = ctx.AddPageBoundaries(c, pages, pb); err != nil {
		return fmt.Errorf("add boxes: apply page boundaries: %w", err)
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("add boxes: write output: %w", err)
	}
	return nil
}

// AddBoxesFile adds page boundaries for selected pages of inFile, writes the result to outFile and supports cancellation.
func AddBoxesFile(c context.Context, inFile, outFile string, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := validatePageBoundariesRequest(pb, false); err != nil {
		return fmt.Errorf("add boxes: validate page boundaries: %w", err)
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	f1, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("add boxes: open input %s: %w", inFile, err)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "add boxes")
	if err != nil {
		return errors.Join(fmt.Errorf("add boxes: create output: %w", err), closeFile(f1, "add boxes: close input"))
	}
	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = AddBoxes(c, f1, staged.output.file, selectedPages, pb, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}

// RemoveBoxes removes page boundaries for selected pages of rs, writes the result to w and supports cancellation.
func RemoveBoxes(c context.Context, rs io.ReadSeeker, w io.Writer, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (err error) {
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
	if err := validatePageBoundariesRequest(pb, true); err != nil {
		return fmt.Errorf("remove boxes: validate page boundaries: %w", err)
	}

	conf = operationConfiguration(conf, model.REMOVEBOXES)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("remove boxes: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("remove boxes: parse page selection: %w", err)
	}

	if err = ctx.RemovePageBoundaries(c, pages, pb); err != nil {
		return fmt.Errorf("remove boxes: remove page boundaries: %w", err)
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("remove boxes: write output: %w", err)
	}
	return nil
}

// RemoveBoxesFile removes page boundaries for selected pages of inFile, writes the result to outFile and supports cancellation.
func RemoveBoxesFile(c context.Context, inFile, outFile string, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := validatePageBoundariesRequest(pb, true); err != nil {
		return fmt.Errorf("remove boxes: validate page boundaries: %w", err)
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	f1, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("remove boxes: open input %s: %w", inFile, err)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "remove boxes")
	if err != nil {
		return errors.Join(fmt.Errorf("remove boxes: create output: %w", err), closeFile(f1, "remove boxes: close input"))
	}
	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = RemoveBoxes(c, f1, staged.output.file, selectedPages, pb, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}

// Crop adds crop boxes for selected pages of rs, writes the result to w and supports cancellation.
func Crop(c context.Context, rs io.ReadSeeker, w io.Writer, selectedPages []string, b *model.Box, conf *model.Configuration) (err error) {
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
	if b == nil {
		return ErrMissingBoxConfiguration
	}

	conf = operationConfiguration(conf, model.CROP)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("crop: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("crop: parse page selection: %w", err)
	}

	if err = ctx.Crop(c, pages, b); err != nil {
		return fmt.Errorf("crop: apply crop box: %w", err)
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("crop: write output: %w", err)
	}
	return nil
}

// CropFile adds crop boxes for selected pages of inFile, writes the result to outFile and supports cancellation.
func CropFile(c context.Context, inFile, outFile string, selectedPages []string, b *model.Box, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if b == nil {
		return ErrMissingBoxConfiguration
	}

	f1, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("crop: open input %s: %w", inFile, err)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "crop")
	if err != nil {
		return errors.Join(fmt.Errorf("crop: create output: %w", err), closeFile(f1, "crop: close input"))
	}
	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = Crop(c, f1, staged.output.file, selectedPages, b, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}
