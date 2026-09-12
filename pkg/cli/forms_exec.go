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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func validateFormPDFCommand(cmd *Command, operation string) error {
	return validateCommandRequirements(cmd, commandRequirements{
		operation: operation,
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequired,
	})
}

func validateMultiFillFormCommand(cmd *Command) error {
	requirements := commandRequirements{
		operation: "multi-fill form",
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequired,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return err
	}
	if err := validateCommandString(cmd.InFileJSON, commandStringRequiredNonEmpty, api.ErrMissingFormInput); err != nil {
		return commandValidationError(requirements.operation, err)
	}
	if *cmd.OutFile == "-" {
		return nil
	}
	return commandValidationError(
		requirements.operation,
		validateCommandString(cmd.OutDir, commandStringRequiredNonEmpty, api.ErrMissingPDFOutput),
	)
}

func listFormFields(c context.Context, rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	return api.ListFormFields(c, rs, conf)
}

func exportFormGroup(c context.Context, rs io.ReadSeeker, source string, conf *model.Configuration) (*form.FormGroup, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	formGroup, err := api.ExportForm(c, rs, source, conf)
	if err != nil {
		return nil, fmt.Errorf("list form fields: export data: %w", err)
	}
	return formGroup, nil
}

func exportFormGroupFile(c context.Context, fileName string, conf *model.Configuration) (*form.FormGroup, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("list form fields: open input %s: %w", fileName, err)
	}
	formGroup, opErr := exportFormGroup(c, f, formFieldSource(fileName), conf)
	return formGroup, errors.Join(opErr, closeStreamFile(f, "list form fields: close input"))
}

func listFormFieldsJSON(c context.Context, inFiles []string, conf *model.Configuration) ([]string, error) {
	if c == nil {
		return nil, ErrMissingContext
	}
	formGroup := &form.FormGroup{}

	for _, fn := range inFiles {
		if err := c.Err(); err != nil {
			return nil, err
		}
		source := formFieldSource(fn)
		var fg *form.FormGroup
		var err error
		if fn == "-" {
			fg, err = withStdinReadSeeker(c, "list form fields", func(rs io.ReadSeeker) (*form.FormGroup, error) {
				return exportFormGroup(c, rs, source, conf)
			})
		} else {
			fg, err = exportFormGroupFile(c, fn, conf)
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
	return []string{string(bb)}, c.Err()
}

func listFormFieldsFile(c context.Context, fileName string, conf *model.Configuration) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("list form fields: open input %s: %w", fileName, err)
	}
	ss, opErr := listFormFields(c, f, conf)
	return ss, errors.Join(opErr, closeStreamFile(f, "list form fields: close input"))
}

func formFieldSource(fn string) string {
	if fn == "-" {
		return "stdin"
	}
	return fn
}

// ListFormFieldsFile returns form field IDs in inFiles and supports cancellation.
func ListFormFieldsFile(c context.Context, inFiles []string, conf *model.Configuration) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandInputFiles(inFiles, 1, 0, nil); err != nil {
		return nil, commandValidationError("list form fields", err)
	}
	log.SetCLILogger(nil)

	ss := []string{}
	var errs []error

	for _, fn := range inFiles {
		if err := c.Err(); err != nil {
			return nil, err
		}
		output, err := listFormFieldsFile(c, fn, conf)
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

	return ss, errors.Join(c.Err(), errors.Join(errs...))
}

func listFormFieldsForCommand(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	requirements := commandRequirements{
		operation:     "list form fields",
		minInputFiles: 1,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return nil, err
	}
	if cmd.BoolVal1 {
		log.SetCLILogger(nil)
		return listFormFieldsJSON(c, cmd.InFiles, cmd.Conf)
	}

	stdin := slices.Contains(cmd.InFiles, "-")
	if !stdin {
		return ListFormFieldsFile(c, cmd.InFiles, cmd.Conf)
	}

	log.SetCLILogger(nil)
	var ss []string
	var errs []error
	for _, fn := range cmd.InFiles {
		if err := c.Err(); err != nil {
			return nil, err
		}
		var output []string
		var err error
		if fn == "-" {
			output, err = withStdinReadSeeker(c, "list form fields", func(rs io.ReadSeeker) ([]string, error) {
				return listFormFields(c, rs, cmd.Conf)
			})
		} else {
			output, err = listFormFieldsFile(c, fn, cmd.Conf)
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

	return ss, errors.Join(c.Err(), errors.Join(errs...))
}

func formTemplateFileFromStdin(c context.Context) (string, func(error) error, error) {
	if c == nil {
		return "", nil, ErrMissingContext
	}
	in, err := readSeekerFromStdin(c, "multi-fill form")
	if err != nil {
		return "", nil, err
	}
	return in.path, func(opErr error) error { return in.finalize("multi-fill form", opErr) }, nil
}

func formPDFFileCommand(c context.Context, inFile, outFile, operation string, fileFn func(context.Context) error, readerFn func(context.Context, io.ReadSeeker, io.Writer) error) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if inFile != "-" && outFile != "-" {
		return nil, fileFn(c)
	}

	rs, w, finalize, err := streamInOutForOperation(c, inFile, outFile, operation)
	if err != nil {
		return nil, err
	}
	return nil, finalize(readerFn(c, rs, w))
}

func formPDFWithData(c context.Context, cmd *Command, operation string, fileFn func(context.Context) error, readerFn func(context.Context, io.ReadSeeker, io.Reader, io.Writer) error) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, fileFn(c)
	}

	rd, err := os.Open(*cmd.InFileJSON)
	if err != nil {
		return nil, fmt.Errorf("%s: open form data %s: %w", operation, *cmd.InFileJSON, err)
	}

	rs, w, finalize, err := streamInOutForOperation(c, *cmd.InFile, *cmd.OutFile, operation)
	if err != nil {
		return nil, errors.Join(err, closeStreamFile(rd, operation+": close form data"))
	}

	err = readerFn(c, rs, rd, w)
	err = errors.Join(err, closeStreamFile(rd, operation+": close form data"))
	return nil, finalize(err)
}

func removeFormFields(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateFormPDFCommand(cmd, "remove form fields"); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	return formPDFFileCommand(
		c,
		*cmd.InFile,
		*cmd.OutFile,
		"remove form fields",
		func(c context.Context) error {
			return api.RemoveFormFieldsFile(c, *cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
		},
		func(c context.Context, rs io.ReadSeeker, w io.Writer) error {
			return api.RemoveFormFields(c, rs, w, cmd.StringVals, cmd.Conf)
		},
	)
}

func lockFormFields(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateFormPDFCommand(cmd, "lock form fields"); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	return formPDFFileCommand(
		c,
		*cmd.InFile,
		*cmd.OutFile,
		"lock form fields",
		func(c context.Context) error {
			return api.LockFormFieldsFile(c, *cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
		},
		func(c context.Context, rs io.ReadSeeker, w io.Writer) error {
			return api.LockFormFields(c, rs, w, cmd.StringVals, cmd.Conf)
		},
	)
}

func unlockFormFields(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateFormPDFCommand(cmd, "unlock form fields"); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	return formPDFFileCommand(
		c,
		*cmd.InFile,
		*cmd.OutFile,
		"unlock form fields",
		func(c context.Context) error {
			return api.UnlockFormFieldsFile(c, *cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
		},
		func(c context.Context, rs io.ReadSeeker, w io.Writer) error {
			return api.UnlockFormFields(c, rs, w, cmd.StringVals, cmd.Conf)
		},
	)
}

func resetFormFields(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateFormPDFCommand(cmd, "reset form fields"); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	return formPDFFileCommand(
		c,
		*cmd.InFile,
		*cmd.OutFile,
		"reset form fields",
		func(c context.Context) error {
			return api.ResetFormFieldsFile(c, *cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
		},
		func(c context.Context, rs io.ReadSeeker, w io.Writer) error {
			return api.ResetFormFields(c, rs, w, cmd.StringVals, cmd.Conf)
		},
	)
}

func exportFormFields(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	requirements := commandRequirements{
		operation:   "export form",
		inFile:      commandStringRequiredNonEmpty,
		outFileJSON: commandStringRequiredNonEmpty,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return nil, err
	}
	reportOutputPath(*cmd.OutFileJSON)
	if *cmd.InFile == "-" {
		rs, w, finalize, err := streamInOutForOperation(c, "-", *cmd.OutFileJSON, "export form")
		if err != nil {
			return nil, err
		}

		return nil, finalize(api.ExportFormJSON(c, rs, w, "stdin", cmd.Conf))
	}

	return nil, api.ExportFormFile(c, *cmd.InFile, *cmd.OutFileJSON, cmd.Conf)
}

func fillFormFields(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	requirements := commandRequirements{
		operation:  "fill form",
		inFile:     commandStringRequiredNonEmpty,
		outFile:    commandStringRequired,
		inFileJSON: commandStringRequiredNonEmpty,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return nil, err
	}
	reportCommandProgress(cmd, "filling...\n")
	reportCommandOutputPath(cmd)
	return formPDFWithData(
		c,
		cmd,
		"fill form",
		func(c context.Context) error {
			return api.FillFormFile(c, *cmd.InFile, *cmd.InFileJSON, *cmd.OutFile, cmd.Conf)
		},
		func(c context.Context, rs io.ReadSeeker, rd io.Reader, w io.Writer) error {
			return api.FillForm(c, rs, rd, w, cmd.Conf)
		},
	)
}

func multiFillFormInputFile(c context.Context, cmd *Command) (string, func(error) error, error) {
	if err := contextutil.Check(c); err != nil {
		return "", nil, err
	}
	if *cmd.InFile != "-" {
		return *cmd.InFile, nil, nil
	}
	return formTemplateFileFromStdin(c)
}

func multiFillFormOutputFile(cmd *Command) string {
	if *cmd.OutFile == "" && *cmd.InFile == "-" {
		return "stdin.pdf"
	}
	return *cmd.OutFile
}

func reportMultiFillFormProgress(cmd *Command) {
	format := "JSON"
	if strings.EqualFold(filepath.Ext(*cmd.InFileJSON), ".csv") {
		format = "CSV"
	}
	inFile := *cmd.InFile
	if inFile == "-" {
		inFile = "stdin"
	}
	outDir := optionalCommandString(cmd.OutDir)
	reportCommandProgress(
		cmd,
		"filling multiple forms via %s based on %s data from %s into %s/%s ...\n",
		inFile,
		format,
		*cmd.InFileJSON,
		outDir,
		filepath.Base(multiFillFormOutputFile(cmd)),
	)
}

func multiFillFormFieldsToStdout(c context.Context, cmd *Command, inFile string) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
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
	if err := api.MultiFillFormFile(
		c, inFile, *cmd.InFileJSON, outDir, outFile, true, cmd.Conf,
	); err != nil {
		return nil, cleanup(err)
	}

	log.SetCLILogger(nil)
	outPath := filepath.Join(outDir, outFile)
	f, err := os.Open(outPath)
	if err != nil {
		return nil, cleanup(fmt.Errorf("multi-fill form: open merged output %s: %w", outPath, err))
	}

	_, err = io.Copy(os.Stdout, contextReader{ctx: c, r: f})
	if err != nil {
		err = fmt.Errorf("multi-fill form: write stdout: %w", err)
	}
	err = errors.Join(err, closeStreamFile(f, "multi-fill form: close merged output"))
	return nil, cleanup(err)
}

func multiFillFormFields(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateMultiFillFormCommand(cmd); err != nil {
		return nil, err
	}
	reportMultiFillFormProgress(cmd)
	inFile, finalize, err := multiFillFormInputFile(c, cmd)
	if err != nil {
		return nil, err
	}

	var result []string
	if *cmd.OutFile == "-" {
		result, err = multiFillFormFieldsToStdout(c, cmd, inFile)
	} else {
		err = api.MultiFillFormFile(
			c, inFile, *cmd.InFileJSON, *cmd.OutDir, multiFillFormOutputFile(cmd), cmd.BoolVal1, cmd.Conf,
		)
	}
	if finalize != nil {
		err = finalize(err)
	}
	return result, err
}
