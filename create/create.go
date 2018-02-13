// Package create contains primitives for generating a PDF file.
package create

import (
	"path"
	"time"

	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/hhrutter/pdfcpu/write"
)

func createXRefTableWithRootDict() (*types.XRefTable, error) {

	xRefTable := &types.XRefTable{
		Table: map[int]*types.XRefTableEntry{},
		Stats: types.NewPDFStats(),
	}

	xRefTable.Table[0] = types.NewFreeHeadXRefTableEntry()

	one := 1
	xRefTable.Size = &one

	v := (types.V17)
	xRefTable.HeaderVersion = &v

	xRefTable.PageCount = 0

	// Optional infoDict.
	xRefTable.Info = nil

	// Additional streams not implemented.
	xRefTable.AdditionalStreams = nil

	rootDict := types.NewPDFDict()
	rootDict.InsertName("Type", "Catalog")

	indRef, err := xRefTable.IndRefForNewObject(rootDict)
	if err != nil {
		return nil, err
	}

	xRefTable.Root = indRef

	return xRefTable, nil
}

func createFontDict(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	d := types.NewPDFDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "Type1")
	d.InsertName("BaseFont", "Helvetica")

	return xRefTable.IndRefForNewObject(d)
}

func addResources(xRefTable *types.XRefTable, pageDict *types.PDFDict) error {

	fIndRef, err := createFontDict(xRefTable)
	if err != nil {
		return err
	}

	fResDict := types.NewPDFDict()
	fResDict.Insert("F1", *fIndRef)

	resourceDict := types.NewPDFDict()
	resourceDict.Insert("Font", fResDict)

	pageDict.Insert("Resources", resourceDict)

	return nil
}

func addContents(xRefTable *types.XRefTable, pageDict *types.PDFDict) error {

	contents := &types.PDFStreamDict{PDFDict: types.NewPDFDict()}
	contents.InsertName("Filter", "FlateDecode")
	contents.FilterPipeline = []types.PDFFilter{{Name: "FlateDecode", DecodeParms: nil}}

	t := `BT /F1 12 Tf 0 0 Td 0 Tr 0.5 g (lower left) Tj ET `

	t = t + "BT /F1 12 Tf 0 288 Td 1 Tr (upper left) Tj ET "
	t = t + "BT /F1 12 Tf 240 288 Td 2 Tr (upper right) Tj ET "
	t = t + "BT /F1 12 Tf 240 0 Td 0 Tr (lower right) Tj ET "
	t = t + "BT /F1 12 Tf 150 150 Td (X) Tj ET "

	contents.Content = []byte(t)

	err := filter.EncodeStream(contents)
	if err != nil {
		return err
	}

	indRef, err := xRefTable.IndRefForNewObject(*contents)
	if err != nil {
		return err
	}

	pageDict.Insert("Contents", *indRef)

	return nil
}

func createBoxColorDict() *types.PDFDict {

	cropBoxColorInfoDict := types.PDFDict{
		Dict: map[string]interface{}{
			"C": types.NewNumberArray(1.0, 1.0, 0.0),
			"W": types.PDFFloat(1.0),
			"S": types.PDFName("D"),
			"D": types.NewIntegerArray(3, 2),
		},
	}

	bleedBoxColorInfoDict := types.PDFDict{
		Dict: map[string]interface{}{
			"C": types.NewNumberArray(1.0, 0.0, 0.0),
			"W": types.PDFFloat(3.0),
			"S": types.PDFName("S"),
		},
	}

	trimBoxColorInfoDict := types.PDFDict{
		Dict: map[string]interface{}{
			"C": types.NewNumberArray(0.0, 1.0, 0.0),
			"W": types.PDFFloat(1.0),
			"S": types.PDFName("D"),
			"D": types.NewIntegerArray(3, 2),
		},
	}

	artBoxColorInfoDict := types.PDFDict{
		Dict: map[string]interface{}{
			"C": types.NewNumberArray(0.0, 0.0, 1.0),
			"W": types.PDFFloat(1.0),
			"S": types.PDFName("S"),
		},
	}

	return &types.PDFDict{
		Dict: map[string]interface{}{
			"CropBox":  cropBoxColorInfoDict,
			"BleedBox": bleedBoxColorInfoDict,
			"Trim":     trimBoxColorInfoDict,
			"ArtBox":   artBoxColorInfoDict,
		},
	}

}

func addViewportDict(pageDict *types.PDFDict) {

	measureDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":    types.PDFName("Measure"),
			"Subtype": types.PDFName("RL"),
			"R":       types.PDFStringLiteral("1in = 0.1m"),
			"X": types.PDFArray{
				types.PDFDict{
					Dict: map[string]interface{}{
						"Type": types.PDFName("NumberFormat"),
						"U":    types.PDFStringLiteral("mi"),
						"C":    types.PDFFloat(0.00139),
						"D":    types.PDFInteger(100000),
					},
				},
			},
			"D": types.PDFArray{
				types.PDFDict{
					Dict: map[string]interface{}{
						"Type": types.PDFName("NumberFormat"),
						"U":    types.PDFStringLiteral("mi"),
						"C":    types.PDFFloat(1),
					},
				},
				types.PDFDict{
					Dict: map[string]interface{}{
						"Type": types.PDFName("NumberFormat"),
						"U":    types.PDFStringLiteral("feet"),
						"C":    types.PDFFloat(5280),
					},
				},
				types.PDFDict{
					Dict: map[string]interface{}{
						"Type": types.PDFName("NumberFormat"),
						"U":    types.PDFStringLiteral("inch"),
						"C":    types.PDFFloat(12),
						"F":    types.PDFName("F"),
						"D":    types.PDFInteger(8),
					},
				},
			},
			"A": types.PDFArray{
				types.PDFDict{
					Dict: map[string]interface{}{
						"Type": types.PDFName("NumberFormat"),
						"U":    types.PDFStringLiteral("acres"),
						"C":    types.PDFFloat(640),
					},
				},
			},
			"O": types.NewIntegerArray(0, 1),
		}}

	vpDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":    types.PDFName("Viewport"),
			"BBox":    types.NewRectangle(10, 10, 60, 60),
			"Name":    types.PDFStringLiteral("viewPort"),
			"Measure": measureDict,
		},
	}

	pageDict.Insert("VP", types.PDFArray{vpDict})
}

func createLinkAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

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
			"Rect":     types.NewRectangle(80, 50, 100, 70),
			"P":        *pageIndRef,
			"A":        *indRef,
			"PA":       *indRef,
			"OC":       optionalContentGroupDict,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createPolyLineAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

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
			"Rect":     types.NewRectangle(100, 100, 200, 200),
			"P":        *pageIndRef,
			"OC":       optionalContentGroupDict,
			"Vertices": types.NewNumberArray(100, 100, 200, 200),
			"IC":       types.NewNumberArray(0.3, 0, 5, 0.0),
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

func createMarkupAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

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
			"Rect":       types.NewRectangle(100, 100, 200, 200),
			"P":          *pageIndRef,
			"OC":         optionalContentMembershipDict,
			"QuadPoints": types.NewNumberArray(100, 100, 200, 100, 200, 200, 100, 200),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createCaretAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Caret"),
			"Contents": types.PDFStringLiteral("Caret Annotation"),
			"Rect":     types.NewRectangle(100, 100, 200, 200),
			"P":        *pageIndRef,
			"RD":       types.NewRectangle(5, 5, 5, 5),
			"Sy":       types.PDFName("P"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createInkAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Ink"),
			"Contents": types.PDFStringLiteral("Ink Annotation"),
			"Rect":     types.NewRectangle(100, 100, 200, 200),
			"P":        *pageIndRef,
			"InkList": types.PDFArray{
				types.NewNumberArray(102, 102, 110, 102, 117, 117),
				types.NewNumberArray(112, 110, 113, 113),
			},
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createFileAttachmentAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

	filename := "testdata/a2.wav"

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
			"Rect":         types.NewRectangle(50, 50, 70, 70),
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

func createSoundAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

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
			"Rect":     types.NewRectangle(100, 200, 200, 300),
			"P":        *pageIndRef,
			"Sound":    *indRef,
			"Name":     types.PDFName("Speaker"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createMovieDict(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	fileName := "testdata/a2.wav"

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

func createMovieAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

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
			"Rect":     types.NewRectangle(100, 200, 200, 300),
			"P":        *pageIndRef,
			"Movie":    *indRef,
			"T":        types.PDFStringLiteral("Sample Movie"),
			"A":        movieActivationDict,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createWidgetAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

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
			"Rect":     types.NewRectangle(100, 200, 200, 300),
			"P":        *pageIndRef,
			"MK":       appearanceCharacteristicsDict,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createScreenAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("Screen"),
			"Contents": types.PDFStringLiteral("Screen Annotation"),
			"Rect":     types.NewRectangle(100, 200, 200, 300),
			"P":        *pageIndRef,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createXObject(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	// TODO use image
	sd, err := xRefTable.NewEmbeddedFileStreamDict("testdata/a2.wav")
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createPrinterMarkAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

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
			"Rect":     types.NewRectangle(100, 200, 200, 300),
			"P":        *pageIndRef,
			"F":        types.PDFInteger(0),
			//"AP":       types.PDFDict{Dict: map[string]interface{}{"N": *indRef}}, REQUIRED!!!
			"AP": types.NewPDFDict(),
			"MN": types.PDFName("ColorBar"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createWaterMarkAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

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
			"Rect":       types.NewRectangle(100, 200, 200, 300),
			"P":          *pageIndRef,
			"F":          types.PDFInteger(0),
			"FixedPrint": d1,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func create3DAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Annot"),
			"Subtype":  types.PDFName("3D"),
			"Contents": types.PDFStringLiteral("3D Annotation"),
			"Rect":     types.NewRectangle(100, 200, 200, 300),
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

func createRedactAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":        types.PDFName("Annot"),
			"Subtype":     types.PDFName("Redact"),
			"Contents":    types.PDFStringLiteral("Redact Annotation"),
			"Rect":        types.NewRectangle(100, 200, 200, 300),
			"P":           *pageIndRef,
			"F":           types.PDFInteger(0),
			"QuadPoints":  types.NewNumberArray(10, 10, 110, 10, 110, 100, 10, 100),
			"IC":          types.NewNumberArray(0.5, 0.0, 0.9),
			"OverlayText": types.PDFStringLiteral("An overlay"),
			"Repeat":      types.PDFBoolean(false),
			"DA":          types.PDFStringLiteral(""),
			"Q":           types.PDFInteger(1),
		}}

	return xRefTable.IndRefForNewObject(d)
}

func createTrapNetAnnotation(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

	indRef, err := createFontDict(xRefTable)
	if err != nil {
		return nil, err
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":         types.PDFName("Annot"),
			"Subtype":      types.PDFName("TrapNet"),
			"Contents":     types.PDFStringLiteral("TrapNet Annotation"),
			"Rect":         types.NewRectangle(100, 200, 200, 300),
			"P":            *pageIndRef,
			"F":            types.PDFInteger(0),
			"LastModified": types.DateStringLiteral(time.Now()),
			"FontFauxing":  types.PDFArray{*indRef},
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createAnnotsArray(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFArray, error) {

	arr := types.PDFArray{}

	for _, f := range []func(*types.XRefTable, *types.PDFIndirectRef) (*types.PDFIndirectRef, error){
		createLinkAnnotation,
		createPolyLineAnnotation,
		createMarkupAnnotation,
		createCaretAnnotation,
		createInkAnnotation,
		createFileAttachmentAnnotation,
		createSoundAnnotation,
		createMovieAnnotation,
		createWidgetAnnotation,
		createScreenAnnotation,
		createPrinterMarkAnnotation,
		createWaterMarkAnnotation,
		create3DAnnotation,
		createRedactAnnotation,
		createTrapNetAnnotation, // must be the last annotation for this page!
	} {
		indRef, err := f(xRefTable, pageIndRef)
		if err != nil {
			return nil, err
		}
		arr = append(arr, *indRef)
	}

	return &arr, nil
}

func createPage(xRefTable *types.XRefTable, parentPageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

	pageDict := &types.PDFDict{
		Dict: map[string]interface{}{
			"Type":         types.PDFName("Page"),
			"Parent":       *parentPageIndRef,
			"BleedBox":     types.NewRectangle(10, 10, 290, 290),
			"TrimBox":      types.NewRectangle(15, 15, 285, 285),
			"ArtBox":       types.NewRectangle(20, 20, 280, 280),
			"BoxColorInfo": *createBoxColorDict(),
			"UserUnit":     types.PDFFloat(1.5)}, // Note: not honored by Apple Preview
	}

	err := addResources(xRefTable, pageDict)
	if err != nil {
		return nil, err
	}

	err = addContents(xRefTable, pageDict)
	if err != nil {
		return nil, err
	}

	pageIndRef, err := xRefTable.IndRefForNewObject(*pageDict)
	if err != nil {
		return nil, err
	}

	// Fake SeparationInfo related to a single page only.
	separationInfoDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Pages":          types.PDFArray{*pageIndRef},
			"DeviceColorant": types.PDFName("Cyan"),
			"ColorSpace": types.PDFArray{
				types.PDFName("Separation"),
				types.PDFName("Green"),
				types.PDFName("DeviceCMYK"),
				types.PDFDict{
					Dict: map[string]interface{}{
						"FunctionType": types.PDFInteger(2),
						"Domain":       types.NewNumberArray(0.0, 1.0),
						"C0":           types.NewNumberArray(0.0),
						"C1":           types.NewNumberArray(1.0),
						"N":            types.PDFFloat(1),
					},
				},
			},
		},
	}
	pageDict.Insert("SeparationInfo", separationInfoDict)

	annotsArray, err := createAnnotsArray(xRefTable, pageIndRef)
	if err != nil {
		return nil, err
	}
	pageDict.Insert("Annots", *annotsArray)

	addViewportDict(pageDict)

	return pageIndRef, nil
}

func addPages(xRefTable *types.XRefTable, rootDict *types.PDFDict) (*types.PDFIndirectRef, error) {

	// mediabox = physical page dimensions
	r := types.NewRectangle(0, 0, 300, 300)

	pagesDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Pages"),
			"Count":    types.PDFInteger(1),
			"MediaBox": r,
			"CropBox":  r,
		},
	}

	parentPageIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return nil, err
	}

	pageIndRef, err := createPage(xRefTable, parentPageIndRef)
	if err != nil {
		return nil, err
	}

	pagesDict.Insert("Kids", types.PDFArray{*pageIndRef})

	rootDict.Insert("Pages", *parentPageIndRef)

	return pageIndRef, nil
}

// create a thread with 2 beads.
func createThreadDict(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef) (*types.PDFIndirectRef, error) {

	infoDict := types.NewPDFDict()
	infoDict.InsertString("Title", "DummyArticle")

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type": types.PDFName("Thread"),
			"I":    infoDict,
		},
	}

	dIndRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	// create first bead
	d1 := types.PDFDict{
		Dict: map[string]interface{}{
			"Type": types.PDFName("Bead"),
			"T":    *dIndRef,
			"P":    *pageIndRef,
			"R":    types.NewRectangle(0, 0, 100, 100),
		},
	}

	d1IndRef, err := xRefTable.IndRefForNewObject(d1)
	if err != nil {
		return nil, err
	}

	d.Insert("F", *d1IndRef)

	// create last bead
	d2 := types.PDFDict{
		Dict: map[string]interface{}{
			"Type": types.PDFName("Bead"),
			"T":    *dIndRef,
			"N":    *d1IndRef,
			"V":    *d1IndRef,
			"P":    *pageIndRef,
			"R":    types.NewRectangle(0, 100, 100, 200),
		},
	}

	d2IndRef, err := xRefTable.IndRefForNewObject(d2)
	if err != nil {
		return nil, err
	}

	d1.Insert("N", *d2IndRef)
	d1.Insert("V", *d2IndRef)

	return dIndRef, nil
}

func addThreads(xRefTable *types.XRefTable, rootDict *types.PDFDict, pageIndRef *types.PDFIndirectRef) error {

	indRef, err := createThreadDict(xRefTable, pageIndRef)
	if err != nil {
		return err
	}

	indRef, err = xRefTable.IndRefForNewObject(types.PDFArray{*indRef})
	if err != nil {
		return err
	}

	rootDict.Insert("Threads", *indRef)

	return nil
}

func addOpenAction(xRefTable *types.XRefTable, rootDict *types.PDFDict) error {

	script := `var a = this.getAnnot(0, 'SoundFileAttachmentAnnot');
	app.alert('Hello Gopher! AnnotType=' + a.type + ' ' + a.doc.URL);
	var annot = this.addAnnot({ page: 0,
		type: "Stamp",
		author: "A. C. Robat",
		name: "myStamp",
		rect: [100, 100, 130, 130],
		contents: "Try it again, this time with order and method!", AP: "NotApproved" });`

	d := types.NewPDFDict()
	d.InsertName("Type", "Action")
	d.InsertName("S", "JavaScript")
	d.InsertString("JS", script)

	rootDict.Insert("OpenAction", d)

	return nil
}

func addURI(xRefTable *types.XRefTable, rootDict *types.PDFDict) {

	d := types.NewPDFDict()
	d.InsertString("Base", "http://www.adobe.com")

	rootDict.Insert("URI", d)
}

func addSpiderInfo(xRefTable *types.XRefTable, rootDict *types.PDFDict) error {

	// webCaptureInfoDict
	webCaptureInfoDict := types.NewPDFDict()
	webCaptureInfoDict.InsertInt("V", 1.0)

	arr := types.PDFArray{}
	captureCmdDict := types.NewPDFDict()
	captureCmdDict.InsertString("URL", (""))

	cmdSettingsDict := types.NewPDFDict()
	captureCmdDict.Insert("S", cmdSettingsDict)

	indRef, err := xRefTable.IndRefForNewObject(captureCmdDict)
	if err != nil {
		return err
	}

	arr = append(arr, *indRef)

	webCaptureInfoDict.Insert("C", arr)

	indRef, err = xRefTable.IndRefForNewObject(webCaptureInfoDict)
	if err != nil {
		return err
	}

	rootDict.Insert("SpiderInfo", *indRef)

	return nil
}

func addOCProperties(xRefTable *types.XRefTable, rootDict *types.PDFDict) error {

	usageAppDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Event":    types.PDFName("View"),
			"OCGs":     types.PDFArray{}, // of indRefs
			"Category": types.NewNameArray("Language"),
		},
	}

	optionalContentConfigDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Name":      types.PDFStringLiteral("OCConf"),
			"Creator":   types.PDFStringLiteral("Horst Rutter"),
			"BaseState": types.PDFName("ON"),
			"OFF":       types.PDFArray{},
			"Intent":    types.PDFName("Design"),
			"AS":        types.PDFArray{usageAppDict},
			"Order":     types.PDFArray{},
			"ListMode":  types.PDFName("AllPages"),
			"RBGroups":  types.PDFArray{},
			"Locked":    types.PDFArray{},
		},
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"OCGs":    types.PDFArray{}, // of indRefs
			"D":       optionalContentConfigDict,
			"Configs": types.PDFArray{optionalContentConfigDict},
		},
	}

	rootDict.Insert("OCProperties", d)

	return nil
}

func addRequirements(xRefTable *types.XRefTable, rootDict *types.PDFDict) {

	d := types.NewPDFDict()
	d.InsertName("Type", "Requirement")
	d.InsertName("S", "EnableJavaScripts")

	rootDict.Insert("Requirements", types.PDFArray{d})
}

// DemoXRef creates a demoXRef for testing validation.
func DemoXRef() (*types.XRefTable, error) {

	xRefTable, err := createXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	pageIndRef, err := addPages(xRefTable, rootDict)
	if err != nil {
		return nil, err
	}

	err = addThreads(xRefTable, rootDict, pageIndRef)
	if err != nil {
		return nil, err
	}

	err = addOpenAction(xRefTable, rootDict)
	if err != nil {
		return nil, err
	}

	addURI(xRefTable, rootDict)

	err = addSpiderInfo(xRefTable, rootDict)
	if err != nil {
		return nil, err
	}

	err = addOCProperties(xRefTable, rootDict)
	if err != nil {
		return nil, err
	}

	addRequirements(xRefTable, rootDict)

	return xRefTable, nil
}

// DemoPDF creates a demo PDF file for testing validation.
func DemoPDF(xRefTable *types.XRefTable, dirName, fileName string) error {

	config := types.NewDefaultConfiguration()

	ctx := &types.PDFContext{
		Configuration: config,
		XRefTable:     xRefTable,
		Write:         types.NewWriteContext(config.Eol),
	}

	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	return write.PDFFile(ctx)
}
