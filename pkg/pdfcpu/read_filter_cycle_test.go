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
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func selfReferencingStreamPDF(entry string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	var offsets []int
	for _, body := range []string{
		"<</Type/Catalog/Pages 2 0 R>>",
		"<</Type/Pages/Kids[]/Count 0>>",
		fmt.Sprintf("<</Length 1%s>>\nstream\nx\nendstream", entry),
	} {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", len(offsets), body)
	}
	xref := b.Len()
	b.WriteString("xref\n0 4\n0000000000 65535 f \n")
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<</Size 4/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", xref)
	return b.Bytes()
}

// TestReadSelfReferencingStreamEntry isolates malformed input so a pre-fix stack overflow cannot crash the test suite.
func TestReadSelfReferencingStreamEntry(t *testing.T) {
	const childEnv = "PDFCPU_SELF_REF_STREAM_ENTRY"
	if entry := os.Getenv(childEnv); entry != "" {
		debug.SetMaxStack(1 << 20)
		var dictEntry string
		switch entry {
		case "Filter":
			dictEntry = "/Filter 3 0 R"
		case "DecodeParms":
			dictEntry = "/Filter /FlateDecode/DecodeParms 3 0 R"
		default:
			t.Fatalf("unknown stream entry %q", entry)
		}
		pdf := selfReferencingStreamPDF(dictEntry)
		offset := int64(bytes.Index(pdf, []byte("3 0 obj")))
		if offset < 0 {
			t.Fatal("stream object missing from test input")
		}
		ctx, err := model.NewContext(bytes.NewReader(pdf), nil)
		if err != nil {
			t.Fatal(err)
		}
		generation := 0
		ctx.Table[3] = &model.XRefTableEntry{Offset: &offset, Generation: &generation}
		_, err = ParseObject(t.Context(), ctx, offset, 3, 0)
		if !errors.Is(err, errCircularObjectReference) {
			t.Fatalf("got %v, want circular object reference", err)
		}
		if _, err := Read(t.Context(), bytes.NewReader(pdf), model.NewDefaultConfiguration()); err == nil {
			t.Fatal("full read accepted self-referencing stream entry")
		}
		return
	}

	for _, entry := range []string{"Filter", "DecodeParms"} {
		t.Run(entry, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestReadSelfReferencingStreamEntry$")
			cmd.Env = append(os.Environ(), childEnv+"="+entry)
			out, err := cmd.CombinedOutput()
			if err != nil {
				if len(out) > 1000 {
					out = out[:1000]
				}
				t.Fatalf("read %s cycle in subprocess: %v\n%s", entry, err, out)
			}
		})
	}
}

// TestParseObjectAllowsIndirectFilter verifies that a valid indirection remains usable.
func TestParseObjectAllowsIndirectFilter(t *testing.T) {
	pdf := []byte("3 0 obj\n<</Length 3/Filter 4 0 R>>\nstream\n78>\nendstream\nendobj\n" +
		"4 0 obj\n/ASCIIHexDecode\nendobj\n")
	ctx, err := model.NewContext(bytes.NewReader(pdf), nil)
	if err != nil {
		t.Fatal(err)
	}
	offset := int64(bytes.Index(pdf, []byte("4 0 obj")))
	generation := 0
	ctx.Table[4] = &model.XRefTableEntry{Offset: &offset, Generation: &generation}

	obj, err := ParseObject(t.Context(), ctx, 0, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	sd, ok := obj.(types.StreamDict)
	if !ok || len(sd.FilterPipeline) != 1 || sd.FilterPipeline[0].Name != "ASCIIHexDecode" {
		t.Fatalf("got %T %v, want stream with indirect ASCIIHexDecode filter", obj, obj)
	}
}

// TestParseObjectLimitsReferenceChain checks the configured bound on distinct nested object parses.
func TestParseObjectLimitsReferenceChain(t *testing.T) {
	pdf := []byte("3 0 obj\n<</Length 1/Filter 4 0 R>>\nstream\nx\nendstream\nendobj\n" +
		"4 0 obj\n<</Length 1/Filter 5 0 R>>\nstream\nx\nendstream\nendobj\n" +
		"5 0 obj\n/ASCIIHexDecode\nendobj\n")
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxRecursionDepth = 2
	ctx, err := model.NewContext(bytes.NewReader(pdf), conf)
	if err != nil {
		t.Fatal(err)
	}
	for _, objNr := range []int{4, 5} {
		offset := int64(bytes.Index(pdf, []byte(fmt.Sprintf("%d 0 obj", objNr))))
		generation := 0
		ctx.Table[objNr] = &model.XRefTableEntry{Offset: &offset, Generation: &generation}
	}
	_, err = ParseObject(t.Context(), ctx, 0, 3, 0)
	if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want max recursion depth exceeded", err)
	}
}
