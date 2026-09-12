/*
Copyright 2018 The pdfcpu Authors.

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
	"context"
	"fmt"
	"sort"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func composePageRotation(current, delta int) int {
	rotation := (current%360 + delta%360) % 360
	if rotation < 0 {
		rotation += 360
	}
	return rotation
}

func rotatePage(xRefTable *model.XRefTable, i, j int) error {
	if log.DebugEnabled() {
		log.Debug.Printf("rotate page:%d\n", i)
	}

	consolidateRes := false
	d, _, inhPAttrs, err := xRefTable.PageDict(i, consolidateRes)
	if err != nil {
		return err
	}

	d.Update("Rotate", types.Integer(composePageRotation(inhPAttrs.Rotate, j)))

	return nil
}

// RotatePages rotates all selected pages by a multiple of 90 degrees and supports cancellation.
func RotatePages(c context.Context, ctx *model.Context, selectedPages types.IntSet, rotation int) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := requireContextWithXRefTable(ctx); err != nil {
		return fmt.Errorf("rotate pages: source context: %w", err)
	}
	return rotatePagesUsing(c, ctx, selectedPages, rotation, rotatePage)
}

func rotatePagesUsing(
	c context.Context,
	ctx *model.Context,
	selectedPages types.IntSet,
	rotation int,
	rotate func(*model.XRefTable, int, int) error,
) error {
	pageNrs := make([]int, 0, len(selectedPages))
	for pageNr, selected := range selectedPages {
		if err := c.Err(); err != nil {
			return err
		}
		if selected {
			pageNrs = append(pageNrs, pageNr)
		}
	}
	sort.Ints(pageNrs)

	for _, pageNr := range pageNrs {
		if err := c.Err(); err != nil {
			return err
		}
		if err := rotate(ctx.XRefTable, pageNr, rotation); err != nil {
			return fmt.Errorf("page %d: page dict: %w", pageNr, err)
		}
	}
	return nil
}
