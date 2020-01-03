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
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func testAddWatermarks(t *testing.T, msg, inFile, outFile string, selectedPages []string, mode, modeParam, desc string, onTop bool) {
	t.Helper()
	inFile = filepath.Join(inDir, inFile)
	outFile = filepath.Join(outDir, outFile)

	var (
		wm  *pdf.Watermark
		err error
	)
	switch mode {
	case "text":
		wm, err = pdf.ParseTextWatermarkDetails(modeParam, desc, onTop)
	case "image":
		wm, err = pdf.ParseImageWatermarkDetails(modeParam, desc, onTop)
	case "pdf":
		wm, err = pdf.ParsePDFWatermarkDetails(modeParam, desc, onTop)
	}
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	if err := api.AddWatermarksFile(inFile, outFile, selectedPages, wm, nil); err != nil {
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
		onTop           bool
		mode            string
		modeParm        string
		wmConf          string
	}{
		// Add text watermark to all pages of inFile starting at page 1 using a rotation angle of 20 degrees.
		{"TestWatermarkText",
			"Acroforms2.pdf",
			"TestWatermarkText.pdf",
			[]string{"1-"},
			false,
			"text",
			"Line1\n \nLine3",
			"f:Helvetica, s:0.5, rot:20"},

		// Add a greenish, slightly transparent stroked and filled text stamp to all odd pages of inFile other than page 1
		// using the default rotation which is aligned along the first diagonal running from lower left to upper right corner.
		{"TestStampText",
			"pike-stanford.pdf",
			"TestStampText.pdf",
			[]string{"odd", "!1"},
			true,
			"text",
			"Demo",
			"f:Courier, c: 0 .8 0, op:0.8, m:2"},

		// Add a red filled text stamp to all odd pages of inFile other than page 1 using a font size of 48 points
		// and the default rotation which is aligned along the first diagonal running from lower left to upper right corner.
		{"TestStampTextUsingFontsize",
			"pike-stanford.pdf",
			"TestStampTextUsingFontsize.pdf",
			[]string{"odd", "!1"},
			true,
			"text",
			"Demo",
			"font:Courier, c: 1 0 0, op:1, s:1 abs, points:48"},

		// Add image watermark to inFile starting at page 1 using no rotation.
		{"TestWatermarkImage",
			"Acroforms2.pdf",
			"TestWatermarkImage.pdf",
			[]string{"1-"},
			false,
			"image",
			filepath.Join(resDir, "pdfchip3.png"),
			"rot:0"},

		// Add image watermark to inFile for all pages using defaults..
		{"TestWatermarkImage2",
			"empty.pdf",
			"TestWatermarkImage2.pdf",
			nil,
			false,
			"image",
			filepath.Join(resDir, "qr.png"),
			""},

		// Add image stamp to inFile using absolute scaling and a negative rotation of 90 degrees.
		{"TestStampImageAbsScaling",
			"Acroforms2.pdf",
			"testWMImageAbs.pdf",
			[]string{"1-"},
			true,
			"pdf",
			filepath.Join(resDir, "pdfchip3.png"),
			"s:.5 a, rot:-90"},

		// Add a PDF stamp to all pages of inFile using the 3rd page of pdfFile
		// and rotate along the 2nd diagonal running from upper left to lower right corner.
		{"TestWatermarkText",
			"Acroforms2.pdf",
			"testStampPDF.pdf",
			nil,
			true,
			"pdf",
			filepath.Join(inDir, "Wonderwall.pdf:3"),
			"d:2"},

		// Add a PDF multistamp to all pages of inFile
		// and rotate along the 2nd diagonal running from upper left to lower right corner.
		{"TestWatermarkText",
			"Acroforms2.pdf",
			"testMultistampPDF.pdf",
			nil,
			true,
			"pdf",
			filepath.Join(inDir, "Wonderwall.pdf"),
			"d:2"},
	} {
		testAddWatermarks(t, tt.msg, tt.inFile, tt.outFile, tt.selectedPages, tt.mode, tt.modeParm, tt.wmConf, tt.onTop)
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

func TestStampingLifecyle(t *testing.T) {
	msg := "TestStampingLifecyle"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "stampLC.pdf")
	onTop := true // we are testing stamps

	// Check for existing stamps.
	if ok := hasWatermarks(inFile, t); ok {
		t.Fatalf("Watermarks found: %s\n", inFile)
	}

	// Stamp all pages.
	wm, err := pdf.ParseTextWatermarkDetails("Demo", "", onTop)
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
	wm, err = pdf.ParseTextWatermarkDetails("Confidential", "", onTop)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	wm.Update = true
	if err := api.AddWatermarksFile(outFile, "", []string{"1"}, wm, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Add another stamp on top for all pages.
	// This is a redish transparent footer.
	wm, err = pdf.ParseTextWatermarkDetails("Footer", "pos:bc, c:0.8 0 0, op:.6, rot:0", onTop)
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
