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

type pageCommandExecutor func(*Command) ([]string, error)

// TestPageExecutorsRejectNilCommand verifies every public page executor has a safe nil boundary.
func TestPageExecutorsRejectNilCommand(t *testing.T) {
	tests := []struct {
		name string
		run  pageCommandExecutor
	}{
		{"NUp", NUp},
		{"Grid", Grid},
		{"Booklet", Booklet},
		{"Resize", Resize},
		{"Poster", Poster},
		{"NDown", NDown},
		{"Cut", Cut},
		{"Zoom", Zoom},
		{"Rotate", Rotate},
		{"InsertPages", InsertPages},
		{"RemovePages", RemovePages},
		{"Crop", Crop},
		{"ListBoxes", ListBoxes},
		{"AddBoxes", AddBoxes},
		{"RemoveBoxes", RemoveBoxes},
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

// TestPageExecutorsRejectIncompleteCommand verifies configuration fields are checked before I/O.
func TestPageExecutorsRejectIncompleteCommand(t *testing.T) {
	inFile := "missing.pdf"
	outFile := "out.pdf"
	outDir := "out"
	tests := []struct {
		name string
		run  pageCommandExecutor
		cmd  *Command
		want error
	}{
		{"NUpConfig", NUp, &Command{OutFile: &outFile}, api.ErrMissingNUpConfiguration},
		{"GridConfig", Grid, &Command{OutFile: &outFile}, api.ErrMissingGridConfiguration},
		{"BookletConfig", Booklet, &Command{OutFile: &outFile}, api.ErrMissingBookletConfiguration},
		{"ResizeConfig", Resize, &Command{InFile: &inFile, OutFile: &outFile},
			api.ErrMissingResizeConfiguration},
		{"PosterConfig", Poster, &Command{InFile: &inFile, OutFile: &outFile, OutDir: &outDir},
			api.ErrMissingCutConfiguration},
		{"NDownConfig", NDown, &Command{InFile: &inFile, OutFile: &outFile, OutDir: &outDir},
			api.ErrMissingCutConfiguration},
		{"CutConfig", Cut, &Command{InFile: &inFile, OutFile: &outFile, OutDir: &outDir},
			api.ErrMissingCutConfiguration},
		{"ZoomConfig", Zoom, &Command{InFile: &inFile, OutFile: &outFile}, api.ErrMissingZoomConfiguration},
		{"RotateInput", Rotate, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"InsertPagesInput", InsertPages, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"RemovePagesInput", RemovePages, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"CropConfig", Crop, &Command{InFile: &inFile, OutFile: &outFile}, api.ErrMissingBoxConfiguration},
		{"ListBoxesInput", ListBoxes, &Command{}, api.ErrMissingPDFInput},
		{"AddBoxesConfig", AddBoxes, &Command{InFile: &inFile, OutFile: &outFile},
			api.ErrMissingPageBoundaries},
		{"RemoveBoxesConfig", RemoveBoxes, &Command{InFile: &inFile, OutFile: &outFile},
			api.ErrMissingPageBoundaries},
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

// TestListBoxesFileRejectsMissingInput verifies the exported file helper rejects an empty path.
func TestListBoxesFileRejectsMissingInput(t *testing.T) {
	if _, err := ListBoxesFile("", nil, nil, nil); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}
}

// TestDispatchRejectsIncompletePageCommandsWithoutPanic verifies caller mistakes remain ordinary errors.
func TestDispatchRejectsIncompletePageCommandsWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		mode model.CommandMode
		want error
	}{
		{"NUp", model.NUP, api.ErrMissingPDFOutput},
		{"Grid", model.GRID, api.ErrMissingPDFOutput},
		{"Booklet", model.BOOKLET, api.ErrMissingPDFOutput},
		{"Resize", model.RESIZE, api.ErrMissingPDFInput},
		{"Poster", model.POSTER, api.ErrMissingPDFInput},
		{"NDown", model.NDOWN, api.ErrMissingPDFInput},
		{"Cut", model.CUT, api.ErrMissingPDFInput},
		{"Zoom", model.ZOOM, api.ErrMissingPDFInput},
		{"Rotate", model.ROTATE, api.ErrMissingPDFInput},
		{"InsertPages", model.INSERTPAGESBEFORE, api.ErrMissingPDFInput},
		{"RemovePages", model.REMOVEPAGES, api.ErrMissingPDFInput},
		{"Crop", model.CROP, api.ErrMissingPDFInput},
		{"ListBoxes", model.LISTBOXES, api.ErrMissingPDFInput},
		{"AddBoxes", model.ADDBOXES, api.ErrMissingPDFInput},
		{"RemoveBoxes", model.REMOVEBOXES, api.ErrMissingPDFInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Dispatch(&Command{Mode: tt.mode})
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
