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
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func testNUp(t *testing.T, msg string, inFiles []string, outFile string, selectedPages []string, desc string, n int, isImg bool) {
	t.Helper()

	var (
		nup *pdf.NUp
		err error
	)

	if isImg {
		if nup, err = pdf.ImageNUpConfig(n, desc); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	} else {
		if nup, err = pdf.PDFNUpConfig(n, desc); err != nil {
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

func imageFileNames(t *testing.T, dir string) []string {
	t.Helper()
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	fn := []string{}
	for _, fi := range files {
		if strings.HasSuffix(fi.Name(), "png") || strings.HasSuffix(fi.Name(), "jpg") {
			fn = append(fn, filepath.Join(dir, fi.Name()))
		}
	}
	return fn
}

func TestNUp(t *testing.T) {
	outDir := "../../samples/nup"

	for _, tt := range []struct {
		msg           string
		inFiles       []string
		outFile       string
		selectedPages []string
		desc          string
		n             int
		isImg         bool
	}{
		// 4-Up a PDF
		{"TestNUpFromPDF",
			[]string{filepath.Join(inDir, "WaldenFull.pdf")},
			filepath.Join(outDir, "TestNUpFromPDF.pdf"),
			nil,
			"",
			9,
			false},

		// 9-Up an image
		{"TestNUpFromSingleImage",
			[]string{filepath.Join(resDir, "logoSmall.png")},
			filepath.Join(outDir, "NUpFromSingleImage.pdf"),
			nil,
			"f:A3P",
			16,
			true},

		// 6-Up a sequence of images.
		{"TestNUpFromImages",
			imageFileNames(t, "../../../resources"),
			filepath.Join(outDir, "NUpFromImages.pdf"),
			nil,
			"f:Tabloid, b:on, m:0",
			6,
			true},
	} {
		testNUp(t, tt.msg, tt.inFiles, tt.outFile, tt.selectedPages, tt.desc, tt.n, tt.isImg)
	}
}
