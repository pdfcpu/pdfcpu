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

package pdfcpu

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"strconv"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestParseImportDetailsReportsClauseContext verifies malformed clauses identify their position.
func TestParseImportDetailsReportsClauseContext(t *testing.T) {
	_, err := ParseImportDetails("dpi:72,broken", types.POINTS)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "import configuration clause 2") {
		t.Fatalf("expected clause context, got %q", err)
	}
}

// TestParseImportDetailsReportsParameterContext verifies parameter errors identify the resolved name.
func TestParseImportDetailsReportsParameterContext(t *testing.T) {
	_, err := ParseImportDetails("offset:x 1", types.POINTS)
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Fatalf("expected strconv.NumError, got %v", err)
	}
	if !strings.Contains(err.Error(), `import configuration clause 1 parameter "offset"`) {
		t.Fatalf("expected parameter context, got %q", err)
	}
}

// TestParseImportDetailsPreservesDPINumericCause verifies DPI parsing keeps strconv causes discoverable.
func TestParseImportDetailsPreservesDPINumericCause(t *testing.T) {
	_, err := ParseImportDetails("dpi:nope", types.POINTS)
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Fatalf("expected strconv.NumError, got %v", err)
	}
	if !strings.Contains(err.Error(), `import configuration clause 1 parameter "dpi"`) {
		t.Fatalf("expected DPI parameter context, got %q", err)
	}
}

// TestParseImportDetailsPreservesScaleNumericCause verifies shared scale parsing wraps strconv errors.
func TestParseImportDetailsPreservesScaleNumericCause(t *testing.T) {
	_, err := ParseImportDetails("scalefactor:nope", types.POINTS)
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Fatalf("expected strconv.NumError, got %v", err)
	}
	if !strings.Contains(err.Error(), `import configuration clause 1 parameter "scalefactor"`) {
		t.Fatalf("expected scale-factor parameter context, got %q", err)
	}
}

func newImportImageContext(t *testing.T) (*model.Context, *types.IndirectRef) {
	t.Helper()
	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	pagesIndRef, err := ctx.Pages()
	if err != nil {
		t.Fatal(err)
	}
	return ctx, pagesIndRef
}

func importImageConstructionError(
	t *testing.T,
	xRefTable *model.XRefTable,
	parentIndRef *types.IndirectRef,
	imp *Import,
) (err error) {
	t.Helper()
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("panic: %v", v)
		}
	}()
	_, err = NewPagesForImage(
		xRefTable,
		bytes.NewReader(imageOperationPNG(t, 1, 1)),
		parentIndRef,
		imp,
	)
	return err
}

// TestNewPagesForImageRejectsMissingConstructionState verifies boundary nils return errors instead of panicking.
func TestNewPagesForImageRejectsMissingConstructionState(t *testing.T) {
	ctx, pagesIndRef := newImportImageContext(t)
	tests := []struct {
		name    string
		xRef    *model.XRefTable
		parent  *types.IndirectRef
		imp     *Import
		wantErr error
		want    string
	}{
		{
			name:    "xref table",
			parent:  pagesIndRef,
			imp:     DefaultImportConfig(),
			wantErr: model.ErrMissingXRefTable,
			want:    "create image resources",
		},
		{
			name: "parent reference",
			xRef: ctx.XRefTable,
			imp:  DefaultImportConfig(),
			want: "missing page tree parent",
		},
		{
			name:   "configuration",
			xRef:   ctx.XRefTable,
			parent: pagesIndRef,
			want:   "missing import configuration",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := importImageConstructionError(t, tt.xRef, tt.parent, tt.imp)
			if err == nil {
				t.Fatal("expected error")
			}
			if strings.Contains(err.Error(), "panic:") {
				t.Fatalf("unexpected panic: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, err)
			}
		})
	}
}

// TestNewPagesForImageReportsImageResourcePhase verifies construction failures identify resource creation.
func TestNewPagesForImageReportsImageResourcePhase(t *testing.T) {
	ctx, pagesIndRef := newImportImageContext(t)
	missingObjectOffset := int64(999)
	ctx.XRefTable.Table[0].Offset = &missingObjectOffset

	_, err := NewPagesForImage(
		ctx.XRefTable,
		bytes.NewReader(imageOperationPNG(t, 1, 1)),
		pagesIndRef,
		DefaultImportConfig(),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create image resources") {
		t.Fatalf("expected image-resource phase, got %q", err)
	}
}

// TestNewPagesForImageDecodeErrorContext verifies image decoder sentinels survive operation wrapping.
func TestNewPagesForImageDecodeErrorContext(t *testing.T) {
	ctx, pagesIndRef := newImportImageContext(t)
	_, err := NewPagesForImage(
		ctx.XRefTable,
		bytes.NewReader(nil),
		pagesIndRef,
		DefaultImportConfig(),
	)
	if !errors.Is(err, image.ErrFormat) {
		t.Fatalf("expected %v, got %v", image.ErrFormat, err)
	}
	for _, want := range []string{"create image resources", "decode image configuration"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
}
