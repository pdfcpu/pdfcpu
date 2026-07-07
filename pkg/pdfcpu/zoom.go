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
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ParseZoomConfig parses a Zoom command string into an internal structure.
func ParseZoomConfig(s string, u types.DisplayUnit) (*model.Zoom, error) {
	if s == "" {
		return nil, errors.New("missing zoom configuration string")
	}

	zoom := &model.Zoom{Unit: u}

	for i, item := range strings.Split(s, ",") {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf(`zoom configuration clause %d: missing ":" separator`, i+1)
		}

		paramPrefix := strings.TrimSpace(parts[0])
		if paramPrefix == "" {
			return nil, fmt.Errorf("zoom configuration clause %d: missing parameter name", i+1)
		}
		if strings.Contains(parts[1], ":") {
			return nil, fmt.Errorf(`zoom configuration clause %d: zoom parameter %q: too many ":" separators`, i+1, paramPrefix)
		}
		paramValueStr := strings.TrimSpace(parts[1])
		if paramValueStr == "" {
			return nil, fmt.Errorf("zoom configuration clause %d: missing parameter value", i+1)
		}

		if err := handleParameter(model.ZoomParamMap, paramPrefix, paramValueStr, zoom); err != nil {
			return nil, fmt.Errorf("zoom configuration clause %d: zoom parameter %q: %w", i+1, paramPrefix, err)
		}
	}

	if zoom.Factor != 0 && (zoom.HMargin != 0 || zoom.VMargin != 0) {
		return nil, errors.New("zoom: factor conflicts with margins")
	}

	return zoom, nil
}

func handleZoomOutBgColAndBorder(cropBox *types.Rectangle, bb *[]byte, zoom *model.Zoom) {
	if zoom.Factor < 1 && (zoom.BgColor != nil || zoom.Border) {

		var buf bytes.Buffer

		if zoom.BgColor != nil {
			llx, lly := cropBox.LL.X, cropBox.LL.Y
			w, h := cropBox.Width(), cropBox.Height()
			draw.FillRectNoBorder(&buf, types.RectForWidthAndHeight(llx, lly, w, zoom.VMargin), *zoom.BgColor)
			draw.FillRectNoBorder(&buf, types.RectForWidthAndHeight(llx, lly+h-zoom.VMargin, w, zoom.VMargin), *zoom.BgColor)
			draw.FillRectNoBorder(&buf, types.RectForWidthAndHeight(llx, lly+zoom.VMargin, zoom.HMargin, h-2*zoom.VMargin), *zoom.BgColor)
			draw.FillRectNoBorder(&buf, types.RectForWidthAndHeight(llx+w-zoom.HMargin, lly+zoom.VMargin, zoom.HMargin, h-2*zoom.VMargin), *zoom.BgColor)
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
		return fmt.Errorf("page dictionary: %w", err)
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

	pageZoom := *zoom
	zoom = &pageZoom
	if err := zoom.EnsureFactorAndMargins(cropBox.Width(), cropBox.Height()); err != nil {
		return fmt.Errorf("derive factor and margins: %w", err)
	}

	sc := zoom.Factor
	sin, cos := 0., 1.
	dx := zoom.HMargin
	dy := zoom.VMargin

	m := matrix.CalcTransformMatrix(sc, sc, sin, cos, dx, dy)

	var trans bytes.Buffer
	fmt.Fprintf(&trans, "q %.5f %.5f %.5f %.5f %.5f %.5f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	bb, err := ctx.PageContent(d, pageNr)
	if errors.Is(err, model.ErrNoContent) {
		if zoom.BgColor == nil && !zoom.Border {
			return nil
		}
		bb = nil
	} else if err != nil {
		return fmt.Errorf("read page content: %w", err)
	}

	if inhPAttrs.Rotate != 0 {
		bbInvRot := append([]byte(" q "), model.ContentBytesForPageRotation(inhPAttrs.Rotate, cropBox.Width(), cropBox.Height())...)
		bb = append(bbInvRot, bb...)
		bb = append(bb, []byte(" Q")...)
	}

	bb = append(trans.Bytes(), bb...)
	bb = append(bb, []byte(" Q ")...)

	handleZoomOutBgColAndBorder(cropBox, &bb, zoom)

	sd, err := ctx.NewStreamDictForBuf(bb)
	if err != nil {
		return fmt.Errorf("create content stream: %w", err)
	}
	if err := sd.Encode(); err != nil {
		return fmt.Errorf("encode content stream: %w", err)
	}

	ir, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		return fmt.Errorf("insert content stream: %w", err)
	}

	d["Contents"] = *ir

	d.Update("MediaBox", cropBox.Array())
	d.Delete("Rotate")
	d.Delete("CropBox")

	return nil
}

func zoomPageNumbers(pageCount int, selectedPages types.IntSet) []int {
	if len(selectedPages) == 0 {
		pageNrs := make([]int, pageCount)
		for i := range pageCount {
			pageNrs[i] = i + 1
		}
		return pageNrs
	}

	pageNrs := make([]int, 0, len(selectedPages))
	for pageNr, selected := range selectedPages {
		if selected {
			pageNrs = append(pageNrs, pageNr)
		}
	}
	sort.Ints(pageNrs)
	return pageNrs
}

// Zoom applies zoom to selected pages in ctx.
func Zoom(ctx *model.Context, selectedPages types.IntSet, zoom *model.Zoom) error {
	if log.DebugEnabled() {
		log.Debug.Printf("Zoom:\n%s\n", zoom)
	}

	for _, pageNr := range zoomPageNumbers(ctx.PageCount, selectedPages) {
		if err := zoomPage(ctx, pageNr, zoom); err != nil {
			return fmt.Errorf("page %d: %w", pageNr, err)
		}
	}

	ctx.EnsureVersionForWriting()

	return nil
}
