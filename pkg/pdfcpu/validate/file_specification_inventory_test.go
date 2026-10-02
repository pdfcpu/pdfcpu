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
