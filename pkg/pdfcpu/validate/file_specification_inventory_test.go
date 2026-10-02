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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func urlSpecification(uri string) types.Dict {
	return types.Dict{
		"Type": types.Name("Filespec"),
		"FS":   types.Name("URL"),
		"F":    types.StringLiteral(uri),
	}
}

func inventoryTarget(ctx *model.Context, target string) (linkTarget, bool) {
	return inventoryTargetAtPage(ctx, ctx.CurPage, target)
}

func inventoryTargetAtPage(ctx *model.Context, page int, target string) (linkTarget, bool) {
	targets := ctx.URIs[page]
	if targets == nil {
		return linkTarget{}, false
	}
	encoded, found := targets[target]
	if !found {
		return linkTarget{}, false
	}
	return decodeLinkTarget(encoded)
}

func TestFileAttachmentExternalReferenceInventory(t *testing.T) {
	t.Run("plain file", func(t *testing.T) {
		ctx := externalReferenceContext(t)
		fileName := "attachments/report.txt"
		d := types.Dict{"FS": types.StringLiteral(fileName)}
		if err := validateAnnotationDictFileAttachment(ctx.XRefTable, d, "FileAttachment"); err != nil {
			t.Fatal(err)
		}
		want := linkTarget{kind: linkTargetFile, source: linkSourceFileAttachment}
		if got, found := inventoryTarget(ctx, fileName); !found || got != want {
			t.Fatalf("plain attachment: got %+v, found=%t, want %+v", got, found, want)
		}
	})

	t.Run("URL file specification", func(t *testing.T) {
		ctx := externalReferenceContext(t)
		uri := "https://example.invalid/attachment.txt"
		d := types.Dict{"FS": urlSpecification(uri)}
		if err := validateAnnotationDictFileAttachment(ctx.XRefTable, d, "FileAttachment"); err != nil {
			t.Fatal(err)
		}
		want := linkTarget{kind: linkTargetURI, source: linkSourceFileAttachment}
		if got, found := inventoryTarget(ctx, uri); !found || got != want {
			t.Fatalf("URL attachment: got %+v, found=%t, want %+v", got, found, want)
		}
	})

	t.Run("embedded file", func(t *testing.T) {
		ctx := externalReferenceContext(t)
		fileName := "attachments/embedded.txt"
		ctx.XRefTable.Table[20] = model.NewXRefTableEntryGen0(types.StreamDict{
			Dict: types.Dict{"Type": types.Name("EmbeddedFile")},
		})
		fileSpec := types.Dict{
			"Type": types.Name("Filespec"),
			"F":    types.StringLiteral(fileName),
			"UF":   types.StringLiteral(fileName),
			"EF":   types.Dict{"F": *types.NewIndirectRef(20, 0)},
		}
		d := types.Dict{"FS": fileSpec}
		if err := validateAnnotationDictFileAttachment(ctx.XRefTable, d, "FileAttachment"); err != nil {
			t.Fatal(err)
		}
		if target, found := inventoryTarget(ctx, fileName); found {
			t.Fatalf("embedded attachment unexpectedly collected as external: %+v", target)
		}
	})

	t.Run("empty embedded-file dictionary", func(t *testing.T) {
		ctx := externalReferenceContext(t)
		fileName := "attachments/report.txt"
		fileSpec := types.Dict{
			"Type": types.Name("Filespec"),
			"F":    types.StringLiteral(fileName),
			"EF":   types.Dict{},
		}
		d := types.Dict{"FS": fileSpec}
		if err := validateAnnotationDictFileAttachment(ctx.XRefTable, d, "FileAttachment"); err != nil {
			t.Fatal(err)
		}
		want := linkTarget{kind: linkTargetFile, source: linkSourceFileAttachment}
		if got, found := inventoryTarget(ctx, fileName); !found || got != want {
			t.Fatalf("attachment with empty EF: got %+v, found=%t, want %+v", got, found, want)
		}
	})
}

func TestEmbeddedFilesNameTreeRequiresEmbeddedStream(t *testing.T) {
	validateTree := func(t *testing.T, ctx *model.Context, value types.Object) error {
		t.Helper()
		tree := types.Dict{"Names": types.Array{types.StringLiteral("attachment.txt"), value}}
		_, _, _, err := validateNameTree(t.Context(), ctx.XRefTable, "EmbeddedFiles", tree, 0, true)
		return err
	}

	t.Run("matching stream", func(t *testing.T) {
		ctx := externalReferenceContext(t)
		fileName := "attachment.txt"
		ctx.XRefTable.Table[20] = model.NewXRefTableEntryGen0(types.StreamDict{
			Dict: types.Dict{"Type": types.Name("EmbeddedFile")},
		})
		fileSpec := types.Dict{
			"Type": types.Name("Filespec"),
			"F":    types.StringLiteral(fileName),
			"UF":   types.StringLiteral(fileName),
			"EF":   types.Dict{"F": *types.NewIndirectRef(20, 0)},
		}
		if err := validateTree(t, ctx, fileSpec); err != nil {
			t.Fatal(err)
		}
		if target, found := inventoryTarget(ctx, fileName); found {
			t.Fatalf("embedded-file tree value unexpectedly collected as external: %+v", target)
		}
	})

	for _, tt := range []struct {
		name  string
		value func(*model.Context) types.Object
	}{
		{
			name: "plain file",
			value: func(*model.Context) types.Object {
				return types.StringLiteral("attachment.txt")
			},
		},
		{
			name: "URL file specification",
			value: func(*model.Context) types.Object {
				return urlSpecification("https://example.invalid/attachment.txt")
			},
		},
		{
			name: "empty EF dictionary",
			value: func(*model.Context) types.Object {
				return types.Dict{
					"Type": types.Name("Filespec"),
					"F":    types.StringLiteral("attachment.txt"),
					"EF":   types.Dict{},
				}
			},
		},
		{
			name: "mismatched EF key",
			value: func(ctx *model.Context) types.Object {
				ctx.XRefTable.Table[20] = model.NewXRefTableEntryGen0(types.StreamDict{
					Dict: types.Dict{"Type": types.Name("EmbeddedFile")},
				})
				return types.Dict{
					"Type": types.Name("Filespec"),
					"F":    types.StringLiteral("attachment.txt"),
					"EF":   types.Dict{"UF": *types.NewIndirectRef(20, 0)},
				}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := externalReferenceContext(t)
			value := tt.value(ctx)
			if err := validateTree(t, ctx, value); err == nil {
				t.Fatal("non-embedded name-tree value accepted")
			}
			if len(ctx.URIs) != 0 {
				t.Fatalf("rejected name-tree value collected targets: %+v", ctx.URIs)
			}
		})
	}
}

func TestURLFileSpecificationIsCheckedAsURI(t *testing.T) {
	ctx := externalReferenceContext(t)
	uri := "https://example.invalid/file-spec"
	if _, err := validateFileSpecification(ctx.XRefTable, urlSpecification(uri)); err != nil {
		t.Fatal(err)
	}
	wantTarget := linkTarget{kind: linkTargetURI, source: linkSourceURLFileSpecification}
	if got, found := inventoryTarget(ctx, uri); !found || got != wantTarget {
		t.Fatalf("collected target: got %+v, want %+v", got, wantTarget)
	}
	tr := &successfulLinkTransport{}
	failed, err := checkLinks(t.Context(), ctx.XRefTable, http.Client{Transport: tr}, []int{ctx.CurPage})
	if err != nil {
		t.Fatal(err)
	}
	if failed || tr.calls != 1 {
		t.Fatalf("URL file specification: failed=%t, transport calls=%d", failed, tr.calls)
	}
	if got := ctx.URIs[ctx.CurPage][uri]; got != "" {
		t.Fatalf("link status: got %q, want success", got)
	}
}

func TestAlternatePresentationsResourceInventory(t *testing.T) {
	const uri = "https://example.invalid/slideshow.svg"

	newContext := func(t *testing.T) *model.Context {
		t.Helper()
		ctx := externalReferenceContext(t)
		fileSpec := urlSpecification(uri)
		fileSpec["Type"] = types.Name("Filespec")
		ctx.XRefTable.Table[30] = model.NewXRefTableEntryGen0(fileSpec)
		return ctx
	}

	t.Run("specification name tree", func(t *testing.T) {
		ctx := newContext(t)
		d := types.Dict{
			"Type":    types.Name("SlideShow"),
			"Subtype": types.Name("Embedded"),
			"Resources": types.Dict{
				"Names": types.Array{types.StringLiteral("slideshow.svg"), *types.NewIndirectRef(30, 0)},
			},
			"StartResource": types.StringLiteral("slideshow.svg"),
		}
		if err := validateAlternatePresentationsNameTreeValue(t.Context(), ctx.XRefTable, d, model.V14); err != nil {
			t.Fatal(err)
		}
		want := linkTarget{kind: linkTargetURI, source: linkSourceAlternatePresentation}
		if got, found := inventoryTargetAtPage(ctx, 0, uri); !found || got != want {
			t.Fatalf("slide show URL resource: got %+v, found=%t, want %+v", got, found, want)
		}
		if _, found := inventoryTarget(ctx, uri); found {
			t.Fatal("document resource assigned to current page")
		}
	})

	t.Run("image XObject", func(t *testing.T) {
		ctx := externalReferenceContext(t)
		ctx.XRefTable.Table[31] = model.NewXRefTableEntryGen0(types.StreamDict{
			Dict: types.Dict{"Type": types.Name("XObject"), "Subtype": types.Name("Image")},
		})
		d := types.Dict{
			"Type":    types.Name("SlideShow"),
			"Subtype": types.Name("Embedded"),
			"Resources": types.Dict{
				"Names": types.Array{types.StringLiteral("image.jpg"), *types.NewIndirectRef(31, 0)},
			},
			"StartResource": types.StringLiteral("image.jpg"),
		}
		if err := validateAlternatePresentationsNameTreeValue(t.Context(), ctx.XRefTable, d, model.V14); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("embedded file", func(t *testing.T) {
		ctx := externalReferenceContext(t)
		fileName := "slideshow.svg"
		ctx.XRefTable.Table[31] = model.NewXRefTableEntryGen0(types.StreamDict{
			Dict: types.Dict{"Type": types.Name("EmbeddedFile")},
		})
		ctx.XRefTable.Table[30] = model.NewXRefTableEntryGen0(types.Dict{
			"Type": types.Name("Filespec"),
			"F":    types.StringLiteral(fileName),
			"EF":   types.Dict{"F": *types.NewIndirectRef(31, 0)},
		})
		d := types.Dict{
			"Type":    types.Name("SlideShow"),
			"Subtype": types.Name("Embedded"),
			"Resources": types.Dict{
				"Names": types.Array{types.StringLiteral(fileName), *types.NewIndirectRef(30, 0)},
			},
			"StartResource": types.StringLiteral(fileName),
		}
		if err := validateAlternatePresentationsNameTreeValue(t.Context(), ctx.XRefTable, d, model.V14); err != nil {
			t.Fatal(err)
		}
		if target, found := inventoryTarget(ctx, fileName); found {
			t.Fatalf("embedded slide show resource unexpectedly collected as external: %+v", target)
		}
	})

	t.Run("missing start resource", func(t *testing.T) {
		ctx := newContext(t)
		d := types.Dict{
			"Type":    types.Name("SlideShow"),
			"Subtype": types.Name("Embedded"),
			"Resources": types.Dict{
				"Names": types.Array{types.StringLiteral("slideshow.svg"), *types.NewIndirectRef(30, 0)},
			},
			"StartResource": types.StringLiteral("missing.svg"),
		}
		if err := validateAlternatePresentationsNameTreeValue(t.Context(), ctx.XRefTable, d, model.V14); err == nil {
			t.Fatal("missing StartResource accepted")
		}
	})

	t.Run("missing resource type", func(t *testing.T) {
		ctx := externalReferenceContext(t)
		ctx.XRefTable.Table[31] = model.NewXRefTableEntryGen0(types.StreamDict{Dict: types.Dict{}})
		d := types.Dict{
			"Type":    types.Name("SlideShow"),
			"Subtype": types.Name("Embedded"),
			"Resources": types.Dict{
				"Names": types.Array{types.StringLiteral("image.jpg"), *types.NewIndirectRef(31, 0)},
			},
			"StartResource": types.StringLiteral("image.jpg"),
		}
		if err := validateAlternatePresentationsNameTreeValue(t.Context(), ctx.XRefTable, d, model.V14); err == nil {
			t.Fatal("slide show resource without Type accepted")
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		ctx := newContext(t)
		d := types.Dict{
			"Type":    types.Name("SlideShow"),
			"Subtype": types.Name("Embedded"),
			"Resources": types.Dict{
				"Names": types.Array{types.StringLiteral("slideshow.svg"), *types.NewIndirectRef(30, 0)},
			},
			"StartResource": types.StringLiteral("slideshow.svg"),
		}
		c, cancel := context.WithCancel(t.Context())
		cancel()
		err := validateAlternatePresentationsNameTreeValue(c, ctx.XRefTable, d, model.V14)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v, want context cancellation", err)
		}
	})
}

func TestAlternatePresentationsLegacyResourceArray(t *testing.T) {
	const uri = "https://example.invalid/slideshow.svg"
	d := types.Dict{
		"Type":          types.Name("SlideShow"),
		"Subtype":       types.Name("Embedded"),
		"Resources":     types.Array{types.StringLiteral("slideshow.svg"), *types.NewIndirectRef(30, 0)},
		"StartResource": types.StringLiteral("slideshow.svg"),
	}
	newContext := func(t *testing.T, resource types.Object) *model.Context {
		t.Helper()
		ctx := externalReferenceContext(t)
		ctx.XRefTable.Table[30] = model.NewXRefTableEntryGen0(resource)
		return ctx
	}

	ctx := newContext(t, urlSpecification(uri))
	if err := validateAlternatePresentationsNameTreeValue(t.Context(), ctx.XRefTable, d, model.V14); err == nil {
		t.Fatal("legacy flat Resources array accepted")
	}

	ctx = newContext(t, urlSpecification(uri))
	ctx.XRefTable.ValidationMode = model.ValidationRelaxed
	if err := validateAlternatePresentationsNameTreeValue(t.Context(), ctx.XRefTable, d, model.V14); err != nil {
		t.Fatalf("relaxed legacy Resources array rejected: %v", err)
	}
	want := linkTarget{kind: linkTargetURI, source: linkSourceAlternatePresentation}
	if got, found := inventoryTargetAtPage(ctx, 0, uri); !found || got != want {
		t.Fatalf("legacy slide show URL resource: got %+v, found=%t, want %+v", got, found, want)
	}
	notices := ctx.ValidationReport().Notices()
	if len(notices) != 1 || notices[0].Disposition != model.NoticeSkipped ||
		notices[0].Message != "slideShowDict.Resources: accepted legacy flat resource array" {
		t.Fatalf("legacy slide show notices: %+v", notices)
	}

	ctx = newContext(t, types.StreamDict{Dict: types.Dict{}})
	ctx.XRefTable.ValidationMode = model.ValidationRelaxed
	if err := validateAlternatePresentationsNameTreeValue(t.Context(), ctx.XRefTable, d, model.V14); err != nil {
		t.Fatalf("relaxed untyped legacy resource rejected: %v", err)
	}
}
