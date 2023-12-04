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
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/primitives"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// FieldType represents a form field type.
type FieldType int

const (
	FTText FieldType = iota
	FTDate
	FTCheckBox
	FTComboBox
	FTListBox
	FTRadioButtonGroup
)

func (ft FieldType) string() string {
	var s string
	switch ft {
	case FTText:
		s = "Textfield"
	case FTDate:
		s = "Datefield"
	case FTCheckBox:
		s = "CheckBox"
	case FTComboBox:
		s = "ComboBox"
	case FTListBox:
		s = "ListBox"
	case FTRadioButtonGroup:
		s = "RadioBGr."
	}
	return s
}

// Field represents a form field for s particular page number.
type Field struct {
	Pages  []int
	Locked bool
	Typ    FieldType
	ID     string
	Name   string
	Dv     string
	V      string
	Opts   string
}

func (f Field) pageString() string {
	if len(f.Pages) == 1 {
		return strconv.Itoa(f.Pages[0])
	}
	sort.Ints(f.Pages)
	ss := []string{}
	for _, p := range f.Pages {
		ss = append(ss, strconv.Itoa(p))
	}
	return strings.Join(ss, ",")
}

type FieldMeta struct {
	def, val, opt                           bool
	pageMax, defMax, valMax, idMax, nameMax int
}

func fields(xRefTable *model.XRefTable) (types.Array, error) {

	if xRefTable.Form == nil {
		return nil, errors.New("pdfcpu: no form available")
	}

	o, ok := xRefTable.Form.Find("Fields")
	if !ok {
		return nil, errors.New("pdfcpu: no form fields available")
	}

	fields, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return nil, err
	}

	if len(fields) == 0 {
		return nil, errors.New("pdfcpu: no form fields available")
	}

	return fields, nil
}

func fullyQualifiedFieldName(xRefTable *model.XRefTable, indRef types.IndirectRef, fields types.Array, id, name *string) (bool, error) {

	d, err := xRefTable.DereferenceDict(indRef)
	if err != nil {
		return false, err
	}
	if len(d) == 0 {
		return false, errors.Errorf("pdfcpu: corrupt field")
	}

	thisID := indRef.ObjectNumber.String()
	thisName := ""
	s, err := d.StringOrHexLiteralEntry("T")
	if err != nil {
		return false, err
	}
	if s != nil {
		thisName = *s
	}

	pIndRef := d.IndirectRefEntry("Parent")
	if pIndRef == nil {
		for i := 0; i < len(fields); i++ {
			if ir, ok := fields[i].(types.IndirectRef); ok && ir == indRef {
				*id = thisID
				*name = thisName
				return true, nil
			}
		}
		return false, nil
	}

	// non-terminal field

	ok, err := fullyQualifiedFieldName(xRefTable, *pIndRef, fields, id, name)
	if !ok || err != nil {
		return false, err
	}

	*id += "." + thisID
	if len(*name) > 0 && len(thisName) > 0 {
		*name += "." + thisName
	}

	return true, nil
}

type fieldInfo struct {
	id     string
	name   string
	ft     *string
	indRef *types.IndirectRef
}

func isField(xRefTable *model.XRefTable, indRef types.IndirectRef, fields types.Array) (bool, *fieldInfo, error) {

	d, err := xRefTable.DereferenceDict(indRef)
	if err != nil {
		return false, nil, err
	}
	if len(d) == 0 {
		return false, nil, nil
	}

	var (
		id, name string
		ft       *string
	)

	pIndRef := d.IndirectRefEntry("Parent")
	if pIndRef != nil {
		dp, err := xRefTable.DereferenceDict(*pIndRef)
		if err != nil {
			return false, nil, err
		}
		if len(dp) == 0 {
			return false, nil, nil
		}
		ft = dp.NameEntry("FT")
		if ft != nil && (*ft == "Btn" || *ft == "Tx") {
			// rbg or text/datefield hierarchy
			ok, err := fullyQualifiedFieldName(xRefTable, *pIndRef, fields, &id, &name)
			if !ok || err != nil {
				return false, nil, err
			}
			return true, &fieldInfo{id: id, name: name, ft: ft, indRef: pIndRef}, nil
		}
	}

	ok, err := fullyQualifiedFieldName(xRefTable, indRef, fields, &id, &name)
	if !ok || err != nil {
		return false, nil, err
	}

	if ft == nil {
		ft = d.NameEntry("FT")
	}
	return true, &fieldInfo{id: id, name: name, ft: ft}, nil
}

func extractStringSlice(a types.Array) ([]string, error) {
	var ss []string
	for _, o := range a {
		sl, _ := o.(types.StringLiteral)
		s, err := types.StringLiteralToString(sl)
		if err != nil {
			return nil, err
		}
		ss = append(ss, s)
	}
	return ss, nil
}

func parseOptions(xRefTable *model.XRefTable, d types.Dict) ([]string, error) {
	o, _ := d.Find("Opt")
	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return nil, err
	}
	return extractStringSlice(a)
}

func parseStringLiteralArray(xRefTable *model.XRefTable, d types.Dict, key string) ([]string, error) {
	o, _ := d.Find(key)
	if o == nil {
		return nil, nil
	}

	switch o := o.(type) {

	case types.StringLiteral:
		s, err := types.StringLiteralToString(o)
		if err != nil {
			return nil, err
		}
		return []string{s}, nil

	case types.Array:
		a, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return nil, err
		}
		return extractStringSlice(a)
	}

	return nil, nil
}

func collectRadioButtonGroupOptions(xRefTable *model.XRefTable, d types.Dict) (string, error) {

	var vv []string

	for _, o := range d.ArrayEntry("Kids") {
		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			return "", err
		}
		d1 := d.DictEntry("AP")
		if d1 == nil {
			return "", errors.New("corrupt form field: missing entry AP")
		}
		d2 := d1.DictEntry("N")
		if d2 == nil {
			return "", errors.New("corrupt AP field: missing entry N")
		}
		for k := range d2 {
			k, err := types.DecodeName(k)
			if err != nil {
				return "", err
			}
			if k != "Off" {
				found := false
				for _, opt := range vv {
					if opt == k {
						found = true
						break
					}
				}
				if !found {
					vv = append(vv, k)
				}
				break
			}
		}
	}

	return strings.Join(vv, ","), nil
}

func collectRadioButtonGroup(xRefTable *model.XRefTable, d types.Dict, f *Field, fm *FieldMeta) error {

	f.Typ = FTRadioButtonGroup

	if s := d.NameEntry("V"); s != nil {
		v, err := types.DecodeName(*s)
		if err != nil {
			return err
		}
		if v != "Off" {
			if w := runewidth.StringWidth(v); w > fm.valMax {
				fm.valMax = w
			}
			fm.val = true
			f.V = v
		}
	}

	s, err := collectRadioButtonGroupOptions(xRefTable, d)
	if err != nil {
		return err
	}

	f.Opts = s
	if len(f.Opts) > 0 {
		fm.opt = true
	}

	return nil
}

func collectBtn(xRefTable *model.XRefTable, d types.Dict, f *Field, fm *FieldMeta) error {

	ff := d.IntEntry("Ff")
	if ff != nil && primitives.FieldFlags(*ff)&primitives.FieldPushbutton > 0 {
		return nil
	}

	v := types.Name("Off")
	if s, found := d.Find("DV"); found {
		v = s.(types.Name)
	}
	dv, err := types.DecodeName(v.String())
	if err != nil {
		return err
	}

	if dv != "Off" {
		if w := runewidth.StringWidth(dv); w > fm.defMax {
			fm.defMax = w
		}
		fm.def = true
		f.Dv = dv
	}

	if len(d.ArrayEntry("Kids")) > 0 {
		return collectRadioButtonGroup(xRefTable, d, f, fm)
	}

	f.Typ = FTCheckBox
	if o, found := d.Find("V"); found {
		if o.(types.Name) == "Yes" {
			v := "Yes"
			if len(v) > fm.valMax {
				fm.valMax = len(v)
			}
			fm.val = true
			f.V = v
		}
	}

	return nil
}

func collectComboBox(xRefTable *model.XRefTable, d types.Dict, f *Field, fm *FieldMeta) error {
	f.Typ = FTComboBox
	if sl := d.StringLiteralEntry("V"); sl != nil {
		v, err := types.StringLiteralToString(*sl)
		if err != nil {
			return err
		}
		if w := runewidth.StringWidth(v); w > fm.valMax {
			fm.valMax = w
		}
		fm.val = true
		f.V = v
	}
	if sl := d.StringLiteralEntry("DV"); sl != nil {
		dv, err := types.StringLiteralToString(*sl)
		if err != nil {
			return err
		}
		if w := runewidth.StringWidth(dv); w > fm.defMax {
			fm.defMax = w
		}
		fm.def = true
		f.Dv = dv
	}
	return nil
}

func collectListBox(xRefTable *model.XRefTable, multi bool, d types.Dict, f *Field, fm *FieldMeta) error {
	f.Typ = FTListBox
	if !multi {
		if sl := d.StringLiteralEntry("V"); sl != nil {
			v, err := types.StringLiteralToString(*sl)
			if err != nil {
				return err
			}
			if w := runewidth.StringWidth(v); w > fm.valMax {
				fm.valMax = w
			}
			fm.val = true
			f.V = v
		}
		if sl := d.StringLiteralEntry("DV"); sl != nil {
			dv, err := types.StringLiteralToString(*sl)
			if err != nil {
				return err
			}
			if w := runewidth.StringWidth(dv); w > fm.defMax {
				fm.defMax = w
			}
			fm.def = true
			f.Dv = dv
		}
	} else {
		vv, err := parseStringLiteralArray(xRefTable, d, "V")
		if err != nil {
			return err
		}
		if len(vv) > 0 {
			v := strings.Join(vv, ",")
			if w := runewidth.StringWidth(v); w > fm.valMax {
				fm.valMax = w
			}
			fm.val = true
			f.V = v
		}
		vv, err = parseStringLiteralArray(xRefTable, d, "DV")
		if err != nil {
			return err
		}
		if len(vv) > 0 {
			dv := strings.Join(vv, ",")
			if w := runewidth.StringWidth(dv); w > fm.defMax {
				fm.defMax = w
			}
			fm.def = true
			f.Dv = dv
		}
	}
	return nil
}

func collectCh(xRefTable *model.XRefTable, d types.Dict, f *Field, fm *FieldMeta) error {
	ff := d.IntEntry("Ff")

	vv, err := parseOptions(xRefTable, d)
	if err != nil {
		return err
	}

	f.Opts = strings.Join(vv, ",")
	if len(f.Opts) > 0 {
		fm.opt = true
	}

	if ff != nil && primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 {
		return collectComboBox(xRefTable, d, f, fm)
	}

	multi := ff != nil && (primitives.FieldFlags(*ff)&primitives.FieldMultiselect > 0)

	return collectListBox(xRefTable, multi, d, f, fm)
}

func collectTx(xRefTable *model.XRefTable, d types.Dict, f *Field, fm *FieldMeta) error {
	if o, found := d.Find("V"); found {
		sl, _ := o.(types.StringLiteral)
		s, err := types.StringLiteralToString(sl)
		if err != nil {
			return err
		}
		v := s
		if i := strings.Index(s, "\n"); i >= 0 {
			v = s[:i]
			v += "\\n"
		}
		if w := runewidth.StringWidth(v); w > fm.valMax {
			fm.valMax = w
		}
		fm.val = true
		f.V = v
	}
	if o, found := d.Find("DV"); found {
		sl, _ := o.(types.StringLiteral)
		s, err := types.StringLiteralToString(sl)
		if err != nil {
			return err
		}
		dv := s
		if i := strings.Index(s, "\n"); i >= 0 {
			dv = dv[:i]
			dv += "\\n"
		}

		if w := runewidth.StringWidth(dv); w > fm.defMax {
			fm.defMax = w
		}
		fm.def = true
		f.Dv = dv
	}
	df, err := extractDateFormat(xRefTable, d)
	if err != nil {
		return err
	}
	f.Typ = FTText
	if df != nil {
		f.Typ = FTDate
	}
	return nil
}

func collectPageField(
	xRefTable *model.XRefTable,
	d types.Dict,
	i int,
	fi *fieldInfo,
	fm *FieldMeta,
	fs *[]Field) error {

	exists := false
	for j, field := range *fs {
		if field.ID == fi.id && field.Name == fi.name {
			field.Pages = append(field.Pages, i)
			ps := field.pageString()
			if len(ps) > fm.pageMax {
				fm.pageMax = len(ps)
			}
			(*fs)[j] = field
			exists = true
		}
	}

	f := Field{Pages: []int{i}}

	f.ID = fi.id
	if w := runewidth.StringWidth(fi.id); w > fm.idMax {
		fm.idMax = w
	}

	f.Name = fi.name
	if w := runewidth.StringWidth(fi.name); w > fm.nameMax {
		fm.nameMax = w
	}

	var locked bool
	ff := d.IntEntry("Ff")
	if ff != nil {
		locked = uint(primitives.FieldFlags(*ff))&uint(primitives.FieldReadOnly) > 0
	}
	f.Locked = locked

	ft := fi.ft
	if ft == nil {
		ft = d.NameEntry("FT")
		if ft == nil {
			return errors.Errorf("pdfcpu: corrupt form field %s: missing entry FT\n%s", f.ID, d)
		}
	}

	var err error

	switch *ft {
	case "Btn":
		err = collectBtn(xRefTable, d, &f, fm)

	case "Ch":
		err = collectCh(xRefTable, d, &f, fm)

	case "Tx":
		err = collectTx(xRefTable, d, &f, fm)
	}

	if err != nil {
		return err
	}

	if !exists {
		*fs = append(*fs, f)
	}

	return nil
}

func collectPageFields(
	xRefTable *model.XRefTable,
	wAnnots model.Annot,
	fields types.Array,
	p int,
	fm *FieldMeta,
	fs *[]Field) error {

	indRefs := map[types.IndirectRef]bool{}

	for _, ir := range *(wAnnots.IndRefs) {

		ok, fi, err := isField(xRefTable, ir, fields)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		if fi.indRef != nil {
			if indRefs[*fi.indRef] {
				continue
			}
			indRefs[*fi.indRef] = true
			ir = *fi.indRef
		}

		d, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return err
		}
		if len(d) == 0 {
			continue
		}

		if err := collectPageField(xRefTable, d, p, fi, fm, fs); err != nil {
			return err
		}
	}

	return nil
}

func collectFields(xRefTable *model.XRefTable, fields types.Array, fm *FieldMeta) ([]Field, error) {
	var fs []Field

	for p := 1; p <= xRefTable.PageCount; p++ {

		pgAnnots := xRefTable.PageAnnots[p]
		if len(pgAnnots) == 0 {
			continue
		}

		wAnnots, ok := pgAnnots[model.AnnWidget]
		if !ok {
			continue
		}

		if err := collectPageFields(xRefTable, wAnnots, fields, p, fm, &fs); err != nil {
			return nil, err
		}
	}

	return fs, nil
}

func calcListHeader(fm *FieldMeta) (string, []int) {
	horSep := []int{}

	s := "Pg "
	if fm.pageMax > 2 {
		s += strings.Repeat(" ", fm.pageMax-2)
		horSep = append(horSep, 15+fm.pageMax-2)
	} else {
		horSep = append(horSep, 15)
	}

	s += "L Field     " + draw.VBar + " Id  "
	if fm.idMax > 3 {
		s += strings.Repeat(" ", fm.idMax-3)
		horSep = append(horSep, 5+fm.idMax-3)
	} else {
		horSep = append(horSep, 5)
	}

	s += draw.VBar + " Name "
	if fm.nameMax > 4 {
		s += strings.Repeat(" ", fm.nameMax-4)
		horSep = append(horSep, 6+fm.nameMax-4)
	} else {
		horSep = append(horSep, 6)
	}

	if fm.def {
		s += draw.VBar + " Default "
		if fm.defMax > 7 {
			s += strings.Repeat(" ", fm.defMax-7)
			horSep = append(horSep, 9+fm.defMax-7)
		} else {
			horSep = append(horSep, 9)
		}
	}
	if fm.val {
		s += draw.VBar + " Value "
		if fm.valMax > 5 {
			s += strings.Repeat(" ", fm.valMax-5)
			horSep = append(horSep, 7+fm.valMax-5)
		} else {
			horSep = append(horSep, 7)
		}
	}
	if fm.opt {
		s += draw.VBar + " Options"
		horSep = append(horSep, 8)
	}

	return s, horSep
}

func multiPageFieldsMap(fs []Field) map[string][]Field {

	m := map[string][]Field{}

	for _, f := range fs {
		if len(f.Pages) == 1 {
			continue
		}
		ps := f.pageString()
		var fields []Field
		if fs, ok := m[ps]; ok {
			fields = append(fs, f)
		} else {
			fields = []Field{f}
		}
		m[ps] = fields
	}

	return m
}

func renderMultiPageFields(ctx *model.Context, m map[string][]Field, fm *FieldMeta) ([]string, error) {

	var ss []string

	s, horSep := calcListHeader(fm)

	ss = append(ss, "Multi page fields:")
	ss = append(ss, s)
	ss = append(ss, draw.HorSepLine(horSep))

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	p := ""

	for _, k := range keys {

		if p != "" {
			ss = append(ss, draw.HorSepLine(horSep))
		}
		p = k

		for _, f := range m[k] {
			l := " "
			if f.Locked {
				l = "*"
			}

			t := f.Typ.string()

			pageFill := strings.Repeat(" ", fm.pageMax-runewidth.StringWidth(f.pageString()))
			idFill := strings.Repeat(" ", fm.idMax-runewidth.StringWidth(f.ID))
			nameFill := strings.Repeat(" ", fm.nameMax-runewidth.StringWidth(f.Name))
			s := fmt.Sprintf("%s%s %s %-9s %s %s%s %s %s%s ", p, pageFill, l, t, draw.VBar, f.ID, idFill, draw.VBar, f.Name, nameFill)
			p = strings.Repeat(" ", len(p))
			if fm.def {
				dvFill := strings.Repeat(" ", fm.defMax-runewidth.StringWidth(f.Dv))
				s += fmt.Sprintf("%s %s%s ", draw.VBar, f.Dv, dvFill)
			}
			if fm.val {
				vFill := strings.Repeat(" ", fm.valMax-runewidth.StringWidth(f.V))
				s += fmt.Sprintf("%s %s%s ", draw.VBar, f.V, vFill)
			}
			if fm.opt {
				s += fmt.Sprintf("%s %s", draw.VBar, f.Opts)
			}

			ss = append(ss, s)
		}
	}

	ss = append(ss, "")

	return ss, nil
}

func renderFields(ctx *model.Context, fs []Field, fm *FieldMeta) ([]string, error) {

	ss := []string{}

	m := multiPageFieldsMap(fs)

	if len(m) > 0 {
		ss1, err := renderMultiPageFields(ctx, m, fm)
		if err != nil {
			return nil, err
		}
		ss = ss1
	}

	s, horSep := calcListHeader(fm)

	if ctx.SignatureExist || ctx.AppendOnly {
		ss = append(ss, "(signed)")
	}
	ss = append(ss, s)
	ss = append(ss, draw.HorSepLine(horSep))

	i, needSep := 0, false
	for _, f := range fs {

		if len(f.Pages) > 1 {
			continue
		}

		p := " "
		pg := f.Pages[0]
		if pg != i {
			if pg > 1 && needSep {
				ss = append(ss, draw.HorSepLine(horSep))
			}
			i += pg - i
			p = fmt.Sprintf("%d", i)
			needSep = true
		}

		l := " "
		if f.Locked {
			l = "*"
		}

		t := f.Typ.string()

		pageFill := strings.Repeat(" ", fm.pageMax-runewidth.StringWidth(f.pageString()))
		idFill := strings.Repeat(" ", fm.idMax-runewidth.StringWidth(f.ID))
		nameFill := strings.Repeat(" ", fm.nameMax-runewidth.StringWidth(f.Name))
		s := fmt.Sprintf("%s%s %s %-9s %s %s%s %s %s%s ", p, pageFill, l, t, draw.VBar, f.ID, idFill, draw.VBar, f.Name, nameFill)
		if fm.def {
			dvFill := strings.Repeat(" ", fm.defMax-runewidth.StringWidth(f.Dv))
			s += fmt.Sprintf("%s %s%s ", draw.VBar, f.Dv, dvFill)
		}
		if fm.val {
			vFill := strings.Repeat(" ", fm.valMax-runewidth.StringWidth(f.V))
			s += fmt.Sprintf("%s %s%s ", draw.VBar, f.V, vFill)
		}
		if fm.opt {
			s += fmt.Sprintf("%s %s", draw.VBar, f.Opts)
		}

		ss = append(ss, s)
	}

	return ss, nil
}

// FormFields returns all form fields present in ctx.
func FormFields(ctx *model.Context) ([]Field, *FieldMeta, error) {

	xRefTable := ctx.XRefTable

	fields, err := fields(xRefTable)
	if err != nil {
		return nil, nil, err
	}

	fm := &FieldMeta{pageMax: 2, idMax: 3, nameMax: 4, defMax: 7, valMax: 5}

	fs, err := collectFields(xRefTable, fields, fm)
	if err != nil {
		return nil, nil, err
	}

	return fs, fm, nil
}

// ListFormFields returns a list of all form fields present in ctx.
func ListFormFields(ctx *model.Context) ([]string, error) {

	// TODO Align output for Bangla, Hindi, Marathi.

	fs, fm, err := FormFields(ctx)
	if err != nil {
		return nil, err
	}

	return renderFields(ctx, fs, fm)
}

func annotIndRefs(xRefTable *model.XRefTable, fields types.Array) ([]types.IndirectRef, error) {
	var indRefs []types.IndirectRef
	for _, v := range fields {
		indRef := v.(types.IndirectRef)
		d, err := xRefTable.DereferenceDict(indRef)
		if err != nil {
			return nil, err
		}
		o, ok := d.Find("Kids")
		if !ok {
			indRefs = append(indRefs, indRef)
			continue
		}
		kids, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return nil, err
		}
		if _, ok := d.Find("FT"); ok {
			indRefs = append(indRefs, indRef)
			continue
		}
		// Non terminal field
		kidIndRefs, err := annotIndRefs(xRefTable, kids)
		if err != nil {
			return nil, err
		}
		indRefs = append(indRefs, kidIndRefs...)
	}
	return indRefs, nil
}

func annotIndRefSameLevel(xRefTable *model.XRefTable, fields types.Array, fieldIDOrName string) (*types.IndirectRef, error) {
	for _, v := range fields {
		indRef := v.(types.IndirectRef)
		d, err := xRefTable.DereferenceDict(indRef)
		if err != nil {
			return nil, err
		}
		_, hasKids := d.Find("Kids")
		_, hasFT := d.Find("FT")
		if !hasKids || hasFT {
			if indRef.ObjectNumber.String() == fieldIDOrName {
				return &indRef, nil
			}
			id, err := d.StringOrHexLiteralEntry("T")
			if err != nil {
				return nil, err
			}
			if id != nil && *id == fieldIDOrName {
				return &indRef, nil
			}
		}
	}
	return nil, nil
}

func annotIndRefForField(xRefTable *model.XRefTable, fields types.Array, fieldIDOrName string) (*types.IndirectRef, error) {
	if strings.IndexByte(fieldIDOrName, '.') < 0 {
		// Must be on this level
		return annotIndRefSameLevel(xRefTable, fields, fieldIDOrName)
	}

	// Must be below
	ss := strings.Split(fieldIDOrName, ".")
	partialName := ss[0]
	for _, v := range fields {
		indRef := v.(types.IndirectRef)
		d, err := xRefTable.DereferenceDict(indRef)
		if err != nil {
			return nil, err
		}
		o, hasKids := d.Find("Kids")
		_, hasFT := d.Find("FT")
		if !hasKids || hasFT {
			continue
		}
		kids, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return nil, err
		}
		if indRef.ObjectNumber.String() == partialName {
			return annotIndRefForField(xRefTable, kids, fieldIDOrName[len(partialName)+1:])
		}
		id, err := d.StringOrHexLiteralEntry("T")
		if err != nil {
			return nil, err
		}
		if id != nil {
			if *id == partialName {
				return annotIndRefForField(xRefTable, kids, fieldIDOrName[len(partialName)+1:])
			}
		}
	}

	return nil, nil
}

func annotIndRefsForFields(xRefTable *model.XRefTable, f []string, fields types.Array) ([]types.IndirectRef, error) {
	if len(f) == 0 {
		return annotIndRefs(xRefTable, fields)
	}
	var indRefs []types.IndirectRef
	for _, idOrName := range f {
		indRef, err := annotIndRefForField(xRefTable, fields, idOrName)
		if err != nil {
			return nil, err
		}
		if indRef != nil {
			indRefs = append(indRefs, *indRef)
			continue
		}
		if log.CLIEnabled() {
			log.CLI.Printf("unable to resolve field id/name: %s\n", idOrName)
		}
	}
	return indRefs, nil
}

func removeIndRefByIndex(indRefs []types.IndirectRef, i int) []types.IndirectRef {
	l := len(indRefs)
	lastIndex := l - 1
	if i != lastIndex {
		indRefs[i] = indRefs[lastIndex]
	}
	return indRefs[:lastIndex]
}

func removeFromFields(xRefTable *model.XRefTable, indRefs *[]types.IndirectRef, fields *types.Array) error {
	f := types.Array{}
	for _, v := range *fields {
		indRef1 := v.(types.IndirectRef)
		if len(*indRefs) == 0 {
			f = append(f, indRef1)
			continue
		}
		d, err := xRefTable.DereferenceDict(indRef1)
		if err != nil {
			return err
		}
		o, hasKids := d.Find("Kids")
		_, hasFT := d.Find("FT")
		if !hasKids || hasFT {
			// terminal field
			match := false
			for j, indRef2 := range *indRefs {
				if indRef1 == indRef2 {
					*indRefs = removeIndRefByIndex(*indRefs, j)
					match = true
					break
				}
			}
			if !match {
				f = append(f, indRef1)
			}
			continue
		}
		// non terminal fields
		kids, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return err
		}
		if err := removeFromFields(xRefTable, indRefs, &kids); err != nil {
			return err
		}
		if len(kids) > 0 {
			d["Kids"] = kids
			f = append(f, indRef1)
		}
	}
	*fields = f
	return nil
}

func deletePageAnnots(xRefTable *model.XRefTable, m map[types.IndirectRef]bool, ok *bool) error {
	for i := 1; i <= xRefTable.PageCount && len(m) > 0; i++ {

		d, _, _, err := xRefTable.PageDict(i, false)
		if err != nil {
			return err
		}

		o, found := d.Find("Annots")
		if !found {
			continue
		}

		arr, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return err
		}

		// Delete page annotations for removed form fields.

		for indRef1 := range m {
			if len(arr) == 0 {
				break
			}
			for j, v := range arr {
				indRef2 := v.(types.IndirectRef)
				if indRef1 == indRef2 {
					arr = append(arr[:j], arr[j+1:]...)
					delete(m, indRef1)
					if err := xRefTable.DeleteObject(indRef1); err != nil {
						return err
					}
					*ok = true
					break
				}
			}
		}

		if len(arr) == 0 {
			d.Delete("Annots")
			continue
		}
		d.Update("Annots", arr)
	}

	return nil
}

// RemoveFormFields deletes all form fields with given ID or name from the form represented by xRefTable.
func RemoveFormFields(ctx *model.Context, fieldIDsOrNames []string) (bool, error) {

	xRefTable := ctx.XRefTable

	fields, err := fields(xRefTable)
	if err != nil {
		return false, err
	}

	indRefs, err := annotIndRefsForFields(xRefTable, fieldIDsOrNames, fields)
	if err != nil {
		return false, err
	}

	indRefsClone := make([]types.IndirectRef, len(indRefs))
	copy(indRefsClone, indRefs)

	// Remove fields from AcroDict.
	if err := removeFromFields(xRefTable, &indRefsClone, &fields); err != nil {
		return false, err
	}

	if len(indRefsClone) > 0 {
		return false, errors.New("pdfcpu: Some form fields could not be removed")
	}

	if len(fields) == 0 {
		ctx.RootDict.Delete("AcroForm")
	} else {
		xRefTable.Form["Fields"] = fields
	}

	var ok bool

	m := map[types.IndirectRef]bool{}
	for _, indRef := range indRefs {
		d, err := xRefTable.DereferenceDict(indRef)
		if err != nil {
			return false, err
		}
		o, ok := d.Find("Kids")
		if !ok {
			m[indRef] = true
			continue
		}
		kids, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return false, err
		}
		for _, indRef := range kids {
			m[indRef.(types.IndirectRef)] = true
		}
	}

	if err := deletePageAnnots(xRefTable, m, &ok); err != nil {
		return false, err
	}

	if len(m) > 0 {
		return false, errors.New("pdfcpu: Some form fields could not be removed")
	}

	// pdfcpu provides all appearance streams for form fields.
	// Yet for some files and viewers form fields don't get rendered.
	// In these cases you can order the viewer to provide form field appearance streams.
	if ctx.NeedAppearances {
		xRefTable.Form["NeedAppearances"] = types.Boolean(true)
	}

	return ok, nil
}

func resetBtn(xRefTable *model.XRefTable, d types.Dict) error {

	ff := d.IntEntry("Ff")
	if ff != nil && primitives.FieldFlags(*ff)&primitives.FieldPushbutton > 0 {
		return nil
	}

	v := types.Name("Off")
	if s, found := d.Find("DV"); found {
		v = s.(types.Name)
	}

	d["V"] = v
	if _, found := d.Find("AS"); found {
		// Checkbox
		d["AS"] = v
	}

	vraw, err := types.DecodeName(v.String())
	if err != nil {
		return err
	}

	// RadiobuttonGroup

	for _, o := range d.ArrayEntry("Kids") {
		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			return err
		}
		d1 := d.DictEntry("AP")
		if d1 == nil {
			return errors.New("corrupt form field: missing entry AP")
		}
		d2 := d1.DictEntry("N")
		if d2 == nil {
			return errors.New("corrupt AP field: missing entry N")
		}
		for k := range d2 {
			k, err := types.DecodeName(k)
			if err != nil {
				return err
			}
			if k != "Off" {
				d["AS"] = types.Name("Off")
				if k == vraw {
					d["AS"] = v
				}
				break
			}
		}
	}
	return nil
}

func resetComboBoxOrRegularListBox(d types.Dict, opts []string, ff *int) (types.Array, error) {
	ind := types.Array{}
	sl := d.StringLiteralEntry("DV")
	if sl == nil {
		d.Delete("I")
		d.Delete("V")
	} else {
		dv, err := types.StringLiteralToString(*sl)
		if err != nil {
			return nil, err
		}
		// Check if dv is a valid option.
		for i, o := range opts {
			if o == dv {
				ind = append(ind, types.Integer(i))
				break
			}
		}
		if len(ind) > 0 {
			d["I"] = ind
			d["V"] = *sl
		} else {
			d.Delete("I")
			d.Delete("V")
		}
	}
	if primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 {
		d.Delete("AP")
	}
	return ind, nil
}

func resetMultiListBox(xRefTable *model.XRefTable, d types.Dict, opts []string) (types.Array, error) {
	ind := types.Array{}
	defaults, err := parseStringLiteralArray(xRefTable, d, "DV")
	if err != nil {
		return nil, err
	}
	for _, dv := range defaults {
		for i, o := range opts {
			if o == dv {
				ind = append(ind, types.Integer(i))
				break
			}
		}
	}
	if len(defaults) > 0 {
		d["I"] = ind
		d["V"] = d["DV"]
	} else {
		d.Delete("I")
		d.Delete("V")
	}

	return ind, nil
}

func resetCh(ctx *model.Context, d types.Dict, fonts map[string]types.IndirectRef) error {
	ff := d.IntEntry("Ff")
	if ff == nil {
		return errors.New("pdfcpu: corrupt form field: missing entry Ff")
	}

	opts, err := parseOptions(ctx.XRefTable, d)
	if err != nil {
		return err
	}
	if len(opts) == 0 {
		return errors.New("pdfcpu: missing Opts")
	}

	var ind types.Array

	if primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 || primitives.FieldFlags(*ff)&primitives.FieldMultiselect == 0 {
		ind, err = resetComboBoxOrRegularListBox(d, opts, ff)
	} else { // primitives.FieldFlags(*ff)&primitives.FieldMultiselect > 0
		ind, err = resetMultiListBox(ctx.XRefTable, d, opts)
	}

	if err != nil {
		return err
	}

	if primitives.FieldFlags(*ff)&primitives.FieldCombo == 0 {
		if err := primitives.EnsureListBoxAP(ctx, d, opts, ind, fonts); err != nil {
			return err
		}
	}

	return nil
}

func resetTx(ctx *model.Context, d types.Dict, fonts map[string]types.IndirectRef) error {
	var (
		s   string
		err error
	)
	if o, found := d.Find("DV"); found {
		d["V"] = o
		sl, _ := o.(types.StringLiteral)
		s, err = types.StringLiteralToString(sl)
		if err != nil {
			return err
		}
	} else {
		if _, found := d["V"]; !found {
			return nil
		}
		d.Delete("V")
	}

	isDate := true
	if s != "" {
		_, err := primitives.DateFormatForDate(s)
		isDate = err == nil
	}

	if isDate {
		err = primitives.EnsureDateFieldAP(ctx, d, s, fonts)
	} else {
		ff := d.IntEntry("Ff")
		multiLine := ff != nil && uint(primitives.FieldFlags(*ff))&uint(primitives.FieldMultiline) > 0
		err = primitives.EnsureTextFieldAP(ctx, d, s, multiLine, fonts)
	}

	return err
}

func matchField(fi *fieldInfo, fieldIDsOrNames []string) bool {
	return len(fieldIDsOrNames) == 0 ||
		types.MemberOf(fi.id, fieldIDsOrNames) ||
		types.MemberOf(fi.name, fieldIDsOrNames)
}

func resetPageFields(
	ctx *model.Context,
	fieldIDsOrNames []string,
	wAnnots model.Annot,
	fields types.Array,
	fonts map[string]types.IndirectRef,
	ok *bool) error {

	indRefs := map[types.IndirectRef]bool{}

	for _, ir := range *(wAnnots.IndRefs) {

		found, fi, err := isField(ctx.XRefTable, ir, fields)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		if !matchField(fi, fieldIDsOrNames) {
			continue
		}

		if fi.indRef != nil {
			if indRefs[*fi.indRef] {
				continue
			}
			indRefs[*fi.indRef] = true
			ir = *fi.indRef
		}

		d, err := ctx.DereferenceDict(ir)
		if err != nil {
			return err
		}
		if len(d) == 0 {
			continue
		}

		ft := fi.ft
		if ft == nil {
			ft = d.NameEntry("FT")
			if ft == nil {
				return errors.Errorf("pdfcpu: corrupt form field %s: missing entry FT\n%s", fi.id, d)
			}
		}

		switch *ft {
		case "Btn":
			err = resetBtn(ctx.XRefTable, d)

		case "Ch":
			err = resetCh(ctx, d, fonts)

		case "Tx":
			err = resetTx(ctx, d, fonts)
		}

		if err != nil {
			return err
		}

		*ok = true
	}

	return nil
}

// ResetFormFields clears or resets all form fields contained in fieldIDsOrNames to its default.
func ResetFormFields(ctx *model.Context, fieldIDsOrNames []string) (bool, error) {

	xRefTable := ctx.XRefTable

	fields, err := fields(xRefTable)
	if err != nil {
		return false, err
	}

	var ok bool
	fonts := map[string]types.IndirectRef{}

	for i := 1; i <= xRefTable.PageCount; i++ {

		pgAnnots := xRefTable.PageAnnots[i]
		if len(pgAnnots) == 0 {
			continue
		}

		wAnnots, found := pgAnnots[model.AnnWidget]
		if !found {
			continue
		}

		if err := resetPageFields(ctx, fieldIDsOrNames, wAnnots, fields, fonts, &ok); err != nil {
			return false, err
		}
	}

	for fName, indRef := range fonts {

		if len(ctx.UsedGIDs[fName]) == 0 {
			continue
		}

		fDict, err := xRefTable.DereferenceDict(indRef)
		if err != nil {
			return false, err
		}

		fr := model.FontResource{}
		if err := pdffont.IndRefsForUserfontUpdate(xRefTable, fDict, "", &fr); err != nil {
			return false, pdffont.ErrCorruptFontDict
		}

		if err := pdffont.UpdateUserfont(xRefTable, fName, fr); err != nil {
			return false, nil
		}
	}

	// pdfcpu provides all appearance streams for form fields.
	// Yet for some files and viewers form fields don't get rendered.
	// In these cases you can order the viewer to provide form field appearance streams.
	if ctx.NeedAppearances {
		xRefTable.Form["NeedAppearances"] = types.Boolean(true)
	}

	return ok, nil
}

func lockFormField(d types.Dict) {
	ff := d.IntEntry("Ff")
	i := primitives.FieldFlags(0)
	if ff != nil {
		i = primitives.FieldFlags(*ff)
	}
	d["Ff"] = types.Integer(i | primitives.FieldReadOnly)
}

func ensureAP(ctx *model.Context, d types.Dict, fi *fieldInfo, fonts map[string]types.IndirectRef) error {
	ft := fi.ft
	if ft == nil {
		ft = d.NameEntry("FT")
		if ft == nil {
			return errors.Errorf("pdfcpu: corrupt form field %s: missing entry FT\n%s", fi.id, d)
		}
	}

	if *ft == "Ch" {

		ff := d.IntEntry("Ff")
		if ff != nil && primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 {

			v := ""
			if sl := d.StringLiteralEntry("V"); sl != nil {
				s, err := types.StringLiteralToString(*sl)
				if err != nil {
					return err
				}
				v = s
			}

			if err := primitives.EnsureComboBoxAP(ctx, d, v, fonts); err != nil {
				return err
			}

		}
	}

	return nil
}

func lockPageFields(
	ctx *model.Context,
	fieldIDsOrNames []string,
	fields types.Array,
	wAnnots model.Annot,
	fonts map[string]types.IndirectRef,
	ok *bool) error {

	indRefs := map[types.IndirectRef]bool{}

	for _, ir := range *(wAnnots.IndRefs) {

		found, fi, err := isField(ctx.XRefTable, ir, fields)
		if err != nil {
			return err
		}
		if !found {
			continue
		}

		if !matchField(fi, fieldIDsOrNames) {
			continue
		}

		if fi.indRef != nil {
			if indRefs[*fi.indRef] {
				continue
			}
			indRefs[*fi.indRef] = true
			ir = *fi.indRef
		}

		d, err := ctx.DereferenceDict(ir)
		if err != nil {
			return err
		}
		if len(d) == 0 {
			continue
		}

		lockFormField(d)
		*ok = true

		for _, o := range d.ArrayEntry("Kids") {
			d, err := ctx.DereferenceDict(o)
			if err != nil {
				return err
			}
			lockFormField(d)
		}

		if err := ensureAP(ctx, d, fi, fonts); err != nil {
			return err
		}
	}

	return nil
}

// LockFormFields turns all form fields contained in fieldIDsOrNames into read-only.
func LockFormFields(ctx *model.Context, fieldIDsOrNames []string) (bool, error) {

	// Note: Not honoured by Apple Preview for Checkboxes, RadiobuttonGroups and ComboBoxes.

	xRefTable := ctx.XRefTable

	fields, err := fields(xRefTable)
	if err != nil {
		return false, err
	}

	var ok bool
	fonts := map[string]types.IndirectRef{}

	for i := 1; i <= xRefTable.PageCount; i++ {

		pgAnnots := xRefTable.PageAnnots[i]
		if len(pgAnnots) == 0 {
			continue
		}

		wAnnots, found := pgAnnots[model.AnnWidget]
		if !found {
			continue
		}

		if err := lockPageFields(ctx, fieldIDsOrNames, fields, wAnnots, fonts, &ok); err != nil {
			return false, err
		}
	}

	for fName, indRef := range fonts {

		if len(ctx.UsedGIDs[fName]) == 0 {
			continue
		}

		fDict, err := xRefTable.DereferenceDict(indRef)
		if err != nil {
			return false, err
		}

		fr := model.FontResource{}
		if err := pdffont.IndRefsForUserfontUpdate(xRefTable, fDict, "", &fr); err != nil {
			return false, pdffont.ErrCorruptFontDict
		}

		if err := pdffont.UpdateUserfont(xRefTable, fName, fr); err != nil {
			return false, nil
		}
	}

	// pdfcpu provides all appearance streams for form fields.
	// Yet for some files and viewers form fields don't get rendered.
	// In these cases you can order the viewer to provide form field appearance streams.
	if ctx.NeedAppearances {
		xRefTable.Form["NeedAppearances"] = types.Boolean(true)
	}

	return ok, nil
}

func unlockFormField(d types.Dict) {
	ff := d.IntEntry("Ff")
	if ff != nil {
		d["Ff"] = types.Integer(uint(primitives.FieldFlags(*ff)) & ^uint(primitives.FieldReadOnly))
	}
}

func deleteAP(d types.Dict, fi *fieldInfo) error {
	ft := fi.ft
	if ft == nil {
		ft = d.NameEntry("FT")
		if ft == nil {
			return errors.Errorf("pdfcpu: corrupt form field %s: missing entry FT\n%s", fi.id, d)
		}
	}
	if *ft == "Ch" {
		ff := d.IntEntry("Ff")
		if ff != nil && primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 {
			d.Delete("AP")
		}
	}
	return nil
}

func unlockPageFields(
	xRefTable *model.XRefTable,
	fieldIDsOrNames []string,
	fields types.Array,
	wAnnots model.Annot,
	ok *bool) error {

	indRefs := map[types.IndirectRef]bool{}

	for _, ir := range *(wAnnots.IndRefs) {

		found, fi, err := isField(xRefTable, ir, fields)
		if err != nil {
			return err
		}
		if !found {
			continue
		}

		if !matchField(fi, fieldIDsOrNames) {
			continue
		}

		if fi.indRef != nil {
			if indRefs[*fi.indRef] {
				continue
			}
			indRefs[*fi.indRef] = true
			ir = *fi.indRef
		}

		d, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return err
		}
		if len(d) == 0 {
			continue
		}

		unlockFormField(d)

		*ok = true

		for _, o := range d.ArrayEntry("Kids") {
			d, err := xRefTable.DereferenceDict(o)
			if err != nil {
				return err
			}
			unlockFormField(d)
		}

		if err := deleteAP(d, fi); err != nil {
			return err
		}

	}

	return nil
}

// UnlockFields turns all form fields contained in fieldIDsOrNames writeable.
func UnlockFormFields(ctx *model.Context, fieldIDsOrNames []string) (bool, error) {

	xRefTable := ctx.XRefTable

	fields, err := fields(xRefTable)
	if err != nil {
		return false, err
	}

	var ok bool

	for i := 1; i <= xRefTable.PageCount; i++ {

		pgAnnots := xRefTable.PageAnnots[i]
		if len(pgAnnots) == 0 {
			continue
		}

		wAnnots, found := pgAnnots[model.AnnWidget]
		if !found {
			continue
		}

		if err := unlockPageFields(xRefTable, fieldIDsOrNames, fields, wAnnots, &ok); err != nil {
			return false, err
		}
	}

	// pdfcpu provides all appearance streams for form fields.
	// Yet for some files and viewers form fields don't get rendered.
	// In these cases you can order the viewer to provide form field appearance streams.
	if ctx.NeedAppearances {
		xRefTable.Form["NeedAppearances"] = types.Boolean(true)
	}

	return ok, nil
}
