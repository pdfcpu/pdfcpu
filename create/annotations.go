package create

import (
	"path"
	"time"

	"github.com/hhrutter/pdfcpu/attach"
	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/types"
)

func createTextAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Text"),
			"Contents": types.PDFStringLiteral("Text Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 5),
			"C":        types.NewNumberArray(1, 0, 0),
			"Name":     types.PDFName("Note"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

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
			"Border":   types.NewIntegerArray(0, 0, 5),
			"C":        types.NewNumberArray(0, 0, 1),
			"A":        *indRef,
			"H":        types.PDFName("I"),
			"PA":       *indRef,
			"OC":       optionalContentGroupDict,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createFreeTextAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("FreeText"),
			"Contents": types.PDFStringLiteral("FreeText Annotation"),
			"F":        types.PDFInteger(128), // Lock
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 1, 0),
			"DA":       types.PDFStringLiteral("DA"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLineAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Line"),
			"Contents": types.PDFStringLiteral("Line Annotation"),
			"F":        types.PDFInteger(128), // Lock
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 1, 0),
			"L":        *annotRect,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createSquareAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Square"),
			"Contents": types.PDFStringLiteral("Square Annotation"),
			"F":        types.PDFInteger(128), // Lock
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, .3, .3),
			"IC":       types.NewNumberArray(0.8, .8, .8),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createCircleAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Circle"),
			"Contents": types.PDFStringLiteral("Circle Annotation"),
			"F":        types.PDFInteger(128), // Lock
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 10),
			"C":        types.NewNumberArray(0.5, 0, 5, 0),
			"IC":       types.NewNumberArray(0.8, .8, .8),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createPolygonAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	// Construct a polyline using the annot rects both lower corners and the upper right corner.
	v := types.PDFArray{nil, nil, nil, nil}
	copy(v, *annotRect)
	v = append(v, (*annotRect)[2])
	v = append(v, (*annotRect)[1])

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Polygon"),
			"Contents": types.PDFStringLiteral("Polygon Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 1, 0),
			"Vertices": v,
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

func createPolyLineAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	// Construct a polyline using the annot rects both lower corners and the upper right corner.
	v := types.PDFArray{nil, nil, nil, nil}
	copy(v, *annotRect)
	v = append(v, (*annotRect)[2])
	v = append(v, (*annotRect)[1])

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
			"Subtype":  types.PDFName("PolyLine"),
			"Contents": types.PDFStringLiteral("PolyLine Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 1, 0),
			"Vertices": v,
			"OC":       optionalContentGroupDict,
			"IC":       types.NewNumberArray(0.3, 0.5, 0.0),
			"BS": types.PDFDict{
				Dict: map[string]interface{}{
					"Type": types.PDFName("Border"),
					"W":    types.PDFFloat(0.5),
					"S":    types.PDFName("D"),
				},
			},
			"IT": types.PDFName("PolygonCloud"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createHighlightAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	// Create a quad points array corresponding to the annot rect.
	ar := *annotRect
	qp := types.PDFArray{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

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

	_ = optionalContentMembershipDict

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":       types.PDFName("Annot"),
			"Subtype":    types.PDFName("Highlight"),
			"Contents":   types.PDFStringLiteral("Highlight Annotation"),
			"Rect":       *annotRect,
			"P":          *pageIndRef,
			"Border":     types.NewIntegerArray(0, 0, 1),
			"C":          types.NewNumberArray(.2, 0, 0),
			"OC":         optionalContentMembershipDict,
			"QuadPoints": qp,
			"T":          types.PDFStringLiteral("MyTitle"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createUnderlineAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	ar := *annotRect
	qp := types.PDFArray{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":       types.PDFName("Annot"),
			"Subtype":    types.PDFName("Underline"),
			"Contents":   types.PDFStringLiteral("Underline Annotation"),
			"Rect":       *annotRect,
			"P":          *pageIndRef,
			"Border":     types.NewIntegerArray(0, 0, 1),
			"C":          types.NewNumberArray(.5, 0, 0),
			"QuadPoints": qp,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createSquigglyAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	ar := *annotRect
	qp := types.PDFArray{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":       types.PDFName("Annot"),
			"Subtype":    types.PDFName("Squiggly"),
			"Contents":   types.PDFStringLiteral("Squiggly Annotation"),
			"Rect":       *annotRect,
			"P":          *pageIndRef,
			"Border":     types.NewIntegerArray(0, 0, 1),
			"C":          types.NewNumberArray(.5, 0, 0),
			"QuadPoints": qp,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createStrikeOutAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	// Create a quad points array corresponding to annot rect.
	ar := *annotRect
	qp := types.PDFArray{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":       types.PDFName("Annot"),
			"Subtype":    types.PDFName("StrikeOut"),
			"Contents":   types.PDFStringLiteral("StrikeOut Annotation"),
			"Rect":       *annotRect,
			"P":          *pageIndRef,
			"Border":     types.NewIntegerArray(0, 0, 1),
			"C":          types.NewNumberArray(.5, 0, 0),
			"QuadPoints": qp,
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
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0.5, 0.5, 0),
			"RD":       types.NewRectangle(0, 0, 0, 0),
			"Sy":       types.PDFName("None"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createStampAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Stamp"),
			"Contents": types.PDFStringLiteral("Stamp Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0.5, 0.5, 0.9),
			"Name":     types.PDFName("Approved"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createInkAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	ar := *annotRect

	l := types.PDFArray{
		types.PDFArray{ar[0], ar[1], ar[2], ar[1]},
		types.PDFArray{ar[2], ar[1], ar[2], ar[3]},
		types.PDFArray{ar[2], ar[3], ar[0], ar[3]},
		types.PDFArray{ar[0], ar[3], ar[0], ar[1]},
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Ink"),
			"Contents": types.PDFStringLiteral("Ink Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0.5, 0, 0.3),
			"InkList":  l,
			"ExData": types.PDFDict{
				Dict: map[string]interface{}{
					"Type":    types.PDFName("ExData"),
					"Subtype": types.PDFName("Markup3D"),
				},
			},
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createPopupAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Popup"),
			"Contents": types.PDFStringLiteral("Ink Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0.5, 0, 0.3),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createFileAttachmentAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	// Mac starts up iTunes for FileAttachments.

	fileName := testAudioFileWAV

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

	fileSpecDict, err := xRefTable.NewFileSpecDict(path.Base(fileName), *indRef)
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
			"Border":       types.NewIntegerArray(0, 0, 1),
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

func createSoundObject(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	fileName := testAudioFileWAV

	fileSpecDict, err := createFileSpecDict(xRefTable, fileName)
	if err != nil {
		return nil, err
	}

	sd, err := xRefTable.NewSoundStreamDict(fileName, 44100, fileSpecDict)
	if err != nil {
		return nil, err
	}

	err = filter.EncodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createSoundAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	indRef, err := createSoundObject(xRefTable)
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
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0.5, 0.5),
			"Sound":    *indRef,
			"Name":     types.PDFName("Speaker"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createMovieDict(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	// not supported: mp3,mp4,m4a

	fileSpecDict, err := createFileSpecDict(xRefTable, testAudioFileWAV)
	if err != nil {
		return nil, err
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"F":      *fileSpecDict,
			"Aspect": types.NewIntegerArray(200, 200),
			"Rotate": types.PDFInteger(0),
			"Poster": types.PDFBoolean(true),
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
			"Border":   types.NewIntegerArray(0, 0, 3), // rounded corners don't work
			"C":        types.NewNumberArray(0.3, 0.5, 0.5),
			"Movie":    *indRef,
			"T":        types.PDFStringLiteral("Sample Movie"),
			"A":        movieActivationDict,
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
			"Border":   types.NewIntegerArray(0, 0, 3),
			"C":        types.NewNumberArray(0.2, 0.8, 0.5),
			"A":        *mediaRenditionAction,
			"AA": types.PDFDict{
				Dict: map[string]interface{}{
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

func createWidgetAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	appearanceCharacteristicsDict := types.PDFDict{
		Dict: map[string]interface{}{
			"R":  types.PDFInteger(0),
			"BC": types.NewNumberArray(0.0, 0.0, 0.0),
			"BG": types.NewNumberArray(0.5, 0.0, 0.5),
			"RC": types.PDFStringLiteral("Rollover caption"),
			"IF": types.PDFDict{
				Dict: map[string]interface{}{
					"SW": types.PDFName("A"),
					"S":  types.PDFName("A"),
					"FB": types.PDFBoolean(true),
				},
			},
		},
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Widget"),
			"Contents": types.PDFStringLiteral("Widget Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 3),
			"C":        types.NewNumberArray(0.5, 0.5, 0.5),
			"MK":       appearanceCharacteristicsDict,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createXObjectForPrinterMark(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	buf := `0 0 m 0 25 l 25 25 l 25 0 l s`

	sd := &types.PDFStreamDict{
		PDFDict: types.PDFDict{
			Dict: map[string]interface{}{
				"Type":     types.PDFName("XObject"),
				"Subtype":  types.PDFName("Form"),
				"FormType": types.PDFInteger(1),
				"BBox":     types.NewRectangle(0, 0, 25, 25),
				"Matrix":   types.NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		},
		Content:        []byte(buf),
		FilterPipeline: []types.PDFFilter{{Name: "FlateDecode", DecodeParms: nil}}}

	sd.InsertName("Filter", "FlateDecode")

	err := filter.EncodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createPrinterMarkAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	indRef, err := createXObjectForPrinterMark(xRefTable)
	if err != nil {
		return nil, err
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("PrinterMark"),
			"Contents": types.PDFStringLiteral("PrinterMark Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 3),
			"C":        types.NewNumberArray(0.2, 0.8, 0.5),
			"F":        types.PDFInteger(0),
			"AP": types.PDFDict{
				Dict: map[string]interface{}{
					"N": *indRef,
				},
			},
			"MN": types.PDFName("ColorBar"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createXObjectForWaterMark(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	fIndRef, err := createFontDict(xRefTable)
	if err != nil {
		return nil, err
	}

	fResDict := types.NewPDFDict()
	fResDict.Insert("F1", *fIndRef)
	resourceDict := types.NewPDFDict()
	resourceDict.Insert("Font", fResDict)

	buf := `0 0 m 0 200 l 200 200 l 200 0 l s BT /F1 48 Tf 0.7 0.7 -0.7 0.7 30 10 Tm 1 Tr 2 w (Watermark) Tj ET`

	sd := &types.PDFStreamDict{
		PDFDict: types.PDFDict{
			Dict: map[string]interface{}{
				"Type":      types.PDFName("XObject"),
				"Subtype":   types.PDFName("Form"),
				"FormType":  types.PDFInteger(1),
				"BBox":      types.NewRectangle(0, 0, 200, 200),
				"Matrix":    types.NewIntegerArray(1, 0, 0, 1, 0, 0),
				"Resources": resourceDict,
			},
		},
		Content:        []byte(buf),
		FilterPipeline: []types.PDFFilter{{Name: "FlateDecode", DecodeParms: nil}}}

	sd.InsertName("Filter", "FlateDecode")

	err = filter.EncodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createWaterMarkAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	indRef, err := createXObjectForWaterMark(xRefTable)
	if err != nil {
		return nil, err
	}

	d1 := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":   types.PDFName("FixedPrint"),
			"Matrix": types.NewIntegerArray(1, 0, 0, 1, 72, -72),
			"H":      types.PDFFloat(0),
			"V":      types.PDFFloat(0),
		},
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Watermark"),
			"Contents": types.PDFStringLiteral("Watermark Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 3),
			"C":        types.NewNumberArray(0.2, 0.8, 0.5),
			"F":        types.PDFInteger(0),
			"AP": types.PDFDict{
				Dict: map[string]interface{}{
					"N": *indRef,
				},
			},
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
			"Border":   types.NewIntegerArray(0, 0, 3),
			"C":        types.NewNumberArray(0.2, 0.8, 0.5),
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

	// Create a quad points array corresponding to annot rect.
	ar := *annotRect
	qp := types.PDFArray{}
	qp = append(qp, ar[0])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[1])
	qp = append(qp, ar[2])
	qp = append(qp, ar[3])
	qp = append(qp, ar[0])
	qp = append(qp, ar[3])

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":        types.PDFName("Annot"),
			"Subtype":     types.PDFName("Redact"),
			"Contents":    types.PDFStringLiteral("Redact Annotation"),
			"Rect":        *annotRect,
			"P":           *pageIndRef,
			"Border":      types.NewIntegerArray(0, 0, 3),
			"C":           types.NewNumberArray(0.2, 0.8, 0.5),
			"F":           types.PDFInteger(0),
			"QuadPoints":  qp,
			"IC":          types.NewNumberArray(0.5, 0.0, 0.9),
			"OverlayText": types.PDFStringLiteral("An overlay"),
			"Repeat":      types.PDFBoolean(true),
			"DA":          types.PDFStringLiteral("x"),
			"Q":           types.PDFInteger(1),
		}}

	return xRefTable.IndRefForNewObject(d)
}

func createRemoteGoToAction(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":      types.PDFName("Action"),
			"S":         types.PDFName("GoToR"),
			"F":         types.PDFStringLiteral(".\\/go.pdf"),
			"D":         types.PDFArray{types.PDFInteger(0), types.PDFName("Fit")},
			"NewWindow": types.PDFBoolean(true),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationWithRemoteGoToAction(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	indRef, err := createRemoteGoToAction(xRefTable)
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
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A":        *indRef,
			"H":        types.PDFName("I"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createEmbeddedGoToAction(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	// fileSpecDict, err := createFileSpecDict(xRefTable, "testdata/go.pdf")
	// if err != nil {
	// 	return nil, err
	// }

	_, err := attach.Add(xRefTable, types.StringSet{"testdata/go.pdf": true})
	if err != nil {
		return nil, err
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type": types.PDFName("Action"),
			"S":    types.PDFName("GoToE"),
			//"F":         *fileSpecDict,
			"D":         types.PDFArray{types.PDFInteger(0), types.PDFName("Fit")},
			"NewWindow": types.PDFBoolean(true), // not honored by Acrobat Reader.
			"T": types.PDFDict{
				Dict: map[string]interface{}{
					"R": types.PDFName("C"),
					"N": types.PDFStringLiteral("testdata/go.pdf"),
				},
			},
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationWithEmbeddedGoToAction(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	indRef, err := createEmbeddedGoToAction(xRefTable)
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
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A":        *indRef,
			"H":        types.PDFName("I"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithLaunchAction(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Link"),
			"Contents": types.PDFStringLiteral("Link Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A": types.PDFDict{
				Dict: map[string]interface{}{
					"Type": types.PDFName("Action"),
					"S":    types.PDFName("Launch"),
					"F":    types.PDFStringLiteral(".\\/golang.pdf"), // e.g pdf, wav..
					"Win": types.PDFDict{
						Dict: map[string]interface{}{
							"F": types.PDFStringLiteral("golang.pdf"),
							"O": types.PDFStringLiteral("O"),
						},
					},
					"NewWindow": types.PDFBoolean(true),
				},
			},
			"H": types.PDFName("I"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithThreadAction(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Link"),
			"Contents": types.PDFStringLiteral("Link Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A": types.PDFDict{
				Dict: map[string]interface{}{
					"Type": types.PDFName("Action"),
					"S":    types.PDFName("Thread"),
					"D":    types.PDFInteger(0), // jump to first article thread
					"B":    types.PDFInteger(0), // jump to first bead
				},
			},
			"H": types.PDFName("I"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithSoundAction(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	indRef, err := createSoundObject(xRefTable)
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
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A": types.PDFDict{
				Dict: map[string]interface{}{
					"Type":        types.PDFName("Action"),
					"S":           types.PDFName("Sound"),
					"Sound":       *indRef,
					"Synchronous": types.PDFBoolean(false),
					"Repeat":      types.PDFBoolean(false),
					"Mix":         types.PDFBoolean(false),
				},
			},
			"H": types.PDFName("I"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithMovieAction(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Link"),
			"Contents": types.PDFStringLiteral("Link Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A": types.PDFDict{
				Dict: map[string]interface{}{
					"Type":      types.PDFName("Action"),
					"S":         types.PDFName("Movie"),
					"T":         types.PDFStringLiteral("Sample Movie"),
					"Operation": types.PDFName("Play"),
				},
			},
			"H": types.PDFName("I"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createLinkAnnotationDictWithHideAction(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, annotRect *types.PDFArray) (*types.PDFIndirectRef, error) {

	hideActionDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Type": types.PDFName("Action"),
			"S":    types.PDFName("Hide"),
			"H":    types.PDFBoolean(true),
		},
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Link"),
			"Contents": types.PDFStringLiteral("Link Annotation"),
			"Rect":     *annotRect,
			"P":        *pageIndRef,
			"Border":   types.NewIntegerArray(0, 0, 1),
			"C":        types.NewNumberArray(0, 0, 1),
			"A":        hideActionDict,
			"H":        types.PDFName("I"),
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
			"Border":       types.NewIntegerArray(0, 0, 3),
			"C":            types.NewNumberArray(0.2, 0.8, 0.5),
			"F":            types.PDFInteger(0),
			"LastModified": types.DateStringLiteral(time.Now()),
			"FontFauxing":  types.PDFArray{*indRef},
		},
	}

	return xRefTable.IndRefForNewObject(d)
}
