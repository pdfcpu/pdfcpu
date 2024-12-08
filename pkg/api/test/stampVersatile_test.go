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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestAlternatingPageNumbersViaWatermarkMap(t *testing.T) {
	msg := "TestAlternatingPageNumbersViaWatermarkMap"
	inFile := filepath.Join(inDir, "WaldenFull.pdf")
	outFile := filepath.Join(samplesDir, "stamp", "mixed", "AlternatingPageNumbersViaWatermarkMap.pdf")

	pageCount, err := api.PageCountFile(inFile)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Prepare a map of watermarks.
	// This maps pages to corresponding watermarks.
	// Any page may be assigned a single watermark of type text, image or PDF.
	m := map[int]*model.Watermark{}

	// Start stamping with page 2.
	// For odd page numbers add a blue stamp on the bottom right corner using Roboto-Regular
	// For even page numbers add a green stamp on the bottom left corner using Times-Italic
	for i := 2; i <= pageCount; i++ {
		text := fmt.Sprintf("%d of %d", i, pageCount)
		fontName := "Times-Italic"
		pos := "bl"
		dx := 10
		fillCol := "#008000"
		if i%2 > 0 {
			fontName = "Roboto-Regular"
			pos = "br"
			dx = -10
			fillCol = "#0000E0"
		}
		desc := fmt.Sprintf("font:%s, points:12, scale:1 abs, pos:%s, off:%d 10, fillcol:%s, rot:0", fontName, pos, dx, fillCol)
		wm, err := api.TextWatermark(text, desc, true, false, types.POINTS)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		m[i] = wm
	}

	if err := api.AddWatermarksMapFile(inFile, outFile, m, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Add a stamp with the creation date on the center of the bottom of every page.
	text := fmt.Sprintf("%%p of %%P - Creation date: %v", time.Now().Format("2006-01-02 15:04"))
	if err := api.AddTextWatermarksFile(outFile, outFile, nil, true, text, "fo:Roboto-Regular, points:12, scale:1 abs, pos:bc, off:0 10, rot:0", nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Add a "Draft" stamp with opacity 0.6 along the 1st diagonal in light blue using Courier.
	if err := api.AddTextWatermarksFile(outFile, outFile, nil, true, "Draft", "fo:Courier, scale:.9, fillcol:#00aacc, op:.6", nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestAlternatingPageNumbersViaWatermarkMapLowLevel(t *testing.T) {
	msg := "TestAlternatingPageNumbersViaWatermarkMapLowLevel"
	inFile := filepath.Join(inDir, "WaldenFull.pdf")
	outFile := filepath.Join(samplesDir, "stamp", "mixed", "AlternatingPageNumbersViaWatermarkMapLowLevel.pdf")

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	m := map[int]*model.Watermark{}
	unit := types.POINTS

	// Start stamping with page 2.
	// For odd page numbers add a blue stamp on the bottom right corner using Roboto-Regular
	// For even page numbers add a green stamp on the bottom left corner using Times-Italic
	for i := 2; i <= ctx.PageCount; i++ {
		text := fmt.Sprintf("%d of %d", i, ctx.PageCount)
		fontName := "Times-Italic"
		pos := "bl"
		dx := 10
		fillCol := "#008000"
		if i%2 > 0 {
			fontName = "Roboto-Regular"
			pos = "br"
			dx = -10
			fillCol = "#0000E0"
		}
		desc := fmt.Sprintf("font:%s, points:12, scale:1 abs, pos:%s, off:%d 10, fillcol:%s, rot:0", fontName, pos, dx, fillCol)
		wm, err := api.TextWatermark(text, desc, true, false, unit)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		m[i] = wm
	}

	if err := pdfcpu.AddWatermarksMap(ctx, m); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Add a stamp with the creation date on the center of the bottom of every page.
	text := fmt.Sprintf("%%p of %%P - Creation date: %v", time.Now().Format("2006-01-02 15:04"))
	wm, err := api.TextWatermark(text, "fo:Roboto-Regular, points:12, scale:1 abs, pos:bc, off:0 10, rot:0", true, false, unit)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := pdfcpu.AddWatermarks(ctx, nil, wm); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Add a "Draft" stamp with opacity 0.6 along the 1st diagonal in light blue using Courier.
	wm, err = api.TextWatermark("Draft", "fo:Courier, scale:.9, fillcol:#00aacc, op:.6", true, false, unit)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := pdfcpu.AddWatermarks(ctx, nil, wm); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Write context to file.
	if err := api.WriteContextFile(ctx, outFile); err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}
}

func TestAlternatingPageNumbersViaWatermarkSliceMap(t *testing.T) {
	msg := "TestAlternatingPageNumbersViaWatermarkSliceMap"
	inFile := filepath.Join(inDir, "WaldenFull.pdf")
	outFile := filepath.Join(samplesDir, "stamp", "mixed", "AlternatingPageNumbersViaWatermarkSliceMap.pdf")

	pageCount, err := api.PageCountFile(inFile)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	m := map[int][]*model.Watermark{}
	opacity := 1.0
	onTop := true // All stamps!
	update := false
	unit := types.POINTS

	// Prepare a map of watermark slices.
	// This maps pages to corresponding watermarks.
	// Each page may be assigned an arbitrary number of watermarks of type text, image or PDF.
	for i := 2; i <= pageCount; i++ {

		wms := []*model.Watermark{}

		// 1st watermark on page
		// For odd page numbers add a blue stamp on the bottom right corner using Roboto-Regular
		// For even page numbers add a green stamp on the bottom left corner using Times-Italic
		text := fmt.Sprintf("%d of %d", i, pageCount)
		fontName := "Times-Italic"
		pos := "bl"
		dx := 10
		fillCol := "#008000"
		if i%2 > 0 {
			fontName = "Roboto-Regular"
			pos = "br"
			dx = -10
			fillCol = "#0000E0"
		}
		desc := fmt.Sprintf("font:%s, points:12, scale:1 abs, pos:%s, off:%d 10, fillcol:%s, rot:0, op:%f", fontName, pos, dx, fillCol, opacity)
		wm, err := api.TextWatermark(text, desc, onTop, update, unit)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		wms = append(wms, wm)

		// 2nd watermark on page
		// Add a stamp with the creation date on the center of the bottom of every page.
		text = fmt.Sprintf("%%p of %%P - Creation date: %v", time.Now().Format("2006-01-02 15:04"))
		desc = fmt.Sprintf("fo:Roboto-Regular, points:12, scale:1 abs, pos:bc, off:0 10, rot:0, op:%f", opacity)
		wm, err = api.TextWatermark(text, desc, onTop, update, unit)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		wms = append(wms, wm)

		// 3rd watermark on page
		// Add a "Draft" stamp with opacity 0.6 along the 1st diagonal in light blue using Courier.
		text = "Draft"
		desc = fmt.Sprintf("fo:Courier, scale:.9, fillcol:#00aacc, op:%f", opacity)
		wm, err = api.TextWatermark(text, desc, onTop, update, unit)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		wms = append(wms, wm)

		m[i] = wms
	}

	// Apply all watermarks in one Go.
	// Assumption: All watermarks share the same opacity and onTop (all stamps or watermarks).
	// If you cannot ensure this you have to do something along the lines of func TestAlternatingPageNumbersViaWatermarkMap
	if err := api.AddWatermarksSliceMapFile(inFile, outFile, m, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestImagesTextAndPDFWMViaWatermarkMap(t *testing.T) {
	msg := "TestImagesTextAndPDFWMViaWatermarkMap"
	inFile := filepath.Join(inDir, "WaldenFull.pdf")
	outFile := filepath.Join(samplesDir, "stamp", "mixed", "ImagesTextAndPDFWMViaWatermarkMap.pdf")

	pageCount, err := api.PageCountFile(inFile)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	m := map[int]*model.Watermark{}
	fileNames := imageFileNames(t, resDir)

	opacity := 1.0
	onTop := true // All stamps!
	update := false
	unit := types.POINTS

	// Apply a mix of image, text and PDF watermarks in one go.
	for i := 1; i <= pageCount; i++ {
		if i <= len(fileNames) {
			desc := fmt.Sprintf("pos:bl, scale:.25, rot:0, op:%f", opacity)
			wm, err := api.ImageWatermark(fileNames[i-1], desc, onTop, update, unit)
			if err != nil {
				t.Fatalf("%s: %v\n", msg, err)
			}
			m[i] = wm
			continue
		}

		if i%2 > 0 {
			desc := fmt.Sprintf("scale:.25, pos:br, rot:0, op:%f", opacity)
			wm, err := api.PDFWatermark(inFile+":1", desc, onTop, update, unit)
			if err != nil {
				t.Fatalf("%s: %v\n", msg, err)
			}
			m[i] = wm
			continue
		}

		desc := fmt.Sprintf("rot:0, op:%f", opacity)
		wm, err := api.TextWatermark("Even page number", desc, onTop, update, unit)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		m[i] = wm
	}

	// Apply all watermarks in one Go.
	// Assumption: All watermarks share the same opacity and onTop (all stamps or watermarks).
	// If you cannot ensure this you have to do something along the lines of func TestAlternatingPageNumbersViaWatermarkMap
	if err := api.AddWatermarksMapFile(inFile, outFile, m, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestPdfSingleStampVariations(t *testing.T) {
	msg := "TestPdfSingleStampVariations"
	inFile := filepath.Join(inDir, "zineTest.pdf")
	stampFile := inFile

	// Stamp selected pages of inFile with one specific page of some PDF file - the stampFile.

	rs, err := os.Open(stampFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, stampFile, err)
	}
	defer rs.Close()

	for _, tt := range []struct {
		msg, outFile string
		pageNrSrc    int
	}{
		{"TestPdfSingleStampDefault", // Use page 2 of stampFile to stamp inFile pages.
			"PdfSingleStampDefault.pdf",
			2,
		},
		{"TestPdfMultiStampDefault", // Start stamping at page 1 using page 1 of stampFile.
			"TestPdfMultiStampDefault.pdf",
			0, // special case defaulting to multistamping
		},
	} {
		wm, err := api.PDFWatermarkForReadSeeker(
			rs,
			tt.pageNrSrc,
			"scale:.2, pos:tr, off:-10 -10, rot:0", // scaled @ top right corner using some offset and 0 rotation.
			true,                                   // stamp
			false,                                  // no update
			conf.Unit,
		)

		if err != nil {
			t.Fatalf("%s: %v\n", tt.msg, err)
		}

		outFile := filepath.Join(samplesDir, "stamp", "mixed", tt.outFile)

		if err = api.AddWatermarksFile(inFile, outFile, nil, wm, conf); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, outFile, err)
		}
	}
}

func TestPdfMultiStampVariations(t *testing.T) {
	msg := "TestPdfMultiStampVariations"
	inFile := filepath.Join(inDir, "zineTest.pdf")
	stampFile := inFile

	// Stamp selected pages of inFile with different pages of some PDF file - the stampFile.
	// Stamping proceeds in ascending manner where each new inFile page gets stamped with the next page of stampFile.
	// Set the first page of the stampFile initiating the sequence = startPageNrSrc
	// Set the first page of inFile that gets stamped = startPageNrDest

	rs, err := os.Open(stampFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, stampFile, err)
	}
	defer rs.Close()

	for _, tt := range []struct {
		msg, outFile    string
		startPageNrSrc  int
		startPageNrDest int
	}{
		{"TestPdfMultiStamp11", // Start stamping at page 1 using page 1 of stampFile. (=TestPdfMultiStampDefault)
			"PdfMultiStamp11.pdf",
			1,
			1,
		},
		{"TestPdfMultiStamp13", // Skip first 2 page and start stamping at page 3 using page 1 of stampFile.
			"PdfMultiStamp13.pdf",
			1,
			3,
		},
		{"TestPdfMultiStamp31", // Start stamping at page 1 using page 3 of stampFile.
			"PdfMultiStamp31.pdf",
			3,
			1,
		},
		{"TestPdfMultiStamp33", // Skip first 2 page and start stamping at page 3 using page 3 of stampFile.
			"PdfMultiStamp33.pdf",
			3,
			3,
		},
	} {
		wm, err := api.PDFMultiWatermarkForReadSeeker(
			rs,
			tt.startPageNrSrc,
			tt.startPageNrDest,
			"scale:.2, pos:tr, off:-10 -10, rot:0", // scaled @ top right corner using some offset and 0 rotation.
			true,                                   // stamp
			false,                                  // no update
			conf.Unit,
		)

		if err != nil {
			t.Fatalf("%s: %v\n", tt.msg, err)
		}

		outFile := filepath.Join(samplesDir, "stamp", "mixed", tt.outFile)

		if err = api.AddWatermarksFile(inFile, outFile, nil, wm, conf); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, outFile, err)
		}
	}
}
