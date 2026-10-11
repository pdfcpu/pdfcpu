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

package pdfcpu

import (
	"context"
	"fmt"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func prepareCoreFontDict(ctx *model.Context, d types.Dict, owner int) error {
	o, err := ctx.Dereference(d["Type"])
	if err != nil {
		return err
	}
	if name, ok := o.(types.Name); !ok || name != "Font" {
		return nil
	}
	fd, err := pdffont.EnsureCoreFontMetrics(ctx.XRefTable, d)
	if err != nil {
		return err
	}
	if fd != nil && ctx.Write.Increment {
		ctx.Write.IncrementWithObjNr(owner)
		ctx.Write.IncrementWithObjNr(fd.ObjectNumber.Value())
	}
	return nil
}

func prepareCoreFontObject(c context.Context, ctx *model.Context, o types.Object, owner, depth int) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := ctx.CheckRecursionDepth("prepare core font metrics", depth); err != nil {
		return err
	}
	switch o := o.(type) {
	case types.Dict:
		if err := prepareCoreFontDict(ctx, o, owner); err != nil {
			return err
		}
		for _, v := range o {
			if err := prepareCoreFontObject(c, ctx, v, owner, depth+1); err != nil {
				return err
			}
		}
	case types.Array:
		for _, v := range o {
			if err := prepareCoreFontObject(c, ctx, v, owner, depth+1); err != nil {
				return err
			}
		}
	case types.StreamDict:
		return prepareCoreFontObject(c, ctx, o.Dict, owner, depth)
	}
	return nil
}

func prepareCoreFontsForWriting(c context.Context, ctx *model.Context) error {
	// See 9.6.2.1, Table 109: imported standard fonts also need explicit metrics in PDF 2.0.
	if !ctx.PDF20() {
		return nil
	}
	for objNr, e := range ctx.Table {
		if e == nil || e.Free || e.Object == nil {
			continue
		}
		o := e.Object
		if _, ok := o.(types.LazyObjectStreamObject); ok {
			var err error
			o, err = dereferencedObject(c, ctx, objNr)
			if err != nil {
				return err
			}
		}
		if err := prepareCoreFontObject(c, ctx, o, objNr, 0); err != nil {
			return fmt.Errorf("prepare PDF 2.0 fonts: object %d: %w", objNr, err)
		}
	}
	return nil
}
