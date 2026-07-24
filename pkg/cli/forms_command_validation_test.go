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
	"errors"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type formCommandExecutor func(*Command) ([]string, error)

// TestFormExecutorsRejectNilCommand verifies every public form executor has a safe nil boundary.
func TestFormExecutorsRejectNilCommand(t *testing.T) {
	tests := []struct {
		name string
		run  formCommandExecutor
	}{
		{"ListFormFields", ListFormFields},
		{"RemoveFormFields", RemoveFormFields},
		{"LockFormFields", LockFormFields},
		{"UnlockFormFields", UnlockFormFields},
		{"ResetFormFields", ResetFormFields},
		{"ExportFormFields", ExportFormFields},
		{"FillFormFields", FillFormFields},
		{"MultiFillFormFields", MultiFillFormFields},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.run(nil)
			if !errors.Is(err, ErrMissingCommand) {
				t.Fatalf("expected %v, got %v", ErrMissingCommand, err)
			}
		})
	}
}

// TestFormExecutorsRejectIncompleteCommand verifies required paths are checked before I/O.
func TestFormExecutorsRejectIncompleteCommand(t *testing.T) {
	inFile := "missing.pdf"
	dataFile := "missing.json"
	outFile := "out.pdf"
	tests := []struct {
		name string
		run  formCommandExecutor
		cmd  *Command
		want error
	}{
		{"ListInput", ListFormFields, &Command{}, api.ErrMissingPDFInput},
		{"RemoveInput", RemoveFormFields, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"RemoveOutput", RemoveFormFields, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"LockInput", LockFormFields, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"UnlockInput", UnlockFormFields, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"ResetInput", ResetFormFields, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"ExportInput", ExportFormFields, &Command{}, api.ErrMissingPDFInput},
		{"ExportOutput", ExportFormFields, &Command{InFile: &inFile}, api.ErrMissingJSONOutput},
		{"FillInput", FillFormFields, &Command{}, api.ErrMissingPDFInput},
		{"FillOutput", FillFormFields, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"FillData", FillFormFields, &Command{InFile: &inFile, OutFile: &outFile}, api.ErrMissingJSONInput},
		{"MultiFillInput", MultiFillFormFields, &Command{}, api.ErrMissingPDFInput},
		{"MultiFillOutput", MultiFillFormFields, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"MultiFillData", MultiFillFormFields, &Command{InFile: &inFile, OutFile: &outFile},
			api.ErrMissingFormInput},
		{"MultiFillOutputDirectory", MultiFillFormFields, &Command{
			InFile:     &inFile,
			InFileJSON: &dataFile,
			OutFile:    &outFile,
		}, api.ErrMissingPDFOutput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.run(tt.cmd)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

// TestValidateMultiFillFormCommandPreservesOutputModes verifies output-directory requirements remain mode-aware.
func TestValidateMultiFillFormCommandPreservesOutputModes(t *testing.T) {
	inFile := "missing.pdf"
	dataFile := "missing.json"
	stdout := "-"
	if err := validateMultiFillFormCommand(&Command{
		InFile:     &inFile,
		InFileJSON: &dataFile,
		OutFile:    &stdout,
	}); err != nil {
		t.Fatalf("stdout mode: %v", err)
	}

	outFile := "out.pdf"
	outDir := "out"
	if err := validateMultiFillFormCommand(&Command{
		InFile:     &inFile,
		InFileJSON: &dataFile,
		OutFile:    &outFile,
		OutDir:     &outDir,
	}); err != nil {
		t.Fatalf("directory mode: %v", err)
	}
}

// TestListFormFieldsFileRejectsMissingInput verifies the exported file helper validates its input list.
func TestListFormFieldsFileRejectsMissingInput(t *testing.T) {
	tests := [][]string{nil, {""}}
	for _, inFiles := range tests {
		if _, err := ListFormFieldsFile(inFiles, nil); !errors.Is(err, api.ErrMissingPDFInput) {
			t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
		}
	}
}

// TestDispatchRejectsIncompleteFormCommandsWithoutPanic verifies caller mistakes remain ordinary errors.
func TestDispatchRejectsIncompleteFormCommandsWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		mode model.CommandMode
	}{
		{"ListFormFields", model.LISTFORMFIELDS},
		{"RemoveFormFields", model.REMOVEFORMFIELDS},
		{"LockFormFields", model.LOCKFORMFIELDS},
		{"UnlockFormFields", model.UNLOCKFORMFIELDS},
		{"ResetFormFields", model.RESETFORMFIELDS},
		{"ExportFormFields", model.EXPORTFORMFIELDS},
		{"FillFormFields", model.FILLFORMFIELDS},
		{"MultiFillFormFields", model.MULTIFILLFORMFIELDS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Dispatch(&Command{Mode: tt.mode})
			if !errors.Is(err, api.ErrMissingPDFInput) {
				t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
			}
			var panicErr fault.Panic
			if errors.As(err, &panicErr) {
				t.Fatalf("caller error returned as panic: %v", err)
			}
		})
	}
}
