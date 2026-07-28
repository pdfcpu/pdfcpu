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

func stampTestWatermark(t *testing.T, update bool) *model.Watermark {
	t.Helper()

	wm, err := TextWatermark("draft", "", false, update, types.POINTS)
	if err != nil {
		t.Fatal(err)
	}
	return wm
}

func stampTestInputFile() string {
	return filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
}

func TestStampMapErrorsPreserveSentinelsWithoutPanic(t *testing.T) {
	tests := []struct {
		name        string
		fn          func() error
		wantErr     error
		wantContext string
	}{
		{
			name: "missing watermark map",
			fn: func() error {
				return AddWatermarksMap(bytes.NewReader(nil), io.Discard, nil, nil)
			},
			wantErr: ErrMissingWatermarks,
		},
		{
			name: "nil map watermark",
			fn: func() error {
				return AddWatermarksMap(bytes.NewReader(nil), io.Discard, map[int]*model.Watermark{1: nil}, nil)
			},
			wantErr:     ErrMissingWatermarkConfiguration,
			wantContext: "page 1",
		},
		{
			name: "empty watermark slice",
			fn: func() error {
				return AddWatermarksSliceMap(bytes.NewReader(nil), io.Discard, map[int][]*model.Watermark{2: nil}, nil)
			},
			wantErr:     ErrMissingWatermarks,
			wantContext: "page 2",
		},
		{
			name: "nil slice watermark",
			fn: func() error {
				return AddWatermarksSliceMap(bytes.NewReader(nil), io.Discard, map[int][]*model.Watermark{3: {nil}}, nil)
			},
			wantErr:     ErrMissingWatermarkConfiguration,
			wantContext: "page 3, watermark 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if tt.wantContext != "" && !strings.Contains(err.Error(), tt.wantContext) {
				t.Fatalf("expected %q in %q", tt.wantContext, err.Error())
			}
		})
	}
}

func TestValidateWatermarkMapsUsesSortedPages(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "watermark map",
			fn: func() error {
				return validateWatermarkMap(map[int]*model.Watermark{2: nil, 1: nil})
			},
		},
		{
			name: "watermark slice map",
			fn: func() error {
				return validateWatermarkSliceMap(map[int][]*model.Watermark{2: nil, 1: nil})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), "page 1") {
				t.Fatalf("expected page 1 error, got %v", err)
			}
		})
	}
}

func TestStampReaderEntryPointsPreserveMissingIO(t *testing.T) {
	wm := stampTestWatermark(t, false)
	tests := []struct {
		name    string
		fn      func() error
		wantErr error
	}{
		{
			name: "map reader",
			fn: func() error {
				return AddWatermarksMap(nil, io.Discard, map[int]*model.Watermark{1: wm}, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "slice map writer",
			fn: func() error {
				return AddWatermarksSliceMap(bytes.NewReader(nil), nil, map[int][]*model.Watermark{1: {wm}}, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "add reader",
			fn: func() error {
				return AddWatermarks(nil, io.Discard, nil, wm, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "remove writer",
			fn: func() error {
				return RemoveWatermarks(bytes.NewReader(nil), nil, nil, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "detect reader",
			fn: func() error {
				_, err := HasWatermarks(nil, nil)
				return err
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestStampReadErrorsIncludePhaseContext(t *testing.T) {
	wm := stampTestWatermark(t, false)
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "watermark map",
			fn: func() error {
				return AddWatermarksMap(bytes.NewReader(nil), io.Discard, map[int]*model.Watermark{1: wm}, nil)
			},
			want: "add watermarks: prepare PDF context",
		},
		{
			name: "watermark slice map",
			fn: func() error {
				return AddWatermarksSliceMap(bytes.NewReader(nil), io.Discard, map[int][]*model.Watermark{1: {wm}}, nil)
			},
			want: "add watermarks: prepare PDF context",
		},
		{
			name: "add watermarks",
			fn: func() error {
				return AddWatermarks(bytes.NewReader(nil), io.Discard, nil, wm, nil)
			},
			want: "add watermarks: prepare PDF context",
		},
		{
			name: "remove watermarks",
			fn: func() error {
				return RemoveWatermarks(bytes.NewReader(nil), io.Discard, nil, nil)
			},
			want: "remove watermarks: prepare PDF context",
		},
		{
			name: "detect watermarks",
			fn: func() error {
				_, err := HasWatermarks(bytes.NewReader(nil), nil)
				return err
			},
			want: "detect watermarks: prepare PDF context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
			if strings.Contains(err.Error(), "prepare PDF context: prepare PDF context") {
				t.Fatalf("unexpected duplicate prepare context in %q", err.Error())
			}
		})
	}
}

func TestUpdateWatermarkReadErrorUsesUpdateOperation(t *testing.T) {
	err := AddWatermarks(bytes.NewReader(nil), io.Discard, nil, stampTestWatermark(t, true), nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "update watermarks: prepare PDF context") {
		t.Fatalf("expected update prepare context in %q", err.Error())
	}
	if strings.Contains(err.Error(), "add watermarks") {
		t.Fatalf("unexpected add operation context in %q", err.Error())
	}
}

func TestWatermarkContextRejectsMissingXRefTable(t *testing.T) {
	err := WatermarkContext(&model.Context{}, nil, stampTestWatermark(t, false))
	if !errors.Is(err, ErrMissingXRefTable) {
		t.Fatalf("expected %v, got %v", ErrMissingXRefTable, err)
	}
}

func TestWatermarkAPIErrorsAliasPdfcpuSentinels(t *testing.T) {
	if ErrMissingImageReader != pdfcpu.ErrMissingImageReader {
		t.Fatal("ErrMissingImageReader is not an alias of pdfcpu.ErrMissingImageReader")
	}
	if ErrMissingImageReader != model.ErrMissingImageReader {
		t.Fatal("ErrMissingImageReader is not an alias of model.ErrMissingImageReader")
	}
	if ErrMissingWatermarkConfiguration != pdfcpu.ErrMissingWatermarkConfiguration {
		t.Fatal("ErrMissingWatermarkConfiguration is not an alias of pdfcpu.ErrMissingWatermarkConfiguration")
	}
	if ErrMissingWatermarks != pdfcpu.ErrMissingWatermarks {
		t.Fatal("ErrMissingWatermarks is not an alias of pdfcpu.ErrMissingWatermarks")
	}
}

func TestStampPageSelectionErrorsIncludePhaseContext(t *testing.T) {
	wm := stampTestWatermark(t, false)
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "add watermarks",
			fn: func() error {
				return AddWatermarks(openAPITestPDF(t, stampTestInputFile()), io.Discard, []string{"foo"}, wm, nil)
			},
			want: "add watermarks: parse page selection",
		},
		{
			name: "remove watermarks",
			fn: func() error {
				return RemoveWatermarks(openAPITestPDF(t, stampTestInputFile()), io.Discard, []string{"foo"}, nil)
			},
			want: "remove watermarks: parse page selection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
			if !strings.Contains(err.Error(), "invalid syntax") {
				t.Fatalf("expected page selection detail in %q", err.Error())
			}
		})
	}
}

func TestAddWatermarksWriteErrorIncludesPhaseContext(t *testing.T) {
	wantErr := errors.New("stamp write failed")
	err := AddWatermarks(
		openAPITestPDF(t, stampTestInputFile()),
		failingWriter{err: wantErr},
		[]string{"1"},
		stampTestWatermark(t, false),
		nil,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "add watermarks: write output") {
		t.Fatalf("expected write output context in %q", err.Error())
	}
}

func TestStampMapWriteErrorsIncludePhaseContext(t *testing.T) {
	wantErr := errors.New("stamp map write failed")
	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "watermark map",
			fn: func() error {
				return AddWatermarksMap(
					openAPITestPDF(t, stampTestInputFile()),
					failingWriter{err: wantErr},
					map[int]*model.Watermark{1: stampTestWatermark(t, false)},
					nil,
				)
			},
		},
		{
			name: "watermark slice map",
			fn: func() error {
				return AddWatermarksSliceMap(
					openAPITestPDF(t, stampTestInputFile()),
					failingWriter{err: wantErr},
					map[int][]*model.Watermark{1: {stampTestWatermark(t, false)}},
					nil,
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), "add watermarks: write output") {
				t.Fatalf("expected write phase in %q", err.Error())
			}
		})
	}
}

func TestStampApplyErrorsIncludeOperationContext(t *testing.T) {
	err := RemoveWatermarks(openAPITestPDF(t, stampTestInputFile()), io.Discard, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "remove watermarks: apply") {
		t.Fatalf("expected apply context in %q", err.Error())
	}
}

func TestUpdateWatermarksWriteErrorUsesUpdateOperation(t *testing.T) {
	wantErr := errors.New("update watermark write failed")
	err := AddWatermarks(
		openAPITestPDF(t, stampTestInputFile()),
		failingWriter{err: wantErr},
		[]string{"1"},
		stampTestWatermark(t, true),
		nil,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "update watermarks: write output") {
		t.Fatalf("expected update write context in %q", err.Error())
	}
}

func TestStampFileEntryPointsPreserveMissingInput(t *testing.T) {
	wm := stampTestWatermark(t, false)
	reader := bytes.NewReader(nil)
	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "watermark map", fn: func() error { return AddWatermarksMapFile("", "", nil, nil) }},
		{name: "watermark slice map", fn: func() error { return AddWatermarksSliceMapFile("", "", nil, nil) }},
		{name: "add watermarks", fn: func() error { return AddWatermarksFile("", "", nil, wm, nil) }},
		{name: "remove watermarks", fn: func() error { return RemoveWatermarksFile("", "", nil, nil) }},
		{name: "detect watermarks", fn: func() error { _, err := HasWatermarksFile("", nil); return err }},
		{name: "add text", fn: func() error { return AddTextWatermarksFile("", "", nil, false, "draft", "", nil) }},
		{name: "add image", fn: func() error { return AddImageWatermarksFile("", "", nil, false, "image.png", "", nil) }},
		{name: "add image reader", fn: func() error { return AddImageWatermarksForReaderFile("", "", nil, false, reader, "", nil) }},
		{name: "add PDF", fn: func() error { return AddPDFWatermarksFile("", "", nil, false, "stamp.pdf", "", nil) }},
		{name: "add PDF reader", fn: func() error { return AddPDFWatermarksForReadSeekerFile("", "", nil, false, reader, 1, "", nil) }},
		{name: "update text", fn: func() error { return UpdateTextWatermarksFile("", "", nil, false, "draft", "", nil) }},
		{name: "update image", fn: func() error { return UpdateImageWatermarksFile("", "", nil, false, "image.png", "", nil) }},
		{name: "update PDF", fn: func() error { return UpdatePDFWatermarksFile("", "", nil, false, "stamp.pdf", "", nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, ErrMissingPDFInput) {
				t.Fatalf("expected %v, got %v", ErrMissingPDFInput, err)
			}
		})
	}
}

func TestStampFileOpenErrorsIncludeOperationContext(t *testing.T) {
	wm := stampTestWatermark(t, false)
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{name: "watermark map", fn: func() error { return AddWatermarksMapFile("missing.pdf", "", nil, nil) }, want: "add watermarks: open input missing.pdf"},
		{name: "watermark slice map", fn: func() error { return AddWatermarksSliceMapFile("missing.pdf", "", nil, nil) }, want: "add watermarks: open input missing.pdf"},
		{name: "add watermarks", fn: func() error { return AddWatermarksFile("missing.pdf", "", nil, wm, nil) }, want: "add watermarks: open input missing.pdf"},
		{name: "remove watermarks", fn: func() error { return RemoveWatermarksFile("missing.pdf", "", nil, nil) }, want: "remove watermarks: open input missing.pdf"},
		{name: "detect watermarks", fn: func() error { _, err := HasWatermarksFile("missing.pdf", nil); return err }, want: "detect watermarks: open input missing.pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
		})
	}
}

func TestCloseStampFilesErrorsIncludeOperationContext(t *testing.T) {
	tests := []struct {
		name    string
		fn      func(t *testing.T) error
		wantErr error
		want    string
	}{
		{
			name: "close output",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				if err := f2.Close(); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "add watermarks").commit()
			},
			wantErr: os.ErrClosed,
			want:    "add watermarks: close output",
		},
		{
			name: "close input",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				if err := f1.Close(); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "remove watermarks").commit()
			},
			wantErr: os.ErrClosed,
			want:    "remove watermarks: close input",
		},
		{
			name: "replace input",
			fn: func(t *testing.T) error {
				return commitStagedOutputWithReplaceError(t, "update watermarks")
			},
			wantErr: os.ErrNotExist,
			want:    "update watermarks: replace input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(t)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
		})
	}
}

func TestWatermarkConstructorErrorsIncludePhaseContext(t *testing.T) {
	_, err := TextWatermark("", "", false, false, types.POINTS)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create watermark: validate configuration") {
		t.Fatalf("expected validation context in %q", err.Error())
	}
	if !strings.Contains(err.Error(), "watermark text must not be empty") {
		t.Fatalf("expected validation detail in %q", err.Error())
	}

	if _, err = ImageWatermarkForReader(nil, "", false, false, types.POINTS); !errors.Is(err, ErrMissingImageReader) {
		t.Fatalf("expected %v, got %v", ErrMissingImageReader, err)
	}
}

func TestUpdateWatermarkFileErrorsUseUpdateOperation(t *testing.T) {
	err := AddWatermarksFile("missing.pdf", "", nil, stampTestWatermark(t, true), nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "update watermarks: open input missing.pdf") {
		t.Fatalf("expected update operation context in %q", err.Error())
	}
	if strings.Contains(err.Error(), "add watermarks") {
		t.Fatalf("unexpected add operation context in %q", err.Error())
	}
}
