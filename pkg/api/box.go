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
	"io"
	"os"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// PageBoundariesFromBoxList parses a list of box types.
func PageBoundariesFromBoxList(s string) (*model.PageBoundaries, error) {
	return model.ParseBoxList(s)
}

// PageBoundaries parses a list of box definitions and assignments.
func PageBoundaries(s string, unit types.DisplayUnit) (*model.PageBoundaries, error) {
	return model.ParsePageBoundaries(s, unit)
}

// Box parses a box definition.
func Box(s string, u types.DisplayUnit) (*model.Box, error) {
	return model.ParseBox(s, u)
}

// Boxes returns rs's page boundaries for selected pages of rs.
func Boxes(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) ([]model.PageBoundaries, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: Boxes: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTBOXES

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

	return ctx.PageBoundaries(pages)
}

// AddBoxes adds page boundaries for selected pages of rs and writes result to w.
func AddBoxes(rs io.ReadSeeker, w io.Writer, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: AddBoxes: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDBOXES

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

	if err = ctx.AddPageBoundaries(pages, pb); err != nil {
		return err
	}

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// AddBoxesFile adds page boundaries for selected pages of inFile and writes result to outFile.
func AddBoxesFile(inFile, outFile string, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	if log.CLIEnabled() {
		log.CLI.Printf("adding %s for %s\n", pb, inFile)
	}

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}

	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
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
			err = os.Rename(tmpFile, inFile)
		}
	}()

	return AddBoxes(f1, f2, selectedPages, pb, conf)
}

// RemoveBoxes removes page boundaries as specified in pb for selected pages of rs and writes result to w.
func RemoveBoxes(rs io.ReadSeeker, w io.Writer, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: RemoveBoxes: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEBOXES

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

	if err = ctx.RemovePageBoundaries(pages, pb); err != nil {
		return err
	}

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// RemoveBoxesFile removes page boundaries as specified in pb for selected pages of inFile and writes result to outFile.
func RemoveBoxesFile(inFile, outFile string, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if log.CLIEnabled() {
		log.CLI.Printf("removing %s for %s\n", pb, inFile)
	}

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}

	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
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
			err = os.Rename(tmpFile, inFile)
		}
	}()

	return RemoveBoxes(f1, f2, selectedPages, pb, conf)
}

// Crop adds crop boxes for selected pages of rs and writes result to w.
func Crop(rs io.ReadSeeker, w io.Writer, selectedPages []string, b *model.Box, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: Crop: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CROP

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

	if err = ctx.Crop(pages, b); err != nil {
		return err
	}

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// CropFile adds crop boxes for selected pages of inFile and writes result to outFile.
func CropFile(inFile, outFile string, selectedPages []string, b *model.Box, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if log.CLIEnabled() {
		log.CLI.Printf("cropping %s\n", inFile)
	}

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}

	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
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
			err = os.Rename(tmpFile, inFile)
		}
	}()

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CROP

	return Crop(f1, f2, selectedPages, b, conf)
}
