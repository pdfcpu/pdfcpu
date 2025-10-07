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
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestReadFileContext(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "test.pdf")

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	if doc, err := ReadFileWithContext(ctx, inFile, nil); err == nil {
		t.Errorf("reading should have failed, got %+v", doc)
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("should have failed with timeout, got %s", err)
	}
}

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
		t.Errorf("reading should have failed, got %+v", doc)
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("should have failed with timeout, got %s", err)
	}
}

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
