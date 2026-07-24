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

type extractCommandExecutor func(*Command) ([]string, error)

// TestExtractExecutorsRejectNilCommand verifies every public extraction executor has a safe nil boundary.
func TestExtractExecutorsRejectNilCommand(t *testing.T) {
	tests := []struct {
		name string
		run  extractCommandExecutor
	}{
		{"ExtractImages", ExtractImages},
		{"ExtractFonts", ExtractFonts},
		{"ExtractPages", ExtractPages},
		{"ExtractContent", ExtractContent},
		{"ExtractMetadata", ExtractMetadata},
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

// TestExtractExecutorsRejectIncompleteCommand verifies required paths are checked before I/O.
func TestExtractExecutorsRejectIncompleteCommand(t *testing.T) {
	inFile := "missing.pdf"
	outDir := "out"
	tests := []struct {
		name string
		run  extractCommandExecutor
		cmd  *Command
		want error
	}{
		{"ExtractImagesInput", ExtractImages, &Command{OutDir: &outDir}, api.ErrMissingPDFInput},
		{"ExtractImagesOutput", ExtractImages, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"ExtractFontsInput", ExtractFonts, &Command{OutDir: &outDir}, api.ErrMissingPDFInput},
		{"ExtractFontsOutput", ExtractFonts, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"ExtractPagesInput", ExtractPages, &Command{OutDir: &outDir}, api.ErrMissingPDFInput},
		{"ExtractPagesOutput", ExtractPages, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"ExtractContentInput", ExtractContent, &Command{OutDir: &outDir}, api.ErrMissingPDFInput},
		{"ExtractContentOutput", ExtractContent, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"ExtractMetadataInput", ExtractMetadata, &Command{OutDir: &outDir}, api.ErrMissingPDFInput},
		{"ExtractMetadataOutput", ExtractMetadata, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
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

// TestDispatchRejectsIncompleteExtractCommandsWithoutPanic verifies caller mistakes remain ordinary errors.
func TestDispatchRejectsIncompleteExtractCommandsWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		mode model.CommandMode
	}{
		{"ExtractImages", model.EXTRACTIMAGES},
		{"ExtractFonts", model.EXTRACTFONTS},
		{"ExtractPages", model.EXTRACTPAGES},
		{"ExtractContent", model.EXTRACTCONTENT},
		{"ExtractMetadata", model.EXTRACTMETADATA},
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
