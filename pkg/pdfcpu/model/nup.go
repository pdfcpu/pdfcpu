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

package model

import (
	"bytes"
	"fmt"
	"io"
	"math"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

type orientation int

func (o orientation) String() string {
	switch o {

	case RightDown:
		return "right down"

	case DownRight:
		return "down right"

	case LeftDown:
		return "left down"

	case DownLeft:
		return "down left"

	}

	return ""
}

// These are the defined anchors for relative positioning.
const (
	RightDown orientation = iota
	DownRight
	LeftDown
	DownLeft
)

// NUp represents the command details for the command "NUp".
type NUp struct {
	PageDim       *types.Dim         // Page dimensions in display unit.
	PageSize      string             // Paper size eg. A4L, A4P, A4(=default=A4P), see paperSize.go
	UserDim       bool               // true if one of dimensions or paperSize provided overriding the default.
	Orient        orientation        // One of rd(=default),dr,ld,dl
	Grid          *types.Dim         // Intra page grid dimensions eg (2,2)
	PageGrid      bool               // Create a m x n grid of pages for PDF inputfiles only (think "extra page n-Up").
	ImgInputFile  bool               // Process image or PDF input files.
	Margin        float64            // Cropbox for n-Up content.
	Border        bool               // Draw bounding box.
	BookletGuides bool               // Draw folding and cutting lines.
	MultiFolio    bool               // Render booklet as sequence of folios.
	FolioSize     int                // Booklet multifolio folio size: default: 8
	InpUnit       types.DisplayUnit  // input display unit.
	BgColor       *color.SimpleColor // background color
}

// DefaultNUpConfig returns the default NUp configuration.
func DefaultNUpConfig() *NUp {
	return &NUp{
		PageSize: "A4",
		Orient:   RightDown,
		Margin:   3,
		Border:   true,
	}
}

func (nup NUp) String() string {
	return fmt.Sprintf("N-Up conf: %s %s, orient=%s, grid=%s, pageGrid=%t, isImage=%t\n",
		nup.PageSize, *nup.PageDim, nup.Orient, *nup.Grid, nup.PageGrid, nup.ImgInputFile)
}

// N returns the nUp value.
func (nup NUp) N() int {
	return int(nup.Grid.Height * nup.Grid.Width)
}

// RectsForGrid calculates dest rectangles for given grid.
func (nup NUp) RectsForGrid() []*types.Rectangle {
	cols := int(nup.Grid.Width)
	rows := int(nup.Grid.Height)

	maxX := float64(nup.PageDim.Width)
	maxY := float64(nup.PageDim.Height)

	gw := maxX / float64(cols)
	gh := maxY / float64(rows)

	var llx, lly float64
	rr := []*types.Rectangle{}

	switch nup.Orient {

	case RightDown:
		for i := rows - 1; i >= 0; i-- {
			for j := 0; j < cols; j++ {
				llx = float64(j) * gw
				lly = float64(i) * gh
				rr = append(rr, types.NewRectangle(llx, lly, llx+gw, lly+gh))
			}
		}

	case DownRight:
		for i := 0; i < cols; i++ {
			for j := rows - 1; j >= 0; j-- {
				llx = float64(i) * gw
				lly = float64(j) * gh
				rr = append(rr, types.NewRectangle(llx, lly, llx+gw, lly+gh))
			}
		}

	case LeftDown:
		for i := rows - 1; i >= 0; i-- {
			for j := cols - 1; j >= 0; j-- {
				llx = float64(j) * gw
				lly = float64(i) * gh
				rr = append(rr, types.NewRectangle(llx, lly, llx+gw, lly+gh))
			}
		}

	case DownLeft:
		for i := cols - 1; i >= 0; i-- {
			for j := rows - 1; j >= 0; j-- {
				llx = float64(i) * gw
				lly = float64(j) * gh
				rr = append(rr, types.NewRectangle(llx, lly, llx+gw, lly+gh))
			}
		}
	}

	return rr
}

func createNUpFormForPDF(xRefTable *XRefTable, resDict *types.IndirectRef, content []byte, cropBox *types.Rectangle) (*types.IndirectRef, error) {
	sd := types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":      types.Name("XObject"),
				"Subtype":   types.Name("Form"),
				"BBox":      cropBox.Array(),
				"Matrix":    types.NewNumberArray(1, 0, 0, 1, -cropBox.LL.X, -cropBox.LL.Y),
				"Resources": *resDict,
			},
		),
		Content:        content,
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate, DecodeParms: nil}},
	}

	sd.InsertName("Filter", filter.Flate)

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(sd)
}

// NUpTilePDFBytesForPDF applies nup tiles to content bytes.
func NUpTilePDFBytes(wr io.Writer, rSrc, rDest *types.Rectangle, formResID string, nup *NUp, rotate, enforceOrient bool) {

	// rScr is a rectangular region represented by form formResID in form space.

	// rDest is an arbitrary rectangular region in dest space.
	// It is the location where we want the form content to get rendered on a "best fit" basis.
	// Accounting for the aspect ratios of rSrc and rDest "best fit" tries to fit the largest version of rScr into rDest.
	// This may result in a 90 degree rotation.
	//
	// rotate:
	//			indicates if we need to apply a post rotation of 180 degrees eg for booklets.
	//
	// enforceOrient:
	//			indicates if we need to enforce dest's orientation.

	// Draw bounding box.
	if nup.Border {
		fmt.Fprintf(wr, "[]0 d 0.1 w %.2f %.2f m %.2f %.2f l %.2f %.2f l %.2f %.2f l s ",
			rDest.LL.X, rDest.LL.Y, rDest.UR.X, rDest.LL.Y, rDest.UR.X, rDest.UR.Y, rDest.LL.X, rDest.UR.Y,
		)
	}

	// Apply margin to rDest which potentially makes it smaller.
	rDestCr := rDest.CroppedCopy(nup.Margin)

	// Calculate transform matrix.

	// Best fit translation of a source rectangle into a destination rectangle.
	// For nup we enforce the dest orientation,
	// whereas in cases where the original orientation needs to be preserved eg. for booklets, we don't.
	w, h, dx, dy, r := types.BestFitRectIntoRect(rSrc, rDestCr, enforceOrient, false)

	if nup.BgColor != nil {
		if nup.ImgInputFile {
			// Fill background.
			draw.FillRectNoBorder(wr, rDest, *nup.BgColor)
		} else if nup.Margin > 0 {
			// Fill margins.
			m := nup.Margin
			DrawMargins(wr, *nup.BgColor, rDest, 0, m, m, m, m)
		}
	}

	// Apply additional rotation.
	if rotate {
		r += 180
	}

	sx := w
	sy := h
	if !nup.ImgInputFile {
		sx /= rSrc.Width()
		sy /= rSrc.Height()
	}

	sin := math.Sin(r * float64(matrix.DegToRad))
	cos := math.Cos(r * float64(matrix.DegToRad))

	switch r {
	case 90:
		dx += h
	case 180:
		dx += w
		dy += h
	case 270:
		dy += w
	}

	dx += rDestCr.LL.X
	dy += rDestCr.LL.Y

	m := matrix.CalcTransformMatrix(sx, sy, sin, cos, dx, dy)

	// Apply transform matrix and display form.
	fmt.Fprintf(wr, "q %.5f %.5f %.5f %.5f %.5f %.5f cm /%s Do Q ",
		m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1], formResID)
}

func translationForPageRotation(pageRot int, w, h float64) (float64, float64) {
	var dx, dy float64

	switch pageRot {
	case 90, -270:
		dy = h
	case -90, 270:
		dx = w
	case 180, -180:
		dx, dy = w, h
	}

	return dx, dy
}

// ContentBytesForPageRotation returns content bytes compensating for rot.
func ContentBytesForPageRotation(rot int, w, h float64) []byte {
	dx, dy := translationForPageRotation(rot, w, h)
	// Note: PDF rotation is clockwise!
	m := matrix.CalcRotateAndTranslateTransformMatrix(float64(-rot), dx, dy)
	var b bytes.Buffer
	fmt.Fprintf(&b, "%.5f %.5f %.5f %.5f %.5f %.5f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])
	return b.Bytes()
}

// NUpTilePDFBytesForPDF applies nup tiles from PDF.
func (ctx *Context) NUpTilePDFBytesForPDF(
	pageNr int,
	formsResDict types.Dict,
	buf *bytes.Buffer,
	rDest *types.Rectangle,
	nup *NUp,
	rotate bool) error {

	consolidateRes := true
	d, _, inhPAttrs, err := ctx.PageDict(pageNr, consolidateRes)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.Errorf("pdfcpu: unknown page number: %d\n", pageNr)
	}

	// Retrieve content stream bytes.
	bb, err := ctx.PageContent(d)
	if err == ErrNoContent {
		// TODO render if has annotations.
		return nil
	}
	if err != nil {
		return err
	}

	// Create an object for this resDict in xRefTable.
	ir, err := ctx.IndRefForNewObject(inhPAttrs.Resources)
	if err != nil {
		return err
	}

	cropBox := inhPAttrs.MediaBox
	if inhPAttrs.CropBox != nil {
		cropBox = inhPAttrs.CropBox
	}

	// Account for existing rotation.
	if inhPAttrs.Rotate != 0 {
		if types.IntMemberOf(inhPAttrs.Rotate, []int{+90, -90, +270, -270}) {
			w := cropBox.Width()
			cropBox.UR.X = cropBox.LL.X + cropBox.Height()
			cropBox.UR.Y = cropBox.LL.Y + w
		}
		bb = append(ContentBytesForPageRotation(inhPAttrs.Rotate, cropBox.Width(), cropBox.Height()), bb...)
	}

	formIndRef, err := createNUpFormForPDF(ctx.XRefTable, ir, bb, cropBox)
	if err != nil {
		return err
	}

	formResID := fmt.Sprintf("Fm%d", pageNr)
	formsResDict.Insert(formResID, *formIndRef)

	// Append to content stream buf of destination page.
	NUpTilePDFBytes(buf, cropBox, rDest, formResID, nup, rotate, true)

	return nil
}

// AppendPageTree appends a pagetree d1 to page tree d2.
func AppendPageTree(d1 *types.IndirectRef, countd1 int, d2 types.Dict) error {
	a := d2.ArrayEntry("Kids")
	a = append(a, *d1)
	d2.Update("Kids", a)
	return d2.IncrementBy("Count", countd1)
}
