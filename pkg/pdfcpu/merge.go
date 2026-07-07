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
	"errors"
	"fmt"
	"maps"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func newOutlinesDict(ctx *model.Context, fName string) (types.Dict, *types.IndirectRef, *types.IndirectRef, error) {
	if ctx == nil {
		return nil, nil, nil, errors.New("ensure outlines: missing context")
	}

	outlinesDict := types.Dict(map[string]types.Object{"Type": types.Name("Outlines")})
	indRef, err := ctx.IndRefForNewObject(outlinesDict)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ensure outlines: add outlines object: %w", err)
	}

	first, last, total, visible, err := createOutlineItemDict(ctx, []Bookmark{{PageFrom: 1, Title: fName}}, indRef, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ensure outlines: create outline item: %w", err)
	}
	if first == nil || last == nil {
		return nil, nil, nil, errors.New("ensure outlines: missing outline item")
	}

	outlinesDict["First"] = *first
	outlinesDict["Last"] = *last
	outlinesDict["Count"] = types.Integer(total + visible)
	return outlinesDict, indRef, first, nil
}

func foldExistingOutlines(ctx *model.Context, rootDict types.Dict, first *types.IndirectRef, append bool) error {
	if obj, ok := rootDict.Find("Outlines"); ok {
		if append {
			return nil
		}
		d, err := ctx.DereferenceDict(obj)
		if err != nil {
			return fmt.Errorf("ensure outlines: dereference existing outlines: %w", err)
		}
		if d == nil {
			return errors.New("ensure outlines: missing existing outlines dict")
		}
		count := d.IntEntry("Count")
		c := 0
		f, l := d.IndirectRefEntry("First"), d.IndirectRefEntry("Last")
		if f == nil || l == nil {
			return errors.New("ensure outlines: existing outlines missing first or last item")
		}
		for ir := f; ir != nil; ir = d.IndirectRefEntry("Next") {
			d, err = ctx.DereferenceDict(*ir)
			if err != nil {
				return fmt.Errorf("ensure outlines: dereference outline item: %w", err)
			}
			if d == nil {
				return errors.New("ensure outlines: missing outline item dict")
			}
			d["Parent"] = *first
			c++
		}
		d, err = ctx.DereferenceDict(*first)
		if err != nil {
			return fmt.Errorf("ensure outlines: dereference wrapper item: %w", err)
		}
		if d == nil {
			return errors.New("ensure outlines: missing wrapper item dict")
		}

		d["First"] = *f
		d["Last"] = *l
		if count != nil && *count != 0 {
			c = *count
		}
		d["Count"] = types.Integer(-c)
	}
	return nil
}

// EnsureOutlines ensures outlines.
func EnsureOutlines(ctx *model.Context, fName string, append bool) error {
	if ctx == nil {
		return errors.New("ensure outlines: missing context")
	}

	rootDict, err := ctx.Catalog()
	if err != nil {
		return fmt.Errorf("ensure outlines: catalog: %w", err)
	}
	if rootDict == nil {
		return errors.New("ensure outlines: missing catalog")
	}

	if err := ctx.LocateNameTree("Dests", true); err != nil {
		return fmt.Errorf("ensure outlines: locate dests name tree: %w", err)
	}

	_, indRef, first, err := newOutlinesDict(ctx, fName)
	if err != nil {
		return err
	}
	if err := foldExistingOutlines(ctx, rootDict, first, append); err != nil {
		return err
	}

	rootDict["Outlines"] = *indRef
	return nil
}

func mergeOutlinesWrapped(fName string, p int, ctxSrc, ctxDest *model.Context) error {
	if ctxSrc == nil || ctxDest == nil {
		return errors.New("merge outlines: missing context")
	}

	indRef, outlinesDict, oldLast, err := destOutlines(ctxDest)
	if err != nil {
		return fmt.Errorf("merge outlines: destination outlines: %w", err)
	}
	if indRef == nil || oldLast == nil {
		return nil
	}

	first, wrapperDict, err := appendOutlineWrapper(ctxDest, outlinesDict, indRef, oldLast, fName, p)
	if err != nil {
		return fmt.Errorf("merge outlines: append wrapper: %w", err)
	}
	if first == nil {
		return nil
	}

	topCount := outlineTopCount(outlinesDict) + 1

	rootDictSource, err := ctxSrc.Catalog()
	if err != nil {
		return fmt.Errorf("merge outlines: source catalog: %w", err)
	}
	if rootDictSource == nil {
		return errors.New("merge outlines: missing source catalog")
	}

	if c, err := attachWrappedSourceOutlines(ctxDest, rootDictSource, wrapperDict, first); err != nil {
		return err
	} else if c > 0 {
		topCount += c
	}

	outlinesDict["Count"] = types.Integer(topCount)
	return nil
}

func attachWrappedSourceOutlines(ctxDest *model.Context, rootDictSource, wrapperDict types.Dict, first *types.IndirectRef) (int, error) {
	obj, ok := rootDictSource.Find("Outlines")
	if !ok {
		return 0, nil
	}

	d, err := ctxDest.DereferenceDict(obj)
	if err != nil {
		return 0, fmt.Errorf("merge outlines: dereference source outlines: %w", err)
	}
	if d == nil {
		return 0, errors.New("merge outlines: missing source outlines dict")
	}

	f, l := d.IndirectRefEntry("First"), d.IndirectRefEntry("Last")
	if f == nil || l == nil {
		return 0, nil
	}

	wrapperDict["First"] = *f
	wrapperDict["Last"] = *l

	c := 0
	for ir := f; ir != nil; ir = d.IndirectRefEntry("Next") {
		d, err = ctxDest.DereferenceDict(*ir)
		if err != nil {
			return 0, fmt.Errorf("merge outlines: dereference source outline item: %w", err)
		}
		if d == nil {
			return 0, errors.New("merge outlines: missing source outline item dict")
		}

		d["Parent"] = *first
		if i := d.IntEntry("Count"); i != nil && *i > 0 {
			c += *i
		}
		c++
	}

	wrapperDict["Count"] = types.Integer(c)
	return c, nil
}

func destOutlines(ctxDest *model.Context) (*types.IndirectRef, types.Dict, *types.IndirectRef, error) {
	if ctxDest == nil {
		return nil, nil, nil, errors.New("destination outlines: missing context")
	}

	rootDictDest, err := ctxDest.Catalog()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("destination outlines: catalog: %w", err)
	}

	indRef := rootDictDest.IndirectRefEntry("Outlines")
	if indRef == nil {
		return nil, nil, nil, nil
	}

	outlinesDict, err := ctxDest.DereferenceDict(*indRef)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("destination outlines: dereference outlines: %w", err)
	}

	return indRef, outlinesDict, outlinesDict.IndirectRefEntry("Last"), nil
}

func outlineTopCount(outlinesDict types.Dict) int {
	count := outlinesDict.IntEntry("Count")
	if count == nil {
		return 0
	}
	return *count
}

func appendOutlineWrapper(ctxDest *model.Context, outlinesDict types.Dict, indRef, oldLast *types.IndirectRef, fName string, p int) (*types.IndirectRef, types.Dict, error) {
	if ctxDest == nil {
		return nil, nil, errors.New("outline wrapper: missing context")
	}
	if outlinesDict == nil {
		return nil, nil, errors.New("outline wrapper: missing outlines dict")
	}
	if indRef == nil || oldLast == nil {
		return nil, nil, errors.New("outline wrapper: missing outline references")
	}

	first, last, _, _, err := createOutlineItemDict(ctxDest, []Bookmark{{PageFrom: p, Title: fName}}, indRef, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("outline wrapper: create item: %w", err)
	}

	if first == nil || last == nil {
		return nil, nil, nil
	}

	if first != last {
		return nil, nil, errors.New("mergeOutlines: unexpected first != last indRefs")
	}

	d1, err := ctxDest.DereferenceDict(*oldLast)
	if err != nil {
		return nil, nil, fmt.Errorf("outline wrapper: dereference previous last: %w", err)
	}
	if d1 == nil {
		return nil, nil, errors.New("outline wrapper: missing previous last dict")
	}
	d1["Next"] = *first

	d2, err := ctxDest.DereferenceDict(*first)
	if err != nil {
		return nil, nil, fmt.Errorf("outline wrapper: dereference wrapper item: %w", err)
	}
	if d2 == nil {
		return nil, nil, errors.New("outline wrapper: missing wrapper item dict")
	}
	d2["Previous"] = *oldLast

	outlinesDict["Last"] = *first
	return first, d2, nil
}

func ensureOutlinesRoot(ctx *model.Context) (*types.IndirectRef, types.Dict, error) {
	if ctx == nil {
		return nil, nil, errors.New("outlines root: missing context")
	}

	rootDict, err := ctx.Catalog()
	if err != nil {
		return nil, nil, fmt.Errorf("outlines root: catalog: %w", err)
	}
	if rootDict == nil {
		return nil, nil, errors.New("outlines root: missing catalog")
	}

	indRef := rootDict.IndirectRefEntry("Outlines")
	if indRef != nil {
		d, err := ctx.DereferenceDict(*indRef)
		if err != nil {
			return nil, nil, fmt.Errorf("outlines root: dereference outlines: %w", err)
		}
		if d == nil {
			return nil, nil, errors.New("outlines root: missing outlines dict")
		}
		return indRef, d, nil
	}

	outlinesDict := types.Dict(map[string]types.Object{"Type": types.Name("Outlines")})
	indRef, err = ctx.IndRefForNewObject(outlinesDict)
	if err != nil {
		return nil, nil, fmt.Errorf("outlines root: add outlines object: %w", err)
	}

	rootDict["Outlines"] = *indRef
	return indRef, outlinesDict, nil
}

func sourceOutlines(ctxSrc, ctxDest *model.Context) (*types.IndirectRef, *types.IndirectRef, int, error) {
	if ctxSrc == nil || ctxDest == nil {
		return nil, nil, 0, errors.New("source outlines: missing context")
	}

	rootDictSrc, err := ctxSrc.Catalog()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("source outlines: catalog: %w", err)
	}

	indRef := rootDictSrc.IndirectRefEntry("Outlines")
	if indRef == nil {
		return nil, nil, 0, nil
	}

	d, err := ctxDest.DereferenceDict(*indRef)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("source outlines: dereference outlines: %w", err)
	}
	if d == nil {
		return nil, nil, 0, errors.New("source outlines: missing outlines dict")
	}

	first, last := d.IndirectRefEntry("First"), d.IndirectRefEntry("Last")
	if first == nil || last == nil {
		return nil, nil, 0, nil
	}

	count := d.IntEntry("Count")
	if count != nil {
		return first, last, *count, nil
	}

	return first, last, 0, nil
}

func reparentOutlineItems(ctx *model.Context, first, parent *types.IndirectRef) (int, error) {
	if ctx == nil {
		return 0, errors.New("reparent outline item: missing context")
	}
	if first == nil || parent == nil {
		return 0, errors.New("reparent outline item: missing outline references")
	}

	count := 0
	for ir := first; ir != nil; {
		d, err := ctx.DereferenceDict(*ir)
		if err != nil {
			return 0, fmt.Errorf("reparent outline item: dereference item: %w", err)
		}
		if d == nil {
			return 0, errors.New("reparent outline item: missing item dict")
		}
		d["Parent"] = *parent

		if i := d.IntEntry("Count"); i != nil && *i > 0 {
			count += *i
		}
		count++
		ir = d.IndirectRefEntry("Next")
	}
	return count, nil
}

func mergeOutlinesPreserve(ctxSrc, ctxDest *model.Context) error {
	first, last, sourceCount, err := sourceOutlines(ctxSrc, ctxDest)
	if err != nil {
		return fmt.Errorf("merge outlines preserve: source outlines: %w", err)
	}
	if first == nil {
		return nil
	}

	indRef, outlinesDict, err := ensureOutlinesRoot(ctxDest)
	if err != nil {
		return fmt.Errorf("merge outlines preserve: ensure destination root: %w", err)
	}

	count, err := reparentOutlineItems(ctxDest, first, indRef)
	if err != nil {
		return fmt.Errorf("merge outlines preserve: reparent source items: %w", err)
	}
	if sourceCount != 0 {
		count = sourceCount
	}

	if l := outlinesDict.IndirectRefEntry("Last"); l != nil {
		d, err := ctxDest.DereferenceDict(*l)
		if err != nil {
			return fmt.Errorf("merge outlines preserve: dereference destination last: %w", err)
		}
		if d == nil {
			return errors.New("merge outlines preserve: missing destination last dict")
		}
		d["Next"] = *first
		d, err = ctxDest.DereferenceDict(*first)
		if err != nil {
			return fmt.Errorf("merge outlines preserve: dereference source first: %w", err)
		}
		if d == nil {
			return errors.New("merge outlines preserve: missing source first dict")
		}
		d["Prev"] = *l
	} else {
		outlinesDict["First"] = *first
	}

	outlinesDict["Last"] = *last
	if c := outlinesDict.IntEntry("Count"); c != nil {
		count += *c
	}
	outlinesDict["Count"] = types.Integer(count)

	return nil
}

func handleNeedAppearances(ctxSrc *model.Context, dSrc, dDest types.Dict) error {
	o, found := dSrc.Find("NeedAppearances")
	if !found || o == nil {
		return nil
	}
	b, err := ctxSrc.DereferenceBoolean(o, model.V10)
	if err != nil {
		return fmt.Errorf("form NeedAppearances: dereference boolean: %w", err)
	}
	if b != nil && *b {
		dDest["NeedAppearances"] = types.Boolean(true)
	}
	return nil
}

func handleCO(ctxSrc, ctxDest *model.Context, dSrc, dDest types.Dict) error {
	o, found := dSrc.Find("CO")
	if !found {
		return nil
	}
	arrSrc, err := ctxSrc.DereferenceArray(o)
	if err != nil {
		return fmt.Errorf("form CO: dereference source array: %w", err)
	}
	o, found = dDest.Find("CO")
	if !found {
		dDest["CO"] = arrSrc
		return nil
	}
	arrDest, err := ctxDest.DereferenceArray(o)
	if err != nil {
		return fmt.Errorf("form CO: dereference destination array: %w", err)
	}
	if len(arrDest) == 0 {
		dDest["CO"] = arrSrc
	} else {
		arrDest = append(arrDest, arrSrc...)
		dDest["CO"] = arrDest
	}
	return nil
}

func handleDR(ctxSrc *model.Context, dSrc, dDest types.Dict) error {
	o, found := dSrc.Find("DR")
	if !found {
		return nil
	}

	dSrc, err := ctxSrc.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("form DR: dereference source dict: %w", err)
	}
	if len(dSrc) == 0 {
		return nil
	}

	_, found = dDest.Find("DR")
	if !found {
		dDest["DR"] = dSrc
	}

	return nil
}

func handleDA(ctxSrc *model.Context, dSrc, dDest types.Dict, arrFieldsSrc types.Array) error {
	// (for each with field type  /FT /Tx w/o DA, set DA to default DA)
	// TODO Walk field tree and inspect terminal fields.

	sSrc := dSrc.StringEntry("DA")
	if sSrc == nil || len(*sSrc) == 0 {
		return nil
	}
	sDest := dDest.StringEntry("DA")
	if sDest == nil {
		dDest["DA"] = types.StringLiteral(*sSrc)
		return nil
	}
	// Push sSrc down to all top level fields of dSource
	for _, o := range arrFieldsSrc {
		d, err := ctxSrc.DereferenceDict(o)
		if err != nil {
			return fmt.Errorf("form DA: dereference source field: %w", err)
		}
		n := d.NameEntry("FT")
		if n != nil && *n == "Tx" {
			_, found := d.Find("DA")
			if !found {
				d["DA"] = types.StringLiteral(*sSrc)
			}
		}
	}
	return nil
}

func handleQ(ctxSrc *model.Context, dSrc, dDest types.Dict, arrFieldsSrc types.Array) error {
	// (for each with field type /FT /Tx w/o Q, set Q to default Q)
	// TODO Walk field tree and inspect terminal fields.

	iSrc := dSrc.IntEntry("Q")
	if iSrc == nil {
		return nil
	}
	iDest := dDest.IntEntry("Q")
	if iDest == nil {
		dDest["Q"] = types.Integer(*iSrc)
		return nil
	}
	// Push iSrc down to all top level fields of dSource
	for _, o := range arrFieldsSrc {
		d, err := ctxSrc.DereferenceDict(o)
		if err != nil {
			return fmt.Errorf("form Q: dereference source field: %w", err)
		}
		n := d.NameEntry("FT")
		if n != nil && *n == "Tx" {
			_, found := d.Find("Q")
			if !found {
				d["Q"] = types.Integer(*iSrc)
			}
		}
	}
	return nil
}

func handleFormAttributes(ctxSrc, ctxDest *model.Context, dSrc, dDest types.Dict, arrFieldsSrc types.Array) error {
	// NeedAppearances: try: set to true only
	if err := handleNeedAppearances(ctxSrc, dSrc, dDest); err != nil {
		return fmt.Errorf("NeedAppearances: %w", err)
	}

	// SigFlags: set bit 1 to true only (SignaturesExist)
	//           set bit 2 to true only (AppendOnly)
	dDest.Delete("SigFlags")

	// CO: add all indrefs
	if err := handleCO(ctxSrc, ctxDest, dSrc, dDest); err != nil {
		return fmt.Errorf("CO: %w", err)
	}

	// DR: default resource dict
	if err := handleDR(ctxSrc, dSrc, dDest); err != nil {
		return fmt.Errorf("DR: %w", err)
	}

	// DA: default appearance streams for variable text fields
	if err := handleDA(ctxSrc, dSrc, dDest, arrFieldsSrc); err != nil {
		return fmt.Errorf("DA: %w", err)
	}

	// Q: left, center, right for variable text fields
	if err := handleQ(ctxSrc, dSrc, dDest, arrFieldsSrc); err != nil {
		return fmt.Errorf("Q: %w", err)
	}

	// XFA: ignore
	delete(dDest, "XFA")

	return nil
}

func rootDicts(ctxSrc, ctxDest *model.Context) (types.Dict, types.Dict, error) {
	if ctxSrc == nil || ctxDest == nil {
		return nil, nil, errors.New("merge root dicts: missing context")
	}

	rootDictSource, err := ctxSrc.Catalog()
	if err != nil {
		return nil, nil, fmt.Errorf("merge root dicts: source catalog: %w", err)
	}

	rootDictDest, err := ctxDest.Catalog()
	if err != nil {
		return nil, nil, fmt.Errorf("merge root dicts: destination catalog: %w", err)
	}

	return rootDictSource, rootDictDest, nil
}

func mergeInFields(ctxDest *model.Context, arrFieldsSrc, arrFieldsDest types.Array, dDest types.Dict) error {
	if ctxDest == nil {
		return errors.New("merge form fields: missing destination context")
	}

	parentDict :=
		types.Dict(map[string]types.Object{
			"Kids": arrFieldsSrc,
			"T":    types.StringLiteral(fmt.Sprintf("%d", len(arrFieldsDest))),
		})

	ir, err := ctxDest.IndRefForNewObject(parentDict)
	if err != nil {
		return fmt.Errorf("merge form fields: add parent field: %w", err)
	}

	for _, ir1 := range arrFieldsSrc {
		d, err := ctxDest.DereferenceDict(ir1)
		if err != nil {
			return fmt.Errorf("merge form fields: dereference source field: %w", err)
		}
		if len(d) == 0 {
			continue
		}
		d["Parent"] = *ir
	}

	dDest["Fields"] = append(arrFieldsDest, *ir)

	return nil
}

func fieldWidgetObjNrs(ctx *model.Context, fields types.Array, m types.IntSet) error {
	if ctx == nil {
		return errors.New("field widget object numbers: missing context")
	}
	if m == nil {
		return errors.New("field widget object numbers: missing result set")
	}

	for _, obj := range fields {
		ir, ok := obj.(types.IndirectRef)
		if !ok {
			continue
		}
		m[ir.ObjectNumber.Value()] = true
		d, err := ctx.DereferenceDict(ir)
		if err != nil {
			return fmt.Errorf("field widget object numbers: dereference field: %w", err)
		}
		kids, err := ctx.DereferenceArray(d["Kids"])
		if err != nil {
			return fmt.Errorf("field widget object numbers: dereference kids: %w", err)
		}
		if len(kids) > 0 {
			if err := fieldWidgetObjNrs(ctx, kids, m); err != nil {
				return fmt.Errorf("field widget object numbers: kids: %w", err)
			}
		}
	}
	return nil
}

func sourceFieldWidgetObjNrs(ctx *model.Context) (types.IntSet, error) {
	if ctx == nil {
		return nil, errors.New("source field widget object numbers: missing context")
	}

	m := types.IntSet{}
	if ctx.Form == nil {
		return m, nil
	}
	obj, found := ctx.Form.Find("Fields")
	if !found {
		return m, nil
	}
	fields, err := ctx.DereferenceArray(obj)
	if err != nil {
		return nil, fmt.Errorf("source field widget object numbers: dereference fields: %w", err)
	}
	if err := fieldWidgetObjNrs(ctx, fields, m); err != nil {
		return nil, fmt.Errorf("source field widget object numbers: collect fields: %w", err)
	}
	return m, nil
}

func renameOrphanWidgetField(d types.Dict, namespace string) error {
	if typ := d.NameEntry("Subtype"); typ == nil || *typ != "Widget" {
		return nil
	}
	name, err := d.StringOrHexLiteralEntry("T")
	if err != nil {
		return fmt.Errorf("orphan widget field: T: %w", err)
	}
	if name == nil || *name == "" {
		return nil
	}
	d["T"] = types.StringLiteral(namespace + "." + *name)
	return nil
}

func renameSourceOrphanWidgetFields(ctx *model.Context, namespace string) error {
	fieldWidgets, err := sourceFieldWidgetObjNrs(ctx)
	if err != nil {
		return fmt.Errorf("rename orphan widget fields: source widget fields: %w", err)
	}
	for objNr, entry := range ctx.Table {
		if entry.Free || fieldWidgets[objNr] {
			continue
		}
		d, ok := entry.Object.(types.Dict)
		if !ok {
			continue
		}
		if err := renameOrphanWidgetField(d, namespace); err != nil {
			return fmt.Errorf("rename orphan widget fields: obj#%d: %w", objNr, err)
		}
	}
	return nil
}

func mergeDests(ctxSource, ctxDest *model.Context) error {
	rootDictSource, rootDictDest, err := rootDicts(ctxSource, ctxDest)
	if err != nil {
		return fmt.Errorf("root dicts: %w", err)
	}

	o1, found := rootDictSource.Find("Dests")
	if !found {
		return nil
	}

	o2, found := rootDictDest.Find("Dests")
	if !found {
		rootDictDest["Dests"] = o1
		return nil
	}

	destsSrc, err := ctxSource.DereferenceDict(o1)
	if err != nil {
		return fmt.Errorf("dereference source Dests: %w", err)
	}

	destsDest, err := ctxDest.DereferenceDict(o2)
	if err != nil {
		return fmt.Errorf("dereference destination Dests: %w", err)
	}

	// Note: We ignore duplicate keys
	maps.Copy(destsDest, destsSrc)

	return nil
}

func mergeNames(ctxSrc, ctxDest *model.Context) error {
	rootDictSrc, rootDictDest, err := rootDicts(ctxSrc, ctxDest)
	if err != nil {
		return fmt.Errorf("root dicts: %w", err)
	}

	_, found := rootDictSrc.Find("Names")
	if !found {
		// Nothing to merge in.
		return nil
	}

	if _, found := rootDictDest.Find("Names"); !found {
		ctxDest.Names = ctxSrc.Names
		return nil
	}
	if ctxDest.Names == nil {
		ctxDest.Names = map[string]*model.Node{}
	}

	// We need to merge src Names into dest Names.

	for id, namesSrc := range ctxSrc.Names {
		if namesDest, ok := ctxDest.Names[id]; ok {
			// Merge src tree into dest tree including collision detection.
			if err := namesDest.AddTree(ctxDest.XRefTable, namesSrc, ctxSrc.NameRefs[id], []string{"D", "Dest"}); err != nil {
				return fmt.Errorf("name tree %s: %w", id, err)
			}
			continue
		}

		// Name tree missing in dest ctx => copy over names from src ctx
		ctxDest.Names[id] = namesSrc
	}

	return nil
}

func sourceFormFields(ctxSrc *model.Context, rootDictSource types.Dict) (types.Dict, types.Array, bool, error) {
	o, found := rootDictSource.Find("AcroForm")
	if !found {
		return nil, nil, false, nil
	}

	dSrc, err := ctxSrc.DereferenceDict(o)
	if err != nil || len(dSrc) == 0 {
		if err != nil {
			return nil, nil, false, fmt.Errorf("dereference source AcroForm: %w", err)
		}
		return nil, nil, false, nil
	}

	o, found = dSrc.Find("Fields")
	if !found {
		return nil, nil, false, nil
	}
	arrFieldsSrc, err := ctxSrc.DereferenceArray(o)
	if err != nil {
		return nil, nil, false, fmt.Errorf("dereference source Fields: %w", err)
	}
	if len(arrFieldsSrc) == 0 {
		return nil, nil, false, nil
	}
	return dSrc, arrFieldsSrc, true, nil
}

func destinationFormFields(ctxDest *model.Context, rootDictDest, dSrc types.Dict) (types.Dict, types.Array, bool, error) {
	o, found := rootDictDest.Find("AcroForm")
	if !found {
		rootDictDest["AcroForm"] = dSrc
		return nil, nil, false, nil
	}

	dDest, err := ctxDest.DereferenceDict(o)
	if err != nil {
		return nil, nil, false, fmt.Errorf("dereference destination AcroForm: %w", err)
	}

	if len(dDest) == 0 {
		rootDictDest["AcroForm"] = dSrc
		return nil, nil, false, nil
	}

	o, found = dDest.Find("Fields")
	if !found {
		rootDictDest["AcroForm"] = dSrc
		return nil, nil, false, nil
	}
	arrFieldsDest, err := ctxDest.DereferenceArray(o)
	if err != nil {
		return nil, nil, false, fmt.Errorf("dereference destination Fields: %w", err)
	}
	if len(arrFieldsDest) == 0 {
		rootDictDest["AcroForm"] = dSrc
		return nil, nil, false, nil
	}
	return dDest, arrFieldsDest, true, nil
}

func mergeForms(ctxSrc, ctxDest *model.Context) error {
	rootDictSource, rootDictDest, err := rootDicts(ctxSrc, ctxDest)
	if err != nil {
		return fmt.Errorf("root dicts: %w", err)
	}

	dSrc, arrFieldsSrc, ok, err := sourceFormFields(ctxSrc, rootDictSource)
	if err != nil || !ok {
		return err
	}

	dDest, arrFieldsDest, ok, err := destinationFormFields(ctxDest, rootDictDest, dSrc)
	if err != nil || !ok {
		return err
	}

	if err := mergeInFields(ctxDest, arrFieldsSrc, arrFieldsDest, dDest); err != nil {
		return fmt.Errorf("merge field arrays: %w", err)
	}

	if err := handleFormAttributes(ctxSrc, ctxDest, dSrc, dDest, arrFieldsSrc); err != nil {
		return fmt.Errorf("form attributes: %w", err)
	}
	return nil
}

func patchIndRef(ir *types.IndirectRef, lookup map[int]int) {
	if ir == nil {
		return
	}
	i := ir.ObjectNumber.Value()
	j, ok := lookup[i]
	if !ok {
		return
	}
	ir.ObjectNumber = types.Integer(j)
}

func patchObject(o types.Object, lookup map[int]int) types.Object {
	if log.TraceEnabled() {
		log.Trace.Printf("patchObject before: %v\n", o)
	}

	var ob types.Object

	switch obj := o.(type) {

	case types.IndirectRef:
		patchIndRef(&obj, lookup)
		ob = obj

	case types.Dict:
		patchDict(obj, lookup)
		ob = obj

	case types.StreamDict:
		patchDict(obj.Dict, lookup)
		ob = obj

	case types.ObjectStreamDict:
		patchDict(obj.Dict, lookup)
		ob = obj

	case types.XRefStreamDict:
		patchDict(obj.Dict, lookup)
		ob = obj

	case types.Array:
		patchArray(&obj, lookup)
		ob = obj
	}

	if log.TraceEnabled() {
		log.Trace.Printf("patchObject end: %v\n", ob)
	}

	return ob
}

func patchDict(d types.Dict, lookup map[int]int) {
	if log.TraceEnabled() {
		log.Trace.Printf("patchDict before: %v\n", d)
	}

	for k, obj := range d {
		o := patchObject(obj, lookup)
		if o != nil {
			d[k] = o
		}
	}

	if log.TraceEnabled() {
		log.Trace.Printf("patchDict after: %v\n", d)
	}
}

func patchArray(a *types.Array, lookup map[int]int) {
	if a == nil {
		return
	}
	if log.TraceEnabled() {
		log.Trace.Printf("patchArray begin: %v\n", *a)
	}

	for i, obj := range *a {
		o := patchObject(obj, lookup)
		if o != nil {
			(*a)[i] = o
		}
	}

	if log.TraceEnabled() {
		log.Trace.Printf("patchArray end: %v\n", a)
	}
}

func objNrsIntSet(ctx *model.Context) types.IntSet {
	objNrs := types.IntSet{}
	if ctx == nil {
		return objNrs
	}

	for k := range ctx.Table {
		if k == 0 {
			// obj#0 is always the head of the freelist.
			continue
		}
		objNrs[k] = true
	}

	return objNrs
}

func lookupTable(keys types.IntSet, i int) map[int]int {
	m := map[int]int{}

	for k := range keys {
		m[k] = i
		i++
	}

	return m
}

// Patch an IntSet of objNrs using lookup.
func patchObjects(s types.IntSet, lookup map[int]int) types.IntSet {
	t := types.IntSet{}

	for k, v := range s {
		if v {
			if j, ok := lookup[k]; ok {
				t[j] = v
			}
		}
	}

	return t
}

func patchNameTree(n *model.Node, lookup map[int]int) error {
	if n == nil {
		return nil
	}

	patchValues := func(xRefTable *model.XRefTable, k string, v *types.Object) error {
		if v == nil {
			return nil
		}
		*v = patchObject(*v, lookup)
		return nil
	}

	return n.Process(nil, patchValues)
}

func validatePatchSourceContexts(ctxSrc, ctxDest *model.Context) error {
	if ctxSrc == nil || ctxDest == nil {
		return errors.New("patch source object numbers: missing context")
	}
	if ctxSrc.XRefTable == nil || ctxDest.XRefTable == nil {
		return errors.New("patch source object numbers: missing xref table")
	}
	if ctxSrc.Root == nil {
		return errors.New("patch source object numbers: missing source root")
	}
	if ctxSrc.Size == nil || ctxDest.Size == nil {
		return errors.New("patch source object numbers: missing xref size")
	}
	if ctxSrc.Read == nil || ctxDest.Read == nil {
		return errors.New("patch source object numbers: missing read context")
	}
	return nil
}

func patchFreeListHead(ctxSrc *model.Context, lookup map[int]int) error {
	entry := ctxSrc.Table[0]
	if entry == nil || entry.Offset == nil {
		return errors.New("patch source object numbers: missing free list head")
	}
	off := int(*entry.Offset)
	if off != 0 {
		j, ok := lookup[off]
		if !ok {
			return fmt.Errorf("patch source object numbers: missing free list target %d", off)
		}
		i := int64(j)
		entry.Offset = &i
	}
	return nil
}

func patchSourceXRefEntries(ctxSrc *model.Context, objNrs types.IntSet, lookup map[int]int) error {
	for k := range objNrs {
		entry := ctxSrc.Table[k]
		if entry == nil {
			return fmt.Errorf("patch source object numbers: missing xref entry %d", k)
		}

		if entry.Free {
			if entry.Offset == nil {
				return fmt.Errorf("patch source object numbers: missing free entry offset %d", k)
			}
			if log.DebugEnabled() {
				log.Debug.Printf("patch free entry: old offset:%d\n", *entry.Offset)
			}
			off := int(*entry.Offset)
			if off == 0 {
				continue
			}
			j, ok := lookup[off]
			if !ok {
				return fmt.Errorf("patch source object numbers: missing free entry target %d", off)
			}
			i := int64(j)
			entry.Offset = &i
			if log.DebugEnabled() {
				log.Debug.Printf("patch free entry: new offset:%d\n", *entry.Offset)
			}
			continue
		}

		patchObject(entry.Object, lookup)
	}
	return nil
}

func remapSourceXRefTable(ctxSrc *model.Context, lookup map[int]int) {
	m := make(map[int]*model.XRefTableEntry, *ctxSrc.Size)
	for k, v := range lookup {
		m[v] = ctxSrc.Table[k]
	}
	m[0] = ctxSrc.Table[0]
	ctxSrc.Table = m
}

func patchSourceCaches(ctxSrc *model.Context, lookup map[int]int) error {
	if ctxSrc.Optimize == nil {
		return errors.New("patch source object numbers: missing source optimization context")
	}
	ctxSrc.Optimize.DuplicateInfoObjects = patchObjects(ctxSrc.Optimize.DuplicateInfoObjects, lookup)

	// Patch Linearization object numbers.
	ctxSrc.LinearizationObjs = patchObjects(ctxSrc.LinearizationObjs, lookup)

	// Patch XRefStream objects numbers.
	ctxSrc.Read.XRefStreams = patchObjects(ctxSrc.Read.XRefStreams, lookup)

	// Patch object stream object numbers.
	ctxSrc.Read.ObjectStreams = patchObjects(ctxSrc.Read.ObjectStreams, lookup)

	// Patch cached name trees.
	for id, v := range ctxSrc.Names {
		if err := patchNameTree(v, lookup); err != nil {
			return fmt.Errorf("patch source object numbers: name tree %s: %w", id, err)
		}
	}
	return nil
}

func patchSourceObjectNumbers(ctxSrc, ctxDest *model.Context) error {
	if err := validatePatchSourceContexts(ctxSrc, ctxDest); err != nil {
		return err
	}

	if log.DebugEnabled() {
		log.Debug.Printf("patchSourceObjectNumbers:  ctxSrc: xRefTableSize:%d trailer.Size:%d - %s\n", len(ctxSrc.Table), *ctxSrc.Size, ctxSrc.Read.FileName)
		log.Debug.Printf("patchSourceObjectNumbers: ctxDest: xRefTableSize:%d trailer.Size:%d - %s\n", len(ctxDest.Table), *ctxDest.Size, ctxDest.Read.FileName)
	}

	objNrs := objNrsIntSet(ctxSrc)
	lookup := lookupTable(objNrs, *ctxDest.Size)

	patchIndRef(ctxSrc.Root, lookup)
	if ctxSrc.Info != nil {
		patchIndRef(ctxSrc.Info, lookup)
	}

	if err := patchFreeListHead(ctxSrc, lookup); err != nil {
		return err
	}
	if err := patchSourceXRefEntries(ctxSrc, objNrs, lookup); err != nil {
		return err
	}
	remapSourceXRefTable(ctxSrc, lookup)
	if err := patchSourceCaches(ctxSrc, lookup); err != nil {
		return err
	}

	if log.DebugEnabled() {
		log.Debug.Printf("patchSourceObjectNumbers end")
	}
	return nil
}

func createDividerPagesDict(ctx *model.Context, parentIndRef types.IndirectRef) (*types.IndirectRef, error) {
	if ctx == nil || ctx.XRefTable == nil {
		return nil, errors.New("divider page tree: missing context")
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":   types.Name("Pages"),
			"Parent": parentIndRef,
			"Count":  types.Integer(1),
		},
	)

	indRef, err := ctx.IndRefForNewObject(d)
	if err != nil {
		return nil, fmt.Errorf("divider page tree: add pages object: %w", err)
	}

	dims, err := ctx.XRefTable.PageDims()
	if err != nil {
		return nil, fmt.Errorf("divider page tree: page dimensions: %w", err)
	}
	if len(dims) == 0 {
		return nil, errors.New("divider page tree: missing page dimensions")
	}

	last := len(dims) - 1
	mediaBox := types.NewRectangle(0, 0, dims[last].Width, dims[last].Height)

	indRefPageDict, err := ctx.EmptyPage(indRef, mediaBox, 0)
	if err != nil {
		return nil, fmt.Errorf("divider page tree: create empty page: %w", err)
	}
	if indRefPageDict == nil {
		return nil, errors.New("divider page tree: missing empty page reference")
	}
	ctx.SetValid(*indRefPageDict)

	d.Insert("Kids", types.Array{*indRefPageDict})

	return indRef, nil
}

func pageTreeRoot(ctx *model.Context) (*types.IndirectRef, types.Dict, error) {
	if ctx == nil || ctx.XRefTable == nil {
		return nil, nil, errors.New("page tree root: missing context")
	}

	indRef, err := ctx.Pages()
	if err != nil {
		return nil, nil, fmt.Errorf("page tree root: pages entry: %w", err)
	}
	if indRef == nil {
		return nil, nil, errors.New("page tree root: missing /Pages")
	}

	d, err := ctx.XRefTable.DereferenceDict(*indRef)
	if err != nil {
		return nil, nil, fmt.Errorf("page tree root: dereference root: %w", err)
	}

	pageCount := d.IntEntry("Count")
	if pageCount == nil || *pageCount != ctx.PageCount {
		return nil, nil, fmt.Errorf("corrupt page node at obj #%d", indRef.ObjectNumber)
	}

	return indRef, d, nil
}

func hasInheritedPageAttrs(d types.Dict) bool {
	for _, key := range []string{"Resources", "MediaBox", "CropBox", "Rotate"} {
		if _, ok := d[key]; ok {
			return true
		}
	}
	return false
}

func ensureNeutralPageTreeRoot(ctx *model.Context, indRef *types.IndirectRef, d types.Dict) (*types.IndirectRef, types.Dict, error) {
	if ctx == nil {
		return nil, nil, errors.New("neutral page tree root: missing context")
	}
	if indRef == nil {
		return nil, nil, errors.New("neutral page tree root: missing page tree root")
	}

	if !hasInheritedPageAttrs(d) {
		return indRef, d, nil
	}

	newRoot := types.Dict(
		map[string]types.Object{
			"Type":  types.Name("Pages"),
			"Kids":  types.Array{*indRef},
			"Count": types.Integer(ctx.PageCount),
		},
	)

	newRootIndRef, err := ctx.IndRefForNewObject(newRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("neutral page tree root: add root: %w", err)
	}
	if newRootIndRef == nil {
		return nil, nil, errors.New("neutral page tree root: missing new root reference")
	}

	rootDict, err := ctx.Catalog()
	if err != nil {
		return nil, nil, fmt.Errorf("neutral page tree root: catalog: %w", err)
	}

	d["Parent"] = *newRootIndRef
	rootDict["Pages"] = *newRootIndRef

	return newRootIndRef, newRoot, nil
}

func pageTreeKids(d types.Dict, indRef types.IndirectRef) (types.Array, error) {
	obj, ok := d["Kids"]
	if !ok {
		return nil, fmt.Errorf("page node at obj #%d missing /Kids", indRef.ObjectNumber)
	}

	kids, ok := obj.(types.Array)
	if !ok {
		return nil, fmt.Errorf("page node at obj #%d /Kids is not an Array", indRef.ObjectNumber)
	}

	return kids, nil
}

func appendSourcePageTreeToDestPageTree(ctxSrc, ctxDest *model.Context, dividerPage bool) error {
	if log.DebugEnabled() {
		log.Debug.Println("appendSourcePageTreeToDestPageTree begin")
	}

	destRootIndRef, destRoot, err := pageTreeRoot(ctxDest)
	if err != nil {
		return fmt.Errorf("destination page tree root: %w", err)
	}

	destRootIndRef, destRoot, err = ensureNeutralPageTreeRoot(ctxDest, destRootIndRef, destRoot)
	if err != nil {
		return fmt.Errorf("destination page tree root: ensure neutral root: %w", err)
	}

	destKids, err := pageTreeKids(destRoot, *destRootIndRef)
	if err != nil {
		return fmt.Errorf("destination page tree root: kids: %w", err)
	}

	addedPageCount := 0
	if dividerPage {
		dividerIndRef, err := createDividerPagesDict(ctxDest, *destRootIndRef)
		if err != nil {
			return fmt.Errorf("divider page: %w", err)
		}
		destKids = append(destKids, *dividerIndRef)
		addedPageCount++
	}

	srcRootIndRef, srcRoot, err := pageTreeRoot(ctxSrc)
	if err != nil {
		return fmt.Errorf("source page tree root: %w", err)
	}

	srcRoot["Parent"] = *destRootIndRef
	destKids = append(destKids, *srcRootIndRef)
	addedPageCount += ctxSrc.PageCount

	destRoot["Kids"] = destKids
	destRoot["Count"] = types.Integer(ctxDest.PageCount + addedPageCount)
	ctxDest.PageCount += addedPageCount

	if log.DebugEnabled() {
		log.Debug.Println("appendSourcePageTreeToDestPageTree end")
	}

	return nil
}

func zipSourcePageTreeIntoDestPageTree(ctxSrc, ctxDest *model.Context) error {
	if log.DebugEnabled() {
		log.Debug.Println("zipSourcePageTreeIntoDestPageTree begin")
	}

	appendFromPageNr := 0
	if ctxSrc.PageCount > ctxDest.PageCount {
		appendFromPageNr = ctxDest.PageCount + 1
	}

	rootPageIndRef, err := ctxDest.Pages()
	if err != nil {
		return fmt.Errorf("zip page tree: destination pages entry: %w", err)
	}
	if rootPageIndRef == nil {
		return errors.New("zip page tree: missing destination /Pages")
	}

	// Process dest page tree recursively and weave in src pages
	p := 0
	if ctxDest.PageCount, err = ctxDest.InsertPages(rootPageIndRef, &p, ctxSrc); err != nil {
		return fmt.Errorf("zip page tree: insert source pages: %w", err)
	}

	if appendFromPageNr > 0 {
		// append remaining src pages
		if ctxDest.PageCount, err = ctxDest.AppendPages(rootPageIndRef, appendFromPageNr, ctxSrc); err != nil {
			return fmt.Errorf("zip page tree: append remaining source pages: %w", err)
		}
	}

	if log.DebugEnabled() {
		log.Debug.Println("zipSourcePageTreeIntoDestPageTree end")
	}

	return nil
}

func appendSourceObjectsToDest(ctxSrc, ctxDest *model.Context) error {
	if ctxSrc == nil || ctxDest == nil {
		return errors.New("append source objects: missing context")
	}
	if ctxSrc.Table == nil {
		return errors.New("append source objects: missing source xref table")
	}
	if ctxDest.Table == nil {
		return errors.New("append source objects: missing destination xref table")
	}
	if ctxDest.Size == nil {
		return errors.New("append source objects: missing destination xref size")
	}

	if log.DebugEnabled() {
		log.Debug.Println("appendSourceObjectsToDest begin")
	}

	for objNr, entry := range ctxSrc.Table {
		// Do not copy free list head.
		if objNr == 0 {
			continue
		}

		if log.DebugEnabled() {
			log.Debug.Printf("adding obj %d from src to dest\n", objNr)
		}

		if entry == nil {
			continue
		}
		ctxDest.Table[objNr] = entry
		*ctxDest.Size++
	}

	if log.DebugEnabled() {
		log.Debug.Println("appendSourceObjectsToDest end")
	}
	return nil
}

// merge two disjunct IntSets
func mergeIntSets(src, dest types.IntSet) {
	if dest == nil {
		return
	}
	for k := range src {
		dest[k] = true
	}
}

func mergeDuplicateObjNumberIntSets(ctxSrc, ctxDest *model.Context) {
	if ctxSrc == nil || ctxDest == nil || ctxSrc.Optimize == nil || ctxDest.Optimize == nil || ctxSrc.Read == nil || ctxDest.Read == nil {
		return
	}

	if log.DebugEnabled() {
		log.Debug.Println("mergeDuplicateObjNumberIntSets begin")
	}

	mergeIntSets(ctxSrc.Optimize.DuplicateInfoObjects, ctxDest.Optimize.DuplicateInfoObjects)
	mergeIntSets(ctxSrc.LinearizationObjs, ctxDest.LinearizationObjs)
	mergeIntSets(ctxSrc.Read.XRefStreams, ctxDest.Read.XRefStreams)
	mergeIntSets(ctxSrc.Read.ObjectStreams, ctxDest.Read.ObjectStreams)

	if log.DebugEnabled() {
		log.Debug.Println("mergeDuplicateObjNumberIntSets end")
	}
}

func mergeConfiguredOutlines(fName string, origDestPageCount int, ctxSrc, ctxDest *model.Context, zip bool) error {
	if ctxDest == nil || ctxDest.Configuration == nil {
		return errors.New("merge configured outlines: missing destination configuration")
	}
	if zip || !ctxDest.Configuration.CreateBookmarks {
		return nil
	}

	if ctxDest.Configuration.MergeBookmarkMode == model.MergeBookmarkModePreserve {
		return mergeOutlinesPreserve(ctxSrc, ctxDest)
	}

	return mergeOutlinesWrapped(fName, origDestPageCount+1, ctxSrc, ctxDest)
}

func mergeSourcePageTree(ctxSrc, ctxDest *model.Context, zip, dividerPage bool) error {
	if zip {
		if err := zipSourcePageTreeIntoDestPageTree(ctxSrc, ctxDest); err != nil {
			return fmt.Errorf("zip source pages: %w", err)
		}
		return nil
	}
	if err := appendSourcePageTreeToDestPageTree(ctxSrc, ctxDest, dividerPage); err != nil {
		return fmt.Errorf("append source pages: %w", err)
	}
	return nil
}

func freeMergedSourceObjects(ctxSrc, ctxDest *model.Context) error {
	if ctxSrc == nil || ctxDest == nil {
		return errors.New("free merged source objects: missing context")
	}
	if ctxSrc.Root == nil {
		return errors.New("free merged source objects: missing source root")
	}

	if err := ctxDest.FreeObject(int(ctxSrc.Root.ObjectNumber)); err != nil {
		return fmt.Errorf("free source root: %w", err)
	}

	if ctxSrc.Info != nil {
		if err := ctxDest.FreeObject(int(ctxSrc.Info.ObjectNumber)); err != nil {
			return fmt.Errorf("free source info: %w", err)
		}
	}

	return nil
}

// MergeXRefTables merges Context ctxSrc into ctxDest by appending its page tree.
// zip         ... zip 2 files together (eg. 1A,1B,2A,2B,3A,3B...)
// dividerPage ... insert blank page between merged files (not applicable for zipping)
func MergeXRefTables(fName string, ctxSrc, ctxDest *model.Context, zip, dividerPage bool) (err error) {
	if ctxSrc == nil || ctxDest == nil {
		return errors.New("merge: missing context")
	}

	origDestPageCount := ctxDest.PageCount

	if err = patchSourceObjectNumbers(ctxSrc, ctxDest); err != nil {
		return fmt.Errorf("merge: patch source object numbers: %w", err)
	}
	if err = renameSourceOrphanWidgetFields(ctxSrc, fmt.Sprintf("%d", origDestPageCount)); err != nil {
		return fmt.Errorf("merge forms: rename orphan widgets: %w", err)
	}
	if err = appendSourceObjectsToDest(ctxSrc, ctxDest); err != nil {
		return fmt.Errorf("merge: append source objects: %w", err)
	}

	if dividerPage {
		origDestPageCount++
	}

	if err = mergeSourcePageTree(ctxSrc, ctxDest, zip, dividerPage); err != nil {
		return fmt.Errorf("merge page tree: %w", err)
	}

	if err = mergeForms(ctxSrc, ctxDest); err != nil {
		return fmt.Errorf("merge forms: %w", err)
	}

	if err = mergeDests(ctxSrc, ctxDest); err != nil {
		return fmt.Errorf("merge dests: %w", err)
	}

	if err = mergeNames(ctxSrc, ctxDest); err != nil {
		return fmt.Errorf("merge names: %w", err)
	}

	if err = mergeConfiguredOutlines(fName, origDestPageCount, ctxSrc, ctxDest, zip); err != nil {
		return fmt.Errorf("merge outlines: %w", err)
	}

	if err = freeMergedSourceObjects(ctxSrc, ctxDest); err != nil {
		return fmt.Errorf("merge: %w", err)
	}

	// Merge all IntSets containing redundant object numbers.
	mergeDuplicateObjNumberIntSets(ctxSrc, ctxDest)

	if log.InfoEnabled() {
		log.Info.Printf("Dest XRefTable after merge:\n%s\n", ctxDest)
	}
	return nil
}
