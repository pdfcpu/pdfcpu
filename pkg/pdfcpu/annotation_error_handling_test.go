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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type failingAnnotationRenderer struct {
	model.LinkAnnotation
	err error
}

func (a failingAnnotationRenderer) RenderDict(*model.XRefTable, *types.IndirectRef) (types.Dict, error) {
	return nil, a.err
}

func annotationTestContext(t *testing.T) *model.Context {
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
	ctx.PageCount = 1
	return ctx
}

func annotationTestRenderer() model.LinkAnnotation {
	return annotationTestRendererWithID("pdfcpu-test-annotation")
}

func annotationTestRendererWithID(id string) model.LinkAnnotation {
	return model.NewLinkAnnotation(
		*types.NewRectangle(0, 0, 10, 10),
		0,
		"",
		id,
		"",
		0,
		nil,
		nil,
		"https://pdfcpu.io",
		nil,
		false,
		0,
		model.BSSolid,
	)
}

func annotationTestPageDictIndRef(t *testing.T, ctx *model.Context) *types.IndirectRef {
	t.Helper()

	pageDictIndRef, err := ctx.PageDictIndRef(1)
	if err != nil {
		t.Fatal(err)
	}
	return pageDictIndRef
}

func annotationTestPageDict(t *testing.T, ctx *model.Context) types.Dict {
	t.Helper()

	pageDictIndRef := annotationTestPageDictIndRef(t, ctx)
	d, err := ctx.DereferenceDict(*pageDictIndRef)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func assertMalformedIndirectAnnotsState(
	t *testing.T,
	ctx *model.Context,
	pageDict types.Dict,
	annotsIndRef types.IndirectRef,
	generation,
	refCount,
	cacheObjNr int) {
	t.Helper()

	entry, found := ctx.FindTableEntryForIndRef(&annotsIndRef)
	if !found || entry.Free || entry.Object == nil {
		t.Fatalf("expected live Annots xref entry, got found=%t entry=%+v", found, entry)
	}
	if entry.Generation == nil || *entry.Generation != generation || entry.RefCount != refCount {
		t.Fatalf("expected unchanged Annots xref metadata, got %+v", entry)
	}
	o, found := pageDict.Find("Annots")
	if !found {
		t.Fatal("expected page Annots entry")
	}
	gotIndRef, ok := o.(types.IndirectRef)
	if !ok || gotIndRef.ObjectNumber != annotsIndRef.ObjectNumber || gotIndRef.GenerationNumber != annotsIndRef.GenerationNumber {
		t.Fatalf("expected unchanged page Annots reference, got %v", o)
	}
	cached, found := ctx.PageAnnots[1][model.AnnLink].Map[cacheObjNr]
	if !found || cached.ID() != "cached" {
		t.Fatalf("expected unchanged annotation cache, got %v", ctx.PageAnnots)
	}
}

type annotationXRefSnapshot struct {
	generation int
	refCount   int
}

func annotationTestAddPair(
	t *testing.T,
	ctx *model.Context,
	pageDict types.Dict) (types.IndirectRef, types.IndirectRef) {
	t.Helper()

	annotations := map[int][]model.AnnotationRenderer{
		1: {
			annotationTestRendererWithID("first"),
			annotationTestRendererWithID("second"),
		},
	}
	if ok, err := AddAnnotationsMap(ctx, annotations, false); err != nil || !ok {
		t.Fatalf("add annotations: ok=%t err=%v", ok, err)
	}
	o, found := pageDict.Find("Annots")
	if !found {
		t.Fatal("expected Annots entry")
	}
	annots, ok := o.(types.Array)
	if !ok || len(annots) != 2 {
		t.Fatalf("expected two direct Annots entries, got %v", o)
	}
	return annots[0].(types.IndirectRef), annots[1].(types.IndirectRef)
}

func annotationTestXRefSnapshot(t *testing.T, ctx *model.Context, indRef types.IndirectRef) annotationXRefSnapshot {
	t.Helper()

	entry, found := ctx.FindTableEntryForIndRef(&indRef)
	if !found || entry.Generation == nil {
		t.Fatal("expected annotation xref entry")
	}
	return annotationXRefSnapshot{generation: *entry.Generation, refCount: entry.RefCount}
}

func assertLiveAnnotationXRef(
	t *testing.T,
	ctx *model.Context,
	indRef types.IndirectRef,
	want annotationXRefSnapshot) {
	t.Helper()

	entry, found := ctx.FindTableEntryForIndRef(&indRef)
	if !found || entry.Free || entry.Object == nil {
		t.Fatalf("expected annotation xref entry unchanged, got found=%t entry=%+v", found, entry)
	}
	if entry.Generation == nil || *entry.Generation != want.generation || entry.RefCount != want.refCount {
		t.Fatalf("expected annotation xref metadata unchanged, got %+v", entry)
	}
}

func assertTwoPageAnnotations(t *testing.T, pageDict types.Dict) {
	t.Helper()

	o, found := pageDict.Find("Annots")
	if !found {
		t.Fatal("expected page Annots entry unchanged")
	}
	annots, ok := o.(types.Array)
	if !ok || len(annots) != 2 {
		t.Fatalf("expected two page Annots entries, got %v", o)
	}
}

func annotationTestAddSingle(
	t *testing.T,
	ctx *model.Context,
	pageDict types.Dict,
	id string) types.IndirectRef {
	t.Helper()

	pageDict.Delete("Annots")
	ctx.PageAnnots = map[int]model.PgAnnots{}
	if ok, err := AddAnnotations(ctx, types.IntSet{1: true}, annotationTestRendererWithID(id), false); err != nil || !ok {
		t.Fatalf("add annotation: ok=%t err=%v", ok, err)
	}
	o, found := pageDict.Find("Annots")
	if !found {
		t.Fatal("expected Annots entry")
	}
	annots, ok := o.(types.Array)
	if !ok || len(annots) != 1 {
		t.Fatalf("expected one annotation, got %v", o)
	}
	return annots[0].(types.IndirectRef)
}

func assertSingleAnnotationState(
	t *testing.T,
	ctx *model.Context,
	pageDict types.Dict,
	indRef types.IndirectRef,
	want annotationXRefSnapshot,
	id string) {
	t.Helper()

	assertLiveAnnotationXRef(t, ctx, indRef, want)
	o, found := pageDict.Find("Annots")
	if !found {
		t.Fatal("expected page Annots entry unchanged")
	}
	annots, ok := o.(types.Array)
	if !ok || len(annots) != 1 || annots[0] != indRef {
		t.Fatalf("expected one unchanged page annotation, got %v", o)
	}
	cached, found := ctx.PageAnnots[1][model.AnnLink].Map[indRef.ObjectNumber.Value()]
	if !found || cached.ID() != id {
		t.Fatalf("expected annotation cache unchanged, got %v", ctx.PageAnnots)
	}
}

func assertNoAnnotationAddMutation(
	t *testing.T,
	ctx *model.Context,
	pageDict types.Dict,
	xrefEntries int) {
	t.Helper()

	if _, found := pageDict.Find("Annots"); found {
		t.Fatal("expected page Annots entry unchanged")
	}
	if len(ctx.PageAnnots) != 0 {
		t.Fatalf("expected annotation cache unchanged, got %v", ctx.PageAnnots)
	}
	if len(ctx.Table) != xrefEntries {
		t.Fatalf("expected xref table unchanged, got %d entries", len(ctx.Table))
	}
	if ctx.Write.Increment {
		t.Fatal("expected incremental write state unchanged")
	}
}

func TestAnnotationOperationsRejectMissingInput(t *testing.T) {
	ctx := annotationTestContext(t)
	ann := annotationTestRenderer()

	tests := []struct {
		name    string
		fn      func() error
		wantErr error
	}{
		{
			name: "add annotation missing context",
			fn: func() error {
				_, _, err := AddAnnotation(nil, nil, nil, 1, ann, false)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "add annotations missing xref table",
			fn: func() error {
				_, err := AddAnnotations(&model.Context{}, types.IntSet{1: true}, ann, false)
				return err
			},
			wantErr: ErrMissingXRefTable,
		},
		{
			name: "add annotations missing annotation",
			fn: func() error {
				_, err := AddAnnotations(ctx, types.IntSet{1: true}, nil, false)
				return err
			},
			wantErr: ErrMissingAnnotation,
		},
		{
			name: "add annotations increment missing read context",
			fn: func() error {
				incrCtx := annotationTestContext(t)
				incrCtx.Read = nil
				_, err := AddAnnotations(incrCtx, types.IntSet{1: true}, ann, true)
				return err
			},
			wantErr: ErrMissingReadContext,
		},
		{
			name: "add annotation increment missing write context",
			fn: func() error {
				incrCtx := annotationTestContext(t)
				incrCtx.Write = nil
				pageDictIndRef := annotationTestPageDictIndRef(t, incrCtx)
				_, _, err := AddAnnotation(incrCtx, pageDictIndRef, annotationTestPageDict(t, incrCtx), 1, ann, true)
				return err
			},
			wantErr: ErrMissingWriteContext,
		},
		{
			name: "add annotations invalid page",
			fn: func() error {
				_, err := AddAnnotations(ctx, types.IntSet{0: true}, ann, false)
				return err
			},
			wantErr: ErrInvalidPageNumber,
		},
		{
			name: "remove annotations missing context",
			fn: func() error {
				_, err := RemoveAnnotations(nil, nil, nil, nil, false)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "remove annotations from page dict missing xref table",
			fn: func() error {
				_, err := RemoveAnnotationsFromPageDict(&model.Context{}, nil, nil, nil, types.Dict{}, 1, 1, false)
				return err
			},
			wantErr: ErrMissingXRefTable,
		},
		{
			name: "remove annotations increment missing read context",
			fn: func() error {
				incrCtx := annotationTestContext(t)
				incrCtx.Read = nil
				_, err := RemoveAnnotations(incrCtx, nil, nil, nil, true)
				return err
			},
			wantErr: ErrMissingReadContext,
		},
		{
			name: "remove from page dict increment missing write context",
			fn: func() error {
				incrCtx := annotationTestContext(t)
				incrCtx.Write = nil
				_, err := RemoveAnnotationsFromPageDict(incrCtx, nil, nil, nil, types.Dict{}, 1, 1, true)
				return err
			},
			wantErr: ErrMissingWriteContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestAddAnnotationsWrapsRendererErrorWithPageContext(t *testing.T) {
	ctx := annotationTestContext(t)
	wantErr := errors.New("render failed")
	ann := failingAnnotationRenderer{
		LinkAnnotation: annotationTestRenderer(),
		err:            wantErr,
	}

	_, err := AddAnnotations(ctx, types.IntSet{1: true}, ann, false)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	for _, want := range []string{"page 1", "create annotation", "render annotation dict"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestAddAnnotationsReportsInvalidAnnotsEntry(t *testing.T) {
	ctx := annotationTestContext(t)
	annotationTestPageDict(t, ctx)["Annots"] = types.Integer(1)

	_, err := AddAnnotations(ctx, types.IntSet{1: true}, annotationTestRenderer(), false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "page 1 Annots: expected array") {
		t.Fatalf("expected Annots type context, got %q", err.Error())
	}
}

func TestAddAnnotationsCleansCacheOnAnnotsFailure(t *testing.T) {
	ctx := annotationTestContext(t)
	annotationTestPageDict(t, ctx)["Annots"] = types.Integer(1)
	ctx.PageAnnots = map[int]model.PgAnnots{}

	_, err := AddAnnotations(ctx, types.IntSet{1: true}, annotationTestRenderer(), false)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(ctx.PageAnnots) != 0 {
		t.Fatalf("expected annotation cache cleanup, got %v", ctx.PageAnnots)
	}
}

func TestAddAnnotationsReportsIndirectAnnotsDereference(t *testing.T) {
	ctx := annotationTestContext(t)
	annotationTestPageDict(t, ctx)["Annots"] = *types.NewIndirectRef(999, 0)

	_, err := AddAnnotations(ctx, types.IntSet{1: true}, annotationTestRenderer(), false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "page 1 Annots obj#999: dereference") {
		t.Fatalf("expected Annots dereference context, got %q", err.Error())
	}
}

func TestAddAnnotationsInitializesNilCacheMap(t *testing.T) {
	ctx := annotationTestContext(t)
	ctx.PageAnnots[1] = model.PgAnnots{model.AnnLink: model.Annot{}}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected success, got panic: %v", r)
		}
	}()

	ok, err := AddAnnotations(ctx, types.IntSet{1: true}, annotationTestRenderer(), false)
	if err != nil {
		t.Fatalf("add annotation: %v", err)
	}
	if !ok {
		t.Fatal("expected annotation added")
	}
}

func TestAddAnnotationsInitializesNilPageAnnots(t *testing.T) {
	ctx := annotationTestContext(t)
	annotationTestPageDict(t, ctx).Delete("Annots")
	ctx.PageAnnots = nil

	ok, err := AddAnnotations(ctx, types.IntSet{1: true}, annotationTestRenderer(), false)
	if err != nil {
		t.Fatalf("add annotation: %v", err)
	}
	if !ok || ctx.PageAnnots == nil || len(ctx.PageAnnots[1]) == 0 {
		t.Fatalf("expected initialized annotation cache, got %v", ctx.PageAnnots)
	}
}

func TestAddAnnotationCleansAllocatedObjectOnCacheFailure(t *testing.T) {
	ctx := annotationTestContext(t)
	pageDict := annotationTestPageDict(t, ctx)
	pageDict.Delete("Annots")

	recycledIndRef, err := ctx.IndRefForNewObject(types.NewDict())
	if err != nil {
		t.Fatal(err)
	}
	objNr := recycledIndRef.ObjectNumber.Value()
	if err := ctx.FreeObject(objNr); err != nil {
		t.Fatal(err)
	}
	ctx.PageAnnots = map[int]model.PgAnnots{
		1: {
			model.AnnLink: model.Annot{Map: model.AnnotMap{objNr: annotationTestRendererWithID("existing")}},
		},
	}

	pageDictIndRef := annotationTestPageDictIndRef(t, ctx)
	_, _, err = AddAnnotation(ctx, pageDictIndRef, pageDict, 1, annotationTestRenderer(), false)
	if err == nil || !strings.Contains(err.Error(), "cache") {
		t.Fatalf("expected cache insertion error, got %v", err)
	}
	entry, found := ctx.FindTableEntryLight(objNr)
	if !found || !entry.Free || entry.Object != nil {
		t.Fatalf("expected allocated annotation object cleanup, got found=%t entry=%+v", found, entry)
	}
	if cached := ctx.PageAnnots[1][model.AnnLink].Map[objNr]; cached.ID() != "existing" {
		t.Fatalf("expected existing cache entry preserved, got %v", cached)
	}
	if _, found := pageDict.Find("Annots"); found {
		t.Fatal("expected page Annots entry unchanged")
	}
}

func TestSelectedAnnotationPageNrsAreSorted(t *testing.T) {
	got := selectedAnnotationPageNrs(types.IntSet{3: true, 1: true, 2: false, 4: true})
	want := []int{1, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestAddAnnotationsPreflightPreventsMutation(t *testing.T) {
	tests := []struct {
		name    string
		add     func(*model.Context) error
		wantErr error
	}{
		{
			name: "invalid later page",
			add: func(ctx *model.Context) error {
				_, err := AddAnnotations(ctx, types.IntSet{1: true, 2: true}, annotationTestRenderer(), true)
				return err
			},
			wantErr: ErrInvalidPageNumber,
		},
		{
			name: "invalid later renderer",
			add: func(ctx *model.Context) error {
				m := map[int][]model.AnnotationRenderer{1: {annotationTestRenderer(), nil}}
				_, err := AddAnnotationsMap(ctx, m, true)
				return err
			},
			wantErr: ErrMissingAnnotation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := annotationTestContext(t)
			pageDict := annotationTestPageDict(t, ctx)
			pageDict.Delete("Annots")
			ctx.PageAnnots = map[int]model.PgAnnots{}
			xrefEntries := len(ctx.Table)

			err := tt.add(ctx)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			assertNoAnnotationAddMutation(t, ctx, pageDict, xrefEntries)
		})
	}
}

func TestAddAnnotationsMapReturnsDeterministicPageError(t *testing.T) {
	ctx := annotationTestContext(t)
	m := map[int][]model.AnnotationRenderer{
		2: {annotationTestRenderer()},
		0: {annotationTestRenderer()},
	}

	for i := 0; i < 25; i++ {
		_, err := AddAnnotationsMap(ctx, m, false)
		if !errors.Is(err, ErrInvalidPageNumber) || !strings.Contains(err.Error(), "page 0") {
			t.Fatalf("iteration %d: expected page 0 error, got %v", i, err)
		}
	}
}

func TestRemoveAnnotationsReportsInvalidAnnotsEntry(t *testing.T) {
	ctx := annotationTestContext(t)
	ann := annotationTestRenderer()
	annotationTestPageDict(t, ctx)["Annots"] = types.Integer(1)
	ctx.PageAnnots[1] = model.PgAnnots{
		model.AnnLink: model.Annot{Map: model.AnnotMap{1: ann}},
	}

	_, err := RemoveAnnotations(ctx, nil, nil, nil, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "page 1 Annots: expected array") {
		t.Fatalf("expected Annots type context, got %q", err.Error())
	}
}

func TestRemoveAllAnnotationsMalformedIndirectAnnotsDoesNotMutateState(t *testing.T) {
	tests := []struct {
		name        string
		annots      types.Object
		wantErrPart string
	}{
		{
			name:        "Annots object is not an array",
			annots:      types.Integer(1),
			wantErrPart: "Annots: expected array",
		},
		{
			name:        "Annots contains a non-dictionary target",
			annots:      types.Array{types.Integer(1)},
			wantErrPart: "annotation array[0]: expected dict or indirect reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := annotationTestContext(t)
			pageDict := annotationTestPageDict(t, ctx)
			annotsIndRef, err := ctx.IndRefForNewObject(tt.annots)
			if err != nil {
				t.Fatal(err)
			}
			pageDict["Annots"] = *annotsIndRef
			cacheObjNr := 99
			ctx.PageAnnots[1] = model.PgAnnots{
				model.AnnLink: model.Annot{Map: model.AnnotMap{cacheObjNr: annotationTestRendererWithID("cached")}},
			}

			entry, found := ctx.FindTableEntryForIndRef(annotsIndRef)
			if !found || entry.Generation == nil {
				t.Fatal("expected Annots xref entry")
			}
			generation := *entry.Generation
			refCount := entry.RefCount
			pageObjNr := annotationTestPageDictIndRef(t, ctx).ObjectNumber.Value()

			_, err = RemoveAnnotationsFromPageDict(ctx, nil, nil, nil, pageDict, pageObjNr, 1, false)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrPart) {
				t.Fatalf("expected %q error, got %v", tt.wantErrPart, err)
			}
			assertMalformedIndirectAnnotsState(t, ctx, pageDict, *annotsIndRef, generation, refCount, cacheObjNr)
		})
	}
}

func TestSelectiveAnnotationRemovalPreflightPreventsPartialMutation(t *testing.T) {
	ctx := annotationTestContext(t)
	pageDict := annotationTestPageDict(t, ctx)
	pageDict.Delete("Annots")
	ctx.PageAnnots = map[int]model.PgAnnots{}

	firstIndRef, secondIndRef := annotationTestAddPair(t, ctx, pageDict)
	firstXRef := annotationTestXRefSnapshot(t, ctx, firstIndRef)

	delete(ctx.PageAnnots[1][model.AnnLink].Map, secondIndRef.ObjectNumber.Value())
	pageObjNr := annotationTestPageDictIndRef(t, ctx).ObjectNumber.Value()
	_, err := RemoveAnnotationsFromPageDict(ctx, nil, []string{"first", "second"}, nil, pageDict, pageObjNr, 1, false)
	if err == nil || !strings.Contains(err.Error(), "expected one cache entry, got 0") {
		t.Fatalf("expected cache consistency error, got %v", err)
	}

	assertLiveAnnotationXRef(t, ctx, firstIndRef, firstXRef)
	assertTwoPageAnnotations(t, pageDict)
	if _, found := ctx.PageAnnots[1][model.AnnLink].Map[firstIndRef.ObjectNumber.Value()]; !found {
		t.Fatal("expected first annotation cache entry unchanged")
	}
}

func TestSelectiveAnnotationRemovalPreflightDetectsXRefInconsistency(t *testing.T) {
	ctx := annotationTestContext(t)
	pageDict := annotationTestPageDict(t, ctx)
	pageDict.Delete("Annots")
	ctx.PageAnnots = map[int]model.PgAnnots{}

	firstIndRef, secondIndRef := annotationTestAddPair(t, ctx, pageDict)
	firstXRef := annotationTestXRefSnapshot(t, ctx, firstIndRef)
	delete(ctx.Table, secondIndRef.ObjectNumber.Value())

	pageObjNr := annotationTestPageDictIndRef(t, ctx).ObjectNumber.Value()
	_, err := RemoveAnnotationsFromPageDict(ctx, nil, []string{"first", "second"}, nil, pageDict, pageObjNr, 1, false)
	if err == nil || !strings.Contains(err.Error(), "missing xref table entry") {
		t.Fatalf("expected xref consistency error, got %v", err)
	}

	assertLiveAnnotationXRef(t, ctx, firstIndRef, firstXRef)
	assertTwoPageAnnotations(t, pageDict)
	if _, found := ctx.PageAnnots[1][model.AnnLink].Map[firstIndRef.ObjectNumber.Value()]; !found {
		t.Fatal("expected first annotation cache entry unchanged")
	}
}

func TestAnnotationRemovalDeletionGraphFailureDoesNotMutateState(t *testing.T) {
	tests := []struct {
		name   string
		remove func(*model.Context) error
	}{
		{
			name: "selective removal",
			remove: func(ctx *model.Context) error {
				_, err := RemoveAnnotations(ctx, nil, []string{"broken"}, nil, false)
				return err
			},
		},
		{
			name: "remove all",
			remove: func(ctx *model.Context) error {
				_, err := RemoveAnnotations(ctx, nil, nil, nil, false)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := annotationTestContext(t)
			pageDict := annotationTestPageDict(t, ctx)
			indRef := annotationTestAddSingle(t, ctx, pageDict, "broken")
			xref := annotationTestXRefSnapshot(t, ctx, indRef)
			d, err := ctx.DereferenceDict(indRef)
			if err != nil {
				t.Fatal(err)
			}
			d["Broken"] = *types.NewIndirectRef(999, 0)

			err = tt.remove(ctx)
			if err == nil || !strings.Contains(err.Error(), "annotation deletion") {
				t.Fatalf("expected deletion graph error, got %v", err)
			}
			assertSingleAnnotationState(t, ctx, pageDict, indRef, xref, "broken")
		})
	}
}

func TestSelectiveRemovalSecondTargetFailureDoesNotMutateFirst(t *testing.T) {
	ctx := annotationTestContext(t)
	pageDict := annotationTestPageDict(t, ctx)
	pageDict.Delete("Annots")
	ctx.PageAnnots = map[int]model.PgAnnots{}

	firstIndRef, secondIndRef := annotationTestAddPair(t, ctx, pageDict)
	firstXRef := annotationTestXRefSnapshot(t, ctx, firstIndRef)
	secondXRef := annotationTestXRefSnapshot(t, ctx, secondIndRef)
	secondDict, err := ctx.DereferenceDict(secondIndRef)
	if err != nil {
		t.Fatal(err)
	}
	secondDict["Broken"] = *types.NewIndirectRef(999, 0)

	_, err = RemoveAnnotations(ctx, nil, []string{"first", "second"}, nil, false)
	if err == nil || !strings.Contains(err.Error(), "deletion target 2") {
		t.Fatalf("expected second-target deletion error, got %v", err)
	}
	assertLiveAnnotationXRef(t, ctx, firstIndRef, firstXRef)
	assertLiveAnnotationXRef(t, ctx, secondIndRef, secondXRef)
	assertTwoPageAnnotations(t, pageDict)
	for _, indRef := range []types.IndirectRef{firstIndRef, secondIndRef} {
		if _, found := ctx.PageAnnots[1][model.AnnLink].Map[indRef.ObjectNumber.Value()]; !found {
			t.Fatalf("expected annotation obj#%d cached", indRef.ObjectNumber.Value())
		}
	}
}

func TestRemoveAllResolvesCatalogBeforeMutation(t *testing.T) {
	ctx := annotationTestContext(t)
	pageDict := annotationTestPageDict(t, ctx)
	indRef := annotationTestAddSingle(t, ctx, pageDict, "catalog")
	xref := annotationTestXRefSnapshot(t, ctx, indRef)
	ctx.Root = nil
	ctx.RootDict = nil

	_, err := RemoveAnnotations(ctx, nil, nil, nil, false)
	if err == nil || !strings.Contains(err.Error(), "catalog") {
		t.Fatalf("expected catalog error, got %v", err)
	}
	assertSingleAnnotationState(t, ctx, pageDict, indRef, xref, "catalog")
}

func TestRemoveAllPreservesStructTreeWithoutRemoval(t *testing.T) {
	ctx := annotationTestContext(t)
	root, err := ctx.Catalog()
	if err != nil {
		t.Fatal(err)
	}
	root["StructTreeRoot"] = types.Name("keep")
	ctx.PageAnnots = map[int]model.PgAnnots{}

	removed, err := RemoveAnnotations(ctx, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("remove annotations: %v", err)
	}
	if removed {
		t.Fatal("expected no annotations removed")
	}
	if _, found := root.Find("StructTreeRoot"); !found {
		t.Fatal("expected StructTreeRoot preserved")
	}
}

func TestRemoveAnnotationsByTypeWithoutIndRefsDoesNotPanic(t *testing.T) {
	ctx := annotationTestContext(t)
	ann := annotationTestRenderer()
	annotationTestPageDict(t, ctx)["Annots"] = types.Array{}
	ctx.PageAnnots[1] = model.PgAnnots{
		model.AnnText: model.Annot{Map: model.AnnotMap{1: ann}},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected error, got panic: %v", r)
		}
	}()

	_, err := RemoveAnnotations(ctx, nil, []string{"Text"}, nil, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Text annotations: missing indirect references") {
		t.Fatalf("expected missing indirect refs context, got %q", err.Error())
	}
}

func TestRemoveLastDirectAnnotationDeletesAnnotsEntry(t *testing.T) {
	ctx := annotationTestContext(t)
	annotationTestPageDict(t, ctx).Delete("Annots")
	ctx.PageAnnots = map[int]model.PgAnnots{}

	ok, err := AddAnnotations(ctx, types.IntSet{1: true}, annotationTestRenderer(), false)
	if err != nil {
		t.Fatalf("add annotation: %v", err)
	}
	if !ok {
		t.Fatal("expected annotation added")
	}

	ok, err = RemoveAnnotations(ctx, nil, []string{"pdfcpu-test-annotation"}, nil, false)
	if err != nil {
		t.Fatalf("remove annotation: %v", err)
	}
	if !ok {
		t.Fatal("expected annotation removed")
	}

	if _, found := annotationTestPageDict(t, ctx).Find("Annots"); found {
		t.Fatal("expected Annots entry removed")
	}
}
