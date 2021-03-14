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
	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func testImportImages(t *testing.T, msg string, imgFiles []string, outFile, impConf string) {
	t.Helper()
	var err error

	outFile = filepath.Join(outDir, outFile)

	// The default import conf uses the special pos:full argument
	// which overrides all other import conf parms.
	imp := pdfcpu.DefaultImportConfig()
	if impConf != "" {
		if imp, err = api.Import(impConf, pdfcpu.POINTS); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	}
	cmd := cli.ImportImagesCommand(imgFiles, outFile, imp, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := validateFile(t, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestImportCommand(t *testing.T) {
	for _, tt := range []struct {
		msg      string
		imgFiles []string
		outFile  string
		impConf  string
	}{
		// Render image on an A4 portrait mode page.
		{"TestCenteredGraySepia",
			[]string{filepath.Join(resDir, "mountain.jpg")},
			"CenteredGraySepia.pdf",
			"f:A4, pos:c, bgcol:#beded9"},

		// Import another image as a new page of testfile1 and convert image to gray.
		{"TestCenteredGraySepia",
			[]string{filepath.Join(resDir, "mountain.png")},
			"CenteredGraySepia.pdf",
			"f:A4, pos:c, sc:.75, bgcol:#beded9, gray:true"},

		// Import another image as a new page of testfile1 and apply a sepia filter.
		{"TestCenteredGraySepia",
			[]string{filepath.Join(resDir, "mountain.webp")},
			"CenteredGraySepia.pdf",
			"f:A4, pos:c, sc:.9, bgcol:#beded9, sepia:true"},

		// Import another image as a new page of testfile1.
		{"TestCenteredGraySepia",
			[]string{filepath.Join(resDir, "mountain.tif")},
			"CenteredGraySepia.pdf",
			"f:A4, pos:c, sc:1, bgcol:#beded9"},

		// Page dimensions match image dimensions.
		{"TestFull",
			imageFileNames(t, filepath.Join("..", "..", "..", "resources")),
			"Full.pdf",
			"pos:full"},
	} {
		testImportImages(t, tt.msg, tt.imgFiles, tt.outFile, tt.impConf)
	}
}
