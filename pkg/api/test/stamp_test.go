/*
Copyright 2020 The pdfcpu Authors.

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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func testAddWatermarks(t *testing.T, msg, inFile, outFile string, selectedPages []string, mode, modeParam, desc string, onTop bool) {
	t.Helper()
	inFile = filepath.Join(inDir, inFile)
	s := "watermark"
	if onTop {
		s = "stamp"
	}
	outFile = filepath.Join(samplesDir, s, mode, outFile)

	var err error
	switch mode {
	case "text":
		err = api.AddTextWatermarksFile(inFile, outFile, selectedPages, onTop, modeParam, desc, nil)
	case "image":
		err = api.AddImageWatermarksFile(inFile, outFile, selectedPages, onTop, modeParam, desc, nil)
	case "pdf":
		err = api.AddPDFWatermarksFile(inFile, outFile, selectedPages, onTop, modeParam, desc, nil)
	}
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestAddWatermarks(t *testing.T) {
	for _, tt := range []struct {
		msg             string
		inFile, outFile string
		selectedPages   []string
		mode            string
		modeParm        string
		wmConf          string
	}{

		// Avoid font embedding for CJK fonts like so:

		// Any font name ending with GB2312 will be recognized as using HANS:

		// {"TestWatermarkText",
		// 	"sample.pdf",
		// 	"chinese.pdf",
		// 	[]string{"1-"},
		// 	"text",
		// 	"测试中文字体水印增加的文件大小\n2023-10-16",
		// 	"font: KaiTi_GB2312, points: 36, scale: 1 abs, color: #ff0000, op: 0.3, ro: 30"},

		// Configure script manually:

		// {"TestWatermarkText",
		// 	"sample.pdf",
		// 	"chinese1.pdf",
		// 	[]string{"1-"},
		// 	"text",
		// 	"测试中文字体水印增加的文件大小\n2023-10-16",
		// 	"font: KaiTi_GB2312, script: hans,  points: 36, scale: 1 abs, color: #ff0000, op: 0.3, ro: 30"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextDefaults.pdf",
			[]string{"1-"},
			"text",
			"A simple text watermark using defaults:\n" +
				"\"font:Helvetica, points:24, aligntext:center,\n" +
				"position:c, offset:0 0, scale:0.5 rel, diagonal:1,\n" +
				"opacity:1, rendermode:0, fillcolor: 0.5 0.5 0.5,\n" +
				"strokecol: 0.5 0.5 0.5\"",
			""},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextDefaultsAbbr.pdf",
			[]string{"1-"},
			"text",
			`A simple text watermark using defaults:
					Unique abbreviations also work:
					"fo:Helvetica, poi:24, align:c,
					pos:c, off:0 0, scale:0.5 rel, d:1,
					op:1, mode:0, fillc: 0.5 0.5 0.5,
					strokec: #808080"`,
			""},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextAlongLeftBorder.pdf",
			[]string{"1-"},
			"text",
			"Welcome to pdfcpu",
			"pos:l, off:0 0, rot:-90"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextPagenumbers.pdf",
			[]string{"1-"},
			"text",
			"Page %p of %P",
			"scale:1 abs, pos:bc, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextRenderMode0.pdf",
			[]string{"1-"},
			"text",
			"Rendermode 0 fills text using fill color.\n" +
				"\"rendermode\" or \"mode\" works - also abbreviated: \n" +
				"\"mode:0, fillc:#3277d3, rot:0, scale:.8\"",
			"mode:0, fillc:#3277d3, rot:0, scale:.8"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextRenderMode1.pdf",
			[]string{"1-"},
			"text",
			"Rendermode 1 strokes text using stroke color.\n" +
				"\"rendermode\" or \"mode\" works - also abbreviated: \n" +
				"\"mo:1, strokec:#335522, rot:0, scale:.8\"",
			"mo:1, strokec:#335522, rot:0, scale:.8"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextRenderMode2.pdf",
			[]string{"1-"},
			"text",
			"Rendermode 2 strokes text using stroke color\n" +
				"and fills text using fill color\n" +
				"\"rendermode\" or \"mode\" works - also abbreviated: \n" +
				"\"re:2, fillc:#3277d3, strokec:#335522, rot:0, scale:.8\"",
			"re:2, fillc:#3277d3, strokec:#335522, rot:0, scale:.8"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextAlignLeft.pdf",
			[]string{"1-"},
			"text",
			"Here we have\n" +
				"some left aligned text lines\n" +
				"\"align:l, fillc:#3277d3, rot:0\"",
			"align:l, fillc:#3277d3, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextAlignRight.pdf",
			[]string{"1-"},
			"text",
			"Here we have\n" +
				"some right aligned text lines\n" +
				"with background color\n" +
				"\"align:l, fillc:#3277d3, bgcol:#f7e6c7, rot:0\"",
			"align:r, fillc:#3277d3, bgcol:#f7e6c7, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextAlignCenter.pdf",
			[]string{"1-"},
			"text",
			"Here we have\n" +
				"some centered text lines\n" +
				"with background color\n" +
				"\"fillc:#3277d3, bgcol:#beded9, rot:0\"",
			"fillc:#3277d3, bgcol:#beded9, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextAlignJustify.pdf",
			[]string{"1-"},
			"text",
			"Here we have\n" +
				"some justified text lines\n" +
				"with background color\n" +
				"\"al:j, fillc:#3277d3, bgcol:#000000, rot:0\"",
			"al:j, fillc:#3277d3, bgcol:#000000, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextScaleRel25.pdf",
			[]string{"1-"},
			"text",
			"Relative scale factor: .25\n" +
				"scales relative to page dimensions.\n" +
				"\"scale:.25 rel, fillc:#3277d3, rot:0\"",
			"scale:.25 rel, fillc:#3277d3, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextScaleRel50.pdf",
			[]string{"1-"},
			"text",
			"Relative scale factor: .5\n" +
				"scales relative to page dimensions.\n" +
				"\"scale:.5, fillc:#3277d3, rot:0\"",
			"scale:.5, fillc:#3277d3, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextScaleRel100.pdf",
			[]string{"1-"},
			"text",
			"Relative scale factor: 1\n" +
				"scales relative to page dimensions.\n" +
				"\"scale:1, fillc:#3277d3, rot:0\"",
			"scale:1, fillc:#3277d3, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextScaleAbs50.pdf",
			[]string{"1-"},
			"text",
			"Absolute scale factor: .5\n" +
				"scales fontsize\n" +
				"(here using the 24 points default)\n" +
				"\"scale:.5 abs, font:Courier, rot:0\"",
			"scale:.5 abs, font:Courier, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextScaleAbs100.pdf",
			[]string{"1-"},
			"text",
			"Absolute scale factor: 1\n" +
				"scales fontsize\n" +
				"(here using the 24 points default)\n" +
				"\"scale:1 abs, font:Courier, rot:0\"",
			"scale:1 abs, font:Courier, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextScaleAbs150.pdf",
			[]string{"1-"},
			"text",
			"Absolute scale factor: 1.5\n" +
				"scales fontsize\n" +
				"(here using the 24 points default)\n" +
				"\"scale:1.5 abs, font:Courier, rot:0\"",
			"scale:1.5 abs, font:Courier, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextPosBotLeft.pdf",
			[]string{"1-"},
			"text",
			"Positioning using anchors:\n" +
				"bottom left corner with left alignment\n" +
				"\"pos:bl, bgcol:#f7e6c7, rot:0\"",
			"pos:bl, bgcol:#f7e6c7, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextPosBotRightWithOffset.pdf",
			[]string{"1-"},
			"text",
			"Positioning using anchors and offset:\n" +
				"bottom right corner with right alignment\n" +
				"\"pos:br, off: -10 10, align:r, bgcol:#f7e6c7, rot:0\"",
			"pos:br, off: -10 10, align:r, bgcol:#f7e6c7, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextOffAndRot.pdf",
			[]string{"1-"},
			"text",
			"Confidential\n\"scale:1 abs, points:20, pos:c, off:0 50, fillc:#000000, rot:20\"",
			"scale:1 abs, points:20, pos:c, off:0 50, fillc:#000000, rot:20"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextMargins1Value.pdf",
			[]string{"1-"},
			"text",
			"Set all margins:\n" +
				"(needs \"bgcol\")\n" +
				"\"margins: 10, fillc:#3277d3, bgcol:#beded9, rot:0\"",
			"margins: 10,fillc:#3277d3, bgcol:#beded9, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextMargins2Values.pdf",
			[]string{"1-"},
			"text",
			"Set top/bottom and left/right margins:\n" +
				"(needs \"bgcol\")\n" +
				"\"ma: 5 10, fillc:#3277d3, bgcol:#beded9, rot:0\"",
			"ma: 5 10, fillc:#3277d3, bgcol:#beded9, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextMargins3Values.pdf",
			[]string{"1-"},
			"text",
			"Set top, left/right and  bottom margins:\n" +
				"(needs \"bgcol\")\n" +
				"\"ma: 5 10 15, fillc:#3277d3, bgcol:#beded9, rot:0\"",
			"ma: 5 10 15, fillc:#3277d3, bgcol:#beded9, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextMargins4Values.pdf",
			[]string{"1-"},
			"text",
			"Set all margins individually:\n" +
				"(needs \"bgcol\")\n" +
				"\"ma: 5 10 15 20, fillc:#3277d3, bgcol:#beded9, rot:0\"",
			"ma: 5 10 15 20, fillc:#3277d3, bgcol:#beded9, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextRoundCornersAndBorder5.pdf",
			[]string{"1-"},
			"text",
			"Set round corners and border:\n" +
				"(needs \"bgcol\" and a border)\n" +
				"round corner effect depends on border width\n" +
				"\"border: 5 round, fillc:#3277d3, bgcol:#beded9, rot:0\"",
			"border: 5 round, fillc:#3277d3, bgcol:#beded9, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextRoundCornersAndBorder10.pdf",
			[]string{"1-"},
			"text",
			"Set round corners and border:\n" +
				"(needs \"bgcol\" and a border)\n" +
				"round corner effect depends on border width\n" +
				"\"border: 10 round, fillc:#3277d3, bgcol:#beded9, rot:0\"",
			"border: 10 round, fillc:#3277d3, bgcol:#beded9, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextRoundCornersAndColoredBorder.pdf",
			[]string{"1-"},
			"text",
			"Set round corners and colored border:\n" +
				"(needs \"bgcol\")\n" +
				"round corner effect depends on border width\n" +
				"\"border: 10 round #f7e6c7, fillc:#3277d3, bgcol:#beded9, rot:0\"",
			"border: 10 round #f7e6c7, fillc:#3277d3, bgcol:#beded9, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextMarginsAndColoredBorder.pdf",
			[]string{"1-"},
			"text",
			"Set margins and colored border:\n" +
				"(needs \"bgcol\")\n" +
				"\"ma: 10, bo: 5 .3 .7 .7, fillc:#3277d3, bgcol:#beded9, rot:0\"",
			"ma: 10, bo: 5 .3 .7 .7, fillc:#3277d3, bgcol:#beded9, rot:0"},

		{"TestWatermarkText",
			"Walden.pdf",
			"TextMarginsRoundCornersAndColoredBorder.pdf",
			[]string{"1-"},
			"text",
			"Set margins and round colored border:\n" +
				"(needs \"bgcol\")\n" +
				"round corner effect depends on border width\n" +
				"\"ma: 5, bo: 7 round .3 .7 .7, fillc:#3277d3, bgcol:#beded9, rot:0\"",
			"ma: 5, bo: 7 round .3 .7 .7, fillc:#3277d3, bgcol:#beded9, rot:0"},

		// Add image watermark to inFile starting at page 1 using no rotation.
		{"TestWatermarkImage",
			"Walden.pdf",
			"ImageRotate90.pdf",
			[]string{"1-"},
			"image",
			filepath.Join(resDir, "logoSmall.png"),
			"scale:.25, rot:90"},

		// Add image watermark to inFile for all pages using defaults..
		{"TestWatermarkImage2",
			"Walden.pdf",
			"ImagePosBottomLeftWithOffset.pdf",
			nil,
			"image",
			filepath.Join(resDir, "logoSmall.png"),
			"scale:.1, pos:bl, off:15 20, rot:0"},

		// Add image stamp to inFile using absolute scaling and a rotation of 45 degrees.
		{"TestStampImageAbsScaling",
			"Walden.pdf",
			"ImageAbsScaling.pdf",
			[]string{"1-"},
			"image",
			filepath.Join(resDir, "logoSmall.png"),
			"scale:.33 abs, rot:45"},

		// Add a PDF stamp to all pages of inFile using the 1st page of pdfFile
		// and rotate along the 2nd diagonal running from upper left to lower right corner.
		{"TestWatermarkPDF",
			"Walden.pdf",
			"PdfSingleStampDefault.pdf",
			nil,
			"pdf",
			filepath.Join(inDir, "Walden.pdf:1"),
			"d:2"},

		// Add a PDF multistamp in the top right corner to all pages of inFile.
		{"TestWatermarkPDF",
			"Walden.pdf",
			"PdfMultistampDefault.pdf",
			nil,
			"pdf",
			filepath.Join(inDir, "Walden.pdf"),
			"scale:.2, pos:tr, off:-10 -10, rot:0"},

		// Add a PDF multistamp to all pages of inFile.
		// Start by stamping page 3 with page 1.
		// You may filter stamping by defining selected Pages.
		{"TestWatermarkPDF",
			"zineTest.pdf",
			"PdfMultistamp13.pdf",
			nil,
			"pdf",
			filepath.Join(inDir, "zineTest.pdf:1:3"),
			"scale:.2, pos:tr, off:-10 -10, rot:0"},

		// Add a PDF multistamp to all pages of inFile.
		// Start by stamping page 1 with page 3.
		// You may filter stamping by defining selected Pages.
		{"TestWatermarkPDF",
			"zineTest.pdf",
			"PdfMultistamp31.pdf",
			nil,
			"pdf",
			filepath.Join(inDir, "zineTest.pdf:3:1"),
			"scale:.2, pos:tr, off:-10 -10, rot:0"},

		// Add a PDF multistamp to all pages of inFile.
		// Start by stamping page 3 with page 3.
		// You may filter stamping by defining selected Pages.
		{"TestWatermarkPDF",
			"zineTest.pdf",
			"PdfMultistamp33.pdf",
			nil,
			"pdf",
			filepath.Join(inDir, "zineTest.pdf:3:3"),
			"scale:.2, pos:tr, off:-10 -10, rot:0"},
	} {
		testAddWatermarks(t, tt.msg, tt.inFile, tt.outFile, tt.selectedPages, tt.mode, tt.modeParm, tt.wmConf, false)
		testAddWatermarks(t, tt.msg, tt.inFile, tt.outFile, tt.selectedPages, tt.mode, tt.modeParm, tt.wmConf, true)
	}
}

func TestAddStampWithLink(t *testing.T) {
	for _, tt := range []struct {
		msg             string
		inFile, outFile string
		selectedPages   []string
		mode            string
		modeParm        string
		wmConf          string
	}{
		{"TestStampTextWithLink",
			"Walden.pdf",
			"TextWithLink.pdf",
			[]string{"1-"},
			"text",
			"A simple text watermark with link",
			"url:pdfcpu.io"},

		{"TestStampImageWithLink",
			"Walden.pdf",
			"ImageWithLink.pdf",
			[]string{"1-"},
			"image",
			filepath.Join(resDir, "logoSmall.png"),
			"url:pdfcpu.io, scale:.33 abs, rot:45"},
	} {
		// Links supported for stamps only (watermark onTop:true).
		testAddWatermarks(t, tt.msg, tt.inFile, tt.outFile, tt.selectedPages, tt.mode, tt.modeParm, tt.wmConf, true)
	}

}

func TestCropBox(t *testing.T) {
	msg := "TestCropBox"
	inFile := filepath.Join(inDir, "empty.pdf")
	outFile := filepath.Join(samplesDir, "stamp", "pdf", "PdfWithCropBox.pdf")
	pdfFile := filepath.Join(inDir, "grid_example.pdf")

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	for _, pos := range []string{"tl", "tc", "tr", "l", "c", "r", "bl", "bc", "br"} {
		wm, err := api.PDFWatermark(pdfFile+":1", fmt.Sprintf("scale:.25 rel, pos:%s, rot:0", pos), true, false, types.POINTS)
		if err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
		if err := pdfcpu.AddWatermarks(ctx, nil, wm); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	}

	// Write context to file.
	if err := api.WriteContextFile(ctx, outFile); err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func hasWatermarks(inFile string, t *testing.T) bool {
	t.Helper()
	ok, err := api.HasWatermarksFile(inFile, nil)
	if err != nil {
		t.Fatalf("Checking for watermarks: %s: %v\n", inFile, err)
	}
	return ok
}

func TestStampingLifecycle(t *testing.T) {
	msg := "TestStampingLifecycle"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "stampLC.pdf")
	onTop := true // we are testing stamps

	// Check for existing stamps.
	if ok := hasWatermarks(inFile, t); ok {
		t.Fatalf("Watermarks found: %s\n", inFile)
	}

	unit := types.POINTS

	// Stamp all pages.
	wm, err := api.TextWatermark("Demo", "", onTop, false, unit)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := api.AddWatermarksFile(inFile, outFile, nil, wm, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Check for existing stamps.
	if ok := hasWatermarks(outFile, t); !ok {
		t.Fatalf("No watermarks found: %s\n", outFile)
	}

	// // Update stamp on page 1.
	wm, err = api.TextWatermark("Confidential", "", onTop, true, unit)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := api.AddWatermarksFile(outFile, "", []string{"1"}, wm, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Add another stamp on top for all pages.
	// This is a redish transparent footer.
	wm, err = api.TextWatermark("Footer", "pos:bc, c:0.8 0 0, op:.6, rot:0", onTop, false, unit)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := api.AddWatermarksFile(outFile, "", nil, wm, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Remove stamp on page 1.
	if err := api.RemoveWatermarksFile(outFile, "", []string{"1"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Check for existing stamps.
	if ok := hasWatermarks(outFile, t); !ok {
		t.Fatalf("No watermarks found: %s\n", outFile)
	}

	// Remove all stamps.
	if err := api.RemoveWatermarksFile(outFile, "", nil, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Validate the result.
	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Check for existing stamps.
	if ok := hasWatermarks(outFile, t); ok {
		t.Fatalf("Watermarks found: %s\n", outFile)
	}
}

func TestRecycleWM(t *testing.T) {
	msg := "TestRecycleWM"
	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(samplesDir, "watermark", "text", "TextRecycled.pdf")
	onTop := false // we are testing watermarks

	desc := "pos:tl, points:22, rot:0, scale:1 abs, off:0 -5, opacity:0.3"
	wm, err := api.TextWatermark("This is a watermark", desc, onTop, false, types.POINTS)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	if err = api.AddWatermarksFile(inFile, outFile, nil, wm, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	wm.Recycle()

	// Shift down watermark.
	wm.Dy = -55

	if err = api.AddWatermarksFile(outFile, outFile, nil, wm, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}
