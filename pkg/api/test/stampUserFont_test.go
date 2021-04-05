/*
Copyright 2021 The pdfcpu Authors.

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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func TestStampUserFont(t *testing.T) {
	msg := "TestStampUserFont"
	inFile := filepath.Join(inDir, "mountain.pdf")
	outDir := filepath.Join("..", "..", "samples", "stamp", "text", "utf8")

	api.LoadConfiguration()
	if err := api.InstallFonts(userFonts(t, filepath.Join("..", "..", "testdata", "fonts"))); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	for _, sample := range langSamples {
		outFile := filepath.Join(outDir, sample.lang+".pdf")
		align, rtl := "l", "off"
		if sample.rtl {
			align, rtl = "r", "on"
		}
		desc := fmt.Sprintf("font:%s, rtl:%s, align:%s, scale:1.0 rel, rot:0, fillc:#000000, bgcol:#ab6f30, margin:10, border:10 round, opacity:.7", sample.fontName, rtl, align)
		err := api.AddTextWatermarksFile(inFile, outFile, nil, true, sample.text, desc, nil)
		if err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
		if err := api.ValidateFile(outFile, nil); err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
	}
}
