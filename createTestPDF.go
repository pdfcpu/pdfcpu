package pdfcpu

// Functions needed to create a test.pdf that gets used for validation testing (see process_test.go)

import (
	"bytes"
	"fmt"
)

const testAudioFileWAV = "testdata/test.wav"

func createXRefTableWithRootDict() (*XRefTable, error) {

	xRefTable := &XRefTable{
		Table: map[int]*XRefTableEntry{},
		Names: map[string]*Node{},
		Stats: NewPDFStats(),
	}

	xRefTable.Table[0] = NewFreeHeadXRefTableEntry()

	one := 1
	xRefTable.Size = &one

	v := (V17)
	xRefTable.HeaderVersion = &v

	xRefTable.PageCount = 0

	// Optional infoDict.
	xRefTable.Info = nil

	// Additional streams not implemented.
	xRefTable.AdditionalStreams = nil

	rootDict := NewPDFDict()
	rootDict.InsertName("Type", "Catalog")

	indRef, err := xRefTable.IndRefForNewObject(rootDict)
	if err != nil {
		return nil, err
	}

	xRefTable.Root = indRef

	return xRefTable, nil
}

func createFontDict(xRefTable *XRefTable) (*PDFIndirectRef, error) {

	d := NewPDFDict()
	d.InsertName("Type", "Font")
	//d.InsertName("Name", "Helvetica")
	d.InsertName("Subtype", "Type1")
	d.InsertName("BaseFont", "Helvetica")

	return xRefTable.IndRefForNewObject(d)
}

func createZapfDingbatsFontDict(xRefTable *XRefTable) (*PDFIndirectRef, error) {

	d := NewPDFDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "Type1")
	d.InsertName("BaseFont", "ZapfDingbats")

	return xRefTable.IndRefForNewObject(d)
}

func createFunctionalShadingDict(xRefTable *XRefTable) *PDFDict {

	f := PDFDict{
		Dict: map[string]PDFObject{
			"FunctionType": PDFInteger(2),
			"Domain":       NewNumberArray(1.0, 1.2, 1.4, 1.6, 1.8, 2.0),
			"N":            PDFFloat(1),
		},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"ShadingType": PDFInteger(1),
			"Function":    PDFArray{f},
		},
	}

	return &d
}

func createRadialShadingDict(xRefTable *XRefTable) *PDFDict {

	f := PDFDict{
		Dict: map[string]PDFObject{
			"FunctionType": PDFInteger(2),
			"Domain":       NewNumberArray(1.0, 1.2, 1.4, 1.6, 1.8, 2.0),
			"N":            PDFFloat(1),
		},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"ShadingType": PDFInteger(3),
			"Coords":      NewNumberArray(0, 0, 50, 10, 10, 100),
			"Function":    PDFArray{f},
		},
	}

	return &d
}

func createStreamObjForHalftoneDictType6(xRefTable *XRefTable) (*PDFIndirectRef, error) {

	sd := &PDFStreamDict{
		PDFDict: PDFDict{
			Dict: map[string]PDFObject{
				"Type":             PDFName("Halftone"),
				"HalftoneType":     PDFInteger(6),
				"Width":            PDFInteger(100),
				"Height":           PDFInteger(100),
				"TransferFunction": PDFName("Identity"),
			},
		},
		Content: []byte{},
	}

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createStreamObjForHalftoneDictType10(xRefTable *XRefTable) (*PDFIndirectRef, error) {

	sd := &PDFStreamDict{
		PDFDict: PDFDict{
			Dict: map[string]PDFObject{
				"Type":         PDFName("Halftone"),
				"HalftoneType": PDFInteger(10),
				"Xsquare":      PDFInteger(100),
				"Ysquare":      PDFInteger(100),
			},
		},
		Content: []byte{},
	}

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createStreamObjForHalftoneDictType16(xRefTable *XRefTable) (*PDFIndirectRef, error) {

	sd := &PDFStreamDict{
		PDFDict: PDFDict{
			Dict: map[string]PDFObject{
				"Type":         PDFName("Halftone"),
				"HalftoneType": PDFInteger(16),
				"Width":        PDFInteger(100),
				"Height":       PDFInteger(100),
			},
		},
		Content: []byte{},
	}

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createPostScriptCalculatorFunctionStreamDict(xRefTable *XRefTable) (*PDFIndirectRef, error) {

	sd := &PDFStreamDict{
		PDFDict: PDFDict{
			Dict: map[string]PDFObject{
				"FunctionType": PDFInteger(4),
				"Domain":       NewNumberArray(100.),
				"Range":        NewNumberArray(100.),
			},
		},
		Content: []byte{},
	}

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func addResources(xRefTable *XRefTable, pageDict *PDFDict) error {

	fIndRef, err := createFontDict(xRefTable)
	if err != nil {
		return err
	}

	functionalBasedShDict := createFunctionalShadingDict(xRefTable)

	radialShDict := createRadialShadingDict(xRefTable)

	f := PDFDict{
		Dict: map[string]PDFObject{
			"FunctionType": PDFInteger(2),
			"Domain":       NewNumberArray(0.0, 1.0),
			"C0":           NewNumberArray(0.0),
			"C1":           NewNumberArray(1.0),
			"N":            PDFFloat(1),
		},
	}

	fontResources := PDFDict{
		Dict: map[string]PDFObject{
			"F1": *fIndRef,
		},
	}

	shadingResources := PDFDict{
		Dict: map[string]PDFObject{
			"S1": *functionalBasedShDict,
			"S3": *radialShDict,
		},
	}

	colorSpaceResources := PDFDict{
		Dict: map[string]PDFObject{
			"CSCalGray": PDFArray{
				PDFName("CalGray"),
				PDFDict{
					Dict: map[string]PDFObject{
						"WhitePoint": NewNumberArray(0.9505, 1.0000, 1.0890),
					},
				},
			},
			"CSCalRGB": PDFArray{
				PDFName("CalRGB"),
				PDFDict{
					Dict: map[string]PDFObject{
						"WhitePoint": NewNumberArray(0.9505, 1.0000, 1.0890),
					},
				},
			},
			"CSLab": PDFArray{
				PDFName("Lab"),
				PDFDict{
					Dict: map[string]PDFObject{
						"WhitePoint": NewNumberArray(0.9505, 1.0000, 1.0890),
					},
				},
			},
			"CS4DeviceN": PDFArray{
				PDFName("DeviceN"),
				NewNameArray("Orange", "Green", "None"),
				PDFName("DeviceCMYK"),
				f,
				PDFDict{
					Dict: map[string]PDFObject{
						"SubType": PDFName("DeviceN"),
					},
				},
			},
			"CS6DeviceN": PDFArray{
				PDFName("DeviceN"),
				NewNameArray("L", "a", "b", "Spot1"),
				PDFName("DeviceCMYK"),
				f,
				PDFDict{
					Dict: map[string]PDFObject{
						"SubType": PDFName("NChannel"),
						"Process": PDFDict{
							Dict: map[string]PDFObject{
								"ColorSpace": PDFArray{
									PDFName("Lab"),
									PDFDict{
										Dict: map[string]PDFObject{
											"WhitePoint": NewNumberArray(0.9505, 1.0000, 1.0890),
										},
									},
								},
								"Components": NewNameArray("L", "a", "b"),
							},
						},
						"Colorants": PDFDict{
							Dict: map[string]PDFObject{
								"Spot1": PDFArray{
									PDFName("Separation"),
									PDFName("Spot1"),
									PDFName("DeviceCMYK"),
									f,
								},
							},
						},
						"MixingHints": PDFDict{
							Dict: map[string]PDFObject{
								"Solidities": PDFDict{
									Dict: map[string]PDFObject{
										"Spot1": PDFFloat(1.0),
									},
								},
								"DotGain": PDFDict{
									Dict: map[string]PDFObject{
										"Spot1":   f,
										"Magenta": f,
										"Yellow":  f,
									},
								},
								"PrintingOrder": NewNameArray("Magenta", "Yellow", "Spot1"),
							},
						},
					},
				},
			},
		},
	}

	anyXObject, err := createNormalAppearanceForFormField(xRefTable, 20., 20.)
	if err != nil {
		return err
	}

	indRefHalfToneType6, err := createStreamObjForHalftoneDictType6(xRefTable)
	if err != nil {
		return err
	}

	indRefHalfToneType10, err := createStreamObjForHalftoneDictType10(xRefTable)
	if err != nil {
		return err
	}

	indRefHalfToneType16, err := createStreamObjForHalftoneDictType16(xRefTable)
	if err != nil {
		return err
	}

	indRefFunctionStream, err := createPostScriptCalculatorFunctionStreamDict(xRefTable)
	if err != nil {
		return err
	}

	graphicStateResources := PDFDict{
		Dict: map[string]PDFObject{
			"GS1": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("ExtGState"),
					"HT": PDFDict{
						Dict: map[string]PDFObject{
							"Type":             PDFName("Halftone"),
							"HalftoneType":     PDFInteger(1),
							"Frequency":        PDFInteger(120),
							"Angle":            PDFInteger(30),
							"Spotfunction":     PDFName("CosineDot"),
							"TransferFunction": PDFName("Identity"),
						},
					},
					"BM": NewNameArray("Overlay", "Darken", "Normal"),
					"SMask": PDFDict{
						Dict: map[string]PDFObject{
							"Type": PDFName("Mask"),
							"S":    PDFName("Alpha"),
							"G":    *anyXObject,
							"TR":   f,
						},
					},
					"TR":  f,
					"TR2": f,
				},
			},
			"GS2": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("ExtGState"),
					"HT": PDFDict{
						Dict: map[string]PDFObject{
							"Type":         PDFName("Halftone"),
							"HalftoneType": PDFInteger(5),
							"Default": PDFDict{
								Dict: map[string]PDFObject{
									"Type":             PDFName("Halftone"),
									"HalftoneType":     PDFInteger(1),
									"Frequency":        PDFInteger(120),
									"Angle":            PDFInteger(30),
									"Spotfunction":     PDFName("CosineDot"),
									"TransferFunction": PDFName("Identity"),
								},
							},
						},
					},
					"BM": NewNameArray("Overlay", "Darken", "Normal"),
					"SMask": PDFDict{
						Dict: map[string]PDFObject{
							"Type": PDFName("Mask"),
							"S":    PDFName("Alpha"),
							"G":    *anyXObject,
							"TR":   PDFName("Identity"),
						},
					},
					"TR":   PDFArray{f, f, f, f},
					"TR2":  PDFArray{f, f, f, f},
					"BG2":  f,
					"UCR2": f,
					"D":    PDFArray{PDFArray{}, PDFInteger(0)},
				},
			},
			"GS3": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("ExtGState"),
					"HT":   *indRefHalfToneType6,
					"SMask": PDFDict{
						Dict: map[string]PDFObject{
							"Type": PDFName("Mask"),
							"S":    PDFName("Alpha"),
							"G":    *anyXObject,
							"TR":   *indRefFunctionStream,
						},
					},
					"BG2":  *indRefFunctionStream,
					"UCR2": *indRefFunctionStream,
					"TR":   *indRefFunctionStream,
					"TR2":  *indRefFunctionStream,
				},
			},
			"GS4": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("ExtGState"),
					"HT":   *indRefHalfToneType10,
				},
			},
			"GS5": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("ExtGState"),
					"HT":   *indRefHalfToneType16,
				},
			},
			"GS6": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("ExtGState"),
					"HT": PDFDict{
						Dict: map[string]PDFObject{
							"Type":         PDFName("Halftone"),
							"HalftoneType": PDFInteger(1),
							"Frequency":    PDFInteger(120),
							"Angle":        PDFInteger(30),
							"Spotfunction": *indRefFunctionStream,
						},
					},
				},
			},
			"GS7": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("ExtGState"),
					"HT": PDFDict{
						Dict: map[string]PDFObject{
							"Type":         PDFName("Halftone"),
							"HalftoneType": PDFInteger(1),
							"Frequency":    PDFInteger(120),
							"Angle":        PDFInteger(30),
							"Spotfunction": f,
						},
					},
				},
			},
		},
	}

	resourceDict := PDFDict{
		Dict: map[string]PDFObject{
			"Font":       fontResources,
			"Shading":    shadingResources,
			"ColorSpace": colorSpaceResources,
			"ExtGState":  graphicStateResources,
		},
	}

	pageDict.Insert("Resources", resourceDict)

	return nil
}

func addContents(xRefTable *XRefTable, pageDict *PDFDict) error {

	contents := &PDFStreamDict{PDFDict: NewPDFDict()}
	contents.InsertName("Filter", "FlateDecode")
	contents.FilterPipeline = []PDFFilter{{Name: "FlateDecode", DecodeParms: nil}}

	// Page dimensions: 595.27, 841.89

	var b bytes.Buffer

	b.WriteString("BT /F1 12 Tf 0 1 Td 0 Tr 0.5 g (lower left) Tj ET ")
	b.WriteString("BT /F1 12 Tf 0 832 Td 0 Tr (upper left) Tj ET ")
	b.WriteString("BT /F1 12 Tf 537 832 Td 0 Tr (upper right) Tj ET ")
	b.WriteString("BT /F1 12 Tf 540 1 Td 0 Tr (lower right) Tj ET ")
	b.WriteString("BT /F1 12 Tf 297.55 420.5 Td (X) Tj ET ")

	// t := `BT /F1 12 Tf 0 1 Td 0 Tr 0.5 g (lower left) Tj ET `
	// t += "BT /F1 12 Tf 0 832 Td 0 Tr (upper left) Tj ET "
	// t += "BT /F1 12 Tf 537 832 Td 0 Tr (upper right) Tj ET "
	// t += "BT /F1 12 Tf 540 1 Td 0 Tr (lower right) Tj ET "
	// t += "BT /F1 12 Tf 297.55 420.5 Td (X) Tj ET "

	contents.Content = b.Bytes()

	err := encodeStream(contents)
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

func createBoxColorDict() *PDFDict {

	cropBoxColorInfoDict := PDFDict{
		Dict: map[string]PDFObject{
			"C": NewNumberArray(1.0, 1.0, 0.0),
			"W": PDFFloat(1.0),
			"S": PDFName("D"),
			"D": NewIntegerArray(3, 2),
		},
	}

	bleedBoxColorInfoDict := PDFDict{
		Dict: map[string]PDFObject{
			"C": NewNumberArray(1.0, 0.0, 0.0),
			"W": PDFFloat(3.0),
			"S": PDFName("S"),
		},
	}

	trimBoxColorInfoDict := PDFDict{
		Dict: map[string]PDFObject{
			"C": NewNumberArray(0.0, 1.0, 0.0),
			"W": PDFFloat(1.0),
			"S": PDFName("D"),
			"D": NewIntegerArray(3, 2),
		},
	}

	artBoxColorInfoDict := PDFDict{
		Dict: map[string]PDFObject{
			"C": NewNumberArray(0.0, 0.0, 1.0),
			"W": PDFFloat(1.0),
			"S": PDFName("S"),
		},
	}

	return &PDFDict{
		Dict: map[string]PDFObject{
			"CropBox":  cropBoxColorInfoDict,
			"BleedBox": bleedBoxColorInfoDict,
			"Trim":     trimBoxColorInfoDict,
			"ArtBox":   artBoxColorInfoDict,
		},
	}

}

func addViewportDict(pageDict *PDFDict) {

	measureDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type":    PDFName("Measure"),
			"Subtype": PDFName("RL"),
			"R":       PDFStringLiteral("1in = 0.1m"),
			"X": PDFArray{
				PDFDict{
					Dict: map[string]PDFObject{
						"Type": PDFName("NumberFormat"),
						"U":    PDFStringLiteral("mi"),
						"C":    PDFFloat(0.00139),
						"D":    PDFInteger(100000),
					},
				},
			},
			"D": PDFArray{
				PDFDict{
					Dict: map[string]PDFObject{
						"Type": PDFName("NumberFormat"),
						"U":    PDFStringLiteral("mi"),
						"C":    PDFFloat(1),
					},
				},
				PDFDict{
					Dict: map[string]PDFObject{
						"Type": PDFName("NumberFormat"),
						"U":    PDFStringLiteral("feet"),
						"C":    PDFFloat(5280),
					},
				},
				PDFDict{
					Dict: map[string]PDFObject{
						"Type": PDFName("NumberFormat"),
						"U":    PDFStringLiteral("inch"),
						"C":    PDFFloat(12),
						"F":    PDFName("F"),
						"D":    PDFInteger(8),
					},
				},
			},
			"A": PDFArray{
				PDFDict{
					Dict: map[string]PDFObject{
						"Type": PDFName("NumberFormat"),
						"U":    PDFStringLiteral("acres"),
						"C":    PDFFloat(640),
					},
				},
			},
			"O": NewIntegerArray(0, 1),
		}}

	vpDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type":    PDFName("Viewport"),
			"BBox":    NewRectangle(10, 10, 60, 60),
			"Name":    PDFStringLiteral("viewPort"),
			"Measure": measureDict,
		},
	}

	pageDict.Insert("VP", PDFArray{vpDict})
}

func annotRect(i int, w, h, d, l float64) PDFArray {

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

	return NewRectangle(llx, lly, urx, ury)
}

func createAnnotsArray(xRefTable *XRefTable, pageIndRef *PDFIndirectRef, mediaBox *PDFArray) (*PDFArray, error) {

	// Generate side by side lined up annotations starting in the lower left corner of the page.

	pageWidth := (*mediaBox)[2].(PDFFloat)
	pageHeight := (*mediaBox)[3].(PDFFloat)

	arr := PDFArray{}

	for i, f := range []func(*XRefTable, *PDFIndirectRef, *PDFArray) (*PDFIndirectRef, error){
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

func createPageWithAnnotations(xRefTable *XRefTable, parentPageIndRef *PDFIndirectRef, mediaBox *PDFArray) (*PDFIndirectRef, error) {

	pageDict := &PDFDict{
		Dict: map[string]PDFObject{
			"Type":         PDFName("Page"),
			"Parent":       *parentPageIndRef,
			"BleedBox":     *mediaBox,
			"TrimBox":      *mediaBox,
			"ArtBox":       *mediaBox,
			"BoxColorInfo": *createBoxColorDict(),
			"UserUnit":     PDFFloat(1.5)}, // Note: not honored by Apple Preview
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
	separationInfoDict := PDFDict{
		Dict: map[string]PDFObject{
			"Pages":          PDFArray{*pageIndRef},
			"DeviceColorant": PDFName("Cyan"),
			"ColorSpace": PDFArray{
				PDFName("Separation"),
				PDFName("Green"),
				PDFName("DeviceCMYK"),
				PDFDict{
					Dict: map[string]PDFObject{
						"FunctionType": PDFInteger(2),
						"Domain":       NewNumberArray(0.0, 1.0),
						"C0":           NewNumberArray(0.0),
						"C1":           NewNumberArray(1.0),
						"N":            PDFFloat(1),
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

func createPageWithAcroForm(xRefTable *XRefTable, parentPageIndRef *PDFIndirectRef, annotsArray *PDFArray, mediaBox *PDFArray) (*PDFIndirectRef, error) {

	pageDict := &PDFDict{
		Dict: map[string]PDFObject{
			"Type":         PDFName("Page"),
			"Parent":       *parentPageIndRef,
			"BleedBox":     *mediaBox,
			"TrimBox":      *mediaBox,
			"ArtBox":       *mediaBox,
			"BoxColorInfo": *createBoxColorDict(),
			"UserUnit":     PDFFloat(1.0), // Note: not honored by Apple Preview
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

func addPageTreeWithAnnotations(xRefTable *XRefTable, rootDict *PDFDict) (*PDFIndirectRef, error) {

	// mediabox = physical page dimensions
	mediaBox := NewRectangle(0, 0, 595.27, 841.89)

	pagesDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Pages"),
			"Count":    PDFInteger(1),
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

	pagesDict.Insert("Kids", PDFArray{*pageIndRef})

	rootDict.Insert("Pages", *parentPageIndRef)

	return pageIndRef, nil
}

func addPageTreeWithAcroFields(xRefTable *XRefTable, rootDict *PDFDict, annotsArray *PDFArray) (*PDFIndirectRef, error) {

	// mediabox = physical page dimensions
	mediaBox := NewRectangle(0, 0, 595.27, 841.89)

	pagesDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type":     PDFName("Pages"),
			"Count":    PDFInteger(1),
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

	pagesDict.Insert("Kids", PDFArray{*pageIndRef})

	rootDict.Insert("Pages", *parentPageIndRef)

	return pageIndRef, nil
}

// create a thread with 2 beads.
func createThreadDict(xRefTable *XRefTable, pageIndRef *PDFIndirectRef) (*PDFIndirectRef, error) {

	infoDict := NewPDFDict()
	infoDict.InsertString("Title", "DummyArticle")

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Thread"),
			"I":    infoDict,
		},
	}

	dIndRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	// create first bead
	d1 := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Bead"),
			"T":    *dIndRef,
			"P":    *pageIndRef,
			"R":    NewRectangle(0, 0, 100, 100),
		},
	}

	d1IndRef, err := xRefTable.IndRefForNewObject(d1)
	if err != nil {
		return nil, err
	}

	d.Insert("F", *d1IndRef)

	// create last bead
	d2 := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Bead"),
			"T":    *dIndRef,
			"N":    *d1IndRef,
			"V":    *d1IndRef,
			"P":    *pageIndRef,
			"R":    NewRectangle(0, 100, 200, 100),
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

func addThreads(xRefTable *XRefTable, rootDict *PDFDict, pageIndRef *PDFIndirectRef) error {

	indRef, err := createThreadDict(xRefTable, pageIndRef)
	if err != nil {
		return err
	}

	indRef, err = xRefTable.IndRefForNewObject(PDFArray{*indRef})
	if err != nil {
		return err
	}

	rootDict.Insert("Threads", *indRef)

	return nil
}

func addOpenAction(xRefTable *XRefTable, rootDict *PDFDict) error {

	nextActionDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Action"),
			"S":    PDFName("Movie"),
			"T":    PDFStringLiteral("Sample Movie"),
		},
	}

	script := `app.alert('Hello Gopher!');`

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Action"),
			"S":    PDFName("JavaScript"),
			"JS":   PDFStringLiteral(script),
			"Next": nextActionDict,
		},
	}

	rootDict.Insert("OpenAction", d)

	return nil
}

func addURI(xRefTable *XRefTable, rootDict *PDFDict) {

	d := NewPDFDict()
	d.InsertString("Base", "http://www.adobe.com")

	rootDict.Insert("URI", d)
}

func addSpiderInfo(xRefTable *XRefTable, rootDict *PDFDict) error {

	// webCaptureInfoDict
	webCaptureInfoDict := NewPDFDict()
	webCaptureInfoDict.InsertInt("V", 1.0)

	arr := PDFArray{}
	captureCmdDict := NewPDFDict()
	captureCmdDict.InsertString("URL", (""))

	cmdSettingsDict := NewPDFDict()
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

func addOCProperties(xRefTable *XRefTable, rootDict *PDFDict) error {

	usageAppDict := PDFDict{
		Dict: map[string]PDFObject{
			"Event":    PDFName("View"),
			"OCGs":     PDFArray{}, // of indRefs
			"Category": NewNameArray("Language"),
		},
	}

	optionalContentConfigDict := PDFDict{
		Dict: map[string]PDFObject{
			"Name":      PDFStringLiteral("OCConf"),
			"Creator":   PDFStringLiteral("Horst Rutter"),
			"BaseState": PDFName("ON"),
			"OFF":       PDFArray{},
			"Intent":    PDFName("Design"),
			"AS":        PDFArray{usageAppDict},
			"Order":     PDFArray{},
			"ListMode":  PDFName("AllPages"),
			"RBGroups":  PDFArray{},
			"Locked":    PDFArray{},
		},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"OCGs":    PDFArray{}, // of indRefs
			"D":       optionalContentConfigDict,
			"Configs": PDFArray{optionalContentConfigDict},
		},
	}

	rootDict.Insert("OCProperties", d)

	return nil
}

func addRequirements(xRefTable *XRefTable, rootDict *PDFDict) {

	d := NewPDFDict()
	d.InsertName("Type", "Requirement")
	d.InsertName("S", "EnableJavaScripts")

	rootDict.Insert("Requirements", PDFArray{d})
}

// AnnotationDemoXRef creates a PDF file with examples of annotations and actions.
func createAnnotationDemoXRef() (*XRefTable, error) {

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

func createNormalAppearanceForFormField(xRefTable *XRefTable, w, h float64) (*PDFIndirectRef, error) {

	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &PDFStreamDict{
		PDFDict: PDFDict{
			Dict: map[string]PDFObject{
				"Type":     PDFName("XObject"),
				"Subtype":  PDFName("Form"),
				"FormType": PDFInteger(1),
				"BBox":     NewRectangle(0, 0, w, h),
				"Matrix":   NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		},
		Content: b.Bytes(),
	}

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createRolloverAppearanceForFormField(xRefTable *XRefTable, w, h float64) (*PDFIndirectRef, error) {

	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "1 0 0 RG 0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &PDFStreamDict{
		PDFDict: PDFDict{
			Dict: map[string]PDFObject{
				"Type":     PDFName("XObject"),
				"Subtype":  PDFName("Form"),
				"FormType": PDFInteger(1),
				"BBox":     NewRectangle(0, 0, w, h),
				"Matrix":   NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		},
		Content: b.Bytes(),
		//FilterPipeline: []PDFFilter{{Name: "FlateDecode", DecodeParms: nil}},
	}

	//sd.InsertName("Filter", "FlateDecode")

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createDownAppearanceForFormField(xRefTable *XRefTable, w, h float64) (*PDFIndirectRef, error) {

	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &PDFStreamDict{
		PDFDict: PDFDict{
			Dict: map[string]PDFObject{
				"Type":     PDFName("XObject"),
				"Subtype":  PDFName("Form"),
				"FormType": PDFInteger(1),
				"BBox":     NewRectangle(0, 0, w, h),
				"Matrix":   NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		},
		Content: b.Bytes(),
		//FilterPipeline: []PDFFilter{{Name: "FlateDecode", DecodeParms: nil}},
	}

	//sd.InsertName("Filter", "FlateDecode")

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createTextField(xRefTable *XRefTable, pageAnnots *PDFArray) (*PDFIndirectRef, error) {

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

	resourceDict := PDFDict{
		Dict: map[string]PDFObject{
			"Font": PDFDict{
				Dict: map[string]PDFObject{
					"Helvetica": *fontDict,
				},
			},
		},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"AP": PDFDict{
				Dict: map[string]PDFObject{
					"N": *fN,
					"R": *fR,
					"D": *fD,
				},
			},
			"DA":      PDFStringLiteral("/Helvetica 12 Tf 0 g"),
			"DR":      resourceDict,
			"FT":      PDFName("Tx"),
			"Rect":    NewRectangle(x, y, x+w, y+h),
			"Border":  NewIntegerArray(0, 0, 1),
			"Type":    PDFName("Annot"),
			"Subtype": PDFName("Widget"),
			"T":       PDFStringLiteral("inputField"),
			"TU":      PDFStringLiteral("inputField"),
			"DV":      PDFStringLiteral("Default value"),
			"V":       PDFStringLiteral("Default value"),
		},
	}

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	*pageAnnots = append(*pageAnnots, *indRef)

	return indRef, nil
}

func createYesAppearance(xRefTable *XRefTable, resourceDict *PDFDict, w, h float64) (*PDFIndirectRef, error) {

	var b bytes.Buffer
	fmt.Fprintf(&b, "q 0 0 1 rg BT /ZaDb 12 Tf 0 0 Td (8) Tj ET Q")

	sd := &PDFStreamDict{
		PDFDict: PDFDict{
			Dict: map[string]PDFObject{
				"Resources": *resourceDict,
				"Subtype":   PDFName("Form"),
				"BBox":      NewRectangle(0, 0, w, h),
				"OPI": PDFDict{
					Dict: map[string]PDFObject{
						"2.0": PDFDict{
							Dict: map[string]PDFObject{
								"Type":    PDFName("OPI"),
								"Version": PDFFloat(2.0),
								"F":       PDFStringLiteral("Proxy"),
								"Inks":    PDFName("full_color"),
							},
						},
					},
				},
				"Ref": PDFDict{
					Dict: map[string]PDFObject{
						"F":    PDFStringLiteral("Proxy"),
						"Page": PDFInteger(1),
					},
				},
			},
		},
		Content: b.Bytes(),
	}

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createOffAppearance(xRefTable *XRefTable, resourceDict *PDFDict, w, h float64) (*PDFIndirectRef, error) {

	var b bytes.Buffer
	fmt.Fprintf(&b, "q 0 0 1 rg BT /ZaDb 12 Tf 0 0 Td (4) Tj ET Q")

	sd := &PDFStreamDict{
		PDFDict: PDFDict{
			Dict: map[string]PDFObject{
				"Resources": *resourceDict,
				"Subtype":   PDFName("Form"),
				"BBox":      NewRectangle(0, 0, w, h),
				"OPI": PDFDict{
					Dict: map[string]PDFObject{
						"1.3": PDFDict{
							Dict: map[string]PDFObject{
								"Type":     PDFName("OPI"),
								"Version":  PDFFloat(1.3),
								"F":        PDFStringLiteral("Proxy"),
								"Size":     NewIntegerArray(400, 400),
								"CropRect": NewIntegerArray(0, 400, 400, 0),
								"Position": NewNumberArray(0, 0, 0, 400, 400, 400, 400, 0),
							},
						},
					},
				},
			},
		},
		Content: b.Bytes(),
	}

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createCheckBoxButtonField(xRefTable *XRefTable, pageAnnots *PDFArray) (*PDFIndirectRef, error) {

	fontDict, err := createZapfDingbatsFontDict(xRefTable)
	if err != nil {
		return nil, err
	}

	resDict := PDFDict{
		Dict: map[string]PDFObject{
			"Font": PDFDict{
				Dict: map[string]PDFObject{
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

	apDict := PDFDict{
		Dict: map[string]PDFObject{
			"N": PDFDict{
				Dict: map[string]PDFObject{
					"Yes": *yesForm,
					"Off": *offForm,
				},
			},
		},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"FT":      PDFName("Btn"),
			"Rect":    NewRectangle(250, 300, 270, 320),
			"Type":    PDFName("Annot"),
			"Subtype": PDFName("Widget"),
			"T":       PDFStringLiteral("CheckBox"),
			"TU":      PDFStringLiteral("CheckBox"),
			"V":       PDFName("Yes"),
			"AS":      PDFName("Yes"),
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

func createRadioButtonField(xRefTable *XRefTable, pageAnnots *PDFArray) (*PDFIndirectRef, error) {

	var flags uint32
	flags = setBit(flags, 16)

	d := PDFDict{
		Dict: map[string]PDFObject{
			"FT":   PDFName("Btn"),
			"Ff":   PDFInteger(flags),
			"Rect": NewRectangle(250, 400, 280, 420),
			//"Type":    PDFName("Annot"),
			//"Subtype": PDFName("Widget"),
			"T": PDFStringLiteral("Credit card"),
			"V": PDFName("card1"),
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

	resDict := PDFDict{
		Dict: map[string]PDFObject{
			"Font": PDFDict{
				Dict: map[string]PDFObject{
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

	r1 := PDFDict{
		Dict: map[string]PDFObject{
			"Rect":    NewRectangle(250, 400, 280, 420),
			"Type":    PDFName("Annot"),
			"Subtype": PDFName("Widget"),
			"Parent":  *indRef,
			"T":       PDFStringLiteral("Radio1"),
			"TU":      PDFStringLiteral("Radio1"),
			"AS":      PDFName("card1"),
			"AP": PDFDict{
				Dict: map[string]PDFObject{
					"N": PDFDict{
						Dict: map[string]PDFObject{
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

	r2 := PDFDict{
		Dict: map[string]PDFObject{
			"Rect":    NewRectangle(300, 400, 330, 420),
			"Type":    PDFName("Annot"),
			"Subtype": PDFName("Widget"),
			"Parent":  *indRef,
			"T":       PDFStringLiteral("Radio2"),
			"TU":      PDFStringLiteral("Radio2"),
			"AS":      PDFName("Off"),
			"AP": PDFDict{
				Dict: map[string]PDFObject{
					"N": PDFDict{
						Dict: map[string]PDFObject{
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

	d.Insert("Kids", PDFArray{*indRefR1, *indRefR2})

	*pageAnnots = append(*pageAnnots, *indRefR1)
	*pageAnnots = append(*pageAnnots, *indRefR2)

	return indRef, nil
}

func createResetButton(xRefTable *XRefTable, pageAnnots *PDFArray) (*PDFIndirectRef, error) {

	var flags uint32
	flags = setBit(flags, 17)

	fN, err := createNormalAppearanceForFormField(xRefTable, 20, 20)
	if err != nil {
		return nil, err
	}

	resetFormActionDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type":   PDFName("Action"),
			"S":      PDFName("ResetForm"),
			"Fields": NewStringArray("inputField"),
			"Flags":  PDFInteger(0),
		},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"FT":      PDFName("Btn"),
			"Ff":      PDFInteger(flags),
			"Rect":    NewRectangle(100, 400, 120, 420),
			"Type":    PDFName("Annot"),
			"Subtype": PDFName("Widget"),
			"AP":      PDFDict{Dict: map[string]PDFObject{"N": *fN}},
			"T":       PDFStringLiteral("Reset"),
			"TU":      PDFStringLiteral("Reset"),
			"A":       resetFormActionDict,
		},
	}

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	*pageAnnots = append(*pageAnnots, *indRef)

	return indRef, nil
}

func createSubmitButton(xRefTable *XRefTable, pageAnnots *PDFArray) (*PDFIndirectRef, error) {

	var flags uint32
	flags = setBit(flags, 17)

	fN, err := createNormalAppearanceForFormField(xRefTable, 20, 20)
	if err != nil {
		return nil, err
	}

	urlSpec := PDFDict{
		Dict: map[string]PDFObject{
			"FS": PDFName("URL"),
			"F":  PDFStringLiteral("http://www.me.com"),
		},
	}

	submitFormActionDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type":   PDFName("Action"),
			"S":      PDFName("SubmitForm"),
			"F":      urlSpec,
			"Fields": NewStringArray("inputField"),
			"Flags":  PDFInteger(0),
		},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"FT":      PDFName("Btn"),
			"Ff":      PDFInteger(flags),
			"Rect":    NewRectangle(140, 400, 160, 420),
			"Type":    PDFName("Annot"),
			"Subtype": PDFName("Widget"),
			"AP":      PDFDict{Dict: map[string]PDFObject{"N": *fN}},
			"T":       PDFStringLiteral("Submit"),
			"TU":      PDFStringLiteral("Submit"),
			"A":       submitFormActionDict,
		},
	}

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	*pageAnnots = append(*pageAnnots, *indRef)

	return indRef, nil
}

func streamObjForXFAElement(xRefTable *XRefTable, s string) (*PDFIndirectRef, error) {

	sd := &PDFStreamDict{
		PDFDict: PDFDict{
			Dict: map[string]PDFObject{},
		},
		Content: []byte(s),
	}

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createXFAArray(xRefTable *XRefTable) (*PDFArray, error) {

	sd1, err := streamObjForXFAElement(xRefTable, "<xdp:xdp xmlns:xdp=\"http://ns.adobe.com/xdp/\">")
	if err != nil {
		return nil, err
	}

	sd3, err := streamObjForXFAElement(xRefTable, "</xdp:xdp>")
	if err != nil {
		return nil, err
	}

	return &PDFArray{
		PDFStringLiteral("xdp:xdp"), *sd1,
		PDFStringLiteral("/xdp:xdp"), *sd3,
	}, nil
}

func createAcroFormDict(xRefTable *XRefTable) (*PDFDict, *PDFArray, error) {

	pageAnnots := PDFArray{}

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

	resetButton, err := createResetButton(xRefTable, &pageAnnots)
	if err != nil {
		return nil, nil, err
	}

	submitButton, err := createSubmitButton(xRefTable, &pageAnnots)
	if err != nil {
		return nil, nil, err
	}

	xfaArr, err := createXFAArray(xRefTable)
	if err != nil {
		return nil, nil, err
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Fields":          PDFArray{*text, *checkBox, *radioButton, *resetButton, *submitButton}, // indRefs of fieldDicts
			"NeedAppearances": PDFBoolean(true),
			"CO":              PDFArray{*text},
			"XFA":             *xfaArr,
		},
	}

	return &d, &pageAnnots, nil
}

// AcroFormDemoXRef creates a PDF file with an AcroForm example.
func createAcroFormDemoXRef() (*XRefTable, error) {

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
		PDFDict{
			Dict: map[string]PDFObject{
				"FitWindow":    PDFBoolean(true),
				"CenterWindow": PDFBoolean(true),
			},
		},
	)

	return xRefTable, nil
}

// DemoPDF creates a demo PDF file for testing validation.
func createDemoPDF(xRefTable *XRefTable, dirName, fileName string) error {

	config := NewDefaultConfiguration()

	ctx := &PDFContext{
		Configuration: config,
		XRefTable:     xRefTable,
		Write:         NewWriteContext(config.Eol),
	}

	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	return writePDFFile(ctx)
}
