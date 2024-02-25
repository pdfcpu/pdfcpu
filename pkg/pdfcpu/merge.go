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
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func EnsureOutlines(ctx *model.Context, fName string, append bool) error {

	rootDict, err := ctx.Catalog()
	if err != nil {
		return err
	}

	if err := ctx.LocateNameTree("Dests", true); err != nil {
		return err
	}

	outlinesDict := types.Dict(map[string]types.Object{"Type": types.Name("Outlines")})
	indRef, err := ctx.IndRefForNewObject(outlinesDict)
	if err != nil {
		return err
	}

	first, last, total, visible, err := createOutlineItemDict(ctx, []Bookmark{{PageFrom: 1, Title: fName}}, indRef, nil)
	if err != nil {
		return err
	}

	outlinesDict["First"] = *first
	outlinesDict["Last"] = *last
	outlinesDict["Count"] = types.Integer(total + visible)

	if obj, ok := rootDict.Find("Outlines"); ok {
		if append {
			return nil
		}
		d, err := ctx.DereferenceDict(obj)
		if err != nil {
			return err
		}
		count := d.IntEntry("Count")
		c := 0
		f, l := d.IndirectRefEntry("First"), d.IndirectRefEntry("Last")
		for ir := f; ir != nil; ir = d.IndirectRefEntry("Next") {
			d, err = ctx.DereferenceDict(*ir)
			if err != nil {
				return err
			}
			d["Parent"] = *first
			c++
		}
		d, err = ctx.DereferenceDict(*first)
		if err != nil {
			return err
		}

		d["First"] = *f
		d["Last"] = *l
		if count != nil && *count != 0 {
			c = *count
		}
		d["Count"] = types.Integer(-c)
	}

	rootDict["Outlines"] = *indRef

	return nil
}

func mergeOutlines(fName string, p int, ctxSrc, ctxDest *model.Context) error {
	rootDictDest, _ := ctxDest.Catalog()
	indRef := rootDictDest.IndirectRefEntry("Outlines")
	outlinesDict, err := ctxDest.DereferenceDict(*indRef)
	if err != nil {
		return err
	}

	first, last, _, _, err := createOutlineItemDict(ctxDest, []Bookmark{{PageFrom: p, Title: fName}}, indRef, nil)
	if err != nil {
		return err
	}

	l := outlinesDict.IndirectRefEntry("Last")
	outlinesDict["Last"] = *last

	topCount := 0

	count := outlinesDict.IntEntry("Count")
	if count != nil {
		topCount = *count
	}

	topCount++

	d1, err := ctxDest.DereferenceDict(*l)
	if err != nil {
		return err
	}
	d1["Next"] = *last

	d2, err := ctxDest.DereferenceDict(*last)
	if err != nil {
		return err
	}
	d2["Previous"] = *l

	rootDictSource, err := ctxSrc.Catalog()
	if err != nil {
		return err
	}

	if obj, ok := rootDictSource.Find("Outlines"); ok {

		// Integrate existing outlines from ctxSource.

		d, err := ctxDest.DereferenceDict(obj)
		if err != nil {
			return err
		}

		f, l := d.IndirectRefEntry("First"), d.IndirectRefEntry("Last")
		if f == nil && l == nil {
			outlinesDict["Count"] = types.Integer(topCount)
			return nil
		}

		d2["First"] = *f
		d2["Last"] = *l

		c := 0

		// Update parents.
		// TODO Collapse outline dicts.
		for ir := f; ir != nil; ir = d.IndirectRefEntry("Next") {
			d, err = ctxDest.DereferenceDict(*ir)
			if err != nil {
				return err
			}
			d["Parent"] = *first

			i := d.IntEntry("Count")
			if i != nil && *i > 0 {
				c += *i
			}

			c++
		}

		d2["Count"] = types.Integer(c)
		topCount += c
	}

	outlinesDict["Count"] = types.Integer(topCount)
	return nil
}

func handleNeedAppearances(ctxSrc *model.Context, dSrc, dDest types.Dict) error {
	o, found := dSrc.Find("NeedAppearances")
	if !found || o == nil {
		return nil
	}
	b, err := ctxSrc.DereferenceBoolean(o, model.V10)
	if err != nil {
		return err
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
		return err
	}
	o, found = dDest.Find("CO")
	if !found {
		dDest["CO"] = arrSrc
		return nil
	}
	arrDest, err := ctxDest.DereferenceArray(o)
	if err != nil {
		return err
	}
	if len(arrDest) == 0 {
		dDest["CO"] = arrSrc
	} else {
		arrDest = append(arrDest, arrSrc...)
		dDest["CO"] = arrDest
	}
	return nil
}

func handleDR(ctxSrc, ctxDest *model.Context, dSrc, dDest types.Dict) error {
	o, found := dSrc.Find("DR")
	if !found {
		return nil
	}
	dSrc, err := ctxSrc.DereferenceDict(o)
	if err != nil {
		return err
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
			return err
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
			return err
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
		return err
	}

	// SigFlags: set bit 1 to true only (SignaturesExist)
	//           set bit 2 to true only (AppendOnly)
	dDest.Delete("SigFields")

	// CO: add all indrefs
	if err := handleCO(ctxSrc, ctxDest, dSrc, dDest); err != nil {
		return err
	}

	// DR: default resource dict
	if err := handleDR(ctxSrc, ctxDest, dSrc, dDest); err != nil {
		return err
	}

	// DA: default appearance streams for variable text fields
	if err := handleDA(ctxSrc, dSrc, dDest, arrFieldsSrc); err != nil {
		return err
	}

	// Q: left, center, right for variable text fields
	if err := handleQ(ctxSrc, dSrc, dDest, arrFieldsSrc); err != nil {
		return err
	}

	// XFA: ignore
	delete(dDest, "XFA")

	return nil
}

func rootDicts(ctxSrc, ctxDest *model.Context) (types.Dict, types.Dict, error) {
	rootDictSource, err := ctxSrc.Catalog()
	if err != nil {
		return nil, nil, err
	}

	rootDictDest, err := ctxDest.Catalog()
	if err != nil {
		return nil, nil, err
	}

	return rootDictSource, rootDictDest, nil
}

func mergeInFields(ctxDest *model.Context, arrFieldsSrc, arrFieldsDest types.Array, dDest types.Dict) error {
	parentDict :=
		types.Dict(map[string]types.Object{
			"Kids": arrFieldsSrc,
			"T":    types.StringLiteral(fmt.Sprintf("%d", len(arrFieldsDest))),
		})

	ir, err := ctxDest.IndRefForNewObject(parentDict)
	if err != nil {
		return err
	}

	for _, ir1 := range arrFieldsSrc {
		d, err := ctxDest.DereferenceDict(ir1)
		if err != nil {
			return err
		}
		if len(d) == 0 {
			continue
		}
		d["Parent"] = *ir
	}

	dDest["Fields"] = append(arrFieldsDest, *ir)

	return nil
}

func mergeDests(ctxSource, ctxDest *model.Context) error {
	rootDictSource, rootDictDest, err := rootDicts(ctxSource, ctxDest)
	if err != nil {
		return err
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
		return err
	}

	destsDest, err := ctxDest.DereferenceDict(o2)
	if err != nil {
		return err
	}

	// Note: We ignore duplicate keys
	for k, v := range destsSrc {
		destsDest[k] = v
	}

	return nil
}

func mergeNames(ctxSrc, ctxDest *model.Context) error {

	rootDictSrc, rootDictDest, err := rootDicts(ctxSrc, ctxDest)
	if err != nil {
		return err
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

	// We need to merge src Names into dest Names.

	for id, namesSrc := range ctxSrc.Names {
		if namesDest, ok := ctxDest.Names[id]; ok {
			// Merge src tree into dest tree including collision detection.
			if err := namesDest.AddTree(ctxDest.XRefTable, namesSrc, ctxSrc.NameRefs[id], []string{"D", "Dest"}); err != nil {
				return err
			}
			continue
		}

		// Name tree missing in dest ctx => copy over names from src ctx
		ctxDest.Names[id] = namesSrc
	}

	return nil
}

func mergeForms(ctxSrc, ctxDest *model.Context) error {

	rootDictSource, rootDictDest, err := rootDicts(ctxSrc, ctxDest)
	if err != nil {
		return err
	}

	o, found := rootDictSource.Find("AcroForm")
	if !found {
		return nil
	}

	dSrc, err := ctxSrc.DereferenceDict(o)
	if err != nil || len(dSrc) == 0 {
		return err
	}

	// Retrieve ctxSrc Form Fields
	o, found = dSrc.Find("Fields")
	if !found {
		return nil
	}
	arrFieldsSrc, err := ctxSrc.DereferenceArray(o)
	if err != nil {
		return err
	}
	if len(arrFieldsSrc) == 0 {
		return nil
	}

	// We have a ctxSrc.Form with fields.

	o, found = rootDictDest.Find("AcroForm")
	if !found {
		rootDictDest["AcroForm"] = dSrc
		return nil
	}

	dDest, err := ctxDest.DereferenceDict(o)
	if err != nil {
		return err
	}

	if len(dDest) == 0 {
		rootDictDest["AcroForm"] = dSrc
		return nil
	}

	// Retrieve ctxDest AcroForm Fields
	o, found = dDest.Find("Fields")
	if !found {
		rootDictDest["AcroForm"] = dSrc
		return nil
	}
	arrFieldsDest, err := ctxDest.DereferenceArray(o)
	if err != nil {
		return err
	}
	if len(arrFieldsDest) == 0 {
		rootDictDest["AcroForm"] = dSrc
		return nil
	}

	if err := mergeInFields(ctxDest, arrFieldsSrc, arrFieldsDest, dDest); err != nil {
		return err
	}

	return handleFormAttributes(ctxSrc, ctxDest, dSrc, dDest, arrFieldsSrc)
}

func patchIndRef(ir *types.IndirectRef, lookup map[int]int) {
	i := ir.ObjectNumber.Value()
	ir.ObjectNumber = types.Integer(lookup[i])
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
			t[lookup[k]] = v
		}
	}

	return t
}

func patchNameTree(n *model.Node, lookup map[int]int) error {

	patchValues := func(xRefTable *model.XRefTable, k string, v *types.Object) error {
		*v = patchObject(*v, lookup)
		return nil
	}

	return n.Process(nil, patchValues)
}

func patchSourceObjectNumbers(ctxSrc, ctxDest *model.Context) {
	if log.DebugEnabled() {
		log.Debug.Printf("patchSourceObjectNumbers:  ctxSrc: xRefTableSize:%d trailer.Size:%d - %s\n", len(ctxSrc.Table), *ctxSrc.Size, ctxSrc.Read.FileName)
		log.Debug.Printf("patchSourceObjectNumbers: ctxDest: xRefTableSize:%d trailer.Size:%d - %s\n", len(ctxDest.Table), *ctxDest.Size, ctxDest.Read.FileName)
	}

	// Patch source xref tables obj numbers which are essentially the keys.
	//logInfoMerge.Printf("Source XRefTable before:\n%s\n", ctxSource)

	objNrs := objNrsIntSet(ctxSrc)

	// Create lookup table for object numbers.
	// The first number is the successor of the last number in ctxDest.
	lookup := lookupTable(objNrs, *ctxDest.Size)

	// Patch pointer to root object
	patchIndRef(ctxSrc.Root, lookup)

	// Patch pointer to info object
	if ctxSrc.Info != nil {
		patchIndRef(ctxSrc.Info, lookup)
	}

	// Patch free object zero
	entry := ctxSrc.Table[0]
	off := int(*entry.Offset)
	if off != 0 {
		i := int64(lookup[off])
		entry.Offset = &i
	}

	// Patch all indRefs for xref table entries.
	for k := range objNrs {

		//logDebugMerge.Printf("patching obj #%d\n", k)

		entry := ctxSrc.Table[k]

		if entry.Free {
			if log.DebugEnabled() {
				log.Debug.Printf("patch free entry: old offset:%d\n", *entry.Offset)
			}
			off := int(*entry.Offset)
			if off == 0 {
				continue
			}
			i := int64(lookup[off])
			entry.Offset = &i
			if log.DebugEnabled() {
				log.Debug.Printf("patch free entry: new offset:%d\n", *entry.Offset)
			}
			continue
		}

		patchObject(entry.Object, lookup)
	}

	// Patch xref entry object numbers.
	m := make(map[int]*model.XRefTableEntry, *ctxSrc.Size)
	for k, v := range lookup {
		m[v] = ctxSrc.Table[k]
	}
	m[0] = ctxSrc.Table[0]
	ctxSrc.Table = m

	// Patch DuplicateInfo object numbers.
	ctxSrc.Optimize.DuplicateInfoObjects = patchObjects(ctxSrc.Optimize.DuplicateInfoObjects, lookup)

	// Patch Linearization object numbers.
	ctxSrc.LinearizationObjs = patchObjects(ctxSrc.LinearizationObjs, lookup)

	// Patch XRefStream objects numbers.
	ctxSrc.Read.XRefStreams = patchObjects(ctxSrc.Read.XRefStreams, lookup)

	// Patch object stream object numbers.
	ctxSrc.Read.ObjectStreams = patchObjects(ctxSrc.Read.ObjectStreams, lookup)

	// Patch cached name trees.
	for _, v := range ctxSrc.Names {
		patchNameTree(v, lookup)
	}

	if log.DebugEnabled() {
		log.Debug.Printf("patchSourceObjectNumbers end")
	}
}

func createDividerPagesDict(ctx *model.Context, parentIndRef types.IndirectRef) (*types.IndirectRef, error) {
	d := types.Dict(
		map[string]types.Object{
			"Type":   types.Name("Pages"),
			"Parent": parentIndRef,
			"Count":  types.Integer(1),
		},
	)

	indRef, err := ctx.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	dims, err := ctx.XRefTable.PageDims()
	if err != nil {
		return nil, err
	}

	last := len(dims) - 1
	mediaBox := types.NewRectangle(0, 0, dims[last].Width, dims[last].Height)

	indRefPageDict, err := ctx.EmptyPage(indRef, mediaBox)
	if err != nil {
		return nil, err
	}
	ctx.SetValid(*indRefPageDict)

	d.Insert("Kids", types.Array{*indRefPageDict})

	return indRef, nil
}

func appendSourcePageTreeToDestPageTree(ctxSrc, ctxDest *model.Context, dividerPage bool) error {
	if log.DebugEnabled() {
		log.Debug.Println("appendSourcePageTreeToDestPageTree begin")
	}

	indRefPageTreeRootDictDest, err := ctxDest.Pages()
	if err != nil {
		return err
	}

	pageTreeRootDictDest, err := ctxDest.XRefTable.DereferenceDict(*indRefPageTreeRootDictDest)
	if err != nil {
		return err
	}

	pageCountDest := pageTreeRootDictDest.IntEntry("Count")
	if pageCountDest == nil || *pageCountDest != ctxDest.PageCount {
		return errors.Errorf("pdfcpu: corrupt page node at obj #%d\n", indRefPageTreeRootDictDest.ObjectNumber)
	}

	c := ctxDest.PageCount

	d := types.NewDict()
	d.InsertName("Type", "Pages")
	kids := types.Array{*indRefPageTreeRootDictDest}

	indRef, err := ctxDest.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	if dividerPage {
		dividerPagesNodeIndRef, err := createDividerPagesDict(ctxDest, *indRef)
		if err != nil {
			return err
		}
		kids = append(kids, *dividerPagesNodeIndRef)
		c++
	}

	pageTreeRootDictDest["Parent"] = *indRef

	indRefPageTreeRootDictSource, err := ctxSrc.Pages()
	if err != nil {
		return err
	}

	d.Insert("Kids", append(kids, *indRefPageTreeRootDictSource))

	pageTreeRootDictSource, err := ctxSrc.XRefTable.DereferenceDict(*indRefPageTreeRootDictSource)
	if err != nil {
		return err
	}

	pageTreeRootDictSource["Parent"] = *indRef

	pageCountSource := pageTreeRootDictSource.IntEntry("Count")
	if pageCountSource == nil || *pageCountSource != ctxSrc.PageCount {
		return errors.Errorf("pdfcpu: corrupt page node at obj #%d\n", indRefPageTreeRootDictSource.ObjectNumber)
	}

	c += ctxSrc.PageCount
	d.InsertInt("Count", c)
	ctxDest.PageCount = c

	rootDict, err := ctxDest.Catalog()
	if err != nil {
		return err
	}

	rootDict["Pages"] = *indRef

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
		return err
	}

	// Process dest page tree recursively and weave in src pages
	p := 0
	if ctxDest.PageCount, err = ctxDest.InsertPages(rootPageIndRef, &p, ctxSrc); err != nil {
		return err
	}

	if appendFromPageNr > 0 {
		// append remaining src pages
		if ctxDest.PageCount, err = ctxDest.AppendPages(rootPageIndRef, appendFromPageNr, ctxSrc); err != nil {
			return err
		}
	}

	if log.DebugEnabled() {
		log.Debug.Println("zipSourcePageTreeIntoDestPageTree end")
	}

	return nil
}

func appendSourceObjectsToDest(ctxSrc, ctxDest *model.Context) {
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

		ctxDest.Table[objNr] = entry

		*ctxDest.Size++

	}

	if log.DebugEnabled() {
		log.Debug.Println("appendSourceObjectsToDest end")
	}
}

// merge two disjunct IntSets
func mergeIntSets(src, dest types.IntSet) {
	for k := range src {
		dest[k] = true
	}
}

func mergeDuplicateObjNumberIntSets(ctxSrc, ctxDest *model.Context) {
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

// MergeXRefTables merges Context ctxSrc into ctxDest by appending its page tree.
// zip         ... zip 2 files together (eg. 1A,1B,2A,2B,3A,3B...)
// dividerPage ... insert blank page between merged files (not applicable for zipping)
func MergeXRefTables(fName string, ctxSrc, ctxDest *model.Context, zip, dividerPage bool) (err error) {

	patchSourceObjectNumbers(ctxSrc, ctxDest)

	appendSourceObjectsToDest(ctxSrc, ctxDest)

	origDestPageCount := ctxDest.PageCount
	if dividerPage {
		origDestPageCount++
	}

	if zip {
		err = zipSourcePageTreeIntoDestPageTree(ctxSrc, ctxDest)
	} else {
		err = appendSourcePageTreeToDestPageTree(ctxSrc, ctxDest, dividerPage)
	}

	if err != nil {
		return nil
	}

	if err = mergeForms(ctxSrc, ctxDest); err != nil {
		return err
	}

	if err = mergeDests(ctxSrc, ctxDest); err != nil {
		return err
	}

	if err = mergeNames(ctxSrc, ctxDest); err != nil {
		return err
	}

	if !zip && ctxDest.Configuration.CreateBookmarks {
		if err = mergeOutlines(fName, origDestPageCount+1, ctxSrc, ctxDest); err != nil {
			return err
		}
	}

	// Mark src's root object as free.
	if err = ctxDest.FreeObject(int(ctxSrc.Root.ObjectNumber)); err != nil {
		return
	}

	// Mark source's info object as free.
	// Note: Any indRefs this info object depends on are missed.
	if ctxSrc.Info != nil {
		if err = ctxDest.FreeObject(int(ctxSrc.Info.ObjectNumber)); err != nil {
			return
		}
	}

	// Merge all IntSets containing redundant object numbers.
	mergeDuplicateObjNumberIntSets(ctxSrc, ctxDest)

	if log.InfoEnabled() {
		log.Info.Printf("Dest XRefTable after merge:\n%s\n", ctxDest)
	}

	return nil
}
