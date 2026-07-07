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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestRotateRejectsMissingCommandFields verifies rotate command boundary guards.
func TestRotateRejectsMissingCommandFields(t *testing.T) {
	if _, err := Rotate(nil); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}
	if _, err := Rotate(&Command{}); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}
	inFile := "-"
	if _, err := Rotate(&Command{InFile: &inFile}); !errors.Is(err, api.ErrMissingPDFOutput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFOutput, err)
	}
}

// TestInsertRemovePagesRejectMissingCommandFields verifies page command boundary guards.
func TestInsertRemovePagesRejectMissingCommandFields(t *testing.T) {
	operations := []struct {
		name string
		fn   func(*Command) ([]string, error)
	}{
		{name: "insert pages", fn: InsertPages},
		{name: "remove pages", fn: RemovePages},
	}

	for _, tt := range operations {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.fn(nil); !errors.Is(err, api.ErrMissingPDFInput) {
				t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
			}
			if _, err := tt.fn(&Command{}); !errors.Is(err, api.ErrMissingPDFInput) {
				t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
			}
			empty := ""
			fileOut := filepath.Join(t.TempDir(), "out.pdf")
			if _, err := tt.fn(&Command{InFile: &empty, OutFile: &fileOut}); !errors.Is(err, api.ErrMissingPDFInput) {
				t.Fatalf("file route: expected %v, got %v", api.ErrMissingPDFInput, err)
			}
			streamOut := "-"
			if _, err := tt.fn(&Command{InFile: &empty, OutFile: &streamOut}); !errors.Is(err, api.ErrMissingPDFInput) {
				t.Fatalf("streaming route: expected %v, got %v", api.ErrMissingPDFInput, err)
			}
			inFile := "-"
			if _, err := tt.fn(&Command{InFile: &inFile}); !errors.Is(err, api.ErrMissingPDFOutput) {
				t.Fatalf("expected %v, got %v", api.ErrMissingPDFOutput, err)
			}
		})
	}
}

// TestInsertRemovePagesAllowEmptyOutput verifies empty output remains valid for files and stdout.
func TestInsertRemovePagesAllowEmptyOutput(t *testing.T) {
	operations := []struct {
		name string
		fn   func(*Command) ([]string, error)
	}{
		{name: "insert pages", fn: InsertPages},
		{name: "remove pages", fn: RemovePages},
	}

	for _, tt := range operations {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("file", func(t *testing.T) {
				inFile := filepath.Join(t.TempDir(), "missing.pdf")
				outFile := ""
				_, err := tt.fn(&Command{InFile: &inFile, OutFile: &outFile})
				if errors.Is(err, api.ErrMissingPDFOutput) {
					t.Fatalf("empty output must remain valid: %v", err)
				}
				if !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("expected input open error, got %v", err)
				}
			})

			t.Run("stdout", func(t *testing.T) {
				stdin, err := os.CreateTemp(t.TempDir(), "empty-stdin-*")
				if err != nil {
					t.Fatal(err)
				}
				previousStdin := os.Stdin
				os.Stdin = stdin
				t.Cleanup(func() {
					os.Stdin = previousStdin
					_ = stdin.Close()
				})

				inFile, outFile := "-", ""
				_, err = tt.fn(&Command{InFile: &inFile, OutFile: &outFile})
				if errors.Is(err, api.ErrMissingPDFOutput) {
					t.Fatalf("empty output must remain valid: %v", err)
				}
				if err == nil || !strings.Contains(err.Error(), tt.name+": read stdin") {
					t.Fatalf("expected stdin read error, got %v", err)
				}
			})
		})
	}
}

// TestInsertRemovePagesStreamingFailurePreservesExistingOutput verifies protected CLI output.
func TestInsertRemovePagesStreamingFailurePreservesExistingOutput(t *testing.T) {
	operations := []struct {
		name string
		fn   func(*Command) ([]string, error)
		want string
	}{
		{name: "insert pages", fn: InsertPages, want: "insert pages: parse page selection"},
		{name: "remove pages", fn: RemovePages, want: "remove pages: parse page selection"},
	}

	for _, tt := range operations {
		t.Run(tt.name, func(t *testing.T) {
			pageStreamingStdin(t)
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			want := []byte("existing output")
			if err := os.WriteFile(outFile, want, 0o600); err != nil {
				t.Fatal(err)
			}
			inFile := "-"
			_, err := tt.fn(&Command{
				InFile:        &inFile,
				OutFile:       &outFile,
				PageSelection: []string{"foo"},
			})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
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

func pageStreamingStdin(t *testing.T) {
	t.Helper()
	sourcePath := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	b, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.CreateTemp(t.TempDir(), "rotate-stdin-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(source, bytes.NewReader(b)); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	stdin := os.Stdin
	os.Stdin = source
	t.Cleanup(func() {
		os.Stdin = stdin
		_ = source.Close()
	})
}

// TestRotateStreamingFailurePreservesExistingOutput verifies protected CLI streaming output.
func TestRotateStreamingFailurePreservesExistingOutput(t *testing.T) {
	pageStreamingStdin(t)
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Rotate(RotateCommand("-", outFile, 90, []string{"foo"}, nil))
	if err == nil || !strings.Contains(err.Error(), "rotate: parse page selection") {
		t.Fatalf("expected rotate page-selection error, got %v", err)
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

// TestRotateStreamingIOErrorsIncludeOperationContext verifies rotate-specific I/O context.
func TestRotateStreamingIOErrorsIncludeOperationContext(t *testing.T) {
	outFile := "-"
	missingInput := filepath.Join(t.TempDir(), "missing.pdf")
	_, err := Rotate(&Command{InFile: &missingInput, OutFile: &outFile, IntVal: 90})
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "rotate: open input "+missingInput) {
		t.Fatalf("expected rotate input context, got %v", err)
	}

	pageStreamingStdin(t)
	inFile := "-"
	missingOutput := filepath.Join(t.TempDir(), "missing", "out.pdf")
	_, err = Rotate(&Command{InFile: &inFile, OutFile: &missingOutput, IntVal: 90})
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "rotate: create output "+missingOutput) {
		t.Fatalf("expected rotate output context, got %v", err)
	}
}

// TestRotateStreamingFinalizationPreservesCloseAndRemovalErrors verifies joined cleanup failures.
func TestRotateStreamingFinalizationPreservesCloseAndRemovalErrors(t *testing.T) {
	output, err := os.CreateTemp(t.TempDir(), "rotate-output-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	nonEmptyDir := filepath.Join(t.TempDir(), "output")
	if err := os.Mkdir(nonEmptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "keep"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	primaryErr := errors.New("rotate operation failed")
	finalizer := &streamInOutFinalizer{output: output, outFile: nonEmptyDir}

	err = finalizer.finalize("rotate", primaryErr)
	if !errors.Is(err, primaryErr) || !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected operation and close errors, got %v", err)
	}
	for _, want := range []string{"rotate: close output", "rotate: remove output"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestNUpImageStdoutDoesNotOpenPDFInput(t *testing.T) {
	nup, err := api.ImageNUpConfig(4, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	missingImage := filepath.Join(t.TempDir(), "missing.png")
	cmd := NUpCommand([]string{missingImage}, "-", nil, nup, nil)
	defer log.SetDefaultCLILogger()

	_, err = NUp(cmd)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "n-up image 1") {
		t.Fatalf("expected indexed image context, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "n-up: open input") {
		t.Fatalf("unexpected PDF input context: %q", err.Error())
	}
}

func TestNUpRejectsMissingCommandFields(t *testing.T) {
	_, err := NUp(nil)
	if !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}

	cmd := &Command{}
	_, err = NUp(cmd)
	if !errors.Is(err, api.ErrMissingPDFOutput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFOutput, err)
	}

	cmd = NUpCommand([]string{"-"}, "-", nil, nil, nil)
	_, err = NUp(cmd)
	if !errors.Is(err, api.ErrMissingNUpConfiguration) {
		t.Fatalf("expected %v, got %v", api.ErrMissingNUpConfiguration, err)
	}
}

func TestGridRejectsMissingCommandFields(t *testing.T) {
	_, err := Grid(nil)
	if !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}

	cmd := &Command{}
	_, err = Grid(cmd)
	if !errors.Is(err, api.ErrMissingPDFOutput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFOutput, err)
	}

	cmd = GridCommand([]string{"-"}, "-", nil, nil, nil)
	_, err = Dispatch(cmd)
	if !errors.Is(err, api.ErrMissingGridConfiguration) {
		t.Fatalf("expected %v, got %v", api.ErrMissingGridConfiguration, err)
	}
}

func TestBoxCommandsRejectMissingFields(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func(*Command) error
	}{
		{name: "list", run: func(cmd *Command) error {
			_, err := ListBoxes(cmd)
			return err
		}},
		{name: "add", run: func(cmd *Command) error {
			_, err := AddBoxes(cmd)
			return err
		}},
		{name: "remove", run: func(cmd *Command) error {
			_, err := RemoveBoxes(cmd)
			return err
		}},
		{name: "crop", run: func(cmd *Command) error {
			_, err := Crop(cmd)
			return err
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(nil); !errors.Is(err, api.ErrMissingPDFInput) {
				t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
			}
			if err := tt.run(&Command{}); !errors.Is(err, api.ErrMissingPDFInput) {
				t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
			}
		})
	}

	inFile := "in.pdf"
	outFile := "out.pdf"
	for _, tt := range []struct {
		name string
		run  func(*Command) error
		want error
	}{
		{name: "add output", run: func(cmd *Command) error {
			_, err := AddBoxes(cmd)
			return err
		}, want: api.ErrMissingPDFOutput},
		{name: "remove output", run: func(cmd *Command) error {
			_, err := RemoveBoxes(cmd)
			return err
		}, want: api.ErrMissingPDFOutput},
		{name: "crop output", run: func(cmd *Command) error {
			_, err := Crop(cmd)
			return err
		}, want: api.ErrMissingPDFOutput},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(&Command{InFile: &inFile}); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}

	configTests := []struct {
		name string
		run  func(*Command) error
		want error
	}{
		{name: "add boundaries", run: func(cmd *Command) error {
			_, err := AddBoxes(cmd)
			return err
		}, want: api.ErrMissingPageBoundaries},
		{name: "remove boundaries", run: func(cmd *Command) error {
			_, err := RemoveBoxes(cmd)
			return err
		}, want: api.ErrMissingPageBoundaries},
		{name: "crop box", run: func(cmd *Command) error {
			_, err := Crop(cmd)
			return err
		}, want: api.ErrMissingBoxConfiguration},
	}
	for _, tt := range configTests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Command{InFile: &inFile, OutFile: &outFile}
			if err := tt.run(cmd); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestBoxFileCommandsUseAPIContext(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "missing.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	pb := &model.PageBoundaries{Crop: &model.Box{}}
	b := &model.Box{}
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{name: "list", run: func() error {
			_, err := ListBoxes(ListBoxesCommand(inFile, nil, nil, nil))
			return err
		}, want: "list boxes: open input"},
		{name: "add", run: func() error {
			_, err := AddBoxes(AddBoxesCommand(inFile, outFile, nil, pb, nil))
			return err
		}, want: "add boxes: open input"},
		{name: "remove", run: func() error {
			_, err := RemoveBoxes(RemoveBoxesCommand(inFile, outFile, nil, pb, nil))
			return err
		}, want: "remove boxes: open input"},
		{name: "crop", run: func() error {
			_, err := Crop(CropCommand(inFile, outFile, nil, b, nil))
			return err
		}, want: "crop: open input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestBoxCommandsUseAPIPageBoundaryValidation(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name string
		run  func(string) error
		want string
	}{
		{name: "empty add", run: func(outFile string) error {
			_, err := AddBoxes(AddBoxesCommand(inFile, outFile, nil, &model.PageBoundaries{}, nil))
			return err
		}, want: "add boxes: validate page boundaries: empty request"},
		{name: "empty remove", run: func(outFile string) error {
			_, err := RemoveBoxes(RemoveBoxesCommand(inFile, outFile, nil, &model.PageBoundaries{}, nil))
			return err
		}, want: "remove boxes: validate page boundaries: empty request"},
		{name: "remove MediaBox", run: func(outFile string) error {
			pb := &model.PageBoundaries{Media: &model.Box{}}
			_, err := RemoveBoxes(RemoveBoxesCommand(inFile, outFile, nil, pb, nil))
			return err
		}, want: "remove boxes: validate page boundaries: MediaBox removal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			err := tt.run(outFile)
			if !errors.Is(err, api.ErrInvalidPageBoundaries) {
				t.Fatalf("expected %v, got %v", api.ErrInvalidPageBoundaries, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %q", tt.want, err.Error())
			}
			if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("expected invalid request output cleanup, got %v", statErr)
			}
		})
	}
}

func TestGridCommandMode(t *testing.T) {
	nup, err := api.PDFGridConfig(2, 3, "", nil)
	if err != nil {
		t.Fatal(err)
	}

	cmd := GridCommand([]string{"in.pdf"}, "out.pdf", nil, nup, nil)
	if cmd.Mode != model.GRID {
		t.Fatalf("expected command mode %d, got %d", model.GRID, cmd.Mode)
	}
	if cmd.Conf.Cmd != model.GRID {
		t.Fatalf("expected configuration command %d, got %d", model.GRID, cmd.Conf.Cmd)
	}
}

func TestGridFileOutputDelegatesToAPIFileLayer(t *testing.T) {
	nup, err := api.ImageGridConfig(2, 3, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	imageFile := filepath.Join(t.TempDir(), "image.png")
	want := []byte("image input")
	if err := os.WriteFile(imageFile, want, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = Dispatch(GridCommand([]string{imageFile}, imageFile, nil, nup, nil))
	if !errors.Is(err, api.ErrGridImageOutputConflict) {
		t.Fatalf("expected %v, got %v", api.ErrGridImageOutputConflict, err)
	}
	if !strings.Contains(err.Error(), "grid image 1") {
		t.Fatalf("expected grid image context, got %q", err.Error())
	}
	got, err := os.ReadFile(imageFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("image input changed: got %q, want %q", got, want)
	}
}

func TestGridImageStdoutDoesNotOpenPDFInput(t *testing.T) {
	nup, err := api.ImageGridConfig(2, 3, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	missingImage := filepath.Join(t.TempDir(), "missing.png")
	defer log.SetDefaultCLILogger()

	_, err = Dispatch(GridCommand([]string{missingImage}, "-", nil, nup, nil))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "grid image 1") {
		t.Fatalf("expected indexed image context, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "grid: open input") {
		t.Fatalf("unexpected PDF input context: %q", err.Error())
	}
}

func TestGridPDFStdin(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	bb, err := os.ReadFile(inFile)
	if err != nil {
		t.Fatal(err)
	}
	useStdinBytes(t, bb)
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	nup, err := api.PDFGridConfig(2, 3, "", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = Dispatch(GridCommand([]string{"-"}, outFile, nil, nup, nil)); err != nil {
		t.Fatal(err)
	}
	if _, err = api.ReadContextFile(outFile); err != nil {
		t.Fatalf("expected valid grid output: %v", err)
	}
}

func TestGridImageStdout(t *testing.T) {
	out, err := os.CreateTemp(t.TempDir(), "stdout-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = out
	t.Cleanup(func() {
		os.Stdout = stdout
		_ = out.Close()
		log.SetDefaultCLILogger()
	})

	nup, err := api.ImageGridConfig(2, 3, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	inFile := filepath.Join("..", "samples", "images", "any.jpg")
	_, gridErr := Dispatch(GridCommand([]string{inFile}, "-", nil, nup, nil))
	os.Stdout = stdout
	if gridErr != nil {
		t.Fatal(gridErr)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := api.ReadContextFile(out.Name()); err != nil {
		t.Fatalf("expected valid stdout grid: %v", err)
	}
}

func TestNUpRejectsMissingInputs(t *testing.T) {
	for _, tt := range []struct {
		name    string
		nup     func() (*model.NUp, error)
		wantErr error
	}{
		{name: "PDF", nup: func() (*model.NUp, error) { return api.PDFNUpConfig(4, "", nil) }, wantErr: api.ErrMissingPDFInput},
		{name: "image", nup: func() (*model.NUp, error) { return api.ImageNUpConfig(4, "", nil) }, wantErr: api.ErrMissingImageInput},
	} {
		t.Run(tt.name, func(t *testing.T) {
			nup, err := tt.nup()
			if err != nil {
				t.Fatal(err)
			}
			_, err = NUp(NUpCommand(nil, "-", nil, nup, nil))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNUpFileOutputDelegatesToAPIFileLayer(t *testing.T) {
	nup, err := api.ImageNUpConfig(4, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	imageFiles := []string{
		filepath.Join(t.TempDir(), "first.png"),
		filepath.Join(t.TempDir(), "later.png"),
	}
	want := []byte("image input")
	for _, fileName := range imageFiles {
		if err := os.WriteFile(fileName, want, 0600); err != nil {
			t.Fatal(err)
		}
	}

	_, err = NUp(NUpCommand(imageFiles, imageFiles[1], nil, nup, nil))
	if !errors.Is(err, api.ErrNUpImageOutputConflict) {
		t.Fatalf("expected API file-layer sentinel %v, got %v", api.ErrNUpImageOutputConflict, err)
	}
	got, err := os.ReadFile(imageFiles[1])
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("image input changed: got %q, want %q", got, want)
	}
}

func TestNUpMissingPDFInput(t *testing.T) {
	nup, err := api.PDFNUpConfig(4, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	inFile := filepath.Join(t.TempDir(), "missing.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	_, err = NUp(NUpCommand([]string{inFile}, outFile, nil, nup, nil))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "n-up: open input "+inFile) {
		t.Fatalf("expected contextual input error, got %q", err)
	}
	requireMissingFile(t, outFile)
}

func TestNUpStdinReadFailure(t *testing.T) {
	nup, err := api.PDFNUpConfig(4, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	stdin := os.Stdin
	f, err := os.CreateTemp(t.TempDir(), "closed-stdin-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdin = f
	t.Cleanup(func() { os.Stdin = stdin })
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	_, err = NUp(NUpCommand([]string{"-"}, outFile, nil, nup, nil))
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected %v, got %v", os.ErrClosed, err)
	}
	if !strings.Contains(err.Error(), "n-up: read stdin") {
		t.Fatalf("expected stdin read context, got %q", err)
	}
	requireMissingFile(t, outFile)
}

func TestNUpStdinRewindFailure(t *testing.T) {
	nup, err := api.PDFNUpConfig(4, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	useStdin(t, "%PDF-1.7\n")
	wantErr := errors.New("rewind failed")
	rewind := rewindTemporaryInputFile
	rewindTemporaryInputFile = func(*os.File) error { return wantErr }
	t.Cleanup(func() { rewindTemporaryInputFile = rewind })
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	_, err = NUp(NUpCommand([]string{"-"}, outFile, nil, nup, nil))
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "n-up: rewind temporary input") {
		t.Fatalf("expected stdin rewind context, got %q", err)
	}
	requireMissingFile(t, outFile)
}

func TestNUpStreamOutputCreationFailure(t *testing.T) {
	nup, err := api.PDFNUpConfig(4, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	useStdin(t, "%PDF-1.7\n")
	outFile := filepath.Join(t.TempDir(), "missing", "out.pdf")

	_, err = NUp(NUpCommand([]string{"-"}, outFile, nil, nup, nil))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "n-up: create output") || !strings.Contains(err.Error(), outFile) {
		t.Fatalf("expected contextual output error, got %q", err)
	}
}

func TestNUpStdoutWriterFailure(t *testing.T) {
	nup, err := api.ImageNUpConfig(4, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := pipeReader.Close(); err != nil {
		t.Fatal(err)
	}
	_, pipeErr := pipeWriter.Write([]byte("probe"))
	if pipeErr == nil {
		t.Fatal("expected broken-pipe probe to fail")
	}
	var pathErr *os.PathError
	if errors.As(pipeErr, &pathErr) {
		pipeErr = pathErr.Err
	}

	stdout := os.Stdout
	os.Stdout = pipeWriter
	t.Cleanup(func() {
		os.Stdout = stdout
		_ = pipeWriter.Close()
		log.SetDefaultCLILogger()
	})
	inFile := filepath.Join("..", "samples", "images", "any.jpg")
	_, nUpErr := NUp(NUpCommand([]string{inFile}, "-", nil, nup, nil))
	os.Stdout = stdout

	if !errors.Is(nUpErr, pipeErr) {
		t.Fatalf("expected broken-pipe error %v, got %v", pipeErr, nUpErr)
	}
	if !strings.Contains(nUpErr.Error(), "n-up: write output") {
		t.Fatalf("expected write output context, got %q", nUpErr)
	}
}

func TestNUpStreamFailurePreservesExistingOutput(t *testing.T) {
	nup, err := api.PDFNUpConfig(4, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	useStdin(t, "not a PDF")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = NUp(NUpCommand([]string{"-"}, outFile, nil, nup, nil))
	if err == nil {
		t.Fatal("expected error")
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != string(want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

func TestNUpOutputCloseFailurePreservesPrimaryError(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	_, w, finalize, err := streamInOutForOperation("", outFile, "n-up")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.(*os.File).Close(); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("n-up operation failed")

	err = finalize(wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected primary error %v, got %v", wantErr, err)
	}
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected close error %v, got %v", os.ErrClosed, err)
	}
	if !strings.Contains(err.Error(), "n-up: close output") {
		t.Fatalf("expected close output context, got %q", err)
	}
	requireMissingFile(t, outFile)
}

func TestNUpPartialOutputCleanup(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	_, w, finalize, err := streamInOutForOperation("", outFile, "n-up")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("partial n-up output")); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("n-up operation failed")

	err = finalize(wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	requireMissingFile(t, outFile)
}

func TestBookletImageStdoutDoesNotOpenPDFInput(t *testing.T) {
	nup, err := api.ImageBookletConfig(2, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	missingImage := filepath.Join(t.TempDir(), "missing.png")
	cmd := BookletCommand([]string{missingImage}, "-", nil, nup, nil)
	defer log.SetDefaultCLILogger()

	_, err = Booklet(cmd)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "booklet image 1") {
		t.Fatalf("expected indexed image context, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "booklet: open input") {
		t.Fatalf("unexpected PDF input context: %q", err.Error())
	}
}

func TestBookletStreamRejectsMissingConfiguration(t *testing.T) {
	cmd := BookletCommand([]string{"-"}, "-", nil, nil, nil)
	_, err := Booklet(cmd)
	if !errors.Is(err, api.ErrMissingBookletConfiguration) {
		t.Fatalf("expected %v, got %v", api.ErrMissingBookletConfiguration, err)
	}
}

func TestBookletRejectsMissingCommandFields(t *testing.T) {
	_, err := Booklet(nil)
	if !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}

	cmd := &Command{}
	_, err = Booklet(cmd)
	if !errors.Is(err, api.ErrMissingPDFOutput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFOutput, err)
	}
}

func TestBookletFileOutputDelegatesToAPIFileLayer(t *testing.T) {
	nup, err := api.ImageBookletConfig(2, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	imageFiles := []string{
		filepath.Join(t.TempDir(), "first.png"),
		filepath.Join(t.TempDir(), "later.png"),
	}
	want := []byte("image input")
	for _, fileName := range imageFiles {
		if err := os.WriteFile(fileName, want, 0600); err != nil {
			t.Fatal(err)
		}
	}

	_, err = Booklet(BookletCommand(imageFiles, imageFiles[1], nil, nup, nil))
	if !errors.Is(err, api.ErrBookletImageOutputConflict) {
		t.Fatalf("expected API file-layer sentinel %v, got %v", api.ErrBookletImageOutputConflict, err)
	}
	got, err := os.ReadFile(imageFiles[1])
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("image input changed: got %q, want %q", got, want)
	}
}

func TestBookletMissingPDFInput(t *testing.T) {
	nup, err := api.PDFBookletConfig(2, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	inFile := filepath.Join(t.TempDir(), "missing.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	_, err = Booklet(BookletCommand([]string{inFile}, outFile, nil, nup, nil))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "booklet: open input "+inFile) {
		t.Fatalf("expected contextual input error, got %q", err)
	}
	requireMissingFile(t, outFile)
}

func TestBookletMissingImages(t *testing.T) {
	nup, err := api.ImageBookletConfig(2, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	imageFile := filepath.Join("..", "samples", "images", "any.jpg")

	for _, tt := range []struct {
		name    string
		inFiles []string
		index   string
	}{
		{name: "first", inFiles: []string{filepath.Join(t.TempDir(), "missing-first.jpg")}, index: "1"},
		{name: "later", inFiles: []string{imageFile, filepath.Join(t.TempDir(), "missing-later.jpg")}, index: "2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			_, err := Booklet(BookletCommand(tt.inFiles, outFile, nil, nup, nil))
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), "booklet image "+tt.index) {
				t.Fatalf("expected indexed image context, got %q", err)
			}
			if strings.Contains(err.Error(), "booklet: open input") {
				t.Fatalf("unexpected PDF input context: %q", err)
			}
			requireMissingFile(t, outFile)
		})
	}
}

func TestBookletStdinReadFailure(t *testing.T) {
	nup, err := api.PDFBookletConfig(2, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	stdin := os.Stdin
	f, err := os.CreateTemp(t.TempDir(), "closed-stdin-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdin = f
	t.Cleanup(func() { os.Stdin = stdin })
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	_, err = Booklet(BookletCommand([]string{"-"}, outFile, nil, nup, nil))
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected %v, got %v", os.ErrClosed, err)
	}
	if !strings.Contains(err.Error(), "booklet: read stdin") {
		t.Fatalf("expected stdin read context, got %q", err)
	}
	requireMissingFile(t, outFile)
}

func TestBookletStdinRewindFailure(t *testing.T) {
	nup, err := api.PDFBookletConfig(2, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	useStdin(t, "%PDF-1.7\n")
	wantErr := errors.New("rewind failed")
	rewind := rewindTemporaryInputFile
	rewindTemporaryInputFile = func(*os.File) error { return wantErr }
	t.Cleanup(func() { rewindTemporaryInputFile = rewind })
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	_, err = Booklet(BookletCommand([]string{"-"}, outFile, nil, nup, nil))
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "booklet: rewind temporary input") {
		t.Fatalf("expected stdin rewind context, got %q", err)
	}
	requireMissingFile(t, outFile)
}

func TestBookletOutputCreationFailure(t *testing.T) {
	nup, err := api.ImageBookletConfig(2, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	inFile := filepath.Join("..", "samples", "images", "any.jpg")
	outFile := filepath.Join(t.TempDir(), "missing", "out.pdf")

	_, err = Booklet(BookletCommand([]string{inFile}, outFile, nil, nup, nil))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "booklet: create output") || !strings.Contains(err.Error(), outFile) {
		t.Fatalf("expected contextual output error, got %q", err)
	}
}

func TestBookletStdoutWriterFailure(t *testing.T) {
	nup, err := api.ImageBookletConfig(2, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := pipeReader.Close(); err != nil {
		t.Fatal(err)
	}
	_, pipeErr := pipeWriter.Write([]byte("probe"))
	if pipeErr == nil {
		t.Fatal("expected broken-pipe probe to fail")
	}
	var pathErr *os.PathError
	if errors.As(pipeErr, &pathErr) {
		pipeErr = pathErr.Err
	}

	stdout := os.Stdout
	os.Stdout = pipeWriter
	t.Cleanup(func() {
		os.Stdout = stdout
		_ = pipeWriter.Close()
		log.SetDefaultCLILogger()
	})
	inFile := filepath.Join("..", "samples", "images", "any.jpg")
	_, bookletErr := Booklet(BookletCommand([]string{inFile}, "-", nil, nup, nil))
	os.Stdout = stdout

	if !errors.Is(bookletErr, pipeErr) {
		t.Fatalf("expected broken-pipe error %v, got %v", pipeErr, bookletErr)
	}
	if !strings.Contains(bookletErr.Error(), "booklet: write output") {
		t.Fatalf("expected write output context, got %q", bookletErr)
	}
}

func TestBookletOutputCloseFailurePreservesPrimaryError(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	_, w, finalize, err := streamInOutForOperation("", outFile, "booklet")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.(*os.File).Close(); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("booklet operation failed")

	err = finalize(wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected primary error %v, got %v", wantErr, err)
	}
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected close error %v, got %v", os.ErrClosed, err)
	}
	if !strings.Contains(err.Error(), "booklet: close output") {
		t.Fatalf("expected close output context, got %q", err)
	}
	requireMissingFile(t, outFile)
}

func TestBookletPartialOutputCleanup(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	_, w, finalize, err := streamInOutForOperation("", outFile, "booklet")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("partial booklet output")); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("booklet operation failed")

	err = finalize(wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	requireMissingFile(t, outFile)
}

func TestBookletFailurePreservesExistingOutput(t *testing.T) {
	nup, err := api.ImageBookletConfig(2, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0600); err != nil {
		t.Fatal(err)
	}
	inFiles := []string{
		filepath.Join("..", "samples", "images", "any.jpg"),
		filepath.Join(t.TempDir(), "missing.jpg"),
	}

	_, err = Booklet(BookletCommand(inFiles, outFile, nil, nup, nil))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}
