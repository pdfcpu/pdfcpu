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
)

// Functions needed to create a test.pdf that gets used for validation testing (see process_test.go)

func createTextAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	d := Dict(map[string]Object{
		"Type":     Name("Annot"),
		"Subtype":  Name("Text"),
		"Contents": StringLiteral("Text Annotation"),
		"Rect":     annotRect,
		"P":        pageIndRef,
		//"NM": "",
		//"Border":   NewIntegerArray(0, 0, 5),
		//"C":        NewNumberArray(1, 0, 0),
		//"Name":     Name("Note"),
	})

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	usageDict := Dict(
		map[string]Object{
			"CreatorInfo": Dict(
				map[string]Object{
					"Creator": StringLiteral("pdfcpu"),
					"Subtype": Name("Technical"),
				},
			),
			"Language": Dict(
				map[string]Object{
					"Lang":      StringLiteral("en-us"),
					"Preferred": Name("ON"),
				},
			),
			"Export": Dict(
				map[string]Object{
					"ExportState": Name("ON"),
				},
			),
			"Zoom": Dict(
				map[string]Object{
					"min": Float(0),
				},
			),
			"Print": Dict(
				map[string]Object{
					"Subtype":    Name("Watermark"),
					"PrintState": Name("ON"),
				},
			),
			"View": Dict(
				map[string]Object{
					"ViewState": Name("Ind"),
				},
			),
			"User": Dict(
				map[string]Object{
					"Type": Name("ON"),
					"Name": StringLiteral("Horst Rutter"),
				},
			),
			"PageElement": Dict(
				map[string]Object{
					"Subtype": Name("FG"),
				},
			),
		},
	)

	optionalContentGroupDict := Dict(
		map[string]Object{
			"Type":   Name("OCG"),
			"Name":   StringLiteral("OCG"),
			"Intent": Name("Design"),
			"Usage":  usageDict,
		},
	)

	uriActionDict := Dict(
		map[string]Object{
			"Type": Name("Action"),
			"S":    Name("URI"),
			"URI":  StringLiteral("https://golang.org"),
		},
	)

	indRef, err := xRefTable.IndRefForNewObject(uriActionDict)
	if err != nil {
		return nil, err
	}

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Link"),
			"Contents": StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 5),
			"C":        NewNumberArray(0, 0, 1),
			"A":        *indRef,
			"H":        Name("I"),
			"PA":       *indRef,
			"OC":       optionalContentGroupDict,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createFreeTextAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("FreeText"),
			"Contents": StringLiteral("FreeText Annotation"),
			"F":        Integer(128), // Lock
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 1, 0),
			"DA":       StringLiteral("DA"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLineAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Line"),
			"Contents": StringLiteral("Line Annotation"),
			"F":        Integer(128), // Lock
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 1, 0),
			"L":        annotRect,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createSquareAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Square"),
			"Contents": StringLiteral("Square Annotation"),
			"F":        Integer(128), // Lock
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, .3, .3),
			"IC":       NewNumberArray(0.8, .8, .8),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createCircleAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Circle"),
			"Contents": StringLiteral("Circle Annotation"),
			"F":        Integer(128), // Lock
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 10),
			"C":        NewNumberArray(0.5, 0, 5, 0),
			"IC":       NewNumberArray(0.8, .8, .8),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createPolygonAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	// Construct a polyline using the annot rects both lower corners and the upper right corner.
	v := Array{nil, nil, nil, nil}
	copy(v, annotRect)
	v = append(v, annotRect[2])
	v = append(v, annotRect[1])

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Polygon"),
			"Contents": StringLiteral("Polygon Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 1, 0),
			"Vertices": v,
			"IC":       NewNumberArray(0.3, 0.5, 0.0),
			"BS": Dict(
				map[string]Object{
					"Type": Name("Border"),
					"W":    Float(0.5),
					"S":    Name("D"),
				},
			),
			"BE": Dict(
				map[string]Object{
					"S": Name("C"),
					"I": Float(1),
				},
			),
			"IT": Name("PolygonCloud"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createPolyLineAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	// Construct a polyline using the annot rects both lower corners and the upper right corner.
	v := Array{nil, nil, nil, nil}
	copy(v, annotRect)
	v = append(v, annotRect[2])
	v = append(v, annotRect[1])

	optionalContentGroupDict := Dict(
		map[string]Object{
			"Type":   Name("OCG"),
			"Name":   StringLiteral("OCG"),
			"Intent": NewNameArray("Design", "View"),
		},
	)

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("PolyLine"),
			"Contents": StringLiteral("PolyLine Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 1, 0),
			"Vertices": v,
			"OC":       optionalContentGroupDict,
			"IC":       NewNumberArray(0.3, 0.5, 0.0),
			"BS": Dict(
				map[string]Object{
					"Type": Name("Border"),
					"W":    Float(0.5),
					"S":    Name("D"),
				},
			),
			"IT": Name("PolygonCloud"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createHighlightAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	// Create a quad points array corresponding to the annot rect.
	ar := annotRect

	qp := Array{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	optionalContentGroupDict := Dict(
		map[string]Object{
			"Type": Name("OCG"),
			"Name": StringLiteral("OCG"),
		},
	)

	optionalContentMembershipDict := Dict(
		map[string]Object{
			"Type": Name("OCMD"),
			"OCGs": Array{nil, optionalContentGroupDict},
			"P":    Name("AllOn"),
			"VE":   Array{},
		},
	)

	_ = optionalContentMembershipDict

	d := Dict(
		map[string]Object{
			"Type":       Name("Annot"),
			"Subtype":    Name("Highlight"),
			"Contents":   StringLiteral("Highlight Annotation"),
			"Rect":       annotRect,
			"P":          pageIndRef,
			"Border":     NewIntegerArray(0, 0, 1),
			"C":          NewNumberArray(.2, 0, 0),
			"OC":         optionalContentMembershipDict,
			"QuadPoints": qp,
			"T":          StringLiteral("MyTitle"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createUnderlineAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	ar := annotRect

	qp := Array{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := Dict(
		map[string]Object{
			"Type":       Name("Annot"),
			"Subtype":    Name("Underline"),
			"Contents":   StringLiteral("Underline Annotation"),
			"Rect":       annotRect,
			"P":          pageIndRef,
			"Border":     NewIntegerArray(0, 0, 1),
			"C":          NewNumberArray(.5, 0, 0),
			"QuadPoints": qp,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createSquigglyAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	ar := annotRect

	qp := Array{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := Dict(
		map[string]Object{
			"Type":       Name("Annot"),
			"Subtype":    Name("Squiggly"),
			"Contents":   StringLiteral("Squiggly Annotation"),
			"Rect":       annotRect,
			"P":          pageIndRef,
			"Border":     NewIntegerArray(0, 0, 1),
			"C":          NewNumberArray(.5, 0, 0),
			"QuadPoints": qp,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createStrikeOutAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	ar := annotRect

	qp := Array{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := Dict(
		map[string]Object{
			"Type":       Name("Annot"),
			"Subtype":    Name("StrikeOut"),
			"Contents":   StringLiteral("StrikeOut Annotation"),
			"Rect":       annotRect,
			"P":          pageIndRef,
			"Border":     NewIntegerArray(0, 0, 1),
			"C":          NewNumberArray(.5, 0, 0),
			"QuadPoints": qp,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createCaretAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Caret"),
			"Contents": StringLiteral("Caret Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0.5, 0.5, 0),
			"RD":       NewNumberArray(0, 0, 0, 0),
			"Sy":       Name("None"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createStampAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Stamp"),
			"Contents": StringLiteral("Stamp Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0.5, 0.5, 0.9),
			"Name":     Name("Approved"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createInkAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	ar := annotRect

	l := Array{
		Array{ar[0], ar[1], ar[2], ar[1]},
		Array{ar[2], ar[1], ar[2], ar[3]},
		Array{ar[2], ar[3], ar[0], ar[3]},
		Array{ar[0], ar[3], ar[0], ar[1]},
	}

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Ink"),
			"Contents": StringLiteral("Ink Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0.5, 0, 0.3),
			"InkList":  l,
			"ExData": Dict(
				map[string]Object{
					"Type":    Name("ExData"),
					"Subtype": Name("Markup3D"),
				},
			),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createPopupAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Popup"),
			"Contents": StringLiteral("Ink Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0.5, 0, 0.3),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createFileAttachmentAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	// macOS starts up iTunes for audio file attachments.

	fileName := testAudioFileWAV

	ir, err := xRefTable.NewEmbeddedFileStreamDict(fileName)
	if err != nil {
		return nil, err
	}

	fn := filepath.Base(fileName)
	fileSpecDict, err := xRefTable.NewFileSpecDict(fn, EncodeUTF16String(fn), "attached by pdfcpu", *ir)
	if err != nil {
		return nil, err
	}

	ir, err = xRefTable.IndRefForNewObject(fileSpecDict)
	if err != nil {
		return nil, err
	}

	now := StringLiteral(DateString(time.Now()))

	d := Dict(
		map[string]Object{
			"Type":         Name("Annot"),
			"Subtype":      Name("FileAttachment"),
			"Contents":     StringLiteral("FileAttachment Annotation"),
			"Rect":         annotRect,
			"P":            pageIndRef,
			"M":            now,
			"F":            Integer(0),
			"Border":       NewIntegerArray(0, 0, 1),
			"C":            NewNumberArray(0.5, 0.0, 0.5),
			"CA":           Float(0.95),
			"CreationDate": now,
			"Name":         Name("Paperclip"),
			"FS":           *ir,
			"NM":           StringLiteral("SoundFileAttachmentAnnot"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createFileSpecDict(xRefTable *XRefTable, fileName string) (Dict, error) {
	ir, err := xRefTable.NewEmbeddedFileStreamDict(fileName)
	if err != nil {
		return nil, err
	}
	fn := filepath.Base(fileName)
	return xRefTable.NewFileSpecDict(fn, EncodeUTF16String(fn), "attached by pdfcpu", *ir)
}

func createSoundObject(xRefTable *XRefTable) (*IndirectRef, error) {
	fileName := testAudioFileWAV
	fileSpecDict, err := createFileSpecDict(xRefTable, fileName)
	if err != nil {
		return nil, err
	}
	return xRefTable.NewSoundStreamDict(fileName, 44100, fileSpecDict)
}

func createSoundAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	indRef, err := createSoundObject(xRefTable)
	if err != nil {
		return nil, err
	}

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Sound"),
			"Contents": StringLiteral("Sound Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0.5, 0.5),
			"Sound":    *indRef,
			"Name":     Name("Speaker"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createMovieDict(xRefTable *XRefTable) (*IndirectRef, error) {

	// not supported: mp3,mp4,m4a

	fileSpecDict, err := createFileSpecDict(xRefTable, testAudioFileWAV)
	if err != nil {
		return nil, err
	}

	d := Dict(
		map[string]Object{
			"F":      fileSpecDict,
			"Aspect": NewIntegerArray(200, 200),
			"Rotate": Integer(0),
			"Poster": Boolean(true),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createMovieAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	indRef, err := createMovieDict(xRefTable)
	if err != nil {
		return nil, err
	}

	movieActivationDict := Dict(
		map[string]Object{
			"Start":        Integer(10),
			"Duration":     Integer(60),
			"Rate":         Float(1.0),
			"Volume":       Float(1.0),
			"ShowControls": Boolean(true),
			"Mode":         Name("Once"),
			"Synchronous":  Boolean(false),
		},
	)

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Movie"),
			"Contents": StringLiteral("Movie Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 3), // rounded corners don't work
			"C":        NewNumberArray(0.3, 0.5, 0.5),
			"Movie":    *indRef,
			"T":        StringLiteral("Sample Movie"),
			"A":        movieActivationDict,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createMediaRenditionAction(xRefTable *XRefTable, mediaClipDataDict *IndirectRef) Dict {

	r := createMediaRendition(xRefTable, mediaClipDataDict)

	return Dict(
		map[string]Object{
			"Type": Name("Action"),
			"S":    Name("Rendition"),
			"R":    *r,         // rendition object
			"OP":   Integer(0), // Play
		},
	)
}

func createSelectorRenditionAction(mediaClipDataDict *IndirectRef) Dict {

	r := createSelectorRendition(mediaClipDataDict)

	return Dict(
		map[string]Object{
			"Type": Name("Action"),
			"S":    Name("Rendition"),
			"R":    *r,         // rendition object
			"OP":   Integer(0), // Play
		},
	)
}

func createScreenAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	ir, err := createMediaClipDataDict(xRefTable)
	if err != nil {
		return nil, err
	}

	mediaRenditionAction := createMediaRenditionAction(xRefTable, ir)

	selectorRenditionAction := createSelectorRenditionAction(ir)

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Screen"),
			"Contents": StringLiteral("Screen Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 3),
			"C":        NewNumberArray(0.2, 0.8, 0.5),
			"A":        mediaRenditionAction,
			"AA": Dict(
				map[string]Object{
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

func createWidgetAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	appearanceCharacteristicsDict := Dict(
		map[string]Object{
			"R":  Integer(0),
			"BC": NewNumberArray(0.0, 0.0, 0.0),
			"BG": NewNumberArray(0.5, 0.0, 0.5),
			"RC": StringLiteral("Rollover caption"),
			"IF": Dict(
				map[string]Object{
					"SW": Name("A"),
					"S":  Name("A"),
					"FB": Boolean(true),
				},
			),
		},
	)

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Widget"),
			"Contents": StringLiteral("Widget Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 3),
			"C":        NewNumberArray(0.5, 0.5, 0.5),
			"MK":       appearanceCharacteristicsDict,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createXObjectForPrinterMark(xRefTable *XRefTable) (*IndirectRef, error) {
	buf := `0 0 m 0 25 l 25 25 l 25 0 l s`
	sd, _ := xRefTable.NewStreamDictForBuf([]byte(buf))
	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", NewNumberArray(0, 0, 25, 25))
	sd.Insert("Matrix", NewIntegerArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createPrinterMarkAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	ir, err := createXObjectForPrinterMark(xRefTable)
	if err != nil {
		return nil, err
	}

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("PrinterMark"),
			"Contents": StringLiteral("PrinterMark Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 3),
			"C":        NewNumberArray(0.2, 0.8, 0.5),
			"F":        Integer(0),
			"AP": Dict(
				map[string]Object{
					"N": *ir,
				},
			),
			"MN": Name("ColorBar"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createXObjectForWaterMark(xRefTable *XRefTable) (*IndirectRef, error) {
	fIndRef, err := EnsureFontDict(xRefTable, "Helvetica", false, nil)
	if err != nil {
		return nil, err
	}

	fResDict := NewDict()
	fResDict.Insert("F1", *fIndRef)
	resourceDict := NewDict()
	resourceDict.Insert("Font", fResDict)

	buf := `0 0 m 0 200 l 200 200 l 200 0 l s BT /F1 48 Tf 0.7 0.7 -0.7 0.7 30 10 Tm 1 Tr 2 w (Watermark) Tj ET`
	sd, _ := xRefTable.NewStreamDictForBuf([]byte(buf))
	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", NewNumberArray(0, 0, 200, 200))
	sd.Insert("Matrix", NewIntegerArray(1, 0, 0, 1, 0, 0))
	sd.Insert("Resources", resourceDict)

	if err = sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createWaterMarkAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	ir, err := createXObjectForWaterMark(xRefTable)
	if err != nil {
		return nil, err
	}

	d1 := Dict(
		map[string]Object{
			"Type":   Name("FixedPrint"),
			"Matrix": NewIntegerArray(1, 0, 0, 1, 72, -72),
			"H":      Float(0),
			"V":      Float(0),
		},
	)

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Watermark"),
			"Contents": StringLiteral("Watermark Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 3),
			"C":        NewNumberArray(0.2, 0.8, 0.5),
			"F":        Integer(0),
			"AP": Dict(
				map[string]Object{
					"N": *ir,
				},
			),
			"FixedPrint": d1,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func create3DAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("3D"),
			"Contents": StringLiteral("3D Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 3),
			"C":        NewNumberArray(0.2, 0.8, 0.5),
			"F":        Integer(0),
			"3DD":      NewDict(), // stream or 3D reference dict
			"3DV":      Name("F"),
			"3DA":      NewDict(), // activation dict
			"3DI":      Boolean(true),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createRedactAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	qp := Array{}
	qp = append(qp, annotRect[0])
	qp = append(qp, annotRect[1])
	qp = append(qp, annotRect[2])
	qp = append(qp, annotRect[1])
	qp = append(qp, annotRect[2])
	qp = append(qp, annotRect[3])
	qp = append(qp, annotRect[0])
	qp = append(qp, annotRect[3])

	d := Dict(
		map[string]Object{
			"Type":        Name("Annot"),
			"Subtype":     Name("Redact"),
			"Contents":    StringLiteral("Redact Annotation"),
			"Rect":        annotRect,
			"P":           pageIndRef,
			"Border":      NewIntegerArray(0, 0, 3),
			"C":           NewNumberArray(0.2, 0.8, 0.5),
			"F":           Integer(0),
			"QuadPoints":  qp,
			"IC":          NewNumberArray(0.5, 0.0, 0.9),
			"OverlayText": StringLiteral("An overlay"),
			"Repeat":      Boolean(true),
			"DA":          StringLiteral("x"),
			"Q":           Integer(1),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createRemoteGoToAction(xRefTable *XRefTable) (*IndirectRef, error) {

	d := Dict(
		map[string]Object{
			"Type":      Name("Action"),
			"S":         Name("GoToR"),
			"F":         StringLiteral("./go.pdf"),
			"D":         Array{Integer(0), Name("Fit")},
			"NewWindow": Boolean(true),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationWithRemoteGoToAction(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	ir, err := createRemoteGoToAction(xRefTable)
	if err != nil {
		return nil, err
	}

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Link"),
			"Contents": StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A":        *ir,
			"H":        Name("I"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createEmbeddedGoToAction(xRefTable *XRefTable) (*IndirectRef, error) {

	f := filepath.Join(testDir, "go.pdf")
	fileSpecDict, err := createFileSpecDict(xRefTable, f)
	if err != nil {
		return nil, err
	}

	d := Dict(
		map[string]Object{
			"Type":      Name("Action"),
			"S":         Name("GoToE"),
			"F":         fileSpecDict,
			"D":         Array{Integer(0), Name("Fit")},
			"NewWindow": Boolean(true), // not honored by Acrobat Reader.
			"T": Dict(
				map[string]Object{
					"R": Name("C"),
					"N": StringLiteral(filepath.Base(f)),
				},
			),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationWithEmbeddedGoToAction(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	ir, err := createEmbeddedGoToAction(xRefTable)
	if err != nil {
		return nil, err
	}

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Link"),
			"Contents": StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A":        *ir,
			"H":        Name("I"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithLaunchAction(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Link"),
			"Contents": StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A": Dict(
				map[string]Object{
					"Type": Name("Action"),
					"S":    Name("Launch"),
					"F":    StringLiteral("golang.pdf"),
					"Win": Dict(
						map[string]Object{
							"F": StringLiteral("golang.pdf"),
							"O": StringLiteral("O"),
						},
					),
					"NewWindow": Boolean(true),
				},
			),
			"H": Name("I"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithThreadAction(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Link"),
			"Contents": StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A": Dict(
				map[string]Object{
					"Type": Name("Action"),
					"S":    Name("Thread"),
					"D":    Integer(0), // jump to first article thread
					"B":    Integer(0), // jump to first bead
				},
			),
			"H": Name("I"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithSoundAction(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	ir, err := createSoundObject(xRefTable)
	if err != nil {
		return nil, err
	}

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Link"),
			"Contents": StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A": Dict(
				map[string]Object{
					"Type":        Name("Action"),
					"S":           Name("Sound"),
					"Sound":       *ir,
					"Synchronous": Boolean(false),
					"Repeat":      Boolean(false),
					"Mix":         Boolean(false),
				},
			),
			"H": Name("I"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithMovieAction(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Link"),
			"Contents": StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A": Dict(
				map[string]Object{
					"Type":      Name("Action"),
					"S":         Name("Movie"),
					"T":         StringLiteral("Sample Movie"),
					"Operation": Name("Play"),
				},
			),
			"H": Name("I"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithHideAction(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	hideActionDict := Dict(
		map[string]Object{
			"Type": Name("Action"),
			"S":    Name("Hide"),
			"H":    Boolean(true),
		},
	)

	d := Dict(
		map[string]Object{
			"Type":     Name("Annot"),
			"Subtype":  Name("Link"),
			"Contents": StringLiteral("Link Annotation"),
			"Rect":     annotRect,
			"P":        pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A":        hideActionDict,
			"H":        Name("I"),
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

func createTrapNetAnnotation(xRefTable *XRefTable, pageIndRef IndirectRef, annotRect Array) (*IndirectRef, error) {

	ir, err := EnsureFontDict(xRefTable, "Helvetica", false, nil)
	if err != nil {
		return nil, err
	}

	d := Dict(
		map[string]Object{
			"Type":         Name("Annot"),
			"Subtype":      Name("TrapNet"),
			"Contents":     StringLiteral("TrapNet Annotation"),
			"Rect":         annotRect,
			"P":            pageIndRef,
			"Border":       NewIntegerArray(0, 0, 3),
			"C":            NewNumberArray(0.2, 0.8, 0.5),
			"F":            Integer(0),
			"LastModified": StringLiteral(DateString(time.Now())),
			"FontFauxing":  Array{*ir},
		},
	)

	return xRefTable.IndRefForNewObject(d)
}
