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

	"github.com/pdfcpu/pdfcpu/pkg/font"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func writeCoreFontDemoContent(p pdf.Page, fontName string) {
	baseFontName := "Helvetica"
	baseFontSize := 24
	baseFontKey := p.Fm.EnsureKey(baseFontName)

	fontKey := p.Fm.EnsureKey(fontName)
	fontSize := 24

	fillCol := pdf.NewSimpleColor(0xf7e6c7)
	pdf.DrawGrid(p.Buf, 16, 14, pdf.RectForWidthAndHeight(55, 2, 480, 422), pdf.Black, &fillCol)

	td := pdf.TextDescriptor{
		FontName:       baseFontName,
		FontKey:        baseFontKey,
		FontSize:       baseFontSize,
		HAlign:         pdf.AlignCenter,
		VAlign:         pdf.AlignBaseline,
		Scale:          1.0,
		ScaleAbs:       true,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.Black,
		FillCol:        pdf.NewSimpleColor(0xab6f30),
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: 1., G: .98, B: .77},
	}

	s := fmt.Sprintf("%s %d points", fontName, fontSize)
	if fontName != "ZapfDingbats" && fontName != "Symbol" {
		s = s + " (CP1252)"
	}
	td.X, td.Y, td.Text = p.MediaBox.Width()/2, 500, s
	td.StrokeCol, td.FillCol = pdf.NewSimpleColor(0x77bdbd), pdf.NewSimpleColor(0xab6f30)
	pdf.WriteMultiLine(p.Buf, p.MediaBox, nil, td)

	for i := 0; i < 16; i++ {
		s = fmt.Sprintf("#%02X", i)
		td.X, td.Y, td.Text, td.FontSize = float64(70+i*30), 427, s, 14
		td.StrokeCol, td.FillCol = pdf.Black, pdf.SimpleColor{B: .8}
		pdf.WriteMultiLine(p.Buf, p.MediaBox, nil, td)
	}

	for j := 0; j < 14; j++ {
		s = fmt.Sprintf("#%02X", j*16+32)
		td.X, td.Y, td.Text = 41, float64(403-j*30), s
		td.StrokeCol, td.FillCol = pdf.Black, pdf.SimpleColor{B: .8}
		td.FontName, td.FontKey, td.FontSize = baseFontName, baseFontKey, 14
		pdf.WriteMultiLine(p.Buf, p.MediaBox, nil, td)
		for i := 0; i < 16; i++ {
			b := byte(32 + j*16 + i)
			s = string([]byte{b})
			td.X, td.Y, td.Text = float64(70+i*30), float64(400-j*30), s
			td.StrokeCol, td.FillCol = pdf.Black, pdf.Black
			td.FontName, td.FontKey, td.FontSize = fontName, fontKey, fontSize
			pdf.WriteMultiLine(p.Buf, p.MediaBox, nil, td)
		}
	}
}

func writeCP1252SpecialMappings(p pdf.Page, fontName string) {

	k := p.Fm.EnsureKey(fontName)

	td := pdf.TextDescriptor{
		Text:           "€‚ƒ„…†‡ˆ‰Š‹ŒŽ‘’“”•–—˜™š›œžŸ",
		FontName:       fontName,
		FontKey:        k,
		FontSize:       24,
		HAlign:         pdf.AlignCenter,
		VAlign:         pdf.AlignBaseline,
		X:              -1,
		Y:              550,
		Scale:          1.0,
		ScaleAbs:       true,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.SimpleColor{},
		FillCol:        pdf.NewSimpleColor(0xab6f30),
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: 1., G: .98, B: .77},
	}

	pdf.WriteMultiLine(p.Buf, p.MediaBox, nil, td)
}

func createCoreFontDemoPage(w, h int, fontName string) pdf.Page {
	mediaBox := pdf.RectForDim(float64(w), float64(h))
	p := pdf.NewPageWithBg(mediaBox, pdf.NewSimpleColor(0xbeded9))
	writeCoreFontDemoContent(p, fontName)
	//writeCP1252SpecialMappings(p, fontName)
	return p
}
func TestCoreFontDemoPDF(t *testing.T) {
	msg := "TestCoreFontDemoPDF"
	w, h := 600, 600
	for _, fn := range font.CoreFontNames() {
		p := createCoreFontDemoPage(w, h, fn)
		xRefTable, err := pdf.CreateDemoXRef(p)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		outFile := filepath.Join("../../samples/fonts/core", fn+".pdf")
		createAndValidate(t, xRefTable, outFile, msg)
	}
}
