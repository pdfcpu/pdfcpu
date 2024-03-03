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
	outFile := filepath.Join(samplesDir, "zoom", "zoomInByFactor2.pdf")
	if err := api.ZoomFile(inFile, outFile, nil, zoom, nil); err != nil {
		t.Fatalf("%s zoom: %v\n", msg, err)
	}

	zoom, err = pdfcpu.ParseZoomConfig("factor:4", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid zoom configuration: %v\n", msg, err)
	}
	outFile = filepath.Join(samplesDir, "zoom", "zoomInByFactor4.pdf")
	if err := api.ZoomFile(inFile, outFile, nil, zoom, nil); err != nil {
		t.Fatalf("%s zoom: %v\n", msg, err)
	}
}

func TestZoomOutByFactor(t *testing.T) {
	msg := "TestZoomOutByFactor"

	inFile := filepath.Join(inDir, "test.pdf")

	zoom, err := pdfcpu.ParseZoomConfig("factor:.5", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid zoom configuration: %v\n", msg, err)
	}
	outFile := filepath.Join(samplesDir, "zoom", "zoomOutByFactor05.pdf")
	if err := api.ZoomFile(inFile, outFile, nil, zoom, nil); err != nil {
		t.Fatalf("%s zoom: %v\n", msg, err)
	}

	zoom, err = pdfcpu.ParseZoomConfig("factor:.25, border:true", types.POINTS)
	if err != nil {
		t.Fatalf("%s invalid zoom configuration: %v\n", msg, err)
	}
	outFile = filepath.Join(samplesDir, "zoom", "zoomOutByFactor025.pdf")
	if err := api.ZoomFile(inFile, outFile, nil, zoom, nil); err != nil {
		t.Fatalf("%s zoom: %v\n", msg, err)
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
	outFile := filepath.Join(samplesDir, "zoom", "zoomOutByHMarginPoints.pdf")
	if err := api.ZoomFile(inFile, outFile, nil, zoom, nil); err != nil {
		t.Fatalf("%s zoom: %v\n", msg, err)
	}

	zoom, err = pdfcpu.ParseZoomConfig("hmargin:1, border:true, bgcol:lightgray", types.CENTIMETRES)
	if err != nil {
		t.Fatalf("%s invalid zoom configuration: %v\n", msg, err)
	}
	outFile = filepath.Join(samplesDir, "zoom", "zoomOutByHMarginCm.pdf")
	if err := api.ZoomFile(inFile, outFile, nil, zoom, nil); err != nil {
		t.Fatalf("%s zoom: %v\n", msg, err)
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
	outFile := filepath.Join(samplesDir, "zoom", "zoomOutByVMarginInches.pdf")
	if err := api.ZoomFile(inFile, outFile, nil, zoom, nil); err != nil {
		t.Fatalf("%s zoom: %v\n", msg, err)
	}

	zoom, err = pdfcpu.ParseZoomConfig("vmargin:30, border:false, bgcol:lightgray", types.MILLIMETRES)
	if err != nil {
		t.Fatalf("%s invalid zoom configuration: %v\n", msg, err)
	}
	outFile = filepath.Join(samplesDir, "zoom", "zoomOutByVMarginMm.pdf")
	if err := api.ZoomFile(inFile, outFile, nil, zoom, nil); err != nil {
		t.Fatalf("%s zoom: %v\n", msg, err)
	}
}
