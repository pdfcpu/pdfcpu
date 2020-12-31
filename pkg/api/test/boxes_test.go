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
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func TestListBoxes(t *testing.T) {
	msg := "TestListBoxes"
	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")

	if _, err := api.ListBoxesFile(inFile, nil, nil, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// List crop box for all pages.
	pb, err := api.PageBoundariesFromBoxList("crop")
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if _, err := api.ListBoxesFile(inFile, nil, pb, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestCrop(t *testing.T) {
	msg := "TestCrop"
	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(outDir, "out.pdf")

	for _, tt := range []struct {
		s string
		u pdfcpu.DisplayUnit
	}{
		{"[0 0 5 5]", pdfcpu.CENTIMETRES},
		{"100", pdfcpu.POINTS},
		{"20% 40%", pdfcpu.POINTS},
		{"dim:30 30", pdfcpu.POINTS},
		{"dim:50% 50%", pdfcpu.POINTS},
		{"pos:bl, dim:50% 50%", pdfcpu.POINTS},
		{"pos:tl, off: 10 -10, dim:50% 50%", pdfcpu.POINTS},
		{"-1", pdfcpu.INCHES},
		{"-25%", pdfcpu.POINTS},
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
		u pdfcpu.DisplayUnit
	}{
		{"art:10%", pdfcpu.POINTS}, // When using relative positioning unit is irrelevant.
		{"trim:10", pdfcpu.POINTS},
		{"crop:[0 0 5 5]", pdfcpu.CENTIMETRES}, // Crop 5 x 5 cm at bottom left corner
		{"crop:10", pdfcpu.POINTS},
		{"crop:-10", pdfcpu.POINTS},
		{"crop:10 20, trim:crop, art:bleed, bleed:art", pdfcpu.POINTS},
		{"crop:10 20, trim:crop, art:bleed, bleed:media", pdfcpu.POINTS},
		{"c:10 20, t:c, a:b, b:m", pdfcpu.POINTS},
		{"crop:10, trim:20, art:trim", pdfcpu.POINTS},
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

	pb, err := api.PageBoundaries("crop:[0 0 100 100]", pdfcpu.POINTS)
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
