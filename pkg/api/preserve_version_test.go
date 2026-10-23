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
	"fmt"
	"io"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func appendPDFObject(buf *bytes.Buffer, objNr int, body string) int {
	offset := buf.Len()
	fmt.Fprintf(buf, "%d 0 obj\n%s\nendobj\n", objNr, body)
	return offset
}

func minimalVersionPDF(version, catalogExtra string) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%%PDF-%s\n%%\xE2\xE3\xCF\xD3\n", version)

	catalog := "<< /Type /Catalog /Pages 2 0 R"
	if catalogExtra != "" {
		catalog += " " + catalogExtra
	}
	catalog += " >>"

	offsets := []int{0}
	offsets = append(offsets, appendPDFObject(&buf, 1, catalog))
	offsets = append(offsets, appendPDFObject(&buf, 2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>"))
	offsets = append(offsets, appendPDFObject(&buf, 3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Resources << >> >>"))

	xrefOffset := buf.Len()
	buf.WriteString("xref\n0 4\n")
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size 4 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset)
	return buf.Bytes()
}

func preservedConfiguration(preserve bool) *model.Configuration {
	conf := model.NewStatelessConfiguration()
	conf.PreserveVersion = preserve
	return conf
}

func assertPDFHeader(t *testing.T, pdf []byte, version string) {
	t.Helper()
	want := []byte("%PDF-" + version + "\n")
	if !bytes.HasPrefix(pdf, want) {
		end := min(24, len(pdf))
		t.Fatalf("header: got %q, want %q", pdf[:end], want)
	}
}

func assertClassicXRef(t *testing.T, pdf []byte) {
	t.Helper()
	if !bytes.Contains(pdf, []byte("\nxref\n")) {
		t.Fatal("missing classic xref section")
	}
	if bytes.Contains(pdf, []byte("/Type /XRef")) || bytes.Contains(pdf, []byte("/Type/XRef")) || bytes.Contains(pdf, []byte("/ObjStm")) {
		t.Fatal("wrote xref stream or object stream")
	}
}

func optimizePDF(t *testing.T, in []byte, conf *model.Configuration) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := Optimize(t.Context(), bytes.NewReader(in), &out, conf, nil); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestOptimizePreservesInputVersion(t *testing.T) {
	for _, version := range []string{"1.5", "1.6"} {
		t.Run(version, func(t *testing.T) {
			got := optimizePDF(t, minimalVersionPDF(version, ""), preservedConfiguration(true))
			assertPDFHeader(t, got, version)
		})
	}
}

func TestOptimizeWithoutPreserveVersionWrites17(t *testing.T) {
	got := optimizePDF(t, minimalVersionPDF("1.5", ""), preservedConfiguration(false))
	assertPDFHeader(t, got, "1.7")
}

func TestOptimizePreserves14WithClassicXRef(t *testing.T) {
	conf := preservedConfiguration(true)
	if !conf.WriteXRefStream || !conf.WriteObjectStream {
		t.Fatal("default configuration should write xref streams")
	}
	got := optimizePDF(t, minimalVersionPDF("1.4", ""), conf)
	assertPDFHeader(t, got, "1.4")
	assertClassicXRef(t, got)
	if !conf.WriteXRefStream || !conf.WriteObjectStream {
		t.Fatal("preserve-version changed the caller's xref stream configuration")
	}
}

func TestOptimizePreservesEffectiveCatalogVersion(t *testing.T) {
	got := optimizePDF(t, minimalVersionPDF("1.4", "/Version /1.6"), preservedConfiguration(true))
	assertPDFHeader(t, got, "1.6")
}

func TestOptimizePreservesPDF20(t *testing.T) {
	in := minimalVersionPDF("2.0", "")
	for _, preserve := range []bool{false, true} {
		t.Run(fmt.Sprintf("preserve=%t", preserve), func(t *testing.T) {
			got := optimizePDF(t, in, preservedConfiguration(preserve))
			assertPDFHeader(t, got, "2.0")
		})
	}
}

func TestWatermarkRaisesPreservedVersionTo15(t *testing.T) {
	conf := preservedConfiguration(true)
	wm, err := TextWatermark(t.Context(), "Draft", "", false, false, types.POINTS, conf)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := AddWatermarks(t.Context(), bytes.NewReader(minimalVersionPDF("1.4", "")), &out, nil, wm, conf); err != nil {
		t.Fatal(err)
	}
	assertPDFHeader(t, out.Bytes(), "1.5")
}

func TestCollectionRaisesPreservedVersionTo17(t *testing.T) {
	conf := preservedConfiguration(true)
	ctx, err := ReadAndValidate(t.Context(), bytes.NewReader(minimalVersionPDF("1.5", "")), conf)
	if err != nil {
		t.Fatal(err)
	}
	if err := ctx.EnsureCollection(); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Write(t.Context(), ctx, &out, conf); err != nil {
		t.Fatal(err)
	}
	assertPDFHeader(t, out.Bytes(), "1.7")
}

func TestAES256RaisesPreservedVersionTo17(t *testing.T) {
	conf := preservedConfiguration(true)
	conf.UserPW = "user"
	conf.OwnerPW = "owner"
	var out bytes.Buffer
	if err := Encrypt(t.Context(), bytes.NewReader(minimalVersionPDF("1.4", "")), &out, conf); err != nil {
		t.Fatal(err)
	}
	assertPDFHeader(t, out.Bytes(), "1.7")
}

func TestAES128RaisesPreservedVersionTo16(t *testing.T) {
	conf := preservedConfiguration(true)
	conf.UserPW = "user"
	conf.OwnerPW = "owner"
	conf.EncryptUsingAES = true
	conf.EncryptKeyLength = 128
	var out bytes.Buffer
	if err := Encrypt(t.Context(), bytes.NewReader(minimalVersionPDF("1.4", "")), &out, conf); err != nil {
		t.Fatal(err)
	}
	assertPDFHeader(t, out.Bytes(), "1.6")
}

func TestMergePreservesMaximumInputVersion(t *testing.T) {
	conf := preservedConfiguration(true)
	var out bytes.Buffer
	err := MergeRaw(t.Context(), []io.ReadSeeker{
		bytes.NewReader(minimalVersionPDF("1.4", "")),
		bytes.NewReader(minimalVersionPDF("1.6", "")),
	}, &out, false, conf)
	if err != nil {
		t.Fatal(err)
	}
	assertPDFHeader(t, out.Bytes(), "1.6")
}

func TestMergeWithoutPreserveVersionWrites17(t *testing.T) {
	conf := preservedConfiguration(false)
	var out bytes.Buffer
	err := MergeRaw(t.Context(), []io.ReadSeeker{
		bytes.NewReader(minimalVersionPDF("1.4", "")),
		bytes.NewReader(minimalVersionPDF("1.6", "")),
	}, &out, false, conf)
	if err != nil {
		t.Fatal(err)
	}
	assertPDFHeader(t, out.Bytes(), "1.7")
}
