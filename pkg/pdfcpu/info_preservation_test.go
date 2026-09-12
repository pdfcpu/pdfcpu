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
	"reflect"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func newInfoWriteContext(t *testing.T, conf *model.Configuration, infoDict types.Dict) *model.Context {
	t.Helper()
	ctx, err := CreateContextWithXRefTable(conf, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	runtimeCtx, err := model.NewContext(strings.NewReader(""), conf)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Read = runtimeCtx.Read
	ctx.Optimize = runtimeCtx.Optimize

	if infoDict != nil {
		ctx.Info, err = ctx.IndRefForNewObject(infoDict)
		if err != nil {
			t.Fatal(err)
		}
	}
	return ctx
}

func infoDictForContext(t *testing.T, ctx *model.Context) types.Dict {
	t.Helper()
	if ctx.Info == nil {
		t.Fatal("missing Info reference")
	}
	d, err := ctx.DereferenceDict(*ctx.Info)
	if err != nil {
		t.Fatal(err)
	}
	if d == nil {
		t.Fatal("missing Info dictionary")
	}
	return d
}

func writeInfoContext(t *testing.T, ctx *model.Context) []byte {
	t.Helper()
	var buf bytes.Buffer
	ctx.Write.Writer = bufio.NewWriter(&buf)
	if err := WriteContext(t.Context(), ctx); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func utf16StringLiteral(t *testing.T, s string) types.StringLiteral {
	t.Helper()
	escaped, err := types.EscapedUTF16String(s)
	if err != nil {
		t.Fatal(err)
	}
	return types.StringLiteral(*escaped)
}

// TestPreserveInfoDictDefaultsToDisabled protects the existing write behavior.
func TestPreserveInfoDictDefaultsToDisabled(t *testing.T) {
	if model.NewDefaultConfiguration().PreserveInfoDict {
		t.Fatal("PreserveInfoDict defaults to true")
	}
}

// TestEnsureInfoDictPreservesExistingEntryObjects verifies values and their representation remain intact.
func TestEnsureInfoDictPreservesExistingEntryObjects(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.PreserveInfoDict = true
	infoDict := types.Dict{
		"Producer":     utf16StringLiteral(t, "製品名"),
		"CreationDate": types.HexLiteral("443A32303234303130323033303430355A"),
		"ModDate":      types.StringLiteral("pipeline-controlled-date"),
	}
	before := infoDict.Clone()
	ctx := newInfoWriteContext(t, conf, infoDict)

	if err := ensureInfoDict(ctx); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(infoDictForContext(t, ctx), before) {
		t.Fatalf("Info dictionary changed:\n got: %#v\nwant: %#v", infoDict, before)
	}
}

// TestEnsureInfoDictPreserveModeDoesNotSynthesizeMissingEntries verifies an existing dictionary remains sparse.
func TestEnsureInfoDictPreserveModeDoesNotSynthesizeMissingEntries(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.PreserveInfoDict = true
	infoDict := types.Dict{"Author": types.StringLiteral("author")}
	before := infoDict.Clone()
	ctx := newInfoWriteContext(t, conf, infoDict)

	if err := ensureInfoDict(ctx); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(infoDictForContext(t, ctx), before) {
		t.Fatalf("Info dictionary changed:\n got: %#v\nwant: %#v", infoDict, before)
	}
}

// TestEnsureInfoDictPreserveModeStillCreatesMissingDictionary retains pdfcpu's behavior for documents without Info.
func TestEnsureInfoDictPreserveModeStillCreatesMissingDictionary(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.PreserveInfoDict = true
	ctx := newInfoWriteContext(t, conf, nil)

	if err := ensureInfoDict(ctx); err != nil {
		t.Fatal(err)
	}
	d := infoDictForContext(t, ctx)
	for _, key := range []string{"Producer", "CreationDate", "ModDate"} {
		if _, found := d[key]; !found {
			t.Fatalf("missing generated Info entry %q", key)
		}
	}
}

// TestEnsureInfoDictPreservesIndirectEntries verifies preserved objects remain reachable and are not marked redundant.
func TestEnsureInfoDictPreservesIndirectEntries(t *testing.T) {
	for _, tt := range []struct {
		name     string
		preserve bool
	}{
		{name: "default stamping"},
		{name: "preserve", preserve: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			conf := model.NewDefaultConfiguration()
			conf.PreserveInfoDict = tt.preserve
			ctx := newInfoWriteContext(t, conf, types.Dict{})
			producer := types.NewHexLiteral([]byte(types.EncodeUTF16String("製品名")))
			producerRef, err := ctx.IndRefForNewObject(producer)
			if err != nil {
				t.Fatal(err)
			}
			creationDateRef, err := ctx.IndRefForNewObject(types.StringLiteral("creation-date"))
			if err != nil {
				t.Fatal(err)
			}
			modDateRef, err := ctx.IndRefForNewObject(types.StringLiteral("mod-date"))
			if err != nil {
				t.Fatal(err)
			}
			d := infoDictForContext(t, ctx)
			d["Producer"] = *producerRef
			d["CreationDate"] = *creationDateRef
			d["ModDate"] = *modDateRef
			before := d.Clone()
			ctx.Write.Increment = true

			if err := ensureInfoDict(ctx); err != nil {
				t.Fatal(err)
			}
			preserved := reflect.DeepEqual(d, before)
			if preserved != tt.preserve {
				t.Fatalf("dictionary preservation = %t, want %t", preserved, tt.preserve)
			}
			if !tt.preserve {
				for _, key := range []string{"Producer", "CreationDate", "ModDate"} {
					if _, ok := d[key].(types.StringLiteral); !ok {
						t.Fatalf("entry %q has type %T, want StringLiteral", key, d[key])
					}
				}
				if got := d["Producer"]; got != types.StringLiteral("pdfcpu "+model.VersionStr) {
					t.Fatalf("Producer = %v, want pdfcpu version", got)
				}
			}
			for _, ref := range []*types.IndirectRef{producerRef, creationDateRef, modDateRef} {
				objNr := ref.ObjectNumber.Value()
				wantRedundant := !tt.preserve
				if got := ctx.Optimize.DuplicateInfoObjects[objNr]; got != wantRedundant {
					t.Fatalf("object %d marked redundant: %t, want %t", objNr, got, wantRedundant)
				}
			}
			if len(ctx.Write.ObjNrs) != 1 || ctx.Write.ObjNrs[0] != ctx.Info.ObjectNumber.Value() {
				t.Fatalf("increment objects = %v, want Info object %d", ctx.Write.ObjNrs, ctx.Info.ObjectNumber.Value())
			}
		})
	}
}

// TestWriteContextPreservesInfoEntryEncoding verifies the preserved object representation reaches serialized output.
func TestWriteContextPreservesInfoEntryEncoding(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.PreserveInfoDict = true
	conf.WriteObjectStream = false
	conf.WriteXRefStream = false
	producer := utf16StringLiteral(t, "製品名")
	creationDate := types.HexLiteral("443A32303234303130323033303430355A")
	modDate := types.StringLiteral("pipeline-controlled-date")
	ctx := newInfoWriteContext(t, conf, types.Dict{
		"Producer":     producer,
		"CreationDate": creationDate,
		"ModDate":      modDate,
	})
	output := writeInfoContext(t, ctx)
	for _, want := range []string{
		"/Producer" + producer.PDFString(),
		"/CreationDate" + creationDate.PDFString(),
		"/ModDate" + modDate.PDFString(),
	} {
		if !bytes.Contains(output, []byte(want)) {
			t.Fatalf("serialized output does not contain %q", want)
		}
	}
}

// TestWriteContextPreservesIndirectInfoEntries verifies indirect entry objects remain referenced and serialized.
func TestWriteContextPreservesIndirectInfoEntries(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.PreserveInfoDict = true
	conf.WriteObjectStream = false
	conf.WriteXRefStream = false
	ctx := newInfoWriteContext(t, conf, types.Dict{})
	producer := types.NewHexLiteral([]byte(types.EncodeUTF16String("製品名")))
	producerRef, err := ctx.IndRefForNewObject(producer)
	if err != nil {
		t.Fatal(err)
	}
	d := infoDictForContext(t, ctx)
	d["Producer"] = *producerRef
	output := writeInfoContext(t, ctx)

	for _, want := range []string{
		"/Producer " + producerRef.PDFString(),
		strings.TrimSuffix(producerRef.PDFString(), " R") + " obj",
		producer.PDFString(),
	} {
		if !bytes.Contains(output, []byte(want)) {
			t.Fatalf("serialized output does not contain %q", want)
		}
	}
}
