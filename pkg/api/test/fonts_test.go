/*
Copyright 2020 The pdf Authors.

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
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func createFontSample(fontName string, t *testing.T) {
	t.Helper()
	msg := "createFontSampleSpecial"
	inFile := filepath.Join(inDir, "empty.pdf")
	outFile := filepath.Join(inDir, "fontSamples", fontName+".pdf")

	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s: ReadContextFile %s: %v\n", msg, inFile, err)
	}

	wm, err := pdf.ParseTextWatermarkDetails(fontName+" (CP1252)", "rot:0, scal:0.8 abs, pos:tl", true)
	if err != nil {
		t.Fatalf("%s: ParseTextWatermarkDetails: %v\n", msg, err)
	}
	if err := api.WatermarkContext(ctx, nil, wm); err != nil {
		t.Fatalf("%s: WatermarkContext: %v\n", msg, err)
	}

	var sb strings.Builder
	for i := 32; i <= 255; i++ {
		if i%8 == 0 {
			sb.WriteString(fmt.Sprintf("\n%3d (%03o): ", i, i))
		}
	}
	a := sb.String()
	wm, err = pdf.ParseTextWatermarkDetails(a, "rot:0, scal:0.8 abs, pos:tl, off:5 -40", true)
	if err != nil {
		t.Fatalf("%s: ParseTextWatermarkDetails: %v\n", msg, err)
	}
	if err := api.WatermarkContext(ctx, nil, wm); err != nil {
		t.Fatalf("%s: WatermarkContext: %v\n", msg, err)
	}

	sb.Reset()
	x := 100
	y := -38
	if fontName == "ZapfDingbats" {
		y = -45
	}
	for i := 32; i <= 256; i++ {
		if i > 32 && i%8 == 0 {
			desc := fmt.Sprintf("font:%s, rot:0, scal:0.8 abs, pos:tl, off:%d %d", fontName, x, y)
			wm, err = pdf.ParseTextWatermarkDetails(sb.String(), desc, true)
			if err != nil {
				t.Fatalf("%s: ParseTextWatermarkDetails: %v\n", msg, err)
			}
			if err := api.WatermarkContext(ctx, nil, wm); err != nil {
				t.Fatalf("%s: WatermarkContext: %v\n", msg, err)
			}
			y -= 19
			sb.Reset()
		}
		sb.WriteRune(rune(i))
		sb.WriteString(" ")
	}

	if err := api.WriteContextFile(ctx, outFile); err != nil {
		t.Fatalf("%s: WriteContextFile %s: %v\n", msg, outFile, err)
	}
}

func TestCreateFontSamples(t *testing.T) {
	l, err := api.ListFonts()
	if err != nil {
		t.Fatalf("%v", err)
	}
	for _, fn := range l {
		createFontSample(strings.Fields(fn)[0], t)
	}
}

func XXXTestFontTestPage(t *testing.T) {

	// Generate a sample page for an Adobe Type 1 standard font.
	inFile := filepath.Join(inDir, "empty.pdf")

	var sb strings.Builder
	for i := 32; i <= 255; i++ {
		if i%8 == 0 {
			sb.WriteString(fmt.Sprintf("\n%03o ", i))
		}
		sb.WriteRune(rune(i))
		sb.WriteString(" ")
	}

	a := sb.String()

	wmConf := "font:Symbol, rot:0, scal:0.8 abs, pos:tl"
	wm, err := pdf.ParseTextWatermarkDetails("Symbol\n\n"+a, wmConf, true)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if err = api.AddWatermarksFile(inFile, filepath.Join(inDir, "Symbol.pdf"), []string{}, wm, nil); err != nil {
		t.Fatalf("%v\n", err)
	}

}
