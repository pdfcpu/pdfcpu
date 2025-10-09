/*
Copyright 2024 The pdf Authors.

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
)

func testUpdateImages(t *testing.T, msg string, inFile, imgFile, outFile string, objNr, pageNr int, id string) {
	t.Helper()

	if err := api.UpdateImagesFile(inFile, imgFile, outFile, objNr, pageNr, id, conf); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestUpdateImages(t *testing.T) {

	outDir := filepath.Join(samplesDir, "images")
	inDir := outDir

	for _, tt := range []struct {
		msg     string
		inFile  string
		imgFile string
		outFile string
		objNr   int // by objNr
		pageNr  int // or by (pageNr, id)
		id      string
	}{
		{"TestUpdateByObjNr",
			"test.pdf",
			"test_1_Im1.png",
			"ImageUpdatedByObjNr.pdf",
			8,
			0,
			""},

		{"TestUpdateByPageNrAndId",
			"test.pdf",
			"test_1_Im1.png",
			"imageUpdatedByPageNrAndIdPage1.pdf",
			0,
			1,
			"Im1"},

		{"TestUpdateByPageNrAndId",
			"test.pdf",
			"test_1_Im1.png",
			"imageUpdatedByPageNrAndIdPage2.pdf",
			0,
			2,
			"Im1"},

		{"TestUpdateByImageFileName",
			"test.pdf",
			"test_1_Im1.png",
			"imageUpdatedByFileName.pdf",
			0,
			0,
			""},

		{"TestUpdateByPageNrAndId",
			"test.pdf",
			"any.png",
			"imageUpdatedByPageNrAndIdAny.pdf",
			0,
			1,
			"Im1"},

		{"TestUpdateByObjNrPNG",
			"test.pdf",
			"any.png",
			"imageUpdatedByObjNrPNG.pdf",
			8,
			0,
			""},

		{"TestUpdateByObjNrJPG",
			"test.pdf",
			"any.jpg",
			"imageUpdatedByObjNrJPG.pdf",
			8,
			0,
			""},

		{"TestUpdateByObjNrTIFF",
			"test.pdf",
			"any.tiff",
			"imageUpdatedByObjNrTIFF.pdf",
			8,
			0,
			""},

		{"TestUpdateByObjNrWEBP",
			"test.pdf",
			"any.webp",
			"imageUpdatedByObjNrWEBP.pdf",
			8,
			0,
			""},
		{"TestUpdateByObjNrPNGGray",
			"test.pdf",
			"any_gray.png",
			"imageUpdatedByObjNrPNGGray.pdf",
			8,
			0,
			""},
		{"TestUpdateByPageNrAndIdGray",
			"test.pdf",
			"any_gray.png",
			"imageUpdatedByPageNrAndIdAnyGray.pdf",
			0,
			1,
			"Im1"},
	} {
		testUpdateImages(t, tt.msg,
			filepath.Join(inDir, tt.inFile),
			filepath.Join(outDir, tt.imgFile),
			filepath.Join(outDir, tt.outFile),
			tt.objNr,
			tt.pageNr,
			tt.id)
	}
}
