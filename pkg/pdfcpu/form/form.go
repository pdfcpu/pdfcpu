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
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/pdfcpu/pdfcpu/pkg/font"
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

// Field represents a form field for s particular page number.
type Field struct {
	page   int
	locked bool
	typ    FieldType
	id     string
	dv     string
	v      string
	opts   string
}

// FontResDict returns form dict's font resource dict.
func FontResDict(xRefTable *model.XRefTable) (types.Dict, error) {

	d := xRefTable.AcroForm
	if len(d) == 0 {
		return nil, nil
	}

	o, found := d.Find("DR")
	if !found {
		return nil, nil
	}

	resDict, err := xRefTable.DereferenceDict(o)
	if err != nil || len(resDict) == 0 {
		return nil, err
	}

	o, found = resDict.Find("Font")
	if !found {
		return nil, nil
	}

	return xRefTable.DereferenceDict(o)
}

func fields(xRefTable *model.XRefTable) (types.Array, error) {

	if xRefTable.AcroForm == nil {
		return nil, errors.New("pdfcpu: no form available")
	}

	o, ok := xRefTable.AcroForm.Find("Fields")
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

func fullyQualifiedFieldName(xRefTable *model.XRefTable, indRef types.IndirectRef, fields types.Array, path *string) (bool, error) {

	d, err := xRefTable.DereferenceDict(indRef)
	if err != nil {
		return false, err
	}
	if len(d) == 0 {
		return false, errors.Errorf("pdfcpu: corrupt field")
	}

	id := indRef.ObjectNumber.String()
	sl := d.StringLiteralEntry("T")
	if sl != nil {
		id, err = types.StringLiteralToString(*sl)
		if err != nil {
			return false, err
		}
	}

	pIndRef := d.IndirectRefEntry("Parent")
	if pIndRef == nil {
		for i := 0; i < len(fields); i++ {
			if ir, ok := fields[i].(types.IndirectRef); ok && ir == indRef {
				*path = id
				return true, nil
			}
		}
		return false, nil
	}

	// non-terminal field

	ok, err := fullyQualifiedFieldName(xRefTable, *pIndRef, fields, path)
	if !ok || err != nil {
		return false, err
	}

	*path += "." + id

	return true, nil
}

func isField(xRefTable *model.XRefTable, ir1 types.IndirectRef, fields types.Array) (bool, *types.IndirectRef, string, error) {

	d, err := xRefTable.DereferenceDict(ir1)
	if err != nil {
		return false, nil, "", err
	}
	if len(d) == 0 {
		return false, nil, "", nil
	}

	var path string

	ir := d.IndirectRefEntry("Parent")
	if ir != nil {
		dp, err := xRefTable.DereferenceDict(*ir)
		if err != nil {
			return false, nil, "", err
		}
		if len(dp) == 0 {
			return false, nil, "", nil
		}
		ff := dp.IntEntry("Ff")
		if ff != nil {
			// rbg
			ok, err := fullyQualifiedFieldName(xRefTable, *ir, fields, &path)
			if !ok || err != nil {
				return false, nil, "", err
			}
			return true, ir, path, nil
		}
	}

	ok, err := fullyQualifiedFieldName(xRefTable, ir1, fields, &path)
	if !ok || err != nil {
		return false, nil, "", err
	}
	return true, nil, path, nil
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

// ListFormFields returns a list of all form fields present in xRefTable.
func ListFormFields(ctx *model.Context) ([]string, error) {

	// TODO Align output for Bangla, Hindi, Marathi.

	xRefTable := ctx.XRefTable

	fields, err := fields(xRefTable)
	if err != nil {
		return nil, err
	}

	nameMax, defMax, valMax := 4, 7, 5
	var def, val, opt bool

	var fs []Field
	pIndRefs := map[types.IndirectRef]bool{}

	for i := 1; i <= xRefTable.PageCount; i++ {
		pgAnnots := xRefTable.PageAnnots[i]
		if len(pgAnnots) == 0 {
			continue
		}
		wAnnots, ok := pgAnnots[model.AnnWidget]
		if !ok {
			continue
		}

		for _, ir := range *(wAnnots.IndRefs) {

			ok, pIndRef, id, err := isField(xRefTable, ir, fields)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}

			if pIndRef != nil {
				if pIndRefs[*pIndRef] {
					continue
				}
				pIndRefs[*pIndRef] = true
				ir = *pIndRef
			}

			d, err := xRefTable.DereferenceDict(ir)
			if err != nil {
				return nil, err
			}
			if len(d) == 0 {
				continue
			}

			f := Field{page: i}
			f.id = id
			if w := runewidth.StringWidth(id); w > nameMax {
				nameMax = w
			}

			var locked bool
			ff := d.IntEntry("Ff")
			if ff != nil {
				locked = uint(primitives.FieldFlags(*ff))&uint(primitives.FieldReadOnly) > 0
			}
			f.locked = locked

			ft := d.NameEntry("FT")
			if ft == nil {
				return nil, errors.Errorf("pdfcpu: corrupt form field %s: missing entry FT\n%s", f.id, d)
			}

			switch *ft {

			case "Btn":
				v := types.Name("Off")
				if s, found := d.Find("DV"); found {
					v = s.(types.Name)
				}
				dv, err := types.DecodeName(v.String())
				if err != nil {
					return nil, err
				}
				if dv != "Off" {
					if w := runewidth.StringWidth(dv); w > defMax {
						defMax, def = w, true
					}
					f.dv = dv
				}

				if len(d.ArrayEntry("Kids")) > 0 {
					f.typ = FTRadioButtonGroup
					if s := d.NameEntry("V"); s != nil {
						v, err := types.DecodeName(*s)
						if err != nil {
							return nil, err
						}
						if v != "Off" {
							if w := runewidth.StringWidth(v); w > valMax {
								valMax, val = w, true
							}
							f.v = v
						}
					}
					var vv []string
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
								vv = append(vv, k)
								break
							}
						}
					}
					f.opts = strings.Join(vv, ",")
					if len(f.opts) > 0 {
						opt = true
					}
				} else {
					f.typ = FTCheckBox
					if o, found := d.Find("V"); found {
						if o.(types.Name) == "Yes" {
							v := "Yes"
							if len(v) > valMax {
								valMax, val = len(v), true
							}
							f.v = v
						}
					}
				}

			case "Ch":
				ff := d.IntEntry("Ff")
				//if ff == nil {
				//	return nil, errors.New("pdfcpu: corrupt form field: missing entry Ff")
				//}
				vv, err := parseOptions(xRefTable, d)
				if err != nil {
					return nil, err
				}
				f.opts = strings.Join(vv, ",")
				if len(f.opts) > 0 {
					opt = true
				}
				if ff != nil && primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 {
					f.typ = FTComboBox
					if sl := d.StringLiteralEntry("V"); sl != nil {
						v, err := types.StringLiteralToString(*sl)
						if err != nil {
							return nil, err
						}
						if w := runewidth.StringWidth(v); w > valMax {
							valMax, val = w, true
						}
						f.v = v
					}
					if sl := d.StringLiteralEntry("DV"); sl != nil {
						dv, err := types.StringLiteralToString(*sl)
						if err != nil {
							return nil, err
						}
						if w := runewidth.StringWidth(dv); w > defMax {
							defMax, def = w, true
						}
						f.dv = dv
					}
				} else {
					f.typ = FTListBox
					multi := ff != nil && (primitives.FieldFlags(*ff)&primitives.FieldMultiselect > 0)
					if !multi {
						if sl := d.StringLiteralEntry("V"); sl != nil {
							v, err := types.StringLiteralToString(*sl)
							if err != nil {
								return nil, err
							}
							if w := runewidth.StringWidth(v); w > valMax {
								valMax, val = w, true
							}
							f.v = v
						}
						if sl := d.StringLiteralEntry("DV"); sl != nil {
							dv, err := types.StringLiteralToString(*sl)
							if err != nil {
								return nil, err
							}
							if w := runewidth.StringWidth(dv); w > defMax {
								defMax, def = w, true
							}
							f.dv = dv
						}
					} else {
						vv, err := parseStringLiteralArray(xRefTable, d, "V")
						if err != nil {
							return nil, err
						}
						v := strings.Join(vv, ",")
						if w := runewidth.StringWidth(v); w > valMax {
							valMax = w
						}
						f.v = v
						vv, err = parseStringLiteralArray(xRefTable, d, "DV")
						if err != nil {
							return nil, err
						}
						dv := strings.Join(vv, ",")
						if w := runewidth.StringWidth(dv); w > defMax {
							defMax, def = w, true
						}
						f.dv = dv
					}
				}

			case "Tx":
				if o, found := d.Find("V"); found {
					sl, _ := o.(types.StringLiteral)
					s, err := types.StringLiteralToString(sl)
					if err != nil {
						return nil, err
					}
					v := s
					if i := strings.Index(s, "\n"); i >= 0 {
						v = s[:i]
						v += "\\n"
					}
					if w := runewidth.StringWidth(v); w > valMax {
						valMax, val = w, true
					}
					f.v = v
				}
				if o, found := d.Find("DV"); found {
					sl, _ := o.(types.StringLiteral)
					s, err := types.StringLiteralToString(sl)
					if err != nil {
						return nil, err
					}
					dv := s
					if i := strings.Index(s, "\n"); i >= 0 {
						dv = dv[:i]
						dv += "\\n"
					}

					if w := runewidth.StringWidth(dv); w > defMax {
						defMax, def = w, true
					}
					f.dv = dv
				}
				df, err := extractDateFormat(xRefTable, d)
				if err != nil {
					return nil, err
				}
				f.typ = FTText
				if df != nil {
					f.typ = FTDate
				}

			}

			fs = append(fs, f)
		}
	}

	var ss []string

	horSep := []int{15}

	s := "Pg L Field     " + draw.VBar + " Name "
	if nameMax > 4 {
		s += strings.Repeat(" ", nameMax-4)
		horSep = append(horSep, 6+nameMax-4)
	} else {
		horSep = append(horSep, 6)
	}
	if def {
		s += draw.VBar + " Default "
		if defMax > 7 {
			s += strings.Repeat(" ", defMax-7)
			horSep = append(horSep, 9+defMax-7)
		} else {
			horSep = append(horSep, 9)
		}
	}
	if val {
		s += draw.VBar + " Value "
		if valMax > 5 {
			s += strings.Repeat(" ", valMax-5)
			horSep = append(horSep, 7+valMax-5)
		} else {
			horSep = append(horSep, 7)
		}
	}
	if opt {
		s += draw.VBar + " Options"
		horSep = append(horSep, 8)
	}

	ss = append(ss, s)
	ss = append(ss, draw.HorSepLine(horSep))

	i, needSep := 0, false
	for _, f := range fs {

		p := "  "
		if f.page != i {
			if f.page > 1 && needSep {
				ss = append(ss, draw.HorSepLine(horSep))
			}
			i += f.page - i
			p = fmt.Sprintf("%2d", i)
			needSep = true
		}

		l := " "
		if f.locked {
			l = "*"
		}
		t := ""
		switch f.typ {
		case FTText:
			t = "Textfield"
		case FTDate:
			t = "Datefield"
		case FTCheckBox:
			t = "CheckBox"
		case FTRadioButtonGroup:
			t = "RadioBGr."
		case FTComboBox:
			t = "ComboBox"
		case FTListBox:
			t = "ListBox"
		}

		idFill := strings.Repeat(" ", nameMax-runewidth.StringWidth(f.id))
		s := fmt.Sprintf("%s %s %-9s %s %s%s ", p, l, t, draw.VBar, f.id, idFill)
		if def {
			dvFill := strings.Repeat(" ", defMax-runewidth.StringWidth(f.dv))
			s += fmt.Sprintf("%s %s%s ", draw.VBar, f.dv, dvFill)
		}
		if val {
			vFill := strings.Repeat(" ", valMax-runewidth.StringWidth(f.v))
			s += fmt.Sprintf("%s %s%s ", draw.VBar, f.v, vFill)
		}
		if opt {
			s += fmt.Sprintf("%s %s", draw.VBar, f.opts)
		}

		ss = append(ss, s)
	}

	return ss, nil
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

func annotIndRefForFieldID(xRefTable *model.XRefTable, fields types.Array, fieldID string) (*types.IndirectRef, error) {
	if strings.IndexByte(fieldID, '.') < 0 {
		// Must be on this level
		for _, v := range fields {
			indRef := v.(types.IndirectRef)
			d, err := xRefTable.DereferenceDict(indRef)
			if err != nil {
				return nil, err
			}
			_, hasKids := d.Find("Kids")
			_, hasFT := d.Find("FT")
			if !hasKids || hasFT {
				sl := d.StringLiteralEntry("T")
				if sl == nil {
					return nil, errors.New("pdfcpu: corrupt form field: missing entry T")
				}
				id, err := types.StringLiteralToString(*sl)
				if err != nil {
					return nil, err
				}
				if id == fieldID {
					return &indRef, nil
				}
			}
		}
		return nil, nil
	}
	// Must be below
	ss := strings.Split(fieldID, ".")
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
		sl := d.StringLiteralEntry("T")
		if sl != nil {
			id, err := types.StringLiteralToString(*sl)
			if err != nil {
				return nil, err
			}
			if partialName == id {
				fieldID = fieldID[len(partialName)+1:]
				return annotIndRefForFieldID(xRefTable, kids, fieldID)
			}
			continue
		}
		if partialName == indRef.ObjectNumber.String() {
			fieldID = fieldID[len(partialName)+1:]
			return annotIndRefForFieldID(xRefTable, kids, fieldID)
		}
	}
	return nil, nil
}

func annotIndRefsForFieldIDs(xRefTable *model.XRefTable, fieldIDs []string, fields types.Array) ([]types.IndirectRef, error) {
	if len(fieldIDs) == 0 {
		return annotIndRefs(xRefTable, fields)
	}
	var indRefs []types.IndirectRef
	for _, id := range fieldIDs {
		indRef, err := annotIndRefForFieldID(xRefTable, fields, id)
		if err != nil {
			return nil, err
		}
		if indRef != nil {
			indRefs = append(indRefs, *indRef)
			continue
		}
		log.CLI.Printf("unable to resolve field name: %s\n", id)
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

// RemoveFormFields deletes all form fields with given ID from the form represented by xRefTable.
func RemoveFormFields(ctx *model.Context, fieldIDs []string) (bool, error) {

	xRefTable := ctx.XRefTable

	fields, err := fields(xRefTable)
	if err != nil {
		return false, err
	}

	indRefs, err := annotIndRefsForFieldIDs(xRefTable, fieldIDs, fields)
	if err != nil {
		return false, err
	}

	indRefsClone := make([]types.IndirectRef, len(indRefs))
	copy(indRefsClone, indRefs)

	if err := removeFromFields(xRefTable, &indRefsClone, &fields); err != nil {
		return false, err
	}

	if len(indRefsClone) > 0 {
		return false, errors.New("pdfcpu: Some form fields could not be removed")
	}

	if len(fields) == 0 {
		ctx.RootDict.Delete("AcroForm")
	} else {
		xRefTable.AcroForm["Fields"] = fields
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

	for i := 1; i <= xRefTable.PageCount && len(m) > 0; i++ {

		// arr, err := xRefTable.AnnotationsForPage(i)

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
					if err := ctx.DeleteObject(indRef1); err != nil {
						return false, err
					}
					ok = true
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

	if len(m) > 0 {
		return false, errors.New("pdfcpu: Some form fields could not be removed")
	}

	return ok, nil
}

func formFontNameAndLangForID(xRefTable *model.XRefTable, indRef types.IndirectRef) (*string, *string, error) {

	objNr := int(indRef.ObjectNumber)
	fontDict, err := xRefTable.DereferenceDict(indRef)
	if err != nil || fontDict == nil {
		return nil, nil, err
	}

	_, fName, err := pdffont.Name(xRefTable, fontDict, objNr)
	if err != nil {
		return nil, nil, err
	}

	var fLang *string
	if font.IsUserFont(fName) {
		fLang, err = pdffont.Lang(xRefTable, fontDict)
		if err != nil {
			return nil, nil, err
		}
	}

	return &fName, fLang, nil
}

func extractFontDetails(xRefTable *model.XRefTable, indRef types.IndirectRef, fonts map[string]types.IndirectRef) (string, string, string, error) {

	sd, _, _ := xRefTable.DereferenceStreamDict(indRef)

	d := sd.DictEntry("Resources")
	if d == nil {
		return "", "", "", errors.New("pdfcpu: missing resource dict")
	}

	d1 := d.DictEntry("Font")
	if d1 == nil {
		return "", "", "", errors.New("pdfcpu: missing font resource dict")
	}

	if len(d1) != 1 {
		return "", "", "", errors.New("pdfcpu: corrupt form resource dict")
	}

	var fontID string
	var ir types.IndirectRef
	for k, v := range d1 {
		fontID = k
		ir = v.(types.IndirectRef)
	}

	fName, fLang, err := formFontNameAndLangForID(xRefTable, ir)
	if err != nil {
		return "", "", "", err
	}

	if fName == nil {
		return "", "", "", errors.Errorf("pdfcpu: Unable to detect fontName for: %s", fontID)
	}

	var lang string
	if fLang != nil {
		lang = *fLang
	}

	if font.IsUserFont(*fName) {
		d, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return "", "", "", err
		}
		if enc := d.NameEntry("Encoding"); *enc == "Identity-H" {
			indRef, ok := fonts[*fName]
			if !ok {
				fonts[*fName] = ir
			} else if indRef != ir {
				return "", "", "", errors.Errorf("pdfcpu: %s: duplicate fontDicts", *fName)
			}
		}
	}

	return fontID, *fName, lang, nil
}

// ResetFormFields clears or resets all form fields contained in fieldIDs to its default.
func ResetFormFields(ctx *model.Context, fieldIDs []string) (bool, error) {

	xRefTable := ctx.XRefTable

	fields, err := fields(xRefTable)
	if err != nil {
		return false, err
	}

	indRefs, err := annotIndRefsForFieldIDs(xRefTable, fieldIDs, fields)
	if err != nil {
		return false, err
	}

	var ok bool

	fonts := map[string]types.IndirectRef{}

	for _, ir := range indRefs {

		d, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return false, err
		}
		if len(d) == 0 {
			continue
		}

		ft := d.NameEntry("FT")
		if ft == nil {
			return false, errors.New("pdfcpu: corrupt form field: missing entry FT")
		}

		switch *ft {
		case "Btn":
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
				return false, err
			}

			// RadiobuttonGroup

			for _, o := range d.ArrayEntry("Kids") {
				d, err := xRefTable.DereferenceDict(o)
				if err != nil {
					return false, err
				}
				d1 := d.DictEntry("AP")
				if d1 == nil {
					return false, errors.New("corrupt form field: missing entry AP")
				}
				d2 := d1.DictEntry("N")
				if d2 == nil {
					return false, errors.New("corrupt AP field: missing entry N")
				}
				for k := range d2 {
					k, err := types.DecodeName(k)
					if err != nil {
						return false, err
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

		case "Ch":
			// AP for listbox, combobox

			ff := d.IntEntry("Ff")
			if ff == nil {
				return false, errors.New("pdfcpu: corrupt form field: missing entry Ff")
			}

			opts, err := parseOptions(xRefTable, d)
			if err != nil {
				return false, err
			}
			if len(opts) == 0 {
				return false, errors.New("pdfcpu: missing Opts")
			}

			ind := types.Array{}

			if primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 || primitives.FieldFlags(*ff)&primitives.FieldMultiselect == 0 {

				// combobox or regular listbox

				sl := d.StringLiteralEntry("DV")
				if sl == nil {
					d.Delete("I")
					d.Delete("V")
				} else {
					dv, err := types.StringLiteralToString(*sl)
					if err != nil {
						return false, err
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

			} else { // primitives.FieldFlags(*ff)&primitives.FieldMultiselect > 0

				// multi listbox:

				defaults, err := parseStringLiteralArray(xRefTable, d, "DV")
				if err != nil {
					return false, err
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
			}

			if primitives.FieldFlags(*ff)&primitives.FieldCombo == 0 {

				lb, err := primitives.NewListBox(xRefTable, d, opts, ind)
				if err != nil {
					return false, err
				}

				apd := d.DictEntry("AP")
				if apd != nil {
					irN := apd.IndirectRefEntry("N")
					if irN != nil {
						id, name, lang, err := extractFontDetails(xRefTable, *irN, fonts)
						if err != nil {
							return false, err
						}
						lb.SetFontID(id)
						lb.Font.Name = name
						lb.Font.Lang = lang
						if err := lb.RefreshN(xRefTable, irN); err != nil {
							return false, err
						}
					}
				}

			}

		case "Tx":

			var s string
			if o, found := d.Find("DV"); found {
				d["V"] = o
				sl, _ := o.(types.StringLiteral)
				s, err = types.StringLiteralToString(sl)
				if err != nil {
					return false, err
				}
			} else {
				if _, found := d["V"]; !found {
					continue
				}
				d.Delete("V")
			}

			isDate := true
			if s != "" {
				_, err := primitives.DateFormatForDate(s)
				isDate = err == nil
			}

			var tf *primitives.TextField

			var df *primitives.DateField
			var err error
			if isDate {
				df, err = primitives.NewDateField(xRefTable, d, s)
			} else {
				ff := d.IntEntry("Ff")
				multiLine := ff != nil && uint(primitives.FieldFlags(*ff))&uint(primitives.FieldMultiline) > 0
				tf, err = primitives.NewTextField(xRefTable, d, s, multiLine)
			}
			if err != nil {
				return false, err
			}

			apd := d.DictEntry("AP")
			if apd != nil {
				irN := apd.IndirectRefEntry("N")
				if irN != nil {
					id, name, lang, err := extractFontDetails(xRefTable, *irN, fonts)
					if err != nil {
						return false, err
					}
					if isDate {
						df.SetFontID(id)
						df.Font.Name = name
						df.Font.Lang = lang
						err = df.RefreshN(xRefTable, irN)
					} else {
						tf.SetFontID(id)
						tf.Font.Name = name
						tf.Font.Lang = lang
						err = tf.RefreshN(xRefTable, irN)
					}
					if err != nil {
						return false, err
					}
				}
			}
		}

		ok = true
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

	return ok, nil
}

func formFontIndRef(xRefTable *model.XRefTable, fontID string) (*types.IndirectRef, error) {
	d, err := FontResDict(xRefTable)
	if err != nil {
		return nil, err
	}

	for k, v := range d {
		if k == fontID {
			indRef, _ := v.(types.IndirectRef)
			return &indRef, nil
		}
	}

	return nil, errors.Errorf("pdfcpu: missing form font %s", fontID)
}

func fontIndRef(fName string, ctx *model.Context, fonts map[string]types.IndirectRef) (*types.IndirectRef, error) {

	indRef, ok := fonts[fName]
	if ok {
		d, err := ctx.DereferenceDict(indRef)
		if err != nil {
			return nil, err
		}
		if enc := d.NameEntry("Encoding"); *enc == "Identity-H" {
			return &indRef, nil
		}
	}

	for objNr, fo := range ctx.Optimize.FontObjects {
		if fo.FontName == fName {
			indRef := types.NewIndirectRef(objNr, 0)
			d, err := ctx.DereferenceDict(*indRef)
			if err != nil {
				return nil, err
			}
			if enc := d.NameEntry("Encoding"); *enc == "Identity-H" {
				fonts[fName] = *indRef
				return indRef, nil
			}
		}
	}

	return nil, nil
}

func extractFormFontDetails(ctx *model.Context, fontID string, fonts map[string]types.IndirectRef) (string, string, string, *types.IndirectRef, error) {

	xRefTable := ctx.XRefTable

	ir, err := formFontIndRef(xRefTable, fontID)
	if err != nil {
		return "", "", "", nil, err
	}

	fName, fLang, err := formFontNameAndLangForID(xRefTable, *ir)
	if err != nil {
		return "", "", "", nil, err
	}

	if fName == nil {
		return "", "", "", nil, errors.Errorf("pdfcpu: Unable to detect fontName for: %s", fontID)
	}

	var lang string
	if fLang != nil {
		lang = *fLang
	}

	if font.IsUserFont(*fName) {
		d, err := xRefTable.DereferenceDict(*ir)
		if err != nil {
			return "", "", "", nil, err
		}
		if enc := d.NameEntry("Encoding"); *enc == "Identity-H" {
			indRef, ok := fonts[*fName]
			if !ok {
				fonts[*fName] = *ir
			} else if indRef != *ir {
				return "", "", "", nil, errors.Errorf("pdfcpu: %s: duplicate fontDicts", *fName)
			}
		} else {
			ir, err = fontIndRef(*fName, ctx, fonts)
			return fontID, *fName, lang, ir, err
		}
	}

	return fontID, *fName, lang, ir, nil
}

func lockFormField(d types.Dict) {
	ff := d.IntEntry("Ff")
	i := primitives.FieldFlags(0)
	if ff != nil {
		i = primitives.FieldFlags(*ff)
	}
	d["Ff"] = types.Integer(i | primitives.FieldReadOnly)
}

// LockFormFields turns all form fields contained in fieldIDs into read-only.
func LockFormFields(ctx *model.Context, fieldIDs []string) (bool, error) {

	// Note: Not honoured by Apple Preview for Checkboxes, RadiobuttonGroups and ComboBoxes.

	xRefTable := ctx.XRefTable

	fields, err := fields(xRefTable)
	if err != nil {
		return false, err
	}

	indRefs, err := annotIndRefsForFieldIDs(xRefTable, fieldIDs, fields)
	if err != nil {
		return false, err
	}

	fonts := map[string]types.IndirectRef{}

	var ok bool

	for _, ir := range indRefs {

		d, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return false, err
		}
		if len(d) == 0 {
			continue
		}

		lockFormField(d)
		ok = true

		for _, o := range d.ArrayEntry("Kids") {
			d, err := xRefTable.DereferenceDict(o)
			if err != nil {
				return false, err
			}
			lockFormField(d)
		}

		ft := d.NameEntry("FT")
		if ft == nil {
			return false, errors.New("pdfcpu: corrupt form field: missing entry FT")
		}
		if *ft == "Ch" {
			ff := d.IntEntry("Ff")
			if ff != nil && primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 {
				cb, err := primitives.NewComboBox(xRefTable, d, nil)
				if err != nil {
					return false, err
				}

				vOld := ""
				if sl := d.StringLiteralEntry("V"); sl != nil {
					s, err := types.StringLiteralToString(*sl)
					if err != nil {
						return false, err
					}
					vOld = s
				}
				cb.Value = vOld

				id, name, lang, ir, err := extractFormFontDetails(ctx, cb.FontID, fonts)
				if err != nil {
					return false, err
				}
				if ir != nil {
					cb.SetFontID(id)
					cb.Font.Name = name
					cb.Font.Lang = lang
					cb.RTL = pdffont.RTL(lang)
					irN, err := primitives.ComboBoxN(xRefTable, cb, *ir)
					if err != nil {
						return false, err
					}
					d["AP"] = types.Dict(map[string]types.Object{"N": *irN})
				}
			}
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

	return ok, nil
}

func unlockFormField(d types.Dict) {
	ff := d.IntEntry("Ff")
	if ff != nil {
		d["Ff"] = types.Integer(uint(primitives.FieldFlags(*ff)) & ^uint(primitives.FieldReadOnly))
	}
}

// UnlockFields turns all form fields contained in fieldIDs writeable.
func UnlockFormFields(ctx *model.Context, fieldIDs []string) (bool, error) {

	xRefTable := ctx.XRefTable

	fields, err := fields(xRefTable)
	if err != nil {
		return false, err
	}

	indRefs, err := annotIndRefsForFieldIDs(xRefTable, fieldIDs, fields)
	if err != nil {
		return false, err
	}

	var ok bool

	for _, ir := range indRefs {

		d, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return false, err
		}
		if len(d) == 0 {
			continue
		}

		unlockFormField(d)
		ok = true

		for _, o := range d.ArrayEntry("Kids") {
			d, err := xRefTable.DereferenceDict(o)
			if err != nil {
				return false, err
			}
			unlockFormField(d)
		}

		ft := d.NameEntry("FT")
		if ft == nil {
			return false, errors.New("pdfcpu: corrupt form field: missing entry FT")
		}
		if *ft == "Ch" {
			ff := d.IntEntry("Ff")
			if ff != nil && primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 {
				d.Delete("AP")
			}
		}
	}

	return ok, nil
}
