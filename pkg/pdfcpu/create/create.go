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

// Package create generates PDF documents from declarative descriptions.
package create

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/font"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/primitives"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ErrMissingJSONReader signals a missing required JSON input reader.
var ErrMissingJSONReader = errors.New("missing JSON reader")

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := contextutil.Check(r.ctx); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

func ensureFontIndRef(c context.Context, xRefTable *model.XRefTable, fontName string, frPage model.FontResource, fonts model.FontMap) (*types.IndirectRef, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	frGlobal, ok := fonts[fontName]
	if !ok {
		return nil, fmt.Errorf("missing global font: %s", fontName)
	}

	// Do we have an already created indRef or an indRef from AP form fonts or fonts we are reusing?
	if frGlobal.Res.IndRef != nil {
		if frPage.Res.IndRef != nil && *frPage.Res.IndRef != *frGlobal.Res.IndRef {
			return nil, fmt.Errorf("multiple objstreams for font: %s detected", fontName)
		}

		userFont, err := xRefTable.FontRepository().IsUserFont(c, fontName)
		if err != nil {
			return nil, fmt.Errorf("font %s: load metrics: %w", fontName, err)
		}
		if userFont && frGlobal.FontFile != nil {
			if err := pdffont.UpdateUserfont(c, xRefTable, fontName, frGlobal); err != nil {
				return nil, fmt.Errorf("font %s: update user font: %w", fontName, err)
			}
			frGlobal.FontFile = nil
		}
	} else {
		ir, err := pdffont.EnsureFontDict(c, xRefTable, fontName, frPage.Lang, "", false, frPage.Res.IndRef)
		if err != nil {
			return nil, fmt.Errorf("font %s: ensure font dict: %w", fontName, err)
		}

		frGlobal.Res.IndRef = ir
	}

	fonts[fontName] = frGlobal

	return frGlobal.Res.IndRef, nil
}

func addPageResources(c context.Context, xRefTable *model.XRefTable, d types.Dict, p model.Page, fonts model.FontMap) error {
	fontRes := types.Dict{}
	for fontName, frPage := range p.Fm {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		ir, err := ensureFontIndRef(c, xRefTable, fontName, frPage, fonts)
		if err != nil {
			return fmt.Errorf("font resource %s: %w", frPage.Res.ID, err)
		}
		fontRes[frPage.Res.ID] = *ir
	}

	imgRes := types.Dict{}
	for _, img := range p.Im {
		if img.Res.IndRef == nil {
			return fmt.Errorf("image resource %s: missing indirect reference", img.Res.ID)
		}
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

func updatePageResources(c context.Context, xRefTable *model.XRefTable, d, resDict types.Dict, p model.Page, fonts model.FontMap) error {
	if len(p.Fm) > 0 {
		fontRes, ok := resDict["Font"].(types.Dict)
		if !ok {
			fontRes = types.Dict{}
		}
		for fontName, frPage := range p.Fm {
			if err := contextutil.Check(c); err != nil {
				return err
			}
			ir, err := ensureFontIndRef(c, xRefTable, fontName, frPage, fonts)
			if err != nil {
				return fmt.Errorf("font resource %s: %w", frPage.Res.ID, err)
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
			if img.Res.IndRef == nil {
				return fmt.Errorf("image resource %s: missing indirect reference", img.Res.ID)
			}
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
				return fmt.Errorf("annotation tab %d: create object: %w", k, err)
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
				return fmt.Errorf("annotation %d: create object: %w", k, err)
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

// CreatePage generates a page dictionary for p and supports cancellation.
func CreatePage(c context.Context, xRefTable *model.XRefTable, parentPageIndRef types.IndirectRef, p *model.Page, fonts model.FontMap) (*types.IndirectRef, types.Dict, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, nil, err
	}
	pageDict := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Page"),
			"Parent":   parentPageIndRef,
			"MediaBox": p.MediaBox.Array(),
			"CropBox":  p.CropBox.Array(),
		},
	)

	err := addPageResources(c, xRefTable, pageDict, *p, fonts)
	if err != nil {
		return nil, nil, fmt.Errorf("page resources: %w", err)
	}

	ir, err := xRefTable.StreamDictIndRef(p.Buf.Bytes())
	if err != nil {
		return nil, pageDict, fmt.Errorf("content stream: %w", err)
	}
	pageDict.Insert("Contents", *ir)

	pageDictIndRef, err := xRefTable.IndRefForNewObject(pageDict)
	if err != nil {
		return nil, nil, fmt.Errorf("page dict: create object: %w", err)
	}

	if len(p.AnnotTabs) == 0 && len(p.Annots) == 0 && len(p.LinkAnnots) == 0 {
		return pageDictIndRef, pageDict, nil
	}

	if err := setAnnotationParentsAndFields(xRefTable, p, *pageDictIndRef); err != nil {
		return nil, nil, fmt.Errorf("annotations: set parents: %w", err)
	}

	arr, err := mergeAnnotations(nil, p.Annots, p.AnnotTabs)
	if err != nil {
		return nil, nil, fmt.Errorf("annotations: merge: %w", err)
	}

	for i, la := range p.LinkAnnots {
		if err := contextutil.Check(c); err != nil {
			return nil, nil, err
		}
		d, err := la.RenderDict(xRefTable, pageDictIndRef)
		if err != nil {
			return nil, nil, fmt.Errorf("link annotation %d: render: %w", i+1, err)
		}
		ir, err := xRefTable.IndRefForNewObject(d)
		if err != nil {
			return nil, nil, fmt.Errorf("link annotation %d: create object: %w", i+1, err)
		}
		arr = append(arr, *ir)
	}

	pageDict["Annots"] = arr

	return pageDictIndRef, pageDict, contextutil.Check(c)
}

// UpdatePage updates the existing page dictionary d with content provided by p and supports cancellation.
func UpdatePage(c context.Context, xRefTable *model.XRefTable, dIndRef types.IndirectRef, d, res types.Dict, p *model.Page, fonts model.FontMap) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	// TODO Account for existing page rotation.

	err := updatePageResources(c, xRefTable, d, res, *p, fonts)
	if err != nil {
		return fmt.Errorf("page resources: %w", err)
	}

	err = xRefTable.AppendContent(d, p.Buf.Bytes())
	if err != nil {
		return fmt.Errorf("append content: %w", err)
	}

	if len(p.AnnotTabs) == 0 && len(p.Annots) == 0 && len(p.LinkAnnots) == 0 {
		return nil
	}

	if err := setAnnotationParentsAndFields(xRefTable, p, dIndRef); err != nil {
		return fmt.Errorf("annotations: set parents: %w", err)
	}

	annots, err := xRefTable.DereferenceArray(d["Annots"])
	if err != nil {
		return fmt.Errorf("annotations: dereference existing array: %w", err)
	}

	arr, err := mergeAnnotations(annots, p.Annots, p.AnnotTabs)
	if err != nil {
		return fmt.Errorf("annotations: merge: %w", err)
	}

	for i, la := range p.LinkAnnots {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		d, err := la.RenderDict(xRefTable, &dIndRef)
		if err != nil {
			return fmt.Errorf("link annotation %d: render: %w", i+1, err)
		}
		ir, err := xRefTable.IndRefForNewObject(d)
		if err != nil {
			return fmt.Errorf("link annotation %d: create object: %w", i+1, err)
		}
		arr = append(arr, *ir)
	}

	d["Annots"] = arr

	return contextutil.Check(c)
}

func cacheFormFieldIDs(c context.Context, ctx *model.Context, pdf *primitives.PDF) error {
	if ctx.Form == nil {
		return nil
	}

	o, found := ctx.Form.Find("Fields")
	if !found {
		return nil
	}

	arr, err := ctx.DereferenceArray(o)
	if err != nil {
		return fmt.Errorf("form fields: dereference array: %w", err)
	}

	for i, ir := range arr {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		d, err := ctx.DereferenceDict(ir)
		if err != nil {
			return fmt.Errorf("form field %d: dereference dict: %w", i+1, err)
		}
		if len(d) == 0 {
			continue
		}
		id, _, err := ctx.DereferenceStringEntry(d, "T")
		if err != nil {
			return fmt.Errorf("form field %d: entry T: %w", i+1, err)
		}
		if id != nil {
			pdf.OldFieldIDs[*id] = true
		}
	}

	return nil
}

func cacheResIDs(c context.Context, ctx *model.Context, pdf *primitives.PDF) error {
	// Iterate over all pages of ctx and prepare a resIds []string for inherited "Font" and "XObject" resources.
	for i := 1; i <= ctx.PageCount; i++ {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		_, _, inhPA, err := ctx.PageDict(i, true)
		if err != nil {
			return fmt.Errorf("page %d: collect inherited resources: %w", i, err)
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

func parseFromJSON(c context.Context, ctx *model.Context, bb []byte) (*primitives.PDF, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if !json.Valid(bb) {
		return nil, fmt.Errorf("invalid JSON encoding detected")
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
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}

	if pdf.Update() {

		_, found := ctx.RootDict.Find("AcroForm")

		pdf.HasForm = found

		if pdf.HasForm {
			if err := cacheFormFieldIDs(c, ctx, pdf); err != nil {
				return nil, fmt.Errorf("cache form field IDs: %w", err)
			}
		}

		if err := cacheResIDs(c, ctx, pdf); err != nil {
			return nil, fmt.Errorf("cache resource IDs: %w", err)
		}

	}

	if err := pdf.Validate(c); err != nil {
		return nil, fmt.Errorf("validate JSON model: %w", err)
	}

	return pdf, contextutil.Check(c)
}

func appendPage(
	c context.Context,
	ctx *model.Context,
	pagesDictIndRef types.IndirectRef,
	pagesDict types.Dict,
	p *model.Page,
	fonts model.FontMap) error {
	ir, _, err := CreatePage(c, ctx.XRefTable, pagesDictIndRef, p, fonts)
	if err != nil {
		return fmt.Errorf("create page: %w", err)
	}

	if err := ctx.SetValid(*ir); err != nil {
		return fmt.Errorf("mark page object valid: %w", err)
	}

	if err := model.AppendPageTree(ir, 1, pagesDict); err != nil {
		return fmt.Errorf("append to page tree: %w", err)
	}

	ctx.PageCount++

	return nil
}

func updatePage(
	c context.Context,
	ctx *model.Context,
	pageNr int,
	p *model.Page,
	fonts model.FontMap,
) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	pageDict, pageDictIndRef, inhPAttrs, err := ctx.PageDict(pageNr, false)
	if err != nil {
		return fmt.Errorf("read page dict: %w", err)
	}

	// You have to make sure the media/crop boxes align in order to avoid unexpected results!

	if inhPAttrs.Resources == nil {
		inhPAttrs.Resources = types.Dict{}
	}

	return UpdatePage(c, ctx.XRefTable, *pageDictIndRef, pageDict, inhPAttrs.Resources, p, fonts)
}

// UpdatePageTree merges new pages or updates existing pages into ctx and supports cancellation.
func UpdatePageTree(c context.Context, ctx *model.Context, pages []*model.Page, fontMap model.FontMap) (types.Array, model.FontMap, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, nil, err
	}
	pageCount := ctx.PageCount

	ir, err := ctx.Pages()
	if err != nil {
		return nil, nil, fmt.Errorf("page tree root: %w", err)
	}

	d, err := ctx.DereferenceDict(*ir)
	if err != nil {
		return nil, nil, fmt.Errorf("page tree root: dereference dict: %w", err)
	}

	fields := types.Array{}

	for i, p := range pages {
		if err := contextutil.Check(c); err != nil {
			return nil, nil, err
		}

		if p == nil {
			continue
		}

		pageNr := i + 1

		var err error

		if pageNr > pageCount {
			err = appendPage(c, ctx, *ir, d, p, fontMap)
		} else {
			err = updatePage(c, ctx, pageNr, p, fontMap)
		}

		if err != nil {
			return nil, nil, fmt.Errorf("page %d: %w", pageNr, err)
		}

		fields = append(fields, p.Fields...)
	}

	return fields, fontMap, contextutil.Check(c)
}

func prepareFormFontResDict(c context.Context, ctx *model.Context, pdf *primitives.PDF, fonts model.FontMap) (types.Dict, error) {
	d := types.Dict{}

	for id, f := range pdf.FormFonts {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}

		if font.IsCoreFont(f.Name) {
			frGlobal := fonts[f.Name]
			if frGlobal.Res.IndRef != nil {
				d.Insert(id, *frGlobal.Res.IndRef)
				continue
			}
			ir, err := pdffont.EnsureFontDict(c, ctx.XRefTable, f.Name, "", "", true, nil)
			if err != nil {
				return nil, fmt.Errorf("form font %s: ensure font dict: %w", id, err)
			}
			d.Insert(id, *ir)
			continue
		}

		var ir *types.IndirectRef
		frGlobal, ok := fonts["cjk:"+f.Name]
		if ok && frGlobal.Res.IndRef != nil {
			ir = frGlobal.Res.IndRef
		}

		ir, err := pdffont.EnsureFontDict(c, ctx.XRefTable, f.Name, f.Lang, f.Script, true, ir)
		if err != nil {
			return nil, fmt.Errorf("form font %s: ensure font dict: %w", id, err)
		}

		d.Insert(id, *ir)
	}

	return d, nil
}

func createForm(
	c context.Context,
	ctx *model.Context,
	pdf *primitives.PDF,
	fields types.Array,
	fonts model.FontMap) error {
	d := types.Dict{"Fields": fields}

	if len(pdf.FormFonts) > 0 {
		d1, err := prepareFormFontResDict(c, ctx, pdf, fonts)
		if err != nil {
			return fmt.Errorf("prepare font resources: %w", err)
		}
		d["DR"] = types.Dict{"Font": d1}
	}

	ctx.RootDict.Insert("AcroForm", d)

	return nil
}

func updateForm(
	c context.Context,
	ctx *model.Context,
	pdf *primitives.PDF,
	fields types.Array,
	fonts model.FontMap) error {
	d := ctx.Form

	o, _ := d.Find("Fields")
	arr, err := ctx.DereferenceArray(o)
	if err != nil {
		return fmt.Errorf("fields: dereference array: %w", err)
	}
	d["Fields"] = append(arr, fields...)

	if len(pdf.FormFonts) == 0 {
		return nil
	}

	// Update resources.

	o, found := d.Find("DR")
	if !found {
		d1, err := prepareFormFontResDict(c, ctx, pdf, fonts)
		if err != nil {
			return fmt.Errorf("prepare font resources: %w", err)
		}
		d["DR"] = types.Dict{"Font": d1}
		return nil
	}

	resDict, err := ctx.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("default resources: dereference dict: %w", err)
	}

	o, found = resDict.Find("Font")
	if !found {
		return fmt.Errorf("default resources: missing font dict")
	}

	fontResDict, err := ctx.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("font resources: dereference dict: %w", err)
	}

	d1, err := prepareFormFontResDict(c, ctx, pdf, fonts)
	if err != nil {
		return fmt.Errorf("prepare font resources: %w", err)
	}

	for k, v := range d1 {
		if !fontResDict.Insert(k, v) {
			return fmt.Errorf("duplicate font resource id detected: %s", k)
		}
	}

	return nil
}

func handleForm(
	c context.Context,
	ctx *model.Context,
	pdf *primitives.PDF,
	fields types.Array,
	fonts model.FontMap) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	var err error
	if pdf.Update() && pdf.HasForm {
		err = updateForm(c, ctx, pdf, fields, fonts)
	} else {
		err = createForm(c, ctx, pdf, fields, fonts)
	}
	if err != nil {
		return fmt.Errorf("form fields: %w", err)
	}

	for fName, frGlobal := range fonts {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		userFont := false
		var err error
		if !strings.HasPrefix(fName, "cjk:") {
			userFont, err = ctx.XRefTable.FontRepository().IsUserFont(c, fName)
			if err != nil {
				return fmt.Errorf("font %s: load metrics: %w", fName, err)
			}
		}
		if userFont {
			_, err := pdffont.EnsureFontDict(c, ctx.XRefTable, fName, frGlobal.Lang, "", false, frGlobal.Res.IndRef)
			if err != nil {
				return fmt.Errorf("font %s: ensure user font dict: %w", fName, err)
			}
		}
	}

	return contextutil.Check(c)
}

// FromJSON generates PDF content into ctx as provided by rd and supports cancellation.
func FromJSON(c context.Context, ctx *model.Context, rd io.Reader) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if ctx == nil {
		return model.ErrMissingPDFContext
	}
	if ctx.XRefTable == nil {
		return model.ErrMissingXRefTable
	}
	if rd == nil {
		return ErrMissingJSONReader
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, contextReader{ctx: c, r: rd}); err != nil {
		return fmt.Errorf("read JSON: %w", err)
	}

	pdf, err := parseFromJSON(c, ctx, buf.Bytes())
	if err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}

	pages, fontMap, err := pdf.RenderPages(c)
	if err != nil {
		return fmt.Errorf("render pages: %w", err)
	}

	fields, fonts, err := UpdatePageTree(c, ctx, pages, fontMap)
	if err != nil {
		return fmt.Errorf("update page tree: %w", err)
	}

	if len(fields) > 0 {
		if err := handleForm(c, ctx, pdf, fields, fonts); err != nil {
			return fmt.Errorf("update form: %w", err)
		}
	}

	return contextutil.Check(c)
}
