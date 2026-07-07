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

package model

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestNUpStringAllowsDerivedPageDimension(t *testing.T) {
	nup := DefaultNUpConfig()
	nup.Grid = &types.Dim{Width: 2, Height: 2}
	if got := nup.String(); !strings.Contains(got, "A4 <nil>") {
		t.Fatalf("expected unresolved page dimensions, got %q", got)
	}
}

func TestNUpTilePDFBytesForPDFPreservesPageNotFound(t *testing.T) {
	ctx, err := NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	err = ctx.NUpTilePDFBytesForPDF(1, types.NewDict(), &bytes.Buffer{}, types.RectForDim(10, 10), &NUp{}, false)
	if !errors.Is(err, ErrPageNotFound) {
		t.Fatalf("expected %v, got %v", ErrPageNotFound, err)
	}
	if !strings.Contains(err.Error(), "n-up source page 1: resolve page dictionary") {
		t.Fatalf("expected source page context, got %q", err.Error())
	}
}

func TestCreateNUpFormForPDFErrorIncludesFormContext(t *testing.T) {
	ctx, err := NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Table[0] = NewFreeHeadXRefTableEntry()
	missingObjectOffset := int64(999)
	ctx.Table[0].Offset = &missingObjectOffset

	_, err = createNUpFormForPDF(ctx.XRefTable, types.NewIndirectRef(1, 0), nil, types.RectForDim(10, 10))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "n-up PDF form: store form stream") {
		t.Fatalf("expected PDF form context, got %q", err.Error())
	}
}
