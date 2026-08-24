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
	"encoding/base64"
	"strconv"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// RC4 V=1 R=2 empty-user-password PDF (issue 1467). startxref correctly
// points at the xref keyword.
const rc4EmptyUserPWAlignedStartXRef = `
JVBERi0xLjQKJeLjz9MKMSAwIG9iago8PCAvVHlwZSAvQ2F0YWxvZyAvUGFnZXMgMiAwIFIgPj4K
ZW5kb2JqCjIgMCBvYmoKPDwgL1R5cGUgL1BhZ2VzIC9LaWRzIFszIDAgUl0gL0NvdW50IDEgPj4K
ZW5kb2JqCjMgMCBvYmoKPDwgL1R5cGUgL1BhZ2UgL1BhcmVudCAyIDAgUiAvTWVkaWFCb3ggWzAg
MCA1OTUgODQyXSAvQ29udGVudHMgNCAwIFIgL1Jlc291cmNlcyA8PCAvRm9udCA8PCAvRjEgNSAw
IFIgPj4gPj4gPj4KZW5kb2JqCjQgMCBvYmoKPDwgL0xlbmd0aCA2MyAvRmlsdGVyIC9GbGF0ZURl
Y29kZSA+PgpzdHJlYW0Km18SQc4WXrbl+bn78RV101hgBduotGf1z2O/QSf8hOKPLSWdaGK0mq0d
tBSHfAJ4MIngCBjXso93t0lAME6QCmVuZHN0cmVhbQplbmRvYmoKNSAwIG9iago8PCAvVHlwZSAv
Rm9udCAvU3VidHlwZSAvVHlwZTEgL0Jhc2VGb250IC9IZWx2ZXRpY2EgPj4KZW5kb2JqCjYgMCBv
YmoKPDwgL1Byb2R1Y2VyIDxhZWZhNTE5MjFiZmQ2ZjQzOGE2M2IxMjM0MDFmMjI0MWE3MDY5Mzhm
YmEyYjM4MTAwYz4gL0NyZWF0aW9uRGF0ZSA8OTliOTBkZDY0MWFlMmIxMmRiNzNmMjc0MDA1ZDdk
NTFlYjUzYzVjZGY4N2E2Yj4gPj4KZW5kb2JqCjcgMCBvYmoKPDwgL0xlbmd0aCAzOCAvRmlsdGVy
IC9GbGF0ZURlY29kZSAvTGVuZ3RoMSAyMDAwID4+CnN0cmVhbQpxUbKOSihSvS3yU8N2QaxfpfHW
dwLEOoUvvSoXvOVfPDELBywPRgplbmRzdHJlYW0KZW5kb2JqCjggMCBvYmoKPDwgL0ZpbHRlciAv
U3RhbmRhcmQgL1YgMSAvUiAyIC9PIDxhMGIxMzBlZDY5ZmU1NTkwMmRjMzNkOGM2MDIzNzljODU2
ZmU3ZWQxNWU5ZDhlMGJhY2Q3MzY3YTVmNjdiNmRjPiAvVSA8ODkyOTgwN2NlOWIzYjNiMmY3MzZh
NDNlMTFmNTdhOTIwYzgzNDRhZTAxN2E0Nzc2ZDI1NTQzYmZkN2RhYTFlMT4gL1AgLTYwID4+CmVu
ZG9iagp4cmVmCjAgOQowMDAwMDAwMDAwIDY1NTM1IGYgCjAwMDAwMDAwMTUgMDAwMDAgbiAKMDAw
MDAwMDA2NCAwMDAwMCBuIAowMDAwMDAwMTIxIDAwMDAwIG4gCjAwMDAwMDAyNDcgMDAwMDAgbiAK
MDAwMDAwMDM4MSAwMDAwMCBuIAowMDAwMDAwNDUxIDAwMDAwIG4gCjAwMDAwMDA1OTggMDAwMDAg
biAKMDAwMDAwMDcyMSAwMDAwMCBuIAp0cmFpbGVyCjw8IC9TaXplIDkgL1Jvb3QgMSAwIFIgL0lu
Zm8gNiAwIFIgL0VuY3J5cHQgOCAwIFIgL0lEIFs8MzAzMTMyMzMzNDM1MzYzNzM4Mzk2MTYyNjM2
NDY1NjY+IDwzMDMxMzIzMzM0MzUzNjM3MzgzOTYxNjI2MzY0NjU2Nj5dID4+CnN0YXJ0eHJlZgo5
MTcKJSVFT0YK
`

func decodePDFFixture(t *testing.T, b64 string) []byte {
	t.Helper()
	pdf, err := base64.StdEncoding.DecodeString(strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, b64))
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	return pdf
}

func pdfWithStartXRefShifted(t *testing.T, pdf []byte, delta int64) []byte {
	t.Helper()
	key := []byte("startxref")
	i := bytes.LastIndex(pdf, key)
	if i < 0 {
		t.Fatal("missing startxref")
	}
	j := i + len(key)
	for j < len(pdf) && (pdf[j] == '\n' || pdf[j] == '\r' || pdf[j] == ' ') {
		j++
	}
	k := j
	for k < len(pdf) && pdf[k] >= '0' && pdf[k] <= '9' {
		k++
	}
	n, err := strconv.ParseInt(string(pdf[j:k]), 10, 64)
	if err != nil {
		t.Fatalf("startxref: %v", err)
	}
	repl := strconv.FormatInt(n+delta, 10)
	if len(repl) != k-j {
		t.Fatalf("startxref replacement %q changes field width of %q", repl, pdf[j:k])
	}
	out := bytes.Clone(pdf)
	copy(out[j:k], repl)
	return out
}

func decodedPageContent(t *testing.T, pdf []byte) []byte {
	t.Helper()
	ctx, err := Read(bytes.NewReader(pdf), nil)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if ctx.EncKey == nil {
		t.Fatal("expected encryption key")
	}
	entry := ctx.Table[4]
	if entry == nil || entry.Object == nil {
		t.Fatal("missing content stream object 4")
	}
	sd, ok := entry.Object.(types.StreamDict)
	if !ok {
		t.Fatalf("object 4: %T", entry.Object)
	}
	if err := sd.Decode(); err != nil {
		t.Fatalf("content stream decode: %v", err)
	}
	if len(sd.Content) == 0 {
		t.Fatal("empty decoded content")
	}
	return sd.Content
}

// TestReadDecryptsRC4StreamsAfterXRefRepair covers xref reconstruction of an
// RC4-encrypted PDF (empty user password). Repair loads objects before the
// file key exists; those cached streams must still be decrypted so later
// optimize/decode sees plaintext Flate streams.
func TestReadDecryptsRC4StreamsAfterXRefRepair(t *testing.T) {
	aligned := decodePDFFixture(t, rc4EmptyUserPWAlignedStartXRef)
	repaired := pdfWithStartXRefShifted(t, aligned, -15)

	want := decodedPageContent(t, aligned)
	got := decodedPageContent(t, repaired)
	if !bytes.Equal(want, got) {
		t.Fatalf("repaired content stream %d bytes, aligned %d bytes", len(got), len(want))
	}

	ctx, err := Read(bytes.NewReader(repaired), nil)
	if err != nil {
		t.Fatalf("Read repaired: %v", err)
	}
	if err := OptimizeXRefTable(ctx); err != nil {
		t.Fatalf("OptimizeXRefTable: %v", err)
	}
}
