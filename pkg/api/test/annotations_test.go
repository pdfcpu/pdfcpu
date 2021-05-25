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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func TestAddAnnotations(t *testing.T) {
	msg := "TestAddAnnotations"

	ann1 := pdfcpu.NewTextAnnotation(
		*pdfcpu.Rect(0, 0, 100, 100),
		"Test Content", // Contents
		"ID1",          // NM // entry=NM: unsupported in version 1.2 !!! when incremented !!!
		"Title1",       // T
		152,            // F
		&pdfcpu.Gray,   // backgrCol
		nil,            // CA
		"",             // RC
		"",             // Subject
		false,          // Open
		"Comment",      // Name
	)

	ann2 := pdfcpu.NewLinkAnnotation(
		*pdfcpu.Rect(0, 0, 100, 100),
		"https://pdfcpu.io",
		"ID2",
		0,
		nil,
	)

	file := "test.pdf"

	increment := false
	copyFile(t, filepath.Join(inDir, file), filepath.Join(outDir, file))
	inFile := filepath.Join(outDir, file)
	if err := api.AddAnnotationsFile(inFile, "", []string{"1"}, ann1, nil, increment); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := api.AddAnnotationsFile(inFile, "", []string{"1"}, ann2, nil, increment); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	increment = true
	copyFile(t, filepath.Join(inDir, file), filepath.Join(outDir, file))
	inFile = filepath.Join(outDir, file)
	if err := api.AddAnnotationsFile(inFile, "", []string{"1"}, ann1, nil, increment); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := api.AddAnnotationsFile(inFile, "", []string{"1"}, ann2, nil, increment); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestAddAndRemoveAnnotations(t *testing.T) {
	msg := "TestAddAndRemoveAnnotations"

	// Add text annotation on page 1.
	ann1 := pdfcpu.NewTextAnnotation(
		*pdfcpu.Rect(0, 0, 100, 100),
		"Test Content", // Contents
		"ID1",          // NM // entry=NM: unsupported in version 1.2 !!! when incremented !!!
		"Title1",       // T
		152,            // F
		&pdfcpu.Gray,   // backgrCol
		nil,            // CA
		"",             // RC
		"",             // Subject
		false,          // Open
		"Comment",      // Name
	)

	ann2 := pdfcpu.NewLinkAnnotation(
		*pdfcpu.Rect(0, 0, 100, 100),
		"https://pdfcpu.io",
		"ID2",
		0,
		nil,
	)

	file := "test.pdf"

	increment := false
	copyFile(t, filepath.Join(inDir, file), filepath.Join(outDir, file))
	inFile := filepath.Join(outDir, file)
	if err := api.AddAnnotationsFile(inFile, "", []string{"1"}, ann1, nil, increment); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := api.AddAnnotationsFile(inFile, "", []string{"1"}, ann2, nil, increment); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := api.RemoveAnnotationsFile(inFile, inFile, []string{"1"}, ann2.ID(), nil, increment); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	increment = true
	copyFile(t, filepath.Join(inDir, file), filepath.Join(outDir, file))
	inFile = filepath.Join(outDir, file)
	if err := api.AddAnnotationsFile(inFile, "", []string{"1"}, ann1, nil, increment); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := api.AddAnnotationsFile(inFile, "", []string{"1"}, ann2, nil, increment); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := api.RemoveAnnotationsFile(inFile, inFile, []string{"1"}, ann2.ID(), nil, increment); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestAddAnnotationsLowLevel(t *testing.T) {
	msg := "TestAddAnnotationsLowLevel"

	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(outDir, "test.pdf")
	increment := false

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	if err := ctx.EnsurePageCount(); err != nil {
		t.Fatalf("%s ensurePageCount: %v\n", msg, err)
	}

	pages, err := api.PagesForPageSelection(ctx.PageCount, []string{"1"}, true)
	if err != nil {
		t.Fatalf("%s pagesForPageSelection: %v\n", msg, err)
	}

	ann := pdfcpu.AnnotationRenderer(nil)

	// Add text annotation on page 1.
	ann = pdfcpu.NewTextAnnotation(
		*pdfcpu.Rect(0, 0, 100, 100),
		"Test Content", // Contents
		"ID1",          // NM // entry=NM: unsupported in versions < 1.4
		"Title1",       // T
		152,            // F
		&pdfcpu.Gray,   // backgrCol
		nil,            // CA
		"",             // RC
		"",             // Subject
		false,          // Open
		"Comment",      // Name
	)

	if err = ctx.AddAnnotations(pages, ann, increment); err != nil {
		t.Fatalf("%s addAnnotations: %v\n", msg, err)
	}

	// Add link annotation on page 1.
	ann = pdfcpu.NewLinkAnnotation(
		*pdfcpu.Rect(0, 0, 100, 100),
		"https://pdfcpu.io",
		"ID2",
		0,
		nil,
	)

	if err = ctx.AddAnnotations(pages, ann, increment); err != nil {
		t.Fatalf("%s addAnnotations: %v\n", msg, err)
	}

	// Write context to file.
	if err := api.WriteContextFile(ctx, outFile); err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}
}
