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
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func testImportImages(t *testing.T, msg string, imgFiles []string, outFile, impConf string) {
	t.Helper()
	var err error

	// The default import conf uses the special pos:full argument
	// which overrides all other import conf parms.
	imp := pdf.DefaultImportConfig()
	if impConf != "" {
		if imp, err = api.Import(impConf, pdf.POINTS); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	}
	if err := api.ImportImagesFile(imgFiles, outFile, imp, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestImportImages(t *testing.T) {
	outDir := filepath.Join("..", "..", "samples", "import")
	testFile1 := filepath.Join(outDir, "CenteredGraySepia.pdf")
	testFile2 := filepath.Join(outDir, "Full.pdf")
	os.Remove(testFile1)
	os.Remove(testFile2)

	for _, tt := range []struct {
		msg      string
		imgFiles []string
		outFile  string
		impConf  string
	}{
		// Render image on an A4 portrait mode page.
		{"TestCenteredGraySepia",
			[]string{filepath.Join(resDir, "mountain.jpg")},
			testFile1,
			"f:A4, pos:c, bgcol:#beded9"},

		// Import another image as a new page of testfile1 and convert image to gray.
		{"TestCenteredGraySepia",
			[]string{filepath.Join(resDir, "mountain.png")},
			testFile1,
			"f:A4, pos:c, sc:.75, bgcol:#beded9, gray:true"},

		// Import another image as a new page of testfile1 and apply a sepia filter.
		{"TestCenteredGraySepia",
			[]string{filepath.Join(resDir, "mountain.webp")},
			testFile1,
			"f:A4, pos:c, sc:.9, bgcol:#beded9, sepia:true"},

		// Import another image as a new page of testfile1.
		{"TestCenteredGraySepia",
			[]string{filepath.Join(resDir, "mountain.tif")},
			testFile1,
			"f:A4, pos:c, sc:1, bgcol:#beded9"},

		// Page dimensions match image dimensions.
		{"TestFull",
			imageFileNames(t, filepath.Join("..", "..", "..", "resources")),
			testFile2,
			"pos:full"},
	} {
		testImportImages(t, tt.msg, tt.imgFiles, tt.outFile, tt.impConf)
	}
}

func TestMemBasedWriterPanic(t *testing.T) {

	imgFiles := []string{filepath.Join(resDir, "logoSmall.png")}

	rr := make([]io.Reader, len(imgFiles))
	for i, fn := range imgFiles {
		f, err := os.Open(fn)
		if err != nil {
			t.Fatal(err)
		}
		rr[i] = bufio.NewReader(f)
	}

	outBuf := &bytes.Buffer{}

	if err := api.ImportImages(nil, outBuf, rr, nil, nil); err != nil {
		t.Fatal(err)
	}

}
