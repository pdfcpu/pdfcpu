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
	return cryptFilterObjectsPDF(objects, id)
}

func cryptFilterObjectsPDF(objects []string, id []byte) []byte {
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

// TestExplicitCryptFilterMethodCompatibility checks explicit methods against the document's encryption version.
func TestExplicitCryptFilterMethodCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name    string
		v, r    int
		cfm     string
		aes     bool
		wantErr error
	}{
		{"RC4", 4, 4, "V2", false, nil},
		{"AES128", 4, 4, "AESV2", true, nil},
		{"AES256 revision 5", 5, 5, "AESV3", true, nil},
		{"AES256 revision 6", 5, 6, "AESV3", true, nil},
		{"AES256 with V4", 4, 4, "AESV3", false, ErrMalformedEncryption},
		{"AES128 with V5", 5, 6, "AESV2", false, ErrMalformedEncryption},
		{"RC4 with V5", 5, 6, "V2", false, ErrMalformedEncryption},
		{"security handler", 4, 4, "None", false, ErrUnsupportedEncryptionFeature},
		{"unknown method", 4, 4, "Unknown", false, ErrUnsupportedEncryptionFeature},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, err := model.NewContext(bytes.NewReader(nil), model.NewDefaultConfiguration())
			if err != nil {
				t.Fatal(err)
			}
			ctx.E = &model.Enc{V: tc.v, R: tc.r}
			ctx.Encrypt = types.NewIndirectRef(1, 0)
			ctx.Table[1] = model.NewXRefTableEntryGen0(types.Dict{
				"CF":   types.Dict{"StdCF": types.Dict{"CFM": types.Name(tc.cfm)}},
				"StmF": types.Name("Identity"), "StrF": types.Name("Identity"),
			})
			sd := &types.StreamDict{FilterPipeline: []types.PDFFilter{
				{Name: "Crypt", DecodeParms: types.Dict{"Name": types.Name("StdCF")}},
			}}
			encrypted, aes, err := streamCryptMethod(ctx, sd)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error: got %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && (!encrypted || aes != tc.aes) {
				t.Fatalf("encrypted=%t aes=%t, want encrypted=true aes=%t", encrypted, aes, tc.aes)
			}
		})
	}
}

// TestReadExplicitCryptFilterMethodCompatibility rejects AESV3 with V=4/R=4 before decrypting a complete PDF.
func TestReadExplicitCryptFilterMethodCompatibility(t *testing.T) {
	content := []byte("q Q")
	key := encKey("", &model.Enc{O: bytes.Repeat([]byte{0x11}, 32), P: -4, ID: bytes.Repeat([]byte{0x22}, 16), R: 4, L: 128, Emd: true})
	raw, err := encryptStream(content, 4, 0, key, true, 4)
	if err != nil {
		t.Fatal(err)
	}
	base := cryptFilterTestPDF(t, raw, "/Crypt", "<< /Name /StdCF >>", "/StmF /Identity /StrF /Identity", nil)
	for _, version := range []string{"1.7", "2.0"} {
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			for _, method := range []string{"AESV2", "AESV3"} {
				t.Run(fmt.Sprintf("PDF%s/mode%d/%s", version, mode, method), func(t *testing.T) {
					// Equal-length replacements preserve xref offsets. There is no catalog Version or Extensions entry.
					pdf := bytes.Replace(base, []byte("%PDF-1.7"), []byte("%PDF-"+version), 1)
					pdf = bytes.Replace(pdf, []byte("/CFM /AESV2"), []byte("/CFM /"+method), 1)
					conf := model.NewDefaultConfiguration()
					conf.ValidationMode = mode
					ctx, err := Read(t.Context(), bytes.NewReader(pdf), conf)
					if method == "AESV3" {
						if !errors.Is(err, ErrMalformedEncryption) {
							t.Fatalf("V=4/R=4 with AESV3: got %v, want malformed encryption", err)
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					sd, _, err := ctx.DereferenceStreamDict(*types.NewIndirectRef(4, 0))
					if err != nil || sd == nil {
						t.Fatalf("stream: %v", err)
					}
					if !bytes.Equal(sd.Content, content) {
						t.Fatalf("content: got %q, want %q", sd.Content, content)
					}
				})
			}
		}
	}
}

func adobeAES256CryptFilterPDF(t *testing.T, revision, extension int, content []byte) []byte {
	t.Helper()
	ctx, d := aes256PasswordContext(t, revision, "", "owner")
	if err := calcOAndU(ctx, d); err != nil {
		t.Fatal(err)
	}
	if err := writePermissions(ctx, d); err != nil {
		t.Fatal(err)
	}
	d["StmF"], d["StrF"] = types.Name("Identity"), types.Name("Identity")
	d.DictEntry("CF").DictEntry("StdCF")["Length"] = types.Integer(32)
	raw, err := encryptStream(content, 4, 0, ctx.EncKey, true, revision)
	if err != nil {
		t.Fatal(err)
	}
	objects := []string{
		fmt.Sprintf("<< /Type /Catalog /Pages 2 0 R /Extensions << /ADBE << /BaseVersion /1.7 /ExtensionLevel %d >> >> >>", extension),
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources <<>> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d /Filter /Crypt /DecodeParms << /Name /StdCF >> >>\nstream\n%s\nendstream", len(raw), raw),
		d.PDFString(),
	}
	return cryptFilterObjectsPDF(objects, ctx.E.ID)
}

// TestReadAdobeAES256CryptFilters preserves PDF 1.7 AESV3 input with Adobe extension declarations.
func TestReadAdobeAES256CryptFilters(t *testing.T) {
	for _, tc := range []struct{ revision, extension int }{
		{5, 3}, // Adobe Extension Level 3, section 3.5, introduces V=5/R=5/AESV3.
		{5, 5}, // Adobe extension levels are cumulative.
		{6, 8}, // Reader compatibility coverage; this does not validate the extension-level mapping.
	} {
		content := []byte("q Q")
		pdf := adobeAES256CryptFilterPDF(t, tc.revision, tc.extension, content)
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			t.Run(fmt.Sprintf("R%d/ADBE%d/mode%d", tc.revision, tc.extension, mode), func(t *testing.T) {
				conf := model.NewDefaultConfiguration()
				conf.ValidationMode = mode
				ctx, err := Read(t.Context(), bytes.NewReader(pdf), conf)
				if err != nil {
					t.Fatal(err)
				}
				if ctx.XRefTable.Version() != model.V17 || ctx.E.V != 5 || ctx.E.R != tc.revision {
					t.Fatalf("unexpected PDF/encryption version: %s, V=%d, R=%d", ctx.XRefTable.Version(), ctx.E.V, ctx.E.R)
				}
				sd, _, err := ctx.DereferenceStreamDict(*types.NewIndirectRef(4, 0))
				if err != nil || sd == nil {
					t.Fatalf("stream: %v", err)
				}
				if !bytes.Equal(sd.Content, content) {
					t.Fatalf("content: got %q, want %q", sd.Content, content)
				}
			})
		}
	}
}

func cryptFilterLengthPDF(t *testing.T, method string, revision int, catalogVersion string, explicit bool, length *int) []byte {
	t.Helper()
	ctx, err := model.NewContext(bytes.NewReader(nil), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	v := model.V17
	ctx.HeaderVersion = &v
	keyLength := 128
	if revision >= 5 {
		keyLength = 256
	}
	d := newEncryptDict(false, method != "V2", keyLength, -4)
	d["R"] = types.Integer(revision)
	ctx.E, err = supportedEncryption(ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	ctx.E.ID = bytes.Repeat([]byte{1}, 16)
	if err := calcOAndU(ctx, d); err != nil {
		t.Fatal(err)
	}
	if err := writePermissions(ctx, d); err != nil {
		t.Fatal(err)
	}
	d["StrF"] = types.Name("Identity")
	cf := d.DictEntry("CF").DictEntry("StdCF")
	cf.Delete("Length")
	if length != nil {
		cf["Length"] = types.Integer(*length)
	}
	filter := ""
	if explicit {
		d["StmF"] = types.Name("Identity")
		filter = "/Filter /Crypt /DecodeParms << /Name /StdCF >>"
	}
	raw, err := encryptStream([]byte("q Q"), 4, 0, ctx.EncKey, method != "V2", revision)
	if err != nil {
		t.Fatal(err)
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R " + catalogVersion + " >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources <<>> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d %s >>\nstream\n%s\nendstream", len(raw), filter, raw),
		d.PDFString(),
	}
	return cryptFilterObjectsPDF(objects, ctx.E.ID)
}

func checkReadCryptFilterWithoutLength(t *testing.T, pdf []byte, mode int, pdf20 bool) {
	t.Helper()
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = mode
	ctx, err := Read(t.Context(), bytes.NewReader(pdf), conf)
	if pdf20 {
		if !errors.Is(err, ErrMalformedEncryption) {
			t.Fatalf("got %v, want missing crypt-filter Length error", err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	sd, _, err := ctx.DereferenceStreamDict(*types.NewIndirectRef(4, 0))
	if err != nil || sd == nil {
		t.Fatalf("stream: %v", err)
	}
	if err := sd.Decode(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sd.Content, []byte("q Q")) {
		t.Fatalf("content: got %q, want q Q", sd.Content)
	}
}

// TestReadCryptFilterWithoutLength checks complete inputs, including a catalog upgrade to PDF 2.0.
func TestReadCryptFilterWithoutLength(t *testing.T) {
	for _, tc := range []struct {
		name, method, catalog string
		revision              int
		pdf20                 bool
	}{
		{"PDF 1.7 RC4", "V2", "", 4, false},
		{"PDF 1.7 AES128", "AESV2", "", 4, false},
		{"PDF 1.7 Adobe AES256", "AESV3", "/Extensions << /ADBE << /BaseVersion /1.7 /ExtensionLevel 3 >> >>", 5, false},
		{"PDF 2.0 AES128", "AESV2", "", 4, true},
		{"PDF 2.0 AES256", "AESV3", "", 6, true},
		{"catalog PDF 2.0 AES128", "AESV2", "/Version /2.0", 4, true},
		{"PDF 2.0 RC4", "V2", "", 4, true},
	} {
		for _, explicit := range []bool{false, true} {
			pdf := cryptFilterLengthPDF(t, tc.method, tc.revision, tc.catalog, explicit, nil)
			if tc.pdf20 && tc.catalog == "" {
				pdf = bytes.Replace(pdf, []byte("%PDF-1.7"), []byte("%PDF-2.0"), 1)
			}
			for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
				t.Run(fmt.Sprintf("%s/explicit%t/mode%d", tc.name, explicit, mode), func(t *testing.T) {
					checkReadCryptFilterWithoutLength(t, pdf, mode, tc.pdf20)
				})
			}
		}
	}
}

// TestReadCryptFilterEffectiveVersion checks length values only after the effective PDF version is known.
func TestReadCryptFilterEffectiveVersion(t *testing.T) {
	for _, tc := range []struct {
		header, catalog string
		length          int
		wantErr         bool
	}{
		{"2.0", "", 16, false},
		{"1.7", "/Version /2.0", 16, false},
		{"2.0", "", 128, true},
		{"1.7", "/Version /2.0", 128, true},
	} {
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			t.Run(fmt.Sprintf("header%s/catalog%s/length%d/mode%d", tc.header, tc.catalog, tc.length, mode), func(t *testing.T) {
				pdf := cryptFilterLengthPDF(t, "AESV2", 4, tc.catalog, false, &tc.length)
				pdf = bytes.Replace(pdf, []byte("%PDF-1.7"), []byte("%PDF-"+tc.header), 1)
				conf := model.NewDefaultConfiguration()
				conf.ValidationMode = mode
				ctx, err := Read(t.Context(), bytes.NewReader(pdf), conf)
				if tc.wantErr {
					if !errors.Is(err, ErrMalformedEncryption) {
						t.Fatalf("got %v, want malformed encryption", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if ctx.XRefTable.Version() != model.V20 {
					t.Fatalf("got effective version %s, want 2.0", ctx.XRefTable.Version())
				}
				sd, _, err := ctx.DereferenceStreamDict(*types.NewIndirectRef(4, 0))
				if err != nil || sd == nil {
					t.Fatalf("stream: %v", err)
				}
				if err := sd.Decode(); err != nil || string(sd.Content) != "q Q" {
					t.Fatalf("content=%q error=%v", sd.Content, err)
				}
			})
		}
	}
}

// TestReadEncryptedObjectStreamCatalog resolves the version only after decrypting the catalog's object stream.
func TestReadEncryptedObjectStreamCatalog(t *testing.T) {
	length := 16
	pdf := cryptFilterLengthPDF(t, "AESV2", 4, "", false, &length)
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationStrict
	conf.WriteObjectStream, conf.WriteXRefStream = true, true
	ctx, err := Read(t.Context(), bytes.NewReader(pdf), conf)
	if err != nil {
		t.Fatal(err)
	}
	ctx.RootDict["Version"] = types.Name("2.0")
	d, err := ctx.EncryptDict()
	if err != nil {
		t.Fatal(err)
	}
	d.DictEntry("CF").DictEntry("StdCF")["Length"] = types.Integer(16)
	// Enable the encrypted xref-stream writer when constructing this object-stream fixture.
	ctx.Read.UsingXRefStreams = true
	var out bytes.Buffer
	ctx.Write.Writer = bufio.NewWriter(&out)
	ctx.Write.WriteToObjectStream = true
	if err := WriteContext(t.Context(), ctx); err != nil {
		t.Fatal(err)
	}
	if err := ctx.Write.Flush(); err != nil {
		t.Fatal(err)
	}
	got, err := Read(t.Context(), bytes.NewReader(out.Bytes()), conf)
	if err != nil {
		t.Fatal(err)
	}
	if *got.HeaderVersion != model.V17 || got.XRefTable.Version() != model.V20 {
		t.Fatalf("header=%s effective=%s, want 1.7 and 2.0", got.HeaderVersion, got.XRefTable.Version())
	}
	root := got.Table[got.Root.ObjectNumber.Value()]
	if root.ObjectStream == nil || got.E == nil || got.E.V != 4 {
		t.Fatal("fixture must have an encrypted object stream containing the catalog")
	}
}
