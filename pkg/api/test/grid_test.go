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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func testGrid(t *testing.T, msg string, inFiles []string, outFile string, selectedPages []string, desc string, rows, cols int, isImg bool) {
	t.Helper()

	var (
		nup *pdfcpu.NUp
		err error
	)

	if isImg {
		if nup, err = api.ImageGridConfig(rows, cols, desc); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	} else {
		if nup, err = api.PDFGridConfig(rows, cols, desc); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	}

	if err := api.NUpFile(inFiles, outFile, selectedPages, nup, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestGrid(t *testing.T) {
	for _, tt := range []struct {
		msg           string
		inFiles       []string
		outFile       string
		selectedPages []string
		desc          string
		rows, cols    int
		isImg         bool
	}{
		{"TestGridFromPDF",
			[]string{filepath.Join(inDir, "read.go.pdf")},
			filepath.Join("..", "..", "samples", "grid", "GridFromPDF.pdf"),
			nil, "form:LegalP, o:dr, border:off", 4, 6, false},

		{"TestGridFromPDFWithCropBox",
			[]string{filepath.Join(inDir, "grid_example.pdf")},
			filepath.Join("..", "..", "samples", "grid", "GridFromPDFWithCropBox.pdf"),
			nil, "form:A5L, border:on, margin:0", 2, 1, false},

		{"TestGridFromImages",
			imageFileNames(t, "../../../resources"),
			filepath.Join("..", "..", "samples", "grid", "GridFromImages.pdf"),
			nil, "d:500 500, margin:20, bo:off", 1, 4, true},
	} {
		testGrid(t, tt.msg, tt.inFiles, tt.outFile, tt.selectedPages, tt.desc, tt.rows, tt.cols, tt.isImg)
	}
}
