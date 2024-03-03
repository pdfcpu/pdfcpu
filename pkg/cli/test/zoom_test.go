/*
Copyright 2024 The pdfcpu Authors.

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

func TestZoomInByFactor(t *testing.T) {
	msg := "TestZoomInByFactor"

	inFile := filepath.Join(inDir, "test.pdf")

	zoom, err := pdfcpu.ParseZoomConfig("factor:2", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid zoom configuration: %v\n", msg, err)
	}
	outFile := filepath.Join(outDir, "zoomInByFactor2.pdf")
	cmd := cli.ZoomCommand(inFile, outFile, nil, zoom, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	zoom, err = pdfcpu.ParseZoomConfig("factor:4", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid zoom configuration: %v\n", msg, err)
	}
	outFile = filepath.Join(outDir, "zoomInByFactor4.pdf")
	cmd = cli.ZoomCommand(inFile, outFile, nil, zoom, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestZoomOutByFactor(t *testing.T) {
	msg := "TestZoomOutByFactor"

	inFile := filepath.Join(inDir, "test.pdf")

	zoom, err := pdfcpu.ParseZoomConfig("factor:.5", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid zoom configuration: %v\n", msg, err)
	}
	outFile := filepath.Join(outDir, "zoomOutByFactor05.pdf")
	cmd := cli.ZoomCommand(inFile, outFile, nil, zoom, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	zoom, err = pdfcpu.ParseZoomConfig("factor:.25, border:true", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid zoom configuration: %v\n", msg, err)
	}
	outFile = filepath.Join(outDir, "zoomOutByFactor025.pdf")
	cmd = cli.ZoomCommand(inFile, outFile, nil, zoom, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestZoomOutByHorizontalMargin(t *testing.T) {
	// Zoom out of page content resulting in a preferred horizontal margin.
	msg := "TestZoomOutByHMargin"
	inFile := filepath.Join(inDir, "test.pdf")

	zoom, err := pdfcpu.ParseZoomConfig("hmargin:149", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid zoom configuration: %v\n", msg, err)
	}
	outFile := filepath.Join(outDir, "zoomOutByHMarginPoints.pdf")
	cmd := cli.ZoomCommand(inFile, outFile, nil, zoom, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	zoom, err = pdfcpu.ParseZoomConfig("hmargin:1, border:true, bgcol:lightgray", types.CENTIMETRES)
	if err != nil {
		t.Fatalf("%s invalid zoom configuration: %v\n", msg, err)
	}
	outFile = filepath.Join(outDir, "zoomOutByHMarginCm.pdf")
	cmd = cli.ZoomCommand(inFile, outFile, nil, zoom, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestZoomOutByVerticalMargin(t *testing.T) {
	// Zoom out of page content resulting in a preferred vertical margin.
	msg := "TestZoomOutByVMargin"
	inFile := filepath.Join(inDir, "test.pdf")

	zoom, err := pdfcpu.ParseZoomConfig("vmargin:1", types.INCHES)
	if err != nil {
		t.Fatalf("%s invalid zoom configuration: %v\n", msg, err)
	}
	outFile := filepath.Join(outDir, "zoomOutByVMarginInches.pdf")
	cmd := cli.ZoomCommand(inFile, outFile, nil, zoom, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	zoom, err = pdfcpu.ParseZoomConfig("vmargin:30, border:false, bgcol:lightgray", types.MILLIMETRES)
	if err != nil {
		t.Fatalf("%s invalid zoom configuration: %v\n", msg, err)
	}
	outFile = filepath.Join(outDir, "zoomOutByVMarginMm.pdf")
	cmd = cli.ZoomCommand(inFile, outFile, nil, zoom, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}
