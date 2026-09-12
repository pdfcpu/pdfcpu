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
	"bufio"
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func directInfoTrailerXRefTable(mode int, version model.Version) *model.XRefTable {
	size := 3
	return &model.XRefTable{
		Table:          map[int]*model.XRefTableEntry{},
		Size:           &size,
		HeaderVersion:  &version,
		ValidationMode: mode,
	}
}

func directInfoTrailer() types.Dict {
	return types.Dict{
		"Info": types.Dict{
			"Producer":     types.HexLiteral("FEFF88FD54C1540D"),
			"CreationDate": types.StringLiteral("D:20240102030405Z"),
			"ModDate":      types.StringLiteral("pipeline-controlled-date"),
		},
	}
}

func assertDirectInfoMaterialized(t *testing.T, xRefTable *model.XRefTable, want types.Dict) {
	t.Helper()
	if xRefTable.Info == nil {
		t.Fatal("missing materialized Info reference")
	}
	if got := xRefTable.Info.ObjectNumber.Value(); got != 3 {
		t.Fatalf("Info object number = %d, want 3", got)
	}
	d, err := xRefTable.DereferenceDict(*xRefTable.Info)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d, want) {
		t.Fatalf("Info dictionary:\n got: %#v\nwant: %#v", d, want)
	}
	if *xRefTable.Size != 4 || xRefTable.MaxObjNr != 3 {
		t.Fatalf("Size/MaxObjNr = %d/%d, want 4/3", *xRefTable.Size, xRefTable.MaxObjNr)
	}
}

// TestParseTrailerInfoMaterializesDirectPDF20Dictionary verifies both validation modes retain direct PDF 2.0 Info.
func TestParseTrailerInfoMaterializesDirectPDF20Dictionary(t *testing.T) {
	for _, tt := range []struct {
		name string
		mode int
	}{
		{name: "strict", mode: model.ValidationStrict},
		{name: "relaxed", mode: model.ValidationRelaxed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			xRefTable := directInfoTrailerXRefTable(tt.mode, model.V20)
			trailer := directInfoTrailer()
			before := trailer.Clone()

			if err := parseTrailerInfo(xRefTable, trailer); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(trailer, before) {
				t.Fatalf("trailer changed:\n got: %#v\nwant: %#v", trailer, before)
			}
			if xRefTable.Info != nil || !reflect.DeepEqual(xRefTable.DirectInfoDict, trailer["Info"]) {
				t.Fatalf("pending direct Info = %#v, Info = %v", xRefTable.DirectInfoDict, xRefTable.Info)
			}
			if err := materializeDirectInfoDict(xRefTable); err != nil {
				t.Fatal(err)
			}
			assertDirectInfoMaterialized(t, xRefTable, trailer["Info"].(types.Dict))
		})
	}
}

// TestParseTrailerInfoLeavesDirectPDF17DictionaryUnsupported preserves the pre-PDF-2.0 reader behavior.
func TestParseTrailerInfoLeavesDirectPDF17DictionaryUnsupported(t *testing.T) {
	xRefTable := directInfoTrailerXRefTable(model.ValidationRelaxed, model.V17)
	trailer := directInfoTrailer()

	if err := parseTrailerInfo(xRefTable, trailer); err != nil {
		t.Fatal(err)
	}
	if err := materializeDirectInfoDict(xRefTable); err != nil {
		t.Fatal(err)
	}
	if xRefTable.Info != nil {
		t.Fatalf("Info = %v, want nil", xRefTable.Info)
	}
	if xRefTable.DirectInfoDict != nil || *xRefTable.Size != 3 || len(xRefTable.Table) != 0 {
		t.Fatalf("Size/table length = %d/%d, want 3/0", *xRefTable.Size, len(xRefTable.Table))
	}
}

func directInfoPDF20(headerVersion string, rootVersion bool) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%%PDF-%s\n", headerVersion)
	offsets := make([]int, 3)
	offsets[1] = buf.Len()
	buf.WriteString("1 0 obj\n<</Type/Catalog/Pages 2 0 R")
	if rootVersion {
		buf.WriteString("/Version/2.0")
	}
	buf.WriteString(">>\nendobj\n")
	offsets[2] = buf.Len()
	buf.WriteString("2 0 obj\n<</Type/Pages/Count 0/Kids[]>>\nendobj\n")
	xrefOffset := buf.Len()
	buf.WriteString("xref\n0 3\n0000000000 65535 f \n")
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	buf.WriteString("trailer\n")
	buf.WriteString("<</Size 3/Root 1 0 R/Info<</Producer<FEFF88FD54C1540D>")
	buf.WriteString("/CreationDate(D:20240102030405Z)/ModDate(pipeline-controlled-date)>>>>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOffset)
	return buf.Bytes()
}

func assertReadDirectInfo(t *testing.T, ctx *model.Context) {
	t.Helper()
	want := directInfoTrailer()["Info"].(types.Dict)
	assertDirectInfoMaterialized(t, ctx.XRefTable, want)
}

// TestReadAndRewriteDirectPDF20Info verifies direct Info survives the complete reader and writer path.
func TestReadAndRewriteDirectPDF20Info(t *testing.T) {
	for _, tt := range []struct {
		name          string
		mode          int
		headerVersion string
		rootVersion   bool
	}{
		{name: "strict", mode: model.ValidationStrict, headerVersion: "2.0"},
		{name: "relaxed", mode: model.ValidationRelaxed, headerVersion: "2.0"},
		{name: "catalog version override", mode: model.ValidationStrict, headerVersion: "1.7", rootVersion: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			conf := model.NewDefaultConfiguration()
			conf.ValidationMode = tt.mode
			conf.PreserveInfoDict = true
			conf.WriteObjectStream = false
			conf.WriteXRefStream = false
			ctx, err := Read(t.Context(), bytes.NewReader(directInfoPDF20(tt.headerVersion, tt.rootVersion)), conf)
			if err != nil {
				t.Fatal(err)
			}
			assertReadDirectInfo(t, ctx)

			var output bytes.Buffer
			ctx.Write.Writer = bufio.NewWriter(&output)
			if err := WriteContext(t.Context(), ctx); err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(output.Bytes(), []byte("/Producer<FEFF88FD54C1540D>")) {
				t.Fatal("rewritten output does not preserve Producer hex encoding")
			}
			rewritten, err := Read(t.Context(), bytes.NewReader(output.Bytes()), conf)
			if err != nil {
				t.Fatal(err)
			}
			assertReadDirectInfo(t, rewritten)
		})
	}
}
