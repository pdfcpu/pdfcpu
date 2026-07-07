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
	"errors"
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

var (
	// ErrMissingJSONWriter signals a missing JSON output writer.
	ErrMissingJSONWriter = errors.New("missing JSON writer")

	errNoFormFieldsAvailable = errors.New("no form fields available")
	errFormFieldsNotRemoved  = errors.New("some form fields could not be removed")
)

// String returns the string value of ft.
func (ft FieldType) String() string {
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

// Field represents a form field for a particular page number.
type Field struct {
	Pages   []int
	Locked  bool
	Typ     FieldType
	ID      string
	Name    string
	AltName string
	Dv      string
	V       string
	Opts    string
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
	altName, def, val, opt                              bool
	pageMax, defMax, valMax, idMax, nameMax, altNameMax int
}

// Fields returns the form field array.
func Fields(xRefTable *model.XRefTable) (types.Array, error) {
	if xRefTable.Form == nil {
		return nil, errors.New("no form available")
	}

	o, ok := xRefTable.Form.Find("Fields")
	if !ok {
		return nil, errNoFormFieldsAvailable
	}

	fields, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return nil, fmt.Errorf("entry Fields: dereference: %w", err)
	}

	if len(fields) == 0 {
		return nil, errNoFormFieldsAvailable
	}

	return fields, nil
}

func indirectRef(o types.Object, context string, index int) (types.IndirectRef, error) {
	indRef, ok := o.(types.IndirectRef)
	if !ok {
		return types.IndirectRef{}, fmt.Errorf("%s entry %d: expected indirect reference, got %T", context, index, o)
	}
	return indRef, nil
}

func dictNameEntry(xRefTable *model.XRefTable, d types.Dict, key string) (types.Name, bool, error) {
	o, found := d.Find(key)
	if !found {
		return "", false, nil
	}
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return "", false, fmt.Errorf("entry %q: dereference: %w", key, err)
	}
	n, ok := o.(types.Name)
	if !ok {
		return "", false, fmt.Errorf("entry %q: expected name, got %T", key, o)
	}
	return n, true, nil
}

func fullyQualifiedFieldNameDepth(xRefTable *model.XRefTable, indRef types.IndirectRef, fields types.Array, id, name *string, depth int, visit *model.FormFieldVisit) (bool, error) {
	if err := xRefTable.CheckRecursionDepth("form field tree", depth); err != nil {
		return false, err
	}
	objNr := indRef.ObjectNumber.Value()
	if err := visit.Enter(objNr); err != nil {
		return false, err
	}
	defer visit.Leave(objNr)

	d, err := xRefTable.DereferenceDict(indRef)
	if err != nil {
		return false, fmt.Errorf("field obj#%d: dereference: %w", objNr, err)
	}
	if len(d) == 0 {
		return false, fmt.Errorf("corrupt field")
	}

	thisID := indRef.ObjectNumber.String()
	thisName := ""
	s, err := d.StringOrHexLiteralEntry("T")
	if err != nil {
		return false, fmt.Errorf("field obj#%d: entry T: %w", objNr, err)
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

	ok, err := fullyQualifiedFieldNameDepth(xRefTable, *pIndRef, fields, id, name, depth+1, visit)
	if err != nil {
		return false, fmt.Errorf("field obj#%d: parent obj#%d: %w", objNr, pIndRef.ObjectNumber.Value(), err)
	}
	if !ok {
		return false, nil
	}

	*id += "." + thisID
	if len(*name) > 0 && len(thisName) > 0 {
		*name += "." + thisName
	}

	return true, nil
}

func fullyQualifiedFieldName(xRefTable *model.XRefTable, indRef types.IndirectRef, fields types.Array, id, name *string) (bool, error) {
	return fullyQualifiedFieldNameDepth(xRefTable, indRef, fields, id, name, 0, model.NewFormFieldVisit())
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
		return false, nil, fmt.Errorf("widget obj#%d: dereference: %w", indRef.ObjectNumber.Value(), err)
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
			return false, nil, fmt.Errorf("widget obj#%d: parent obj#%d: dereference: %w", indRef.ObjectNumber.Value(), pIndRef.ObjectNumber.Value(), err)
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
		sl, ok := o.(types.StringLiteral)
		if ok {
			s, err := types.StringLiteralToString(sl)
			if err != nil {
				return nil, fmt.Errorf("decode string: %w", err)
			}
			s = strings.TrimSpace(s)
			if len(s) > 0 {
				ss = append(ss, s)
			}
			continue
		}
		arr, ok := o.(types.Array)
		if !ok || len(arr) != 2 {
			return nil, errors.New("corrupt choice field")
		}
		sl, ok = arr[1].(types.StringLiteral)
		if !ok {
			return nil, errors.New("corrupt choice field")
		}
		s, err := types.StringLiteralToString(sl)
		if err != nil {
			return nil, fmt.Errorf("decode string: %w", err)
		}
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			ss = append(ss, s)
		}
	}
	return ss, nil
}

func parseOptions(xRefTable *model.XRefTable, d types.Dict, required bool) ([]string, error) {
	o, ok := d.Find("Opt")
	if !ok {
		if required {
			return nil, errors.New("corrupt form field: missing entry \"Opt\"")
		}
		return nil, nil
	}
	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return nil, fmt.Errorf("entry Opt: dereference: %w", err)
	}
	ss, err := extractStringSlice(a)
	if err != nil {
		return nil, fmt.Errorf("entry Opt: %w", err)
	}
	return ss, nil
}

func parseStringLiteralArray(xRefTable *model.XRefTable, d types.Dict, key string) ([]string, error) {
	o, _ := d.Find(key)
	if o == nil {
		return nil, nil
	}
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, fmt.Errorf("entry %s: dereference: %w", key, err)
	}

	switch o := o.(type) {

	case types.StringLiteral:
		s, err := types.StringLiteralToString(o)
		if err != nil {
			return nil, fmt.Errorf("entry %s: decode string: %w", key, err)
		}
		return []string{s}, nil

	case types.Array:
		ss, err := extractStringSlice(o)
		if err != nil {
			return nil, fmt.Errorf("entry %s: %w", key, err)
		}
		return ss, nil
	}

	return nil, nil
}

func collectRadioButtonGroupOptions(xRefTable *model.XRefTable, d types.Dict) ([]string, error) {
	opts, err := parseOptions(xRefTable, d, OPTIONAL)
	if err != nil {
		return nil, err
	}
	if len(opts) > 0 {
		return opts, nil
	}

	for i, o := range d.ArrayEntry("Kids") {
		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			return nil, fmt.Errorf("kid %d: dereference: %w", i+1, err)
		}

		d1, err := locateAPN(xRefTable, d)
		if err != nil {
			return nil, fmt.Errorf("kid %d: appearance: %w", i+1, err)
		}

		for k := range d1 {
			k, err := types.DecodeName(k)
			if err != nil {
				return nil, fmt.Errorf("kid %d: decode appearance state: %w", i+1, err)
			}
			if k != "Off" {
				found := false
				for _, opt := range opts {
					if opt == k {
						found = true
						break
					}
				}
				if !found {
					opts = append(opts, k)
				}
				break
			}
		}
	}

	return opts, nil
}

func collectRadioButtonGroup(xRefTable *model.XRefTable, d types.Dict, f *Field, fm *FieldMeta) error {
	f.Typ = FTRadioButtonGroup

	opts, err := collectRadioButtonGroupOptions(xRefTable, d)
	if err != nil {
		return err
	}

	f.Opts = strings.Join(opts, ",")
	if len(f.Opts) > 0 {
		fm.opt = true
	}

	if s := d.NameEntry("V"); s != nil {
		v, err := types.DecodeName(*s)
		if err != nil {
			return fmt.Errorf("entry V: decode name: %w", err)
		}
		if v != "Off" {
			if len(opts) > 0 {
				j, err := strconv.Atoi(v)
				if err == nil {
					for i, o := range opts {
						if i == j {
							v = o
							break
						}
					}
				}
			}
			if w := runewidth.StringWidth(v); w > fm.valMax {
				fm.valMax = w
			}
			fm.val = true
			f.V = v
		}
	}

	return nil
}

func collectBtn(xRefTable *model.XRefTable, d types.Dict, f *Field, fm *FieldMeta) error {
	ff := d.IntEntry("Ff")
	if ff != nil && primitives.FieldFlags(*ff)&primitives.FieldPushbutton > 0 {
		return nil
	}

	v := types.Name("Off")
	if n, found, err := dictNameEntry(xRefTable, d, "DV"); err != nil {
		return err
	} else if found {
		v = n
	}
	dv, err := types.DecodeName(v.String())
	if err != nil {
		return fmt.Errorf("entry DV: decode name: %w", err)
	}

	if dv != "Off" {
		if w := runewidth.StringWidth(dv); w > fm.defMax {
			fm.defMax = w
		}
		fm.def = true
		f.Dv = dv
	}

	if len(d.ArrayEntry("Kids")) > 1 {
		return collectRadioButtonGroup(xRefTable, d, f, fm)
	}

	f.Typ = FTCheckBox
	if n, found, err := dictNameEntry(xRefTable, d, "V"); err != nil {
		return err
	} else if found {
		if len(n) > 0 && n != "Off" {
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

func collectComboBox(d types.Dict, f *Field, fm *FieldMeta) error {
	f.Typ = FTComboBox
	if sl := d.StringLiteralEntry("V"); sl != nil {
		v, err := types.StringLiteralToString(*sl)
		if err != nil {
			return fmt.Errorf("entry V: decode string: %w", err)
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
			return fmt.Errorf("entry DV: decode string: %w", err)
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
				return fmt.Errorf("entry V: decode string: %w", err)
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
				return fmt.Errorf("entry DV: decode string: %w", err)
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

	opts, err := parseOptions(xRefTable, d, OPTIONAL)
	if err != nil {
		return err
	}

	f.Opts = strings.Join(opts, ",")
	if len(f.Opts) > 0 {
		fm.opt = true
	}

	if ff != nil && primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 {
		return collectComboBox(d, f, fm)
	}

	multi := ff != nil && (primitives.FieldFlags(*ff)&primitives.FieldMultiselect > 0)

	return collectListBox(xRefTable, multi, d, f, fm)
}

func inheritedV(xRefTable *model.XRefTable, d types.Dict) (string, error) {
	if o, found := d.Find("V"); found {
		o, err := xRefTable.Dereference(o)
		if err != nil {
			return "", fmt.Errorf("entry V: dereference: %w", err)
		}
		s1, err := types.StringOrHexLiteral(o)
		if err != nil {
			return "", fmt.Errorf("entry V: decode string: %w", err)
		}
		if s1 != nil {
			return *s1, nil
		}
	}
	indRef := d.IndirectRefEntry("Parent")
	if indRef == nil {
		return "", nil
	}
	d, err := xRefTable.DereferenceDict(*indRef)
	if err != nil {
		return "", fmt.Errorf("entry Parent obj#%d: dereference: %w", indRef.ObjectNumber.Value(), err)
	}
	return inheritedV(xRefTable, d)
}

func getV(xRefTable *model.XRefTable, d types.Dict) (string, error) {
	v, err := inheritedV(xRefTable, d)
	if err != nil {
		return "", err
	}
	return v, nil
}

func inheritedDV(xRefTable *model.XRefTable, d types.Dict) (string, error) {
	if o, found := d.Find("DV"); found {
		o1, err := xRefTable.Dereference(o)
		if err != nil {
			return "", fmt.Errorf("entry DV: dereference: %w", err)
		}
		s1, err := types.StringOrHexLiteral(o1)
		if err != nil {
			return "", fmt.Errorf("entry DV: decode string: %w", err)
		}
		if s1 != nil {
			return *s1, nil
		}
	}
	indRef := d.IndirectRefEntry("Parent")
	if indRef == nil {
		return "", nil
	}
	d, err := xRefTable.DereferenceDict(*indRef)
	if err != nil {
		return "", fmt.Errorf("entry Parent obj#%d: dereference: %w", indRef.ObjectNumber.Value(), err)
	}
	return inheritedDV(xRefTable, d)
}

func getDV(xRefTable *model.XRefTable, d types.Dict) (string, error) {
	dv, err := inheritedDV(xRefTable, d)
	if err != nil {
		return "", err
	}
	return dv, nil
}

func cleanTextForListCmd(s string, maxWidth int) string {
	s = strings.ReplaceAll(s, "\x0A", "\\n")
	s = strings.ReplaceAll(s, "\x0D", "\\n")
	if maxWidth > 0 && len(s) > maxWidth {
		s = s[:maxWidth]
	}
	return s
}

func collectTx(xRefTable *model.XRefTable, d types.Dict, f *Field, fm *FieldMeta, maxWidth int) error {
	v, err := getV(xRefTable, d)
	if err != nil {
		return err
	}
	if v != "" {
		v = cleanTextForListCmd(v, maxWidth)
		if w := runewidth.StringWidth(v); w > fm.valMax {
			fm.valMax = w
		}
		fm.val = true
		f.V = v
	}

	dv, err := getDV(xRefTable, d)
	if err != nil {
		return err
	}
	if dv != "" {
		dv = cleanTextForListCmd(dv, maxWidth)
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

func collectField(xRefTable *model.XRefTable, ft string, d types.Dict, f *Field, fm *FieldMeta, maxWidth int) error {
	var err error

	switch ft {
	case "Btn":
		err = collectBtn(xRefTable, d, f, fm)
	case "Ch":
		err = collectCh(xRefTable, d, f, fm)
	case "Tx":
		err = collectTx(xRefTable, d, f, fm, maxWidth)
	}

	return err
}

func locateField(fs *[]Field, fi *fieldInfo, fm *FieldMeta, pageNr int) bool {
	for j, field := range *fs {
		if field.ID == fi.id && field.Name == fi.name {
			field.Pages = append(field.Pages, pageNr)
			ps := field.pageString()
			if len(ps) > fm.pageMax {
				fm.pageMax = len(ps)
			}
			(*fs)[j] = field
			return true
		}
	}
	return false
}

func collectPageField(
	xRefTable *model.XRefTable,
	d types.Dict,
	pageNr int,
	fi *fieldInfo,
	fm *FieldMeta,
	fs *[]Field,
	maxWidth int) error {
	foundField := locateField(fs, fi, fm, pageNr)

	f := Field{Pages: []int{pageNr}}

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
			return fmt.Errorf("corrupt form field %s: missing entry \"FT\": %s", f.ID, d)
		}
	}

	if o, found := d.Find("TU"); found {
		s1, err := types.StringOrHexLiteral(o)
		if err != nil {
			return err
		}
		s := ""
		if s1 != nil {
			s = *s1
		}
		altName := cleanTextForListCmd(s, maxWidth)
		if w := runewidth.StringWidth(altName); w > fm.altNameMax {
			fm.altNameMax = w
		}
		fm.altName = true
		f.AltName = altName
	}

	if err := collectField(xRefTable, *ft, d, &f, fm, maxWidth); err != nil {
		return err
	}

	if !foundField {
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
	fs *[]Field,
	maxWidth int) error {
	indRefs := map[types.IndirectRef]bool{}

	for _, ir := range *(wAnnots.IndRefs) {

		ok, fi, err := isField(xRefTable, ir, fields)
		if err != nil {
			return fmt.Errorf("resolve field: %w", err)
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
			return fmt.Errorf("field %s obj#%d: dereference: %w", fi.id, ir.ObjectNumber.Value(), err)
		}
		if len(d) == 0 {
			continue
		}

		if err := collectPageField(xRefTable, d, p, fi, fm, fs, maxWidth); err != nil {
			return fmt.Errorf("field %s: %w", fi.id, err)
		}
	}

	return nil
}

func collectFields(xRefTable *model.XRefTable, fields types.Array, fm *FieldMeta, maxWidth int) ([]Field, error) {
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

		if err := collectPageFields(xRefTable, wAnnots, fields, p, fm, &fs, maxWidth); err != nil {
			return nil, fmt.Errorf("page %d: %w", p, err)
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

	if fm.altName {
		s += draw.VBar + " AltName "
		if fm.altNameMax > 7 {
			s += strings.Repeat(" ", fm.altNameMax-7)
			horSep = append(horSep, 9+fm.altNameMax-7)
		} else {
			horSep = append(horSep, 9)
		}
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

func renderMultiPageFields(m map[string][]Field, fm *FieldMeta) ([]string, error) {
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

			t := f.Typ.String()

			pageFill := strings.Repeat(" ", fm.pageMax-runewidth.StringWidth(f.pageString()))
			idFill := strings.Repeat(" ", fm.idMax-runewidth.StringWidth(f.ID))
			nameFill := strings.Repeat(" ", fm.nameMax-runewidth.StringWidth(f.Name))
			s := fmt.Sprintf("%s%s %s %-9s %s %s%s %s %s%s ", p, pageFill, l, t, draw.VBar, f.ID, idFill, draw.VBar, f.Name, nameFill)
			p = strings.Repeat(" ", len(p))
			if fm.altName {
				altNameFill := strings.Repeat(" ", fm.altNameMax-runewidth.StringWidth(f.AltName))
				s += fmt.Sprintf("%s %s%s ", draw.VBar, f.AltName, altNameFill)
			}
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
		ss1, err := renderMultiPageFields(m, fm)
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

		t := f.Typ.String()

		pageFill := strings.Repeat(" ", fm.pageMax-runewidth.StringWidth(f.pageString()))
		idFill := strings.Repeat(" ", fm.idMax-runewidth.StringWidth(f.ID))
		nameFill := strings.Repeat(" ", fm.nameMax-runewidth.StringWidth(f.Name))
		s := fmt.Sprintf("%s%s %s %-9s %s %s%s %s %s%s ", p, pageFill, l, t, draw.VBar, f.ID, idFill, draw.VBar, f.Name, nameFill)
		if fm.altName {
			altNameFill := strings.Repeat(" ", fm.altNameMax-runewidth.StringWidth(f.AltName))
			s += fmt.Sprintf("%s %s%s ", draw.VBar, f.AltName, altNameFill)
		}
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
// maxWidth > 0 limits content for printing.
func FormFields(ctx *model.Context) ([]Field, *FieldMeta, error) {
	xRefTable := ctx.XRefTable

	fields, err := Fields(xRefTable)
	if err != nil {
		return nil, nil, fmt.Errorf("AcroForm Fields: %w", err)
	}

	fm := &FieldMeta{pageMax: 2, idMax: 3, nameMax: 4, altNameMax: 7, defMax: 7, valMax: 5}

	fs, err := collectFields(xRefTable, fields, fm, ctx.Conf.FormFieldListMaxColWidth)
	if err != nil {
		return nil, nil, fmt.Errorf("field tree: %w", err)
	}

	return fs, fm, nil
}

// ListFormFields returns a list of all form fields present in ctx.
func ListFormFields(ctx *model.Context) ([]string, error) {
	// TODO Align output for Bangla, Hindi, Marathi.

	fs, fm, err := FormFields(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect fields: %w", err)
	}

	fields, err := renderFields(ctx, fs, fm)
	if err != nil {
		return nil, fmt.Errorf("render fields: %w", err)
	}
	return fields, nil
}

func annotIndRefsDepth(xRefTable *model.XRefTable, fields types.Array, depth int, visit *model.FormFieldVisit) ([]types.IndirectRef, error) {
	if err := xRefTable.CheckRecursionDepth("form field tree", depth); err != nil {
		return nil, err
	}

	var indRefs []types.IndirectRef
	for i, v := range fields {
		indRef, err := indirectRef(v, "form field tree", i)
		if err != nil {
			return nil, err
		}
		objNr := indRef.ObjectNumber.Value()
		if err := visit.Enter(objNr); err != nil {
			return nil, err
		}
		d, err := xRefTable.DereferenceDict(indRef)
		if err != nil {
			visit.Leave(objNr)
			return nil, err
		}
		o, ok := d.Find("Kids")
		if !ok {
			visit.Leave(objNr)
			indRefs = append(indRefs, indRef)
			continue
		}
		kids, err := xRefTable.DereferenceArray(o)
		if err != nil {
			visit.Leave(objNr)
			return nil, err
		}
		if _, ok := d.Find("FT"); ok {
			visit.Leave(objNr)
			indRefs = append(indRefs, indRef)
			continue
		}
		// Non terminal field
		kidIndRefs, err := annotIndRefsDepth(xRefTable, kids, depth+1, visit)
		visit.Leave(objNr)
		if err != nil {
			return nil, err
		}
		indRefs = append(indRefs, kidIndRefs...)
	}
	return indRefs, nil
}

func annotIndRefs(xRefTable *model.XRefTable, fields types.Array) ([]types.IndirectRef, error) {
	return annotIndRefsDepth(xRefTable, fields, 0, model.NewFormFieldVisit())
}

func annotIndRefSameLevel(xRefTable *model.XRefTable, fields types.Array, fieldIDOrName string) (*types.IndirectRef, error) {
	for i, v := range fields {
		indRef, err := indirectRef(v, "form field tree", i)
		if err != nil {
			return nil, err
		}
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

func annotIndRefForFieldDepth(xRefTable *model.XRefTable, fields types.Array, fieldIDOrName string, depth int, visit *model.FormFieldVisit) (*types.IndirectRef, error) {
	if err := xRefTable.CheckRecursionDepth("form field tree", depth); err != nil {
		return nil, err
	}

	if strings.IndexByte(fieldIDOrName, '.') < 0 {
		// Must be on this level
		return annotIndRefSameLevel(xRefTable, fields, fieldIDOrName)
	}

	// Must be below
	ss := strings.Split(fieldIDOrName, ".")
	partialName := ss[0]
	for i, v := range fields {
		indRef, err := indirectRef(v, "form field tree", i)
		if err != nil {
			return nil, err
		}
		objNr := indRef.ObjectNumber.Value()
		if err := visit.Enter(objNr); err != nil {
			return nil, err
		}
		d, err := xRefTable.DereferenceDict(indRef)
		if err != nil {
			visit.Leave(objNr)
			return nil, err
		}
		o, hasKids := d.Find("Kids")
		_, hasFT := d.Find("FT")
		if !hasKids || hasFT {
			visit.Leave(objNr)
			continue
		}
		kids, err := xRefTable.DereferenceArray(o)
		if err != nil {
			visit.Leave(objNr)
			return nil, err
		}
		if indRef.ObjectNumber.String() == partialName {
			ir, err := annotIndRefForFieldDepth(xRefTable, kids, fieldIDOrName[len(partialName)+1:], depth+1, visit)
			visit.Leave(objNr)
			return ir, err
		}
		id, err := d.StringOrHexLiteralEntry("T")
		if err != nil {
			visit.Leave(objNr)
			return nil, err
		}
		if id != nil {
			if *id == partialName {
				ir, err := annotIndRefForFieldDepth(xRefTable, kids, fieldIDOrName[len(partialName)+1:], depth+1, visit)
				visit.Leave(objNr)
				return ir, err
			}
		}
		visit.Leave(objNr)
	}

	return nil, nil
}

func annotIndRefForField(xRefTable *model.XRefTable, fields types.Array, fieldIDOrName string) (*types.IndirectRef, error) {
	return annotIndRefForFieldDepth(xRefTable, fields, fieldIDOrName, 0, model.NewFormFieldVisit())
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

func removeFormFieldsDepth(xRefTable *model.XRefTable, indRefs *[]types.IndirectRef, fields *types.Array, depth int, visit *model.FormFieldVisit) error {
	if err := xRefTable.CheckRecursionDepth("form field tree", depth); err != nil {
		return err
	}

	f := types.Array{}
	for i, v := range *fields {
		indRef1, err := indirectRef(v, "form field tree", i)
		if err != nil {
			return err
		}
		objNr := indRef1.ObjectNumber.Value()
		if err := visit.Enter(objNr); err != nil {
			return err
		}
		if len(*indRefs) == 0 {
			visit.Leave(objNr)
			f = append(f, indRef1)
			continue
		}
		d, err := xRefTable.DereferenceDict(indRef1)
		if err != nil {
			visit.Leave(objNr)
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
			visit.Leave(objNr)
			continue
		}
		// non terminal fields
		kids, err := xRefTable.DereferenceArray(o)
		if err != nil {
			visit.Leave(objNr)
			return err
		}
		if err := removeFormFieldsDepth(xRefTable, indRefs, &kids, depth+1, visit); err != nil {
			visit.Leave(objNr)
			return err
		}
		if len(kids) > 0 {
			d["Kids"] = kids
			f = append(f, indRef1)
		}
		visit.Leave(objNr)
	}
	*fields = f
	return nil
}

func removeFormFields(xRefTable *model.XRefTable, indRefs *[]types.IndirectRef, fields *types.Array) error {
	return removeFormFieldsDepth(xRefTable, indRefs, fields, 0, model.NewFormFieldVisit())
}

func deletePageAnnots(xRefTable *model.XRefTable, m map[types.IndirectRef]bool, ok *bool) error {
	for i := 1; i <= xRefTable.PageCount && len(m) > 0; i++ {

		d, _, _, err := xRefTable.PageDict(i, false)
		if err != nil {
			return fmt.Errorf("page %d: page dictionary: %w", i, err)
		}

		o, found := d.Find("Annots")
		if !found {
			continue
		}

		arr, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return fmt.Errorf("page %d: Annots: %w", i, err)
		}

		// Delete page annotations for removed form fields.

		for indRef1 := range m {
			if len(arr) == 0 {
				break
			}
			for j, v := range arr {
				indRef2, err := indirectRef(v, fmt.Sprintf("page %d Annots", i), j)
				if err != nil {
					return err
				}
				if indRef1 == indRef2 {
					arr = append(arr[:j], arr[j+1:]...)
					delete(m, indRef1)
					if err := xRefTable.DeleteObject(indRef1); err != nil {
						return fmt.Errorf("page %d: delete field annotation obj#%d: %w", i, indRef1.ObjectNumber.Value(), err)
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

	fields, err := Fields(xRefTable)
	if err != nil {
		return false, fmt.Errorf("AcroForm Fields: %w", err)
	}

	indRefs, err := annotIndRefsForFields(xRefTable, fieldIDsOrNames, fields)
	if err != nil {
		return false, fmt.Errorf("resolve selected fields: %w", err)
	}

	indRefsClone := make([]types.IndirectRef, len(indRefs))
	copy(indRefsClone, indRefs)

	// Remove fields from AcroDict.
	if err := removeFormFields(xRefTable, &indRefsClone, &fields); err != nil {
		return false, fmt.Errorf("field tree: remove fields: %w", err)
	}

	if len(indRefsClone) > 0 {
		return false, errFormFieldsNotRemoved
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
			return false, fmt.Errorf("field obj#%d: dereference: %w", indRef.ObjectNumber.Value(), err)
		}
		o, ok := d.Find("Kids")
		if !ok {
			m[indRef] = true
			continue
		}
		kids, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return false, fmt.Errorf("field obj#%d: Kids: %w", indRef.ObjectNumber.Value(), err)
		}
		for i, o := range kids {
			kidIndRef, err := indirectRef(o, fmt.Sprintf("field obj#%d Kids", indRef.ObjectNumber.Value()), i)
			if err != nil {
				return false, err
			}
			m[kidIndRef] = true
		}
	}

	if err := deletePageAnnots(xRefTable, m, &ok); err != nil {
		return false, fmt.Errorf("page annotations: remove fields: %w", err)
	}

	if len(m) > 0 {
		return false, errFormFieldsNotRemoved
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
	if n, found, err := dictNameEntry(xRefTable, d, "DV"); err != nil {
		return err
	} else if found {
		v = n
	}

	d["V"] = v
	if _, found := d.Find("AS"); found {
		// Checkbox
		d["AS"] = v
	}

	vraw, err := types.DecodeName(v.String())
	if err != nil {
		return fmt.Errorf("entry DV: decode name: %w", err)
	}

	// RadiobuttonGroup

	for i, o := range d.ArrayEntry("Kids") {
		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			return fmt.Errorf("kid %d: dereference: %w", i+1, err)
		}

		d1, err := locateAPN(xRefTable, d)
		if err != nil {
			return fmt.Errorf("kid %d: appearance: %w", i+1, err)
		}

		for k := range d1 {
			k, err := types.DecodeName(k)
			if err != nil {
				return fmt.Errorf("kid %d: decode appearance state: %w", i+1, err)
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
			return nil, fmt.Errorf("entry DV: decode string: %w", err)
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

	opts, err := parseOptions(ctx.XRefTable, d, OPTIONAL)
	if err != nil {
		return err
	}

	var ind types.Array

	if ff != nil && (primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 || primitives.FieldFlags(*ff)&primitives.FieldMultiselect == 0) {
		ind, err = resetComboBoxOrRegularListBox(d, opts, ff)
	} else {
		ind, err = resetMultiListBox(ctx.XRefTable, d, opts)
	}

	if err != nil {
		return err
	}

	da := d.StringEntry("DA")

	if ff != nil && primitives.FieldFlags(*ff)&primitives.FieldCombo == 0 {
		if err := primitives.EnsureListBoxAP(ctx, d, opts, ind, da, fonts); err != nil {
			return fmt.Errorf("appearance: %w", err)
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
		o1, err := ctx.Dereference(o)
		if err != nil {
			return fmt.Errorf("entry DV: dereference: %w", err)
		}
		d["V"] = o1
		sl, _ := o1.(types.StringLiteral)
		s, err = types.StringLiteralToString(sl)
		if err != nil {
			return fmt.Errorf("entry DV: decode string: %w", err)
		}
	} else {
		if _, found := d["V"]; !found {
			return nil
		}
		d.Delete("V")
	}

	isDate := false
	if s != "" {
		_, err := primitives.DateFormatForDate(s)
		isDate = err == nil
	}

	ff := d.IntEntry("Ff")
	multiLine := ff != nil && uint(primitives.FieldFlags(*ff))&uint(primitives.FieldMultiline) > 0
	comb := ff != nil && uint(primitives.FieldFlags(*ff))&uint(primitives.FieldComb) > 0

	da := d.StringEntry("DA")

	kids := d.ArrayEntry("Kids")
	if len(kids) > 0 {

		for i, o := range kids {
			d, err := ctx.DereferenceDict(o)
			if err != nil {
				return fmt.Errorf("kid %d: dereference: %w", i+1, err)
			}

			if isDate {
				err = primitives.EnsureDateFieldAP(ctx, d, s, da, fonts)
			} else {
				err = primitives.EnsureTextFieldAP(ctx, d, s, multiLine, comb, 0, da, fonts)
			}

			if err != nil {
				return fmt.Errorf("kid %d: appearance: %w", i+1, err)
			}
		}

		return nil
	}

	if isDate {
		err = primitives.EnsureDateFieldAP(ctx, d, s, da, fonts)
	} else {
		err = primitives.EnsureTextFieldAP(ctx, d, s, multiLine, comb, 0, da, fonts)
	}
	if err != nil {
		return fmt.Errorf("appearance: %w", err)
	}
	return nil
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
			return fmt.Errorf("resolve field: %w", err)
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
			return fmt.Errorf("field %s obj#%d: dereference: %w", fi.id, ir.ObjectNumber.Value(), err)
		}
		if len(d) == 0 {
			continue
		}

		ft := fi.ft
		if ft == nil {
			ft = d.NameEntry("FT")
			if ft == nil {
				return fmt.Errorf("corrupt form field %s: missing entry \"FT\": %s", fi.id, d)
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
			return fmt.Errorf("field %s: %w", fi.id, err)
		}

		*ok = true
	}

	return nil
}

// ResetFormFields clears or resets all form fields contained in fieldIDsOrNames to its default.
func ResetFormFields(ctx *model.Context, fieldIDsOrNames []string) (bool, error) {
	xRefTable := ctx.XRefTable

	fields, err := Fields(xRefTable)
	if err != nil {
		return false, fmt.Errorf("AcroForm Fields: %w", err)
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
			return false, fmt.Errorf("page %d: reset fields: %w", i, err)
		}
	}

	if err := pdffont.UpdateUserfonts(ctx.XRefTable, fonts); err != nil {
		return false, fmt.Errorf("form fonts: update: %w", err)
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
			return fmt.Errorf("corrupt form field %s: missing entry \"FT\": %s", fi.id, d)
		}
	}

	da := d.StringEntry("DA")

	if *ft == "Ch" {

		ff := d.IntEntry("Ff")
		if ff != nil && primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 {

			v := ""
			if sl := d.StringLiteralEntry("V"); sl != nil {
				s, err := types.StringLiteralToString(*sl)
				if err != nil {
					return fmt.Errorf("entry V: decode string: %w", err)
				}
				v = s
			}

			if err := primitives.EnsureComboBoxAP(ctx, d, v, da, fonts); err != nil {
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
			return fmt.Errorf("resolve field: %w", err)
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
			return fmt.Errorf("field %s obj#%d: dereference: %w", fi.id, ir.ObjectNumber.Value(), err)
		}
		if len(d) == 0 {
			continue
		}

		lockFormField(d)
		*ok = true

		for i, o := range d.ArrayEntry("Kids") {
			d, err := ctx.DereferenceDict(o)
			if err != nil {
				return fmt.Errorf("field %s: kid %d: dereference: %w", fi.id, i+1, err)
			}
			lockFormField(d)
		}

		if err := ensureAP(ctx, d, fi, fonts); err != nil {
			return fmt.Errorf("field %s: appearance: %w", fi.id, err)
		}
	}

	return nil
}

// LockFormFields turns all form fields contained in fieldIDsOrNames into read-only.
func LockFormFields(ctx *model.Context, fieldIDsOrNames []string) (bool, error) {
	// Note: Not honoured by Apple Preview for Checkboxes, RadiobuttonGroups and ComboBoxes.

	xRefTable := ctx.XRefTable

	fields, err := Fields(xRefTable)
	if err != nil {
		return false, fmt.Errorf("AcroForm Fields: %w", err)
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
			return false, fmt.Errorf("page %d: lock fields: %w", i, err)
		}
	}

	if err := pdffont.UpdateUserfonts(ctx.XRefTable, fonts); err != nil {
		return false, fmt.Errorf("form fonts: update: %w", err)
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
			return fmt.Errorf("corrupt form field %s: missing entry \"FT\": %s", fi.id, d)
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
			return fmt.Errorf("resolve field: %w", err)
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
			return fmt.Errorf("field %s obj#%d: dereference: %w", fi.id, ir.ObjectNumber.Value(), err)
		}
		if len(d) == 0 {
			continue
		}

		unlockFormField(d)

		*ok = true

		for i, o := range d.ArrayEntry("Kids") {
			d, err := xRefTable.DereferenceDict(o)
			if err != nil {
				return fmt.Errorf("field %s: kid %d: dereference: %w", fi.id, i+1, err)
			}
			unlockFormField(d)
		}

		if err := deleteAP(d, fi); err != nil {
			return fmt.Errorf("field %s: appearance: %w", fi.id, err)
		}

	}

	return nil
}

// UnlockFormFields turns all form fields contained in fieldIDsOrNames writable.
func UnlockFormFields(ctx *model.Context, fieldIDsOrNames []string) (bool, error) {
	xRefTable := ctx.XRefTable

	fields, err := Fields(xRefTable)
	if err != nil {
		return false, fmt.Errorf("AcroForm Fields: %w", err)
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
			return false, fmt.Errorf("page %d: unlock fields: %w", i, err)
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
