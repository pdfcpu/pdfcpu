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

package api

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// RC4 V=1 R=2 empty-user-password PDF (issue 1467).
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

func decodeAlignedRC4PDF(t *testing.T) []byte {
	t.Helper()
	pdf, err := base64.StdEncoding.DecodeString(strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, rc4EmptyUserPWAlignedStartXRef))
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	return pdf
}

func shiftStartXRef(t *testing.T, pdf []byte, delta int64) []byte {
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

func writeTempPDF(t *testing.T, pdf []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "in.pdf")
	if err := os.WriteFile(path, pdf, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestOptimizeAndDecryptRepairedEmptyUserPWRC4 covers issue 1467: xref
// reconstruction used to leave RC4 streams encrypted, so OptimizeFile
// failed with zlib: invalid header and DecryptFile reported success
// while writing a still-encrypted (corrupt) file.
func TestOptimizeAndDecryptRepairedEmptyUserPWRC4(t *testing.T) {
	aligned := writeTempPDF(t, decodeAlignedRC4PDF(t))
	repaired := writeTempPDF(t, shiftStartXRef(t, decodeAlignedRC4PDF(t), -15))
	conf := model.NewDefaultConfiguration()

	if err := OptimizeFile(aligned, filepath.Join(t.TempDir(), "opt-a.pdf"), nil); err != nil {
		t.Fatalf("OptimizeFile aligned: %v", err)
	}
	if err := OptimizeFile(repaired, filepath.Join(t.TempDir(), "opt-b.pdf"), nil); err != nil {
		t.Fatalf("OptimizeFile repaired: %v", err)
	}

	outA := filepath.Join(t.TempDir(), "dec-a.pdf")
	outB := filepath.Join(t.TempDir(), "dec-b.pdf")
	if err := DecryptFile(aligned, outA, conf); err != nil {
		t.Fatalf("DecryptFile aligned: %v", err)
	}
	if err := DecryptFile(repaired, outB, conf); err != nil {
		t.Fatalf("DecryptFile repaired: %v", err)
	}

	ctxA, err := ReadContextFile(outA)
	if err != nil {
		t.Fatalf("ReadContextFile decrypted aligned: %v", err)
	}
	ctxB, err := ReadContextFile(outB)
	if err != nil {
		t.Fatalf("ReadContextFile decrypted repaired: %v", err)
	}
	if ctxA.Encrypt != nil || ctxB.Encrypt != nil {
		t.Fatalf("decrypted output still encrypted: aligned=%v repaired=%v", ctxA.Encrypt, ctxB.Encrypt)
	}
	if err := ValidateFile(outB, nil); err != nil {
		t.Fatalf("ValidateFile decrypted repaired: %v", err)
	}
}
