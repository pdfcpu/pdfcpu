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
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// FieldFlags represents the PDF form field flags.
// See table 221 et.al.
type FieldFlags int

const (
	FieldReadOnly FieldFlags = 1 << iota
	FieldRequired
	FieldNoExport
	UnusedFlag4
	UnusedFlag5
	UnusedFlag6
	UnusedFlag7
	UnusedFlag8
	UnusedFlag9
	UnusedFlag10
	UnusedFlag11
	UnusedFlag12
	FieldMultiline
	FieldPassword
	FieldNoToggleToOff
	FieldRadio
	FieldPushbutton
	FieldCombo
	FieldEdit
	FieldSort
	FieldFileSelect
	FieldMultiselect
	FieldDoNotSpellCheck
	FieldDoNotScroll
	FieldComb
	FieldRichTextAndRadiosInUnison
	FieldCommitOnSelChange
)

// PDF is the central structure for PDF generation.
type PDF struct {
	Paper           string               // default paper size
	mediaBox        *types.Rectangle     // default media box
	Crop            string               // default crop box
	cropBox         *types.Rectangle     // default crop box
	Origin          string               // origin of the coordinate system
	origin          types.Corner         // one of 4 page corners
	Guides          bool                 // render guides for layouting
	ContentBox      bool                 // render contentBox = cropBox - header - footer
	Debug           bool                 // highlight element positions
	BackgroundColor string               `json:"bgCol"`
	bgCol           *color.SimpleColor   // default background color
	Fonts           map[string]*FormFont // global fonts
	FormFonts       map[string]*FormFont
	FieldIDs        types.StringSet
	Fields          types.Array
	InheritedDA     string
	Header          *HorizontalBand
	Footer          *HorizontalBand
	Pages           map[string]*PDFPage
	pages           []*PDFPage
	Margin          *Margin                // the global margin named "margin"
	Border          *Border                // the global border named "border"
	Padding         *Padding               // the global padding named "padding"
	Margins         map[string]*Margin     // global named margins
	Borders         map[string]*Border     // global named borders
	Paddings        map[string]*Padding    // global named paddings
	SimpleBoxPool   map[string]*SimpleBox  `json:"boxes"`
	TextBoxPool     map[string]*TextBox    `json:"texts"`
	ImageBoxPool    map[string]*ImageBox   `json:"images"`
	TablePool       map[string]*Table      `json:"tables"`
	FieldGroupPool  map[string]*FieldGroup `json:"fieldgroups"`
	Colors          map[string]string
	colors          map[string]color.SimpleColor
	DirNames        map[string]string          `json:"dirs"`
	FileNames       map[string]string          `json:"files"`
	TimestampFormat string                     `json:"timestamp"`
	DateFormat      string                     `json:"dateFormat"`
	Conf            *model.Configuration       `json:"-"`
	XRefTable       *model.XRefTable           `json:"-"`
	Optimize        *model.OptimizationContext `json:"-"`
	FontResIDs      map[int]types.Dict         `json:"-"`
	XObjectResIDs   map[int]types.Dict         `json:"-"`
	CheckBoxAPs     map[float64]*AP            `json:"-"`
	RadioBtnAPs     map[float64]*AP            `json:"-"`
	HasForm         bool                       `json:"-"`
	OldFieldIDs     types.StringSet            `json:"-"`
	httpClient      *http.Client
}

func (pdf *PDF) Update() bool {
	return pdf.Optimize != nil
}

func (pdf *PDF) pageCount() int {
	return len(pdf.pages)
}

func (pdf *PDF) color(s string) *color.SimpleColor {
	sc, ok := pdf.colors[strings.ToLower(s)]
	if !ok {
		return nil
	}
	return &sc
}

func (pdf *PDF) parseColor(s string) (*color.SimpleColor, error) {
	sc, err := color.ParseColor(s)
	if err == nil {
		return &sc, nil
	}
	if err != color.ErrInvalidColor || s[0] != '$' {
		return nil, err
	}
	sc1 := pdf.color(s[1:])
	if sc1 == nil {
		return nil, color.ErrInvalidColor
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

func (pdf *PDF) validatePageBoundaries() error {
	// Default paper size
	defaultPaperSize := "A4"

	// Default media box
	pdf.mediaBox = types.RectForFormat(defaultPaperSize)
	if pdf.Paper != "" {
		dim, _, err := types.ParsePageFormat(pdf.Paper)
		if err != nil {
			return err
		}
		pdf.mediaBox = types.RectForDim(dim.Width, dim.Height)
	}
	pdf.cropBox = pdf.mediaBox.CroppedCopy(0)

	if pdf.Crop != "" {
		box, err := model.ParseBox(pdf.Crop, types.POINTS)
		if err != nil {
			return err
		}
		pdf.cropBox = model.ApplyBox("CropBox", box, nil, pdf.mediaBox)
	}
	return nil
}

func (pdf *PDF) validateOrigin() error {
	// Layout coordinate system
	pdf.origin = types.LowerLeft
	if pdf.Origin != "" {
		corner, err := types.ParseOrigin(pdf.Origin)
		if err != nil {
			return err
		}
		pdf.origin = corner
	}
	return nil
}

func (pdf *PDF) validateColors() error {
	// Custom colors
	pdf.colors = map[string]color.SimpleColor{}
	for n, c := range pdf.Colors {
		if c == "" {
			continue
		}
		sc, err := color.NewSimpleColorForHexCode(c)
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
	return nil
}

func (pdf *PDF) validateFonts() error {
	// Default fonts
	for _, f := range pdf.Fonts {
		f.pdf = pdf
		if err := f.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (pdf *PDF) validateHeader() error {
	if pdf.Header != nil {
		if err := pdf.Header.validate(); err != nil {
			return err
		}
		pdf.Header.position = types.TopCenter
		pdf.Header.pdf = pdf
	}
	return nil
}

func (pdf *PDF) validateFooter() error {
	if pdf.Footer != nil {
		if err := pdf.Footer.validate(); err != nil {
			return err
		}
		pdf.Footer.position = types.BottomCenter
		pdf.Footer.pdf = pdf
	}
	return nil
}

func (pdf *PDF) validateBorders() error {
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
	return nil
}

func (pdf *PDF) validateMargins() error {
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
	return nil
}

func (pdf *PDF) validatePaddings() error {
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
	return nil
}

func (pdf *PDF) validateSimpleBoxPool() error {
	// box templates
	for _, sb := range pdf.SimpleBoxPool {
		sb.pdf = pdf
		if err := sb.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (pdf *PDF) validateTextBoxPool() error {
	// text templates
	for _, tb := range pdf.TextBoxPool {
		tb.pdf = pdf
		if err := tb.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (pdf *PDF) validateImageBoxPool() error {
	// image templates
	for _, ib := range pdf.ImageBoxPool {
		ib.pdf = pdf
		if err := ib.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (pdf *PDF) validateTablePool() error {
	// table templates
	for _, t := range pdf.TablePool {
		t.pdf = pdf
		if err := t.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (pdf *PDF) validateFieldGroupPool() error {
	for _, fg := range pdf.FieldGroupPool {
		fg.pdf = pdf
		if err := fg.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (pdf *PDF) validatePools() error {
	if err := pdf.validateSimpleBoxPool(); err != nil {
		return err
	}
	if err := pdf.validateTextBoxPool(); err != nil {
		return err
	}
	if err := pdf.validateImageBoxPool(); err != nil {
		return err
	}
	if err := pdf.validateTablePool(); err != nil {
		return err
	}
	return pdf.validateFieldGroupPool()
}

func (pdf *PDF) validateBordersMarginsPaddings() error {
	if err := pdf.validateBorders(); err != nil {
		return err
	}

	if err := pdf.validateMargins(); err != nil {
		return err
	}

	return pdf.validatePaddings()
}

func (pdf *PDF) Validate() error {

	if err := pdf.validatePageBoundaries(); err != nil {
		return err
	}

	if err := pdf.validateOrigin(); err != nil {
		return err
	}

	if err := pdf.validateColors(); err != nil {
		return err
	}

	if err := pdf.validateFonts(); err != nil {
		return err
	}

	if err := pdf.validateHeader(); err != nil {
		return err
	}

	if err := pdf.validateFooter(); err != nil {
		return err
	}

	if pdf.TimestampFormat == "" {
		pdf.TimestampFormat = pdf.Conf.TimestampFormat
	}

	if pdf.DateFormat == "" {
		pdf.DateFormat = pdf.Conf.DateFormat
	}

	if len(pdf.Pages) == 0 {
		return errors.New("pdfcpu: Please supply \"pages\"")
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

	if err := pdf.validateBordersMarginsPaddings(); err != nil {
		return err
	}

	return pdf.validatePools()
}

func (pdf *PDF) DuplicateField(ID string) bool {
	if pdf.FieldIDs[ID] || pdf.OldFieldIDs[ID] {
		return true
	}
	oldID, err := types.EscapeUTF16String(ID)
	if err != nil {
		return true
	}
	return pdf.OldFieldIDs[*oldID]
}

func (pdf *PDF) calcFont(f *FormFont) error {
	// called by non content primitives using fonts
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
	if f.Lang == "" {
		f.Lang = f0.Lang
	}
	if f.Script == "" {
		f.Script = f0.Script
	}
	return nil
}

func (pdf *PDF) newPageFontID(indRef *types.IndirectRef, nextInd, pageNr int) string {
	id := "F" + strconv.Itoa(nextInd)
	if pdf.Update() {
		fontResIDs := pdf.FontResIDs[pageNr]
		id = fontResIDs.NewIDForPrefix("F", nextInd)
		if indRef == nil {
			return id
		}
		for k, v := range fontResIDs {
			if v == *indRef {
				id = k
				break
			}
		}
	}
	return id
}

func (pdf *PDF) idForFontName(fontName, fontLang string, pageFonts, globalFonts model.FontMap, pageNr int) (string, error) {

	// Used for textdescriptor configuration.

	var (
		id     string
		indRef *types.IndirectRef
	)

	fr, ok := pageFonts[fontName]
	if ok {
		// Return id of known page fontResource.
		return fr.Res.ID, nil
	}

	fr, ok = globalFonts[fontName]
	if ok {
		// Add global fontResource to page fontResource and return its id.
		fr.Res.ID = pdf.newPageFontID(fr.Res.IndRef, len(pageFonts), pageNr)
		pageFonts[fontName] = fr
		return fr.Res.ID, nil
	}

	// Create new fontResource.

	fr = model.FontResource{}

	if pdf.Update() {

		for objNr, fo := range pdf.Optimize.FormFontObjects {
			//fmt.Printf("searching for %s - obj:%d fontName:%s prefix:%s\n", fontName, objNr, fo.FontName, fo.Prefix)
			if fontName == fo.FontName {
				if font.IsCoreFont(fontName) {
					indRef = types.NewIndirectRef(objNr, 0)
					break
				}
				err := pdffont.IndRefsForUserfontUpdate(pdf.XRefTable, fo.FontDict, fontLang, &fr)
				if err == nil {
					indRef = types.NewIndirectRef(objNr, 0)
					break
				}
				if err != pdffont.ErrCorruptFontDict {
					return "", err
				}
				break
			}
		}

		if indRef == nil {
			for objNr, fo := range pdf.Optimize.FontObjects {
				//fmt.Printf("searching for %s - obj:%d fontName:%s prefix:%s\n", fontName, objNr, fo.FontName, fo.Prefix)
				if fontName == fo.FontName {
					indRef = types.NewIndirectRef(objNr, 0)
					if font.IsUserFont(fontName) {
						if err := pdffont.IndRefsForUserfontUpdate(pdf.XRefTable, fo.FontDict, fontLang, &fr); err != nil {
							return "", err
						}
					}
					break
				}
			}
		}

	}

	id = pdf.newPageFontID(indRef, len(pageFonts), pageNr)
	fr.Res = model.Resource{ID: id, IndRef: indRef}
	fr.Lang = fontLang

	globalFonts[fontName] = fr // id unique for pageFonts only.
	pageFonts[fontName] = fr

	return id, nil
}

func fontIndRef(xRefTable *model.XRefTable, fontName, fontLang string) (*types.IndirectRef, error) {
	fName := fontName
	if strings.HasPrefix(fontName, "cjk:") {
		fName = strings.TrimPrefix(fontName, "cjk:")
	}
	if font.IsUserFont(fName) {
		// Postpone font creation.
		return xRefTable.IndRefForNewObject(nil)
	}
	return pdffont.EnsureFontDict(xRefTable, fName, fontLang, "", false, false, nil)
}

func (pdf *PDF) ensureFont(fontID, fontName, fontLang string, fonts model.FontMap) (*types.IndirectRef, error) {

	fr, ok := fonts[fontName]
	if ok {
		if fr.Res.IndRef != nil {
			return fr.Res.IndRef, nil
		}
		var (
			ir  *types.IndirectRef
			err error
		)
		if font.IsUserFont(fontName) {
			// Postpone font creation.
			ir, err = pdf.XRefTable.IndRefForNewObject(nil)
		} else {
			ir, err = pdffont.EnsureFontDict(pdf.XRefTable, fontName, fr.Lang, "", false, false, nil)
		}
		if err != nil {
			return nil, err
		}
		fr.Res.IndRef = ir
		fonts[fontName] = fr
		return ir, nil
	}

	var (
		indRef *types.IndirectRef
		err    error
	)

	if pdf.Update() {

		fName := fontName
		if strings.HasPrefix(fontName, "cjk:") {
			fName = strings.TrimPrefix(fontName, "cjk:")
		}

		for objNr, fo := range pdf.Optimize.FormFontObjects {
			if fName == fo.FontName {
				indRef = types.NewIndirectRef(objNr, 0)
				break
			}
		}

		if indRef == nil {
			for objNr, fo := range pdf.Optimize.FontObjects {
				if fontName == fo.FontName {
					indRef = types.NewIndirectRef(objNr, 0)
					break
				}
			}
		}
	}

	if indRef == nil {
		if indRef, err = fontIndRef(pdf.XRefTable, fontName, fontLang); err != nil {
			return nil, err
		}
	}

	fr.Res = model.Resource{IndRef: indRef}
	fr.Lang = fontLang

	fonts[fontName] = fr

	return indRef, nil
}

func (pdf *PDF) ensureFormFont(font *FormFont) (string, error) {
	for id, f := range pdf.FormFonts {
		if f.Name == font.Name {
			return id, nil
		}
	}

	id := "F" + strconv.Itoa(len(pdf.FormFonts))

	if pdf.Update() && pdf.HasForm {

		for _, fo := range pdf.Optimize.FormFontObjects {
			if font.Name == fo.FontName {
				id := fo.ResourceNames[0]
				return id, nil
			}
		}

		// TODO Check for unique id
		id = "F" + strconv.Itoa(len(pdf.Optimize.FormFontObjects))
	}

	pdf.FormFonts[id] = font
	return id, nil
}

func (pdf *PDF) calcTopLevelFonts() {
	for _, f0 := range pdf.Fonts {
		if f0.Name[0] == '$' {
			fName := f0.Name[1:]
			for id, f1 := range pdf.Fonts {
				if id == fName {
					f0.Name = f1.Name
					if f0.Size == 0 {
						f0.Size = f1.Size
					}
					if f0.col == nil {
						f0.col = f1.col
					}
					if f0.Lang == "" {
						f0.Lang = f1.Lang
					}
					if f0.Script == "" {
						f0.Script = f1.Script
					}
				}
			}
		}
	}
}

func (pdf *PDF) calcInheritedPageFonts() {
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
}

func (pdf *PDF) calcInheritedContentFonts() {
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
}

func (pdf *PDF) calcInheritedFonts() {
	pdf.calcTopLevelFonts()
	pdf.calcInheritedPageFonts()
	pdf.calcInheritedContentFonts()
}

func (pdf *PDF) calcInheritedMargins() {
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
}

func (pdf *PDF) calcInheritedBorders() {
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
}

func (pdf *PDF) calcInheritedPaddings() {
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
}

func (pdf *PDF) calcInheritedSimpleBoxes() {
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
}

func (pdf *PDF) calcInheritedTextBoxes() {
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
}

func (pdf *PDF) calcInheritedImageBoxes() {
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
}

func (pdf *PDF) calcInheritedTables() {
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

func (pdf *PDF) calcInheritedFieldGroups() {
	// Calc inherited field groups.
	for id, fg0 := range pdf.FieldGroupPool {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			fg1 := page.FieldGroupPool[id]
			if fg1 != nil {
				fg1.mergeIn(fg0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		fg := map[string]*FieldGroup{}
		for k, v := range pdf.FieldGroupPool {
			fg[k] = v
		}
		for k, v := range page.FieldGroupPool {
			fg[k] = v
		}
		page.Content.calcFieldGroups(fg)
	}
}

func (pdf *PDF) calcInheritedAttrs() {
	pdf.calcInheritedFonts()
	pdf.calcInheritedMargins()
	pdf.calcInheritedBorders()
	pdf.calcInheritedPaddings()
	pdf.calcInheritedSimpleBoxes()
	pdf.calcInheritedTextBoxes()
	pdf.calcInheritedImageBoxes()
	pdf.calcInheritedTables()
	pdf.calcInheritedFieldGroups()
}

func (pdf *PDF) highlightPos(w io.Writer, x, y float64, cBox *types.Rectangle) {
	draw.DrawCircle(w, x, y, 5, color.Black, &color.Red)
	draw.DrawHairCross(w, x, y, cBox)
}

func (pdf *PDF) renderPageBackground(page *PDFPage, w io.Writer, cropBox *types.Rectangle) {
	if page.bgCol == nil {
		page.bgCol = pdf.bgCol
	}
	if page.bgCol != nil {
		draw.FillRectNoBorder(w, cropBox, *page.bgCol)
	}
}

// RenderPages renders page content into model.Pages
func (pdf *PDF) RenderPages() ([]*model.Page, model.FontMap, error) {

	pdf.calcInheritedAttrs()

	pp := []*model.Page{}
	fontMap := model.FontMap{}
	imageMap := model.ImageMap{}

	for i, page := range pdf.pages {

		pageNr := i + 1
		mediaBox := pdf.mediaBox
		cropBox := pdf.cropBox

		// check taborders

		p := model.Page{
			MediaBox:  mediaBox,
			CropBox:   cropBox,
			Fm:        model.FontMap{},
			Im:        model.ImageMap{},
			AnnotTabs: map[int]model.FieldAnnotation{},
			Buf:       new(bytes.Buffer),
		}

		if page == nil {
			if pageNr <= pdf.XRefTable.PageCount {
				pp = append(pp, nil)
				continue
			}

			// Create blank page with optional background color.
			if pdf.bgCol != nil {
				draw.FillRectNoBorder(p.Buf, p.CropBox, *pdf.bgCol)
			}

			// Render page header.
			if pdf.Header != nil {
				if err := pdf.Header.render(&p, pageNr, fontMap, imageMap, true); err != nil {
					return nil, nil, err
				}
			}

			// Render page footer.
			if pdf.Footer != nil {
				if err := pdf.Footer.render(&p, pageNr, fontMap, imageMap, false); err != nil {
					return nil, nil, err
				}
			}

			pp = append(pp, &p)
			continue
		}

		if page.cropBox != nil {
			cropBox = page.cropBox
		}

		pdf.renderPageBackground(page, p.Buf, cropBox)

		var headerHeight, headerDy float64
		var footerHeight, footerDy float64

		// Render page header.
		if pdf.Header != nil {
			if err := pdf.Header.render(&p, pageNr, fontMap, imageMap, true); err != nil {
				return nil, nil, err
			}
			headerHeight = pdf.Header.Height
			headerDy = float64(pdf.Header.Dy)
		}

		// Render page footer.
		if pdf.Footer != nil {
			if err := pdf.Footer.render(&p, pageNr, fontMap, imageMap, false); err != nil {
				return nil, nil, err
			}
			footerHeight = pdf.Footer.Height
			footerDy = float64(pdf.Footer.Dy)
		}

		// Render page content.
		r := cropBox.CroppedCopy(0)
		r.LL.Y += footerHeight + footerDy
		r.UR.Y -= headerHeight + headerDy
		page.Content.mediaBox = r
		if err := page.Content.render(&p, pageNr, fontMap, imageMap); err != nil {
			return nil, nil, err
		}

		pp = append(pp, &p)
	}

	return pp, fontMap, nil
}
