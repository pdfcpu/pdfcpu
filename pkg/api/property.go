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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Properties returns rs's properties as recorded in infoDict and supports cancellation.
func Properties(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (m map[string]string, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.LISTPROPERTIES)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return nil, fmt.Errorf("list properties: %w", err)
	}

	return ctx.Properties, contextutil.Check(c)
}

// AddProperties adds properties to rs's infodict, writes the result to w and supports cancellation.
func AddProperties(c context.Context, rs io.ReadSeeker, w io.Writer, properties map[string]string, conf *model.Configuration) (err error) {
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

	if err := validateProperties(properties); err != nil {
		return fmt.Errorf("add properties: validate properties: %w", err)
	}

	conf = operationConfiguration(conf, model.ADDPROPERTIES)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("add properties: %w", err)
	}

	if err = pdfcpu.PropertiesAdd(c, ctx, properties); err != nil {
		return fmt.Errorf("add properties: update document properties: %w", err)
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("add properties: write output: %w", err)
	}
	return nil
}

// AddPropertiesFile adds properties to inFile's infodict, writes the result to outFile and supports cancellation.
func AddPropertiesFile(c context.Context, inFile, outFile string, properties map[string]string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if err := validateProperties(properties); err != nil {
		return fmt.Errorf("add properties: validate properties: %w", err)
	}
	f1, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("add properties: open input %s: %w", inFile, err)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "add properties")
	if err != nil {
		return errors.Join(
			fmt.Errorf("add properties: create output: %w", err),
			closeFile(f1, "add properties: close input"),
		)
	}
	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = AddProperties(c, f1, staged.output.file, properties, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}

// RemoveProperties deletes properties from rs's infodict, writes the result to w and supports cancellation.
func RemoveProperties(c context.Context, rs io.ReadSeeker, w io.Writer, properties []string, conf *model.Configuration) (err error) {
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

	if err := validateNoEmptyStrings(properties, "property name"); err != nil {
		return fmt.Errorf("remove properties: validate properties: %w", err)
	}
	if err := validatePropertyNames(properties); err != nil {
		return fmt.Errorf("remove properties: validate properties: %w", err)
	}

	conf = operationConfiguration(conf, model.REMOVEPROPERTIES)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("remove properties: %w", err)
	}

	var ok bool
	if ok, err = pdfcpu.PropertiesRemove(c, ctx, properties); err != nil {
		return fmt.Errorf("remove properties: update document properties: %w", err)
	}
	if !ok {
		return fmt.Errorf("remove properties: %w", ErrNoPropertyRemoved)
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("remove properties: write output: %w", err)
	}
	return nil
}

// RemovePropertiesFile deletes properties from inFile's infodict, writes the result to outFile and supports cancellation.
func RemovePropertiesFile(c context.Context, inFile, outFile string, properties []string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if err := validateNoEmptyStrings(properties, "property name"); err != nil {
		return fmt.Errorf("remove properties: validate properties: %w", err)
	}
	if err := validatePropertyNames(properties); err != nil {
		return fmt.Errorf("remove properties: validate properties: %w", err)
	}
	f1, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("remove properties: open input %s: %w", inFile, err)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "remove properties")
	if err != nil {
		return errors.Join(
			fmt.Errorf("remove properties: create output: %w", err),
			closeFile(f1, "remove properties: close input"),
		)
	}
	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = RemoveProperties(c, f1, staged.output.file, properties, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}
