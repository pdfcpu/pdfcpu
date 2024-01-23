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

package create

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/primitives"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func ensureFontIndRef(xRefTable *model.XRefTable, fontName string, frPage model.FontResource, fonts model.FontMap) (*types.IndirectRef, error) {

	frGlobal, ok := fonts[fontName]
	if !ok {
		return nil, errors.Errorf("pdfcpu: missing global font: %s", fontName)
	}

	// Do we have an already created indRef or an indRef from AP form fonts or fonts we are reusing?
	if frGlobal.Res.IndRef != nil {

		if frPage.Res.IndRef != nil && *frPage.Res.IndRef != *frGlobal.Res.IndRef {
			return nil, errors.Errorf("pdfcpu: multiple objstreams for font: %s detected: ", fontName)
		}

		if font.IsUserFont(fontName) && frGlobal.FontFile != nil {
			if err := pdffont.UpdateUserfont(xRefTable, fontName, frGlobal); err != nil {
				return nil, err
			}
			frGlobal.FontFile = nil
		}

	} else {

		ir, err := pdffont.EnsureFontDict(xRefTable, fontName, frPage.Lang, "", true, false, frPage.Res.IndRef)
		if err != nil {
			return nil, err
		}

		frGlobal.Res.IndRef = ir
	}

	fonts[fontName] = frGlobal

	return frGlobal.Res.IndRef, nil
}

func addPageResources(xRefTable *model.XRefTable, d types.Dict, p model.Page, fonts model.FontMap) error {

	fontRes := types.Dict{}
	for fontName, frPage := range p.Fm {
		ir, err := ensureFontIndRef(xRefTable, fontName, frPage, fonts)
		if err != nil {
			return err
		}
		fontRes[frPage.Res.ID] = *ir
	}

	imgRes := types.Dict{}
	for _, img := range p.Im {
		imgRes[img.Res.ID] = *img.Res.IndRef
	}

	if len(fontRes) > 0 || len(imgRes) > 0 {
		resDict := types.Dict{}
		if len(fontRes) > 0 {
			resDict["Font"] = fontRes
		}
		if len(imgRes) > 0 {
			resDict["XObject"] = imgRes
		}
		d["Resources"] = resDict
	}

	return nil
}

func updatePageResources(xRefTable *model.XRefTable, d, resDict types.Dict, p model.Page, fonts model.FontMap) error {

	if len(p.Fm) > 0 {
		fontRes, ok := resDict["Font"].(types.Dict)
		if !ok {
			fontRes = types.Dict{}
		}
		for fontName, frPage := range p.Fm {
			ir, err := ensureFontIndRef(xRefTable, fontName, frPage, fonts)
			if err != nil {
				return err
			}
			if ir != nil {
				fontRes[frPage.Res.ID] = *ir
			}
		}

		resDict["Font"] = fontRes
	}

	if len(p.Im) > 0 {
		imgRes, ok := resDict["XObject"].(types.Dict)
		if !ok {
			imgRes = types.Dict{}
		}
		for _, img := range p.Im {
			imgRes[img.Res.ID] = *img.Res.IndRef
		}
		resDict["XObject"] = imgRes
	}

	if len(p.Fm) > 0 || len(p.Im) > 0 {
		d["Resources"] = resDict
	}

	return nil
}

func setAnnotationParentsAndFields(xRefTable *model.XRefTable, p *model.Page, pIndRef types.IndirectRef) error {
	for k, an := range p.AnnotTabs {
		if an.IndRef == nil {
			an.Dict["P"] = pIndRef
			indRef, err := xRefTable.IndRefForNewObject(an.Dict)
			if err != nil {
				return err
			}
			an.IndRef = indRef
			p.AnnotTabs[k] = an
		}
		p.Fields = append(p.Fields, *an.IndRef)
	}
	for k, an := range p.Annots {
		if an.IndRef == nil {
			an.Dict["P"] = pIndRef
			indRef, err := xRefTable.IndRefForNewObject(an.Dict)
			if err != nil {
				return err
			}
			an.IndRef = indRef
			p.Annots[k] = an
		}
		p.Fields = append(p.Fields, *an.IndRef)
	}
	return nil
}

func addAnnotations(ff []model.FieldAnnotation, m map[int]model.FieldAnnotation) types.Array {

	arr := types.Array{}

	for i, j := 0, 0; j < len(ff); i++ {
		an, ok := m[i+1]
		if ok {
			if an.Kids == nil {
				arr = append(arr, *an.IndRef)
			} else {
				arr = append(arr, an.Kids...)
			}
			an.Field = true
			m[i+1] = an
			continue
		}
		if j < len(ff) {
			an = ff[j]
			if an.Kids == nil {
				arr = append(arr, *an.IndRef)
			} else {
				arr = append(arr, an.Kids...)
			}
			j++
			continue
		}
		break
	}

	keys := make([]int, 0, len(m))
	for k, an := range m {
		if !an.Field {
			keys = append(keys, k)
		}
	}
	sort.Ints(keys)

	for _, k := range keys {
		an := m[k]
		if an.Kids == nil {
			arr = append(arr, *an.IndRef)
		} else {
			arr = append(arr, an.Kids...)
		}
	}

	return arr
}

func mergeAnnotations(oldAnnots types.Array, ff []model.FieldAnnotation, m map[int]model.FieldAnnotation) (types.Array, error) {

	if len(oldAnnots) == 0 {
		return addAnnotations(ff, m), nil
	}

	arr := types.Array{}
	i := 0
	for j := 0; j < len(oldAnnots); i++ {
		an, ok := m[i+1]
		if ok {
			if an.Kids == nil {
				arr = append(arr, *an.IndRef)
			} else {
				arr = append(arr, an.Kids...)
			}
			an.Field = true
			m[i+1] = an
			continue
		}
		if j < len(oldAnnots) {
			arr = append(arr, oldAnnots[j])
			j++
			continue
		}
		break
	}

	for j := 0; j < len(ff); i++ {
		an, ok := m[i+1]
		if ok {
			if an.Kids == nil {
				arr = append(arr, *an.IndRef)
			} else {
				arr = append(arr, an.Kids...)
			}
			an.Field = true
			m[i+1] = an
			continue
		}
		if j < len(ff) {
			an = ff[j]
			if an.Kids == nil {
				arr = append(arr, *an.IndRef)
			} else {
				arr = append(arr, an.Kids...)
			}
			j++
			continue
		}
		break
	}

	keys := make([]int, 0, len(m))
	for k, an := range m {
		if !an.Field {
			keys = append(keys, k)
		}
	}

	sort.Ints(keys)

	for _, k := range keys {
		an := m[k]
		if an.Kids == nil {
			arr = append(arr, *an.IndRef)
		} else {
			arr = append(arr, an.Kids...)
		}
	}

	return arr, nil
}

// CreatePage generates a page dict for p.
func CreatePage(
	xRefTable *model.XRefTable,
	parentPageIndRef types.IndirectRef,
	p *model.Page,
	fonts model.FontMap) (*types.IndirectRef, types.Dict, error) {

	pageDict := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Page"),
			"Parent":   parentPageIndRef,
			"MediaBox": p.MediaBox.Array(),
			"CropBox":  p.CropBox.Array(),
		},
	)

	err := addPageResources(xRefTable, pageDict, *p, fonts)
	if err != nil {
		return nil, nil, err
	}

	ir, err := xRefTable.StreamDictIndRef(p.Buf.Bytes())
	if err != nil {
		return nil, pageDict, err
	}
	pageDict.Insert("Contents", *ir)

	pageDictIndRef, err := xRefTable.IndRefForNewObject(pageDict)
	if err != nil {
		return nil, nil, err
	}

	if len(p.AnnotTabs) == 0 && len(p.Annots) == 0 && len(p.LinkAnnots) == 0 {
		return pageDictIndRef, pageDict, nil
	}

	if err := setAnnotationParentsAndFields(xRefTable, p, *pageDictIndRef); err != nil {
		return nil, nil, err
	}

	arr, err := mergeAnnotations(nil, p.Annots, p.AnnotTabs)
	if err != nil {
		return nil, nil, err
	}

	for _, la := range p.LinkAnnots {
		d, err := la.RenderDict(xRefTable, *pageDictIndRef)
		if err != nil {
			return nil, nil, &json.UnsupportedTypeError{}
		}
		ir, err := xRefTable.IndRefForNewObject(d)
		if err != nil {
			return nil, nil, err
		}
		arr = append(arr, *ir)
	}

	pageDict["Annots"] = arr

	return pageDictIndRef, pageDict, err
}

// UpdatePage updates the existing page dict d with content provided by p.
func UpdatePage(xRefTable *model.XRefTable, dIndRef types.IndirectRef, d, res types.Dict, p *model.Page, fonts model.FontMap) error {

	// TODO Account for existing page rotation.

	err := updatePageResources(xRefTable, d, res, *p, fonts)
	if err != nil {
		return err
	}

	err = xRefTable.AppendContent(d, p.Buf.Bytes())
	if err != nil {
		return err
	}

	if len(p.AnnotTabs) == 0 && len(p.Annots) == 0 && len(p.LinkAnnots) == 0 {
		return nil
	}

	if err := setAnnotationParentsAndFields(xRefTable, p, dIndRef); err != nil {
		return err
	}

	annots, err := xRefTable.DereferenceArray(d["Annots"])
	if err != nil {
		return err
	}

	arr, err := mergeAnnotations(annots, p.Annots, p.AnnotTabs)
	if err != nil {
		return err
	}

	for _, la := range p.LinkAnnots {
		d, err := la.RenderDict(xRefTable, dIndRef)
		if err != nil {
			return err
		}
		ir, err := xRefTable.IndRefForNewObject(d)
		if err != nil {
			return err
		}
		arr = append(arr, *ir)
	}

	d["Annots"] = arr

	return nil
}

func cacheFormFieldIDs(ctx *model.Context, pdf *primitives.PDF) error {

	if ctx.Form == nil {
		return nil
	}

	o, found := ctx.Form.Find("Fields")
	if !found {
		return nil
	}

	arr, err := ctx.DereferenceArray(o)
	if err != nil {
		return err
	}

	for _, ir := range arr {
		d, err := ctx.DereferenceDict(ir)
		if err != nil {
			return err
		}
		if len(d) == 0 {
			continue
		}
		id := d.StringEntry("T")
		if id != nil {
			pdf.OldFieldIDs[*id] = true
		}
	}

	return nil
}

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

func parseFromJSON(ctx *model.Context, bb []byte) (*primitives.PDF, error) {

	if !json.Valid(bb) {
		return nil, errors.Errorf("pdfcpu: invalid JSON encoding detected.")
	}

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
	}

	if err := json.Unmarshal(bb, pdf); err != nil {
		return nil, err
	}

	if pdf.Update() {

		_, found := ctx.RootDict.Find("AcroForm")

		pdf.HasForm = found

		if pdf.HasForm {
			if err := cacheFormFieldIDs(ctx, pdf); err != nil {
				return nil, err
			}
		}

		if err := cacheResIDs(ctx, pdf); err != nil {
			return nil, err
		}

	}

	if err := pdf.Validate(); err != nil {
		return nil, err
	}

	return pdf, nil
}

func appendPage(
	ctx *model.Context,
	pagesDictIndRef types.IndirectRef,
	pagesDict types.Dict,
	p *model.Page,
	fonts model.FontMap) error {

	ir, _, err := CreatePage(ctx.XRefTable, pagesDictIndRef, p, fonts)
	if err != nil {
		return err
	}

	if err := model.AppendPageTree(ir, 1, pagesDict); err != nil {
		return err
	}

	ctx.PageCount++

	return nil
}

func updatePage(ctx *model.Context, pageNr int, p *model.Page, fonts model.FontMap) error {

	pageDict, pageDictIndRef, inhPAttrs, err := ctx.PageDict(pageNr, false)
	if err != nil {
		return err
	}

	// You have to make sure the media/crop boxes align in order to avoid unexpected results!

	if inhPAttrs.Resources == nil {
		inhPAttrs.Resources = types.Dict{}
	}

	return UpdatePage(ctx.XRefTable, *pageDictIndRef, pageDict, inhPAttrs.Resources, p, fonts)
}

// UpdatePageTree merges new pages or updates existing pages into ctx.
func UpdatePageTree(ctx *model.Context, pages []*model.Page, fontMap model.FontMap) (types.Array, model.FontMap, error) {

	pageCount := ctx.PageCount

	ir, err := ctx.Pages()
	if err != nil {
		return nil, nil, err
	}

	d, err := ctx.DereferenceDict(*ir)
	if err != nil {
		return nil, nil, err
	}

	fields := types.Array{}

	for i, p := range pages {

		if p == nil {
			continue
		}

		pageNr := i + 1

		var err error

		if pageNr > pageCount {
			err = appendPage(ctx, *ir, d, p, fontMap)
		} else {
			err = updatePage(ctx, pageNr, p, fontMap)
		}

		if err != nil {
			return nil, nil, err
		}

		fields = append(fields, p.Fields...)
	}

	return fields, fontMap, nil
}

func prepareFormFontResDict(ctx *model.Context, pdf *primitives.PDF, fonts model.FontMap) (types.Dict, error) {

	d := types.Dict{}

	for id, f := range pdf.FormFonts {

		if font.IsCoreFont(f.Name) {
			frGlobal := fonts[f.Name]
			if frGlobal.Res.IndRef != nil {
				d.Insert(id, *frGlobal.Res.IndRef)
				continue
			}
			ir, err := pdffont.EnsureFontDict(ctx.XRefTable, f.Name, "", "", false, false, nil)
			if err != nil {
				return nil, err
			}
			d.Insert(id, *ir)
			continue
		}

		var ir *types.IndirectRef
		frGlobal, ok := fonts["cjk:"+f.Name]
		if ok && frGlobal.Res.IndRef != nil {
			ir = frGlobal.Res.IndRef
		}

		ir, err := pdffont.EnsureFontDict(ctx.XRefTable, f.Name, f.Lang, f.Script, false, true, ir)
		if err != nil {
			return nil, err
		}

		d.Insert(id, *ir)
	}

	return d, nil
}

func createForm(
	ctx *model.Context,
	pdf *primitives.PDF,
	fields types.Array,
	fonts model.FontMap) error {

	d := types.Dict{"Fields": fields}

	if len(pdf.FormFonts) > 0 {
		d1, err := prepareFormFontResDict(ctx, pdf, fonts)
		if err != nil {
			return err
		}
		d["DR"] = types.Dict{"Font": d1}
	}

	ctx.RootDict.Insert("AcroForm", d)

	return nil
}

func updateForm(
	ctx *model.Context,
	pdf *primitives.PDF,
	fields types.Array,
	fonts model.FontMap) error {

	d := ctx.Form

	o, _ := d.Find("Fields")
	arr, err := ctx.DereferenceArray(o)
	if err != nil {
		return err
	}
	d["Fields"] = append(arr, fields...)

	if len(pdf.FormFonts) == 0 {
		return nil
	}

	// Update resources.

	o, found := d.Find("DR")
	if !found {
		d1, err := prepareFormFontResDict(ctx, pdf, fonts)
		if err != nil {
			return err
		}
		d["DR"] = types.Dict{"Font": d1}
		return nil
	}

	resDict, err := ctx.DereferenceDict(o)
	if err != nil {
		return err
	}

	o, found = resDict.Find("Font")
	if !found {
		return err
	}

	fontResDict, err := ctx.DereferenceDict(o)
	if err != nil {
		return err
	}

	d1, err := prepareFormFontResDict(ctx, pdf, fonts)
	if err != nil {
		return err
	}

	for k, v := range d1 {
		if !fontResDict.Insert(k, v) {
			return errors.Errorf("pdfcpu: duplicate font resource id detected: %s", k)
		}
	}

	return nil
}

func handleForm(
	ctx *model.Context,
	pdf *primitives.PDF,
	fields types.Array,
	fonts model.FontMap) error {

	var err error
	if pdf.Update() && pdf.HasForm {
		err = updateForm(ctx, pdf, fields, fonts)
	} else {
		err = createForm(ctx, pdf, fields, fonts)
	}
	if err != nil {
		return err
	}

	for fName, frGlobal := range fonts {
		if !strings.HasPrefix(fName, "cjk:") && font.IsUserFont(fName) {
			_, err := pdffont.EnsureFontDict(ctx.XRefTable, fName, frGlobal.Lang, "", true, false, frGlobal.Res.IndRef)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// FromJSON generates PDF content into ctx as provided by rd.
func FromJSON(ctx *model.Context, rd io.Reader) error {

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rd); err != nil {
		return err
	}

	pdf, err := parseFromJSON(ctx, buf.Bytes())
	if err != nil {
		return err
	}

	pages, fontMap, err := pdf.RenderPages()
	if err != nil {
		return err
	}

	fields, fonts, err := UpdatePageTree(ctx, pages, fontMap)
	if err != nil {
		return err
	}

	if len(fields) > 0 {
		if err := handleForm(ctx, pdf, fields, fonts); err != nil {
			return err
		}
	}

	return nil
}
