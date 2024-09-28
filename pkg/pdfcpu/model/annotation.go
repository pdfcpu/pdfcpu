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

var AnnotTypes = map[string]AnnotationType{
	"Text":           AnnText,
	"Link":           AnnLink,
	"FreeText":       AnnFreeText,
	"Line":           AnnLine,
	"Square":         AnnSquare,
	"Circle":         AnnCircle,
	"Polygon":        AnnPolygon,
	"PolyLine":       AnnPolyLine,
	"Highlight":      AnnHighLight,
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
	AnnHighLight:      "Highlight",
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

// BorderStyle (see table 168)
type BorderStyle int

const (
	BSSolid BorderStyle = iota
	BSDashed
	BSBeveled
	BSInset
	BSUnderline
)

func borderStyleDict(width float64, style BorderStyle) types.Dict {
	d := types.Dict(map[string]types.Object{
		"Type": types.Name("Border"),
		"W":    types.Float(width),
	})

	var s string

	switch style {
	case BSSolid:
		s = "S"
	case BSDashed:
		s = "D"
	case BSBeveled:
		s = "B"
	case BSInset:
		s = "I"
	case BSUnderline:
		s = "U"
	}

	d["S"] = types.Name(s)

	return d
}

func borderEffectDict(cloudyBorder bool, intensity int) types.Dict {
	s := "S"
	if cloudyBorder {
		s = "C"
	}

	return types.Dict(map[string]types.Object{
		"S": types.Name(s),
		"I": types.Integer(intensity),
	})
}

func borderArray(rx, ry, width float64) types.Array {
	return types.NewNumberArray(rx, ry, width)
}

// LineEndingStyle (see table 179)
type LineEndingStyle int

const (
	LESquare LineEndingStyle = iota
	LECircle
	LEDiamond
	LEOpenArrow
	LEClosedArrow
	LENone
	LEButt
	LEROpenArrow
	LERClosedArrow
	LESlash
)

func LineEndingStyleName(les LineEndingStyle) string {
	var s string
	switch les {
	case LESquare:
		s = "Square"
	case LECircle:
		s = "Circle"
	case LEDiamond:
		s = "Diamond"
	case LEOpenArrow:
		s = "OpenArrow"
	case LEClosedArrow:
		s = "ClosedArrow"
	case LENone:
		s = "None"
	case LEButt:
		s = "Butt"
	case LEROpenArrow:
		s = "ROpenArrow"
	case LERClosedArrow:
		s = "RClosedArrow"
	case LESlash:
		s = "Slash"
	}
	return s
}

// AnnotationRenderer is the interface for PDF annotations.
type AnnotationRenderer interface {
	RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error)
	Type() AnnotationType
	RectString() string
	ID() string
	ContentString() string
}

// Annotation represents a PDF annnotation.
type Annotation struct {
	SubType          AnnotationType     // The type of annotation that this dictionary describes.
	Rect             types.Rectangle    // The annotation rectangle, defining the location of the annotation on the page in default user space units.
	Contents         string             // Text that shall be displayed for the annotation.
	NM               string             // (Since V1.4) The annotation name, a text string uniquely identifying it among all the annotations on its page.
	ModificationDate string             // M - The date and time when the annotation was most recently modified.
	P                *types.IndirectRef // An indirect reference to the page object with which this annotation is associated.
	F                AnnotationFlags    // A set of flags specifying various characteristics of the annotation.
	C                *color.SimpleColor // The background color of the annotation’s icon when closed, pop up title bar color, link ann border color.
	BorderRadX       float64            // Border radius X
	BorderRadY       float64            // Border radius Y
	BorderWidth      float64            // Border width
	// StructParent int
	// OC types.dict
}

// NewAnnotation returns a new annotation.
func NewAnnotation(
	typ AnnotationType,
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	borderRadX float64,
	borderRadY float64,
	borderWidth float64) Annotation {

	return Annotation{
		SubType:          typ,
		Rect:             rect,
		Contents:         contents,
		NM:               id,
		ModificationDate: modDate,
		F:                f,
		C:                col,
		BorderRadX:       borderRadX,
		BorderRadY:       borderRadY,
		BorderWidth:      borderWidth,
	}
}

// NewAnnotationForRawType returns a new annotation of a specific type.
func NewAnnotationForRawType(
	typ string,
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,

	col *color.SimpleColor,
	borderRadX float64,
	borderRadY float64,
	borderWidth float64) Annotation {
	return NewAnnotation(AnnotTypes[typ], rect, contents, id, modDate, f, col, borderRadX, borderRadY, borderWidth)
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

func (ann Annotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {
	d := types.Dict(map[string]types.Object{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name(ann.TypeString()),
		"Rect":    ann.Rect.Array(),
	})

	if pageIndRef != nil {
		d["P"] = *pageIndRef
	}

	if ann.Contents != "" {
		s, err := types.EscapedUTF16String(ann.Contents)
		if err != nil {
			return nil, err
		}
		d.InsertString("Contents", *s)
	}

	if ann.NM != "" {
		d.InsertString("NM", ann.NM)
	}

	modDate := types.DateString(time.Now())
	if ann.ModificationDate != "" {
		_, ok := types.DateTime(ann.ModificationDate, xRefTable.ValidationMode == ValidationRelaxed)
		if !ok {
			return nil, errors.Errorf("pdfcpu: annotation renderDict - validateDateEntry: <%s> invalid date", ann.ModificationDate)
		}
		modDate = ann.ModificationDate
	}
	d.InsertString("ModDate", modDate)

	if ann.F != 0 {
		d["F"] = types.Integer(ann.F)
	}

	if ann.C != nil {
		d["C"] = ann.C.Array()
	}

	if ann.BorderWidth > 0 {
		d["Border"] = borderArray(ann.BorderRadX, ann.BorderRadY, ann.BorderWidth)
	}

	return d, nil
}

// PopupAnnotation represents PDF Popup annotations.
type PopupAnnotation struct {
	Annotation
	ParentIndRef *types.IndirectRef // The optional parent markup annotation with which this pop-up annotation shall be associated.
	Open         bool               // A flag specifying whether the annotation shall initially be displayed open.
}

// NewPopupAnnotation returns a new popup annotation.
func NewPopupAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	borderRadX float64,
	borderRadY float64,
	borderWidth float64,

	parentIndRef *types.IndirectRef,
	displayOpen bool) PopupAnnotation {

	ann := NewAnnotation(AnnPopup, rect, contents, id, modDate, f, col, borderRadX, borderRadY, borderWidth)

	return PopupAnnotation{
		Annotation:   ann,
		ParentIndRef: parentIndRef,
		Open:         displayOpen,
	}
}

// ContentString returns a string representation of ann's content.
func (ann PopupAnnotation) ContentString() string {
	s := "\"" + ann.Contents + "\""
	if ann.ParentIndRef != nil {
		s = "-> #" + ann.ParentIndRef.ObjectNumber.String()
	}
	return s
}

func (ann PopupAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {
	d, err := ann.Annotation.RenderDict(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
	}

	if ann.ParentIndRef != nil {
		d["Parent"] = *ann.ParentIndRef
	}

	d["Open"] = types.Boolean(ann.Open)

	return d, nil
}

// LinkAnnotation represents a PDF link annotation.
type LinkAnnotation struct {
	Annotation
	Dest        *Destination     // internal link
	URI         string           // external link
	Quad        types.QuadPoints // shall be ignored if any coordinate lies outside the region specified by Rect.
	Border      bool             // render border using borderColor.
	BorderWidth float64
	BorderStyle BorderStyle
}

// NewLinkAnnotation returns a new link annotation.
func NewLinkAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	borderCol *color.SimpleColor,

	dest *Destination, // supply dest or uri, dest takes precedence
	uri string,
	quad types.QuadPoints,
	border bool,
	borderWidth float64,
	borderStyle BorderStyle) LinkAnnotation {

	ann := NewAnnotation(AnnLink, rect, contents, id, modDate, f, borderCol, 0, 0, 0)

	return LinkAnnotation{
		Annotation:  ann,
		Dest:        dest,
		URI:         uri,
		Quad:        quad,
		Border:      border,
		BorderWidth: borderWidth,
		BorderStyle: borderStyle,
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
func (ann LinkAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {
	d, err := ann.Annotation.RenderDict(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
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

	if ann.Quad != nil {
		d.Insert("QuadPoints", ann.Quad.Array())
	}

	if !ann.Border {
		d["Border"] = types.NewIntegerArray(0, 0, 0)
	} else {
		if ann.C != nil {
			d["C"] = ann.C.Array()
		}
	}

	d["BS"] = borderStyleDict(ann.BorderWidth, ann.BorderStyle)

	return d, nil
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
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	borderRadX float64,
	borderRadY float64,
	borderWidth float64,

	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string) MarkupAnnotation {

	ann := NewAnnotation(subType, rect, contents, id, modDate, f, col, borderRadX, borderRadY, borderWidth)

	return MarkupAnnotation{
		Annotation:   ann,
		T:            title,
		PopupIndRef:  popupIndRef,
		CA:           ca,
		RC:           rc,
		CreationDate: types.DateString(time.Now()),
		Subj:         subject}
}

// ContentString returns a string representation of ann's content.
func (ann MarkupAnnotation) ContentString() string {
	s := "\"" + ann.Contents + "\""
	if ann.PopupIndRef != nil {
		s += "-> #" + ann.PopupIndRef.ObjectNumber.String()
	}
	return s
}

func (ann MarkupAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {
	d, err := ann.Annotation.RenderDict(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
	}

	if ann.T != "" {
		s, err := types.EscapedUTF16String(ann.T)
		if err != nil {
			return nil, err
		}
		d.InsertString("T", *s)
	}

	if ann.PopupIndRef != nil {
		d.Insert("Popup", *ann.PopupIndRef)
	}

	if ann.CA != nil {
		d.Insert("CA", types.Float(*ann.CA))
	}

	if ann.RC != "" {
		s, err := types.EscapedUTF16String(ann.RC)
		if err != nil {
			return nil, err
		}
		d.InsertString("RC", *s)
	}

	d.InsertString("CreationDate", ann.CreationDate)

	if ann.Subj != "" {
		s, err := types.EscapedUTF16String(ann.Subj)
		if err != nil {
			return nil, err
		}
		d.InsertString("Subj", *s)
	}

	return d, nil
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
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,
	borderRadX float64,
	borderRadY float64,
	borderWidth float64,

	displayOpen bool,
	name string) TextAnnotation {

	ma := NewMarkupAnnotation(AnnText, rect, contents, id, modDate, f, col, borderRadX, borderRadY, borderWidth, title, popupIndRef, ca, rc, subject)

	return TextAnnotation{
		MarkupAnnotation: ma,
		Open:             displayOpen,
		Name:             name,
	}
}

// RenderDict renders ann into a PDF annotation dict.
func (ann TextAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {
	d, err := ann.MarkupAnnotation.RenderDict(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
	}

	d["Open"] = types.Boolean(ann.Open)

	if ann.Name != "" {
		d.InsertName("Name", ann.Name)
	}

	return d, nil
}

// FreeTextIntent represents the various free text annotation intents.
type FreeTextIntent int

const (
	IntentFreeText FreeTextIntent = 1 << iota
	IntentFreeTextCallout
	IntentFreeTextTypeWriter
)

func FreeTextIntentName(fti FreeTextIntent) string {
	var s string
	switch fti {
	case IntentFreeText:
		s = "FreeText"
	case IntentFreeTextCallout:
		s = "FreeTextCallout"
	case IntentFreeTextTypeWriter:
		s = "FreeTextTypeWriter"
	}
	return s
}

// FreeText Annotation displays text directly on the page.
type FreeTextAnnotation struct {
	MarkupAnnotation
	Text                   string             // Rich text string, see XFA 3.3
	HAlign                 types.HAlignment   // Code specifying the form of quadding (justification)
	FontName               string             // font name
	FontSize               int                // font size
	FontCol                *color.SimpleColor // font color
	DS                     string             // Default style string
	Intent                 string             // Description of the intent of the free text annotation
	CallOutLine            types.Array        // if intent is FreeTextCallout
	CallOutLineEndingStyle string
	Margins                types.Array
	BorderWidth            float64
	BorderStyle            BorderStyle
	CloudyBorder           bool
	CloudyBorderIntensity  int // 0,1,2
}

// XFA conform rich text string examples:
// The<b> second </b>and<span style="font-weight:bold"> fourth </span> words are bold.
// The<i> second </i>and<span style="font-style:italic"> fourth </span> words are italicized.
// For more information see <a href="http://www.example.com/">this</a> web site.

// NewFreeTextAnnotation returns a new free text annotation.
func NewFreeTextAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,

	text string,
	hAlign types.HAlignment,
	fontName string,
	fontSize int,
	fontCol *color.SimpleColor,
	ds string,
	intent *FreeTextIntent,
	callOutLine types.Array,
	callOutLineEndingStyle *LineEndingStyle,
	MLeft, MTop, MRight, MBot float64,
	borderWidth float64,
	borderStyle BorderStyle,
	cloudyBorder bool,
	cloudyBorderIntensity int) FreeTextAnnotation {

	// validate required DA, DS

	// validate callOutline: 2 or 3 points => array of 4 or 6 numbers.

	ma := NewMarkupAnnotation(AnnFreeText, rect, contents, id, modDate, f, col, 0, 0, 0, title, popupIndRef, ca, rc, subject)

	if cloudyBorderIntensity < 0 || cloudyBorderIntensity > 2 {
		cloudyBorderIntensity = 0
	}

	freeTextIntent := ""
	if intent != nil {
		freeTextIntent = FreeTextIntentName(*intent)
	}

	leStyle := ""
	if callOutLineEndingStyle != nil {
		leStyle = LineEndingStyleName(*callOutLineEndingStyle)
	}

	freeTextAnn := FreeTextAnnotation{
		MarkupAnnotation:       ma,
		Text:                   text,
		HAlign:                 hAlign,
		FontName:               fontName,
		FontSize:               fontSize,
		FontCol:                fontCol,
		DS:                     ds,
		Intent:                 freeTextIntent,
		CallOutLine:            callOutLine,
		CallOutLineEndingStyle: leStyle,
		BorderWidth:            borderWidth,
		BorderStyle:            borderStyle,
		CloudyBorder:           cloudyBorder,
		CloudyBorderIntensity:  cloudyBorderIntensity,
	}

	if MLeft > 0 || MTop > 0 || MRight > 0 || MBot > 0 {
		freeTextAnn.Margins = types.NewNumberArray(MLeft, MTop, MRight, MBot)
	}

	return freeTextAnn
}

// RenderDict renders ann into a PDF annotation dict.
func (ann FreeTextAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {
	d, err := ann.MarkupAnnotation.RenderDict(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
	}

	da := ""

	// TODO Implement Tf operator

	// fontID, err := xRefTable.EnsureFont(ann.FontName) // in root page Resources?
	// if err != nil {
	// 	return nil, err
	// }

	// da := fmt.Sprintf("/%s %d Tf", fontID, ann.FontSize)

	if ann.FontCol != nil {
		da += fmt.Sprintf(" %.2f %.2f %.2f rg", ann.FontCol.R, ann.FontCol.G, ann.FontCol.B)
	}
	d["DA"] = types.StringLiteral(da)

	d.InsertInt("Q", int(ann.HAlign))

	if ann.Text == "" {
		if ann.Contents == "" {
			return nil, errors.New("pdfcpu: FreeTextAnnotation missing \"text\"")
		}
		ann.Text = ann.Contents
	}
	s, err := types.EscapedUTF16String(ann.Text)
	if err != nil {
		return nil, err
	}
	d.InsertString("RC", *s)

	if ann.DS != "" {
		d.InsertString("DS", ann.DS)
	}

	if ann.Intent != "" {
		d.InsertName("IT", ann.Intent)
		if ann.Intent == "FreeTextCallout" {
			if len(ann.CallOutLine) > 0 {
				d["CL"] = ann.CallOutLine
				d.InsertName("LE", ann.CallOutLineEndingStyle)
			}
		}
	}

	if ann.Margins != nil {
		d["RD"] = ann.Margins
	}

	if ann.BorderWidth > 0 {
		d["BS"] = borderStyleDict(ann.BorderWidth, ann.BorderStyle)
	}

	if ann.CloudyBorder && ann.CloudyBorderIntensity > 0 {
		d["BE"] = borderEffectDict(ann.CloudyBorder, ann.CloudyBorderIntensity)
	}

	return d, nil
}

// LineIntent represents the various line annotation intents.
type LineIntent int

const (
	IntentLineArrow LineIntent = 1 << iota
	IntentLineDimension
)

func LineIntentName(li LineIntent) string {
	var s string
	switch li {
	case IntentLineArrow:
		s = "LineArrow"
	case IntentLineDimension:
		s = "LineDimension"
	}
	return s
}

// LineAnnotation represents a line annotation.
type LineAnnotation struct {
	MarkupAnnotation
	P1, P2                    types.Point // Two points in default user space.
	LineEndings               types.Array // Optional array of two names that shall specify the line ending styles.
	LeaderLineLength          float64     // Length of leader lines in default user space that extend from each endpoint of the line perpendicular to the line itself.
	LeaderLineOffset          float64     // Non-negative number that shall represent the length of the leader line offset, which is the amount of empty space between the endpoints of the annotation and the beginning of the leader lines.
	LeaderLineExtensionLength float64     // Non-negative number that shall represents the length of leader line extensions that extend from the line proper 180 degrees from the leader lines,
	Intent                    string      // Optional description of the intent of the line annotation.
	Measure                   types.Dict  // Optional measure dictionary that shall specify the scale and units that apply to the line annotation.
	Caption                   bool        // Use text specified by "Contents" or "RC" as caption.
	CaptionPositionTop        bool        // if true the caption shall be on top of the line else caption shall be centred inside the line.
	CaptionOffsetX            float64
	CaptionOffsetY            float64
	FillCol                   *color.SimpleColor
	BorderWidth               float64
	BorderStyle               BorderStyle
}

// NewLineAnnotation returns a new line annotation.
func NewLineAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,

	p1, p2 types.Point,
	beginLineEndingStyle *LineEndingStyle,
	endLineEndingStyle *LineEndingStyle,
	leaderLineLength float64,
	leaderLineOffset float64,
	leaderLineExtensionLength float64,
	intent *LineIntent,
	measure types.Dict,
	caption bool,
	captionPosTop bool,
	captionOffsetX float64,
	captionOffsetY float64,
	fillCol *color.SimpleColor,
	borderWidth float64,
	borderStyle BorderStyle) LineAnnotation {

	ma := NewMarkupAnnotation(AnnLine, rect, contents, id, modDate, f, col, 0, 0, 0, title, popupIndRef, ca, rc, subject)

	lineIntent := ""
	if intent != nil {
		lineIntent = LineIntentName(*intent)
	}

	lineAnn := LineAnnotation{
		MarkupAnnotation:          ma,
		P1:                        p1,
		P2:                        p2,
		LeaderLineLength:          leaderLineLength,
		LeaderLineOffset:          leaderLineOffset,
		LeaderLineExtensionLength: leaderLineExtensionLength,
		Intent:                    lineIntent,
		Measure:                   measure,
		Caption:                   caption,
		CaptionPositionTop:        captionPosTop,
		CaptionOffsetX:            captionOffsetX,
		CaptionOffsetY:            captionOffsetY,
		FillCol:                   fillCol,
		BorderWidth:               borderWidth,
		BorderStyle:               borderStyle,
	}

	if beginLineEndingStyle != nil && endLineEndingStyle != nil {
		lineAnn.LineEndings =
			types.NewNameArray(
				LineEndingStyleName(*beginLineEndingStyle),
				LineEndingStyleName(*endLineEndingStyle),
			)
	}

	return lineAnn
}

func (ann LineAnnotation) validateLeaderLineAttrs() error {
	if ann.LeaderLineExtensionLength < 0 {
		return errors.New("pdfcpu: LineAnnotation leader line extension length must not be negative.")
	}

	if ann.LeaderLineExtensionLength > 0 && ann.LeaderLineLength == 0 {
		return errors.New("pdfcpu: LineAnnotation leader line length missing.")
	}

	if ann.LeaderLineOffset < 0 {
		return errors.New("pdfcpu: LineAnnotation leader line offset must not be negative.")
	}

	return nil
}

// RenderDict renders ann into a PDF annotation dict.
func (ann LineAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {

	d, err := ann.MarkupAnnotation.RenderDict(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
	}

	if err := ann.validateLeaderLineAttrs(); err != nil {
		return nil, err
	}

	d["L"] = types.NewNumberArray(ann.P1.X, ann.P1.Y, ann.P2.X, ann.P2.Y)

	if ann.LeaderLineExtensionLength > 0 {
		d["LLE"] = types.Float(ann.LeaderLineExtensionLength)
	}

	if ann.LeaderLineLength > 0 {
		d["LL"] = types.Float(ann.LeaderLineLength)
		if ann.LeaderLineOffset > 0 {
			d["LLO"] = types.Float(ann.LeaderLineOffset)
		}
	}

	if len(ann.Measure) > 0 {
		d["Measure"] = ann.Measure
	}

	if ann.Intent != "" {
		d.InsertName("IT", ann.Intent)

	}

	d["Cap"] = types.Boolean(ann.Caption)
	if ann.Caption {
		if ann.CaptionPositionTop {
			d["CP"] = types.Name("Top")
		}
		d["CO"] = types.NewNumberArray(ann.CaptionOffsetX, ann.CaptionOffsetY)
	}

	if ann.FillCol != nil {
		d["IC"] = ann.FillCol.Array()
	}

	if ann.BorderWidth > 0 {
		d["BS"] = borderStyleDict(ann.BorderWidth, ann.BorderStyle)
	}

	if len(ann.LineEndings) == 2 {
		d["LE"] = ann.LineEndings
	}

	return d, nil
}

// SquareAnnotation represents a square annotation.
type SquareAnnotation struct {
	MarkupAnnotation
	FillCol               *color.SimpleColor
	Margins               types.Array
	BorderWidth           float64
	BorderStyle           BorderStyle
	CloudyBorder          bool
	CloudyBorderIntensity int // 0,1,2
}

// NewSquareAnnotation returns a new square annotation.
func NewSquareAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,

	fillCol *color.SimpleColor,
	MLeft, MTop, MRight, MBot float64,
	borderWidth float64,
	borderStyle BorderStyle,
	cloudyBorder bool,
	cloudyBorderIntensity int) SquareAnnotation {

	ma := NewMarkupAnnotation(AnnSquare, rect, contents, id, modDate, f, col, 0, 0, 0, title, popupIndRef, ca, rc, subject)

	if cloudyBorderIntensity < 0 || cloudyBorderIntensity > 2 {
		cloudyBorderIntensity = 0
	}

	squareAnn := SquareAnnotation{
		MarkupAnnotation:      ma,
		FillCol:               fillCol,
		BorderWidth:           borderWidth,
		BorderStyle:           borderStyle,
		CloudyBorder:          cloudyBorder,
		CloudyBorderIntensity: cloudyBorderIntensity,
	}

	if MLeft > 0 || MTop > 0 || MRight > 0 || MBot > 0 {
		squareAnn.Margins = types.NewNumberArray(MLeft, MTop, MRight, MBot)
	}

	return squareAnn
}

// RenderDict renders ann into a page annotation dict.
func (ann SquareAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {
	d, err := ann.MarkupAnnotation.RenderDict(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
	}

	if ann.FillCol != nil {
		d["IC"] = ann.FillCol.Array()
	}

	if ann.Margins != nil {
		d["RD"] = ann.Margins
	}

	if ann.BorderWidth > 0 {
		d["BS"] = borderStyleDict(ann.BorderWidth, ann.BorderStyle)
	}

	if ann.CloudyBorder && ann.CloudyBorderIntensity > 0 {
		d["BE"] = borderEffectDict(ann.CloudyBorder, ann.CloudyBorderIntensity)
	}

	return d, nil
}

// CircleAnnotation represents a square annotation.
type CircleAnnotation struct {
	MarkupAnnotation
	FillCol               *color.SimpleColor
	Margins               types.Array
	BorderWidth           float64
	BorderStyle           BorderStyle
	CloudyBorder          bool
	CloudyBorderIntensity int // 0,1,2
}

// NewCircleAnnotation returns a new circle annotation.
func NewCircleAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,

	fillCol *color.SimpleColor,
	MLeft, MTop, MRight, MBot float64,
	borderWidth float64,
	borderStyle BorderStyle,
	cloudyBorder bool,
	cloudyBorderIntensity int) CircleAnnotation {

	ma := NewMarkupAnnotation(AnnCircle, rect, contents, id, modDate, f, col, 0, 0, 0, title, popupIndRef, ca, rc, subject)

	if cloudyBorderIntensity < 0 || cloudyBorderIntensity > 2 {
		cloudyBorderIntensity = 0
	}

	circleAnn := CircleAnnotation{
		MarkupAnnotation:      ma,
		FillCol:               fillCol,
		BorderWidth:           borderWidth,
		BorderStyle:           borderStyle,
		CloudyBorder:          cloudyBorder,
		CloudyBorderIntensity: cloudyBorderIntensity,
	}

	if MLeft > 0 || MTop > 0 || MRight > 0 || MBot > 0 {
		circleAnn.Margins = types.NewNumberArray(MLeft, MTop, MRight, MBot)
	}

	return circleAnn
}

// RenderDict renders ann into a page annotation dict.
func (ann CircleAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {
	d, err := ann.MarkupAnnotation.RenderDict(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
	}

	if ann.FillCol != nil {
		d["IC"] = ann.FillCol.Array()
	}

	if ann.Margins != nil {
		d["RD"] = ann.Margins
	}

	if ann.BorderWidth > 0 {
		d["BS"] = borderStyleDict(ann.BorderWidth, ann.BorderStyle)
	}

	if ann.CloudyBorder && ann.CloudyBorderIntensity > 0 {
		d["BE"] = borderEffectDict(ann.CloudyBorder, ann.CloudyBorderIntensity)
	}

	return d, nil
}

// PolygonIntent represents the various polygon annotation intents.
type PolygonIntent int

const (
	IntentPolygonCloud PolygonIntent = 1 << iota
	IntentPolygonDimension
)

func PolygonIntentName(pi PolygonIntent) string {
	var s string
	switch pi {
	case IntentPolygonCloud:
		s = "PolygonCloud"
	case IntentPolygonDimension:
		s = "PolygonDimension"
	}
	return s
}

// PolygonAnnotation represents a polygon annotation.
type PolygonAnnotation struct {
	MarkupAnnotation
	Vertices              types.Array // Array of numbers specifying the alternating horizontal and vertical coordinates, respectively, of each vertex, in default user space.
	Path                  types.Array // Array of n arrays, each supplying the operands for a path building operator (m, l or c).
	Intent                string      // Optional description of the intent of the polygon annotation.
	Measure               types.Dict  // Optional measure dictionary that shall specify the scale and units that apply to the annotation.
	FillCol               *color.SimpleColor
	BorderWidth           float64
	BorderStyle           BorderStyle
	CloudyBorder          bool
	CloudyBorderIntensity int // 0,1,2
}

// NewPolygonAnnotation returns a new polygon annotation.
func NewPolygonAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,

	vertices types.Array,
	path types.Array,
	intent *PolygonIntent,
	measure types.Dict,
	fillCol *color.SimpleColor,
	borderWidth float64,
	borderStyle BorderStyle,
	cloudyBorder bool,
	cloudyBorderIntensity int) PolygonAnnotation {

	ma := NewMarkupAnnotation(AnnPolygon, rect, contents, id, modDate, f, col, 0, 0, 0, title, popupIndRef, ca, rc, subject)

	polygonIntent := ""
	if intent != nil {
		polygonIntent = PolygonIntentName(*intent)
	}

	if cloudyBorderIntensity < 0 || cloudyBorderIntensity > 2 {
		cloudyBorderIntensity = 0
	}

	polygonAnn := PolygonAnnotation{
		MarkupAnnotation:      ma,
		Vertices:              vertices,
		Path:                  path,
		Intent:                polygonIntent,
		Measure:               measure,
		FillCol:               fillCol,
		BorderWidth:           borderWidth,
		BorderStyle:           borderStyle,
		CloudyBorder:          cloudyBorder,
		CloudyBorderIntensity: cloudyBorderIntensity,
	}

	return polygonAnn
}

// RenderDict renders ann into a PDF annotation dict.
func (ann PolygonAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {

	d, err := ann.MarkupAnnotation.RenderDict(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
	}

	if len(ann.Measure) > 0 {
		d["Measure"] = ann.Measure
	}

	if len(ann.Vertices) > 0 && len(ann.Path) > 0 {
		return nil, errors.New("pdfcpu: PolygonAnnotation supports \"Vertices\" or \"Path\" only")
	}

	if len(ann.Vertices) > 0 {
		d["Vertices"] = ann.Vertices
	} else {
		d["Path"] = ann.Path
	}

	if ann.Intent != "" {
		d.InsertName("IT", ann.Intent)

	}

	if ann.FillCol != nil {
		d["IC"] = ann.FillCol.Array()
	}

	if ann.BorderWidth > 0 {
		d["BS"] = borderStyleDict(ann.BorderWidth, ann.BorderStyle)
	}

	if ann.CloudyBorder && ann.CloudyBorderIntensity > 0 {
		d["BE"] = borderEffectDict(ann.CloudyBorder, ann.CloudyBorderIntensity)
	}

	return d, nil
}

// PolyLineIntent represents the various polyline annotation intents.
type PolyLineIntent int

const (
	IntentPolyLinePolygonCloud PolyLineIntent = 1 << iota
	IntentPolyLineDimension
)

func PolyLineIntentName(pi PolyLineIntent) string {
	var s string
	switch pi {
	case IntentPolyLineDimension:
		s = "PolyLineDimension"
	}
	return s
}

type PolyLineAnnotation struct {
	MarkupAnnotation
	Vertices    types.Array // Array of numbers specifying the alternating horizontal and vertical coordinates, respectively, of each vertex, in default user space.
	Path        types.Array // Array of n arrays, each supplying the operands for a path building operator (m, l or c).
	Intent      string      // Optional description of the intent of the polyline annotation.
	Measure     types.Dict  // Optional measure dictionary that shall specify the scale and units that apply to the annotation.
	FillCol     *color.SimpleColor
	BorderWidth float64
	BorderStyle BorderStyle
	LineEndings types.Array // Optional array of two names that shall specify the line ending styles.
}

// NewPolyLineAnnotation returns a new polyline annotation.
func NewPolyLineAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,

	vertices types.Array,
	path types.Array,
	intent *PolyLineIntent,
	measure types.Dict,
	fillCol *color.SimpleColor,
	borderWidth float64,
	borderStyle BorderStyle,
	beginLineEndingStyle *LineEndingStyle,
	endLineEndingStyle *LineEndingStyle) PolyLineAnnotation {

	ma := NewMarkupAnnotation(AnnPolyLine, rect, contents, id, modDate, f, col, 0, 0, 0, title, popupIndRef, ca, rc, subject)

	polyLineIntent := ""
	if intent != nil {
		polyLineIntent = PolyLineIntentName(*intent)
	}

	polyLineAnn := PolyLineAnnotation{
		MarkupAnnotation: ma,
		Vertices:         vertices,
		Path:             path,
		Intent:           polyLineIntent,
		Measure:          measure,
		FillCol:          fillCol,
		BorderWidth:      borderWidth,
		BorderStyle:      borderStyle,
	}

	if beginLineEndingStyle != nil && endLineEndingStyle != nil {
		polyLineAnn.LineEndings =
			types.NewNameArray(
				LineEndingStyleName(*beginLineEndingStyle),
				LineEndingStyleName(*endLineEndingStyle),
			)
	}

	return polyLineAnn
}

// RenderDict renders ann into a PDF annotation dict.
func (ann PolyLineAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {

	d, err := ann.MarkupAnnotation.RenderDict(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
	}

	if len(ann.Measure) > 0 {
		d["Measure"] = ann.Measure
	}

	if len(ann.Vertices) > 0 && len(ann.Path) > 0 {
		return nil, errors.New("pdfcpu: PolyLineAnnotation supports \"Vertices\" or \"Path\" only")
	}

	if len(ann.Vertices) > 0 {
		d["Vertices"] = ann.Vertices
	} else {
		d["Path"] = ann.Path
	}

	if ann.Intent != "" {
		d.InsertName("IT", ann.Intent)

	}

	if ann.FillCol != nil {
		d["IC"] = ann.FillCol.Array()
	}

	if ann.BorderWidth > 0 {
		d["BS"] = borderStyleDict(ann.BorderWidth, ann.BorderStyle)
	}

	if len(ann.LineEndings) == 2 {
		d["LE"] = ann.LineEndings
	}

	return d, nil
}

type TextMarkupAnnotation struct {
	MarkupAnnotation
	Quad types.QuadPoints
}

func NewTextMarkupAnnotation(
	subType AnnotationType,
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	borderRadX float64,
	borderRadY float64,
	borderWidth float64,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,

	quad types.QuadPoints) TextMarkupAnnotation {

	ma := NewMarkupAnnotation(subType, rect, contents, id, modDate, f, col, borderRadX, borderRadY, borderWidth, title, popupIndRef, ca, rc, subject)

	return TextMarkupAnnotation{
		MarkupAnnotation: ma,
		Quad:             quad,
	}
}

func (ann TextMarkupAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {
	d, err := ann.MarkupAnnotation.RenderDict(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
	}

	if ann.Quad != nil {
		d.Insert("QuadPoints", ann.Quad.Array())
	}

	return d, nil
}

type HighlightAnnotation struct {
	TextMarkupAnnotation
}

func NewHighlightAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	borderRadX float64,
	borderRadY float64,
	borderWidth float64,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,

	quad types.QuadPoints) HighlightAnnotation {

	return HighlightAnnotation{
		NewTextMarkupAnnotation(AnnHighLight, rect, contents, id, modDate, f, col, borderRadX, borderRadY, borderWidth, title, popupIndRef, ca, rc, subject, quad),
	}
}

type UnderlineAnnotation struct {
	TextMarkupAnnotation
}

func NewUnderlineAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	borderRadX float64,
	borderRadY float64,
	borderWidth float64,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,

	quad types.QuadPoints) UnderlineAnnotation {

	return UnderlineAnnotation{
		NewTextMarkupAnnotation(AnnUnderline, rect, contents, id, modDate, f, col, borderRadX, borderRadY, borderWidth, title, popupIndRef, ca, rc, subject, quad),
	}
}

type SquigglyAnnotation struct {
	TextMarkupAnnotation
}

func NewSquigglyAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	borderRadX float64,
	borderRadY float64,
	borderWidth float64,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,

	quad types.QuadPoints) SquigglyAnnotation {

	return SquigglyAnnotation{
		NewTextMarkupAnnotation(AnnSquiggly, rect, contents, id, modDate, f, col, borderRadX, borderRadY, borderWidth, title, popupIndRef, ca, rc, subject, quad),
	}
}

type StrikeOutAnnotation struct {
	TextMarkupAnnotation
}

func NewStrikeOutAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	borderRadX float64,
	borderRadY float64,
	borderWidth float64,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,

	quad types.QuadPoints) StrikeOutAnnotation {

	return StrikeOutAnnotation{
		NewTextMarkupAnnotation(AnnStrikeOut, rect, contents, id, modDate, f, col, borderRadX, borderRadY, borderWidth, title, popupIndRef, ca, rc, subject, quad),
	}
}

type CaretAnnotation struct {
	MarkupAnnotation
	RD        *types.Rectangle // A set of four numbers that shall describe the numerical differences between two rectangles: the Rect entry of the annotation and the actual boundaries of the underlying caret.
	Paragraph bool             // A new paragraph symbol (¶) shall be associated with the caret.
}

func NewCaretAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	borderRadX float64,
	borderRadY float64,
	borderWidth float64,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,

	rd *types.Rectangle,
	paragraph bool) CaretAnnotation {

	ma := NewMarkupAnnotation(AnnCaret, rect, contents, id, modDate, f, col, borderRadX, borderRadY, borderWidth, title, popupIndRef, ca, rc, subject)

	return CaretAnnotation{
		MarkupAnnotation: ma,
		RD:               rd,
		Paragraph:        paragraph,
	}
}

func (ann CaretAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {
	d, err := ann.MarkupAnnotation.RenderDict(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
	}

	if ann.RD != nil {
		d["RD"] = ann.RD.Array()
	}

	if ann.Paragraph {
		d["Sy"] = types.Name("P")
	}

	return d, nil
}

// A series of alternating x and y coordinates in PDF user space, specifying points along the path.
type InkPath []float64

type InkAnnotation struct {
	MarkupAnnotation
	InkList     []InkPath // Array of n arrays, each representing a stroked path of points in user space.
	BorderWidth float64
	BorderStyle BorderStyle
}

func NewInkAnnotation(
	rect types.Rectangle,
	contents, id string,
	modDate string,
	f AnnotationFlags,
	col *color.SimpleColor,
	title string,
	popupIndRef *types.IndirectRef,
	ca *float64,
	rc, subject string,

	ink []InkPath,
	borderWidth float64,
	borderStyle BorderStyle) InkAnnotation {

	ma := NewMarkupAnnotation(AnnInk, rect, contents, id, modDate, f, col, 0, 0, 0, title, popupIndRef, ca, rc, subject)

	return InkAnnotation{
		MarkupAnnotation: ma,
		InkList:          ink,
		BorderWidth:      borderWidth,
		BorderStyle:      borderStyle,
	}
}

func (ann InkAnnotation) RenderDict(xRefTable *XRefTable, pageIndRef *types.IndirectRef) (types.Dict, error) {
	d, err := ann.MarkupAnnotation.RenderDict(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
	}

	ink := types.Array{}
	for i := range ann.InkList {
		ink = append(ink, types.NewNumberArray(ann.InkList[i]...))
	}
	d["InkList"] = ink

	if ann.BorderWidth > 0 {
		d["BS"] = borderStyleDict(ann.BorderWidth, ann.BorderStyle)
	}

	return d, nil
}
