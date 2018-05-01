package pdfcpu

import (
	"path"
	"time"
)

// Functions needed to create a test.pdf that gets used for validation testing (see process_test.go)

func createTextAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Text"),
			"Contents": PDFStringLiteral("Text Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 5),
			"C":        NewNumberArray(1, 0, 0),
			"Name":     PDFName("Note"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	usageDict := PDFDict{
		Dict: map[string]PDFObject{
			"CreatorInfo": PDFDict{
				Dict: map[string]PDFObject{
					"Creator": PDFStringLiteral("pdfcpu"),
					"Subtype": PDFName("Technical"),
				},
			},
			"Language": PDFDict{
				Dict: map[string]PDFObject{
					"Lang":      PDFStringLiteral("en-us"),
					"Preferred": PDFName("ON"),
				},
			},
			"Export": PDFDict{
				Dict: map[string]PDFObject{
					"ExportState": PDFName("ON"),
				},
			},
			"Zoom": PDFDict{
				Dict: map[string]PDFObject{
					"min": PDFFloat(0),
				},
			},
			"Print": PDFDict{
				Dict: map[string]PDFObject{
					"Subtype":    PDFName("Watermark"),
					"PrintState": PDFName("ON"),
				},
			},
			"View": PDFDict{
				Dict: map[string]PDFObject{
					"ViewState": PDFName("Ind"),
				},
			},
			"User": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("ON"),
					"Name": PDFStringLiteral("Horst Rutter"),
				},
			},
			"PageElement": PDFDict{
				Dict: map[string]PDFObject{
					"Subtype": PDFName("FG"),
				},
			},
		},
	}

	optionalContentGroupDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type":   PDFName("OCG"),
			"Name":   PDFStringLiteral("OCG"),
			"Intent": PDFName("Design"),
			"Usage":  usageDict,
		},
	}

	uriActionDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Action"),
			"S":    PDFName("URI"),
			"URI":  PDFStringLiteral("https://golang.org"),
		},
	}

	indRef, err := xRefTable.IndRefForNewObject(uriActionDict)
	if err != nil {
		return nil, err
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Link"),
			"Contents": PDFStringLiteral("Link Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 5),
			"C":        NewNumberArray(0, 0, 1),
			"A":        *indRef,
			"H":        PDFName("I"),
			"PA":       *indRef,
			"OC":       optionalContentGroupDict,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createFreeTextAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("FreeText"),
			"Contents": PDFStringLiteral("FreeText Annotation"),
			"F":        PDFInteger(128), // Lock
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 1, 0),
			"DA":       PDFStringLiteral("DA"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLineAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Line"),
			"Contents": PDFStringLiteral("Line Annotation"),
			"F":        PDFInteger(128), // Lock
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 1, 0),
			"L":        *annotRect,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createSquareAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Square"),
			"Contents": PDFStringLiteral("Square Annotation"),
			"F":        PDFInteger(128), // Lock
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, .3, .3),
			"IC":       NewNumberArray(0.8, .8, .8),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createCircleAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Circle"),
			"Contents": PDFStringLiteral("Circle Annotation"),
			"F":        PDFInteger(128), // Lock
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 10),
			"C":        NewNumberArray(0.5, 0, 5, 0),
			"IC":       NewNumberArray(0.8, .8, .8),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createPolygonAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	// Construct a polyline using the annot rects both lower corners and the upper right corner.
	v := PDFArray{nil, nil, nil, nil}
	copy(v, *annotRect)
	v = append(v, (*annotRect)[2])
	v = append(v, (*annotRect)[1])

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Polygon"),
			"Contents": PDFStringLiteral("Polygon Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 1, 0),
			"Vertices": v,
			"IC":       NewNumberArray(0.3, 0.5, 0.0),
			"BS": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("Border"),
					"W":    PDFFloat(0.5),
					"S":    PDFName("D"),
				},
			},
			"BE": PDFDict{
				Dict: map[string]PDFObject{
					"S": PDFName("C"),
					"I": PDFFloat(1),
				},
			},
			"IT": PDFName("PolygonCloud"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createPolyLineAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	// Construct a polyline using the annot rects both lower corners and the upper right corner.
	v := PDFArray{nil, nil, nil, nil}
	copy(v, *annotRect)
	v = append(v, (*annotRect)[2])
	v = append(v, (*annotRect)[1])

	optionalContentGroupDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type":   PDFName("OCG"),
			"Name":   PDFStringLiteral("OCG"),
			"Intent": NewNameArray("Design", "View"),
		},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("PolyLine"),
			"Contents": PDFStringLiteral("PolyLine Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 1, 0),
			"Vertices": v,
			"OC":       optionalContentGroupDict,
			"IC":       NewNumberArray(0.3, 0.5, 0.0),
			"BS": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("Border"),
					"W":    PDFFloat(0.5),
					"S":    PDFName("D"),
				},
			},
			"IT": PDFName("PolygonCloud"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createHighlightAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	// Create a quad points array corresponding to the annot rect.
	ar := *annotRect
	qp := PDFArray{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	optionalContentGroupDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("OCG"),
			"Name": PDFStringLiteral("OCG"),
		},
	}

	optionalContentMembershipDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("OCMD"),
			"OCGs": PDFArray{nil, optionalContentGroupDict},
			"P":    PDFName("AllOn"),
			"VE":   PDFArray{},
		},
	}

	_ = optionalContentMembershipDict

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":       PDFName("Annot"),
			"Subtype":    PDFName("Highlight"),
			"Contents":   PDFStringLiteral("Highlight Annotation"),
			"Rect":       *annotRect,
			"P":          *pageIndRef,
			"Border":     NewIntegerArray(0, 0, 1),
			"C":          NewNumberArray(.2, 0, 0),
			"OC":         optionalContentMembershipDict,
			"QuadPoints": qp,
			"T":          PDFStringLiteral("MyTitle"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createUnderlineAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	ar := *annotRect
	qp := PDFArray{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":       PDFName("Annot"),
			"Subtype":    PDFName("Underline"),
			"Contents":   PDFStringLiteral("Underline Annotation"),
			"Rect":       *annotRect,
			"P":          *pageIndRef,
			"Border":     NewIntegerArray(0, 0, 1),
			"C":          NewNumberArray(.5, 0, 0),
			"QuadPoints": qp,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createSquigglyAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	ar := *annotRect
	qp := PDFArray{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":       PDFName("Annot"),
			"Subtype":    PDFName("Squiggly"),
			"Contents":   PDFStringLiteral("Squiggly Annotation"),
			"Rect":       *annotRect,
			"P":          *pageIndRef,
			"Border":     NewIntegerArray(0, 0, 1),
			"C":          NewNumberArray(.5, 0, 0),
			"QuadPoints": qp,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createStrikeOutAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	ar := *annotRect
	qp := PDFArray{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":       PDFName("Annot"),
			"Subtype":    PDFName("StrikeOut"),
			"Contents":   PDFStringLiteral("StrikeOut Annotation"),
			"Rect":       *annotRect,
			"P":          *pageIndRef,
			"Border":     NewIntegerArray(0, 0, 1),
			"C":          NewNumberArray(.5, 0, 0),
			"QuadPoints": qp,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createCaretAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Caret"),
			"Contents": PDFStringLiteral("Caret Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0.5, 0.5, 0),
			"RD":       NewRectangle(0, 0, 0, 0),
			"Sy":       PDFName("None"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createStampAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Stamp"),
			"Contents": PDFStringLiteral("Stamp Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0.5, 0.5, 0.9),
			"Name":     PDFName("Approved"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createInkAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	ar := *annotRect

	l := PDFArray{
		PDFArray{ar[0], ar[1], ar[2], ar[1]},
		PDFArray{ar[2], ar[1], ar[2], ar[3]},
		PDFArray{ar[2], ar[3], ar[0], ar[3]},
		PDFArray{ar[0], ar[3], ar[0], ar[1]},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Ink"),
			"Contents": PDFStringLiteral("Ink Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0.5, 0, 0.3),
			"InkList":  l,
			"ExData": PDFDict{
				Dict: map[string]PDFObject{
					"Type":    PDFName("ExData"),
					"Subtype": PDFName("Markup3D"),
				},
			},
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createPopupAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Popup"),
			"Contents": PDFStringLiteral("Ink Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0.5, 0, 0.3),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createFileAttachmentAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	// Mac starts up iTunes for FileAttachments.

	fileName := testAudioFileWAV

	sd, err := xRefTable.NewEmbeddedFileStreamDict(fileName)
	if err != nil {
		return nil, err
	}

	err = encodeStream(sd)
	if err != nil {
		return nil, err
	}

	indRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	fileSpecDict, err := xRefTable.NewFileSpecDict(path.Base(fileName), *indRef)
	if err != nil {
		return nil, err
	}

	indRef, err = xRefTable.IndRefForNewObject(*fileSpecDict)
	if err != nil {
		return nil, err
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":         PDFName("Annot"),
			"Subtype":      PDFName("FileAttachment"),
			"Contents":     PDFStringLiteral("FileAttachment Annotation"),
			"Rect":         *annotRect,
			"P":            *pageIndRef,
			"M":            DateStringLiteral(time.Now()),
			"F":            PDFInteger(0),
			"Border":       NewIntegerArray(0, 0, 1),
			"C":            NewNumberArray(0.5, 0.0, 0.5),
			"CA":           PDFFloat(0.95),
			"CreationDate": DateStringLiteral(time.Now()),
			"Name":         PDFName("Paperclip"),
			"FS":           *indRef,
			"NM":           PDFStringLiteral("SoundFileAttachmentAnnot"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createFileSpecDict(xRefTable *XRefTable, fileName string) (*PDFDict, error) {

	sd, err := xRefTable.NewEmbeddedFileStreamDict(fileName)
	if err != nil {
		return nil, err
	}

	err = encodeStream(sd)
	if err != nil {
		return nil, err
	}

	indRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.NewFileSpecDict(path.Base(fileName), *indRef)
}

func createSoundObject(xRefTable *XRefTable) (*PDFIndirectRef, error) {

	fileName := testAudioFileWAV

	fileSpecDict, err := createFileSpecDict(xRefTable, fileName)
	if err != nil {
		return nil, err
	}

	sd, err := xRefTable.NewSoundStreamDict(fileName, 44100, fileSpecDict)
	if err != nil {
		return nil, err
	}

	err = encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createSoundAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	indRef, err := createSoundObject(xRefTable)
	if err != nil {
		return nil, err
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Sound"),
			"Contents": PDFStringLiteral("Sound Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0.5, 0.5),
			"Sound":    *indRef,
			"Name":     PDFName("Speaker"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createMovieDict(xRefTable *XRefTable) (*PDFIndirectRef, error) {

	// not supported: mp3,mp4,m4a

	fileSpecDict, err := createFileSpecDict(xRefTable, testAudioFileWAV)
	if err != nil {
		return nil, err
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"F":      *fileSpecDict,
			"Aspect": NewIntegerArray(200, 200),
			"Rotate": PDFInteger(0),
			"Poster": PDFBoolean(true),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createMovieAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	indRef, err := createMovieDict(xRefTable)
	if err != nil {
		return nil, err
	}

	movieActivationDict := PDFDict{
		Dict: map[string]PDFObject{
			"Start":        PDFInteger(10),
			"Duration":     PDFInteger(60),
			"Rate":         PDFFloat(1.0),
			"Volume":       PDFFloat(1.0),
			"ShowControls": PDFBoolean(true),
			"Mode":         PDFName("Once"),
			"Synchronous":  PDFBoolean(false),
		},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Movie"),
			"Contents": PDFStringLiteral("Movie Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 3), // rounded corners don't work
			"C":        NewNumberArray(0.3, 0.5, 0.5),
			"Movie":    *indRef,
			"T":        PDFStringLiteral("Sample Movie"),
			"A":        movieActivationDict,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createMediaRenditionAction(xRefTable *XRefTable, mediaClipDataDict *PDFIndirectRef) *PDFDict {

	r := createMediaRendition(xRefTable, mediaClipDataDict)

	return &PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Action"),
			"S":    PDFName("Rendition"),
			"R":    *r,            // rendition object
			"OP":   PDFInteger(0), // Play
		},
	}

}

func createSelectorRenditionAction(mediaClipDataDict *PDFIndirectRef) *PDFDict {

	r := createSelectorRendition(mediaClipDataDict)

	return &PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Action"),
			"S":    PDFName("Rendition"),
			"R":    *r,            // rendition object
			"OP":   PDFInteger(0), // Play
		},
	}

}

func createScreenAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	indRef, err := createMediaClipDataDict(xRefTable)
	if err != nil {
		return nil, err
	}

	mediaRenditionAction := createMediaRenditionAction(xRefTable, indRef)

	selectorRenditionAction := createSelectorRenditionAction(indRef)

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Screen"),
			"Contents": PDFStringLiteral("Screen Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 3),
			"C":        NewNumberArray(0.2, 0.8, 0.5),
			"A":        *mediaRenditionAction,
			"AA": PDFDict{
				Dict: map[string]PDFObject{
					"D": *selectorRenditionAction,
				},
			},
		},
	}

	indRef, err = xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	// Inject indRef of screen annotation into action dicts.
	mediaRenditionAction.Insert("AN", *indRef)
	selectorRenditionAction.Insert("AN", *indRef)

	return indRef, nil
}

func createWidgetAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	appearanceCharacteristicsDict := PDFDict{
		Dict: map[string]PDFObject{
			"R":  PDFInteger(0),
			"BC": NewNumberArray(0.0, 0.0, 0.0),
			"BG": NewNumberArray(0.5, 0.0, 0.5),
			"RC": PDFStringLiteral("Rollover caption"),
			"IF": PDFDict{
				Dict: map[string]PDFObject{
					"SW": PDFName("A"),
					"S":  PDFName("A"),
					"FB": PDFBoolean(true),
				},
			},
		},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Widget"),
			"Contents": PDFStringLiteral("Widget Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 3),
			"C":        NewNumberArray(0.5, 0.5, 0.5),
			"MK":       appearanceCharacteristicsDict,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createXObjectForPrinterMark(xRefTable *XRefTable) (*PDFIndirectRef, error) {

	buf := `0 0 m 0 25 l 25 25 l 25 0 l s`

	sd := &PDFStreamDict{
		PDFDict: PDFDict{
			Dict: map[string]PDFObject{
				"Type":     PDFName("XObject"),
				"Subtype":  PDFName("Form"),
				"FormType": PDFInteger(1),
				"BBox":     NewRectangle(0, 0, 25, 25),
				"Matrix":   NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		},
		Content:        []byte(buf),
		FilterPipeline: []PDFFilter{{Name: "FlateDecode", DecodeParms: nil}}}

	sd.InsertName("Filter", "FlateDecode")

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createPrinterMarkAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	indRef, err := createXObjectForPrinterMark(xRefTable)
	if err != nil {
		return nil, err
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("PrinterMark"),
			"Contents": PDFStringLiteral("PrinterMark Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 3),
			"C":        NewNumberArray(0.2, 0.8, 0.5),
			"F":        PDFInteger(0),
			"AP": PDFDict{
				Dict: map[string]PDFObject{
					"N": *indRef,
				},
			},
			"MN": PDFName("ColorBar"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createXObjectForWaterMark(xRefTable *XRefTable) (*PDFIndirectRef, error) {

	fIndRef, err := createFontDict(xRefTable)
	if err != nil {
		return nil, err
	}

	fResDict := NewPDFDict()
	fResDict.Insert("F1", *fIndRef)
	resourceDict := NewPDFDict()
	resourceDict.Insert("Font", fResDict)

	buf := `0 0 m 0 200 l 200 200 l 200 0 l s BT /F1 48 Tf 0.7 0.7 -0.7 0.7 30 10 Tm 1 Tr 2 w (Watermark) Tj ET`

	sd := &PDFStreamDict{
		PDFDict: PDFDict{
			Dict: map[string]PDFObject{
				"Type":      PDFName("XObject"),
				"Subtype":   PDFName("Form"),
				"FormType":  PDFInteger(1),
				"BBox":      NewRectangle(0, 0, 200, 200),
				"Matrix":    NewIntegerArray(1, 0, 0, 1, 0, 0),
				"Resources": resourceDict,
			},
		},
		Content:        []byte(buf),
		FilterPipeline: []PDFFilter{{Name: "FlateDecode", DecodeParms: nil}}}

	sd.InsertName("Filter", "FlateDecode")

	err = encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createWaterMarkAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	indRef, err := createXObjectForWaterMark(xRefTable)
	if err != nil {
		return nil, err
	}

	d1 := PDFDict{
		Dict: map[string]PDFObject{
			"Type":   PDFName("FixedPrint"),
			"Matrix": NewIntegerArray(1, 0, 0, 1, 72, -72),
			"H":      PDFFloat(0),
			"V":      PDFFloat(0),
		},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Watermark"),
			"Contents": PDFStringLiteral("Watermark Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 3),
			"C":        NewNumberArray(0.2, 0.8, 0.5),
			"F":        PDFInteger(0),
			"AP": PDFDict{
				Dict: map[string]PDFObject{
					"N": *indRef,
				},
			},
			"FixedPrint": d1,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func create3DAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("3D"),
			"Contents": PDFStringLiteral("3D Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 3),
			"C":        NewNumberArray(0.2, 0.8, 0.5),
			"F":        PDFInteger(0),
			"3DD":      NewPDFDict(), // stream or 3D reference dict
			"3DV":      PDFName("F"),
			"3DA":      NewPDFDict(), // activation dict
			"3DI":      PDFBoolean(true),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createRedactAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	ar := *annotRect
	qp := PDFArray{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":        PDFName("Annot"),
			"Subtype":     PDFName("Redact"),
			"Contents":    PDFStringLiteral("Redact Annotation"),
			"Rect":        *annotRect,
			"P":           *pageIndRef,
			"Border":      NewIntegerArray(0, 0, 3),
			"C":           NewNumberArray(0.2, 0.8, 0.5),
			"F":           PDFInteger(0),
			"QuadPoints":  qp,
			"IC":          NewNumberArray(0.5, 0.0, 0.9),
			"OverlayText": PDFStringLiteral("An overlay"),
			"Repeat":      PDFBoolean(true),
			"DA":          PDFStringLiteral("x"),
			"Q":           PDFInteger(1),
		}}

	return xRefTable.IndRefForNewObject(d)
}

func createRemoteGoToAction(xRefTable *XRefTable) (*PDFIndirectRef, error) {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":      PDFName("Action"),
			"S":         PDFName("GoToR"),
			"F":         PDFStringLiteral(".\\/go.pdf"),
			"D":         PDFArray{PDFInteger(0), PDFName("Fit")},
			"NewWindow": PDFBoolean(true),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationWithRemoteGoToAction(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	indRef, err := createRemoteGoToAction(xRefTable)
	if err != nil {
		return nil, err
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Link"),
			"Contents": PDFStringLiteral("Link Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A":        *indRef,
			"H":        PDFName("I"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createEmbeddedGoToAction(xRefTable *XRefTable) (*PDFIndirectRef, error) {

	// fileSpecDict, err := createFileSpecDict(xRefTable, "testdata/go.pdf")
	// if err != nil {
	// 	return nil, err
	// }

	_, err := attachAdd(xRefTable, StringSet{"testdata/go.pdf": true})
	if err != nil {
		return nil, err
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Action"),
			"S":    PDFName("GoToE"),
			//"F":         *fileSpecDict,
			"D":         PDFArray{PDFInteger(0), PDFName("Fit")},
			"NewWindow": PDFBoolean(true), // not honored by Acrobat Reader.
			"T": PDFDict{
				Dict: map[string]PDFObject{
					"R": PDFName("C"),
					"N": PDFStringLiteral("testdata/go.pdf"),
				},
			},
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationWithEmbeddedGoToAction(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	indRef, err := createEmbeddedGoToAction(xRefTable)
	if err != nil {
		return nil, err
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Link"),
			"Contents": PDFStringLiteral("Link Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A":        *indRef,
			"H":        PDFName("I"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithLaunchAction(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Link"),
			"Contents": PDFStringLiteral("Link Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("Action"),
					"S":    PDFName("Launch"),
					"F":    PDFStringLiteral(".\\/golang.pdf"), // e.g pdf, wav..
					"Win": PDFDict{
						Dict: map[string]PDFObject{
							"F": PDFStringLiteral("golang.pdf"),
							"O": PDFStringLiteral("O"),
						},
					},
					"NewWindow": PDFBoolean(true),
				},
			},
			"H": PDFName("I"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithThreadAction(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Link"),
			"Contents": PDFStringLiteral("Link Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("Action"),
					"S":    PDFName("Thread"),
					"D":    PDFInteger(0), // jump to first article thread
					"B":    PDFInteger(0), // jump to first bead
				},
			},
			"H": PDFName("I"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithSoundAction(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	indRef, err := createSoundObject(xRefTable)
	if err != nil {
		return nil, err
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Link"),
			"Contents": PDFStringLiteral("Link Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A": PDFDict{
				Dict: map[string]PDFObject{
					"Type":        PDFName("Action"),
					"S":           PDFName("Sound"),
					"Sound":       *indRef,
					"Synchronous": PDFBoolean(false),
					"Repeat":      PDFBoolean(false),
					"Mix":         PDFBoolean(false),
				},
			},
			"H": PDFName("I"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithMovieAction(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Link"),
			"Contents": PDFStringLiteral("Link Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A": PDFDict{
				Dict: map[string]PDFObject{
					"Type":      PDFName("Action"),
					"S":         PDFName("Movie"),
					"T":         PDFStringLiteral("Sample Movie"),
					"Operation": PDFName("Play"),
				},
			},
			"H": PDFName("I"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithHideAction(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	hideActionDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Action"),
			"S":    PDFName("Hide"),
			"H":    PDFBoolean(true),
		},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Annot"),
			"Subtype":  PDFName("Link"),
			"Contents": PDFStringLiteral("Link Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   NewIntegerArray(0, 0, 1),
			"C":        NewNumberArray(0, 0, 1),
			"A":        hideActionDict,
			"H":        PDFName("I"),
		},
	}

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	// We hide the link annotation itself.
	hideActionDict.Insert("T", *indRef)

	return indRef, nil
}

func createTrapNetAnnotation(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, annotRect *PDFArray) (*PDFIndirectRef, error) {

	indRef, err := createFontDict(xRefTable)
	if err != nil {
		return nil, err
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type":         PDFName("Annot"),
			"Subtype":      PDFName("TrapNet"),
			"Contents":     PDFStringLiteral("TrapNet Annotation"),
			"Rect":         *annotRect,
			"P":            *pageIndRef,
			"Border":       NewIntegerArray(0, 0, 3),
			"C":            NewNumberArray(0.2, 0.8, 0.5),
			"F":            PDFInteger(0),
			"LastModified": DateStringLiteral(time.Now()),
			"FontFauxing":  PDFArray{*indRef},
		},
	}

	return xRefTable.IndRefForNewObject(d)
}
