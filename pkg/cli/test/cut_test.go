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

func testCut(t *testing.T, msg, inFile, outFile, cutConf string) {
	t.Helper()

	cut, err := pdfcpu.ParseCutConfig(cutConf, types.POINTS)
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
		msg     string
		inFile  string
		outFile string
		cutConf string
	}{
		{"TestRotatedCutHor",
			"testRot.pdf",
			"cutHorRot",
			"hor:.5"},

		{"TestCutHor",
			"test.pdf",
			"cutHor",
			"hor:.5"},

		{"TestCutVer",
			"test.pdf",
			"cutVer",
			"ver:.5"},

		{"TestCutHorAndVerQuarter",
			"test.pdf",
			"cutHorAndVerQuarter",
			"h:.5, v:.5"},

		{"TestCutHorAndVerThird",
			"test.pdf",
			"cutHorAndVerThird",
			"h:.33, h:.66, v:.33, v:.66"},
	} {
		testCut(t, tt.msg, tt.inFile, tt.outFile, tt.cutConf)
	}
}

func testNDown(t *testing.T, msg, inFile, outFile string, n int, cutConf string) {
	t.Helper()

	cut, err := pdfcpu.ParseCutConfigForN(n, cutConf, types.POINTS)
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
		cutConf string
	}{
		{"TestNDownRot2",
			"testRot.pdf",
			"ndownRot2",
			2,
			""}, // optional border, margin, bgcolor

		{"TestNDown2",
			"test.pdf",
			"ndown2",
			2,
			""}, // optional border, margin, bgcolor

		{"TestNDown9",
			"test.pdf",
			"ndown9",
			9,
			""}, // optional border, margin, bgcolor

		{"TestNDown16",
			"test.pdf",
			"ndown16",
			16,
			""}, // optional border, margin, bgcolor
	} {
		testNDown(t, tt.msg, tt.inFile, tt.outFile, tt.n, tt.cutConf)
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
			types.POINTS,
			"f:A6, scale:2.0"},

		{"TestPosterDim", // grid made up of 15x10 cm tiles => A4
			"test.pdf", // A4
			"posterDim",
			types.CENTIMETRES,
			"dim:15 10"},

		{"TestPosterDimScaled", // grid made up of 15x10 cm tiles => A2
			"test.pdf", // A4
			"posterDimScaled",
			types.CENTIMETRES,
			"dim:15 10, scale:2.0"},
	} {
		testPoster(t, tt.msg, tt.inFile, tt.outFile, tt.unit, tt.cutConf)
	}
}
