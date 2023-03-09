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

// TextField represents an Acroform text field.
type TextField struct {
	ID        string `json:"id"`
	ObjectNr  int    `json:"objectnr,omitempty"`
	Default   string `json:"default,omitempty"`
	Value     string `json:"value"`
	Multiline bool   `json:"multiline"`
	Locked    bool   `json:"locked"`
}

// DateField represents an Acroform date field.
type DateField struct {
	ID       string `json:"id"`
	ObjectNr int    `json:"objectnr,omitempty"`
	Format   string `json:"format"`
	Default  string `json:"default,omitempty"`
	Value    string `json:"value"`
	Locked   bool   `json:"locked"`
}

// RadioButtonGroup represents an Acroform checkbox.
type CheckBox struct {
	ID       string `json:"id"`
	ObjectNr int    `json:"objectnr,omitempty"`
	Default  bool   `json:"default"`
	Value    bool   `json:"value"`
	Locked   bool   `json:"locked"`
}

// RadioButtonGroup represents an Acroform radio button group.
type RadioButtonGroup struct {
	ID       string   `json:"id"`
	ObjectNr int      `json:"objectnr,omitempty"`
	Options  []string `json:"options"`
	Default  string   `json:"default,omitempty"`
	Value    string   `json:"value"`
	Locked   bool     `json:"locked"`
}

// ListBox represents an Acroform combobox.
type ComboBox struct {
	ID       string   `json:"id"`
	ObjectNr int      `json:"objectnr,omitempty"`
	Editable bool     `json:"editable"`
	Options  []string `json:"options"`
	Default  string   `json:"default,omitempty"`
	Value    string   `json:"value"`
	Locked   bool     `json:"locked"`
}

// ListBox represents an Acroform listbox.
type ListBox struct {
	ID       string   `json:"id"`
	ObjectNr int      `json:"objectnr,omitempty"`
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

func (f Form) textFieldValueAndLock(id string) (string, bool, bool) {
	for _, tf := range f.TextFields {
		if tf.ID == id {
			return tf.Value, tf.Locked, true
		}
	}
	return "", false, false
}

func (f Form) dateFieldValueAndLock(id string) (string, bool, bool) {
	for _, df := range f.DateFields {
		if df.ID == id {
			return df.Value, df.Locked, true
		}
	}
	return "", false, false
}

func (f Form) checkBoxValueAndLock(id string) (bool, bool, bool) {
	for _, cb := range f.CheckBoxes {
		if cb.ID == id {
			return cb.Value, cb.Locked, true
		}
	}
	return false, false, false
}

func (f Form) radioButtonGroupValueAndLock(id string) (string, bool, bool) {
	for _, rbg := range f.RadioButtonGroups {
		if rbg.ID == id {
			return rbg.Value, rbg.Locked, true
		}
	}
	return "", false, false
}

func (f Form) comboBoxValueAndLock(id string) (string, bool, bool) {
	for _, cb := range f.ComboBoxes {
		if cb.ID == id {
			return cb.Value, cb.Locked, true
		}
	}
	return "", false, false
}

func (f Form) listBoxValuesAndLock(id string) ([]string, bool, bool) {
	for _, lb := range f.ListBoxes {
		if lb.ID == id {
			return lb.Values, lb.Locked, true
		}
	}
	return nil, false, false
}

func extractRadioButtonGroup(xRefTable *model.XRefTable, d types.Dict, id string, objectNr int, locked bool) (*RadioButtonGroup, error) {

	rbg := &RadioButtonGroup{ID: id, Locked: locked, ObjectNr: objectNr}

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

	var opts []string

	for _, o := range d.ArrayEntry("Kids") {
		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			return nil, err
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
				opts = append(opts, k)
			}
		}
	}

	rbg.Options = opts

	return rbg, nil
}

func extractCheckBox(d types.Dict, id string, objectNr int, locked bool) (*CheckBox, error) {

	cb := &CheckBox{ID: id, Locked: locked, ObjectNr: objectNr}

	if o, ok := d.Find("DV"); ok {
		cb.Default = o.(types.Name) == "Yes"
	}

	if o, ok := d.Find("V"); ok {
		cb.Value = o.(types.Name) == "Yes"
	}

	return cb, nil
}

func extractComboBox(xRefTable *model.XRefTable, d types.Dict, id string, objectNr int, locked bool) (*ComboBox, error) {

	cb := &ComboBox{ID: id, Locked: locked, ObjectNr: objectNr}

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

func extractDateField(d types.Dict, id string, df *primitives.DateFormat, objectNr int, locked bool) (*DateField, error) {

	dfield := &DateField{ID: id, Format: df.Ext, Locked: locked, ObjectNr: objectNr}

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

func extractTextField(d types.Dict, id string, ff *int, objectNr int, locked bool) (*TextField, error) {

	multiLine := ff != nil && uint(primitives.FieldFlags(*ff))&uint(primitives.FieldMultiline) > 0

	tf := &TextField{ID: id, Multiline: multiLine, Locked: locked, ObjectNr: objectNr}

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

func extractListBox(xRefTable *model.XRefTable, d types.Dict, id string, objectNr int, locked, multi bool) (*ListBox, error) {

	lb := &ListBox{ID: id, Locked: locked, Multi: multi, ObjectNr: objectNr}

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

func fieldsForAnnots(xRefTable *model.XRefTable, annots, fields types.Array) ([]string, map[string]types.IndirectRef, map[string]*string, error) {

	var ids []string
	m := map[string]types.IndirectRef{}
	tm := map[string]*string{}
	var prevId string

	for _, v := range annots {

		indRef := v.(types.IndirectRef)

		ok, pIndRef, id, ft, err := isField(xRefTable, indRef, fields)
		if err != nil {
			return nil, nil, nil, err
		}
		if !ok {
			continue
		}

		if pIndRef != nil {
			indRef = *pIndRef
		}
		m[id] = indRef

		tm[id] = ft

		if id != prevId {
			ids = append(ids, id)
			prevId = id
		}
	}

	return ids, m, tm, nil
}

// ExportForm extracts form data originating from source from xRefTable and writes a JSON representation to w.
func ExportForm(xRefTable *model.XRefTable, source string, w io.Writer) (bool, error) {

	fields, err := fields(xRefTable)
	if err != nil {
		return false, err
	}

	formGroup := FormGroup{}
	formGroup.Header = header(xRefTable, source)

	form := Form{}

	var ok bool

	for i := 1; i <= xRefTable.PageCount; i++ {

		d, _, _, err := xRefTable.PageDict(i, false)
		if err != nil {
			return false, err
		}

		o, found := d.Find("Annots")
		if !found {
			continue
		}

		arr, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return false, err
		}

		ids, fieldMap, typeMap, err := fieldsForAnnots(xRefTable, arr, fields)
		if err != nil {
			return false, err
		}

		for _, id := range ids {

			indRef := fieldMap[id]

			d, err := xRefTable.DereferenceDict(indRef)
			if err != nil {
				return false, err
			}
			if len(d) == 0 {
				continue
			}

			var locked bool
			ff := d.IntEntry("Ff")
			if ff != nil {
				locked = uint(primitives.FieldFlags(*ff))&uint(primitives.FieldReadOnly) > 0
			}

			ft := typeMap[id]
			if ft == nil {
				ft = d.NameEntry("FT")
				if ft == nil {
					return false, errors.New("pdfcpu: corrupt form field: missing entry FT")
				}
			}

			switch *ft {

			case "Btn":
				if len(d.ArrayEntry("Kids")) > 0 {
					rbg, err := extractRadioButtonGroup(xRefTable, d, id, int(indRef.ObjectNumber), locked)
					if err != nil {
						return false, err
					}
					form.RadioButtonGroups = append(form.RadioButtonGroups, rbg)
					ok = true
					continue
				}
				cb, err := extractCheckBox(d, id, int(indRef.ObjectNumber), locked)
				if err != nil {
					return false, err
				}
				form.CheckBoxes = append(form.CheckBoxes, cb)
				ok = true

			case "Ch":
				ff := d.IntEntry("Ff")
				if ff == nil {
					return false, errors.New("pdfcpu: corrupt form field: missing entry Ff")
				}
				if primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 {
					cb, err := extractComboBox(xRefTable, d, id, int(indRef.ObjectNumber), locked)
					if err != nil {
						return false, err
					}
					form.ComboBoxes = append(form.ComboBoxes, cb)
					ok = true
					continue
				}
				multi := primitives.FieldFlags(*ff)&primitives.FieldMultiselect > 0
				lb, err := extractListBox(xRefTable, d, id, int(indRef.ObjectNumber), locked, multi)
				if err != nil {
					return false, err
				}
				form.ListBoxes = append(form.ListBoxes, lb)
				ok = true

			case "Tx":

				df, err := extractDateFormat(xRefTable, d)
				if err != nil {
					return false, err
				}
				if df != nil {
					df, err := extractDateField(d, id, df, int(indRef.ObjectNumber), locked)
					if err != nil {
						return false, err
					}
					form.DateFields = append(form.DateFields, df)
					ok = true
					continue
				}
				tf, err := extractTextField(d, id, ff, int(indRef.ObjectNumber), locked)
				if err != nil {
					return false, err
				}
				form.TextFields = append(form.TextFields, tf)
				ok = true
			}

		}

	}

	formGroup.Forms = []Form{form}

	bb, err := json.MarshalIndent(formGroup, "", "\t")
	if err != nil {
		return false, err
	}

	_, err = w.Write(bb)

	return ok, err
}
