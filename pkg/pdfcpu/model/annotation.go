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

package model

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
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
	RenderDict(xRefTable *XRefTable, pageIndRef types.IndirectRef) (types.Dict, error)
	Type() AnnotationType
	RectString() string
	ID() string
	ContentString() string
}

// Annotation represents a PDF annnotation.
type Annotation struct {
	SubType  AnnotationType     // The type of annotation that this dictionary describes.
	Rect     types.Rectangle    // The annotation rectangle, defining the location of the annotation on the page in default user space units.
	Contents string             // Text that shall be displayed for the annotation.
	P        *types.IndirectRef // An indirect reference to the page object with which this annotation is associated.
	NM       string             // (Since V1.4) The annotation name, a text string uniquely identifying it among all the annotations on its page.
	ModDate  string             // The date and time when the annotation was most recently modified.
	F        AnnotationFlags    // A set of flags specifying various characteristics of the annotation.
	C        *color.SimpleColor // The background color of the annotation’s icon when closed.
}

// NewAnnotation returns a new annotation.
func NewAnnotation(
	typ AnnotationType,
	rect types.Rectangle,
	contents string,
	pageIndRef *types.IndirectRef,
	nm string,
	f AnnotationFlags,
	col *color.SimpleColor) Annotation {

	return Annotation{
		SubType:  typ,
		Rect:     rect,
		Contents: contents,
		P:        pageIndRef,
		NM:       nm,
		F:        f,
		C:        col}
}

// NewAnnotationForRawType returns a new annotation of a specific type.
func NewAnnotationForRawType(
	typ string,
	rect types.Rectangle,
	contents string,
	pageIndRef *types.IndirectRef,
	nm string,
	f AnnotationFlags,
	col *color.SimpleColor) Annotation {
	return NewAnnotation(annotTypes[typ], rect, contents, pageIndRef, nm, f, col)
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
func (ann Annotation) RenderDict(xRefTable *XRefTable, pageIndRef types.IndirectRef) (types.Dict, error) {
	return nil, nil
}

// PopupAnnotation represents PDF Popup annotations.
type PopupAnnotation struct {
	Annotation
	ParentIndRef *types.IndirectRef // The parent annotation with which this pop-up annotation shall be associated.
	Open         bool               // A flag specifying whether the annotation shall initially be displayed open.
}

// NewPopupAnnotation returns a new popup annotation.
func NewPopupAnnotation(
	rect types.Rectangle,
	pageIndRef *types.IndirectRef,
	contents, id string,
	f AnnotationFlags,
	bgCol *color.SimpleColor,
	parentIndRef *types.IndirectRef) PopupAnnotation {

	ann := NewAnnotation(AnnPopup, rect, contents, pageIndRef, id, f, bgCol)

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
	T            string             // The text label that shall be displayed in the title bar of the annotation’s pop-up window when open and active. This entry shall identify the user who added the annotation.
	PopupIndRef  *types.IndirectRef // An indirect reference to a pop-up annotation for entering or editing the text associated with this annotation.
	CA           *float64           // (Default: 1.0) The constant opacity value that shall be used in painting the annotation.
	RC           string             // A rich text string that shall be displayed in the pop-up window when the annotation is opened.
	CreationDate string             // The date and time when the annotation was created.
	Subj         string             // Text representing a short description of the subject being addressed by the annotation.
}

// NewMarkupAnnotation returns a new markup annotation.
func NewMarkupAnnotation(
	subType AnnotationType,
	rect types.Rectangle,
	pageIndRef *types.IndirectRef,
	contents, id, title string,
	f AnnotationFlags,
	bgCol *color.SimpleColor,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string) MarkupAnnotation {

	ann := NewAnnotation(subType, rect, contents, pageIndRef, id, f, bgCol)

	return MarkupAnnotation{
		Annotation:   ann,
		T:            title,
		PopupIndRef:  popupIndRef,
		CreationDate: types.DateString(time.Now()),
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
	rect types.Rectangle,
	contents, id, title string,
	f AnnotationFlags,
	bgCol *color.SimpleColor,
	ca *float64,
	rc, subj string,
	open bool,
	name string) TextAnnotation {

	ma := NewMarkupAnnotation(AnnText, rect, nil, contents, id, title, f, bgCol, nil, ca, rc, subj)

	return TextAnnotation{
		MarkupAnnotation: ma,
		Open:             open,
		Name:             name,
	}
}

// RenderDict renders ann into a PDF annotation dict.
func (ann TextAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef types.IndirectRef) (types.Dict, error) {
	subject := "Sticky Note"
	if ann.Subj != "" {
		subject = ann.Subj
	}
	d := types.Dict(map[string]types.Object{
		"Type":         types.Name("Annot"),
		"Subtype":      types.Name(ann.TypeString()),
		"Rect":         ann.Rect.Array(),
		"P":            pageIndRef,
		"F":            types.Integer(ann.F),
		"CreationDate": types.StringLiteral(ann.CreationDate),
		"Subj":         types.StringLiteral(subject),
		"Open":         types.Boolean(ann.Open),
	})
	if ann.CA != nil {
		d.Insert("CA", types.Float(*ann.CA))
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
		d.InsertString("NM", ann.NM) // TODO check for uniqueness across annotations on this page.
	}
	if ann.T != "" {
		d.InsertString("T", ann.T)
	}
	if ann.C != nil {
		d.Insert("C", ann.C.Array())
	}
	return d, nil
}

// A series of alternating x and y coordinates in PDF user space, specifying points along the path.
type InkPath []float64

type InkAnnotation struct {
	MarkupAnnotation
	InkList []InkPath
	BS      *types.Dict
	AP      *types.Dict
}

// NewInkAnnotation returns a new ink annotation.
func NewInkAnnotation(
	rect types.Rectangle,
	contents, id, title string,
	ink []InkPath,
	bs *types.Dict,
	f AnnotationFlags,
	bgCol *color.SimpleColor,
	ca *float64,
	rc, subj string,
	ap *types.Dict,
) InkAnnotation {

	ann := NewMarkupAnnotation(AnnInk, rect, nil, contents, id, title, f, bgCol, nil, ca, rc, subj)

	return InkAnnotation{
		MarkupAnnotation: ann,
		InkList:          ink,
		BS:               bs,
		AP:               ap,
	}
}

func (ann InkAnnotation) RenderDict(pageIndRef types.IndirectRef) types.Dict {
	subject := "Ink Annotation"
	if ann.Subj != "" {
		subject = ann.Subj
	}
	ink := types.Array{}
	for i := range ann.InkList {
		ink = append(ink, types.NewNumberArray(ann.InkList[i]...))
	}

	d := types.Dict(map[string]types.Object{
		"Type":         types.Name("Annot"),
		"Subtype":      types.Name(ann.TypeString()),
		"Rect":         ann.Rect.Array(),
		"P":            pageIndRef,
		"F":            types.Integer(ann.F),
		"CreationDate": types.StringLiteral(ann.CreationDate),
		"Subj":         types.StringLiteral(subject),
		"InkList":      ink,
	})
	if ann.AP != nil {
		d.Insert("AP", *ann.AP)
	}
	if ann.CA != nil {
		d.Insert("CA", types.Float(*ann.CA))
	}
	if ann.PopupIndRef != nil {
		d.Insert("Popup", *ann.PopupIndRef)
	}
	if ann.RC != "" {
		d.InsertString("RC", ann.RC)
	}
	if ann.BS != nil {
		d.Insert("BS", ann.BS)
	}
	if ann.Contents != "" {
		d.InsertString("Contents", ann.Contents)
	}
	if ann.NM != "" {
		d.InsertString("NM", ann.NM) // TODO check for uniqueness across annotations on this page.
	}
	if ann.T != "" {
		d.InsertString("T", ann.T)
	}
	if ann.C != nil {
		d.Insert("C", ann.C.Array())
	}

	return d
}

// LinkAnnotation represents a PDF link annotation.
type LinkAnnotation struct {
	Annotation
	Dest   *Destination     // internal link
	URI    string           // external link
	Quad   types.QuadPoints // shall be ignored if any coordinate lies outside the region specified by Rect.
	Border bool             // render border using borderColor.
}

// NewLinkAnnotation returns a new link annotation.
func NewLinkAnnotation(
	rect types.Rectangle,
	quad types.QuadPoints,
	dest *Destination, // supply dest or uri, dest takes precedence
	uri string,
	id string,
	f AnnotationFlags,
	borderCol *color.SimpleColor,
	border bool) LinkAnnotation {

	ann := NewAnnotation(AnnLink, rect, "", nil, id, f, borderCol)

	return LinkAnnotation{
		Annotation: ann,
		Dest:       dest,
		URI:        uri,
		Quad:       quad,
		Border:     border,
	}
}

// ContentString returns a string representation of ann's content.
func (ann LinkAnnotation) ContentString() string {
	if len(ann.URI) > 0 {
		return ann.URI
	}
	if ann.Dest != nil {
		// eg. page /XYZ left top zoom
		return fmt.Sprintf("Page %d %s", ann.Dest.PageNr, ann.Dest)
	}
	return "internal link"
}

// RenderDict renders ann into a page annotation dict.
func (ann LinkAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef types.IndirectRef) (types.Dict, error) {

	d := types.Dict(map[string]types.Object{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name(ann.TypeString()),
		"Rect":    ann.Rect.Array(),
		"P":       pageIndRef,
		"F":       types.Integer(ann.F),
	})

	if !ann.Border {
		d["Border"] = types.NewIntegerArray(0, 0, 0)
	} else {
		if ann.C != nil {
			d["C"] = ann.C.Array()
		}
	}

	if ann.Dest != nil {
		dest := ann.Dest
		if dest.Zoom == 0 {
			dest.Zoom = 1
		}
		_, indRef, pAttr, err := xRefTable.PageDict(dest.PageNr, false)
		if err != nil {
			return nil, err
		}
		if dest.Typ == DestXYZ && dest.Left < 0 && dest.Top < 0 {
			// Show top left corner of destination page.
			dest.Left = int(pAttr.MediaBox.LL.X)
			dest.Top = int(pAttr.MediaBox.UR.Y)
			if pAttr.CropBox != nil {
				dest.Left = int(pAttr.CropBox.LL.X)
				dest.Top = int(pAttr.CropBox.UR.Y)
			}
		}
		d["Dest"] = dest.Array(*indRef)
	} else {
		actionDict := types.Dict(map[string]types.Object{
			"Type": types.Name("Action"),
			"S":    types.Name("URI"),
			"URI":  types.StringLiteral(ann.URI),
		})
		d["A"] = actionDict
	}
	if ann.NM != "" {
		d.InsertString("NM", ann.NM) // TODO check for uniqueness across annotations on this page.
	}
	if ann.Quad != nil {
		d.Insert("QuadPoints", ann.Quad.Array())
	}
	return d, nil
}

// AnnotMap represents annotations by object number of the corresponding annotation dict.
type AnnotMap map[int]AnnotationRenderer

type Annot struct {
	IndRefs *[]types.IndirectRef
	Map     AnnotMap
}

// PgAnnots represents a map of page annotations by type.
type PgAnnots map[AnnotationType]Annot

// AnnotationObjNrs returns a list of object numbers representing known annotation dict indirect references.
func (ctx *Context) AnnotationObjNrs() ([]int, error) {
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

func (ctx *Context) addAnnotation(ann AnnotationRenderer, pageNr, objNr int) error {
	pgAnnots, ok := ctx.PageAnnots[pageNr]
	if !ok {
		pgAnnots = PgAnnots{}
		ctx.PageAnnots[pageNr] = pgAnnots
	}
	annots, ok := pgAnnots[ann.Type()]
	if !ok {
		annots = Annot{}
		annots.Map = AnnotMap{}
		pgAnnots[ann.Type()] = annots
	}
	if _, ok := annots.Map[objNr]; ok {
		return errors.Errorf("addAnnotation: obj#%d already cached", objNr)
	}
	annots.Map[objNr] = ann
	return nil
}

func (ctx *Context) removeAnnotation(pageNr, objNr int) error {
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

func (ctx *Context) findAnnotByID(id string, annots types.Array) (int, error) {
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

func (ctx *Context) findAnnotByObjNr(objNr int, annots types.Array) (int, error) {
	for i, o := range annots {
		indRef, _ := o.(types.IndirectRef)
		if indRef.ObjectNumber.Value() == objNr {
			return i, nil
		}
	}
	return -1, nil
}

func (ctx *Context) createAnnot(ar AnnotationRenderer, pageIndRef *types.IndirectRef) (*types.IndirectRef, error) {
	d, err := ar.RenderDict(ctx.XRefTable, *pageIndRef)
	if err != nil {
		return nil, err
	}
	return ctx.IndRefForNewObject(d)
}

// Annotation returns an annotation renderer.
// Validation sets up a cache of annotation renderers.
func (xRefTable *XRefTable) Annotation(d types.Dict) (AnnotationRenderer, error) {

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
		dest := (*Destination)(nil) // will not collect link dest during validation.
		ann = NewLinkAnnotation(*r, nil, dest, uri, nm, f, nil, false)

	case "Popup":
		parentIndRef := d.IndirectRefEntry("Parent")
		ann = NewPopupAnnotation(*r, nil, contents, nm, f, nil, parentIndRef)

	// TODO handle remaining annotation types.

	default:
		ann = NewAnnotationForRawType(*subtype, *r, contents, nil, nm, f, nil)
	}

	return ann, nil
}

// ListAnnotations returns a formatted list of annotations for selected pages.
func (xRefTable *XRefTable) ListAnnotations(selectedPages types.IntSet) (int, []string, error) {
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

		var annTypes []string
		for t := range pageAnnots {
			annTypes = append(annTypes, AnnotTypeStrings[t])
		}
		sort.Strings(annTypes)

		ss = append(ss, "")
		ss = append(ss, fmt.Sprintf("Page %d:", i))

		for _, annType := range annTypes {
			annots := pageAnnots[annotTypes[annType]]
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

func (ctx *Context) addAnnotationToDirectObj(
	annots types.Array,
	annotIndRef, pageDictIndRef *types.IndirectRef,
	pageDict types.Dict,
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
func (ctx *Context) AddAnnotation(pageDictIndRef *types.IndirectRef, pageDict types.Dict, pageNr int, ar AnnotationRenderer, incr bool) (bool, error) {
	// Create xreftable entry for annotation.
	annotIndRef, err := ctx.createAnnot(ar, pageDictIndRef)
	if err != nil {
		return false, err
	}

	// Add annotation to xreftable page annotation cache.
	err = ctx.addAnnotation(ar, pageNr, annotIndRef.ObjectNumber.Value())
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
		return ctx.addAnnotationToDirectObj(obj.(types.Array), annotIndRef, pageDictIndRef, pageDict, pageNr, ar, incr)
	}

	// Annots array is an IndirectReference.

	o, err := ctx.Dereference(ir)
	if err != nil || o == nil {
		return false, err
	}

	annots, _ := o.(types.Array)
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
func (ctx *Context) AddAnnotations(selectedPages types.IntSet, ar AnnotationRenderer, incr bool) (bool, error) {
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

func (ctx *Context) removeAllAnnotations(pageDict types.Dict, pageDictObjNr, pageNr int, incr bool) (bool, error) {
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

	return true, nil
}

func (ctx *Context) removeAnnotationsFromPageDictByType(annotTypes []AnnotationType, pageNr int, annots types.Array, incr bool) (types.Array, bool, error) {

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
			i, err := ctx.findAnnotByObjNr(objNr, annots)
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

func (ctx *Context) removeAnnotationFromPageDictByID(id string, pageNr int, annots types.Array, incr bool) (types.Array, bool, error) {
	i, err := ctx.findAnnotByID(id, annots)
	if err != nil || i < 0 {
		return annots, false, err
	}

	indRef, _ := annots[i].(types.IndirectRef)

	// Remove annotation from xreftable page annotation cache.
	err = ctx.removeAnnotation(pageNr, indRef.ObjectNumber.Value())
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

func (ctx *Context) removeAnnotationsFromPageDictByID(ids []string, objNrSet types.IntSet, pageNr int, annots types.Array, incr bool) (types.Array, bool, error) {

	var (
		ok, ok1 bool
		err     error
	)

	for _, id := range ids {
		annots, ok1, err = ctx.removeAnnotationFromPageDictByID(id, pageNr, annots, incr)
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
		annots, ok1, err = ctx.removeAnnotationFromPageDictByID(strconv.Itoa(objNr), pageNr, annots, incr)
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

func (ctx *Context) removeAnnotationsFromPageDictByObjNr(objNrSet types.IntSet, pageNr int, annots types.Array, incr bool) (types.Array, bool, error) {
	var ok bool
	for objNr, v := range objNrSet {
		if !v || objNr < 0 {
			continue
		}
		i, err := ctx.findAnnotByObjNr(objNr, annots)
		if err != nil {
			return nil, false, err
		}
		if i >= 0 {
			ok = true
			indRef, _ := annots[i].(types.IndirectRef)

			// Remove annotation from xreftable page annotation cache.
			err = ctx.removeAnnotation(pageNr, indRef.ObjectNumber.Value())
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

func (ctx *Context) removeAnnotationsFromPageDict(
	annotTypes []AnnotationType,
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
		annots, ok1, err = ctx.removeAnnotationsFromPageDictByType(annotTypes, pageNr, annots, incr)
		if err != nil || annots == nil {
			return nil, ok1, err
		}
	}

	// 2. Remove by obj#.
	if len(objNrSet) > 0 {
		annots, ok2, err = ctx.removeAnnotationsFromPageDictByObjNr(objNrSet, pageNr, annots, incr)
		if err != nil || annots == nil {
			return nil, ok2, err
		}
	}

	// 3. Remove by id for ids and objNrs considering possibly numeric ids.
	if len(ids) > 0 || len(objNrSet) > 0 {
		annots, ok3, err = ctx.removeAnnotationsFromPageDictByID(ids, objNrSet, pageNr, annots, incr)
		if err != nil || annots == nil {
			return nil, ok3, err
		}
	}

	return annots, ok1 || ok2 || ok3, nil
}

// RemoveAnnotationsFromPageDict removes an annotation by annotType, id and obj# from pageDict.
func (ctx *Context) RemoveAnnotationsFromPageDict(
	annotTypes []AnnotationType,
	ids []string,
	objNrSet types.IntSet,
	pageDict types.Dict,
	pageDictObjNr,
	pageNr int,
	incr bool) (bool, error) {

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

	if len(annotTypes) == 0 && len(ids) == 0 && len(objNrSet) == 0 {
		ok, err = ctx.removeAllAnnotations(pageDict, pageDictObjNr, pageNr, incr)
		return ok, err
	}

	obj, found := pageDict.Find("Annots")
	if !found {
		return false, nil
	}

	ir, ok1 := obj.(types.IndirectRef)
	if !ok1 {
		annots, _ := obj.(types.Array)
		annots, ok, err = ctx.removeAnnotationsFromPageDict(annotTypes, ids, objNrSet, pageNr, annots, incr)
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
		return ok, nil
		//return ctx.removeAnnotationsFromDirectObj(obj.(types.Array), annotTypes, ids, objNrSet, pageDict, pageDictObjNr, pageNr, incr)
	}

	// Annots array is an IndirectReference.
	o, err := ctx.Dereference(ir)
	if err != nil || o == nil {
		return false, err
	}

	annots, _ := o.(types.Array)
	annots, ok, err = ctx.removeAnnotationsFromPageDict(annotTypes, ids, objNrSet, pageNr, annots, incr)
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
		if err := ctx.DeleteObject(ir); err != nil {
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

// RemoveAnnotations removes annotations for selected pages by id, type or object number.
// All annotations for selected pages are removed if neither idsAndTypes nor objNrs are provided.
func (ctx *Context) RemoveAnnotations(selectedPages types.IntSet, idsAndTypes []string, objNrs []int, incr bool) (bool, error) {

	var annTypes []AnnotationType
	var ids []string

	if len(idsAndTypes) > 0 {
		for _, s := range idsAndTypes {
			if at, ok := annotTypes[s]; ok {
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

	var removed bool

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

		ok, err := ctx.RemoveAnnotationsFromPageDict(annTypes, ids, objNrSet, d, objNr, pageNr, incr)
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
