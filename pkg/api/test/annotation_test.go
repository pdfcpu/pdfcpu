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
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var textAnn model.AnnotationRenderer = model.NewTextAnnotation(
	*types.NewRectangle(0, 0, 100, 100), // rect
	"Text Annotation",                   // contents
	"ID1",                               // id
	"",                                  // modDate
	0,                                   // f
	&color.Gray,                         // col
	"Title1",                            // title
	nil,                                 // popupIndRef
	nil,                                 // ca
	"",                                  // rc
	"",                                  // subject
	false,                               // displayOpen
	"Comment")                           // name

var textAnnCJK model.AnnotationRenderer = model.NewTextAnnotation(
	*types.NewRectangle(0, 100, 100, 200), // rect
	"文字注释",                                // contents
	"ID1CJK",                              // id
	"",                                    // modDate
	0,                                     // f
	&color.Gray,                           // col
	"标题1",                                 // title
	nil,                                   // popupIndRef
	nil,                                   // ca
	"RC",                                  // rc
	"",                                    // subject
	true,                                  // displayOpen
	"Comment")                             // name

var linkAnn model.AnnotationRenderer = model.NewLinkAnnotation(
	*types.NewRectangle(200, 0, 300, 100), // rect
	"",                                    // contents
	"ID2",                                 // id
	"",                                    // modDate
	0,                                     // f
	&color.Red,                            // borderCol
	nil,                                   // dest
	"https://pdfcpu.io",                   // uri
	nil,                                   // quad
	true,                                  // border
	1,                                     // borderWidth
	model.BSSolid,                         // borderStyle
)

var squareAnn model.AnnotationRenderer = model.NewSquareAnnotation(
	*types.NewRectangle(300, 0, 350, 50), // rect
	"Square Annotation",                  // contents
	"ID3",                                // id
	"",                                   // modDate
	0,                                    // f
	&color.Gray,                          // col
	"Title1",                             // title
	nil,                                  // popupIndRef
	nil,                                  // ca
	"",                                   // rc
	"",                                   // subject
	&color.Blue,                          // fillCol
	0,                                    // MLeft
	0,                                    // MTop
	0,                                    // MRight
	0,                                    // MBot
	1,                                    // borderWidth
	model.BSSolid,                        // borderStyle
	false,                                // cloudyBorder
	0,                                    // cloudyBorderIntensity
)

var squareAnnCJK model.AnnotationRenderer = model.NewSquareAnnotation(
	*types.NewRectangle(300, 50, 350, 100), // rect
	"方形注释",                                 // contents
	"ID3CJK",                               // id
	"",                                     // modDate
	0,                                      // f
	&color.Gray,                            // col
	"Title1",                               // title
	nil,                                    // popupIndRef
	nil,                                    // ca
	"",                                     // rc
	"",                                     // subject
	&color.Green,                           // fillCol
	0,                                      // MLeft
	0,                                      // MTop
	0,                                      // MRight
	0,                                      // MBot
	1,                                      // borderWidth
	model.BSDashed,                         // borderStyle
	false,                                  // cloudyBorder
	0,                                      // cloudyBorderIntensity
)

var circleAnn model.AnnotationRenderer = model.NewCircleAnnotation(
	*types.NewRectangle(400, 0, 450, 50), // rect
	"Circle Annotation",                  // contents
	"ID4",                                // id
	"",                                   // modDate
	0,                                    // f
	&color.Gray,                          // col
	"Title1",                             // title
	nil,                                  // popupIndRef
	nil,                                  // ca
	"",                                   // rc
	"",                                   // subject
	&color.Blue,                          // fillCol
	0,                                    // MLeft
	0,                                    // MTop
	0,                                    // MRight
	0,                                    // MBot
	1,                                    // borderWidth
	model.BSSolid,                        // borderStyle
	false,                                // cloudyBorder
	0,                                    // cloudyBorderIntensity
)

var circleAnnCJK model.AnnotationRenderer = model.NewCircleAnnotation(
	*types.NewRectangle(400, 50, 450, 100), // rect
	"圆圈注释",                                 // contents
	"ID4CJK",                               // id
	"",                                     // modDate
	0,                                      // f
	&color.Green,                           // col
	"Title1",                               // title
	nil,                                    // popupIndRef
	nil,                                    // ca
	"",                                     // rc
	"",                                     // subject
	&color.Blue,                            // fillCol
	10,                                     // MLeft
	10,                                     // MTop
	10,                                     // MRight
	10,                                     // MBot
	1,                                      // borderWidth
	model.BSBeveled,                        // borderStyle
	false,                                  // cloudyBorder
	0,                                      // cloudyBorderIntensity
)

func annotationCount(t *testing.T, inFile string) int {
	t.Helper()

	msg := "annotationCount"

	f, err := os.Open(inFile)
	if err != nil {
		t.Fatalf("%s open: %v\n", msg, err)
	}
	defer f.Close()

	annots, err := api.Annotations(f, nil, conf)
	if err != nil {
		t.Fatalf("%s annotations: %v\n", msg, err)
	}

	count, _, err := pdfcpu.ListAnnotations(annots)
	if err != nil {
		t.Fatalf("%s listAnnotations: %v\n", msg, err)
	}

	return count
}

func add2Annotations(t *testing.T, msg, inFile string, incr bool) {
	t.Helper()

	// We start with 0 annotations.
	if i := annotationCount(t, inFile); i > 0 {
		t.Fatalf("%s count: got %d want 0\n", msg, i)
	}

	// Add a text annotation to page 1.
	if err := api.AddAnnotationsFile(inFile, "", []string{"1"}, textAnn, nil, incr); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Add a link annotation to page 1.
	if err := api.AddAnnotationsFile(inFile, "", []string{"1"}, linkAnn, nil, incr); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Now we should have 2 annotations.
	if i := annotationCount(t, inFile); i != 2 {
		t.Fatalf("%s count: got %d want 2\n", msg, i)
	}
}

func TestAddRemoveAnnotationsByAnnotType(t *testing.T) {
	msg := "TestAddRemoveAnnotationsByAnnotType"

	incr := false // incremental updates

	fn := "test.pdf"
	copyFile(t, filepath.Join(inDir, fn), filepath.Join(outDir, fn))
	inFile := filepath.Join(outDir, fn)

	add2Annotations(t, msg, inFile, incr)

	// Remove annotations by annotation type.
	if err := api.RemoveAnnotationsFile(inFile, "", nil, []string{"Link", "Text"}, nil, nil, false); err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// We should have 0 annotations as at the beginning.
	if i := annotationCount(t, inFile); i > 0 {
		t.Fatalf("%s count: got %d want 0\n", msg, i)
	}
}

func TestAddRemoveAnnotationsById(t *testing.T) {
	msg := "TestAddRemoveAnnotationsById"

	incr := false // incremental updates

	fn := "test.pdf"
	copyFile(t, filepath.Join(inDir, fn), filepath.Join(outDir, fn))
	inFile := filepath.Join(outDir, fn)

	add2Annotations(t, msg, inFile, incr)

	// Remove annotations by id.
	if err := api.RemoveAnnotationsFile(inFile, "", nil, []string{"ID1", "ID2"}, nil, nil, incr); err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// We should have 0 annotations as at the beginning.
	if i := annotationCount(t, inFile); i > 0 {
		t.Fatalf("%s count: got %d want 0\n", msg, i)
	}
}

func TestAddRemoveAnnotationsByIdAndAnnotType(t *testing.T) {
	msg := "TestAddRemoveAnnotationsByIdAndAnnotType"

	incr := false // incremental updates

	fn := "test.pdf"
	copyFile(t, filepath.Join(inDir, fn), filepath.Join(outDir, fn))
	inFile := filepath.Join(outDir, fn)

	add2Annotations(t, msg, inFile, incr)

	// Remove annotations by id annotation type.
	if err := api.RemoveAnnotationsFile(inFile, "", nil, []string{"ID1", "Link"}, nil, nil, incr); err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// We should have 0 annotations as at the beginning.
	if i := annotationCount(t, inFile); i > 0 {
		t.Fatalf("%s count: got %d want 0\n", msg, i)
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

	allPages, err := api.PagesForPageSelection(ctx.PageCount, nil, true, true)
	if err != nil {
		t.Fatalf("%s pagesForPageSelection: %v\n", msg, err)
	}

	// Add link annotation to all pages.
	ok, err := pdfcpu.AddAnnotations(ctx, allPages, linkAnn, false)
	if err != nil || !ok {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Write context to file.
	err = api.WriteContextFile(ctx, inFile)
	if err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}

	// We should have 1 annotation
	if i := annotationCount(t, inFile); i != 1 {
		t.Fatalf("%s count: got %d want 0\n", msg, i)
	}

	// Create a context.
	ctx, err = api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Identify object numbers for located annotations
	objNrs, err := pdfcpu.CachedAnnotationObjNrs(ctx)
	if err != nil {
		t.Fatalf("%s annObjNrs: %v\n", msg, err)
	}
	if len(objNrs) != 1 {
		t.Fatalf("%s want 1 annotation, got: %d\n", msg, len(objNrs))
	}

	// Remove annotations by their object numbers
	// We could also do: api.RemoveAnnotationsFile
	// but since we already have the ctx this is more straight forward.
	_, err = pdfcpu.RemoveAnnotations(ctx, allPages, nil, objNrs, false)
	if err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// Write context to file.
	err = api.WriteContextFile(ctx, inFile)
	if err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}

	// We should have 0 annotations like at the beginning.
	if i := annotationCount(t, inFile); i > 0 {
		t.Fatalf("%s count: got %d want 0\n", msg, i)
	}
}

func TestAddRemoveAnnotationsByObjNrAndAnnotType(t *testing.T) {
	msg := "TestAddRemoveAnnotationsByObjNrAndAnnotType"

	incr := false // incremental updates

	fn := "test.pdf"
	copyFile(t, filepath.Join(inDir, fn), filepath.Join(outDir, fn))
	inFile := filepath.Join(outDir, fn)

	add2Annotations(t, msg, inFile, incr)

	// Remove annotations by obj and annotation type.
	// Here we use the obj# of the link Annotation to be removed.
	if err := api.RemoveAnnotationsFile(inFile, "", nil, []string{"Link"}, []int{6}, nil, incr); err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// We should have 1 annotations.
	if i := annotationCount(t, inFile); i != 1 {
		t.Fatalf("%s count: got %d want 0\n", msg, i)
	}
}

func TestAddRemoveAnnotationsByIdAndObjNrAndAnnotType(t *testing.T) {
	msg := "TestAddRemoveAnnotationsByObjNrAndAnnotType"

	incr := false // incremental updates

	fn := "test.pdf"
	copyFile(t, filepath.Join(inDir, fn), filepath.Join(outDir, fn))
	inFile := filepath.Join(outDir, fn)

	add2Annotations(t, msg, inFile, incr)

	// Remove annotations by id annotation type.
	if err := api.RemoveAnnotationsFile(inFile, "", nil, []string{"ID1", "Link"}, nil, nil, incr); err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// We should have 0 annotations as at the beginning.
	if i := annotationCount(t, inFile); i > 0 {
		t.Fatalf("%s count: got %d want 0\n", msg, i)
	}
}

func TestRemoveAllAnnotations(t *testing.T) {
	msg := "TestRemoveAllAnnotations"

	incr := false

	fn := "test.pdf"
	copyFile(t, filepath.Join(inDir, fn), filepath.Join(outDir, fn))
	inFile := filepath.Join(outDir, fn)

	m := map[int][]model.AnnotationRenderer{}
	anns := make([]model.AnnotationRenderer, 2)
	anns[0] = textAnn
	anns[1] = linkAnn
	m[1] = anns

	err := api.AddAnnotationsMapFile(inFile, "", m, nil, incr)
	if err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// We should have 2 annotations.
	if i := annotationCount(t, inFile); i != 2 {
		t.Fatalf("%s count: got %d want 0\n", msg, i)
	}

	// Remove all annotations.
	err = api.RemoveAnnotationsFile(inFile, "", nil, nil, nil, nil, incr)
	if err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// We should have 0 annotations like at the beginning.
	if i := annotationCount(t, inFile); i > 0 {
		t.Fatalf("%s count: got %d want 0\n", msg, i)
	}
}

func TestAddRemoveAllAnnotationsAsIncrements(t *testing.T) {
	msg := "TestAddRemoveAnnotationsAsIncrements"

	incr := true // incremental updates

	fn := "test.pdf"
	copyFile(t, filepath.Join(inDir, fn), filepath.Join(outDir, fn))
	inFile := filepath.Join(outDir, fn)

	add2Annotations(t, msg, inFile, incr)

	// Remove all page annotations and append the result as PDF increment to inFile.
	if err := api.RemoveAnnotationsFile(inFile, "", nil, nil, nil, nil, true); err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// We should have 0 annotations like at the beginning.
	if i := annotationCount(t, inFile); i > 0 {
		t.Fatalf("%s count: got %d want 0\n", msg, i)
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

	m := map[int][]model.AnnotationRenderer{}
	anns := make([]model.AnnotationRenderer, 2)
	anns[0] = textAnn
	anns[1] = linkAnn
	m[1] = anns

	// Add 2 annotations to page 1.
	if ok, err := pdfcpu.AddAnnotationsMap(ctx, m, false); err != nil || !ok {
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
	i, _, err := pdfcpu.ListAnnotations(ctx.PageAnnots)
	if err != nil || i != 2 {
		t.Fatalf("%s list: %v\n", msg, err)
	}

	// Remove all annotations.
	_, err = pdfcpu.RemoveAnnotations(ctx, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("%s remove: %v\n", msg, err)
	}

	// (before writing) We should have 0 annotations like at the beginning.
	i, _, err = pdfcpu.ListAnnotations(ctx.PageAnnots)
	if err != nil || i != 0 {
		t.Fatalf("%s list: %v\n", msg, err)
	}

	// Write context to file.
	if err := api.WriteContextFile(ctx, outFile); err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}

	// (after writing) We should have 0 annotations like at the beginning.
	if i := annotationCount(t, inFile); i > 0 {
		t.Fatalf("%s count: got %d want 0\n", msg, i)
	}
}

func TestAddLinkAnnotationWithDest(t *testing.T) {
	msg := "TestAddLinkAnnotationWithDest"

	// Best viewed with Adobe Reader.

	inFile := filepath.Join(inDir, "Walden.pdf")
	outFile := filepath.Join(samplesDir, "annotations", "LinkAnnotWithDestTopLeft.pdf")

	// Create internal link:
	// Add a 100x100 link rectangle on the bottom left corner of page 2.
	// Set destination to top left corner of page 1.
	dest := &model.Destination{Typ: model.DestXYZ, PageNr: 1, Left: -1, Top: -1}

	internalLink := model.NewLinkAnnotation(
		*types.NewRectangle(0, 0, 100, 100), // rect
		"",                                  // contents
		"ID2",                               // id
		"",                                  // modDate
		0,                                   // f
		&color.Red,                          // borderCol
		dest,                                // dest
		"",                                  // uri
		nil,                                 // quad
		true,                                // border
		1,                                   // borderWidth
		model.BSSolid,                       // borderStyle
	)

	err := api.AddAnnotationsFile(inFile, outFile, []string{"2"}, internalLink, nil, false)
	if err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}
}

func TestAddAnnotationsFile(t *testing.T) {
	msg := "TestAddAnnotationsFile"

	// Best viewed with Adobe Reader.

	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(samplesDir, "annotations", "Annotations.pdf")

	// Add text annotation.
	if err := api.AddAnnotationsFile(inFile, outFile, nil, textAnn, nil, false); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Add CJK text annotation.
	if err := api.AddAnnotationsFile(outFile, outFile, nil, textAnnCJK, nil, false); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Add link annotation.
	if err := api.AddAnnotationsFile(outFile, outFile, nil, linkAnn, nil, false); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Add square annotation.
	if err := api.AddAnnotationsFile(outFile, outFile, nil, squareAnn, nil, false); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Add CJK square annotation.
	if err := api.AddAnnotationsFile(outFile, outFile, nil, squareAnnCJK, nil, false); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Add circle annotation.
	if err := api.AddAnnotationsFile(outFile, outFile, nil, circleAnn, nil, false); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Add CJK circle annotation.
	if err := api.AddAnnotationsFile(outFile, outFile, nil, circleAnnCJK, nil, false); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}

}

func TestAddAnnotations(t *testing.T) {
	msg := "TestAddAnnotations"

	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(outDir, "Annotations.pdf")

	// Create a context from inFile.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Prepare annotations for page 1.
	m := map[int][]model.AnnotationRenderer{}
	anns := make([]model.AnnotationRenderer, 7)

	anns[0] = textAnn
	anns[1] = textAnnCJK
	anns[2] = squareAnn
	anns[3] = squareAnnCJK
	anns[4] = circleAnn
	anns[5] = circleAnnCJK
	anns[6] = linkAnn

	m[1] = anns

	// Add 7 annotations to page 1.
	if ok, err := pdfcpu.AddAnnotationsMap(ctx, m, false); err != nil || !ok {
		t.Fatalf("%s add: %v\n", msg, err)
	}

	// Write context to outFile.
	if err := api.WriteContextFile(ctx, outFile); err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}

}

func TestPopupAnnotation(t *testing.T) {
	msg := "TestPopupAnnotation"

	// Add a Markup annotation and a linked Popup annotation.
	// Best viewed with Adobe Reader.

	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(samplesDir, "annotations", "PopupAnnotation.pdf")

	incr := false
	pageNr := 1

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Add Markup annotation.
	parentIndRef, textAnnotDict, err := pdfcpu.AddAnnotationToPage(ctx, pageNr, textAnn, incr)
	if err != nil {
		t.Fatalf("%s Add Text AnnotationToPage: %v\n", msg, err)
	}

	// Add Markup annotation as parent of Popup annotation.
	popupAnn := model.NewPopupAnnotation(
		*types.NewRectangle(0, 0, 100, 100), // rect
		"Popup content",                     // contents
		"IDPopup",                           // id
		"",                                  // modDate
		0,                                   // f
		&color.Green,                        // col
		parentIndRef,                        // parentIndRef,
		false,                               // displayOpen
	)

	// Add Popup annotation.
	popupIndRef, _, err := pdfcpu.AddAnnotationToPage(ctx, pageNr, popupAnn, incr)
	if err != nil {
		t.Fatalf("%s Add Popup AnnotationToPage: %v\n", msg, err)
	}

	// Add Popup annotation to Markup annotation.
	textAnnotDict["Popup"] = *popupIndRef

	// Write context to file.
	if err := api.WriteContextFile(ctx, outFile); err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}
}

func TestInkAnnotation(t *testing.T) {
	msg := "TestInkAnnotation"

	// Best viewed with Adobe Reader.

	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(samplesDir, "annotations", "InkAnnotation.pdf")

	p1 := model.InkPath{100., 542., 150., 492., 200., 542.}
	p2 := model.InkPath{100, 592, 150, 592}

	inkAnn := model.NewInkAnnotation(
		*types.NewRectangle(0, 0, 100, 100), // rect
		"Ink content",                       // contents
		"IDInk",                             // id
		"",                                  // modDate
		0,                                   // f
		&color.Red,                          // col
		"Title1",                            // title
		nil,                                 // popupIndRef
		nil,                                 // ca
		"",                                  // rc
		"",                                  // subject
		[]model.InkPath{p1, p2},             // InkList
		0,                                   // borderWidth
		model.BSSolid,                       // borderStyle
	)

	// Add Ink annotation.
	if err := api.AddAnnotationsFile(inFile, outFile, nil, inkAnn, nil, false); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}
}

func TestHighlightAnnotation(t *testing.T) {
	msg := "TestHighlightAnnotation"

	// Best viewed with Adobe Reader.

	inFile := filepath.Join(inDir, "testWithText.pdf")
	outFile := filepath.Join(samplesDir, "annotations", "HighlightAnnotation.pdf")

	r := types.NewRectangle(205, 624.16, 400, 645.88)

	ql := types.NewQuadLiteralForRect(r)

	inkAnn := model.NewHighlightAnnotation(
		*r,                    // rect
		"Highlight content",   // contents
		"IDHighlight",         // id
		"",                    // modDate
		model.AnnLocked,       // f
		&color.Yellow,         // col
		"Comment by Horst",    // title
		nil,                   // popupIndRef
		nil,                   // ca
		"",                    // rc
		"Subject",             // subject
		types.QuadPoints{*ql}, // quad points
	)

	// Add Highlight annotation.
	if err := api.AddAnnotationsFile(inFile, outFile, nil, inkAnn, nil, false); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}
}

func TestUnderlineAnnotation(t *testing.T) {
	msg := "TestUnderlineAnnotation"

	// Best viewed with Adobe Reader.

	inFile := filepath.Join(inDir, "testWithText.pdf")
	outFile := filepath.Join(samplesDir, "annotations", "UnderlineAnnotation.pdf")

	r := types.NewRectangle(205, 624.16, 400, 645.88)

	ql := types.NewQuadLiteralForRect(r)

	underlineAnn := model.NewUnderlineAnnotation(
		*r,                    // rect
		"Underline content",   // contents
		"IDUnderline",         // id
		"",                    // modDate
		model.AnnLocked,       // f
		&color.Yellow,         // col
		"Title1",              // title
		nil,                   // popupIndRef
		nil,                   // ca
		"",                    // rc
		"",                    // subject
		types.QuadPoints{*ql}, // quad points
	)

	// Add Underline annotation.
	if err := api.AddAnnotationsFile(inFile, outFile, nil, underlineAnn, nil, false); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}
}

func TestSquigglyAnnotation(t *testing.T) {
	msg := "TestSquigglyAnnotation"

	// Best viewed with Adobe Reader.

	inFile := filepath.Join(inDir, "testWithText.pdf")
	outFile := filepath.Join(samplesDir, "annotations", "SquigglyAnnotation.pdf")

	r := types.NewRectangle(205, 624.16, 400, 645.88)

	ql := types.NewQuadLiteralForRect(r)

	squigglyAnn := model.NewSquigglyAnnotation(
		*r,                    // rect
		"Squiggly content",    // contents
		"IDSquiggly",          // id
		"",                    // modDate
		model.AnnLocked,       // f
		&color.Yellow,         // col
		"Title1",              // title
		nil,                   // popupIndRef
		nil,                   // ca
		"",                    // rc
		"",                    // subject
		types.QuadPoints{*ql}, // quad points
	)

	// Add Squiggly annotation.
	if err := api.AddAnnotationsFile(inFile, outFile, nil, squigglyAnn, nil, false); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}
}

func TestStrikeOutAnnotation(t *testing.T) {
	msg := "TestStrikeOutAnnotation"

	// Best viewed with Adobe Reader.

	inFile := filepath.Join(inDir, "testWithText.pdf")
	outFile := filepath.Join(samplesDir, "annotations", "StrikeOutAnnotation.pdf")

	r := types.NewRectangle(205, 624.16, 400, 645.88)

	ql := types.NewQuadLiteralForRect(r)

	strikeOutAnn := model.NewStrikeOutAnnotation(
		*r,                    // rect
		"StrikeOut content",   // contents
		"IDStrikeOut",         // id
		"",                    // modDate
		model.AnnLocked,       // f
		&color.Yellow,         // col
		"Title1",              // title
		nil,                   // popupIndRef
		nil,                   // ca
		"",                    // rc
		"",                    // subject
		types.QuadPoints{*ql}, // quad points
	)

	// Add StrikeOut annotation.
	if err := api.AddAnnotationsFile(inFile, outFile, nil, strikeOutAnn, nil, false); err != nil {
		t.Fatalf("%s add: %v\n", msg, err)
	}
}
