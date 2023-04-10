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

func TestResizeByScaleFactor(t *testing.T) {
	msg := "TestResizeByScaleFactor"
	inFile := filepath.Join(inDir, "test.pdf")

	// Enlarge by scale factor 2.
	res, err := pdfcpu.ParseResizeConfig("sc:2", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile := filepath.Join(outDir, "enlargeByScaleFactor.pdf")
	cmd := cli.ResizeCommand(inFile, outFile, nil, res, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Shrink by 50%.
	res, err = pdfcpu.ParseResizeConfig("sc:.5", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile = filepath.Join(outDir, "shrinkByScaleFactor.pdf")
	cmd = cli.ResizeCommand(inFile, outFile, nil, res, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestResizeByWidthOrHeight(t *testing.T) {
	msg := "TestResizeByWidthOrHeight"

	inFile := filepath.Join(inDir, "test.pdf")

	// Set width to 200 points.
	res, err := pdfcpu.ParseResizeConfig("dim:200 0", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile := filepath.Join(outDir, "resizeByWidth.pdf")
	cmd := cli.ResizeCommand(inFile, outFile, nil, res, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Set height to 200 mm.
	res, err = pdfcpu.ParseResizeConfig("dim:0 200", types.MILLIMETRES)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile = filepath.Join(outDir, "resizeByHeight.pdf")
	cmd = cli.ResizeCommand(inFile, outFile, nil, res, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestResizeToFormSize(t *testing.T) {
	msg := "TestResizeToPaperSize"

	inFile := filepath.Join(inDir, "test.pdf")

	// Resize to A3 and keep orientation.
	res, err := pdfcpu.ParseResizeConfig("form:A3", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile := filepath.Join(outDir, "resizeToA3.pdf")
	cmd := cli.ResizeCommand(inFile, outFile, nil, res, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Resize to A4 and enforce orientation (here landscape mode).
	res, err = pdfcpu.ParseResizeConfig("form:A4L", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile = filepath.Join(outDir, "resizeToA4L.pdf")
	cmd = cli.ResizeCommand(inFile, outFile, nil, res, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestResizeToDimensions(t *testing.T) {
	msg := "TestResizeToDimensions"

	inFile := filepath.Join(inDir, "test.pdf")

	// Resize to 400 x 200 and keep orientation of input file.
	// Apply background color to unused space.
	res, err := pdfcpu.ParseResizeConfig("dim:400 200, bgcol:#E9967A", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile := filepath.Join(outDir, "resizeToDimensionsKeep.pdf")
	cmd := cli.ResizeCommand(inFile, outFile, nil, res, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Resize to 400 x 200 and enforce new orientation.
	// Render border of original crop box.
	res, err = pdfcpu.ParseResizeConfig("dim:400 200, enforce:true, border:on", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid resize configuration: %v\n", msg, err)
	}

	outFile = filepath.Join(outDir, "resizeToDimensionsEnforce.pdf")
	cmd = cli.ResizeCommand(inFile, outFile, nil, res, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}
