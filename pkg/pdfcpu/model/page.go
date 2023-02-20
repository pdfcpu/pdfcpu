/*
	Copyright 2022 The pdfcpu Authors.

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

package model

import (
	"bytes"
	"strconv"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type Resource struct {
	ID     string
	IndRef *types.IndirectRef
}

// FontResource represents an existing PDF font resource.
type FontResource struct {
	Res       Resource
	Lang      string
	CIDSet    *types.IndirectRef
	FontFile  *types.IndirectRef
	ToUnicode *types.IndirectRef
	W         *types.IndirectRef
}

// FontMap maps font names to font resources.
type FontMap map[string]FontResource

// EnsureKey registers fontName with corresponding font resource id.
func (fm FontMap) EnsureKey(fontName string) string {
	// TODO userfontname prefix
	for k, v := range fm {
		if k == fontName {
			return v.Res.ID
		}
	}
	key := "F" + strconv.Itoa(len(fm))
	fm[fontName] = FontResource{Res: Resource{ID: key}}
	return key
}

// ImageResource represents an existing PDF image resource.
type ImageResource struct {
	Res    Resource
	Width  int
	Height int
}

// ImageMap maps image filenames to image resources.
type ImageMap map[string]ImageResource

type FieldAnnotation struct {
	Dict   types.Dict
	IndRef *types.IndirectRef
	Ind    int
	Field  bool
	Kids   types.Array
}

// Page represents rendered page content.
type Page struct {
	MediaBox   *types.Rectangle
	CropBox    *types.Rectangle
	Fm         FontMap
	Im         ImageMap
	Annots     []FieldAnnotation
	AnnotTabs  map[int]FieldAnnotation
	LinkAnnots []LinkAnnotation
	Buf        *bytes.Buffer
	Fields     types.Array
}

// NewPage creates a page for a mediaBox.
func NewPage(mediaBox *types.Rectangle) Page {
	return Page{MediaBox: mediaBox, Fm: FontMap{}, Im: ImageMap{}, Buf: new(bytes.Buffer)}
}

// NewPageWithBg creates a page for a mediaBox.
func NewPageWithBg(mediaBox *types.Rectangle, c color.SimpleColor) Page {
	p := Page{MediaBox: mediaBox, Fm: FontMap{}, Im: ImageMap{}, Buf: new(bytes.Buffer)}
	draw.FillRectNoBorder(p.Buf, mediaBox, c)
	return p
}
