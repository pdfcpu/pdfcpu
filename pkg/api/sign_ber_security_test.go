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

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestValidateSignaturesRawRejectsNestedBERContents(t *testing.T) {
	path := filepath.Join("..", "testdata", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf")
	pdf, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	marker := []byte("/Contents <")
	start := bytes.Index(pdf, marker)
	if start < 0 {
		t.Fatal("signature Contents not found")
	}
	start += len(marker)
	end := bytes.IndexByte(pdf[start:], '>')
	if end < 0 {
		t.Fatal("signature Contents not terminated")
	}
	end += start

	const depth = 65
	ber := make([]byte, 0, depth*4+3)
	for range depth {
		ber = append(ber, 0x30, 0x80)
	}
	ber = append(ber, 0x02, 0x01, 0x00)
	for range depth {
		ber = append(ber, 0, 0)
	}
	contents := make([]byte, (end-start)/2)
	if len(ber) > len(contents) {
		t.Fatal("signature Contents too small for BER test input")
	}
	copy(contents, ber)
	hex.Encode(pdf[start:end], contents)

	conf := model.NewStatelessConfiguration()
	conf.Offline = true
	results, err := ValidateSignaturesRaw(t.Context(), bytes.NewReader(pdf), false, conf)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !strings.Contains(strings.Join(results[0].Problems, " "), "BER depth exceeds limit 64") {
		t.Fatalf("got %d results, want one malformed CMS result with BER depth limit", len(results))
	}
}
