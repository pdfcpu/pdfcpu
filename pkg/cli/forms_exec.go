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
	return api.ListFormFields(rs, conf)
}

func exportFormGroup(rs io.ReadSeeker, source string, conf *model.Configuration) (*form.FormGroup, error) {
	formGroup, err := api.ExportForm(rs, source, conf)
	if err != nil {
		return nil, fmt.Errorf("list form fields: export data: %w", err)
	}
	return formGroup, nil
}

func exportFormGroupFile(fileName string, conf *model.Configuration) (*form.FormGroup, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("list form fields: open input %s: %w", fileName, err)
	}
	formGroup, opErr := exportFormGroup(f, formFieldSource(fileName), conf)
	return formGroup, errors.Join(opErr, closeStreamFile(f, "list form fields: close input"))
}

func listFormFieldsJSON(inFiles []string, conf *model.Configuration) ([]string, error) {
	formGroup := &form.FormGroup{}

	for _, fn := range inFiles {
		source := formFieldSource(fn)
		var fg *form.FormGroup
		var err error
		if fn == "-" {
			fg, err = withStdinReadSeeker("list form fields", func(rs io.ReadSeeker) (*form.FormGroup, error) {
				return exportFormGroup(rs, source, conf)
			})
		} else {
			fg, err = exportFormGroupFile(fn, conf)
		}
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
		return nil, fmt.Errorf("list form fields: encode JSON: %w", err)
	}
	return []string{string(bb)}, nil
}

func listFormFieldsFile(fileName string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("list form fields: open input %s: %w", fileName, err)
	}
	ss, opErr := listFormFields(f, conf)
	return ss, errors.Join(opErr, closeStreamFile(f, "list form fields: close input"))
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
		output, err := listFormFieldsFile(fn, conf)
		if err != nil {
			if len(inFiles) > 1 {
				errs = append(errs, err)
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
		var output []string
		var err error
		if fn == "-" {
			output, err = withStdinReadSeeker("list form fields", func(rs io.ReadSeeker) ([]string, error) {
				return listFormFields(rs, cmd.Conf)
			})
		} else {
			output, err = listFormFieldsFile(fn, cmd.Conf)
		}
		if err != nil {
			if len(cmd.InFiles) == 1 {
				return nil, err
			}
			errs = append(errs, err)
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

func formTemplateFileFromStdin() (string, func(error) error, error) {
	in, err := readSeekerFromStdin("multi-fill form")
	if err != nil {
		return "", nil, err
	}
	return in.path, func(opErr error) error { return in.finalize("multi-fill form", opErr) }, nil
}

func formPDFFileCommand(inFile, outFile, operation string, fileFn func() error, readerFn func(io.ReadSeeker, io.Writer) error) ([]string, error) {
	if inFile != "-" && outFile != "-" {
		return nil, fileFn()
	}

	rs, w, finalize, err := streamInOutForOperation(inFile, outFile, operation)
	if err != nil {
		return nil, err
	}
	return nil, finalize(readerFn(rs, w))
}

func formPDFWithData(cmd *Command, operation string, fileFn func() error, readerFn func(io.ReadSeeker, io.Reader, io.Writer) error) ([]string, error) {
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, fileFn()
	}

	rd, err := os.Open(*cmd.InFileJSON)
	if err != nil {
		return nil, fmt.Errorf("%s: open form data %s: %w", operation, *cmd.InFileJSON, err)
	}

	rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, operation)
	if err != nil {
		return nil, errors.Join(err, closeStreamFile(rd, operation+": close form data"))
	}

	err = readerFn(rs, rd, w)
	err = errors.Join(err, closeStreamFile(rd, operation+": close form data"))
	return nil, finalize(err)
}

// RemoveFormFields removes some form fields from inFile.
func RemoveFormFields(cmd *Command) ([]string, error) {
	return formPDFFileCommand(
		*cmd.InFile,
		*cmd.OutFile,
		"remove form fields",
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
		"lock form fields",
		func() error {
			return api.LockFormFieldsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
		},
		func(rs io.ReadSeeker, w io.Writer) error {
			return api.LockFormFields(rs, w, cmd.StringVals, cmd.Conf)
		},
	)
}

// UnlockFormFields makes some or all form fields of inFile writable.
func UnlockFormFields(cmd *Command) ([]string, error) {
	return formPDFFileCommand(
		*cmd.InFile,
		*cmd.OutFile,
		"unlock form fields",
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
		"reset form fields",
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
		rs, w, finalize, err := streamInOutForOperation("-", *cmd.OutFileJSON, "export form")
		if err != nil {
			return nil, err
		}

		return nil, finalize(api.ExportFormJSON(rs, w, "stdin", cmd.Conf))
	}

	return nil, api.ExportFormFile(*cmd.InFile, *cmd.OutFileJSON, cmd.Conf)
}

// FillFormFields fills out inFile's form using data represented by inFileJSON.
func FillFormFields(cmd *Command) ([]string, error) {
	return formPDFWithData(
		cmd,
		"fill form",
		func() error {
			return api.FillFormFile(*cmd.InFile, *cmd.InFileJSON, *cmd.OutFile, cmd.Conf)
		},
		func(rs io.ReadSeeker, rd io.Reader, w io.Writer) error {
			return api.FillForm(rs, rd, w, cmd.Conf)
		},
	)
}

func multiFillFormInputFile(cmd *Command) (string, func(error) error, error) {
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
		return nil, errors.New("multi-fill form: stdout requires merge mode")
	}

	outDir, err := os.MkdirTemp("", "pdfcpu-form-multifill-")
	if err != nil {
		return nil, fmt.Errorf("multi-fill form: create temporary output directory: %w", err)
	}
	cleanup := func(err error) error {
		if removeErr := os.RemoveAll(outDir); removeErr != nil {
			err = errors.Join(err, fmt.Errorf("multi-fill form: remove temporary output directory: %w", removeErr))
		}
		return err
	}

	outFile := "stdout.pdf"
	if err := api.MultiFillFormFile(inFile, *cmd.InFileJSON, outDir, outFile, true, cmd.Conf); err != nil {
		return nil, cleanup(err)
	}

	log.SetCLILogger(nil)
	outPath := filepath.Join(outDir, outFile)
	f, err := os.Open(outPath)
	if err != nil {
		return nil, cleanup(fmt.Errorf("multi-fill form: open merged output %s: %w", outPath, err))
	}

	_, err = io.Copy(os.Stdout, f)
	if err != nil {
		err = fmt.Errorf("multi-fill form: write stdout: %w", err)
	}
	err = errors.Join(err, closeStreamFile(f, "multi-fill form: close merged output"))
	return nil, cleanup(err)
}

// MultiFillFormFields fills out multiple instances of inFile's form using JSON or CSV data.
func MultiFillFormFields(cmd *Command) ([]string, error) {
	inFile, finalize, err := multiFillFormInputFile(cmd)
	if err != nil {
		return nil, err
	}

	var result []string
	if *cmd.OutFile == "-" {
		result, err = multiFillFormFieldsToStdout(cmd, inFile)
	} else {
		err = api.MultiFillFormFile(inFile, *cmd.InFileJSON, *cmd.OutDir, multiFillFormOutputFile(cmd), cmd.BoolVal1, cmd.Conf)
	}
	if finalize != nil {
		err = finalize(err)
	}
	return result, err
}
