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

func validPageMode(pm model.PageMode) bool {
	return pm >= model.PageModeUseNone && pm <= model.PageModeUseAttachments
}

func invalidPageModeError(pm model.PageMode) error {
	return fmt.Errorf("set page mode: invalid value %d: %w", pm, ErrInvalidPageMode)
}

func closePageModeInput(err error, f *os.File, context string) error {
	return errors.Join(err, closeFile(f, context))
}

// PageMode returns rs's page mode and supports cancellation.
func PageMode(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (pm *model.PageMode, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.LISTPAGEMODE)

	ctx, err := ReadAndValidate(c, rs, conf)
	if err != nil {
		return nil, fmt.Errorf("list page mode: prepare PDF context: %w", err)
	}

	return ctx.PageMode, contextutil.Check(c)
}

// PageModeFile returns inFile's page mode and supports cancellation.
func PageModeFile(c context.Context, inFile string, conf *model.Configuration) (pm *model.PageMode, err error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
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

	return PageMode(c, f, conf)
}

// ListPageMode lists rs's page mode and supports cancellation.
func ListPageMode(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (ss []string, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.LISTPAGEMODE)

	ctx, err := ReadAndValidate(c, rs, conf)
	if err != nil {
		return nil, fmt.Errorf("list page mode: prepare PDF context: %w", err)
	}

	if ctx.PageMode != nil {
		return []string{ctx.PageMode.String()}, nil
	}

	return []string{"No page mode set, PDF viewers will default to \"UseNone\""}, nil
}

// ListPageModeFile lists inFile's page mode and supports cancellation.
func ListPageModeFile(c context.Context, inFile string, conf *model.Configuration) (ss []string, err error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
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

	return ListPageMode(c, f, conf)
}

// SetPageMode sets rs's page mode, writes the result to w and supports cancellation.
func SetPageMode(c context.Context, rs io.ReadSeeker, w io.Writer, val model.PageMode, conf *model.Configuration) (err error) {
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

	if !validPageMode(val) {
		return invalidPageModeError(val)
	}

	conf = operationConfiguration(conf, model.SETPAGEMODE)

	ctx, err := ReadAndValidate(c, rs, conf)
	if err != nil {
		return fmt.Errorf("set page mode: prepare PDF context: %w", err)
	}

	ctx.RootDict["PageMode"] = types.Name(val.String())

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("set page mode: write output: %w", err)
	}
	return nil
}

// SetPageModeFile sets inFile's page mode, writes the result to outFile and supports cancellation.
func SetPageModeFile(c context.Context, inFile, outFile string, val model.PageMode, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	if err = SetPageMode(c, f1, f2, val, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

// ResetPageMode resets rs's page mode, writes the result to w and supports cancellation.
// It is idempotent and writes output even when rs has no page mode.
func ResetPageMode(c context.Context, rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
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

	conf = operationConfiguration(conf, model.RESETPAGEMODE)

	ctx, err := ReadAndValidate(c, rs, conf)
	if err != nil {
		return fmt.Errorf("reset page mode: prepare PDF context: %w", err)
	}

	delete(ctx.RootDict, "PageMode")

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("reset page mode: write output: %w", err)
	}
	return nil
}

// ResetPageModeFile resets inFile's page mode, writes the result to outFile and supports cancellation.
// It is idempotent and writes output even when inFile has no page mode.
func ResetPageModeFile(c context.Context, inFile, outFile string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	if err = ResetPageMode(c, f1, f2, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}
