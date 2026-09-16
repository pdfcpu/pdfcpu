/*
Copyright 2026 The pdfcpu Authors.

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
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/pdfcpu/pdfcpu/internal/corefont/metrics"
	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type simpleFontEncoding struct {
	codes   map[byte]byte
	missing map[byte]string
}

func encodingDifferences(xRefTable *model.XRefTable, d types.Dict) (map[byte]string, error) {
	o, found := d.Find("Differences")
	if !found {
		return nil, nil
	}
	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return nil, fmt.Errorf("Differences: %w", err)
	}
	differences := map[byte]string{}
	code := -1
	for i, o := range a {
		o, err = xRefTable.Dereference(o)
		if err != nil {
			return nil, fmt.Errorf("Differences[%d]: %w", i, err)
		}
		switch o := o.(type) {
		case types.Integer:
			code = o.Value()
			if code < 0 || code > 255 {
				return nil, fmt.Errorf("Differences[%d]: invalid character code %d", i, code)
			}
		case types.Name:
			if code < 0 || code > 255 {
				return nil, fmt.Errorf("Differences[%d]: missing character code", i)
			}
			differences[byte(code)] = o.Value()
			code++
		default:
			return nil, fmt.Errorf("Differences[%d]: expected integer or name, got %T", i, o)
		}
	}
	return differences, nil
}

func newSimpleFontEncoding(baseEncoding string, differences map[byte]string) *simpleFontEncoding {
	if baseEncoding != "WinAnsiEncoding" || len(differences) == 0 {
		return nil
	}
	effective := map[byte]string{}
	for code, glyphName := range metrics.WinAnsiGlyphMap {
		effective[byte(code)] = glyphName
	}
	for code, glyphName := range differences {
		effective[code] = glyphName
	}
	glyphCodes := map[string]byte{}
	for code := 0; code <= 255; code++ {
		glyphName := effective[byte(code)]
		if glyphName == "" {
			continue
		}
		if _, found := glyphCodes[glyphName]; !found {
			glyphCodes[glyphName] = byte(code)
		}
	}
	enc := &simpleFontEncoding{codes: map[byte]byte{}, missing: map[byte]string{}}
	for code := 0; code <= 255; code++ {
		glyphName := metrics.WinAnsiGlyphMap[code]
		if glyphName == "" || effective[byte(code)] == glyphName {
			continue
		}
		mappedCode, found := glyphCodes[glyphName]
		if !found {
			enc.missing[byte(code)] = glyphName
			continue
		}
		enc.codes[byte(code)] = mappedCode
	}
	return enc
}

func formFontEncoding(xRefTable *model.XRefTable, fontDict types.Dict) (string, *simpleFontEncoding, error) {
	o, found := fontDict.Find("Encoding")
	if !found {
		return "", nil, nil
	}
	objNr := 0
	if indRef, ok := o.(types.IndirectRef); ok {
		objNr = indRef.ObjectNumber.Value()
	}
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return "", nil, fmt.Errorf("entry=Encoding: %w", err)
	}
	switch o := o.(type) {
	case types.Name:
		return o.Value(), nil, nil
	case types.Dict:
		baseEncoding, _, err := xRefTable.DereferenceNameEntry(o, "BaseEncoding")
		if err != nil {
			return "", nil, fmt.Errorf("BaseEncoding: %w", err)
		}
		baseName := ""
		if baseEncoding != nil {
			baseName = baseEncoding.Value()
		}
		differences, err := encodingDifferences(xRefTable, o)
		if err != nil {
			return "", nil, err
		}
		return baseName, newSimpleFontEncoding(baseName, differences), nil
	default:
		err := fmt.Errorf("entry=Encoding: expected name or encoding dictionary, got %T", o)
		return "", nil, model.WithValidationErrorObject(err, objNr)
	}
}

func applyFormFontEncoding(xRefTable *model.XRefTable, f *FormFont, indRef *types.IndirectRef) error {
	if f == nil || indRef == nil || !f.FillFont || !font.IsCoreFont(f.Name) {
		return nil
	}
	fontDict, err := xRefTable.DereferenceDict(*indRef)
	if err != nil {
		return err
	}
	_, f.encoding, err = formFontEncoding(xRefTable, fontDict)
	return err
}

func (enc *simpleFontEncoding) encode(s string) (string, error) {
	bb := []byte(s)
	for i, code := range bb {
		if glyphName, missing := enc.missing[code]; missing {
			return "", fmt.Errorf("glyph %q unavailable in form font encoding", glyphName)
		}
		if mappedCode, found := enc.codes[code]; found {
			bb[i] = mappedCode
		}
	}
	return string(bb), nil
}

func (f *FormFont) prepareBytes(c context.Context, xRefTable *model.XRefTable, s string, embed, rtl bool) (string, error) {
	if font.IsCoreFont(f.Name) && utf8.ValidString(s) {
		s = model.DecodeUTF8ToByte(s)
	}
	if f.encoding != nil {
		var err error
		s, err = f.encoding.encode(s)
		if err != nil {
			return "", err
		}
	}
	return model.PrepBytes(c, xRefTable, s, f.Name, embed, rtl, f.FillFont)
}
