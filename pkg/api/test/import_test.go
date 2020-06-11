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
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func testImportImages(t *testing.T, msg string, imgFiles []string, outFile, impConf string, ensureOutFile bool) {
	t.Helper()
	var err error
	outDir := "../../samples/import"
	outFile = filepath.Join(outDir, outFile)
	if ensureOutFile {
		// We want to test appending to an existing PDF.
		copyFile(t, filepath.Join(inDir, outFile), outFile)
	}

	// The default import conf uses the special pos:full argument
	// which overrides all other import conf parms.
	imp := pdf.DefaultImportConfig()
	if impConf != "" {
		if imp, err = pdf.ParseImportDetails(impConf); err != nil {
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
	for _, tt := range []struct {
		msg           string
		imgFiles      []string
		outFile       string
		impConf       string
		ensureOutFile bool
	}{
		//Convert an image into a single page PDF.
		//The page dimensions will match the image dimensions.
		{"TestConvertImageToPDF",
			[]string{filepath.Join(resDir, "logoSmall.png")},
			"ConvertImage.pdf",
			"",
			false},

		// Import an image as a new page of the existing output file.
		// {"TestImportImage",
		// 	[]string{filepath.Join(resDir, "logoSmall.png")},
		// 	"importImage.pdf",
		// 	"",
		// 	true},

		//Import images by creating an A3 page for each image.
		//Images are page centered with 1.0 relative scaling.
		//Import an image as a new page of the existing output file.
		{"TestCenteredImportImage",
			imageFileNames(t, "../../../resources"),
			"ImportImageToA3.pdf",
			"f:A3, pos:c, s:1.0",
			false},
	} {
		testImportImages(t, tt.msg, tt.imgFiles, tt.outFile, tt.impConf, tt.ensureOutFile)
	}
}
