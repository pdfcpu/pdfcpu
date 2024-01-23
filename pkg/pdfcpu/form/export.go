/*
Copyright 2023 The pdfcpu Authors.

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

package form

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/primitives"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// Header represents form meta data.
type Header struct {
	Source   string   `json:"source"`
	Version  string   `json:"version"`
	Creation string   `json:"creation"`
	ID       []string `json:"id,omitempty"`
	Title    string   `json:"title,omitempty"`
	Author   string   `json:"author,omitempty"`
	Creator  string   `json:"creator,omitempty"`
	Producer string   `json:"producer,omitempty"`
	Subject  string   `json:"subject,omitempty"`
	Keywords string   `json:"keywords,omitempty"`
}

// TextField represents a form text field.
type TextField struct {
	Pages     []int  `json:"pages"`
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Default   string `json:"default,omitempty"`
	Value     string `json:"value"`
	Multiline bool   `json:"multiline"`
	Locked    bool   `json:"locked"`
}

// DateField represents an Acroform date field.
type DateField struct {
	Pages   []int  `json:"pages"`
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Format  string `json:"format"`
	Default string `json:"default,omitempty"`
	Value   string `json:"value"`
	Locked  bool   `json:"locked"`
}

// RadioButtonGroup represents a form checkbox.
type CheckBox struct {
	Pages   []int  `json:"pages"`
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Default bool   `json:"default"`
	Value   bool   `json:"value"`
	Locked  bool   `json:"locked"`
}

// RadioButtonGroup represents a form radio button group.
type RadioButtonGroup struct {
	Pages   []int    `json:"pages"`
	ID      string   `json:"id"`
	Name    string   `json:"name,omitempty"`
	Options []string `json:"options"`
	Default string   `json:"default,omitempty"`
	Value   string   `json:"value"`
	Locked  bool     `json:"locked"`
}

// ComboBox represents a form combobox.
type ComboBox struct {
	Pages    []int    `json:"pages"`
	ID       string   `json:"id"`
	Name     string   `json:"name,omitempty"`
	Editable bool     `json:"editable"`
	Options  []string `json:"options"`
	Default  string   `json:"default,omitempty"`
	Value    string   `json:"value"`
	Locked   bool     `json:"locked"`
}

// ListBox represents a form listbox.
type ListBox struct {
	Pages    []int    `json:"pages"`
	ID       string   `json:"id"`
	Name     string   `json:"name,omitempty"`
	Multi    bool     `json:"multi"`
	Options  []string `json:"options"`
	Defaults []string `json:"defaults,omitempty"`
	Values   []string `json:"values,omitempty"`
	Locked   bool     `json:"locked"`
}

// Page is a container for page imageboxes.
type Page struct {
	ImageBoxes []*primitives.ImageBox `json:"image,omitempty"`
}

// Form represents a PDF form (aka. Acroform).
type Form struct {
	TextFields        []*TextField        `json:"textfield,omitempty"`
	DateFields        []*DateField        `json:"datefield,omitempty"`
	CheckBoxes        []*CheckBox         `json:"checkbox,omitempty"`
	RadioButtonGroups []*RadioButtonGroup `json:"radiobuttongroup,omitempty"`
	ComboBoxes        []*ComboBox         `json:"combobox,omitempty"`
	ListBoxes         []*ListBox          `json:"listbox,omitempty"`
	Pages             map[string]*Page    `json:"pages,omitempty"`
}

// FormGroup represents a JSON struct containing a sequence of form instances.
type FormGroup struct {
	Header Header `json:"header"`
	Forms  []Form `json:"forms"`
}

func (f Form) textFieldValueAndLock(id, name string) (string, bool, bool) {
	for _, tf := range f.TextFields {
		if tf.ID == id || tf.Name == name {
			return tf.Value, tf.Locked, true
		}
	}
	return "", false, false
}

func (f Form) dateFieldValueAndLock(id, name string) (string, bool, bool) {
	for _, df := range f.DateFields {
		if df.ID == id || df.Name == name {
			return df.Value, df.Locked, true
		}
	}
	return "", false, false
}

func (f Form) checkBoxValueAndLock(id, name string) (bool, bool, bool) {
	for _, cb := range f.CheckBoxes {
		if cb.ID == id || cb.Name == name {
			return cb.Value, cb.Locked, true
		}
	}
	return false, false, false
}

func (f Form) radioButtonGroupValueAndLock(id, name string) (string, bool, bool) {
	for _, rbg := range f.RadioButtonGroups {
		if rbg.ID == id || rbg.Name == name {
			return rbg.Value, rbg.Locked, true
		}
	}
	return "", false, false
}

func (f Form) comboBoxValueAndLock(id, name string) (string, bool, bool) {
	for _, cb := range f.ComboBoxes {
		if cb.ID == id || cb.Name == name {
			return cb.Value, cb.Locked, true
		}
	}
	return "", false, false
}

func (f Form) listBoxValuesAndLock(id, name string) ([]string, bool, bool) {
	for _, lb := range f.ListBoxes {
		if lb.ID == id || lb.Name == name {
			return lb.Values, lb.Locked, true
		}
	}
	return nil, false, false
}

func extractRadioButtonGroupOptions(xRefTable *model.XRefTable, d types.Dict) ([]string, error) {

	var opts []string
	p := 0

	for _, o := range d.ArrayEntry("Kids") {
		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			return nil, err
		}

		indRef := d.IndirectRefEntry("P")
		if indRef != nil {
			if p == 0 {
				p = indRef.ObjectNumber.Value()
			} else if p != indRef.ObjectNumber.Value() {
				continue
			}
		}

		d1 := d.DictEntry("AP")
		if d1 == nil {
			return nil, errors.New("corrupt form field: missing entry AP")
		}
		d2 := d1.DictEntry("N")
		if d2 == nil {
			return nil, errors.New("corrupt AP field: missing entry N")
		}
		for k := range d2 {
			k, err := types.DecodeName(k)
			if err != nil {
				return nil, err
			}
			if k != "Off" {
				for _, opt := range opts {
					if opt == k {
						continue
					}
				}
				opts = append(opts, k)
			}
		}
	}

	return opts, nil
}

func extractRadioButtonGroup(xRefTable *model.XRefTable, page int, d types.Dict, id, name string, locked bool) (*RadioButtonGroup, error) {

	rbg := &RadioButtonGroup{Pages: []int{page}, ID: id, Name: name, Locked: locked}

	if s := d.NameEntry("DV"); s != nil {
		n, err := types.DecodeName(*s)
		if err != nil {
			return nil, err
		}
		rbg.Default = n
	}

	if s := d.NameEntry("V"); s != nil {
		n, err := types.DecodeName(*s)
		if err != nil {
			return nil, err
		}
		if n != "Off" {
			rbg.Value = n
		}
	}

	opts, err := extractRadioButtonGroupOptions(xRefTable, d)
	if err != nil {
		return nil, err
	}

	rbg.Options = opts

	return rbg, nil
}

func extractCheckBox(page int, d types.Dict, id, name string, locked bool) (*CheckBox, error) {

	cb := &CheckBox{Pages: []int{page}, ID: id, Name: name, Locked: locked}

	if o, ok := d.Find("DV"); ok {
		cb.Default = o.(types.Name) == "Yes"
	}

	if o, ok := d.Find("V"); ok {
		cb.Value = o.(types.Name) == "Yes"
	}

	return cb, nil
}

func extractComboBox(xRefTable *model.XRefTable, page int, d types.Dict, id, name string, locked bool) (*ComboBox, error) {

	cb := &ComboBox{Pages: []int{page}, ID: id, Name: name, Locked: locked}

	if sl := d.StringLiteralEntry("DV"); sl != nil {
		s, err := types.StringLiteralToString(*sl)
		if err != nil {
			return nil, err
		}
		cb.Default = s
	}

	if sl := d.StringLiteralEntry("V"); sl != nil {
		s, err := types.StringLiteralToString(*sl)
		if err != nil {
			return nil, err
		}
		cb.Value = s
	}

	opts, err := parseOptions(xRefTable, d)
	if err != nil {
		return nil, err
	}
	if len(opts) == 0 {
		return nil, errors.New("pdfcpu: combobox missing Opts")
	}

	cb.Options = opts

	return cb, nil
}

func extractDateFormat(xRefTable *model.XRefTable, d types.Dict) (*primitives.DateFormat, error) {

	d1 := d.DictEntry("AA")
	if len(d1) > 0 {
		d2 := d1.DictEntry("F")
		if len(d2) > 0 {
			sl := d2.StringLiteralEntry("JS")
			if sl != nil {
				s, err := types.StringLiteralToString(*sl)
				if err != nil {
					return nil, err
				}
				i := strings.Index(s, "AFDate_FormatEx(\"")
				if i >= 0 {
					from := i + len("AFDate_FormatEx(\"")
					s = s[from : from+10]
				}
				if df, err := primitives.DateFormatForFmtExt(s); err == nil {
					return df, nil
				}
			}
		}
	}

	if o, found := d.Find("DV"); found {
		sl, _ := o.(types.StringLiteral)
		s, err := types.StringLiteralToString(sl)
		if err != nil {
			return nil, err
		}
		if df, err := primitives.DateFormatForDate(s); err == nil {
			return df, nil
		}
	}

	if o, found := d.Find("V"); found {
		sl, _ := o.(types.StringLiteral)
		s, err := types.StringLiteralToString(sl)
		if err != nil {
			return nil, err
		}
		if df, err := primitives.DateFormatForDate(s); err == nil {
			return df, nil
		}
	}

	return nil, nil
}

func extractDateField(page int, d types.Dict, id, name string, df *primitives.DateFormat, locked bool) (*DateField, error) {

	dfield := &DateField{Pages: []int{page}, ID: id, Name: name, Format: df.Ext, Locked: locked}

	if o, found := d.Find("DV"); found {
		sl, _ := o.(types.StringLiteral)
		s, err := types.StringLiteralToString(sl)
		if err != nil {
			return nil, err
		}
		dfield.Default = s
	}

	if o, found := d.Find("V"); found {
		sl, _ := o.(types.StringLiteral)
		s, err := types.StringLiteralToString(sl)
		if err != nil {
			return nil, err
		}
		dfield.Value = s
	}

	return dfield, nil
}

func extractTextField(page int, d types.Dict, id, name string, ff *int, locked bool) (*TextField, error) {

	multiLine := ff != nil && uint(primitives.FieldFlags(*ff))&uint(primitives.FieldMultiline) > 0

	tf := &TextField{Pages: []int{page}, ID: id, Name: name, Multiline: multiLine, Locked: locked}

	if o, found := d.Find("DV"); found {
		sl, _ := o.(types.StringLiteral)
		s, err := types.StringLiteralToString(sl)
		if err != nil {
			return nil, err
		}
		tf.Default = s
	}

	if o, found := d.Find("V"); found {
		sl, _ := o.(types.StringLiteral)
		s, err := types.StringLiteralToString(sl)
		if err != nil {
			return nil, err
		}
		tf.Value = s
	}

	return tf, nil
}

func extractListBox(xRefTable *model.XRefTable, page int, d types.Dict, id, name string, locked, multi bool) (*ListBox, error) {

	lb := &ListBox{Pages: []int{page}, ID: id, Name: name, Locked: locked, Multi: multi}

	if !multi {
		if sl := d.StringLiteralEntry("DV"); sl != nil {
			s, err := types.StringLiteralToString(*sl)
			if err != nil {
				return nil, err
			}
			lb.Defaults = []string{s}
		}
		if sl := d.StringLiteralEntry("V"); sl != nil {
			s, err := types.StringLiteralToString(*sl)
			if err != nil {
				return nil, err
			}
			lb.Values = []string{s}
		}
	} else {
		ss, err := parseStringLiteralArray(xRefTable, d, "DV")
		if err != nil {
			return nil, err
		}
		lb.Defaults = ss
		ss, err = parseStringLiteralArray(xRefTable, d, "V")
		if err != nil {
			return nil, err
		}
		lb.Values = ss
	}

	opts, err := parseOptions(xRefTable, d)
	if err != nil {
		return nil, err
	}
	if len(opts) == 0 {
		return nil, errors.New("pdfcpu: listbox missing Opts")
	}

	lb.Options = opts

	return lb, nil
}

func header(xRefTable *model.XRefTable, source string) Header {
	h := Header{}
	h.Source = filepath.Base(source)
	h.Version = "pdfcpu " + model.VersionStr
	h.Creation = time.Now().Format("2006-01-02 15:04:05 MST")
	h.ID = []string{}
	h.Title = xRefTable.Title
	h.Author = xRefTable.Author
	h.Creator = xRefTable.Creator
	h.Producer = xRefTable.Producer
	h.Subject = xRefTable.Subject
	h.Keywords = xRefTable.Keywords
	return h
}

func fieldsForAnnots(xRefTable *model.XRefTable, annots, fields types.Array) (map[string]fieldInfo, error) {

	m := map[string]fieldInfo{}
	var prevId string

	for _, v := range annots {

		indRef := v.(types.IndirectRef)

		ok, fi, err := isField(xRefTable, indRef, fields)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		if fi.indRef == nil {
			fi.indRef = &indRef
		}

		if fi.id != prevId {
			m[fi.id] = *fi
			prevId = fi.id
		}
	}

	return m, nil
}

func exportBtn(
	xRefTable *model.XRefTable,
	i int,
	form *Form,
	d types.Dict,
	id, name string,
	locked bool,
	ok *bool) error {

	if len(d.ArrayEntry("Kids")) > 0 {

		for _, rb := range form.RadioButtonGroups {
			if rb.ID == id && rb.Name == name {
				rb.Pages = append(rb.Pages, i)
				return nil
			}
		}

		rbg, err := extractRadioButtonGroup(xRefTable, i, d, id, name, locked)
		if err != nil {
			return err
		}

		form.RadioButtonGroups = append(form.RadioButtonGroups, rbg)
		*ok = true
		return nil
	}

	for _, cb := range form.CheckBoxes {
		if cb.Name == name && cb.ID == id {
			cb.Pages = append(cb.Pages, i)
			return nil
		}
	}

	cb, err := extractCheckBox(i, d, id, name, locked)
	if err != nil {
		return err
	}

	form.CheckBoxes = append(form.CheckBoxes, cb)
	*ok = true
	return nil
}

func exportCh(
	xRefTable *model.XRefTable,
	i int,
	form *Form,
	d types.Dict,
	id, name string,
	locked bool,
	ok *bool) error {

	ff := d.IntEntry("Ff")
	if ff == nil {
		return errors.New("pdfcpu: corrupt form field: missing entry Ff")
	}

	if primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 {

		for _, cb := range form.ComboBoxes {
			if cb.Name == name && cb.ID == id {
				cb.Pages = append(cb.Pages, i)
				return nil
			}
		}

		cb, err := extractComboBox(xRefTable, i, d, id, name, locked)
		if err != nil {
			return err
		}
		form.ComboBoxes = append(form.ComboBoxes, cb)
		*ok = true
		return nil
	}

	for _, lb := range form.ListBoxes {
		if lb.Name == name && lb.ID == id {
			lb.Pages = append(lb.Pages, i)
			return nil
		}
	}

	multi := primitives.FieldFlags(*ff)&primitives.FieldMultiselect > 0
	lb, err := extractListBox(xRefTable, i, d, id, name, locked, multi)
	if err != nil {
		return err
	}

	form.ListBoxes = append(form.ListBoxes, lb)
	*ok = true
	return nil
}

func exportTx(
	xRefTable *model.XRefTable,
	i int,
	form *Form,
	d types.Dict,
	id, name string,
	ff *int,
	locked bool,
	ok *bool) error {

	df, err := extractDateFormat(xRefTable, d)
	if err != nil {
		return err
	}

	if df != nil {

		for _, df := range form.DateFields {
			if df.Name == name && df.ID == id {
				df.Pages = append(df.Pages, i)
				return nil
			}
		}

		df, err := extractDateField(i, d, id, name, df, locked)
		if err != nil {
			return err
		}

		form.DateFields = append(form.DateFields, df)
		*ok = true
		return nil
	}

	for _, tf := range form.TextFields {
		if tf.Name == name && tf.ID == id {
			tf.Pages = append(tf.Pages, i)
			return nil
		}
	}

	tf, err := extractTextField(i, d, id, name, ff, locked)
	if err != nil {
		return err
	}

	form.TextFields = append(form.TextFields, tf)
	*ok = true
	return nil
}

func exportPageFields(xRefTable *model.XRefTable, i int, form *Form, m map[string]fieldInfo, ok *bool) error {
	for id, fi := range m {

		name := fi.name

		d, err := xRefTable.DereferenceDict(*fi.indRef)
		if err != nil {
			return err
		}
		if len(d) == 0 {
			continue
		}

		var locked bool
		ff := d.IntEntry("Ff")
		if ff != nil {
			locked = uint(primitives.FieldFlags(*ff))&uint(primitives.FieldReadOnly) > 0
		}

		ft := fi.ft
		if ft == nil {
			ft = d.NameEntry("FT")
			if ft == nil {
				return errors.New("pdfcpu: corrupt form field: missing entry FT")
			}
		}

		switch *ft {
		case "Btn":
			if err := exportBtn(xRefTable, i, form, d, id, name, locked, ok); err != nil {
				return err
			}

		case "Ch":
			if err := exportCh(xRefTable, i, form, d, id, name, locked, ok); err != nil {
				return err
			}

		case "Tx":
			if err := exportTx(xRefTable, i, form, d, id, name, ff, locked, ok); err != nil {
				return err
			}
		}

	}

	return nil
}

// ExportForm extracts form data originating from source from xRefTable.
func ExportForm(xRefTable *model.XRefTable, source string) (*FormGroup, bool, error) {

	fields, err := fields(xRefTable)
	if err != nil {
		return nil, false, err
	}

	formGroup := FormGroup{}
	formGroup.Header = header(xRefTable, source)

	form := Form{}

	var ok bool

	for i := 1; i <= xRefTable.PageCount; i++ {

		d, _, _, err := xRefTable.PageDict(i, false)
		if err != nil {
			return nil, false, err
		}

		o, found := d.Find("Annots")
		if !found {
			continue
		}

		arr, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return nil, false, err
		}

		m, err := fieldsForAnnots(xRefTable, arr, fields)
		if err != nil {
			return nil, false, err
		}

		if err := exportPageFields(xRefTable, i, &form, m, &ok); err != nil {
			return nil, false, err
		}
	}

	formGroup.Forms = []Form{form}

	return &formGroup, ok, nil
}

// ExportFormJSON extracts form data originating from source from xRefTable and writes a JSON representation to w.
func ExportFormJSON(xRefTable *model.XRefTable, source string, w io.Writer) (bool, error) {

	formGroup, ok, err := ExportForm(xRefTable, source)
	if err != nil || !ok {
		return false, err
	}

	bb, err := json.MarshalIndent(formGroup, "", "\t")
	if err != nil {
		return false, err
	}

	_, err = w.Write(bb)

	return ok, err
}
