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
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestReadFileContext verifies context-aware file reading.
func TestReadFileContext(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "test.pdf")

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	if doc, err := ReadFileWithContext(ctx, inFile, nil); err == nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	if doc, err := ReadWithContext(ctx, fp, nil); err == nil {
		t.Errorf("reading should have failed, got %v", doc)
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("should have failed with timeout, got %s", err)
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
	if err == nil {
		t.Fatal("expected an invalid W array error")
	}
	if !strings.Contains(err.Error(), "invalid W array") {
		t.Fatalf("got %v, want invalid W array error", err)
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Dummy pdfcpu context to be used for parsing a single object.
	c := &model.Context{
		Read: &model.ReadContext{
			RS: bytes.NewReader(fp.Bytes()),
		},
		XRefTable: &model.XRefTable{},
	}
	o, err := ParseObjectWithContext(ctx, c, 0, 123, 0)
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

// TestEnsureStreamLengthRejectsNilDict verifies malformed stream dicts do not panic.
func TestEnsureStreamLengthRejectsNilDict(t *testing.T) {
	sd := &types.StreamDict{
		Raw: []byte("abc"),
	}

	err := ensureStreamLength(sd, true)
	if err == nil || !strings.Contains(err.Error(), "corrupt stream dict") {
		t.Fatalf("got %v, want corrupt stream dict error", err)
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Dummy pdfcpu context to be used for parsing a single object.
	c := &model.Context{
		Read: &model.ReadContext{
			RS: bytes.NewReader(fp.Bytes()),
		},
		XRefTable: &model.XRefTable{},
	}
	o, err := ParseObjectWithContext(ctx, c, 0, 123, 0)
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
