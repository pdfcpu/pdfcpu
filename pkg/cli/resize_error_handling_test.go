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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func cliResizeConfiguration() *model.Resize {
	return &model.Resize{Scale: 0.5}
}

// TestResizeRejectsMissingCommandFields verifies CLI resize boundary guards.
func TestResizeRejectsMissingCommandFields(t *testing.T) {
	if _, err := Resize(nil); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}
	if _, err := Resize(&Command{}); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}
	inFile := "-"
	if _, err := Resize(&Command{InFile: &inFile}); !errors.Is(err, api.ErrMissingPDFOutput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFOutput, err)
	}
	outFile := "-"
	if _, err := Resize(&Command{InFile: &inFile, OutFile: &outFile}); !errors.Is(err, api.ErrMissingResizeConfiguration) {
		t.Fatalf("expected %v, got %v", api.ErrMissingResizeConfiguration, err)
	}
}

// TestResizeRejectsEmptyInputFile verifies an empty input filename is rejected at the CLI boundary.
func TestResizeRejectsEmptyInputFile(t *testing.T) {
	inFile := ""
	outFile := "-"
	_, err := Resize(&Command{InFile: &inFile, OutFile: &outFile, Resize: cliResizeConfiguration()})
	if !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}
}

// TestResizeStreamingIOErrorsIncludeOperationContext verifies resize-specific stream setup context.
func TestResizeStreamingIOErrorsIncludeOperationContext(t *testing.T) {
	outFile := "-"
	missingInput := filepath.Join(t.TempDir(), "missing.pdf")
	_, err := Resize(&Command{InFile: &missingInput, OutFile: &outFile, Resize: cliResizeConfiguration()})
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "resize: open input "+missingInput) {
		t.Fatalf("expected input context, got %v", err)
	}

	pageStreamingStdin(t)
	inFile := "-"
	missingOutput := filepath.Join(t.TempDir(), "missing", "out.pdf")
	_, err = Resize(&Command{InFile: &inFile, OutFile: &missingOutput, Resize: cliResizeConfiguration()})
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "resize: create output "+missingOutput) {
		t.Fatalf("expected output context, got %v", err)
	}
}

// TestResizeStreamingFailurePreservesExistingOutput verifies protected streaming output.
func TestResizeStreamingFailurePreservesExistingOutput(t *testing.T) {
	pageStreamingStdin(t)
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Resize(ResizeCommand("-", outFile, []string{"foo"}, cliResizeConfiguration(), nil))
	if err == nil || !strings.Contains(err.Error(), "resize: parse page selection") {
		t.Fatalf("expected page-selection error, got %v", err)
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}
