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
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func cryptFilterStringObjects(t *testing.T, encrypted bool) []string {
	t.Helper()
	objects := []types.Object{
		types.Dict{
			"Literal": types.StringLiteral("plain string"), "Hex": types.HexLiteral("4142"),
			"LiteralRef": *types.NewIndirectRef(7, 0), "HexRef": *types.NewIndirectRef(8, 0),
			"ArrayRef": *types.NewIndirectRef(9, 0),
		},
		types.StringLiteral("indirect string"),
		types.HexLiteral("4344"),
		types.Array{types.StringLiteral("array string"), types.Dict{"Nested": types.StringLiteral("nested string")}},
	}
	var result []string
	key := encKey("", &model.Enc{O: bytes.Repeat([]byte{0x11}, 32), P: -4, ID: bytes.Repeat([]byte{0x22}, 16), R: 4, L: 128, Emd: true})
	for i, o := range objects {
		if encrypted {
			v, err := encryptDeepObject(t.Context(), o, i+6, 0, key, true, 4)
			if err != nil {
				t.Fatal(err)
			}
			if v != nil {
				o = v
			}
		}
		result = append(result, o.PDFString())
	}
	return result
}

func checkCryptFilterContent(t *testing.T, ctx *model.Context, content []byte) {
	t.Helper()
	sd, _, err := ctx.DereferenceStreamDict(*types.NewIndirectRef(4, 0))
	if err != nil || sd == nil {
		t.Fatalf("stream: %v", err)
	}
	if err := sd.Decode(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sd.Content, content) {
		t.Fatalf("stream content: got %q, want %q", sd.Content, content)
	}
	for i, want := range cryptFilterStringObjects(t, false) {
		o, err := ctx.Dereference(*types.NewIndirectRef(i+6, 0))
		if err != nil || o == nil {
			t.Fatalf("object %d: %v", i+6, err)
		}
		if got := o.PDFString(); got != want {
			t.Fatalf("object %d: got %s, want %s", i+6, got, want)
		}
	}
}

// TestIdentityCryptFilterDefaults verifies independent stream/string defaults on read and rewrite.
func TestIdentityCryptFilterDefaults(t *testing.T) {
	for _, tc := range []struct {
		name, defaults, filter, parms string
		streams, strings              bool
	}{
		{"Identity", "/StmF /Identity /StrF /Identity", "null", "null", false, false},
		{"omitted", "", "null", "null", false, false},
		{"streams only", "/StmF /StdCF /StrF /Identity", "null", "null", true, false},
		{"strings only", "/StmF /Identity /StrF /StdCF", "null", "null", false, true},
		{"omitted StmF", "/StrF /StdCF", "null", "null", false, true},
		{"omitted StrF", "/StmF /StdCF", "null", "null", true, false},
		{"both encrypted", "/StmF /StdCF /StrF /StdCF", "null", "null", true, true},
		{"explicit named override", "/StmF /Identity /StrF /Identity", "/Crypt", "<< /Name /StdCF >>", true, false},
		{"explicit array override", "/StmF /Identity /StrF /Identity", "[/Crypt]", "[<< /Name /StdCF >>]", true, false},
		{"explicit Identity override", "/StmF /StdCF /StrF /StdCF", "/Crypt", "<< /Name /Identity >>", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content := []byte("q Q")
			raw := content
			if tc.streams {
				key := encKey("", &model.Enc{O: bytes.Repeat([]byte{0x11}, 32), P: -4, ID: bytes.Repeat([]byte{0x22}, 16), R: 4, L: 128, Emd: true})
				var err error
				raw, err = encryptStream(raw, 4, 0, key, true, 4)
				if err != nil {
					t.Fatal(err)
				}
			}
			pdf := cryptFilterTestPDF(t, raw, tc.filter, tc.parms, tc.defaults, cryptFilterStringObjects(t, tc.strings))
			conf := model.NewDefaultConfiguration()
			conf.WriteObjectStream, conf.WriteXRefStream = false, false
			ctx, err := Read(t.Context(), bytes.NewReader(pdf), conf)
			if err != nil {
				t.Fatal(err)
			}
			checkCryptFilterContent(t, ctx, content)
			var out bytes.Buffer
			ctx.Write.Writer = bufio.NewWriter(&out)
			if err := WriteContext(t.Context(), ctx); err != nil {
				t.Fatal(err)
			}
			if err := ctx.Write.Flush(); err != nil {
				t.Fatal(err)
			}
			rewritten, err := Read(t.Context(), bytes.NewReader(out.Bytes()), conf)
			if err != nil {
				t.Fatal(err)
			}
			checkCryptFilterContent(t, rewritten, content)
		})
	}
}

// TestDefaultCryptFilterSelection verifies version gating and malformed selector errors.
func TestDefaultCryptFilterSelection(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	d := types.Dict{}
	ctx.Table[1] = model.NewXRefTableEntryGen0(d)
	ctx.Encrypt = types.NewIndirectRef(1, 0)
	for _, v := range []int{1, 2, 4, 5} {
		ctx.E = &model.Enc{V: v}
		for _, key := range []string{"StmF", "StrF"} {
			for _, selector := range []types.Object{nil, types.Name("Identity"), types.Name("StdCF")} {
				d[key] = selector
				got, err := defaultCryptFilterEncrypted(ctx, key)
				want := v < 4 || selector == types.Name("StdCF")
				if err != nil || got != want {
					t.Fatalf("V=%d %s=%v: encrypted=%t, want %t, error %v", v, key, selector, got, want, err)
				}
			}
		}
	}
	d["StrF"] = types.Integer(1)
	if _, err := stringsEncrypted(ctx); err != nil {
		t.Fatalf("no key should skip selection: %v", err)
	}
	ctx.EncKey = []byte("key")
	if _, err := stringsEncrypted(ctx); !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("invalid selector: %v", err)
	}
	delete(ctx.Table, 1)
	if _, err := stringsEncrypted(ctx); !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("missing encryption dictionary: %v", err)
	}
}

func cryptStreamTestPDF(t *testing.T, raw []byte, filterSpec, parmsSpec string) []byte {
	t.Helper()
	return cryptFilterTestPDF(t, raw, filterSpec, parmsSpec, "/StmF /StdCF /StrF /StdCF", nil)
}

func cryptFilterTestPDF(t *testing.T, raw []byte, filterSpec, parmsSpec, defaults string, extraObjects []string) []byte {
	t.Helper()
	o := bytes.Repeat([]byte{0x11}, 32)
	id := bytes.Repeat([]byte{0x22}, 16)
	enc := &model.Enc{O: o, P: -4, ID: id, R: 4, L: 128, Emd: true}
	ctx := &model.Context{Configuration: model.NewDefaultConfiguration(), XRefTable: &model.XRefTable{E: enc}}
	u, _, err := u(ctx)
	if err != nil {
		t.Fatal(err)
	}
	stream := fmt.Sprintf("<< /Length %d /Filter %s /DecodeParms %s >>\nstream\n%s\nendstream", len(raw), filterSpec, parmsSpec, raw)
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources <<>> /Contents 4 0 R >>",
		stream,
		fmt.Sprintf("<< /Filter /Standard /V 4 /R 4 /Length 128 /P -4 /CF << /StdCF << /CFM /AESV2 /AuthEvent /DocOpen /Length 16 >> >> %s /O <%s> /U <%s> >>", defaults, hex.EncodeToString(o), hex.EncodeToString(u)),
	}
	if len(extraObjects) > 0 {
		objects[2] = "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources << /Properties << /TestData 6 0 R >> >> /Contents 4 0 R >>"
		objects = append(objects, extraObjects...)
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objects))
	for i, object := range objects {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R /Encrypt 5 0 R /ID [<%s> <%s>] >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, hex.EncodeToString(id), hex.EncodeToString(id), xref)
	return b.Bytes()
}

// TestNamedCryptStreamFilters verifies explicit stream crypt filters override the document default.
func TestNamedCryptStreamFilters(t *testing.T) {
	content := []byte("q Q")
	tests := []struct {
		name       string
		filterSpec string
		parmsSpec  string
		identity   bool
	}{
		{"named", "/Crypt", "<< /Name /StdCF >>", false},
		{"array", "[/Crypt]", "[<< /Name /StdCF >>]", false},
		{"Identity", "/Crypt", "<< /Name /Identity >>", true},
		{"default Identity", "/Crypt", "<< >>", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := content
			if !tc.identity {
				var err error
				raw, err = encryptStream(raw, 4, 0, encKey("", &model.Enc{O: bytes.Repeat([]byte{0x11}, 32), P: -4, ID: bytes.Repeat([]byte{0x22}, 16), R: 4, L: 128, Emd: true}), true, 4)
				if err != nil {
					t.Fatal(err)
				}
			}
			conf := model.NewDefaultConfiguration()
			conf.ValidationMode = model.ValidationRelaxed
			ctx, err := Read(t.Context(), bytes.NewReader(cryptStreamTestPDF(t, raw, tc.filterSpec, tc.parmsSpec)), conf)
			if err != nil {
				t.Fatal(err)
			}
			sd, _, err := ctx.DereferenceStreamDict(*types.NewIndirectRef(4, 0))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(sd.Content, content) {
				t.Fatalf("stream content: got %q, want %q", sd.Content, content)
			}
			var output bytes.Buffer
			ctx.Write.Writer = bufio.NewWriter(&output)
			if err := WriteContext(t.Context(), ctx); err != nil {
				t.Fatal(err)
			}
			if err := ctx.Write.Flush(); err != nil {
				t.Fatal(err)
			}
			rewritten, err := Read(t.Context(), bytes.NewReader(output.Bytes()), conf)
			if err != nil {
				t.Fatal(err)
			}
			stream, _, err := rewritten.DereferenceStreamDict(*types.NewIndirectRef(4, 0))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(stream.Content, content) {
				t.Fatalf("rewritten stream content: got %q, want %q", stream.Content, content)
			}
		})
	}
}
