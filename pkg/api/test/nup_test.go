/*
Copyright 2020 The pdf Authors.

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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func testNUp(t *testing.T, msg string, inFiles []string, outFile string, selectedPages []string, desc string, n int, isImg bool, conf *model.Configuration) {
	t.Helper()

	var (
		nup *model.NUp
		err error
	)

	if isImg {
		if nup, err = api.ImageNUpConfig(n, desc, conf); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	} else {
		if nup, err = api.PDFNUpConfig(n, desc, conf); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	}

	if err := api.NUpFile(inFiles, outFile, selectedPages, nup, conf); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestNUp(t *testing.T) {

	outDir := filepath.Join(samplesDir, "nup")

	for _, tt := range []struct {
		msg           string
		inFiles       []string
		outFile       string
		selectedPages []string
		desc          string
		unit          string
		n             int
		isImg         bool
	}{
		// 4-Up a PDF
		{"TestNUpFromPDF",
			[]string{filepath.Join(inDir, "WaldenFull.pdf")},
			filepath.Join(outDir, "NUpFromPDF.pdf"),
			nil,
			"dim: 400 800, margin:10, bgcol:#f7e6c7",
			"mm",
			9,
			false},

		// 2-Up a PDF with CropBox
		{"TestNUpFromPdfWithCropBox",
			[]string{filepath.Join(inDir, "grid_example.pdf")},
			filepath.Join(outDir, "NUpFromPDFWithCropBox.pdf"),
			nil,
			"form:A5L, border:on, margin:0, bgcol:#f7e6c7",
			"points",
			2,
			false},

		// 16-Up an image
		{"TestNUpFromSingleImage",
			[]string{filepath.Join(resDir, "logoSmall.png")},
			filepath.Join(outDir, "NUpFromSingleImage.pdf"),
			nil,
			"form:A3P, ma:10, bgcol:#f7e6c7",
			"points",
			16,
			true},

		// 6-Up a sequence of images.
		{"TestNUpFromImages",
			imageFileNames(t, resDir),
			filepath.Join(outDir, "NUpFromImages.pdf"),
			nil,
			"form:Tabloid, border:on, ma:10, bgcol:#f7e6c7",
			"points",
			6,
			true},
	} {
		conf := model.NewDefaultConfiguration()
		conf.SetUnit(tt.unit)
		testNUp(t, tt.msg, tt.inFiles, tt.outFile, tt.selectedPages, tt.desc, tt.n, tt.isImg, conf)
	}
}
