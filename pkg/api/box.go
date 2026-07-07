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

	"github.com/pdfcpu/pdfcpu/pkg/log"
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

func prepareBoxListing(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (*model.Context, types.IntSet, error) {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTBOXES
	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, nil, fmt.Errorf("list boxes: prepare PDF context: %w", err)
	}
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return nil, nil, fmt.Errorf("list boxes: parse page selection: %w", err)
	}
	return ctx, pages, nil
}

// Boxes returns rs's page boundaries for selected pages of rs.
func Boxes(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (pb []model.PageBoundaries, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	ctx, pages, err := prepareBoxListing(rs, selectedPages, conf)
	if err != nil {
		return nil, err
	}
	pb, err = ctx.PageBoundaries(pages)
	if err != nil {
		return nil, fmt.Errorf("list boxes: inspect page boundaries: %w", err)
	}
	return pb, nil
}

// ListBoxes returns formatted page boundaries for selected pages of rs.
func ListBoxes(rs io.ReadSeeker, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (ss []string, err error) {
	defer fault.Catch(&err)
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}
	if pb == nil {
		pb = &model.PageBoundaries{}
		pb.SelectAll()
	}
	ctx, pages, err := prepareBoxListing(rs, selectedPages, conf)
	if err != nil {
		return nil, err
	}
	ss, err = ctx.ListPageBoundaries(pages, pb)
	if err != nil {
		return nil, fmt.Errorf("list boxes: format page boundaries: %w", err)
	}
	return ss, nil
}

func closeBoxInput(err error, f *os.File, context string) error {
	return errors.Join(err, closeFile(f, context))
}

func processBoxFile(inFile, outFile, op string, process func(io.ReadSeeker, io.Writer) error) (err error) {
	var f1, f2 *os.File
	ok := false
	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("%s: open input %s: %w", op, inFile, err)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, op)
	if err != nil {
		return errors.Join(fmt.Errorf("%s: create output: %w", op, err), closeFile(f1, op+": close input"))
	}
	f2 = staged.output.file
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = process(f1, f2); err != nil {
		return err
	}
	ok = true
	return nil
}

// ListBoxesFile returns formatted page boundaries for selected pages of inFile.
func ListBoxesFile(inFile string, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (ss []string, err error) {
	if inFile == "" {
		return nil, ErrMissingPDFInput
	}
	if pb == nil {
		pb = &model.PageBoundaries{}
		pb.SelectAll()
	}
	if log.CLIEnabled() {
		log.CLI.Printf("listing %s for %s\n", pb, inFile)
	}
	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list boxes: open input %s: %w", inFile, err)
	}
	defer func() {
		err = closeBoxInput(err, f, "list boxes: close input")
	}()
	return ListBoxes(f, selectedPages, pb, conf)
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

// AddBoxes adds page boundaries for selected pages of rs and writes result to w.
func AddBoxes(rs io.ReadSeeker, w io.Writer, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}
	if err := validatePageBoundariesRequest(pb, false); err != nil {
		return fmt.Errorf("add boxes: validate page boundaries: %w", err)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDBOXES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("add boxes: %w", err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return fmt.Errorf("add boxes: parse page selection: %w", err)
	}

	if err = ctx.AddPageBoundaries(pages, pb); err != nil {
		return fmt.Errorf("add boxes: apply page boundaries: %w", err)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("add boxes: write output: %w", err)
	}
	return nil
}

// AddBoxesFile adds page boundaries for selected pages of inFile and writes result to outFile.
func AddBoxesFile(inFile, outFile string, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (err error) {
	if err := validatePageBoundariesRequest(pb, false); err != nil {
		return fmt.Errorf("add boxes: validate page boundaries: %w", err)
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if log.CLIEnabled() {
		log.CLI.Printf("adding %s for %s\n", pb, inFile)
	}
	logWritingTo(boxOutputFile(inFile, outFile))
	return processBoxFile(inFile, outFile, "add boxes", func(rs io.ReadSeeker, w io.Writer) error {
		return AddBoxes(rs, w, selectedPages, pb, conf)
	})
}

// RemoveBoxes removes page boundaries as specified in pb for selected pages of rs and writes result to w.
func RemoveBoxes(rs io.ReadSeeker, w io.Writer, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}
	if err := validatePageBoundariesRequest(pb, true); err != nil {
		return fmt.Errorf("remove boxes: validate page boundaries: %w", err)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEBOXES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("remove boxes: %w", err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return fmt.Errorf("remove boxes: parse page selection: %w", err)
	}

	if err = ctx.RemovePageBoundaries(pages, pb); err != nil {
		return fmt.Errorf("remove boxes: remove page boundaries: %w", err)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("remove boxes: write output: %w", err)
	}
	return nil
}

// RemoveBoxesFile removes page boundaries as specified in pb for selected pages of inFile and writes result to outFile.
func RemoveBoxesFile(inFile, outFile string, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (err error) {
	if err := validatePageBoundariesRequest(pb, true); err != nil {
		return fmt.Errorf("remove boxes: validate page boundaries: %w", err)
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if log.CLIEnabled() {
		log.CLI.Printf("removing %s for %s\n", pb, inFile)
	}
	logWritingTo(boxOutputFile(inFile, outFile))
	return processBoxFile(inFile, outFile, "remove boxes", func(rs io.ReadSeeker, w io.Writer) error {
		return RemoveBoxes(rs, w, selectedPages, pb, conf)
	})
}

// Crop adds crop boxes for selected pages of rs and writes result to w.
func Crop(rs io.ReadSeeker, w io.Writer, selectedPages []string, b *model.Box, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}
	if b == nil {
		return ErrMissingBoxConfiguration
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CROP

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("crop: %w", err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return fmt.Errorf("crop: parse page selection: %w", err)
	}

	if err = ctx.Crop(pages, b); err != nil {
		return fmt.Errorf("crop: apply crop box: %w", err)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("crop: write output: %w", err)
	}
	return nil
}

// CropFile adds crop boxes for selected pages of inFile and writes result to outFile.
func CropFile(inFile, outFile string, selectedPages []string, b *model.Box, conf *model.Configuration) (err error) {
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if b == nil {
		return ErrMissingBoxConfiguration
	}

	if log.CLIEnabled() {
		log.CLI.Printf("cropping %s\n", inFile)
	}
	logWritingTo(boxOutputFile(inFile, outFile))
	return processBoxFile(inFile, outFile, "crop", func(rs io.ReadSeeker, w io.Writer) error {
		return Crop(rs, w, selectedPages, b, conf)
	})
}

func boxOutputFile(inFile, outFile string) string {
	if outFile != "" && inFile != outFile {
		return outFile
	}
	return inFile
}
