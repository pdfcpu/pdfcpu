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
	"math"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ParseResizeConfig parses a Resize command string into an internal structure.
// "scale:.5, form:A4, dim:400 200 bgcol:#D00000"
func ParseResizeConfig(s string, u types.DisplayUnit) (*model.Resize, error) {

	if s == "" {
		return nil, errors.New("pdfcpu: missing resize configuration string")
	}

	res := &model.Resize{Unit: u}

	for s := range strings.SplitSeq(s, ",") {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, errors.New("pdfcpu: Invalid resize configuration string. Please consult pdfcpu help resize")
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := handleParameter(model.ResizeParamMap, paramPrefix, paramValueStr, res); err != nil {
			return nil, err
		}
	}

	if res.Scale > 0 && res.PageSize != "" {
		return nil, errors.New("pdfcpu: resize - please supply either scale factor or dimensions or form size ")
	}

	if res.UserDim && res.PageSize != "" {
		return nil, errors.New("pdfcpu: resize - please supply either dimensions or form size ")
	}

	return res, nil
}

func prepTransform(rSrc, rDest *types.Rectangle, enforce bool) (float64, float64, float64, float64, float64) {

	if !enforce && (rSrc.Portrait() && rDest.Landscape()) || (rSrc.Landscape() && rDest.Portrait()) {
		w1 := rDest.Width()
		rDest.UR.X = rDest.LL.X + rDest.Height()
		rDest.UR.Y = rDest.LL.Y + w1
	}

	w, h, dx, dy, rot := types.BestFitRectIntoRect(rSrc, rDest, enforce, true)

	sc := w / rSrc.Width()

	sin := math.Sin(rot * float64(model.DegToRad))
	cos := math.Cos(rot * float64(model.DegToRad))

	if rot == 90 {
		dx += h
	}

	dx += rDest.LL.X
	dy += rDest.LL.Y

	return sc, sin, cos, dx, dy
}

func prepResize(res *model.Resize, cropBox *types.Rectangle) (*types.Rectangle, float64, float64, float64, float64, float64) {
	ar := cropBox.AspectRatio()

	var (
		sc, dx, dy float64
		r          *types.Rectangle
	)

	sin, cos := 0., 1.

	if res.Scale > 0 {
		sc = res.Scale
	} else {
		if res.PageDim != nil {
			w := res.PageDim.Width
			h := res.PageDim.Height
			if w == 0 {
				sc = h / cropBox.Height()
				w = h * ar
				r = types.RectForDim(w, h)
			} else if h == 0 {
				sc = w / cropBox.Width()
				h = w / ar
				r = types.RectForDim(w, h)
			} else {
				r = types.RectForDim(w, h)
				sc, sin, cos, dx, dy = prepTransform(cropBox, r, res.EnforceOrientation())
			}
		}
	}

	return r, sc, sin, cos, dx, dy
}

func handleBgColAndBorder(dx, dy float64, cropBox *types.Rectangle, bb *[]byte, res *model.Resize) {
	if (dx > 0 || dy > 0) && (res.BgColor != nil || res.Border) {
		w, h := cropBox.Width(), cropBox.Height()
		if dx > 0 {
			w -= 2 * dx
		}
		if dy > 0 {
			h -= 2 * dy
		}
		r1 := types.RectForWidthAndHeight(dx, dy, w, h)
		var buf bytes.Buffer

		if res.BgColor != nil {
			draw.FillRectNoBorder(&buf, cropBox, *res.BgColor)
			draw.FillRectNoBorder(&buf, r1, color.White)
		}

		if res.Border {
			draw.DrawRect(&buf, r1, 1, &color.Black, nil)
		}

		*bb = append(buf.Bytes(), *bb...)
	}
}

func transformedRect(r *types.Rectangle, m matrix.Matrix) *types.Rectangle {
	p1 := m.Transform(types.Point{X: r.LL.X, Y: r.LL.Y})
	p2 := m.Transform(types.Point{X: r.UR.X, Y: r.LL.Y})
	p3 := m.Transform(types.Point{X: r.UR.X, Y: r.UR.Y})
	p4 := m.Transform(types.Point{X: r.LL.X, Y: r.UR.Y})
	return (types.QuadLiteral{P1: p1, P2: p2, P3: p3, P4: p4}).EnclosingRectangle(0)
}

func resizeAnnotationRect(ctx *model.Context, d types.Dict, m matrix.Matrix) error {
	obj, found := d.Find("Rect")
	if !found || obj == nil {
		return nil
	}
	arr, err := ctx.DereferenceArray(obj)
	if err != nil {
		return err
	}
	if len(arr) != 4 {
		return errors.New("pdfcpu: resize: invalid annotation rect")
	}

	r, err := ctx.RectForArray(arr)
	if err != nil {
		return err
	}

	d.Update("Rect", transformedRect(r, m).Array())
	return nil
}

func resizeAnnotationQuadPoints(ctx *model.Context, d types.Dict, m matrix.Matrix) error {
	obj, found := d.Find("QuadPoints")
	if !found || obj == nil {
		return nil
	}
	arr, err := ctx.DereferenceArray(obj)
	if err != nil {
		return err
	}
	if len(arr)%8 != 0 {
		return errors.New("pdfcpu: resize: invalid annotation quadpoints")
	}

	a := types.Array{}
	for i := 0; i < len(arr); i += 2 {
		x, err := ctx.DereferenceNumber(arr[i])
		if err != nil {
			return err
		}
		y, err := ctx.DereferenceNumber(arr[i+1])
		if err != nil {
			return err
		}
		p := m.Transform(types.Point{X: x, Y: y})
		a = append(a, types.Float(p.X), types.Float(p.Y))
	}

	d.Update("QuadPoints", a)
	return nil
}

func resizeAnnotation(ctx *model.Context, d types.Dict, m matrix.Matrix) error {
	if err := resizeAnnotationRect(ctx, d, m); err != nil {
		return err
	}
	return resizeAnnotationQuadPoints(ctx, d, m)
}

func resizePageAnnotations(ctx *model.Context, d types.Dict, m matrix.Matrix) error {
	obj, found := d.Find("Annots")
	if !found || obj == nil {
		return nil
	}
	arr, err := ctx.DereferenceArray(obj)
	if err != nil {
		return err
	}

	for _, o := range arr {
		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return err
		}
		if len(d) == 0 {
			continue
		}
		if err := resizeAnnotation(ctx, d, m); err != nil {
			return err
		}
	}

	return nil
}

func resizePage(ctx *model.Context, pageNr int, res *model.Resize) error {

	d, _, inhPAttrs, err := ctx.PageDict(pageNr, false)
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
	}

	r, sc, sin, cos, dx, dy := prepResize(res, cropBox)

	m := matrix.CalcTransformMatrix(sc, sc, sin, cos, dx, dy)

	var trans bytes.Buffer
	fmt.Fprintf(&trans, "q %.5f %.5f %.5f %.5f %.5f %.5f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	bb, err := ctx.PageContent(d, pageNr)
	if err == model.ErrNoContent {
		return nil
	}
	if err != nil {
		return err
	}

	if inhPAttrs.Rotate != 0 {
		bbInvRot := append([]byte(" q "), model.ContentBytesForPageRotation(inhPAttrs.Rotate, cropBox.Width(), cropBox.Height())...)
		bb = append(bbInvRot, bb...)
		bb = append(bb, []byte(" Q")...)
	}

	bb = append(trans.Bytes(), bb...)
	bb = append(bb, []byte(" Q")...)

	if res.Scale > 0 {
		cropBox.UR.X = cropBox.LL.X + sc*cropBox.Width()
		cropBox.UR.Y = cropBox.LL.Y + sc*cropBox.Height()
	} else {
		cropBox.UR.X = cropBox.LL.X + r.Width()
		cropBox.UR.Y = cropBox.LL.Y + r.Height()
	}

	handleBgColAndBorder(dx, dy, cropBox, &bb, res)

	sd, _ := ctx.NewStreamDictForBuf(bb)
	if err := sd.Encode(); err != nil {
		return err
	}

	if err := resizePageAnnotations(ctx, d, m); err != nil {
		return err
	}

	ir, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	d["Contents"] = *ir

	d.Update("MediaBox", cropBox.Array())
	d.Delete("Rotate")
	d.Delete("CropBox")

	return nil
}

// Resize resizes selectedPages using res.
func Resize(ctx *model.Context, selectedPages types.IntSet, res *model.Resize) error {
	if log.DebugEnabled() {
		log.Debug.Printf("Resize:\n%s\n", res)
	}

	if len(selectedPages) == 0 {
		selectedPages = types.IntSet{}
		for i := 1; i <= ctx.PageCount; i++ {
			selectedPages[i] = true
		}
	}

	for k, v := range selectedPages {
		if v {
			if err := resizePage(ctx, k, res); err != nil {
				return err
			}
		}
	}

	ctx.EnsureVersionForWriting()

	return nil
}
