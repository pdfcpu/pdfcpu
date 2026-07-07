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

package pdfcpu

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ParseCutConfigForPoster parses a Cut command string into an internal structure.
// formsize(=papersize) or dimensions, optionally: scalefactor, border, margin, bgcolor
func ParseCutConfigForPoster(s string, u types.DisplayUnit) (*model.Cut, error) {
	if s == "" {
		return nil, errors.New("missing poster configuration string")
	}

	cut := &model.Cut{Unit: u, Scale: 1.0}

	for s := range strings.SplitSeq(s, ",") {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, errors.New("invalid poster configuration string")
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := handleParameter(model.CutParamMap, paramPrefix, paramValueStr, cut); err != nil {
			return nil, err
		}
	}

	return cut, nil
}

// ParseCutConfigForN parses a NDown command string into an internal structure.
// n, Optionally: border, margin, bgcolor
func ParseCutConfigForN(n int, s string, u types.DisplayUnit) (*model.Cut, error) {
	cut := &model.Cut{Unit: u}

	if !types.IntMemberOf(n, []int{2, 3, 4, 6, 8, 9, 12, 16}) {
		return nil, errors.New("invalid n: choose one of 2, 3, 4, 6, 8, 9, 12, 16")
	}

	if s == "" {
		return cut, nil
	}

	for s := range strings.SplitSeq(s, ",") {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, errors.New("invalid ndown configuration string")
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := handleParameter(model.CutParamMap, paramPrefix, paramValueStr, cut); err != nil {
			return nil, err
		}
	}

	return cut, nil
}

// ParseCutConfig parses a Cut command string into an internal structure.
// optionally: horizontalCut, verticalCut, bgcolor, border, margin, origin
func ParseCutConfig(s string, u types.DisplayUnit) (*model.Cut, error) {
	if s == "" {
		return nil, errors.New("missing cut configuration string")
	}

	cut := &model.Cut{Unit: u}

	for s := range strings.SplitSeq(s, ",") {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, errors.New("invalid cut configuration string")
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := handleParameter(model.CutParamMap, paramPrefix, paramValueStr, cut); err != nil {
			return nil, err
		}
	}

	return cut, nil
}

func drawOutlineCuts(w io.Writer, cropBox, cb *types.Rectangle, cut *model.Cut) {
	for i, f := range cut.Hor {
		if i == 0 {
			continue
		}
		y := cropBox.UR.Y - f*cropBox.Height()
		draw.DrawLineSimple(w, cb.LL.X, y, cb.UR.X, y)
	}

	for i, f := range cut.Vert {
		if i == 0 {
			continue
		}
		x := cropBox.LL.X + f*cropBox.Width()
		draw.DrawLineSimple(w, x, cb.LL.Y, x, cb.UR.Y)
	}
}

func cutPageContent(ctx *model.Context, d types.Dict, pageNr int) ([]byte, error) {
	bb, err := ctx.PageContent(d, pageNr)
	if err != nil {
		if errors.Is(err, model.ErrNoContent) {
			return nil, nil
		}
		return nil, fmt.Errorf("read page content: %w", err)
	}
	return bb, nil
}

func createOutline(
	ctxSrc, ctxDest *model.Context,
	pagesIndRef types.IndirectRef,
	pagesDict, d types.Dict,
	pageNr int,
	cropBox *types.Rectangle,
	migrated map[int]int,
	cut *model.Cut) error {
	cb := cropBox.Clone()

	var expCropBox bool
	if len(cut.Hor) > 0 && cut.Hor[len(cut.Hor)-1] > 1 {
		h := cut.Hor[len(cut.Hor)-1] * cropBox.Height()
		cb.LL.Y = cb.UR.Y - h
		expCropBox = true
	}

	if len(cut.Vert) > 0 && cut.Vert[len(cut.Vert)-1] > 1 {
		w := cut.Vert[len(cut.Vert)-1] * cropBox.Width()
		cb.UR.X = cb.LL.X + w
		expCropBox = true
	}

	d1 := d.Clone().(types.Dict)

	var buf bytes.Buffer

	fmt.Fprint(&buf, "[3] 0 d ")
	draw.SetStrokeColor(&buf, color.Red)

	// Assumption: origin = top left corner

	drawOutlineCuts(&buf, cropBox, cb, cut)

	bb, err := cutPageContent(ctxSrc, d1, pageNr)
	if err != nil {
		return err
	}

	bb = append([]byte("q "), bb...)
	bb = append(bb, []byte("Q ")...)
	bb = append(bb, buf.Bytes()...)

	sd, err := ctxSrc.NewStreamDictForBuf(bb)
	if err != nil {
		return fmt.Errorf("create content stream: %w", err)
	}
	if err := sd.Encode(); err != nil {
		return fmt.Errorf("encode content stream: %w", err)
	}

	indRef, err := ctxSrc.IndRefForNewObject(*sd)
	if err != nil {
		return fmt.Errorf("insert content stream: %w", err)
	}

	d1["Contents"] = *indRef
	d1["Parent"] = pagesIndRef
	if expCropBox {
		d1["MediaBox"] = cb.Array()
		d1["CropBox"] = cb.Array()
	}

	pageIndRef, err := ctxDest.IndRefForNewObject(d1)
	if err != nil {
		return fmt.Errorf("insert outline page: %w", err)
	}

	if err := ctxDest.SetValid(*pageIndRef); err != nil {
		return fmt.Errorf("outline page obj#%d: mark valid: %w", pageIndRef.ObjectNumber.Value(), err)
	}

	if err := migratePageDict(d1, *pageIndRef, ctxSrc, ctxDest, migrated); err != nil {
		return fmt.Errorf("outline page obj#%d: migrate dictionary: %w", pageIndRef.ObjectNumber.Value(), err)
	}

	if err := model.AppendPageTree(pageIndRef, 1, pagesDict); err != nil {
		return fmt.Errorf("outline page obj#%d: append page tree: %w", pageIndRef.ObjectNumber.Value(), err)
	}

	return nil
}

func prepForCut(ctxSrc *model.Context, pageNr int) (
	*model.Context,
	*types.Rectangle,
	*types.IndirectRef,
	types.Dict,
	types.Dict,
	*model.InheritedPageAttrs,
	error) {
	ctxDest, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("create destination context: %w", err)
	}

	pagesIndRef, err := ctxDest.Pages()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("access destination page tree: %w", err)
	}

	pagesDict, err := ctxDest.DereferenceDict(*pagesIndRef)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("destination page tree obj#%d: dereference dictionary: %w", pagesIndRef.ObjectNumber.Value(), err)
	}

	d, _, inhPAttrs, err := ctxSrc.PageDict(pageNr, false)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("source page dictionary: %w", err)
	}
	if d == nil {
		return nil, nil, nil, nil, nil, nil, errors.New("source page dictionary missing")
	}
	d = d.Clone().(types.Dict)
	d.Delete("Annots")

	cropBox := inhPAttrs.MediaBox.Clone()
	if inhPAttrs.CropBox != nil {
		cropBox = inhPAttrs.CropBox.Clone()
	}

	return ctxDest, cropBox, pagesIndRef, pagesDict, d, inhPAttrs, nil
}

func internPageRot(ctxSrc *model.Context, rotate int, cropBox *types.Rectangle, d types.Dict, pageNr int, trans []byte) error {
	bb, err := cutPageContent(ctxSrc, d, pageNr)
	if err != nil {
		return err
	}

	if rotate != 0 {
		bbInvRot := append([]byte(" q "), model.ContentBytesForPageRotation(rotate, cropBox.Width(), cropBox.Height())...)
		bb = append(bbInvRot, bb...)
		bb = append(bb, []byte(" Q ")...)
	}

	if len(trans) == 0 {
		trans = []byte("q ")
	}
	bb = append(trans, bb...)
	bb = append(bb, []byte("Q ")...)

	sd, err := ctxSrc.NewStreamDictForBuf(bb)
	if err != nil {
		return fmt.Errorf("create content stream: %w", err)
	}
	if err := sd.Encode(); err != nil {
		return fmt.Errorf("encode content stream: %w", err)
	}

	indRef, err := ctxSrc.IndRefForNewObject(*sd)
	if err != nil {
		return fmt.Errorf("insert content stream: %w", err)
	}

	d["Contents"] = *indRef

	return nil
}

func handleCutMargin(ctxSrc *model.Context, d, d1 types.Dict, pageNr int, cropBox, cb *types.Rectangle, i, j int, w, h float64, sc *float64, cut *model.Cut) error {
	ar := cb.AspectRatio()
	mv := cut.Margin / ar

	// Scale & translate content.
	if *sc == 0 {
		*sc = (cb.Width() - 2*cut.Margin) / cb.Width()
	}

	cbsc := cropBox.Clone()
	cbsc.UR.X = cbsc.LL.X + cbsc.Width()**sc
	cbsc.UR.Y = cbsc.LL.Y + cbsc.Height()**sc

	llx := cbsc.LL.X + cut.Vert[j]*cbsc.Width()

	lly := cbsc.LL.Y
	if i+1 < len(cut.Hor) {
		lly = cbsc.UR.Y - cut.Hor[i+1]*cbsc.Height()
	}

	cbb := types.RectForWidthAndHeight(llx, lly, w, h)

	d1["MediaBox"] = cbb.Array()
	d1["CropBox"] = cbb.Array()

	cb1 := cbb.Clone()
	cb1.LL.X += cut.Margin
	cb1.LL.Y += mv
	cb1.UR.X -= cut.Margin
	cb1.UR.Y -= mv

	var buf bytes.Buffer

	c := color.White
	if cut.BgColor != nil {
		c = *cut.BgColor
	}

	w, h = cb1.Width(), mv
	r := types.RectForWidthAndHeight(cb1.LL.X, cb1.UR.Y, w, h)
	draw.FillRectNoBorder(&buf, r, c)
	r = types.RectForWidthAndHeight(cb1.LL.X, cb1.LL.Y-mv, w, h)
	draw.FillRectNoBorder(&buf, r, c)

	w, h = cut.Margin, cbb.Height()
	r = types.RectForWidthAndHeight(cb1.UR.X, cb1.LL.Y-mv, w, h)
	draw.FillRectNoBorder(&buf, r, c)
	r = types.RectForWidthAndHeight(cb1.LL.X-cut.Margin, cb1.LL.Y-mv, w, h)
	draw.FillRectNoBorder(&buf, r, c)

	if cut.Border {
		draw.DrawRect(&buf, cb1, 1, &color.Black, nil)
	}

	m := matrix.CalcTransformMatrix(*sc, *sc, 0, 1, cut.Margin, mv)
	var trans bytes.Buffer
	fmt.Fprintf(&trans, "q %.5f %.5f %.5f %.5f %.5f %.5f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	bbOrig, err := cutPageContent(ctxSrc, d, pageNr)
	if err != nil {
		return err
	}

	bb := append(trans.Bytes(), bbOrig...)
	bb = append(bb, []byte(" Q ")...)
	bb = append(bb, buf.Bytes()...)

	sd, err := ctxSrc.NewStreamDictForBuf(bb)
	if err != nil {
		return fmt.Errorf("create content stream: %w", err)
	}
	if err := sd.Encode(); err != nil {
		return fmt.Errorf("encode content stream: %w", err)
	}

	indRef, err := ctxSrc.IndRefForNewObject(*sd)
	if err != nil {
		return fmt.Errorf("insert content stream: %w", err)
	}

	d1["Contents"] = *indRef

	return nil
}

func validateCutMargin(pageNr, row, column int, w, h, margin float64) error {
	if w <= 0 || h <= 0 {
		return fmt.Errorf("page %d tile row %d column %d: invalid tile dimensions %.2f x %.2f", pageNr, row, column, w, h)
	}
	verticalMargin := margin * h / w
	if 2*margin >= w || 2*verticalMargin >= h {
		return fmt.Errorf("page %d tile row %d column %d: margin %.2f does not fit tile dimensions %.2f x %.2f", pageNr, row, column, margin, w, h)
	}
	return nil
}

func createTiles(
	ctxSrc, ctxDest *model.Context,
	pagesIndRef types.IndirectRef,
	pagesDict, d types.Dict,
	pageNr int,
	cropBox *types.Rectangle,
	inhPAttrs *model.InheritedPageAttrs,
	migrated map[int]int,
	cut *model.Cut) error {
	var sc float64

	for i := 0; i < len(cut.Hor); i++ {
		ury := cropBox.UR.Y - cut.Hor[i]*cropBox.Height()
		if ury < cropBox.LL.Y {
			continue
		}
		lly := cropBox.LL.Y
		if i+1 < len(cut.Hor) {
			lly = cropBox.UR.Y - cut.Hor[i+1]*cropBox.Height()
		}

		h := ury - lly

		for j := 0; j < len(cut.Vert); j++ {
			llx := cropBox.LL.X + cut.Vert[j]*cropBox.Width()
			if llx > cropBox.UR.X {
				continue
			}
			urx := cropBox.UR.X
			if j+1 < len(cut.Vert) {
				urx = cropBox.LL.X + cut.Vert[j+1]*cropBox.Width()
			}
			w := urx - llx
			if cut.Margin > 0 {
				if err := validateCutMargin(pageNr, i+1, j+1, w, h, cut.Margin); err != nil {
					return err
				}
			}

			cb := types.NewRectangle(llx, lly, urx, ury)

			d1 := d.Clone().(types.Dict)
			d1["Resources"] = inhPAttrs.Resources.Clone()
			d1["Parent"] = pagesIndRef
			d1["MediaBox"] = cb.Array()
			d1["CropBox"] = cb.Array()

			if cut.Margin > 0 {
				if err := handleCutMargin(ctxSrc, d, d1, pageNr, cropBox, cb, i, j, w, h, &sc, cut); err != nil {
					return fmt.Errorf("tile row %d column %d: apply margin: %w", i+1, j+1, err)
				}
			}

			pageIndRef, err := ctxDest.IndRefForNewObject(d1)
			if err != nil {
				return fmt.Errorf("tile row %d column %d: insert page: %w", i+1, j+1, err)
			}

			if err := ctxDest.SetValid(*pageIndRef); err != nil {
				return fmt.Errorf("tile row %d column %d obj#%d: mark valid: %w", i+1, j+1, pageIndRef.ObjectNumber.Value(), err)
			}

			if err := migratePageDict(d1, *pageIndRef, ctxSrc, ctxDest, migrated); err != nil {
				return fmt.Errorf("tile row %d column %d obj#%d: migrate dictionary: %w", i+1, j+1, pageIndRef.ObjectNumber.Value(), err)
			}

			if err := model.AppendPageTree(pageIndRef, 1, pagesDict); err != nil {
				return fmt.Errorf("tile row %d column %d obj#%d: append page tree: %w", i+1, j+1, pageIndRef.ObjectNumber.Value(), err)
			}
		}
	}

	return nil
}

// CutPage cuts pageNr into tiles using horizontal or vertical cut points and optional styling.
func CutPage(ctxSrc *model.Context, pageNr int, cut *model.Cut) (*model.Context, error) {
	ctxDest, cropBox, pagesIndRef, pagesDict, d, inhPAttrs, err := prepForCut(ctxSrc, pageNr)
	if err != nil {
		return nil, fmt.Errorf("prepare page: %w", err)
	}

	rotate := inhPAttrs.Rotate

	if types.IntMemberOf(rotate, []int{+90, -90, +270, -270}) {
		w := cropBox.Width()
		cropBox.UR.X = cropBox.LL.X + cropBox.Height()
		cropBox.UR.Y = cropBox.LL.Y + w
		d["MediaBox"] = cropBox.Array()
		d["CropBox"] = cropBox.Array()
		d.Delete("Rotate")
	}

	if err := internPageRot(ctxSrc, rotate, cropBox, d, pageNr, nil); err != nil {
		return nil, fmt.Errorf("transform page content: %w", err)
	}

	migrated := map[int]int{}

	if err := createOutline(ctxSrc, ctxDest, *pagesIndRef, pagesDict, d, pageNr, cropBox, migrated, cut); err != nil {
		return nil, fmt.Errorf("create outline: %w", err)
	}

	if err := createTiles(ctxSrc, ctxDest, *pagesIndRef, pagesDict, d, pageNr, cropBox, inhPAttrs, migrated, cut); err != nil {
		return nil, fmt.Errorf("create tiles: %w", err)
	}

	return ctxDest, nil
}

func createNDownCuts(n int, cropBox *types.Rectangle) ([]float64, []float64, error) {
	var s1, s2 []float64

	switch n {
	case 2:
		s1 = append(s1, 0, .5)
		s2 = append(s2, 0)
	case 3:
		s1 = append(s1, 0, .33333, .66666)
		s2 = append(s2, 0)
	case 4:
		s1 = append(s1, 0, .5)
		s2 = append(s2, 0, .5)
	case 6:
		s1 = append(s1, 0, .33333, .66666)
		s2 = append(s2, 0, .5)
	case 8:
		s1 = append(s1, 0, .25, .5, .75)
		s2 = append(s2, 0, .5)
	case 9:
		s1 = append(s1, 0, .33333, .66666)
		s2 = append(s2, 0, .33333, .66666)
	case 12:
		s1 = append(s1, 0, .25, .5, .75)
		s2 = append(s2, 0, .33333, .66666)
	case 16:
		s1 = append(s1, 0, .25, .5, .75)
		s2 = append(s2, 0, .25, .5, .75)
	default:
		return nil, nil, errors.New("n-down value must be one of 2, 3, 4, 6, 8, 9, 12, 16")
	}

	if cropBox.Portrait() {
		return s1, s2, nil
	}
	return s2, s1, nil
}

// NDownPage creates n-down tiles for pageNr with optional styling.
func NDownPage(ctxSrc *model.Context, pageNr, n int, cut *model.Cut) (*model.Context, error) {
	ctxDest, cropBox, pagesIndRef, pagesDict, d, inhPAttrs, err := prepForCut(ctxSrc, pageNr)
	if err != nil {
		return nil, fmt.Errorf("prepare page: %w", err)
	}

	rotate := inhPAttrs.Rotate

	if types.IntMemberOf(rotate, []int{+90, -90, +270, -270}) {
		w := cropBox.Width()
		cropBox.UR.X = cropBox.LL.X + cropBox.Height()
		cropBox.UR.Y = cropBox.LL.Y + w
		d["MediaBox"] = cropBox.Array()
		d["CropBox"] = cropBox.Array()
		d.Delete("Rotate")
	}

	if err := internPageRot(ctxSrc, rotate, cropBox, d, pageNr, nil); err != nil {
		return nil, fmt.Errorf("transform page content: %w", err)
	}

	hor, vert, err := createNDownCuts(n, cropBox)
	if err != nil {
		return nil, fmt.Errorf("create cuts: %w", err)
	}
	cutLocal := *cut
	cutLocal.Hor = hor
	cutLocal.Vert = vert

	migrated := map[int]int{}

	if err := createOutline(ctxSrc, ctxDest, *pagesIndRef, pagesDict, d, pageNr, cropBox, migrated, &cutLocal); err != nil {
		return nil, fmt.Errorf("create outline: %w", err)
	}

	if err := createTiles(ctxSrc, ctxDest, *pagesIndRef, pagesDict, d, pageNr, cropBox, inhPAttrs, migrated, &cutLocal); err != nil {
		return nil, fmt.Errorf("create tiles: %w", err)
	}

	return ctxDest, nil
}

func createPosterCuts(cropBox *types.Rectangle, dim *types.Dim) ([]float64, []float64) {
	vert := []float64{0.}
	for x := 0.; ; x += dim.Width {
		f := (x + dim.Width) / cropBox.Width()
		fr := math.Round(f*100) / 100
		if fr != 1 {
			vert = append(vert, f)
		}
		if fr >= 1 {
			break
		}
	}

	hor := []float64{0.}
	for y := 0.; ; y += dim.Height {
		f := (y + dim.Height) / cropBox.Height()
		fr := math.Round(f*100) / 100
		if fr != 1 {
			hor = append(hor, f)
		}
		if fr >= 1 {
			break
		}
	}
	return hor, vert
}

// PosterPage creates poster tiles for pageNr using a form size or explicit dimensions.
func PosterPage(ctxSrc *model.Context, pageNr int, cut *model.Cut) (*model.Context, error) {
	ctxDest, cropBox, pagesIndRef, pagesDict, d, inhPAttrs, err := prepForCut(ctxSrc, pageNr)
	if err != nil {
		return nil, fmt.Errorf("prepare page: %w", err)
	}

	cropBox.UR.X = cropBox.LL.X + cropBox.Width()*cut.Scale
	cropBox.UR.Y = cropBox.LL.Y + cropBox.Height()*cut.Scale

	// Ensure cut.PageDim fits into scaled cropBox.
	dim := cut.PageDim
	if dim.Width > cropBox.Width() || dim.Height > cropBox.Height() {
		return nil, errors.New("selected poster tile dimensions too big")
	}

	rotate := inhPAttrs.Rotate

	if types.IntMemberOf(rotate, []int{+90, -90, +270, -270}) {
		w := cropBox.Width()
		cropBox.UR.X = cropBox.LL.X + cropBox.Height()
		cropBox.UR.Y = cropBox.LL.Y + w
	}

	d["MediaBox"] = cropBox.Array()
	d["CropBox"] = cropBox.Array()
	d.Delete("Rotate")

	// Scale transform
	m := matrix.IdentMatrix
	m[0][0] = cut.Scale
	m[1][1] = cut.Scale

	var trans bytes.Buffer
	fmt.Fprintf(&trans, "q %.5f %.5f %.5f %.5f %.5f %.5f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	if err := internPageRot(ctxSrc, rotate, cropBox, d, pageNr, trans.Bytes()); err != nil {
		return nil, fmt.Errorf("transform page content: %w", err)
	}

	hor, vert := createPosterCuts(cropBox, dim)
	cutLocal := *cut
	cutLocal.Hor = hor
	cutLocal.Vert = vert

	migrated := map[int]int{}

	if err := createOutline(ctxSrc, ctxDest, *pagesIndRef, pagesDict, d, pageNr, cropBox, migrated, &cutLocal); err != nil {
		return nil, fmt.Errorf("create outline: %w", err)
	}

	if err := createTiles(ctxSrc, ctxDest, *pagesIndRef, pagesDict, d, pageNr, cropBox, inhPAttrs, migrated, &cutLocal); err != nil {
		return nil, fmt.Errorf("create tiles: %w", err)
	}

	return ctxDest, nil
}
