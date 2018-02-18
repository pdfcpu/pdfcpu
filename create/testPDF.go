// Package create contains primitives for generating a PDF file.
package create

import (
	"fmt"

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

	// 595.27, 841.89)

	// use buffer
	t := `BT /F1 12 Tf 0 1 Td 0 Tr 0.5 g (lower left) Tj ET `
	t += "BT /F1 12 Tf 0 832 Td 1 Tr (upper left) Tj ET "
	t += "BT /F1 12 Tf 537 832 Td 2 Tr (upper right) Tj ET "
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

	// d..distance between rectangles
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

	r := types.NewRectangle(llx, lly, urx, ury)

	fmt.Printf("annotRect(%d) = %v\n", i, r)

	return r
}

func createAnnotsArray(xRefTable *types.XRefTable, pageIndRef *types.PDFIndirectRef, mediaBox *types.PDFArray) (*types.PDFArray, error) {

	// Generate side by side lined up annotations starting in the lower left corner of the page.
	//

	pageWidth := (*mediaBox)[2].(types.PDFFloat)
	pageHeight := (*mediaBox)[3].(types.PDFFloat)
	fmt.Printf("w=%3.2f h=%3.2f\n", pageWidth, pageHeight)

	arr := types.PDFArray{}

	for i, f := range []func(*types.XRefTable, *types.PDFIndirectRef, *types.PDFArray) (*types.PDFIndirectRef, error){
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
		r := annotRect(i, pageWidth.Value(), pageHeight.Value(), 30, 80)
		indRef, err := f(xRefTable, pageIndRef, &r)
		if err != nil {
			return nil, err
		}
		arr = append(arr, *indRef)
	}

	return &arr, nil
}

func createPage(xRefTable *types.XRefTable, parentPageIndRef *types.PDFIndirectRef, mediaBox *types.PDFArray) (*types.PDFIndirectRef, error) {

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

func addPages(xRefTable *types.XRefTable, rootDict *types.PDFDict) (*types.PDFIndirectRef, error) {

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

	pageIndRef, err := createPage(xRefTable, parentPageIndRef, &mediaBox)
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
	_ = script

	d := types.NewPDFDict()
	d.InsertName("Type", "Action")
	//d.InsertName("S", "JavaScript")
	d.InsertName("S", "Movie")
	d.InsertString("T", "Sample Movie")
	//d.InsertString("JS", script)

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
