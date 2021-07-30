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
	"fmt"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pkg/errors"
)

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
		p = RelPosTop
	case "b", "bottom":
		p = RelPosBottom
	default:
		return p, errors.Errorf("pdfcpu: unknown textfield alignment (left, right, top, bottom): %s", s)
	}
	return p, nil
}

// Refactor because of orientation in nup.go
type Orientation int

const (
	Horizontal Orientation = iota
	Vertical
)

func parseRadioButtonOrientation(s string) (Orientation, error) {
	var o Orientation
	switch strings.ToLower(s) {
	case "h", "hor", "horizontal":
		o = Horizontal
	case "v", "vert", "vertical":
		o = Vertical
	default:
		return o, errors.Errorf("pdfcpu: unknown radiobutton orientation (hor, vert): %s", s)
	}
	return o, nil
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

type TextFieldLabel struct {
	TextField
	Width    int
	Gap      int    // horizontal space between textfield and label
	Position string `json:"pos"` // relative to textfield
	relPos   RelPosition
}

func (tfl *TextFieldLabel) validate() error {

	if tfl.Value == "" {
		return errors.New("pdfcpu: missing label value")
	}

	if tfl.Width <= 0 {
		// only for pos left align left or pos right align right!
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

type TextBox struct {
	Value           string     // text, content
	Position        [2]float64 `json:"pos"` // x,y
	x, y            float64
	Width           float64
	Font            *FormFont
	Border          *Border
	BackgroundColor string `json:"bgCol"`
	bgCol           *SimpleColor
	Alignment       string `json:"align"` // "Left", "Center", "Right"
	horAlign        HAlignment
	RTL             bool
}

func (tb *TextBox) validate() error {

	tb.x = tb.Position[0]
	tb.y = tb.Position[1]

	if tb.Font == nil {
		return errors.New("pdfcpu: textbox missing font definition")
	}
	if err := tb.Font.validate(); err != nil {
		return err
	}

	if tb.Border != nil {
		if err := tb.Border.validate(); err != nil {
			return err
		}
	}

	if tb.BackgroundColor != "" {
		sc, err := parseHexColor(tb.BackgroundColor)
		if err != nil {
			return err
		}
		tb.bgCol = &sc
	}

	tb.horAlign = AlignLeft
	if tb.Alignment != "" {
		ha, err := parseHorAlignment(tb.Alignment)
		if err != nil {
			return err
		}
		tb.horAlign = ha
	}

	return nil
}

type TextField struct {
	ID              string
	Value           string     // (Default) value or input data during extraction
	Rect            [4]float64 // xmin ymin xmax ymax
	boundingBox     *Rectangle
	Multiline       bool
	Font            *FormFont
	Border          *Border
	BackgroundColor string `json:"bgCol"`
	bgCol           *SimpleColor
	Alignment       string `json:"align"` // "Left", "Center", "Right"
	horAlign        HAlignment
	RTL             bool
	Label           *TextFieldLabel
}

func (tf *TextField) validate() error {

	// TODO validate value: Numeric, Date...

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

type CheckBox struct {
	ID              string
	Value           bool       // checked state
	Position        [2]float64 `json:"pos"` // x,y
	x, y            float64
	Width           float64
	Font            *FormFont
	Border          *Border
	BackgroundColor string `json:"bgCol"`
	bgCol           *SimpleColor
	Label           *TextFieldLabel
}

func (cb *CheckBox) boundingBox() *Rectangle {
	w := 12
	if cb.Font != nil {
		w = cb.Font.Size
	} else if cb.Label != nil && cb.Label.Font != nil {
		w = cb.Label.Font.Size
	}
	return RectForWidthAndHeight(cb.x, cb.y, float64(w), float64(w))
}

func (cb *CheckBox) validate() error {

	cb.x = cb.Position[0]
	cb.y = cb.Position[1]

	if cb.Font != nil {
		if err := cb.Font.validate(); err != nil {
			return err
		}
	}

	if cb.Border != nil {
		if err := cb.Border.validate(); err != nil {
			return err
		}
	}

	if cb.BackgroundColor != "" {
		sc, err := parseHexColor(cb.BackgroundColor)
		if err != nil {
			return err
		}
		cb.bgCol = &sc
	}

	if cb.Label != nil {
		if err := cb.Label.validate(); err != nil {
			return err
		}
	}

	return nil
}

type Buttons struct {
	Values []string
	Label  *TextFieldLabel
}

func (b *Buttons) validate() error {

	if len(b.Values) < 2 {
		return errors.New("pdfcpu: radiobuttongroups.buttons missing values")
	}

	if b.Label == nil {
		return errors.New("pdfcpu: radiobuttongroups.buttons: missing label")
	}

	if err := b.Label.validate(); err != nil {
		return err
	}

	pos := b.Label.relPos
	if pos == RelPosTop || pos == RelPosBottom {
		return errors.New("pdfcpu: radiobuttongroups.buttons.label: pos must be left or right")
	}

	b.Label.horAlign = AlignLeft
	if pos == RelPosLeft {
		// A radio button label on the left side of a radio button is right aligned.
		b.Label.horAlign = AlignRight
	}

	return nil
}

func (b *Buttons) maxLabelWidth(hor bool) (float64, float64) {
	maxw, lastw := 0.0, 0.0
	fontName := b.Label.Font.Name
	fontSize := b.Label.Font.Size
	for i, v := range b.Values {
		td := TextDescriptor{
			Text:     v,
			FontName: fontName,
			FontSize: fontSize,
			Scale:    1.,
			ScaleAbs: true,
		}
		bb := WriteMultiLine(new(bytes.Buffer), RectForFormat("A4"), nil, td)
		if hor {
			if b.Label.horAlign == AlignLeft {
				// Leave last label width like it is.
				if i == len(b.Values)-1 {
					lastw = maxw
					if bb.Width() > maxw {
						lastw = bb.Width()
					}
					continue
				}
			}
			if b.Label.horAlign == AlignRight {
				// Leave first label width like it is.
				if i == 0 {
					lastw = bb.Width()
					continue
				}
			}
		}
		if bb.Width() > maxw {
			maxw = bb.Width()
		}
	}
	if b.Label.horAlign == AlignRight {
		// This is actually the width of the first (left most) label in this case.
		if lastw < maxw {
			lastw = maxw
		}
	}
	return maxw, lastw
}

type RadioButtonGroup struct {
	ID          string
	Value       string // checked button
	Orientation string
	Position    [2]float64 `json:"pos"` // x,y
	x, y        float64
	hor         bool
	Buttons     *Buttons
	//Width           float64
	//Font            *FormFont
	//Border          *Border
	//BackgroundColor string `json:"bgCol"`
	//bgCol           *SimpleColor
	Label *TextFieldLabel
}

func (rbg *RadioButtonGroup) buttonLabelPosition(i int, maxWidth, firstWidth float64) (float64, float64) {
	rbw := float64(rbg.Buttons.Label.Font.Size)
	g := float64(rbg.Buttons.Label.Gap)
	w := float64(rbg.Buttons.Label.Width)

	if rbg.hor {
		if maxWidth+g > w {
			w = maxWidth + g
		}
		var x float64
		if rbg.Buttons.Label.horAlign == AlignLeft {
			x = rbg.x + float64(i)*(rbw+w) + rbw
		}
		if rbg.Buttons.Label.horAlign == AlignRight {
			x = rbg.x + firstWidth
			if i > 0 {
				x += float64(i) * (rbw + w)
			}
			//x -= 3
		}
		return x, rbg.y
	}

	if maxWidth > w {
		w = maxWidth
	}
	dx := rbw
	if rbg.Buttons.Label.horAlign == AlignRight {
		dx = w
	}
	dy := float64(i) * (rbw + g)
	return rbg.x + dx, rbg.y - dy
}

func (rbg *RadioButtonGroup) rect(i, maxWidth, firstWidth float64) *Rectangle {
	rbw := float64(rbg.Buttons.Label.Font.Size)
	g := float64(rbg.Buttons.Label.Gap)
	w := float64(rbg.Buttons.Label.Width)

	if rbg.hor {
		if maxWidth+g > w {
			w = maxWidth + g
		}
		var x float64
		if rbg.Buttons.Label.horAlign == AlignLeft {
			x = rbg.x + i*(rbw+w)
		}
		if rbg.Buttons.Label.horAlign == AlignRight {
			x = rbg.x + firstWidth
			if i > 0 {
				x += i * (rbw + w)
			}
		}
		return RectForWidthAndHeight(x, rbg.y, rbw, rbw)
	}
	if maxWidth > w {
		w = maxWidth
	}
	dx := 0.
	if rbg.Buttons.Label.horAlign == AlignRight {
		dx = w
	}
	dy := i * (rbw + g)
	return RectForWidthAndHeight(rbg.x+dx, rbg.y-dy, rbw, rbw)
}

func (rbg *RadioButtonGroup) boundingBox() *Rectangle {
	maxWidth, lastWidth := rbg.Buttons.maxLabelWidth(rbg.hor)
	g := float64(rbg.Buttons.Label.Gap)
	w := float64(rbg.Buttons.Label.Width)

	rbSize := 12.
	rbCount := float64(len(rbg.Buttons.Values))

	if rbg.hor {
		if maxWidth+g > w {
			w = maxWidth + g
		}

		width := (rbCount-1)*(w+rbSize) + rbSize + lastWidth
		// if rbg.Buttons.Label.horAlign == AlignRight {
		// 	width += 3
		// }
		return RectForWidthAndHeight(rbg.x, rbg.y, width, rbSize)
	}

	if maxWidth > w {
		w = maxWidth
	}
	y := rbg.y - (rbCount-1)*(rbSize+g) // g is better smth derived from fontsize
	h := rbSize + (rbCount-1)*(rbSize+g)

	return RectForWidthAndHeight(rbg.x, y, w+rbSize, h)
}

func (rbg *RadioButtonGroup) validate() error {

	rbg.x = rbg.Position[0]
	rbg.y = rbg.Position[1]

	rbg.hor = true
	if rbg.Orientation != "" {
		o, err := parseRadioButtonOrientation(rbg.Orientation)
		if err != nil {
			return err
		}
		rbg.hor = o == Horizontal
	}

	// if cb.Font != nil {
	// 	if err := cb.Font.validate(); err != nil {
	// 		return err
	// 	}
	// }

	// if cb.Border != nil {
	// 	if err := cb.Border.validate(); err != nil {
	// 		return err
	// 	}
	// }

	// if cb.BackgroundColor != "" {
	// 	sc, err := parseHexColor(cb.BackgroundColor)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	cb.bgCol = &sc
	// }

	if rbg.Label != nil {
		if err := rbg.Label.validate(); err != nil {
			return err
		}
	}

	if rbg.Buttons == nil {
		return errors.New("pdfcpu: radiobutton missing buttons")
	}

	return rbg.Buttons.validate()
}

type ListBox interface {
	Id() string
	BoundingBox() *Rectangle
	label() *TextFieldLabel
	Editable() bool
	MultiSelect() bool
	Opts() []string
	values() []string
	border() *Border
	backgroundColor() *SimpleColor
}

type ScrollableListBox struct {
	ID              string
	Value           string
	Values          []string
	Options         []string
	Rect            [4]float64 // xmin ymin xmax ymax
	boundingBox     *Rectangle
	Multi           bool `json:"multi"`
	Font            *FormFont
	Border          *Border
	BackgroundColor string `json:"bgCol"`
	bgCol           *SimpleColor
	Label           *TextFieldLabel
}

func (lb *ScrollableListBox) Id() string {
	return lb.ID
}

func (lb *ScrollableListBox) BoundingBox() *Rectangle {
	return lb.boundingBox
}

func (lb *ScrollableListBox) Editable() bool {
	return false
}

func (lb *ScrollableListBox) MultiSelect() bool {
	return lb.Multi
}

func (lb *ScrollableListBox) Opts() []string {
	return lb.Options
}

func (lb *ScrollableListBox) label() *TextFieldLabel {
	return lb.Label
}

func (lb *ScrollableListBox) values() []string {
	return lb.Values
}

func (lb *ScrollableListBox) border() *Border {
	return lb.Border
}

func (lb *ScrollableListBox) backgroundColor() *SimpleColor {
	return lb.bgCol
}

func (lb *ScrollableListBox) validateValues() error {

	vv := []string{}
	if lb.Value != "" {
		vv = append(vv, lb.Value)
	}
	for _, v1 := range lb.Values {
		if !MemberOf(v1, vv) {
			vv = append(vv, v1)
		}
	}
	if len(vv) == 0 {
		return nil
	}

	if !lb.MultiSelect() && len(vv) > 1 {
		return errors.Errorf("pdfcpu: field %s only 1 value allowed", lb.ID)
	}

	for _, s := range vv {
		if !MemberOf(s, lb.Options) {
			return errors.Errorf("pdfcpu: field: %s invalid value", lb.ID, s)
		}
	}

	lb.Values = vv

	return nil
}

func (lb *ScrollableListBox) validate() error {

	r := lb.Rect
	if r[0] == 0 && r[1] == 0 && r[2] == 0 && r[3] == 0 {
		return errors.Errorf("pdfcpu: field: %s missing rect", lb.ID)
	}
	lb.boundingBox = Rect(r[0], r[1], r[2], r[3])

	if len(lb.Options) == 0 {
		return errors.Errorf("pdfcpu: field: %s missing options", lb.ID)
	}

	if err := lb.validateValues(); err != nil {
		return err
	}

	if lb.Font != nil {
		if err := lb.Font.validate(); err != nil {
			return err
		}
	}

	if lb.Border != nil {
		if err := lb.Border.validate(); err != nil {
			return err
		}
	}

	if lb.BackgroundColor != "" {
		sc, err := parseHexColor(lb.BackgroundColor)
		if err != nil {
			return err
		}
		lb.bgCol = &sc
	}

	if lb.Label != nil {
		if err := lb.Label.validate(); err != nil {
			return err
		}
	}

	return nil
}

type ComboBox struct {
	ID              string
	Value           string
	Options         []string
	Position        [2]float64 `json:"pos"` // x,y
	x, y            float64
	Width           float64
	Edit            bool
	Font            *FormFont
	Border          *Border
	BackgroundColor string `json:"bgCol"`
	bgCol           *SimpleColor
	Label           *TextFieldLabel
}

func (lb *ComboBox) Id() string {
	return lb.ID
}

func (lb *ComboBox) Editable() bool {
	return lb.Edit
}

func (lb *ComboBox) MultiSelect() bool {
	return false
}

func (lb *ComboBox) Opts() []string {
	return lb.Options
}

func (cb *ComboBox) BoundingBox() *Rectangle {
	w := 12
	if cb.Font != nil {
		w = cb.Font.Size
	} else if cb.Label != nil && cb.Label.Font != nil {
		w = cb.Label.Font.Size
	}
	return RectForWidthAndHeight(cb.x, cb.y, cb.Width, float64(w))
}

func (cb *ComboBox) label() *TextFieldLabel {
	return cb.Label
}

func (cb *ComboBox) border() *Border {
	return cb.Border
}

func (cb *ComboBox) backgroundColor() *SimpleColor {
	return cb.bgCol
}

func (cb *ComboBox) values() []string {
	return []string{cb.Value}
}

func (cb *ComboBox) validate() error {

	cb.x = cb.Position[0]
	cb.y = cb.Position[1]

	if len(cb.Options) == 0 {
		return errors.Errorf("pdfcpu: field: %s missing options", cb.ID)
	}

	if len(cb.Value) > 0 && !MemberOf(cb.Value, cb.Options) {
		return errors.Errorf("pdfcpu: field: %s invalid value", cb.ID, cb.Value)
	}

	if cb.Font != nil {
		if err := cb.Font.validate(); err != nil {
			return err
		}
	}

	if cb.Border != nil {
		if err := cb.Border.validate(); err != nil {
			return err
		}
	}

	if cb.BackgroundColor != "" {
		sc, err := parseHexColor(cb.BackgroundColor)
		if err != nil {
			return err
		}
		cb.bgCol = &sc
	}

	if cb.Label != nil {
		if err := cb.Label.validate(); err != nil {
			return err
		}
	}

	return nil
}

type FontResource struct {
	resID  string
	indRef IndirectRef
}

type Form struct {
	Paper             string
	mediaBox          *Rectangle
	BackgroundColor   string `json:"bgCol"`
	bgCol             *SimpleColor
	InputFont         *FormFont
	LabelFont         *FormFont
	TextFields        []*TextField         // input text fields with optional label
	TextBoxes         []*TextBox           // plain textboxes
	CheckBoxes        []*CheckBox          // input checkboxes with optional label
	RadioButtonGroups []*RadioButtonGroup  // input radiobutton groups with optional label
	ListBoxes         []*ScrollableListBox // input listboxes with optional label and multi selection
	ComboBoxes        []*ComboBox          // input comboboxes with optional label and editable.
	fields            StringSet
	annots            Array
	fonts             map[string]FontResource
	DA                Object
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

	for _, tf := range f.TextFields {
		if tf.ID == "" {
			return errors.New("pdfcpu: missing field id")
		}
		if f.fields[tf.ID] {
			return errors.Errorf("pdfcpu: duplicate form field: %s", tf.ID)
		}
		f.fields[tf.ID] = true
		if err := tf.validate(); err != nil {
			return err
		}
	}

	for _, cb := range f.CheckBoxes {
		if cb.ID == "" {
			return errors.New("pdfcpu: missing field id")
		}
		if f.fields[cb.ID] {
			return errors.Errorf("pdfcpu: duplicate form field: %s", cb.ID)
		}
		f.fields[cb.ID] = true
		if err := cb.validate(); err != nil {
			return err
		}
	}

	for _, rbg := range f.RadioButtonGroups {
		if rbg.ID == "" {
			return errors.New("pdfcpu: missing field id")
		}
		if f.fields[rbg.ID] {
			return errors.Errorf("pdfcpu: duplicate form field: %s", rbg.ID)
		}
		f.fields[rbg.ID] = true
		if err := rbg.validate(); err != nil {
			return err
		}
	}

	for _, lb := range f.ListBoxes {
		if lb.ID == "" {
			return errors.New("pdfcpu: missing field id")
		}
		if f.fields[lb.ID] {
			return errors.Errorf("pdfcpu: duplicate form field: %s", lb.ID)
		}
		f.fields[lb.ID] = true
		if err := lb.validate(); err != nil {
			return err
		}
	}

	for _, cb := range f.ComboBoxes {
		if cb.ID == "" {
			return errors.New("pdfcpu: missing field id")
		}
		if f.fields[cb.ID] {
			return errors.Errorf("pdfcpu: duplicate form field: %s", cb.ID)
		}
		f.fields[cb.ID] = true
		if err := cb.validate(); err != nil {
			return err
		}
	}

	for _, tb := range f.TextBoxes {
		if err := tb.validate(); err != nil {
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

func parseForm(bb []byte) (*Form, error) {

	if !json.Valid(bb) {
		return nil, errors.Errorf("pdfcpu: invalid JSON encoding detected.")
	}

	form := &Form{
		fields: StringSet{},
		fonts:  map[string]FontResource{},
	}

	if err := json.Unmarshal(bb, form); err != nil {
		return nil, err
	}

	if err := form.validate(); err != nil {
		return nil, err
	}

	return form, nil
}

func labelPosition(relPos RelPosition, horAlign HAlignment, boundingBox *Rectangle, labelHeight, w, g float64, multiline bool) (float64, float64) {
	var x, y float64

	switch relPos {

	case RelPosLeft:
		x = boundingBox.LL.X - g
		if horAlign == AlignLeft {
			x -= w
			if x < 0 {
				x = 0
			}
		}
		if multiline {
			y = boundingBox.UR.Y - labelHeight
		} else {
			y = boundingBox.LL.Y
		}

	case RelPosRight:
		x = boundingBox.UR.X + g
		if horAlign == AlignRight {
			x += w
		}
		if multiline {
			y = boundingBox.UR.Y - labelHeight
		} else {
			y = boundingBox.LL.Y
		}

	case RelPosTop:
		y = boundingBox.UR.Y + g
		x = boundingBox.LL.X
		if horAlign == AlignRight {
			x += boundingBox.Width()
		} else if horAlign == AlignCenter {
			x += boundingBox.Width() / 2
		}

	case RelPosBottom:
		y = boundingBox.LL.Y - g - labelHeight
		x = boundingBox.LL.X
		if horAlign == AlignRight {
			x += boundingBox.Width()
		} else if horAlign == AlignCenter {
			x += boundingBox.Width() / 2
		}

	}

	return x, y
}

func createTextFieldLabel(tf *TextField, p *Page, font *FormFont) error {

	if tf.Label == nil {
		return nil
	}

	l := tf.Label
	v := "Default"
	if l.Value != "" {
		v = l.Value
	}

	w := float64(l.Width)
	g := float64(l.Gap)

	if l.Font == nil && font == nil {
		return errors.New("pdfcpu: missing label font")
	}

	var (
		fontSize int
		fontName string
		col      *SimpleColor
	)

	if l.Font != nil {
		fontName = l.Font.Name
		fontSize = l.Font.Size
		col = l.Font.col
	} else {
		fontName = font.Name
		fontSize = font.Size
		col = font.col
	}

	k := p.Fm.EnsureKey(fontName)

	td := TextDescriptor{
		Text:     v,
		FontName: fontName,
		FontKey:  k,
		FontSize: fontSize,
		Scale:    1.,
		ScaleAbs: true,
		RTL:      l.RTL, // for user fonts only!
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if l.bgCol != nil {
		//td.ShowBorder = true
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *l.bgCol
	}

	td.MBot = 2

	// calc real label height
	bb := WriteMultiLine(new(bytes.Buffer), RectForFormat("A4"), nil, td)

	td.X, td.Y = labelPosition(l.relPos, l.horAlign, tf.boundingBox, bb.Height(), w, g, tf.Multiline)

	td.HAlign, td.VAlign = l.horAlign, AlignBottom

	WriteColumn(p.Buf, p.MediaBox, nil, td, 0)

	return nil
}

func createCheckBoxLabel(cb *CheckBox, p *Page, font *FormFont) error {

	if cb.Label == nil {
		return nil
	}

	l := cb.Label
	v := "Default"
	if l.Value != "" {
		v = l.Value
	}

	w := float64(l.Width)
	g := float64(l.Gap)

	if l.Font == nil && font == nil {
		return errors.New("pdfcpu: missing label font")
	}

	var (
		fontSize int
		fontName string
		col      *SimpleColor
	)

	if l.Font != nil {
		fontName = l.Font.Name
		fontSize = l.Font.Size
		col = l.Font.col
	} else {
		fontName = font.Name
		fontSize = font.Size
		col = font.col
	}

	k := p.Fm.EnsureKey(fontName)

	td := TextDescriptor{
		Text:     v,
		FontName: fontName,
		FontKey:  k,
		FontSize: fontSize,
		Scale:    1.,
		ScaleAbs: true,
		RTL:      l.RTL, // for user fonts only!
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if l.bgCol != nil {
		//td.ShowBorder = true
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *l.bgCol
	}

	boundingBox := cb.boundingBox()

	bb := WriteMultiLine(new(bytes.Buffer), RectForFormat("A4"), nil, td)

	if l.relPos == RelPosLeft {
		td.X = boundingBox.LL.X - g
		if l.horAlign == AlignLeft {
			td.X -= w
		}
		td.Y = boundingBox.LL.Y
	} else if l.relPos == RelPosRight {
		td.X = boundingBox.UR.X + g
		if l.horAlign == AlignRight {
			td.X += w
		}
		td.Y = boundingBox.LL.Y
	} else if l.relPos == RelPosTop {
		td.Y = boundingBox.UR.Y + g
		td.X = boundingBox.LL.X
		if l.horAlign == AlignRight {
			td.X += boundingBox.Width()
		} else if l.horAlign == AlignCenter {
			td.X += boundingBox.Width() / 2
		}
	} else if l.relPos == RelPosBottom {
		td.Y = boundingBox.LL.Y - g - bb.Height()
		td.X = boundingBox.LL.X
		if l.horAlign == AlignRight {
			td.X += boundingBox.Width()
		} else if l.horAlign == AlignCenter {
			td.X += boundingBox.Width() / 2
		}
	}

	td.HAlign, td.VAlign = l.horAlign, AlignBottom

	WriteColumn(p.Buf, p.MediaBox, nil, td, 0)

	return nil
}

func createRadioButtonLabels(rbg *RadioButtonGroup, p *Page, font *FormFont) error {

	l := rbg.Buttons.Label

	var (
		fontSize int
		fontName string
		col      *SimpleColor
	)

	if l.Font != nil {
		fontName = l.Font.Name
		fontSize = l.Font.Size
		col = l.Font.col
		rbg.Buttons.Label.Font = l.Font
	} else {
		fontName = font.Name
		fontSize = font.Size
		col = font.col
		rbg.Buttons.Label.Font = font
	}

	k := p.Fm.EnsureKey(fontName)

	td := TextDescriptor{
		FontName: fontName,
		FontKey:  k,
		FontSize: fontSize,
		Scale:    1.,
		ScaleAbs: true,
		RTL:      l.RTL, // for user fonts only!
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if l.bgCol != nil {
		//td.ShowBorder = true
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *l.bgCol
	}

	w, firstw := rbg.Buttons.maxLabelWidth(rbg.hor)

	td.HAlign = l.horAlign

	for i, v := range rbg.Buttons.Values {
		td.Text = v
		td.X, td.Y = rbg.buttonLabelPosition(i, w, firstw)
		WriteColumn(p.Buf, p.MediaBox, nil, td, 0)
	}

	return nil
}

func createRadioButtonGroupLabels(rbg *RadioButtonGroup, p *Page, font *FormFont) error {

	createRadioButtonLabels(rbg, p, font)

	// Main label:
	if rbg.Label == nil {
		return nil
	}

	l := rbg.Label
	v := "Default"
	if l.Value != "" {
		v = l.Value
	}

	w := float64(l.Width)
	g := float64(l.Gap)

	if l.Font == nil && font == nil {
		return errors.New("pdfcpu: missing label font")
	}

	var (
		fontSize int
		fontName string
		col      *SimpleColor
	)

	if l.Font != nil {
		fontName = l.Font.Name
		fontSize = l.Font.Size
		col = l.Font.col
	} else {
		fontName = font.Name
		fontSize = font.Size
		col = font.col
	}

	k := p.Fm.EnsureKey(fontName)

	td := TextDescriptor{
		Text:     v,
		FontName: fontName,
		FontKey:  k,
		FontSize: fontSize,
		HAlign:   l.horAlign,
		Scale:    1.,
		ScaleAbs: true,
		RTL:      l.RTL, // for user fonts only!
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if l.bgCol != nil {
		//td.ShowBorder = true
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *l.bgCol
	}

	// calc real label height
	bb := WriteMultiLine(new(bytes.Buffer), RectForFormat("A4"), nil, td)

	buttonGroupBB := rbg.boundingBox()

	td.X, td.Y = labelPosition(l.relPos, l.horAlign, buttonGroupBB, bb.Height(), w, g, !rbg.hor)

	WriteColumn(p.Buf, p.MediaBox, nil, td, 0)

	return nil
}

func createListBoxLabel(lb ListBox, p *Page, font *FormFont) error {

	if lb.label() == nil {
		return nil
	}

	l := lb.label()
	v := "Default"
	if l.Value != "" {
		v = l.Value
	}

	w := float64(l.Width)
	g := float64(l.Gap)

	if l.Font == nil && font == nil {
		return errors.New("pdfcpu: missing label font")
	}

	var (
		fontSize int
		fontName string
		col      *SimpleColor
	)

	if l.Font != nil {
		fontName = l.Font.Name
		fontSize = l.Font.Size
		col = l.Font.col
	} else {
		fontName = font.Name
		fontSize = font.Size
		col = font.col
	}

	k := p.Fm.EnsureKey(fontName)

	td := TextDescriptor{
		Text:     v,
		FontName: fontName,
		FontKey:  k,
		FontSize: fontSize,
		Scale:    1.,
		ScaleAbs: true,
		RTL:      l.RTL, // for user fonts only!
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if l.bgCol != nil {
		//td.ShowBorder = true
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *l.bgCol
	}

	// calc real label height
	bb := WriteMultiLine(new(bytes.Buffer), RectForFormat("A4"), nil, td)

	td.X, td.Y = labelPosition(l.relPos, l.horAlign, lb.BoundingBox(), bb.Height(), w, g, true) // lb.Multiline)

	td.HAlign, td.VAlign = l.horAlign, AlignBottom

	WriteColumn(p.Buf, p.MediaBox, nil, td, 0)

	return nil
}

func createTextBox(tb *TextBox, p *Page) error {

	w := float64(tb.Width)

	fontName := tb.Font.Name
	fontSize := tb.Font.Size
	col := tb.Font.col

	k := p.Fm.EnsureKey(fontName)

	td := TextDescriptor{
		Text:     tb.Value,
		X:        tb.x,
		Y:        tb.y,
		HAlign:   tb.horAlign,
		VAlign:   AlignBottom,
		FontName: fontName,
		FontKey:  k,
		FontSize: fontSize,
		Scale:    1.,
		ScaleAbs: true,
		RTL:      tb.RTL, // for user fonts only!
		// MTop, MBot, MLeft, MRight
		// Borderwidth, BorderStyle, BordeCol
		// Rotation
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	// Set Border color etc.

	if tb.bgCol != nil {
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *tb.bgCol
		// ShowMargins
	}

	WriteColumn(p.Buf, p.MediaBox, nil, td, w)

	return nil
}

func createTextField(
	xRefTable *XRefTable,
	tf *TextField,
	fonts map[string]FontResource,
	inheritedDA string,
	pageAnnots *Array,
	fields *Array,
	pageIndRef *IndirectRef) error {

	id := StringLiteral(encodeUTF16String(tf.ID))

	ff := FieldDoNotSpellCheck
	if tf.Multiline {
		// If set, the field may contain multiple lines of text;
		// if clear, the field’s text shall be restricted to a single line.
		// Adobe Reader ok, Mac Preview nope
		ff += FieldMultiline
	} else {
		// If set, the field shall not scroll (horizontally for single-line fields, vertically for multiple-line fields)
		// to accommodate more text than fits within its annotation rectangle.
		// Once the field is full, no further text shall be accepted for interactive form filling;
		// for non- interactive form filling, the filler should take care
		// not to add more character than will visibly fit in the defined area.
		// Adobe Reader ok, Mac Preview nope
		ff += FieldDoNotScroll
	}

	d := Dict(
		map[string]Object{
			"FT":   Name("Tx"),
			"Rect": tf.boundingBox.Array(),
			//"H":       Name("O"),
			"F":       Integer(AnnPrint),
			"Ff":      Integer(ff),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"Q":       Integer(tf.horAlign), // Adjustment: (0:L) 1:C 2:R
			"T":       id,                   // required
			"TU":      id,                   // Acrobat Reader Hover over field
			"P":       *pageIndRef,
		},
	)

	if tf.bgCol != nil || tf.Border != nil {
		appCharDict := Dict(map[string]Object{})
		if tf.bgCol != nil {
			// Acrobat Reader shows background color for fields in focus only.
			appCharDict["BG"] = tf.bgCol.Array()
		}
		if tf.Border != nil && tf.Border.col != nil && tf.Border.Width > 0 {
			appCharDict["BC"] = tf.Border.col.Array()
		}
		d["MK"] = appCharDict
	}

	if tf.Border != nil && tf.Border.Width > 0 {
		// BorderWidth Acrobat only: steady default background color blue gray
		d["Border"] = NewIntegerArray(0, 0, tf.Border.Width)
	}

	if tf.Value != "" {
		sl := StringLiteral(encodeUTF16String(tf.Value))
		d["DV"] = sl
		d["V"] = sl
	}

	if inheritedDA != "" {
		d["DA"] = StringLiteral(inheritedDA)
	}

	if tf.Font != nil {

		resID, err := ensureFont(xRefTable, tf.Font.Name, fonts)
		if err != nil {
			return err
		}

		da := fmt.Sprintf("/%s %d Tf ", resID, tf.Font.Size)

		if tf.Font.col != nil {
			c := tf.Font.col
			da += fmt.Sprintf("%.2f %.2f %.2f rg", c.R, c.G, c.B)
		}

		// Mac Preview does not honor inherited "DA"
		d["DA"] = StringLiteral(da)
	}
	// else if f.Inputfont == nil return err

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	*pageAnnots = append(*pageAnnots, *ir)
	*fields = append(*fields, *ir)

	return nil
}

func createCheckBox(
	xRefTable *XRefTable,
	cb *CheckBox,
	fonts map[string]FontResource,
	inheritedDA string,
	pageAnnots *Array,
	fields *Array,
	pageIndRef *IndirectRef) error {

	// No AP works also, but Adobe Reader clears rect once clicked until clicked outside.

	id := StringLiteral(encodeUTF16String(cb.ID))

	ff := FieldDoNotSpellCheck
	// if tf.Multiline {
	// 	// If set, the field may contain multiple lines of text;
	// 	// if clear, the field’s text shall be restricted to a single line.
	// 	// Adobe Reader ok, Mac Preview nope
	// 	ff += FieldMultiline
	// } else {
	// 	// If set, the field shall not scroll (horizontally for single-line fields, vertically for multiple-line fields)
	// 	// to accommodate more text than fits within its annotation rectangle.
	// 	// Once the field is full, no further text shall be accepted for interactive form filling;
	// 	// for non- interactive form filling, the filler should take care
	// 	// not to add more character than will visibly fit in the defined area.
	// 	// Adobe Reader ok, Mac Preview nope
	// 	ff += FieldDoNotScroll
	// }

	v := "Off"
	if cb.Value {
		v = "Yes"
	}

	d := Dict(
		map[string]Object{
			"FT":   Name("Btn"),
			"V":    Name(v), // -> extractValue: Off or Yes
			"AS":   Name(v),
			"Rect": cb.boundingBox().Array(),
			"F":    Integer(AnnPrint),
			"Ff":   Integer(ff),
			//"H":       Name("N"),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"T":       id, // required
			"TU":      id, // Acrobat Reader Hover over field
			"P":       *pageIndRef,
		},
	)

	/*
		q
			0 0 1 rg
			BT
				/ZaDb 12 Tf 0 0 Td
				(8) Tj
			ET
		Q

	*/

	// if tf.bgCol != nil || tf.Border != nil {
	//appCharDict := Dict(map[string]Object{})
	// 	if tf.bgCol != nil {
	// 		// Acrobat Reader shows background color for fields in focus only.
	// 		appCharDict["BG"] = tf.bgCol.Array()
	// 	}
	// 	if tf.Border != nil && tf.Border.col != nil && tf.Border.Width > 0 {
	//appCharDict["BC"] = .Border.col.Array()
	// 	}
	//d["MK"] = appCharDict
	// }

	// if tf.Border != nil && tf.Border.Width > 0 {
	// 	// BorderWidth Acrobat only: steady default background color blue gray
	//d["Border"] = NewIntegerArray(0, 0, 1)
	// }

	// if tf.Value != "" {
	// 	sl := StringLiteral(encodeUTF16String(tf.Value))
	// 	d["DV"] = sl
	// 	d["V"] = sl
	// }

	// if inheritedDA != "" {
	// 	d["DA"] = StringLiteral(inheritedDA)
	// }

	// if tf.Font != nil {

	// 	resID, err := ensureFont(xRefTable, tf.Font.Name, fonts)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	da := fmt.Sprintf("/%s %d Tf ", resID, tf.Font.Size)

	// 	if tf.Font.col != nil {
	// 		c := tf.Font.col
	// 		da += fmt.Sprintf("%.2f %.2f %.2f rg", c.R, c.G, c.B)
	// 	}

	// 	// Mac Preview does not honor inherited "DA"
	// 	d["DA"] = StringLiteral(da)
	// }

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	*pageAnnots = append(*pageAnnots, *ir)
	*fields = append(*fields, *ir)

	return nil
}

func createAPForm(xRefTable *XRefTable, w, h float64, s string, resDictIR *IndirectRef) (*IndirectRef, error) {

	var b bytes.Buffer
	fmt.Fprint(&b, s)

	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":     Name("XObject"),
				"Subtype":  Name("Form"),
				"FormType": Integer(1),
				"BBox":     NewNumberArray(0, 0, w, h),
				"Matrix":   NewIntegerArray(1, 0, 0, 1, 0, 0),
				"Filter":   Name(filter.Flate),
			},
		),
		Content:        b.Bytes(),
		FilterPipeline: []PDFFilter{{Name: filter.Flate, DecodeParms: nil}},
	}

	if resDictIR != nil {
		sd.Insert("Resources", *resDictIR)
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createRadioOnAP(xRefTable *XRefTable, w, h float64, resDictIR *IndirectRef) (*IndirectRef, error) {
	s := "q 0 0 1 rg BT /ZaDb 12 Tf 0 0 Td (4) Tj ET Q"
	return createAPForm(xRefTable, w, h, s, resDictIR)
}

func createRadioOffAP(xRefTable *XRefTable, w, h float64, resDictIR *IndirectRef) (*IndirectRef, error) {
	//r := w
	// s := fmt.Sprintf("q 0 0 1 rg %.1f 0 m %.1f %.1f %.1f %.1f 0 %.1f c %.1f %.1f %.1f %.1f %.1f 0 c %.1f %.1f %.1f %.1f 0 %.1f c %.1f %.1f %.1f %.1f %.1f 0 c b* Q",
	// 	r, r, r/2, r/2, r, r, -r/2, r, -r, r/2, -r, -r, -r/2, -r/2, -r, -r, r/2, -r, r, -r/2, r)
	s := "q 0 0 1 rg BT /ZaDb 12 Tf 0 0 Td (8) Tj ET Q"
	return createAPForm(xRefTable, w, h, s, resDictIR)
}

func createAPResDict(xRefTable *XRefTable) (*IndirectRef, error) {

	fontDict, err := createFontDict(xRefTable, "ZapfDingbats")
	if err != nil {
		return nil, err
	}

	d := Dict(
		map[string]Object{
			"Font": Dict(
				map[string]Object{
					"ZaDb": *fontDict,
				},
			),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createRadioButtonFields(
	xRefTable *XRefTable,
	rbg *RadioButtonGroup,
	parent *IndirectRef,
	pageAnnots *Array,
	pageIndRef *IndirectRef) (Array, error) {

	w, h := 12.0, 12.0

	ir, err := createAPResDict(xRefTable)
	if err != nil {
		return nil, err
	}

	onIndRef, err := createRadioOnAP(xRefTable, w, h, ir)
	if err != nil {
		return nil, err
	}

	offIndRef, err := createRadioOffAP(xRefTable, w, h, ir)
	if err != nil {
		return nil, err
	}

	kids := Array{}

	maxw, firstw := rbg.Buttons.maxLabelWidth(rbg.hor)

	for i, v := range rbg.Buttons.Values {

		r := rbg.rect(float64(i), maxw, firstw)
		id := StringLiteral(encodeUTF16String(v))
		buttonVal := strconv.Itoa(i)

		d := Dict(map[string]Object{
			//"FT":      Name("Btn"),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"F":       Integer(AnnPrint),
			"Parent":  *parent,
			"AS":      Name("Off"),
			"Rect":    r.Array(),
			"T":       id, // required
			"TU":      id, // Acrobat Reader Hover over field
			"P":       *pageIndRef,
			"AP": Dict(map[string]Object{
				"N": Dict(map[string]Object{
					buttonVal: *onIndRef,
					"Off":     *offIndRef,
				}),
			}),
		})

		if i == 0 {
			d["AS"] = Name(buttonVal)
		}

		ir, err := xRefTable.IndRefForNewObject(d)
		if err != nil {
			return nil, err
		}

		kids = append(kids, *ir)
		*pageAnnots = append(*pageAnnots, *ir)
	}

	return kids, nil
}

func createRadioButtonGroup(
	xRefTable *XRefTable,
	rbg *RadioButtonGroup,
	fonts map[string]FontResource,
	inheritedDA string,
	pageAnnots *Array,
	fields *Array,
	pageIndRef *IndirectRef) error {

	id := StringLiteral(encodeUTF16String(rbg.ID))

	ff := FieldNoToggleToOff + FieldRadio

	d := Dict(
		map[string]Object{
			"FT": Name("Btn"),
			"V":  Name("Off"), // -> extract set radio button
			"DV": Name("Off"),
			"Ff": Integer(ff),
			"T":  id, // required
			"TU": id, // Acrobat Reader Hover over field
			//"Opt"
		},
	)

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	kids, err := createRadioButtonFields(xRefTable, rbg, ir, pageAnnots, pageIndRef)
	if err != nil {
		return err
	}

	d["Kids"] = kids

	*fields = append(*fields, *ir)

	return nil
}

func createListBox(
	xRefTable *XRefTable,
	lb ListBox,
	combo bool,
	fonts map[string]FontResource,
	inheritedDA string,
	pageAnnots *Array,
	fields *Array,
	pageIndRef *IndirectRef) error {

	id := StringLiteral(encodeUTF16String(lb.Id()))

	opt := Array{}
	for _, s := range lb.Opts() {
		opt = append(opt, StringLiteral(encodeUTF16String(s)))
	}

	ff := FieldFlags(0)

	if combo {
		ff += FieldCombo
		if lb.Editable() {
			// Note: unsupported in Mac Preview
			ff += FieldEdit + FieldDoNotSpellCheck
		}
	} else if lb.MultiSelect() {
		// Note: unsupported in Mac Preview
		ff += FieldMultiselect
	}

	d := Dict(
		map[string]Object{
			"FT":      Name("Ch"),
			"Rect":    lb.BoundingBox().Array(),
			"F":       Integer(AnnPrint),
			"Ff":      Integer(ff),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"Opt":     opt,
			"T":       id, // required
			"TU":      id, // Acrobat Reader Tooltip
			"P":       *pageIndRef,
		},
	)

	if lb.backgroundColor() != nil || lb.border() != nil {
		appCharDict := Dict(map[string]Object{})
		if lb.backgroundColor() != nil {
			// when listbox is active only.
			appCharDict["BG"] = lb.backgroundColor().Array()
		}
		if lb.border() != nil && lb.border().col != nil && lb.border().Width > 0 {
			appCharDict["BC"] = lb.border().col.Array()
		}
		d["MK"] = appCharDict
	}

	if lb.border() != nil && lb.border().Width > 0 {
		d["Border"] = NewIntegerArray(0, 0, lb.border().Width)
	}

	vv := lb.values()
	if len(vv) == 1 {
		d["V"] = StringLiteral(encodeUTF16String(vv[0]))
	}
	arr, ind := Array{}, Array{}
	if len(vv) > 1 {
		// For multi select scrollable listboxes only.
		for _, v := range vv {
			arr = append(arr, StringLiteral(encodeUTF16String(v)))
			for i, o := range lb.Opts() {
				if v == o {
					ind = append(ind, Integer(i))
				}
			}
		}
		d["V"] = arr // d["DV"] ??
		d["I"] = ind
	}

	if inheritedDA != "" {
		d["DA"] = StringLiteral(inheritedDA)
	}

	// if tf.Font != nil {

	// 	resID, err := ensureFont(xRefTable, tf.Font.Name, fonts)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	da := fmt.Sprintf("/%s %d Tf ", resID, tf.Font.Size)

	// 	if tf.Font.col != nil {
	// 		c := tf.Font.col
	// 		da += fmt.Sprintf("%.2f %.2f %.2f rg", c.R, c.G, c.B)
	// 	}

	// 	// Mac Preview does not honor inherited "DA"
	// 	d["DA"] = StringLiteral(da)
	// }
	// else if f.Inputfont == nil return err

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	*pageAnnots = append(*pageAnnots, *ir)
	*fields = append(*fields, *ir)

	return nil
}

func createLabels(form *Form) (Page, error) {

	p := Page{MediaBox: form.mediaBox, Fm: FontMap{}, Buf: new(bytes.Buffer)}
	if form.bgCol != nil {
		FillRect(p.Buf, form.mediaBox, *form.bgCol)
	}

	for _, tf := range form.TextFields {
		if err := createTextFieldLabel(tf, &p, form.LabelFont); err != nil {
			return p, err
		}
	}

	for _, cb := range form.CheckBoxes {
		if err := createCheckBoxLabel(cb, &p, form.LabelFont); err != nil {
			return p, err
		}
	}

	for _, rbg := range form.RadioButtonGroups {
		if err := createRadioButtonGroupLabels(rbg, &p, form.LabelFont); err != nil {
			return p, err
		}
	}

	for _, lb := range form.ListBoxes {
		if err := createListBoxLabel(lb, &p, form.LabelFont); err != nil {
			return p, err
		}
	}

	for _, cb := range form.ComboBoxes {
		if err := createListBoxLabel(cb, &p, form.LabelFont); err != nil {
			return p, err
		}
	}

	for _, tb := range form.TextBoxes {
		if err := createTextBox(tb, &p); err != nil {
			return p, err
		}
	}

	//DrawHairCross(p.Buf, 0, 0, p.MediaBox)

	return p, nil
}

func ensureFont(xRefTable *XRefTable, fontName string, fonts map[string]FontResource) (string, error) {
	fontResource, ok := fonts[fontName]
	if ok {
		return fontResource.resID, nil
	}
	resID := fmt.Sprintf("F%d", len(fonts))

	indRef, err := createFontDict(xRefTable, fontName)
	if err != nil {
		return "", err
	}

	fonts[fontName] = FontResource{resID: resID, indRef: *indRef}

	return resID, nil
}

func createForm(xRefTable *XRefTable, form *Form, pageIndRef *IndirectRef) (Dict, error) {

	var da string

	d := Dict(map[string]Object{"NeedAppearances": Boolean(true)})

	if form.InputFont != nil {

		resID, err := ensureFont(xRefTable, form.InputFont.Name, form.fonts)
		if err != nil {
			return nil, err
		}

		da = fmt.Sprintf("/%s %d Tf ", resID, form.InputFont.Size)

		if form.InputFont.col != nil {
			c := form.InputFont.col
			da += fmt.Sprintf("%.2f %.2f %.2f rg", c.R, c.G, c.B)
		}

	}

	annots := Array{}
	fields := Array{}

	for _, tf := range form.TextFields {
		if err := createTextField(xRefTable, tf, form.fonts, da, &annots, &fields, pageIndRef); err != nil {
			return nil, err
		}
	}

	for _, cb := range form.CheckBoxes {
		if err := createCheckBox(xRefTable, cb, form.fonts, da, &annots, &fields, pageIndRef); err != nil {
			return nil, err
		}
	}

	for _, rbg := range form.RadioButtonGroups {
		if err := createRadioButtonGroup(xRefTable, rbg, form.fonts, da, &annots, &fields, pageIndRef); err != nil {
			return nil, err
		}
	}

	for _, lb := range form.ListBoxes {
		if err := createListBox(xRefTable, lb, false, form.fonts, da, &annots, &fields, pageIndRef); err != nil {
			return nil, err
		}
	}

	for _, lb := range form.ComboBoxes {
		if err := createListBox(xRefTable, lb, true, form.fonts, da, &annots, &fields, pageIndRef); err != nil {
			return nil, err
		}
	}

	d["Fields"] = fields
	form.annots = annots

	if len(form.fonts) > 0 {
		d1 := Dict{}
		for _, fontRes := range form.fonts {
			d1.Insert(fontRes.resID, fontRes.indRef)
		}
		d["DR"] = Dict(map[string]Object{"Font": d1})
	}

	return d, nil
}

func createFormPage(xRefTable *XRefTable, parentPageIndRef IndirectRef, p Page, f *Form) (*IndirectRef, Dict, error) {

	pageDict := Dict(
		map[string]Object{
			"Type":   Name("Page"),
			"Parent": parentPageIndRef,
		},
	)

	fontRes := Dict{}
	for k, fontName := range p.Fm {
		fontResource, ok := f.fonts[fontName]
		if !ok {
			ir, err := createFontDict(xRefTable, fontName)
			if err != nil {
				return nil, pageDict, err
			}
			fontRes.Insert(k, *ir)
			continue
		}
		fontRes.Insert(k, fontResource.indRef)
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
		return nil, pageDict, err
	}

	pageDict.Insert("Contents", *ir)

	pageDictIndRef, err := xRefTable.IndRefForNewObject(pageDict)

	return pageDictIndRef, pageDict, err
}

func addPageTreeWithFormFields(xRefTable *XRefTable, rootDict Dict, p Page, f *Form) (*IndirectRef, error) {

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

	// adding annotations to page
	pageIndRef, pageDict, err := createFormPage(xRefTable, *parentPageIndRef, p, f)
	if err != nil {
		return nil, err
	}

	formDict, err := createForm(xRefTable, f, pageIndRef)
	if err != nil {
		return nil, err
	}

	rootDict.Insert("AcroForm", formDict)

	pageDict.Insert("Annots", f.annots)
	pagesDict.Insert("Kids", Array{*pageIndRef})

	rootDict.Insert("Pages", *parentPageIndRef)

	return pageIndRef, nil
}

// Create a PDF with a single page form with 1 text field.
func CreateFormXRef(bb []byte) (*XRefTable, error) {

	form, err := parseForm(bb)
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

	p, err := createLabels(form)
	if err != nil {
		return nil, err
	}

	_, err = addPageTreeWithFormFields(xRefTable, rootDict, p, form)
	if err != nil {
		return nil, err
	}

	return xRefTable, nil
}
