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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func writeCoreFontDemoContent(xRefTable *model.XRefTable, p model.Page, fontName string) {
	baseFontName := "Helvetica"
	baseFontSize := 24
	baseFontKey := p.Fm.EnsureKey(baseFontName)

	fontKey := p.Fm.EnsureKey(fontName)
	fontSize := 24

	fillCol := color.NewSimpleColor(0xf7e6c7)
	draw.DrawGrid(p.Buf, 16, 14, types.RectForWidthAndHeight(55, 2, 480, 422), color.Black, &fillCol)

	td := model.TextDescriptor{
		FontName:       baseFontName,
		FontKey:        baseFontKey,
		FontSize:       baseFontSize,
		HAlign:         types.AlignCenter,
		VAlign:         types.AlignBaseline,
		Scale:          1.0,
		ScaleAbs:       true,
		RMode:          draw.RMFill,
		StrokeCol:      color.Black,
		FillCol:        color.NewSimpleColor(0xab6f30),
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: 1., G: .98, B: .77},
	}

	s := fmt.Sprintf("%s %d points", fontName, fontSize)
	if fontName != "ZapfDingbats" && fontName != "Symbol" {
		s = s + " (CP1252)"
	}
	td.X, td.Y, td.Text = p.MediaBox.Width()/2, 500, s
	td.StrokeCol, td.FillCol = color.NewSimpleColor(0x77bdbd), color.NewSimpleColor(0xab6f30)
	model.WriteMultiLine(xRefTable, p.Buf, p.MediaBox, nil, td)

	for i := 0; i < 16; i++ {
		s = fmt.Sprintf("#%02X", i)
		td.X, td.Y, td.Text, td.FontSize = float64(70+i*30), 427, s, 14
		td.StrokeCol, td.FillCol = color.Black, color.SimpleColor{B: .8}
		model.WriteMultiLine(xRefTable, p.Buf, p.MediaBox, nil, td)
	}

	for j := 0; j < 14; j++ {
		s = fmt.Sprintf("#%02X", j*16+32)
		td.X, td.Y, td.Text = 41, float64(403-j*30), s
		td.StrokeCol, td.FillCol = color.Black, color.SimpleColor{B: .8}
		td.FontName, td.FontKey, td.FontSize = baseFontName, baseFontKey, 14
		model.WriteMultiLine(xRefTable, p.Buf, p.MediaBox, nil, td)
		for i := 0; i < 16; i++ {
			b := byte(32 + j*16 + i)
			s = string([]byte{b})
			td.X, td.Y, td.Text = float64(70+i*30), float64(400-j*30), s
			td.StrokeCol, td.FillCol = color.Black, color.Black
			td.FontName, td.FontKey, td.FontSize = fontName, fontKey, fontSize
			model.WriteMultiLine(xRefTable, p.Buf, p.MediaBox, nil, td)
		}
	}
}

func createCoreFontDemoPage(xRefTable *model.XRefTable, w, h int, fontName string) model.Page {
	mediaBox := types.RectForDim(float64(w), float64(h))
	p := model.NewPageWithBg(mediaBox, color.NewSimpleColor(0xbeded9))
	writeCoreFontDemoContent(xRefTable, p, fontName)
	return p
}
func TestCoreFontDemoPDF(t *testing.T) {
	msg := "TestCoreFontDemoPDF"
	w, h := 600, 600
	for _, fn := range font.CoreFontNames() {
		xRefTable, err := pdfcpu.CreateDemoXRef()
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		rootDict, err := xRefTable.Catalog()
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		p := createCoreFontDemoPage(xRefTable, w, h, fn)
		if err = pdfcpu.AddPageTreeWithSamplePage(xRefTable, rootDict, p); err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		outFile := filepath.Join("..", "..", "samples", "fonts", "core", fn+".pdf")
		createAndValidate(t, xRefTable, outFile, msg)
	}
}

func TestUserFontDemoPDF(t *testing.T) {
	msg := "TestUserFontDemoPDF"

	// For each installed user font create a single page pdf cheat sheet for every unicode plane covered
	// in pkg/samples/fonts/user.
	for _, fn := range font.UserFontNames() {
		fmt.Println(fn)
		if err := api.CreateUserFontDemoFiles(filepath.Join("..", "..", "samples", "fonts", "user"), fn); err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
	}
}
