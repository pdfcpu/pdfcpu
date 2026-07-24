/*
	Copyright 2023 The pdfcpu Authors.

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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validPageMode(pm model.PageMode) bool {
	return pm >= model.PageModeUseNone && pm <= model.PageModeUseAttachments
}

func invalidPageModeError(pm model.PageMode) error {
	return fmt.Errorf("set page mode: invalid value %d: %w", pm, ErrInvalidPageMode)
}

func closePageModeInput(err error, f *os.File, context string) error {
	return errors.Join(err, closeFile(f, context))
}

// PageMode returns rs's page mode.
func PageMode(rs io.ReadSeeker, conf *model.Configuration) (pm *model.PageMode, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTPAGEMODE

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("list page mode: prepare PDF context: %w", err)
	}

	return ctx.PageMode, nil
}

// PageModeFile returns inFile's page mode.
func PageModeFile(inFile string, conf *model.Configuration) (pm *model.PageMode, err error) {
	if inFile == "" {
		return nil, ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list page mode: open input %s: %w", inFile, err)
	}
	defer func() {
		err = closePageModeInput(err, f, "list page mode: close input")
	}()

	return PageMode(f, conf)
}

// ListPageMode lists rs's page mode.
func ListPageMode(rs io.ReadSeeker, conf *model.Configuration) (ss []string, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTPAGEMODE

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("list page mode: prepare PDF context: %w", err)
	}

	if ctx.PageMode != nil {
		return []string{ctx.PageMode.String()}, nil
	}

	return []string{"No page mode set, PDF viewers will default to \"UseNone\""}, nil
}

// ListPageModeFile lists inFile's page mode.
func ListPageModeFile(inFile string, conf *model.Configuration) (ss []string, err error) {
	if inFile == "" {
		return nil, ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list page mode: open input %s: %w", inFile, err)
	}
	defer func() {
		err = closePageModeInput(err, f, "list page mode: close input")
	}()

	return ListPageMode(f, conf)
}

// SetPageMode sets rs's page mode and writes the result to w.
func SetPageMode(rs io.ReadSeeker, w io.Writer, val model.PageMode, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if !validPageMode(val) {
		return invalidPageModeError(val)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.SETPAGEMODE

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return fmt.Errorf("set page mode: prepare PDF context: %w", err)
	}

	ctx.RootDict["PageMode"] = types.Name(val.String())

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("set page mode: write output: %w", err)
	}
	return nil
}

// SetPageModeFile sets inFile's page mode and writes the result to outFile.
func SetPageModeFile(inFile, outFile string, val model.PageMode, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}

	if !validPageMode(val) {
		return invalidPageModeError(val)
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("set page mode: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "set page mode")
	if err != nil {
		return errors.Join(
			fmt.Errorf("set page mode: create output: %w", err),
			closeFile(f1, "set page mode: close input"),
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

	if err = SetPageMode(f1, f2, val, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// ResetPageMode resets rs's page mode and writes the result to w.
// It is idempotent and writes output even when rs has no page mode.
func ResetPageMode(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.RESETPAGEMODE

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return fmt.Errorf("reset page mode: prepare PDF context: %w", err)
	}

	delete(ctx.RootDict, "PageMode")

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("reset page mode: write output: %w", err)
	}
	return nil
}

// ResetPageModeFile resets inFile's page mode and writes the result to outFile.
// It is idempotent and writes output even when inFile has no page mode.
func ResetPageModeFile(inFile, outFile string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("reset page mode: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "reset page mode")
	if err != nil {
		return errors.Join(
			fmt.Errorf("reset page mode: create output: %w", err),
			closeFile(f1, "reset page mode: close input"),
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

	if err = ResetPageMode(f1, f2, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
