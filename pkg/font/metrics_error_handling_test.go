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
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"maps"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeUserFontMetric(t *testing.T, dir, name string) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, name+".gob"))
	if err != nil {
		t.Fatal(err)
	}
	fd := TTFLight{
		PostscriptName:  name,
		UnitsPerEm:      1000,
		LLx:             -100,
		LLy:             -200,
		URx:             1000,
		URy:             800,
		HorMetricsCount: 1,
		GlyphCount:      1,
		GlyphWidths:     []int{500},
		Chars:           map[uint32]uint16{'A': 0},
		ToUnicode:       map[uint16]uint32{0: 'A'},
		Planes:          map[int]bool{0: true},
	}
	if err := gob.NewEncoder(f).Encode(fd); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSupportsScriptChecksLaterUnicodeRangeBits(t *testing.T) {
	for _, tt := range []struct {
		script string
		bit    int
	}{
		{"JPAN", 50},
		{"KORE", 56},
		{"HANG", 52},
		{"LATN", 3},
	} {
		t.Run(tt.script, func(t *testing.T) {
			fd := TTFLight{}
			fd.UnicodeRange[tt.bit/32] = 1 << (tt.bit % 32)
			ok, err := fd.SupportsScript(tt.script)
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Fatalf("expected %s support from later bit %d", tt.script, tt.bit)
			}
		})
	}
}

func TestUserFontReturnsDetachedMetrics(t *testing.T) {
	originalDir := UserFontDir
	UserFontDir = t.TempDir()
	writeUserFontMetric(t, UserFontDir, "Detached")
	if err := ReloadUserFonts(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		UserFontDir = originalDir
		if err := ReloadUserFonts(); err != nil {
			t.Errorf("restore user fonts: %v", err)
		}
	})

	got, ok, err := UserFont("Detached")
	if err != nil || !ok {
		t.Fatalf("load detached font: ok=%t err=%v", ok, err)
	}
	got.GlyphWidths[0] = 999
	got.Chars['A'] = 1
	got.ToUnicode[0] = 'B'
	got.Planes[0] = false

	got, ok, err = UserFont("Detached")
	if err != nil || !ok {
		t.Fatalf("reload detached font: ok=%t err=%v", ok, err)
	}
	if got.GlyphWidths[0] != 500 || got.Chars['A'] != 0 || got.ToUnicode[0] != 'A' || !got.Planes[0] {
		t.Fatalf("registry metrics were mutated through returned value: %+v", got)
	}
}

func TestReloadUserFontsRefreshesMetrics(t *testing.T) {
	originalDir := UserFontDir
	t.Cleanup(func() {
		UserFontDir = originalDir
		if err := ReloadUserFonts(); err != nil {
			t.Errorf("restore user fonts: %v", err)
		}
	})

	UserFontDir = t.TempDir()
	writeUserFontMetric(t, UserFontDir, "First")
	if err := ReloadUserFonts(); err != nil {
		t.Fatal(err)
	}
	ok, err := IsUserFont("First")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected first font after reload")
	}

	writeUserFontMetric(t, UserFontDir, "Second")
	if err := ReloadUserFonts(); err != nil {
		t.Fatal(err)
	}
	ok, err = IsUserFont("Second")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected newly installed font after reload")
	}
}

func TestReloadUserFontsReplacesMetricsAtomically(t *testing.T) {
	originalDir := UserFontDir
	userFontMetricsLock.Lock()
	originalMetrics := userFontMetrics
	userFontMetrics = map[string]TTFLight{"Existing": {PostscriptName: "Existing"}}
	userFontMetricsLock.Unlock()
	t.Cleanup(func() {
		UserFontDir = originalDir
		userFontMetricsLock.Lock()
		userFontMetrics = originalMetrics
		userFontMetricsLock.Unlock()
	})

	UserFontDir = t.TempDir()
	if err := os.WriteFile(filepath.Join(UserFontDir, "Malformed.gob"), []byte("not a gob"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ReloadUserFonts(); err == nil {
		t.Fatal("expected malformed metrics error")
	}

	userFontMetricsLock.RLock()
	_, existing := userFontMetrics["Existing"]
	_, malformed := userFontMetrics["Malformed"]
	userFontMetricsLock.RUnlock()
	if !existing || malformed {
		t.Fatalf("expected only previous metrics to remain intact, existing=%t malformed=%t", existing, malformed)
	}
}

func truncatedInstalledGob(t *testing.T, fileName string) {
	t.Helper()
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(ttf{PostscriptName: "Broken", FontFile: []byte("font data")}); err != nil {
		t.Fatal(err)
	}
	bb := b.Bytes()
	if err := os.WriteFile(fileName, bb[:len(bb)-1], 0600); err != nil {
		t.Fatal(err)
	}
}

func TestInstalledGobReadersPreserveInvalidDataAndDecodeCause(t *testing.T) {
	dir := t.TempDir()
	fileName := filepath.Join(dir, "Broken.gob")
	truncatedInstalledGob(t, fileName)
	originalDir := UserFontDir
	UserFontDir = dir
	t.Cleanup(func() { UserFontDir = originalDir })

	tests := []struct {
		name string
		fn   func() error
	}{
		{"load", func() error { return load(fileName, &TTFLight{}) }},
		{"Read", func() error {
			_, err := Read("Broken")
			return err
		}},
		{"readGob", func() error { return readGob(fileName, &ttf{}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, ErrInvalidFontData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
			}
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("expected preserved gob cause %v, got %v", io.ErrUnexpectedEOF, err)
			}
		})
	}
}

func TestInstalledGobReadersEnforceSizeLimit(t *testing.T) {
	dir := t.TempDir()
	fileName := filepath.Join(dir, "Oversized.gob")
	if err := os.WriteFile(fileName, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(fileName, maxInstalledFontSize+1); err != nil {
		t.Fatal(err)
	}
	originalDir := UserFontDir
	UserFontDir = dir
	t.Cleanup(func() { UserFontDir = originalDir })

	tests := []struct {
		name string
		fn   func() error
	}{
		{"load", func() error { return load(fileName, &TTFLight{}) }},
		{"Read", func() error {
			_, err := Read("Oversized")
			return err
		}},
		{"readGob", func() error { return readGob(fileName, &ttf{}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, ErrInvalidFontData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
			}
		})
	}
}

func TestValidateTTFLightRejectsSemanticCorruption(t *testing.T) {
	valid := TTFLight{
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
	tests := []struct {
		name string
		edit func(*TTFLight)
		want string
	}{
		{"PostScript name missing", func(fd *TTFLight) { fd.PostscriptName = "" }, "PostScript name"},
		{"PostScript name invalid", func(fd *TTFLight) { fd.PostscriptName = "Bad Name" }, "PostScript name"},
		{"units per em", func(fd *TTFLight) { fd.UnitsPerEm = 0 }, "units per em"},
		{"glyph count", func(fd *TTFLight) { fd.GlyphCount = 0 }, "glyph count"},
		{"horizontal metrics count", func(fd *TTFLight) { fd.HorMetricsCount = 0 }, "horizontal metrics count"},
		{"width count", func(fd *TTFLight) { fd.GlyphWidths = fd.GlyphWidths[:1] }, "glyph width count"},
		{"glyph width", func(fd *TTFLight) { fd.GlyphWidths[1] = -1 }, "glyph ID 1 width"},
		{"bounding box order", func(fd *TTFLight) { fd.URx = fd.LLx }, "font bounding box"},
		{"bounding box finite", func(fd *TTFLight) { fd.LLy = math.NaN() }, "font bounding box LLy"},
		{"character range", func(fd *TTFLight) { fd.FirstChar, fd.LastChar = 'B', 'A' }, "first character"},
		{"character map", func(fd *TTFLight) { fd.Chars = nil }, "character map"},
		{"character glyph", func(fd *TTFLight) { fd.Chars['A'] = 2 }, "character U+0041"},
		{"character Unicode", func(fd *TTFLight) { fd.Chars[0x110000] = 1 }, "Unicode scalar"},
		{"ToUnicode map", func(fd *TTFLight) { fd.ToUnicode = nil }, "ToUnicode map"},
		{"ToUnicode glyph", func(fd *TTFLight) { fd.ToUnicode[2] = 'B' }, "ToUnicode glyph ID 2"},
		{"ToUnicode scalar", func(fd *TTFLight) { fd.ToUnicode[1] = 0xD800 }, "non-scalar"},
		{"planes", func(fd *TTFLight) { fd.Planes = nil }, "Unicode planes map"},
		{"empty planes", func(fd *TTFLight) { clear(fd.Planes) }, "empty Unicode planes map"},
		{"plane range", func(fd *TTFLight) { fd.Planes[17] = true }, "Unicode plane 17"},
		{"plane value", func(fd *TTFLight) { fd.Planes[0] = false }, "marked unused"},
		{"plane coverage", func(fd *TTFLight) {
			fd.Chars[0x10000] = 1
			fd.ToUnicode[1] = 0x10000
		}, "no Unicode plane entry"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fd := valid
			fd.GlyphWidths = append([]int(nil), valid.GlyphWidths...)
			fd.Chars = maps.Clone(valid.Chars)
			fd.ToUnicode = maps.Clone(valid.ToUnicode)
			fd.Planes = maps.Clone(valid.Planes)
			tt.edit(&fd)
			err := ValidateTTFLight(fd)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected semantic error containing %q, got %v", tt.want, err)
			}
			if !errors.Is(err, ErrInvalidFontData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
			}
		})
	}
}

func TestGobDecodersWrapSemanticAndEmbeddedFontFailures(t *testing.T) {
	dir := t.TempDir()
	metricFile := filepath.Join(dir, "Metric.gob")
	f, err := os.Create(metricFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := gob.NewEncoder(f).Encode(TTFLight{PostscriptName: "Metric"}); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	err = load(metricFile, &TTFLight{})
	if !errors.Is(err, ErrInvalidFontData) || !strings.Contains(err.Error(), "validate font metrics") {
		t.Fatalf("expected wrapped semantic metrics error, got %v", err)
	}

	installedFile := filepath.Join(dir, "Broken.gob")
	f, err = os.Create(installedFile)
	if err != nil {
		t.Fatal(err)
	}
	fd := validInstalledTTF("Broken", testTTFHeader(1))
	if err := gob.NewEncoder(f).Encode(fd); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	originalDir := UserFontDir
	UserFontDir = dir
	t.Cleanup(func() { UserFontDir = originalDir })

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Read", func() error {
			_, err := Read("Broken")
			return err
		}},
		{"readGob", func() error { return readGob(installedFile, &ttf{}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, ErrInvalidFontData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
			}
			if !strings.Contains(err.Error(), "font file table directory") {
				t.Fatalf("expected table-directory context, got %q", err)
			}
		})
	}
}

func TestUserFontMetricAccessorsReturnLoadErrors(t *testing.T) {
	originalDir := UserFontDir
	UserFontDir = filepath.Join(t.TempDir(), "missing")
	wantErr := os.ErrNotExist
	err := ReloadUserFonts()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	t.Cleanup(func() {
		UserFontDir = originalDir
		if err := ReloadUserFonts(); err != nil {
			t.Errorf("restore user fonts: %v", err)
		}
	})

	if _, err := UserFontNames(); !errors.Is(err, wantErr) {
		t.Fatalf("expected names error %v, got %v", wantErr, err)
	}
	if _, err := UserFontNamesVerbose(); !errors.Is(err, wantErr) {
		t.Fatalf("expected verbose names error %v, got %v", wantErr, err)
	}
	if _, _, err := UserFont("Demo"); !errors.Is(err, wantErr) {
		t.Fatalf("expected metric error %v, got %v", wantErr, err)
	}
	if _, err := IsUserFont("Demo"); !errors.Is(err, wantErr) {
		t.Fatalf("expected predicate error %v, got %v", wantErr, err)
	}
	if _, err := BoundingBox("Demo"); !errors.Is(err, wantErr) {
		t.Fatalf("expected bounding box error %v, got %v", wantErr, err)
	}
	if _, err := CharWidth("Demo", 'A'); !errors.Is(err, wantErr) {
		t.Fatalf("expected character width error %v, got %v", wantErr, err)
	}
	if _, err := TextWidth("Demo", "A", 12); !errors.Is(err, wantErr) {
		t.Fatalf("expected text width error %v, got %v", wantErr, err)
	}
	if _, err := SupportedFont("Demo"); !errors.Is(err, wantErr) {
		t.Fatalf("expected supported font error %v, got %v", wantErr, err)
	}
}

func TestCoreFontMetricAccessorsDoNotLoadUserFonts(t *testing.T) {
	originalDir := UserFontDir
	UserFontDir = filepath.Join(t.TempDir(), "missing")
	t.Cleanup(func() {
		UserFontDir = originalDir
		if err := ReloadUserFonts(); err != nil {
			t.Errorf("restore user fonts: %v", err)
		}
	})

	if _, err := BoundingBox("Helvetica"); err != nil {
		t.Fatal(err)
	}
	if _, err := CharWidth("Helvetica", 'A'); err != nil {
		t.Fatal(err)
	}
	if _, err := TextWidth("A", "Helvetica", 12); err != nil {
		t.Fatal(err)
	}
	ok, err := SupportedFont("Helvetica")
	if err != nil || !ok {
		t.Fatalf("expected supported core font, ok=%t err=%v", ok, err)
	}
	ok, err = IsUserFont("Helvetica")
	if err != nil || ok {
		t.Fatalf("expected core font not to be a user font, ok=%t err=%v", ok, err)
	}
}

func TestMetricAccessorsPreserveUnknownFontSentinel(t *testing.T) {
	originalDir := UserFontDir
	UserFontDir = t.TempDir()
	if err := ReloadUserFonts(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		UserFontDir = originalDir
		if err := ReloadUserFonts(); err != nil {
			t.Errorf("restore user fonts: %v", err)
		}
	})

	if _, err := BoundingBox("Missing"); !errors.Is(err, ErrUnknownFont) {
		t.Fatalf("expected %v, got %v", ErrUnknownFont, err)
	}
	if _, err := CharWidth("Missing", 'A'); !errors.Is(err, ErrUnknownFont) {
		t.Fatalf("expected %v, got %v", ErrUnknownFont, err)
	}
}
