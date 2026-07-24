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
	"fmt"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type signatureValidationFileOperation func(
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

// ListCertificatesAll returns information about installed certificates.
func ListCertificatesAll(json bool, _ *model.Configuration) ([]string, error) {
	return api.ListCertificates(json)
}

// ListCertificates returns installed certificates.
func ListCertificates(cmd *Command) ([]string, error) {
	if err := validateCommandRequirements(cmd, commandRequirements{operation: "list certificates"}); err != nil {
		return nil, err
	}
	return ListCertificatesAll(cmd.BoolVal1, cmd.Conf)
}

// ImportCertificates imports certificates, replacing existing destinations with matching names.
func ImportCertificates(cmd *Command) ([]string, error) {
	if err := validateCertificateCommand(cmd, "import certificates"); err != nil {
		return nil, err
	}
	return api.ImportCertificates(cmd.InFiles)
}

// InspectCertificates prints the certificate details.
func InspectCertificates(cmd *Command) ([]string, error) {
	if err := validateCertificateCommand(cmd, "inspect certificates"); err != nil {
		return nil, err
	}
	return api.InspectCertificates(cmd.InFiles)
}

// ValidateSignatures presents observed signature, certificate, timestamp and
// revocation evidence together with a local assessment.
func ValidateSignatures(cmd *Command) ([]string, error) {
	return validateSignatures(cmd, api.ValidateSignaturesFile)
}

func validateSignatures(
	cmd *Command,
	operation signatureValidationFileOperation,
) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "validate signatures")
	if err != nil {
		return nil, err
	}

	if inFile == "-" {
		in, err := readSeekerFromStdin("validate signatures")
		if err != nil {
			return nil, err
		}
		result, opErr := operation(in.path, cmd.BoolVal1, cmd.BoolVal2, cmd.Conf)
		return result, in.finalize("validate signatures", opErr)
	}

	return operation(inFile, cmd.BoolVal1, cmd.BoolVal2, cmd.Conf)
}

// RemoveSignatures removes contained digital signatures.
func RemoveSignatures(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "remove signatures")
	if err != nil {
		return nil, err
	}
	outFile := optionalCommandString(cmd.OutFile)

	if inFile != "-" && outFile != "-" {
		return nil, api.RemoveSignaturesFile(inFile, outFile, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(inFile, outFile, "remove signatures")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.RemoveSignatures(rs, w, cmd.Conf))
}
