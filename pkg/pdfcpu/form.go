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

package pdfcpu

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pkg/errors"
)

var JSON = `{
	"paper": "A4",
	"bgCol": "#beded9",
	"inputFont" : {
		"name": "Courier",
		"size": 12,
		"col": "#beded9"
	},
	"labelFont" : {
		"name": "Courier",
		"size": 12,
		"col": "#beded9"
	},
	"textfields": [
		{
			"id": "firstName",
			"value": "Enter first name",
			"rect": [200, 200, 300, 220],
			"font" : {
				"name": "Courier",
				"size": 12,
				"col": "#beded9"
			},
			"border": {
				"width": 1,
				"col": "#FF0000"
			},
			"bgCol": "#beded9",
			"align": "right",
			"rtl": true,
			"label": {
				"value": "First Name:",
				"width": 100,
				"position": "left",
				"alignment": "right"
			}
		},
		{
			"id": "lastName",
			"value": "Enter last name",
			"rect": [200, 170, 300, 190]
		}
	]
}`

func parseHorAlignment(s string) (HAlignment, error) {
	var a HAlignment
	switch strings.ToLower(s) {
	case "l", "left":
		a = AlignLeft
	case "r", "right":
		a = AlignRight
	case "c", "center":
		a = AlignCenter
	default:
		return a, errors.Errorf("pdfcpu: unknown textfield alignment (left, center, right): %s", s)
	}
	return a, nil
}

// RelPosition represents the relative position of a text field's label.
type RelPosition int

// These are the options for relative label positions.
const (
	RelPosLeft RelPosition = iota
	RelPosRight
	RelPosTop
	RelPosBottom
)

func parseRelPosition(s string) (RelPosition, error) {
	var p RelPosition
	switch strings.ToLower(s) {
	case "l", "left":
		p = RelPosLeft
	case "r", "right":
		p = RelPosRight
	case "t", "top":
		p = RelPosRight
	case "b", "bottom":
		p = RelPosBottom
	default:
		return p, errors.Errorf("pdfcpu: unknown textfield alignment (left, right, top, bottom): %s", s)
	}
	return p, nil
}

type FormFont struct {
	Name  string
	Size  int
	Color string `json:"col"`
	col   *SimpleColor
}

func (f *FormFont) validate() error {

	if !font.SupportedFont(f.Name) {
		return errors.Errorf("pdfcpu: font %s is unsupported, please refer to \"pdfcpu fonts list\".\n", f.Name)
	}

	if f.Size <= 0 {
		return errors.Errorf("pdfcpu: invalid font size: %d", f.Size)
	}

	if f.Color != "" {
		sc, err := parseHexColor(f.Color)
		if err != nil {
			return err
		}
		f.col = &sc
	}

	return nil
}

type Border struct {
	Width int
	Color string `json:"col"`
	col   *SimpleColor
}

func (b *Border) validate() error {

	if b.Width < 0 {
		return errors.Errorf("pdfcpu: invalid border width: %d", b.Width)
	}

	if b.Color != "" {
		sc, err := parseHexColor(b.Color)
		if err != nil {
			return err
		}
		b.col = &sc
	}

	return nil
}

type TextfieldLabel struct {
	Textfield
	Width    int
	Position string // relative to textfield
	relPos   RelPosition
}

func (tfl *TextfieldLabel) validate() error {

	if tfl.Value == "" {
		return errors.New("pdfcpu: missing label value")
	}

	if tfl.Width <= 0 {
		return errors.Errorf("pdfcpu: invalid label width: %d", tfl.Width)
	}

	tfl.relPos = RelPosLeft
	if tfl.Position != "" {
		rp, err := parseRelPosition(tfl.Position)
		if err != nil {
			return err
		}
		tfl.relPos = rp
	}

	if tfl.Font != nil {
		if err := tfl.Font.validate(); err != nil {
			return err
		}
	}

	if tfl.Border != nil {
		if err := tfl.Border.validate(); err != nil {
			return err
		}
	}

	if tfl.BackgroundColor != "" {
		sc, err := parseHexColor(tfl.BackgroundColor)
		if err != nil {
			return err
		}
		tfl.bgCol = &sc
	}

	tfl.horAlign = AlignLeft
	if tfl.Alignment != "" {
		ha, err := parseHorAlignment(tfl.Alignment)
		if err != nil {
			return err
		}
		tfl.horAlign = ha
	}

	return nil
}

type Textfield struct {
	ID              string
	Value           string     // (Default) value or input data during extraction
	Rect            [4]float64 // xmin ymin xmax ymax
	boundingBox     *Rectangle
	Font            *FormFont
	Border          *Border
	BackgroundColor string
	bgCol           *SimpleColor
	Alignment       string // "Left", "Center", "Right"
	horAlign        HAlignment
	RTL             bool
	Label           *TextfieldLabel
}

func (tf *Textfield) validate() error {

	// Value

	r := tf.Rect
	if r[0] == 0 && r[1] == 0 && r[2] == 0 && r[3] == 0 {
		return errors.Errorf("pdfcpu: field: %s missing rect", tf.ID)
	}
	tf.boundingBox = Rect(r[0], r[1], r[2], r[3])

	if tf.Font != nil {
		if err := tf.Font.validate(); err != nil {
			return err
		}
	}

	if tf.Border != nil {
		if err := tf.Border.validate(); err != nil {
			return err
		}
	}

	if tf.BackgroundColor != "" {
		sc, err := parseHexColor(tf.BackgroundColor)
		if err != nil {
			return err
		}
		tf.bgCol = &sc
	}

	tf.horAlign = AlignLeft
	if tf.Alignment != "" {
		ha, err := parseHorAlignment(tf.Alignment)
		if err != nil {
			return err
		}
		tf.horAlign = ha
	}

	if tf.Label != nil {
		if err := tf.Label.validate(); err != nil {
			return err
		}
	}

	return nil
}

type Form struct {
	Paper           string
	mediaBox        *Rectangle
	BackgroundColor string `json:"bgCol"`
	bgCol           *SimpleColor
	InputFont       *FormFont
	LabelFont       *FormFont
	Textfields      []*Textfield // all input text fields with optional labels.
	fields          map[string]*Textfield
	// Labels
	// Textboxes
}

func (f *Form) validate() error {

	f.mediaBox = RectForFormat("A4")
	if f.Paper != "" {
		dim, _, err := parsePageFormat(f.Paper)
		if err != nil {
			return err
		}
		f.mediaBox = RectForDim(dim.Width, dim.Height)
	}

	if f.BackgroundColor != "" {
		sc, err := parseHexColor(f.BackgroundColor)
		if err != nil {
			return err
		}
		f.bgCol = &sc
	}

	if f.InputFont != nil {
		if err := f.InputFont.validate(); err != nil {
			return err
		}
	}

	if f.LabelFont != nil {
		if err := f.LabelFont.validate(); err != nil {
			return err
		}
	}

	for _, tf := range f.Textfields {
		if tf.ID == "" {
			return errors.New("pdfcpu: missing field id")
		}
		if f.fields[tf.ID] != nil {
			return errors.Errorf("pdfcpu: duplicate form field: %s", tf.ID)
		}
		f.fields[tf.ID] = tf
		if err := tf.validate(); err != nil {
			return err
		}
	}

	return nil
}

// FieldFlags represents the PDF form field flags.
type FieldFlags int

const ( // See table 221 et.al.
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

func parseForm(jsonString string) (*Form, error) {

	bb := []byte(jsonString)
	if !json.Valid(bb) {
		return nil, errors.Errorf("pdfcpu: invalid JSON encoding detected.")
	}

	form := &Form{fields: map[string]*Textfield{}}

	if err := json.Unmarshal(bb, form); err != nil {
		return nil, err
	}

	if err := form.validate(); err != nil {
		return nil, err
	}

	return form, nil
}

func createLabel(xRefTable *XRefTable, p *Page, tf *Textfield) error {

	if tf.Label == nil {
		return nil
	}

	l := tf.Label

	v := "Default"
	if l.Value != "" {
		v = l.Value // TODO ensure utf16
	}

	w := float64(l.Width)

	// if l.bgCol != nil {

	// }

	k := p.Fm.EnsureKey("Courier")

	td := TextDescriptor{
		FontName: "Courier",
		FontKey:  k,
		FontSize: 24,
		Scale:    1.,
		ScaleAbs: true,
		// RMode:     RMFill,
		// StrokeCol: Black,
		// FillCol:   Black,
		//ShowBackground: true,
		//BackgroundCol:  Gray,
		//ShowTextBB:     true,
	}

	if l.relPos == RelPosLeft {
		td.X, td.Y, td.HAlign, td.VAlign, td.Text = tf.boundingBox.LL.X-w, tf.boundingBox.LL.Y, AlignLeft, AlignBottom, v
	} else if l.relPos == RelPosRight {
		td.X, td.Y, td.HAlign, td.VAlign, td.Text = tf.boundingBox.LL.X, tf.boundingBox.LL.Y, AlignRight, AlignBottom, v
	}

	WriteColumn(p.Buf, p.MediaBox, nil, td, w)

	return nil
}

func createTextfield(xRefTable *XRefTable, tf *Textfield, pageAnnots *Array) error {

	// Acrobat positions to vertical middle and enforces single line entry.
	// Preview positions to UL corner of rect ignoring border and wraps into multi line text field.

	appearanceCharacteristicsDict := Dict(
		map[string]Object{
			//"BG": NewNumberArray(1, 0, 0),    // Backgroundcolor Acrobat only during editing, Preview always.
			"BC": NewNumberArray(.5, .5, .5), // Bordercolor
		},
	)

	d := Dict(
		map[string]Object{
			"FT":      Name("Tx"),
			"Rect":    tf.boundingBox.Array(),
			"Ff":      Integer(FieldDoNotSpellCheck), // + FieldMultiline), //) + FieldDoNotScroll), not in MacPreview - multiline for textarea
			"Border":  NewIntegerArray(0, 0, 1),      // BorderWidth Acrobat only: steady default background color blue gray
			"MK":      appearanceCharacteristicsDict,
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"Q":       Integer(0),           // Adjustment: (0:L) 1:C 2:R
			"T":       StringLiteral(tf.ID), // required
			"TU":      StringLiteral(tf.ID), // Acrobat Reader Hover over field
			//"DV":      StringLiteral("Default value"),
			//"V":       StringLiteral("Default value"),
			"DA": StringLiteral("/F1 24 Tf 0 0 1 rg"), // Mac Preview does not honor inherited "DA"
		},
	)

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	*pageAnnots = append(*pageAnnots, *ir)

	return nil
}

func createLabels(xRefTable *XRefTable, form *Form) (Page, error) {

	p := Page{MediaBox: form.mediaBox, Fm: FontMap{}, Buf: new(bytes.Buffer)}
	if form.bgCol != nil {
		FillRect(p.Buf, form.mediaBox, *form.bgCol)
	}

	for _, tf := range form.Textfields {
		if err := createLabel(xRefTable, &p, tf); err != nil {
			return p, err
		}
	}

	DrawHairCross(p.Buf, 0, 0, p.MediaBox)

	return p, nil
}

func createForm(xRefTable *XRefTable, form *Form) (Dict, Array, error) {

	annots := Array{}

	for _, tf := range form.Textfields {
		if err := createTextfield(xRefTable, tf, &annots); err != nil {
			return nil, nil, err
		}
	}

	fontDict, err := createFontDict(xRefTable, "Courier")
	if err != nil {
		return nil, nil, err
	}

	resourceDict := Dict(
		map[string]Object{
			"Font": Dict(
				map[string]Object{
					"F1": *fontDict,
				},
			),
		},
	)

	d := Dict(
		map[string]Object{
			"Fields": annots,
			"DA":     StringLiteral("/F1 24 Tf 0 0 1 rg"),
			"DR":     resourceDict,
		},
	)

	return d, annots, nil
}

func createFormPage(xRefTable *XRefTable, parentPageIndRef IndirectRef, p Page, annotsArray Array) (*IndirectRef, error) {

	pageDict := Dict(
		map[string]Object{
			"Type":   Name("Page"),
			"Parent": parentPageIndRef,
		},
	)

	fontRes, err := fontResources(xRefTable, p.Fm)
	if err != nil {
		return nil, err
	}

	if len(fontRes) > 0 {
		resDict := Dict(
			map[string]Object{
				"Font": fontRes,
			},
		)
		pageDict.Insert("Resources", resDict)
	}

	ir, err := createDemoContentStreamDict(xRefTable, pageDict, p.Buf.Bytes())
	if err != nil {
		return nil, err
	}
	pageDict.Insert("Contents", *ir)

	pageDict.Insert("Annots", annotsArray)

	return xRefTable.IndRefForNewObject(pageDict)
}

func addPageTreeWithFormFields(xRefTable *XRefTable, rootDict Dict, p Page, annots Array) (*IndirectRef, error) {

	pagesDict := Dict(
		map[string]Object{
			"Type":     Name("Pages"),
			"Count":    Integer(1),
			"MediaBox": p.MediaBox.Array(),
		},
	)

	parentPageIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return nil, err
	}

	pageIndRef, err := createFormPage(xRefTable, *parentPageIndRef, p, annots)
	if err != nil {
		return nil, err
	}

	pagesDict.Insert("Kids", Array{*pageIndRef})

	rootDict.Insert("Pages", *parentPageIndRef)

	return pageIndRef, nil
}

// Create a PDF with a single page form with 1 text field.
func CreateFormXRef() (*XRefTable, error) {

	form, err := parseForm(JSON)
	if err != nil {
		return nil, err
	}

	xRefTable, err := createXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	p, err := createLabels(xRefTable, form)
	if err != nil {
		return nil, err
	}

	formDict, annots, err := createForm(xRefTable, form)
	if err != nil {
		return nil, err
	}

	rootDict.Insert("AcroForm", formDict)

	_, err = addPageTreeWithFormFields(xRefTable, rootDict, p, annots)
	if err != nil {
		return nil, err
	}

	return xRefTable, nil
}
