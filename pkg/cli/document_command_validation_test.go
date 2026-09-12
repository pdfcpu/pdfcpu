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
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type documentCommandExecutor = dispatchFunc

// TestDocumentExecutorsRejectNilCommand verifies every public document executor has a safe nil boundary.
func TestDocumentExecutorsRejectNilCommand(t *testing.T) {
	tests := []struct {
		name string
		run  documentCommandExecutor
	}{
		{"Validate", validateCommand},
		{"Optimize", optimize},
		{"MergeCreate", mergeCreate},
		{"MergeCreateZip", mergeCreateZip},
		{"MergeAppend", mergeAppend},
		{"Split", split},
		{"SplitByPageNr", splitByPageNr},
		{"Trim", trim},
		{"Collect", collect},
		{"ListInfo", listInfoCommand},
		{"Dump", dump},
		{"Create", create},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.run(t.Context(), nil)
			if !errors.Is(err, ErrMissingCommand) {
				t.Fatalf("expected %v, got %v", ErrMissingCommand, err)
			}
		})
	}
}

// TestDocumentExecutorsRejectIncompleteCommand verifies required fields are checked before I/O.
func TestDocumentExecutorsRejectIncompleteCommand(t *testing.T) {
	empty := ""
	inFile := "missing.pdf"
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	outDir := filepath.Join(t.TempDir(), "out")
	jsonFile := "missing.json"
	tests := []struct {
		name string
		run  documentCommandExecutor
		cmd  *Command
		want error
	}{
		{"ValidateInput", validateCommand, &Command{}, api.ErrMissingPDFInput},
		{"OptimizeInput", optimize, &Command{}, api.ErrMissingPDFInput},
		{"OptimizeOutput", optimize, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"MergeCreateInput", mergeCreate, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"MergeCreateOutput", mergeCreate, &Command{InFiles: []string{inFile}}, api.ErrMissingPDFOutput},
		{"MergeZipInputs", mergeCreateZip, &Command{InFiles: []string{inFile}, OutFile: &outFile}, api.ErrMissingPDFInput},
		{"MergeZipTooManyInputs", mergeCreateZip, &Command{
			InFiles: []string{"one.pdf", "two.pdf", "three.pdf"},
			OutFile: &outFile,
		}, ErrInvalidCommandArguments},
		{"MergeAppendInput", mergeAppend, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"SplitInput", split, &Command{OutDir: &outDir}, api.ErrMissingPDFInput},
		{"SplitOutput", split, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"SplitByPageNumbers", splitByPageNr, &Command{InFile: &inFile, OutDir: &outDir}, api.ErrMissingSplitPageNumbers},
		{"TrimInput", trim, &Command{OutFile: &empty}, api.ErrMissingPDFInput},
		{"TrimOutput", trim, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"CollectInput", collect, &Command{OutFile: &empty}, api.ErrMissingPDFInput},
		{"CollectOutput", collect, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"ListInfoInput", listInfoCommand, &Command{}, api.ErrMissingPDFInput},
		{"DumpValues", dump, &Command{InFile: &inFile}, ErrInvalidCommandArguments},
		{"CreateInputField", create, &Command{InFileJSON: &jsonFile, OutFile: &outFile}, api.ErrMissingPDFInput},
		{"CreateJSON", create, &Command{InFile: &empty, OutFile: &outFile}, api.ErrMissingJSONInput},
		{"CreateOutputField", create, &Command{InFile: &inFile, InFileJSON: &jsonFile}, api.ErrMissingPDFOutput},
		{"CreateInputOrOutput", create, &Command{
			InFile:     &empty,
			InFileJSON: &jsonFile,
			OutFile:    &empty,
		}, api.ErrMissingPDFInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.run(t.Context(), tt.cmd)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
	if _, err := os.Stat(outFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("validation created output: %v", err)
	}
}

// TestDispatchRejectsIncompleteDocumentCommandsWithoutPanic verifies caller mistakes remain ordinary errors.
func TestDispatchRejectsIncompleteDocumentCommandsWithoutPanic(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	inFile := "missing.pdf"
	tests := []struct {
		name string
		cmd  *Command
		want error
	}{
		{"Optimize", &Command{Mode: model.OPTIMIZE}, api.ErrMissingPDFInput},
		{"MergeCreateZip", &Command{
			Mode:    model.MERGECREATEZIP,
			InFiles: []string{inFile},
			OutFile: &outFile,
		}, api.ErrMissingPDFInput},
		{"Dump", &Command{Mode: model.DUMP, InFile: &inFile}, ErrInvalidCommandArguments},
		{"Create", &Command{Mode: model.CREATE}, api.ErrMissingPDFInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Dispatch(t.Context(), tt.cmd)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
			var panicErr fault.Panic
			if errors.As(err, &panicErr) {
				t.Fatalf("caller error returned as panic: %v", err)
			}
		})
	}
}
