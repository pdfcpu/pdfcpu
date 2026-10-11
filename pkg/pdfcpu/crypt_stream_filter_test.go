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
	"fmt"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func cryptStreamTestPDF(t *testing.T, raw []byte, filterSpec, parmsSpec string) []byte {
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
		fmt.Sprintf("<< /Filter /Standard /V 4 /R 4 /Length 128 /P -4 /CF << /StdCF << /CFM /AESV2 /AuthEvent /DocOpen /Length 16 >> >> /StmF /StdCF /StrF /StdCF /O <%s> /U <%s> >>", hex.EncodeToString(o), hex.EncodeToString(u)),
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
