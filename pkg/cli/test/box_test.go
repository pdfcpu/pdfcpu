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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestListBoxesCommand(t *testing.T) {
	msg := "TestListBoxesCommand"
	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")

	// List all page boundaries for all pages.
	cmd := cli.ListBoxesCommand(inFile, nil, nil, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// List crop box for all pages.
	pb, err := api.PageBoundariesFromBoxList("crop")
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	cmd.PageBoundaries = pb
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestCropCommand(t *testing.T) {
	msg := "TestCropCommand"
	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(outDir, "out.pdf")

	for _, tt := range []struct {
		s string
		u types.DisplayUnit
	}{
		{"0 0 10 0", types.POINTS},
		{"[0 0 5 5]", types.CENTIMETRES},
		{"100", types.POINTS},
		{"20% 40%", types.POINTS},
		{"dim:30 30", types.POINTS},
		{"dim:50% 50%", types.POINTS},
		{"pos:bl, dim:50% 50%", types.POINTS},
		{"pos:tl, off: 10 -10, dim:50% 50%", types.POINTS},
		{"-1", types.INCHES},
		{"-25%", types.POINTS},
	} {
		box, err := api.Box(tt.s, tt.u)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}

		cmd := cli.CropCommand(inFile, outFile, nil, box, conf)
		if _, err := cli.Process(cmd); err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}

	}
}

func TestAddBoxesCommand(t *testing.T) {
	msg := "TestAddBoxesCommand"
	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(outDir, "out.pdf")

	for _, tt := range []struct {
		s string
		u types.DisplayUnit
	}{
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

		cmd := cli.AddBoxesCommand(inFile, outFile, nil, pb, conf)
		if _, err := cli.Process(cmd); err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
	}
}

func TestAddRemoveBoxesCommand(t *testing.T) {
	msg := "TestAddRemoveBoxesCommand"
	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(outDir, "out.pdf")

	pb, err := api.PageBoundaries("crop:[0 0 100 100]", types.POINTS)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	cmd := cli.AddBoxesCommand(inFile, outFile, nil, pb, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	pb, err = api.PageBoundariesFromBoxList("crop")
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	cmd = cli.RemoveBoxesCommand(inFile, outFile, nil, pb, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}
