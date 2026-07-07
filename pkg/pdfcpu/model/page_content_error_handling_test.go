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
	"compress/zlib"
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func corruptZlibContent(t *testing.T, content []byte) []byte {
	t.Helper()

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	if _, err := w.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	bb := b.Bytes()
	bb[len(bb)-1] ^= 0xff
	return bb
}

// TestPageContentPreservesUnsupportedFilter verifies the corresponding behavior.
func TestPageContentPreservesUnsupportedFilter(t *testing.T) {
	xRefTable := &XRefTable{ValidationMode: ValidationStrict}
	sd := types.StreamDict{
		Raw: []byte("content"),
		FilterPipeline: []types.PDFFilter{
			{Name: filter.JBIG2},
			{Name: filter.Flate},
		},
	}
	d := types.Dict{"Contents": sd}

	_, err := xRefTable.PageContent(d, 3)
	if !errors.Is(err, filter.ErrUnsupportedFilter) {
		t.Fatalf("expected %v, got %v", filter.ErrUnsupportedFilter, err)
	}
	if !strings.Contains(err.Error(), "page 3 content decode") {
		t.Fatalf("expected page content context, got %q", err.Error())
	}
}

// TestPageContentArrayPreservesUnsupportedFilter verifies the corresponding behavior.
func TestPageContentArrayPreservesUnsupportedFilter(t *testing.T) {
	sd := types.StreamDict{
		Raw: []byte("content"),
		FilterPipeline: []types.PDFFilter{
			{Name: filter.JBIG2},
			{Name: filter.Flate},
		},
	}
	xRefTable := &XRefTable{
		Table: map[int]*XRefTableEntry{
			7: NewXRefTableEntryGen0(sd),
		},
	}
	d := types.Dict{
		"Contents": types.Array{*types.NewIndirectRef(7, 0)},
	}

	_, err := xRefTable.PageContent(d, 4)
	if !errors.Is(err, filter.ErrUnsupportedFilter) {
		t.Fatalf("expected %v, got %v", filter.ErrUnsupportedFilter, err)
	}
	if !strings.Contains(err.Error(), "page 4 content decode") {
		t.Fatalf("expected page content context, got %q", err.Error())
	}
}

// TestPageContentPreservesFlateChecksumError verifies checksum failures receive page context without losing their sentinel.
func TestPageContentPreservesFlateChecksumError(t *testing.T) {
	sd := types.StreamDict{
		Raw:            corruptZlibContent(t, []byte("content")),
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate}},
	}
	d := types.Dict{"Contents": sd}

	_, err := (&XRefTable{}).PageContent(d, 5)
	if !errors.Is(err, zlib.ErrChecksum) {
		t.Fatalf("expected %v, got %v", zlib.ErrChecksum, err)
	}
	if !strings.Contains(err.Error(), "page 5 content decode") {
		t.Fatalf("expected page content context, got %q", err.Error())
	}
}

// TestConsolidateResourcesRelaxesFlateChecksumError verifies relaxed resource analysis tolerates invalid Flate checksums.
func TestConsolidateResourcesRelaxesFlateChecksumError(t *testing.T) {
	sd := types.StreamDict{
		Raw:            corruptZlibContent(t, []byte("content")),
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate}},
	}
	d := types.Dict{"Contents": sd}

	xRefTable := &XRefTable{ValidationMode: ValidationRelaxed}
	if err := xRefTable.consolidateResourcesWithContent(d, types.Dict{}, 5, true); err != nil {
		t.Fatalf("expected relaxed checksum handling, got %v", err)
	}
}
