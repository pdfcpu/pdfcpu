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

func testCut(t *testing.T, msg, inFile, outDir, outFile, cutConf string) {
	t.Helper()

	cut, err := pdfcpu.ParseCutConfig(cutConf, types.POINTS)
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
		cutConf         string
	}{
		{"TestRotatedCutHor",
			"testRot.pdf",
			"cut",
			"cutHorRot",
			"hor:.5"},

		{"TestCutHor",
			"test.pdf",
			"cut",
			"cutHor",
			"hor:.5"},

		{"TestCutVer",
			"test.pdf",
			"cut",
			"cutVer",
			"ver:.5"},

		{"TestCutHorAndVerQuarter",
			"test.pdf",
			"cut",
			"cutHorAndVerQuarter",
			"h:.5, v:.5"},

		{"TestCutHorAndVerThird",
			"test.pdf",
			"cut",
			"cutHorAndVerThird",
			"h:.33, h:.66, v:.33, v:.66"},
	} {
		testCut(t, tt.msg, tt.inFile, tt.outDir, tt.outFile, tt.cutConf)
	}
}

func testNDown(t *testing.T, msg, inFile, outDir, outFile string, n int, cutConf string) {
	t.Helper()

	cut, err := pdfcpu.ParseCutConfigForN(n, cutConf, types.POINTS)
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
		cutConf         string
	}{
		{"TestNDownRot2",
			"testRot.pdf",
			"cut",
			"ndownRot2",
			2,
			""}, // optional border, margin, bgcolor

		{"TestNDown2",
			"test.pdf",
			"cut",
			"ndown2",
			2,
			""}, // optional border, margin, bgcolor

		{"TestNDown9",
			"test.pdf",
			"cut",
			"ndown9",
			9,
			""}, // optional border, margin, bgcolor

		{"TestNDown16",
			"test.pdf",
			"cut",
			"ndown16",
			16,
			""}, // optional border, margin, bgcolor
	} {
		testNDown(t, tt.msg, tt.inFile, tt.outDir, tt.outFile, tt.n, tt.cutConf)
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
			types.POINTS,
			"f:A6, scale:2.0"},

		{"TestPosterDim", // grid made up of 15x10 cm tiles => A4
			"test.pdf", // A4
			"cut",
			"posterDim",
			types.CENTIMETRES,
			"dim:15 10"},

		{"TestPosterDimScaled", // grid made up of 15x10 cm tiles => A2
			"test.pdf", // A4
			"cut",
			"posterDimScaled",
			types.CENTIMETRES,
			"dim:15 10, scale:2.0"},
	} {
		testPoster(t, tt.msg, tt.inFile, tt.outDir, tt.outFile, tt.unit, tt.cutConf)
	}
}
