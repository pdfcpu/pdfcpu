/*
Copyright 2024 The pdfcpu Authors.

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
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type headerErrorReader struct {
	seekErr error
	readErr error
}

func (r headerErrorReader) Read(_ []byte) (int, error) {
	return 0, r.readErr
}

func (r headerErrorReader) Seek(_ int64, _ int) (int64, error) {
	return 0, r.seekErr
}

func intPtr(i int) *int {
	return &i
}

type wrappedEOFReader struct {
	data []byte
}

func (r *wrappedEOFReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, fmt.Errorf("wrapped eof: %w", io.EOF)
}

// TestReadFileContext verifies context-aware file reading.
func TestReadFileContext(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "test.pdf")

	ctx, cancel := context.WithTimeout(t.Context(), 0)
	defer cancel()

	if doc, err := ReadFile(ctx, inFile, nil); err == nil {
		t.Errorf("reading should have failed, got %v", doc)
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("should have failed with timeout, got %s", err)
	}
}

// TestReadContext verifies context-aware reading.
func TestReadContext(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "test.pdf")

	fp, err := os.Open(inFile)
	if err != nil {
		t.Fatal(err)
	}
	defer fp.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 0)
	defer cancel()

	if doc, err := Read(ctx, fp, nil); err == nil {
		t.Errorf("reading should have failed, got %v", doc)
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("should have failed with timeout, got %s", err)
	}
}

func TestReadClassifiesEmptyInput(t *testing.T) {
	_, err := Read(t.Context(), bytes.NewReader(nil), nil)
	if !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("got %v, want ErrEmptyInput", err)
	}
}

func TestReadClassifiesTruncatedStartXRef(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationStrict
	_, err := Read(t.Context(), bytes.NewReader([]byte("%PDF-1.7\nstartxref")), conf)
	if !errors.Is(err, errMissingXRefEOF) {
		t.Fatalf("got %v, want errMissingXRefEOF", err)
	}
}

func TestReadRepairsTruncatedStartXRefInRelaxedMode(t *testing.T) {
	bb, err := os.ReadFile(filepath.Join("..", "testdata", "test.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	i := bytes.LastIndex(bb, []byte("startxref"))
	if i < 0 {
		t.Fatal("missing startxref")
	}
	bb = bb[:i+len("startxref")]

	if _, err := Read(t.Context(), bytes.NewReader(bb), nil); err != nil {
		t.Fatalf("relaxed read: %v", err)
	}

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationStrict
	if _, err := Read(t.Context(), bytes.NewReader(bb), conf); !errors.Is(err, errMissingXRefEOF) {
		t.Fatalf("strict read: got %v, want errMissingXRefEOF", err)
	}
}

func TestReadMissingReaderReturnsError(t *testing.T) {
	_, err := Read(t.Context(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMissingXRefSectionClassificationAcceptsWrappedSentinel(t *testing.T) {
	err := fmt.Errorf("scan trailer: %w", ErrMissingXRefSection)
	if !isMissingXRefSection(err) {
		t.Fatalf("expected wrapped %v to classify as missing xref section", ErrMissingXRefSection)
	}
	if isMissingXRefSection(errors.New("not xref")) {
		t.Fatal("unexpected missing xref section classification")
	}
}

func TestOffsetLastXRefSectionClassifiesEpilogueErrors(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr error
	}{
		{
			name:    "missing EOF",
			in:      "%PDF-1.7\nstartxref\n12\n",
			wantErr: errMissingXRefEOF,
		},
		{
			name:    "truncated after startxref",
			in:      "%PDF-1.7\nstartxref",
			wantErr: errMissingXRefEOF,
		},
		{
			name:    "invalid offset",
			in:      "%PDF-1.7\nstartxref\nnot-an-offset\n%%EOF",
			wantErr: errInvalidLastXRefSection,
		},
		{
			name:    "missing startxref",
			in:      "%PDF-1.7\n%%EOF",
			wantErr: ErrMissingXRefSection,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := model.NewContext(bytes.NewReader([]byte(tt.in)), nil)
			if err != nil {
				t.Fatal(err)
			}

			_, err = offsetLastXRefSection(ctx, 0)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestOffsetLastXRefSectionPreservesReadError(t *testing.T) {
	readErr := errors.New("read failed")
	ctx := &model.Context{
		Read: &model.ReadContext{
			RS:       headerErrorReader{readErr: readErr},
			FileSize: 10,
		},
	}

	_, err := offsetLastXRefSection(ctx, 0)
	if !errors.Is(err, readErr) {
		t.Fatalf("expected %v, got %v", readErr, err)
	}
	if !strings.Contains(err.Error(), "scan for startxref") {
		t.Fatalf("expected startxref scan context, got %q", err.Error())
	}
}

// TestProcessObjectSkipsBoundedMalformedObjectRelaxed checks the offset and state after a relaxed skip.
func TestProcessObjectSkipsBoundedMalformedObjectRelaxed(t *testing.T) {
	input := "1 0 obj\n<</D[0/a0 obj\n<</E 1>>\nendobj\n2 0 obj\nnull\nendobj\n"
	ctx, err := model.NewContext(bytes.NewReader([]byte(input)), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	var offset int64
	s, withinObj, err := processObject(t.Context(), ctx, "1 0 obj", &offset, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("missing scanner")
	}
	if withinObj {
		t.Fatal("scanner remains inside the skipped object")
	}
	wantOffset := int64(strings.Index(input, "endobj") + len("endobj"))
	if offset != wantOffset {
		t.Fatalf("offset = %d, want %d", offset, wantOffset)
	}
}

// TestProcessObjectRejectsBoundedMalformedObjectStrict checks that strict parsing returns the malformed-object error.
func TestProcessObjectRejectsBoundedMalformedObjectStrict(t *testing.T) {
	input := "1 0 obj\n<</D[0/a0 obj\n<</E 1>>\nendobj\n"
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationStrict
	ctx, err := model.NewContext(bytes.NewReader([]byte(input)), conf)
	if err != nil {
		t.Fatal(err)
	}

	var offset int64
	_, _, err = processObject(t.Context(), ctx, "1 0 obj", &offset, 1, nil)
	if err == nil || !strings.Contains(err.Error(), "unterminated array") {
		t.Fatalf("got %v, want malformed object error", err)
	}
}

func TestDecodeSubsectionClassifiesCorruptEntryType(t *testing.T) {
	_, _, _, err := decodeSubsection([]string{"0000000000", "65535", "x"})
	if !errors.Is(err, errCorruptXRefSubsection) {
		t.Fatalf("got %v, want %v", err, errCorruptXRefSubsection)
	}
}

func TestParseXRefTableSubSectionClassifiesIncompleteSubsection(t *testing.T) {
	xRefTable := &model.XRefTable{Table: map[int]*model.XRefTableEntry{}}
	s := bufio.NewScanner(strings.NewReader("trailer\n"))

	_, err := parseXRefTableSubSection(xRefTable, model.DefaultResourceLimits(), s, []string{"0", "1"}, 0, 0)
	if !errors.Is(err, errIncompleteXRefSubsection) {
		t.Fatalf("got %v, want %v", err, errIncompleteXRefSubsection)
	}
}

func TestParseXRefTableSubSectionRejectsLimits(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		limits model.ResourceLimits
	}{
		{"negative start", []string{"-1", "1"}, model.ResourceLimits{MaxObjectCount: 2, MaxXRefEntries: 2}},
		{"negative count", []string{"0", "-1"}, model.ResourceLimits{MaxObjectCount: 2, MaxXRefEntries: 2}},
		{"object range", []string{"1", "2"}, model.ResourceLimits{MaxObjectCount: 2, MaxXRefEntries: 2}},
		{"entry count", []string{"0", "2"}, model.ResourceLimits{MaxObjectCount: 3, MaxXRefEntries: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xRefTable := &model.XRefTable{Table: map[int]*model.XRefTableEntry{}}
			s := bufio.NewScanner(strings.NewReader("trailer\n"))
			if _, err := parseXRefTableSubSection(xRefTable, tt.limits, s, tt.fields, 0, 0); err == nil {
				t.Fatal("expected xref subsection limit error")
			}
		})
	}
}

func TestParseXRefTableSubSectionBoundsExtraEntries(t *testing.T) {
	xRefTable := &model.XRefTable{Table: map[int]*model.XRefTableEntry{}}
	s := bufio.NewScanner(strings.NewReader("0000000000 65535 f\n0000000000 65535 f\ntrailer\n"))
	limits := model.ResourceLimits{MaxObjectCount: 2, MaxXRefEntries: 1}

	_, err := parseXRefTableSubSection(xRefTable, limits, s, []string{"0", "1"}, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "xref entry count exceeds limit 1") {
		t.Fatalf("got %v, want xref entry count limit error", err)
	}
	if len(xRefTable.Table) != 1 {
		t.Fatalf("parsed %d xref entries, want 1", len(xRefTable.Table))
	}
}

func TestParseXRefTableSubSectionRepairsBoundedExtraEntry(t *testing.T) {
	xRefTable := &model.XRefTable{Table: map[int]*model.XRefTableEntry{}}
	s := bufio.NewScanner(strings.NewReader("0000000000 65535 f\n0000000000 65535 f\ntrailer\n"))
	limits := model.ResourceLimits{MaxObjectCount: 2, MaxXRefEntries: 2}

	line, err := parseXRefTableSubSection(xRefTable, limits, s, []string{"0", "1"}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if line != "trailer" || len(xRefTable.Table) != 2 {
		t.Fatalf("got line %q and %d entries, want trailer and 2 entries", line, len(xRefTable.Table))
	}
}

func TestParseXRefSectionUsesConfiguredLimits(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxObjectCount = 1
	ctx := &model.Context{
		Configuration: conf,
		XRefTable:     &model.XRefTable{Table: map[int]*model.XRefTableEntry{}},
	}
	s := bufio.NewScanner(strings.NewReader(""))
	ssCount := 0
	offset := int64(0)

	_, err := parseXRefSection(t.Context(), ctx, s, []string{"0", "2"}, &ssCount, &offset, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "xref subsection object range exceeds limit 1") {
		t.Fatalf("got %v, want configured object range limit error", err)
	}
}

func TestScanLineRawClassifiesMissingLine(t *testing.T) {
	_, err := scanLineRaw(bufio.NewScanner(strings.NewReader("")))
	if !errors.Is(err, errMissingScannerLine) {
		t.Fatalf("got %v, want %v", err, errMissingScannerLine)
	}
}

func TestParseTrailerClassifiesMissingEntries(t *testing.T) {
	tests := []struct {
		name    string
		table   *model.XRefTable
		d       types.Dict
		wantErr error
	}{
		{
			name:    "missing size",
			table:   &model.XRefTable{},
			d:       types.Dict{},
			wantErr: errMissingTrailerSize,
		},
		{
			name:    "indirect size",
			table:   &model.XRefTable{},
			d:       types.Dict{"Size": *types.NewIndirectRef(7, 0)},
			wantErr: errCorruptTrailerDict,
		},
		{
			name:    "missing root",
			table:   &model.XRefTable{Size: intPtr(1)},
			d:       types.Dict{},
			wantErr: errMissingTrailerRoot,
		},
		{
			name: "invalid id",
			table: &model.XRefTable{
				Size:           intPtr(1),
				Root:           types.NewIndirectRef(1, 0),
				ValidationMode: model.ValidationStrict,
			},
			d:       types.Dict{"ID": types.Array{}},
			wantErr: errInvalidTrailerID,
		},
		{
			name: "missing id",
			table: &model.XRefTable{
				Size:    intPtr(1),
				Root:    types.NewIndirectRef(1, 0),
				Encrypt: types.NewIndirectRef(2, 0),
			},
			d:       types.Dict{},
			wantErr: errMissingTrailerID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := parseTrailer(tt.table, tt.d); !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestOffsetPrevRejectsIndirectValue(t *testing.T) {
	d := types.Dict{"Prev": *types.NewIndirectRef(7, 0)}
	if _, err := offsetPrev(&model.Context{}, d, nil); !errors.Is(err, errCorruptTrailerDict) {
		t.Fatalf("got %v, want %v", err, errCorruptTrailerDict)
	}
}

func TestLinearizationMarkerRequiresDirectInteger(t *testing.T) {
	d := types.Dict{"Linearized": *types.NewIndirectRef(7, 0)}
	if d.IsLinearizationParmDict() {
		t.Fatal("indirect Linearized value identified as a linearization parameter dictionary")
	}
}

func TestXRefStreamDictErrorsIncludeObjectContext(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = xRefStreamDict(t.Context(), ctx, types.Integer(1), 7, 0)
	if !errors.Is(err, errMissingXRefStreamDict) {
		t.Fatalf("got %v, want %v", err, errMissingXRefStreamDict)
	}
	if !strings.Contains(err.Error(), "xref stream obj#7") {
		t.Fatalf("expected object context, got %q", err.Error())
	}

	_, err = xRefStreamDict(t.Context(), ctx, types.Dict{}, 8, 0)
	if !errors.Is(err, errMissingXRefStreamLength) {
		t.Fatalf("got %v, want %v", err, errMissingXRefStreamLength)
	}
	if !strings.Contains(err.Error(), "xref stream obj#8") {
		t.Fatalf("expected object context, got %q", err.Error())
	}
}

func TestParseXRefStreamClassifiesCorruptStreamObject(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	offset := int64(0)

	_, err = parseXRefStream(t.Context(), ctx, strings.NewReader("1 0 obj\n<<>>\nendobj\n"), &offset, 0, 0)
	if !errors.Is(err, errCorruptXRefStream) {
		t.Fatalf("got %v, want %v", err, errCorruptXRefStream)
	}
	if !strings.Contains(err.Error(), "xref stream object") {
		t.Fatalf("expected xref stream object context, got %q", err.Error())
	}
}

func TestDereferencedObjectClassifiesUnregisteredObject(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = dereferencedObject(t.Context(), ctx, 7)
	if !errors.Is(err, errUnregisteredObject) {
		t.Fatalf("got %v, want %v", err, errUnregisteredObject)
	}
	if !strings.Contains(err.Error(), "object 7") {
		t.Fatalf("expected object context, got %q", err.Error())
	}
}

// TestDereferencedObjectKeepsEndobjAfterLiteral covers keyword scanning through the indexed-object read path.
func TestDereferencedObjectKeepsEndobjAfterLiteral(t *testing.T) {
	input := "4 0 obj\n(abc)endobj\n5 0 obj\n<</T (def)/Length 3>>\nstream\nxyz\nendstream\nendobj\n"
	ctx, err := model.NewContext(strings.NewReader(input), nil)
	if err != nil {
		t.Fatal(err)
	}
	offset, generation := int64(0), 0
	ctx.Table[4] = &model.XRefTableEntry{Offset: &offset, Generation: &generation}
	obj, err := dereferencedObject(t.Context(), ctx, 4)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := obj.(types.StringLiteral); !ok || got != "abc" {
		t.Fatalf("got %T %v, want string literal abc", obj, obj)
	}
	_, end, _, _, err := object(t.Context(), ctx, offset, 4, generation)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Index(input, "endobj"); end != want {
		t.Fatalf("endobj at %d, want %d", end, want)
	}
}

func TestRegisterXRefObjectsTracksEOLOffsets(t *testing.T) {
	for _, eol := range []string{"\n", "\r\n"} {
		t.Run(fmt.Sprintf("EOL=%q", eol), func(t *testing.T) {
			pdf := []byte("%PDF-1.7" + eol + "%\x80\x81\x82\x83" + eol + "7 0 obj" + eol + "<<>>" + eol + "endobj" + eol)
			ctx, err := model.NewContext(bytes.NewReader(pdf), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := registerXRefObjects(ctx, 1); err != nil {
				t.Fatal(err)
			}
			entry, ok := ctx.Find(7)
			if !ok || *entry.Offset != int64(bytes.Index(pdf, []byte("7 0 obj"))) {
				t.Fatalf("unexpected xref entry: %+v", entry)
			}
		})
	}
}

func TestDereferencedTypedObjectClassification(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Table[7] = model.NewXRefTableEntryGen0(types.Name("NotInteger"))
	ctx.Table[8] = model.NewXRefTableEntryGen0(types.Integer(1))

	_, err = dereferencedInteger(t.Context(), ctx, 7)
	if !errors.Is(err, errCorruptIntegerObject) {
		t.Fatalf("got %v, want %v", err, errCorruptIntegerObject)
	}
	if !strings.Contains(err.Error(), "object 7") {
		t.Fatalf("expected object context, got %q", err.Error())
	}

	_, err = dereferencedDict(t.Context(), ctx, 8)
	if !errors.Is(err, errCorruptDictObject) {
		t.Fatalf("got %v, want %v", err, errCorruptDictObject)
	}
	if !strings.Contains(err.Error(), "object 8") {
		t.Fatalf("expected object context, got %q", err.Error())
	}
}

func TestBootstrapIntegerEntry(t *testing.T) {
	compressed := model.NewXRefTableEntryGen0(types.Integer(11))
	compressed.Compressed = true
	ctx := &model.Context{XRefTable: &model.XRefTable{Table: map[int]*model.XRefTableEntry{
		7: model.NewXRefTableEntryGen0(types.Integer(42)),
		8: model.NewXRefTableEntryGen0(types.Name("wrong")),
		9: compressed,
	}}}

	tests := []struct {
		name     string
		value    types.Object
		want     int
		wantText string
	}{
		{"direct", types.Integer(42), 42, ""},
		{"indirect", *types.NewIndirectRef(7, 0), 42, ""},
		{"missing target", *types.NewIndirectRef(6, 0), 0, "missing indirect target"},
		{"wrong target type", *types.NewIndirectRef(8, 0), 0, "corrupt integer object"},
		{"compressed target", *types.NewIndirectRef(9, 0), 0, "compressed during bootstrap"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := types.Dict{"Value": tt.value}
			got, found, err := bootstrapIntegerEntry(t.Context(), ctx, d, "Value")
			if tt.wantText != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantText) {
					t.Fatalf("got %v, want error containing %q", err, tt.wantText)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !found || got == nil || got.Value() != tt.want {
				t.Fatalf("got value=%v found=%t, want %d", got, found, tt.want)
			}
		})
	}
}

func TestBootstrapIntegerEntryMaterializesUncompressedTarget(t *testing.T) {
	pdf := []byte("7 0 obj\n42\nendobj\n")
	ctx, err := model.NewContext(bytes.NewReader(pdf), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	offset, generation := int64(0), 0
	ctx.XRefTable.Table[7] = &model.XRefTableEntry{Offset: &offset, Generation: &generation}
	d := types.Dict{"Value": *types.NewIndirectRef(7, 0)}

	i, _, err := bootstrapIntegerEntry(t.Context(), ctx, d, "Value")
	if err != nil {
		t.Fatal(err)
	}
	if i == nil || i.Value() != 42 {
		t.Fatalf("got %v, want 42", i)
	}
	if ctx.XRefTable.Table[7].Object != types.Integer(42) {
		t.Fatalf("target object was not materialized: %v", ctx.XRefTable.Table[7].Object)
	}
}

func TestXRefStreamOffsetResolvesIndirectValue(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{Table: map[int]*model.XRefTableEntry{
		7: model.NewXRefTableEntryGen0(types.Integer(128)),
	}}}
	indRef := *types.NewIndirectRef(7, 0)
	d := types.Dict{"XRefStm": indRef}

	offset, err := xrefStreamOffset(t.Context(), ctx, d, false)
	if err != nil {
		t.Fatal(err)
	}
	if offset == nil || *offset != 128 {
		t.Fatalf("offset = %v, want 128", offset)
	}
	if d["XRefStm"] != indRef {
		t.Fatalf("XRefStm = %v, want original indirect reference", d["XRefStm"])
	}
}

func TestObjectStreamBootstrapIntegers(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{Table: map[int]*model.XRefTableEntry{
		7: model.NewXRefTableEntryGen0(types.Integer(2)),
		8: model.NewXRefTableEntryGen0(types.Integer(10)),
	}}}
	sd := types.StreamDict{Dict: types.Dict{
		"Type":  types.Name("ObjStm"),
		"N":     *types.NewIndirectRef(7, 0),
		"First": *types.NewIndirectRef(8, 0),
	}}
	n, _, err := bootstrapIntegerEntry(t.Context(), ctx, sd.Dict, "N")
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := bootstrapIntegerEntry(t.Context(), ctx, sd.Dict, "First")
	if err != nil {
		t.Fatal(err)
	}
	osd, err := model.ObjectStreamDictWithResolvedIntegers(&sd, model.DefaultResourceLimits(), n, first)
	if err != nil {
		t.Fatal(err)
	}
	if osd.ObjCount != 2 || osd.FirstObjOffset != 10 {
		t.Fatalf("got N=%d First=%d, want N=2 First=10", osd.ObjCount, osd.FirstObjOffset)
	}
}

func TestDecompressXRefTableEntryClassifiesObjectStreamErrors(t *testing.T) {
	objStreamNr := 9
	objStreamInd := 0
	tests := []struct {
		name    string
		table   map[int]*model.XRefTableEntry
		wantErr error
	}{
		{
			name:    "missing xref entry",
			table:   map[int]*model.XRefTableEntry{},
			wantErr: errMissingObjectStreamEntry,
		},
		{
			name: "missing object stream",
			table: map[int]*model.XRefTableEntry{
				objStreamNr: model.NewXRefTableEntryGen0(types.Dict{}),
			},
			wantErr: errMissingObjectStream,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xRefTable := &model.XRefTable{Table: tt.table}
			entry := &model.XRefTableEntry{
				Compressed:      true,
				ObjectStream:    &objStreamNr,
				ObjectStreamInd: &objStreamInd,
			}

			err := decompressXRefTableEntry(xRefTable, 7, entry)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
			if !strings.Contains(err.Error(), "object stream 9") {
				t.Fatalf("expected object stream context, got %q", err.Error())
			}
		})
	}
}

func TestCompressedObjectRejectsStreamObjects(t *testing.T) {
	_, err := compressedObject(t.Context(), "<< /Length 1 >>", model.ValidationRelaxed)
	if !errors.Is(err, errObjectStreamContainsStream) {
		t.Fatalf("got %v, want %v", err, errObjectStreamContainsStream)
	}
}

func TestParseObjectStreamClassifiesCorruptDict(t *testing.T) {
	osd := &types.ObjectStreamDict{
		StreamDict:     types.StreamDict{Content: []byte("1")},
		FirstObjOffset: 1,
	}

	err := parseObjectStream(t.Context(), osd, model.DefaultResourceLimits(), model.ValidationRelaxed)
	if !errors.Is(err, errCorruptObjectStreamDict) {
		t.Fatalf("got %v, want %v", err, errCorruptObjectStreamDict)
	}
}

func TestParseObjectStreamClassifiesObjectCountLimit(t *testing.T) {
	content := []byte("1 0 2 1")
	osd := &types.ObjectStreamDict{
		StreamDict:     types.StreamDict{Content: content},
		FirstObjOffset: len(content),
	}
	limits := model.DefaultResourceLimits()
	limits.MaxObjectStreamCount = 1

	err := parseObjectStream(t.Context(), osd, limits, model.ValidationRelaxed)
	if !errors.Is(err, errObjectStreamObjectCountLimit) {
		t.Fatalf("got %v, want %v", err, errObjectStreamObjectCountLimit)
	}
}

func TestParseObjectStreamRejectsFirstBeyondDecodeLimit(t *testing.T) {
	const content = "1 0 42"
	osd := &types.ObjectStreamDict{
		StreamDict: types.StreamDict{
			Raw:            []byte(hex.EncodeToString([]byte(content)) + ">"),
			FilterPipeline: []types.PDFFilter{{Name: filter.ASCIIHex}},
		},
		FirstObjOffset: 4,
		ObjCount:       1,
	}
	limits := model.DefaultResourceLimits()
	limits.MaxDecodeBytes = 3
	if err := parseObjectStream(t.Context(), osd, limits, model.ValidationRelaxed); !errors.Is(err, filter.ErrDecodeLimitExceeded) {
		t.Fatalf("got %v, want %v", err, filter.ErrDecodeLimitExceeded)
	}
}

// TestDecodeObjectStreamRejectsInvalidEntries checks the entry before its offset and generation are dereferenced.
func TestDecodeObjectStreamRejectsInvalidEntries(t *testing.T) {
	offset, generation := int64(0), 0
	tests := []struct {
		name  string
		entry *model.XRefTableEntry
	}{
		{"missing", nil},
		{"empty", &model.XRefTableEntry{}},
		{"compressed", &model.XRefTableEntry{Compressed: true}},
		{"compressed with offset", &model.XRefTableEntry{Compressed: true, Offset: &offset, Generation: &generation}},
		{"free", &model.XRefTableEntry{Free: true, Offset: &offset, Generation: &generation}},
		{"missing offset", &model.XRefTableEntry{Generation: &generation}},
		{"missing generation", &model.XRefTableEntry{Offset: &offset}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, err := model.NewContext(bytes.NewReader(nil), model.NewStatelessConfiguration())
			if err != nil {
				t.Fatal(err)
			}
			if tc.entry != nil {
				ctx.Table[7] = tc.entry
			}
			err = decodeObjectStream(t.Context(), ctx, 7)
			if !errors.Is(err, errMissingObjectStreamEntry) {
				t.Fatalf("got %v, want %v", err, errMissingObjectStreamEntry)
			}
			if !strings.Contains(err.Error(), "object stream obj#7") {
				t.Fatalf("expected object stream context, got %q", err.Error())
			}
		})
	}
}

func invalidObjectStreamEntryPDF(version model.Version, free bool) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%%PDF-%s\n", version)
	offsets := make([]int, 7)
	for i, body := range []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Resources << >> >>",
	} {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	offsets[6] = b.Len()
	streamEntry := [3]int{2, 5, 0} // Object stream 5 incorrectly claims to live inside itself.
	if free {
		streamEntry = [3]int{0, 0, 0}
	}
	data := []byte{}
	for _, row := range [][3]int{
		{0, 0, 65535}, {1, offsets[1], 0}, {1, offsets[2], 0}, {1, offsets[3], 0},
		{2, 5, 0}, streamEntry, {1, offsets[6], 0},
	} {
		data = append(data, byte(row[0]), byte(row[1]>>24), byte(row[1]>>16), byte(row[1]>>8), byte(row[1]),
			byte(row[2]>>8), byte(row[2]))
	}
	fmt.Fprintf(&b, "6 0 obj\n<< /Type /XRef /Size 7 /W [1 4 2] /Root 1 0 R /Length %d >>\nstream\n", len(data))
	b.Write(data)
	fmt.Fprintf(&b, "\nendstream\nendobj\nstartxref\n%d\n%%%%EOF\n", offsets[6])
	return b.Bytes()
}

// TestReadRejectsInvalidObjectStreamEntries verifies malformed cross-references return an error without panicking.
func TestReadRejectsInvalidObjectStreamEntries(t *testing.T) {
	for _, version := range []model.Version{model.V15, model.V17, model.V20} {
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			for _, free := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/mode%d/free=%t", version, mode, free), func(t *testing.T) {
					conf := model.NewStatelessConfiguration()
					conf.ValidationMode = mode
					conf.Offline = true
					_, err := Read(t.Context(), bytes.NewReader(invalidObjectStreamEntryPDF(version, free)), conf)
					if !errors.Is(err, errMissingObjectStreamEntry) {
						t.Fatalf("got %v, want %v", err, errMissingObjectStreamEntry)
					}
					if !strings.Contains(err.Error(), "object stream obj#5") {
						t.Fatalf("expected object stream context, got %q", err.Error())
					}
				})
			}
		}
	}
}

func TestDecodeObjectStreamClassifiesCorruptObjectStream(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader([]byte("7 0 obj\n1\nendobj\n")), nil)
	if err != nil {
		t.Fatal(err)
	}
	zeroOffset := int64(0)
	zeroGen := 0
	ctx.Table[7] = &model.XRefTableEntry{
		Offset:     &zeroOffset,
		Generation: &zeroGen,
	}

	err = decodeObjectStream(t.Context(), ctx, 7)
	if !errors.Is(err, errCorruptObjectStream) {
		t.Fatalf("got %v, want %v", err, errCorruptObjectStream)
	}
	if !strings.Contains(err.Error(), "object stream obj#7") {
		t.Fatalf("expected object stream context, got %q", err.Error())
	}
}

// TestHeaderVersionAcrossBlockBoundaries verifies false prefixes cannot hide a later PDF header.
func TestHeaderVersionAcrossBlockBoundaries(t *testing.T) {
	for _, version := range []model.Version{model.V17, model.V20} {
		for _, falsePrefix := range []string{"", "%", "%P", "%PD", "%%PDFX"} {
			for pad := 0; pad < 205; pad++ {
				in := strings.Repeat("a", pad) + falsePrefix + "%PDF-" + version.String() + "\r\n" + strings.Repeat("b", 200)
				v, eol, off, err := headerVersion(strings.NewReader(in))
				if err != nil {
					t.Fatalf("version %s, padding %d, prefix %q: %v", version, pad, falsePrefix, err)
				}
				if v == nil || *v != version || eol != 2 || off != int64(pad+len(falsePrefix)) {
					t.Fatalf("version %s, padding %d, prefix %q: got version %v, EOL %d, offset %d", version, pad, falsePrefix, v, eol, off)
				}
			}
		}
	}
}

// TestReadHeaderAfterFalsePrefix verifies header scanning preserves offsets for a complete PDF.
func TestReadHeaderAfterFalsePrefix(t *testing.T) {
	bb, err := os.ReadFile(filepath.Join("..", "testdata", "test.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	for _, pad := range []int{85, 98, 99, 185} {
		in := append([]byte(strings.Repeat("a", pad)+"%"), bb...)
		conf := model.NewDefaultConfiguration()
		conf.ValidationMode = model.ValidationStrict
		if _, err := Read(t.Context(), bytes.NewReader(in), conf); err != nil {
			t.Fatalf("padding %d: %v", pad, err)
		}
	}
}

// TestHeaderVersionShortLookahead verifies the minimum header buffer at different block positions.
func TestHeaderVersionShortLookahead(t *testing.T) {
	for _, pad := range []int{0, 85, 95, 99, 100} {
		for _, size := range []int{9, 17, 18, 19} {
			in := strings.Repeat("a", pad) + "%PDF-1.7\n" + strings.Repeat("b", size-9)
			v, eol, off, err := headerVersion(strings.NewReader(in))
			if size < 18 {
				if !errors.Is(err, ErrCorruptHeader) {
					t.Fatalf("padding %d, size %d: got %v, want ErrCorruptHeader", pad, size, err)
				}
				continue
			}
			if err != nil || v == nil || *v != model.V17 || eol != 1 || off != int64(pad) {
				t.Fatalf("padding %d, size %d: version %v, EOL %d, offset %d, error %v", pad, size, v, eol, off, err)
			}
		}
	}
}

func TestHeaderVersionRejectsPostScript(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"Plain", "%!PS-Adobe-3.0\n%%Title: report.txt\n"},
		{"BOMAndWhitespace", "\xef\xbb\xbf \r\n%!PS-Adobe-3.0\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := headerVersion(strings.NewReader(tt.in))
			if !errors.Is(err, ErrPostScriptInput) {
				t.Fatalf("got %v, want ErrPostScriptInput", err)
			}
		})
	}
}

func TestHeaderVersionClassifiesCorruptHeader(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "NoPDFHeader", in: "not a pdf\n"},
		{name: "MissingEOL", in: "%PDF-1.7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := headerVersion(strings.NewReader(tt.in))
			if !errors.Is(err, ErrCorruptHeader) {
				t.Fatalf("got %v, want ErrCorruptHeader", err)
			}
		})
	}
}

func TestHeaderVersionUnknownVersionIncludesContext(t *testing.T) {
	_, _, _, err := headerVersion(strings.NewReader("%PDF-9.9\n% padding for header scan\n"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown PDF header version") {
		t.Fatalf("expected unknown version context, got %q", err.Error())
	}
}

func TestHeaderVersionPreservesIOErrors(t *testing.T) {
	seekErr := errors.New("seek failed")
	_, _, _, err := headerVersion(headerErrorReader{seekErr: seekErr})
	if !errors.Is(err, seekErr) {
		t.Fatalf("expected %v, got %v", seekErr, err)
	}
	if !strings.Contains(err.Error(), "seek header") {
		t.Fatalf("expected seek context, got %q", err.Error())
	}

	readErr := errors.New("read failed")
	_, _, _, err = headerVersion(headerErrorReader{readErr: readErr})
	if !errors.Is(err, readErr) {
		t.Fatalf("expected %v, got %v", readErr, err)
	}
	if !strings.Contains(err.Error(), "read header") {
		t.Fatalf("expected read context, got %q", err.Error())
	}
}

func TestHandleUnencryptedFileClassifiesErrors(t *testing.T) {
	ctx := &model.Context{Configuration: model.NewDefaultConfiguration()}

	ctx.Cmd = model.DECRYPT
	if err := handleUnencryptedFile(ctx); !errors.Is(err, ErrNotEncrypted) {
		t.Fatalf("got %v, want ErrNotEncrypted", err)
	}

	ctx.Cmd = model.ENCRYPT
	if err := handleUnencryptedFile(ctx); !errors.Is(err, ErrOwnerPasswordRequired) {
		t.Fatalf("got %v, want ErrOwnerPasswordRequired", err)
	}
}

func TestOwnerPasswordRequiredWithOPWPreservesSentinel(t *testing.T) {
	err := fmt.Errorf("%w with --opw", ErrOwnerPasswordRequired)
	if !errors.Is(err, ErrOwnerPasswordRequired) {
		t.Fatalf("got %v, want ErrOwnerPasswordRequired", err)
	}
}

func TestCheckForEncryptionClassifiesEncryptedError(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Cmd = model.ENCRYPT
	ctx.Encrypt = types.NewIndirectRef(1, 0)

	if err := checkForEncryption(t.Context(), ctx); !errors.Is(err, ErrEncrypted) {
		t.Fatalf("got %v, want ErrEncrypted", err)
	} else if !strings.Contains(err.Error(), "encryption status") {
		t.Fatalf("expected encryption status context, got %q", err)
	}
}

func TestCheckForEncryptionClassifiesUnencryptedError(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Cmd = model.DECRYPT

	err = checkForEncryption(t.Context(), ctx)
	if !errors.Is(err, ErrNotEncrypted) {
		t.Fatalf("got %v, want ErrNotEncrypted", err)
	}
	if !strings.Contains(err.Error(), "encryption status") {
		t.Fatalf("expected encryption status context, got %q", err)
	}
}

func TestCheckForEncryptionRejectsIncompleteContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  *model.Context
		want error
	}{
		{name: "missing context", want: ErrMissingPDFContext},
		{
			name: "missing xref table",
			ctx:  &model.Context{Configuration: model.NewDefaultConfiguration()},
			want: ErrMissingXRefTable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkForEncryption(t.Context(), tt.ctx); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCheckForEncryptionWrapsEncryptionDictionaryError(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Encrypt = types.NewIndirectRef(7, 0)

	err = checkForEncryption(t.Context(), ctx)
	if !errors.Is(err, errUnregisteredObject) {
		t.Fatalf("got %v, want %v", err, errUnregisteredObject)
	}
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
	}
	if !strings.Contains(err.Error(), "encryption dictionary obj#7") {
		t.Fatalf("expected encryption dictionary context, got %q", err.Error())
	}
}

// TestCheckForEncryptionClassifiesWrongTypeDictionary verifies both structural and semantic causes survive.
func TestCheckForEncryptionClassifiesWrongTypeDictionary(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Encrypt = types.NewIndirectRef(7, 0)
	ctx.Table[7] = model.NewXRefTableEntryGen0(types.Integer(1))

	err = checkForEncryption(t.Context(), ctx)
	if !errors.Is(err, errCorruptDictObject) {
		t.Fatalf("got %v, want %v", err, errCorruptDictObject)
	}
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
	}
}

// TestCheckForEncryptionPreservesDictionaryReadError verifies operational I/O errors are not classified as malformed.
func TestCheckForEncryptionPreservesDictionaryReadError(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	readErr := errors.New("read failed")
	ctx.Read.RS = headerErrorReader{readErr: readErr}
	ctx.Encrypt = types.NewIndirectRef(7, 0)
	entry := model.NewXRefTableEntryGen0(nil)
	offset := int64(0)
	entry.Offset = &offset
	ctx.Table[7] = entry

	err = checkForEncryption(t.Context(), ctx)
	if !errors.Is(err, readErr) {
		t.Fatalf("got %v, want %v", err, readErr)
	}
	if errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("unexpected malformed encryption classification: %v", err)
	}
	if !strings.Contains(err.Error(), "encryption dictionary obj#7") {
		t.Fatalf("missing encryption object context: %v", err)
	}
}

// TestCheckForEncryptionClassifiesUnterminatedDictionary verifies parser and encryption causes survive.
func TestCheckForEncryptionClassifiesUnterminatedDictionary(t *testing.T) {
	rs := bytes.NewReader([]byte("7 0 obj\n<< /Filter /Standard\nendobj\n"))
	ctx, err := model.NewContext(rs, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = model.ValidationStrict
	ctx.Encrypt = types.NewIndirectRef(7, 0)
	entry := model.NewXRefTableEntryGen0(nil)
	offset := int64(0)
	entry.Offset = &offset
	ctx.Table[7] = entry

	err = checkForEncryption(t.Context(), ctx)
	if !errors.Is(err, model.ErrDictionaryCorrupt) {
		t.Fatalf("got %v, want %v", err, model.ErrDictionaryCorrupt)
	}
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
	}
	if !strings.Contains(err.Error(), "encryption dictionary obj#") {
		t.Fatalf("missing encryption dictionary object context: %v", err)
	}
}

func TestSetupEncryptionKeyRejectsMissingID(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	setStrictPDF17XRefTable(ctx)

	err = setupEncryptionKey(ctx, newEncryptDict(false, false, 40, 0))
	if !errors.Is(err, errMissingTrailerID) {
		t.Fatalf("got %v, want %v", err, errMissingTrailerID)
	}
	if !strings.Contains(err.Error(), "read encryption ID") {
		t.Fatalf("expected encryption ID context, got %q", err)
	}
}

func TestSetupEncryptionKeyPreservesAuthenticationSentinels(t *testing.T) {
	tests := []struct {
		name string
		cmd  model.CommandMode
		want error
	}{
		{name: "wrong password", cmd: model.DECRYPT, want: ErrWrongPassword},
		{name: "owner password required", cmd: model.CHANGEOPW, want: ErrOwnerPasswordRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := model.NewContext(bytes.NewReader(nil), nil)
			if err != nil {
				t.Fatal(err)
			}
			setStrictPDF17XRefTable(ctx)
			id := types.NewHexLiteral([]byte("id"))
			ctx.ID = types.Array{id, id}
			ctx.Cmd = tt.cmd

			err = setupEncryptionKey(ctx, newEncryptDict(false, false, 40, 0))
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestHandlePermissionsClassifiesPermissionDenied(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Cmd = model.ADDATTACHMENTS
	ctx.OwnerPW = "opw"
	ctx.E = &model.Enc{R: 4}

	if err := handlePermissions(ctx); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("got %v, want ErrPermissionDenied", err)
	}
}

func TestHandlePermissionsClassifiesInvalidPermissions(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.E = &model.Enc{R: 5, Perms: make([]byte, 16)}
	ctx.EncKey = make([]byte, 16)

	err = handlePermissions(ctx)
	if !errors.Is(err, errInvalidPermissions) {
		t.Fatalf("got %v, want %v", err, errInvalidPermissions)
	}
	if !strings.Contains(err.Error(), "user password permissions") {
		t.Fatalf("expected user password permissions context, got %q", err.Error())
	}
}

func TestExtractXRefStreamEntriesDefaultType(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	const objCount = 67
	xsd := &types.XRefStreamDict{
		Objects: make([]int, objCount),
		W:       [3]int{0, 3, 0},
	}
	buf := make([]byte, 0, objCount*3)
	offsets := make([]int64, objCount)
	for objNr := 0; objNr < objCount; objNr++ {
		xsd.Objects[objNr] = objNr
		offset := int64(1 + objNr*257)
		offsets[objNr] = offset
		buf = append(buf, byte(offset>>16), byte(offset>>8), byte(offset))
	}

	if err := extractXRefTableEntriesFromXRefStream(buf, 0, xsd, ctx, 0); err != nil {
		t.Fatal(err)
	}
	if len(ctx.Table) != objCount {
		t.Fatalf("got %d xref entries, want %d", len(ctx.Table), objCount)
	}
	for objNr, wantOffset := range offsets {
		entry := ctx.Table[objNr]
		if entry == nil {
			t.Fatalf("missing xref entry for object %d", objNr)
		}
		if entry.Free || entry.Compressed {
			t.Fatalf("object %d: expected an in-use xref entry", objNr)
		}
		if entry.Offset == nil || *entry.Offset != wantOffset {
			t.Fatalf("object %d: got offset %v, want %d", objNr, entry.Offset, wantOffset)
		}
	}
}

func TestExtractXRefStreamEntriesRejectsZeroWidths(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	xsd := &types.XRefStreamDict{W: [3]int{0, 0, 0}}
	err = extractXRefTableEntriesFromXRefStream(nil, 0, xsd, ctx, 0)
	if !errors.Is(err, errInvalidXRefStreamWArray) {
		t.Fatalf("got %v, want %v", err, errInvalidXRefStreamWArray)
	}
}

func TestXRefStreamEntryLenRejectsOverflow(t *testing.T) {
	for _, tc := range []struct {
		name string
		w    [3]int
	}{
		{"first addition", [3]int{math.MaxInt, 1, 0}},
		{"second addition", [3]int{math.MaxInt - 1, 1, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := xRefStreamEntryLen(tc.w); !errors.Is(err, errInvalidXRefStreamWArray) {
				t.Fatalf("got %v, want %v", err, errInvalidXRefStreamWArray)
			}
		})
	}
}

func TestExtractXRefStreamEntriesRejectsWidthOverflow(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	w := 2*(math.MaxInt/3) + 1
	xsd := &types.XRefStreamDict{
		Objects: []int{0, 1, 2},
		W:       [3]int{w, w, w + 2},
	}
	err = extractXRefTableEntriesFromXRefStream(make([]byte, 3), 0, xsd, ctx, 0)
	if !errors.Is(err, errInvalidXRefStreamWArray) {
		t.Fatalf("got %v, want %v", err, errInvalidXRefStreamWArray)
	}
}

func TestExtractXRefStreamEntriesRejectsImpossibleObjectCount(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	xsd := &types.XRefStreamDict{
		Objects: []int{0, 1},
		W:       [3]int{math.MaxInt/2 + 1, 0, 0},
	}
	err = extractXRefTableEntriesFromXRefStream(nil, 0, xsd, ctx, 0)
	if !errors.Is(err, errCorruptXRefStream) {
		t.Fatalf("got %v, want %v", err, errCorruptXRefStream)
	}
}

func TestExtractXRefStreamEntriesClassifiesCorruptStream(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	xsd := &types.XRefStreamDict{
		Objects: []int{0},
		W:       [3]int{1, 1, 1},
	}
	err = extractXRefTableEntriesFromXRefStream([]byte{1, 2}, 0, xsd, ctx, 0)
	if !errors.Is(err, errCorruptXRefStream) {
		t.Fatalf("got %v, want %v", err, errCorruptXRefStream)
	}
}

// TestReadLargeDictObject verifies large dictionary objects are handled.
func TestReadLargeDictObject(t *testing.T) {
	// Test with "stream" and "endobj" inside the dictionary.
	var fp bytes.Buffer
	fp.WriteString("123 0 obj\n")
	data := make([]byte, 10*1024*1024)
	fp.WriteString("<<")
	fp.WriteString("/Foo <")
	fp.WriteString(hex.EncodeToString(data))
	fp.WriteString(">\n")
	fp.WriteString("/Bar (stream)\n")
	fp.WriteString("/Baz (endobj)\n")
	fp.WriteString("/Test <")
	fp.WriteString(hex.EncodeToString(data))
	fp.WriteString(">\n")
	fp.WriteString(">>\n")
	fp.WriteString("stream\n")
	fp.WriteString("Hello world!\n")
	fp.WriteString("endstream\n")
	fp.WriteString("endobj\n")

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	// Dummy pdfcpu context to be used for parsing a single object.
	c := &model.Context{
		Read: &model.ReadContext{
			RS: bytes.NewReader(fp.Bytes()),
		},
		XRefTable: &model.XRefTable{},
	}
	o, err := ParseObject(ctx, c, 0, 123, 0)
	if err != nil {
		t.Fatal(err)
	}

	d, ok := o.(types.StreamDict)
	if !ok {
		t.Fatalf("expected StreamDict, got %T", o)
	}

	if err := loadEncodedStreamContent(ctx, c, &d, true); err != nil {
		t.Fatal(err)
	}

	if foo := d.HexLiteralEntry("Foo"); foo == nil {
		t.Error("expected Foo entry")
	} else if expected := hex.EncodeToString(data); foo.Value() != expected {
		t.Errorf("Foo value mismatch, expected %d bytes, got %d", len(expected), len(foo.Value()))
	}

	if bar := d.StringEntry("Bar"); bar == nil {
		t.Error("expected Bar entry")
	} else if expected := "stream"; *bar != expected {
		t.Errorf("expected %s for Bar, got %s", expected, *bar)
	}

	if baz := d.StringEntry("Baz"); baz == nil {
		t.Error("expected Baz entry")
	} else if expected := "endobj"; *baz != expected {
		t.Errorf("expected %s for Baz, got %s", expected, *baz)
	}

	if err := d.Decode(); err != nil {
		t.Fatal(err)
	}

	if expected := "Hello world!"; string(d.Content) != expected {
		t.Errorf("expected stream content %s, got %s", expected, string(d.Content))
	}
}

// TestReadStreamContentRejectsStreamLimit verifies stream limits are enforced.
func TestReadStreamContentRejectsStreamLimit(t *testing.T) {
	_, err := readStreamContent(bytes.NewReader([]byte("abc")), 3, 2)
	if err == nil || !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("got %v, want stream limit error", err)
	}
}

func TestReadStreamContentAcceptsWrappedEOF(t *testing.T) {
	rd := &wrappedEOFReader{data: []byte("abc\nendstream\n")}

	buf, err := readStreamContent(rd, 20, 20)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf) != "abc\n" {
		t.Fatalf("got %q, want %q", string(buf), "abc\n")
	}
}

// TestEnsureStreamLengthRejectsNilDict verifies malformed stream dicts do not panic.
func TestEnsureStreamLengthRejectsNilDict(t *testing.T) {
	sd := &types.StreamDict{
		Raw: []byte("abc"),
	}

	err := ensureStreamLength(sd, true)
	if !errors.Is(err, errCorruptStreamDict) {
		t.Fatalf("got %v, want %v", err, errCorruptStreamDict)
	}
}

func TestEnsureIndirectStreamLengthClassifiesMissingLength(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = model.ValidationStrict
	sd := &types.StreamDict{}

	err = loadEncodedStreamContent(t.Context(), ctx, sd, false)
	if !errors.Is(err, errMissingStreamLength) {
		t.Fatalf("got %v, want %v", err, errMissingStreamLength)
	}
	if !strings.Contains(err.Error(), "stream length") {
		t.Fatalf("expected stream length context, got %q", err.Error())
	}
}

func TestEnsureIndirectStreamLengthIgnoresMissingReference(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Table[7] = &model.XRefTableEntry{Free: true}
	sd := &types.StreamDict{StreamLengthObjNr: intPtr(7)}

	if err := ensureIndirectStreamLength(t.Context(), ctx, sd, false); err != nil {
		t.Fatalf("expected missing stream length reference to be ignored, got %v", err)
	}
	if sd.StreamLength != nil {
		t.Fatalf("expected missing stream length reference to leave length nil, got %d", *sd.StreamLength)
	}
}

func TestStreamDictForObjectClassifiesMissingStreamOffset(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = streamDictForObject(t.Context(), ctx, types.Dict{}, 7, 0, 0, 0)
	if !errors.Is(err, errMissingStreamOffset) {
		t.Fatalf("got %v, want %v", err, errMissingStreamOffset)
	}
	if !strings.Contains(err.Error(), "stream obj#7") {
		t.Fatalf("expected stream object context, got %q", err.Error())
	}
}

func TestFilterPipelineClassifiesCorruptFilterArray(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = pdfFilterPipeline(t.Context(), ctx, types.Dict{"Filter": types.Integer(1)})
	if !errors.Is(err, errCorruptFilterArray) {
		t.Fatalf("got %v, want %v", err, errCorruptFilterArray)
	}
	if !strings.Contains(err.Error(), "filter pipeline") {
		t.Fatalf("expected filter pipeline context, got %q", err.Error())
	}

	_, err = filterNames(t.Context(), ctx, types.Array{types.Integer(1)})
	if !errors.Is(err, errCorruptFilterArray) {
		t.Fatalf("got %v, want %v", err, errCorruptFilterArray)
	}
	if !strings.Contains(err.Error(), "entry 0") {
		t.Fatalf("expected filter entry context, got %q", err.Error())
	}
}

func TestFilterPipelineClassifiesCorruptDecodeParms(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = singleFilter(t.Context(), ctx, filter.Flate, types.Dict{
		"DecodeParms": types.Array{types.Dict{}, types.Dict{}},
	})
	if !errors.Is(err, errCorruptDecodeParms) {
		t.Fatalf("got %v, want %v", err, errCorruptDecodeParms)
	}
	if !strings.Contains(err.Error(), filter.Flate) {
		t.Fatalf("expected filter name context, got %q", err.Error())
	}

	_, err = buildFilterPipeline(
		t.Context(),
		ctx,
		[]string{filter.Flate},
		types.Array{types.Integer(1)},
	)
	if !errors.Is(err, errCorruptDecodeParms) {
		t.Fatalf("got %v, want %v", err, errCorruptDecodeParms)
	}
	if !strings.Contains(err.Error(), "entry 0") {
		t.Fatalf("expected DecodeParms entry context, got %q", err.Error())
	}
}

func TestFilterPipelineNormalizesAliasesInRelaxedMode(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	d := types.Dict{"Filter": types.Name("A85")}
	pipeline, err := pdfFilterPipeline(t.Context(), ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	if got := pipeline[0].Name; got != filter.ASCII85 {
		t.Fatalf("got filter %q, want %q", got, filter.ASCII85)
	}
	if got := d.NameEntry("Filter"); got == nil || *got != filter.ASCII85 {
		t.Fatalf("got dictionary filter %v, want %q", got, filter.ASCII85)
	}

	aliases := types.Array{
		types.Name("AHx"),
		types.Name("A85"),
		types.Name("LZW"),
		types.Name("Fl"),
		types.Name("RL"),
		types.Name("CCF"),
		types.Name("DCT"),
	}
	want := []string{
		filter.ASCIIHex,
		filter.ASCII85,
		filter.LZW,
		filter.Flate,
		filter.RunLength,
		filter.CCITTFax,
		filter.DCT,
	}
	d = types.Dict{"Filter": aliases}
	pipeline, err = pdfFilterPipeline(t.Context(), ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	for i, f := range pipeline {
		if f.Name != want[i] {
			t.Errorf("filter %d: got %q, want %q", i, f.Name, want[i])
		}
	}
	filterArray := d.ArrayEntry("Filter")
	for i, obj := range filterArray {
		if got := obj.(types.Name).String(); got != want[i] {
			t.Errorf("dictionary filter %d: got %q, want %q", i, got, want[i])
		}
	}
}

func TestFilterPipelinePreservesAliasesInStrictMode(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationStrict
	ctx, err := model.NewContext(bytes.NewReader(nil), conf)
	if err != nil {
		t.Fatal(err)
	}

	d := types.Dict{"Filter": types.Name("A85")}
	pipeline, err := pdfFilterPipeline(t.Context(), ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	if got := pipeline[0].Name; got != "A85" {
		t.Fatalf("got filter %q, want A85", got)
	}
	sd := types.StreamDict{Raw: []byte("~>"), FilterPipeline: pipeline}
	if err := sd.Decode(); err == nil || !strings.Contains(err.Error(), "invalid filter") {
		t.Fatalf("got %v, want invalid filter error", err)
	}
}

func TestSaveDecodedStreamContentIgnoresUnsupportedFilter(t *testing.T) {
	sd := &types.StreamDict{
		Raw:            []byte("abc"),
		FilterPipeline: []types.PDFFilter{{Name: filter.JBIG2}},
	}

	if err := saveDecodedStreamContent(nil, sd, 0, 0, true); err != nil {
		t.Fatalf("expected unsupported filter to be ignored, got %v", err)
	}
}

// TestReadLargeDictObjectStream verifies large object streams are handled.
func TestReadLargeDictObjectStream(t *testing.T) {
	// Test without "stream" and "endobj" inside the dictionary.
	var fp bytes.Buffer
	fp.WriteString("123 0 obj\n")
	data := make([]byte, 10*1024*1024)
	fp.WriteString("<<")
	fp.WriteString("/Foo <")
	fp.WriteString(hex.EncodeToString(data))
	fp.WriteString(">\n")
	fp.WriteString("/Bar (Test)\n")
	fp.WriteString("/Baz <")
	fp.WriteString(hex.EncodeToString(data))
	fp.WriteString(">\n")
	fp.WriteString(">>\n")
	fp.WriteString("stream\n")
	fp.WriteString("Hello world!\n")
	fp.WriteString("endstream\n")
	fp.WriteString("endobj\n")

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	// Dummy pdfcpu context to be used for parsing a single object.
	c := &model.Context{
		Read: &model.ReadContext{
			RS: bytes.NewReader(fp.Bytes()),
		},
		XRefTable: &model.XRefTable{},
	}
	o, err := ParseObject(ctx, c, 0, 123, 0)
	if err != nil {
		t.Fatal(err)
	}

	d, ok := o.(types.StreamDict)
	if !ok {
		t.Fatalf("expected StreamDict, got %T", o)
	}

	if err := loadEncodedStreamContent(ctx, c, &d, true); err != nil {
		t.Fatal(err)
	}

	if foo := d.HexLiteralEntry("Foo"); foo == nil {
		t.Error("expected Foo entry")
	} else if expected := hex.EncodeToString(data); foo.Value() != expected {
		t.Errorf("Foo value mismatch, expected %d bytes, got %d", len(expected), len(foo.Value()))
	}

	if bar := d.StringEntry("Bar"); bar == nil {
		t.Error("expected Bar entry")
	} else if expected := "Test"; *bar != expected {
		t.Errorf("expected %s for Bar, got %s", expected, *bar)
	}

	if baz := d.HexLiteralEntry("Baz"); baz == nil {
		t.Error("expected Baz entry")
	} else if expected := hex.EncodeToString(data); baz.Value() != expected {
		t.Errorf("Foo value mismatch, expected %d bytes, got %d", len(expected), len(baz.Value()))
	}

	if err := d.Decode(); err != nil {
		t.Fatal(err)
	}

	if expected := "Hello world!"; string(d.Content) != expected {
		t.Errorf("expected stream content %s, got %s", expected, string(d.Content))
	}
}

// TestParseXRefTableEntryZeroOffsetPolicy verifies strict rejection and relaxed notices for in-use entries.
func TestParseXRefTableEntryZeroOffsetPolicy(t *testing.T) {
	for _, tt := range []struct {
		name       string
		mode       int
		objNr      int
		offset     string
		entryType  string
		wantError  bool
		wantNotice bool
	}{
		{"strict", model.ValidationStrict, 4, "0000000000", "n", true, false},
		{"relaxed", model.ValidationRelaxed, 4, "0000000000", "n", false, true},
		{"free", model.ValidationStrict, 4, "0000000000", "f", false, false},
		{"object zero", model.ValidationStrict, 0, "0000000000", "n", false, false},
		{"valid", model.ValidationStrict, 4, "0000000020", "n", false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			conf := model.NewStatelessConfiguration()
			conf.ValidationMode = tt.mode
			ctx, err := model.NewContext(bytes.NewReader(nil), conf)
			if err != nil {
				t.Fatal(err)
			}
			err = parseXRefTableEntry(ctx.XRefTable, []string{tt.offset, "00000", tt.entryType}, tt.objNr, 0, 1)
			if tt.wantError {
				var validationErr *model.ValidationError
				if !errors.As(err, &validationErr) || validationErr.ObjectNumber() != tt.objNr {
					t.Fatalf("expected attributed error for object %d, got %v", tt.objNr, err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			notices := ctx.ValidationReport().Notices()
			if tt.wantNotice {
				if len(notices) != 1 || notices[0].ObjectNumber != tt.objNr ||
					notices[0].Phase != model.NoticePhaseRead || notices[0].Disposition != model.NoticeSkipped {
					t.Fatalf("unexpected notices: %+v", notices)
				}
			} else if len(notices) != 0 {
				t.Fatalf("unexpected notices: %+v", notices)
			}
			wantEntry := !tt.wantError && !tt.wantNotice
			if ctx.Exists(tt.objNr) != wantEntry {
				t.Fatalf("entry present = %t, want %t", ctx.Exists(tt.objNr), wantEntry)
			}
		})
	}
}

// TestResolveObjectStreamErrorsReturnNil checks failed stream construction never returns a placeholder object.
func TestResolveObjectStreamErrorsReturnNil(t *testing.T) {
	for _, tc := range []struct {
		name      string
		dict      types.Dict
		streamInd int
		wantErr   error
	}{
		{"filter", types.Dict{"Filter": types.Integer(5)}, 1, errCorruptFilterArray},
		{"decode parameters", types.Dict{"Filter": types.Name("FlateDecode"), "DecodeParms": types.Integer(5)}, 1, errCorruptDecodeParms},
		{"stream offset", types.Dict{}, 0, errMissingStreamOffset},
	} {
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			t.Run(fmt.Sprintf("%s/mode%d", tc.name, mode), func(t *testing.T) {
				conf := model.NewStatelessConfiguration()
				conf.ValidationMode = mode
				ctx, err := model.NewContext(bytes.NewReader(nil), conf)
				if err != nil {
					t.Fatal(err)
				}
				o, err := resolveObject(t.Context(), ctx, tc.dict, 0, 3, 0, -1, tc.streamInd, 0)
				if o != nil || !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %T, %v; want nil, %v", o, err, tc.wantErr)
				}
			})
		}
	}
}

// TestReadMalformedStreamPolicy checks relaxed skipping, strict errors, and valid stream loading.
func TestReadMalformedStreamPolicy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		entry   string
		wantErr error
	}{
		{"filter", "/Filter 5", errCorruptFilterArray},
		{"decode parameters", "/Filter /FlateDecode/DecodeParms 5", errCorruptDecodeParms},
		{"valid", "", nil},
	} {
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			t.Run(fmt.Sprintf("%s/mode%d", tc.name, mode), func(t *testing.T) {
				conf := model.NewStatelessConfiguration()
				conf.ValidationMode = mode
				ctx, err := Read(t.Context(), bytes.NewReader(selfReferencingStreamPDF(tc.entry)), conf)
				if tc.wantErr != nil && mode == model.ValidationStrict {
					if !errors.Is(err, tc.wantErr) {
						t.Fatalf("got %v, want %v", err, tc.wantErr)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				entry := ctx.Table[3]
				if entry == nil {
					t.Fatal("missing stream xref entry")
				}
				if tc.wantErr != nil {
					if entry.Object != nil {
						t.Fatalf("skipped stream retained object %T", entry.Object)
					}
					return
				}
				sd, ok := entry.Object.(types.StreamDict)
				if !ok || sd.Dict == nil || string(sd.Raw) != "x" {
					t.Fatalf("valid stream not loaded: %T %v", entry.Object, entry.Object)
				}
			})
		}
	}
}

// TestRelaxedStreamReadSkipsOverdeepObject checks parsing stops at the limit before relaxed reading skips the object.
func TestRelaxedStreamReadSkipsOverdeepObject(t *testing.T) {
	pdf := []byte("3 0 obj\n<</Length 1/Filter 4 0 R>>\nstream\nx\nendstream\nendobj\n" +
		"4 0 obj\n<</Length 1/Filter 5 0 R>>\nstream\nx\nendstream\nendobj\n" +
		"5 0 obj\n/ASCIIHexDecode\nendobj\n")
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	conf.Limits.MaxRecursionDepth = 2
	ctx, err := model.NewContext(bytes.NewReader(pdf), conf)
	if err != nil {
		t.Fatal(err)
	}
	for _, objNr := range []int{3, 4, 5} {
		offset := int64(bytes.Index(pdf, []byte(fmt.Sprintf("%d 0 obj", objNr))))
		generation := 0
		ctx.Table[objNr] = &model.XRefTableEntry{Offset: &offset, Generation: &generation}
	}
	if _, err := ParseObject(t.Context(), ctx, 0, 3, 0); !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want recursion limit error", err)
	}
	if err := dereferenceAndLoad(t.Context(), ctx, 3, ctx.Table[3]); err != nil {
		t.Fatalf("relaxed skip: %v", err)
	}
	if ctx.Table[3].Object != nil {
		t.Fatal("retained an object after hitting the recursion limit")
	}
}

// TestReadSkipsOverdeepUnusedObject keeps relaxed recovery while enforcing the parser depth limit in both modes.
func TestReadSkipsOverdeepUnusedObject(t *testing.T) {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	var offsets []int
	for i, body := range []string{
		"<</Type/Catalog/Pages 2 0 R>>",
		"<</Type/Pages/Kids[]/Count 0>>",
		strings.Repeat("[", 20) + "0" + strings.Repeat("]", 20),
	} {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xref := b.Len()
	b.WriteString("xref\n0 4\n0000000000 65535 f \n")
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<</Size 4/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", xref)
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		conf := model.NewStatelessConfiguration()
		conf.ValidationMode = mode
		conf.Limits.MaxRecursionDepth = 8
		ctx, err := Read(t.Context(), bytes.NewReader(b.Bytes()), conf)
		if mode == model.ValidationStrict {
			if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
				t.Fatalf("strict read: got %v, want recursion limit error", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if entry := ctx.Table[3]; entry == nil || entry.Object != nil {
			t.Fatalf("expected skipped object, got %v", entry)
		}
	}
}

// TestRepairScannersRespectObjectLimit checks registration, repair, and scanners resumed after parsing.
func TestRepairScannersRespectObjectLimit(t *testing.T) {
	for _, limit := range []int64{32, 64 << 10} {
		for _, path := range []string{"register", "repair", "resume"} {
			t.Run(fmt.Sprintf("%s/limit=%d", path, limit), func(t *testing.T) {
				input := "1 0 obj true endobj\n" + strings.Repeat("x", int(limit)*2) + "\n"
				conf := model.NewStatelessConfiguration()
				conf.Limits.MaxObjectBytes = limit
				ctx, err := model.NewContext(strings.NewReader(input), conf)
				if err != nil {
					t.Fatal(err)
				}
				switch path {
				case "register":
					err = registerXRefObjects(ctx, 1)
				case "repair":
					err = bypassXrefSection(t.Context(), ctx, 0, nil, 1)
				case "resume":
					var offset int64
					var scanner *bufio.Scanner
					scanner, _, err = processObject(t.Context(), ctx, "1 0 obj", &offset, 1, nil)
					if err != nil {
						t.Fatal(err)
					}
					for scanner.Scan() {
					}
					err = scanner.Err()
				}
				if !errors.Is(err, bufio.ErrTooLong) {
					t.Fatalf("got %v, want scanner buffer limit error", err)
				}
			})
		}
	}
}

// TestRepairScannerLimitBoundaries checks default selection, growth, and small buffer boundaries.
func TestRepairScannerLimitBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name  string
		limit int64
		size  int
		fail  bool
	}{
		{"default", 0, 2 * defaultBufSize, false},
		{"grow", 4 * defaultBufSize, 2 * defaultBufSize, false},
		{"small fits", 32, 31, false},
		{"small exceeds", 32, 32, true},
		{"negative", -1, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conf := model.NewStatelessConfiguration()
			conf.Limits.MaxObjectBytes = tc.limit
			ctx, err := model.NewContext(strings.NewReader(""), conf)
			if err != nil {
				t.Fatal(err)
			}
			s := newRepairScanner(ctx, strings.NewReader(strings.Repeat("x", tc.size)+"\n"))
			if s.Scan() == tc.fail {
				t.Fatalf("scan result: err=%v, want failure=%t", s.Err(), tc.fail)
			}
			if tc.fail && !errors.Is(s.Err(), bufio.ErrTooLong) {
				t.Fatalf("got %v, want scanner buffer limit error", s.Err())
			}
		})
	}
}

// TestRepairKeepsObjectsAfterMalformedSkip checks consecutive skips and subsequent dictionaries and streams.
func TestRepairKeepsObjectsAfterMalformedSkip(t *testing.T) {
	bad := "1 0 obj\n<</A ]>>\nendobj\n"
	stream := "4 0 obj\n<</Length 3>>\nstream\nabc\nendstream\nendobj\n"
	for _, tc := range []struct {
		name, objects string
		stream        bool
	}{
		{"catalog", bad, false},
		{"consecutive malformed", bad + "5 0 obj\n<</B ]>>\nendobj\n", false},
		{"stream", bad + stream, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := "%PDF-1.4\n" + tc.objects +
				"2 0 obj\n<</Type/Catalog/Pages 3 0 R>>\nendobj\n" +
				"3 0 obj\n<</Type/Pages/Kids[]/Count 0>>\nendobj\n"
			conf := model.NewStatelessConfiguration()
			conf.ValidationMode = model.ValidationRelaxed
			ctx, err := Read(t.Context(), strings.NewReader(input), conf)
			if err != nil {
				t.Fatal(err)
			}
			if ctx.Root == nil || ctx.Root.ObjectNumber.Value() != 2 {
				t.Fatalf("root = %v, want 2 0 R", ctx.Root)
			}
			for _, objNr := range []int{2, 3} {
				if e := ctx.Table[objNr]; e == nil || e.Object == nil {
					t.Fatalf("obj#%d was lost", objNr)
				}
			}
			if tc.stream {
				sd, _, err := ctx.DereferenceStreamDict(*types.NewIndirectRef(4, 0))
				if err != nil || sd == nil || sd.StreamLength == nil || string(sd.Raw) != "abc" {
					t.Fatalf("stream after skipped object was not loaded: %v, %v", sd, err)
				}
			}
		})
	}
}

// TestProcessObjectResumesWithinValidObject checks the repair state after a successful parse.
func TestProcessObjectResumesWithinValidObject(t *testing.T) {
	input := "1 0 obj\ntrue\nendobj\n"
	ctx, err := model.NewContext(strings.NewReader(input), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	var offset int64
	s, withinObj, err := processObject(t.Context(), ctx, "1 0 obj", &offset, 1, nil)
	if err != nil || s == nil || !withinObj {
		t.Fatalf("scanner=%v, withinObj=%t, err=%v; want a scanner inside the object", s, withinObj, err)
	}
}

func assertUnknownXRefEntry(t *testing.T, entry model.XRefTableEntry) {
	t.Helper()
	if !entry.Free || entry.Compressed || entry.Object != nil || entry.Incr != 2 {
		t.Fatalf("unexpected null entry: %+v", entry)
	}
	if entry.Offset == nil || *entry.Offset != 0 || entry.Generation == nil || *entry.Generation != 0 {
		t.Fatal("unknown entry must have initialized zero offset and generation")
	}
}

// TestXRefStreamUnknownEntryTypes verifies null resolution and safe free-list state for unknown types.
func TestXRefStreamUnknownEntryTypes(t *testing.T) {
	for _, entryType := range []int64{3, 255, 256, math.MaxInt64, -1} {
		t.Run(fmt.Sprint(entryType), func(t *testing.T) {
			ctx, err := model.NewContext(bytes.NewReader(nil), nil)
			if err != nil {
				t.Fatal(err)
			}
			entry := createXRefTableEntryFromXRefStream(entryType, 7, 123, 456, 789, ctx.Read.ObjectStreams, 2)
			assertUnknownXRefEntry(t, entry)
			if len(ctx.Read.ObjectStreams) != 0 {
				t.Fatal("unknown entry registered an object stream")
			}
			ctx.Table[7] = &entry
			obj, err := ctx.Dereference(*types.NewIndirectRef(7, 0))
			if err != nil || obj != nil {
				t.Fatalf("dereference = %v, %v; want null", obj, err)
			}
			// An older in-use entry must not replace the newer null entry.
			xsd := &types.XRefStreamDict{Objects: []int{7}, W: [3]int{1, 1, 1}}
			if err := extractXRefTableEntriesFromXRefStream([]byte{1, 42, 0}, 0, xsd, ctx, 3); err != nil {
				t.Fatal(err)
			}
			if ctx.Table[7] != &entry || !ctx.Table[7].Free {
				t.Fatal("older entry replaced the null entry")
			}
			if err := ctx.EnsureValidFreeList(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func xrefStreamUnknownTypePDF(version string, objNr, entryType int, encrypt bool) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%%PDF-%s\n", version)
	offsets := make([]int, 5)
	for i, body := range []string{"<</Type/Catalog/Pages 2 0 R>>", "<</Type/Pages/Kids[]/Count 0>>"} {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	offsets[4] = b.Len()
	rows := [][3]int{{0, 0, 65535}, {1, offsets[1], 0}, {1, offsets[2], 0}, {0, 0, 0}, {1, offsets[4], 0}}
	rows[objNr] = [3]int{entryType, 123456, 42}
	var data bytes.Buffer
	for _, row := range rows {
		data.WriteByte(byte(row[0]))
		data.Write([]byte{byte(row[1] >> 24), byte(row[1] >> 16), byte(row[1] >> 8), byte(row[1])})
		data.Write([]byte{byte(row[2] >> 8), byte(row[2])})
	}
	extra := ""
	if encrypt {
		extra = "/Encrypt 3 0 R/ID[<00><00>]"
	}
	fmt.Fprintf(&b, "4 0 obj\n<</Type/XRef/Size 5/W[1 4 2]/Root 1 0 R%s/Length %d>>\nstream\n", extra, data.Len())
	b.Write(data.Bytes())
	b.WriteString("\nendstream\nendobj\n")
	fmt.Fprintf(&b, "startxref\n%d\n%%%%EOF\n", offsets[4])
	return b.Bytes()
}

// TestReadXRefStreamUnknownEntryType checks the reported panic paths and an unreferenced unknown entry.
func TestReadXRefStreamUnknownEntryType(t *testing.T) {
	for _, version := range []string{"1.5", "1.7", "2.0"} {
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			for _, tt := range []struct {
				name    string
				objNr   int
				encrypt bool
			}{
				{"free list head", 0, false},
				{"unreferenced", 3, false},
				{"encryption dictionary", 3, true},
			} {
				t.Run(fmt.Sprintf("%s/mode=%d/%s", version, mode, tt.name), func(t *testing.T) {
					conf := model.NewDefaultConfiguration()
					conf.ValidationMode = mode
					pdf := xrefStreamUnknownTypePDF(version, tt.objNr, 255, tt.encrypt)
					ctx, err := Read(t.Context(), bytes.NewReader(pdf), conf)
					if tt.encrypt {
						if !errors.Is(err, ErrMalformedEncryption) {
							t.Fatalf("got %v, want malformed encryption error", err)
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					obj, err := ctx.Dereference(*types.NewIndirectRef(tt.objNr, 0))
					if err != nil || obj != nil {
						t.Fatalf("dereference = %v, %v; want null", obj, err)
					}
				})
			}
		}
	}
}
