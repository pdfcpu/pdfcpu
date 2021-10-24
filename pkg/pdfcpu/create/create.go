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
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/primitives"
	"github.com/pkg/errors"
)

func parseFromJSON(bb []byte, ctx *pdfcpu.Context) (*primitives.PDF, error) {

	if !json.Valid(bb) {
		return nil, errors.Errorf("pdfcpu: invalid JSON encoding detected.")
	}

	pdf := &primitives.PDF{
		FieldIDs:      pdfcpu.StringSet{},
		Fields:        pdfcpu.Array{},
		FormFontIDs:   map[string]string{},
		Pages:         map[string]*primitives.PDFPage{},
		FontResIDs:    map[int]pdfcpu.Dict{},
		XObjectResIDs: map[int]pdfcpu.Dict{},
		Conf:          ctx.Configuration,
		XRefTable:     ctx.XRefTable,
		Optimize:      ctx.Optimize,
		CheckBoxAPs:   map[float64]*primitives.AP{},
		RadioBtnAPs:   map[float64]*primitives.AP{},
	}

	if err := json.Unmarshal(bb, pdf); err != nil {
		return nil, err
	}

	if err := pdf.Validate(); err != nil {
		return nil, err
	}

	return pdf, nil
}

func cacheResIDs(ctx *pdfcpu.Context, pdf *primitives.PDF) error {
	// Iterate over all pages of ctx and prepare a resIds []string for inherited "Font" and "XObject" resources.
	for i := 1; i <= ctx.PageCount; i++ {
		_, _, inhPA, err := ctx.PageDict(i, true)
		if err != nil {
			return err
		}
		pdf.FontResIDs[i] = pdfcpu.Dict{}
		if inhPA.Resources["Font"] != nil {
			pdf.FontResIDs[i] = inhPA.Resources["Font"].(pdfcpu.Dict)
		}
		pdf.XObjectResIDs[i] = pdfcpu.Dict{}
		if inhPA.Resources["XObject"] != nil {
			pdf.XObjectResIDs[i] = inhPA.Resources["XObject"].(pdfcpu.Dict)
		}
	}
	return nil
}

func appendPage(
	ctx *pdfcpu.Context,
	pagesIndRef pdfcpu.IndirectRef,
	pagesDict pdfcpu.Dict,
	p *pdfcpu.Page,
	fonts map[string]pdfcpu.IndirectRef,
	fields *pdfcpu.Array,
	formFontIDs map[string]string) error {

	ir, _, err := pdfcpu.CreatePage(ctx.XRefTable, pagesIndRef, *p, fonts, fields, formFontIDs)
	if err != nil {
		return err
	}

	return pdfcpu.AppendPageTree(ir, 1, pagesDict)
}

func modifyPageContent(
	ctx *pdfcpu.Context,
	currPageNr int,
	p *pdfcpu.Page,
	fonts map[string]pdfcpu.IndirectRef,
	fields *pdfcpu.Array,
	formFontIDs map[string]string) error {

	pageDict, pageDictIndRef, inhPAttrs, err := ctx.PageDict(currPageNr, false)
	if err != nil {
		return err
	}

	if !inhPAttrs.MediaBox.Equals(*p.MediaBox) {
		return errors.Errorf("pdfcpu: can't update page %d - mediaBox mismatch", currPageNr)
	}

	if !inhPAttrs.CropBox.Equals(*p.CropBox) {
		return errors.Errorf("pdfcpu: can't update page %d - cropBox mismatch", currPageNr)
	}

	if inhPAttrs.Resources == nil {
		inhPAttrs.Resources = pdfcpu.Dict{}
	}

	return pdfcpu.ModifyPageContent(ctx.XRefTable, *pageDictIndRef, pageDict, inhPAttrs.Resources, *p, fonts, fields, formFontIDs)
}

func updatePageTree(
	ctx *pdfcpu.Context,
	pages []*pdfcpu.Page,
	fontMap pdfcpu.FontMap,
	fields *pdfcpu.Array,
	formFontIDs map[string]string) (map[string]pdfcpu.IndirectRef, error) {

	fonts := map[string]pdfcpu.IndirectRef{}
	for fontName, font := range fontMap {
		if font.Res.IndRef != nil {
			fonts[fontName] = *font.Res.IndRef
		}
	}

	pageCount := ctx.PageCount

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		return nil, err
	}

	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		return nil, err
	}

	for i, p := range pages {
		if p == nil {
			continue
		}
		currPageNr := i + 1
		if currPageNr > pageCount {

			if err := appendPage(ctx, *pagesIndRef, pagesDict, p, fonts, fields, formFontIDs); err != nil {
				return nil, err
			}

			ctx.PageCount++

		} else {

			if err := modifyPageContent(ctx, currPageNr, p, fonts, fields, formFontIDs); err != nil {
				return nil, err
			}
		}
	}

	return fonts, nil
}

func createAcroForm(
	ctx *pdfcpu.Context,
	pdf *primitives.PDF,
	fields pdfcpu.Array,
	fonts map[string]pdfcpu.IndirectRef) (pdfcpu.Dict, error) {

	d := pdfcpu.Dict{
		//"NeedAppearances": pdfcpu.Boolean(true),
		"Fields": fields,
	}

	if len(pdf.FormFontIDs) > 0 {
		d1 := pdfcpu.Dict{}
		for fontName, id := range pdf.FormFontIDs {
			indRef, ok := fonts[fontName]
			if ok {
				d1.Insert(id, indRef)
				continue
			}
			var ir *pdfcpu.IndirectRef
			if pdf.Optimize != nil {
				for objNr, fo := range pdf.Optimize.FontObjects {
					//fmt.Printf("searching for %s - obj:%d fontName:%s prefix:%s\n", fontName, objNr, fo.FontName, fo.Prefix)
					if fontName == fo.FontName {
						//fmt.Println("Match!")
						ir = pdfcpu.NewIndirectRef(objNr, 0)
						break
					}
				}
			}

			ir, err := pdfcpu.EnsureFontDict(ctx.XRefTable, fontName, false, ir)
			if err != nil {
				return nil, err
			}

			indRef = *ir
			fonts[fontName] = indRef // Redundant
			d1.Insert(id, indRef)
		}
		d["DR"] = pdfcpu.Dict{"Font": d1}
	}

	return d, nil
}

func FromJSON(rd io.Reader, ctx *pdfcpu.Context) error {

	bb, err := ioutil.ReadAll(rd)
	if err != nil {
		return err
	}

	pdf, err := parseFromJSON(bb, ctx)
	if err != nil {
		return err
	}

	if ctx.Optimize != nil {
		if err := cacheResIDs(ctx, pdf); err != nil {
			return err
		}
		_, found := ctx.RootDict.Find("AcroForm")
		pdf.HasForm = found
	}

	pages, fontMap, fields, err := pdf.RenderPages()
	if err != nil {
		return err
	}

	fonts, err := updatePageTree(ctx, pages, fontMap, &fields, pdf.FormFontIDs)
	if err != nil {
		return err
	}

	if len(fields) > 0 {
		d, err := createAcroForm(ctx, pdf, fields, fonts)
		if err != nil {
			return err
		}

		ctx.RootDict.Insert("AcroForm", d)
	}

	return nil
}
