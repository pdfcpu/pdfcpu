/*
Copyright 2021 The pdfcpu Authors.

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

package test

import (
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

var textAnn pdf.AnnotationRenderer = pdf.NewTextAnnotation(
	*pdf.Rect(0, 0, 100, 100),
	"Test Content",
	"ID1",
	"Title1",
	pdf.AnnNoZoom+pdf.AnnNoRotate,
	&pdf.Gray,
	nil,
	"",
	"",
	false,
	"Comment")

var linkAnn pdf.AnnotationRenderer = pdf.NewLinkAnnotation(
	*pdf.Rect(0, 0, 100, 100),
	nil,
	"https://pdfcpu.io",
	"ID2",
	pdf.AnnNoZoom+pdf.AnnNoRotate,
	nil)

func TestAddRemoveAnnotationsById(t *testing.T) {
	msg := "TestAddRemoveAnnotationsById"

	fn := "test.pdf"
	copyFile(t, filepath.Join(inDir, fn), filepath.Join(outDir, fn))
	inFile := filepath.Join(outDir, fn)

	// We start with 0 annotations.
	i, _, err := api.ListAnnotationsFile(inFile, nil, nil)
	if err != nil || i > 0 {
		t.Fatalf("%s list: %v\n", msg, err)
	}

	// Add a text annotation to page 1.
	err = api.AddAnnotationsFile(inFile, "", []string{"1"}, textAnn, nil, false)
	if err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Add a link annotation to page 1.
	err = api.AddAnnotationsFile(inFile, "", []string{"1"}, linkAnn, nil, false)
	if err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Now we should have 2 annotations.
	i, _, err = api.ListAnnotationsFile(inFile, nil, nil)
	if err != nil || i != 2 {
		t.Fatalf("%s list: %v\n", msg, err)
	}

	// Remove both annotations by id.
	err = api.RemoveAnnotationsFile(inFile, "", nil, []string{"ID1", "ID2"}, nil, nil, false)
	if err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// We should have 0 annotations as at the beginning.
	i, _, err = api.ListAnnotationsFile(inFile, nil, nil)
	if err != nil || i > 0 {
		t.Fatalf("%s list: %v\n", msg, err)
	}
}

func TestAddRemoveAnnotationsByObjNr(t *testing.T) {
	msg := "TestAddRemoveAnnotationsByObjNr"

	fn := "test.pdf"
	copyFile(t, filepath.Join(inDir, fn), filepath.Join(outDir, fn))
	inFile := filepath.Join(outDir, fn)

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	allPages, err := api.PagesForPageSelection(ctx.PageCount, nil, true)
	if err != nil {
		t.Fatalf("%s pagesForPageSelection: %v\n", msg, err)
	}

	// Add link annotation to all pages.
	ok, err := ctx.AddAnnotations(allPages, linkAnn, false)
	if err != nil || !ok {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Write context to file.
	err = api.WriteContextFile(ctx, inFile)
	if err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}

	// We should have 1 annotation
	i, _, err := api.ListAnnotationsFile(inFile, nil, nil)
	if err != nil || i != 1 {
		t.Fatalf("%s list: %v\n", msg, err)
	}

	// Create a context.
	ctx, err = api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Identify object numbers for located annotations
	objNrs, err := ctx.AnnotationObjNrs()
	if err != nil {
		t.Fatalf("%s annObjNrs: %v\n", msg, err)
	}
	if len(objNrs) != 1 {
		t.Fatalf("%s want 1 annotation, got: %d\n", msg, len(objNrs))
	}

	// Remove annotations by their object numbers
	// We could also do: api.RemoveAnnotationsFile
	// but since we already have the ctx this is more straight forward.
	_, err = ctx.RemoveAnnotations(allPages, nil, objNrs, false)
	if err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// Write context to file.
	err = api.WriteContextFile(ctx, inFile)
	if err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}

	// We should have 0 annotations like at the beginning.
	i, _, err = api.ListAnnotationsFile(inFile, nil, nil)
	if err != nil || i > 0 {
		t.Fatalf("%s list: %v\n", msg, err)
	}

}

func TestRemoveAllAnnotations(t *testing.T) {
	msg := "TestRemoveAllAnnotations"

	fn := "test.pdf"
	copyFile(t, filepath.Join(inDir, fn), filepath.Join(outDir, fn))
	inFile := filepath.Join(outDir, fn)

	m := map[int][]pdf.AnnotationRenderer{}
	anns := make([]pdf.AnnotationRenderer, 2)
	anns[0] = textAnn
	anns[1] = linkAnn
	m[1] = anns

	err := api.AddAnnotationsMapFile(inFile, "", m, nil, false)
	if err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// We should have 2 annotations.
	i, _, err := api.ListAnnotationsFile(inFile, nil, nil)
	if err != nil || i != 2 {
		t.Fatalf("%s list: %v\n", msg, err)
	}

	// Remove all annotations.
	err = api.RemoveAnnotationsFile(inFile, "", nil, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// We should have 0 annotations like at the beginning.
	i, _, err = api.ListAnnotationsFile(inFile, nil, nil)
	if err != nil || i > 0 {
		t.Fatalf("%s list: %v\n", msg, err)
	}
}

func TestAddAnnotationsLowLevel(t *testing.T) {
	msg := "TestAddAnnotationsLowLevel"

	fn := "test.pdf"
	inFile := filepath.Join(inDir, fn)
	outFile := filepath.Join(outDir, fn)

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	m := map[int][]pdf.AnnotationRenderer{}
	anns := make([]pdf.AnnotationRenderer, 2)
	anns[0] = textAnn
	anns[1] = linkAnn
	m[1] = anns

	// Add 2 annotations to page 1.
	if ok, err := ctx.AddAnnotationsMap(m, false); err != nil || !ok {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Write context to file.
	if err := api.WriteContextFile(ctx, outFile); err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}

	// Create a context.
	ctx, err = api.ReadContextFile(outFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// We should have 2 annotations.
	i, _, err := ctx.ListAnnotations(nil)
	if err != nil || i != 2 {
		t.Fatalf("%s list: %v\n", msg, err)
	}

	// Remove all annotations.
	_, err = ctx.RemoveAnnotations(nil, nil, nil, false)
	if err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// (before writing) We should have 0 annotations like at the beginning.
	i, _, err = ctx.ListAnnotations(nil)
	if err != nil || i != 0 {
		t.Fatalf("%s list: %v\n", msg, err)
	}

	// Write context to file.
	if err := api.WriteContextFile(ctx, outFile); err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}

	// (after writing) We should have 0 annotations like at the beginning.
	i, _, err = api.ListAnnotationsFile(outFile, nil, nil)
	if err != nil || i > 0 {
		t.Fatalf("%s list: %v\n", msg, err)
	}
}

func TestAddRemoveAnnotationsAsIncrements(t *testing.T) {
	msg := "TestAddRemoveAnnotationsAsIncrements"

	fn := "test.pdf"
	copyFile(t, filepath.Join(inDir, fn), filepath.Join(outDir, fn))
	inFile := filepath.Join(outDir, fn)

	// We start with 0 annotations.
	i, _, err := api.ListAnnotationsFile(inFile, nil, nil)
	if err != nil || i > 0 {
		t.Fatalf("%s list: %v\n", msg, err)
	}

	increment := true

	// Add a text annotation to page 1 and append as PDF increment to inFile.
	err = api.AddAnnotationsFile(inFile, "", []string{"1"}, textAnn, nil, increment)
	if err != nil {
		t.Fatalf("%s add 1st: %v\n", msg, err)
	}

	// Add a link annotation to page 1 and append as PDF increment to inFile.
	err = api.AddAnnotationsFile(inFile, "", []string{"1"}, linkAnn, nil, increment)
	if err != nil {
		t.Fatalf("%s add 2nd: %v\n", msg, err)
	}

	// Now we should have 2 annotations.
	i, _, err = api.ListAnnotationsFile(inFile, nil, nil)
	if err != nil || i != 2 {
		t.Fatalf("%s list: %v\n", msg, err)
	}

	// Remove all page annotations and append the result as PDF increment to inFile.
	err = api.RemoveAnnotationsFile(inFile, "", nil, nil, nil, nil, true)
	if err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// We should have 0 annotations like at the beginning.
	i, _, err = api.ListAnnotationsFile(inFile, nil, nil)
	if err != nil || i > 0 {
		t.Fatalf("%s list: %v\n", msg, err)
	}
}
