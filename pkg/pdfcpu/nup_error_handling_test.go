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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func nUpOperationTestContext(t *testing.T, images bool) (*model.Context, *model.NUp, types.Dict, *types.IndirectRef) {
	t.Helper()

	var (
		nup *model.NUp
		err error
	)
	if images {
		nup, err = ImageNUpConfig(4, "", nil)
	} else {
		nup, err = PDFNUpConfig(4, "", nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	pageDim := *types.PaperSize[nup.PageSize]
	nup.PageDim = &pageDim

	ctx, err := CreateContextWithXRefTable(nil, nup.PageDim)
	if err != nil {
		t.Fatal(err)
	}
	pagesIndRef, err := ctx.Pages()
	if err != nil {
		t.Fatal(err)
	}
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, nup, pagesDict, pagesIndRef
}

func corruptNUpFreeList(ctx *model.Context) {
	missingObjectOffset := int64(999)
	ctx.XRefTable.Table[0].Offset = &missingObjectOffset
}

func nUpPDFTestContext(t *testing.T, dim types.Dim, pageCount int) *model.Context {
	t.Helper()

	ctx, err := CreateContextWithXRefTable(nil, &dim)
	if err != nil {
		t.Fatal(err)
	}
	pagesIndRef, err := ctx.Pages()
	if err != nil {
		t.Fatal(err)
	}
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		t.Fatal(err)
	}
	for range pageCount {
		pageIndRef, err := ctx.EmptyPage(pagesIndRef, types.RectForDim(dim.Width, dim.Height), 0)
		if err != nil {
			t.Fatal(err)
		}
		if err := model.AppendPageTree(pageIndRef, 1, pagesDict); err != nil {
			t.Fatal(err)
		}
	}
	ctx.PageCount = pageCount
	return ctx
}

func nUpPageTreeCount(t *testing.T, ctx *model.Context) int {
	t.Helper()

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		t.Fatal(err)
	}
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		t.Fatal(err)
	}
	count := pagesDict.IntEntry("Count")
	if count == nil {
		t.Fatal("missing page tree count")
	}
	return *count
}

func nUpPageTreeDimensions(t *testing.T, ctx *model.Context) types.Dim {
	t.Helper()

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		t.Fatal(err)
	}
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		t.Fatal(err)
	}
	mediaBox, ok := pagesDict["MediaBox"].(types.Array)
	if !ok {
		t.Fatal("missing page tree media box")
	}
	return types.RectForArray(mediaBox).Dimensions()
}

func TestNUpFromPDFReusesConfigurationWithoutLeakingDimensions(t *testing.T) {
	nup, err := PDFNUpConfig(4, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	dims := []types.Dim{
		{Width: 200, Height: 300},
		{Width: 500, Height: 250},
	}
	for i, dim := range dims {
		ctx := nUpPDFTestContext(t, dim, 1)
		if err := NUpFromPDF(ctx, types.IntSet{1: true}, nup); err != nil {
			t.Fatalf("document %d: %v", i+1, err)
		}
		if nup.PageDim != nil {
			t.Fatalf("document %d: caller page dimensions changed to %v", i+1, nup.PageDim)
		}
		if got := nUpPageTreeDimensions(t, ctx); got != dim {
			t.Fatalf("document %d: got output dimensions %v, want %v", i+1, got, dim)
		}
	}
}

func TestNUpFromPDFFailureDoesNotMutateConfiguration(t *testing.T) {
	nup, err := PDFNUpConfig(4, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := nUpPDFTestContext(t, types.Dim{Width: 200, Height: 300}, 1)
	err = NUpFromPDF(ctx, types.IntSet{2: true}, nup)
	if !errors.Is(err, model.ErrPageNotFound) {
		t.Fatalf("expected %v, got %v", model.ErrPageNotFound, err)
	}
	if nup.PageDim != nil {
		t.Fatalf("caller page dimensions changed to %v", nup.PageDim)
	}
	if ctx.PageCount != 1 || nUpPageTreeCount(t, ctx) != 1 {
		t.Fatalf("original page tree lost authority: context=%d tree=%d", ctx.PageCount, nUpPageTreeCount(t, ctx))
	}
}

func TestNUpFromPDFPreservesExplicitPageDimensions(t *testing.T) {
	nup, err := PDFNUpConfig(4, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	pageDim := types.Dim{Width: 612, Height: 792}
	nup.PageDim = &pageDim
	wantPageDim := nup.PageDim
	want := *nup.PageDim

	for _, tt := range []struct {
		name          string
		selectedPages types.IntSet
		wantErr       bool
	}{
		{name: "success", selectedPages: types.IntSet{1: true}},
		{name: "failure", selectedPages: types.IntSet{2: true}, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := nUpPDFTestContext(t, types.Dim{Width: 200, Height: 300}, 1)
			err := NUpFromPDF(ctx, tt.selectedPages, nup)
			if tt.wantErr && !errors.Is(err, model.ErrPageNotFound) {
				t.Fatalf("expected %v, got %v", model.ErrPageNotFound, err)
			}
			if !tt.wantErr && err != nil {
				t.Fatal(err)
			}
			if nup.PageDim != wantPageDim || *nup.PageDim != want {
				t.Fatalf("caller page dimensions changed: got %v, want %v", nup.PageDim, want)
			}
		})
	}
}

func TestNUpOperationsSynchronizeContextAndPageTreeCounts(t *testing.T) {
	imageFile := filepath.Join("..", "samples", "images", "any.jpg")
	tests := []struct {
		name      string
		wantCount int
		run       func(t *testing.T) *model.Context
	}{
		{name: "one image", wantCount: 1, run: func(t *testing.T) *model.Context {
			ctx, nup, pagesDict, pagesIndRef := nUpOperationTestContext(t, true)
			if err := NUpFromOneImage(ctx, imageFile, nup, pagesDict, pagesIndRef); err != nil {
				t.Fatal(err)
			}
			return ctx
		}},
		{name: "multiple images", wantCount: 1, run: func(t *testing.T) *model.Context {
			ctx, nup, pagesDict, pagesIndRef := nUpOperationTestContext(t, true)
			if err := NUpFromMultipleImages(ctx, []string{imageFile, imageFile}, nup, pagesDict, pagesIndRef); err != nil {
				t.Fatal(err)
			}
			return ctx
		}},
		{name: "PDF", wantCount: 2, run: func(t *testing.T) *model.Context {
			ctx := nUpPDFTestContext(t, types.Dim{Width: 200, Height: 300}, 5)
			nup, err := PDFNUpConfig(4, "", nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := NUpFromPDF(ctx, types.IntSet{1: true, 2: true, 3: true, 4: true, 5: true}, nup); err != nil {
				t.Fatal(err)
			}
			return ctx
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.run(t)
			if ctx.PageCount != tt.wantCount {
				t.Fatalf("got context page count %d, want %d", ctx.PageCount, tt.wantCount)
			}
			if treeCount := nUpPageTreeCount(t, ctx); treeCount != ctx.PageCount {
				t.Fatalf("page tree count %d != context page count %d", treeCount, ctx.PageCount)
			}
		})
	}
}

func TestNUpFromMultipleImagesFailureLeavesPageCountUncommitted(t *testing.T) {
	ctx, nup, pagesDict, pagesIndRef := nUpOperationTestContext(t, true)
	imageFile := filepath.Join("..", "samples", "images", "any.jpg")
	missingImage := filepath.Join(t.TempDir(), "missing.jpg")
	fileNames := []string{imageFile, imageFile, imageFile, imageFile, missingImage}

	err := NUpFromMultipleImages(ctx, fileNames, nup, pagesDict, pagesIndRef)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if ctx.PageCount != 0 {
		t.Fatalf("partial operation committed context page count %d", ctx.PageCount)
	}
	if treeCount := nUpPageTreeCount(t, ctx); treeCount != 1 {
		t.Fatalf("expected one partially appended output page, got %d", treeCount)
	}
}

func TestNUpImageOpenErrorIncludesSourceContext(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "missing.png")
	for _, tt := range []struct {
		name string
		run  func(*model.Context, *model.NUp, types.Dict, *types.IndirectRef) error
	}{
		{name: "one", run: func(ctx *model.Context, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
			return NUpFromOneImage(ctx, fileName, nup, pagesDict, pagesIndRef)
		}},
		{name: "multiple", run: func(ctx *model.Context, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
			return NUpFromMultipleImages(ctx, []string{fileName, fileName}, nup, pagesDict, pagesIndRef)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, nup, pagesDict, pagesIndRef := nUpOperationTestContext(t, true)
			err := tt.run(ctx, nup, pagesDict, pagesIndRef)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			want := fmt.Sprintf("n-up image 1 %q: open", fileName)
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %q", want, err)
			}
		})
	}
}

func TestGridImageOpenErrorIncludesSourceContext(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "missing.png")
	for _, tt := range []struct {
		name string
		run  func(*model.Context, *model.NUp, types.Dict, *types.IndirectRef) error
	}{
		{name: "one", run: func(ctx *model.Context, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
			return GridFromOneImage(ctx, fileName, nup, pagesDict, pagesIndRef)
		}},
		{name: "multiple", run: func(ctx *model.Context, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
			return GridFromMultipleImages(ctx, []string{fileName, fileName}, nup, pagesDict, pagesIndRef)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, nup, pagesDict, pagesIndRef := nUpOperationTestContext(t, true)
			nup.PageGrid = true
			err := tt.run(ctx, nup, pagesDict, pagesIndRef)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			want := fmt.Sprintf("grid image 1 %q: open", fileName)
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %q", want, err)
			}
			if strings.Contains(err.Error(), "n-up image") {
				t.Fatalf("unexpected n-up context: %q", err)
			}
		})
	}
}

func TestNUpImageDecodeErrorIncludesSourceContext(t *testing.T) {
	ctx, nup, pagesDict, pagesIndRef := nUpOperationTestContext(t, true)
	fileName := filepath.Join(t.TempDir(), "broken.png")
	if err := os.WriteFile(fileName, []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	err := NUpFromOneImage(ctx, fileName, nup, pagesDict, pagesIndRef)
	if err == nil {
		t.Fatal("expected error")
	}
	want := fmt.Sprintf("n-up image 1 %q: create resource", fileName)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
	if err := os.Rename(fileName, fileName+".moved"); err != nil {
		t.Fatalf("expected closed image input: %v", err)
	}
}

func TestLoadNUpImageResourceClosesFileOnPanic(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(fileName, []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}
	wantPanic := "create image resource panic"
	var imageFile *os.File
	createImageResource := func(_ *model.XRefTable, r io.Reader) (*types.IndirectRef, int, int, error) {
		imageFile = r.(*os.File)
		panic(wantPanic)
	}

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_, _, _, _ = loadNUpImageResourceWith(nil, 3, fileName, createImageResource)
	}()
	if recovered != wantPanic {
		t.Fatalf("expected panic %q, got %v", wantPanic, recovered)
	}
	if imageFile == nil {
		t.Fatal("expected captured image file")
	}
	if _, err := imageFile.Read(make([]byte, 1)); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected panic-safe image close, got %v", err)
	}
}

func TestLoadNUpImageResourcePreservesResourceAndCloseErrors(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(fileName, []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("create resource failed")
	createImageResource := func(_ *model.XRefTable, r io.Reader) (*types.IndirectRef, int, int, error) {
		if err := r.(*os.File).Close(); err != nil {
			t.Fatal(err)
		}
		return nil, 0, 0, wantErr
	}

	_, _, _, err := loadNUpImageResourceWith(nil, 4, fileName, createImageResource)
	if !errors.Is(err, wantErr) || !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected resource and close errors, got %v", err)
	}
	resourceContext := fmt.Sprintf("n-up image 4 %q: create resource", fileName)
	closeContext := fmt.Sprintf("n-up image 4 %q: close", fileName)
	if !strings.Contains(err.Error(), resourceContext) || !strings.Contains(err.Error(), closeContext) {
		t.Fatalf("expected %q and %q, got %q", resourceContext, closeContext, err)
	}
}

func TestNUpFromMultipleImagesRetryDoesNotMutatePageDimensions(t *testing.T) {
	ctx, nup, pagesDict, pagesIndRef := nUpOperationTestContext(t, true)
	nup.PageGrid = true
	want := *nup.PageDim
	fileName := filepath.Join(t.TempDir(), "missing.png")

	for attempt := 1; attempt <= 2; attempt++ {
		err := NUpFromMultipleImages(ctx, []string{fileName, fileName}, nup, pagesDict, pagesIndRef)
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("attempt %d: expected %v, got %v", attempt, os.ErrNotExist, err)
		}
		if got := *nup.PageDim; got != want {
			t.Fatalf("attempt %d: page dimensions changed: got %v, want %v", attempt, got, want)
		}
	}
}

func TestNUpPagesTileErrorIncludesSourcePage(t *testing.T) {
	ctx, nup, _, _ := nUpOperationTestContext(t, false)
	err := NUpFromPDF(ctx, types.IntSet{1: true}, nup)
	if !errors.Is(err, model.ErrPageNotFound) {
		t.Fatalf("expected %v, got %v", model.ErrPageNotFound, err)
	}
	if !strings.Contains(err.Error(), "n-up page imposition: n-up source page 1: resolve page dictionary") {
		t.Fatalf("expected source page context, got %q", err.Error())
	}
}

func TestNUpPagesWrapErrorIncludesOutputPage(t *testing.T) {
	ctx, nup, pagesDict, pagesIndRef := nUpOperationTestContext(t, false)
	corruptNUpFreeList(ctx)
	_, err := impositionPages("n-up", ctx, nil, nup, pagesDict, pagesIndRef)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "n-up output page 1: wrap page") {
		t.Fatalf("expected output page context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "n-up output page: store resource dictionary") {
		t.Fatalf("expected resource context, got %q", err.Error())
	}
}

func TestNUpFromPDFDimensionErrorPreservesPageNotFound(t *testing.T) {
	ctx, nup, _, _ := nUpOperationTestContext(t, false)
	nup.PageDim = nil
	err := NUpFromPDF(ctx, nil, nup)
	if !errors.Is(err, model.ErrPageNotFound) {
		t.Fatalf("expected %v, got %v", model.ErrPageNotFound, err)
	}
	if !strings.Contains(err.Error(), "n-up page tree: derive dimensions from source page 1") {
		t.Fatalf("expected dimension context, got %q", err.Error())
	}
}

func TestNUpFromPDFRootErrorIncludesPageTreeContext(t *testing.T) {
	ctx, nup, _, _ := nUpOperationTestContext(t, false)
	corruptNUpFreeList(ctx)
	err := NUpFromPDF(ctx, nil, nup)
	if err == nil || !strings.Contains(err.Error(), "n-up page tree: create root") {
		t.Fatalf("expected page tree root context, got %v", err)
	}
}

func TestNUpFromPDFCatalogErrorIncludesPageTreeContext(t *testing.T) {
	ctx, nup, _, _ := nUpOperationTestContext(t, false)
	ctx.Root = nil
	ctx.RootDict = nil
	err := NUpFromPDF(ctx, nil, nup)
	if err == nil || !strings.Contains(err.Error(), "n-up page tree: access catalog") {
		t.Fatalf("expected catalog context, got %v", err)
	}
}
