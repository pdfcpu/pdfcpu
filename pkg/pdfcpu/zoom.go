/*
Copyright 2024 The pdfcpu Authors.

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
	"fmt"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// ParseZoomConfig parses a Zoom command string into an internal structure.
func ParseZoomConfig(s string, u types.DisplayUnit) (*model.Zoom, error) {

	if s == "" {
		return nil, errors.New("pdfcpu: missing zoom configuration string")
	}

	zoom := &model.Zoom{Unit: u}

	ss := strings.Split(s, ",")

	for _, s := range ss {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, errors.New("pdfcpu: Invalid zoom configuration string. Please consult pdfcpu help zoom")
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := model.ZoomParamMap.Handle(paramPrefix, paramValueStr, zoom); err != nil {
			return nil, err
		}
	}

	if zoom.Factor != 0 && (zoom.HMargin != 0 || zoom.VMargin != 0) {
		return nil, errors.New("pdfcpu: please supply either zoom \"factor\" or \"hmargin\" or \"vmargin\"")
	}

	return zoom, nil
}

func handleZoomOutBgColAndBorder(cropBox *types.Rectangle, bb *[]byte, zoom *model.Zoom) {
	if zoom.Factor < 1 && (zoom.BgColor != nil || zoom.Border) {

		var buf bytes.Buffer

		if zoom.BgColor != nil {
			draw.FillRectNoBorder(&buf, types.RectForWidthAndHeight(cropBox.LL.X, cropBox.LL.Y, cropBox.Width(), zoom.VMargin), *zoom.BgColor)
			draw.FillRectNoBorder(&buf, types.RectForWidthAndHeight(cropBox.LL.X, cropBox.Height()-zoom.VMargin, cropBox.Width(), zoom.VMargin), *zoom.BgColor)
			draw.FillRectNoBorder(&buf, types.RectForWidthAndHeight(cropBox.LL.X, zoom.VMargin, zoom.HMargin, cropBox.Height()-2*zoom.VMargin), *zoom.BgColor)
			draw.FillRectNoBorder(&buf, types.RectForWidthAndHeight(cropBox.Width()-zoom.HMargin, zoom.VMargin, zoom.HMargin, cropBox.Height()-2*zoom.VMargin), *zoom.BgColor)
		}

		if zoom.Border {
			r := types.RectForWidthAndHeight(
				cropBox.LL.X+zoom.HMargin,
				cropBox.LL.Y+zoom.VMargin,
				cropBox.Width()-2*zoom.HMargin,
				cropBox.Height()-2*zoom.VMargin)
			draw.DrawRect(&buf, r, 1, &color.Black, nil)
		}

		*bb = append(*bb, buf.Bytes()...)
	}
}

func zoomPage(ctx *model.Context, pageNr int, zoom *model.Zoom) error {
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

	if err := zoom.EnsureFactorAndMargins(cropBox.Width(), cropBox.Height()); err != nil {
		return err
	}

	sc := zoom.Factor
	sin, cos := 0., 1.
	dx := zoom.HMargin
	dy := zoom.VMargin

	m := matrix.CalcTransformMatrix(sc, sc, sin, cos, dx, dy)

	var trans bytes.Buffer
	fmt.Fprintf(&trans, "q %.5f %.5f %.5f %.5f %.5f %.5f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	bb, err := ctx.PageContent(d)
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
	bb = append(bb, []byte(" Q ")...)

	handleZoomOutBgColAndBorder(cropBox, &bb, zoom)

	sd, _ := ctx.NewStreamDictForBuf(bb)
	if err := sd.Encode(); err != nil {
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

func Zoom(ctx *model.Context, selectedPages types.IntSet, zoom *model.Zoom) error {
	if log.DebugEnabled() {
		log.Debug.Printf("Zoom:\n%s\n", zoom)
	}

	if len(selectedPages) == 0 {
		selectedPages = types.IntSet{}
		for i := 1; i <= ctx.PageCount; i++ {
			selectedPages[i] = true
		}
	}

	for k, v := range selectedPages {
		if v {
			if err := zoomPage(ctx, k, zoom); err != nil {
				return err
			}
		}
	}

	ctx.EnsureVersionForWriting()

	return nil
}
