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

package primitives

import (
	"bytes"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	pdffont "github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

type PDF struct {
	Paper           string               // default paper size
	mediaBox        *pdfcpu.Rectangle    // default media box
	Crop            string               // default crop box
	cropBox         *pdfcpu.Rectangle    // default crop box
	Origin          string               // origin of the coordinate system
	origin          pdfcpu.Corner        // one of 4 page corners
	Guides          bool                 // render guides for layouting
	ContentBox      bool                 // render contentBox = cropBox - header - footer
	BackgroundColor string               `json:"bgCol"`
	bgCol           *pdfcpu.SimpleColor  // default background color
	Fonts           map[string]*FormFont // default fonts
	FormFontIDs     map[string]string
	FieldIDs        pdfcpu.StringSet
	Fields          pdfcpu.Array
	InheritedDA     string
	Header          *HorizontalBand
	Footer          *HorizontalBand
	Pages           map[string]*PDFPage
	pages           []*PDFPage
	Margin          *Margin               // the global margin named "margin"
	Border          *Border               // the global border named "border"
	Padding         *Padding              // the global padding named "padding"
	Margins         map[string]*Margin    // global named margins
	Borders         map[string]*Border    // global named borders
	Paddings        map[string]*Padding   // global named paddings
	SimpleBoxPool   map[string]*SimpleBox `json:"boxes"`
	TextBoxPool     map[string]*TextBox   `json:"texts"`
	ImageBoxPool    map[string]*ImageBox  `json:"images"`
	TablePool       map[string]*Table     `json:"tables"`
	Colors          map[string]string
	colors          map[string]pdfcpu.SimpleColor
	DirNames        map[string]string `json:"dirs"`
	FileNames       map[string]string `json:"files"`
	TimestampFormat string            `json:"timestamp"`
	Conf            *pdfcpu.Configuration
	XRefTable       *pdfcpu.XRefTable
	Optimize        *pdfcpu.OptimizationContext
	FontResIDs      map[int]pdfcpu.Dict
	XObjectResIDs   map[int]pdfcpu.Dict
	CheckBoxAPs     map[float64]*AP
	RadioBtnAPs     map[float64]*AP
	HasForm         bool
}

func (pdf *PDF) pageCount() int {
	return len(pdf.pages)
}

func (pdf *PDF) Validate() error {

	// Default paper size
	defaultPaperSize := "A4"

	// Default media box
	pdf.mediaBox = pdfcpu.RectForFormat(defaultPaperSize)
	if pdf.Paper != "" {
		dim, _, err := pdfcpu.ParsePageFormat(pdf.Paper)
		if err != nil {
			return err
		}
		pdf.mediaBox = pdfcpu.RectForDim(dim.Width, dim.Height)
	}
	pdf.cropBox = pdf.mediaBox.CroppedCopy(0)

	if pdf.Crop != "" {
		box, err := pdfcpu.ParseBox(pdf.Crop, pdfcpu.POINTS)
		if err != nil {
			return err
		}
		pdf.cropBox = pdfcpu.ApplyBox("CropBox", box, nil, pdf.mediaBox)
	}

	// Layout coordinate system
	pdf.origin = pdfcpu.LowerLeft
	if pdf.Origin != "" {
		corner, err := pdfcpu.ParseOrigin(pdf.Origin)
		if err != nil {
			return err
		}
		pdf.origin = corner
	}

	// Custom colors
	pdf.colors = map[string]pdfcpu.SimpleColor{}
	for n, c := range pdf.Colors {
		if c == "" {
			continue
		}
		sc, err := pdfcpu.ParseHexColor(c)
		if err != nil {
			return err
		}
		pdf.colors[strings.ToLower(n)] = sc
	}

	// Default background color
	if pdf.BackgroundColor != "" {
		sc, err := pdf.parseColor(pdf.BackgroundColor)
		if err != nil {
			return err
		}
		pdf.bgCol = sc
	}

	// Default fonts
	for _, f := range pdf.Fonts {
		f.pdf = pdf
		if err := f.validate(); err != nil {
			return err
		}
	}

	if pdf.Header != nil {
		if err := pdf.Header.validate(); err != nil {
			return err
		}
		pdf.Header.position = pdfcpu.TopCenter
		pdf.Header.pdf = pdf
	}

	if pdf.Footer != nil {
		if err := pdf.Footer.validate(); err != nil {
			return err
		}
		pdf.Footer.position = pdfcpu.BottomCenter
		pdf.Footer.pdf = pdf
	}

	if pdf.TimestampFormat == "" {
		pdf.TimestampFormat = pdf.Conf.TimestampFormat
	}

	// What follows is a quirky way of turning a map of pages into a sorted slice of pages
	// including entries for pages that are missing in the map.

	var pageNrs []int

	for pageNr, p := range pdf.Pages {
		nr, err := strconv.Atoi(pageNr)
		if err != nil {
			return errors.Errorf("pdfcpu: invalid page number: %s", pageNr)
		}
		pageNrs = append(pageNrs, nr)
		p.number = nr
		p.pdf = pdf
		if err := p.validate(); err != nil {
			return err
		}
	}

	sort.Ints(pageNrs)

	pp := []*PDFPage{}

	maxPageNr := pageNrs[len(pageNrs)-1]
	for i := 1; i <= maxPageNr; i++ {
		pp = append(pp, pdf.Pages[strconv.Itoa(i)])
	}

	pdf.pages = pp

	if pdf.Border != nil {
		if len(pdf.Borders) > 0 {
			return errors.New("pdfcpu: Please supply either \"border\" or \"borders\"")
		}
		pdf.Border.pdf = pdf
		if err := pdf.Border.validate(); err != nil {
			return err
		}
		pdf.Borders = map[string]*Border{}
		pdf.Borders["border"] = pdf.Border
	}
	for _, b := range pdf.Borders {
		b.pdf = pdf
		if err := b.validate(); err != nil {
			return err
		}
	}

	if pdf.Margin != nil {
		if len(pdf.Margins) > 0 {
			return errors.New("pdfcpu: Please supply either \"margin\" or \"margins\"")
		}
		if err := pdf.Margin.validate(); err != nil {
			return err
		}
		pdf.Margins = map[string]*Margin{}
		pdf.Margins["margin"] = pdf.Margin
	}
	for _, m := range pdf.Margins {
		if err := m.validate(); err != nil {
			return err
		}
	}

	if pdf.Padding != nil {
		if len(pdf.Paddings) > 0 {
			return errors.New("pdfcpu: Please supply either \"padding\" or \"paddings\"")
		}
		if err := pdf.Padding.validate(); err != nil {
			return err
		}
		pdf.Paddings = map[string]*Padding{}
		pdf.Paddings["padding"] = pdf.Padding
	}
	for _, p := range pdf.Paddings {
		if err := p.validate(); err != nil {
			return err
		}
	}

	// box templates
	for _, sb := range pdf.SimpleBoxPool {
		sb.pdf = pdf
		if err := sb.validate(); err != nil {
			return err
		}
	}

	// text templates
	for _, tb := range pdf.TextBoxPool {
		tb.pdf = pdf
		if err := tb.validate(); err != nil {
			return err
		}
	}

	// image templates
	for _, ib := range pdf.ImageBoxPool {
		ib.pdf = pdf
		if err := ib.validate(); err != nil {
			return err
		}
	}

	// table templates
	for _, t := range pdf.TablePool {
		t.pdf = pdf
		if err := t.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (pdf *PDF) calcInheritedAttrs() {

	// Calc inherited fonts.
	for id, f0 := range pdf.Fonts {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			f1 := page.Fonts[id]
			if f1 != nil {
				f1.mergeIn(f0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		ff := map[string]*FormFont{}
		for k, v := range pdf.Fonts {
			ff[k] = v
		}
		for k, v := range page.Fonts {
			ff[k] = v
		}
		page.Content.calcFont(ff)
	}

	// Calc inherited margins.
	for id, m0 := range pdf.Margins {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			m1 := page.Margins[id]
			if m1 != nil {
				m1.mergeIn(m0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		mm := map[string]*Margin{}
		for k, v := range pdf.Margins {
			mm[k] = v
		}
		for k, v := range page.Margins {
			mm[k] = v
		}
		page.Content.calcMargin(mm)
	}

	// Calc inherited borders.
	for id, b0 := range pdf.Borders {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			b1 := page.Borders[id]
			if b1 != nil {
				b1.mergeIn(b0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		bb := map[string]*Border{}
		for k, v := range pdf.Borders {
			bb[k] = v
		}
		for k, v := range page.Borders {
			bb[k] = v
		}
		page.Content.calcBorder(bb)
	}

	// Calc inherited paddings.
	for id, p0 := range pdf.Paddings {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			p1 := page.Paddings[id]
			if p1 != nil {
				p1.mergeIn(p0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		pp := map[string]*Padding{}
		for k, v := range pdf.Paddings {
			pp[k] = v
		}
		for k, v := range page.Paddings {
			pp[k] = v
		}
		page.Content.calcPadding(pp)
	}

	// Calc inherited SimpleBoxes.
	for id, sb0 := range pdf.SimpleBoxPool {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			sb1 := page.SimpleBoxPool[id]
			if sb1 != nil {
				sb1.mergeIn(sb0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		bb := map[string]*SimpleBox{}
		for k, v := range pdf.SimpleBoxPool {
			bb[k] = v
		}
		for k, v := range page.SimpleBoxPool {
			bb[k] = v
		}
		page.Content.calcSimpleBoxes(bb)
	}

	// Calc inherited TextBoxes.
	for id, tb0 := range pdf.TextBoxPool {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			tb1 := page.TextBoxPool[id]
			if tb1 != nil {
				tb1.mergeIn(tb0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		tb := map[string]*TextBox{}
		for k, v := range pdf.TextBoxPool {
			tb[k] = v
		}
		for k, v := range page.TextBoxPool {
			tb[k] = v
		}
		page.Content.calcTextBoxes(tb)
	}

	// Calc inherited ImageBoxes.
	for id, ib0 := range pdf.ImageBoxPool {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			ib1 := page.ImageBoxPool[id]
			if ib1 != nil {
				ib1.mergeIn(ib0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		ib := map[string]*ImageBox{}
		for k, v := range pdf.ImageBoxPool {
			ib[k] = v
		}
		for k, v := range page.ImageBoxPool {
			ib[k] = v
		}
		page.Content.calcImageBoxes(ib)
	}

	// Calc inherited Tables.
	for id, t0 := range pdf.TablePool {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			t1 := page.TablePool[id]
			if t1 != nil {
				t1.mergeIn(t0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		t := map[string]*Table{}
		for k, v := range pdf.TablePool {
			t[k] = v
		}
		for k, v := range page.TablePool {
			t[k] = v
		}
		page.Content.calcTables(t)
	}
}

func (pdf *PDF) color(s string) *pdfcpu.SimpleColor {
	sc, ok := pdf.colors[strings.ToLower(s)]
	if !ok {
		return nil
	}
	return &sc
}

func (pdf *PDF) parseColor(s string) (*pdfcpu.SimpleColor, error) {
	sc, err := pdfcpu.ParseColor(s)
	if err == nil {
		return &sc, nil
	}
	if err != pdfcpu.ErrInvalidColor || s[0] != '$' {
		return nil, err
	}
	sc1 := pdf.color(s[1:])
	if sc1 == nil {
		return nil, pdfcpu.ErrInvalidColor
	}
	return sc1, nil
}

func (pdf *PDF) resolveFileName(s string) (string, error) {
	filePath, ok := pdf.FileNames[s]
	if !ok {
		return "", errors.Errorf("pdfcpu: can't resolve filename: %s", s)
	}
	if filePath[0] != '$' {
		return filePath, nil
	}

	filePath = filePath[1:]
	i := strings.Index(filePath, "/")
	if i <= 0 {
		return "", errors.Errorf("pdfcpu: corrupt filename: %s", s)
	}

	dirName := filePath[:i]
	fileName := filePath[i:]

	dirPath, ok := pdf.DirNames[dirName]
	if !ok {
		return "", errors.Errorf("pdfcpu: can't resolve dirname: %s", dirName)
	}

	s1 := filepath.Join(dirPath, fileName)

	return s1, nil
}

func (pdf *PDF) calcFont(f *FormFont) error {
	if f.Name[0] != '$' {
		return nil
	}
	fName := f.Name[1:]
	f0 := pdf.Fonts[fName]
	if f0 == nil {
		return errors.Errorf("pdfcpu: unknown font %s", fName)
	}
	f.Name = f0.Name
	if f.Size == 0 {
		f.Size = f0.Size
	}
	if f.col == nil {
		f.col = f0.col
	}
	return nil
}

func (pdf *PDF) idForFontName(fontName string, pageFonts, fonts pdfcpu.FontMap, pageNr int) (string, error) {

	var (
		id     string
		indRef *pdfcpu.IndirectRef
	)

	font, ok := pageFonts[fontName]
	if ok {
		return font.Res.ID, nil
	}

	fontResIDs := pdf.FontResIDs[pageNr]

	font, ok = fonts[fontName]
	if ok {
		font.Res.ID = "F" + strconv.Itoa(len(pageFonts))
		if pdf.Optimize != nil && font.Res.IndRef != nil {
			id := fontResIDs.NewIDForPrefix("F", len(pageFonts))
			for k, v := range fontResIDs {
				if v == *font.Res.IndRef {
					id = k
					break
				}
			}
			font.Res.ID = id
		}
		pageFonts[fontName] = font
		return font.Res.ID, nil
	}

	font = pdfcpu.FontResource{}

	if pdf.Optimize != nil {

		for objNr, fo := range pdf.Optimize.FormFontObjects {
			//fmt.Printf("searching for %s - obj:%d fontName:%s prefix:%s\n", fontName, objNr, fo.FontName, fo.Prefix)
			if fontName == fo.FontName {
				//fmt.Println("Match!")
				indRef = pdfcpu.NewIndirectRef(objNr, 0)
				break
			}
		}

		if indRef == nil {
			for objNr, fo := range pdf.Optimize.FontObjects {
				//fmt.Printf("searching for %s - obj:%d fontName:%s prefix:%s\n", fontName, objNr, fo.FontName, fo.Prefix)
				if fontName == fo.FontName {
					//fmt.Println("Match!")
					indRef = pdfcpu.NewIndirectRef(objNr, 0)
					if pdffont.IsUserFont(fontName) {
						if err := pdfcpu.IndRefsForUserfontUpdate(pdf.XRefTable, fo.FontDict, &font); err != nil {
							return "", err
						}
					}
					break
				}
			}
		}

		if indRef != nil {
			id = fontResIDs.NewIDForPrefix("F", len(fonts))
			for k, v := range fontResIDs {
				if v == *indRef {
					id = k
					break
				}
			}
		}

	}

	if indRef == nil {
		id = "F" + strconv.Itoa(len(pageFonts))
		if pdf.Optimize != nil {
			id = fontResIDs.NewIDForPrefix("F", len(pageFonts))
		}
	}

	font.Res = pdfcpu.Resource{ID: id, IndRef: indRef}
	fonts[fontName] = font
	pageFonts[fontName] = font

	return id, nil
}

func (pdf *PDF) ensureFormFont(fontName string) (string, error) {
	id, ok := pdf.FormFontIDs[fontName]
	if ok {
		return id, nil
	}
	id = "F" + strconv.Itoa(len(pdf.FormFontIDs))
	pdf.FormFontIDs[fontName] = id
	return id, nil
}

func (pdf *PDF) RenderPages() ([]*pdfcpu.Page, pdfcpu.FontMap, pdfcpu.Array, error) {

	pdf.calcInheritedAttrs()

	pp := []*pdfcpu.Page{}
	fontMap := pdfcpu.FontMap{}
	imageMap := pdfcpu.ImageMap{}
	fields := pdfcpu.Array{}

	// f := pdf.Fonts["input"]
	// if f != nil {

	// 	if f.col == nil {
	// 		f.col = &pdfcpu.Black
	// 	}

	// 	fontName := f.Name
	// 	fontSize := f.Size
	// 	col := f.col

	// 	id, err := pdf.ensureFormFont(fontName)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	da := fmt.Sprintf("/%s %d Tf %.2f %.2f %.2f rg", id, fontSize, col.R, col.G, col.B)

	// 	pdf.InheritedDA = da
	// }

	for i, page := range pdf.pages {

		pageNr := i + 1
		mediaBox := pdf.mediaBox
		cropBox := pdf.cropBox

		p := pdfcpu.Page{
			MediaBox: mediaBox,
			CropBox:  cropBox,
			Fm:       pdfcpu.FontMap{},
			Im:       pdfcpu.ImageMap{},
			Buf:      new(bytes.Buffer),
		}

		if page == nil {
			if pageNr <= pdf.XRefTable.PageCount {
				pp = append(pp, nil)
				continue
			}
			// Create blank page with optional background color.
			if pdf.bgCol != nil {
				pdfcpu.FillRectNoBorder(p.Buf, p.CropBox, *pdf.bgCol)
			}
			pp = append(pp, &p)
			continue
		}

		if page.mediaBox != nil {
			mediaBox = page.mediaBox
		}
		if page.cropBox != nil {
			cropBox = page.cropBox
		}

		// Render page background.
		if page.bgCol == nil {
			page.bgCol = pdf.bgCol
		}
		if page.bgCol != nil {
			pdfcpu.FillRectNoBorder(p.Buf, cropBox, *page.bgCol)
		}

		var headerHeight, headerDy float64
		var footerHeight, footerDy float64

		// Render page header.
		if pdf.Header != nil {
			if err := pdf.Header.render(&p, pageNr, fontMap, imageMap, true); err != nil {
				return nil, nil, nil, err
			}
			headerHeight = pdf.Header.Height
			headerDy = float64(pdf.Header.Dy)
		}

		// Render page footer.
		if pdf.Footer != nil {
			if err := pdf.Footer.render(&p, pageNr, fontMap, imageMap, false); err != nil {
				return nil, nil, nil, err
			}
			footerHeight = pdf.Footer.Height
			footerDy = float64(pdf.Footer.Dy)
		}

		// Render page content.
		r := cropBox.CroppedCopy(0)
		r.LL.Y += footerHeight + footerDy
		r.UR.Y -= headerHeight + headerDy
		page.Content.mediaBox = r
		if err := page.Content.render(&p, pageNr, fontMap, imageMap, &fields); err != nil {
			return nil, nil, nil, err
		}

		pp = append(pp, &p)
	}

	return pp, fontMap, fields, nil
}
