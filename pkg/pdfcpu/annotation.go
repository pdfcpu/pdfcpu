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
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
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

func validateAnnotationOperationContext(ctx *model.Context, incr bool) error {
	if ctx == nil {
		return ErrMissingPDFContext
	}
	if ctx.XRefTable == nil {
		return ErrMissingXRefTable
	}
	if !incr {
		return nil
	}
	if ctx.Read == nil {
		return ErrMissingReadContext
	}
	if ctx.Write == nil {
		return ErrMissingWriteContext
	}
	return nil
}

func ensureAnnotationCache(ctx *model.Context) {
	if ctx.PageAnnots == nil {
		ctx.PageAnnots = map[int]model.PgAnnots{}
	}
}

func validateAnnotationRenderer(ar model.AnnotationRenderer) error {
	if ar == nil {
		return ErrMissingAnnotation
	}
	return nil
}

func validateAnnotationPage(ctx *model.Context, pageNr int) error {
	if pageNr < 1 || pageNr > ctx.PageCount {
		return fmt.Errorf("page %d: %w", pageNr, ErrInvalidPageNumber)
	}
	return nil
}

func annotsArrayFromObject(o types.Object, pageNr int) (types.Array, error) {
	annots, ok := o.(types.Array)
	if !ok {
		return nil, fmt.Errorf("page %d Annots: expected array, got %T", pageNr, o)
	}
	return annots, nil
}

func dereferenceAnnotsArray(ctx *model.Context, indRef types.IndirectRef, pageNr int) (types.Array, error) {
	objNr := indRef.ObjectNumber.Value()
	o, err := ctx.Dereference(indRef)
	if err != nil {
		return nil, fmt.Errorf("page %d Annots obj#%d: dereference: %w", pageNr, objNr, err)
	}
	if o == nil {
		return nil, fmt.Errorf("page %d Annots obj#%d: dereference: object is nil", pageNr, objNr)
	}
	return annotsArrayFromObject(o, pageNr)
}

func pageDictForAnnotation(ctx *model.Context, pageNr int) (*types.IndirectRef, types.Dict, error) {
	if err := validateAnnotationPage(ctx, pageNr); err != nil {
		return nil, nil, err
	}

	pageDictIndRef, err := ctx.PageDictIndRef(pageNr)
	if err != nil {
		return nil, nil, fmt.Errorf("page %d: page dict indirect reference: %w", pageNr, err)
	}

	d, err := ctx.DereferenceDict(*pageDictIndRef)
	if err != nil {
		return nil, nil, fmt.Errorf("page %d obj#%d: dereference page dict: %w", pageNr, pageDictIndRef.ObjectNumber.Value(), err)
	}
	return pageDictIndRef, d, nil
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
	if annots.Map == nil {
		annots.Map = model.AnnotMap{}
		pgAnnots[ann.Type()] = annots
	}
	if _, ok := annots.Map[objNr]; ok {
		return fmt.Errorf("addAnnotation: obj#%d already cached", objNr)
	}
	annots.Map[objNr] = ann
	return nil
}

func removeAnnotationFromCache(ctx *model.Context, pageNr, objNr int) error {
	pgAnnots, ok := ctx.PageAnnots[pageNr]
	if !ok {
		return fmt.Errorf("removeAnnotation: no page annotations cached for page %d", pageNr)
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
	return fmt.Errorf("removeAnnotation: no page annotation cached for obj#%d", objNr)
}

func stripAnnotationBackReferences(d types.Dict) {
	d.Delete("P")
	d.Delete("Parent")
}

func deleteAnnotationObject(ctx *model.Context, o types.Object, pageNr int) error {
	switch obj := o.(type) {
	case types.IndirectRef:
		objNr := obj.ObjectNumber.Value()
		d, err := ctx.DereferenceDict(obj)
		if err != nil {
			return fmt.Errorf("page %d annotation obj#%d: dereference before delete: %w", pageNr, objNr, err)
		}
		stripAnnotationBackReferences(d)
		if err := ctx.DeleteObject(obj); err != nil {
			return fmt.Errorf("page %d annotation obj#%d: delete object: %w", pageNr, objNr, err)
		}
	case types.Dict:
		stripAnnotationBackReferences(obj)
		if err := ctx.DeleteObject(obj); err != nil {
			return fmt.Errorf("page %d annotation: delete object: %w", pageNr, err)
		}
	default:
		if err := ctx.DeleteObject(obj); err != nil {
			return fmt.Errorf("page %d annotation: delete object: %w", pageNr, err)
		}
	}
	return nil
}

func findAnnotByID(ctx *model.Context, id string, annots types.Array) (int, error) {
	for i, o := range annots {
		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return -1, fmt.Errorf("annotation array[%d]: dereference dict: %w", i, err)
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

func findAnnotByObjNr(objNr int, annots types.Array) (int, error) {
	for i, o := range annots {
		indRef, ok := o.(types.IndirectRef)
		if !ok {
			continue
		}
		if indRef.ObjectNumber.Value() == objNr {
			return i, nil
		}
	}
	return -1, nil
}

func createAnnot(ctx *model.Context, ar model.AnnotationRenderer, pageIndRef *types.IndirectRef) (*types.IndirectRef, types.Dict, error) {
	d, err := ar.RenderDict(ctx.XRefTable, pageIndRef)
	if err != nil {
		return nil, nil, fmt.Errorf("render annotation dict: %w", err)
	}
	indRef, err := ctx.IndRefForNewObject(d)
	if err != nil {
		return nil, nil, fmt.Errorf("create annotation object: %w", err)
	}
	return indRef, d, nil
}

func linkAnnotation(xRefTable *model.XRefTable, d types.Dict, r *types.Rectangle, apObjNr int, contents, nm string, f model.AnnotationFlags) (model.AnnotationRenderer, error) {
	var uri string
	o, found := d.Find("A")
	if found && o != nil {
		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			if xRefTable.ValidationMode == model.ValidationStrict {
				return nil, err
			}
			model.ShowSkipped("invalid link annotation entry \"A\"")

		}
		if d != nil {
			bb, err := xRefTable.DereferenceStringEntryBytes(d, "URI")
			if err != nil {
				return nil, err
			}
			if len(bb) > 0 {
				uri = string(bb)
			}
		}
	}
	dest := (*model.Destination)(nil) // will not collect link dest during validation.
	return model.NewLinkAnnotation(*r, apObjNr, contents, nm, "", f, nil, dest, uri, nil, false, 0, model.BSSolid), nil
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

	var r *types.Rectangle

	if len(arr) == 4 {
		r, err = xRefTable.RectForArray(arr)
		if err != nil {
			return nil, err
		}
	} else if xRefTable.ValidationMode == model.ValidationRelaxed {
		r = types.NewRectangle(0, 0, 0, 0)
	}

	var apObjNr int
	indRef := d.IndirectRefEntry("AP")
	if indRef != nil {
		apObjNr = indRef.ObjectNumber.Value()
	}

	contents := ""
	if c, ok := d["Contents"]; ok {
		contents, err = xRefTable.DereferenceStringOrHexLiteral(c, model.V10, nil)
		if err != nil {
			return nil, err
		}
		contents = types.RemoveControlChars(contents)
	}

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
		popupIndRef := d.IndirectRefEntry("Popup")
		ann = model.NewTextAnnotation(*r, apObjNr, contents, nm, "", f, nil, "", popupIndRef, nil, "", "", 0, 0, 0, true, "")

	case "Link":
		ann, err = linkAnnotation(xRefTable, d, r, apObjNr, contents, nm, f)
		if err != nil {
			return nil, err
		}

	case "Popup":
		parentIndRef := d.IndirectRefEntry("Parent")
		ann = model.NewPopupAnnotation(*r, apObjNr, contents, nm, "", f, nil, 0, 0, 0, parentIndRef, false)

	// TODO handle remaining annotation types.

	default:
		ann = model.NewAnnotationForRawType(*subtype, *r, apObjNr, contents, nm, "", f, nil, 0, 0, 0)

	}

	return ann, nil
}

// AnnotationsForSelectedPages annotations for selected pages.
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

func prepareHeader(horSep *[]int, maxLen *AnnotListMaxLengths, customAnnot bool) string {
	s := "     Obj# "
	if maxLen.ObjNr > 4 {
		s += strings.Repeat(" ", maxLen.ObjNr-4)
		*horSep = append(*horSep, 10+maxLen.ObjNr-4)
	} else {
		*horSep = append(*horSep, 10)
	}

	s += draw.VBar + " Id "
	if maxLen.ID > 2 {
		s += strings.Repeat(" ", maxLen.ID-2)
		*horSep = append(*horSep, 4+maxLen.ID-2)
	} else {
		*horSep = append(*horSep, 4)
	}

	s += draw.VBar + " Rect "
	if maxLen.Rect > 4 {
		s += strings.Repeat(" ", maxLen.Rect-4)
		*horSep = append(*horSep, 6+maxLen.Rect-4)
	} else {
		*horSep = append(*horSep, 6)
	}

	s += draw.VBar + " Content"
	if maxLen.Content > 7 {
		s += strings.Repeat(" ", maxLen.Content-7)
		*horSep = append(*horSep, 8+maxLen.Content-7)
	} else {
		*horSep = append(*horSep, 8)
	}

	if customAnnot {
		s += draw.VBar + " Type"
		if maxLen.Type > 4 {
			s += strings.Repeat(" ", maxLen.Type-4)
			*horSep = append(*horSep, 5+maxLen.Type-4)
		} else {
			*horSep = append(*horSep, 5)
		}
	}

	return s
}

type AnnotListMaxLengths struct {
	ObjNr, ID, Rect, Content, Type int
}

// AnnotationList represents page annotations formatted for JSON output.
type AnnotationList struct {
	Header      Header                        `json:"header"`
	Annotations map[int][]AnnotationListEntry `json:"annotations"`
}

// AnnotationListEntry represents one page annotation formatted for JSON output.
type AnnotationListEntry struct {
	Type      string     `json:"type"`
	ObjNr     int        `json:"objNr"`
	ID        string     `json:"id,omitempty"`
	Rect      [4]float64 `json:"rect"`
	Content   *string    `json:"content"`
	CustomTyp string     `json:"customType,omitempty"`
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

			var maxLen AnnotListMaxLengths
			maxLen.ID = 2
			maxLen.Content = len("Content")
			maxLen.Type = len("Type")

			var objNrs []int
			for objNr, ann := range annots.Map {
				objNrs = append(objNrs, objNr)
				s := strconv.Itoa(objNr)
				if len(s) > maxLen.ObjNr {
					maxLen.ObjNr = len(s)
				}
				if len(ann.RectString()) > maxLen.Rect {
					maxLen.Rect = len(ann.RectString())
				}
				if len(ann.ID()) > maxLen.ID {
					maxLen.ID = len(ann.ID())
				}
				if len(ann.ContentString()) > maxLen.Content {
					maxLen.Content = len(ann.ContentString())
				}
				if len(ann.CustomTypeString()) > maxLen.Type {
					maxLen.Type = len(ann.CustomTypeString())
				}
			}
			sort.Ints(objNrs)
			ss = append(ss, "")
			ss = append(ss, fmt.Sprintf("  %s:", annType))

			horSep := []int{}

			// Render header.
			ss = append(ss, prepareHeader(&horSep, &maxLen, annType == "Custom"))

			// Render separator.
			ss = append(ss, draw.HorSepLine(horSep))

			// Render content.
			for _, objNr := range objNrs {
				ann := annots.Map[objNr]

				s := strconv.Itoa(objNr)
				fill1 := strings.Repeat(" ", maxLen.ObjNr-len(s))
				if maxLen.ObjNr < 4 {
					fill1 += strings.Repeat(" ", 4-maxLen.ObjNr)
				}

				s = ann.ID()
				fill2 := strings.Repeat(" ", maxLen.ID-len(s))
				if maxLen.ID < 2 {
					fill2 += strings.Repeat(" ", 2-maxLen.ID)
				}

				s = ann.RectString()
				fill3 := strings.Repeat(" ", maxLen.Rect-len(s))

				if ann.Type() != model.AnnCustom {
					ss = append(ss, fmt.Sprintf("     %s%d %s %s%s %s %s%s %s %s",
						fill1, objNr, draw.VBar, fill2, ann.ID(), draw.VBar, fill3, ann.RectString(), draw.VBar, ann.ContentString()))
				} else {
					s = ann.ContentString()
					fill4 := strings.Repeat(" ", maxLen.Content-len(s))
					ss = append(ss, fmt.Sprintf("     %s%d %s %s%s %s %s%s %s %s%s%s %s",
						fill1, objNr, draw.VBar, fill2, ann.ID(), draw.VBar, fill3, ann.RectString(), draw.VBar, fill4, ann.ContentString(), draw.VBar, ann.CustomTypeString()))
				}

				j++
			}
		}
	}

	return j, append([]string{fmt.Sprintf("%d annotations available", j)}, ss...), nil
}

func annotationContent(ann model.AnnotationRenderer) *string {
	s := ann.Content()
	switch a := ann.(type) {
	case model.LinkAnnotation:
		s = a.ContentString()
	case model.PopupAnnotation:
		if a.ParentIndRef != nil {
			s = a.ContentString()
		}
	}
	if s == "" {
		return nil
	}
	return &s
}

func annotationListEntry(annType string, objNr int, ann model.AnnotationRenderer) AnnotationListEntry {
	r := ann.Rectangle()
	return AnnotationListEntry{
		Type:      annType,
		ObjNr:     objNr,
		ID:        ann.ID(),
		Rect:      [4]float64{r.LL.X, r.LL.Y, r.UR.X, r.UR.Y},
		Content:   annotationContent(ann),
		CustomTyp: ann.CustomTypeString(),
	}
}

func annotationJSONHeader() Header {
	return Header{
		Version:  "pdfcpu " + model.VersionStr,
		Creation: time.Now().Format("2006-01-02 15:04:05 MST"),
	}
}

func sortedAnnotationObjNrs(annots model.Annot) []int {
	objNrs := make([]int, 0, len(annots.Map))
	for objNr := range annots.Map {
		objNrs = append(objNrs, objNr)
	}
	sort.Ints(objNrs)
	return objNrs
}

func addAnnotationListEntries(list *AnnotationList, pageNr int, annType string, annots model.Annot) int {
	objNrs := sortedAnnotationObjNrs(annots)
	for _, objNr := range objNrs {
		ann := annots.Map[objNr]
		list.Annotations[pageNr] = append(list.Annotations[pageNr], annotationListEntry(annType, objNr, ann))
	}
	return len(objNrs)
}

func annotationList(annots map[int]model.PgAnnots) (AnnotationList, int) {
	var count int
	list := AnnotationList{
		Header:      annotationJSONHeader(),
		Annotations: map[int][]AnnotationListEntry{},
	}

	for _, pageNr := range sortedAnnotationPages(annots) {
		pageAnnots := annots[pageNr]
		for _, annType := range sortedAnnotationTypeNames(pageAnnots) {
			count += addAnnotationListEntries(&list, pageNr, annType, pageAnnots[model.AnnotTypes[annType]])
		}
	}

	return list, count
}

func sortedAnnotationPages(annots map[int]model.PgAnnots) []int {
	pageNrs := make([]int, 0, len(annots))
	for k := range annots {
		pageNrs = append(pageNrs, k)
	}
	sort.Ints(pageNrs)
	return pageNrs
}

func sortedAnnotationTypeNames(pageAnnots model.PgAnnots) []string {
	annTypes := make([]string, 0, len(pageAnnots))
	for t := range pageAnnots {
		annTypes = append(annTypes, model.AnnotTypeStrings[t])
	}
	sort.Strings(annTypes)
	return annTypes
}

// ListAnnotationsJSON returns a JSON-formatted list of annotations.
func ListAnnotationsJSON(annots map[int]model.PgAnnots) (int, []string, error) {
	list, count := annotationList(annots)
	bb, err := json.MarshalIndent(list, "", "\t")
	if err != nil {
		return 0, nil, err
	}
	return count, []string{string(bb)}, nil
}

func addAnnotationToDirectObj(
	ctx *model.Context,
	annots types.Array,
	annotIndRef, pageDictIndRef *types.IndirectRef,
	pageDict types.Dict,
	pageNr int,
	ar model.AnnotationRenderer,
	incr bool) error {
	i, err := findAnnotByID(ctx, ar.ID(), annots)
	if err != nil {
		return fmt.Errorf("page %d Annots: find duplicate id %q: %w", pageNr, ar.ID(), err)
	}
	if i >= 0 {
		return fmt.Errorf("page %d: duplicate annotation with id:%s", pageNr, ar.ID())
	}
	pageDict.Update("Annots", append(annots, *annotIndRef))
	if incr {
		// Mark page dict obj for incremental writing.
		ctx.Write.IncrementWithObjNr(pageDictIndRef.ObjectNumber.Value())
	}
	ctx.EnsureVersionForWriting()
	return nil
}

func validateAddAnnotationInput(
	ctx *model.Context,
	pageDictIndRef *types.IndirectRef,
	pageDict types.Dict,
	pageNr int,
	ar model.AnnotationRenderer,
	incr bool) error {
	if err := validateAnnotationOperationContext(ctx, incr); err != nil {
		return err
	}
	if pageDictIndRef == nil {
		return fmt.Errorf("page %d: missing page dict indirect reference", pageNr)
	}
	if pageDict == nil {
		return fmt.Errorf("page %d: missing page dict", pageNr)
	}
	if err := validateAnnotationRenderer(ar); err != nil {
		return err
	}
	ensureAnnotationCache(ctx)
	return nil
}

func addAnnotationToPageAnnots(
	ctx *model.Context,
	annotIndRef, pageDictIndRef *types.IndirectRef,
	pageDict types.Dict,
	pageNr int,
	ar model.AnnotationRenderer,
	incr bool) error {
	obj, found := pageDict.Find("Annots")
	if !found {
		pageDict.Insert("Annots", types.Array{*annotIndRef})
		if incr {
			// Mark page dict obj for incremental writing.
			ctx.Write.IncrementWithObjNr(pageDictIndRef.ObjectNumber.Value())
		}
		ctx.EnsureVersionForWriting()
		return nil
	}

	ir, ok := obj.(types.IndirectRef)
	if !ok {
		annots, err := annotsArrayFromObject(obj, pageNr)
		if err != nil {
			return err
		}
		return addAnnotationToDirectObj(ctx, annots, annotIndRef, pageDictIndRef, pageDict, pageNr, ar, incr)
	}

	return addAnnotationToIndirectAnnots(ctx, ir, annotIndRef, pageNr, ar, incr)
}

func addAnnotationToIndirectAnnots(
	ctx *model.Context,
	annotsIndRef types.IndirectRef,
	annotIndRef *types.IndirectRef,
	pageNr int,
	ar model.AnnotationRenderer,
	incr bool) error {
	annots, err := dereferenceAnnotsArray(ctx, annotsIndRef, pageNr)
	if err != nil {
		return err
	}
	i, err := findAnnotByID(ctx, ar.ID(), annots)
	if err != nil {
		return fmt.Errorf("page %d Annots obj#%d: find duplicate id %q: %w", pageNr, annotsIndRef.ObjectNumber.Value(), ar.ID(), err)
	}
	if i >= 0 {
		return fmt.Errorf("page %d: duplicate annotation with id:%s", pageNr, ar.ID())
	}

	entry, ok := ctx.FindTableEntryForIndRef(&annotsIndRef)
	if !ok {
		return fmt.Errorf("page %d Annots obj#%d: missing xref table entry", pageNr, annotsIndRef.ObjectNumber.Value())
	}
	entry.Object = append(annots, *annotIndRef)
	if incr {
		// Mark Annot array obj for incremental writing.
		ctx.Write.IncrementWithObjNr(annotsIndRef.ObjectNumber.Value())
	}

	ctx.EnsureVersionForWriting()
	return nil
}

func cleanupAddedAnnotation(ctx *model.Context, pageNr, objNr int, annotIndRef *types.IndirectRef) error {
	return errors.Join(
		removeAnnotationFromCache(ctx, pageNr, objNr),
		deleteAnnotationObject(ctx, *annotIndRef, pageNr),
	)
}

// AddAnnotation adds ar to pageDict.
func AddAnnotation(
	ctx *model.Context,
	pageDictIndRef *types.IndirectRef,
	pageDict types.Dict,
	pageNr int,
	ar model.AnnotationRenderer,
	incr bool) (*types.IndirectRef, types.Dict, error) {
	if err := validateAddAnnotationInput(ctx, pageDictIndRef, pageDict, pageNr, ar, incr); err != nil {
		return nil, nil, err
	}

	// Create xreftable entry for annotation.
	annotIndRef, d, err := createAnnot(ctx, ar, pageDictIndRef)
	if err != nil {
		return nil, nil, fmt.Errorf("page %d: create annotation: %w", pageNr, err)
	}

	// Add annotation to xreftable page annotation cache.
	err = addAnnotationToCache(ctx, ar, pageNr, annotIndRef.ObjectNumber.Value())
	if err != nil {
		cacheErr := fmt.Errorf("page %d annotation obj#%d: cache: %w", pageNr, annotIndRef.ObjectNumber.Value(), err)
		cleanupErr := deleteAnnotationObject(ctx, *annotIndRef, pageNr)
		return nil, nil, errors.Join(cacheErr, cleanupErr)
	}

	if incr {
		// Mark new annotaton dict obj for incremental writing.
		ctx.Write.IncrementWithObjNr(annotIndRef.ObjectNumber.Value())
	}

	if err := addAnnotationToPageAnnots(ctx, annotIndRef, pageDictIndRef, pageDict, pageNr, ar, incr); err != nil {
		cleanupErr := cleanupAddedAnnotation(ctx, pageNr, annotIndRef.ObjectNumber.Value(), annotIndRef)
		return nil, nil, errors.Join(err, cleanupErr)
	}
	return annotIndRef, d, nil
}

// AddAnnotationToPage adds annotation to page.
func AddAnnotationToPage(ctx *model.Context, pageNr int, ar model.AnnotationRenderer, incr bool) (*types.IndirectRef, types.Dict, error) {
	if err := validateAnnotationOperationContext(ctx, incr); err != nil {
		return nil, nil, err
	}
	if err := validateAnnotationRenderer(ar); err != nil {
		return nil, nil, err
	}
	pageDictIndRef, d, err := pageDictForAnnotation(ctx, pageNr)
	if err != nil {
		return nil, nil, err
	}

	return AddAnnotation(ctx, pageDictIndRef, d, pageNr, ar, incr)
}

type annotationAddPage struct {
	pageNr         int
	pageDictIndRef *types.IndirectRef
	pageDict       types.Dict
	annots         []model.AnnotationRenderer
}

func selectedAnnotationPageNrs(selectedPages types.IntSet) []int {
	pageNrs := make([]int, 0, len(selectedPages))
	for pageNr, selected := range selectedPages {
		if selected {
			pageNrs = append(pageNrs, pageNr)
		}
	}
	sort.Ints(pageNrs)
	return pageNrs
}

func prepareAnnotationAddPage(ctx *model.Context, pageNr int, annots []model.AnnotationRenderer) (annotationAddPage, error) {
	pageDictIndRef, pageDict, err := pageDictForAnnotation(ctx, pageNr)
	if err != nil {
		return annotationAddPage{}, err
	}
	for i, annot := range annots {
		if err := validateAnnotationRenderer(annot); err != nil {
			return annotationAddPage{}, fmt.Errorf("page %d annotation %d: %w", pageNr, i+1, err)
		}
	}
	return annotationAddPage{pageNr: pageNr, pageDictIndRef: pageDictIndRef, pageDict: pageDict, annots: annots}, nil
}

func prepareAnnotationAddPages(ctx *model.Context, selectedPages types.IntSet, ar model.AnnotationRenderer) ([]annotationAddPage, error) {
	pageNrs := selectedAnnotationPageNrs(selectedPages)
	pages := make([]annotationAddPage, 0, len(pageNrs))
	for _, pageNr := range pageNrs {
		page, err := prepareAnnotationAddPage(ctx, pageNr, []model.AnnotationRenderer{ar})
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}
	return pages, nil
}

func prepareAnnotationAddMap(ctx *model.Context, m map[int][]model.AnnotationRenderer) ([]annotationAddPage, error) {
	pageNrs := make([]int, 0, len(m))
	for pageNr := range m {
		pageNrs = append(pageNrs, pageNr)
	}
	sort.Ints(pageNrs)

	pages := make([]annotationAddPage, 0, len(pageNrs))
	for _, pageNr := range pageNrs {
		page, err := prepareAnnotationAddPage(ctx, pageNr, m[pageNr])
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}
	return pages, nil
}

func applyAnnotationAddPages(ctx *model.Context, pages []annotationAddPage, incr bool) (bool, error) {
	var added bool
	for _, page := range pages {
		for i, annot := range page.annots {
			indRef, _, err := AddAnnotation(ctx, page.pageDictIndRef, page.pageDict, page.pageNr, annot, incr)
			if err != nil {
				return false, fmt.Errorf("page %d annotation %d: %w", page.pageNr, i+1, err)
			}
			added = added || indRef != nil
		}
	}
	return added, nil
}

// AddAnnotations adds ar to selected pages.
func AddAnnotations(ctx *model.Context, selectedPages types.IntSet, ar model.AnnotationRenderer, incr bool) (bool, error) {
	if err := validateAnnotationOperationContext(ctx, incr); err != nil {
		return false, err
	}
	if err := validateAnnotationRenderer(ar); err != nil {
		return false, err
	}
	pages, err := prepareAnnotationAddPages(ctx, selectedPages, ar)
	if err != nil {
		return false, err
	}

	ensureAnnotationCache(ctx)
	if incr {
		ctx.Write.Increment = true
		ctx.Write.Offset = ctx.Read.FileSize
	}
	return applyAnnotationAddPages(ctx, pages, incr)
}

// AddAnnotationsMap adds annotations in m to corresponding pages.
func AddAnnotationsMap(ctx *model.Context, m map[int][]model.AnnotationRenderer, incr bool) (bool, error) {
	if err := validateAnnotationOperationContext(ctx, incr); err != nil {
		return false, err
	}
	pages, err := prepareAnnotationAddMap(ctx, m)
	if err != nil {
		return false, err
	}

	ensureAnnotationCache(ctx)
	if incr {
		ctx.Write.Increment = true
		ctx.Write.Offset = ctx.Read.FileSize
	}
	return applyAnnotationAddPages(ctx, pages, incr)
}

func validateAnnotationObjectForRemoval(ctx *model.Context, o types.Object, pageNr, index int) error {
	switch obj := o.(type) {
	case types.IndirectRef:
		objNr := obj.ObjectNumber.Value()
		entry, found := ctx.FindTableEntryForIndRef(&obj)
		if !found || entry.Free {
			return fmt.Errorf("page %d annotation array[%d] obj#%d: missing xref table entry", pageNr, index, objNr)
		}
		d, err := ctx.DereferenceDict(obj)
		if err != nil {
			return fmt.Errorf("page %d annotation array[%d] obj#%d: dereference dict: %w", pageNr, index, objNr, err)
		}
		if d == nil {
			return fmt.Errorf("page %d annotation array[%d] obj#%d: dereference dict: object is nil", pageNr, index, objNr)
		}
	case types.Dict:
		return nil
	default:
		return fmt.Errorf("page %d annotation array[%d]: expected dict or indirect reference, got %T", pageNr, index, o)
	}
	return nil
}

func validateAnnotationObjectsForRemoval(ctx *model.Context, annots types.Array, pageNr int) error {
	for i, o := range annots {
		if err := validateAnnotationObjectForRemoval(ctx, o, pageNr, i); err != nil {
			return err
		}
	}
	return nil
}

type annotationDeletionValidator struct {
	ctx             *model.Context
	remainingRefs   map[int]int
	deletedObjNrSet types.IntSet
	pageNr          int
}

func validateAnnotationFreeList(ctx *model.Context, pageNr int) error {
	entry, found := ctx.FindTableEntryLight(0)
	if !found || entry == nil || !entry.Free {
		return fmt.Errorf("page %d annotation deletion: invalid xref free-list head", pageNr)
	}
	return nil
}

func (v *annotationDeletionValidator) remainingRefCount(objNr int, entry *model.XRefTableEntry) int {
	if count, found := v.remainingRefs[objNr]; found {
		return count
	}
	return entry.RefCount
}

func (v *annotationDeletionValidator) validateIndirectRef(indRef types.IndirectRef, stripBackRefs bool) error {
	objNr := indRef.ObjectNumber.Value()
	if v.deletedObjNrSet[objNr] {
		return nil
	}
	entry, found := v.ctx.FindTableEntryForIndRef(&indRef)
	if !found {
		return fmt.Errorf("page %d annotation deletion obj#%d: missing xref table entry", v.pageNr, objNr)
	}
	if entry.Free {
		return nil
	}
	refCount := v.remainingRefCount(objNr, entry)
	if refCount > 1 {
		v.remainingRefs[objNr] = refCount - 1
		return nil
	}

	o, err := v.ctx.Dereference(indRef)
	if err != nil {
		return fmt.Errorf("page %d annotation deletion obj#%d: dereference: %w", v.pageNr, objNr, err)
	}
	if o == nil {
		return nil
	}
	v.deletedObjNrSet[objNr] = true
	return v.validateObject(o, stripBackRefs)
}

func (v *annotationDeletionValidator) validateDict(d types.Dict, stripBackRefs bool) error {
	keys := make([]string, 0, len(d))
	for key := range d {
		if stripBackRefs && (key == "P" || key == "Parent") {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := v.validateObject(d[key], false); err != nil {
			return fmt.Errorf("entry %s: %w", key, err)
		}
	}
	return nil
}

func (v *annotationDeletionValidator) validateObject(o types.Object, stripBackRefs bool) error {
	switch obj := o.(type) {
	case types.IndirectRef:
		return v.validateIndirectRef(obj, stripBackRefs)
	case types.Dict:
		return v.validateDict(obj, stripBackRefs)
	case types.StreamDict:
		return v.validateDict(obj.Dict, false)
	case types.Array:
		for i, value := range obj {
			if err := v.validateObject(value, false); err != nil {
				return fmt.Errorf("array[%d]: %w", i, err)
			}
		}
	}
	return nil
}

func validateAnnotationDeletionTargets(ctx *model.Context, targets []types.Object, pageNr int) error {
	if len(targets) == 0 {
		return nil
	}
	if err := validateAnnotationFreeList(ctx, pageNr); err != nil {
		return err
	}
	v := annotationDeletionValidator{
		ctx:             ctx,
		remainingRefs:   map[int]int{},
		deletedObjNrSet: types.IntSet{},
		pageNr:          pageNr,
	}
	for i, target := range targets {
		if err := v.validateObject(target, true); err != nil {
			return fmt.Errorf("page %d annotation deletion target %d: %w", pageNr, i+1, err)
		}
	}
	return nil
}

func prepareRemoveAllAnnotations(ctx *model.Context, obj types.Object, pageNr int) (types.Array, *types.IndirectRef, error) {
	ir, indirect := obj.(types.IndirectRef)
	if !indirect {
		annots, err := annotsArrayFromObject(obj, pageNr)
		if err != nil {
			return nil, nil, err
		}
		if err := validateAnnotationObjectsForRemoval(ctx, annots, pageNr); err != nil {
			return nil, nil, err
		}
		if err := validateAnnotationDeletionTargets(ctx, annots, pageNr); err != nil {
			return nil, nil, err
		}
		return annots, nil, nil
	}

	annots, err := dereferenceAnnotsArray(ctx, ir, pageNr)
	if err != nil {
		return nil, nil, err
	}
	entry, found := ctx.FindTableEntryForIndRef(&ir)
	if !found || entry.Free {
		return nil, nil, fmt.Errorf("page %d Annots obj#%d: missing xref table entry", pageNr, ir.ObjectNumber.Value())
	}
	if err := validateAnnotationObjectsForRemoval(ctx, annots, pageNr); err != nil {
		return nil, nil, err
	}
	if err := validateAnnotationFreeList(ctx, pageNr); err != nil {
		return nil, nil, err
	}
	if err := validateAnnotationDeletionTargets(ctx, annots, pageNr); err != nil {
		return nil, nil, err
	}
	return annots, &ir, nil
}

func removeAllAnnotations(
	ctx *model.Context,
	pageDict types.Dict,
	pageDictObjNr,
	pageNr int,
	incr bool) (bool, error) {
	obj, found := pageDict.Find("Annots")
	if !found {
		return false, nil
	}

	annots, annotsIndRef, err := prepareRemoveAllAnnotations(ctx, obj, pageNr)
	if err != nil {
		return false, err
	}

	if annotsIndRef != nil {
		objNr := annotsIndRef.ObjectNumber.Value()
		if err := ctx.FreeObject(objNr); err != nil {
			return false, fmt.Errorf("page %d Annots obj#%d: free object: %w", pageNr, objNr, err)
		}
		if incr {
			// Modify Annots array obj for incremental writing.
			ctx.Write.IncrementWithObjNr(objNr)
		}
	}

	for _, o := range annots {
		if err := deleteAnnotationObject(ctx, o, pageNr); err != nil {
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

type annotationRemovalTarget struct {
	index  int
	indRef types.IndirectRef
}

type annotationRemovalPlan struct {
	targets         map[int]annotationRemovalTarget
	matchedObjNrSet types.IntSet
}

func newAnnotationRemovalPlan() *annotationRemovalPlan {
	return &annotationRemovalPlan{
		targets:         map[int]annotationRemovalTarget{},
		matchedObjNrSet: types.IntSet{},
	}
}

func (p *annotationRemovalPlan) sortedTargets() []annotationRemovalTarget {
	targets := make([]annotationRemovalTarget, 0, len(p.targets))
	for _, target := range p.targets {
		targets = append(targets, target)
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].index < targets[j].index
	})
	return targets
}

func (p *annotationRemovalPlan) deletionTargets() []types.Object {
	targets := p.sortedTargets()
	objects := make([]types.Object, 0, len(targets))
	for _, target := range targets {
		objects = append(objects, target.indRef)
	}
	return objects
}

func cachedAnnotationType(ctx *model.Context, pageNr, objNr int) (model.AnnotationType, error) {
	pgAnnots, found := ctx.PageAnnots[pageNr]
	if !found {
		return 0, fmt.Errorf("page %d annotation obj#%d: missing page annotation cache", pageNr, objNr)
	}

	var (
		annotType model.AnnotationType
		matches   int
	)
	for candidateType, annots := range pgAnnots {
		if _, found := annots.Map[objNr]; found {
			annotType = candidateType
			matches++
		}
	}
	if matches != 1 {
		return 0, fmt.Errorf("page %d annotation obj#%d: expected one cache entry, got %d", pageNr, objNr, matches)
	}
	return annotType, nil
}

func resolveAnnotationRemovalTarget(
	ctx *model.Context,
	pageNr int,
	annots types.Array,
	index int,
	expectedType *model.AnnotationType) (annotationRemovalTarget, error) {
	if index < 0 || index >= len(annots) {
		return annotationRemovalTarget{}, fmt.Errorf("page %d annotation array index %d: out of bounds", pageNr, index)
	}

	indRef, ok := annots[index].(types.IndirectRef)
	if !ok {
		return annotationRemovalTarget{}, fmt.Errorf("page %d annotation array[%d]: expected indirect reference, got %T", pageNr, index, annots[index])
	}
	objNr := indRef.ObjectNumber.Value()
	entry, found := ctx.FindTableEntryForIndRef(&indRef)
	if !found || entry.Free {
		return annotationRemovalTarget{}, fmt.Errorf("page %d annotation obj#%d: missing xref table entry", pageNr, objNr)
	}
	d, err := ctx.DereferenceDict(indRef)
	if err != nil {
		return annotationRemovalTarget{}, fmt.Errorf("page %d annotation obj#%d: dereference dict: %w", pageNr, objNr, err)
	}
	if d == nil {
		return annotationRemovalTarget{}, fmt.Errorf("page %d annotation obj#%d: dereference dict: object is nil", pageNr, objNr)
	}

	annotType, err := cachedAnnotationType(ctx, pageNr, objNr)
	if err != nil {
		return annotationRemovalTarget{}, err
	}
	if expectedType != nil && annotType != *expectedType {
		return annotationRemovalTarget{}, fmt.Errorf("page %d annotation obj#%d: cache type mismatch", pageNr, objNr)
	}
	return annotationRemovalTarget{index: index, indRef: indRef}, nil
}

func addAnnotationRemovalTarget(
	ctx *model.Context,
	plan *annotationRemovalPlan,
	pageNr int,
	annots types.Array,
	index int,
	expectedType *model.AnnotationType) error {
	target, err := resolveAnnotationRemovalTarget(ctx, pageNr, annots, index, expectedType)
	if err != nil {
		return err
	}
	plan.targets[target.indRef.ObjectNumber.Value()] = target
	return nil
}

func preflightAnnotationTypes(
	ctx *model.Context,
	plan *annotationRemovalPlan,
	annotTypes []model.AnnotationType,
	pageNr int,
	annots types.Array) error {
	pgAnnots, found := ctx.PageAnnots[pageNr]
	if !found {
		return nil
	}
	for _, annotType := range annotTypes {
		cached, found := pgAnnots[annotType]
		if !found {
			continue
		}
		if cached.IndRefs == nil {
			return fmt.Errorf("page %d %s annotations: missing indirect references", pageNr, model.AnnotTypeStrings[annotType])
		}
		for _, indRef := range *cached.IndRefs {
			objNr := indRef.ObjectNumber.Value()
			index, err := findAnnotByObjNr(objNr, annots)
			if err != nil {
				return fmt.Errorf("page %d annotation obj#%d: find in Annots: %w", pageNr, objNr, err)
			}
			if index < 0 {
				return fmt.Errorf("page %d annotation obj#%d: missing Annots entry", pageNr, objNr)
			}
			if err := addAnnotationRemovalTarget(ctx, plan, pageNr, annots, index, &annotType); err != nil {
				return err
			}
		}
	}
	return nil
}

func sortedSelectedAnnotationObjNrs(objNrSet types.IntSet) []int {
	objNrs := make([]int, 0, len(objNrSet))
	for objNr, selected := range objNrSet {
		if selected {
			objNrs = append(objNrs, objNr)
		}
	}
	sort.Ints(objNrs)
	return objNrs
}

func preflightAnnotationObjNrs(
	ctx *model.Context,
	plan *annotationRemovalPlan,
	objNrSet types.IntSet,
	pageNr int,
	annots types.Array) error {
	for _, objNr := range sortedSelectedAnnotationObjNrs(objNrSet) {
		if objNr < 0 {
			continue
		}
		index, err := findAnnotByObjNr(objNr, annots)
		if err != nil {
			return fmt.Errorf("page %d annotation obj#%d: find in Annots: %w", pageNr, objNr, err)
		}
		if index < 0 {
			continue
		}
		if err := addAnnotationRemovalTarget(ctx, plan, pageNr, annots, index, nil); err != nil {
			return err
		}
		plan.matchedObjNrSet[objNr] = true
	}
	return nil
}

func preflightAnnotationID(
	ctx *model.Context,
	plan *annotationRemovalPlan,
	id string,
	matchedObjNr,
	pageNr int,
	annots types.Array) error {
	index, err := findAnnotByID(ctx, id, annots)
	if err != nil {
		return fmt.Errorf("page %d annotation id %q: find in Annots: %w", pageNr, id, err)
	}
	if index < 0 {
		return nil
	}
	if err := addAnnotationRemovalTarget(ctx, plan, pageNr, annots, index, nil); err != nil {
		return err
	}
	if matchedObjNr >= 0 {
		plan.matchedObjNrSet[matchedObjNr] = true
	}
	return nil
}

func preflightAnnotationIDs(
	ctx *model.Context,
	plan *annotationRemovalPlan,
	ids []string,
	objNrSet types.IntSet,
	pageNr int,
	annots types.Array) error {
	for _, id := range ids {
		if err := preflightAnnotationID(ctx, plan, id, -1, pageNr, annots); err != nil {
			return fmt.Errorf("remove id %q: %w", id, err)
		}
	}
	for _, objNr := range sortedSelectedAnnotationObjNrs(objNrSet) {
		if plan.matchedObjNrSet[objNr] {
			continue
		}
		if err := preflightAnnotationID(ctx, plan, strconv.Itoa(objNr), objNr, pageNr, annots); err != nil {
			return fmt.Errorf("remove numeric id %d: %w", objNr, err)
		}
	}
	return nil
}

func validateCachedAnnotationMembership(ctx *model.Context, pageNr int, objNrSet types.IntSet) error {
	for _, annots := range ctx.PageAnnots[pageNr] {
		for objNr := range annots.Map {
			if objNr > 0 && !objNrSet[objNr] {
				return fmt.Errorf("page %d annotation obj#%d: cache entry missing from Annots", pageNr, objNr)
			}
		}
	}
	return nil
}

func preflightSelectiveAnnotationArray(ctx *model.Context, pageNr int, annots types.Array) error {
	objNrSet := types.IntSet{}
	for i, o := range annots {
		if _, direct := o.(types.Dict); direct {
			continue
		}
		target, err := resolveAnnotationRemovalTarget(ctx, pageNr, annots, i, nil)
		if err != nil {
			return err
		}
		objNr := target.indRef.ObjectNumber.Value()
		if objNrSet[objNr] {
			return fmt.Errorf("page %d annotation obj#%d: duplicate Annots entry", pageNr, objNr)
		}
		objNrSet[objNr] = true
	}
	return validateCachedAnnotationMembership(ctx, pageNr, objNrSet)
}

func preflightSelectiveAnnotationRemoval(
	ctx *model.Context,
	annotTypes []model.AnnotationType,
	ids []string,
	objNrSet types.IntSet,
	pageNr int,
	annots types.Array) (*annotationRemovalPlan, error) {
	plan := newAnnotationRemovalPlan()
	if err := preflightAnnotationTypes(ctx, plan, annotTypes, pageNr, annots); err != nil {
		return nil, fmt.Errorf("remove by type: %w", err)
	}
	if err := preflightSelectiveAnnotationArray(ctx, pageNr, annots); err != nil {
		return nil, fmt.Errorf("validate Annots: %w", err)
	}
	if err := preflightAnnotationObjNrs(ctx, plan, objNrSet, pageNr, annots); err != nil {
		return nil, fmt.Errorf("remove by object number: %w", err)
	}
	if err := preflightAnnotationIDs(ctx, plan, ids, objNrSet, pageNr, annots); err != nil {
		return nil, fmt.Errorf("remove by id: %w", err)
	}
	if err := validateAnnotationDeletionTargets(ctx, plan.deletionTargets(), pageNr); err != nil {
		return nil, fmt.Errorf("validate deletion graph: %w", err)
	}
	return plan, nil
}

func applySelectiveAnnotationRemoval(
	ctx *model.Context,
	plan *annotationRemovalPlan,
	objNrSet types.IntSet,
	pageNr int,
	annots types.Array,
	incr bool) (types.Array, bool, error) {
	targets := plan.sortedTargets()
	if len(targets) == 0 {
		return annots, false, nil
	}
	for _, target := range targets {
		if err := deleteAnnotationObject(ctx, target.indRef, pageNr); err != nil {
			return nil, false, err
		}
	}
	for _, target := range targets {
		objNr := target.indRef.ObjectNumber.Value()
		if err := removeAnnotationFromCache(ctx, pageNr, objNr); err != nil {
			return nil, false, fmt.Errorf("page %d annotation obj#%d: remove from cache: %w", pageNr, objNr, err)
		}
		if incr {
			ctx.Write.IncrementWithObjNr(objNr)
		}
	}
	for objNr := range plan.matchedObjNrSet {
		delete(objNrSet, objNr)
	}

	removed := make(map[int]bool, len(targets))
	for _, target := range targets {
		removed[target.index] = true
	}
	result := make(types.Array, 0, len(annots)-len(targets))
	for i, o := range annots {
		if !removed[i] {
			result = append(result, o)
		}
	}
	return result, true, nil
}

func removeAnnotationsFromAnnots(
	ctx *model.Context,
	annotTypes []model.AnnotationType,
	ids []string,
	objNrSet types.IntSet,
	pageNr int,
	annots types.Array,
	incr bool) (types.Array, bool, error) {
	plan, err := preflightSelectiveAnnotationRemoval(ctx, annotTypes, ids, objNrSet, pageNr, annots)
	if err != nil {
		return nil, false, err
	}
	return applySelectiveAnnotationRemoval(ctx, plan, objNrSet, pageNr, annots, incr)
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
	objNr := indRef.ObjectNumber.Value()
	genNr := indRef.GenerationNumber.Value()
	entry, found := ctx.FindTableEntry(objNr, genNr)
	if !found || entry.Free {
		return false, fmt.Errorf("page %d Annots obj#%d: missing xref table entry", pageNr, objNr)
	}

	ann, ok, err := removeAnnotationsFromAnnots(ctx, annotTypes, ids, objNrSet, pageNr, annots, incr)
	if err != nil {
		return false, fmt.Errorf("page %d Annots obj#%d: %w", pageNr, indRef.ObjectNumber.Value(), err)
	}
	if !ok {
		return false, nil
	}

	if incr {
		// Modify Annots array obj for incremental writing.
		ctx.Write.IncrementWithObjNr(objNr)
	}

	ctx.EnsureVersionForWriting()

	if len(ann) == 0 {
		pageDict.Delete("Annots")
		if err := ctx.DeleteObject(indRef); err != nil {
			return false, fmt.Errorf("page %d Annots obj#%d: delete object: %w", pageNr, objNr, err)
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
	if err := validateAnnotationOperationContext(ctx, incr); err != nil {
		return false, err
	}
	if pageDict == nil {
		return false, fmt.Errorf("page %d: missing page dict", pageNr)
	}

	if len(annotTypes) == 0 && len(ids) == 0 && len(objNrSet) == 0 {
		return removeAllAnnotations(ctx, pageDict, pageDictObjNr, pageNr, incr)
	}

	obj, found := pageDict.Find("Annots")
	if !found {
		return false, nil
	}

	indRef, ok1 := obj.(types.IndirectRef)
	if !ok1 {
		annots, err := annotsArrayFromObject(obj, pageNr)
		if err != nil {
			return false, err
		}
		ann, ok, err := removeAnnotationsFromAnnots(ctx, annotTypes, ids, objNrSet, pageNr, annots, incr)
		if err != nil {
			return false, fmt.Errorf("page %d Annots: %w", pageNr, err)
		}
		if !ok {
			return false, nil
		}
		if incr {
			// Mark page dict obj for incremental writing.
			ctx.Write.IncrementWithObjNr(pageDictObjNr)
		}
		ctx.EnsureVersionForWriting()
		if len(ann) == 0 {
			pageDict.Delete("Annots")
			return ok, nil
		}
		pageDict.Update("Annots", ann)
		return ok, nil
	}

	// Annots array is an IndirectReference.
	annots, err := dereferenceAnnotsArray(ctx, indRef, pageNr)
	if err != nil {
		return false, err
	}

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

func annotationRemovalCatalog(ctx *model.Context, removeAll bool) (types.Dict, error) {
	if !removeAll {
		return nil, nil
	}
	root, err := ctx.Catalog()
	if err != nil {
		return nil, fmt.Errorf("remove annotations: catalog: %w", err)
	}
	if root == nil {
		return nil, errors.New("remove annotations: catalog: missing root dict")
	}
	return root, nil
}

// RemoveAnnotations removes annotations for selected pages by id, type or object number.
// All annotations for selected pages are removed if neither idsAndTypes nor objNrs are provided.
func RemoveAnnotations(ctx *model.Context, selectedPages types.IntSet, idsAndTypes []string, objNrs []int, incr bool) (bool, error) {
	if err := validateAnnotationOperationContext(ctx, incr); err != nil {
		return false, err
	}

	removeAll := len(idsAndTypes) == 0 && len(objNrs) == 0
	root, err := annotationRemovalCatalog(ctx, removeAll)
	if err != nil {
		return false, err
	}
	annTypes, ids, objNrSet, removeAll := prepForRemoveAnnotations(ctx, idsAndTypes, objNrs, incr)

	var removed bool

	for _, pageNr := range sortedPageNrsForAnnotsFromCache(ctx) {

		if selectedPages != nil {
			if _, found := selectedPages[pageNr]; !found {
				continue
			}
		}

		pageDictIndRef, d, err := pageDictForAnnotation(ctx, pageNr)
		if err != nil {
			return false, err
		}

		objNr := pageDictIndRef.ObjectNumber.Value()

		ok, err := RemoveAnnotationsFromPageDict(ctx, annTypes, ids, objNrSet, d, objNr, pageNr, incr)
		if err != nil {
			return false, fmt.Errorf("page %d: remove annotations: %w", pageNr, err)
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

	if removeAll && removed {
		// Hacky, actually we only want to remove struct tree elements using removed annotations
		// but this is most probably what we want anyway.
		root.Delete("StructTreeRoot")
	}

	return removed, nil
}
