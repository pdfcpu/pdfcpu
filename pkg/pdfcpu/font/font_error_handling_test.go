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

package font

import (
	"encoding/binary"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	corefont "github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validTTFLight() corefont.TTFLight {
	return corefont.TTFLight{
		PostscriptName:  "Demo",
		UnitsPerEm:      1000,
		FirstChar:       'A',
		LastChar:        'A',
		LLx:             -100,
		LLy:             -200,
		URx:             1000,
		URy:             800,
		HorMetricsCount: 2,
		GlyphCount:      2,
		GlyphWidths:     []int{500, 600},
		Chars:           map[uint32]uint16{'A': 1},
		ToUnicode:       map[uint16]uint32{1: 'A'},
		Planes:          map[int]bool{0: true},
	}
}

func minimalSubsetFontData() []byte {
	const (
		tableCount     = 4
		directorySize  = tableCount * 16
		firstTable     = 12 + directorySize
		headOffset     = firstTable
		locaOffset     = headOffset + 52
		maxpOffset     = locaOffset + 4
		fontFileLength = maxpOffset + 8
	)
	bb := make([]byte, fontFileLength)
	copy(bb, []byte{0x00, 0x01, 0x00, 0x00})
	binary.BigEndian.PutUint16(bb[4:], tableCount)
	tables := []struct {
		tag    string
		offset uint32
		size   uint32
	}{
		{"glyf", firstTable, 0},
		{"head", headOffset, 52},
		{"loca", locaOffset, 4},
		{"maxp", maxpOffset, 6},
	}
	for i, table := range tables {
		off := 12 + i*16
		copy(bb[off:], table.tag)
		binary.BigEndian.PutUint32(bb[off+8:], table.offset)
		binary.BigEndian.PutUint32(bb[off+12:], table.size)
	}
	binary.BigEndian.PutUint16(bb[maxpOffset+4:], 1)
	return bb
}

type installedFontFixture struct {
	PostscriptName  string
	UnitsPerEm      int
	FirstChar       uint16
	LastChar        uint16
	LLx             float64
	LLy             float64
	URx             float64
	URy             float64
	HorMetricsCount int
	GlyphCount      int
	GlyphWidths     []int
	Chars           map[uint32]uint16
	ToUnicode       map[uint16]uint32
	Planes          map[int]bool
	FontFile        []byte
}

func userFontDictMissingReference(missing string) types.Dict {
	indRef := types.NewIndirectRef(7, 0)
	fd := types.Dict{}
	if missing != "FontFile2" {
		fd["FontFile2"] = *indRef
	}
	if missing != "CIDSet" {
		fd["CIDSet"] = *indRef
	}
	descendant := types.Dict{"FontDescriptor": fd}
	if missing != "W" {
		descendant["W"] = *indRef
	}
	return types.Dict{
		"Name":            types.Name("Demo"),
		"Encoding":        types.Name("Identity-H"),
		"ToUnicode":       *indRef,
		"DescendantFonts": types.Array{descendant},
	}
}

func installUpdateUserfontMetrics(t *testing.T) {
	t.Helper()
	originalDir := corefont.UserFontDir
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "Demo.gob"))
	if err != nil {
		t.Fatal(err)
	}
	ttf := validTTFLight()
	fixture := installedFontFixture{
		PostscriptName:  ttf.PostscriptName,
		UnitsPerEm:      ttf.UnitsPerEm,
		FirstChar:       ttf.FirstChar,
		LastChar:        ttf.LastChar,
		LLx:             ttf.LLx,
		LLy:             ttf.LLy,
		URx:             ttf.URx,
		URy:             ttf.URy,
		HorMetricsCount: ttf.HorMetricsCount,
		GlyphCount:      ttf.GlyphCount,
		GlyphWidths:     ttf.GlyphWidths,
		Chars:           ttf.Chars,
		ToUnicode:       ttf.ToUnicode,
		Planes:          ttf.Planes,
		FontFile:        minimalSubsetFontData(),
	}
	if err := gob.NewEncoder(f).Encode(fixture); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	corefont.UserFontDir = dir
	if err := corefont.ReloadUserFonts(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		corefont.UserFontDir = originalDir
		if err := corefont.ReloadUserFonts(); err != nil {
			t.Errorf("restore user fonts: %v", err)
		}
	})
}

func updateUserfontFixture() (*model.XRefTable, model.FontResource) {
	toUnicode := types.NewIndirectRef(7, 0)
	fontFile := types.NewIndirectRef(8, 0)
	widths := types.NewIndirectRef(9, 0)
	cidSet := types.NewIndirectRef(10, 0)
	xRefTable := &model.XRefTable{
		Table: map[int]*model.XRefTableEntry{
			7:  model.NewXRefTableEntryGen0(types.StreamDict{Dict: types.Dict{}, Raw: []byte("malformed cmap")}),
			8:  model.NewXRefTableEntryGen0(types.StreamDict{Dict: types.Dict{}, Raw: []byte("font data")}),
			9:  model.NewXRefTableEntryGen0(types.Array{types.Integer(500)}),
			10: model.NewXRefTableEntryGen0(types.StreamDict{Dict: types.Dict{}, Raw: []byte{0x80}}),
		},
		UsedGIDs: map[string]map[uint16]bool{},
	}
	return xRefTable, model.FontResource{
		FontFile:  fontFile,
		ToUnicode: toUnicode,
		W:         widths,
		CIDSet:    cidSet,
	}
}

func cloneXRefEntries(table map[int]*model.XRefTableEntry) map[int]*model.XRefTableEntry {
	cloned := make(map[int]*model.XRefTableEntry, len(table))
	for objNr, entry := range table {
		e := *entry
		if entry.Object != nil {
			e.Object = entry.Object.Clone()
		}
		cloned[objNr] = &e
	}
	return cloned
}

func TestIndRefsForUserfontUpdateRequiresUpdateReferences(t *testing.T) {
	for _, missing := range []string{"FontFile2", "W", "CIDSet"} {
		t.Run(missing, func(t *testing.T) {
			fontResource := model.FontResource{}
			err := IndRefsForUserfontUpdate(&model.XRefTable{}, userFontDictMissingReference(missing), "", &fontResource)
			if !errors.Is(err, ErrCorruptFontDict) {
				t.Fatalf("expected %v, got %v", ErrCorruptFontDict, err)
			}
			for _, context := range []string{"font Demo", "inspect user font references", missing} {
				if !strings.Contains(err.Error(), context) {
					t.Fatalf("expected context %q, got %q", context, err)
				}
			}
		})
	}
}

func TestUpdateUserfontRequiresUpdateReferences(t *testing.T) {
	installUpdateUserfontMetrics(t)
	for _, missing := range []string{"ToUnicode", "FontFile", "W", "CIDSet"} {
		t.Run(missing, func(t *testing.T) {
			xRefTable, fontResource := updateUserfontFixture()
			switch missing {
			case "ToUnicode":
				fontResource.ToUnicode = nil
			case "FontFile":
				fontResource.FontFile = nil
			case "W":
				fontResource.W = nil
			case "CIDSet":
				fontResource.CIDSet = nil
			}
			before := cloneXRefEntries(xRefTable.Table)
			err := UpdateUserfont(xRefTable, "Demo", fontResource)
			if err == nil {
				t.Error("expected missing-reference error")
				return
			}
			if !errors.Is(err, ErrCorruptFontDict) {
				t.Errorf("expected %v, got %v", ErrCorruptFontDict, err)
			}
			for _, context := range []string{"font Demo", "update user font", missing} {
				if !strings.Contains(err.Error(), context) {
					t.Errorf("expected context %q, got %q", context, err)
				}
			}
			if !reflect.DeepEqual(cloneXRefEntries(xRefTable.Table), before) {
				t.Error("xref entries changed before required references were validated")
			}
			if len(xRefTable.UsedGIDs) != 0 {
				t.Fatalf("used glyphs changed before required references were validated: %v", xRefTable.UsedGIDs)
			}
		})
	}
}

func TestUpdateUserfontUpdatesExistingObjectsInPlace(t *testing.T) {
	installUpdateUserfontMetrics(t)
	xRefTable, fontResource := updateUserfontFixture()
	toUnicode := xRefTable.Table[7].Object.(types.StreamDict)
	toUnicode.Raw = []byte("endcodespacerange\n1 beginbfchar\n<0000> <0000>\nendbfchar\nendcmap")
	xRefTable.Table[7].Object = toUnicode
	xRefTable.Table[9].Object = types.Array{types.Integer(999)}
	before := cloneXRefEntries(xRefTable.Table)

	if err := UpdateUserfont(xRefTable, "Demo", fontResource); err != nil {
		t.Fatal(err)
	}
	if len(xRefTable.Table) != len(before) {
		t.Fatalf("expected %d existing xref entries, got %d", len(before), len(xRefTable.Table))
	}
	for objNr, wantType := range map[int]any{
		7:  types.StreamDict{},
		8:  types.StreamDict{},
		9:  types.Array{},
		10: types.StreamDict{},
	} {
		entry := xRefTable.Table[objNr]
		if reflect.TypeOf(entry.Object) != reflect.TypeOf(wantType) {
			t.Fatalf("obj#%d: expected %T, got %T", objNr, wantType, entry.Object)
		}
		if reflect.DeepEqual(entry.Object.Clone(), before[objNr].Object) {
			t.Errorf("obj#%d was not updated", objNr)
		}
	}
	if !xRefTable.UsedGIDs["Demo"][0] {
		t.Fatalf("expected glyph ID 0 from existing ToUnicode CMap, got %v", xRefTable.UsedGIDs)
	}
}

func TestReferencedFontObjectRejectsMissingXRefEntry(t *testing.T) {
	indRef := types.NewIndirectRef(7, 0)
	xRefTable := &model.XRefTable{Table: map[int]*model.XRefTableEntry{}}
	_, _, err := referencedFontObject(xRefTable, "Demo", "update subset stream", indRef, fontStreamObject)
	if !errors.Is(err, ErrCorruptFontDict) {
		t.Fatalf("expected %v, got %v", ErrCorruptFontDict, err)
	}
	for _, context := range []string{"font Demo", "update subset stream", "missing xref entry"} {
		if !strings.Contains(err.Error(), context) {
			t.Fatalf("expected context %q, got %q", context, err)
		}
	}
}

func TestReferencedFontObjectRejectsWrongObjectTypes(t *testing.T) {
	for _, tt := range []struct {
		name     string
		object   types.Object
		expected referencedFontObjectType
		want     string
	}{
		{name: "stream", object: types.Dict{}, expected: fontStreamObject, want: "stream dictionary"},
		{name: "array", object: types.StreamDict{}, expected: fontArrayObject, want: "array"},
		{name: "dictionary", object: types.Array{}, expected: fontDictObject, want: "dictionary"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			indRef := types.NewIndirectRef(7, 0)
			xRefTable := &model.XRefTable{Table: map[int]*model.XRefTableEntry{
				7: model.NewXRefTableEntryGen0(tt.object),
			}}
			_, _, err := referencedFontObject(xRefTable, "Demo", "update "+tt.name, indRef, tt.expected)
			if !errors.Is(err, ErrCorruptFontDict) {
				t.Fatalf("expected %v, got %v", ErrCorruptFontDict, err)
			}
			if !strings.Contains(err.Error(), "expected "+tt.want) {
				t.Fatalf("expected object-type context, got %q", err)
			}
		})
	}
}

func TestReferencedFontObjectRejectsMissingInputs(t *testing.T) {
	_, _, err := referencedFontObject(nil, "Demo", "update CID set", types.NewIndirectRef(7, 0), fontStreamObject)
	if !errors.Is(err, model.ErrMissingXRefTable) {
		t.Fatalf("expected %v, got %v", model.ErrMissingXRefTable, err)
	}
	if !errors.Is(err, ErrCorruptFontDict) {
		t.Fatalf("expected %v, got %v", ErrCorruptFontDict, err)
	}
	_, _, err = referencedFontObject(&model.XRefTable{}, "Demo", "update CID set", nil, fontStreamObject)
	if !errors.Is(err, ErrCorruptFontDict) || !strings.Contains(err.Error(), "missing indirect reference") {
		t.Fatalf("expected missing-reference corruption error, got %v", err)
	}
}

func TestCIDSetRejectsGlyphIDOutsideGlyphCount(t *testing.T) {
	xRefTable := &model.XRefTable{
		UsedGIDs: map[string]map[uint16]bool{"Demo": {2: true}},
	}
	_, err := CIDSet(xRefTable, validTTFLight(), "Demo", nil)
	if !errors.Is(err, ErrCorruptFontDict) {
		t.Fatalf("expected %v, got %v", ErrCorruptFontDict, err)
	}
	if !strings.Contains(err.Error(), "glyph ID 2 outside 0..1") {
		t.Fatalf("expected glyph bounds context, got %q", err)
	}
}

func TestReferencedFontUpdatesUseCheckedObjectBoundary(t *testing.T) {
	indRef := types.NewIndirectRef(7, 0)
	ttf := validTTFLight()
	tests := []struct {
		name  string
		phase string
		fn    func(*model.XRefTable) error
	}{
		{
			name:  "subset stream",
			phase: "update subset stream",
			fn: func(xRefTable *model.XRefTable) error {
				_, err := ttfSubFontFile(xRefTable, "Demo", indRef)
				return err
			},
		},
		{
			name:  "CID set",
			phase: "update CID set",
			fn: func(xRefTable *model.XRefTable) error {
				_, err := CIDSet(xRefTable, ttf, "Demo", indRef)
				return err
			},
		},
		{
			name:  "CID widths",
			phase: "update CID widths",
			fn: func(xRefTable *model.XRefTable) error {
				_, err := CIDWidths(xRefTable, ttf, "Demo", true, indRef)
				return err
			},
		},
		{
			name:  "ToUnicode map",
			phase: "update ToUnicode map",
			fn: func(xRefTable *model.XRefTable) error {
				_, err := toUnicodeCMap(xRefTable, ttf, "Demo", true, indRef)
				return err
			},
		},
		{
			name:  "Type0 font",
			phase: "update Type0 font",
			fn: func(xRefTable *model.XRefTable) error {
				_, err := type0FontDict(xRefTable, "Demo", "", "", indRef)
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xRefTable := &model.XRefTable{
				Table:    map[int]*model.XRefTableEntry{7: model.NewXRefTableEntryGen0(types.Name("wrong"))},
				UsedGIDs: map[string]map[uint16]bool{"Demo": {0: true}},
			}
			err := tt.fn(xRefTable)
			if !errors.Is(err, ErrCorruptFontDict) {
				t.Fatalf("expected %v, got %v", ErrCorruptFontDict, err)
			}
			for _, context := range []string{"font Demo", tt.phase} {
				if !strings.Contains(err.Error(), context) {
					t.Fatalf("expected context %q, got %q", context, err)
				}
			}
		})
	}
}

func TestExportedFontConstructorsRejectNilXRef(t *testing.T) {
	ttf := validTTFLight()
	tests := []struct {
		name  string
		phase string
		fn    func() error
	}{
		{"PDFDoc encoding", "create PDFDoc encoding", func() error { _, err := PDFDocEncoding(nil); return err }},
		{"core font", "create core font dictionary", func() error { _, err := CoreFontDict(nil, "Helvetica"); return err }},
		{"CID set", "update CID set", func() error { _, err := CIDSet(nil, ttf, "Demo", nil); return err }},
		{"CID font file", "create CID font file", func() error { _, err := CIDFontFile(nil, "Demo", false); return err }},
		{"CID descriptor", "create CID font descriptor", func() error {
			_, err := CIDFontDescriptor(nil, ttf, "Demo", "Demo", "", false)
			return err
		}},
		{"TrueType descriptor", "create TrueType font descriptor", func() error {
			_, err := NewFontDescriptor(nil, ttf, "Demo", "")
			return err
		}},
		{"CID widths", "update CID widths", func() error { _, err := CIDWidths(nil, ttf, "Demo", false, nil); return err }},
		{"TrueType widths", "create TrueType widths", func() error { _, err := Widths(nil, ttf, 0, 1); return err }},
		{"CID font dictionary", "create CID font dictionary", func() error {
			_, err := CIDFontDict(nil, ttf, "Demo", "Demo", "", nil)
			return err
		}},
		{"ensure font dictionary", "ensure font dictionary", func() error {
			_, err := EnsureFontDict(nil, "Helvetica", "", "", false, nil)
			return err
		}},
		{"font resources", "create font resources", func() error { _, err := FontResources(nil, nil); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, model.ErrMissingXRefTable) {
				t.Fatalf("expected %v, got %v", model.ErrMissingXRefTable, err)
			}
			if !strings.Contains(err.Error(), tt.phase) {
				t.Fatalf("expected phase %q, got %q", tt.phase, err)
			}
		})
	}
}

func TestTTFLightInvariantValidation(t *testing.T) {
	tests := []struct {
		name string
		edit func(*corefont.TTFLight)
		want string
	}{
		{"glyph count", func(ttf *corefont.TTFLight) { ttf.GlyphCount = 0 }, "glyph count"},
		{"width count", func(ttf *corefont.TTFLight) { ttf.GlyphWidths = ttf.GlyphWidths[:1] }, "glyph width count"},
		{"character map", func(ttf *corefont.TTFLight) { ttf.Chars = nil }, "character map"},
		{"character glyph", func(ttf *corefont.TTFLight) { ttf.Chars['A'] = 2 }, "character U+0041"},
		{"ToUnicode map", func(ttf *corefont.TTFLight) { ttf.ToUnicode = nil }, "ToUnicode map"},
		{"ToUnicode glyph", func(ttf *corefont.TTFLight) { ttf.ToUnicode[2] = 'B' }, "ToUnicode glyph ID 2"},
		{"planes map", func(ttf *corefont.TTFLight) { ttf.Planes = nil }, "Unicode planes map"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttf := validTTFLight()
			tt.edit(&ttf)
			_, err := Widths(&model.XRefTable{}, ttf, 0, 1)
			if !errors.Is(err, corefont.ErrInvalidFontData) {
				t.Fatalf("expected %v, got %v", corefont.ErrInvalidFontData, err)
			}
			for _, context := range []string{"font Demo", "create TrueType widths", tt.want} {
				if !strings.Contains(err.Error(), context) {
					t.Fatalf("expected context %q, got %q", context, err)
				}
			}
		})
	}
}

func TestPublicEmbeddingBoundariesValidateTTFLight(t *testing.T) {
	ttf := validTTFLight()
	ttf.Planes = nil
	xRefTable := &model.XRefTable{UsedGIDs: map[string]map[uint16]bool{}}
	tests := []struct {
		name string
		fn   func() error
	}{
		{"CID set", func() error { _, err := CIDSet(xRefTable, ttf, "Demo", nil); return err }},
		{"CID descriptor", func() error {
			_, err := CIDFontDescriptor(xRefTable, ttf, "Demo", "Demo", "", false)
			return err
		}},
		{"TrueType descriptor", func() error {
			_, err := NewFontDescriptor(xRefTable, ttf, "Demo", "")
			return err
		}},
		{"CID widths", func() error { _, err := CIDWidths(xRefTable, ttf, "Demo", false, nil); return err }},
		{"TrueType widths", func() error { _, err := Widths(xRefTable, ttf, 0, 1); return err }},
		{"CID font dictionary", func() error {
			_, err := CIDFontDict(xRefTable, ttf, "Demo", "Demo", "", nil)
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, corefont.ErrInvalidFontData) {
				t.Fatalf("expected %v, got %v", corefont.ErrInvalidFontData, err)
			}
		})
	}
}

func TestFontObjectInsertionPreservesFontAndPhase(t *testing.T) {
	xRefTable := &model.XRefTable{Table: map[int]*model.XRefTableEntry{
		0: model.NewXRefTableEntryGen0(nil),
	}}
	_, err := flateEncodedStreamIndRef(xRefTable, "Demo", "create test stream", []byte("data"))
	if err == nil {
		t.Fatal("expected insertion error")
	}
	for _, context := range []string{"font Demo", "create test stream", "insert object", "not free"} {
		if !strings.Contains(err.Error(), context) {
			t.Fatalf("expected context %q, got %q", context, err)
		}
	}
}

func TestConstructorFailuresPreserveFontAndPhase(t *testing.T) {
	ttf := validTTFLight()
	newBrokenXRef := func() *model.XRefTable {
		return &model.XRefTable{
			Table:    map[int]*model.XRefTableEntry{0: model.NewXRefTableEntryGen0(nil)},
			UsedGIDs: map[string]map[uint16]bool{},
		}
	}
	tests := []struct {
		name  string
		phase string
		fn    func() error
	}{
		{"descriptor", "create CID font descriptor", func() error {
			_, err := CIDFontDescriptor(newBrokenXRef(), ttf, "Demo", "Demo", "", false)
			return err
		}},
		{"widths", "create CID widths", func() error {
			_, err := CIDWidths(newBrokenXRef(), ttf, "Demo", false, nil)
			return err
		}},
		{"CID dictionary", "create CID font dictionary", func() error {
			_, err := CIDFontDict(newBrokenXRef(), ttf, "Demo", "Demo", "", &cjk{})
			return err
		}},
		{"resources", "create font resources", func() error {
			_, err := FontResources(newBrokenXRef(), model.FontMap{
				"Helvetica": {Res: model.Resource{ID: "F0"}},
			})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected constructor failure")
			}
			for _, context := range []string{tt.phase, "insert object", "not free"} {
				if !strings.Contains(err.Error(), context) {
					t.Fatalf("expected context %q, got %q", context, err)
				}
			}
		})
	}
}
