package create

import (
	"path"
	"time"

	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/types"
)

func createLinkAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	usageDict := types.PDFDict{
		Dict: map[string]interface{}{
			"CreatorInfo": types.PDFDict{
				Dict: map[string]interface{}{
					"Creator": types.PDFStringLiteral("pdfcpu"),
					"Subtype": types.PDFName("Technical"),
				},
			},
			"Language": types.PDFDict{
				Dict: map[string]interface{}{
					"Lang":      types.PDFStringLiteral("en-us"),
					"Preferred": types.PDFName("ON"),
				},
			},
			"Export": types.PDFDict{
				Dict: map[string]interface{}{
					"ExportState": types.PDFName("ON"),
				},
			},
			"Zoom": types.PDFDict{
				Dict: map[string]interface{}{
					"min": types.PDFFloat(0),
				},
			},
			"Print": types.PDFDict{
				Dict: map[string]interface{}{
					"Subtype":    types.PDFName("Watermark"),
					"PrintState": types.PDFName("ON"),
				},
			},
			"View": types.PDFDict{
				Dict: map[string]interface{}{
					"ViewState": types.PDFName("Ind"),
				},
			},
			"User": types.PDFDict{
				Dict: map[string]interface{}{
					"Type": types.PDFName("ON"),
					"Name": types.PDFStringLiteral("Horst Rutter"),
				},
			},
			"PageElement": types.PDFDict{
				Dict: map[string]interface{}{
					"Subtype": types.PDFName("FG"),
				},
			},
		},
	}

	optionalContentGroupDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":   types.PDFName("OCG"),
			"Name":   types.PDFStringLiteral("OCG"),
			"Intent": types.PDFName("Design"),
			"Usage":  usageDict,
		},
	}

	uriActionDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Type": types.PDFName("Action"),
			"S":    types.PDFName("URI"),
			"URI":  types.PDFStringLiteral("https://golang.org"),
		},
	}

	indRef, err := xRefTable.IndRefForNewObject(uriActionDict)
	if err != nil {
		return nil, err
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Link"),
			"Contents": types.PDFStringLiteral("Link Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"A":        *indRef,
			"PA":       *indRef,
			"OC":       optionalContentGroupDict,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createPolyLineAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	optionalContentGroupDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":   types.PDFName("OCG"),
			"Name":   types.PDFStringLiteral("OCG"),
			"Intent": types.NewNameArray("Design", "View"),
		},
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Polygon"),
			"Contents": types.PDFStringLiteral("Polygon Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"OC":       optionalContentGroupDict,
			//"Vertices": types.NewNumberArray(100, 100, 200, 200),
			"Vertices": *annotRect,
			"IC":       types.NewNumberArray(0.3, 0.5, 0.0),
			"BS": types.PDFDict{
				Dict: map[string]interface{}{
					"Type": types.PDFName("Border"),
					"W":    types.PDFFloat(0.5),
					"S":    types.PDFName("D"),
				},
			},
			"BE": types.PDFDict{
				Dict: map[string]interface{}{
					"S": types.PDFName("C"),
					"I": types.PDFFloat(1),
				},
			},
			"IT": types.PDFName("PolygonCloud"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createMarkupAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	optionalContentGroupDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Type": types.PDFName("OCG"),
			"Name": types.PDFStringLiteral("OCG"),
		},
	}

	optionalContentMembershipDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Type": types.PDFName("OCMD"),
			"OCGs": types.PDFArray{nil, optionalContentGroupDict},
			"P":    types.PDFName("AllOn"),
			"VE":   types.PDFArray{},
		},
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":       types.PDFName("Annot"),
			"Subtype":    types.PDFName("Highlight"),
			"Contents":   types.PDFStringLiteral("Highlight Annotation"),
			"Rect":       *annotRect,
			"P":          *pageIndRef,
			"OC":         optionalContentMembershipDict,
			"QuadPoints": types.NewNumberArray(100, 100, 200, 100, 200, 200, 100, 200),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createCaretAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Caret"),
			"Contents": types.PDFStringLiteral("Caret Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"RD":       types.NewRectangle(5, 5, 5, 5),
			"Sy":       types.PDFName("P"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createInkAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Ink"),
			"Contents": types.PDFStringLiteral("Ink Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"InkList": types.PDFArray{
				types.NewNumberArray(102, 102, 110, 102, 117, 117),
				types.NewNumberArray(112, 110, 113, 113),
			},
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createFileAttachmentAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	// Mac starts up iTunes for FileAttachments.
	filename := "testdata/a2.wav"
	//filename := "testdata/departure.mp3"

	sd, err := xRefTable.NewEmbeddedFileStreamDict(filename)
	if err != nil {
		return nil, err
	}

	err = filter.EncodeStream(sd)
	if err != nil {
		return nil, err
	}

	indRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	fileSpecDict, err := xRefTable.NewFileSpecDict(path.Base(filename), *indRef)
	if err != nil {
		return nil, err
	}

	indRef, err = xRefTable.IndRefForNewObject(*fileSpecDict)
	if err != nil {
		return nil, err
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":         types.PDFName("Annot"),
			"Subtype":      types.PDFName("FileAttachment"),
			"Contents":     types.PDFStringLiteral("FileAttachment Annotation"),
			"Rect":         *annotRect,
			"P":            *pageIndRef,
			"M":            types.DateStringLiteral(time.Now()),
			"F":            types.PDFInteger(0),
			"C":            types.NewNumberArray(0.5, 0.0, 0.5),
			"CA":           types.PDFFloat(0.95),
			"CreationDate": types.DateStringLiteral(time.Now()),
			"Name":         types.PDFName("Paperclip"),
			"FS":           *indRef,
			"NM":           types.PDFStringLiteral("SoundFileAttachmentAnnot"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createFileSpecDict(xRefTable *types.XRefTable, fileName string) (*types.PDFDict, error) {

	sd, err := xRefTable.NewEmbeddedFileStreamDict(fileName)
	if err != nil {
		return nil, err
	}

	err = filter.EncodeStream(sd)
	if err != nil {
		return nil, err
	}

	indRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.NewFileSpecDict(path.Base(fileName), *indRef)
}

func createSoundAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	fileName := "testdata/a2.wav"

	fileSpecDict, err := createFileSpecDict(xRefTable, fileName)
	if err != nil {
		return nil, err
	}

	sd, err := xRefTable.NewSoundStreamDict(fileName, 48044, fileSpecDict)
	if err != nil {
		return nil, err
	}

	err = filter.EncodeStream(sd)
	if err != nil {
		return nil, err
	}

	indRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Sound"),
			"Contents": types.PDFStringLiteral("Sound Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Sound":    *indRef,
			"Name":     types.PDFName("Speaker"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createMovieDict(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	fileName := "testdata/a2.wav"
	// not supported: mp3,mp4,m4a

	fileSpecDict, err := createFileSpecDict(xRefTable, fileName)
	if err != nil {
		return nil, err
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"F":      *fileSpecDict,
			"Aspect": types.NewIntegerArray(200, 200),
			"Rotate": types.PDFInteger(0),
			"Poster": types.PDFBoolean(false),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createMovieAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	indRef, err := createMovieDict(xRefTable)
	if err != nil {
		return nil, err
	}

	movieActivationDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Start":        types.PDFInteger(10),
			"Duration":     types.PDFInteger(60),
			"Rate":         types.PDFFloat(1.0),
			"Volume":       types.PDFFloat(1.0),
			"ShowControls": types.PDFBoolean(true),
			"Mode":         types.PDFName("Once"),
			"Synchronous":  types.PDFBoolean(false),
		},
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Movie"),
			"Contents": types.PDFStringLiteral("Movie Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Movie":    *indRef,
			"T":        types.PDFStringLiteral("Sample Movie"),
			"A":        movieActivationDict,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createWidgetAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	appearanceCharacteristicsDict := types.PDFDict{
		Dict: map[string]interface{}{
			"R":  types.PDFInteger(0),
			"BC": types.NewNumberArray(0.5, 0.0, 0.5),
			"IF": types.PDFDict{
				Dict: map[string]interface{}{
					"SW": types.PDFName("A"),
					"S":  types.PDFName("A"),
					"FB": types.PDFBoolean(true),
				}},
		}}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Widget"),
			"Contents": types.PDFStringLiteral("Widget Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"MK":       appearanceCharacteristicsDict,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createMediaRenditionAction(xRefTable *types.XRefTable, mediaClipDataDict *types.PDFIndirectRef) *types.PDFDict {

	r := createMediaRendition(xRefTable, mediaClipDataDict)

	return &types.PDFDict{
		Dict: map[string]interface{}{
			"Type": types.PDFName("Action"),
			"S":    types.PDFName("Rendition"),
			"R":    *r,                  // rendition object
			"OP":   types.PDFInteger(0), // Play
		},
	}

}

func createSelectorRenditionAction(mediaClipDataDict *types.PDFIndirectRef) *types.PDFDict {

	r := createSelectorRendition(mediaClipDataDict)

	return &types.PDFDict{
		Dict: map[string]interface{}{
			"Type": types.PDFName("Action"),
			"S":    types.PDFName("Rendition"),
			"R":    *r,                  // rendition object
			"OP":   types.PDFInteger(0), // Play
		},
	}

}

func createScreenAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	indRef, err := createMediaClipDataDict(xRefTable)
	if err != nil {
		return nil, err
	}

	mediaRenditionAction := createMediaRenditionAction(xRefTable, indRef)

	selectorRenditionAction := createSelectorRenditionAction(indRef)

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Screen"),
			"Contents": types.PDFStringLiteral("Screen Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"A":        *mediaRenditionAction,
			"AA": types.PDFDict{
				Dict: map[string]interface{}{
					"E": *selectorRenditionAction,
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

func createXObject(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	// TODO use image
	sd, err := xRefTable.NewEmbeddedFileStreamDict("testdata/a2.wav")
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createPrinterMarkAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	indRef, err := createXObject(xRefTable)
	if err != nil {
		return nil, err
	}

	_ = indRef

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("PrinterMark"),
			"Contents": types.PDFStringLiteral("PrinterMark Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"F":        types.PDFInteger(0),
			//"AP":       types.PDFDict{Dict: map[string]interface{}{"N": *indRef}}, REQUIRED!!!
			"AP": types.NewPDFDict(),
			"MN": types.PDFName("ColorBar"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createWaterMarkAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d1 := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":   types.PDFName("FixedPrint"),
			"Matrix": types.NewIntegerArray(1, 0, 0, 1, 72, -72),
			"H":      types.PDFFloat(0),
			"V":      types.PDFFloat(1.0),
		},
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":       types.PDFName("Annot"),
			"Subtype":    types.PDFName("Watermark"),
			"Contents":   types.PDFStringLiteral("Watermark Annotation"),
			"Rect":       *annotRect,
			"P":          *pageIndRef,
			"F":          types.PDFInteger(0),
			"FixedPrint": d1,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func create3DAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("3D"),
			"Contents": types.PDFStringLiteral("3D Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"F":        types.PDFInteger(0),
			"3DD":      types.NewPDFDict(), // stream or 3D reference dict
			"3DV":      types.PDFName("F"),
			"3DA":      types.NewPDFDict(), // activation dict
			"3DI":      types.PDFBoolean(true),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createRedactAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":        types.PDFName("Annot"),
			"Subtype":     types.PDFName("Redact"),
			"Contents":    types.PDFStringLiteral("Redact Annotation"),
			"Rect":        *annotRect,
			"P":           *pageIndRef,
			"F":           types.PDFInteger(0),
			"QuadPoints":  types.NewNumberArray(200, 200, 200, 250, 250, 250, 200, 250),
			"IC":          types.NewNumberArray(0.5, 0.0, 0.9),
			"OverlayText": types.PDFStringLiteral("An overlay"),
			"Repeat":      types.PDFBoolean(false),
			"DA":          types.PDFStringLiteral(""),
			"Q":           types.PDFInteger(1),
		}}

	return xRefTable.IndRefForNewObject(d)
}

func createTrapNetAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	indRef, err := createFontDict(xRefTable)
	if err != nil {
		return nil, err
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":         types.PDFName("Annot"),
			"Subtype":      types.PDFName("TrapNet"),
			"Contents":     types.PDFStringLiteral("TrapNet Annotation"),
			"Rect":         *annotRect,
			"P":            *pageIndRef,
			"F":            types.PDFInteger(0),
			"LastModified": types.DateStringLiteral(time.Now()),
			"FontFauxing":  types.PDFArray{*indRef},
		},
	}

	return xRefTable.IndRefForNewObject(d)
}
