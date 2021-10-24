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

package pdfcpu

import (
	"strings"

	pdffont "github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pkg/errors"
)

func ParseHorAlignment(s string) (HAlignment, error) {
	var a HAlignment
	switch strings.ToLower(s) {
	case "l", "left":
		a = AlignLeft
	case "r", "right":
		a = AlignRight
	case "c", "center":
		a = AlignCenter
	default:
		return a, errors.Errorf("pdfcpu: unknown textfield alignment (left, center, right): %s", s)
	}
	return a, nil
}

// RelPosition represents the relative position of a text field's label.
type RelPosition int

// These are the options for relative label positions.
const (
	RelPosLeft RelPosition = iota
	RelPosRight
	RelPosTop
	RelPosBottom
)

func ParseRelPosition(s string) (RelPosition, error) {
	var p RelPosition
	switch strings.ToLower(s) {
	case "l", "left":
		p = RelPosLeft
	case "r", "right":
		p = RelPosRight
	case "t", "top":
		p = RelPosTop
	case "b", "bottom":
		p = RelPosBottom
	default:
		return p, errors.Errorf("pdfcpu: unknown textfield alignment (left, right, top, bottom): %s", s)
	}
	return p, nil
}

// Refactor because of orientation in nup.go
type Orientation int

const (
	Horizontal Orientation = iota
	Vertical
)

func ParseRadioButtonOrientation(s string) (Orientation, error) {
	var o Orientation
	switch strings.ToLower(s) {
	case "h", "hor", "horizontal":
		o = Horizontal
	case "v", "vert", "vertical":
		o = Vertical
	default:
		return o, errors.Errorf("pdfcpu: unknown radiobutton orientation (hor, vert): %s", s)
	}
	return o, nil
}

func AnchorPosition(a Anchor, r *Rectangle, w, h float64) (x float64, y float64) {
	switch a {
	case TopLeft:
		x, y = 0, r.Height()-h
	case TopCenter:
		x, y = r.Width()/2-w/2, r.Height()-h
	case TopRight:
		x, y = r.Width()-w, r.Height()-h
	case Left:
		x, y = 0, r.Height()/2-h/2
	case Center:
		x, y = r.Width()/2-w/2, r.Height()/2-h/2
	case Right:
		x, y = r.Width()-w, r.Height()/2-h/2
	case BottomLeft:
		x, y = 0, 0
	case BottomCenter:
		x, y = r.Width()/2-w/2, 0
	case BottomRight:
		x, y = r.Width()-w, 0
	}
	return
}

// NormalizeCoord transfers P(x,y) from pdfcpu user space into PDF user space,
// which uses a coordinate system with origin in the lower left corner of r.
//
// pdfcpu user space coordinate systems have the origin in one of four corners of r:
//
// LowerLeft corner (default = PDF user space)
//		x extends to the right,
//		y extends upward
// LowerRight corner:
//		x extends to the left,
//		y extends upward
// UpperLeft corner:
//		x extends to the right,
//		y extends downward
// UpperRight corner:
//		x extends to the left,
//		y extends downward
func NormalizeCoord(x, y float64, r *Rectangle, origin Corner, absolute bool) (float64, float64) {
	switch origin {
	case UpperLeft:
		if y >= 0 {
			y = r.Height() - y
			if y < 0 {
				y = 0
			}
		}
	case LowerRight:
		if x >= 0 {
			x = r.Width() - x
			if x < 0 {
				x = 0
			}
		}
	case UpperRight:
		if x >= 0 {
			x = r.Width() - x
			if x < 0 {
				x = 0
			}
		}
		if y >= 0 {
			y = r.Height() - y
			if y < 0 {
				y = 0
			}
		}
	}
	if absolute {
		if x >= 0 {
			x += r.LL.X
		}
		if y >= 0 {
			y += r.LL.Y
		}
	}
	return x, y
}

// Normalize offset transfers x and y into offsets in the PDF user space.
func NormalizeOffset(x, y float64, origin Corner) (float64, float64) {
	switch origin {
	case UpperLeft:
		y = -y
	case LowerRight:
		x = -x
	case UpperRight:
		x = -x
		y = -y
	}
	return x, y
}

func CreatePage(
	xRefTable *XRefTable,
	parentPageIndRef IndirectRef,
	p Page,
	fonts map[string]IndirectRef,
	fields *Array,
	formFontIDs map[string]string) (*IndirectRef, Dict, error) {

	pageDict := Dict(
		map[string]Object{
			"Type":     Name("Page"),
			"Parent":   parentPageIndRef,
			"MediaBox": p.MediaBox.Array(),
			"CropBox":  p.CropBox.Array(),
		},
	)

	// Populate font resources

	fontRes := Dict{}
	for fontName, font := range p.Fm {
		if font.Res.IndRef != nil {
			fonts[fontName] = *font.Res.IndRef
		}
		indRef, ok := fonts[fontName]
		if !ok {
			_, ok := formFontIDs[fontName]
			ir, err := EnsureFontDict(xRefTable, fontName, !ok, nil)
			if err != nil {
				return nil, pageDict, err
			}
			indRef = *ir
			fonts[fontName] = indRef
		}
		fontRes[font.Res.ID] = indRef
	}

	// Populate image resources

	imgRes := Dict{}
	for _, img := range p.Im {
		imgRes[img.Res.ID] = *img.Res.IndRef
	}

	if len(fontRes) > 0 || len(imgRes) > 0 {

		resDict := Dict{}
		//"ProcSet": NewNameArray("PDF", "Text", "ImageB", "ImageC", "ImageI"),

		if len(fontRes) > 0 {
			resDict["Font"] = fontRes
		}

		if len(imgRes) > 0 {
			resDict["XObject"] = imgRes
		}

		pageDict["Resources"] = resDict
	}

	ir, err := xRefTable.streamDictIndRef(p.Buf.Bytes())
	if err != nil {
		return nil, pageDict, err
	}

	pageDict.Insert("Contents", *ir)
	pageDictIndRef, err := xRefTable.IndRefForNewObject(pageDict)

	if len(p.AnnotIndRefs) > 0 || len(p.Annots) > 0 || len(p.LinkAnnots) > 0 {

		a := p.AnnotIndRefs

		for _, ann := range p.Annots {
			ann["P"] = *pageDictIndRef
			if _, ok := ann["Parent"]; ok {
				continue
			}
			ir, err := xRefTable.IndRefForNewObject(ann)
			if err != nil {
				return nil, nil, err
			}
			a = append(a, *ir)
			*fields = append(*fields, *ir)
		}

		for _, ar := range p.LinkAnnots {
			d := ar.RenderDict(*pageDictIndRef)
			ir, err := xRefTable.IndRefForNewObject(d)
			if err != nil {
				return nil, nil, err
			}
			a = append(a, *ir)
		}

		pageDict["Annots"] = a
	}

	return pageDictIndRef, pageDict, err
}

func updateUserfont(xRefTable *XRefTable, fontName string, font FontResource) error {

	ttf, ok := pdffont.UserFontMetrics[fontName]
	if !ok {
		return errors.Errorf("pdfcpu: userfont %s not available", fontName)
	}

	usedGIDs, err := usedGIDsFromCMapIndRef(xRefTable, *font.ToUnicode)
	if err != nil {
		return err
	}

	for _, gid := range usedGIDs {
		ttf.UsedGIDs[gid] = true
	}

	if _, err = toUnicodeCMap(xRefTable, ttf, true, font.ToUnicode); err != nil {
		return err
	}

	if _, err = ttfSubFontFile(xRefTable, ttf, fontName, font.FontFile); err != nil {
		return err
	}

	if _, err = CIDWidths(xRefTable, ttf, true, font.W); err != nil {
		return err
	}

	if _, err := CIDSet(xRefTable, ttf, font.CIDSet); err != nil {
		return err
	}

	return nil
}

func ModifyPageContent(
	xRefTable *XRefTable,
	dIndRef IndirectRef,
	d, res Dict,
	p Page,
	fonts map[string]IndirectRef,
	fields *Array,
	formFontIDs map[string]string) error {

	// TODO Account for existing page rotation.

	// Populate font resources
	if len(p.Fm) > 0 {
		fontRes, ok := res["Font"].(Dict)
		if !ok {
			fontRes = Dict{}
		}
		for fontName, font := range p.Fm {
			indRef, ok := fonts[fontName]
			if ok {
				fontRes.Insert(font.Res.ID, indRef)
				continue
			}
			if font.Res.IndRef == nil {
				// Create corefont or userfont.
				_, ok := formFontIDs[fontName]
				indRef, err := EnsureFontDict(xRefTable, fontName, !ok, nil)
				if err != nil {
					return err
				}
				fonts[fontName] = *indRef
				fontRes.Insert(font.Res.ID, *indRef)
				continue
			}
			if pdffont.IsCoreFont(fontName) || font.FontFile == nil {
				// Reuse font.
				fonts[fontName] = *font.Res.IndRef
				fontRes.Insert(font.Res.ID, *font.Res.IndRef)
				continue
			}
			if err := updateUserfont(xRefTable, fontName, font); err != nil {
				return err
			}
			fonts[fontName] = *font.Res.IndRef
			fontRes.Insert(font.Res.ID, *font.Res.IndRef)
		}
		res["Font"] = fontRes
	}

	// 	Populate image resources
	if len(p.Im) > 0 {
		imgRes, ok := res["XObject"].(Dict)
		if !ok {
			imgRes = Dict{}
		}
		for _, img := range p.Im {
			imgRes.Insert(img.Res.ID, *img.Res.IndRef)
		}
		res["XObject"] = imgRes
	}

	if len(p.Fm) > 0 || len(p.Im) > 0 {
		d["Resources"] = res
	}

	// Append to content stream
	if err := xRefTable.AppendContent(d, p.Buf.Bytes()); err != nil {
		return err
	}

	// Append annotations
	if len(p.AnnotIndRefs) > 0 || len(p.Annots) > 0 || len(p.LinkAnnots) > 0 {
		a, err := xRefTable.DereferenceArray(d["Annots"])
		if err != nil {
			return err
		}
		a = append(a, p.AnnotIndRefs...)
		for _, ann := range p.Annots {
			ann["P"] = dIndRef
			if _, ok := ann["Parent"]; ok {
				// takes effect in xreftable entry?
				continue
			}
			ir, err := xRefTable.IndRefForNewObject(ann)
			if err != nil {
				return err
			}
			a = append(a, *ir)
			*fields = append(*fields, *ir)
		}

		for _, ar := range p.LinkAnnots {
			d := ar.RenderDict(dIndRef)
			ir, err := xRefTable.IndRefForNewObject(d)
			if err != nil {
				return err
			}
			a = append(a, *ir)
		}

		d["Annots"] = a
	}

	return nil
}
