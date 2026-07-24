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
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func TestListAnnotationsFileOpenErrorHasPhaseContext(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "missing.pdf")
	_, _, err := ListAnnotationsFile(inFile, nil, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
	want := "list annotations: open input " + inFile
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q in %q", want, err.Error())
	}
}

func TestCloseListAnnotationsInputPreservesOperationAndCloseErrors(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "annotations-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("list failed")
	err = closeListAnnotationsInput(f, wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected operation error preserved, got %v", err)
	}
	if !strings.Contains(err.Error(), "list annotations: close input") {
		t.Fatalf("expected close phase context, got %q", err.Error())
	}
}

func TestListViewerPreferencesJSONFromStdin(t *testing.T) {
	stdin := os.Stdin
	f, err := os.Open(filepath.Join("..", "testdata", "Hybrid-PDF.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = stdin
		_ = f.Close()
	})

	ss, err := ListViewerPreferences(ListViewerPreferencesCommand("-", false, true, nil))
	if err != nil {
		t.Fatalf("list viewer preferences JSON from stdin: %v", err)
	}
	if len(ss) != 1 {
		t.Fatalf("expected one JSON result, got %d", len(ss))
	}

	var result struct {
		ViewerPreferences json.RawMessage `json:"viewerPreferences"`
	}
	if err := json.Unmarshal([]byte(ss[0]), &result); err != nil {
		t.Fatalf("decode viewer preferences JSON: %v", err)
	}
	if len(result.ViewerPreferences) == 0 {
		t.Fatal("expected viewerPreferences field")
	}
}

func TestSetViewerPreferencesStreamingMissingJSONIncludesPhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "Hybrid-PDF.pdf")
	jsonFile := filepath.Join(t.TempDir(), "missing.json")

	_, err := SetViewerPreferences(SetViewerPreferencesCommand(inFile, jsonFile, "-", "", nil))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	want := "set viewer preferences: read JSON " + jsonFile
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q in error, got %q", want, err.Error())
	}
}

func TestViewerPreferencesCommandsRejectMissingInput(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{
			name: "list nil command",
			run: func() error {
				_, err := ListViewerPreferences(nil)
				return err
			},
			want: ErrMissingCommand,
		},
		{
			name: "set missing input",
			run: func() error {
				_, err := SetViewerPreferences(&Command{})
				return err
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "reset nil command",
			run: func() error {
				_, err := ResetViewerPreferences(nil)
				return err
			},
			want: ErrMissingCommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestPageLayoutCommandsRejectMissingInput(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{
			name: "list nil command",
			run: func() error {
				_, err := ListPageLayout(nil)
				return err
			},
			want: ErrMissingCommand,
		},
		{
			name: "set missing input",
			run: func() error {
				_, err := SetPageLayout(&Command{StringVal: "SinglePage"})
				return err
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "reset nil command",
			run: func() error {
				_, err := ResetPageLayout(nil)
				return err
			},
			want: ErrMissingCommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestSetPageLayoutRejectsInvalidLayout(t *testing.T) {
	cmd := &Command{InFile: stringPtr("missing.pdf"), StringVal: "bogus"}
	_, err := SetPageLayout(cmd)
	if !errors.Is(err, api.ErrInvalidPageLayout) {
		t.Fatalf("expected %v, got %v", api.ErrInvalidPageLayout, err)
	}
	if !strings.Contains(err.Error(), `set page layout "bogus"`) {
		t.Fatalf("expected invalid layout context, got %q", err.Error())
	}
}

func TestSetPageLayoutOneColumnReachesAPI(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "test.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	_, err := Dispatch(SetPageLayoutCommand(inFile, outFile, "OneColumn", nil))
	if errors.Is(err, api.ErrInvalidPageLayout) {
		t.Fatalf("OneColumn rejected as invalid: %v", err)
	}
	if err != nil {
		t.Fatalf("set OneColumn page layout: %v", err)
	}
	ss, err := api.ListPageLayoutFile(outFile, nil)
	if err != nil {
		t.Fatalf("list OneColumn page layout: %v", err)
	}
	if len(ss) != 1 || ss[0] != "OneColumn" {
		t.Fatalf("expected OneColumn, got %v", ss)
	}
}

func TestPageModeCommandsRejectMissingInput(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{
			name: "list nil command",
			run: func() error {
				_, err := ListPageMode(nil)
				return err
			},
			want: ErrMissingCommand,
		},
		{
			name: "set missing input",
			run: func() error {
				_, err := SetPageMode(&Command{StringVal: "UseNone"})
				return err
			},
			want: api.ErrMissingPDFInput,
		},
		{
			name: "reset nil command",
			run: func() error {
				_, err := ResetPageMode(nil)
				return err
			},
			want: ErrMissingCommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestSetPageModeRejectsInvalidMode(t *testing.T) {
	cmd := &Command{InFile: stringPtr("missing.pdf"), StringVal: "bogus"}
	_, err := SetPageMode(cmd)
	if !errors.Is(err, api.ErrInvalidPageMode) {
		t.Fatalf("expected %v, got %v", api.ErrInvalidPageMode, err)
	}
	if !strings.Contains(err.Error(), `set page mode "bogus"`) {
		t.Fatalf("expected invalid mode context, got %q", err.Error())
	}
}

func stringPtr(s string) *string {
	return &s
}

func useStdin(t *testing.T, s string) {
	t.Helper()
	useStdinBytes(t, []byte(s))
}

func useStdinBytes(t *testing.T, bb []byte) {
	t.Helper()
	stdin := os.Stdin
	f, err := os.CreateTemp(t.TempDir(), "stdin-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(bb); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = stdin
		_ = f.Close()
	})
}

func requireNoViewerPreferences(t *testing.T, bb []byte) {
	t.Helper()
	vp, _, err := api.ViewerPreferences(bytes.NewReader(bb), nil)
	if err != nil {
		t.Fatalf("read viewer preferences: %v", err)
	}
	if vp != nil {
		t.Fatalf("expected absent viewer preferences, got %+v", vp)
	}
}

func requireMissingFile(t *testing.T, fileName string) {
	t.Helper()
	if _, err := os.Stat(fileName); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got stat error %v", err)
	}
}

func TestContentStreamingFailuresRemoveNewOutput(t *testing.T) {
	tests := []struct {
		name    string
		wantErr error
		run     func(string) error
	}{
		{
			name:    "page layout",
			wantErr: pdfcpu.ErrCorruptHeader,
			run: func(outFile string) error {
				_, err := SetPageLayout(SetPageLayoutCommand("-", outFile, "SinglePage", nil))
				return err
			},
		},
		{
			name:    "page mode",
			wantErr: pdfcpu.ErrCorruptHeader,
			run: func(outFile string) error {
				_, err := SetPageMode(SetPageModeCommand("-", outFile, "UseNone", nil))
				return err
			},
		},
		{
			name:    "viewer preferences",
			wantErr: api.ErrInvalidJSON,
			run: func(outFile string) error {
				_, err := SetViewerPreferences(SetViewerPreferencesCommand("-", "", outFile, "{", nil))
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useStdin(t, "not a pdf")
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			err := tt.run(outFile)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			requireMissingFile(t, outFile)
		})
	}
}

func TestContentStreamingFailurePreservesExistingOutput(t *testing.T) {
	useStdin(t, "not a pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing")
	if err := os.WriteFile(outFile, want, 0600); err != nil {
		t.Fatal(err)
	}

	_, err := SetViewerPreferences(SetViewerPreferencesCommand("-", "", outFile, "{", nil))
	if !errors.Is(err, api.ErrInvalidJSON) {
		t.Fatalf("expected %v, got %v", api.ErrInvalidJSON, err)
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

func TestContentFileFailuresPreserveExistingOutput(t *testing.T) {
	badInput := filepath.Join(t.TempDir(), "bad.pdf")
	if err := os.WriteFile(badInput, nil, 0600); err != nil {
		t.Fatal(err)
	}
	validInput := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name    string
		wantErr error
		run     func(string) error
	}{
		{name: "page layout", wantErr: pdfcpu.ErrEmptyInput, run: func(outFile string) error {
			_, err := SetPageLayout(SetPageLayoutCommand(badInput, outFile, "SinglePage", nil))
			return err
		}},
		{name: "page mode", wantErr: pdfcpu.ErrEmptyInput, run: func(outFile string) error {
			_, err := SetPageMode(SetPageModeCommand(badInput, outFile, "UseNone", nil))
			return err
		}},
		{name: "viewer preferences", wantErr: api.ErrInvalidJSON, run: func(outFile string) error {
			_, err := SetViewerPreferences(SetViewerPreferencesCommand(validInput, "", outFile, "{", nil))
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			want := []byte("existing")
			if err := os.WriteFile(outFile, want, 0600); err != nil {
				t.Fatal(err)
			}
			if err := tt.run(outFile); !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			got, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("existing output changed: got %q, want %q", got, want)
			}
		})
	}
}

func TestContentStreamingSuccessReplacesExistingOutput(t *testing.T) {
	bb, err := os.ReadFile(filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	useStdinBytes(t, bb)
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := ResetPageMode(ResetPageModeCommand("-", outFile, nil)); err != nil {
		t.Fatalf("reset page mode: %v", err)
	}
	if _, err := api.PageModeFile(outFile, nil); err != nil {
		t.Fatalf("read replaced output: %v", err)
	}
}

func TestResetViewerPreferencesAbsentCLIFileAndStdinToFile(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	bb, err := os.ReadFile(inFile)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("file", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "out.pdf")
		if _, err := ResetViewerPreferences(ResetViewerPreferencesCommand(inFile, outFile, nil)); err != nil {
			t.Fatalf("reset viewer preferences file: %v", err)
		}
		out, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}
		requireNoViewerPreferences(t, out)
	})

	t.Run("stdin to file", func(t *testing.T) {
		useStdinBytes(t, bb)
		outFile := filepath.Join(t.TempDir(), "out.pdf")
		if _, err := ResetViewerPreferences(ResetViewerPreferencesCommand("-", outFile, nil)); err != nil {
			t.Fatalf("reset viewer preferences stdin to file: %v", err)
		}
		out, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}
		requireNoViewerPreferences(t, out)
	})
}

func TestResetViewerPreferencesAbsentCLIStdinStdout(t *testing.T) {
	bb, err := os.ReadFile(filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	useStdinBytes(t, bb)

	stdout := os.Stdout
	f, err := os.CreateTemp(t.TempDir(), "stdout-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = f
	t.Cleanup(func() {
		os.Stdout = stdout
		_ = f.Close()
	})

	if _, err := ResetViewerPreferences(ResetViewerPreferencesCommand("-", "-", nil)); err != nil {
		t.Fatalf("reset viewer preferences stdin/stdout: %v", err)
	}
	os.Stdout = stdout
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	requireNoViewerPreferences(t, out)
}

func TestExportBookmarksRemovesOutputOnFailure(t *testing.T) {
	useStdin(t, "not a pdf")
	outFile := filepath.Join(t.TempDir(), "bookmarks.json")

	_, err := ExportBookmarks(ExportBookmarksCommand("-", outFile, nil))
	if err == nil {
		t.Fatal("expected export failure")
	}
	requireMissingFile(t, outFile)
}

// TestExportBookmarksFailurePreservesExistingOutput verifies staged CLI JSON publication.
func TestExportBookmarksFailurePreservesExistingOutput(t *testing.T) {
	useStdin(t, "not a pdf")
	outFile := filepath.Join(t.TempDir(), "bookmarks.json")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0640); err != nil {
		t.Fatal(err)
	}

	_, err := ExportBookmarks(ExportBookmarksCommand("-", outFile, nil))
	if err == nil {
		t.Fatal("expected export failure")
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

func TestExportBookmarksStdoutDoesNotCreateDashFile(t *testing.T) {
	useStdin(t, "not a pdf")
	t.Chdir(t.TempDir())

	_, err := ExportBookmarks(ExportBookmarksCommand("-", "-", nil))
	if err == nil {
		t.Fatal("expected export failure")
	}
	requireMissingFile(t, "-")
}

func TestBookmarkCommandsRejectMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		run     func() error
		wantErr error
	}{
		{
			name: "list nil command",
			run: func() error {
				_, err := ListBookmarks(nil)
				return err
			},
			wantErr: ErrMissingCommand,
		},
		{
			name: "export nil command",
			run: func() error {
				_, err := ExportBookmarks(nil)
				return err
			},
			wantErr: ErrMissingCommand,
		},
		{
			name: "export missing output",
			run: func() error {
				_, err := ExportBookmarks(&Command{InFile: stringPtr("-")})
				return err
			},
			wantErr: api.ErrMissingJSONOutput,
		},
		{
			name: "import nil command",
			run: func() error {
				_, err := ImportBookmarks(nil)
				return err
			},
			wantErr: ErrMissingCommand,
		},
		{
			name: "import missing JSON input",
			run: func() error {
				_, err := ImportBookmarks(&Command{InFile: stringPtr("-")})
				return err
			},
			wantErr: api.ErrMissingJSONInput,
		},
		{
			name: "remove nil command",
			run: func() error {
				_, err := RemoveBookmarks(nil)
				return err
			},
			wantErr: ErrMissingCommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestImportBookmarksRemovesOutputOnFailure(t *testing.T) {
	useStdin(t, "not a pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	missingJSON := filepath.Join(t.TempDir(), "missing.json")

	_, err := ImportBookmarks(ImportBookmarksCommand("-", missingJSON, outFile, false, nil))
	if err == nil {
		t.Fatal("expected import failure")
	}
	requireMissingFile(t, outFile)
}

func TestImportBookmarksMissingJSONPreservesExistingOutput(t *testing.T) {
	useStdin(t, "not a pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}
	missingJSON := filepath.Join(t.TempDir(), "missing.json")

	_, err := ImportBookmarks(ImportBookmarksCommand("-", missingJSON, outFile, false, nil))
	if err == nil {
		t.Fatal("expected import failure")
	}

	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(bb) != "keep me" {
		t.Fatalf("output was changed: %q", string(bb))
	}
}

func TestRemoveBookmarksRemovesOutputOnFailure(t *testing.T) {
	useStdin(t, "not a pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	_, err := RemoveBookmarks(RemoveBookmarksCommand("-", outFile, nil))
	if err == nil {
		t.Fatal("expected remove failure")
	}
	requireMissingFile(t, outFile)
}
