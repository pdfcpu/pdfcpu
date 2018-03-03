// Package create contains primitives for generating a PDF file.
package create

import (
	"bytes"
	"fmt"

	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/hhrutter/pdfcpu/write"
)

const testAudioFileWAV = "testdata/test.wav"

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
	//d.InsertName("Name", "Helvetica")
	d.InsertName("Subtype", "Type1")
	d.InsertName("BaseFont", "Helvetica")

	return xRefTable.IndRefForNewObject(d)
}

func createZapfDingbatsFontDict(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	d := types.NewPDFDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "Type1")
	d.InsertName("BaseFont", "ZapfDingbats")

	return xRefTable.IndRefForNewObject(d)
}

func createFunctionalShadingDict(xRefTable *types.XRefTable) *types.PDFDict {

	f := types.PDFDict{
		Dict: map[string]interface{}{
			"FunctionType": types.PDFInteger(2),
			"Domain":       types.NewNumberArray(1.0, 1.2, 1.4, 1.6, 1.8, 2.0),
			"N":            types.PDFFloat(1),
		},
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"ShadingType": types.PDFInteger(1),
			"Function":    types.PDFArray{f},
		},
	}

	return &d
}

func createRadialShadingDict(xRefTable *types.XRefTable) *types.PDFDict {

	f := types.PDFDict{
		Dict: map[string]interface{}{
			"FunctionType": types.PDFInteger(2),
			"Domain":       types.NewNumberArray(1.0, 1.2, 1.4, 1.6, 1.8, 2.0),
			"N":            types.PDFFloat(1),
		},
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"ShadingType": types.PDFInteger(3),
			"Coords":      types.NewNumberArray(0, 0, 50, 10, 10, 100),
			"Function":    types.PDFArray{f},
		},
	}

	return &d
}

func addResources(xRefTable *types.XRefTable, pageDict *types.PDFDict) error {

	fIndRef, err := createFontDict(xRefTable)
	if err != nil {
		return err
	}

	functionalBasedShDict := createFunctionalShadingDict(xRefTable)

	radialShDict := createRadialShadingDict(xRefTable)

	resourceDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Font": types.PDFDict{
				Dict: map[string]interface{}{
					"F1": *fIndRef,
				},
			},
			"Shading": types.PDFDict{
				Dict: map[string]interface{}{
					"S1": *functionalBasedShDict,
					"S3": *radialShDict,
				},
			},
		},
	}

	pageDict.Insert("Resources", resourceDict)

	return nil
}

func addContents(xRefTable *types.XRefTable, pageDict *types.PDFDict) error {

	contents := &types.PDFStreamDict{PDFDict: types.NewPDFDict()}
	contents.InsertName("Filter", "FlateDecode")
	contents.FilterPipeline = []types.PDFFilter{{Name: "FlateDecode", DecodeParms: nil}}

	// Page dimensions: 595.27, 841.89

	// TODO use buffer
	t := `BT /F1 12 Tf 0 1 Td 0 Tr 0.5 g (lower left) Tj ET `
	t += "BT /F1 12 Tf 0 832 Td 0 Tr (upper left) Tj ET "
	t += "BT /F1 12 Tf 537 832 Td 0 Tr (upper right) Tj ET "
	t += "BT /F1 12 Tf 540 1 Td 0 Tr (lower right) Tj ET "
	t += "BT /F1 12 Tf 297.55 420.5 Td (X) Tj ET "

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

func annotRect(i int, w, h, d, l float64) types.PDFArray {

	// d..distance between annotation rectangles
	// l..side length of rectangle

	// max number of rectangles fitting into w
	xmax := int((w - d) / (l + d))

	// max number of rectangles fitting into h
	ymax := int((h - d) / (l + d))

	col := float64(i % xmax)
	row := float64(i / xmax % ymax)

	llx := d + col*(l+d)
	lly := d + row*(l+d)

	urx := llx + l
	ury := lly + l

	return types.NewRectangle(llx, lly, urx, ury)
}

func createAnnotsArray(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, mediaBox *types.PDFArray) (*types.PDFArray, error) {

	// Generate side by side lined up annotations starting in the lower left corner of the page.

	pageWidth := (*mediaBox)[2].(types.PDFFloat)
	pageHeight := (*mediaBox)[3].(types.PDFFloat)

	arr := types.PDFArray{}

	for i, f := range []func(*types.XRefTable, *types.PDFIndirectRef, *types.PDFArray) (*types.PDFIndirectRef, error){
		createTextAnnotation,
		createLinkAnnotation,
		createFreeTextAnnotation,
		createLineAnnotation,
		createSquareAnnotation,
		createCircleAnnotation,
		createPolygonAnnotation,
		createPolyLineAnnotation,
		createHighlightAnnotation,
		createUnderlineAnnotation,
		createSquigglyAnnotation,
		createStrikeOutAnnotation,
		createCaretAnnotation,
		createStampAnnotation,
		createInkAnnotation,
		createPopupAnnotation,
		createFileAttachmentAnnotation,
		createSoundAnnotation,
		createMovieAnnotation,
		createScreenAnnotation,
		createWidgetAnnotation,
		createPrinterMarkAnnotation,
		createWaterMarkAnnotation,
		create3DAnnotation,
		createRedactAnnotation,
		createLinkAnnotationWithRemoteGoToAction,
		createLinkAnnotationWithEmbeddedGoToAction,
		createLinkAnnotationDictWithLaunchAction,
		createLinkAnnotationDictWithThreadAction,
		createLinkAnnotationDictWithSoundAction,
		createLinkAnnotationDictWithMovieAction,
		createLinkAnnotationDictWithHideAction,
		createTrapNetAnnotation, // must be the last annotation for this page!
	} {
		r := annotRect(i, pageWidth.Value(), pageHeight.Value(), 30, 80)

		indRef, err := f(xRefTable, pageIndRef, &r)
		if err != nil {
			return nil, err
		}

		arr = append(arr, *indRef)
	}

	return &arr, nil
}

func createPageWithAnnotations(xRefTable *types.XRefTable, parentPageIndRef *types.PDFIndirectRef, mediaBox *types.PDFArray) (*types.PDFIndirectRef, error) {

	pageDict := &types.PDFDict{
		Dict: map[string]interface{}{
			"Type":         types.PDFName("Page"),
			"Parent":       *parentPageIndRef,
			"BleedBox":     *mediaBox,
			"TrimBox":      *mediaBox,
			"ArtBox":       *mediaBox,
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

	annotsArray, err := createAnnotsArray(xRefTable, pageIndRef, mediaBox)
	if err != nil {
		return nil, err
	}
	pageDict.Insert("Annots", *annotsArray)

	addViewportDict(pageDict)

	return pageIndRef, nil
}

func createPageWithAcroForm(xRefTable *types.XRefTable, parentPageIndRef *types.PDFIndirectRef, annotsArray *types.PDFArray, mediaBox *types.PDFArray) (*types.PDFIndirectRef, error) {

	pageDict := &types.PDFDict{
		Dict: map[string]interface{}{
			"Type":         types.PDFName("Page"),
			"Parent":       *parentPageIndRef,
			"BleedBox":     *mediaBox,
			"TrimBox":      *mediaBox,
			"ArtBox":       *mediaBox,
			"BoxColorInfo": *createBoxColorDict(),
			"UserUnit":     types.PDFFloat(1.0), // Note: not honored by Apple Preview
		},
	}

	err := addResources(xRefTable, pageDict)
	if err != nil {
		return nil, err
	}

	err = addContents(xRefTable, pageDict)
	if err != nil {
		return nil, err
	}

	pageDict.Insert("Annots", *annotsArray)

	return xRefTable.IndRefForNewObject(*pageDict)
}

func addPageTreeWithAnnotations(xRefTable *types.XRefTable, rootDict *types.PDFDict) (*types.PDFIndirectRef, error) {

	// mediabox = physical page dimensions
	mediaBox := types.NewRectangle(0, 0, 595.27, 841.89)

	pagesDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Pages"),
			"Count":    types.PDFInteger(1),
			"MediaBox": mediaBox,
			"CropBox":  mediaBox,
		},
	}

	parentPageIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return nil, err
	}

	pageIndRef, err := createPageWithAnnotations(xRefTable, parentPageIndRef, &mediaBox)
	if err != nil {
		return nil, err
	}

	pagesDict.Insert("Kids", types.PDFArray{*pageIndRef})

	rootDict.Insert("Pages", *parentPageIndRef)

	return pageIndRef, nil
}

func addPageTreeWithAcroFields(xRefTable *types.XRefTable, rootDict *types.PDFDict, annotsArray *types.PDFArray) (*types.PDFIndirectRef, error) {

	// mediabox = physical page dimensions
	mediaBox := types.NewRectangle(0, 0, 595.27, 841.89)

	pagesDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Type":     types.PDFName("Pages"),
			"Count":    types.PDFInteger(1),
			"MediaBox": mediaBox,
			"CropBox":  mediaBox,
		},
	}

	parentPageIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return nil, err
	}

	pageIndRef, err := createPageWithAcroForm(xRefTable, parentPageIndRef, annotsArray, &mediaBox)
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
			"R":    types.NewRectangle(0, 100, 200, 100),
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

	nextActionDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Type": types.PDFName("Action"),
			"S":    types.PDFName("Movie"),
			"T":    types.PDFStringLiteral("Sample Movie"),
		},
	}

	script := `app.alert('Hello Gopher!');`

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Type": types.PDFName("Action"),
			"S":    types.PDFName("JavaScript"),
			"JS":   types.PDFStringLiteral(script),
			"Next": nextActionDict,
		},
	}

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

// AnnotationDemoXRef creates a PDF file with examples of annotations and actions.
func AnnotationDemoXRef() (*types.XRefTable, error) {

	xRefTable, err := createXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	pageIndRef, err := addPageTreeWithAnnotations(xRefTable, rootDict)
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

func setBit(i uint32, pos uint) uint32 {

	// pos 1 == bit 0

	var mask uint32 = 1

	mask <<= pos - 1

	i |= mask

	return i
}

func createButtonFieldDict(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	var flags uint32

	flags = setBit(flags, 17)

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"FT": types.PDFName("Btn"),
			"Ff": types.PDFInteger(flags),
			//"T":    types.PDFStringLiteral("Push"),
			"Rect": types.NewRectangle(100, 100, 150, 150),
			"H":    types.PDFName("I"),
			//"Border":  types.NewIntegerArray(0, 0, 3),
			//"C":       types.NewNumberArray(0.5, 0.5, 0.5),
			"Type":    types.PDFName("Annot"),
			"Subtype": types.PDFName("Widget"),
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createNormalAppearanceForFormField(xRefTable *types.XRefTable, w, h float64) (*types.PDFIndirectRef, error) {

	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &types.PDFStreamDict{
		PDFDict: types.PDFDict{
			Dict: map[string]interface{}{
				"Type":     types.PDFName("XObject"),
				"Subtype":  types.PDFName("Form"),
				"FormType": types.PDFInteger(1),
				"BBox":     types.NewRectangle(0, 0, w, h),
				"Matrix":   types.NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		},
		Content: b.Bytes(),
		//FilterPipeline: []types.PDFFilter{{Name: "FlateDecode", DecodeParms: nil}},
	}

	//sd.InsertName("Filter", "FlateDecode")

	err := filter.EncodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createRolloverAppearanceForFormField(xRefTable *types.XRefTable, w, h float64) (*types.PDFIndirectRef, error) {

	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "1 0 0 RG 0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &types.PDFStreamDict{
		PDFDict: types.PDFDict{
			Dict: map[string]interface{}{
				"Type":     types.PDFName("XObject"),
				"Subtype":  types.PDFName("Form"),
				"FormType": types.PDFInteger(1),
				"BBox":     types.NewRectangle(0, 0, w, h),
				"Matrix":   types.NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		},
		Content: b.Bytes(),
		//FilterPipeline: []types.PDFFilter{{Name: "FlateDecode", DecodeParms: nil}},
	}

	//sd.InsertName("Filter", "FlateDecode")

	err := filter.EncodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createDownAppearanceForFormField(xRefTable *types.XRefTable, w, h float64) (*types.PDFIndirectRef, error) {

	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &types.PDFStreamDict{
		PDFDict: types.PDFDict{
			Dict: map[string]interface{}{
				"Type":     types.PDFName("XObject"),
				"Subtype":  types.PDFName("Form"),
				"FormType": types.PDFInteger(1),
				"BBox":     types.NewRectangle(0, 0, w, h),
				"Matrix":   types.NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		},
		Content: b.Bytes(),
		//FilterPipeline: []types.PDFFilter{{Name: "FlateDecode", DecodeParms: nil}},
	}

	//sd.InsertName("Filter", "FlateDecode")

	err := filter.EncodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createTextField(xRefTable *types.XRefTable, pageAnnots *types.PDFArray) (*types.PDFIndirectRef, error) {

	// lower left corner
	x := 100.0
	y := 300.0

	// width
	w := 130.0

	// height
	h := 20.0

	fN, err := createNormalAppearanceForFormField(xRefTable, w, h)
	if err != nil {
		return nil, err
	}

	fR, err := createRolloverAppearanceForFormField(xRefTable, w, h)
	if err != nil {
		return nil, err
	}

	fD, err := createDownAppearanceForFormField(xRefTable, w, h)
	if err != nil {
		return nil, err
	}

	fontDict, err := createFontDict(xRefTable)
	if err != nil {
		return nil, err
	}

	resourceDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Font": types.PDFDict{
				Dict: map[string]interface{}{
					"Helvetica": *fontDict,
				},
			},
		},
	}

	// var flags uint32
	// flags = setBit(flags, 24)

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"AP": types.PDFDict{
				Dict: map[string]interface{}{
					"N": *fN,
					"R": *fR,
					"D": *fD,
				},
			},
			"DA":      types.PDFStringLiteral("/Helvetica 12 Tf 0 g"),
			"DR":      resourceDict,
			"FT":      types.PDFName("Tx"),
			"Rect":    types.NewRectangle(x, y, x+w, y+h),
			"Border":  types.NewIntegerArray(0, 0, 1),
			"Type":    types.PDFName("Annot"),
			"Subtype": types.PDFName("Widget"),
			"T":       types.PDFStringLiteral("Gopher"),
			"TU":      types.PDFStringLiteral("Gopher"),
			"V":       types.PDFStringLiteral("Inputfield default value"),
		},
	}

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	*pageAnnots = append(*pageAnnots, *indRef)

	return indRef, nil
}

func createYesAppearance(xRefTable *types.XRefTable, resourceDict *types.PDFDict, w, h float64) (*types.PDFIndirectRef, error) {

	var b bytes.Buffer
	fmt.Fprintf(&b, "q 0 0 1 rg BT /ZaDb 12 Tf 0 0 Td (8) Tj ET Q")

	sd := &types.PDFStreamDict{
		PDFDict: types.PDFDict{
			Dict: map[string]interface{}{
				"Resources": *resourceDict,
				"Subtype":   types.PDFName("Form"),
				"BBox":      types.NewRectangle(0, 0, w, h),
				"OPI": types.PDFDict{
					Dict: map[string]interface{}{
						"2.0": types.PDFDict{
							Dict: map[string]interface{}{
								"Type":    types.PDFName("OPI"),
								"Version": types.PDFFloat(2.0),
								"F":       types.PDFStringLiteral("Proxy"),
								"Inks":    types.PDFName("full_color"),
							},
						},
					},
				},
				"Ref": types.PDFDict{
					Dict: map[string]interface{}{
						"F":    types.PDFStringLiteral("Proxy"),
						"Page": types.PDFInteger(1),
					},
				},
			},
		},
		Content: b.Bytes(),
	}

	err := filter.EncodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createOffAppearance(xRefTable *types.XRefTable, resourceDict *types.PDFDict, w, h float64) (*types.PDFIndirectRef, error) {

	var b bytes.Buffer
	fmt.Fprintf(&b, "q 0 0 1 rg BT /ZaDb 12 Tf 0 0 Td (4) Tj ET Q")

	sd := &types.PDFStreamDict{
		PDFDict: types.PDFDict{
			Dict: map[string]interface{}{
				"Resources": *resourceDict,
				"Subtype":   types.PDFName("Form"),
				"BBox":      types.NewRectangle(0, 0, w, h),
				"OPI": types.PDFDict{
					Dict: map[string]interface{}{
						"1.3": types.PDFDict{
							Dict: map[string]interface{}{
								"Type":     types.PDFName("OPI"),
								"Version":  types.PDFFloat(1.3),
								"F":        types.PDFStringLiteral("Proxy"),
								"Size":     types.NewIntegerArray(400, 400),
								"CropRect": types.NewIntegerArray(0, 400, 400, 0),
								"Position": types.NewNumberArray(0, 0, 0, 400, 400, 400, 400, 0),
							},
						},
					},
				},
			},
		},
		Content: b.Bytes(),
	}

	err := filter.EncodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createCheckBoxButtonField(xRefTable *types.XRefTable, pageAnnots *types.PDFArray) (*types.PDFIndirectRef, error) {

	fontDict, err := createZapfDingbatsFontDict(xRefTable)
	if err != nil {
		return nil, err
	}

	resDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Font": types.PDFDict{
				Dict: map[string]interface{}{
					"ZaDb": *fontDict,
				},
			},
		},
	}

	yesForm, err := createYesAppearance(xRefTable, &resDict, 20.0, 20.0)
	if err != nil {
		return nil, err
	}

	offForm, err := createOffAppearance(xRefTable, &resDict, 20.0, 20.0)
	if err != nil {
		return nil, err
	}

	apDict := types.PDFDict{
		Dict: map[string]interface{}{
			"N": types.PDFDict{
				Dict: map[string]interface{}{
					"Yes": *yesForm,
					"Off": *offForm,
				},
			},
		},
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"FT":      types.PDFName("Btn"),
			"Rect":    types.NewRectangle(250, 300, 270, 320),
			"Type":    types.PDFName("Annot"),
			"Subtype": types.PDFName("Widget"),
			"T":       types.PDFStringLiteral("Urgent"),
			"V":       types.PDFName("Yes"),
			"AS":      types.PDFName("Yes"),
			"AP":      apDict,
		},
	}

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	*pageAnnots = append(*pageAnnots, *indRef)

	return indRef, nil
}

func createRadioButtonField(xRefTable *types.XRefTable, pageAnnots *types.PDFArray) (*types.PDFIndirectRef, error) {

	var flags uint32
	flags = setBit(flags, 16)

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"FT":   types.PDFName("Btn"),
			"Ff":   types.PDFInteger(flags),
			"Rect": types.NewRectangle(250, 400, 280, 420),
			//"Type":    types.PDFName("Annot"),
			//"Subtype": types.PDFName("Widget"),
			"T": types.PDFStringLiteral("Credit card"),
			"V": types.PDFName("card1"),
		},
	}

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	fontDict, err := createZapfDingbatsFontDict(xRefTable)
	if err != nil {
		return nil, err
	}

	resDict := types.PDFDict{
		Dict: map[string]interface{}{
			"Font": types.PDFDict{
				Dict: map[string]interface{}{
					"ZaDb": *fontDict,
				},
			},
		},
	}

	selectedForm, err := createYesAppearance(xRefTable, &resDict, 20.0, 20.0)
	if err != nil {
		return nil, err
	}

	offForm, err := createOffAppearance(xRefTable, &resDict, 20.0, 20.0)
	if err != nil {
		return nil, err
	}

	r1 := types.PDFDict{
		Dict: map[string]interface{}{
			"Rect":    types.NewRectangle(250, 400, 280, 420),
			"Type":    types.PDFName("Annot"),
			"Subtype": types.PDFName("Widget"),
			"Parent":  *indRef,
			"AS":      types.PDFName("card1"),
			"AP": types.PDFDict{
				Dict: map[string]interface{}{
					"N": types.PDFDict{
						Dict: map[string]interface{}{
							"card1": *selectedForm,
							"Off":   *offForm,
						},
					},
				},
			},
		},
	}

	indRefR1, err := xRefTable.IndRefForNewObject(r1)
	if err != nil {
		return nil, err
	}

	r2 := types.PDFDict{
		Dict: map[string]interface{}{
			"Rect":    types.NewRectangle(300, 400, 330, 420),
			"Type":    types.PDFName("Annot"),
			"Subtype": types.PDFName("Widget"),
			"Parent":  *indRef,
			"AS":      types.PDFName("Off"),
			"AP": types.PDFDict{
				Dict: map[string]interface{}{
					"N": types.PDFDict{
						Dict: map[string]interface{}{
							"card2": *selectedForm,
							"Off":   *offForm,
						},
					},
				},
			},
		},
	}

	indRefR2, err := xRefTable.IndRefForNewObject(r2)
	if err != nil {
		return nil, err
	}

	d.Insert("Kids", types.PDFArray{*indRefR1, *indRefR2})

	*pageAnnots = append(*pageAnnots, *indRefR1)
	*pageAnnots = append(*pageAnnots, *indRefR2)

	return indRef, nil
}

func streamObjForFXAElement(xRefTable *types.XRefTable, s string) (*types.PDFIndirectRef, error) {

	sd := &types.PDFStreamDict{
		PDFDict: types.PDFDict{
			Dict: map[string]interface{}{},
		},
		Content: []byte(s),
	}

	err := filter.EncodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createXFAArray(xRefTable *types.XRefTable) (*types.PDFArray, error) {

	sd1, err := streamObjForFXAElement(xRefTable, "<xdp:xdp xmlns:xdp=\"http://ns.adobe.com/xdp/\">")
	if err != nil {
		return nil, err
	}

	sd3, err := streamObjForFXAElement(xRefTable, "</xdp:xdp>")
	if err != nil {
		return nil, err
	}

	return &types.PDFArray{
		types.PDFStringLiteral("xdp:xdp"), *sd1,
		types.PDFStringLiteral("/xdp:xdp"), *sd3,
	}, nil
}

func createAcroFormDict(xRefTable *types.XRefTable) (*types.PDFDict, *types.PDFArray, error) {

	pageAnnots := types.PDFArray{}

	text, err := createTextField(xRefTable, &pageAnnots)
	if err != nil {
		return nil, nil, err
	}

	checkBox, err := createCheckBoxButtonField(xRefTable, &pageAnnots)
	if err != nil {
		return nil, nil, err
	}

	radioButton, err := createRadioButtonField(xRefTable, &pageAnnots)
	if err != nil {
		return nil, nil, err
	}

	xfaArr, err := createXFAArray(xRefTable)
	if err != nil {
		return nil, nil, err
	}

	d := types.PDFDict{
		Dict: map[string]interface{}{
			"Fields":          types.PDFArray{*text, *checkBox, *radioButton}, // indRefs of fieldDicts
			"NeedAppearances": types.PDFBoolean(true),
			"CO":              types.PDFArray{*text},
			"XFA":             *xfaArr,
		},
	}

	return &d, &pageAnnots, nil
}

// AcroFormDemoXRef creates a PDF file with an AcroForm example.
func AcroFormDemoXRef() (*types.XRefTable, error) {

	xRefTable, err := createXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	acroFormDict, annotsArray, err := createAcroFormDict(xRefTable)
	if err != nil {
		return nil, err
	}

	rootDict.Insert("AcroForm", *acroFormDict)

	_, err = addPageTreeWithAcroFields(xRefTable, rootDict, annotsArray)
	if err != nil {
		return nil, err
	}

	rootDict.Insert("ViewerPreferences",
		types.PDFDict{
			Dict: map[string]interface{}{
				"FitWindow":    types.PDFBoolean(true),
				"CenterWindow": types.PDFBoolean(true),
			},
		},
	)

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
