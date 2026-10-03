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

package validate

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestURLSNameTreeExternalReferenceInventory(t *testing.T) {
	ctx := externalReferenceContext(t)
	treeKey := "https://example.invalid/captured/page.html"
	sourceURL := "https://origin.example.invalid/page.html"
	aliasURL := "https://alias.example.invalid/page.html"
	commandURL := "https://capture.example.invalid/page.html"
	ctx.XRefTable.Table[11] = model.NewXRefTableEntryGen0(types.Dict{"URL": types.StringLiteral(commandURL)})
	contentSet := types.Dict{
		"Type": types.Name("SpiderContentSet"),
		"S":    types.Name("SPS"),
		"ID":   types.StringLiteral("content-id"),
		"O":    types.Array{*types.NewIndirectRef(10, 0)},
		"SI": types.Array{
			types.Dict{"AU": types.StringLiteral(sourceURL)},
			types.Dict{
				"AU": types.Dict{"U": types.StringLiteral(aliasURL)},
				"C":  *types.NewIndirectRef(11, 0),
			},
		},
	}
	tree := types.Dict{"Names": types.Array{types.StringLiteral(treeKey), contentSet}}
	first, last, _, err := validateNameTree(t.Context(), ctx.XRefTable, "URLS", tree, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if first != treeKey || last != treeKey {
		t.Fatalf("URLS bounds: got %q..%q, want %q", first, last, treeKey)
	}
	want := map[string]linkTarget{
		treeKey:    {kind: linkTargetURI, source: linkSourceWebCaptureURL},
		sourceURL:  {kind: linkTargetURI, source: linkSourceWebCaptureSource},
		aliasURL:   {kind: linkTargetURI, source: linkSourceWebCaptureSource},
		commandURL: {kind: linkTargetURI, source: linkSourceWebCaptureCommand},
	}
	for target, wantTarget := range want {
		got, found := inventoryTargetAtPage(ctx, 0, target)
		if !found || got != wantTarget {
			t.Fatalf("URLS target %q: got %+v, found=%t, want %+v", target, got, found, wantTarget)
		}
	}
	tr := &successfulLinkTransport{}
	failed, err := checkLinks(t.Context(), ctx.XRefTable, http.Client{Transport: tr}, []int{0})
	if err != nil {
		t.Fatal(err)
	}
	if failed || tr.calls != len(want) {
		t.Fatalf("URLS checks: failed=%t, transport calls=%d, want %d", failed, tr.calls, len(want))
	}
	for target := range want {
		if status := ctx.URIs[0][target]; status != "" {
			t.Fatalf("URLS target %q: got status %q, want success", target, status)
		}
	}
}

func TestURLSNameTreeArrayValueInventory(t *testing.T) {
	ctx := externalReferenceContext(t)
	contentSet := func(id, sourceURL string, subtype types.Name) types.Dict {
		d := types.Dict{
			"S":  subtype,
			"ID": types.StringLiteral(id),
			"O":  types.Array{*types.NewIndirectRef(10, 0)},
			"SI": types.Dict{"AU": types.StringLiteral(sourceURL)},
		}
		if subtype == "SIS" {
			d["R"] = types.Integer(1)
		}
		return d
	}
	sources := []string{"https://example.invalid/one", "https://example.invalid/two"}
	value := types.Array{
		contentSet("one", sources[0], types.Name("SPS")),
		contentSet("two", sources[1], types.Name("SIS")),
	}
	if err := validateURLSNameTreeValue(t.Context(), ctx.XRefTable, value, model.V13); err != nil {
		t.Fatal(err)
	}
	want := linkTarget{kind: linkTargetURI, source: linkSourceWebCaptureSource}
	for _, source := range sources {
		if got, found := inventoryTargetAtPage(ctx, 0, source); !found || got != want {
			t.Fatalf("URLS array source %q: got %+v, found=%t, want %+v", source, got, found, want)
		}
	}
	if err := validateURLSNameTreeValue(t.Context(), ctx.XRefTable, types.Array{}, model.V13); err != nil {
		t.Fatalf("empty URLS array rejected: %v", err)
	}
	if err := validateURLSNameTreeValue(t.Context(), ctx.XRefTable, types.Array{types.Integer(1)}, model.V13); err == nil {
		t.Fatal("invalid URLS array member accepted")
	}
	c, cancel := context.WithCancel(t.Context())
	cancel()
	if err := validateURLSNameTreeValue(c, ctx.XRefTable, value, model.V13); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context cancellation", err)
	}
}

func TestIDSNameTreeArrayValueInventory(t *testing.T) {
	ctx := externalReferenceContext(t)
	contentSet := func(id, sourceURL string, subtype types.Name) types.Dict {
		d := types.Dict{
			"S":  subtype,
			"ID": types.StringLiteral(id),
			"O":  types.Array{*types.NewIndirectRef(10, 0)},
			"SI": types.Dict{"AU": types.StringLiteral(sourceURL)},
		}
		if subtype == "SIS" {
			d["R"] = types.Integer(1)
		}
		return d
	}
	sources := []string{"https://example.invalid/one", "https://example.invalid/two"}
	value := types.Array{
		contentSet("one", sources[0], types.Name("SPS")),
		contentSet("two", sources[1], types.Name("SIS")),
	}
	if err := validateIDSNameTreeValue(t.Context(), ctx.XRefTable, value, model.V13); err != nil {
		t.Fatal(err)
	}
	want := linkTarget{kind: linkTargetURI, source: linkSourceWebCaptureSource}
	for _, source := range sources {
		if got, found := inventoryTargetAtPage(ctx, 0, source); !found || got != want {
			t.Fatalf("IDS array source %q: got %+v, found=%t, want %+v", source, got, found, want)
		}
	}
	if err := validateIDSNameTreeValue(t.Context(), ctx.XRefTable, types.Array{}, model.V13); err != nil {
		t.Fatalf("empty IDS array rejected: %v", err)
	}
	err := validateIDSNameTreeValue(t.Context(), ctx.XRefTable, types.Array{types.Integer(1)}, model.V13)
	if err == nil || !strings.Contains(err.Error(), "IDS name tree value[0]") {
		t.Fatalf("invalid IDS array member: got %v", err)
	}
	c, cancel := context.WithCancel(t.Context())
	cancel()
	if err := validateIDSNameTreeValue(c, ctx.XRefTable, value, model.V13); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context cancellation", err)
	}
}

func TestRichMediaAssetInventory(t *testing.T) {
	ctx := externalReferenceContext(t)
	assetName := "media/player.swf"
	asset := types.Dict{
		"Type": types.Name("Filespec"),
		"F":    types.StringLiteral(assetName),
		"UF":   types.StringLiteral(assetName),
		"EF":   types.Dict{"F": *types.NewIndirectRef(20, 0)},
	}
	embeddedFile := types.StreamDict{Dict: types.Dict{"Type": types.Name("EmbeddedFile")}}
	ctx.XRefTable.Table[20] = model.NewXRefTableEntryGen0(embeddedFile)
	ctx.XRefTable.Table[21] = model.NewXRefTableEntryGen0(asset)
	annotation := types.Dict{
		"RichMediaContent": types.Dict{
			"Type":   types.Name("RichMediaContent"),
			"Assets": types.Dict{"Names": types.Array{types.StringLiteral(assetName), *types.NewIndirectRef(21, 0)}},
		},
	}
	if err := validateRichMediaAnnotation(t.Context(), ctx.XRefTable, annotation, "RichMedia"); err != nil {
		t.Fatal(err)
	}
	if _, found := inventoryTarget(ctx, assetName); found {
		t.Fatalf("embedded RichMedia asset was classified as an external target: %q", assetName)
	}
}

func TestRichMediaValidationInventory(t *testing.T) {
	ctx := externalReferenceContext(t)
	err := validateRichMediaAnnotation(t.Context(), ctx.XRefTable, types.Dict{}, "RichMedia")
	if err == nil || !strings.Contains(err.Error(), "RichMediaContent") {
		t.Fatalf("missing RichMediaContent: got %v", err)
	}

	annotation := types.Dict{
		"Type":              types.Name("Annot"),
		"Subtype":           types.Name("RichMedia"),
		"Rect":              types.Array{types.Integer(0), types.Integer(0), types.Integer(10), types.Integer(10)},
		"RichMediaSettings": types.Integer(1),
		"RichMediaContent": types.Dict{
			"Type":           types.Name("RichMediaContent"),
			"Configurations": types.Integer(1),
			"Views":          types.Integer(1),
		},
	}
	if _, err := validateAnnotationDict(t.Context(), ctx.XRefTable, annotation, 0); err != nil {
		t.Fatalf("minimal RichMedia validation: %v", err)
	}
}

func TestRichMediaAssetRequiresEmbeddedFile(t *testing.T) {
	for _, tt := range []struct {
		name  string
		asset types.Dict
		want  string
	}{
		{
			name:  "missing EF",
			asset: types.Dict{"Type": types.Name("Filespec"), "F": types.StringLiteral("media.swf")},
			want:  "missing EF dictionary",
		},
		{
			name:  "URL file specification",
			asset: urlSpecification("https://example.invalid/media.swf"),
			want:  "expected embedded file specification",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := externalReferenceContext(t)
			annotation := types.Dict{
				"RichMediaContent": types.Dict{
					"Assets": types.Dict{"Names": types.Array{types.StringLiteral("media.swf"), tt.asset}},
				},
			}
			err := validateRichMediaAnnotation(t.Context(), ctx.XRefTable, annotation, "RichMedia")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want error containing %q", err, tt.want)
			}
			if len(ctx.URIs) != 0 {
				t.Fatalf("rejected RichMedia asset entered external reference inventory: %v", ctx.URIs)
			}
		})
	}
}

func TestRichMediaAssetTraversalGuards(t *testing.T) {
	t.Run("cycle", func(t *testing.T) {
		ctx := externalReferenceContext(t)
		ctx.XRefTable.Table[30] = model.NewXRefTableEntryGen0(types.Dict{
			"Kids": types.Array{*types.NewIndirectRef(30, 0)},
		})
		annotation := types.Dict{
			"RichMediaContent": types.Dict{"Assets": *types.NewIndirectRef(30, 0)},
		}
		err := validateRichMediaAnnotation(t.Context(), ctx.XRefTable, annotation, "RichMedia")
		if !errors.Is(err, model.ErrNameTreeCycle) {
			t.Fatalf("got %v, want name-tree cycle", err)
		}
	})

	t.Run("object context", func(t *testing.T) {
		ctx := externalReferenceContext(t)
		asset := types.Dict{
			"Type": types.Name("Filespec"),
			"F":    types.StringLiteral("media.swf"),
			"EF":   types.Dict{"F": *types.NewIndirectRef(20, 0)},
		}
		annotation := types.Dict{
			"RichMediaContent": types.Dict{
				"Assets": types.Dict{"Names": types.Array{types.StringLiteral("media.swf"), asset}},
			},
		}
		err := validateRichMediaAnnotation(t.Context(), ctx.XRefTable, annotation, "RichMedia")
		if err == nil || !strings.Contains(err.Error(), "obj#20") {
			t.Fatalf("got %v, want object context for object 20", err)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		ctx := externalReferenceContext(t)
		c, cancel := context.WithCancel(t.Context())
		cancel()
		annotation := types.Dict{
			"RichMediaContent": types.Dict{"Assets": types.Dict{"Names": types.Array{}}},
		}
		err := validateRichMediaAnnotation(c, ctx.XRefTable, annotation, "RichMedia")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v, want context cancellation", err)
		}
	})
}
