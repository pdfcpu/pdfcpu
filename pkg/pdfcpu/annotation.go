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

package pdfcpu

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// CachedAnnotationObjNrs returns a list of object numbers representing known annotation dict indirect references.
func CachedAnnotationObjNrs(ctx *model.Context) ([]int, error) {
	// Note: Not all cached annotations are based on IndRefs!
	// pdfcpu also caches direct annot dict objects (violating the PDF spec) for listing purposes.
	// Such annotations may only be removed as part of removing all annotations (for a page).

	objNrs := []int{}

	for _, pageAnnots := range ctx.PageAnnots {
		for _, annots := range pageAnnots {
			for objNr := range annots.Map {
				objNrs = append(objNrs, objNr)
			}
		}
	}

	return objNrs, nil
}

func sortedPageNrsForAnnotsFromCache(ctx *model.Context) []int {
	var pageNrs []int
	for k := range ctx.PageAnnots {
		pageNrs = append(pageNrs, k)
	}
	sort.Ints(pageNrs)
	return pageNrs
}

func addAnnotationToCache(ctx *model.Context, ann model.AnnotationRenderer, pageNr, objNr int) error {
	pgAnnots, ok := ctx.PageAnnots[pageNr]
	if !ok {
		pgAnnots = model.PgAnnots{}
		ctx.PageAnnots[pageNr] = pgAnnots
	}
	annots, ok := pgAnnots[ann.Type()]
	if !ok {
		annots = model.Annot{}
		annots.Map = model.AnnotMap{}
		pgAnnots[ann.Type()] = annots
	}
	if _, ok := annots.Map[objNr]; ok {
		return errors.Errorf("addAnnotation: obj#%d already cached", objNr)
	}
	annots.Map[objNr] = ann
	return nil
}

func removeAnnotationFromCache(ctx *model.Context, pageNr, objNr int) error {
	pgAnnots, ok := ctx.PageAnnots[pageNr]
	if !ok {
		return errors.Errorf("removeAnnotation: no page annotations cached for page %d", pageNr)
	}
	for annType, annots := range pgAnnots {
		if _, ok := annots.Map[objNr]; ok {
			delete(annots.Map, objNr)
			if len(annots.Map) == 0 {
				delete(pgAnnots, annType)
				if len(pgAnnots) == 0 {
					delete(ctx.PageAnnots, pageNr)
				}
			}
			return nil
		}
	}
	return errors.Errorf("removeAnnotation: no page annotation cached for obj#%d", objNr)
}

func findAnnotByID(ctx *model.Context, id string, annots types.Array) (int, error) {
	for i, o := range annots {
		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return -1, err
		}
		s := d.StringEntry("NM")
		if s == nil {
			continue
		}
		if *s == id {
			return i, nil
		}
	}
	return -1, nil
}

func findAnnotByObjNr(ctx *model.Context, objNr int, annots types.Array) (int, error) {
	for i, o := range annots {
		indRef, _ := o.(types.IndirectRef)
		if indRef.ObjectNumber.Value() == objNr {
			return i, nil
		}
	}
	return -1, nil
}

func createAnnot(ctx *model.Context, ar model.AnnotationRenderer, pageIndRef *types.IndirectRef) (*types.IndirectRef, error) {
	d, err := ar.RenderDict(ctx.XRefTable, *pageIndRef)
	if err != nil {
		return nil, err
	}
	return ctx.IndRefForNewObject(d)
}

// Annotation returns an annotation renderer.
// Validation sets up a cache of annotation renderers.
func Annotation(xRefTable *model.XRefTable, d types.Dict) (model.AnnotationRenderer, error) {

	subtype := d.NameEntry("Subtype")

	o, _ := d.Find("Rect")
	arr, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return nil, err
	}

	r, err := types.RectForArray(arr)
	if err != nil {
		return nil, err
	}

	bb, err := d.StringEntryBytes("Contents")
	if err != nil {
		return nil, err
	}
	contents := string(bb)

	var nm string
	s := d.StringEntry("NM") // This is what pdfcpu refers to as the annotation id.
	if s != nil {
		nm = *s
	}

	var f model.AnnotationFlags
	i := d.IntEntry("F")
	if i != nil {
		f = model.AnnotationFlags(*i)
	}

	var ann model.AnnotationRenderer

	switch *subtype {

	case "Text":
		ann = model.NewTextAnnotation(*r, contents, nm, "", f, nil, nil, "", "", true, "")

	case "Link":
		var uri string
		o, found := d.Find("A")
		if found && o != nil {
			d, err := xRefTable.DereferenceDict(o)
			if err != nil {
				return nil, err
			}

			bb, err := xRefTable.DereferenceStringEntryBytes(d, "URI")
			if err != nil {
				return nil, err
			}
			if len(bb) > 0 {
				uri = string(bb)
			}
		}
		dest := (*model.Destination)(nil) // will not collect link dest during validation.
		ann = model.NewLinkAnnotation(*r, nil, dest, uri, nm, f, nil, false)

	case "Popup":
		parentIndRef := d.IndirectRefEntry("Parent")
		ann = model.NewPopupAnnotation(*r, nil, contents, nm, f, nil, parentIndRef)

	// TODO handle remaining annotation types.

	default:
		ann = model.NewAnnotationForRawType(*subtype, *r, contents, nil, nm, f, nil)
	}

	return ann, nil
}

func AnnotationsForSelectedPages(ctx *model.Context, selectedPages types.IntSet) map[int]model.PgAnnots {

	var pageNrs []int
	for k := range ctx.PageAnnots {
		pageNrs = append(pageNrs, k)
	}
	sort.Ints(pageNrs)

	m := map[int]model.PgAnnots{}

	for _, i := range pageNrs {

		if selectedPages != nil {
			if _, found := selectedPages[i]; !found {
				continue
			}
		}

		pageAnnots := ctx.PageAnnots[i]
		if len(pageAnnots) == 0 {
			continue
		}

		m[i] = pageAnnots
	}

	return m
}

// ListAnnotations returns a formatted list of annotations.
func ListAnnotations(annots map[int]model.PgAnnots) (int, []string, error) {
	var (
		j       int
		pageNrs []int
	)
	ss := []string{}

	for k := range annots {
		pageNrs = append(pageNrs, k)
	}
	sort.Ints(pageNrs)

	for _, i := range pageNrs {

		pageAnnots := annots[i]

		var annTypes []string
		for t := range pageAnnots {
			annTypes = append(annTypes, model.AnnotTypeStrings[t])
		}
		sort.Strings(annTypes)

		ss = append(ss, "")
		ss = append(ss, fmt.Sprintf("Page %d:", i))

		for _, annType := range annTypes {
			annots := pageAnnots[model.AnnotTypes[annType]]
			var (
				maxLenRect    int
				maxLenContent int
			)
			maxLenID := 2
			var objNrs []int
			for objNr, ann := range annots.Map {
				objNrs = append(objNrs, objNr)
				if len(ann.RectString()) > maxLenRect {
					maxLenRect = len(ann.RectString())
				}
				if len(ann.ID()) > maxLenID {
					maxLenID = len(ann.ID())
				}
				if len(ann.ContentString()) > maxLenContent {
					maxLenContent = len(ann.ContentString())
				}
			}
			sort.Ints(objNrs)
			ss = append(ss, "")
			ss = append(ss, fmt.Sprintf("  %s:", annType))
			s1 := ("     obj# ")
			s2 := fmt.Sprintf("%%%ds", maxLenRect)
			s3 := fmt.Sprintf("%%%ds", maxLenID)
			s4 := fmt.Sprintf("%%%ds", maxLenContent)
			s := fmt.Sprintf(s1+s2+" "+s3+" "+s4, "rect", "id", "content")
			ss = append(ss, s)
			ss = append(ss, "    "+strings.Repeat("=", len(s)-4))
			for _, objNr := range objNrs {
				ann := annots.Map[objNr]
				ss = append(ss, fmt.Sprintf("    %5d "+s2+" "+s3+" "+s4, objNr, ann.RectString(), ann.ID(), ann.ContentString()))
				j++
			}
		}
	}

	return j, append([]string{fmt.Sprintf("%d annotations available", j)}, ss...), nil
}

func addAnnotationToDirectObj(
	ctx *model.Context,
	annots types.Array,
	annotIndRef, pageDictIndRef *types.IndirectRef,
	pageDict types.Dict,
	pageNr int,
	ar model.AnnotationRenderer,
	incr bool) (bool, error) {

	i, err := findAnnotByID(ctx, ar.ID(), annots)
	if err != nil {
		return false, err
	}
	if i >= 0 {
		return false, errors.Errorf("page %d: duplicate annotation with id:%s\n", pageNr, ar.ID())
	}
	pageDict.Update("Annots", append(annots, *annotIndRef))
	if incr {
		// Mark page dict obj for incremental writing.
		ctx.Write.IncrementWithObjNr(pageDictIndRef.ObjectNumber.Value())
	}
	ctx.EnsureVersionForWriting()
	return true, nil
}

// AddAnnotation adds ar to pageDict.
func AddAnnotation(
	ctx *model.Context,
	pageDictIndRef *types.IndirectRef,
	pageDict types.Dict,
	pageNr int,
	ar model.AnnotationRenderer,
	incr bool) (bool, error) {

	// Create xreftable entry for annotation.
	annotIndRef, err := createAnnot(ctx, ar, pageDictIndRef)
	if err != nil {
		return false, err
	}

	// Add annotation to xreftable page annotation cache.
	err = addAnnotationToCache(ctx, ar, pageNr, annotIndRef.ObjectNumber.Value())
	if err != nil {
		return false, err
	}

	if incr {
		// Mark new annotaton dict obj for incremental writing.
		ctx.Write.IncrementWithObjNr(annotIndRef.ObjectNumber.Value())
	}

	obj, found := pageDict.Find("Annots")
	if !found {
		pageDict.Insert("Annots", types.Array{*annotIndRef})
		if incr {
			// Mark page dict obj for incremental writing.
			ctx.Write.IncrementWithObjNr(pageDictIndRef.ObjectNumber.Value())
		}
		ctx.EnsureVersionForWriting()
		return true, nil
	}

	ir, ok := obj.(types.IndirectRef)
	if !ok {
		return addAnnotationToDirectObj(ctx, obj.(types.Array), annotIndRef, pageDictIndRef, pageDict, pageNr, ar, incr)
	}

	// Annots array is an IndirectReference.

	o, err := ctx.Dereference(ir)
	if err != nil || o == nil {
		return false, err
	}

	annots, _ := o.(types.Array)
	i, err := findAnnotByID(ctx, ar.ID(), annots)
	if err != nil {
		return false, err
	}
	if i >= 0 {
		return false, errors.Errorf("page %d: duplicate annotation with id:%s\n", pageNr, ar.ID())
	}

	entry, ok := ctx.FindTableEntryForIndRef(&ir)
	if !ok {
		return false, errors.Errorf("page %d: can't dereference Annots indirect reference(obj#:%d)\n", pageNr, ir.ObjectNumber)
	}
	entry.Object = append(annots, *annotIndRef)
	if incr {
		// Mark Annot array obj for incremental writing.
		ctx.Write.IncrementWithObjNr(ir.ObjectNumber.Value())
	}

	ctx.EnsureVersionForWriting()
	return true, nil
}

// AddAnnotations adds ar to selected pages.
func AddAnnotations(ctx *model.Context, selectedPages types.IntSet, ar model.AnnotationRenderer, incr bool) (bool, error) {
	var ok bool
	if incr {
		ctx.Write.Increment = true
		ctx.Write.Offset = ctx.Read.FileSize
	}

	for k, v := range selectedPages {
		if !v {
			continue
		}
		if k > ctx.PageCount {
			return false, errors.Errorf("pdfcpu: invalid page number: %d", k)
		}

		pageDictIndRef, err := ctx.PageDictIndRef(k)
		if err != nil {
			return false, err
		}

		d, err := ctx.DereferenceDict(*pageDictIndRef)
		if err != nil {
			return false, err
		}

		added, err := AddAnnotation(ctx, pageDictIndRef, d, k, ar, incr)
		if err != nil {
			return false, err
		}
		if added {
			ok = true
		}
	}

	return ok, nil
}

// AddAnnotationsMap adds annotations in m to corresponding pages.
func AddAnnotationsMap(ctx *model.Context, m map[int][]model.AnnotationRenderer, incr bool) (bool, error) {
	var ok bool
	if incr {
		ctx.Write.Increment = true
		ctx.Write.Offset = ctx.Read.FileSize
	}
	for i, annots := range m {

		if i > ctx.PageCount {
			return false, errors.Errorf("pdfcpu: invalid page number: %d", i)
		}

		pageDictIndRef, err := ctx.PageDictIndRef(i)
		if err != nil {
			return false, err
		}

		d, err := ctx.DereferenceDict(*pageDictIndRef)
		if err != nil {
			return false, err
		}

		for _, annot := range annots {
			added, err := AddAnnotation(ctx, pageDictIndRef, d, i, annot, incr)
			if err != nil {
				return false, err
			}
			if added {
				ok = true
			}
		}

	}

	return ok, nil
}

func removeAllAnnotations(
	ctx *model.Context,
	pageDict types.Dict,
	pageDictObjNr,
	pageNr int,
	incr bool) (bool, error) {

	var err error
	obj, found := pageDict.Find("Annots")
	if !found {
		return false, nil
	}

	ir, ok := obj.(types.IndirectRef)
	if ok {
		obj, err = ctx.Dereference(ir)
		if err != nil || obj == nil {
			return false, err
		}
		objNr := ir.ObjectNumber.Value()
		if err := ctx.DeleteObject(ir); err != nil {
			return false, err
		}
		if incr {
			// Modify Annots array obj for incremental writing.
			ctx.Write.IncrementWithObjNr(objNr)
		}
	}
	annots, _ := obj.(types.Array)

	for _, o := range annots {
		if err := ctx.DeleteObject(o); err != nil {
			return false, err
		}
		ir, ok := o.(types.IndirectRef)
		if !ok {
			continue
		}
		objNr := ir.ObjectNumber.Value()
		if incr {
			// Mark annotation dict obj for incremental writing.
			ctx.Write.IncrementWithObjNr(objNr)
		}
	}

	pageDict.Delete("Annots")
	if incr {
		// Mark page dict obj for incremental writing.
		ctx.Write.IncrementWithObjNr(pageDictObjNr)
	}

	// Remove xref table page annotation cache.
	delete(ctx.PageAnnots, pageNr)

	ctx.EnsureVersionForWriting()

	return true, nil
}

func removeAnnotationsByType(
	ctx *model.Context,
	annotTypes []model.AnnotationType,
	pageNr int,
	annots types.Array,
	incr bool) (types.Array, bool, error) {

	pgAnnots, found := ctx.PageAnnots[pageNr]
	if !found {
		return annots, false, nil
	}

	var ok bool

	for _, annotType := range annotTypes {
		annot, found := pgAnnots[annotType]
		if !found {
			continue
		}
		// We have cached annotType page annotations.
		for _, indRef := range *annot.IndRefs {
			objNr := indRef.ObjectNumber.Value()
			i, err := findAnnotByObjNr(ctx, objNr, annots)
			if err != nil {
				return nil, false, err
			}
			if i < 0 {
				return nil, false, errors.New("pdfcpu: missing annot indRef")
			}
			if err := ctx.DeleteObject(indRef); err != nil {
				return nil, false, err
			}
			if incr {
				// Mark annotation dict obj for incremental writing.
				ctx.Write.IncrementWithObjNr(indRef.ObjectNumber.Value())
			}

			if len(annots) == 1 {
				annots = nil
				break
			}
			annots = append(annots[:i], annots[i+1:]...)
		}

		delete(pgAnnots, annotType)
		if len(pgAnnots) == 0 {
			delete(ctx.PageAnnots, pageNr)
		}

		ok = true
	}

	return annots, ok, nil
}

func removeAnnotationByID(
	ctx *model.Context,
	id string,
	pageNr int,
	annots types.Array,
	incr bool) (types.Array, bool, error) {

	i, err := findAnnotByID(ctx, id, annots)
	if err != nil || i < 0 {
		return annots, false, err
	}

	indRef, _ := annots[i].(types.IndirectRef)

	// Remove annotation from xreftable page annotation cache.
	err = removeAnnotationFromCache(ctx, pageNr, indRef.ObjectNumber.Value())
	if err != nil {
		return nil, false, err
	}
	if err := ctx.DeleteObject(indRef); err != nil {
		return nil, false, err
	}
	if incr {
		// Mark annotation dict obj for incremental writing.
		ctx.Write.IncrementWithObjNr(indRef.ObjectNumber.Value())
	}
	if len(annots) == 1 {
		if i != 0 {
			return nil, false, err
		}
		return nil, true, nil
	}
	annots = append(annots[:i], annots[i+1:]...)

	return annots, true, nil
}

func removeAnnotationsByID(
	ctx *model.Context,
	ids []string,
	objNrSet types.IntSet,
	pageNr int,
	annots types.Array,
	incr bool) (types.Array, bool, error) {

	var (
		ok, ok1 bool
		err     error
	)

	for _, id := range ids {
		annots, ok1, err = removeAnnotationByID(ctx, id, pageNr, annots, incr)
		if err != nil {
			return nil, false, err
		}
		if ok1 {
			ok = true
		}
	}

	for objNr, v := range objNrSet {
		if !v {
			continue
		}
		annots, ok1, err = removeAnnotationByID(ctx, strconv.Itoa(objNr), pageNr, annots, incr)
		if err != nil {
			return nil, false, err
		}
		if ok1 {
			delete(objNrSet, objNr)
			ok = true
		}
	}

	return annots, ok, nil
}

func removeAnnotationsByObjNr(
	ctx *model.Context,
	objNrSet types.IntSet,
	pageNr int,
	annots types.Array,
	incr bool) (types.Array, bool, error) {

	var ok bool
	for objNr, v := range objNrSet {
		if !v || objNr < 0 {
			continue
		}
		i, err := findAnnotByObjNr(ctx, objNr, annots)
		if err != nil {
			return nil, false, err
		}
		if i >= 0 {
			ok = true
			indRef, _ := annots[i].(types.IndirectRef)

			// Remove annotation from xreftable page annotation cache.
			err = removeAnnotationFromCache(ctx, pageNr, indRef.ObjectNumber.Value())
			if err != nil {
				return nil, false, err
			}

			if err := ctx.DeleteObject(indRef); err != nil {
				return nil, false, err
			}
			if incr {
				// Mark annotation dict obj for incremental writing.
				ctx.Write.IncrementWithObjNr(indRef.ObjectNumber.Value())
			}
			delete(objNrSet, objNr)
			if len(annots) == 1 {
				if i != 0 {
					return nil, false, err
				}
				return nil, ok, nil
			}
			annots = append(annots[:i], annots[i+1:]...)
		}
	}
	return annots, ok, nil
}

func removeAnnotationsFromAnnots(
	ctx *model.Context,
	annotTypes []model.AnnotationType,
	ids []string,
	objNrSet types.IntSet,
	pageNr int,
	annots types.Array,
	incr bool) (types.Array, bool, error) {

	var (
		ok1, ok2, ok3 bool
		err           error
	)

	// 1. Remove by annotType.
	if len(annotTypes) > 0 {
		annots, ok1, err = removeAnnotationsByType(ctx, annotTypes, pageNr, annots, incr)
		if err != nil || annots == nil {
			return nil, ok1, err
		}
	}

	// 2. Remove by obj#.
	if len(objNrSet) > 0 {
		annots, ok2, err = removeAnnotationsByObjNr(ctx, objNrSet, pageNr, annots, incr)
		if err != nil || annots == nil {
			return nil, ok2, err
		}
	}

	// 3. Remove by id for ids and objNrs considering possibly numeric ids.
	if len(ids) > 0 || len(objNrSet) > 0 {
		annots, ok3, err = removeAnnotationsByID(ctx, ids, objNrSet, pageNr, annots, incr)
		if err != nil || annots == nil {
			return nil, ok3, err
		}
	}

	return annots, ok1 || ok2 || ok3, nil
}

func removeAnnotationsFromIndAnnots(ctx *model.Context,
	annotTypes []model.AnnotationType,
	ids []string,
	objNrSet types.IntSet,
	pageNr int,
	annots types.Array,
	incr bool,
	pageDict types.Dict,
	pageDictObjNr int,
	indRef types.IndirectRef) (bool, error) {

	ann, ok, err := removeAnnotationsFromAnnots(ctx, annotTypes, ids, objNrSet, pageNr, annots, incr)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	objNr := indRef.ObjectNumber.Value()
	genNr := indRef.GenerationNumber.Value()
	entry, _ := ctx.FindTableEntry(objNr, genNr)

	if incr {
		// Modify Annots array obj for incremental writing.
		ctx.Write.IncrementWithObjNr(objNr)
	}

	ctx.EnsureVersionForWriting()

	if annots == nil {
		pageDict.Delete("Annots")
		if err := ctx.DeleteObject(indRef); err != nil {
			return false, err
		}
		if incr {
			// Mark page dict obj for incremental writing.
			ctx.Write.IncrementWithObjNr(pageDictObjNr)
		}
		return ok, nil
	}

	entry.Object = ann
	return true, nil
}

// RemoveAnnotationsFromPageDict removes an annotation by annotType, id and obj# from pageDict.
func RemoveAnnotationsFromPageDict(
	ctx *model.Context,
	annotTypes []model.AnnotationType,
	ids []string,
	objNrSet types.IntSet,
	pageDict types.Dict,
	pageDictObjNr,
	pageNr int,
	incr bool) (bool, error) {

	//fmt.Printf("ids:%v objNrSet:%v\n", ids, objNrSet)

	if len(annotTypes) == 0 && len(ids) == 0 && len(objNrSet) == 0 {
		return removeAllAnnotations(ctx, pageDict, pageDictObjNr, pageNr, incr)
	}

	obj, found := pageDict.Find("Annots")
	if !found {
		return false, nil
	}

	indRef, ok1 := obj.(types.IndirectRef)
	if !ok1 {
		annots, _ := obj.(types.Array)
		ann, ok, err := removeAnnotationsFromAnnots(ctx, annotTypes, ids, objNrSet, pageNr, annots, incr)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
		if incr {
			// Mark page dict obj for incremental writing.
			ctx.Write.IncrementWithObjNr(pageDictObjNr)
		}
		ctx.EnsureVersionForWriting()
		if annots == nil {
			pageDict.Delete("Annots")
			return ok, nil
		}
		pageDict.Update("Annots", ann)
		return ok, nil
	}

	// Annots array is an IndirectReference.
	o, err := ctx.Dereference(indRef)
	if err != nil || o == nil {
		return false, err
	}

	annots, _ := o.(types.Array)

	return removeAnnotationsFromIndAnnots(ctx, annotTypes, ids, objNrSet, pageNr, annots, incr, pageDict, pageDictObjNr, indRef)
}

func prepForRemoveAnnotations(ctx *model.Context, idsAndTypes []string, objNrs []int, incr bool) ([]model.AnnotationType, []string, types.IntSet, bool) {
	var annTypes []model.AnnotationType
	var ids []string

	if len(idsAndTypes) > 0 {
		for _, s := range idsAndTypes {
			if at, ok := model.AnnotTypes[s]; ok {
				annTypes = append(annTypes, at)
				continue
			}
			ids = append(ids, s)
		}
	}

	objNrSet := types.IntSet{}
	for _, i := range objNrs {
		objNrSet[i] = true
	}

	// Remove all annotations for selectedPages
	removeAll := len(idsAndTypes) == 0 && len(objNrs) == 0
	if removeAll {
		log.CLI.Println("removing all annotations for selected pages!")
	}

	if incr {
		ctx.Write.Increment = true
		ctx.Write.Offset = ctx.Read.FileSize
	}

	return annTypes, ids, objNrSet, removeAll
}

// RemoveAnnotations removes annotations for selected pages by id, type or object number.
// All annotations for selected pages are removed if neither idsAndTypes nor objNrs are provided.
func RemoveAnnotations(ctx *model.Context, selectedPages types.IntSet, idsAndTypes []string, objNrs []int, incr bool) (bool, error) {

	annTypes, ids, objNrSet, removeAll := prepForRemoveAnnotations(ctx, idsAndTypes, objNrs, incr)

	var removed bool

	for _, pageNr := range sortedPageNrsForAnnotsFromCache(ctx) {

		if selectedPages != nil {
			if _, found := selectedPages[pageNr]; !found {
				continue
			}
		}

		pageDictIndRef, err := ctx.PageDictIndRef(pageNr)
		if err != nil {
			return false, err
		}

		d, err := ctx.DereferenceDict(*pageDictIndRef)
		if err != nil {
			return false, err
		}

		objNr := pageDictIndRef.ObjectNumber.Value()

		ok, err := RemoveAnnotationsFromPageDict(ctx, annTypes, ids, objNrSet, d, objNr, pageNr, incr)
		if err != nil {
			return false, err
		}
		if ok {
			removed = true
		}

		// if we only remove by obj#, we delete the obj# on annotation removal from objNrSet
		// and can terminate once objNrSet is empty.
		if !removeAll && len(idsAndTypes) == 0 && len(objNrSet) == 0 {
			break
		}
	}

	return removed, nil
}
