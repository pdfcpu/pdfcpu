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

// GHSA-prvw-v9vv-6966: image extraction must return an error for a missing BitsPerComponent instead of panicking.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-prvw-v9vv-6966

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func ghsaPrvwImageWithoutBitsPerComponentPDF() []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	offsets := make([]int, 6)
	objects := [][]byte{
		nil,
		[]byte("<</Type/Catalog/Pages 2 0 R>>"),
		[]byte("<</Type/Pages/Kids[3 0 R]/Count 1>>"),
		[]byte("<</Type/Page/Parent 2 0 R/MediaBox[0 0 1 1]/Resources<</XObject<</Im0 5 0 R>>>>/Contents 4 0 R>>"),
		[]byte("<</Length 26>>\nstream\nq 1 0 0 1 0 0 cm /Im0 Do Q\nendstream"),
		append([]byte("<</Type/XObject/Subtype/Image/Width 1/Height 1/ColorSpace/DeviceRGB/Length 3>>\nstream\n"),
			[]byte{0xff, 0, 0, '\n'}...),
	}
	objects[5] = append(objects[5], []byte("endstream")...)

	for i := 1; i < len(objects); i++ {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n", i)
		b.Write(objects[i])
		b.WriteString("\nendobj\n")
	}

	xrefOffset := b.Len()
	b.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<</Size 6/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", xrefOffset)
	return b.Bytes()
}

// TestGHSA_prvw_v9vv_6966_ExtractImagesRawRejectsMissingBitsPerComponent verifies malformed images return an error
// instead of panicking.
func TestGHSA_prvw_v9vv_6966_ExtractImagesRawRejectsMissingBitsPerComponent(t *testing.T) {
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	_, err := ExtractImagesRaw(t.Context(), bytes.NewReader(ghsaPrvwImageWithoutBitsPerComponentPDF()), nil, conf)
	if err == nil {
		t.Fatal("expected missing BitsPerComponent error")
	}
	if !strings.Contains(err.Error(), "missing entry=BitsPerComponent") {
		t.Fatalf("unexpected error: %v", err)
	}
}
