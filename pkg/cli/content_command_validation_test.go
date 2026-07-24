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

type contentCommandExecutor func(*Command) ([]string, error)

// TestContentExecutorsRejectNilCommand verifies every public content executor has a safe nil boundary.
func TestContentExecutorsRejectNilCommand(t *testing.T) {
	tests := []struct {
		name string
		run  contentCommandExecutor
	}{
		{"AddWatermarks", AddWatermarks},
		{"RemoveWatermarks", RemoveWatermarks},
		{"ListAnnotations", ListAnnotations},
		{"RemoveAnnotations", RemoveAnnotations},
		{"ListBookmarks", ListBookmarks},
		{"ExportBookmarks", ExportBookmarks},
		{"ImportBookmarks", ImportBookmarks},
		{"RemoveBookmarks", RemoveBookmarks},
		{"ListPageLayout", ListPageLayout},
		{"SetPageLayout", SetPageLayout},
		{"ResetPageLayout", ResetPageLayout},
		{"ListPageMode", ListPageMode},
		{"SetPageMode", SetPageMode},
		{"ResetPageMode", ResetPageMode},
		{"ListViewerPreferences", ListViewerPreferences},
		{"SetViewerPreferences", SetViewerPreferences},
		{"ResetViewerPreferences", ResetViewerPreferences},
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

// TestContentExecutorsRejectIncompleteCommand verifies required fields are checked before I/O.
func TestContentExecutorsRejectIncompleteCommand(t *testing.T) {
	empty := ""
	inFile := "missing.pdf"
	outFile := "out.pdf"
	tests := []struct {
		name string
		run  contentCommandExecutor
		cmd  *Command
		want error
	}{
		{"AddWatermarksInput", AddWatermarks, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"AddWatermarksOutput", AddWatermarks, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"AddWatermarksConfig", AddWatermarks, &Command{InFile: &inFile, OutFile: &outFile},
			api.ErrMissingWatermarkConfiguration},
		{"RemoveWatermarksInput", RemoveWatermarks, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"RemoveWatermarksOutput", RemoveWatermarks, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"ListAnnotationsInput", ListAnnotations, &Command{InFile: &empty}, api.ErrMissingPDFInput},
		{"RemoveAnnotationsInput", RemoveAnnotations, &Command{OutFile: &outFile}, api.ErrMissingPDFInput},
		{"RemoveAnnotationsOutput", RemoveAnnotations, &Command{InFile: &inFile}, api.ErrMissingPDFOutput},
		{"ListBookmarksInput", ListBookmarks, &Command{}, api.ErrMissingPDFInput},
		{"ExportBookmarksOutput", ExportBookmarks, &Command{InFile: &inFile}, api.ErrMissingJSONOutput},
		{"ImportBookmarksInput", ImportBookmarks, &Command{InFile: &inFile}, api.ErrMissingJSONInput},
		{"RemoveBookmarksInput", RemoveBookmarks, &Command{}, api.ErrMissingPDFInput},
		{"ListPageLayoutInput", ListPageLayout, &Command{}, api.ErrMissingPDFInput},
		{"SetPageLayoutInput", SetPageLayout, &Command{}, api.ErrMissingPDFInput},
		{"ResetPageLayoutInput", ResetPageLayout, &Command{}, api.ErrMissingPDFInput},
		{"ListPageModeInput", ListPageMode, &Command{}, api.ErrMissingPDFInput},
		{"SetPageModeInput", SetPageMode, &Command{}, api.ErrMissingPDFInput},
		{"ResetPageModeInput", ResetPageMode, &Command{}, api.ErrMissingPDFInput},
		{"ListViewerPreferencesInput", ListViewerPreferences, &Command{}, api.ErrMissingPDFInput},
		{"SetViewerPreferencesInput", SetViewerPreferences, &Command{}, api.ErrMissingPDFInput},
		{"ResetViewerPreferencesInput", ResetViewerPreferences, &Command{}, api.ErrMissingPDFInput},
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

// TestContentFileHelpersRejectMissingInput verifies exported file helpers reject empty paths consistently.
func TestContentFileHelpersRejectMissingInput(t *testing.T) {
	if _, _, err := ListAnnotationsFile("", nil, nil); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("ListAnnotationsFile: expected %v, got %v", api.ErrMissingPDFInput, err)
	}
	if _, _, err := ListAnnotationsJSONFile("", nil, nil); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("ListAnnotationsJSONFile: expected %v, got %v", api.ErrMissingPDFInput, err)
	}
}

// TestDispatchRejectsIncompleteContentCommandsWithoutPanic verifies caller mistakes remain ordinary errors.
func TestDispatchRejectsIncompleteContentCommandsWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		cmd  *Command
		want error
	}{
		{"AddWatermarks", &Command{Mode: model.ADDWATERMARKS}, api.ErrMissingPDFInput},
		{"RemoveWatermarks", &Command{Mode: model.REMOVEWATERMARKS}, api.ErrMissingPDFInput},
		{"ListAnnotations", &Command{Mode: model.LISTANNOTATIONS}, api.ErrMissingPDFInput},
		{"RemoveAnnotations", &Command{Mode: model.REMOVEANNOTATIONS}, api.ErrMissingPDFInput},
		{"ExportBookmarks", &Command{Mode: model.EXPORTBOOKMARKS}, api.ErrMissingPDFInput},
		{"SetPageMode", &Command{Mode: model.SETPAGEMODE}, api.ErrMissingPDFInput},
		{"SetPageLayout", &Command{Mode: model.SETPAGELAYOUT}, api.ErrMissingPDFInput},
		{"SetViewerPreferences", &Command{Mode: model.SETVIEWERPREFERENCES}, api.ErrMissingPDFInput},
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
