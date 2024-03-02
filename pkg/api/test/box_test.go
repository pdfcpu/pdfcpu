/*
Copyright 2019 The pdf Authors.

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
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func listBoxes(t *testing.T, fileName string, pb *model.PageBoundaries) ([]string, error) {
	t.Helper()

	msg := "listBoxes"

	f, err := os.Open(fileName)
	if err != nil {
		t.Fatalf("%s open: %v\n", msg, err)
	}
	defer f.Close()

	ctx, err := api.ReadValidateAndOptimize(f, conf)
	if err != nil {
		t.Fatalf("%s ReadValidateAndOptimize: %v\n", msg, err)
	}

	if pb == nil {
		pb = &model.PageBoundaries{}
		pb.SelectAll()
	}

	return ctx.ListPageBoundaries(nil, pb)
}

func TestListBoxes(t *testing.T) {
	msg := "TestListBoxes"
	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")

	if _, err := listBoxes(t, inFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// List crop box for all pages.
	pb, err := api.PageBoundariesFromBoxList("crop")
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if _, err := listBoxes(t, inFile, pb); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestCrop(t *testing.T) {
	msg := "TestCrop"
	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(outDir, "out.pdf")

	for _, tt := range []struct {
		s string
		u types.DisplayUnit
	}{
		{"[0 0 5 5]", types.CENTIMETRES},
		{"100", types.POINTS},
		{"20% 40%", types.POINTS},
		{"dim:30 30", types.POINTS},
		{"dim:50% 50%", types.POINTS},
		{"pos:bl, dim:50% 50%", types.POINTS},
		{"pos:tl, off: 10 -10, dim:50% 50%", types.POINTS},
		{"pos:tl, dim:.5 1 rel", types.POINTS},
		{"-1", types.INCHES},
		{"-25%", types.POINTS},
	} {
		box, err := api.Box(tt.s, tt.u)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}

		if err := api.CropFile(inFile, outFile, nil, box, nil); err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
	}
}

func TestAddBoxes(t *testing.T) {
	msg := "TestAddBoxes"
	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(outDir, "out.pdf")

	for _, tt := range []struct {
		s string
		u types.DisplayUnit
	}{
		{"art:10%", types.POINTS}, // When using relative positioning unit is irrelevant.
		{"trim:10", types.POINTS},
		{"crop:[0 0 5 5]", types.CENTIMETRES}, // Crop 5 x 5 cm at bottom left corner
		{"crop:10", types.POINTS},
		{"crop:-10", types.POINTS},
		{"crop:10 20, trim:crop, art:bleed, bleed:art", types.POINTS},
		{"crop:10 20, trim:crop, art:bleed, bleed:media", types.POINTS},
		{"c:10 20, t:c, a:b, b:m", types.POINTS},
		{"crop:10, trim:20, art:trim", types.POINTS},
	} {
		pb, err := api.PageBoundaries(tt.s, tt.u)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}

		if err := api.AddBoxesFile(inFile, outFile, nil, pb, nil); err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
	}
}

func TestAddRemoveBoxes(t *testing.T) {
	msg := "TestAddRemoveBoxes"
	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(outDir, "out.pdf")

	pb, err := api.PageBoundaries("crop:[0 0 100 100]", types.POINTS)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := api.AddBoxesFile(inFile, outFile, nil, pb, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	pb, err = api.PageBoundariesFromBoxList("crop")
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := api.RemoveBoxesFile(outFile, outFile, nil, pb, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}
