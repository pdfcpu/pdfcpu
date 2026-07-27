/*
Copyright 2025 The pdfcpu Authors.

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
	"encoding/gob"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func installTextRenderingMetrics(t *testing.T) {
	t.Helper()
	originalDir := font.UserFontDir
	font.UserFontDir = t.TempDir()
	f, err := os.Create(filepath.Join(font.UserFontDir, "RenderTest.gob"))
	if err != nil {
		t.Fatal(err)
	}
	ttf := font.TTFLight{
		PostscriptName:  "RenderTest",
		UnitsPerEm:      1000,
		FirstChar:       'A',
		LastChar:        'B',
		LLx:             -100,
		LLy:             -200,
		URx:             1000,
		URy:             800,
		HorMetricsCount: 2,
		GlyphCount:      2,
		GlyphWidths:     []int{500, 600},
		Chars:           map[uint32]uint16{'A': 1, 'B': 1},
		ToUnicode:       map[uint16]uint32{1: 'A'},
		Planes:          map[int]bool{0: true},
	}
	if err := gob.NewEncoder(f).Encode(ttf); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := font.ReloadUserFonts(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		font.UserFontDir = originalDir
		if err := font.ReloadUserFonts(); err != nil {
			t.Errorf("restore user fonts: %v", err)
		}
	})
}

func TestPrepBytesPropagatesFontErrors(t *testing.T) {
	if _, err := PrepBytes(nil, "text", "", false, false, false); !errors.Is(err, font.ErrMissingFontName) {
		t.Fatalf("expected %v, got %v", font.ErrMissingFontName, err)
	}
	if _, err := PrepBytes(nil, "text", "Missing", false, false, false); !errors.Is(err, font.ErrUnknownFont) {
		t.Fatalf("expected %v, got %v", font.ErrUnknownFont, err)
	}

	installTextRenderingMetrics(t)
	if _, err := PrepBytes(nil, "A", "RenderTest", true, false, false); !errors.Is(err, ErrMissingXRefTable) {
		t.Fatalf("expected %v, got %v", ErrMissingXRefTable, err)
	}
	xRefTable := &XRefTable{}
	if _, err := PrepBytes(xRefTable, "A", "RenderTest", true, false, false); err != nil {
		t.Fatalf("prepare embedded text: %v", err)
	}
	if !xRefTable.UsedGIDs["RenderTest"][1] {
		t.Fatal("expected initialized used-glyph map")
	}
}

func TestWriteColumnPropagatesBoundingBoxErrors(t *testing.T) {
	td := TextDescriptor{
		Text:     "text",
		FontName: "Missing",
		FontKey:  "F1",
		FontSize: 12,
		Scale:    1,
		ScaleAbs: true,
	}
	_, err := WriteColumn(&XRefTable{}, io.Discard, types.RectForFormat("A4"), nil, td, 100)
	if !errors.Is(err, font.ErrUnknownFont) {
		t.Fatalf("expected %v, got %v", font.ErrUnknownFont, err)
	}
	for _, context := range []string{"create column bounding box", "measure column lines"} {
		if !strings.Contains(err.Error(), context) {
			t.Fatalf("expected context %q, got %q", context, err)
		}
	}

	td.Text, td.FontName = "", "Helvetica"
	_, err = WriteColumn(&XRefTable{}, io.Discard, types.RectForFormat("A4"), nil, td, 100)
	if err == nil || !strings.Contains(err.Error(), "no text lines") {
		t.Fatalf("expected empty-line error, got %v", err)
	}
}

func TestWriteColumnPreservesFractionalFontSize(t *testing.T) {
	td := TextDescriptor{
		Text:     "fractional",
		FontName: "Helvetica",
		FontKey:  "F1",
		FontSize: 10.125,
		Scale:    1,
		ScaleAbs: true,
	}
	var buf bytes.Buffer
	if _, err := WriteColumn(&XRefTable{}, &buf, types.RectForFormat("A4"), nil, td, 100); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "/F1 10.125 Tf") {
		t.Fatalf("fractional font size missing from content stream: %s", buf.String())
	}
}

func TestWordWrapPropagatesCandidateMeasurementError(t *testing.T) {
	installTextRenderingMetrics(t)
	_, err := WordWrap("word", "Missing", 12, 10)
	if !errors.Is(err, font.ErrUnknownFont) {
		t.Fatalf("expected %v, got %v", font.ErrUnknownFont, err)
	}
	if !strings.Contains(err.Error(), "wrap candidate") {
		t.Fatalf("expected candidate context, got %q", err)
	}
}

func TestJustifiedTextWrappingDoesNotInsertLinefeeds(t *testing.T) {
	fontSize := 12.
	preparer, err := newJustifiedTextPreparer(&XRefTable{}, "Helvetica", fontSize)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	linefeeds, err := preparer.prepare(&lines, "one two", 25, "Helvetica", &fontSize, false, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if linefeeds != 0 {
		t.Fatalf("linefeeds = %d, want 0", linefeeds)
	}
	if len(lines) != 1 {
		t.Fatalf("wrapped lines = %d, want 1", len(lines))
	}
}

type failingTextWriter struct {
	err error
}

func (w failingTextWriter) Write([]byte) (int, error) {
	return 0, w.err
}

type shortTextWriter struct{}

func (shortTextWriter) Write(p []byte) (int, error) {
	return len(p) - 1, nil
}

func TestWriteColumnPropagatesWriterError(t *testing.T) {
	wantErr := errors.New("write failed")
	td := TextDescriptor{
		Text:     "text",
		FontName: "Helvetica",
		FontKey:  "F1",
		FontSize: 12,
		Scale:    1,
		ScaleAbs: true,
	}
	_, err := WriteColumn(&XRefTable{}, failingTextWriter{err: wantErr}, types.RectForFormat("A4"), nil, td, 100)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "render column text") {
		t.Fatalf("expected rendering context, got %q", err)
	}

	_, err = WriteColumn(&XRefTable{}, shortTextWriter{}, types.RectForFormat("A4"), nil, td, 100)
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("expected %v, got %v", io.ErrShortWrite, err)
	}

	td.HAlign = types.AlignJustify
	_, err = WriteColumn(&XRefTable{}, failingTextWriter{err: wantErr}, types.RectForFormat("A4"), nil, td, 100)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected justified rendering error %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "justified text") {
		t.Fatalf("expected justified rendering context, got %q", err)
	}
}

func TestWriteColumnRejectsMissingOutputBoundaries(t *testing.T) {
	td := TextDescriptor{FontName: "Helvetica"}
	buf := new(bytes.Buffer)
	_, err := WriteColumn(nil, buf, types.RectForFormat("A4"), nil, td, 100)
	if !errors.Is(err, ErrMissingXRefTable) {
		t.Fatalf("expected %v, got %v", ErrMissingXRefTable, err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no rendering, got %q", buf.String())
	}
	xRefTable := &XRefTable{}
	_, err = WriteColumn(xRefTable, nil, types.RectForFormat("A4"), nil, td, 100)
	if err == nil || !strings.Contains(err.Error(), "missing writer") {
		t.Fatalf("expected missing writer, got %v", err)
	}
	_, err = WriteColumn(xRefTable, io.Discard, nil, nil, td, 100)
	if err == nil || !strings.Contains(err.Error(), "missing media box") {
		t.Fatalf("expected missing media box, got %v", err)
	}
}

// TestWordWrap verifies word wrap.
func TestWordWrap(t *testing.T) {
	testcases := []struct {
		FontName       string
		FontSize       float64
		MaxWidthPoints float64
		Text           string
		Want           []string
	}{
		{"Helvetica", 12, 10,
			"",
			[]string{""},
		},

		{"Helvetica", 12, 10,
			"   ",
			[]string{""},
		},

		{"Helvetica", 12, 10,
			"           ",
			[]string{""},
		},

		{"Helvetica", 12, 10,
			"      Indent line",
			[]string{"      ", "Indent", "line"},
		},

		{"Helvetica", 12, 60,
			"\tTab Indent line",
			[]string{"\tTab", "Indent line"},
		},

		{"Helvetica", 12, 200,
			"\tLong tab Indent line",
			[]string{"\tLong tab Indent line"},
		},

		{"Helvetica", 12, 20,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem", "ipsum", "dolor", "sit", "amet,", "consectetur", "adipiscing", "elit."},
		},

		{"Helvetica", 12, 50,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem", "ipsum", "dolor sit", "amet,", "consectetur", "adipiscing", "elit."},
		},

		{"Helvetica", 12, 70,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem ipsum", "dolor sit", "amet,", "consectetur", "adipiscing", "elit."},
		},

		{"Courier", 24, 70,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem", "ipsum", "dolor", "sit", "amet,", "consectetur", "adipiscing", "elit."},
		},

		{"Helvetica", 12, 100,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem ipsum dolor", "sit amet,", "consectetur", "adipiscing elit."},
		},

		{"Helvetica", 12, 200,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem ipsum dolor sit amet,", "consectetur adipiscing elit."},
		},

		{"Helvetica", 12, 500,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem ipsum dolor sit amet, consectetur adipiscing elit."},
		},

		{"Helvetica", 12, 100,
			"Lorem ipsum\ndolor sit amet,\n consectetur adipiscing\nelit.",
			[]string{"Lorem ipsum", "dolor sit amet,", " consectetur", "adipiscing", "elit."},
		},

		{"Helvetica", 12, 100,
			"Lorem ipsum dolor sit amet,\t\t\t consectetur\nadipiscing\telit.",
			[]string{"Lorem ipsum dolor", "sit amet,", "consectetur", "adipiscing\telit."},
		},

		{"Helvetica", 12, 100,
			"Lorem ipsum dolor sit amet,\n\tconsectetur adipiscing elit.",
			[]string{"Lorem ipsum dolor", "sit amet,", "\tconsectetur", "adipiscing elit."},
		},

		{"Helvetica", 12, 100,
			"Lorem ipsum dolor sit amet, consectetur\nadipiscing elit.",
			[]string{"Lorem ipsum dolor", "sit amet,", "consectetur", "adipiscing elit."},
		},
	}

	for _, tc := range testcases {
		gotLines, err := WordWrapFloat(tc.Text, tc.FontName, tc.FontSize, tc.MaxWidthPoints)
		if err != nil {
			t.Fatal(err)
		}
		if len(gotLines) != len(tc.Want) {
			t.Errorf("expected %d lines when wrapping %s, got %d", len(tc.Want), tc.Text, len(gotLines))
			continue
		}
		for i, s := range gotLines {
			if s != tc.Want[i] {
				t.Errorf("expected %s when wrapping %s, got %s", tc.Want[i], tc.Text, s)
			}
		}
	}
}
