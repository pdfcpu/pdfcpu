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

// ErrInvalidPageLayout signals an unsupported page layout.
var ErrInvalidPageLayout = errors.New("invalid page layout")

func validPageLayout(pl model.PageLayout) bool {
	switch pl {
	case model.PageLayoutSinglePage,
		model.PageLayoutTwoColumnLeft,
		model.PageLayoutTwoColumnRight,
		model.PageLayoutTwoPageLeft,
		model.PageLayoutTwoPageRight,
		model.PageLayoutOneColumn:
		return true
	}
	return false
}

func invalidPageLayoutError(pl model.PageLayout) error {
	return fmt.Errorf("set page layout: invalid value %d: %w", pl, ErrInvalidPageLayout)
}

func closePageLayoutInput(err error, f *os.File, context string) error {
	return errors.Join(err, closeFile(f, context))
}

// PageLayout returns rs's page layout.
func PageLayout(rs io.ReadSeeker, conf *model.Configuration) (pl *model.PageLayout, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTPAGELAYOUT

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("list page layout: prepare PDF context: %w", err)
	}

	return ctx.PageLayout, nil
}

// PageLayoutFile returns inFile's page layout.
func PageLayoutFile(inFile string, conf *model.Configuration) (pl *model.PageLayout, err error) {
	if inFile == "" {
		return nil, ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list page layout: open input %s: %w", inFile, err)
	}
	defer func() {
		err = closePageLayoutInput(err, f, "list page layout: close input")
	}()

	return PageLayout(f, conf)
}

// ListPageLayout lists rs's page layout.
func ListPageLayout(rs io.ReadSeeker, conf *model.Configuration) (ss []string, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTPAGELAYOUT

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("list page layout: prepare PDF context: %w", err)
	}

	if ctx.PageLayout != nil {
		return []string{ctx.PageLayout.String()}, nil
	}

	return []string{"No page layout set, PDF viewers will default to \"SinglePage\""}, nil
}

// ListPageLayoutFile lists inFile's page layout.
func ListPageLayoutFile(inFile string, conf *model.Configuration) (ss []string, err error) {
	if inFile == "" {
		return nil, ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list page layout: open input %s: %w", inFile, err)
	}
	defer func() {
		err = closePageLayoutInput(err, f, "list page layout: close input")
	}()

	return ListPageLayout(f, conf)
}

// SetPageLayout sets rs's page layout and writes the result to w.
func SetPageLayout(rs io.ReadSeeker, w io.Writer, val model.PageLayout, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if !validPageLayout(val) {
		return invalidPageLayoutError(val)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.SETPAGELAYOUT

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return fmt.Errorf("set page layout: prepare PDF context: %w", err)
	}

	ctx.RootDict["PageLayout"] = types.Name(val.String())

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("set page layout: write output: %w", err)
	}
	return nil
}

// SetPageLayoutFile sets inFile's page layout and writes the result to outFile.
func SetPageLayoutFile(inFile, outFile string, val model.PageLayout, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}

	if !validPageLayout(val) {
		return invalidPageLayoutError(val)
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("set page layout: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "set page layout")
	if err != nil {
		return errors.Join(
			fmt.Errorf("set page layout: create output: %w", err),
			closeFile(f1, "set page layout: close input"),
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

	if err = SetPageLayout(f1, f2, val, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// ResetPageLayout resets rs's page layout and writes the result to w.
// It is idempotent and writes output even when rs has no page layout.
func ResetPageLayout(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
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
	conf.Cmd = model.RESETPAGELAYOUT

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return fmt.Errorf("reset page layout: prepare PDF context: %w", err)
	}

	delete(ctx.RootDict, "PageLayout")

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("reset page layout: write output: %w", err)
	}
	return nil
}

// ResetPageLayoutFile resets inFile's page layout and writes the result to outFile.
// It is idempotent and writes output even when inFile has no page layout.
func ResetPageLayoutFile(inFile, outFile string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("reset page layout: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "reset page layout")
	if err != nil {
		return errors.Join(
			fmt.Errorf("reset page layout: create output: %w", err),
			closeFile(f1, "reset page layout: close input"),
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

	if err = ResetPageLayout(f1, f2, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
