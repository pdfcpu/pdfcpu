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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type nopReadWriteSeeker struct {
	*bytes.Reader
}

// Write implements io.Writer.
func (n nopReadWriteSeeker) Write(p []byte) (int, error) {
	return len(p), nil
}

type failingReadWriteSeeker struct {
	*bytes.Reader
	err error
}

// Write implements io.Writer.
func (r failingReadWriteSeeker) Write(_ []byte) (int, error) {
	return 0, r.err
}

type failingWriter struct {
	err error
}

// Write implements io.Writer.
func (w failingWriter) Write(_ []byte) (int, error) {
	return 0, w.err
}

func appendValidationTestObject(buf *bytes.Buffer, objNr int, body string) int {
	offset := buf.Len()
	fmt.Fprintf(buf, "%d 0 obj\n%s\nendobj\n", objNr, body)
	return offset
}

func invalidValidationTestPDF() []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n%\xFF\xFF\xFF\xFF\n")

	offset := appendValidationTestObject(&buf, 1, "<< /Type /Catalog >>")
	xrefOffset := buf.Len()
	buf.WriteString("xref\n0 2\n")
	buf.WriteString("0000000000 65535 f \n")
	fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	fmt.Fprintf(&buf, "trailer\n<< /Size 2 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset)

	return buf.Bytes()
}

// TestAPIArgumentErrors verifies the corresponding behavior.
func TestAPIArgumentErrors(t *testing.T) {
	tests := []struct {
		name        string
		fn          func() error
		wantErr     error
		wantContext string
	}{
		{
			name: "optimize missing reader",
			fn: func() error {
				return Optimize(nil, io.Discard, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "optimize missing writer",
			fn: func() error {
				return Optimize(bytes.NewReader(nil), nil, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "optimize file missing input",
			fn: func() error {
				return OptimizeFile("", filepath.Join(t.TempDir(), "out.pdf"), nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "collect missing reader",
			fn: func() error {
				return Collect(nil, io.Discard, nil, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "collect missing writer",
			fn: func() error {
				return Collect(bytes.NewReader(nil), nil, nil, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "collect file missing input",
			fn: func() error {
				return CollectFile("", filepath.Join(t.TempDir(), "out.pdf"), nil, nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "trim missing reader",
			fn: func() error {
				return Trim(nil, io.Discard, nil, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "trim missing writer",
			fn: func() error {
				return Trim(bytes.NewReader(nil), nil, nil, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "trim file missing input",
			fn: func() error {
				return TrimFile("", filepath.Join(t.TempDir(), "out.pdf"), nil, nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "add annotations missing annotation",
			fn: func() error {
				return AddAnnotations(bytes.NewReader(nil), io.Discard, nil, nil, nil)
			},
			wantErr: ErrMissingAnnotation,
		},
		{
			name: "add annotations increment missing annotation",
			fn: func() error {
				return AddAnnotationsAsIncrement(nopReadWriteSeeker{bytes.NewReader(nil)}, nil, nil, nil)
			},
			wantErr: ErrMissingAnnotation,
		},
		{
			name: "add annotations file missing input",
			fn: func() error {
				return AddAnnotationsFile("", filepath.Join(t.TempDir(), "out.pdf"), nil, nil, nil, false)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "add annotations map missing annotation",
			fn: func() error {
				m := map[int][]model.AnnotationRenderer{1: {nil}}
				return AddAnnotationsMap(bytes.NewReader(nil), io.Discard, m, nil)
			},
			wantErr:     ErrMissingAnnotation,
			wantContext: "page 1 annotation 1",
		},
		{
			name: "add annotations map increment missing annotation",
			fn: func() error {
				m := map[int][]model.AnnotationRenderer{1: {nil}}
				return AddAnnotationsMapAsIncrement(nopReadWriteSeeker{bytes.NewReader(nil)}, m, nil)
			},
			wantErr:     ErrMissingAnnotation,
			wantContext: "page 1 annotation 1",
		},
		{
			name: "add annotations map file missing input",
			fn: func() error {
				return AddAnnotationsMapFile("", filepath.Join(t.TempDir(), "out.pdf"), nil, nil, false)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "remove annotations file missing input",
			fn: func() error {
				return RemoveAnnotationsFile("", filepath.Join(t.TempDir(), "out.pdf"), nil, nil, nil, nil, false)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "encrypt missing configuration",
			fn: func() error {
				return Encrypt(bytes.NewReader(nil), io.Discard, nil)
			},
			wantErr: ErrMissingConfiguration,
		},
		{
			name: "merge raw missing inputs",
			fn: func() error {
				return MergeRaw(nil, io.Discard, false, nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "merge raw missing writer",
			fn: func() error {
				return MergeRaw([]io.ReadSeeker{bytes.NewReader(nil)}, nil, false, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "merge raw missing first reader",
			fn: func() error {
				return MergeRaw([]io.ReadSeeker{nil}, io.Discard, false, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "merge missing writer",
			fn: func() error {
				return Merge("", []string{"in.pdf"}, nil, nil, false)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "merge missing input",
			fn: func() error {
				return Merge("", nil, io.Discard, nil, false)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "merge zip missing first reader",
			fn: func() error {
				return MergeCreateZip(nil, bytes.NewReader(nil), io.Discard, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "merge zip missing second reader",
			fn: func() error {
				return MergeCreateZip(bytes.NewReader(nil), nil, io.Discard, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "merge zip missing writer",
			fn: func() error {
				return MergeCreateZip(bytes.NewReader(nil), bytes.NewReader(nil), nil, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "n-up missing configuration",
			fn: func() error {
				return NUp(bytes.NewReader(nil), io.Discard, nil, nil, nil, nil)
			},
			wantErr: ErrMissingNUpConfiguration,
		},
		{
			name: "booklet missing reader",
			fn: func() error {
				return Booklet(nil, io.Discard, nil, nil, bookletTestConfiguration(t, false), nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "booklet missing writer",
			fn: func() error {
				return Booklet(bytes.NewReader(nil), nil, nil, nil, DefaultBookletConfig(), nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "booklet missing configuration",
			fn: func() error {
				return Booklet(bytes.NewReader(nil), io.Discard, nil, nil, nil, nil)
			},
			wantErr: ErrMissingBookletConfiguration,
		},
		{
			name: "booklet file missing input",
			fn: func() error {
				return BookletFile(nil, "out.pdf", nil, nil, nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "booklet file missing image input",
			fn: func() error {
				return BookletFile(nil, "out.pdf", nil, bookletTestConfiguration(t, true), nil)
			},
			wantErr: ErrMissingImageInput,
		},
		{
			name: "booklet missing image input",
			fn: func() error {
				nup := DefaultBookletConfig()
				nup.ImgInputFile = true
				return Booklet(bytes.NewReader(nil), io.Discard, nil, nil, nup, nil)
			},
			wantErr: ErrMissingImageInput,
		},
		{
			name: "booklet image context missing input",
			fn: func() error {
				_, err := BookletFromImages(nil, nil, DefaultBookletConfig())
				return err
			},
			wantErr: ErrMissingImageInput,
		},
		{
			name: "booklet image context missing configuration",
			fn: func() error {
				_, err := BookletFromImages(nil, nil, nil)
				return err
			},
			wantErr: ErrMissingBookletConfiguration,
		},
		{
			name: "update images missing image input",
			fn: func() error {
				return UpdateImages(bytes.NewReader(nil), nil, io.Discard, 1, 0, "", nil)
			},
			wantErr: ErrMissingImageInput,
		},
		{
			name: "update images missing writer",
			fn: func() error {
				return UpdateImages(bytes.NewReader(nil), bytes.NewReader(nil), nil, 1, 0, "", nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "create missing JSON reader",
			fn: func() error {
				return Create(nil, nil, io.Discard, nil)
			},
			wantErr: ErrMissingJSONInput,
		},
		{
			name: "create missing writer",
			fn: func() error {
				return Create(nil, strings.NewReader("{}"), nil, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "create PDF file missing xref table",
			fn: func() error {
				return CreatePDFFile(nil, filepath.Join(t.TempDir(), "out.pdf"), nil)
			},
			wantErr: ErrMissingXRefTable,
		},
		{
			name: "create file missing JSON input",
			fn: func() error {
				return CreateFile("", "", filepath.Join(t.TempDir(), "out.pdf"), nil)
			},
			wantErr: ErrMissingJSONInput,
		},
		{
			name: "create file missing input or output",
			fn: func() error {
				jsonFile := filepath.Join(t.TempDir(), "create.json")
				if err := os.WriteFile(jsonFile, []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
				return CreateFile("", jsonFile, "", nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "page layout missing reader",
			fn: func() error {
				_, err := PageLayout(nil, nil)
				return err
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "page layout file missing input",
			fn: func() error {
				_, err := PageLayoutFile("", nil)
				return err
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "list page layout missing reader",
			fn: func() error {
				_, err := ListPageLayout(nil, nil)
				return err
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "list page layout file missing input",
			fn: func() error {
				_, err := ListPageLayoutFile("", nil)
				return err
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "set page layout missing reader",
			fn: func() error {
				return SetPageLayout(nil, io.Discard, model.PageLayoutSinglePage, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "set page layout missing writer",
			fn: func() error {
				return SetPageLayout(bytes.NewReader(nil), nil, model.PageLayoutSinglePage, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "set page layout invalid layout",
			fn: func() error {
				return SetPageLayout(bytes.NewReader(nil), io.Discard, model.PageLayout(-1), nil)
			},
			wantErr:     ErrInvalidPageLayout,
			wantContext: "set page layout: invalid value -1",
		},
		{
			name: "set page layout file missing input",
			fn: func() error {
				return SetPageLayoutFile("", filepath.Join(t.TempDir(), "out.pdf"), model.PageLayoutSinglePage, nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "set page layout file invalid layout",
			fn: func() error {
				return SetPageLayoutFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), model.PageLayout(99), nil)
			},
			wantErr:     ErrInvalidPageLayout,
			wantContext: "set page layout: invalid value 99",
		},
		{
			name: "reset page layout missing reader",
			fn: func() error {
				return ResetPageLayout(nil, io.Discard, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "reset page layout missing writer",
			fn: func() error {
				return ResetPageLayout(bytes.NewReader(nil), nil, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "reset page layout file missing input",
			fn: func() error {
				return ResetPageLayoutFile("", filepath.Join(t.TempDir(), "out.pdf"), nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "page mode missing reader",
			fn: func() error {
				_, err := PageMode(nil, nil)
				return err
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "page mode file missing input",
			fn: func() error {
				_, err := PageModeFile("", nil)
				return err
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "list page mode missing reader",
			fn: func() error {
				_, err := ListPageMode(nil, nil)
				return err
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "list page mode file missing input",
			fn: func() error {
				_, err := ListPageModeFile("", nil)
				return err
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "set page mode missing reader",
			fn: func() error {
				return SetPageMode(nil, io.Discard, model.PageModeUseNone, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "set page mode missing writer",
			fn: func() error {
				return SetPageMode(bytes.NewReader(nil), nil, model.PageModeUseNone, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "set page mode invalid mode",
			fn: func() error {
				return SetPageMode(bytes.NewReader(nil), io.Discard, model.PageMode(-1), nil)
			},
			wantErr:     ErrInvalidPageMode,
			wantContext: "set page mode: invalid value -1",
		},
		{
			name: "set page mode file missing input",
			fn: func() error {
				return SetPageModeFile("", filepath.Join(t.TempDir(), "out.pdf"), model.PageModeUseNone, nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "set page mode file invalid mode",
			fn: func() error {
				return SetPageModeFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), model.PageMode(99), nil)
			},
			wantErr:     ErrInvalidPageMode,
			wantContext: "set page mode: invalid value 99",
		},
		{
			name: "reset page mode missing reader",
			fn: func() error {
				return ResetPageMode(nil, io.Discard, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "reset page mode missing writer",
			fn: func() error {
				return ResetPageMode(bytes.NewReader(nil), nil, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "reset page mode file missing input",
			fn: func() error {
				return ResetPageModeFile("", filepath.Join(t.TempDir(), "out.pdf"), nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "viewer preferences missing reader",
			fn: func() error {
				_, _, err := ViewerPreferences(nil, nil)
				return err
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "viewer preferences file missing input",
			fn: func() error {
				_, err := ViewerPreferencesFile("", false, nil)
				return err
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "list viewer preferences missing reader",
			fn: func() error {
				_, err := ListViewerPreferences(nil, false, nil)
				return err
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "list viewer preferences JSON missing reader",
			fn: func() error {
				_, err := ListViewerPreferencesJSON(nil, false, nil)
				return err
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "list viewer preferences file missing input",
			fn: func() error {
				_, err := ListViewerPreferencesFile("", false, false, nil)
				return err
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "set viewer preferences missing reader",
			fn: func() error {
				return SetViewerPreferences(nil, io.Discard, model.ViewerPreferences{}, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "set viewer preferences missing writer",
			fn: func() error {
				return SetViewerPreferences(bytes.NewReader(nil), nil, model.ViewerPreferences{}, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "set viewer preferences invalid JSON",
			fn: func() error {
				return SetViewerPreferencesFromJSONBytes(bytes.NewReader(nil), io.Discard, []byte("{"), nil)
			},
			wantErr: ErrInvalidJSON,
		},
		{
			name: "set viewer preferences missing JSON reader",
			fn: func() error {
				return SetViewerPreferencesFromJSONReader(bytes.NewReader(nil), io.Discard, nil, nil)
			},
			wantErr: ErrMissingJSONReader,
		},
		{
			name: "set viewer preferences file missing input",
			fn: func() error {
				return SetViewerPreferencesFile("", filepath.Join(t.TempDir(), "out.pdf"), model.ViewerPreferences{}, nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "set viewer preferences JSON file missing JSON input",
			fn: func() error {
				return SetViewerPreferencesFileFromJSONFile("in.pdf", "", "", nil)
			},
			wantErr: ErrMissingJSONInput,
		},
		{
			name: "reset viewer preferences missing reader",
			fn: func() error {
				return ResetViewerPreferences(nil, io.Discard, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "reset viewer preferences missing writer",
			fn: func() error {
				return ResetViewerPreferences(bytes.NewReader(nil), nil, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "reset viewer preferences file missing input",
			fn: func() error {
				return ResetViewerPreferencesFile("", filepath.Join(t.TempDir(), "out.pdf"), nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "validate context missing context",
			fn: func() error {
				return ValidateContext(nil)
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "validate context missing xref table",
			fn: func() error {
				return ValidateContext(&model.Context{})
			},
			wantErr: ErrMissingXRefTable,
		},
		{
			name: "optimize context missing context",
			fn: func() error {
				return OptimizeContext(nil)
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "write context missing context",
			fn: func() error {
				return WriteContext(nil, io.Discard)
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "write context missing writer",
			fn: func() error {
				return WriteContext(&model.Context{}, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "write increment missing context",
			fn: func() error {
				return WriteIncrement(nil, io.Discard)
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "write increment missing writer",
			fn: func() error {
				return WriteIncrement(&model.Context{}, nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "write missing context",
			fn: func() error {
				return Write(nil, io.Discard, nil)
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "write incr missing read write seeker",
			fn: func() error {
				return WriteIncr(&model.Context{}, nil, model.NewDefaultConfiguration())
			},
			wantErr: ErrMissingPDFReadWriteSeeker,
		},
		{
			name: "write incr missing configuration",
			fn: func() error {
				return WriteIncr(&model.Context{}, nopReadWriteSeeker{bytes.NewReader(nil)}, nil)
			},
			wantErr: ErrMissingConfiguration,
		},
		{
			name: "extract page missing context",
			fn: func() error {
				_, err := ExtractPage(nil, 1)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "watermark context missing context",
			fn: func() error {
				return WatermarkContext(nil, nil, nil)
			},
			wantErr: ErrMissingPDFContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if tt.wantContext != "" && !strings.Contains(err.Error(), tt.wantContext) {
				t.Fatalf("expected %q in error, got %q", tt.wantContext, err.Error())
			}
		})
	}
}

// TestBookletFileRejectsMissingOutput verifies the corresponding behavior.
func TestBookletFileRejectsMissingOutput(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	err := BookletFile([]string{inFile}, "", nil, DefaultBookletConfig(), nil)
	if !errors.Is(err, ErrMissingPDFOutput) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFOutput, err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing output argument must not report a filesystem lookup failure: %v", err)
	}
}

// TestWriteIncrementReturnsFlushError verifies the corresponding behavior.
func TestWriteIncrementReturnsFlushError(t *testing.T) {
	wantErr := errors.New("flush failed")

	ctx, err := pdfcpu.CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}

	err = WriteIncrement(ctx, failingWriter{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

// TestExtractPageErrorsIncludePhaseContext verifies the corresponding behavior.
func TestExtractPageErrorsIncludePhaseContext(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{PageCount: 1}}

	_, err := ExtractPage(ctx, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "extract page 1") {
		t.Fatalf("expected extract page context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "extract pages [1]") {
		t.Fatalf("expected lower extract pages context, got %q", err.Error())
	}
}

// TestValidateAPIMissingReaderError verifies the corresponding behavior.
func TestValidateAPIMissingReaderError(t *testing.T) {
	if err := Validate(nil, nil); !errors.Is(err, ErrMissingPDFReadSeeker) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFReadSeeker, err)
	}
}

// TestReadContextMissingReaderError verifies the corresponding behavior.
func TestReadContextMissingReaderError(t *testing.T) {
	_, err := ReadContext(nil, nil)
	if !errors.Is(err, ErrMissingPDFReadSeeker) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFReadSeeker, err)
	}
}

// TestReadAndValidateMissingReaderError verifies the corresponding behavior.
func TestReadAndValidateMissingReaderError(t *testing.T) {
	_, err := ReadAndValidate(nil, nil)
	if !errors.Is(err, ErrMissingPDFReadSeeker) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFReadSeeker, err)
	}
}

// TestValidateAPIReadErrorsIncludePhaseContext verifies the corresponding behavior.
func TestValidateAPIReadErrorsIncludePhaseContext(t *testing.T) {
	err := Validate(bytes.NewReader(nil), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "read context") {
		t.Fatalf("expected read context, got %q", err.Error())
	}
}

// TestReadAndValidateReadErrorsIncludePhaseContext verifies the corresponding behavior.
func TestReadAndValidateReadErrorsIncludePhaseContext(t *testing.T) {
	_, err := ReadAndValidate(bytes.NewReader(nil), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "read context") {
		t.Fatalf("expected read context, got %q", err.Error())
	}
}

// TestReadAndValidateValidationErrorsMatchValidateAPIContext verifies the corresponding behavior.
func TestReadAndValidateValidationErrorsMatchValidateAPIContext(t *testing.T) {
	tests := []struct {
		name     string
		mode     int
		wantHint bool
	}{
		{"strict", model.ValidationStrict, true},
		{"relaxed", model.ValidationRelaxed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := model.NewDefaultConfiguration()
			conf.ValidationMode = tt.mode

			_, err := ReadAndValidate(bytes.NewReader(invalidValidationTestPDF()), conf)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), "validation error (obj#:1)") {
				t.Fatalf("expected validation object context, got %q", err.Error())
			}
			if strings.Contains(err.Error(), "try --mode=relaxed") != tt.wantHint {
				t.Fatalf("unexpected strict-mode hint presence in %q", err.Error())
			}
			if strings.Contains(err.Error(), "validate context") {
				t.Fatalf("unexpected old validate context wrapper: %q", err.Error())
			}
		})
	}
}

// TestReadContextEmptyInputPreservesPdfcpuSentinel verifies the corresponding behavior.
func TestReadContextEmptyInputPreservesPdfcpuSentinel(t *testing.T) {
	_, err := ReadContext(bytes.NewReader(nil), nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
}

// TestCreateImportJSONErrorIncludesPhaseContext verifies the corresponding behavior.
func TestCreateImportJSONErrorIncludesPhaseContext(t *testing.T) {
	err := Create(nil, strings.NewReader("{"), io.Discard, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create: import JSON") {
		t.Fatalf("expected import JSON context, got %q", err.Error())
	}
}

// TestCreateRenderErrorShowsWrappedUserChain verifies the corresponding behavior.
func TestCreateRenderErrorShowsWrappedUserChain(t *testing.T) {
	const input = `{
		"pages": {
			"1": {
				"content": {
					"text": [
						{"name": "$missing"}
					]
				}
			}
		}
	}`

	err := Create(nil, strings.NewReader(input), io.Discard, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	msg := err.Error()
	pos := 0
	for _, want := range []string{
		"create: import JSON",
		"render pages",
		"page 1: content",
		"text boxes",
		"text 1",
	} {
		i := strings.Index(msg[pos:], want)
		if i < 0 {
			t.Fatalf("expected ordered fragment %q in %q", want, msg)
		}
		pos += i + len(want)
	}
}

// TestCreateFileMissingJSONPreservesNotExist verifies the corresponding behavior.
func TestCreateFileMissingJSONPreservesNotExist(t *testing.T) {
	err := CreateFile("", "missing.json", filepath.Join(t.TempDir(), "out.pdf"), nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "create: open JSON input missing.json") {
		t.Fatalf("expected JSON open context, got %q", err.Error())
	}
}

// TestCreateFileMissingSourceWithoutOutputReportsInput verifies the corresponding behavior.
func TestCreateFileMissingSourceWithoutOutputReportsInput(t *testing.T) {
	dir := t.TempDir()
	jsonFile := filepath.Join(dir, "create.json")
	if err := os.WriteFile(jsonFile, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	inFile := filepath.Join(dir, "missing.pdf")

	err := CreateFile(inFile, jsonFile, "", nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "create: open input "+inFile) {
		t.Fatalf("expected input open context, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "create: create output") {
		t.Fatalf("unexpected output creation context, got %q", err.Error())
	}
}

// TestCreateFileMissingSourceWithOutputDoesNotCreateOutput verifies the corresponding behavior.
func TestCreateFileMissingSourceWithOutputDoesNotCreateOutput(t *testing.T) {
	dir := t.TempDir()
	jsonFile := filepath.Join(dir, "create.json")
	if err := os.WriteFile(jsonFile, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	inFile := filepath.Join(dir, "missing.pdf")
	outFile := filepath.Join(dir, "out.pdf")

	err := CreateFile(inFile, jsonFile, outFile, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "create: open input "+inFile) {
		t.Fatalf("expected input open context, got %q", err.Error())
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected no output file, got stat error %v", statErr)
	}
}

// TestCreateFileRemovesOutputOnFailure verifies the corresponding behavior.
func TestCreateFileRemovesOutputOnFailure(t *testing.T) {
	jsonFile := filepath.Join(t.TempDir(), "create.json")
	if err := os.WriteFile(jsonFile, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	err := CreateFile("", jsonFile, outFile, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create: import JSON") {
		t.Fatalf("expected import JSON context, got %q", err.Error())
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got stat error %v", statErr)
	}
}

func testAnnotationRenderer() model.AnnotationRenderer {
	return model.NewLinkAnnotation(
		*types.NewRectangle(0, 0, 10, 10),
		0,
		"",
		"api-test-annotation",
		"",
		0,
		nil,
		nil,
		"https://pdfcpu.io",
		nil,
		false,
		0,
		model.BSSolid,
	)
}

// TestReadValidateAndOptimizeCallersAddPrepareContextOnce verifies the shared context preparation contract.
func TestReadValidateAndOptimizeCallersAddPrepareContextOnce(t *testing.T) {
	ann := testAnnotationRenderer()
	tests := []struct {
		name string
		fn   func() error
		op   string
	}{
		{name: "optimize", fn: func() error {
			return Optimize(bytes.NewReader(nil), io.Discard, nil)
		}, op: "optimize"},
		{name: "trim", fn: func() error {
			return Trim(bytes.NewReader(nil), io.Discard, nil, nil)
		}, op: "trim"},
		{name: "create", fn: func() error {
			return Create(bytes.NewReader(nil), strings.NewReader(`{}`), io.Discard, nil)
		}, op: "create"},
		{name: "list annotations", fn: func() error {
			_, err := Annotations(bytes.NewReader(nil), nil, nil)
			return err
		}, op: "list annotations"},
		{name: "add annotations", fn: func() error {
			return AddAnnotations(bytes.NewReader(nil), io.Discard, nil, ann, nil)
		}, op: "add annotations"},
		{name: "add annotation map", fn: func() error {
			return AddAnnotationsMap(bytes.NewReader(nil), io.Discard, map[int][]model.AnnotationRenderer{1: {ann}}, nil)
		}, op: "add annotations"},
		{name: "remove annotations", fn: func() error {
			return RemoveAnnotations(bytes.NewReader(nil), io.Discard, nil, nil, nil, nil)
		}, op: "remove annotations"},
		{name: "bookmarks", fn: func() error {
			_, err := Bookmarks(bytes.NewReader(nil), nil)
			return err
		}, op: "list bookmarks"},
		{name: "list bookmarks", fn: func() error {
			_, err := ListBookmarks(bytes.NewReader(nil), nil)
			return err
		}, op: "list bookmarks"},
		{name: "export bookmarks", fn: func() error {
			return ExportBookmarksJSON(bytes.NewReader(nil), io.Discard, "source.pdf", nil)
		}, op: "export bookmarks"},
		{name: "import bookmarks", fn: func() error {
			return ImportBookmarks(bytes.NewReader(nil), strings.NewReader(`{}`), io.Discard, false, nil)
		}, op: "import bookmarks"},
		{name: "add bookmarks", fn: func() error {
			return AddBookmarks(bytes.NewReader(nil), io.Discard, []pdfcpu.Bookmark{{Title: "test", PageFrom: 1}}, false, nil)
		}, op: "add bookmarks"},
		{name: "remove bookmarks", fn: func() error {
			return RemoveBookmarks(bytes.NewReader(nil), io.Discard, nil)
		}, op: "remove bookmarks"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if !strings.HasPrefix(err.Error(), tt.op+": ") {
				t.Fatalf("expected %q operation prefix, got %q", tt.op, err.Error())
			}
			if got := strings.Count(err.Error(), "prepare PDF context"); got != 1 {
				t.Fatalf("expected prepare context once, got %d in %q", got, err.Error())
			}
		})
	}
}

// TestValidateAnnotationRendererMapReturnsDeterministicError verifies the corresponding behavior.
func TestValidateAnnotationRendererMapReturnsDeterministicError(t *testing.T) {
	m := map[int][]model.AnnotationRenderer{
		2: {nil},
		1: {nil},
	}
	for i := 0; i < 25; i++ {
		err := validateAnnotationRendererMap(m)
		if !errors.Is(err, ErrMissingAnnotation) || !strings.Contains(err.Error(), "page 1 annotation 1") {
			t.Fatalf("iteration %d: expected page 1 annotation error, got %v", i, err)
		}
	}
}

func nopWriteAPITestPDF(t *testing.T, elems ...string) io.ReadWriteSeeker {
	t.Helper()

	bb, err := os.ReadFile(filepath.Join(elems...))
	if err != nil {
		t.Fatal(err)
	}
	return nopReadWriteSeeker{bytes.NewReader(bb)}
}

func failingWriteAPITestPDF(t *testing.T, err error, elems ...string) io.ReadWriteSeeker {
	t.Helper()

	bb, readErr := os.ReadFile(filepath.Join(elems...))
	if readErr != nil {
		t.Fatal(readErr)
	}
	return failingReadWriteSeeker{Reader: bytes.NewReader(bb), err: err}
}

// TestAnnotationReadErrorsIncludePhaseContext verifies the corresponding behavior.
func TestAnnotationReadErrorsIncludePhaseContext(t *testing.T) {
	ann := testAnnotationRenderer()
	annMap := map[int][]model.AnnotationRenderer{1: {ann}}

	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "list",
			fn: func() error {
				_, err := Annotations(bytes.NewReader(nil), nil, nil)
				return err
			},
			want: "list annotations: prepare PDF context",
		},
		{
			name: "add",
			fn: func() error {
				return AddAnnotations(bytes.NewReader(nil), io.Discard, nil, ann, nil)
			},
			want: "add annotations: prepare PDF context",
		},
		{
			name: "add map",
			fn: func() error {
				return AddAnnotationsMap(bytes.NewReader(nil), io.Discard, annMap, nil)
			},
			want: "add annotations: prepare PDF context",
		},
		{
			name: "remove",
			fn: func() error {
				return RemoveAnnotations(bytes.NewReader(nil), io.Discard, nil, nil, nil, nil)
			},
			want: "remove annotations: prepare PDF context",
		},
		{
			name: "add increment",
			fn: func() error {
				return AddAnnotationsAsIncrement(nopReadWriteSeeker{bytes.NewReader(nil)}, nil, ann, nil)
			},
			want: "add annotations: prepare PDF context",
		},
		{
			name: "add map increment",
			fn: func() error {
				return AddAnnotationsMapAsIncrement(nopReadWriteSeeker{bytes.NewReader(nil)}, annMap, nil)
			},
			want: "add annotations: prepare PDF context",
		},
		{
			name: "remove increment",
			fn: func() error {
				return RemoveAnnotationsAsIncrement(nopReadWriteSeeker{bytes.NewReader(nil)}, nil, nil, nil, nil)
			},
			want: "remove annotations: prepare PDF context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
			if !strings.Contains(err.Error(), "read context") {
				t.Fatalf("expected read context, got %q", err.Error())
			}
		})
	}
}

// TestAnnotationPageSelectionErrorsIncludePhaseContext verifies the corresponding behavior.
func TestAnnotationPageSelectionErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	ann := testAnnotationRenderer()

	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "list",
			fn: func() error {
				_, err := Annotations(openAPITestPDF(t, inFile), []string{"foo"}, nil)
				return err
			},
			want: "list annotations: parse page selection",
		},
		{
			name: "add",
			fn: func() error {
				return AddAnnotations(openAPITestPDF(t, inFile), io.Discard, []string{"foo"}, ann, nil)
			},
			want: "add annotations: parse page selection",
		},
		{
			name: "remove",
			fn: func() error {
				return RemoveAnnotations(openAPITestPDF(t, inFile), io.Discard, []string{"foo"}, nil, nil, nil)
			},
			want: "remove annotations: parse page selection",
		},
		{
			name: "add increment",
			fn: func() error {
				return AddAnnotationsAsIncrement(nopWriteAPITestPDF(t, inFile), []string{"foo"}, ann, nil)
			},
			want: "add annotations: parse page selection",
		},
		{
			name: "remove increment",
			fn: func() error {
				return RemoveAnnotationsAsIncrement(nopWriteAPITestPDF(t, inFile), []string{"foo"}, nil, nil, nil)
			},
			want: "remove annotations: parse page selection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
			if !strings.Contains(err.Error(), "invalid syntax") {
				t.Fatalf("expected page selection detail, got %q", err.Error())
			}
		})
	}
}

// TestAnnotationWriteErrorsIncludePhaseContext verifies the corresponding behavior.
func TestAnnotationWriteErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	annotatedFile := filepath.Join("..", "samples", "annotations", "Annotations.pdf")
	ann := testAnnotationRenderer()

	tests := []struct {
		name string
		fn   func(error) error
		want string
	}{
		{
			name: "add",
			fn: func(wantErr error) error {
				return AddAnnotations(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, []string{"1"}, ann, nil)
			},
			want: "add annotations: write output",
		},
		{
			name: "add map",
			fn: func(wantErr error) error {
				m := map[int][]model.AnnotationRenderer{1: {ann}}
				return AddAnnotationsMap(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, m, nil)
			},
			want: "add annotations: write output",
		},
		{
			name: "remove",
			fn: func(wantErr error) error {
				return RemoveAnnotations(openAPITestPDF(t, annotatedFile), failingWriter{err: wantErr}, nil, nil, nil, nil)
			},
			want: "remove annotations: write output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := errors.New(tt.name + " write failed")
			err := tt.fn(wantErr)
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestAnnotationIncrementWriteErrorsIncludePhaseContext verifies the corresponding behavior.
func TestAnnotationIncrementWriteErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	annotatedFile := filepath.Join("..", "samples", "annotations", "Annotations.pdf")
	ann := testAnnotationRenderer()

	tests := []struct {
		name  string
		input string
		fn    func(io.ReadWriteSeeker) error
		want  string
	}{
		{
			name:  "add",
			input: inFile,
			fn: func(rws io.ReadWriteSeeker) error {
				return AddAnnotationsAsIncrement(rws, []string{"1"}, ann, nil)
			},
			want: "add annotations: write increment",
		},
		{
			name:  "add map",
			input: inFile,
			fn: func(rws io.ReadWriteSeeker) error {
				m := map[int][]model.AnnotationRenderer{1: {ann}}
				return AddAnnotationsMapAsIncrement(rws, m, nil)
			},
			want: "add annotations: write increment",
		},
		{
			name:  "remove",
			input: annotatedFile,
			fn: func(rws io.ReadWriteSeeker) error {
				return RemoveAnnotationsAsIncrement(rws, nil, nil, nil, nil)
			},
			want: "remove annotations: write increment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := errors.New(tt.name + " increment failed")
			err := tt.fn(failingWriteAPITestPDF(t, wantErr, tt.input))
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestAnnotationFileErrorsIncludePhaseContext verifies the corresponding behavior.
func TestAnnotationFileErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	ann := testAnnotationRenderer()
	annMap := map[int][]model.AnnotationRenderer{1: {ann}}

	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "add open input",
			fn: func() error {
				return AddAnnotationsFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), nil, ann, nil, false)
			},
			want: "add annotations: open input missing.pdf",
		},
		{
			name: "add create output",
			fn: func() error {
				return AddAnnotationsFile(inFile, filepath.Join(t.TempDir(), "missing-dir", "out.pdf"), nil, ann, nil, false)
			},
			want: "add annotations: create output",
		},
		{
			name: "add map open input",
			fn: func() error {
				return AddAnnotationsMapFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), annMap, nil, false)
			},
			want: "add annotations: open input missing.pdf",
		},
		{
			name: "add map create output",
			fn: func() error {
				return AddAnnotationsMapFile(inFile, filepath.Join(t.TempDir(), "missing-dir", "out.pdf"), annMap, nil, false)
			},
			want: "add annotations: create output",
		},
		{
			name: "remove open input",
			fn: func() error {
				return RemoveAnnotationsFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), nil, nil, nil, nil, false)
			},
			want: "remove annotations: open input missing.pdf",
		},
		{
			name: "remove create output",
			fn: func() error {
				return RemoveAnnotationsFile(inFile, filepath.Join(t.TempDir(), "missing-dir", "out.pdf"), nil, nil, nil, nil, false)
			},
			want: "remove annotations: create output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected not exist error, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestCloseAnnotationFilesErrorsIncludePhaseContext verifies the corresponding behavior.
func TestCloseAnnotationFilesErrorsIncludePhaseContext(t *testing.T) {
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
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "add annotations").commit()
			},
			wantErr: os.ErrClosed,
			want:    "add annotations: close output",
		},
		{
			name: "close input",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				if err := f1.Close(); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "add annotations").commit()
			},
			wantErr: os.ErrClosed,
			want:    "add annotations: close input",
		},
		{
			name: "replace input",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				tmpFile := f2.Name()
				if err := os.Remove(tmpFile); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, tmpFile, f1.Name(), "", "", "add annotations").commit()
			},
			wantErr: os.ErrNotExist,
			want:    "add annotations: replace input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(t)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestCleanupAnnotationFilesJoinsErrors verifies the corresponding behavior.
func TestCleanupAnnotationFilesJoinsErrors(t *testing.T) {
	wantErr := errors.New("add annotations failed")
	f1 := createTempAPITestFile(t)
	f2 := createTempAPITestFile(t)
	if err := f1.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f2.Close(); err != nil {
		t.Fatal(err)
	}

	err := newStagedOutput(f1, f2, f2.Name(), "", "", "", "add annotations").cleanup(wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected closed file error, got %v", err)
	}
	for _, want := range []string{"add annotations: close output", "add annotations: close input"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

// TestPageLayoutFileWrapperErrorsIncludePhaseContext verifies the corresponding behavior.
func TestPageLayoutFileWrapperErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "page layout open input",
			fn: func() error {
				_, err := PageLayoutFile("missing.pdf", nil)
				return err
			},
			want: "list page layout: open input missing.pdf",
		},
		{
			name: "list page layout open input",
			fn: func() error {
				_, err := ListPageLayoutFile("missing.pdf", nil)
				return err
			},
			want: "list page layout: open input missing.pdf",
		},
		{
			name: "set page layout open input",
			fn: func() error {
				return SetPageLayoutFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), model.PageLayoutSinglePage, nil)
			},
			want: "set page layout: open input missing.pdf",
		},
		{
			name: "set page layout create output",
			fn: func() error {
				return SetPageLayoutFile(inFile, filepath.Join(t.TempDir(), "missing-dir", "out.pdf"), model.PageLayoutSinglePage, nil)
			},
			want: "set page layout: create output",
		},
		{
			name: "reset page layout open input",
			fn: func() error {
				return ResetPageLayoutFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), nil)
			},
			want: "reset page layout: open input missing.pdf",
		},
		{
			name: "reset page layout create output",
			fn: func() error {
				return ResetPageLayoutFile(inFile, filepath.Join(t.TempDir(), "missing-dir", "out.pdf"), nil)
			},
			want: "reset page layout: create output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected not exist error, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestClosePageLayoutFilesErrorsIncludePhaseContext verifies the corresponding behavior.
func TestClosePageLayoutFilesErrorsIncludePhaseContext(t *testing.T) {
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
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "set page layout").commit()
			},
			wantErr: os.ErrClosed,
			want:    "set page layout: close output",
		},
		{
			name: "close input",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				if err := f1.Close(); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "set page layout").commit()
			},
			wantErr: os.ErrClosed,
			want:    "set page layout: close input",
		},
		{
			name: "replace input",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				tmpFile := f2.Name()
				if err := os.Remove(tmpFile); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, tmpFile, f1.Name(), "", "", "set page layout").commit()
			},
			wantErr: os.ErrNotExist,
			want:    "set page layout: replace input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(t)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestCleanupPageLayoutFilesJoinsErrors verifies the corresponding behavior.
func TestCleanupPageLayoutFilesJoinsErrors(t *testing.T) {
	wantErr := errors.New("set page layout failed")
	f1 := createTempAPITestFile(t)
	f2 := createTempAPITestFile(t)
	if err := f1.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f2.Close(); err != nil {
		t.Fatal(err)
	}

	err := newStagedOutput(f1, f2, f2.Name(), "", "", "", "set page layout").cleanup(wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected closed file error, got %v", err)
	}
	for _, want := range []string{"set page layout: close output", "set page layout: close input"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

// TestPageLayoutReadErrorsIncludePhaseContext verifies the corresponding behavior.
func TestPageLayoutReadErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "page layout",
			fn: func() error {
				_, err := PageLayout(bytes.NewReader(nil), nil)
				return err
			},
			want: "list page layout: prepare PDF context",
		},
		{
			name: "list page layout",
			fn: func() error {
				_, err := ListPageLayout(bytes.NewReader(nil), nil)
				return err
			},
			want: "list page layout: prepare PDF context",
		},
		{
			name: "set page layout",
			fn: func() error {
				return SetPageLayout(bytes.NewReader(nil), io.Discard, model.PageLayoutSinglePage, nil)
			},
			want: "set page layout: prepare PDF context",
		},
		{
			name: "reset page layout",
			fn: func() error {
				return ResetPageLayout(bytes.NewReader(nil), io.Discard, nil)
			},
			want: "reset page layout: prepare PDF context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
			if !strings.Contains(err.Error(), "read context") {
				t.Fatalf("expected read context, got %q", err.Error())
			}
		})
	}
}

// TestPageLayoutWriteErrorsIncludePhaseContext verifies the corresponding behavior.
func TestPageLayoutWriteErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name string
		fn   func(error) error
		want string
	}{
		{
			name: "set page layout",
			fn: func(wantErr error) error {
				return SetPageLayout(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, model.PageLayoutTwoColumnLeft, nil)
			},
			want: "set page layout: write output",
		},
		{
			name: "reset page layout",
			fn: func(wantErr error) error {
				return ResetPageLayout(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, nil)
			},
			want: "reset page layout: write output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := errors.New(tt.name + " failed")
			err := tt.fn(wantErr)
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestResetPageLayoutWithoutPageLayoutStillWrites verifies the corresponding behavior.
func TestResetPageLayoutWithoutPageLayoutStillWrites(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "test.pdf")
	pl, err := PageLayout(openAPITestPDF(t, inFile), nil)
	if err != nil {
		t.Fatalf("read page layout: %v", err)
	}
	if pl != nil {
		t.Fatalf("expected no page layout, got %s", pl.String())
	}

	wantErr := errors.New("write attempted")
	err = ResetPageLayout(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected write error %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "reset page layout: write output") {
		t.Fatalf("expected write output context, got %q", err.Error())
	}
}

// TestResetPageLayoutFileWithoutPageLayoutStillWrites verifies the corresponding behavior.
func TestResetPageLayoutFileWithoutPageLayoutStillWrites(t *testing.T) {
	bb, err := os.ReadFile(filepath.Join("..", "testdata", "test.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(inFile, bb, 0600); err != nil {
		t.Fatal(err)
	}
	pl, err := PageLayoutFile(inFile, nil)
	if err != nil {
		t.Fatalf("read page layout: %v", err)
	}
	if pl != nil {
		t.Fatalf("expected no page layout, got %s", pl.String())
	}

	outFile := filepath.Join(dir, "out.pdf")
	if err := ResetPageLayoutFile(inFile, outFile, nil); err != nil {
		t.Fatalf("reset page layout: %v", err)
	}
	info, err := os.Stat(outFile)
	if err != nil {
		t.Fatalf("stat reset output: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected non-empty reset output")
	}
	pl, err = PageLayoutFile(outFile, nil)
	if err != nil {
		t.Fatalf("read reset output page layout: %v", err)
	}
	if pl != nil {
		t.Fatalf("expected reset output without page layout, got %s", pl.String())
	}
}

// TestPageLayoutFileRemovesOutputOnFailure verifies the corresponding behavior.
func TestPageLayoutFileRemovesOutputOnFailure(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "bad.pdf")
	if err := os.WriteFile(inFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	err := SetPageLayoutFile(inFile, outFile, model.PageLayoutSinglePage, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got stat error %v", statErr)
	}
}

// TestContentFileFailuresPreserveExistingOutput verifies the corresponding behavior.
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
			return SetPageLayoutFile(badInput, outFile, model.PageLayoutSinglePage, nil)
		}},
		{name: "page mode", wantErr: pdfcpu.ErrEmptyInput, run: func(outFile string) error {
			return SetPageModeFile(badInput, outFile, model.PageModeUseNone, nil)
		}},
		{name: "viewer preferences", wantErr: ErrInvalidJSON, run: func(outFile string) error {
			return SetViewerPreferencesFileFromJSONBytes(validInput, outFile, []byte("{"), nil)
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

// TestPageModeFileWrapperErrorsIncludePhaseContext verifies the corresponding behavior.
func TestPageModeFileWrapperErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "page mode open input",
			fn: func() error {
				_, err := PageModeFile("missing.pdf", nil)
				return err
			},
			want: "list page mode: open input missing.pdf",
		},
		{
			name: "list page mode open input",
			fn: func() error {
				_, err := ListPageModeFile("missing.pdf", nil)
				return err
			},
			want: "list page mode: open input missing.pdf",
		},
		{
			name: "set page mode open input",
			fn: func() error {
				return SetPageModeFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), model.PageModeUseNone, nil)
			},
			want: "set page mode: open input missing.pdf",
		},
		{
			name: "set page mode create output",
			fn: func() error {
				return SetPageModeFile(inFile, filepath.Join(t.TempDir(), "missing-dir", "out.pdf"), model.PageModeUseNone, nil)
			},
			want: "set page mode: create output",
		},
		{
			name: "reset page mode open input",
			fn: func() error {
				return ResetPageModeFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), nil)
			},
			want: "reset page mode: open input missing.pdf",
		},
		{
			name: "reset page mode create output",
			fn: func() error {
				return ResetPageModeFile(inFile, filepath.Join(t.TempDir(), "missing-dir", "out.pdf"), nil)
			},
			want: "reset page mode: create output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected not exist error, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestClosePageModeFilesErrorsIncludePhaseContext verifies the corresponding behavior.
func TestClosePageModeFilesErrorsIncludePhaseContext(t *testing.T) {
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
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "set page mode").commit()
			},
			wantErr: os.ErrClosed,
			want:    "set page mode: close output",
		},
		{
			name: "close input",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				if err := f1.Close(); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "set page mode").commit()
			},
			wantErr: os.ErrClosed,
			want:    "set page mode: close input",
		},
		{
			name: "replace input",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				tmpFile := f2.Name()
				if err := os.Remove(tmpFile); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, tmpFile, f1.Name(), "", "", "set page mode").commit()
			},
			wantErr: os.ErrNotExist,
			want:    "set page mode: replace input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(t)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestCleanupPageModeFilesJoinsErrors verifies the corresponding behavior.
func TestCleanupPageModeFilesJoinsErrors(t *testing.T) {
	wantErr := errors.New("set page mode failed")
	f1 := createTempAPITestFile(t)
	f2 := createTempAPITestFile(t)
	if err := f1.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f2.Close(); err != nil {
		t.Fatal(err)
	}

	err := newStagedOutput(f1, f2, f2.Name(), "", "", "", "set page mode").cleanup(wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected closed file error, got %v", err)
	}
	for _, want := range []string{"set page mode: close output", "set page mode: close input"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

// TestPageModeReadErrorsIncludePhaseContext verifies the corresponding behavior.
func TestPageModeReadErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "page mode",
			fn: func() error {
				_, err := PageMode(bytes.NewReader(nil), nil)
				return err
			},
			want: "list page mode: prepare PDF context",
		},
		{
			name: "list page mode",
			fn: func() error {
				_, err := ListPageMode(bytes.NewReader(nil), nil)
				return err
			},
			want: "list page mode: prepare PDF context",
		},
		{
			name: "set page mode",
			fn: func() error {
				return SetPageMode(bytes.NewReader(nil), io.Discard, model.PageModeUseNone, nil)
			},
			want: "set page mode: prepare PDF context",
		},
		{
			name: "reset page mode",
			fn: func() error {
				return ResetPageMode(bytes.NewReader(nil), io.Discard, nil)
			},
			want: "reset page mode: prepare PDF context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
			if !strings.Contains(err.Error(), "read context") {
				t.Fatalf("expected read context, got %q", err.Error())
			}
		})
	}
}

// TestPageModeWriteErrorsIncludePhaseContext verifies the corresponding behavior.
func TestPageModeWriteErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name string
		fn   func(error) error
		want string
	}{
		{
			name: "set page mode",
			fn: func(wantErr error) error {
				return SetPageMode(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, model.PageModeUseOutlines, nil)
			},
			want: "set page mode: write output",
		},
		{
			name: "reset page mode",
			fn: func(wantErr error) error {
				return ResetPageMode(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, nil)
			},
			want: "reset page mode: write output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := errors.New(tt.name + " failed")
			err := tt.fn(wantErr)
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestResetPageModeWithoutPageModeStillWrites verifies the corresponding behavior.
func TestResetPageModeWithoutPageModeStillWrites(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "test.pdf")
	pm, err := PageMode(openAPITestPDF(t, inFile), nil)
	if err != nil {
		t.Fatalf("read page mode: %v", err)
	}
	if pm != nil {
		t.Fatalf("expected no page mode, got %s", pm.String())
	}

	wantErr := errors.New("write attempted")
	err = ResetPageMode(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected write error %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "reset page mode: write output") {
		t.Fatalf("expected write output context, got %q", err.Error())
	}
}

// TestResetPageModeFileWithoutPageModeStillWrites verifies the corresponding behavior.
func TestResetPageModeFileWithoutPageModeStillWrites(t *testing.T) {
	bb, err := os.ReadFile(filepath.Join("..", "testdata", "test.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(inFile, bb, 0600); err != nil {
		t.Fatal(err)
	}
	pm, err := PageModeFile(inFile, nil)
	if err != nil {
		t.Fatalf("read page mode: %v", err)
	}
	if pm != nil {
		t.Fatalf("expected no page mode, got %s", pm.String())
	}

	outFile := filepath.Join(dir, "out.pdf")
	if err := ResetPageModeFile(inFile, outFile, nil); err != nil {
		t.Fatalf("reset page mode: %v", err)
	}
	info, err := os.Stat(outFile)
	if err != nil {
		t.Fatalf("stat reset output: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected non-empty reset output")
	}
	pm, err = PageModeFile(outFile, nil)
	if err != nil {
		t.Fatalf("read reset output page mode: %v", err)
	}
	if pm != nil {
		t.Fatalf("expected reset output without page mode, got %s", pm.String())
	}
}

// TestPageModeFileRemovesOutputOnFailure verifies the corresponding behavior.
func TestPageModeFileRemovesOutputOnFailure(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "bad.pdf")
	if err := os.WriteFile(inFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	err := SetPageModeFile(inFile, outFile, model.PageModeUseNone, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got stat error %v", statErr)
	}
}

// TestViewerPreferencesFileWrapperErrorsIncludePhaseContext verifies the corresponding behavior.
func TestViewerPreferencesFileWrapperErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "viewer preferences open PDF",
			fn: func() error {
				_, err := ViewerPreferencesFile("missing.pdf", false, nil)
				return err
			},
			want: "list viewer preferences: open input missing.pdf",
		},
		{
			name: "list open PDF",
			fn: func() error {
				_, err := ListViewerPreferencesFile("missing.pdf", false, false, nil)
				return err
			},
			want: "list viewer preferences: open input missing.pdf",
		},
		{
			name: "list JSON open PDF",
			fn: func() error {
				_, err := ListViewerPreferencesFile("missing.pdf", false, true, nil)
				return err
			},
			want: "list viewer preferences: open input missing.pdf",
		},
		{
			name: "set open PDF",
			fn: func() error {
				return SetViewerPreferencesFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), model.ViewerPreferences{}, nil)
			},
			want: "set viewer preferences: open input missing.pdf",
		},
		{
			name: "set create output",
			fn: func() error {
				return SetViewerPreferencesFile(inFile, filepath.Join(t.TempDir(), "missing-dir", "out.pdf"), model.ViewerPreferences{}, nil)
			},
			want: "set viewer preferences: create output",
		},
		{
			name: "set read JSON",
			fn: func() error {
				return SetViewerPreferencesFileFromJSONFile(inFile, filepath.Join(t.TempDir(), "out.pdf"), "missing.json", nil)
			},
			want: "set viewer preferences: read JSON missing.json",
		},
		{
			name: "reset open PDF",
			fn: func() error {
				return ResetViewerPreferencesFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), nil)
			},
			want: "reset viewer preferences: open input missing.pdf",
		},
		{
			name: "reset create output",
			fn: func() error {
				return ResetViewerPreferencesFile(inFile, filepath.Join(t.TempDir(), "missing-dir", "out.pdf"), nil)
			},
			want: "reset viewer preferences: create output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected not exist error, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestCloseViewerPreferencesFilesErrorsIncludePhaseContext verifies the corresponding behavior.
func TestCloseViewerPreferencesFilesErrorsIncludePhaseContext(t *testing.T) {
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
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "set viewer preferences").commit()
			},
			wantErr: os.ErrClosed,
			want:    "set viewer preferences: close output",
		},
		{
			name: "close input",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				if err := f1.Close(); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "set viewer preferences").commit()
			},
			wantErr: os.ErrClosed,
			want:    "set viewer preferences: close input",
		},
		{
			name: "replace input",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				tmpFile := f2.Name()
				if err := os.Remove(tmpFile); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, tmpFile, f1.Name(), "", "", "set viewer preferences").commit()
			},
			wantErr: os.ErrNotExist,
			want:    "set viewer preferences: replace input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(t)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestCleanupViewerPreferencesFilesJoinsErrors verifies the corresponding behavior.
func TestCleanupViewerPreferencesFilesJoinsErrors(t *testing.T) {
	wantErr := errors.New("set viewer preferences failed")
	f1 := createTempAPITestFile(t)
	f2 := createTempAPITestFile(t)
	if err := f1.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f2.Close(); err != nil {
		t.Fatal(err)
	}

	err := newStagedOutput(f1, f2, f2.Name(), "", "", "", "set viewer preferences").cleanup(wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected closed file error, got %v", err)
	}
	for _, want := range []string{"set viewer preferences: close output", "set viewer preferences: close input"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

// TestViewerPreferencesReadErrorsIncludePhaseContext verifies the corresponding behavior.
func TestViewerPreferencesReadErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "viewer preferences",
			fn: func() error {
				_, _, err := ViewerPreferences(bytes.NewReader(nil), nil)
				return err
			},
			want: "list viewer preferences: prepare PDF context",
		},
		{
			name: "list viewer preferences",
			fn: func() error {
				_, err := ListViewerPreferences(bytes.NewReader(nil), false, nil)
				return err
			},
			want: "list viewer preferences: prepare PDF context",
		},
		{
			name: "list viewer preferences JSON",
			fn: func() error {
				_, err := ListViewerPreferencesJSON(bytes.NewReader(nil), false, nil)
				return err
			},
			want: "list viewer preferences: prepare PDF context",
		},
		{
			name: "set viewer preferences",
			fn: func() error {
				return SetViewerPreferences(bytes.NewReader(nil), io.Discard, model.ViewerPreferences{}, nil)
			},
			want: "set viewer preferences: prepare PDF context",
		},
		{
			name: "reset viewer preferences",
			fn: func() error {
				return ResetViewerPreferences(bytes.NewReader(nil), io.Discard, nil)
			},
			want: "reset viewer preferences: prepare PDF context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
			if !strings.Contains(err.Error(), "read context") {
				t.Fatalf("expected read context, got %q", err.Error())
			}
		})
	}
}

// TestViewerPreferencesJSONErrorsIncludePhaseContext verifies the corresponding behavior.
func TestViewerPreferencesJSONErrorsIncludePhaseContext(t *testing.T) {
	err := SetViewerPreferencesFromJSONBytes(bytes.NewReader(nil), io.Discard, []byte("{"), nil)
	if !errors.Is(err, ErrInvalidJSON) {
		t.Fatalf("expected %v, got %v", ErrInvalidJSON, err)
	}
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("expected JSON syntax error, got %v", err)
	}
	if !strings.Contains(err.Error(), "set viewer preferences: decode JSON") {
		t.Fatalf("expected decode JSON context, got %q", err.Error())
	}

	err = SetViewerPreferencesFromJSONBytes(bytes.NewReader(nil), io.Discard, []byte(`{"HideMenubar":"yes"}`), nil)
	if !errors.Is(err, ErrInvalidJSON) {
		t.Fatalf("expected %v, got %v", ErrInvalidJSON, err)
	}
	var typeErr *json.UnmarshalTypeError
	if !errors.As(err, &typeErr) {
		t.Fatalf("expected JSON unmarshal type error, got %v", err)
	}
	if !strings.Contains(err.Error(), "set viewer preferences: decode JSON") {
		t.Fatalf("expected decode JSON context, got %q", err.Error())
	}

	err = SetViewerPreferencesFromJSONReader(bytes.NewReader(nil), io.Discard, nil, nil)
	if !errors.Is(err, ErrMissingJSONReader) {
		t.Fatalf("expected %v, got %v", ErrMissingJSONReader, err)
	}
	if !strings.Contains(err.Error(), "set viewer preferences: read JSON") {
		t.Fatalf("expected read JSON context, got %q", err.Error())
	}
}

// TestListViewerPreferencesJSONWithoutPreferencesReturnsValidJSON verifies the corresponding behavior.
func TestListViewerPreferencesJSONWithoutPreferencesReturnsValidJSON(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	ss, err := ListViewerPreferencesJSON(openAPITestPDF(t, inFile), false, nil)
	if err != nil {
		t.Fatalf("list viewer preferences JSON: %v", err)
	}
	if len(ss) != 1 || !json.Valid([]byte(ss[0])) {
		t.Fatalf("expected one valid JSON result, got %q", ss)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal([]byte(ss[0]), &result); err != nil {
		t.Fatalf("decode viewer preferences JSON: %v", err)
	}
	if got := string(result["viewerPreferences"]); got != "null" {
		t.Fatalf("expected null viewerPreferences, got %s", got)
	}
}

// TestViewerPreferencesWriteErrorsIncludePhaseContext verifies the corresponding behavior.
func TestViewerPreferencesWriteErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name string
		fn   func(error) error
		want string
	}{
		{
			name: "set viewer preferences",
			fn: func(wantErr error) error {
				inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
				vp := model.ViewerPreferences{}
				vp.SetCenterWindow(true)
				return SetViewerPreferences(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, vp, nil)
			},
			want: "set viewer preferences: write output",
		},
		{
			name: "reset viewer preferences",
			fn: func(wantErr error) error {
				inFile := filepath.Join("..", "testdata", "Hybrid-PDF.pdf")
				return ResetViewerPreferences(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, nil)
			},
			want: "reset viewer preferences: write output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := errors.New(tt.name + " failed")
			err := tt.fn(wantErr)
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestResetViewerPreferencesAbsentWritesPDF verifies the corresponding behavior.
func TestResetViewerPreferencesAbsentWritesPDF(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	var out bytes.Buffer

	if err := ResetViewerPreferences(openAPITestPDF(t, inFile), &out, nil); err != nil {
		t.Fatalf("reset absent viewer preferences: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("expected reset to write a PDF")
	}
	vp, _, err := ViewerPreferences(bytes.NewReader(out.Bytes()), nil)
	if err != nil {
		t.Fatalf("read reset output: %v", err)
	}
	if vp != nil {
		t.Fatalf("expected absent viewer preferences, got %+v", vp)
	}
}

// TestResetViewerPreferencesFileAbsentWritesPDF verifies the corresponding behavior.
func TestResetViewerPreferencesFileAbsentWritesPDF(t *testing.T) {
	bb, err := os.ReadFile(filepath.Join("..", "testdata", "Hybrid-PDF.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	inFile := filepath.Join(dir, "hybrid.pdf")
	if err := os.WriteFile(inFile, bb, 0600); err != nil {
		t.Fatal(err)
	}

	if err := ResetViewerPreferencesFile(inFile, "", nil); err != nil {
		t.Fatalf("reset viewer preferences: %v", err)
	}

	outFile := filepath.Join(dir, "out.pdf")
	if err := ResetViewerPreferencesFile(inFile, outFile, nil); err != nil {
		t.Fatalf("reset absent viewer preferences: %v", err)
	}
	vp, err := ViewerPreferencesFile(outFile, false, nil)
	if err != nil {
		t.Fatalf("read reset output: %v", err)
	}
	if vp != nil {
		t.Fatalf("expected absent viewer preferences, got %+v", vp)
	}
}

// TestPDFInfoMissingReaderError verifies the corresponding behavior.
func TestPDFInfoMissingReaderError(t *testing.T) {
	_, err := PDFInfo(nil, "", nil, false, nil)
	if !errors.Is(err, ErrMissingPDFReadSeeker) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFReadSeeker, err)
	}
}

// TestPDFInfoReadErrorsIncludePhaseContext verifies the corresponding behavior.
func TestPDFInfoReadErrorsIncludePhaseContext(t *testing.T) {
	_, err := PDFInfo(bytes.NewReader(nil), "", nil, false, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "info: prepare PDF context") {
		t.Fatalf("expected info prepare context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "read context") {
		t.Fatalf("expected read context, got %q", err.Error())
	}
}

// TestPDFInfoPageSelectionErrorsIncludePhaseContext verifies the corresponding behavior.
func TestPDFInfoPageSelectionErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")

	_, err := PDFInfo(openAPITestPDF(t, inFile), inFile, []string{"foo"}, false, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "info: parse page selection") {
		t.Fatalf("expected page selection context, got %q", err.Error())
	}
}

// TestPDFInfoSuccess verifies the corresponding behavior.
func TestPDFInfoSuccess(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")

	info, err := PDFInfo(openAPITestPDF(t, inFile), inFile, nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal("expected info")
	}
}

// TestCollectReadErrorsLetLowerPhaseContextSpeak verifies the corresponding behavior.
func TestCollectReadErrorsLetLowerPhaseContextSpeak(t *testing.T) {
	err := Collect(bytes.NewReader(nil), io.Discard, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "collect:") {
		t.Fatalf("expected collect context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "prepare PDF context") {
		t.Fatalf("expected prepare context, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "prepare PDF context: prepare PDF context") {
		t.Fatalf("unexpected duplicate prepare context, got %q", err.Error())
	}
}

// TestCollectPageSelectionErrorsIncludePhaseContext verifies the corresponding behavior.
func TestCollectPageSelectionErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")

	tests := []struct {
		name          string
		selectedPages []string
		want          string
	}{
		{
			name:          "invalid syntax",
			selectedPages: []string{"foo"},
			want:          "invalid syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Collect(openAPITestPDF(t, inFile), io.Discard, tt.selectedPages, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "collect: parse page selection") {
				t.Fatalf("expected collect page selection context, got %q", err.Error())
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected page selection detail %q, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestCollectRejectsEmptyPageSelectionWithoutPanic verifies the corresponding behavior.
func TestCollectRejectsEmptyPageSelectionWithoutPanic(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected error, got panic: %v", r)
		}
	}()

	err := Collect(openAPITestPDF(t, inFile), io.Discard, []string{""}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "collect: parse page selection") {
		t.Fatalf("expected collect page selection context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "page selection token 1: empty") {
		t.Fatalf("expected empty page selection token detail, got %q", err.Error())
	}
}

// TestCollectFileErrorsIncludePhaseContext verifies the corresponding behavior.
func TestCollectFileErrorsIncludePhaseContext(t *testing.T) {
	err := CollectFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), nil, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "collect: open input missing.pdf") {
		t.Fatalf("expected input open context, got %q", err.Error())
	}
}

// TestTrimReadErrorsIncludePhaseContext verifies the corresponding behavior.
func TestTrimReadErrorsIncludePhaseContext(t *testing.T) {
	err := Trim(bytes.NewReader(nil), io.Discard, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "trim: prepare PDF context") {
		t.Fatalf("expected trim prepare context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "read context") {
		t.Fatalf("expected read context, got %q", err.Error())
	}
}

// TestTrimPageSelectionErrorsIncludePhaseContext verifies the corresponding behavior.
func TestTrimPageSelectionErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")

	err := Trim(openAPITestPDF(t, inFile), io.Discard, []string{"foo"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "trim: parse page selection") {
		t.Fatalf("expected trim page selection context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "invalid syntax") {
		t.Fatalf("expected page selection detail, got %q", err.Error())
	}
}

// TestTrimWriteErrorIncludesPhaseContext verifies the corresponding behavior.
func TestTrimWriteErrorIncludesPhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	wantErr := errors.New("trim write failed")

	err := Trim(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, []string{"1"}, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "trim: write output") {
		t.Fatalf("expected write output context, got %q", err.Error())
	}
}

// TestTrimFileErrorsIncludePhaseContext verifies the corresponding behavior.
func TestTrimFileErrorsIncludePhaseContext(t *testing.T) {
	err := TrimFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), nil, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "trim: open input missing.pdf") {
		t.Fatalf("expected input open context, got %q", err.Error())
	}

	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	err = TrimFile(inFile, filepath.Join(t.TempDir(), "missing-dir", "out.pdf"), nil, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "trim: create output") {
		t.Fatalf("expected create output context, got %q", err.Error())
	}
}

// TestCloseTrimFilesErrorsIncludePhaseContext verifies the corresponding behavior.
func TestCloseTrimFilesErrorsIncludePhaseContext(t *testing.T) {
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
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "trim").commit()
			},
			wantErr: os.ErrClosed,
			want:    "trim: close output",
		},
		{
			name: "close input",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				if err := f1.Close(); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "trim").commit()
			},
			wantErr: os.ErrClosed,
			want:    "trim: close input",
		},
		{
			name: "replace input",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				tmpFile := f2.Name()
				if err := os.Remove(tmpFile); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, tmpFile, f1.Name(), "", "", "trim").commit()
			},
			wantErr: os.ErrNotExist,
			want:    "trim: replace input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(t)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestOptimizeWriteErrorIncludesPhaseContext verifies the corresponding behavior.
func TestOptimizeWriteErrorIncludesPhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	wantErr := errors.New("optimize write failed")

	err := Optimize(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "optimize: write output") {
		t.Fatalf("expected write output context, got %q", err.Error())
	}
}

// TestOptimizeReadErrorsIncludePhaseContext verifies the corresponding behavior.
func TestOptimizeReadErrorsIncludePhaseContext(t *testing.T) {
	err := Optimize(bytes.NewReader(nil), io.Discard, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "optimize: prepare PDF context") {
		t.Fatalf("expected prepare context, got %q", err.Error())
	}
}

// TestPublicOptimizeContextErrorsIncludeLowerPhaseContext verifies the corresponding behavior.
func TestPublicOptimizeContextErrorsIncludeLowerPhaseContext(t *testing.T) {
	ctx, err := pdfcpu.CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}

	ctx.PageCount = 1
	ctx.RootDict["Pages"] = *types.NewIndirectRef(999, 0)
	ctx.Optimize = &model.OptimizationContext{
		FontObjects:          map[int]*model.FontObject{},
		FormFontObjects:      map[int]*model.FontObject{},
		Fonts:                map[string][]int{},
		DuplicateFonts:       map[int]types.Dict{},
		DuplicateFontObjs:    types.IntSet{},
		ImageObjects:         map[int]*model.ImageObject{},
		DuplicateImages:      map[int]*model.DuplicateImageObject{},
		DuplicateImageObjs:   types.IntSet{},
		DuplicateInfoObjects: types.IntSet{},
		ContentStreamCache:   map[int]*types.StreamDict{},
		FormStreamCache:      map[int]*types.StreamDict{},
		Cache:                map[int]bool{},
	}
	ctx.Conf.OptimizeResourceDicts = false

	err = OptimizeContext(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "optimize context") {
		t.Fatalf("expected optimize context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "optimize fonts and images") {
		t.Fatalf("expected optimize phase context, got %q", err.Error())
	}
}

// TestOptimizeFileErrorsIncludePhaseContext verifies the corresponding behavior.
func TestOptimizeFileErrorsIncludePhaseContext(t *testing.T) {
	err := OptimizeFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "optimize: open input missing.pdf") {
		t.Fatalf("expected input open context, got %q", err.Error())
	}

	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	err = OptimizeFile(inFile, filepath.Join(t.TempDir(), "missing-dir", "out.pdf"), nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "optimize: create output") {
		t.Fatalf("expected create output context, got %q", err.Error())
	}
}

// TestCleanupOptimizeFilesErrorsIncludePhaseContext verifies the corresponding behavior.
func TestCleanupOptimizeFilesErrorsIncludePhaseContext(t *testing.T) {
	wantErr := errors.New("optimize failed")
	f1 := createTempAPITestFile(t)
	f2 := createTempAPITestFile(t)
	if err := f1.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f2.Close(); err != nil {
		t.Fatal(err)
	}

	err := newStagedOutput(f1, f2, f2.Name(), "", "", "", "optimize").cleanup(wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected closed file error, got %v", err)
	}
	for _, want := range []string{
		"optimize: close output",
		"optimize: close input",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

// TestCloseOptimizeFilesErrorsIncludePhaseContext verifies the corresponding behavior.
func TestCloseOptimizeFilesErrorsIncludePhaseContext(t *testing.T) {
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
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "optimize").commit()
			},
			wantErr: os.ErrClosed,
			want:    "optimize: close output",
		},
		{
			name: "close input",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				if err := f1.Close(); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, f2.Name(), f1.Name(), "out.pdf", "", "optimize").commit()
			},
			wantErr: os.ErrClosed,
			want:    "optimize: close input",
		},
		{
			name: "replace input",
			fn: func(t *testing.T) error {
				f1 := createTempAPITestFile(t)
				f2 := createTempAPITestFile(t)
				tmpFile := f2.Name()
				if err := os.Remove(tmpFile); err != nil {
					t.Fatal(err)
				}
				return newStagedOutput(f1, f2, tmpFile, f1.Name(), "", "", "optimize").commit()
			},
			wantErr: os.ErrNotExist,
			want:    "optimize: replace input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(t)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

func createTempAPITestFile(t *testing.T) *os.File {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "api-test-*")
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func pdf20Reader(t *testing.T, elems ...string) io.ReadSeeker {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(elems...))
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < len("%PDF-2.0") {
		t.Fatalf("invalid PDF fixture")
	}
	copy(b, "%PDF-2.0")

	return bytes.NewReader(b)
}

// TestMergeErrorsIncludeSourceContext verifies the corresponding behavior.
func TestMergeErrorsIncludeSourceContext(t *testing.T) {
	err := appendTo(nil, "1", nil, false)
	if !errors.Is(err, ErrMissingPDFReadSeeker) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFReadSeeker, err)
	}
	if !strings.Contains(err.Error(), "merge source 1: read source") {
		t.Fatalf("expected source context, got %q", err.Error())
	}

	err = MergeRaw([]io.ReadSeeker{bytes.NewReader(nil)}, io.Discard, false, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "merge source 0: read and validate") {
		t.Fatalf("expected raw source context, got %q", err.Error())
	}

	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	err = MergeRaw([]io.ReadSeeker{openAPITestPDF(t, inFile), bytes.NewReader(nil)}, io.Discard, false, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "merge source 1: read and validate") {
		t.Fatalf("expected raw source index context, got %q", err.Error())
	}

	badSource := filepath.Join(t.TempDir(), "bad-source.pdf")
	if err := os.WriteFile(badSource, nil, 0600); err != nil {
		t.Fatal(err)
	}
	err = Merge("", []string{inFile, badSource}, io.Discard, nil, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "merge source bad-source.pdf: read and validate") {
		t.Fatalf("expected file source basename context, got %q", err.Error())
	}
}

// TestMergeUnsupportedVersionPreservesSentinel verifies the corresponding behavior.
func TestMergeUnsupportedVersionPreservesSentinel(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")

	f := openAPITestPDF(t, inFile)
	err := MergeRaw([]io.ReadSeeker{f, pdf20Reader(t, inFile)}, io.Discard, false, nil)
	if !errors.Is(err, pdfcpu.ErrUnsupportedVersion) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrUnsupportedVersion, err)
	}
	if !strings.Contains(err.Error(), "merge source 1: validate version") {
		t.Fatalf("expected source version context, got %q", err.Error())
	}
}

// TestMergeZipUnsupportedVersionPreservesSentinel verifies the corresponding behavior.
func TestMergeZipUnsupportedVersionPreservesSentinel(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")

	tests := []struct {
		name string
		rs1  io.ReadSeeker
		rs2  io.ReadSeeker
		want string
	}{
		{
			name: "source 1",
			rs1:  pdf20Reader(t, inFile),
			rs2:  openAPITestPDF(t, inFile),
			want: "merge zip source 1: validate version",
		},
		{
			name: "source 2",
			rs1:  openAPITestPDF(t, inFile),
			rs2:  pdf20Reader(t, inFile),
			want: "merge zip source 2: validate version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MergeCreateZip(tt.rs1, tt.rs2, io.Discard, nil)
			if !errors.Is(err, pdfcpu.ErrUnsupportedVersion) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrUnsupportedVersion, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected version context, got %q", err.Error())
			}
		})
	}
}

// TestMergeFileErrorsIncludePhaseContext verifies the corresponding behavior.
func TestMergeFileErrorsIncludePhaseContext(t *testing.T) {
	err := Merge("missing.pdf", nil, io.Discard, nil, false)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "merge destination: open missing.pdf") {
		t.Fatalf("expected destination open context, got %q", err.Error())
	}

	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	err = Merge("", []string{inFile, "missing.pdf"}, io.Discard, nil, false)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "merge source: open missing.pdf") {
		t.Fatalf("expected source open context, got %q", err.Error())
	}
}

// TestMergeZipFileErrorsIncludePhaseContext verifies the corresponding behavior.
func TestMergeZipFileErrorsIncludePhaseContext(t *testing.T) {
	err := MergeCreateZipFile("missing.pdf", "also-missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "merge zip source 1: open missing.pdf") {
		t.Fatalf("expected zip source open context, got %q", err.Error())
	}

	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	err = MergeCreateZipFile(inFile, "also-missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "merge zip source 2: open also-missing.pdf") {
		t.Fatalf("expected zip source open context, got %q", err.Error())
	}

	outFile := filepath.Join(t.TempDir(), "missing-dir", "out.pdf")
	err = MergeCreateZipFile(inFile, inFile, outFile, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "merge zip: create output") {
		t.Fatalf("expected zip output context, got %q", err.Error())
	}
}

// TestMergeZipWriteErrorIncludesPhaseContext verifies the corresponding behavior.
func TestMergeZipWriteErrorIncludesPhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	wantErr := errors.New("zip write failed")

	err := MergeCreateZip(openAPITestPDF(t, inFile), openAPITestPDF(t, inFile), failingWriter{err: wantErr}, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "merge zip: write output") {
		t.Fatalf("expected zip write context, got %q", err.Error())
	}
}

// TestBookmarkFileWrapperErrorsIncludePhaseContext verifies the corresponding behavior.
func TestBookmarkFileWrapperErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "bookmarks", "bookmarkTree.pdf")
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "export open PDF",
			fn: func() error {
				return ExportBookmarksFile("missing.pdf", filepath.Join(t.TempDir(), "bookmarks.json"), nil)
			},
			want: "export bookmarks: open missing.pdf",
		},
		{
			name: "export create JSON",
			fn: func() error {
				return ExportBookmarksFile(inFile, filepath.Join(t.TempDir(), "missing-dir", "bookmarks.json"), nil)
			},
			want: "export bookmarks: create output",
		},
		{
			name: "import open JSON",
			fn: func() error {
				return ImportBookmarksFile(inFile, "missing.json", filepath.Join(t.TempDir(), "out.pdf"), false, nil)
			},
			want: "import bookmarks: open JSON missing.json",
		},
		{
			name: "add open PDF",
			fn: func() error {
				return AddBookmarksFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), []pdfcpu.Bookmark{{Title: "Root", PageFrom: 1}}, false, nil)
			},
			want: "add bookmarks: open missing.pdf",
		},
		{
			name: "remove open PDF",
			fn: func() error {
				return RemoveBookmarksFile("missing.pdf", filepath.Join(t.TempDir(), "out.pdf"), nil)
			},
			want: "remove bookmarks: open missing.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected not exist error, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestListBookmarksErrorsAndAbsentResult verifies the corresponding behavior.
func TestListBookmarksErrorsAndAbsentResult(t *testing.T) {
	if _, err := ListBookmarks(nil, nil); !errors.Is(err, ErrMissingPDFReadSeeker) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFReadSeeker, err)
	}

	_, err := ListBookmarks(bytes.NewReader(nil), nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "list bookmarks: prepare PDF context") {
		t.Fatalf("expected preparation context, got %q", err.Error())
	}

	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	ss, err := ListBookmarksFile(inFile, nil)
	if err != nil {
		t.Fatalf("list absent bookmarks: %v", err)
	}
	if len(ss) != 1 || ss[0] != "no bookmarks available" {
		t.Fatalf("unexpected absent-bookmark result: %v", ss)
	}
}

// TestListBookmarksFileArgumentAndOpenErrors verifies the corresponding behavior.
func TestListBookmarksFileArgumentAndOpenErrors(t *testing.T) {
	if _, err := ListBookmarksFile("", nil); !errors.Is(err, ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFInput, err)
	}
	_, err := ListBookmarksFile("missing.pdf", nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "list bookmarks: open input missing.pdf") {
		t.Fatalf("expected open context, got %q", err.Error())
	}
}

// TestCloseBookmarkInputPreservesPrimaryAndCloseErrors verifies the corresponding behavior.
func TestCloseBookmarkInputPreservesPrimaryAndCloseErrors(t *testing.T) {
	wantErr := errors.New("list failed")
	f := createTempAPITestFile(t)
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	err := closeBookmarkInput(wantErr, f, "list bookmarks: close input")
	if !errors.Is(err, wantErr) || !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected joined operation and close errors, got %v", err)
	}
	if !strings.Contains(err.Error(), "list bookmarks: close input") {
		t.Fatalf("expected close context, got %q", err.Error())
	}
}

// TestExportBookmarksFileRemovesOutputOnFailure verifies the corresponding behavior.
func TestExportBookmarksFileRemovesOutputOnFailure(t *testing.T) {
	inFile := filepath.Join("..", "samples", "bookmarks", "bookmarkTreeNoBookmarks.pdf")
	outFile := filepath.Join(t.TempDir(), "bookmarks.json")

	err := ExportBookmarksFile(inFile, outFile, nil)
	if !errors.Is(err, ErrNoBookmarks) {
		t.Fatalf("expected %v, got %v", ErrNoBookmarks, err)
	}
	if !strings.Contains(err.Error(), "export bookmarks") {
		t.Fatalf("expected export context, got %q", err.Error())
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got stat error %v", statErr)
	}
}

// TestExportBookmarksFileFailurePreservesExistingOutput verifies staged JSON publication.
func TestExportBookmarksFileFailurePreservesExistingOutput(t *testing.T) {
	inFile := filepath.Join("..", "samples", "bookmarks", "bookmarkTreeNoBookmarks.pdf")
	outFile := filepath.Join(t.TempDir(), "bookmarks.json")
	original := []byte(`{"existing":true}`)
	if err := os.WriteFile(outFile, original, 0640); err != nil {
		t.Fatal(err)
	}

	err := ExportBookmarksFile(inFile, outFile, nil)
	if !errors.Is(err, ErrNoBookmarks) {
		t.Fatalf("expected %v, got %v", ErrNoBookmarks, err)
	}
	bb, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(bb, original) {
		t.Fatalf("existing output changed: got %q, want %q", bb, original)
	}
}

// TestMergeFileCleanupPreservesPrimaryError verifies the corresponding behavior.
func TestMergeFileCleanupPreservesPrimaryError(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	err := MergeCreateFile(nil, outFile, false, nil)
	if !errors.Is(err, ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFInput, err)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got stat error %v", statErr)
	}

	appendFile := filepath.Join(t.TempDir(), "append.pdf")
	err = MergeAppendFile(nil, appendFile, false, nil)
	if !errors.Is(err, ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFInput, err)
	}
	if _, statErr := os.Stat(appendFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected append output cleanup, got stat error %v", statErr)
	}
}

// TestWriteContextFileFailurePreservesExistingOutput verifies staged direct PDF publication.
func TestWriteContextFileFailurePreservesExistingOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(outFile, original, 0640); err != nil {
		t.Fatal(err)
	}

	err := WriteContextFile(nil, outFile)
	if !errors.Is(err, ErrMissingPDFContext) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFContext, err)
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("existing output changed: got %q, want %q", got, original)
	}
}

// TestMergeCreateFileFailurePreservesExistingOutput verifies staged merge publication.
func TestMergeCreateFileFailurePreservesExistingOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(outFile, original, 0640); err != nil {
		t.Fatal(err)
	}

	err := MergeCreateFile(nil, outFile, false, nil)
	if !errors.Is(err, ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFInput, err)
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("existing output changed: got %q, want %q", got, original)
	}
}

// TestMergeZipFileRemovesOutputOnFailure verifies the corresponding behavior.
func TestMergeZipFileRemovesOutputOnFailure(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	badFile := filepath.Join(t.TempDir(), "bad.pdf")
	if err := os.WriteFile(badFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	err := MergeCreateZipFile(inFile, badFile, outFile, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got stat error %v", statErr)
	}
}

func openAPITestPDF(t *testing.T, elems ...string) *os.File {
	t.Helper()

	f, err := os.Open(filepath.Join(elems...))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	})
	return f
}

// TestSplitArgumentErrors verifies the corresponding behavior.
func TestSplitArgumentErrors(t *testing.T) {
	readerTests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "split raw missing reader",
			fn: func() error {
				_, err := SplitRaw(nil, 1, nil)
				return err
			},
		},
		{
			name: "split missing reader",
			fn: func() error {
				return Split(nil, t.TempDir(), "textAndAlignment.pdf", 1, nil)
			},
		},
		{
			name: "split by page number missing reader",
			fn: func() error {
				return SplitByPageNr(nil, t.TempDir(), "textAndAlignment.pdf", []int{2}, nil)
			},
		},
	}

	for _, tt := range readerTests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, ErrMissingPDFReadSeeker) {
				t.Fatalf("expected %v, got %v", ErrMissingPDFReadSeeker, err)
			}
		})
	}

	fileTests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "split file missing input",
			fn: func() error {
				return SplitFile("", t.TempDir(), 1, nil)
			},
		},
		{
			name: "split by page number file missing input",
			fn: func() error {
				return SplitByPageNrFile("", t.TempDir(), []int{2}, nil)
			},
		},
	}

	for _, tt := range fileTests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, ErrMissingPDFInput) {
				t.Fatalf("expected %v, got %v", ErrMissingPDFInput, err)
			}
		})
	}

	tests := []struct {
		name    string
		fn      func(*os.File) error
		wantErr error
	}{
		{
			name: "negative span",
			fn: func(f *os.File) error {
				return Split(f, t.TempDir(), "textAndAlignment.pdf", -1, nil)
			},
			wantErr: ErrInvalidSplitSpan,
		},
		{
			name: "missing page numbers",
			fn: func(f *os.File) error {
				return SplitByPageNr(f, t.TempDir(), "textAndAlignment.pdf", nil, nil)
			},
			wantErr: ErrMissingSplitPageNumbers,
		},
		{
			name: "invalid page number sequence",
			fn: func(f *os.File) error {
				return SplitByPageNr(f, t.TempDir(), "textAndAlignment.pdf", []int{9999}, nil)
			},
			wantErr: ErrInvalidSplitPageNumberSequence,
		},
		{
			name: "split page number below lower bound",
			fn: func(f *os.File) error {
				return SplitByPageNr(f, t.TempDir(), "textAndAlignment.pdf", []int{1}, nil)
			},
			wantErr: ErrInvalidSplitPageNumberSequence,
		},
		{
			name: "duplicate split page numbers",
			fn: func(f *os.File) error {
				return SplitByPageNr(f, t.TempDir(), "textAndAlignment.pdf", []int{2, 2}, nil)
			},
			wantErr: ErrInvalidSplitPageNumberSequence,
		},
		{
			name: "descending split page numbers",
			fn: func(f *os.File) error {
				return SplitByPageNr(f, t.TempDir(), "textAndAlignment.pdf", []int{3, 2}, nil)
			},
			wantErr: ErrInvalidSplitPageNumberSequence,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := openAPITestPDF(t, "..", "samples", "create", "primitives", "textAndAlignment.pdf")
			err := tt.fn(f)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

// TestSplitReadErrorsIncludeOperationContext verifies the corresponding behavior.
func TestSplitReadErrorsIncludeOperationContext(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "split raw",
			fn: func() error {
				_, err := SplitRaw(bytes.NewReader(nil), 1, nil)
				return err
			},
			want: "split: prepare PDF context",
		},
		{
			name: "split",
			fn: func() error {
				return Split(bytes.NewReader(nil), t.TempDir(), "out.pdf", 1, nil)
			},
			want: "split: prepare PDF context",
		},
		{
			name: "split by page number",
			fn: func() error {
				return SplitByPageNr(bytes.NewReader(nil), t.TempDir(), "out.pdf", []int{2}, nil)
			},
			want: "split by page number: prepare PDF context",
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
			if strings.Contains(err.Error(), "prepare PDF context: prepare PDF context") {
				t.Fatalf("unexpected duplicate prepare context in %q", err.Error())
			}
		})
	}
}

// TestSplitByBookmarkNoBookmarksError verifies the corresponding behavior.
func TestSplitByBookmarkNoBookmarksError(t *testing.T) {
	f := openAPITestPDF(t, "..", "samples", "create", "primitives", "textAndAlignment.pdf")

	_, err := SplitRaw(f, 0, nil)
	if !errors.Is(err, ErrNoBookmarks) {
		t.Fatalf("expected %v, got %v", ErrNoBookmarks, err)
	}
	if !strings.Contains(err.Error(), "split along bookmarks") {
		t.Fatalf("expected bookmark split context, got %q", err.Error())
	}
}

// TestAddBookmarksMapsExistingBookmarksError verifies the corresponding behavior.
func TestAddBookmarksMapsExistingBookmarksError(t *testing.T) {
	f := openAPITestPDF(t, "..", "samples", "bookmarks", "bookmarkTree.pdf")

	bms := []pdfcpu.Bookmark{{Title: "new bookmark", PageFrom: 1}}
	err := AddBookmarks(f, io.Discard, bms, false, nil)
	if !errors.Is(err, ErrExistingBookmarks) {
		t.Fatalf("expected %v, got %v", ErrExistingBookmarks, err)
	}
	if !strings.Contains(err.Error(), "add bookmarks") {
		t.Fatalf("expected operation context, got %q", err.Error())
	}
}

// TestBookmarkAPIErrorsAliasPdfcpuSentinels verifies the corresponding behavior.
func TestBookmarkAPIErrorsAliasPdfcpuSentinels(t *testing.T) {
	if ErrNoBookmarks != pdfcpu.ErrNoBookmarks {
		t.Fatal("ErrNoBookmarks is not an alias of pdfcpu.ErrNoBookmarks")
	}
	if ErrExistingBookmarks != pdfcpu.ErrExistingBookmarks {
		t.Fatal("ErrExistingBookmarks is not an alias of pdfcpu.ErrExistingBookmarks")
	}
	if ErrInvalidBookmarkJSON != pdfcpu.ErrInvalidBookmarkJSON {
		t.Fatal("ErrInvalidBookmarkJSON is not an alias of pdfcpu.ErrInvalidBookmarkJSON")
	}
}

// TestSignatureAPIErrorsAliasPdfcpuSentinels verifies the corresponding behavior.
func TestSignatureAPIErrorsAliasPdfcpuSentinels(t *testing.T) {
	if ErrNoSignatures != pdfcpu.ErrNoSignatures {
		t.Fatal("ErrNoSignatures is not an alias of pdfcpu.ErrNoSignatures")
	}
}

// TestContextAPIErrorsAliasPdfcpuSentinels verifies the corresponding behavior.
func TestContextAPIErrorsAliasPdfcpuSentinels(t *testing.T) {
	if ErrMissingPDFContext != pdfcpu.ErrMissingPDFContext {
		t.Fatal("ErrMissingPDFContext is not an alias of pdfcpu.ErrMissingPDFContext")
	}
	if pdfcpu.ErrMissingPDFContext != model.ErrMissingPDFContext {
		t.Fatal("pdfcpu.ErrMissingPDFContext is not an alias of model.ErrMissingPDFContext")
	}
	if ErrMissingXRefTable != pdfcpu.ErrMissingXRefTable {
		t.Fatal("ErrMissingXRefTable is not an alias of pdfcpu.ErrMissingXRefTable")
	}
	if pdfcpu.ErrMissingXRefTable != model.ErrMissingXRefTable {
		t.Fatal("pdfcpu.ErrMissingXRefTable is not an alias of model.ErrMissingXRefTable")
	}
}

// TestExportBookmarksNoBookmarksErrorIncludesSource verifies the corresponding behavior.
func TestExportBookmarksNoBookmarksErrorIncludesSource(t *testing.T) {
	f, err := os.Open("../samples/create/primitives/textAndAlignment.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = ExportBookmarksJSON(f, io.Discard, "textAndAlignment.pdf", nil)
	if !errors.Is(err, ErrNoBookmarks) {
		t.Fatalf("expected %v, got %v", ErrNoBookmarks, err)
	}
	if !strings.Contains(err.Error(), "export bookmarks: textAndAlignment.pdf") {
		t.Fatalf("expected source context, got %q", err.Error())
	}
}

// TestExtractMissingDigestFunctions verifies the corresponding behavior.
func TestExtractMissingDigestFunctions(t *testing.T) {
	tests := []struct {
		name string
		op   string
		fn   func() error
	}{
		{
			name: "images",
			op:   "extract images",
			fn: func() error {
				return ExtractImages(bytes.NewReader(nil), nil, nil, nil)
			},
		},
		{
			name: "fonts",
			op:   "extract fonts",
			fn: func() error {
				return ExtractFonts(bytes.NewReader(nil), nil, nil, nil)
			},
		},
		{
			name: "pages",
			op:   "extract pages",
			fn: func() error {
				return ExtractPages(bytes.NewReader(nil), nil, nil, nil)
			},
		},
		{
			name: "content",
			op:   "extract content",
			fn: func() error {
				return ExtractContent(bytes.NewReader(nil), nil, nil, nil)
			},
		},
		{
			name: "metadata",
			op:   "extract metadata",
			fn: func() error {
				return ExtractMetadata(bytes.NewReader(nil), nil, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, ErrMissingDigestFunction) {
				t.Fatalf("expected %v, got %v", ErrMissingDigestFunction, err)
			}
			if !strings.Contains(err.Error(), tt.op) {
				t.Fatalf("expected %q in error, got %q", tt.op, err.Error())
			}
		})
	}
}

// TestExtractReadErrorsIncludePhaseContext verifies the corresponding behavior.
func TestExtractReadErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name string
		op   string
		fn   func() error
	}{
		{
			name: "images raw",
			op:   "extract images",
			fn: func() error {
				_, err := ExtractImagesRaw(bytes.NewReader(nil), nil, nil)
				return err
			},
		},
		{
			name: "images",
			op:   "extract images",
			fn: func() error {
				return ExtractImages(bytes.NewReader(nil), nil, func(model.Image, bool, int) error { return nil }, nil)
			},
		},
		{
			name: "fonts",
			op:   "extract fonts",
			fn: func() error {
				return ExtractFonts(bytes.NewReader(nil), nil, func(pdfcpu.Font) error { return nil }, nil)
			},
		},
		{
			name: "pages",
			op:   "extract pages",
			fn: func() error {
				return ExtractPages(bytes.NewReader(nil), nil, func(io.Reader, int) error { return nil }, nil)
			},
		},
		{
			name: "content",
			op:   "extract content",
			fn: func() error {
				return ExtractContent(bytes.NewReader(nil), nil, func(io.Reader, int) error { return nil }, nil)
			},
		},
		{
			name: "metadata",
			op:   "extract metadata",
			fn: func() error {
				return ExtractMetadata(bytes.NewReader(nil), func(pdfcpu.Metadata) error { return nil }, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if !strings.Contains(err.Error(), tt.op+": prepare PDF context") {
				t.Fatalf("expected %s prepare context, got %q", tt.op, err.Error())
			}
			if !strings.Contains(err.Error(), "read context") {
				t.Fatalf("expected read context, got %q", err.Error())
			}
			if strings.Contains(err.Error(), "prepare PDF context: prepare PDF context") {
				t.Fatalf("unexpected duplicate prepare context, got %q", err.Error())
			}
		})
	}
}

// TestExtractPageSelectionErrorsIncludePhaseContext verifies the corresponding behavior.
func TestExtractPageSelectionErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name string
		op   string
		fn   func(io.ReadSeeker) error
	}{
		{
			name: "images raw",
			op:   "extract images",
			fn: func(rs io.ReadSeeker) error {
				_, err := ExtractImagesRaw(rs, []string{"foo"}, nil)
				return err
			},
		},
		{
			name: "images",
			op:   "extract images",
			fn: func(rs io.ReadSeeker) error {
				return ExtractImages(rs, []string{"foo"}, func(model.Image, bool, int) error { return nil }, nil)
			},
		},
		{
			name: "fonts",
			op:   "extract fonts",
			fn: func(rs io.ReadSeeker) error {
				return ExtractFonts(rs, []string{"foo"}, func(pdfcpu.Font) error { return nil }, nil)
			},
		},
		{
			name: "pages",
			op:   "extract pages",
			fn: func(rs io.ReadSeeker) error {
				return ExtractPages(rs, []string{"foo"}, func(io.Reader, int) error { return nil }, nil)
			},
		},
		{
			name: "content",
			op:   "extract content",
			fn: func(rs io.ReadSeeker) error {
				return ExtractContent(rs, []string{"foo"}, func(io.Reader, int) error { return nil }, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(openAPITestPDF(t, inFile))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.op+": parse page selection") {
				t.Fatalf("expected %s page selection context, got %q", tt.op, err.Error())
			}
			if !strings.Contains(err.Error(), "invalid syntax") {
				t.Fatalf("expected page selection detail, got %q", err.Error())
			}
		})
	}
}

// TestExtractDigestErrorsIncludePhaseContext verifies the corresponding behavior.
func TestExtractDigestErrorsIncludePhaseContext(t *testing.T) {
	wantErr := errors.New("digest failed")
	tests := []struct {
		name  string
		wants []string
		fn    func() error
	}{
		{
			name: "images",
			wants: []string{
				"extract images: page 1 image obj#",
				"digest",
			},
			fn: func() error {
				inFile := filepath.Join("..", "testdata", "testImage.pdf")
				return ExtractImages(openAPITestPDF(t, inFile), []string{"1"}, func(model.Image, bool, int) error { return wantErr }, nil)
			},
		},
		{
			name: "fonts",
			wants: []string{
				"extract fonts: page ",
				"font \"",
				"obj#",
				"digest",
			},
			fn: func() error {
				inFile := filepath.Join("..", "testdata", "TheGoProgrammingLanguageCh1.pdf")
				return ExtractFonts(openAPITestPDF(t, inFile), nil, func(pdfcpu.Font) error { return wantErr }, nil)
			},
		},
		{
			name: "pages",
			wants: []string{
				"extract pages: page 1: digest",
			},
			fn: func() error {
				inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
				return ExtractPages(openAPITestPDF(t, inFile), []string{"1"}, func(io.Reader, int) error { return wantErr }, nil)
			},
		},
		{
			name: "content",
			wants: []string{
				"extract content: page 1: digest content",
			},
			fn: func() error {
				inFile := filepath.Join("..", "testdata", "5116.DCT_Filter.pdf")
				return ExtractContent(openAPITestPDF(t, inFile), []string{"1"}, func(io.Reader, int) error { return wantErr }, nil)
			},
		},
		{
			name: "metadata",
			wants: []string{
				"extract metadata: parent obj#",
				"metadata obj#",
				"digest",
			},
			fn: func() error {
				inFile := filepath.Join("..", "testdata", "TheGoProgrammingLanguageCh1.pdf")
				return ExtractMetadata(openAPITestPDF(t, inFile), func(pdfcpu.Metadata) error { return wantErr }, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			for _, want := range tt.wants {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected %q in error, got %q", want, err.Error())
				}
			}
		})
	}
}

// TestExtractImagesAllPagesExcludedDoesNotPanic verifies the corresponding behavior.
func TestExtractImagesAllPagesExcludedDoesNotPanic(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "testImage.pdf")
	err := ExtractImages(
		openAPITestPDF(t, inFile),
		[]string{"!1-"},
		func(model.Image, bool, int) error { return nil },
		nil,
	)
	if err != nil {
		t.Fatalf("exclude all pages: %v", err)
	}
}

// TestExtractFileInputErrorsIncludePhaseContext verifies the corresponding behavior.
func TestExtractFileInputErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name string
		op   string
		fn   func(string) error
	}{
		{name: "images", op: "extract images", fn: func(inFile string) error { return ExtractImagesFile(inFile, t.TempDir(), nil, nil) }},
		{name: "fonts", op: "extract fonts", fn: func(inFile string) error { return ExtractFontsFile(inFile, t.TempDir(), nil, nil) }},
		{name: "pages", op: "extract pages", fn: func(inFile string) error { return ExtractPagesFile(inFile, t.TempDir(), nil, nil) }},
		{name: "content", op: "extract content", fn: func(inFile string) error { return ExtractContentFile(inFile, t.TempDir(), nil, nil) }},
		{name: "metadata", op: "extract metadata", fn: func(inFile string) error { return ExtractMetadataFile(inFile, t.TempDir(), nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.name+" empty", func(t *testing.T) {
			if err := tt.fn(""); !errors.Is(err, ErrMissingPDFInput) {
				t.Fatalf("expected %v, got %v", ErrMissingPDFInput, err)
			}
		})
		t.Run(tt.name+" missing", func(t *testing.T) {
			err := tt.fn("missing.pdf")
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected not exist error, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.op+": open input missing.pdf") {
				t.Fatalf("expected %s open context, got %q", tt.op, err.Error())
			}
		})
	}
}

// TestExtractFileWriteErrorsIncludePhaseContext verifies the corresponding behavior.
func TestExtractFileWriteErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name  string
		wants []string
		fn    func(string) error
	}{
		{
			name: "images",
			wants: []string{
				"extract images: page 1 image obj#",
				"digest",
			},
			fn: func(outDir string) error {
				inFile := filepath.Join("..", "testdata", "testImage.pdf")
				return ExtractImagesFile(inFile, outDir, []string{"1"}, nil)
			},
		},
		{
			name: "fonts",
			wants: []string{
				"font \"",
				"obj#",
				"digest",
			},
			fn: func(outDir string) error {
				inFile := filepath.Join("..", "testdata", "TheGoProgrammingLanguageCh1.pdf")
				return ExtractFontsFile(inFile, outDir, nil, nil)
			},
		},
		{
			name: "pages",
			wants: []string{
				"extract pages: page 1: digest",
			},
			fn: func(outDir string) error {
				inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
				return ExtractPagesFile(inFile, outDir, []string{"1"}, nil)
			},
		},
		{
			name: "content",
			wants: []string{
				"extract content: page 1: digest content",
			},
			fn: func(outDir string) error {
				inFile := filepath.Join("..", "testdata", "5116.DCT_Filter.pdf")
				return ExtractContentFile(inFile, outDir, []string{"1"}, nil)
			},
		},
		{
			name: "metadata",
			wants: []string{
				"extract metadata: parent obj#",
				"metadata obj#",
				"digest",
			},
			fn: func(outDir string) error {
				inFile := filepath.Join("..", "testdata", "TheGoProgrammingLanguageCh1.pdf")
				return ExtractMetadataFile(inFile, outDir, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outDir := filepath.Join(t.TempDir(), "missing")
			err := tt.fn(outDir)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected not exist error, got %v", err)
			}
			for _, want := range tt.wants {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected %q in error, got %q", want, err.Error())
				}
			}
		})
	}
}

// TestExtractCloseFileErrorsIncludePhaseContext verifies the corresponding behavior.
func TestExtractCloseFileErrorsIncludePhaseContext(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "extract-close-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	err = closeFile(f, "extract pages: close input")
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected %v, got %v", os.ErrClosed, err)
	}
	if !strings.Contains(err.Error(), "extract pages: close input") {
		t.Fatalf("expected close input context, got %q", err.Error())
	}
}

// TestValidateFilesReturnsJoinedErrors verifies the corresponding behavior.
func TestValidateFilesReturnsJoinedErrors(t *testing.T) {
	stderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
		r.Close()
	}()

	inFiles := []string{"missing1.pdf", "missing2.pdf"}
	err = ValidateFiles(inFiles, nil)
	w.Close()

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	for _, fn := range inFiles {
		if !strings.Contains(err.Error(), fn) {
			t.Fatalf("expected %q in error, got %q", fn, err.Error())
		}
	}

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	if buf.Len() > 0 {
		t.Fatalf("expected no stderr output, got %q", buf.String())
	}
}

// TestValidateFileMissingFilePreservesNotExist verifies the corresponding behavior.
func TestValidateFileMissingFilePreservesNotExist(t *testing.T) {
	err := ValidateFile("missing.pdf", nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), "validate: open missing.pdf") {
		t.Fatalf("expected validate open context, got %q", err.Error())
	}
}

// TestValidateFileEmptyInputPreservesSentinel verifies the corresponding behavior.
func TestValidateFileEmptyInputPreservesSentinel(t *testing.T) {
	err := ValidateFile("", nil)
	if !errors.Is(err, ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFInput, err)
	}
}

// TestValidateFilesEmptyInputIsNoop verifies the corresponding behavior.
func TestValidateFilesEmptyInputIsNoop(t *testing.T) {
	if err := ValidateFiles(nil, nil); err != nil {
		t.Fatalf("expected nil error for nil input list, got %v", err)
	}
	if err := ValidateFiles([]string{}, nil); err != nil {
		t.Fatalf("expected nil error for empty input list, got %v", err)
	}
}
