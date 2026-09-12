/*
Copyright 2023 The pdf Authors.

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
	"math"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestResizeByScaleFactor verifies resize by scale factor.
func TestResizeByScaleFactor(t *testing.T) {
	msg := "TestResizeByScaleFactor"

	inFile := filepath.Join(inDir, "test.pdf")

	// Enlarge by scale factor 2.
	res, err := pdfcpu.ParseResizeConfig("sc:2", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile := filepath.Join(samplesDir, "resize", "enlargeByScaleFactor.pdf")
	if err := api.ResizeFile(t.Context(), inFile, outFile, nil, res, nil); err != nil {
		t.Fatalf("%s resize: %v\n", msg, err)
	}

	// Shrink by 50%.
	res, err = pdfcpu.ParseResizeConfig("sc:.5", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile = filepath.Join(samplesDir, "resize", "shrinkByScaleFactor.pdf")
	if err := api.ResizeFile(t.Context(), inFile, outFile, nil, res, nil); err != nil {
		t.Fatalf("%s resize: %v\n", msg, err)
	}
}

// TestResizeByWidthOrHeight verifies resize by width or height.
func TestResizeByWidthOrHeight(t *testing.T) {
	msg := "TestResizeByWidthOrHeight"

	inFile := filepath.Join(inDir, "test.pdf")

	// Set width to 200 points.
	res, err := pdfcpu.ParseResizeConfig("dim:200 0", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile := filepath.Join(samplesDir, "resize", "resizeByWidth.pdf")
	if err := api.ResizeFile(t.Context(), inFile, outFile, nil, res, nil); err != nil {
		t.Fatalf("%s resize: %v\n", msg, err)
	}

	// Set height to 200 mm.
	res, err = pdfcpu.ParseResizeConfig("dim:0 200", types.MILLIMETRES)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile = filepath.Join(samplesDir, "resize", "resizeByHeight.pdf")
	if err := api.ResizeFile(t.Context(), inFile, outFile, nil, res, nil); err != nil {
		t.Fatalf("%s resize: %v\n", msg, err)
	}
}

// TestResizeToFormSize verifies resize to form size.
func TestResizeToFormSize(t *testing.T) {
	msg := "TestResizeToPaperSize"

	inFile := filepath.Join(inDir, "test.pdf")

	// Resize to A3 and keep orientation.
	res, err := pdfcpu.ParseResizeConfig("form:A3", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile := filepath.Join(samplesDir, "resize", "resizeToA3.pdf")
	if err := api.ResizeFile(t.Context(), inFile, outFile, nil, res, nil); err != nil {
		t.Fatalf("%s resize: %v\n", msg, err)
	}

	// Resize to A4 and enforce orientation (here landscape mode).
	res, err = pdfcpu.ParseResizeConfig("form:A4L", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile = filepath.Join(samplesDir, "resize", "resizeToA4L.pdf")
	if err := api.ResizeFile(t.Context(), inFile, outFile, nil, res, nil); err != nil {
		t.Fatalf("%s resize: %v\n", msg, err)
	}
}

// TestResizeToDimensions verifies resize to dimensions.
func TestResizeToDimensions(t *testing.T) {
	msg := "TestResizeToDimensions"

	inFile := filepath.Join(inDir, "test.pdf")

	// Resize to 400 x 200 and keep orientation of input file.
	// Apply background color to unused space.
	res, err := pdfcpu.ParseResizeConfig("dim:400 200, bgcol:#E9967A", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile := filepath.Join(samplesDir, "resize", "resizeToDimensionsKeep.pdf")
	if err := api.ResizeFile(t.Context(), inFile, outFile, nil, res, nil); err != nil {
		t.Fatalf("%s resize: %v\n", msg, err)
	}

	// Resize to 400 x 200 and enforce new orientation.
	// Render border of original crop box.
	res, err = pdfcpu.ParseResizeConfig("dim:400 200, enforce:true, border:on", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile = filepath.Join(samplesDir, "resize", "resizeToDimensionsEnforce.pdf")
	if err := api.ResizeFile(t.Context(), inFile, outFile, nil, res, nil); err != nil {
		t.Fatalf("%s resize: %v\n", msg, err)
	}
}

func annotationNumberArray(t *testing.T, ctx *model.Context, obj types.Object) []float64 {
	t.Helper()
	arr, err := ctx.DereferenceArray(obj)
	if err != nil {
		t.Fatal(err)
	}
	ff := make([]float64, len(arr))
	for i, o := range arr {
		f, err := ctx.DereferenceNumber(o)
		if err != nil {
			t.Fatal(err)
		}
		ff[i] = f
	}
	return ff
}

func assertFloatSlicesEqual(t *testing.T, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d values, want %d", len(got), len(want))
	}
	for i := range got {
		if math.Abs(got[i]-want[i]) > 0.01 {
			t.Fatalf("value[%d]: got %.2f want %.2f", i, got[i], want[i])
		}
	}
}

func resizeAnnotationExpectedValues(ff []float64) []float64 {
	sc := 792. / 842.
	dx := (612. - 595.*sc) / 2

	a := make([]float64, len(ff))
	for i := 0; i < len(ff); i += 2 {
		a[i] = ff[i]*sc + dx
		a[i+1] = ff[i+1] * sc
	}
	return a
}

func resizeTestAnnotationDict(t *testing.T, fileName string) (*model.Context, types.Dict) {
	t.Helper()
	ctx, err := api.ReadContextFile(t.Context(), fileName)
	if err != nil {
		t.Fatal(err)
	}
	d, _, _, err := ctx.PageDict(t.Context(), 1, false)
	if err != nil {
		t.Fatal(err)
	}
	arr, err := ctx.DereferenceArray(d["Annots"])
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range arr {
		d, err := ctx.DereferenceDict(o)
		if err != nil {
			t.Fatal(err)
		}
		if s := d.StringEntry("NM"); s != nil && *s == "IDResizeLink" {
			return ctx, d
		}
	}
	t.Fatal("missing resize test annotation")
	return nil, nil
}

// TestResizeAnnotationGeometry verifies resize applies the content transform to annotations.
func TestResizeAnnotationGeometry(t *testing.T) {
	msg := "TestResizeAnnotationGeometry"

	inFile := filepath.Join(inDir, "test.pdf")
	annotFile := filepath.Join(outDir, "resizeAnnotationIn.pdf")
	outFile := filepath.Join(outDir, "resizeAnnotationOut.pdf")

	r := types.NewRectangle(95, 595, 160, 615)
	ql := types.NewQuadLiteralForRect(r)
	ann := model.NewLinkAnnotation(
		*r,                    // rect
		0,                     // apObjNr
		"",                    // contents
		"IDResizeLink",        // id
		"",                    // modDate
		0,                     // f
		&color.Red,            // borderCol
		nil,                   // dest
		"https://pdfcpu.io",   // uri
		types.QuadPoints{*ql}, // quad
		false,                 // border
		0,                     // borderWidth
		model.BSSolid,         // borderStyle
	)

	if err := api.AddAnnotationsFile(t.Context(), inFile, annotFile, []string{"1"}, ann, nil, false); err != nil {
		t.Fatalf("%s add annotation: %v\n", msg, err)
	}

	res, err := pdfcpu.ParseResizeConfig("form:Letter", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}
	if err := api.ResizeFile(t.Context(), annotFile, outFile, []string{"1"}, res, nil); err != nil {
		t.Fatalf("%s resize: %v\n", msg, err)
	}

	ctx, d := resizeTestAnnotationDict(t, outFile)
	assertFloatSlicesEqual(t, annotationNumberArray(t, ctx, d["Rect"]), resizeAnnotationExpectedValues([]float64{95, 595, 160, 615}))
	assertFloatSlicesEqual(t, annotationNumberArray(t, ctx, d["QuadPoints"]), resizeAnnotationExpectedValues([]float64{95, 615, 160, 615, 95, 595, 160, 595}))
}
