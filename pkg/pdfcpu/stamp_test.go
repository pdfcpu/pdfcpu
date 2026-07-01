/*
Copyright 2026 The pdfcpu Authors.

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

package pdfcpu

import (
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestPDFResourcePageCount(t *testing.T) {
	for _, tt := range []struct {
		name                                     string
		destPages, srcPages, startSrc, startDest int
		want                                     int
	}{
		{"source offset exceeds destination count", 2, 5, 3, 1, 2},
		{"destination offset", 5, 2, 1, 3, 2},
		{"both offsets", 5, 7, 3, 3, 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pdfResourcePageCount(tt.destPages, tt.srcPages, tt.startSrc, tt.startDest)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %d pages, want %d", got, tt.want)
			}
		})
	}
}

func TestPDFResourcePageCountRejectsInvalidStartPage(t *testing.T) {
	for _, tt := range []struct {
		name                                     string
		destPages, srcPages, startSrc, startDest int
	}{
		{"source page zero", 1, 1, 0, 1},
		{"source page too high", 1, 1, 2, 1},
		{"destination page zero", 1, 1, 1, 0},
		{"destination page too high", 1, 1, 1, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := pdfResourcePageCount(tt.destPages, tt.srcPages, tt.startSrc, tt.startDest); err == nil {
				t.Fatal("expected an invalid start page error")
			}
		})
	}
}

func TestPDFResourceForPage(t *testing.T) {
	wm := model.DefaultWatermarkConfig()
	wm.Mode = model.WMPDF
	wm.PdfRes = map[int]model.PdfResources{}

	if _, err := pdfResourceForPage(wm, 1); err == nil {
		t.Fatal("expected a missing resource error")
	}

	wm.PdfRes[1] = model.PdfResources{}
	if _, err := pdfResourceForPage(wm, 1); err == nil {
		t.Fatal("expected a missing bounding box error")
	}

	want := types.RectForDim(100, 200)
	wm.PdfRes[1] = model.PdfResources{Bb: want}
	got, err := pdfResourceForPage(wm, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bb != want {
		t.Fatalf("got bounding box %v, want %v", got.Bb, want)
	}
}
