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
	"path/filepath"
	"time"

	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Functions needed to create a test.pdf that gets used for validation testing (see process_test.go)

func createTextAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	d := types.Dict(map[string]types.Object{
		"Type":     types.Name("Annot"),
		"Subtype":  types.Name("Text"),
		"Contents": types.StringLiteral("Text Annotation"),
		"Rect":     annotRect,
		"P":        pageIndRef,
		//"NM": "",
		//"Border":   NewIntegerArray(0, 0, 5),
		//"C":        NewNumberArray(1, 0, 0),
		//"Name":     Name("Note"),
	})

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	usageDict := types.Dict(
		map[string]types.Object{
			"CreatorInfo": types.Dict(
				map[string]types.Object{
					"Creator": types.StringLiteral("pdfcpu"),
					"Subtype": types.Name("Technical"),
				},
			),
			"Language": types.Dict(
				map[string]types.Object{
					"Lang":      types.StringLiteral("en-us"),
					"Preferred": types.Name("ON"),
				},
			),
			"Export": types.Dict(
				map[string]types.Object{
					"ExportState": types.Name("ON"),
				},
			),
			"Zoom": types.Dict(
				map[string]types.Object{
					"min": types.Float(0),
				},
			),
			"Print": types.Dict(
				map[string]types.Object{
					"Subtype":    types.Name("Watermark"),
					"PrintState": types.Name("ON"),
				},
			),
			"View": types.Dict(
				map[string]types.Object{
					"ViewState": types.Name("Ind"),
				},
			),
			"User": types.Dict(
				map[string]types.Object{
					"Type": types.Name("ON"),
					"Name": types.StringLiteral("Horst Rutter"),
				},
			),
			"PageElement": types.Dict(
				map[string]types.Object{
					"Subtype": types.Name("FG"),
				},
			),
		},
	)

	optionalContentGroupDict := types.Dict(
		map[string]types.Object{
			"Type":   types.Name("OCG"),
			"Name":   types.StringLiteral("OCG"),
			"Intent": types.Name("Design"),
			"Usage":  usageDict,
		},
	)

	uriActionDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("Action"),
			"S":    types.Name("URI"),
			"URI":  types.StringLiteral("https://golang.org"),
		},
	)

	indRef, err := xRefTable.IndRefForNewObject(uriActionDict)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Link"),
			"Contents": types.StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 5),
			"C":        types.NewNumberArray(0, 0, 1),
			"A":        *indRef,
			"H":        types.Name("I"),
			"PA":       *indRef,
			"OC":       optionalContentGroupDict,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createFreeTextAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("FreeText"),
			"Contents": types.StringLiteral("FreeText Annotation"),
			"F":        types.Integer(128), // Lock
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 1, 0),
			"DA":       types.StringLiteral("DA"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLineAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Line"),
			"Contents": types.StringLiteral("Line Annotation"),
			"F":        types.Integer(128), // Lock
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 1, 0),
			"L":        annotRect,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createSquareAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Square"),
			"Contents": types.StringLiteral("Square Annotation"),
			"F":        types.Integer(128), // Lock
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, .3, .3),
			"IC":       types.NewNumberArray(0.8, .8, .8),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createCircleAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Circle"),
			"Contents": types.StringLiteral("Circle Annotation"),
			"F":        types.Integer(128), // Lock
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 10),
			"C":        types.NewNumberArray(0.5, 0, 5, 0),
			"IC":       types.NewNumberArray(0.8, .8, .8),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createPolygonAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	// Construct a polyline using the annot rects both lower corners and the upper right corner.
	v := types.Array{nil, nil, nil, nil}
	copy(v, annotRect)
	v = append(v, annotRect[2])
	v = append(v, annotRect[1])

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Polygon"),
			"Contents": types.StringLiteral("Polygon Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 1, 0),
			"Vertices": v,
			"IC":       types.NewNumberArray(0.3, 0.5, 0.0),
			"BS": types.Dict(
				map[string]types.Object{
					"Type": types.Name("Border"),
					"W":    types.Float(0.5),
					"S":    types.Name("D"),
				},
			),
			"BE": types.Dict(
				map[string]types.Object{
					"S": types.Name("C"),
					"I": types.Float(1),
				},
			),
			"IT": types.Name("PolygonCloud"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createPolyLineAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	// Construct a polyline using the annot rects both lower corners and the upper right corner.
	v := types.Array{nil, nil, nil, nil}
	copy(v, annotRect)
	v = append(v, annotRect[2])
	v = append(v, annotRect[1])

	optionalContentGroupDict := types.Dict(
		map[string]types.Object{
			"Type":   types.Name("OCG"),
			"Name":   types.StringLiteral("OCG"),
			"Intent": types.NewNameArray("Design", "View"),
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("PolyLine"),
			"Contents": types.StringLiteral("PolyLine Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 1, 0),
			"Vertices": v,
			"OC":       optionalContentGroupDict,
			"IC":       types.NewNumberArray(0.3, 0.5, 0.0),
			"BS": types.Dict(
				map[string]types.Object{
					"Type": types.Name("Border"),
					"W":    types.Float(0.5),
					"S":    types.Name("D"),
				},
			),
			"IT": types.Name("PolygonCloud"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createHighlightAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	// Create a quad points array corresponding to the annot rect.
	ar := annotRect

	qp := types.Array{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	optionalContentGroupDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("OCG"),
			"Name": types.StringLiteral("OCG"),
		},
	)

	optionalContentMembershipDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("OCMD"),
			"OCGs": types.Array{nil, optionalContentGroupDict},
			"P":    types.Name("AllOn"),
			"VE":   types.Array{},
		},
	)

	_ = optionalContentMembershipDict

	d := types.Dict(
		map[string]types.Object{
			"Type":       types.Name("Annot"),
			"Subtype":    types.Name("Highlight"),
			"Contents":   types.StringLiteral("Highlight Annotation"),
			"Rect":       annotRect,
			"P":          pageIndRef,
			"Border":     types.NewIntegerArray(0, 0, 1),
			"C":          types.NewNumberArray(.2, 0, 0),
			"OC":         optionalContentMembershipDict,
			"QuadPoints": qp,
			"T":          types.StringLiteral("MyTitle"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createUnderlineAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	ar := annotRect

	qp := types.Array{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := types.Dict(
		map[string]types.Object{
			"Type":       types.Name("Annot"),
			"Subtype":    types.Name("Underline"),
			"Contents":   types.StringLiteral("Underline Annotation"),
			"Rect":       annotRect,
			"P":          pageIndRef,
			"Border":     types.NewIntegerArray(0, 0, 1),
			"C":          types.NewNumberArray(.5, 0, 0),
			"QuadPoints": qp,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createSquigglyAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	ar := annotRect

	qp := types.Array{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := types.Dict(
		map[string]types.Object{
			"Type":       types.Name("Annot"),
			"Subtype":    types.Name("Squiggly"),
			"Contents":   types.StringLiteral("Squiggly Annotation"),
			"Rect":       annotRect,
			"P":          pageIndRef,
			"Border":     types.NewIntegerArray(0, 0, 1),
			"C":          types.NewNumberArray(.5, 0, 0),
			"QuadPoints": qp,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createStrikeOutAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	ar := annotRect

	qp := types.Array{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := types.Dict(
		map[string]types.Object{
			"Type":       types.Name("Annot"),
			"Subtype":    types.Name("StrikeOut"),
			"Contents":   types.StringLiteral("StrikeOut Annotation"),
			"Rect":       annotRect,
			"P":          pageIndRef,
			"Border":     types.NewIntegerArray(0, 0, 1),
			"C":          types.NewNumberArray(.5, 0, 0),
			"QuadPoints": qp,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createCaretAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Caret"),
			"Contents": types.StringLiteral("Caret Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0.5, 0.5, 0),
			"RD":       types.NewNumberArray(0, 0, 0, 0),
			"Sy":       types.Name("None"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createStampAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Stamp"),
			"Contents": types.StringLiteral("Stamp Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0.5, 0.5, 0.9),
			"Name":     types.Name("Approved"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createInkAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	ar := annotRect

	l := types.Array{
		types.Array{ar[0], ar[1], ar[2], ar[1]},
		types.Array{ar[2], ar[1], ar[2], ar[3]},
		types.Array{ar[2], ar[3], ar[0], ar[3]},
		types.Array{ar[0], ar[3], ar[0], ar[1]},
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Ink"),
			"Contents": types.StringLiteral("Ink Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0.5, 0, 0.3),
			"InkList":  l,
			"ExData": types.Dict(
				map[string]types.Object{
					"Type":    types.Name("ExData"),
					"Subtype": types.Name("Markup3D"),
				},
			),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createPopupAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Popup"),
			"Contents": types.StringLiteral("Ink Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0.5, 0, 0.3),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createFileAttachmentAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	// macOS starts up iTunes for audio file attachments.

	fileName := testAudioFileWAV

	ir, err := xRefTable.NewEmbeddedFileStreamDict(fileName)
	if err != nil {
		return nil, err
	}

	fn := filepath.Base(fileName)

	s, err := types.EscapeUTF16String(fn)
	if err != nil {
		return nil, err
	}

	fileSpecDict, err := xRefTable.NewFileSpecDict(fn, *s, "attached by pdfcpu", *ir)
	if err != nil {
		return nil, err
	}

	ir, err = xRefTable.IndRefForNewObject(fileSpecDict)
	if err != nil {
		return nil, err
	}

	now := types.StringLiteral(types.DateString(time.Now()))

	d := types.Dict(
		map[string]types.Object{
			"Type":         types.Name("Annot"),
			"Subtype":      types.Name("FileAttachment"),
			"Contents":     types.StringLiteral("FileAttachment Annotation"),
			"Rect":         annotRect,
			"P":            pageIndRef,
			"M":            now,
			"F":            types.Integer(0),
			"Border":       types.NewIntegerArray(0, 0, 1),
			"C":            types.NewNumberArray(0.5, 0.0, 0.5),
			"CA":           types.Float(0.95),
			"CreationDate": now,
			"Name":         types.Name("Paperclip"),
			"FS":           *ir,
			"NM":           types.StringLiteral("SoundFileAttachmentAnnot"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createFileSpecDict(xRefTable *model.XRefTable, fileName string) (types.Dict, error) {
	ir, err := xRefTable.NewEmbeddedFileStreamDict(fileName)
	if err != nil {
		return nil, err
	}
	fn := filepath.Base(fileName)

	s, err := types.EscapeUTF16String(fn)
	if err != nil {
		return nil, err
	}

	return xRefTable.NewFileSpecDict(fn, *s, "attached by pdfcpu", *ir)
}

func createSoundObject(xRefTable *model.XRefTable) (*types.IndirectRef, error) {
	fileName := testAudioFileWAV
	fileSpecDict, err := createFileSpecDict(xRefTable, fileName)
	if err != nil {
		return nil, err
	}
	return xRefTable.NewSoundStreamDict(fileName, 44100, fileSpecDict)
}

func createSoundAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	indRef, err := createSoundObject(xRefTable)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Sound"),
			"Contents": types.StringLiteral("Sound Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0.5, 0.5),
			"Sound":    *indRef,
			"Name":     types.Name("Speaker"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createMovieDict(xRefTable *model.XRefTable) (*types.IndirectRef, error) {

	// not supported: mp3,mp4,m4a

	fileSpecDict, err := createFileSpecDict(xRefTable, testAudioFileWAV)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"F":      fileSpecDict,
			"Aspect": types.NewIntegerArray(200, 200),
			"Rotate": types.Integer(0),
			"Poster": types.Boolean(true),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createMovieAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	indRef, err := createMovieDict(xRefTable)
	if err != nil {
		return nil, err
	}

	movieActivationDict := types.Dict(
		map[string]types.Object{
			"Start":        types.Integer(10),
			"Duration":     types.Integer(60),
			"Rate":         types.Float(1.0),
			"Volume":       types.Float(1.0),
			"ShowControls": types.Boolean(true),
			"Mode":         types.Name("Once"),
			"Synchronous":  types.Boolean(false),
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Movie"),
			"Contents": types.StringLiteral("Movie Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 3), // rounded corners don't work
			"C":        types.NewNumberArray(0.3, 0.5, 0.5),
			"Movie":    *indRef,
			"T":        types.StringLiteral("Sample Movie"),
			"A":        movieActivationDict,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createMediaRenditionAction(xRefTable *model.XRefTable, mediaClipDataDict *types.IndirectRef) types.Dict {

	r := createMediaRendition(xRefTable, mediaClipDataDict)

	return types.Dict(
		map[string]types.Object{
			"Type": types.Name("Action"),
			"S":    types.Name("Rendition"),
			"R":    *r,               // rendition object
			"OP":   types.Integer(0), // Play
		},
	)
}

func createSelectorRenditionAction(mediaClipDataDict *types.IndirectRef) types.Dict {

	r := createSelectorRendition(mediaClipDataDict)

	return types.Dict(
		map[string]types.Object{
			"Type": types.Name("Action"),
			"S":    types.Name("Rendition"),
			"R":    *r,               // rendition object
			"OP":   types.Integer(0), // Play
		},
	)
}

func createScreenAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	ir, err := createMediaClipDataDict(xRefTable)
	if err != nil {
		return nil, err
	}

	mediaRenditionAction := createMediaRenditionAction(xRefTable, ir)

	selectorRenditionAction := createSelectorRenditionAction(ir)

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Screen"),
			"Contents": types.StringLiteral("Screen Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 3),
			"C":        types.NewNumberArray(0.2, 0.8, 0.5),
			"A":        mediaRenditionAction,
			"AA": types.Dict(
				map[string]types.Object{
					"D": selectorRenditionAction,
				},
			),
		},
	)

	ir, err = xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	// Inject indRef of screen annotation into action dicts.
	mediaRenditionAction.Insert("AN", *ir)
	selectorRenditionAction.Insert("AN", *ir)

	return ir, nil
}

func createWidgetAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	appearanceCharacteristicsDict := types.Dict(
		map[string]types.Object{
			"R":  types.Integer(0),
			"BC": types.NewNumberArray(0.0, 0.0, 0.0),
			"BG": types.NewNumberArray(0.5, 0.0, 0.5),
			"RC": types.StringLiteral("Rollover caption"),
			"IF": types.Dict(
				map[string]types.Object{
					"SW": types.Name("A"),
					"S":  types.Name("A"),
					"FB": types.Boolean(true),
				},
			),
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Widget"),
			"Contents": types.StringLiteral("Widget Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 3),
			"C":        types.NewNumberArray(0.5, 0.5, 0.5),
			"MK":       appearanceCharacteristicsDict,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createXObjectForPrinterMark(xRefTable *model.XRefTable) (*types.IndirectRef, error) {
	buf := `0 0 m 0 25 l 25 25 l 25 0 l s`
	sd, _ := xRefTable.NewStreamDictForBuf([]byte(buf))
	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, 25, 25))
	sd.Insert("Matrix", types.NewIntegerArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createPrinterMarkAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	ir, err := createXObjectForPrinterMark(xRefTable)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("PrinterMark"),
			"Contents": types.StringLiteral("PrinterMark Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 3),
			"C":        types.NewNumberArray(0.2, 0.8, 0.5),
			"F":        types.Integer(0),
			"AP": types.Dict(
				map[string]types.Object{
					"N": *ir,
				},
			),
			"MN": types.Name("ColorBar"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createXObjectForWaterMark(xRefTable *model.XRefTable) (*types.IndirectRef, error) {
	fIndRef, err := pdffont.EnsureFontDict(xRefTable, "Helvetica", "", "", false, false, nil)
	if err != nil {
		return nil, err
	}

	fResDict := types.NewDict()
	fResDict.Insert("F1", *fIndRef)
	resourceDict := types.NewDict()
	resourceDict.Insert("Font", fResDict)

	buf := `0 0 m 0 200 l 200 200 l 200 0 l s BT /F1 48 Tf 0.7 0.7 -0.7 0.7 30 10 Tm 1 Tr 2 w (Watermark) Tj ET`
	sd, _ := xRefTable.NewStreamDictForBuf([]byte(buf))
	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, 200, 200))
	sd.Insert("Matrix", types.NewIntegerArray(1, 0, 0, 1, 0, 0))
	sd.Insert("Resources", resourceDict)

	if err = sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createWaterMarkAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	ir, err := createXObjectForWaterMark(xRefTable)
	if err != nil {
		return nil, err
	}

	d1 := types.Dict(
		map[string]types.Object{
			"Type":   types.Name("FixedPrint"),
			"Matrix": types.NewIntegerArray(1, 0, 0, 1, 72, -72),
			"H":      types.Float(0),
			"V":      types.Float(0),
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Watermark"),
			"Contents": types.StringLiteral("Watermark Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 3),
			"C":        types.NewNumberArray(0.2, 0.8, 0.5),
			"F":        types.Integer(0),
			"AP": types.Dict(
				map[string]types.Object{
					"N": *ir,
				},
			),
			"FixedPrint": d1,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func create3DAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("3D"),
			"Contents": types.StringLiteral("3D Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 3),
			"C":        types.NewNumberArray(0.2, 0.8, 0.5),
			"F":        types.Integer(0),
			"3DD":      types.NewDict(), // stream or 3D reference dict
			"3DV":      types.Name("F"),
			"3DA":      types.NewDict(), // activation dict
			"3DI":      types.Boolean(true),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createRedactAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	qp := types.Array{}
	qp = append(qp, annotRect[0])
	qp = append(qp, annotRect[1])
	qp = append(qp, annotRect[2])
	qp = append(qp, annotRect[1])
	qp = append(qp, annotRect[2])
	qp = append(qp, annotRect[3])
	qp = append(qp, annotRect[0])
	qp = append(qp, annotRect[3])

	d := types.Dict(
		map[string]types.Object{
			"Type":        types.Name("Annot"),
			"Subtype":     types.Name("Redact"),
			"Contents":    types.StringLiteral("Redact Annotation"),
			"Rect":        annotRect,
			"P":           pageIndRef,
			"Border":      types.NewIntegerArray(0, 0, 3),
			"C":           types.NewNumberArray(0.2, 0.8, 0.5),
			"F":           types.Integer(0),
			"QuadPoints":  qp,
			"IC":          types.NewNumberArray(0.5, 0.0, 0.9),
			"OverlayText": types.StringLiteral("An overlay"),
			"Repeat":      types.Boolean(true),
			"DA":          types.StringLiteral("x"),
			"Q":           types.Integer(1),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createRemoteGoToAction(xRefTable *model.XRefTable) (*types.IndirectRef, error) {

	d := types.Dict(
		map[string]types.Object{
			"Type":      types.Name("Action"),
			"S":         types.Name("GoToR"),
			"F":         types.StringLiteral("./go.pdf"),
			"D":         types.Array{types.Integer(0), types.Name("Fit")},
			"NewWindow": types.Boolean(true),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationWithRemoteGoToAction(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	ir, err := createRemoteGoToAction(xRefTable)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Link"),
			"Contents": types.StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A":        *ir,
			"H":        types.Name("I"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createEmbeddedGoToAction(xRefTable *model.XRefTable) (*types.IndirectRef, error) {

	f := filepath.Join(testDir, "go.pdf")
	fileSpecDict, err := createFileSpecDict(xRefTable, f)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":      types.Name("Action"),
			"S":         types.Name("GoToE"),
			"F":         fileSpecDict,
			"D":         types.Array{types.Integer(0), types.Name("Fit")},
			"NewWindow": types.Boolean(true), // not honoured by Acrobat Reader.
			"T": types.Dict(
				map[string]types.Object{
					"R": types.Name("C"),
					"N": types.StringLiteral(filepath.Base(f)),
				},
			),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationWithEmbeddedGoToAction(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	ir, err := createEmbeddedGoToAction(xRefTable)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Link"),
			"Contents": types.StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A":        *ir,
			"H":        types.Name("I"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithLaunchAction(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Link"),
			"Contents": types.StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A": types.Dict(
				map[string]types.Object{
					"Type": types.Name("Action"),
					"S":    types.Name("Launch"),
					"F":    types.StringLiteral("golang.pdf"),
					"Win": types.Dict(
						map[string]types.Object{
							"F": types.StringLiteral("golang.pdf"),
							"O": types.StringLiteral("O"),
						},
					),
					"NewWindow": types.Boolean(true),
				},
			),
			"H": types.Name("I"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithThreadAction(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Link"),
			"Contents": types.StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A": types.Dict(
				map[string]types.Object{
					"Type": types.Name("Action"),
					"S":    types.Name("Thread"),
					"D":    types.Integer(0), // jump to first article thread
					"B":    types.Integer(0), // jump to first bead
				},
			),
			"H": types.Name("I"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithSoundAction(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	ir, err := createSoundObject(xRefTable)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Link"),
			"Contents": types.StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A": types.Dict(
				map[string]types.Object{
					"Type":        types.Name("Action"),
					"S":           types.Name("Sound"),
					"Sound":       *ir,
					"Synchronous": types.Boolean(false),
					"Repeat":      types.Boolean(false),
					"Mix":         types.Boolean(false),
				},
			),
			"H": types.Name("I"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithMovieAction(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Link"),
			"Contents": types.StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A": types.Dict(
				map[string]types.Object{
					"Type":      types.Name("Action"),
					"S":         types.Name("Movie"),
					"T":         types.StringLiteral("Sample Movie"),
					"Operation": types.Name("Play"),
				},
			),
			"H": types.Name("I"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithHideAction(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	hideActionDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("Action"),
			"S":    types.Name("Hide"),
			"H":    types.Boolean(true),
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Link"),
			"Contents": types.StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A":        hideActionDict,
			"H":        types.Name("I"),
		},
	)

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	// We hide the link annotation itself.
	hideActionDict.Insert("T", *ir)

	return ir, nil
}

func createTrapNetAnnotation(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, annotRect types.Array) (*types.IndirectRef, error) {

	ir, err := pdffont.EnsureFontDict(xRefTable, "Helvetica", "", "", false, false, nil)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":         types.Name("Annot"),
			"Subtype":      types.Name("TrapNet"),
			"Contents":     types.StringLiteral("TrapNet Annotation"),
			"Rect":         annotRect,
			"P":            pageIndRef,
			"Border":       types.NewIntegerArray(0, 0, 3),
			"C":            types.NewNumberArray(0.2, 0.8, 0.5),
			"F":            types.Integer(0),
			"LastModified": types.StringLiteral(types.DateString(time.Now())),
			"FontFauxing":  types.Array{*ir},
		},
	)

	return xRefTable.IndRefForNewObject(d)
}
