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
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type importImageTestCloser struct {
	err error
}

func (c importImageTestCloser) Close() error {
	return c.err
}

type importImageFailingReader struct {
	err error
}

func (r importImageFailingReader) Read([]byte) (int, error) {
	return 0, r.err
}

// TestImportImagesRejectsMalformedCommands verifies malformed import commands use API sentinels.
func TestImportImagesRejectsMalformedCommands(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	tests := []struct {
		name string
		cmd  *Command
		want error
	}{
		{name: "nil command", want: ErrMissingCommand},
		{name: "missing images", cmd: &Command{OutFile: &outFile}, want: api.ErrMissingImageInput},
		{
			name: "empty image",
			cmd:  &Command{InFiles: []string{""}, OutFile: &outFile},
			want: api.ErrMissingImageInput,
		},
		{
			name: "missing output",
			cmd:  &Command{InFiles: []string{"image.png"}},
			want: api.ErrMissingPDFOutput,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ImportImages(tt.cmd); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

// TestImportImagesRejectsDuplicateStdinBeforeRead verifies malformed input does not consume stdin.
func TestImportImagesRejectsDuplicateStdinBeforeRead(t *testing.T) {
	source := setResourcesStdin(t, "image")
	outFile := "-"
	imp := api.DefaultImportConfig()
	imp.PageDim = nil
	_, err := ImportImages(&Command{
		InFiles: []string{"-", "-"},
		OutFile: &outFile,
		Import:  imp,
	})
	if err == nil || !strings.Contains(err.Error(), "import images: only one image may read from stdin") {
		t.Fatalf("expected duplicate stdin error, got %v", err)
	}
	offset, seekErr := source.Seek(0, io.SeekCurrent)
	if seekErr != nil {
		t.Fatal(seekErr)
	}
	if offset != 0 {
		t.Fatalf("stdin was consumed before validation: offset=%d", offset)
	}
}

// TestImportImagesPreflightsConfigurationBeforeInputIO verifies CLI uses the API-owned preflight.
func TestImportImagesPreflightsConfigurationBeforeInputIO(t *testing.T) {
	source := setResourcesStdin(t, "not an image")
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.pdf")
	imp := api.DefaultImportConfig()
	imp.PageDim = nil

	_, err := ImportImages(&Command{
		InFiles: []string{filepath.Join(dir, "missing.png"), "-"},
		OutFile: &outFile,
		Import:  imp,
	})
	if !errors.Is(err, api.ErrInvalidImportConfiguration) {
		t.Fatalf("expected %v, got %v", api.ErrInvalidImportConfiguration, err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("configuration preflight reached filesystem access: %v", err)
	}
	offset, seekErr := source.Seek(0, io.SeekCurrent)
	if seekErr != nil {
		t.Fatal(seekErr)
	}
	if offset != 0 {
		t.Fatalf("stdin was consumed before configuration rejection: offset=%d", offset)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("configuration preflight created output: %v", statErr)
	}
}

// TestImportImagesStdinLimitPlusOne verifies bounded stdin acquisition uses model limit wording.
func TestImportImagesStdinLimitPlusOne(t *testing.T) {
	setResourcesStdin(t, "1234")
	outFile := "-"
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxStreamBytes = 3

	_, err := ImportImages(&Command{
		InFiles: []string{"-"},
		OutFile: &outFile,
		Import:  api.DefaultImportConfig(),
		Conf:    conf,
	})
	const want = "import images: image 1 stdin: read: input size 4 exceeds limit 3"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

// TestImportImageReaderAcceptsExactStdinLimit verifies exact-limit input is retained in full.
func TestImportImageReaderAcceptsExactStdinLimit(t *testing.T) {
	setResourcesStdin(t, "123")
	r, closer, err := importImageReader("-", 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if closer != nil {
		t.Fatal("stdin reader unexpectedly requires closing")
	}
	bb, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(bb) != "123" {
		t.Fatalf("got %q, want %q", bb, "123")
	}
}

// TestReadImportImageStdinPreservesFailure verifies read causes remain discoverable.
func TestReadImportImageStdinPreservesFailure(t *testing.T) {
	wantErr := errors.New("stdin read failure")
	_, err := readImportImageStdin(importImageFailingReader{err: wantErr}, 2, 10)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	const want = "import images: image 2 stdin: read"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

// TestImportImageReaderHandlesMaxInt64Limit verifies limit increment does not overflow.
func TestImportImageReaderHandlesMaxInt64Limit(t *testing.T) {
	setResourcesStdin(t, "x")
	r, closer, err := importImageReader("-", 1, math.MaxInt64)
	if err != nil {
		t.Fatal(err)
	}
	if closer != nil {
		t.Fatal("stdin reader unexpectedly requires closing")
	}
	bb, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(bb) != "x" {
		t.Fatalf("got %q, want %q", bb, "x")
	}
}

// TestImportImagesRejectsOutputAliasBeforeSourceIO verifies CLI preflight precedes files and stdin.
func TestImportImagesRejectsOutputAliasBeforeSourceIO(t *testing.T) {
	source := setResourcesStdin(t, "not an image")
	dir := t.TempDir()
	missingImage := filepath.Join(dir, "missing.png")
	imageFile := filepath.Join(dir, "image.png")
	want := []byte("image input")
	if err := os.WriteFile(imageFile, want, 0600); err != nil {
		t.Fatal(err)
	}

	_, err := ImportImages(&Command{
		InFiles: []string{missingImage, "-", imageFile},
		OutFile: &imageFile,
		Import:  api.DefaultImportConfig(),
	})
	if !errors.Is(err, api.ErrImportImagesOutputConflict) {
		t.Fatalf("expected %v, got %v", api.ErrImportImagesOutputConflict, err)
	}
	wantContext := fmt.Sprintf("image 3 %q: output aliases input", imageFile)
	if !strings.Contains(err.Error(), wantContext) {
		t.Fatalf("expected %q, got %q", wantContext, err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("alias preflight opened an earlier missing image: %v", err)
	}
	offset, seekErr := source.Seek(0, io.SeekCurrent)
	if seekErr != nil {
		t.Fatal(seekErr)
	}
	if offset != 0 {
		t.Fatalf("stdin was consumed before alias rejection: offset=%d", offset)
	}
	got, readErr := os.ReadFile(imageFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != string(want) {
		t.Fatalf("aliased image input changed: got %q, want %q", got, want)
	}
}

// TestImportImagesStreamingRejectsOutputAliasesPreservingInput verifies shared alias policy before stdin I/O.
func TestImportImagesStreamingRejectsOutputAliasesPreservingInput(t *testing.T) {
	tests := []struct {
		name  string
		alias func(string, string) error
	}{
		{name: "identical path"},
		{name: "hard link", alias: os.Link},
		{name: "symlink", alias: os.Symlink},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := setResourcesStdin(t, "not an image")
			dir := t.TempDir()
			imageFile := filepath.Join(dir, "scan.jpg")
			want := []byte("original image")
			if err := os.WriteFile(imageFile, want, 0600); err != nil {
				t.Fatal(err)
			}
			outFile := imageFile
			if tt.alias != nil {
				outFile = filepath.Join(dir, "out.pdf")
				if err := tt.alias(imageFile, outFile); err != nil {
					t.Fatal(err)
				}
			}

			_, err := ImportImages(&Command{
				InFiles: []string{"-", imageFile},
				OutFile: &outFile,
				Import:  api.DefaultImportConfig(),
			})
			if !errors.Is(err, api.ErrImportImagesOutputConflict) {
				t.Fatalf("expected %v, got %v", api.ErrImportImagesOutputConflict, err)
			}
			wantContext := fmt.Sprintf("image 2 %q: output aliases input", imageFile)
			if !strings.Contains(err.Error(), wantContext) {
				t.Fatalf("expected %q, got %q", wantContext, err)
			}
			offset, seekErr := source.Seek(0, io.SeekCurrent)
			if seekErr != nil {
				t.Fatal(seekErr)
			}
			if offset != 0 {
				t.Fatalf("stdin was consumed before alias rejection: offset=%d", offset)
			}
			got, readErr := os.ReadFile(imageFile)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(got) != string(want) {
				t.Fatalf("image changed: got %q, want %q", got, want)
			}
		})
	}
}

// TestImportImagesStreamingPreflightDoesNotCreateOutput verifies rejection precedes stdin and output I/O.
func TestImportImagesStreamingPreflightDoesNotCreateOutput(t *testing.T) {
	source := setResourcesStdin(t, "not an image")
	outFile := filepath.Join(t.TempDir(), "missing.png")
	_, err := ImportImages(&Command{
		InFiles: []string{"-", outFile},
		OutFile: &outFile,
		Import:  api.DefaultImportConfig(),
	})
	if !errors.Is(err, api.ErrImportImagesOutputConflict) {
		t.Fatalf("expected %v, got %v", api.ErrImportImagesOutputConflict, err)
	}
	offset, seekErr := source.Seek(0, io.SeekCurrent)
	if seekErr != nil {
		t.Fatal(seekErr)
	}
	if offset != 0 {
		t.Fatalf("stdin was consumed before alias rejection: offset=%d", offset)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("preflight created output: %v", statErr)
	}
}

// TestImportImagesSourceErrorsIncludeCLIContext verifies file and stdin failures identify their source.
func TestImportImagesSourceErrorsIncludeCLIContext(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		setResourcesStdin(t, "image")
		missing := filepath.Join(t.TempDir(), "missing.png")
		outFile := "-"
		_, err := ImportImages(&Command{
			InFiles: []string{"-", missing},
			OutFile: &outFile,
			Import:  api.DefaultImportConfig(),
		})
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
		}
		want := fmt.Sprintf("import images: image 2 %q: open", missing)
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("stdin", func(t *testing.T) {
		setResourcesStdin(t, "")
		outFile := "-"
		_, err := ImportImages(&Command{
			InFiles: []string{"-"},
			OutFile: &outFile,
			Import:  api.DefaultImportConfig(),
		})
		if err == nil || !strings.Contains(err.Error(), "import images: image 1 stdin: read: stdin is empty") {
			t.Fatalf("expected stdin context, got %v", err)
		}
	})
}

// TestCloseImportImageInputsPreservesAllErrors verifies CLI cleanup reports every source failure.
func TestCloseImportImageInputsPreservesAllErrors(t *testing.T) {
	err1 := errors.New("close image 1")
	err2 := errors.New("close image 2")
	err := closeImportImageInputs([]importImageInputCloser{
		{closer: importImageTestCloser{err: err1}, imageIndex: 1, fileName: "one.png"},
		{closer: importImageTestCloser{err: err2}, imageIndex: 3, fileName: "three.png"},
	})
	for _, want := range []error{err1, err2} {
		if !errors.Is(err, want) {
			t.Fatalf("expected %v, got %v", want, err)
		}
	}
	for _, want := range []string{
		`import images: image 1 "one.png": close`,
		`import images: image 3 "three.png": close`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
}

// TestImportImagesStreamingOutputErrorsIncludeCLIContext verifies destination setup attribution.
func TestImportImagesStreamingOutputErrorsIncludeCLIContext(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		setResourcesStdin(t, "image")
		outFile := filepath.Join(t.TempDir(), "missing", "out.pdf")
		_, err := ImportImages(&Command{
			InFiles: []string{"-"},
			OutFile: &outFile,
			Import:  api.DefaultImportConfig(),
		})
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
		}
		if !strings.Contains(err.Error(), "import images: create output") {
			t.Fatalf("expected output creation context, got %q", err)
		}
	})

	t.Run("inspect", func(t *testing.T) {
		setResourcesStdin(t, "image")
		dir := t.TempDir()
		outFile := filepath.Join(dir, "out.pdf")
		if err := os.Symlink(filepath.Base(outFile), outFile); err != nil {
			t.Fatal(err)
		}
		_, err := ImportImages(&Command{
			InFiles: []string{"-"},
			OutFile: &outFile,
			Import:  api.DefaultImportConfig(),
		})
		if err == nil || !strings.Contains(err.Error(), "import images: inspect output") {
			t.Fatalf("expected output inspection context, got %v", err)
		}
	})
}

// TestImportImagesStreamingPreservesAPIContext verifies CLI handling does not duplicate operation phases.
func TestImportImagesStreamingPreservesAPIContext(t *testing.T) {
	setResourcesStdin(t, "not an image")
	outFile := "-"
	_, err := ImportImages(&Command{
		InFiles: []string{"-"},
		OutFile: &outFile,
		Import:  api.DefaultImportConfig(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	const context = "import images: image 1: create pages"
	if count := strings.Count(err.Error(), context); count != 1 {
		t.Fatalf("expected %q once, got %d occurrences in %q", context, count, err)
	}
}

// TestImportImagesStreamingFailurePreservesExistingOutput verifies failed stdin imports do not replace output.
func TestImportImagesStreamingFailurePreservesExistingOutput(t *testing.T) {
	setResourcesStdin(t, "not an image")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	const original = "original output"
	if err := os.WriteFile(outFile, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := ImportImages(&Command{
		InFiles: []string{"-"},
		OutFile: &outFile,
		Import:  api.DefaultImportConfig(),
	})
	if err == nil || !strings.Contains(err.Error(), "import images: prepare PDF context") {
		t.Fatalf("expected API context preparation error, got %v", err)
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(bb) != original {
		t.Fatalf("existing output changed: %q", bb)
	}
}

// TestListImagesFileUsesAPIBoundary verifies per-file listing adds I/O context and preserves API errors.
func TestListImagesFileUsesAPIBoundary(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.pdf")
	if _, err := listImagesFile(missing, nil, nil); err == nil || !strings.Contains(err.Error(), "list images: open input") {
		t.Fatalf("expected contextual open error, got %v", err)
	}

	inFile := filepath.Join(t.TempDir(), "empty.pdf")
	if err := os.WriteFile(inFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := listImagesFile(inFile, nil, nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "list images: prepare PDF context") {
		t.Fatalf("expected API list context, got %q", err)
	}
}

// TestListImagesRejectsMalformedCommands verifies malformed list commands use the API sentinel.
func TestListImagesRejectsMalformedCommands(t *testing.T) {
	tests := []struct {
		name string
		cmd  *Command
	}{
		{name: "nil command"},
		{name: "missing inputs", cmd: &Command{}},
		{name: "empty input", cmd: &Command{InFiles: []string{""}}},
		{name: "empty second input", cmd: &Command{InFiles: []string{"input.pdf", ""}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := error(api.ErrMissingPDFInput)
			if tt.cmd == nil {
				want = ErrMissingCommand
			}
			if _, err := ListImages(tt.cmd); !errors.Is(err, want) {
				t.Fatalf("expected %v, got %v", want, err)
			}
		})
	}
}

// TestListImagesFileRejectsEmptyInputSlice verifies direct file listing requires input.
func TestListImagesFileRejectsEmptyInputSlice(t *testing.T) {
	if _, err := ListImagesFile(nil, nil, nil); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}
}

// TestListImagesFileRejectsEmptyInputEntry verifies every direct listing input must be named.
func TestListImagesFileRejectsEmptyInputEntry(t *testing.T) {
	if _, err := ListImagesFile([]string{"input.pdf", ""}, nil, nil); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}
}

// TestListImagesFileDoesNotDuplicateFileNameContext verifies multi-file errors name each input once.
func TestListImagesFileDoesNotDuplicateFileNameContext(t *testing.T) {
	tmpDir := t.TempDir()
	inFiles := []string{
		filepath.Join(tmpDir, "missing1.pdf"),
		filepath.Join(tmpDir, "missing2.pdf"),
	}
	_, err := ListImagesFile(inFiles, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, inFile := range inFiles {
		if count := strings.Count(err.Error(), inFile); count != 1 {
			t.Fatalf("expected %s once, got %d occurrences in %q", inFile, count, err)
		}
	}
}

// TestListImagesFileNamesPerFileProcessingError verifies processing failures identify the open input that failed.
func TestListImagesFileNamesPerFileProcessingError(t *testing.T) {
	tmpDir := t.TempDir()
	validFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	invalidFile := filepath.Join(tmpDir, "invalid.pdf")
	if err := os.WriteFile(invalidFile, nil, 0600); err != nil {
		t.Fatal(err)
	}

	output, err := ListImagesFile([]string{validFile, invalidFile}, nil, nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if count := strings.Count(err.Error(), invalidFile); count != 1 {
		t.Fatalf("expected failing input once, got %d occurrences in %q", count, err)
	}
	if strings.Contains(err.Error(), validFile) {
		t.Fatalf("successful input attributed to error: %q", err)
	}
	if len(output) == 0 || output[0] != "\n"+validFile+":" {
		t.Fatalf("expected successful input output, got %v", output)
	}
}

// TestListImagesMixedFileAndStdinAttribution verifies stdin processing failures are attributed independently.
func TestListImagesMixedFileAndStdinAttribution(t *testing.T) {
	setResourcesStdin(t, "not a PDF")
	validFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")

	output, err := ListImages(&Command{InFiles: []string{validFile, "-"}})
	if err == nil {
		t.Fatal("expected stdin processing error")
	}
	if !strings.Contains(err.Error(), "stdin: list images: prepare PDF context") {
		t.Fatalf("expected stdin API context, got %q", err)
	}
	if strings.Contains(err.Error(), validFile) {
		t.Fatalf("successful file attributed to error: %q", err)
	}
	if len(output) == 0 || output[0] != "\n"+validFile+":" {
		t.Fatalf("expected successful file output, got %v", output)
	}
}

// TestListImagesRejectsDuplicateStdinBeforeRead verifies stdin is not consumed for malformed input.
func TestListImagesRejectsDuplicateStdinBeforeRead(t *testing.T) {
	source := setResourcesStdin(t, "not a PDF")
	_, err := ListImages(&Command{InFiles: []string{"-", "-"}})
	if err == nil || !strings.Contains(err.Error(), "list images: only one input may read from stdin") {
		t.Fatalf("expected duplicate stdin error, got %v", err)
	}
	offset, seekErr := source.Seek(0, io.SeekCurrent)
	if seekErr != nil {
		t.Fatal(seekErr)
	}
	if offset != 0 {
		t.Fatalf("stdin was consumed before validation: offset=%d", offset)
	}
}

// TestUpdateImagesRejectsMalformedCommands verifies malformed update commands use API sentinels.
func TestUpdateImagesRejectsMalformedCommands(t *testing.T) {
	tests := []struct {
		name string
		cmd  *Command
		want error
	}{
		{name: "nil command", want: ErrMissingCommand},
		{name: "missing inputs", cmd: &Command{}, want: api.ErrMissingPDFInput},
		{name: "empty PDF input", cmd: &Command{InFiles: []string{""}}, want: api.ErrMissingPDFInput},
		{name: "missing image input", cmd: &Command{InFiles: []string{"input.pdf"}}, want: api.ErrMissingImageInput},
		{name: "empty image input", cmd: &Command{InFiles: []string{"input.pdf", ""}}, want: api.ErrMissingImageInput},
		{name: "missing output", cmd: &Command{InFiles: []string{"input.pdf", "image.png"}}, want: api.ErrMissingPDFOutput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UpdateImages(tt.cmd)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

// TestUpdateImagesRejectsSurplusInput verifies command validation requires exactly two inputs.
func TestUpdateImagesRejectsSurplusInput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	_, err := UpdateImages(&Command{
		InFiles: []string{"input.pdf", "image.png", "surplus.png"},
		OutFile: &outFile,
	})
	if err == nil || !strings.Contains(err.Error(), "update images: expected exactly two inputs, got 3") {
		t.Fatalf("expected surplus-input error, got %v", err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("surplus inputs reached file handling: %v", err)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("surplus inputs created output: %v", statErr)
	}
}

// TestUpdateImageParamsPreservesSelectors verifies the CLI leaves selector validation to the API.
func TestUpdateImageParamsPreservesSelectors(t *testing.T) {
	tests := []struct {
		name          string
		intVal        int
		stringVal     string
		objNr, pageNr int
		id            string
	}{
		{name: "positive object", intVal: 3, objNr: 3},
		{name: "negative object", intVal: -3, objNr: -3},
		{name: "zero object"},
		{name: "positive page", intVal: 4, stringVal: "Im0", pageNr: 4, id: "Im0"},
		{name: "negative page", intVal: -4, stringVal: "Im0", pageNr: -4, id: "Im0"},
		{name: "zero page", stringVal: "Im0", id: "Im0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objNr, pageNr, id := updateImageParams(&Command{IntVal: tt.intVal, StringVal: tt.stringVal})
			if objNr != tt.objNr || pageNr != tt.pageNr || id != tt.id {
				t.Fatalf("expected (%d, %d, %q), got (%d, %d, %q)", tt.objNr, tt.pageNr, tt.id, objNr, pageNr, id)
			}
		})
	}
}

// TestUpdateImagesPreservesInvalidSelectors verifies the API receives negative and partial selectors unchanged.
func TestUpdateImagesPreservesInvalidSelectors(t *testing.T) {
	outFile := "out.pdf"
	tests := []struct {
		name      string
		intVal    int
		stringVal string
		context   string
	}{
		{name: "negative object", intVal: -3, context: "negative object number -3"},
		{name: "negative page", intVal: -4, stringVal: "Im0", context: "missing page number"},
		{name: "partial page resource", stringVal: "Im0", context: "missing page number"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UpdateImages(&Command{
				InFiles:   []string{"input.pdf", "image.png"},
				OutFile:   &outFile,
				IntVal:    tt.intVal,
				StringVal: tt.stringVal,
			})
			if !errors.Is(err, api.ErrInvalidImageSelection) {
				t.Fatalf("expected %v, got %v", api.ErrInvalidImageSelection, err)
			}
			if !strings.Contains(err.Error(), tt.context) {
				t.Fatalf("expected %q context, got %q", tt.context, err)
			}
		})
	}
}

// TestListImagesFileJoinsContextualCloseError verifies close failures retain API and filesystem causes.
func TestListImagesFileJoinsContextualCloseError(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "empty.pdf")
	if err := os.WriteFile(inFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("close failed")
	closeFile := closeListImagesInput
	closeListImagesInput = func(f *os.File) error {
		if err := closeFile(f); err != nil {
			t.Fatal(err)
		}
		return wantErr
	}
	t.Cleanup(func() {
		closeListImagesInput = closeFile
	})

	_, err := listImagesFile(inFile, nil, nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "list images: close input "+inFile) {
		t.Fatalf("expected contextual close error, got %q", err)
	}
}

// TestListImagesFileClosesEachInputImmediately verifies multi-file listing does not accumulate descriptors.
func TestListImagesFileClosesEachInputImmediately(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	openFile := openListImagesInput
	closeFile := closeListImagesInput
	active := 0
	maxActive := 0
	openListImagesInput = func(name string) (*os.File, error) {
		f, err := openFile(name)
		if err != nil {
			return nil, err
		}
		active++
		if active > maxActive {
			maxActive = active
		}
		return f, nil
	}
	closeListImagesInput = func(f *os.File) error {
		err := closeFile(f)
		active--
		return err
	}
	t.Cleanup(func() {
		openListImagesInput = openFile
		closeListImagesInput = closeFile
	})

	inFiles := []string{inFile, inFile, inFile, inFile}
	if _, err := ListImagesFile(inFiles, nil, nil); err != nil {
		t.Fatal(err)
	}
	if maxActive != 1 {
		t.Fatalf("expected at most one active list input, got %d", maxActive)
	}
	if active != 0 {
		t.Fatalf("expected all list inputs closed, got %d active", active)
	}
}

func setResourcesStdin(t *testing.T, content string) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdin-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	stdin := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = stdin
		_ = f.Close()
	})
	return f
}

// TestUpdateImagesMissingImagePreservesExistingOutput verifies image open failure precedes output creation.
func TestUpdateImagesMissingImagePreservesExistingOutput(t *testing.T) {
	source := setResourcesStdin(t, "not a PDF")
	tmpDir := t.TempDir()
	imageFile := filepath.Join(tmpDir, "missing.png")
	outFile := filepath.Join(tmpDir, "out.pdf")
	const original = "original output"
	if err := os.WriteFile(outFile, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := UpdateImages(&Command{
		InFiles: []string{"-", imageFile},
		OutFile: &outFile,
		IntVal:  1,
	})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "update images: open image "+imageFile) {
		t.Fatalf("expected contextual image open error, got %q", err)
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(bb) != original {
		t.Fatalf("existing output changed: %q", bb)
	}
	offset, err := source.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if offset != 0 {
		t.Fatalf("stdin was consumed before image validation: offset=%d", offset)
	}
}

// TestUpdateImagesStreamingRejectsImageOutputAliases verifies streaming preflight detects filesystem aliases.
func TestUpdateImagesStreamingRejectsImageOutputAliases(t *testing.T) {
	tests := []struct {
		name  string
		alias func(string, string) error
	}{
		{name: "same path"},
		{name: "hard link", alias: os.Link},
		{name: "symlink", alias: os.Symlink},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := setResourcesStdin(t, "not a PDF")
			tmpDir := t.TempDir()
			imageFile := filepath.Join(tmpDir, "image.png")
			want := []byte("image input")
			if err := os.WriteFile(imageFile, want, 0600); err != nil {
				t.Fatal(err)
			}
			outFile := imageFile
			if tt.alias != nil {
				outFile = filepath.Join(tmpDir, "out.pdf")
				if err := tt.alias(imageFile, outFile); err != nil {
					t.Fatal(err)
				}
			}

			_, err := UpdateImages(&Command{
				InFiles: []string{"-", imageFile},
				OutFile: &outFile,
				IntVal:  1,
			})
			if !errors.Is(err, api.ErrUpdateImagesOutputConflict) {
				t.Fatalf("expected %v, got %v", api.ErrUpdateImagesOutputConflict, err)
			}
			if !strings.Contains(err.Error(), "aliases image input") {
				t.Fatalf("expected alias context, got %q", err)
			}
			got, err := os.ReadFile(imageFile)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Fatalf("image input changed: got %q, want %q", got, want)
			}
			offset, err := source.Seek(0, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			if offset != 0 {
				t.Fatalf("stdin was consumed before alias validation: offset=%d", offset)
			}
		})
	}
}

// TestUpdateImagesStreamingFailurePreservesOutput verifies failed stdin updates do not truncate existing output.
func TestUpdateImagesStreamingFailurePreservesOutput(t *testing.T) {
	tmpDir := t.TempDir()
	stdinFile := filepath.Join(tmpDir, "stdin.pdf")
	if err := os.WriteFile(stdinFile, []byte("not a PDF"), 0600); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(stdinFile)
	if err != nil {
		t.Fatal(err)
	}
	stdin := os.Stdin
	os.Stdin = source
	t.Cleanup(func() {
		os.Stdin = stdin
		_ = source.Close()
	})

	imageFile := filepath.Join(tmpDir, "image.png")
	if err := os.WriteFile(imageFile, []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(tmpDir, "out.pdf")
	const original = "original output"
	if err := os.WriteFile(outFile, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := &Command{
		InFiles: []string{"-", imageFile},
		OutFile: &outFile,
		IntVal:  -1,
	}
	_, err = UpdateImages(cmd)
	if !errors.Is(err, api.ErrInvalidImageSelection) {
		t.Fatalf("expected %v, got %v", api.ErrInvalidImageSelection, err)
	}
	if !strings.Contains(err.Error(), "negative object number -1") {
		t.Fatalf("expected preserved selector, got %q", err)
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(bb) != original {
		t.Fatalf("existing output changed: %q", bb)
	}
}
