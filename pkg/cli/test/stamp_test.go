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

	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func testAddWatermarks(t *testing.T, msg, inFile, outFile string, selectedPages []string, mode, modeParm, desc string, onTop bool) {
	t.Helper()
	inFile = filepath.Join(inDir, inFile)
	outFile = filepath.Join(outDir, outFile)

	var (
		wm  *pdfcpu.Watermark
		err error
	)
	switch mode {
	case "text":
		wm, err = pdf.ParseTextWatermarkDetails(modeParm, desc, onTop)
	case "image":
		wm, err = pdf.ParseImageWatermarkDetails(modeParm, desc, onTop)
	case "pdf":
		wm, err = pdf.ParsePDFWatermarkDetails(modeParm, desc, onTop)
	}
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	cmd := cli.AddWatermarksCommand(inFile, outFile, selectedPages, wm, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := validateFile(t, outFile, nil); err != nil {
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
			"testwm.pdf",
			[]string{"1-"},
			false,
			"text",
			"Draft",
			"scale:0.7, rot:20"},

		// Add a greenish, slightly transparent stroked and filled text stamp to all odd pages of inFile other than page 1
		// using the default rotation which is aligned along the first diagonal running from lower left to upper right corner.
		{"TestStampText",
			"pike-stanford.pdf",
			"testStampText1.pdf",
			[]string{"odd", "!1"},
			true,
			"text",
			"Demo",
			"font:Courier, c: 0 .8 0, op:0.8, mode:2"},

		// Add a red filled text stamp to all odd pages of inFile other than page 1 using a font size of 48 points
		// and the default rotation which is aligned along the first diagonal running from lower left to upper right corner.
		{"TestStampTextUsingFontsize",
			"pike-stanford.pdf",
			"testStampText2.pdf",
			[]string{"odd", "!1"},
			true,
			"text",
			"Demo",
			"font:Courier, c: 1 0 0, op:1, sc:1 abs, points:48"},

		// Add image watermark to inFile starting at page 1 using no rotation.
		{"TestWatermarkImage",
			"Acroforms2.pdf", "testWMImageRel.pdf",
			[]string{"1-"},
			false,
			"image",
			filepath.Join(resDir, "pdfchip3.png"),
			"rot:0"},

		// Add image stamp to inFile using absolute scaling and a negative rotation of 90 degrees.
		{"TestStampImageAbsScaling",
			"Acroforms2.pdf",
			"testWMImageAbs.pdf",
			[]string{"1-"},
			true,
			"image",
			filepath.Join(resDir, "pdfchip3.png"),
			"sc:.5 a, rot:-90"},

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

func TestStampingLifecyle(t *testing.T) {
	msg := "TestStampingLifecyle"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "stampLC.pdf")
	onTop := true // we are testing stamps

	// Stamp all pages.
	wm, err := pdf.ParseTextWatermarkDetails("Demo", "", onTop)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	cmd := cli.AddWatermarksCommand(inFile, outFile, nil, wm, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// // Update stamp on page 1.
	wm, err = pdf.ParseTextWatermarkDetails("Confidential", "", onTop)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	wm.Update = true
	cmd = cli.AddWatermarksCommand(outFile, "", []string{"1"}, wm, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Add another stamp on top for all pages.
	// This is a redish transparent footer.
	wm, err = pdf.ParseTextWatermarkDetails("Footer", "pos:bc, c:0.8 0 0, op:.6, rot:0", onTop)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	cmd = cli.AddWatermarksCommand(outFile, "", nil, wm, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Remove stamp on page 1.
	cmd = cli.RemoveWatermarksCommand(outFile, "", []string{"1"}, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Remove all stamps.
	cmd = cli.RemoveWatermarksCommand(outFile, "", nil, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Validate the result.
	if err := validateFile(t, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}
