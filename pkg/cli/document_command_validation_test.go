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

type documentCommandExecutor func(*Command) ([]string, error)

// TestDocumentExecutorsRejectNilCommand verifies every public document executor has a safe nil boundary.
func TestDocumentExecutorsRejectNilCommand(t *testing.T) {
	tests := []struct {
		name string
		run  documentCommandExecutor
	}{
		{"Validate", Validate},
		{"Optimize", Optimize},
		{"MergeCreate", MergeCreate},
		{"MergeCreateZip", MergeCreateZip},
		{"MergeAppend", MergeAppend},
		{"Split", Split},
		{"SplitByPageNr", SplitByPageNr},
		{"Trim", Trim},
		{"Collect", Collect},
		{"ListInfo", ListInfo},
		{"Dump", Dump},
		{"Create", Create},
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
		{"ValidateInput", Validate, &Command{}, api.ErrMissingPDFInput},
		{"OptimizeInput", Optimize, &Command{}, api.ErrMissingPDFInput},
		{"OptimizeOutput", Optimize, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"MergeCreateInput", MergeCreate, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"MergeCreateOutput", MergeCreate, &Command{InFiles: []string{inFile}}, api.ErrMissingPDFOutput},
		{"MergeZipInputs", MergeCreateZip, &Command{InFiles: []string{inFile}, OutFile: &outFile}, api.ErrMissingPDFInput},
		{"MergeZipTooManyInputs", MergeCreateZip, &Command{
			InFiles: []string{"one.pdf", "two.pdf", "three.pdf"},
			OutFile: &outFile,
		}, ErrInvalidCommandArguments},
		{"MergeAppendInput", MergeAppend, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"SplitInput", Split, &Command{OutDir: &outDir}, api.ErrMissingPDFInput},
		{"SplitOutput", Split, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"SplitByPageNumbers", SplitByPageNr, &Command{InFile: &inFile, OutDir: &outDir}, api.ErrMissingSplitPageNumbers},
		{"TrimInput", Trim, &Command{OutFile: &empty}, api.ErrMissingPDFInput},
		{"TrimOutput", Trim, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"CollectInput", Collect, &Command{OutFile: &empty}, api.ErrMissingPDFInput},
		{"CollectOutput", Collect, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"ListInfoInput", ListInfo, &Command{}, api.ErrMissingPDFInput},
		{"DumpValues", Dump, &Command{InFile: &inFile}, ErrInvalidCommandArguments},
		{"CreateInputField", Create, &Command{InFileJSON: &jsonFile, OutFile: &outFile}, api.ErrMissingPDFInput},
		{"CreateJSON", Create, &Command{InFile: &empty, OutFile: &outFile}, api.ErrMissingJSONInput},
		{"CreateOutputField", Create, &Command{InFile: &inFile, InFileJSON: &jsonFile}, api.ErrMissingPDFOutput},
		{"CreateInputOrOutput", Create, &Command{
			InFile:     &empty,
			InFileJSON: &jsonFile,
			OutFile:    &empty,
		}, api.ErrMissingPDFInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.run(tt.cmd)
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
			_, err := Dispatch(tt.cmd)
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
