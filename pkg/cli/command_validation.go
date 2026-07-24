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

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

type commandStringRequirement uint8

const (
	commandStringOptional commandStringRequirement = iota
	commandStringRequired
	commandStringRequiredNonEmpty
)

type commandRequirements struct {
	operation         string
	inFile            commandStringRequirement
	outFile           commandStringRequirement
	outDir            commandStringRequirement
	inFileJSON        commandStringRequirement
	outFileJSON       commandStringRequirement
	minInputFiles     int
	maxInputFiles     int
	missingInputFiles error
	minIntValues      int
	missingIntValues  error
}

func commandValidationError(operation string, err error) error {
	if err == nil || operation == "" {
		return err
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func validateCommandString(value *string, requirement commandStringRequirement, missing error) error {
	if requirement == commandStringOptional {
		return nil
	}
	if value == nil || requirement == commandStringRequiredNonEmpty && *value == "" {
		return missing
	}
	return nil
}

func validateCommandInputFiles(inFiles []string, minCount, maxCount int, missing error) error {
	if minCount == 0 && maxCount == 0 {
		return nil
	}
	if missing == nil {
		missing = api.ErrMissingPDFInput
	}
	if len(inFiles) < minCount {
		return missing
	}
	if maxCount > 0 && len(inFiles) > maxCount {
		return fmt.Errorf("%w: expected at most %d inputs, got %d", ErrInvalidCommandArguments, maxCount, len(inFiles))
	}
	for i, inFile := range inFiles {
		if inFile == "" {
			return fmt.Errorf("input %d: %w", i+1, missing)
		}
	}
	return nil
}

func validateCommandRequirements(cmd *Command, requirements commandRequirements) error {
	if cmd == nil {
		return commandValidationError(requirements.operation, ErrMissingCommand)
	}
	checks := []error{
		validateCommandString(cmd.InFile, requirements.inFile, api.ErrMissingPDFInput),
		validateCommandString(cmd.OutFile, requirements.outFile, api.ErrMissingPDFOutput),
		validateCommandString(cmd.OutDir, requirements.outDir, api.ErrMissingPDFOutput),
		validateCommandString(cmd.InFileJSON, requirements.inFileJSON, api.ErrMissingJSONInput),
		validateCommandString(cmd.OutFileJSON, requirements.outFileJSON, api.ErrMissingJSONOutput),
		validateCommandInputFiles(
			cmd.InFiles,
			requirements.minInputFiles,
			requirements.maxInputFiles,
			requirements.missingInputFiles,
		),
	}
	for _, err := range checks {
		if err != nil {
			return commandValidationError(requirements.operation, err)
		}
	}
	if len(cmd.IntVals) < requirements.minIntValues {
		err := requirements.missingIntValues
		if err == nil {
			err = ErrInvalidCommandArguments
		}
		return commandValidationError(requirements.operation, err)
	}
	return nil
}

func validatedCommandInFile(cmd *Command, operation string) (string, error) {
	requirements := commandRequirements{
		operation: operation,
		inFile:    commandStringRequiredNonEmpty,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return "", err
	}
	return *cmd.InFile, nil
}

func optionalCommandString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
