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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func keywordOperationTestContext(t *testing.T) *model.Context {
	t.Helper()

	f, err := os.Open(filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	ctx, err := Read(f, model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func keywordMetadataContent() []byte {
	return []byte(
		"<x:xmpmeta><rdf:RDF><rdf:Description>" +
			"<pdf:Keywords>old</pdf:Keywords>" +
			"<dc:subject><rdf:Bag><rdf:li>old</rdf:li></rdf:Bag></dc:subject>" +
			"</rdf:Description></rdf:RDF></x:xmpmeta>",
	)
}

func installDirectKeywordMetadata(t *testing.T, ctx *model.Context, sd types.StreamDict) {
	t.Helper()

	rootDict, err := ctx.Catalog()
	if err != nil {
		t.Fatal(err)
	}
	rootDict["Metadata"] = sd
	ctx.CatalogXMPMeta = &model.XMPMeta{}
}

func installIndirectKeywordMetadata(t *testing.T, ctx *model.Context, sd types.StreamDict) *types.IndirectRef {
	t.Helper()

	indRef, err := ctx.IndRefForNewObject(sd)
	if err != nil {
		t.Fatal(err)
	}
	rootDict, err := ctx.Catalog()
	if err != nil {
		t.Fatal(err)
	}
	rootDict["Metadata"] = *indRef
	ctx.CatalogXMPMeta = &model.XMPMeta{}
	return indRef
}

func assertKeywordMetadataRemoved(t *testing.T, sd types.StreamDict) {
	t.Helper()

	if strings.Contains(string(sd.Raw), "old") {
		t.Fatalf("expected keyword metadata removal, got %q", sd.Raw)
	}
}

// TestKeywordsAddPersistsDirectMetadata verifies valid direct metadata streams are updated without panicking.
func TestKeywordsAddPersistsDirectMetadata(t *testing.T) {
	ctx := keywordOperationTestContext(t)
	installDirectKeywordMetadata(t, ctx, types.StreamDict{
		Dict: types.Dict{},
		Raw:  keywordMetadataContent(),
	})

	if err := KeywordsAdd(ctx, []string{"new"}); err != nil {
		t.Fatal(err)
	}
	rootDict, err := ctx.Catalog()
	if err != nil {
		t.Fatal(err)
	}
	sd, ok := rootDict["Metadata"].(types.StreamDict)
	if !ok {
		t.Fatalf("expected direct metadata stream, got %T", rootDict["Metadata"])
	}
	assertKeywordMetadataRemoved(t, sd)
}

// TestKeywordsAddPersistsIndirectMetadata verifies indirect metadata stream updates reach the xref entry.
func TestKeywordsAddPersistsIndirectMetadata(t *testing.T) {
	ctx := keywordOperationTestContext(t)
	indRef := installIndirectKeywordMetadata(t, ctx, types.StreamDict{
		Dict: types.Dict{},
		Raw:  keywordMetadataContent(),
	})

	if err := KeywordsAdd(ctx, []string{"new"}); err != nil {
		t.Fatal(err)
	}
	entry, found := ctx.FindTableEntryForIndRef(indRef)
	if !found || entry == nil {
		t.Fatal("expected metadata xref entry")
	}
	sd, ok := entry.Object.(types.StreamDict)
	if !ok {
		t.Fatalf("expected indirect metadata stream, got %T", entry.Object)
	}
	assertKeywordMetadataRemoved(t, sd)
}

// TestKeywordMetadataDecodeErrorsPreserveCause verifies add and remove propagate metadata decode failures.
func TestKeywordMetadataDecodeErrorsPreserveCause(t *testing.T) {
	tests := []struct {
		name string
		run  func(*model.Context) error
	}{
		{name: "add", run: func(ctx *model.Context) error {
			return KeywordsAdd(ctx, []string{"new"})
		}},
		{name: "remove all", run: func(ctx *model.Context) error {
			_, err := KeywordsRemove(ctx, nil)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := keywordOperationTestContext(t)
			installDirectKeywordMetadata(t, ctx, types.StreamDict{
				Dict: types.Dict{},
				Raw:  keywordMetadataContent(),
				FilterPipeline: []types.PDFFilter{
					{Name: filter.JPX},
					{Name: filter.ASCIIHex},
				},
			})

			err := tt.run(ctx)
			if !errors.Is(err, filter.ErrUnsupportedFilter) {
				t.Fatalf("expected %v, got %v", filter.ErrUnsupportedFilter, err)
			}
			if !strings.Contains(err.Error(), "catalog Metadata stream: decode") {
				t.Fatalf("expected metadata decode context, got %v", err)
			}
		})
	}
}

// TestKeywordsAddReportsPreparationPhase verifies Info dictionary and file ID failures remain distinguishable.
func TestKeywordsAddReportsPreparationPhase(t *testing.T) {
	t.Run("Info dictionary", func(t *testing.T) {
		ctx := keywordOperationTestContext(t)
		indRef, err := ctx.IndRefForNewObject(types.Integer(1))
		if err != nil {
			t.Fatal(err)
		}
		ctx.Info = indRef

		err = KeywordsAdd(ctx, []string{"keyword"})
		if err == nil || !strings.Contains(err.Error(), "Info dictionary: ensure") {
			t.Fatalf("expected Info dictionary preparation context, got %v", err)
		}
	})

	t.Run("file ID", func(t *testing.T) {
		ctx := keywordOperationTestContext(t)
		ctx.ID = types.Array{types.HexLiteral("one")}

		err := KeywordsAdd(ctx, []string{"keyword"})
		if err == nil || !strings.Contains(err.Error(), "file ID: ensure") {
			t.Fatalf("expected file ID preparation context, got %v", err)
		}
	})

	t.Run("PDF 2.0 missing Info", func(t *testing.T) {
		ctx := keywordOperationTestContext(t)
		version := model.V20
		ctx.HeaderVersion = &version
		ctx.RootVersion = nil
		ctx.Info = nil
		ctx.ID = nil

		err := KeywordsAdd(ctx, []string{"keyword"})
		if err == nil || !strings.Contains(err.Error(), "Info dictionary: missing") {
			t.Fatalf("expected missing Info dictionary context, got %v", err)
		}
		if ctx.ID != nil {
			t.Fatalf("file ID mutated before Info dictionary failure: %v", ctx.ID)
		}
	})
}

// TestKeywordsRemoveAllReportsActualChange verifies remove-all accurately reports and caches its result.
func TestKeywordsRemoveAllReportsActualChange(t *testing.T) {
	t.Run("no keywords", func(t *testing.T) {
		ctx := keywordOperationTestContext(t)
		if err := ensureInfoDict(ctx); err != nil {
			t.Fatal(err)
		}
		d, err := ctx.DereferenceDict(*ctx.Info)
		if err != nil {
			t.Fatal(err)
		}
		delete(d, "Keywords")
		ctx.KeywordList = types.StringSet{}
		ctx.CatalogXMPMeta = nil

		removed, err := KeywordsRemove(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if removed {
			t.Fatal("expected no keywords removed")
		}
	})

	t.Run("clear list", func(t *testing.T) {
		ctx := keywordOperationTestContext(t)
		if err := KeywordsAdd(ctx, []string{"alpha", "beta"}); err != nil {
			t.Fatal(err)
		}

		removed, err := KeywordsRemove(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !removed {
			t.Fatal("expected keywords removed")
		}
		keywords, err := KeywordsList(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(keywords) != 0 {
			t.Fatalf("expected empty keyword list, got %v", keywords)
		}
	})
}

// TestKeywordsRemoveMatchesTrimmedInput verifies add and remove apply consistent whitespace semantics.
func TestKeywordsRemoveMatchesTrimmedInput(t *testing.T) {
	ctx := keywordOperationTestContext(t)
	if err := KeywordsAdd(ctx, []string{"alpha"}); err != nil {
		t.Fatal(err)
	}

	removed, err := KeywordsRemove(ctx, []string{" alpha "})
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Fatal("expected trimmed keyword match")
	}
	keywords, err := KeywordsList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(keywords) != 0 {
		t.Fatalf("expected empty keyword list, got %v", keywords)
	}
}

// TestKeywordsRemoveReportsInfoDictionaryContext verifies Info dereference errors receive semantic context.
func TestKeywordsRemoveReportsInfoDictionaryContext(t *testing.T) {
	ctx := keywordOperationTestContext(t)
	indRef, err := ctx.IndRefForNewObject(types.Integer(1))
	if err != nil {
		t.Fatal(err)
	}
	ctx.Info = indRef

	_, err = KeywordsRemove(ctx, []string{"keyword"})
	if err == nil || !strings.Contains(err.Error(), "Info dictionary") {
		t.Fatalf("expected Info dictionary context, got %v", err)
	}
}

// TestKeywordsRemoveReportsMissingMetadataStream verifies malformed metadata state returns an error instead of panicking.
func TestKeywordsRemoveReportsMissingMetadataStream(t *testing.T) {
	ctx := keywordOperationTestContext(t)
	rootDict, err := ctx.Catalog()
	if err != nil {
		t.Fatal(err)
	}
	rootDict["Metadata"] = *types.NewIndirectRef(999, 0)
	ctx.CatalogXMPMeta = &model.XMPMeta{}

	_, err = KeywordsRemove(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "catalog Metadata stream") {
		t.Fatalf("expected metadata stream context, got %v", err)
	}
}
