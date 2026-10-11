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

package api

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const watermarkArtifactMarker = "/Artifact <</Subtype /Watermark /Type /Pagination >>BDC"

func twoPageStampInput(t *testing.T) []byte {
	t.Helper()
	ctx, err := pdfcpu.CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	pagesRef, err := ctx.Pages()
	if err != nil {
		t.Fatal(err)
	}
	pagesDict, err := ctx.DereferenceDict(*pagesRef)
	if err != nil {
		t.Fatal(err)
	}
	for pageNr := 1; pageNr <= 2; pageNr++ {
		pageRef, err := ctx.EmptyPage(pagesRef, types.RectForFormat("A4"), 0)
		if err != nil {
			t.Fatal(err)
		}
		if err := model.AppendPageTree(pageRef, pageNr, pagesDict); err != nil {
			t.Fatal(err)
		}
	}
	ctx.PageCount = 2
	var buf bytes.Buffer
	if err := WriteContext(t.Context(), ctx, &buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func pageHasWatermarkArtifact(t *testing.T, ctx *model.Context, pageNr int) bool {
	t.Helper()
	d, _, _, err := ctx.PageDict(t.Context(), pageNr, false)
	if err != nil {
		t.Fatal(err)
	}
	content, err := ctx.PageContent(d, pageNr)
	if errors.Is(err, model.ErrNoContent) {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Contains(content, []byte(watermarkArtifactMarker))
}

func TestRemoveWatermarkFromOnePagePreservesOtherPage(t *testing.T) {
	input := twoPageStampInput(t)
	wm := stampTestWatermark(t, false)
	var stamped bytes.Buffer
	if err := AddWatermarks(t.Context(), bytes.NewReader(input), &stamped, []string{"1-2"}, wm, nil); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := RemoveWatermarks(t.Context(), bytes.NewReader(stamped.Bytes()), &output, []string{"1"}, nil); err != nil {
		t.Fatal(err)
	}

	ctx, err := ReadContext(t.Context(), bytes.NewReader(output.Bytes()), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateContext(t.Context(), ctx); err != nil {
		t.Fatal(err)
	}
	if pageHasWatermarkArtifact(t, ctx, 1) {
		t.Fatal("expected page 1 watermark to be removed")
	}
	if !pageHasWatermarkArtifact(t, ctx, 2) {
		t.Fatal("expected page 2 to remain watermarked")
	}
}

func sharedStampPDF(inherited, blank bool) []byte {
	resources := "/Resources 6 0 R"
	parentResources := ""
	if inherited {
		parentResources, resources = resources, ""
	}
	if blank {
		resources, parentResources = "/Resources <<>>", ""
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		fmt.Sprintf("<< /Type /Pages /Kids [3 0 R 4 0 R 5 0 R] /Count 3 %s >>", parentResources),
	}
	for range 3 {
		objects = append(objects, fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] %s /Contents 9 0 R >>", resources))
	}
	content := "BT /F1 24 Tf 20 100 Td (Hello) Tj ET q 30 0 0 30 20 20 cm /Im1 Do Q"
	if blank {
		content = ""
	}
	objects = append(objects,
		"<< /Font << /F1 7 0 R >> /XObject << /Im1 8 0 R >> >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		"<< /Type /XObject /Subtype /Image /Width 1 /Height 1 /ColorSpace /DeviceRGB "+
			"/BitsPerComponent 8 /Length 3 >>\nstream\n\xff\x00\x00\nendstream",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content))
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(objects))
	for i, object := range objects {
		offsets[i] = appendValidationTestObject(&b, i+1, object)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Root 1 0 R /Size %d >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return b.Bytes()
}

func checkSharedStampForm(t *testing.T, ctx *model.Context, o types.Object) {
	t.Helper()
	o, err := ctx.Dereference(o)
	if err != nil {
		t.Fatal(err)
	}
	form, ok := o.(types.StreamDict)
	if !ok || form.Subtype() == nil || *form.Subtype() != "Form" {
		t.Fatalf("expected stamp form, got %T", o)
	}
	res, err := ctx.DereferenceDict(form.Dict["Resources"])
	if err != nil {
		t.Fatal(err)
	}
	checkSharedStampFont(t, ctx, res)
	checkSharedStampImage(t, ctx, res)
	if err := form.Decode(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(form.Content, []byte("(Hello) Tj")) || !bytes.Contains(form.Content, []byte("/Im1 Do")) {
		t.Fatalf("missing stamp text or image operation: %s", form.Content)
	}
}

func checkSharedStampOutput(t *testing.T, data []byte) {
	t.Helper()
	ctx, err := ReadAndValidate(t.Context(), bytes.NewReader(data), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if ctx.PageCount != 3 {
		t.Fatalf("page count: got %d, want 3", ctx.PageCount)
	}
	for pageNr := 1; pageNr <= 3; pageNr++ {
		_, _, attrs, err := ctx.PageDict(t.Context(), pageNr, false)
		if err != nil {
			t.Fatal(err)
		}
		xObjects, err := ctx.DereferenceDict(attrs.Resources["XObject"])
		if err != nil {
			t.Fatal(err)
		}
		if len(xObjects) != 1 {
			t.Fatalf("page %d: expected one stamp form, got %d", pageNr, len(xObjects))
		}
		for _, o := range xObjects {
			checkSharedStampForm(t, ctx, o)
		}
	}
}

func addSharedStamp(t *testing.T, inherited, onTop, fileAPI bool) []byte {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join(dir, "stamp.pdf")
	if err := os.WriteFile(source, sharedStampPDF(inherited, false), 0600); err != nil {
		t.Fatal(err)
	}
	wm, err := pdfcpu.ParsePDFWatermarkDetails(t.Context(), source, "scalefactor:1 abs, rotation:0", onTop, types.POINTS, nil)
	if err != nil {
		t.Fatal(err)
	}
	input := sharedStampPDF(false, true)
	conf := model.NewStatelessConfiguration()
	if !fileAPI {
		var output bytes.Buffer
		if err := AddWatermarks(t.Context(), bytes.NewReader(input), &output, nil, wm, conf); err != nil {
			t.Fatal(err)
		}
		return output.Bytes()
	}
	inFile, outFile := filepath.Join(dir, "input.pdf"), filepath.Join(dir, "output.pdf")
	if err := os.WriteFile(inFile, input, 0600); err != nil {
		t.Fatal(err)
	}
	if err := AddWatermarksFile(t.Context(), inFile, outFile, nil, wm, conf); err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func runSharedStampConfig(t *testing.T, optimize bool) {
	t.Helper()
	root := t.TempDir()
	if _, _, err := model.InitializeConfiguration(root); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "pdfcpu", "config.yml")
	config, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	config = bytes.ReplaceAll(config, []byte("optimizeResourceDicts: true"),
		[]byte(fmt.Sprintf("optimizeResourceDicts: %t", optimize)))
	if err := os.WriteFile(path, config, 0600); err != nil {
		t.Fatal(err)
	}
	// Isolate the source reader's default configuration from other tests.
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestPDFStampSharedResources$")
	cmd.Env = append(os.Environ(), "PDFCPU_TEST_STAMP_CONFIG="+root)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("stamp resources: %v\n%s", err, output)
	}
}

// TestPDFStampSharedResources preserves shared font and image resources across all imported source pages.
func TestPDFStampSharedResources(t *testing.T) {
	root := os.Getenv("PDFCPU_TEST_STAMP_CONFIG")
	if root == "" {
		for _, optimize := range []bool{false, true} {
			t.Run(fmt.Sprintf("optimize=%t", optimize), func(t *testing.T) { runSharedStampConfig(t, optimize) })
		}
		return
	}
	if err := model.EnsureDefaultConfigAt(root, false); err != nil {
		t.Fatal(err)
	}
	for _, inherited := range []bool{false, true} {
		for _, onTop := range []bool{false, true} {
			for _, fileAPI := range []bool{false, true} {
				name := fmt.Sprintf("inherited=%t/stamp=%t/file=%t", inherited, onTop, fileAPI)
				t.Run(name, func(t *testing.T) {
					checkSharedStampOutput(t, addSharedStamp(t, inherited, onTop, fileAPI))
				})
			}
		}
	}
}

func checkSharedStampFont(t *testing.T, ctx *model.Context, res types.Dict) {
	t.Helper()
	fonts, err := ctx.DereferenceDict(res["Font"])
	if err != nil {
		t.Fatal(err)
	}
	font, err := ctx.DereferenceDict(fonts["F1"])
	if err != nil || font == nil || font.NameEntry("BaseFont") == nil || *font.NameEntry("BaseFont") != "Helvetica" {
		t.Fatalf("missing stamp font: %v, %v", font, err)
	}
}

func checkSharedStampImage(t *testing.T, ctx *model.Context, res types.Dict) {
	t.Helper()
	xObjects, err := ctx.DereferenceDict(res["XObject"])
	if err != nil {
		t.Fatal(err)
	}
	o, err := ctx.Dereference(xObjects["Im1"])
	if err != nil {
		t.Fatal(err)
	}
	image, ok := o.(types.StreamDict)
	if !ok || image.Subtype() == nil || *image.Subtype() != "Image" {
		t.Fatalf("missing stamp image: %T", o)
	}
	if err := image.Decode(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(image.Content, []byte{255, 0, 0}) {
		t.Fatalf("stamp image pixels: %v", image.Content)
	}
}
