/*
Copyright 2023 The pdfcpu Authors.

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

	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func testCut(t *testing.T, msg, inFile, outFile string, unit types.DisplayUnit, cutConf string) {
	t.Helper()

	cut, err := pdfcpu.ParseCutConfig(cutConf, unit)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	inFile = filepath.Join(inDir, inFile)

	cmd := cli.CutCommand(inFile, outDir, outFile, nil, cut, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestCut(t *testing.T) {
	for _, tt := range []struct {
		msg             string
		inFile, outFile string
		unit            types.DisplayUnit
		cutConf         string
	}{
		{"TestRotatedCutHor",
			"testRot.pdf",
			"cutHorRot",
			types.CENTIMETRES,
			"hor:.5, margin:1, border:on"},

		{"TestCutHor",
			"test.pdf",
			"cutHor",
			types.CENTIMETRES,
			"hor:.5, margin:1, bgcol:#E9967A, border:on"},

		{"TestCutVer",
			"test.pdf",
			"cutVer",
			types.CENTIMETRES,
			"ver:.5, margin:1, bgcol:#E9967A"},

		{"TestCutHorAndVerQuarter",
			"test.pdf",
			"cutHorAndVerQuarter",
			types.POINTS,
			"h:.5, v:.5"},

		{"TestCutHorAndVerThird",
			"test.pdf",
			"cutHorAndVerThird",
			types.POINTS,
			"h:.33333, h:.66666, v:.33333, v:.66666"},

		{"Test",
			"test.pdf",
			"cutCustom",
			types.POINTS,
			"h:.25, v:.5"},
	} {
		testCut(t, tt.msg, tt.inFile, tt.outFile, tt.unit, tt.cutConf)
	}
}

func testNDown(t *testing.T, msg, inFile, outFile string, n int, unit types.DisplayUnit, cutConf string) {
	t.Helper()

	cut, err := pdfcpu.ParseCutConfigForN(n, cutConf, unit)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	inFile = filepath.Join(inDir, inFile)

	cmd := cli.NDownCommand(inFile, outDir, outFile, nil, n, cut, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestNDown(t *testing.T) {
	for _, tt := range []struct {
		msg     string
		inFile  string
		outFile string
		n       int
		unit    types.DisplayUnit
		cutConf string
	}{
		{"TestNDownRot2",
			"testRot.pdf",
			"ndownRot2",
			2,
			types.CENTIMETRES,
			"margin:1, bgcol:#E9967A"},

		{"TestNDown2",
			"test.pdf",
			"ndown2",
			2,
			types.CENTIMETRES,
			"margin:1, border:on"},

		{"TestNDown9",
			"test.pdf",
			"ndown9",
			9,
			types.CENTIMETRES,
			"margin:1, bgcol:#E9967A, border:on"},

		{"TestNDown16",
			"test.pdf",
			"ndown16",
			16,
			types.CENTIMETRES,
			""}, // optional border, margin, bgcolor
	} {
		testNDown(t, tt.msg, tt.inFile, tt.outFile, tt.n, tt.unit, tt.cutConf)
	}
}

func testPoster(t *testing.T, msg, inFile, outFile string, unit types.DisplayUnit, cutConf string) {
	t.Helper()

	cut, err := pdfcpu.ParseCutConfigForPoster(cutConf, unit)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	inFile = filepath.Join(inDir, inFile)

	cmd := cli.PosterCommand(inFile, outDir, outFile, nil, cut, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestPoster(t *testing.T) {
	for _, tt := range []struct {
		msg     string
		inFile  string
		outFile string
		unit    types.DisplayUnit
		cutConf string
	}{
		{"TestPoster", // 2x2 grid of A6 => A4
			"test.pdf", // A4
			"poster",
			types.POINTS,
			"f:A6"},

		{"TestPosterScaled", // 4x4 grid of A6 => A2
			"test.pdf", // A4
			"posterScaled",
			types.CENTIMETRES,
			"f:A6, scale:2.0, margin:1, bgcol:#E9967A"},

		{"TestPosterDim", // grid made up of 15x10 cm tiles => A4
			"test.pdf", // A4
			"posterDim",
			types.CENTIMETRES,
			"dim:15 10, margin:1, border:on"},

		{"TestPosterDimScaled", // grid made up of 15x10 cm tiles => A2
			"test.pdf", // A4
			"posterDimScaled",
			types.CENTIMETRES,
			"dim:15 10, scale:2.0, margin:1, bgcol:#E9967A, border:on"},
	} {
		testPoster(t, tt.msg, tt.inFile, tt.outFile, tt.unit, tt.cutConf)
	}
}
