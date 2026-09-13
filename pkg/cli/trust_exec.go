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
	"fmt"
	"strings"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type signatureValidationFileOperation func(
	context.Context,
	string,
	bool,
	bool,
	*model.Configuration,
) ([]string, error)

func validateCertificateCommand(cmd *Command, operation string) error {
	if err := validateCommandRequirements(cmd, commandRequirements{operation: operation}); err != nil {
		return err
	}
	if len(cmd.InFiles) == 0 {
		return commandValidationError(operation, api.ErrMissingCertificateInput)
	}
	for i, inFile := range cmd.InFiles {
		if strings.TrimSpace(inFile) == "" {
			err := fmt.Errorf("certificate input %d: %w", i+1, api.ErrMissingCertificateInput)
			return commandValidationError(operation, err)
		}
	}
	return nil
}

// ListCertificatesAll returns information about installed certificates and supports cancellation.
func ListCertificatesAll(c context.Context, json bool, _ *model.Configuration) ([]string, error) {
	return api.ListCertificates(c, json)
}

func listCertificates(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandRequirements(cmd, commandRequirements{operation: "list certificates"}); err != nil {
		return nil, err
	}
	return ListCertificatesAll(c, cmd.BoolVal1, cmd.Conf)
}

func importCertificates(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCertificateCommand(cmd, "import certificates"); err != nil {
		return nil, err
	}
	return api.ImportCertificates(c, cmd.InFiles)
}

func inspectCertificates(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCertificateCommand(cmd, "inspect certificates"); err != nil {
		return nil, err
	}
	return api.InspectCertificates(c, cmd.InFiles)
}

func validateSignaturesCommand(c context.Context, cmd *Command) ([]string, error) {
	return validateSignatures(c, cmd, api.ValidateSignaturesFile)
}

func validateSignatures(c context.Context, cmd *Command, operation signatureValidationFileOperation) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "validate signatures")
	if err != nil {
		return nil, err
	}

	if inFile == "-" {
		in, err := readSeekerFromStdin(c, cmd.Conf, "validate signatures")
		if err != nil {
			return nil, err
		}
		result, opErr := operation(c, in.path, cmd.BoolVal1, cmd.BoolVal2, cmd.Conf)
		return result, in.finalize("validate signatures", opErr)
	}

	return operation(c, inFile, cmd.BoolVal1, cmd.BoolVal2, cmd.Conf)
}

func removeSignatures(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "remove signatures")
	if err != nil {
		return nil, err
	}
	outFile := optionalCommandString(cmd.OutFile)
	reportCommandOutputPath(cmd)

	if inFile != "-" && outFile != "-" {
		return nil, api.RemoveSignaturesFile(c, inFile, outFile, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, inFile, outFile, "remove signatures")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.RemoveSignatures(c, rs, w, cmd.Conf))
}
