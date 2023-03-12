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
	"bytes"
	"sort"
	"strconv"
	"strings"

	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/primitives"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

type DataFormat int

const (
	CSV DataFormat = iota
	JSON
)

func cacheResIDs(ctx *model.Context, pdf *primitives.PDF) error {
	// Iterate over all pages of ctx and prepare a resIds []string for inherited "Font" and "XObject" resources.
	for i := 1; i <= ctx.PageCount; i++ {
		_, _, inhPA, err := ctx.PageDict(i, true)
		if err != nil {
			return err
		}
		if inhPA.Resources["Font"] != nil {
			pdf.FontResIDs[i] = inhPA.Resources["Font"].(types.Dict)
		}
		if inhPA.Resources["XObject"] != nil {
			pdf.XObjectResIDs[i] = inhPA.Resources["XObject"].(types.Dict)
		}
	}
	return nil
}

func addImages(ctx *model.Context, pages map[string]*Page) ([]*model.Page, error) {

	pdf := &primitives.PDF{
		FieldIDs:      types.StringSet{},
		Fields:        types.Array{},
		FormFonts:     map[string]*primitives.FormFont{},
		Pages:         map[string]*primitives.PDFPage{},
		FontResIDs:    map[int]types.Dict{},
		XObjectResIDs: map[int]types.Dict{},
		Conf:          ctx.Configuration,
		XRefTable:     ctx.XRefTable,
		Optimize:      ctx.Optimize,
		CheckBoxAPs:   map[float64]*primitives.AP{},
		RadioBtnAPs:   map[float64]*primitives.AP{},
		OldFieldIDs:   types.StringSet{},
		Debug:         false,
	}

	if err := cacheResIDs(ctx, pdf); err != nil {
		return nil, err
	}

	// What follows is a quirky way of turning a map of pages into a sorted slice of pages
	// including entries for pages that are missing in the map.

	var pageNrs []int

	for pageNr := range pages {
		nr, err := strconv.Atoi(pageNr)
		if err != nil {
			return nil, errors.Errorf("pdfcpu: invalid page number: %s", pageNr)
		}
		pageNrs = append(pageNrs, nr)
	}

	sort.Ints(pageNrs)

	pp := []*Page{}

	maxPageNr := pageNrs[len(pageNrs)-1]
	for i := 1; i <= maxPageNr; i++ {
		pp = append(pp, pages[strconv.Itoa(i)])
	}

	mp := []*model.Page{}
	imageMap := model.ImageMap{}

	for i, page := range pp {

		pageNr := i + 1

		_, _, inhPAttrs, err := ctx.PageDict(pageNr, false)
		if err != nil {
			return nil, err
		}

		p := model.Page{
			MediaBox:  inhPAttrs.MediaBox,
			CropBox:   inhPAttrs.CropBox,
			Fm:        model.FontMap{},
			Im:        model.ImageMap{},
			AnnotTabs: map[int]model.FieldAnnotation{},
			Buf:       new(bytes.Buffer),
		}

		if page == nil {
			if pageNr <= pdf.XRefTable.PageCount {
				mp = append(mp, nil)
				continue
			}
		}

		for _, ib := range page.ImageBoxes {
			if err := ib.RenderForFill(pdf, &p, pageNr, imageMap); err != nil {
				return nil, err
			}
		}

		mp = append(mp, &p)
	}

	return mp, nil
}

// CSVFieldAttributes represent the value(s) and the lock state for a field.
type CSVFieldAttributes struct {
	Values []string
	Lock   bool
}

func parsePageNr(s string, ib *primitives.ImageBox) error {
	_, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	ib.PageNr = s
	return nil
}

func parseWidth(s string, ib *primitives.ImageBox) error {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	ib.Width = f
	return nil
}

func parseHeight(s string, ib *primitives.ImageBox) error {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	ib.Height = f
	return nil
}

func parsePositionAnchor(s string, ib *primitives.ImageBox) error {
	d := strings.Split(s, " ")
	if len(d) < 1 || len(d) > 2 {
		return errors.Errorf("pdfcpu: illegal position string: need 1 or 2 values, %s\n", s)
	}
	if len(d) == 1 {
		_, err := types.ParsePositionAnchor(s)
		if err != nil {
			return err
		}
		ib.Anchor = s
		return nil
	}
	x, err := strconv.ParseFloat(d[0], 64)
	if err != nil {
		return err
	}

	y, err := strconv.ParseFloat(d[1], 64)
	if err != nil {
		return err
	}
	ib.Position = [2]float64{x, y}
	return nil
}

func parsePositionOffset(s string, ib *primitives.ImageBox) error {
	d := strings.Split(s, " ")
	if len(d) != 2 {
		return errors.Errorf("pdfcpu: illegal position offset string: need 2 numeric values, %s\n", s)
	}

	f, err := strconv.ParseFloat(d[0], 64)
	if err != nil {
		return err
	}
	ib.Dx = f

	f, err = strconv.ParseFloat(d[1], 64)
	if err != nil {
		return err
	}
	ib.Dy = f

	return nil
}

func parseImgBackgroundColor(s string, ib *primitives.ImageBox) error {
	ib.BackgroundColor = s
	return nil
}

func parseImgBorder(s string, ib *primitives.ImageBox) error {

	var err error

	b := strings.Split(s, " ")
	if len(b) == 0 || len(b) > 5 {
		return errors.Errorf("pdfcpu: borders: need between 1 and 5 components, %s\n", s)
	}

	ib.Border = &primitives.Border{}
	border := ib.Border

	border.Width, err = strconv.Atoi(b[0])
	if err != nil {
		return err
	}
	if border.Width == 0 {
		return errors.New("pdfcpu: borders: need width > 0")
	}

	if len(b) == 1 {
		return nil
	}

	st := strings.ToLower(b[1])
	if types.MemberOf(st, []string{"bevel", "miter", "round"}) {
		border.Style = st
		if len(b) > 2 {
			border.Color = strings.Join(b[2:], " ")
		}
		return nil
	}

	border.Color = strings.Join(b[1:], " ")

	return nil
}

type imageBoxParamMap map[string]func(string, *primitives.ImageBox) error

var imgParamMap = imageBoxParamMap{
	"bgcolor":  parseImgBackgroundColor,
	"border":   parseImgBorder,
	"offset":   parsePositionOffset,
	"page":     parsePageNr,
	"position": parsePositionAnchor,
	"width":    parseWidth,
	"height":   parseHeight,
}

func (m imageBoxParamMap) processImageBoxArg(paramPrefix, paramValueStr string, ib *primitives.ImageBox) error {

	var param string

	// Completion support
	for k := range m {
		if !strings.HasPrefix(k, paramPrefix) {
			continue
		}
		if len(param) > 0 {
			return errors.Errorf("pdfcpu: ambiguous parameter prefix \"%s\"", paramPrefix)
		}
		param = k
	}

	if param == "" {
		return errors.Errorf("pdfcpu: unknown parameter prefix \"%s\"", paramPrefix)
	}

	return m[param](paramValueStr, ib)
}

func imageBox(s, src, url string) (*primitives.ImageBox, string, error) {

	if !strings.HasPrefix(s, "@img") || len(s) < 6 {
		return nil, "", errors.Errorf("pdfcpu: parsing cvs fieldNames: missing @img: <%s>", s)
	}

	s = s[4:]
	if s[0] != '(' || s[len(s)-1] != ')' {
		return nil, "", errors.Errorf("pdfcpu: parsing cvs fieldNames: corrupted @img: <%s>", s)
	}

	s = s[1 : len(s)-1]
	if len(s) == 0 {
		return nil, "", errors.Errorf("pdfcpu: parsing cvs fieldNames: empty @img: <%s>", s)
	}

	ib := primitives.ImageBox{Src: src, Dx: 0, Dy: 0, Width: 0, Height: 0}
	if url != "" {
		ib.Url = url
	}
	ss := strings.Split(s, ",")

	for _, s := range ss {
		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, "", errors.Errorf("pdfcpu: parsing cvs fieldNames: corrupted @img: <%s>", s)
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := imgParamMap.processImageBoxArg(paramPrefix, paramValueStr, &ib); err != nil {
			return nil, "", err
		}
	}

	return &ib, ib.PageNr, nil
}

// FieldMap returns structures needed to fill a form via CSV.
func FieldMap(fieldNames, formRecord []string) (map[string]CSVFieldAttributes, map[string]*Page, error) {
	fm := map[string]CSVFieldAttributes{}
	im := map[string]*Page{}
	for i, fieldName := range fieldNames {
		var lock bool
		if fieldName[0] == '*' {
			lock = true
			fieldName = fieldName[1:]
			if fieldName[0] == '@' {
				continue
			}
		}
		vv := strings.Split(formRecord[i], ",")
		if fieldName[0] != '@' {
			v1 := vv[0]
			if len(v1) > 1 && v1[0] == '*' {
				lock = true
				vv[0] = vv[0][1:]
			}
			fm[fieldName] = CSVFieldAttributes{Values: vv, Lock: lock}
			continue
		}

		// @img defines a virtual image field by rendering an imageBox.
		// For CSV we keep it simple and support the most important imageBox attributes only:
		//
		// "@img(page:1, pos:40 350, w:290, h:200, bgcol:#F5F5DC, border:5 round LightGray)"

		if len(vv) == 0 || len(vv) > 2 {
			// Skip invalid image field
			continue
		}

		src, url := "", ""
		if len(vv) == 1 && vv[0][0] == '(' {
			// link only, no image
			url = vv[0][1 : len(vv[0])-1]
		} else {
			src = vv[0]
			if len(vv) == 2 {
				url = vv[1][1 : len(vv[1])-1]
			}
		}

		ib, pageNr, err := imageBox(fieldName, src, url)
		if err != nil {
			return nil, nil, err
		}

		if ib == nil {
			continue
		}

		p, ok := im[pageNr]
		if !ok {
			p = &Page{}
			im[pageNr] = p
		}
		p.ImageBoxes = append(p.ImageBoxes, ib)
	}

	return fm, im, nil
}

// FillDetails returns a closure that returns new form data provided by CSV or JSON.
func FillDetails(form *Form, fieldMap map[string]CSVFieldAttributes) func(id string, fieldType FieldType, format DataFormat) ([]string, bool, bool) {
	f := form
	fm := fieldMap

	return func(id string, fieldType FieldType, format DataFormat) ([]string, bool, bool) {

		if format == CSV {
			fa, ok := fm[id]
			if ok {
				return fa.Values, fa.Lock, true
			}
			return nil, false, false
		}

		switch fieldType {
		case FTCheckBox:
			v, lock, ok := form.checkBoxValueAndLock(id)
			c := "f"
			if v {
				c = "t"
			}
			return []string{c}, lock, ok

		case FTRadioButtonGroup:
			v, lock, ok := form.radioButtonGroupValueAndLock(id)
			return []string{v}, lock, ok

		case FTComboBox:
			v, lock, ok := f.comboBoxValueAndLock(id)
			return []string{v}, lock, ok

		case FTListBox:
			return f.listBoxValuesAndLock(id)

		case FTDate:
			v, lock, ok := f.dateFieldValueAndLock(id)
			return []string{v}, lock, ok

		case FTText:
			v, lock, ok := f.textFieldValueAndLock(id)
			return []string{v}, lock, ok
		}

		return nil, false, false
	}
}

// FillForm populates form fields as provided by fillDetails and also supports virtual image fields.
func FillForm(
	ctx *model.Context,
	fillDetails func(id string, fieldType FieldType, format DataFormat) ([]string, bool, bool),
	imgs map[string]*Page,
	format DataFormat) (bool, []*model.Page, error) {

	xRefTable := ctx.XRefTable

	fields, err := fields(xRefTable)
	if err != nil {
		return false, nil, err
	}

	fonts := map[string]types.IndirectRef{}
	pIndRefs := map[types.IndirectRef]bool{}

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

		for _, ir := range *(wAnnots.IndRefs) {

			found, pIndRef, id, ft, err := isField(xRefTable, ir, fields)
			if err != nil {
				return false, nil, err
			}
			if !found {
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
				return false, nil, err
			}
			if len(d) == 0 {
				continue
			}

			var locked bool
			ff := d.IntEntry("Ff")
			if ff != nil {
				locked = uint(primitives.FieldFlags(*ff))&uint(primitives.FieldReadOnly) > 0
			}

			if ft == nil {
				ft = d.NameEntry("FT")
				if ft == nil {
					return false, nil, errors.Errorf("pdfcpu: corrupt form field %s: missing entry FT\n%s", id, d)
				}
			}

			switch *ft {

			case "Btn":
				if len(d.ArrayEntry("Kids")) > 0 {
					vv, lock, found := fillDetails(id, FTRadioButtonGroup, format)
					if !found {
						continue
					}
					if locked {
						if !lock {
							unlockFormField(d)
							ok = true
						}
					} else {
						if lock {
							lockFormField(d)
							ok = true
						}
					}
					vNew := vv[0]
					vOld := ""
					if s := d.NameEntry("V"); s != nil {
						n, err := types.DecodeName(*s)
						if err != nil {
							return false, nil, err
						}
						if n != "Off" {
							vOld = n
						}
					}
					if vNew == vOld {
						continue
					}
					s := types.EncodeName(vNew)
					v := types.Name(s)
					d["V"] = v

					for _, o := range d.ArrayEntry("Kids") {
						d, err := xRefTable.DereferenceDict(o)
						if err != nil {
							return false, nil, err
						}
						d1 := d.DictEntry("AP")
						if d1 == nil {
							return false, nil, errors.New("pdfcpu: corrupt form field: missing entry AP")
						}
						d2 := d1.DictEntry("N")
						if d2 == nil {
							return false, nil, errors.New("pdfcpu: corrupt AP field: missing entry N")
						}
						for k := range d2 {
							k, err := types.DecodeName(k)
							if err != nil {
								return false, nil, err
							}
							if k != "Off" {
								d["AS"] = types.Name("Off")
								if k == vNew {
									d["AS"] = v
								}
								break
							}
						}
					}
					ok = true

				} else {

					vv, lock, found := fillDetails(id, FTCheckBox, format)
					if !found {
						continue
					}
					if locked {
						if !lock {
							unlockFormField(d)
							ok = true
						}
					} else {
						if lock {
							lockFormField(d)
							ok = true
						}
					}
					s := strings.ToLower(vv[0])
					vNew := strings.HasPrefix(s, "t")
					vOld := false
					if o, found := d.Find("V"); found {
						vOld = o.(types.Name) == "Yes"
					}
					if vNew == vOld {
						continue
					}
					v := types.Name("Off")
					if vNew {
						v = types.Name("Yes")
					}
					d["V"] = v
					if _, found := d.Find("AS"); found {
						offName, yesName := primitives.CalcCheckBoxASNames(d)
						//fmt.Printf("off:<%s> yes:<%s>\n", offName, yesName)
						asName := yesName
						if v == "Off" {
							asName = offName
						}
						d["AS"] = asName
					}
					ok = true
				}

			case "Ch":
				if ff == nil {
					return false, nil, errors.New("pdfcpu: corrupt form field: missing entry Ff")
				}
				opts, err := parseOptions(xRefTable, d)
				if err != nil {
					return false, nil, err
				}
				if len(opts) == 0 {
					return false, nil, errors.New("pdfcpu: missing Opts")
				}

				if primitives.FieldFlags(*ff)&primitives.FieldCombo > 0 {
					vv, lock, found := fillDetails(id, FTComboBox, format)
					if !found {
						continue
					}
					vNew := vv[0]
					if locked {
						if !lock {
							unlockFormField(d)
							d.Delete("AP")
							ok = true
						}
					} else if lock {
						lockFormField(d)
						if err := primitives.EnsureComboBoxAP(ctx, d, vNew, fonts); err != nil {
							return false, nil, err
						}
						ok = true
					}

					vOld := ""
					if sl := d.StringLiteralEntry("V"); sl != nil {
						s, err := types.StringLiteralToString(*sl)
						if err != nil {
							return false, nil, err
						}
						vOld = s
					}
					if vNew == vOld {
						continue
					}
					s, err := types.EscapeUTF16String(vNew)
					if err != nil {
						return false, nil, err
					}

					ind := types.Array{}
					for i, o := range opts {
						if o == vNew {
							ind = append(ind, types.Integer(i))
							break
						}
					}
					if len(ind) > 0 {
						d["I"] = ind
						d["V"] = types.StringLiteral(*s)
					} else {
						d.Delete("I")
						d.Delete("V")
					}
					ok = true
					continue
				}

				vNew, lock, found := fillDetails(id, FTListBox, format)
				if !found {
					continue
				}

				var vOld []string
				multi := primitives.FieldFlags(*ff)&primitives.FieldMultiselect > 0
				if !multi {
					if sl := d.StringLiteralEntry("V"); sl != nil {
						s, err := types.StringLiteralToString(*sl)
						if err != nil {
							return false, nil, err
						}
						vOld = []string{s}
					}
				} else {
					ss, err := parseStringLiteralArray(xRefTable, d, "V")
					if err != nil {
						return false, nil, err
					}
					vOld = ss
				}
				if locked {
					if !lock {
						unlockFormField(d)
						ok = true
					}
					continue
				} else {
					if lock {
						lockFormField(d)
						ok = true
					}
				}

				if types.EqualSlices(vOld, vNew) {
					continue
				}

				ind := types.Array{}
				if multi {
					arr := types.Array{}
					for _, v := range vNew {
						for i, o := range opts {
							if o == v {
								ind = append(ind, types.Integer(i))
								break
							}
						}
						s, err := types.EscapeUTF16String(v)
						if err != nil {
							return false, nil, err
						}
						arr = append(arr, types.StringLiteral(*s))
					}
					if len(vNew) > 0 {
						d["I"] = ind
						d["V"] = arr
					} else {
						d.Delete("I")
						d.Delete("V")
					}
				} else {
					v := vNew[0]
					s, err := types.EscapeUTF16String(v)
					if err != nil {
						return false, nil, err
					}
					for i, o := range opts {
						if o == v {
							ind = append(ind, types.Integer(i))
							break
						}
					}
					if len(ind) > 0 {
						d["I"] = ind
						d["V"] = types.StringLiteral(*s)
					} else {
						d.Delete("I")
						d.Delete("V")
					}
				}

				if err := primitives.EnsureListBoxAP(ctx, d, opts, ind, fonts); err != nil {
					return false, nil, err
				}

				ok = true

			case "Tx":
				df, err := extractDateFormat(xRefTable, d)
				if err != nil {
					return false, nil, err
				}
				vOld := ""
				if o, found := d.Find("V"); found {
					sl, _ := o.(types.StringLiteral)
					s, err := types.StringLiteralToString(sl)
					if err != nil {
						return false, nil, err
					}
					vOld = s
				}
				if df != nil {
					vv, lock, found := fillDetails(id, FTDate, format)
					if !found {
						continue
					}
					if locked {
						if !lock {
							unlockFormField(d)
							ok = true
						}
					} else {
						if lock {
							lockFormField(d)
							ok = true
						}
					}
					vNew := vv[0]
					if vNew == vOld {
						continue
					}
					s, err := types.EscapeUTF16String(vNew)
					if err != nil {
						return false, nil, err
					}
					d["V"] = types.StringLiteral(*s)

					if err := primitives.EnsureDateFieldAP(ctx, d, vNew, fonts); err != nil {
						return false, nil, err
					}

					ok = true

					continue
				}

				vv, lock, found := fillDetails(id, FTText, format)
				if !found {
					continue
				}
				if locked {
					if !lock {
						unlockFormField(d)
						ok = true
					}
				} else {
					if lock {
						lockFormField(d)
						ok = true
					}
				}
				vNew := vv[0]
				if vNew == vOld {
					continue
				}
				s, err := types.EscapeUTF16String(vNew)
				if err != nil {
					return false, nil, err
				}
				d["V"] = types.StringLiteral(*s)

				ff := d.IntEntry("Ff")
				multiLine := ff != nil && uint(primitives.FieldFlags(*ff))&uint(primitives.FieldMultiline) > 0
				if err := primitives.EnsureTextFieldAP(ctx, d, vNew, multiLine, fonts); err != nil {
					return false, nil, err
				}

				ok = true
			}
		}
	}

	for fName, indRef := range fonts {
		if len(ctx.UsedGIDs[fName]) == 0 {
			continue
		}
		fDict, err := xRefTable.DereferenceDict(indRef)
		if err != nil {
			return false, nil, err
		}
		fr := model.FontResource{}
		if err := pdffont.IndRefsForUserfontUpdate(xRefTable, fDict, "", &fr); err != nil {
			return false, nil, pdffont.ErrCorruptFontDict
		}
		if err := pdffont.UpdateUserfont(xRefTable, fName, fr); err != nil {
			return false, nil, err
		}
	}

	var pages []*model.Page

	if len(imgs) > 0 {
		if pages, err = addImages(ctx, imgs); err != nil {
			return false, nil, err
		}
	}

	return ok, pages, nil
}
