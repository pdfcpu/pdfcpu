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
	"bytes"
	"fmt"
	"path/filepath"
	"testing"
	"unicode/utf8"

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

func rowLabel(i int, td pdf.TextDescriptor, baseFontName, baseFontKey string, buf *bytes.Buffer, mb *pdf.Rectangle, left bool) {
	x := 39.
	if !left {
		x = 7750
	}
	s := fmt.Sprintf("#%02X", i)
	td.X, td.Y, td.Text = x, float64(7677-i*30), s
	td.StrokeCol, td.FillCol = pdf.Black, pdf.SimpleColor{B: .8}
	td.FontName, td.FontKey, td.FontSize = baseFontName, baseFontKey, 14
	pdf.WriteMultiLine(buf, mb, nil, td)
}

func columnsLabel(td pdf.TextDescriptor, baseFontName, baseFontKey string, buf *bytes.Buffer, mb *pdf.Rectangle, top bool) {
	y := 7700.
	if !top {
		y = 0
	}
	td.FontName, td.FontKey = baseFontName, baseFontKey
	for i := 0; i < 256; i++ {
		s := fmt.Sprintf("#%02X", i)
		td.X, td.Y, td.Text, td.FontSize = float64(70+i*30), y, s, 14
		td.StrokeCol, td.FillCol = pdf.Black, pdf.SimpleColor{B: .8}
		pdf.WriteMultiLine(buf, mb, nil, td)
	}
}

func writeUserFontDemoContent(p pdf.Page, fontName string, plane int) {
	baseFontName := "Helvetica"
	baseFontSize := 24
	baseFontKey := p.Fm.EnsureKey(baseFontName)

	fontKey := p.Fm.EnsureKey(fontName)
	fontSize := 24

	fillCol := pdf.NewSimpleColor(0xf7e6c7)
	pdf.DrawGrid(p.Buf, 16*16, 16*16, pdf.RectForWidthAndHeight(55, 16, 16*480, 16*480), pdf.Black, &fillCol)

	td := pdf.TextDescriptor{
		FontName:       fontName,
		FontKey:        fontKey,
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

	from := plane * 0x10000
	to := (plane+1)*0x10000 - 1
	s := fmt.Sprintf("%s %d points (%04X - %04X)", fontName, fontSize, from, to)

	td.X, td.Y, td.Text = p.MediaBox.Width()/2, 7750, s
	td.FontName, td.FontKey = baseFontName, baseFontKey
	td.StrokeCol, td.FillCol = pdf.NewSimpleColor(0x77bdbd), pdf.NewSimpleColor(0xab6f30)
	pdf.WriteMultiLine(p.Buf, p.MediaBox, nil, td)

	columnsLabel(td, baseFontName, baseFontKey, p.Buf, p.MediaBox, true)
	base := rune(plane * 0x10000)
	for j := 0; j < 256; j++ {
		rowLabel(j, td, baseFontName, baseFontKey, p.Buf, p.MediaBox, true)
		buf := make([]byte, 4)
		for i := 0; i < 256; i++ {
			n := utf8.EncodeRune(buf, base+rune(j*256+i))
			s = string(buf[:n])
			td.X, td.Y, td.Text = float64(70+i*30), float64(7672-j*30), s
			td.StrokeCol, td.FillCol = pdf.Black, pdf.Black
			td.FontName, td.FontKey, td.FontSize = fontName, fontKey, fontSize-2
			pdf.WriteMultiLine(p.Buf, p.MediaBox, nil, td)
		}
		rowLabel(j, td, baseFontName, baseFontKey, p.Buf, p.MediaBox, false)
	}
	columnsLabel(td, baseFontName, baseFontKey, p.Buf, p.MediaBox, false)
}

func createUserFontDemoPage(w, h, plane int, fontName string) pdf.Page {
	mediaBox := pdf.RectForDim(float64(w), float64(h))
	p := pdf.NewPageWithBg(mediaBox, pdf.NewSimpleColor(0xbeded9))
	writeUserFontDemoContent(p, fontName, plane)
	return p
}

func planeString(i int) string {
	switch i {
	case 0:
		return "BMP"
	case 1:
		return "SMP"
	case 2:
		return "SIP"
	case 3:
		return "TIP"
	case 14:
		return "SSP"
	case 15:
		return "SPUA"
	}
	return ""
}

func createUserFontDemoFiles(t *testing.T, w, h int, fn, msg string) error {
	t.Helper()
	ttf := font.UserFontMetrics[fn]
	for i := range ttf.Planes {
		p := createUserFontDemoPage(w, h, i, fn)
		xRefTable, err := pdf.CreateDemoXRef(p)
		if err != nil {
			return err
		}
		outFile := filepath.Join("../../samples/fonts/user", fn+"_"+planeString(i)+".pdf")
		createAndValidate(t, xRefTable, outFile, msg)
	}
	return nil
}

func TestUserFontDemoPDF(t *testing.T) {
	msg := "TestUserFontDemoPDF"
	w, h := 7800, 7800
	_ = pdf.NewDefaultConfiguration()
	for _, fn := range font.UserFontNames() {
		if err := createUserFontDemoFiles(t, w, h, fn, msg); err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
	}
}
