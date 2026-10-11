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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type annotationWriteCheck struct {
	ctx  *model.Context
	seen map[types.IndirectRef]bool
}

func (v *annotationWriteCheck) check(c context.Context, o types.Object, kind string, depth int) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := v.ctx.CheckRecursionDepth("check annotation appearances for writing", depth); err != nil {
		return err
	}
	if ir, ok := o.(types.IndirectRef); ok {
		if v.seen[ir] {
			return nil
		}
		v.seen[ir] = true
	}
	obj, err := v.ctx.Dereference(o)
	if err != nil {
		return err
	}
	switch obj := obj.(type) {
	case types.Array:
		for _, child := range obj {
			if err := v.check(c, child, kind, depth+1); err != nil {
				return err
			}
		}
	case types.Dict:
		if err := v.checkDict(c, obj, kind, depth); err != nil {
			if ir, ok := o.(types.IndirectRef); ok {
				return fmt.Errorf("%s object %d: %w", kind, ir.ObjectNumber.Value(), err)
			}
			return err
		}
	}
	return nil
}

func (v *annotationWriteCheck) checkPage(c context.Context, d types.Dict, depth int) error {
	typ, err := v.ctx.DereferenceName(d["Type"], model.V10, nil)
	if err != nil {
		return err
	}
	if typ != "Page" {
		return v.check(c, d["Kids"], "Pages", depth+1)
	}
	return v.check(c, d["Annots"], "Annots", depth+1)
}

func (v *annotationWriteCheck) checkDict(c context.Context, d types.Dict, kind string, depth int) error {
	if kind == "Pages" {
		return v.checkPage(c, d, depth)
	}
	subtype, err := v.ctx.DereferenceName(d["Subtype"], model.V10, nil)
	if err != nil {
		return err
	}
	if kind == "Annots" || subtype == "Widget" {
		if err := v.ctx.CheckAnnotationAppearance(d); err != nil {
			return err
		}
	}
	if kind == "Fields" {
		return v.check(c, d["Kids"], "Fields", depth+1)
	}
	for _, key := range []string{"Popup", "IRT"} {
		if err := v.check(c, d[key], "Annots", depth+1); err != nil {
			return err
		}
	}
	if subtype == "Popup" {
		return v.check(c, d["Parent"], "Annots", depth+1)
	}
	return nil
}

func checkAnnotationAppearancesForWriting(c context.Context, ctx *model.Context) error {
	// see 12.5.2, Table 166
	if !ctx.PDF20() {
		return nil
	}
	root, err := ctx.Catalog()
	if err != nil {
		return err
	}
	v := annotationWriteCheck{ctx: ctx, seen: map[types.IndirectRef]bool{}}
	if err := v.check(c, root["Pages"], "Pages", 0); err != nil {
		return err
	}
	form, err := ctx.DereferenceDict(root["AcroForm"])
	if err != nil {
		return err
	}
	return v.check(c, form["Fields"], "Fields", 0)
}

func checkWrittenAnnotation(ctx *model.Context, d types.Dict) (bool, error) {
	if ctx.WritingAnnotation {
		return true, ctx.CheckAnnotationAppearance(d)
	}
	// Type is optional on annotations; Widgets can also be reached through the field tree.
	typ, err := ctx.Dereference(d["Type"])
	if err != nil {
		return false, err
	}
	subtype, err := ctx.Dereference(d["Subtype"])
	if err != nil {
		return false, err
	}
	if typ == types.Name("Annot") || subtype == types.Name("Widget") {
		return true, ctx.CheckAnnotationAppearance(d)
	}
	return false, nil
}

func writingAnnotationEntry(ctx *model.Context, d types.Dict, key string) (bool, error) {
	if key == "Annots" {
		typ, err := ctx.Dereference(d["Type"])
		return typ == types.Name("Page"), err
	}
	if !ctx.WritingAnnotation {
		return false, nil
	}
	if key == "Parent" {
		subtype, err := ctx.Dereference(d["Subtype"])
		return subtype == types.Name("Popup"), err
	}
	return key == "Popup" || key == "IRT", nil
}

func writeAnnotationAwareEntry(c context.Context, ctx *model.Context, d types.Dict, key string, o types.Object) (types.Object, bool, error) {
	if ctx.PDF20() && !ctx.Write.Increment {
		annotation, err := writingAnnotationEntry(ctx, d, key)
		if err != nil {
			return nil, false, err
		}
		previous := ctx.WritingAnnotation
		ctx.WritingAnnotation = annotation
		defer func() { ctx.WritingAnnotation = previous }()
	}
	return writeDeepObject(c, ctx, o)
}
