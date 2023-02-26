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
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
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

var AnnotTypes = map[string]AnnotationType{
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
	return NewAnnotation(AnnotTypes[typ], rect, contents, pageIndRef, nm, f, col)
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
