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
	"bytes"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// TestPropertyCommandConstructorsCloneCallerData verifies commands own their mutable inputs.
func TestPropertyCommandConstructorsCloneCallerData(t *testing.T) {
	properties := map[string]string{"name": "value"}
	addCmd := AddPropertiesCommand("in.pdf", "out.pdf", properties, nil)
	properties["name"] = "changed"
	properties["other"] = "value"
	if !maps.Equal(addCmd.StringMap, map[string]string{"name": "value"}) {
		t.Fatalf("add command properties changed with caller map: %v", addCmd.StringMap)
	}

	names := []string{"name"}
	removeCmd := RemovePropertiesCommand("in.pdf", "out.pdf", names, nil)
	names[0] = "changed"
	names = append(names, "other")
	if !slices.Equal(removeCmd.StringVals, []string{"name"}) {
		t.Fatalf("remove command properties changed with caller slice: %v", removeCmd.StringVals)
	}
}

// TestPropertyCommandArgumentErrors verifies property commands reject incomplete public input.
func TestPropertyCommandArgumentErrors(t *testing.T) {
	inFile := "in.pdf"
	outFile := "out.pdf"
	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{name: "list command", run: func() error {
			_, err := ListProperties(nil)
			return err
		}, want: api.ErrMissingPDFInput},
		{name: "list input", run: func() error {
			_, err := ListProperties(&Command{})
			return err
		}, want: api.ErrMissingPDFInput},
		{name: "add input", run: func() error {
			_, err := AddProperties(&Command{OutFile: &outFile})
			return err
		}, want: api.ErrMissingPDFInput},
		{name: "add output", run: func() error {
			_, err := AddProperties(&Command{InFile: &inFile})
			return err
		}, want: api.ErrMissingPDFOutput},
		{name: "remove input", run: func() error {
			_, err := RemoveProperties(&Command{OutFile: &outFile})
			return err
		}, want: api.ErrMissingPDFInput},
		{name: "remove output", run: func() error {
			_, err := RemoveProperties(&Command{InFile: &inFile})
			return err
		}, want: api.ErrMissingPDFOutput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

// TestPropertyCommandValidatesBeforeIO verifies malformed caller data does not reach stream setup.
func TestPropertyCommandValidatesBeforeIO(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "missing.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{name: "add name", run: func() error {
			_, err := AddProperties(&Command{
				InFile:    &inFile,
				OutFile:   &outFile,
				StringMap: map[string]string{"": "value"},
			})
			return err
		}, want: "add properties: validate properties"},
		{name: "add value", run: func() error {
			_, err := AddProperties(&Command{
				InFile:    &inFile,
				OutFile:   &outFile,
				StringMap: map[string]string{"name": ""},
			})
			return err
		}, want: "add properties: validate properties"},
		{name: "remove name", run: func() error {
			_, err := RemoveProperties(&Command{
				InFile:     &inFile,
				OutFile:    &outFile,
				StringVals: []string{" "},
			})
			return err
		}, want: "remove properties: validate properties"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %v", tt.want, err)
			}
			if errors.Is(err, os.ErrNotExist) {
				t.Fatalf("file I/O occurred before validation: %v", err)
			}
		})
	}
}

// TestPropertyCommandsRejectProtectedNamesBeforeIO verifies every protected name precedes stream and file setup.
func TestPropertyCommandsRejectProtectedNamesBeforeIO(t *testing.T) {
	protected := []string{"Keywords", "Producer", "CreationDate", "ModDate", "Trapped"}
	for _, property := range protected {
		t.Run(property, func(t *testing.T) {
			tests := []struct {
				name string
				run  func(string, string) error
			}{
				{name: "add file", run: func(inFile, outFile string) error {
					_, err := AddProperties(&Command{
						InFile:    &inFile,
						OutFile:   &outFile,
						StringMap: map[string]string{property: "value"},
					})
					return err
				}},
				{name: "remove file", run: func(inFile, outFile string) error {
					_, err := RemoveProperties(&Command{
						InFile:     &inFile,
						OutFile:    &outFile,
						StringVals: []string{property},
					})
					return err
				}},
				{name: "add stream", run: func(inFile, outFile string) error {
					outFile = "-"
					_, err := AddProperties(&Command{
						InFile:    &inFile,
						OutFile:   &outFile,
						StringMap: map[string]string{property: "value"},
					})
					return err
				}},
				{name: "remove stream", run: func(inFile, outFile string) error {
					outFile = "-"
					_, err := RemoveProperties(&Command{
						InFile:     &inFile,
						OutFile:    &outFile,
						StringVals: []string{property},
					})
					return err
				}},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					inFile := filepath.Join(t.TempDir(), "missing.pdf")
					outFile := filepath.Join(t.TempDir(), "out.pdf")
					err := tt.run(inFile, outFile)
					if err == nil || !strings.Contains(err.Error(), "validate properties") ||
						!strings.Contains(err.Error(), "property name \""+property+"\" not allowed") {
						t.Fatalf("expected protected property validation for %q, got %v", property, err)
					}
					if errors.Is(err, os.ErrNotExist) {
						t.Fatalf("I/O occurred before property validation: %v", err)
					}
				})
			}
		})
	}
}

// TestListPropertiesFileErrors verifies file lifecycle failures retain operation context.
func TestListPropertiesFileErrors(t *testing.T) {
	if _, err := ListPropertiesFile("", nil); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}

	missing := filepath.Join(t.TempDir(), "missing.pdf")
	if _, err := ListPropertiesFile(missing, nil); !errors.Is(err, os.ErrNotExist) ||
		!strings.Contains(err.Error(), "list properties: open input") {
		t.Fatalf("expected contextual open error, got %v", err)
	}
}

// TestListPropertiesFileReportsCloseError verifies close failures are not discarded.
func TestListPropertiesFileReportsCloseError(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	original := closeListPropertiesInput
	closeListPropertiesInput = func(f *os.File) error {
		if err := f.Close(); err != nil {
			return err
		}
		return os.ErrClosed
	}
	t.Cleanup(func() {
		closeListPropertiesInput = original
	})

	_, err := ListPropertiesFile(inFile, nil)
	if !errors.Is(err, os.ErrClosed) || !strings.Contains(err.Error(), "list properties: close input") {
		t.Fatalf("expected contextual close error, got %v", err)
	}
}

// TestListPropertiesFileJoinsOperationAndCloseErrors verifies neither file error is discarded.
func TestListPropertiesFileJoinsOperationAndCloseErrors(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "empty.pdf")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("property close failed")
	original := closeListPropertiesInput
	closeListPropertiesInput = func(f *os.File) error {
		if err := f.Close(); err != nil {
			return err
		}
		return wantErr
	}
	t.Cleanup(func() {
		closeListPropertiesInput = original
	})

	_, err := ListPropertiesFile(inFile, nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) || !errors.Is(err, wantErr) {
		t.Fatalf("expected operation and close causes, got %v", err)
	}
	if !strings.Contains(err.Error(), "list properties: close input "+inFile) {
		t.Fatalf("expected close-input context, got %v", err)
	}
}

// TestPropertyStreamingOpenErrorsUseExactOperation verifies operation-aware stream setup context.
func TestPropertyStreamingOpenErrorsUseExactOperation(t *testing.T) {
	outFile := "-"
	tests := []struct {
		name string
		run  func(*Command) error
		op   string
	}{
		{name: "add", run: func(cmd *Command) error {
			_, err := AddProperties(cmd)
			return err
		}, op: "add properties"},
		{name: "remove", run: func(cmd *Command) error {
			_, err := RemoveProperties(cmd)
			return err
		}, op: "remove properties"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inFile := filepath.Join(t.TempDir(), "missing.pdf")
			err := tt.run(&Command{InFile: &inFile, OutFile: &outFile})
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			want := tt.op + ": open input " + inFile
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q context, got %v", want, err)
			}
		})
	}
}

// TestRunPropertyStreamOperationPreservesOutput verifies operation failure preserves an existing destination.
func TestRunPropertyStreamOperationPreservesOutput(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	wantOutput := []byte("existing output")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outFile, wantOutput, 0o600); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("property operation failed")

	err := runPropertyStreamOperation(inFile, outFile, "add properties", func(io.ReadSeeker, io.Writer) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, wantOutput) {
		t.Fatalf("existing output changed: got %q, want %q", got, wantOutput)
	}
}

// TestPropertyStreamingFailurePreservesExistingOutput verifies protected output survives read failure.
func TestPropertyStreamingFailurePreservesExistingOutput(t *testing.T) {
	useStdin(t, "not a PDF")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := AddProperties(AddPropertiesCommand("-", outFile, map[string]string{"name": "value"}, nil))
	if err == nil {
		t.Fatal("expected read failure")
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

// TestPropertyStreamingFailureRemovesNewOutput verifies failed streaming removes a new destination.
func TestPropertyStreamingFailureRemovesNewOutput(t *testing.T) {
	useStdin(t, "not a PDF")
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	_, err := RemoveProperties(RemovePropertiesCommand("-", outFile, nil, nil))
	if err == nil {
		t.Fatal("expected read failure")
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected failed output cleanup, got %v", statErr)
	}
}

// TestPropertyStreamingSuccessReplacesExistingOutput verifies successful delayed replacement.
func TestPropertyStreamingSuccessReplacesExistingOutput(t *testing.T) {
	bb, err := os.ReadFile(filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	useStdinBytes(t, bb)
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("existing output"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := AddProperties(AddPropertiesCommand("-", outFile, map[string]string{"name": "value"}, nil)); err != nil {
		t.Fatal(err)
	}
	properties, err := ListPropertiesFile(outFile, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(properties, "name = value") {
		t.Fatalf("replacement output is missing property: %v", properties)
	}
}

// TestPropertyFinalizerCloseFailurePreservesCause verifies output-close failures retain all causes.
func TestPropertyFinalizerCloseFailurePreservesCause(t *testing.T) {
	output, err := os.CreateTemp(t.TempDir(), "property-output-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	outFile := output.Name()
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("property operation failed")
	finalizer := &streamInOutFinalizer{output: output, outFile: outFile}

	err = finalizer.finalize("add properties", wantErr)
	if !errors.Is(err, wantErr) || !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected operation and close causes, got %v", err)
	}
	if !strings.Contains(err.Error(), "add properties: close output") {
		t.Fatalf("expected close-output context, got %v", err)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected failed output cleanup, got %v", statErr)
	}
}

// TestPropertyFinalizerRemoveFailurePreservesCause verifies cleanup failures retain all causes.
func TestPropertyFinalizerRemoveFailurePreservesCause(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "output")
	if err := os.Mkdir(outFile, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outFile, "keep"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	probeErr := os.Remove(outFile)
	if probeErr == nil {
		t.Fatal("expected non-empty directory removal failure")
	}
	removeCause := errors.Unwrap(probeErr)
	wantErr := errors.New("property operation failed")
	finalizer := &streamInOutFinalizer{outFile: outFile}

	err := finalizer.finalize("remove properties", wantErr)
	if !errors.Is(err, wantErr) || !errors.Is(err, removeCause) {
		t.Fatalf("expected operation and remove causes, got %v", err)
	}
	if !strings.Contains(err.Error(), "remove properties: remove output") {
		t.Fatalf("expected remove-output context, got %v", err)
	}
}

// TestPropertyFinalizerRenameFailurePreservesCause verifies replacement failures retain their cause.
func TestPropertyFinalizerRenameFailurePreservesCause(t *testing.T) {
	output, err := os.CreateTemp(t.TempDir(), "property-output-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	outFile := output.Name()
	replaceOut := filepath.Join(t.TempDir(), "output")
	if err := os.Mkdir(replaceOut, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(replaceOut, "keep"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	finalizer := &streamInOutFinalizer{
		output:     output,
		outFile:    outFile,
		replaceOut: replaceOut,
	}

	err = finalizer.finalize("add properties", nil)
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) || !errors.Is(err, linkErr.Err) {
		t.Fatalf("expected rename cause, got %v", err)
	}
	if !strings.Contains(err.Error(), "add properties: replace output") {
		t.Fatalf("expected replace-output context, got %v", err)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected temporary output cleanup, got %v", statErr)
	}
}
