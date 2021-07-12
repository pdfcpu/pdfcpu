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
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// AnnotationFlags represents the PDF annotation flags.
type AnnotationFlags int

const ( // See table 165
	AnnInvisible AnnotationFlags = 1 << iota
	AnnHidden
	AnnPrint
	AnnNoZoom
	AnnNoRotate
	AnnNoView
	AnnReadOnly
	AnnLocked
	AnnToggleNoView
	AnnLockedContents
)

// AnnotationType represents the various PDF annotation types.
type AnnotationType int

const (
	AnnText AnnotationType = iota
	AnnLink
	AnnFreeText
	AnnLine
	AnnSquare
	AnnCircle
	AnnPolygon
	AnnPolyLine
	AnnHighLight
	AnnUnderline
	AnnSquiggly
	AnnStrikeOut
	AnnStamp
	AnnCaret
	AnnInk
	AnnPopup
	AnnFileAttachment
	AnnSound
	AnnMovie
	AnnWidget
	AnnScreen
	AnnPrinterMark
	AnnTrapNet
	AnnWatermark
	Ann3D
	AnnRedact
)

var annotTypes = map[string]AnnotationType{
	"Text":           AnnText,
	"Link":           AnnLink,
	"FreeText":       AnnFreeText,
	"Line":           AnnLine,
	"Square":         AnnSquare,
	"Circle":         AnnCircle,
	"Polygon":        AnnPolygon,
	"PolyLine":       AnnPolyLine,
	"HighLight":      AnnHighLight,
	"Underline":      AnnUnderline,
	"Squiggly":       AnnSquiggly,
	"StrikeOut":      AnnStrikeOut,
	"Stamp":          AnnStamp,
	"Caret":          AnnCaret,
	"Ink":            AnnInk,
	"Popup":          AnnPopup,
	"FileAttachment": AnnFileAttachment,
	"Sound":          AnnSound,
	"Movie":          AnnMovie,
	"Widget":         AnnWidget,
	"Screen":         AnnScreen,
	"PrinterMark":    AnnPrinterMark,
	"TrapNet":        AnnTrapNet,
	"Watermark":      AnnWatermark,
	"3D":             Ann3D,
	"Redact":         AnnRedact,
}

// AnnotTypeStrings manages string representations for annotation types.
var AnnotTypeStrings = map[AnnotationType]string{
	AnnText:           "Text",
	AnnLink:           "Link",
	AnnFreeText:       "FreeText",
	AnnLine:           "Line",
	AnnSquare:         "Square",
	AnnCircle:         "Circle",
	AnnPolygon:        "Polygon",
	AnnPolyLine:       "PolyLine",
	AnnHighLight:      "HighLight",
	AnnUnderline:      "Underline",
	AnnSquiggly:       "Squiggly",
	AnnStrikeOut:      "StrikeOut",
	AnnStamp:          "Stamp",
	AnnCaret:          "Caret",
	AnnInk:            "Ink",
	AnnPopup:          "Popup",
	AnnFileAttachment: "FileAttachment",
	AnnSound:          "Sound",
	AnnMovie:          "Movie",
	AnnWidget:         "Widget",
	AnnScreen:         "Screen",
	AnnPrinterMark:    "PrinterMark",
	AnnTrapNet:        "TrapNet",
	AnnWatermark:      "Watermark",
	Ann3D:             "3D",
	AnnRedact:         "Redact",
}

// AnnotationRenderer is the interface for PDF annotations.
type AnnotationRenderer interface {
	RenderDict(pageIndRef IndirectRef) Dict
	Type() AnnotationType
	RectString() string
	ID() string
	ContentString() string
}

// Annotation represents a PDF annnotation.
type Annotation struct {
	SubType  AnnotationType  // The type of annotation that this dictionary describes.
	Rect     Rectangle       // The annotation rectangle, defining the location of the annotation on the page in default user space units.
	Contents string          // Text that shall be displayed for the annotation.
	P        *IndirectRef    // An indirect reference to the page object with which this annotation is associated.
	NM       string          // (Since V1.4) The annotation name, a text string uniquely identifying it among all the annotations on its page.
	ModDate  string          // The date and time when the annotation was most recently modified.
	F        AnnotationFlags // A set of flags specifying various characteristics of the annotation.
	C        *SimpleColor    // The background color of the annotation’s icon when closed.
}

// NewAnnotation returns a new annotation.
func NewAnnotation(
	typ AnnotationType,
	rect Rectangle,
	contents string,
	pageIndRef *IndirectRef,
	nm string,
	f AnnotationFlags,
	backgrCol *SimpleColor) Annotation {

	return Annotation{
		SubType:  typ,
		Rect:     rect,
		Contents: contents,
		P:        pageIndRef,
		NM:       nm,
		F:        f,
		C:        backgrCol}
}

// NewAnnotationForRawType returns a new annotation of a specific type.
func NewAnnotationForRawType(
	typ string,
	rect Rectangle,
	contents string,
	pageIndRef *IndirectRef,
	nm string,
	f AnnotationFlags,
	backgrCol *SimpleColor) Annotation {
	return NewAnnotation(annotTypes[typ], rect, contents, pageIndRef, nm, f, backgrCol)
}

// ID returns the annotation id.
func (ann Annotation) ID() string {
	return ann.NM
}

// ContentString returns a string representation of ann's contents.
func (ann Annotation) ContentString() string {
	return ann.Contents
}

// RectString returns ann's positioning rectangle.
func (ann Annotation) RectString() string {
	return ann.Rect.ShortString()
}

// Type returns ann's type.
func (ann Annotation) Type() AnnotationType {
	return ann.SubType
}

// TypeString returns a string representation of ann's type.
func (ann Annotation) TypeString() string {
	return AnnotTypeStrings[ann.SubType]
}

// RenderDict is a stub for behavior that renders ann's PDF dict.
func (ann Annotation) RenderDict(pageIndRef IndirectRef) Dict {
	return nil
}

// PopupAnnotation represents PDF Popup annotations.
type PopupAnnotation struct {
	Annotation
	ParentIndRef *IndirectRef // The parent annotation with which this pop-up annotation shall be associated.
	Open         bool         // A flag specifying whether the annotation shall initially be displayed open.
}

// NewPopupAnnotation returns a new popup annotation.
func NewPopupAnnotation(
	rect Rectangle,
	pageIndRef *IndirectRef,
	contents, id string,
	f AnnotationFlags,
	backgrCol *SimpleColor,
	parentIndRef *IndirectRef) PopupAnnotation {

	ann := NewAnnotation(AnnPopup, rect, contents, pageIndRef, id, f, backgrCol)

	return PopupAnnotation{
		Annotation:   ann,
		ParentIndRef: parentIndRef}
}

// ContentString returns a string representation of ann's content.
func (ann PopupAnnotation) ContentString() string {
	s := "\"" + ann.Contents + "\""
	if ann.ParentIndRef != nil {
		s = "-> #" + ann.ParentIndRef.ObjectNumber.String()
	}
	return s
}

// MarkupAnnotation represents a PDF markup annotation.
type MarkupAnnotation struct {
	Annotation
	T            string       // The text label that shall be displayed in the title bar of the annotation’s pop-up window when open and active. This entry shall identify the user who added the annotation.
	PopupIndRef  *IndirectRef // An indirect reference to a pop-up annotation for entering or editing the text associated with this annotation.
	CA           *float64     // (Default: 1.0) The constant opacity value that shall be used in painting the annotation.
	RC           string       // A rich text string that shall be displayed in the pop-up window when the annotation is opened.
	CreationDate string       // The date and time when the annotation was created.
	Subj         string       // Text representing a short description of the subject being addressed by the annotation.
}

// NewMarkupAnnotation returns a new markup annotation.
func NewMarkupAnnotation(
	subType AnnotationType,
	rect Rectangle,
	pageIndRef *IndirectRef,
	contents, id, title string,
	f AnnotationFlags,
	backgrCol *SimpleColor,
	popupIndRef *IndirectRef,
	ca *float64,
	rc, subject string) MarkupAnnotation {

	ann := NewAnnotation(subType, rect, contents, pageIndRef, id, f, backgrCol)

	return MarkupAnnotation{
		Annotation:   ann,
		T:            title,
		PopupIndRef:  popupIndRef,
		CreationDate: DateString(time.Now()),
		CA:           ca,
		RC:           rc,
		Subj:         subject}
}

// TextAnnotation represents a PDF text annotation aka "Sticky Note".
type TextAnnotation struct {
	MarkupAnnotation
	Open bool   // A flag specifying whether the annotation shall initially be displayed open.
	Name string // The name of an icon that shall be used in displaying the annotation. Comment, Key, (Note), Help, NewParagraph, Paragraph, Insert
}

// NewTextAnnotation returns a new text annotation.
func NewTextAnnotation(
	rect Rectangle,
	contents, id, title string,
	f AnnotationFlags,
	backgrCol *SimpleColor,
	ca *float64,
	rc, subj string,
	open bool,
	name string) TextAnnotation {

	ma := NewMarkupAnnotation(AnnText, rect, nil, contents, id, title, f, backgrCol, nil, ca, rc, subj)

	return TextAnnotation{
		MarkupAnnotation: ma,
		Open:             open,
		Name:             name,
	}
}

// RenderDict renders ann into a PDF annotation dict.
func (ann TextAnnotation) RenderDict(pageIndRef IndirectRef) Dict {
	subject := "Sticky Note"
	if ann.Subj != "" {
		subject = ann.Subj
	}
	d := Dict(map[string]Object{
		"Type":         Name("Annot"),
		"Subtype":      Name(ann.TypeString()),
		"Rect":         ann.Rect.Array(),
		"P":            pageIndRef,
		"F":            Integer(ann.F),
		"CreationDate": StringLiteral(ann.CreationDate),
		"Subj":         StringLiteral(subject),
		"Open":         Boolean(ann.Open),
	})
	if ann.CA != nil {
		d.Insert("CA", Float(*ann.CA))
	}
	if ann.PopupIndRef != nil {
		d.Insert("Popup", *ann.PopupIndRef)
	}
	if ann.RC != "" {
		d.InsertString("RC", ann.RC)
	}
	if ann.Name != "" {
		d.InsertName("Name", ann.Name)
	}
	if ann.Contents != "" {
		d.InsertString("Contents", ann.Contents)
	}
	if ann.NM != "" {
		d.InsertString("NM", ann.NM) // check for uniqueness across annotations on this page
	} else {
		// new UUID
	}
	if ann.T != "" {
		d.InsertString("T", ann.T)
	}
	if ann.C != nil {
		d.Insert("C", NewNumberArray(float64(ann.C.R), float64(ann.C.G), float64(ann.C.B)))
	}
	return d
}

// LinkAnnotation represents a PDF link annotation.
type LinkAnnotation struct {
	Annotation
	URI  string
	Quad QuadPoints // Shall be ignored if any coordinate lies outside the region specified by Rect.
}

// NewLinkAnnotation returns a new link annotation.
func NewLinkAnnotation(
	rect Rectangle,
	quad QuadPoints,
	uri, id string,
	f AnnotationFlags,
	backgrCol *SimpleColor) LinkAnnotation {

	ann := NewAnnotation(AnnLink, rect, "", nil, id, f, backgrCol)

	return LinkAnnotation{
		Annotation: ann,
		URI:        uri,
		Quad:       quad,
	}
}

// ContentString returns a string representation of ann's content.
func (ann LinkAnnotation) ContentString() string {
	s := "(internal)"
	if len(ann.URI) > 0 {
		s = ann.URI
	}
	return s
}

// RenderDict renders ann into a PDF annotation dict.
func (ann LinkAnnotation) RenderDict(pageIndRef IndirectRef) Dict {
	actionDict := Dict(map[string]Object{
		"Type": Name("Action"),
		"S":    Name("URI"),
		"URI":  StringLiteral(ann.URI),
	})

	d := Dict(map[string]Object{
		"Type":    Name("Annot"),
		"Subtype": Name(ann.TypeString()),
		"Rect":    ann.Rect.Array(),
		"P":       pageIndRef,
		"F":       Integer(ann.F),
		"Border":  NewIntegerArray(0, 0, 0), // no border
		"H":       Name("I"),                // default
		"A":       actionDict,
	})

	if ann.NM != "" {
		d.InsertString("NM", ann.NM)
	} else {
		// new UUID
	}
	if ann.C != nil {
		d.Insert("C", NewNumberArray(float64(ann.C.R), float64(ann.C.G), float64(ann.C.B)))
	}
	if ann.Quad != nil {
		d.Insert("QuadPoints", ann.Quad.Array())
	}
	return d
}

// AnnotationObjNrs returns a list of object numbers representing known annotation dict indirect references.
func (ctx *Context) AnnotationObjNrs() ([]int, error) {
	// Note: Not all cached annotations are based on IndRefs!
	// pdfcpu also caches direct annot dict objects (violating the PDF spec) for listing purposes.
	// Such annotations may only be removecd as part of removing all annotations (for a page).

	objNrs := []int{}

	for _, pageAnnots := range ctx.PageAnnots {
		for _, annots := range pageAnnots {
			for k := range annots {
				if k[0] != '?' {
					i, err := strconv.Atoi(k)
					if err != nil {
						return nil, err
					}
					objNrs = append(objNrs, i)
				}
			}
		}
	}

	return objNrs, nil
}

func (ctx *Context) addAnnotation(ann AnnotationRenderer, pageNr int, objNr string) error {
	pgAnnots, ok := ctx.PageAnnots[pageNr]
	if !ok {
		pgAnnots = PgAnnots{}
		ctx.PageAnnots[pageNr] = pgAnnots
	}
	annots, ok := pgAnnots[ann.Type()]
	if !ok {
		annots = AnnotMap{}
		pgAnnots[ann.Type()] = annots
	}
	if _, ok := annots[objNr]; ok {
		return errors.Errorf("addAnnotation: obj#%s already cached", objNr)
	}
	annots[objNr] = ann
	return nil
}

func (ctx *Context) removeAnnotation(pageNr int, objNr string) error {
	pgAnnots, ok := ctx.PageAnnots[pageNr]
	if !ok {
		return errors.Errorf("removeAnnotation: no page annotations cached for page %d", pageNr)
	}
	for annType, annots := range pgAnnots {
		if _, ok := annots[objNr]; ok {
			delete(annots, objNr)
			if len(annots) == 0 {
				delete(pgAnnots, annType)
				if len(pgAnnots) == 0 {
					delete(ctx.PageAnnots, pageNr)
				}
			}
			return nil
		}
	}
	return errors.Errorf("removeAnnotation: no page annotation cached for obj#%s", objNr)
}

func (ctx *Context) findAnnotByID(id string, annots Array) (int, error) {
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

func (ctx *Context) findAnnotByObjNr(objNr int, annots Array) (int, error) {
	for i, o := range annots {
		indRef, _ := o.(IndirectRef)
		if indRef.ObjectNumber.Value() == objNr {
			return i, nil
		}
	}
	return -1, nil
}

func (ctx *Context) createAnnot(ar AnnotationRenderer, pageIndRef *IndirectRef) (*IndirectRef, error) {
	d := ar.RenderDict(*pageIndRef)
	return ctx.IndRefForNewObject(d)
}

// Annotation returns an annotation renderer.
// Validation sets up a cache of annotation renderers.
func (xRefTable *XRefTable) Annotation(d Dict) (AnnotationRenderer, error) {

	subtype := d.NameEntry("Subtype")

	o, _ := d.Find("Rect")
	arr, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return nil, err
	}

	r, err := RectForArray(arr)
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

	var f AnnotationFlags
	i := d.IntEntry("F")
	if i != nil {
		f = AnnotationFlags(*i)
	}

	var ann AnnotationRenderer

	switch *subtype {

	case "Text":
		ann = NewTextAnnotation(*r, contents, nm, "", f, nil, nil, "", "", true, "")

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
		ann = NewLinkAnnotation(*r, nil, uri, nm, f, nil)

	case "Popup":
		parentIndRef := d.IndirectRefEntry("Parent")
		ann = NewPopupAnnotation(*r, nil, contents, nm, f, nil, parentIndRef)

	// TODO handle remaining annotation types.

	default:
		ann = NewAnnotationForRawType(*subtype, *r, contents, nil, nm, f, nil)
	}

	return ann, nil
}

// AnnotMap represents annotations by object number of the corresponding annotation dict.
type AnnotMap map[string]AnnotationRenderer

// PgAnnots represents a map of page annotations by type.
type PgAnnots map[AnnotationType]AnnotMap

// ListAnnotations returns a formatted list of annotations for selected pages.
func (xRefTable *XRefTable) ListAnnotations(selectedPages IntSet) (int, []string, error) {
	var (
		j       int
		pageNrs []int
	)
	ss := []string{}

	for k := range xRefTable.PageAnnots {
		pageNrs = append(pageNrs, k)
	}
	sort.Ints(pageNrs)

	for _, i := range pageNrs {
		if selectedPages != nil {
			if _, found := selectedPages[i]; !found {
				continue
			}
		}
		pageAnnots := xRefTable.PageAnnots[i]
		if len(pageAnnots) == 0 {
			continue
		}
		ss = append(ss, "")
		ss = append(ss, fmt.Sprintf("Page %d:", i))
		for annType, annots := range pageAnnots {
			var (
				maxLenRect    int
				maxLenContent int
			)
			maxLenID := 2
			for _, ann := range annots {
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
			ss = append(ss, "")
			ss = append(ss, fmt.Sprintf("  %s:", AnnotTypeStrings[annType]))
			s1 := ("     obj# ")
			s2 := fmt.Sprintf("%%%ds", maxLenRect)
			s3 := fmt.Sprintf("%%%ds", maxLenID)
			s4 := fmt.Sprintf("%%%ds", maxLenContent)
			s := fmt.Sprintf(s1+s2+" "+s3+" "+s4, "rect", "id", "content")
			ss = append(ss, s)
			ss = append(ss, "    "+strings.Repeat("=", len(s)-4))

			for objNr, ann := range annots {
				s := "?"
				if objNr[0] != '?' {
					s = objNr
				}
				ss = append(ss, fmt.Sprintf("    %5s "+s2+" "+s3+" "+s4, s, ann.RectString(), ann.ID(), ann.ContentString()))
				j++
			}
		}
	}

	return j, append([]string{fmt.Sprintf("%d annotations available", j)}, ss...), nil
}

func (ctx *Context) addAnnotationToDirectObj(
	annots Array,
	annotIndRef, pageDictIndRef *IndirectRef,
	pageDict Dict,
	pageNr int,
	ar AnnotationRenderer,
	incr bool) (bool, error) {

	i, err := ctx.findAnnotByID(ar.ID(), annots)
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
func (ctx *Context) AddAnnotation(pageDictIndRef *IndirectRef, pageDict Dict, pageNr int, ar AnnotationRenderer, incr bool) (bool, error) {
	// Create xreftable entry for annotation.
	annotIndRef, err := ctx.createAnnot(ar, pageDictIndRef)
	if err != nil {
		return false, err
	}

	// Add annotation to xreftable page annotation cache.
	err = ctx.addAnnotation(ar, pageNr, annotIndRef.ObjectNumber.String())
	if err != nil {
		return false, err
	}

	if incr {
		// Mark new annotaton dict obj for incremental writing.
		ctx.Write.IncrementWithObjNr(annotIndRef.ObjectNumber.Value())
	}

	obj, found := pageDict.Find("Annots")
	if !found {
		pageDict.Insert("Annots", Array{*annotIndRef})
		if incr {
			// Mark page dict obj for incremental writing.
			ctx.Write.IncrementWithObjNr(pageDictIndRef.ObjectNumber.Value())
		}
		ctx.EnsureVersionForWriting()
		return true, nil
	}

	ir, ok := obj.(IndirectRef)
	if !ok {
		return ctx.addAnnotationToDirectObj(obj.(Array), annotIndRef, pageDictIndRef, pageDict, pageNr, ar, incr)
	}

	// Annots array is an IndirectReference.

	o, err := ctx.Dereference(ir)
	if err != nil || o == nil {
		return false, err
	}

	annots, _ := o.(Array)
	i, err := ctx.findAnnotByID(ar.ID(), annots)
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
func (ctx *Context) AddAnnotations(selectedPages IntSet, ar AnnotationRenderer, incr bool) (bool, error) {
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

		added, err := ctx.AddAnnotation(pageDictIndRef, d, k, ar, incr)
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
func (ctx *Context) AddAnnotationsMap(m map[int][]AnnotationRenderer, incr bool) (bool, error) {
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
			added, err := ctx.AddAnnotation(pageDictIndRef, d, i, annot, incr)
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

func (ctx *Context) removeAllAnnotations(pageDict Dict, pageDictObjNr, pageNr int, incr bool) (bool, error) {
	var err error
	obj, found := pageDict.Find("Annots")
	if !found {
		return false, nil
	}

	ir, ok := obj.(IndirectRef)
	if ok {
		obj, err = ctx.Dereference(ir)
		if err != nil || obj == nil {
			return false, err
		}
		objNr := ir.ObjectNumber.Value()
		if err := ctx.deleteObject(ir); err != nil {
			return false, err
		}
		if incr {
			// Modify Annots array obj for incremental writing.
			ctx.Write.IncrementWithObjNr(objNr)
		}
	}
	annots, _ := obj.(Array)

	for _, o := range annots {
		ir, _ := o.(IndirectRef)
		objNr := ir.ObjectNumber.Value()
		if err := ctx.deleteObject(ir); err != nil {
			return false, err
		}
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

	return true, nil
}

func (ctx *Context) removeAnnotationsFromPageDictByID(ids []string, pageNr int, annots Array, incr bool) (Array, bool, error) {
	var ok bool
	for _, id := range ids {
		i, err := ctx.findAnnotByID(id, annots)
		if err != nil {
			return nil, false, err
		}
		if i >= 0 {
			ok = true
			ir1, _ := annots[i].(IndirectRef)

			// Remove annotation from xreftable page annotation cache.
			err = ctx.removeAnnotation(pageNr, ir1.ObjectNumber.String())
			if err != nil {
				return nil, false, err
			}

			if err := ctx.deleteObject(ir1); err != nil {
				return nil, false, err
			}
			if incr {
				// Mark annotation dict obj for incremental writing.
				ctx.Write.IncrementWithObjNr(ir1.ObjectNumber.Value())
			}
			if len(annots) == 1 {
				return nil, ok, nil
			}
			annots = append(annots[:i], annots[i+1:]...)
		}
	}
	return annots, ok, nil
}

func (ctx *Context) removeAnnotationsFromPageDictByObjNr(objNrSet IntSet, pageNr int, annots Array, incr bool) (Array, bool, error) {
	var ok bool
	for objNr, v := range objNrSet {
		if !v {
			continue
		}
		i, err := ctx.findAnnotByObjNr(objNr, annots)
		if err != nil {
			return nil, false, err
		}
		if i >= 0 {
			ok = true
			ir1, _ := annots[i].(IndirectRef)

			// Remove annotation from xreftable page annotation cache.
			err = ctx.removeAnnotation(pageNr, ir1.ObjectNumber.String())
			if err != nil {
				return nil, false, err
			}

			if err := ctx.deleteObject(ir1); err != nil {
				return nil, false, err
			}
			if incr {
				// Mark annotation dict obj for incremental writing.
				ctx.Write.IncrementWithObjNr(ir1.ObjectNumber.Value())
			}
			delete(objNrSet, objNr)
			if len(annots) == 1 {
				return nil, ok, nil
			}
			annots = append(annots[:i], annots[i+1:]...)
		}
	}
	return annots, ok, nil
}

func (ctx *Context) removeAnnotationsFromPageDictByIDOrObjNr(ids []string, objNrSet IntSet, pageNr int, annots Array, incr bool) (Array, bool, error) {
	var (
		ok  bool
		err error
	)

	if len(ids) > 0 {
		annots, ok, err = ctx.removeAnnotationsFromPageDictByID(ids, pageNr, annots, incr)
		if err != nil || annots == nil {
			return nil, ok, err
		}
	}

	return ctx.removeAnnotationsFromPageDictByObjNr(objNrSet, pageNr, annots, incr)
}

func (ctx *Context) removeAnnotationsFromDirectObj(
	annots Array,
	ids []string,
	objNrSet IntSet,
	pageDict Dict,
	pageDictObjNr, pageNr int,
	incr bool) (bool, error) {

	var (
		ok  bool
		err error
	)
	annots, ok, err = ctx.removeAnnotationsFromPageDictByIDOrObjNr(ids, objNrSet, pageNr, annots, incr)
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
	if annots == nil {
		pageDict.Delete("Annots")
		return ok, nil
	}
	pageDict.Update("Annots", annots)
	return true, nil
}

// RemoveAnnotationsFromPageDict removes an annotation by its object number annObjNr from pageDict.
func (ctx *Context) RemoveAnnotationsFromPageDict(ids []string, objNrSet IntSet, pageDict Dict, pageDictObjNr, pageNr int, incr bool) (bool, error) {
	var (
		ok  bool
		err error
	)

	defer func() {
		if ok {
			ctx.EnsureVersionForWriting()
		}
	}()

	//fmt.Printf("ids:%v objNrSet:%v\n", ids, objNrSet)

	if len(ids) == 0 && len(objNrSet) == 0 {
		ok, err = ctx.removeAllAnnotations(pageDict, pageDictObjNr, pageNr, incr)
		return ok, err
	}

	obj, found := pageDict.Find("Annots")
	if !found {
		return false, nil
	}

	ir, ok1 := obj.(IndirectRef)
	if !ok1 {
		return ctx.removeAnnotationsFromDirectObj(obj.(Array), ids, objNrSet, pageDict, pageDictObjNr, pageNr, incr)
	}

	// Annots array is an IndirectReference.
	o, err := ctx.Dereference(ir)
	if err != nil || o == nil {
		return false, err
	}
	annots, _ := o.(Array)
	annots, ok, err = ctx.removeAnnotationsFromPageDictByIDOrObjNr(ids, objNrSet, pageNr, annots, incr)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	objNr := ir.ObjectNumber.Value()
	genNr := ir.GenerationNumber.Value()
	entry, _ := ctx.FindTableEntry(objNr, genNr)
	if incr {
		// Modify Annots array obj for incremental writing.
		ctx.Write.IncrementWithObjNr(objNr)
	}
	if annots == nil {
		pageDict.Delete("Annots")
		if err := ctx.deleteObject(ir); err != nil {
			return false, err
		}
		if incr {
			// Mark page dict obj for incremental writing.
			ctx.Write.IncrementWithObjNr(pageDictObjNr)
		}
		return ok, nil
	}
	entry.Object = annots
	return true, nil
}

func (ctx *Context) sortedPageNrsForAnnots() []int {
	var pageNrs []int
	for k := range ctx.PageAnnots {
		pageNrs = append(pageNrs, k)
	}
	sort.Ints(pageNrs)
	return pageNrs
}

// RemoveAnnotations removes annotations for selected pages by id and object number.
// All annotations for selected pages are removed if neither ids nor objNrs are provided.
func (ctx *Context) RemoveAnnotations(selectedPages IntSet, ids []string, objNrs []int, incr bool) (bool, error) {
	// Note: Selected pages only apply if no objNrs provided.
	var removed bool

	// Remove all annotations for selectedPages
	removeAll := len(ids) == 0 && len(objNrs) == 0
	if removeAll {
		log.CLI.Println("removing all annotations for selected pages!")
	}

	objNrSet := IntSet{}
	for _, i := range objNrs {
		objNrSet[i] = true
	}

	if incr {
		ctx.Write.Increment = true
		ctx.Write.Offset = ctx.Read.FileSize
	}

	for _, pageNr := range ctx.sortedPageNrsForAnnots() {
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

		ok, err := ctx.RemoveAnnotationsFromPageDict(ids, objNrSet, d, objNr, pageNr, incr)
		if err != nil {
			return false, err
		}
		if ok {
			removed = true
		}
		if !removeAll && len(ids) == 0 && len(objNrSet) == 0 {
			// If all annotations with objNrs are removed
			// and we don't need to remove by id we are done here!
			break
		}
	}

	return removed, nil
}
