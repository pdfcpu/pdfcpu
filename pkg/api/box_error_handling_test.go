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

package api

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func boxTestPageBoundaries() *model.PageBoundaries {
	return &model.PageBoundaries{Crop: &model.Box{}}
}

func TestBoxParserErrorContext(t *testing.T) {
	tests := []struct {
		name  string
		parse func() error
		want  string
	}{
		{
			name: "box list",
			parse: func() error {
				_, err := PageBoundariesFromBoxList("unknown")
				return err
			},
			want: "parse box list",
		},
		{
			name: "page boundaries",
			parse: func() error {
				_, err := PageBoundaries("unknown", types.POINTS)
				return err
			},
			want: "parse page boundaries",
		},
		{
			name: "box",
			parse: func() error {
				_, err := Box("unknown", types.POINTS)
				return err
			},
			want: "parse box",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.parse()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %v", tt.want, err)
			}
		})
	}
}

func TestPrepareBoxListing(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	ctx, pages, err := prepareBoxListing(openAPITestPDF(t, inFile), []string{"1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Conf.Cmd != model.LISTBOXES {
		t.Fatalf("expected command %d, got %d", model.LISTBOXES, ctx.Conf.Cmd)
	}
	if ctx.Optimized {
		t.Fatal("box listing must not optimize the context")
	}
	if !pages[1] {
		t.Fatalf("expected selected page 1, got %v", pages)
	}
}

func TestBoxAPIMissingConfiguration(t *testing.T) {
	rs := bytes.NewReader(nil)
	w := &bytes.Buffer{}
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "add boxes", err: AddBoxes(rs, w, nil, nil, nil), want: ErrMissingPageBoundaries},
		{name: "remove boxes", err: RemoveBoxes(rs, w, nil, nil, nil), want: ErrMissingPageBoundaries},
		{name: "crop", err: Crop(rs, w, nil, nil, nil), want: ErrMissingBoxConfiguration},
		{name: "add boxes file", err: AddBoxesFile("missing.pdf", "", nil, nil, nil), want: ErrMissingPageBoundaries},
		{name: "remove boxes file", err: RemoveBoxesFile("missing.pdf", "", nil, nil, nil), want: ErrMissingPageBoundaries},
		{name: "crop file", err: CropFile("missing.pdf", "", nil, nil, nil), want: ErrMissingBoxConfiguration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, tt.err)
			}
		})
	}
}

func TestBoxAPIRejectsInvalidPageBoundaryRequests(t *testing.T) {
	tests := []struct {
		name string
		run  func(io.ReadSeeker, io.Writer) error
		want string
	}{
		{name: "empty add", run: func(rs io.ReadSeeker, w io.Writer) error {
			return AddBoxes(rs, w, nil, &model.PageBoundaries{}, nil)
		}, want: "add boxes: validate page boundaries: empty request"},
		{name: "empty remove", run: func(rs io.ReadSeeker, w io.Writer) error {
			return RemoveBoxes(rs, w, nil, &model.PageBoundaries{}, nil)
		}, want: "remove boxes: validate page boundaries: empty request"},
		{name: "remove MediaBox", run: func(rs io.ReadSeeker, w io.Writer) error {
			return RemoveBoxes(rs, w, nil, &model.PageBoundaries{Media: &model.Box{}}, nil)
		}, want: "remove boxes: validate page boundaries: MediaBox removal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(bytes.NewReader(nil), io.Discard)
			if !errors.Is(err, ErrInvalidPageBoundaries) {
				t.Fatalf("expected %v, got %v", ErrInvalidPageBoundaries, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %q", tt.want, err.Error())
			}
			if errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("request validation must precede PDF reading: %v", err)
			}
		})
	}
}

func TestBoxFileAPIRejectsInvalidPageBoundaryRequests(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name string
		run  func(string) error
		want string
	}{
		{name: "empty add", run: func(outFile string) error {
			return AddBoxesFile(inFile, outFile, nil, &model.PageBoundaries{}, nil)
		}, want: "add boxes: validate page boundaries: empty request"},
		{name: "empty remove", run: func(outFile string) error {
			return RemoveBoxesFile(inFile, outFile, nil, &model.PageBoundaries{}, nil)
		}, want: "remove boxes: validate page boundaries: empty request"},
		{name: "remove MediaBox", run: func(outFile string) error {
			return RemoveBoxesFile(inFile, outFile, nil, &model.PageBoundaries{Media: &model.Box{}}, nil)
		}, want: "remove boxes: validate page boundaries: MediaBox removal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			err := tt.run(outFile)
			if !errors.Is(err, ErrInvalidPageBoundaries) {
				t.Fatalf("expected %v, got %v", ErrInvalidPageBoundaries, err)
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

func TestBoxFileAPIValidatesRequestsBeforeFileIO(t *testing.T) {
	tests := []struct {
		name string
		run  func(string, string) error
		want string
	}{
		{name: "empty add", run: func(inFile, outFile string) error {
			return AddBoxesFile(inFile, outFile, nil, &model.PageBoundaries{}, nil)
		}, want: "add boxes: validate page boundaries: empty request"},
		{name: "empty remove", run: func(inFile, outFile string) error {
			return RemoveBoxesFile(inFile, outFile, nil, &model.PageBoundaries{}, nil)
		}, want: "remove boxes: validate page boundaries: empty request"},
		{name: "remove MediaBox", run: func(inFile, outFile string) error {
			return RemoveBoxesFile(inFile, outFile, nil, &model.PageBoundaries{Media: &model.Box{}}, nil)
		}, want: "remove boxes: validate page boundaries: MediaBox removal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			inFile := filepath.Join(dir, "missing.pdf")
			outFile := filepath.Join(dir, "out.pdf")
			err := tt.run(inFile, outFile)
			if !errors.Is(err, ErrInvalidPageBoundaries) {
				t.Fatalf("expected %v, got %v", ErrInvalidPageBoundaries, err)
			}
			if errors.Is(err, os.ErrNotExist) {
				t.Fatalf("file I/O occurred before request validation: %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %q", tt.want, err.Error())
			}
			if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("expected no output, got %v", statErr)
			}
		})
	}
}

func TestBoxAPIMissingIO(t *testing.T) {
	pb := boxTestPageBoundaries()
	b := &model.Box{}
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "boxes reader", err: func() error {
			_, err := Boxes(nil, nil, nil)
			return err
		}(), want: ErrMissingPDFReadSeeker},
		{name: "list reader", err: func() error {
			_, err := ListBoxes(nil, nil, nil, nil)
			return err
		}(), want: ErrMissingPDFReadSeeker},
		{name: "add reader", err: AddBoxes(nil, io.Discard, nil, pb, nil), want: ErrMissingPDFReadSeeker},
		{name: "add writer", err: AddBoxes(bytes.NewReader(nil), nil, nil, pb, nil), want: ErrMissingPDFWriter},
		{name: "remove reader", err: RemoveBoxes(nil, io.Discard, nil, pb, nil), want: ErrMissingPDFReadSeeker},
		{name: "remove writer", err: RemoveBoxes(bytes.NewReader(nil), nil, nil, pb, nil), want: ErrMissingPDFWriter},
		{name: "crop reader", err: Crop(nil, io.Discard, nil, b, nil), want: ErrMissingPDFReadSeeker},
		{name: "crop writer", err: Crop(bytes.NewReader(nil), nil, nil, b, nil), want: ErrMissingPDFWriter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, tt.err)
			}
		})
	}
}

func TestBoxFileAPIMissingInput(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "list boxes", err: func() error {
			_, err := ListBoxesFile("", nil, nil, nil)
			return err
		}()},
		{name: "add boxes", err: AddBoxesFile("", "", nil, boxTestPageBoundaries(), nil)},
		{name: "remove boxes", err: RemoveBoxesFile("", "", nil, boxTestPageBoundaries(), nil)},
		{name: "crop", err: CropFile("", "", nil, nil, nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, ErrMissingPDFInput) {
				t.Fatalf("expected %v, got %v", ErrMissingPDFInput, tt.err)
			}
		})
	}
}

func TestBoxFileCreateOutputErrorContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "missing", "out.pdf")
	pb := boxTestPageBoundaries()
	b := &model.Box{}
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "add", err: AddBoxesFile(inFile, outFile, nil, pb, nil), want: "add boxes: create output"},
		{name: "remove", err: RemoveBoxesFile(inFile, outFile, nil, pb, nil), want: "remove boxes: create output"},
		{name: "crop", err: CropFile(inFile, outFile, nil, b, nil), want: "crop: create output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, tt.err)
			}
			if !strings.Contains(tt.err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %q", tt.want, tt.err.Error())
			}
		})
	}
}

func TestBoxAPIAvoidsDuplicateOperationContext(t *testing.T) {
	tests := []struct {
		name string
		err  error
		op   string
	}{
		{name: "list", err: func() error {
			_, err := ListBoxes(bytes.NewReader(nil), nil, nil, nil)
			return err
		}(), op: "list boxes:"},
		{name: "add", err: AddBoxes(bytes.NewReader(nil), io.Discard, nil, boxTestPageBoundaries(), nil), op: "add boxes:"},
		{name: "remove", err: RemoveBoxes(bytes.NewReader(nil), io.Discard, nil, boxTestPageBoundaries(), nil), op: "remove boxes:"},
		{name: "crop", err: Crop(bytes.NewReader(nil), io.Discard, nil, &model.Box{}, nil), op: "crop:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("expected error")
			}
			if got := strings.Count(tt.err.Error(), tt.op); got != 1 {
				t.Fatalf("expected one %q prefix, got %d in %q", tt.op, got, tt.err.Error())
			}
		})
	}
}

func TestBoxAPIReadErrorContext(t *testing.T) {
	pb := boxTestPageBoundaries()
	b := &model.Box{}
	tests := []struct {
		name string
		run  func(io.ReadSeeker) error
		want string
	}{
		{name: "list", run: func(rs io.ReadSeeker) error {
			_, err := Boxes(rs, nil, nil)
			return err
		}, want: "list boxes: prepare PDF context: read context"},
		{name: "formatted list", run: func(rs io.ReadSeeker) error {
			_, err := ListBoxes(rs, nil, nil, nil)
			return err
		}, want: "list boxes: prepare PDF context: read context"},
		{name: "add", run: func(rs io.ReadSeeker) error {
			return AddBoxes(rs, io.Discard, nil, pb, nil)
		}, want: "add boxes: prepare PDF context: read context"},
		{name: "remove", run: func(rs io.ReadSeeker) error {
			return RemoveBoxes(rs, io.Discard, nil, pb, nil)
		}, want: "remove boxes: prepare PDF context: read context"},
		{name: "crop", run: func(rs io.ReadSeeker) error {
			return Crop(rs, io.Discard, nil, b, nil)
		}, want: "crop: prepare PDF context: read context"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(bytes.NewReader(nil))
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestBoxAPIPageSelectionErrorContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	pb := boxTestPageBoundaries()
	b := &model.Box{}
	tests := []struct {
		name string
		run  func(io.ReadSeeker) error
		want string
	}{
		{name: "list", run: func(rs io.ReadSeeker) error {
			_, err := Boxes(rs, []string{"bogus"}, nil)
			return err
		}, want: "list boxes: parse page selection"},
		{name: "formatted list", run: func(rs io.ReadSeeker) error {
			_, err := ListBoxes(rs, []string{"bogus"}, nil, nil)
			return err
		}, want: "list boxes: parse page selection"},
		{name: "add", run: func(rs io.ReadSeeker) error {
			return AddBoxes(rs, io.Discard, []string{"bogus"}, pb, nil)
		}, want: "add boxes: parse page selection"},
		{name: "remove", run: func(rs io.ReadSeeker) error {
			return RemoveBoxes(rs, io.Discard, []string{"bogus"}, pb, nil)
		}, want: "remove boxes: parse page selection"},
		{name: "crop", run: func(rs io.ReadSeeker) error {
			return Crop(rs, io.Discard, []string{"bogus"}, b, nil)
		}, want: "crop: parse page selection"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(openAPITestPDF(t, inFile))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %v", tt.want, err)
			}
		})
	}
}

func TestBoxAPIWriteErrorContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	wantErr := errors.New("write failed")
	pb := boxTestPageBoundaries()
	b, err := Box("10", types.POINTS)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		run  func(io.ReadSeeker) error
		want string
	}{
		{name: "add", run: func(rs io.ReadSeeker) error {
			return AddBoxes(rs, failingWriter{err: wantErr}, nil, pb, nil)
		}, want: "add boxes: write output"},
		{name: "remove", run: func(rs io.ReadSeeker) error {
			return RemoveBoxes(rs, failingWriter{err: wantErr}, nil, pb, nil)
		}, want: "remove boxes: write output"},
		{name: "crop", run: func(rs io.ReadSeeker) error {
			return Crop(rs, failingWriter{err: wantErr}, nil, b, nil)
		}, want: "crop: write output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(openAPITestPDF(t, inFile))
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestBoxFileOpenErrorContext(t *testing.T) {
	pb := boxTestPageBoundaries()
	b := &model.Box{}
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{name: "list", run: func() error {
			_, err := ListBoxesFile("missing.pdf", nil, nil, nil)
			return err
		}, want: "list boxes: open input missing.pdf"},
		{name: "add", run: func() error {
			return AddBoxesFile("missing.pdf", "", nil, pb, nil)
		}, want: "add boxes: open input missing.pdf"},
		{name: "remove", run: func() error {
			return RemoveBoxesFile("missing.pdf", "", nil, pb, nil)
		}, want: "remove boxes: open input missing.pdf"},
		{name: "crop", run: func() error {
			return CropFile("missing.pdf", "", nil, b, nil)
		}, want: "crop: open input missing.pdf"},
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

func TestBoxFilePreservesExistingOutputOnFailure(t *testing.T) {
	pb := boxTestPageBoundaries()
	b := &model.Box{}
	tests := []struct {
		name string
		run  func(string, string) error
	}{
		{name: "add", run: func(inFile, outFile string) error {
			return AddBoxesFile(inFile, outFile, nil, pb, nil)
		}},
		{name: "remove", run: func(inFile, outFile string) error {
			return RemoveBoxesFile(inFile, outFile, nil, pb, nil)
		}},
		{name: "crop", run: func(inFile, outFile string) error {
			return CropFile(inFile, outFile, nil, b, nil)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			inFile := filepath.Join(dir, "invalid.pdf")
			outFile := filepath.Join(dir, "out.pdf")
			if err := os.WriteFile(inFile, []byte("invalid"), 0600); err != nil {
				t.Fatal(err)
			}
			want := []byte("existing output")
			if err := os.WriteFile(outFile, want, 0600); err != nil {
				t.Fatal(err)
			}

			if err := tt.run(inFile, outFile); err == nil {
				t.Fatal("expected error")
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
