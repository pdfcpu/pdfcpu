/*
Copyright 2026 The pdfcpu Authors.

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

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func listFormFields(rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTFORMFIELDS

	ctx, err := api.ReadAndValidate(rs, conf)
	if err != nil {
		return nil, err
	}

	return form.ListFormFields(ctx)
}

func exportFormGroup(rs io.ReadSeeker, source string, conf *model.Configuration) (*form.FormGroup, error) {
	formGroup, err := api.ExportForm(rs, source, conf)
	if err != nil {
		return nil, err
	}
	return formGroup, nil
}

func listFormFieldsJSON(inFiles []string, conf *model.Configuration) ([]string, error) {
	formGroup := &form.FormGroup{}

	for _, fn := range inFiles {
		rs, err := formFieldReadSeeker(fn)
		if err != nil {
			return nil, err
		}
		if f, ok := rs.(*os.File); ok {
			defer f.Close()
		}

		source := formFieldSource(fn)
		fg, err := exportFormGroup(rs, source, conf)
		if err != nil {
			return nil, err
		}
		if len(formGroup.Forms) == 0 {
			formGroup.Header = fg.Header
		}
		formGroup.Forms = append(formGroup.Forms, fg.Forms...)
	}

	bb, err := json.MarshalIndent(formGroup, "", "\t")
	if err != nil {
		return nil, err
	}
	return []string{string(bb)}, nil
}

func formFieldReadSeeker(fn string) (io.ReadSeeker, error) {
	if fn == "-" {
		return readSeekerFromStdin()
	}
	return os.Open(fn)
}

func formFieldSource(fn string) string {
	if fn == "-" {
		return "stdin"
	}
	return fn
}

// ListFormFieldsFile returns a list of form field ids in inFile.
func ListFormFieldsFile(inFiles []string, conf *model.Configuration) ([]string, error) {
	log.SetCLILogger(nil)

	ss := []string{}
	var errs []error

	for _, fn := range inFiles {

		f, err := os.Open(fn)
		if err != nil {
			if len(inFiles) > 1 {
				errs = append(errs, fmt.Errorf("%s: %w", fn, err))
				continue
			}
			return nil, err
		}
		defer f.Close()

		output, err := listFormFields(f, conf)
		if err != nil {
			if len(inFiles) > 1 {
				errs = append(errs, fmt.Errorf("%s: %w", fn, err))
				continue
			}
			return nil, err
		}

		ss = append(ss, "\n"+fn+":\n")
		ss = append(ss, output...)
	}

	return ss, errors.Join(errs...)
}

// ListFormFields returns inFile's form field ids.
func ListFormFields(cmd *Command) ([]string, error) {
	if cmd.BoolVal1 {
		log.SetCLILogger(nil)
		return listFormFieldsJSON(cmd.InFiles, cmd.Conf)
	}

	stdin := false
	for _, fn := range cmd.InFiles {
		if fn == "-" {
			stdin = true
			break
		}
	}
	if !stdin {
		return ListFormFieldsFile(cmd.InFiles, cmd.Conf)
	}

	log.SetCLILogger(nil)
	var ss []string
	var errs []error
	for _, fn := range cmd.InFiles {
		var rs io.ReadSeeker
		var err error
		if fn == "-" {
			rs, err = readSeekerFromStdin()
		} else {
			rs, err = os.Open(fn)
		}
		if err != nil {
			if len(cmd.InFiles) == 1 {
				return nil, err
			}
			errs = append(errs, fmt.Errorf("%s: %w", fn, err))
			continue
		}
		if f, ok := rs.(*os.File); ok {
			defer f.Close()
		}

		output, err := listFormFields(rs, cmd.Conf)
		if err != nil {
			if len(cmd.InFiles) == 1 {
				return nil, err
			}
			errs = append(errs, fmt.Errorf("%s: %w", fn, err))
			continue
		}

		label := fn
		if label == "-" {
			label = "stdin"
		}
		ss = append(ss, "\n"+label+":\n")
		ss = append(ss, output...)
	}

	return ss, errors.Join(errs...)
}

func formInOut(cmd *Command) (io.ReadSeeker, io.Writer, func(), error) {
	return streamInOut(*cmd.InFile, *cmd.OutFile)
}

func formDataReader(filename string) (*os.File, error) {
	return os.Open(filename)
}

func formTemplateFileFromStdin() (string, func(), error) {
	rs, err := readSeekerFromStdin()
	if err != nil {
		return "", nil, err
	}

	f, err := os.CreateTemp("", "pdfcpu-form-stdin-*.pdf")
	if err != nil {
		return "", nil, err
	}
	name := f.Name()
	cleanup := func() {
		_ = os.Remove(name)
	}

	if _, err := io.Copy(f, rs); err != nil {
		_ = f.Close()
		cleanup()
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, err
	}

	return name, cleanup, nil
}

func fillFormData(cmd *Command) (*os.File, error) {
	return formDataReader(*cmd.InFileJSON)
}

func formPDFFileCommand(inFile, outFile string, fileFn func() error, readerFn func(io.ReadSeeker, io.Writer) error) ([]string, error) {
	if inFile != "-" && outFile != "-" {
		return nil, fileFn()
	}

	rs, w, cleanup, err := streamInOut(inFile, outFile)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return nil, readerFn(rs, w)
}

func formPDFWithData(cmd *Command, fileFn func() error, readerFn func(io.ReadSeeker, io.Reader, io.Writer) error) ([]string, error) {
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, fileFn()
	}

	rd, err := fillFormData(cmd)
	if err != nil {
		return nil, err
	}
	defer rd.Close()

	rs, w, cleanup, err := formInOut(cmd)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return nil, readerFn(rs, rd, w)
}

// RemoveFormFields removes some form fields from inFile.
func RemoveFormFields(cmd *Command) ([]string, error) {
	return formPDFFileCommand(
		*cmd.InFile,
		*cmd.OutFile,
		func() error {
			return api.RemoveFormFieldsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
		},
		func(rs io.ReadSeeker, w io.Writer) error {
			return api.RemoveFormFields(rs, w, cmd.StringVals, cmd.Conf)
		},
	)
}

// LockFormFields makes some or all form fields of inFile read-only.
func LockFormFields(cmd *Command) ([]string, error) {
	return formPDFFileCommand(
		*cmd.InFile,
		*cmd.OutFile,
		func() error {
			return api.LockFormFieldsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
		},
		func(rs io.ReadSeeker, w io.Writer) error {
			return api.LockFormFields(rs, w, cmd.StringVals, cmd.Conf)
		},
	)
}

// UnlockFormFields makes some or all form fields of inFile writeable.
func UnlockFormFields(cmd *Command) ([]string, error) {
	return formPDFFileCommand(
		*cmd.InFile,
		*cmd.OutFile,
		func() error {
			return api.UnlockFormFieldsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
		},
		func(rs io.ReadSeeker, w io.Writer) error {
			return api.UnlockFormFields(rs, w, cmd.StringVals, cmd.Conf)
		},
	)
}

// ResetFormFields sets some or all form fields of inFile to the corresponding default value.
func ResetFormFields(cmd *Command) ([]string, error) {
	return formPDFFileCommand(
		*cmd.InFile,
		*cmd.OutFile,
		func() error {
			return api.ResetFormFieldsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
		},
		func(rs io.ReadSeeker, w io.Writer) error {
			return api.ResetFormFields(rs, w, cmd.StringVals, cmd.Conf)
		},
	)
}

// ExportFormFields returns a representation of inFile's form as outFileJSON.
func ExportFormFields(cmd *Command) ([]string, error) {
	if *cmd.InFile == "-" {
		rs, err := readSeekerFromStdin()
		if err != nil {
			return nil, err
		}
		f, err := os.Create(*cmd.OutFileJSON)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return nil, api.ExportFormJSON(rs, f, "stdin", cmd.Conf)
	}

	return nil, api.ExportFormFile(*cmd.InFile, *cmd.OutFileJSON, cmd.Conf)
}

// FillFormFields fills out inFile's form using data represented by inFileJSON.
func FillFormFields(cmd *Command) ([]string, error) {
	return formPDFWithData(
		cmd,
		func() error {
			return api.FillFormFile(*cmd.InFile, *cmd.InFileJSON, *cmd.OutFile, cmd.Conf)
		},
		func(rs io.ReadSeeker, rd io.Reader, w io.Writer) error {
			return api.FillForm(rs, rd, w, cmd.Conf)
		},
	)
}

func multiFillFormInputFile(cmd *Command) (string, func(), error) {
	if *cmd.InFile != "-" {
		return *cmd.InFile, nil, nil
	}
	return formTemplateFileFromStdin()
}

func multiFillFormOutputFile(cmd *Command) string {
	if *cmd.OutFile == "" && *cmd.InFile == "-" {
		return "stdin.pdf"
	}
	return *cmd.OutFile
}

func multiFillFormFieldsToStdout(cmd *Command, inFile string) ([]string, error) {
	if !cmd.BoolVal1 {
		return nil, fmt.Errorf("pdfcpu: form multifill stdout requires -m merge")
	}

	outDir, err := os.MkdirTemp("", "pdfcpu-form-multifill-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(outDir)

	outFile := "stdout.pdf"
	if err := api.MultiFillFormFile(inFile, *cmd.InFileJSON, outDir, outFile, true, cmd.Conf); err != nil {
		return nil, err
	}

	log.SetCLILogger(nil)
	f, err := os.Open(filepath.Join(outDir, outFile))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = io.Copy(os.Stdout, f)
	return nil, err
}

// MultiFillFormFields fills out multiple instances of inFile's form using JSON or CSV data.
func MultiFillFormFields(cmd *Command) ([]string, error) {
	inFile, cleanup, err := multiFillFormInputFile(cmd)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	if *cmd.OutFile == "-" {
		return multiFillFormFieldsToStdout(cmd, inFile)
	}

	return nil, api.MultiFillFormFile(inFile, *cmd.InFileJSON, *cmd.OutDir, multiFillFormOutputFile(cmd), cmd.BoolVal1, cmd.Conf)
}
