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
	"os"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func infoPreservationSourcePDF(indirect bool) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n")
	infoDict := "<</Producer(source)/CreationDate(D:20240102030405+00'00')/ModDate(D:20240102030405+00'00')>>"
	if indirect {
		infoDict = "<</Producer 4 0 R/CreationDate 5 0 R/ModDate 6 0 R>>"
	}
	objects := []string{
		"<</Type/Catalog/Pages 2 0 R>>",
		"<</Type/Pages/Count 0/Kids[]>>",
		infoDict,
	}
	if indirect {
		objects = append(objects,
			"(source)",
			"(D:20240102030405+00'00')",
			"(D:20240102030405+00'00')",
		)
	}
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	buf.WriteString("trailer\n")
	fmt.Fprintf(&buf, "<</Size %d/Root 1 0 R/Info 3 0 R", len(offsets))
	buf.WriteString("/ID[<00112233445566778899AABBCCDDEEFF><00112233445566778899AABBCCDDEEFF>]>>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOffset)
	return buf.Bytes()
}

func reporterInfoObjects() (types.HexLiteral, types.StringLiteral, types.StringLiteral) {
	producer := types.NewHexLiteral([]byte(types.EncodeUTF16String("製品名")))
	creationDate := types.StringLiteral("D:20240102030405+00'00'")
	modDate := types.StringLiteral("D:20240809010203+09'00'")
	return producer, creationDate, modDate
}

func reporterInfoObjectMap() map[string]types.Object {
	producer, creationDate, modDate := reporterInfoObjects()
	return map[string]types.Object{
		"Producer":     producer,
		"CreationDate": creationDate,
		"ModDate":      modDate,
	}
}

func setReporterInfoObjects(t *testing.T, ctx *model.Context, indirect bool) []int {
	t.Helper()
	d, err := ctx.DereferenceDict(*ctx.Info)
	if err != nil {
		t.Fatal(err)
	}
	var objNrs []int
	for _, key := range []string{"Producer", "CreationDate", "ModDate"} {
		want := reporterInfoObjectMap()[key]
		if !indirect {
			d[key] = want
			continue
		}
		ref, ok := d[key].(types.IndirectRef)
		if !ok {
			t.Fatalf("Info %s has type %T, want IndirectRef", key, d[key])
		}
		entry, found := ctx.FindTableEntryForIndRef(&ref)
		if !found {
			t.Fatalf("Info %s object %s is missing", key, ref.PDFString())
		}
		entry.Object = want
		objNrs = append(objNrs, ref.ObjectNumber.Value())
	}
	return objNrs
}

func infoEntryObject(t *testing.T, ctx *model.Context, d types.Dict, key string, indirect bool) types.Object {
	t.Helper()
	o := d[key]
	if !indirect {
		return o
	}
	ref, ok := o.(types.IndirectRef)
	if !ok {
		t.Fatalf("Info %s has type %T, want IndirectRef", key, o)
	}
	o, err := ctx.Dereference(ref)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func assertReporterInfoObjects(t *testing.T, ctx *model.Context, indirect bool) {
	t.Helper()
	d, err := ctx.DereferenceDict(*ctx.Info)
	if err != nil {
		t.Fatal(err)
	}
	producer, creationDate, modDate := reporterInfoObjects()
	producerObject := infoEntryObject(t, ctx, d, "Producer", indirect)
	gotProducer, ok := producerObject.(types.HexLiteral)
	if !ok {
		t.Fatalf("Info Producer has type %T, want HexLiteral", producerObject)
	}
	gotBytes, err := gotProducer.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, err := producer.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotBytes, wantBytes) {
		t.Fatalf("Info Producer bytes = %X, want %X", gotBytes, wantBytes)
	}
	for key, want := range map[string]types.Object{
		"CreationDate": creationDate,
		"ModDate":      modDate,
	} {
		if got := infoEntryObject(t, ctx, d, key, indirect); got != want {
			t.Fatalf("Info %s = %#v (%T), want %#v (%T)", key, got, got, want, want)
		}
	}
}

type reporterInfoRewrite struct {
	mode        int
	incremental bool
	indirect    bool
	preserve    bool
}

func rewriteReporterInfo(t *testing.T, opts reporterInfoRewrite) []byte {
	t.Helper()
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = opts.mode
	conf.PreserveInfoDict = opts.preserve
	conf.PostProcessValidate = true
	conf.WriteObjectStream = false
	conf.WriteXRefStream = false

	source := infoPreservationSourcePDF(opts.indirect)
	if !opts.incremental {
		ctx, err := ReadAndValidate(bytes.NewReader(source), conf)
		if err != nil {
			t.Fatal(err)
		}
		setReporterInfoObjects(t, ctx, opts.indirect)
		if err := ValidateContext(ctx); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := Write(ctx, &buf, conf); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}

	f, err := os.CreateTemp(t.TempDir(), "pdfcpu-info-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write(source); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	ctx, err := ReadAndValidate(f, conf)
	if err != nil {
		t.Fatal(err)
	}
	objNrs := setReporterInfoObjects(t, ctx, opts.indirect)
	if err := ValidateContext(ctx); err != nil {
		t.Fatal(err)
	}
	ctx.Write.Increment = true
	ctx.Write.Offset = ctx.Read.FileSize
	ctx.Write.ObjNrs = append(ctx.Write.ObjNrs, objNrs...)
	ctx.Write.ObjNrs = append(ctx.Write.ObjNrs, ctx.Info.ObjectNumber.Value())
	if err := WriteIncr(ctx, f, conf); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

// TestPreserveInfoDictReporterWorkflow verifies entry values and object types across validation and both write modes.
func TestPreserveInfoDictReporterWorkflow(t *testing.T) {
	for _, tt := range []struct {
		name        string
		mode        int
		incremental bool
		indirect    bool
	}{
		{name: "strict full", mode: model.ValidationStrict},
		{name: "relaxed full", mode: model.ValidationRelaxed},
		{name: "strict incremental", mode: model.ValidationStrict, incremental: true},
		{name: "relaxed incremental", mode: model.ValidationRelaxed, incremental: true},
		{name: "strict full indirect", mode: model.ValidationStrict, indirect: true},
		{name: "relaxed incremental indirect", mode: model.ValidationRelaxed, incremental: true, indirect: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output := rewriteReporterInfo(t, reporterInfoRewrite{
				mode:        tt.mode,
				incremental: tt.incremental,
				indirect:    tt.indirect,
				preserve:    true,
			})
			conf := model.NewDefaultConfiguration()
			conf.ValidationMode = tt.mode
			ctx, err := ReadAndValidate(bytes.NewReader(output), conf)
			if err != nil {
				t.Fatal(err)
			}
			assertReporterInfoObjects(t, ctx, tt.indirect)
		})
	}
}

// TestPreserveInfoDictDisabledRetainsStamping verifies backward-compatible default writer behavior.
func TestPreserveInfoDictDisabledRetainsStamping(t *testing.T) {
	output := rewriteReporterInfo(t, reporterInfoRewrite{mode: model.ValidationStrict})
	ctx, err := ReadAndValidate(bytes.NewReader(output), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	d, err := ctx.DereferenceDict(*ctx.Info)
	if err != nil {
		t.Fatal(err)
	}
	want := types.StringLiteral("pdfcpu " + model.VersionStr)
	if got := d["Producer"]; got != want {
		t.Fatalf("Info Producer = %#v (%T), want %#v (%T)", got, got, want, want)
	}
}
