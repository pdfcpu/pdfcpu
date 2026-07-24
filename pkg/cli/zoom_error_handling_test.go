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

// TestZoomRejectsMissingCommandFields verifies CLI zoom boundary guards.
func TestZoomRejectsMissingCommandFields(t *testing.T) {
	inFile, empty, outFile := "in.pdf", "", "-"
	tests := []struct {
		name string
		cmd  *Command
		want error
	}{
		{name: "nil command", want: ErrMissingCommand},
		{name: "nil input", cmd: &Command{}, want: api.ErrMissingPDFInput},
		{name: "empty input", cmd: &Command{InFile: &empty}, want: api.ErrMissingPDFInput},
		{name: "nil output", cmd: &Command{InFile: &inFile}, want: api.ErrMissingPDFOutput},
		{name: "nil zoom configuration", cmd: &Command{InFile: &inFile, OutFile: &outFile}, want: api.ErrMissingZoomConfiguration},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Zoom(tt.cmd); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}

	cmd := &Command{InFile: &inFile, OutFile: &outFile, Zoom: &model.Zoom{Factor: 0.5}}
	if err := validateZoomCommand(cmd); err != nil {
		t.Fatalf("expected valid command, got %v", err)
	}
}

func cliZoomConfiguration() *model.Zoom {
	return &model.Zoom{Factor: 0.5}
}

// TestZoomStreamingIOErrorsIncludeOperationContext verifies zoom-specific stream setup context.
func TestZoomStreamingIOErrorsIncludeOperationContext(t *testing.T) {
	outFile := "-"
	missingInput := filepath.Join(t.TempDir(), "missing.pdf")
	_, err := Zoom(&Command{InFile: &missingInput, OutFile: &outFile, Zoom: cliZoomConfiguration()})
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "zoom: open input "+missingInput) {
		t.Fatalf("expected input context, got %v", err)
	}

	pageStreamingStdin(t)
	inFile := "-"
	missingOutput := filepath.Join(t.TempDir(), "missing", "out.pdf")
	_, err = Zoom(&Command{InFile: &inFile, OutFile: &missingOutput, Zoom: cliZoomConfiguration()})
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "zoom: create output "+missingOutput) {
		t.Fatalf("expected output context, got %v", err)
	}
}

func zoomExistingOutput(t *testing.T) (string, []byte) {
	t.Helper()
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0o600); err != nil {
		t.Fatal(err)
	}
	return outFile, want
}

func requireZoomOutput(t *testing.T, outFile string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

// TestZoomStreamingPageSelectionFailurePreservesExistingOutput verifies protected streaming finalization.
func TestZoomStreamingPageSelectionFailurePreservesExistingOutput(t *testing.T) {
	pageStreamingStdin(t)
	outFile, want := zoomExistingOutput(t)
	_, err := Zoom(ZoomCommand("-", outFile, []string{"foo"}, cliZoomConfiguration(), nil))
	if err == nil || !strings.Contains(err.Error(), "zoom: parse page selection") {
		t.Fatalf("expected page-selection error, got %v", err)
	}
	requireZoomOutput(t, outFile, want)
}

// TestZoomStreamingReadFailurePreservesExistingOutput verifies protected output on invalid PDF input.
func TestZoomStreamingReadFailurePreservesExistingOutput(t *testing.T) {
	useStdin(t, "not a PDF")
	outFile, want := zoomExistingOutput(t)
	_, err := Zoom(ZoomCommand("-", outFile, nil, cliZoomConfiguration(), nil))
	if err == nil || !strings.Contains(err.Error(), "zoom: prepare PDF context") {
		t.Fatalf("expected read error, got %v", err)
	}
	requireZoomOutput(t, outFile, want)
}

// TestZoomStreamingWriteFailurePreservesExistingOutput verifies protected output on writer failure.
func TestZoomStreamingWriteFailurePreservesExistingOutput(t *testing.T) {
	pageStreamingStdin(t)
	outFile, want := zoomExistingOutput(t)
	rs, w, finalize, err := streamInOutForOperation("-", outFile, "zoom")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.(*os.File).Close(); err != nil {
		t.Fatal(err)
	}
	err = finalize(api.Zoom(rs, w, nil, cliZoomConfiguration(), nil))
	if !errors.Is(err, os.ErrClosed) || !strings.Contains(err.Error(), "zoom: write output") {
		t.Fatalf("expected write error, got %v", err)
	}
	requireZoomOutput(t, outFile, want)
}
