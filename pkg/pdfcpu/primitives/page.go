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
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

type PDFPage struct {
	pdf             *PDF
	number          int                  // page number
	Paper           string               // page size
	mediaBox        *pdfcpu.Rectangle    // page media box
	Crop            string               // page crop box
	cropBox         *pdfcpu.Rectangle    // page crop box
	BackgroundColor string               `json:"bgCol"`
	bgCol           *pdfcpu.SimpleColor  // page background color
	Fonts           map[string]*FormFont // default fonts
	DA              pdfcpu.Object
	Guides          []*Guide              // hor/vert guidelines for layout
	Margin          *Margin               // page margin
	Border          *Border               // page border
	Padding         *Padding              // page padding
	Margins         map[string]*Margin    // page scoped named margins
	Borders         map[string]*Border    // page scoped named borders
	Paddings        map[string]*Padding   // page scoped named paddings
	SimpleBoxPool   map[string]*SimpleBox `json:"boxes"`
	TextBoxPool     map[string]*TextBox   `json:"texts"`
	ImageBoxPool    map[string]*ImageBox  `json:"images"`
	TablePool       map[string]*Table     `json:"tables"`
	FileNames       map[string]string     `json:"files"`
	Content         *Content
}

func (page *PDFPage) resolveFileName(s string) (string, error) {
	filePath, ok := page.FileNames[s]
	if !ok {
		return page.pdf.resolveFileName(s)
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

	dirPath, ok := page.pdf.DirNames[dirName]
	if !ok {
		return "", errors.Errorf("pdfcpu: can't resolve dirname: %s", dirName)
	}

	s1 := filepath.Join(dirPath, fileName)

	return s1, nil
}

func (page *PDFPage) validate() error {
	pdf := page.pdf
	page.mediaBox = pdf.mediaBox
	page.cropBox = pdf.cropBox
	if page.Paper != "" {
		dim, _, err := pdfcpu.ParsePageFormat(page.Paper)
		if err != nil {
			return err
		}
		page.mediaBox = pdfcpu.RectForDim(dim.Width, dim.Height)
		page.cropBox = page.mediaBox.CroppedCopy(0)
	}

	if page.Crop != "" {
		box, err := pdfcpu.ParseBox(page.Crop, pdfcpu.POINTS)
		if err != nil {
			return err
		}
		page.cropBox = pdfcpu.ApplyBox("CropBox", box, nil, page.mediaBox)
	}

	// Default background color
	if page.BackgroundColor != "" {
		sc, err := page.pdf.parseColor(page.BackgroundColor)
		if err != nil {
			return err
		}
		page.bgCol = sc
	}

	// Default page fonts
	for _, f := range page.Fonts {
		f.pdf = pdf
		if err := f.validate(); err != nil {
			return err
		}
	}

	for _, g := range page.Guides {
		g.validate()
	}

	if page.Border != nil {
		if len(page.Borders) > 0 {
			return errors.New("pdfcpu: Please supply either page \"border\" or \"borders\"")
		}
		page.Border.pdf = pdf
		if err := page.Border.validate(); err != nil {
			return err
		}
		page.Borders = map[string]*Border{}
		page.Borders["border"] = page.Border
	}
	for _, b := range page.Borders {
		b.pdf = pdf
		if err := b.validate(); err != nil {
			return err
		}
	}

	if page.Margin != nil {
		if len(page.Margins) > 0 {
			return errors.New("pdfcpu: Please supply either page \"margin\" or \"margins\"")
		}
		if err := page.Margin.validate(); err != nil {
			return err
		}
		page.Margins = map[string]*Margin{}
		page.Margins["margin"] = page.Margin
	}
	for _, m := range page.Margins {
		if err := m.validate(); err != nil {
			return err
		}
	}

	if page.Padding != nil {
		if len(page.Paddings) > 0 {
			return errors.New("pdfcpu: Please supply either page \"padding\" or \"paddings\"")
		}
		if err := page.Padding.validate(); err != nil {
			return err
		}
		page.Paddings = map[string]*Padding{}
		page.Paddings["padding"] = page.Padding
	}
	for _, p := range page.Paddings {
		if err := p.validate(); err != nil {
			return err
		}
	}

	// box templates
	for _, sb := range page.SimpleBoxPool {
		sb.pdf = pdf
		if err := sb.validate(); err != nil {
			return err
		}
	}

	// text templates
	for _, tb := range page.TextBoxPool {
		tb.pdf = pdf
		if err := tb.validate(); err != nil {
			return err
		}
	}

	// image templates
	for _, ib := range page.ImageBoxPool {
		ib.pdf = pdf
		if err := ib.validate(); err != nil {
			return err
		}
	}

	// table templates
	for _, t := range page.TablePool {
		t.pdf = pdf
		if err := t.validate(); err != nil {
			return err
		}
	}

	if page.Content == nil {
		return errors.New("pdfcpu: Please supply page \"content\"")
	}
	page.Content.page = page
	return page.Content.validate()
}

func (page *PDFPage) namedFont(id string) *FormFont {
	f := page.Fonts[id]
	if f != nil {
		return f
	}
	return page.pdf.Fonts[id]
}

func (page *PDFPage) namedMargin(id string) *Margin {
	m := page.Margins[id]
	if m != nil {
		return m
	}
	return page.pdf.Margins[id]
}

func (page *PDFPage) namedBorder(id string) *Border {
	b := page.Borders[id]
	if b != nil {
		return b
	}
	return page.pdf.Borders[id]
}

func (page *PDFPage) namedPadding(id string) *Padding {
	p := page.Paddings[id]
	if p != nil {
		return p
	}
	return page.pdf.Paddings[id]
}

func (page *PDFPage) namedSimpleBox(id string) *SimpleBox {
	sb := page.SimpleBoxPool[id]
	if sb != nil {
		return sb
	}
	return page.pdf.SimpleBoxPool[id]
}

func (page *PDFPage) namedImageBox(id string) *ImageBox {
	tb := page.ImageBoxPool[id]
	if tb != nil {
		return tb
	}
	return page.pdf.ImageBoxPool[id]
}

func (page *PDFPage) namedTextBox(id string) *TextBox {
	tb := page.TextBoxPool[id]
	if tb != nil {
		return tb
	}
	return page.pdf.TextBoxPool[id]
}

func (page *PDFPage) namedTable(id string) *Table {
	t := page.TablePool[id]
	if t != nil {
		return t
	}
	return page.pdf.TablePool[id]
}
