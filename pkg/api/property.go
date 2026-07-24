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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Properties returns rs's properties as recorded in infoDict.
func Properties(rs io.ReadSeeker, conf *model.Configuration) (m map[string]string, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTPROPERTIES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("list properties: %w", err)
	}

	return ctx.Properties, nil
}

// AddProperties adds properties to rs's infodict and writes the result to w.
func AddProperties(rs io.ReadSeeker, w io.Writer, properties map[string]string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if err := validateProperties(properties); err != nil {
		return fmt.Errorf("add properties: validate properties: %w", err)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.ADDPROPERTIES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("add properties: %w", err)
	}

	if err = pdfcpu.PropertiesAdd(ctx, properties); err != nil {
		return fmt.Errorf("add properties: update document properties: %w", err)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("add properties: write output: %w", err)
	}
	return nil
}

type propertyMutation func(io.ReadSeeker, io.Writer, *model.Configuration) error

func mutatePropertiesFile(
	inFile, outFile string,
	conf *model.Configuration,
	op string,
	mutate propertyMutation,
) (err error) {
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
		return errors.Join(
			fmt.Errorf("%s: create output: %w", op, err),
			closeFile(f1, op+": close input"),
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

	if err = mutate(f1, f2, conf); err != nil {
		return err
	}

	ok = true
	return nil
}

// AddPropertiesFile adds properties to inFile's infodict and writes the result to outFile.
func AddPropertiesFile(inFile, outFile string, properties map[string]string, conf *model.Configuration) error {
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if err := validateProperties(properties); err != nil {
		return fmt.Errorf("add properties: validate properties: %w", err)
	}
	return mutatePropertiesFile(inFile, outFile, conf, "add properties", func(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) error {
		return AddProperties(rs, w, properties, conf)
	})
}

// RemoveProperties deletes properties from rs's infodict and writes the result to w.
func RemoveProperties(rs io.ReadSeeker, w io.Writer, properties []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

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

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.REMOVEPROPERTIES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("remove properties: %w", err)
	}

	var ok bool
	if ok, err = pdfcpu.PropertiesRemove(ctx, properties); err != nil {
		return fmt.Errorf("remove properties: update document properties: %w", err)
	}
	if !ok {
		return fmt.Errorf("remove properties: %w", ErrNoPropertyRemoved)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("remove properties: write output: %w", err)
	}
	return nil
}

// RemovePropertiesFile deletes properties from inFile's infodict and writes the result to outFile.
func RemovePropertiesFile(inFile, outFile string, properties []string, conf *model.Configuration) error {
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if err := validateNoEmptyStrings(properties, "property name"); err != nil {
		return fmt.Errorf("remove properties: validate properties: %w", err)
	}
	if err := validatePropertyNames(properties); err != nil {
		return fmt.Errorf("remove properties: validate properties: %w", err)
	}
	return mutatePropertiesFile(inFile, outFile, conf, "remove properties", func(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) error {
		return RemoveProperties(rs, w, properties, conf)
	})
}
