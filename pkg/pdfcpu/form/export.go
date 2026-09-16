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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/primitives"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const (

	// REQUIRED is used for required dict entries.
	REQUIRED = true

	// OPTIONAL is used for optional dict entries.
	OPTIONAL = false
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
	AltName   string `json:"altname,omitempty"`
	Default   string `json:"default,omitempty"`
	Value     string `json:"value"`
	MaxLen    int    `json:"maxlen,omitempty"`
	Multiline bool   `json:"multiline"`
	Locked    bool   `json:"locked"`
}

// DateField represents an Acroform date field.
type DateField struct {
	Pages   []int  `json:"pages"`
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	AltName string `json:"altname,omitempty"`
	Format  string `json:"format"`
	Default string `json:"default,omitempty"`
	Value   string `json:"value"`
	Locked  bool   `json:"locked"`
}

// CheckBox represents a form checkbox.
type CheckBox struct {
	Pages   []int  `json:"pages"`
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	AltName string `json:"altname,omitempty"`
	Default bool   `json:"default"`
	Value   bool   `json:"value"`
	Locked  bool   `json:"locked"`
}

// RadioButtonGroup represents a form radio button group.
type RadioButtonGroup struct {
	Pages   []int    `json:"pages"`
	ID      string   `json:"id"`
	Name    string   `json:"name,omitempty"`
	AltName string   `json:"altname,omitempty"`
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
	AltName  string   `json:"altname,omitempty"`
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
	AltName  string   `json:"altname,omitempty"`
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
	FileName          string              `json:"filename,omitempty"`
}

// FormGroup represents a JSON struct containing a sequence of form instances.
type FormGroup struct {
	Header Header `json:"header"`
	Forms  []Form `json:"forms"`
}

func (f Form) textFieldValueAndLock(id, name string) (string, bool, bool) {
	for _, tf := range f.TextFields {
		if tf == nil {
			continue
		}
		if tf.ID == id || tf.Name == name {
			return tf.Value, tf.Locked, true
		}
	}
	return "", false, false
}

func (f Form) dateFieldValueAndLock(id, name string) (string, bool, bool) {
	for _, df := range f.DateFields {
		if df == nil {
			continue
		}
		if df.ID == id || df.Name == name {
			return df.Value, df.Locked, true
		}
	}
	return "", false, false
}

func (f Form) checkBoxValueAndLock(id, name string) (bool, bool, bool) {
	for _, cb := range f.CheckBoxes {
		if cb == nil {
			continue
		}
		if cb.ID == id || cb.Name == name {
			return cb.Value, cb.Locked, true
		}
	}
	return false, false, false
}

func (f Form) radioButtonGroupValueAndLock(id, name string) (string, bool, bool) {
	for _, rbg := range f.RadioButtonGroups {
		if rbg == nil {
			continue
		}
		if rbg.ID == id || rbg.Name == name {
			return rbg.Value, rbg.Locked, true
		}
	}
	return "", false, false
}

func (f Form) comboBoxValueAndLock(id, name string) (string, bool, bool) {
	for _, cb := range f.ComboBoxes {
		if cb == nil {
			continue
		}
		if cb.ID == id || cb.Name == name {
			return cb.Value, cb.Locked, true
		}
	}
	return "", false, false
}

func (f Form) listBoxValuesAndLock(id, name string) ([]string, bool, bool) {
	for _, lb := range f.ListBoxes {
		if lb == nil {
			continue
		}
		if lb.ID == id || lb.Name == name {
			return lb.Values, lb.Locked, true
		}
	}
	return nil, false, false
}

func locateAPN(xRefTable *model.XRefTable, d types.Dict) (types.Dict, error) {
	obj, ok := d.Find("AP")
	if !ok {
		return nil, errors.New("corrupt form field: missing entry \"AP\"")
	}
	d1, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return nil, fmt.Errorf("entry AP: dereference: %w", err)
	}
	if len(d1) == 0 {
		return nil, errors.New("corrupt form field: missing entry \"AP\"")
	}

	obj, ok = d1.Find("N")
	if !ok {
		return nil, errors.New("corrupt AP field: missing entry \"N\"")
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return nil, fmt.Errorf("entry AP/N: dereference: %w", err)
	}
	if obj == nil {
		return nil, errors.New("corrupt AP field: missing entry \"N\"")
	}

	switch obj := obj.(type) {
	case types.Dict:
		if len(obj) == 0 {
			return nil, errors.New("corrupt AP field: missing entry \"N\"")
		}
		return obj, nil

	case types.StreamDict:
		// A single appearance stream does not provide named appearance states.
		return nil, nil
	}

	return nil, fmt.Errorf("corrupt AP field: entry \"N\" has unsupported type %T", obj)
}

func extractRadioButtonGroupOptions(xRefTable *model.XRefTable, d types.Dict) ([]string, bool, error) {
	var opts []string
	p := 0

	opts, err := parseOptions(xRefTable, d, OPTIONAL)
	if err != nil {
		return nil, false, err
	}

	if len(opts) > 0 {
		return opts, true, nil
	}

	for i, o := range d.ArrayEntry("Kids") {
		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			return nil, false, fmt.Errorf("entry Kids[%d]: dereference: %w", i, err)
		}

		indRef := d.IndirectRefEntry("P")
		if indRef != nil {
			if p == 0 {
				p = indRef.ObjectNumber.Value()
			} else if p != indRef.ObjectNumber.Value() {
				continue
			}
		}

		d1, err := locateAPN(xRefTable, d)
		if err != nil {
			return nil, false, fmt.Errorf("entry Kids[%d]: appearance: %w", i, err)
		}

		for k := range d1 {
			k, err := types.DecodeName(k)
			if err != nil {
				return nil, false, err
			}
			if k != "Off" && !types.MemberOf(k, opts) {
				opts = append(opts, k)
			}
		}
	}

	return opts, false, nil
}

func resolveOption(s string, opts []string, explicit bool) (string, error) {
	n, err := types.DecodeName(s)
	if err != nil {
		return "", err
	}
	if len(opts) > 0 && explicit {
		j, err := strconv.Atoi(n)
		if err != nil {
			return "", fmt.Errorf("option index %q: %w", n, err)
		}
		if j < 0 || j >= len(opts) {
			return "", fmt.Errorf("option index %d out of range [0,%d)", j, len(opts))
		}
		n = opts[j]
	}
	return n, nil
}

func extractRadioButtonGroup(xRefTable *model.XRefTable, page int, d types.Dict, id, name, altName string, locked bool) (*RadioButtonGroup, error) {
	rbg := &RadioButtonGroup{Pages: []int{page}, ID: id, Name: name, AltName: altName, Locked: locked}

	opts, explicit, err := extractRadioButtonGroupOptions(xRefTable, d)
	if err != nil {
		return nil, err
	}

	rbg.Options = opts

	if s, _, err := xRefTable.DereferenceNameEntry(d, "DV"); err != nil {
		return nil, fmt.Errorf("radio button group %s: %w", id, err)
	} else if s != nil {
		n, err := resolveOption(s.Value(), opts, explicit)
		if err != nil {
			return nil, err
		}
		rbg.Default = n
	}

	if s, _, err := xRefTable.DereferenceNameEntry(d, "V"); err != nil {
		return nil, fmt.Errorf("radio button group %s: %w", id, err)
	} else if s != nil {
		n, err := resolveOption(s.Value(), opts, explicit)
		if err != nil {
			return nil, err
		}
		if n != "Off" {
			rbg.Value = n
		}
	}

	return rbg, nil
}

func extractCheckBox(xRefTable *model.XRefTable, page int, d types.Dict, id, name, altName string, locked bool) (*CheckBox, error) {
	cb := &CheckBox{Pages: []int{page}, ID: id, Name: name, AltName: altName, Locked: locked}

	if n, _, err := xRefTable.DereferenceNameEntry(d, "DV"); err != nil {
		return nil, fmt.Errorf("checkbox %s: %w", id, err)
	} else if n != nil {
		cb.Default = n.Value() != "Off"
	}

	if n, _, err := xRefTable.DereferenceNameEntry(d, "V"); err != nil {
		return nil, fmt.Errorf("checkbox %s: %w", id, err)
	} else if n != nil {
		v := n.Value()
		cb.Value = len(v) > 0 && v != "Off"
	}

	return cb, nil
}

func extractComboBox(xRefTable *model.XRefTable, page int, d types.Dict, id, name, altName string, locked bool) (*ComboBox, error) {
	cb := &ComboBox{Pages: []int{page}, ID: id, Name: name, AltName: altName, Locked: locked}

	dv, _, err := xRefTable.DereferenceStringEntry(d, "DV")
	if err != nil {
		return nil, fmt.Errorf("entry DV: %w", err)
	}
	if dv != nil {
		cb.Default = strings.TrimSpace(*dv)
	}

	v, _, err := xRefTable.DereferenceStringEntry(d, "V")
	if err != nil {
		return nil, fmt.Errorf("entry V: %w", err)
	}
	if v != nil {
		cb.Value = strings.TrimSpace(*v)
	}

	opts, err := parseOptions(xRefTable, d, OPTIONAL)
	if err != nil {
		return nil, err
	}

	cb.Options = opts

	return cb, nil
}

func dateFormatFromJSAction(xRefTable *model.XRefTable, d types.Dict) (*primitives.DateFormat, error) {
	d1 := d.DictEntry("AA")
	if len(d1) > 0 {
		d2 := d1.DictEntry("F")
		if len(d2) > 0 {
			s, _, err := xRefTable.DereferenceStringEntry(d2, "JS")
			if err != nil {
				return nil, fmt.Errorf("date format action entry JS: %w", err)
			}
			if s != nil {
				value := *s
				i := strings.Index(value, "AFDate_FormatEx(\"")
				if i >= 0 {
					from := i + len("AFDate_FormatEx(\"")
					to := strings.IndexByte(value[from:], '"')
					if to < 0 {
						return nil, errors.New("date format action: missing closing quote")
					}
					value = value[from : from+to]
				}
				if df, err := primitives.DateFormatForFmtExt(value); err == nil {
					return df, nil
				}
			}
		}
	}
	return nil, nil
}

func dateStringEntry(xRefTable *model.XRefTable, d types.Dict, key string) (string, bool, error) {
	o, found := d.Find(key)
	if !found {
		return "", false, nil
	}

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return "", false, fmt.Errorf("entry %s: dereference: %w", key, err)
	}

	s, err := types.StringOrHexLiteral(o)
	if err != nil {
		return "", false, fmt.Errorf("entry %s: decode string: %w", key, err)
	}
	if s == nil {
		return "", true, nil
	}

	return *s, true, nil
}

func extractDateFormat(xRefTable *model.XRefTable, d types.Dict) (*primitives.DateFormat, error) {
	df, err := dateFormatFromJSAction(xRefTable, d)
	if err != nil {
		return nil, err
	}
	if df != nil {
		return df, nil
	}

	s, found, err := dateStringEntry(xRefTable, d, "DV")
	if err != nil {
		return nil, err
	}
	if found {
		if df, err := primitives.DateFormatForDate(s); err == nil {
			return df, nil
		}
	}

	s, found, err = dateStringEntry(xRefTable, d, "V")
	if err != nil {
		return nil, err
	}
	if found {
		if df, err := primitives.DateFormatForDate(s); err == nil {
			return df, nil
		}
	}

	return nil, nil
}

func extractDateField(c context.Context, xRefTable *model.XRefTable, page int, d types.Dict, id, name, altName string, df *primitives.DateFormat, locked bool) (*DateField, error) {
	dfield := &DateField{Pages: []int{page}, ID: id, Name: name, AltName: altName, Format: df.Ext, Locked: locked}

	v, err := getV(c, xRefTable, d)
	if err != nil {
		return nil, err
	}
	dfield.Value = v

	dv, err := getDV(c, xRefTable, d)
	if err != nil {
		return nil, err
	}
	dfield.Default = dv

	return dfield, nil
}

func extractTextField(c context.Context, xRefTable *model.XRefTable, page int, d types.Dict, id, name, altName string, ff *types.Integer, locked bool) (*TextField, error) {
	multiLine := ff != nil && uint(primitives.FieldFlags(ff.Value()))&uint(primitives.FieldMultiline) > 0

	maxLen := 0
	i, _, err := xRefTable.DereferenceIntegerEntry(d, "MaxLen")
	if err != nil {
		return nil, err
	}
	if i != nil {
		maxLen = i.Value()
	}

	tf := &TextField{Pages: []int{page}, ID: id, Name: name, AltName: altName, Multiline: multiLine, MaxLen: maxLen, Locked: locked}

	v, err := getV(c, xRefTable, d)
	if err != nil {
		return nil, err
	}
	tf.Value = v

	dv, err := getDV(c, xRefTable, d)
	if err != nil {
		return nil, err
	}
	tf.Default = dv

	return tf, nil
}

func extractListBox(xRefTable *model.XRefTable, page int, d types.Dict, id, name, altName string, locked, multi bool) (*ListBox, error) {
	lb := &ListBox{Pages: []int{page}, ID: id, Name: name, AltName: altName, Locked: locked, Multi: multi}

	if !multi {
		dv, _, err := xRefTable.DereferenceStringEntry(d, "DV")
		if err != nil {
			return nil, fmt.Errorf("entry DV: %w", err)
		}
		if dv != nil {
			lb.Defaults = []string{strings.TrimSpace(*dv)}
		}
		v, _, err := xRefTable.DereferenceStringEntry(d, "V")
		if err != nil {
			return nil, fmt.Errorf("entry V: %w", err)
		}
		if v != nil {
			lb.Values = []string{strings.TrimSpace(*v)}
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

	opts, err := parseOptions(xRefTable, d, OPTIONAL)
	if err != nil {
		return nil, err
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

func fieldsForAnnots(c context.Context, xRefTable *model.XRefTable, annots, fields types.Array) (map[string]fieldInfo, error) {
	m := map[string]fieldInfo{}
	var prevId string

	for i, v := range annots {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		indRef, err := indirectRef(v, "page Annots", i)
		if err != nil {
			return nil, err
		}

		ok, fi, err := isField(c, xRefTable, indRef, fields)
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

func exportBtn(xRefTable *model.XRefTable, i int, form *Form, d types.Dict, id, name, altName string, locked bool, ok *bool, ff *types.Integer) error {
	if ff != nil && primitives.FieldFlags(ff.Value())&primitives.FieldPushbutton > 0 {
		return nil
	}

	if len(d.ArrayEntry("Kids")) > 1 {

		for _, rb := range form.RadioButtonGroups {
			if rb.ID == id && rb.Name == name {
				rb.Pages = append(rb.Pages, i)
				return nil
			}
		}

		rbg, err := extractRadioButtonGroup(xRefTable, i, d, id, name, altName, locked)
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

	cb, err := extractCheckBox(xRefTable, i, d, id, name, altName, locked)
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
	id, name, altName string,
	locked bool,
	ok *bool) error {
	ff, _, err := xRefTable.DereferenceIntegerEntry(d, "Ff")
	if err != nil {
		return err
	}

	if ff != nil && primitives.FieldFlags(ff.Value())&primitives.FieldCombo > 0 {

		for _, cb := range form.ComboBoxes {
			if cb.Name == name && cb.ID == id {
				cb.Pages = append(cb.Pages, i)
				return nil
			}
		}

		cb, err := extractComboBox(xRefTable, i, d, id, name, altName, locked)
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

	multi := ff != nil && primitives.FieldFlags(ff.Value())&primitives.FieldMultiselect > 0
	lb, err := extractListBox(xRefTable, i, d, id, name, altName, locked, multi)
	if err != nil {
		return err
	}

	form.ListBoxes = append(form.ListBoxes, lb)
	*ok = true
	return nil
}

func exportTx(c context.Context, xRefTable *model.XRefTable, i int, form *Form, d types.Dict, id, name, altName string, ff *types.Integer, locked bool, ok *bool) error {
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

		df, err := extractDateField(c, xRefTable, i, d, id, name, altName, df, locked)
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

	tf, err := extractTextField(c, xRefTable, i, d, id, name, altName, ff, locked)
	if err != nil {
		return err
	}

	form.TextFields = append(form.TextFields, tf)
	*ok = true
	return nil
}

func exportPageField(c context.Context, ft string, xRefTable *model.XRefTable, i int, form *Form, d types.Dict, id, name, altName string, locked bool, ok *bool, ff *types.Integer) error {
	var err error

	switch ft {
	case "Btn":
		err = exportBtn(xRefTable, i, form, d, id, name, altName, locked, ok, ff)
	case "Ch":
		err = exportCh(xRefTable, i, form, d, id, name, altName, locked, ok)
	case "Tx":
		err = exportTx(c, xRefTable, i, form, d, id, name, altName, ff, locked, ok)
	}

	return err
}

func exportPageFields(c context.Context, xRefTable *model.XRefTable, i int, form *Form, m map[string]fieldInfo, ok *bool) error {
	for id, fi := range m {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		name := fi.name

		d, err := xRefTable.DereferenceDict(*fi.indRef)
		if err != nil {
			return fmt.Errorf("field %s obj#%d: dereference: %w", id, fi.indRef.ObjectNumber.Value(), err)
		}
		if len(d) == 0 {
			continue
		}

		ff, locked, err := formFieldFlags(xRefTable, d)
		if err != nil {
			return fmt.Errorf("field %s: %w", id, err)
		}

		ft := fi.ft
		if ft == nil {
			ft, _, err = xRefTable.DereferenceNameEntry(d, "FT")
			if err != nil {
				return fmt.Errorf("field %s: entry FT: %w", id, err)
			}
			if ft == nil {
				return errors.New("corrupt form field: missing entry FT")
			}
		}

		altName := ""
		s, found, err := xRefTable.DereferenceStringEntry(d, "TU")
		if found && s == nil {
			if err == nil {
				err = errors.New("expected StringLiteral or HexLiteral")
			}
		}
		if err != nil {
			return fmt.Errorf("field %s: entry TU: %w", id, err)
		}
		if found {
			altName = *s
		}

		if err := exportPageField(c, ft.Value(), xRefTable, i, form, d, id, name, altName, locked, ok, ff); err != nil {
			return fmt.Errorf("field %s: %w", id, err)
		}
	}

	return nil
}

// ExportForm extracts form data originating from source from xRefTable and supports cancellation.
func ExportForm(c context.Context, xRefTable *model.XRefTable, source string) (*FormGroup, bool, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, false, err
	}
	fields, err := Fields(xRefTable)
	if err != nil {
		return nil, false, fmt.Errorf("AcroForm Fields: %w", err)
	}

	formGroup := FormGroup{}
	formGroup.Header = header(xRefTable, source)

	form := Form{}

	var ok bool

	for i := 1; i <= xRefTable.PageCount; i++ {
		if err := contextutil.Check(c); err != nil {
			return nil, false, err
		}

		d, _, _, err := xRefTable.PageDict(c, i, false)
		if err != nil {
			return nil, false, fmt.Errorf("page %d: page dictionary: %w", i, err)
		}

		o, found := d.Find("Annots")
		if !found {
			continue
		}

		arr, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return nil, false, fmt.Errorf("page %d: Annots: %w", i, err)
		}

		m, err := fieldsForAnnots(c, xRefTable, arr, fields)
		if err != nil {
			return nil, false, fmt.Errorf("page %d: resolve fields: %w", i, err)
		}

		if err := exportPageFields(c, xRefTable, i, &form, m, &ok); err != nil {
			return nil, false, fmt.Errorf("page %d: export fields: %w", i, err)
		}
	}

	formGroup.Forms = []Form{form}

	return &formGroup, ok, contextutil.Check(c)
}

type exportFormFunc func(context.Context, *model.XRefTable, string) (*FormGroup, bool, error)

type marshalFormJSONFunc func(any, string, string) ([]byte, error)

func exportFormJSON(
	c context.Context,
	xRefTable *model.XRefTable,
	source string,
	w io.Writer,
	export exportFormFunc,
	marshal marshalFormJSONFunc,
) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	if w == nil {
		return false, ErrMissingJSONWriter
	}
	formGroup, ok, err := export(c, xRefTable, source)
	if err != nil {
		return false, fmt.Errorf("collect data: %w", err)
	}
	if !ok {
		return false, nil
	}

	bb, err := marshal(formGroup, "", "\t")
	if err != nil {
		return false, fmt.Errorf("encode JSON: %w", err)
	}
	if err := contextutil.Check(c); err != nil {
		return false, err
	}

	n, err := w.Write(bb)
	if err != nil {
		return false, fmt.Errorf("write JSON: %w", err)
	}
	if n != len(bb) {
		return false, fmt.Errorf("write JSON: %w", io.ErrShortWrite)
	}
	return true, contextutil.Check(c)
}

// ExportFormJSON extracts form data originating from source from xRefTable and writes a JSON representation to w.
// It returns true when form fields were exported and written. It returns false with a nil error when no exportable
// form fields were found. It supports cancellation.
func ExportFormJSON(c context.Context, xRefTable *model.XRefTable, source string, w io.Writer) (bool, error) {
	return exportFormJSON(c, xRefTable, source, w, ExportForm, json.MarshalIndent)
}
