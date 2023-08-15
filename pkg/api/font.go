/*
	Copyright 2020 The pdfcpu Authors.

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

package api

import (
	"bytes"
	"fmt"

	"path/filepath"
	"sort"
	"unicode/utf8"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// func isSupportedFontFile(filename string) bool {
// 	return strings.HasSuffix(strings.ToLower(filename), ".gob")
// }

// ListFonts returns a list of supported fonts.
func ListFonts() ([]string, error) {

	// Get list of PDF core fonts.
	coreFonts := font.CoreFontNames()
	for i, s := range coreFonts {
		coreFonts[i] = "  " + s
	}
	sort.Strings(coreFonts)

	sscf := []string{"Corefonts:"}
	sscf = append(sscf, coreFonts...)

	// Get installed fonts from pdfcpu config dir in users home dir
	userFonts := font.UserFontNamesVerbose()
	for i, s := range userFonts {
		userFonts[i] = "  " + s
	}
	sort.Strings(userFonts)

	ssuf := []string{fmt.Sprintf("Userfonts(%s):", font.UserFontDir)}
	ssuf = append(ssuf, userFonts...)

	sscf = append(sscf, "")
	return append(sscf, ssuf...), nil
}

// InstallFonts installs true type fonts for embedding.
func InstallFonts(fileNames []string) error {
	log.CLI.Printf("installing to %s...", font.UserFontDir)

	for _, fn := range fileNames {
		switch filepath.Ext(fn) {
		case ".ttf":
			//log.CLI.Println(filepath.Base(fn))
			if err := font.InstallTrueTypeFont(font.UserFontDir, fn); err != nil {
				log.CLI.Printf("%v", err)
			}
		case ".ttc":
			//log.CLI.Println(filepath.Base(fn))
			if err := font.InstallTrueTypeCollection(font.UserFontDir, fn); err != nil {
				log.CLI.Printf("%v", err)
			}
		}
	}

	return font.LoadUserFonts()
}

func rowLabel(xRefTable *model.XRefTable, i int, td model.TextDescriptor, baseFontName, baseFontKey string, buf *bytes.Buffer, mb *types.Rectangle, left bool) {
	x := 39.
	if !left {
		x = 7750
	}
	s := fmt.Sprintf("#%02X", i)
	td.X, td.Y, td.Text = x, float64(7677-i*30), s
	td.StrokeCol, td.FillCol = color.Black, color.SimpleColor{B: .8}
	td.FontName, td.FontKey, td.FontSize = baseFontName, baseFontKey, 14

	model.WriteMultiLine(xRefTable, buf, mb, nil, td)
}

func columnsLabel(xRefTable *model.XRefTable, td model.TextDescriptor, baseFontName, baseFontKey string, buf *bytes.Buffer, mb *types.Rectangle, top bool) {
	y := 7700.
	if !top {
		y = 0
	}

	td.FontName, td.FontKey = baseFontName, baseFontKey

	for i := 0; i < 256; i++ {
		s := fmt.Sprintf("#%02X", i)
		td.X, td.Y, td.Text, td.FontSize = float64(70+i*30), y, s, 14
		td.StrokeCol, td.FillCol = color.Black, color.SimpleColor{B: .8}
		model.WriteMultiLine(xRefTable, buf, mb, nil, td)
	}
}

func surrogate(r rune) bool {
	return r >= 0xD800 && r <= 0xDFFF
}

func writeUserFontDemoContent(xRefTable *model.XRefTable, p model.Page, fontName string, plane int) {
	baseFontName := "Helvetica"
	baseFontSize := 24
	baseFontKey := p.Fm.EnsureKey(baseFontName)

	fontKey := p.Fm.EnsureKey(fontName)
	fontSize := 24

	fillCol := color.NewSimpleColor(0xf7e6c7)
	draw.DrawGrid(p.Buf, 16*16, 16*16, types.RectForWidthAndHeight(55, 16, 16*480, 16*480), color.Black, &fillCol)

	td := model.TextDescriptor{
		FontName:       fontName,
		FontKey:        fontKey,
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

	from := plane * 0x10000
	to := (plane+1)*0x10000 - 1
	s := fmt.Sprintf("%s %d points (%04X - %04X)", fontName, fontSize, from, to)

	td.X, td.Y, td.Text = p.MediaBox.Width()/2, 7750, s
	td.FontName, td.FontKey = baseFontName, baseFontKey
	td.StrokeCol, td.FillCol = color.NewSimpleColor(0x77bdbd), color.NewSimpleColor(0xab6f30)
	model.WriteMultiLine(xRefTable, p.Buf, p.MediaBox, nil, td)

	columnsLabel(xRefTable, td, baseFontName, baseFontKey, p.Buf, p.MediaBox, true)
	base := rune(plane * 0x10000)
	for j := 0; j < 256; j++ {
		rowLabel(xRefTable, j, td, baseFontName, baseFontKey, p.Buf, p.MediaBox, true)
		buf := make([]byte, 4)
		td.StrokeCol, td.FillCol = color.Black, color.Black
		td.FontName, td.FontKey, td.FontSize = fontName, fontKey, fontSize-2
		for i := 0; i < 256; i++ {
			r := base + rune(j*256+i)
			s = " "
			if !surrogate(r) {
				n := utf8.EncodeRune(buf, r)
				s = string(buf[:n])
			}
			td.X, td.Y, td.Text = float64(70+i*30), float64(7672-j*30), s
			model.WriteMultiLine(xRefTable, p.Buf, p.MediaBox, nil, td)
		}
		rowLabel(xRefTable, j, td, baseFontName, baseFontKey, p.Buf, p.MediaBox, false)
	}
	columnsLabel(xRefTable, td, baseFontName, baseFontKey, p.Buf, p.MediaBox, false)
}

func createUserFontDemoPage(xRefTable *model.XRefTable, w, h, plane int, fontName string) model.Page {
	mediaBox := types.RectForDim(float64(w), float64(h))
	p := model.NewPageWithBg(mediaBox, color.NewSimpleColor(0xbeded9))
	writeUserFontDemoContent(xRefTable, p, fontName, plane)
	return p
}

func planeString(i int) string {
	switch i {
	case 0:
		return "BMP" // Basic Multilingual Plane
	case 1:
		return "SMP" // Supplementary Multilingual Plane
	case 2:
		return "SIP" // Supplementary Ideographic Plane
	case 3:
		return "TIP" // Tertiary Ideographic Plane
	case 14:
		return "SSP" // Supplementary Special-purpose Plane
	case 15:
		return "SPUA" // Supplementary Private Use Area Plane
	}
	return ""
}

// CreateUserFontDemoFiles creates single page PDF for each Unicode plane covered.
func CreateUserFontDemoFiles(dir, fn string) error {
	w, h := 7800, 7800
	font.UserFontMetricsLock.RLock()
	ttf, ok := font.UserFontMetrics[fn]
	font.UserFontMetricsLock.RUnlock()
	if !ok {
		return errors.Errorf("pdfcpu: font %s not available\n", fn)
	}
	// Create a single page PDF for each Unicode plane with existing glyphs.
	for i := range ttf.Planes {
		xRefTable, err := pdfcpu.CreateDemoXRef()
		if err != nil {
			return err
		}
		p := createUserFontDemoPage(xRefTable, w, h, i, fn)

		rootDict, err := xRefTable.Catalog()
		if err != nil {
			return err
		}
		if err = pdfcpu.AddPageTreeWithSamplePage(xRefTable, rootDict, p); err != nil {
			return err
		}
		fileName := filepath.Join(dir, fn+"_"+planeString(i)+".pdf")
		if err := CreatePDFFile(xRefTable, fileName, nil); err != nil {
			return err
		}
	}
	return nil
}

// CreateCheatSheetsUserFonts creates single page PDF cheat sheets for installed user fonts.
func CreateCheatSheetsUserFonts(fontNames []string) error {
	if len(fontNames) == 0 {
		fontNames = font.UserFontNames()
	}
	sort.Strings(fontNames)
	for _, fn := range fontNames {
		if !font.IsUserFont(fn) {
			log.CLI.Printf("unknown user font: %s\n", fn)
			continue
		}
		log.CLI.Println("creating cheatsheets for: " + fn)
		if err := CreateUserFontDemoFiles(".", fn); err != nil {
			return err
		}
	}
	return nil
}
