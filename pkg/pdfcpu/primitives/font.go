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

package primitives

import (
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

type FormFont struct {
	pdf      *PDF
	Name     string
	Lang     string // ISO-639
	Script   string // ISO-15924
	Size     int
	Color    string `json:"col"`
	col      *color.SimpleColor
	FillFont bool
}

// ISO-639 country codes
// See https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes
var ISO639Codes = []string{"ab", "aa", "af", "ak", "sq", "am", "ar", "an", "hy", "as", "av", "ae", "ay", "az", "bm", "ba", "eu", "be", "bn", "bi", "bs", "br", "bg",
	"my", "ca", "ch", "ce", "ny", "zh", "cu", "cv", "kw", "co", "cr", "hr", "cs", "da", "dv", "nl", "dz", "en", "eo", "et", "ee", "fo", "fj", "fi", "fr", "fy", "ff",
	"gd", "gl", "lg", "ka", "de", "el", "kl", "gn", "gu", "ht", "ha", "he", "hz", "hi", "ho", "hu", "is", "io", "ig", "id", "ia", "ie", "iu", "ik", "ga", "it", "ja",
	"jv", "kn", "kr", "ks", "kk", "km", "ki", "rw", "ky", "kv", "kg", "ko", "kj", "ku", "lo", "la", "lv", "li", "ln", "lt", "lu", "lb", "mk", "mg", "ms", "ml", "mt",
	"gv", "mi", "mr", "mh", "mn", "na", "nv", "nd", "nr", "ng", "ne", "no", "nb", "nn", "ii", "oc", "oj", "or", "om", "os", "pi", "ps", "fa", "pl", "pt", "pa", "qu",
	"ro", "rm", "rn", "ru", "se", "sm", "sg", "sa", "sc", "sr", "sn", "sd", "si", "sk", "sl", "so", "st", "es", "su", "sw", "ss", "sv", "tl", "ty", "tg", "ta", "tt",
	"te", "th", "bo", "ti", "to", "ts", "tn", "tr", "tk", "tw", "ug", "uk", "ur", "uz", "ve", "vi", "vo", "wa", "cy", "wo", "xh", "yi", "yo", "za", "zu"}

func (f *FormFont) validateISO639() error {
	if !types.MemberOf(f.Lang, ISO639Codes) {
		return errors.Errorf("pdfcpu: invalid ISO-639 code: %s", f.Lang)
	}
	return nil
}

func (f *FormFont) validateScriptSupport() error {
	font.UserFontMetricsLock.RLock()
	fd, ok := font.UserFontMetrics[f.Name]
	font.UserFontMetricsLock.RUnlock()
	if !ok {
		return errors.Errorf("pdfcpu: userfont %s not available", f.Name)
	}
	ok, err := fd.SupportsScript(f.Script)
	if err != nil {
		return err
	}
	if !ok {
		return errors.Errorf("pdfcpu: userfont (%s) does not support script: %s", f.Name, f.Script)
	}
	return nil
}

func (f *FormFont) validate() error {
	if f.Name == "$" {
		return errors.New("pdfcpu: invalid font reference $")
	}

	if f.Name != "" && f.Name[0] != '$' {
		if !font.SupportedFont(f.Name) {
			return errors.Errorf("pdfcpu: font %s is unsupported, please refer to \"pdfcpu fonts list\".\n", f.Name)
		}
		if font.IsUserFont(f.Name) {
			if f.Lang != "" {
				f.Lang = strings.ToLower(f.Lang)
				if err := f.validateISO639(); err != nil {
					return err
				}
			}
			if f.Script != "" {
				f.Script = strings.ToUpper(f.Script)
				if err := f.validateScriptSupport(); err != nil {
					return err
				}
			}
		}
		if f.Size <= 0 {
			return errors.Errorf("pdfcpu: invalid font size: %d", f.Size)
		}
	}

	if f.Color != "" {
		sc, err := f.pdf.parseColor(f.Color)
		if err != nil {
			return err
		}
		f.col = sc
	}

	return nil
}

func (f *FormFont) mergeIn(f0 *FormFont) {
	if f.Name == "" {
		f.Name = f0.Name
	}
	if f.Size == 0 {
		f.Size = f0.Size
	}
	if f.col == nil {
		f.col = f0.col
	}
	if f.Lang == "" {
		f.Lang = f0.Lang
	}
	if f.Script == "" {
		f.Script = f0.Script
	}
}

func (f *FormFont) SetCol(c color.SimpleColor) {
	f.col = &c
}

func (f FormFont) RTL() bool {
	return types.MemberOf(f.Script, []string{"Arab", "Hebr"}) || types.MemberOf(f.Lang, []string{"ar", "fa", "he"})
}

func FormFontNameAndLangForID(xRefTable *model.XRefTable, indRef types.IndirectRef) (string, string, error) {

	objNr := int(indRef.ObjectNumber)
	fontDict, err := xRefTable.DereferenceDict(indRef)
	if err != nil || fontDict == nil {
		return "", "", err
	}

	_, fName, err := pdffont.Name(xRefTable, fontDict, objNr)
	if err != nil {
		return "", "", err
	}

	var fLang string
	if font.IsUserFont(fName) {
		fLang, err = pdffont.Lang(xRefTable, fontDict)
		if err != nil {
			return "", "", err
		}
	}

	return fName, fLang, nil
}

// FormFontResDict returns form dict's font resource dict.
func FormFontResDict(xRefTable *model.XRefTable) (types.Dict, error) {

	d := xRefTable.Form
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

func formFontIndRef(xRefTable *model.XRefTable, fontID string) *types.IndirectRef {

	indRef, ok := xRefTable.FillFonts[fontID]
	if ok {
		return &indRef
	}

	for k, v := range xRefTable.FillFonts {
		if strings.HasPrefix(k, fontID) || strings.HasPrefix(fontID, k) {
			return &v
		}
	}

	return nil
}

func FontIndRef(fName string, ctx *model.Context, fonts map[string]types.IndirectRef) (*types.IndirectRef, error) {

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

func ensureUTF8FormFont(ctx *model.Context, fonts map[string]types.IndirectRef) (string, string, string, *types.IndirectRef, error) {

	// TODO Make name of UTF-8 userfont part of pdfcpu configs.

	fontID, fontName := "F0", "Roboto-Regular"

	if indRef, ok := fonts[fontName]; ok {
		return fontID, fontName, "", &indRef, nil
	}

	for objNr, fo := range ctx.Optimize.FontObjects {
		if fo.FontName == fontName && fo.Prefix != "" {
			indRef := types.NewIndirectRef(objNr, 0)
			fonts[fontName] = *indRef
			return fontID, fontName, "", indRef, nil
		}
	}

	indRef, err := pdffont.EnsureFontDict(ctx.XRefTable, fontName, "", "", false, nil)
	if err != nil {
		return "", "", "", nil, err
	}
	fonts[fontName] = *indRef

	return fontID, fontName, "", indRef, nil
}

func extractFormFontDetails(
	ctx *model.Context,
	fontID string,
	fonts map[string]types.IndirectRef) (string, string, string, *types.IndirectRef, error) {

	xRefTable := ctx.XRefTable

	var (
		fName, fLang string
		fontIndRef   *types.IndirectRef
		err          error
	)

	if len(fontID) > 0 {

		fontIndRef = formFontIndRef(xRefTable, fontID)
		if fontIndRef != nil {
			fName, fLang, err = FormFontNameAndLangForID(xRefTable, *fontIndRef)
			if err != nil {
				return "", "", "", nil, err
			}

			if fName == "" {
				return "", "", "", nil, errors.Errorf("pdfcpu: Unable to detect fontName for: %s", fontID)
			}
		}

	}

	if fontIndRef == nil {
		return ensureUTF8FormFont(ctx, fonts)
	}

	return fontID, fName, fLang, fontIndRef, err
}

func fontFromDA(s string) (string, FormFont, error) {

	da := strings.Fields(s)

	var (
		f      FormFont
		fontID string
	)

	f.SetCol(color.Black)

	for i := 0; i < len(da); i++ {
		if da[i] == "Tf" {
			fontID = da[i-2][1:]
			//tf.SetFontID(fontID)
			fl, err := strconv.ParseFloat(da[i-1], 64)
			if err != nil {
				return fontID, f, err
			}
			if fl == 0 {
				// TODO derive size from acroDict DA and then use a default form font size (add to pdfcpu config)
				fl = 12
			}
			f.Size = int(fl)
			continue
		}
		if da[i] == "rg" {
			r, _ := strconv.ParseFloat(da[i-3], 32)
			g, _ := strconv.ParseFloat(da[i-2], 32)
			b, _ := strconv.ParseFloat(da[i-1], 32)
			f.SetCol(color.SimpleColor{R: float32(r), G: float32(g), B: float32(b)})
			continue
		}
		if da[i] == "g" {
			g, _ := strconv.ParseFloat(da[i-1], 32)
			f.SetCol(color.SimpleColor{R: float32(g), G: float32(g), B: float32(g)})
		}
	}

	return fontID, f, nil
}

func calcFontDetailsFromDA(ctx *model.Context, d types.Dict, da *string, needUTF8 bool, fonts map[string]types.IndirectRef) (string, *FormFont, bool, *types.IndirectRef, error) {
	s := locateDA(ctx, d, da)
	if s == nil {
		return "", nil, false, nil, errors.New("pdfcpu: missing \"DA\"")
	}

	fontID, f, err := fontFromDA(*s)
	if err != nil {
		return "", nil, false, nil, err
	}

	id, name, lang, fontIndRef, err := extractFormFontDetails(ctx, fontID, fonts)
	if err != nil {
		return "", nil, false, nil, err
	}
	if fontIndRef == nil {
		return "", nil, false, nil, errors.New("pdfcpu: unable to detect indirect reference for font")
	}

	fillFont := formFontIndRef(ctx.XRefTable, fontID) != nil

	if needUTF8 && font.IsCoreFont(name) {
		id, name, lang, fontIndRef, err = ensureUTF8FormFont(ctx, fonts)
		if err != nil {
			return "", nil, false, nil, err
		}
	}

	f.Name = name
	f.Lang = lang
	f.FillFont = fillFont

	rtl := pdffont.RTL(lang)

	return id, &f, rtl, fontIndRef, nil
}
