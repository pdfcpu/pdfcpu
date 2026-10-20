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

package api

// GHSA-ppm7-rg95-2c2p: encryption must encrypt strings of compressed objects instead of copying them in plaintext.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-ppm7-rg95-2c2p

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const ghsaPpm7Marker = "ghsa-ppm7-plaintext-marker"

// ghsaPpm7PDF returns a PDF whose catalog /Legal entry references object 5, a legal attestation
// dictionary holding ghsaPpm7Marker that is stored in the uncompressed object stream 4.
// Validation does not decode object 5, so the writer must.
func ghsaPpm7PDF() []byte {
	obj5 := "<</Attestation(" + ghsaPpm7Marker + ")>>"
	objStm := "5 0 " + obj5

	var b bytes.Buffer
	b.WriteString("%PDF-1.5\n")
	offsets := make([]int, 7)
	for i, o := range []string{
		"<</Type/Catalog/Pages 2 0 R/Legal 5 0 R>>",
		"<</Type/Pages/Kids[3 0 R]/Count 1>>",
		"<</Type/Page/Parent 2 0 R/MediaBox[0 0 10 10]/Resources<<>>>>",
		fmt.Sprintf("<</Type/ObjStm/N 1/First 4/Length %d>>\nstream\n%s\nendstream", len(objStm), objStm),
	} {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}

	// Xref stream with /W [1 2 1]: object 5 is entry 0 of object stream 4.
	offsets[6] = b.Len()
	xref := []byte{0, 0, 0, 0xff}
	for _, off := range offsets[1:5] {
		xref = append(xref, 1, byte(off>>8), byte(off), 0)
	}
	xref = append(xref, 2, 0, 4, 0, 1, byte(offsets[6]>>8), byte(offsets[6]), 0)
	fmt.Fprintf(&b, "6 0 obj\n<</Type/XRef/Size 7/W[1 2 1]/Root 1 0 R/Length %d>>\nstream\n", len(xref))
	b.Write(xref)
	fmt.Fprintf(&b, "\nendstream\nendobj\nstartxref\n%d\n%%%%EOF\n", offsets[6])
	return b.Bytes()
}

// TestGHSA_ppm7_rg95_2c2p_EncryptFileEncryptsCompressedStrings follows the advisory path:
// EncryptFile with AES-128 and object-stream and xref-stream output disabled.
func TestGHSA_ppm7_rg95_2c2p_EncryptFileEncryptsCompressedStrings(t *testing.T) {
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		t.Run(fmt.Sprintf("mode=%d", mode), func(t *testing.T) {
			ghsaPpm7EncryptFile(t, mode)
		})
	}
}

func ghsaPpm7EncryptFile(t *testing.T, mode int) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	encFile := filepath.Join(dir, "encrypted.pdf")
	if err := os.WriteFile(inFile, ghsaPpm7PDF(), 0o600); err != nil {
		t.Fatal(err)
	}

	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = mode
	conf.UserPW, conf.OwnerPW = "user", "owner"
	conf.EncryptUsingAES, conf.EncryptKeyLength = true, 128
	conf.WriteObjectStream, conf.WriteXRefStream = false, false

	if err := EncryptFile(t.Context(), inFile, encFile, conf); err != nil {
		t.Fatal(err)
	}
	encrypted, err := os.ReadFile(encFile)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, []byte(ghsaPpm7Marker)) {
		t.Fatal("encrypted output contains the compressed object's string in plaintext")
	}

	// The string must survive encryption, not be dropped. Read without validation,
	// which rejects /Legal once it is decoded.
	ctx, err := ReadContext(t.Context(), bytes.NewReader(encrypted), conf)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := ctx.Catalog()
	if err != nil {
		t.Fatal(err)
	}
	legal, err := ctx.DereferenceDict(catalog["Legal"])
	if err != nil {
		t.Fatal(err)
	}
	if got := legal.StringEntry("Attestation"); got == nil || *got != ghsaPpm7Marker {
		t.Fatalf("decrypted /Legal /Attestation = %v, want %q", got, ghsaPpm7Marker)
	}
}
