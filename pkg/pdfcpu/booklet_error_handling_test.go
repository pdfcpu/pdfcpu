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

func bookletOperationTestContext(t *testing.T, images bool) (*model.Context, *model.NUp, types.Dict, *types.IndirectRef) {
	t.Helper()

	var (
		nup *model.NUp
		err error
	)
	if images {
		nup, err = ImageBookletConfig(2, "", nil)
	} else {
		nup, err = PDFBookletConfig(2, "", nil)
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

func corruptBookletFreeList(ctx *model.Context) {
	missingObjectOffset := int64(999)
	ctx.XRefTable.Table[0].Offset = &missingObjectOffset
}

func TestBookletFromImagesOpenErrorIncludesSourceContext(t *testing.T) {
	ctx, nup, pagesDict, pagesIndRef := bookletOperationTestContext(t, true)
	fileName := filepath.Join(t.TempDir(), "missing.png")
	err := BookletFromImages(ctx, []string{fileName}, nup, pagesDict, pagesIndRef)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	want := fmt.Sprintf("booklet image 1 %q: open", fileName)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

func TestBookletFromImagesRetryDoesNotMutatePageDimensions(t *testing.T) {
	ctx, nup, pagesDict, pagesIndRef := bookletOperationTestContext(t, true)
	nup.PageGrid = true
	want := *nup.PageDim
	fileName := filepath.Join(t.TempDir(), "missing.png")

	for attempt := 1; attempt <= 2; attempt++ {
		err := BookletFromImages(ctx, []string{fileName}, nup, pagesDict, pagesIndRef)
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("attempt %d: expected %v, got %v", attempt, os.ErrNotExist, err)
		}
		if got := *nup.PageDim; got != want {
			t.Fatalf("attempt %d: page dimensions changed: got %v, want %v", attempt, got, want)
		}
	}
}

func TestBookletFromImagesDecodeErrorIncludesSourceContext(t *testing.T) {
	ctx, nup, pagesDict, pagesIndRef := bookletOperationTestContext(t, true)
	fileName := filepath.Join(t.TempDir(), "broken.png")
	if err := os.WriteFile(fileName, []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	err := BookletFromImages(ctx, []string{fileName}, nup, pagesDict, pagesIndRef)
	if err == nil {
		t.Fatal("expected error")
	}
	want := fmt.Sprintf("booklet image 1 %q: create resource", fileName)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
	if err := os.Rename(fileName, fileName+".moved"); err != nil {
		t.Fatalf("expected closed image input: %v", err)
	}
}

func TestLoadBookletImageResourceClosesFileOnPanic(t *testing.T) {
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
		_, _, _, _ = loadBookletImageResourceWith(nil, 3, fileName, createImageResource)
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

func TestLoadBookletImageResourcePreservesResourceAndCloseErrors(t *testing.T) {
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

	_, _, _, err := loadBookletImageResourceWith(nil, 4, fileName, createImageResource)
	if !errors.Is(err, wantErr) || !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected resource and close errors, got %v", err)
	}
	resourceContext := fmt.Sprintf("booklet image 4 %q: create resource", fileName)
	closeContext := fmt.Sprintf("booklet image 4 %q: close", fileName)
	if !strings.Contains(err.Error(), resourceContext) || !strings.Contains(err.Error(), closeContext) {
		t.Fatalf("expected %q and %q, got %q", resourceContext, closeContext, err)
	}
}

func TestBookletPagesTileErrorIncludesSourcePage(t *testing.T) {
	ctx, nup, _, _ := bookletOperationTestContext(t, false)
	err := BookletFromPDF(ctx, types.IntSet{1: true}, nup)
	if !errors.Is(err, model.ErrPageNotFound) {
		t.Fatalf("expected %v, got %v", model.ErrPageNotFound, err)
	}
	if !strings.Contains(err.Error(), "booklet page imposition: n-up source page 1: resolve page dictionary") {
		t.Fatalf("expected source page context, got %q", err.Error())
	}
}

func TestBookletPagesWrapErrorIncludesOutputPage(t *testing.T) {
	ctx, nup, pagesDict, pagesIndRef := bookletOperationTestContext(t, false)
	corruptBookletFreeList(ctx)
	_, err := bookletPages(ctx, nil, nup, pagesDict, pagesIndRef)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "booklet output page 1: wrap page") {
		t.Fatalf("expected output page context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "n-up output page: store resource dictionary") {
		t.Fatalf("expected n-up resource context, got %q", err.Error())
	}
}

func TestCreateNUpFormForImageErrorIncludesFormContext(t *testing.T) {
	ctx, _, _, _ := bookletOperationTestContext(t, true)
	corruptBookletFreeList(ctx)
	_, err := createNUpFormForImage(ctx.XRefTable, types.NewIndirectRef(10, 0), 10, 10, 3)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "n-up image form 4: store resource dictionary") {
		t.Fatalf("expected image form context, got %q", err.Error())
	}
}

func TestBookletFromPDFRootErrorIncludesPageTreeContext(t *testing.T) {
	ctx, nup, _, _ := bookletOperationTestContext(t, false)
	corruptBookletFreeList(ctx)
	err := BookletFromPDF(ctx, nil, nup)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "booklet page tree: create root") {
		t.Fatalf("expected page tree context, got %q", err.Error())
	}
}

func TestBookletFromPDFCatalogErrorIncludesPageTreeContext(t *testing.T) {
	ctx, nup, _, _ := bookletOperationTestContext(t, false)
	ctx.Root = nil
	ctx.RootDict = nil
	err := BookletFromPDF(ctx, nil, nup)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "booklet page tree: access catalog") {
		t.Fatalf("expected catalog context, got %q", err.Error())
	}
}
