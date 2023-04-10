/*
Copyright 2023 The pdf Authors.

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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func testCut(t *testing.T, msg, inFile, outDir, outFile string, unit types.DisplayUnit, cutConf string) {
	t.Helper()

	cut, err := pdfcpu.ParseCutConfig(cutConf, unit)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	inFile = filepath.Join(inDir, inFile)
	outDir = filepath.Join(samplesDir, outDir)

	if err := api.CutFile(inFile, outDir, outFile, nil, cut, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestCut(t *testing.T) {
	for _, tt := range []struct {
		msg             string
		inFile          string
		outDir, outFile string
		unit            types.DisplayUnit
		cutConf         string
	}{
		{"TestRotatedCutHor",
			"testRot.pdf",
			"cut",
			"cutHorRot",
			types.CENTIMETRES,
			"hor:.5, margin:1, border:on"},

		{"TestCutHor",
			"test.pdf",
			"cut",
			"cutHor",
			types.CENTIMETRES,
			"hor:.5, margin:1, bgcol:#E9967A, border:on"},

		{"TestCutVer",
			"test.pdf",
			"cut",
			"cutVer",
			types.CENTIMETRES,
			"ver:.5, margin:1, bgcol:#E9967A"},

		{"TestCutHorAndVerQuarter",
			"test.pdf",
			"cut",
			"cutHorAndVerQuarter",
			types.POINTS,
			"h:.5, v:.5"},

		{"TestCutHorAndVerThird",
			"test.pdf",
			"cut",
			"cutHorAndVerThird",
			types.POINTS,
			"h:.33333, h:.66666, v:.33333, v:.66666"},

		{"Test",
			"test.pdf",
			"cut",
			"cutCustom",
			types.POINTS,
			"h:.25, v:.5"},
	} {
		testCut(t, tt.msg, tt.inFile, tt.outDir, tt.outFile, tt.unit, tt.cutConf)
	}
}

func testNDown(t *testing.T, msg, inFile, outDir, outFile string, n int, unit types.DisplayUnit, cutConf string) {
	t.Helper()

	cut, err := pdfcpu.ParseCutConfigForN(n, cutConf, unit)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	inFile = filepath.Join(inDir, inFile)
	outDir = filepath.Join(samplesDir, outDir)

	if err := api.NDownFile(inFile, outDir, outFile, nil, n, cut, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestNDown(t *testing.T) {
	for _, tt := range []struct {
		msg             string
		inFile          string
		outDir, outFile string
		n               int
		unit            types.DisplayUnit
		cutConf         string
	}{
		{"TestNDownRot2",
			"testRot.pdf",
			"cut",
			"ndownRot2",
			2,
			types.CENTIMETRES,
			"margin:1, bgcol:#E9967A"},

		{"TestNDown2",
			"test.pdf",
			"cut",
			"ndown2",
			2,
			types.CENTIMETRES,
			"margin:1, border:on"},

		{"TestNDown9",
			"test.pdf",
			"cut",
			"ndown9",
			9,
			types.CENTIMETRES,
			"margin:1, bgcol:#E9967A, border:on"},

		{"TestNDown16",
			"test.pdf",
			"cut",
			"ndown16",
			16,
			types.CENTIMETRES,
			""}, // optional border, margin, bgcolor
	} {
		testNDown(t, tt.msg, tt.inFile, tt.outDir, tt.outFile, tt.n, tt.unit, tt.cutConf)
	}
}

func testPoster(t *testing.T, msg, inFile, outDir, outFile string, unit types.DisplayUnit, cutConf string) {
	t.Helper()

	cut, err := pdfcpu.ParseCutConfigForPoster(cutConf, unit)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	inFile = filepath.Join(inDir, inFile)
	outDir = filepath.Join(samplesDir, outDir)

	if err := api.PosterFile(inFile, outDir, outFile, nil, cut, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestPoster(t *testing.T) {
	for _, tt := range []struct {
		msg             string
		inFile          string
		outDir, outFile string
		unit            types.DisplayUnit
		cutConf         string
	}{
		{"TestPoster", // 2x2 grid of A6 => A4
			"test.pdf", // A4
			"cut",
			"poster",
			types.POINTS,
			"f:A6"},

		{"TestPosterScaled", // 4x4 grid of A6 => A2
			"test.pdf", // A4
			"cut",
			"posterScaled",
			types.CENTIMETRES,
			"f:A6, scale:2.0, margin:1, bgcol:#E9967A"},

		{"TestPosterDim", // grid made up of 15x10 cm tiles => A4
			"test.pdf", // A4
			"cut",
			"posterDim",
			types.CENTIMETRES,
			"dim:15 10, margin:1, border:on"},

		{"TestPosterDimScaled", // grid made up of 15x10 cm tiles => A2
			"test.pdf", // A4
			"cut",
			"posterDimScaled",
			types.CENTIMETRES,
			"dim:15 10, scale:2.0, margin:1, bgcol:#E9967A, border:on"},
	} {
		testPoster(t, tt.msg, tt.inFile, tt.outDir, tt.outFile, tt.unit, tt.cutConf)
	}
}
