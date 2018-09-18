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

// Functions needed to create a test.pdf that gets used for validation testing (see process_test.go)

import (
	"bytes"
	"fmt"

	"github.com/hhrutter/pdfcpu/pkg/filter"
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

// CreateDemoXRef creates a minimal PDF file for demo purposes.
func CreateDemoXRef() (*XRefTable, error) {

	xRefTable, err := createXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	err = addPageTree(xRefTable, rootDict)
	if err != nil {
		return nil, err
	}

	return xRefTable, nil
}

func createFontDict(xRefTable *XRefTable) (*IndirectRef, error) {

	d := NewPDFDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "Type1")
	d.InsertName("BaseFont", "Helvetica")

	return xRefTable.IndRefForNewObject(d)
}

func createZapfDingbatsFontDict(xRefTable *XRefTable) (*IndirectRef, error) {

	d := NewPDFDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "Type1")
	d.InsertName("BaseFont", "ZapfDingbats")

	return xRefTable.IndRefForNewObject(d)
}

func createFunctionalShadingDict(xRefTable *XRefTable) *PDFDict {

	f := PDFDict{
		Dict: map[string]Object{
			"FunctionType": Integer(2),
			"Domain":       NewNumberArray(1.0, 1.2, 1.4, 1.6, 1.8, 2.0),
			"N":            Float(1),
		},
	}

	d := PDFDict{
		Dict: map[string]Object{
			"ShadingType": Integer(1),
			"Function":    Array{f},
		},
	}

	return &d
}

func createRadialShadingDict(xRefTable *XRefTable) *PDFDict {

	f := PDFDict{
		Dict: map[string]Object{
			"FunctionType": Integer(2),
			"Domain":       NewNumberArray(1.0, 1.2, 1.4, 1.6, 1.8, 2.0),
			"N":            Float(1),
		},
	}

	d := PDFDict{
		Dict: map[string]Object{
			"ShadingType": Integer(3),
			"Coords":      NewNumberArray(0, 0, 50, 10, 10, 100),
			"Function":    Array{f},
		},
	}

	return &d
}

func createStreamObjForHalftoneDictType6(xRefTable *XRefTable) (*IndirectRef, error) {

	sd := &StreamDict{
		PDFDict: PDFDict{
			Dict: map[string]Object{
				"Type":             Name("Halftone"),
				"HalftoneType":     Integer(6),
				"Width":            Integer(100),
				"Height":           Integer(100),
				"TransferFunction": Name("Identity"),
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

func createStreamObjForHalftoneDictType10(xRefTable *XRefTable) (*IndirectRef, error) {

	sd := &StreamDict{
		PDFDict: PDFDict{
			Dict: map[string]Object{
				"Type":         Name("Halftone"),
				"HalftoneType": Integer(10),
				"Xsquare":      Integer(100),
				"Ysquare":      Integer(100),
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

func createStreamObjForHalftoneDictType16(xRefTable *XRefTable) (*IndirectRef, error) {

	sd := &StreamDict{
		PDFDict: PDFDict{
			Dict: map[string]Object{
				"Type":         Name("Halftone"),
				"HalftoneType": Integer(16),
				"Width":        Integer(100),
				"Height":       Integer(100),
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

func createPostScriptCalculatorFunctionStreamDict(xRefTable *XRefTable) (*IndirectRef, error) {

	sd := &StreamDict{
		PDFDict: PDFDict{
			Dict: map[string]Object{
				"FunctionType": Integer(4),
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
		Dict: map[string]Object{
			"FunctionType": Integer(2),
			"Domain":       NewNumberArray(0.0, 1.0),
			"C0":           NewNumberArray(0.0),
			"C1":           NewNumberArray(1.0),
			"N":            Float(1),
		},
	}

	fontResources := PDFDict{
		Dict: map[string]Object{
			"F1": *fIndRef,
		},
	}

	shadingResources := PDFDict{
		Dict: map[string]Object{
			"S1": *functionalBasedShDict,
			"S3": *radialShDict,
		},
	}

	colorSpaceResources := PDFDict{
		Dict: map[string]Object{
			"CSCalGray": Array{
				Name("CalGray"),
				PDFDict{
					Dict: map[string]Object{
						"WhitePoint": NewNumberArray(0.9505, 1.0000, 1.0890),
					},
				},
			},
			"CSCalRGB": Array{
				Name("CalRGB"),
				PDFDict{
					Dict: map[string]Object{
						"WhitePoint": NewNumberArray(0.9505, 1.0000, 1.0890),
					},
				},
			},
			"CSLab": Array{
				Name("Lab"),
				PDFDict{
					Dict: map[string]Object{
						"WhitePoint": NewNumberArray(0.9505, 1.0000, 1.0890),
					},
				},
			},
			"CS4DeviceN": Array{
				Name("DeviceN"),
				NewNameArray("Orange", "Green", "None"),
				Name("DeviceCMYK"),
				f,
				PDFDict{
					Dict: map[string]Object{
						"SubType": Name("DeviceN"),
					},
				},
			},
			"CS6DeviceN": Array{
				Name("DeviceN"),
				NewNameArray("L", "a", "b", "Spot1"),
				Name("DeviceCMYK"),
				f,
				PDFDict{
					Dict: map[string]Object{
						"SubType": Name("NChannel"),
						"Process": PDFDict{
							Dict: map[string]Object{
								"ColorSpace": Array{
									Name("Lab"),
									PDFDict{
										Dict: map[string]Object{
											"WhitePoint": NewNumberArray(0.9505, 1.0000, 1.0890),
										},
									},
								},
								"Components": NewNameArray("L", "a", "b"),
							},
						},
						"Colorants": PDFDict{
							Dict: map[string]Object{
								"Spot1": Array{
									Name("Separation"),
									Name("Spot1"),
									Name("DeviceCMYK"),
									f,
								},
							},
						},
						"MixingHints": PDFDict{
							Dict: map[string]Object{
								"Solidities": PDFDict{
									Dict: map[string]Object{
										"Spot1": Float(1.0),
									},
								},
								"DotGain": PDFDict{
									Dict: map[string]Object{
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
		Dict: map[string]Object{
			"GS1": PDFDict{
				Dict: map[string]Object{
					"Type": Name("ExtGState"),
					"HT": PDFDict{
						Dict: map[string]Object{
							"Type":             Name("Halftone"),
							"HalftoneType":     Integer(1),
							"Frequency":        Integer(120),
							"Angle":            Integer(30),
							"SpotFunction":     Name("CosineDot"),
							"TransferFunction": Name("Identity"),
						},
					},
					"BM": NewNameArray("Overlay", "Darken", "Normal"),
					"SMask": PDFDict{
						Dict: map[string]Object{
							"Type": Name("Mask"),
							"S":    Name("Alpha"),
							"G":    *anyXObject,
							"TR":   f,
						},
					},
					"TR":  f,
					"TR2": f,
				},
			},
			"GS2": PDFDict{
				Dict: map[string]Object{
					"Type": Name("ExtGState"),
					"HT": PDFDict{
						Dict: map[string]Object{
							"Type":         Name("Halftone"),
							"HalftoneType": Integer(5),
							"Default": PDFDict{
								Dict: map[string]Object{
									"Type":             Name("Halftone"),
									"HalftoneType":     Integer(1),
									"Frequency":        Integer(120),
									"Angle":            Integer(30),
									"SpotFunction":     Name("CosineDot"),
									"TransferFunction": Name("Identity"),
								},
							},
						},
					},
					"BM": NewNameArray("Overlay", "Darken", "Normal"),
					"SMask": PDFDict{
						Dict: map[string]Object{
							"Type": Name("Mask"),
							"S":    Name("Alpha"),
							"G":    *anyXObject,
							"TR":   Name("Identity"),
						},
					},
					"TR":   Array{f, f, f, f},
					"TR2":  Array{f, f, f, f},
					"BG2":  f,
					"UCR2": f,
					"D":    Array{Array{}, Integer(0)},
				},
			},
			"GS3": PDFDict{
				Dict: map[string]Object{
					"Type": Name("ExtGState"),
					"HT":   *indRefHalfToneType6,
					"SMask": PDFDict{
						Dict: map[string]Object{
							"Type": Name("Mask"),
							"S":    Name("Alpha"),
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
				Dict: map[string]Object{
					"Type": Name("ExtGState"),
					"HT":   *indRefHalfToneType10,
				},
			},
			"GS5": PDFDict{
				Dict: map[string]Object{
					"Type": Name("ExtGState"),
					"HT":   *indRefHalfToneType16,
				},
			},
			"GS6": PDFDict{
				Dict: map[string]Object{
					"Type": Name("ExtGState"),
					"HT": PDFDict{
						Dict: map[string]Object{
							"Type":         Name("Halftone"),
							"HalftoneType": Integer(1),
							"Frequency":    Integer(120),
							"Angle":        Integer(30),
							"SpotFunction": *indRefFunctionStream,
						},
					},
				},
			},
			"GS7": PDFDict{
				Dict: map[string]Object{
					"Type": Name("ExtGState"),
					"HT": PDFDict{
						Dict: map[string]Object{
							"Type":         Name("Halftone"),
							"HalftoneType": Integer(1),
							"Frequency":    Integer(120),
							"Angle":        Integer(30),
							"SpotFunction": f,
						},
					},
				},
			},
		},
	}

	resourceDict := PDFDict{
		Dict: map[string]Object{
			"Font":       fontResources,
			"Shading":    shadingResources,
			"ColorSpace": colorSpaceResources,
			"ExtGState":  graphicStateResources,
		},
	}

	pageDict.Insert("Resources", resourceDict)

	return nil
}

func addContents(xRefTable *XRefTable, pageDict *PDFDict, mediaBox *Array) error {

	contents := &StreamDict{PDFDict: NewPDFDict()}
	contents.InsertName("Filter", filter.Flate)
	contents.FilterPipeline = []PDFFilter{{Name: filter.Flate, DecodeParms: nil}}

	mb := rect(xRefTable, *mediaBox)

	// Page dimensions: 595.27, 841.89 xcxcvxcv

	var b bytes.Buffer

	b.WriteString("[3]0 d 0 w ")

	fmt.Fprintf(&b, "0 0 m %f %f l s %f 0 m 0 %f l s ", mb.Width(), mb.Height(), mb.Width(), mb.Height())
	fmt.Fprintf(&b, "%f 0 m %f %f l s 0 %f m %f %f l s ", mb.Width()/2, mb.Width()/2, mb.Height(), mb.Height()/2, mb.Width(), mb.Height()/2)

	// // Horizontal guides
	b.WriteString("0 500 m 400 500 l s ")
	b.WriteString("0 400 m 400 400 l s ")
	b.WriteString("0 200 m 400 200 l s ")
	b.WriteString("0 100 m 400 100 l s ")

	// // Vertical guides
	b.WriteString("100 0 m 100 600 l s ")
	b.WriteString("300 0 m 300 600 l s ")
	// b.WriteString("267.64 0 m 267.64 841.89 l s ")
	// b.WriteString("257.64 0 m 257.64 841.89 l s ")
	// b.WriteString("247.64 0 m 247.64 841.89 l s ")
	// b.WriteString("237.64 0 m 237.64 841.89 l s ")
	// b.WriteString("227.64 0 m 227.64 841.89 l s ")
	// b.WriteString("217.64 0 m 217.64 841.89 l s ")
	// b.WriteString("207.64 0 m 207.64 841.89 l s ")
	// b.WriteString("197.64 0 m 197.64 841.89 l s ")

	// b.WriteString("307.64 0 m 307.64 841.89 l s ")
	// b.WriteString("317.64 0 m 317.64 841.89 l s ")
	// b.WriteString("327.64 0 m 327.64 841.89 l s ")
	// b.WriteString("337.64 0 m 337.64 841.89 l s ")
	// b.WriteString("347.64 0 m 347.64 841.89 l s ")
	// b.WriteString("357.64 0 m 357.64 841.89 l s ")
	// b.WriteString("367.64 0 m 367.64 841.89 l s ")
	// b.WriteString("377.64 0 m 377.64 841.89 l s ")
	// b.WriteString("387.64 0 m 387.64 841.89 l s ")
	// b.WriteString("397.64 0 m 397.64 841.89 l s ")

	// b.WriteString("BT /F1 12 Tf 0 1 Td 0 Tr (lower left) Tj ET ")
	// b.WriteString("BT /F1 12 Tf 0 832 Td 0 Tr (upper left) Tj ET ")
	// b.WriteString("BT /F1 12 Tf 537 832 Td 0 Tr (upper right) Tj ET ")
	// b.WriteString("BT /F1 12 Tf 540 1 Td 0 Tr (lower right) Tj ET ")
	// b.WriteString("BT /F1 12 Tf 297.55 420.5 Td (pdfcpu powered by Go) Tj ET ")

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
		Dict: map[string]Object{
			"C": NewNumberArray(1.0, 1.0, 0.0),
			"W": Float(1.0),
			"S": Name("D"),
			"D": NewIntegerArray(3, 2),
		},
	}

	bleedBoxColorInfoDict := PDFDict{
		Dict: map[string]Object{
			"C": NewNumberArray(1.0, 0.0, 0.0),
			"W": Float(3.0),
			"S": Name("S"),
		},
	}

	trimBoxColorInfoDict := PDFDict{
		Dict: map[string]Object{
			"C": NewNumberArray(0.0, 1.0, 0.0),
			"W": Float(1.0),
			"S": Name("D"),
			"D": NewIntegerArray(3, 2),
		},
	}

	artBoxColorInfoDict := PDFDict{
		Dict: map[string]Object{
			"C": NewNumberArray(0.0, 0.0, 1.0),
			"W": Float(1.0),
			"S": Name("S"),
		},
	}

	return &PDFDict{
		Dict: map[string]Object{
			"CropBox":  cropBoxColorInfoDict,
			"BleedBox": bleedBoxColorInfoDict,
			"Trim":     trimBoxColorInfoDict,
			"ArtBox":   artBoxColorInfoDict,
		},
	}

}

func addViewportDict(pageDict *PDFDict) {

	measureDict := PDFDict{
		Dict: map[string]Object{
			"Type":    Name("Measure"),
			"Subtype": Name("RL"),
			"R":       StringLiteral("1in = 0.1m"),
			"X": Array{
				PDFDict{
					Dict: map[string]Object{
						"Type": Name("NumberFormat"),
						"U":    StringLiteral("mi"),
						"C":    Float(0.00139),
						"D":    Integer(100000),
					},
				},
			},
			"D": Array{
				PDFDict{
					Dict: map[string]Object{
						"Type": Name("NumberFormat"),
						"U":    StringLiteral("mi"),
						"C":    Float(1),
					},
				},
				PDFDict{
					Dict: map[string]Object{
						"Type": Name("NumberFormat"),
						"U":    StringLiteral("feet"),
						"C":    Float(5280),
					},
				},
				PDFDict{
					Dict: map[string]Object{
						"Type": Name("NumberFormat"),
						"U":    StringLiteral("inch"),
						"C":    Float(12),
						"F":    Name("F"),
						"D":    Integer(8),
					},
				},
			},
			"A": Array{
				PDFDict{
					Dict: map[string]Object{
						"Type": Name("NumberFormat"),
						"U":    StringLiteral("acres"),
						"C":    Float(640),
					},
				},
			},
			"O": NewIntegerArray(0, 1),
		}}

	vpDict := PDFDict{
		Dict: map[string]Object{
			"Type":    Name("Viewport"),
			"BBox":    NewRectangle(10, 10, 60, 60),
			"Name":    StringLiteral("viewPort"),
			"Measure": measureDict,
		},
	}

	pageDict.Insert("VP", Array{vpDict})
}

func annotRect(i int, w, h, d, l float64) Array {

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

func createAnnotsArray(xRefTable *XRefTable, pageIndRef *IndirectRef, mediaBox *Array) (*Array, error) {

	// Generate side by side lined up annotations starting in the lower left corner of the page.

	pageWidth := (*mediaBox)[2].(Float)
	pageHeight := (*mediaBox)[3].(Float)

	arr := Array{}

	for i, f := range []func(*XRefTable, *IndirectRef, *Array) (*IndirectRef, error){
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

func createPageWithAnnotations(xRefTable *XRefTable, parentPageIndRef *IndirectRef, mediaBox *Array) (*IndirectRef, error) {

	pageDict := &PDFDict{
		Dict: map[string]Object{
			"Type":         Name("Page"),
			"Parent":       *parentPageIndRef,
			"BleedBox":     *mediaBox,
			"TrimBox":      *mediaBox,
			"ArtBox":       *mediaBox,
			"BoxColorInfo": *createBoxColorDict(),
			"UserUnit":     Float(1.5)}, // Note: not honored by Apple Preview
	}

	err := addResources(xRefTable, pageDict)
	if err != nil {
		return nil, err
	}

	err = addContents(xRefTable, pageDict, mediaBox)
	if err != nil {
		return nil, err
	}

	pageIndRef, err := xRefTable.IndRefForNewObject(*pageDict)
	if err != nil {
		return nil, err
	}

	// Fake SeparationInfo related to a single page only.
	separationInfoDict := PDFDict{
		Dict: map[string]Object{
			"Pages":          Array{*pageIndRef},
			"DeviceColorant": Name("Cyan"),
			"ColorSpace": Array{
				Name("Separation"),
				Name("Green"),
				Name("DeviceCMYK"),
				PDFDict{
					Dict: map[string]Object{
						"FunctionType": Integer(2),
						"Domain":       NewNumberArray(0.0, 1.0),
						"C0":           NewNumberArray(0.0),
						"C1":           NewNumberArray(1.0),
						"N":            Float(1),
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

func createPageWithAcroForm(xRefTable *XRefTable, parentPageIndRef *IndirectRef, annotsArray *Array, mediaBox *Array) (*IndirectRef, error) {

	pageDict := &PDFDict{
		Dict: map[string]Object{
			"Type":         Name("Page"),
			"Parent":       *parentPageIndRef,
			"BleedBox":     *mediaBox,
			"TrimBox":      *mediaBox,
			"ArtBox":       *mediaBox,
			"BoxColorInfo": *createBoxColorDict(),
			"UserUnit":     Float(1.0), // Note: not honored by Apple Preview
		},
	}

	err := addResources(xRefTable, pageDict)
	if err != nil {
		return nil, err
	}

	err = addContents(xRefTable, pageDict, mediaBox)
	if err != nil {
		return nil, err
	}

	pageDict.Insert("Annots", *annotsArray)

	return xRefTable.IndRefForNewObject(*pageDict)
}

func createPage(xRefTable *XRefTable, parentPageIndRef *IndirectRef, mediaBox *Array) (*IndirectRef, error) {

	pageDict := &PDFDict{
		Dict: map[string]Object{
			"Type":   Name("Page"),
			"Parent": *parentPageIndRef,
		},
	}

	fIndRef, err := createFontDict(xRefTable)
	if err != nil {
		return nil, err
	}

	fontResources := PDFDict{
		Dict: map[string]Object{
			"F1": *fIndRef,
		},
	}

	resourceDict := PDFDict{
		Dict: map[string]Object{
			"Font": fontResources,
		},
	}

	pageDict.Insert("Resources", resourceDict)

	err = addContents(xRefTable, pageDict, mediaBox)
	if err != nil {
		return nil, err
	}

	pageIndRef, err := xRefTable.IndRefForNewObject(*pageDict)
	if err != nil {
		return nil, err
	}

	return pageIndRef, nil
}

func addPageTree(xRefTable *XRefTable, rootDict *PDFDict) error {

	// mediabox = physical page dimensions
	//mediaBox := NewRectangle(0, 0, 595.27, 841.89)
	mediaBox := NewRectangle(0, 0, 400, 600)

	pagesDict := PDFDict{
		Dict: map[string]Object{
			"Type":     Name("Pages"),
			"Count":    Integer(1),
			"MediaBox": mediaBox,
		},
	}

	parentPageIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}

	pageIndRef, err := createPage(xRefTable, parentPageIndRef, &mediaBox)
	if err != nil {
		return err
	}

	pagesDict.Insert("Kids", Array{*pageIndRef})

	rootDict.Insert("Pages", *parentPageIndRef)

	return nil
}

func addPageTreeWithAnnotations(xRefTable *XRefTable, rootDict *PDFDict) (*IndirectRef, error) {

	// mediabox = physical page dimensions
	mediaBox := NewRectangle(0, 0, 595.27, 841.89)

	pagesDict := PDFDict{
		Dict: map[string]Object{
			"Type":     Name("Pages"),
			"Count":    Integer(1),
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

	pagesDict.Insert("Kids", Array{*pageIndRef})

	rootDict.Insert("Pages", *parentPageIndRef)

	return pageIndRef, nil
}

func addPageTreeWithAcroFields(xRefTable *XRefTable, rootDict *PDFDict, annotsArray *Array) (*IndirectRef, error) {

	// mediabox = physical page dimensions
	mediaBox := NewRectangle(0, 0, 595.27, 841.89)

	pagesDict := PDFDict{
		Dict: map[string]Object{
			"Type":     Name("Pages"),
			"Count":    Integer(1),
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

	pagesDict.Insert("Kids", Array{*pageIndRef})

	rootDict.Insert("Pages", *parentPageIndRef)

	return pageIndRef, nil
}

// create a thread with 2 beads.
func createThreadDict(xRefTable *XRefTable, pageIndRef *IndirectRef) (*IndirectRef, error) {

	infoDict := NewPDFDict()
	infoDict.InsertString("Title", "DummyArticle")

	d := PDFDict{
		Dict: map[string]Object{
			"Type": Name("Thread"),
			"I":    infoDict,
		},
	}

	dIndRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	// create first bead
	d1 := PDFDict{
		Dict: map[string]Object{
			"Type": Name("Bead"),
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
		Dict: map[string]Object{
			"Type": Name("Bead"),
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

func addThreads(xRefTable *XRefTable, rootDict *PDFDict, pageIndRef *IndirectRef) error {

	indRef, err := createThreadDict(xRefTable, pageIndRef)
	if err != nil {
		return err
	}

	indRef, err = xRefTable.IndRefForNewObject(Array{*indRef})
	if err != nil {
		return err
	}

	rootDict.Insert("Threads", *indRef)

	return nil
}

func addOpenAction(xRefTable *XRefTable, rootDict *PDFDict) error {

	nextActionDict := PDFDict{
		Dict: map[string]Object{
			"Type": Name("Action"),
			"S":    Name("Movie"),
			"T":    StringLiteral("Sample Movie"),
		},
	}

	script := `app.alert('Hello Gopher!');`

	d := PDFDict{
		Dict: map[string]Object{
			"Type": Name("Action"),
			"S":    Name("JavaScript"),
			"JS":   StringLiteral(script),
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

	arr := Array{}
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
		Dict: map[string]Object{
			"Event":    Name("View"),
			"OCGs":     Array{}, // of indRefs
			"Category": NewNameArray("Language"),
		},
	}

	optionalContentConfigDict := PDFDict{
		Dict: map[string]Object{
			"Name":      StringLiteral("OCConf"),
			"Creator":   StringLiteral("Horst Rutter"),
			"BaseState": Name("ON"),
			"OFF":       Array{},
			"Intent":    Name("Design"),
			"AS":        Array{usageAppDict},
			"Order":     Array{},
			"ListMode":  Name("AllPages"),
			"RBGroups":  Array{},
			"Locked":    Array{},
		},
	}

	d := PDFDict{
		Dict: map[string]Object{
			"OCGs":    Array{}, // of indRefs
			"D":       optionalContentConfigDict,
			"Configs": Array{optionalContentConfigDict},
		},
	}

	rootDict.Insert("OCProperties", d)

	return nil
}

func addRequirements(xRefTable *XRefTable, rootDict *PDFDict) {

	d := NewPDFDict()
	d.InsertName("Type", "Requirement")
	d.InsertName("S", "EnableJavaScripts")

	rootDict.Insert("Requirements", Array{d})
}

// CreateAnnotationDemoXRef creates a PDF file with examples of annotations and actions.
func CreateAnnotationDemoXRef() (*XRefTable, error) {

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

func createNormalAppearanceForFormField(xRefTable *XRefTable, w, h float64) (*IndirectRef, error) {

	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &StreamDict{
		PDFDict: PDFDict{
			Dict: map[string]Object{
				"Type":     Name("XObject"),
				"Subtype":  Name("Form"),
				"FormType": Integer(1),
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

func createRolloverAppearanceForFormField(xRefTable *XRefTable, w, h float64) (*IndirectRef, error) {

	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "1 0 0 RG 0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &StreamDict{
		PDFDict: PDFDict{
			Dict: map[string]Object{
				"Type":     Name("XObject"),
				"Subtype":  Name("Form"),
				"FormType": Integer(1),
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

func createDownAppearanceForFormField(xRefTable *XRefTable, w, h float64) (*IndirectRef, error) {

	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &StreamDict{
		PDFDict: PDFDict{
			Dict: map[string]Object{
				"Type":     Name("XObject"),
				"Subtype":  Name("Form"),
				"FormType": Integer(1),
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

func createTextField(xRefTable *XRefTable, pageAnnots *Array) (*IndirectRef, error) {

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
		Dict: map[string]Object{
			"Font": PDFDict{
				Dict: map[string]Object{
					"Helvetica": *fontDict,
				},
			},
		},
	}

	d := PDFDict{
		Dict: map[string]Object{
			"AP": PDFDict{
				Dict: map[string]Object{
					"N": *fN,
					"R": *fR,
					"D": *fD,
				},
			},
			"DA":      StringLiteral("/Helvetica 12 Tf 0 g"),
			"DR":      resourceDict,
			"FT":      Name("Tx"),
			"Rect":    NewRectangle(x, y, x+w, y+h),
			"Border":  NewIntegerArray(0, 0, 1),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"T":       StringLiteral("inputField"),
			"TU":      StringLiteral("inputField"),
			"DV":      StringLiteral("Default value"),
			"V":       StringLiteral("Default value"),
		},
	}

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	*pageAnnots = append(*pageAnnots, *indRef)

	return indRef, nil
}

func createYesAppearance(xRefTable *XRefTable, resourceDict *PDFDict, w, h float64) (*IndirectRef, error) {

	var b bytes.Buffer
	fmt.Fprintf(&b, "q 0 0 1 rg BT /ZaDb 12 Tf 0 0 Td (8) Tj ET Q")

	sd := &StreamDict{
		PDFDict: PDFDict{
			Dict: map[string]Object{
				"Resources": *resourceDict,
				"Subtype":   Name("Form"),
				"BBox":      NewRectangle(0, 0, w, h),
				"OPI": PDFDict{
					Dict: map[string]Object{
						"2.0": PDFDict{
							Dict: map[string]Object{
								"Type":    Name("OPI"),
								"Version": Float(2.0),
								"F":       StringLiteral("Proxy"),
								"Inks":    Name("full_color"),
							},
						},
					},
				},
				"Ref": PDFDict{
					Dict: map[string]Object{
						"F":    StringLiteral("Proxy"),
						"Page": Integer(1),
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

func createOffAppearance(xRefTable *XRefTable, resourceDict *PDFDict, w, h float64) (*IndirectRef, error) {

	var b bytes.Buffer
	fmt.Fprintf(&b, "q 0 0 1 rg BT /ZaDb 12 Tf 0 0 Td (4) Tj ET Q")

	sd := &StreamDict{
		PDFDict: PDFDict{
			Dict: map[string]Object{
				"Resources": *resourceDict,
				"Subtype":   Name("Form"),
				"BBox":      NewRectangle(0, 0, w, h),
				"OPI": PDFDict{
					Dict: map[string]Object{
						"1.3": PDFDict{
							Dict: map[string]Object{
								"Type":     Name("OPI"),
								"Version":  Float(1.3),
								"F":        StringLiteral("Proxy"),
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

func createCheckBoxButtonField(xRefTable *XRefTable, pageAnnots *Array) (*IndirectRef, error) {

	fontDict, err := createZapfDingbatsFontDict(xRefTable)
	if err != nil {
		return nil, err
	}

	resDict := PDFDict{
		Dict: map[string]Object{
			"Font": PDFDict{
				Dict: map[string]Object{
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
		Dict: map[string]Object{
			"N": PDFDict{
				Dict: map[string]Object{
					"Yes": *yesForm,
					"Off": *offForm,
				},
			},
		},
	}

	d := PDFDict{
		Dict: map[string]Object{
			"FT":      Name("Btn"),
			"Rect":    NewRectangle(250, 300, 270, 320),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"T":       StringLiteral("CheckBox"),
			"TU":      StringLiteral("CheckBox"),
			"V":       Name("Yes"),
			"AS":      Name("Yes"),
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

func createRadioButtonField(xRefTable *XRefTable, pageAnnots *Array) (*IndirectRef, error) {

	var flags uint32
	flags = setBit(flags, 16)

	d := PDFDict{
		Dict: map[string]Object{
			"FT":   Name("Btn"),
			"Ff":   Integer(flags),
			"Rect": NewRectangle(250, 400, 280, 420),
			//"Type":    Name("Annot"),
			//"Subtype": Name("Widget"),
			"T": StringLiteral("Credit card"),
			"V": Name("card1"),
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
		Dict: map[string]Object{
			"Font": PDFDict{
				Dict: map[string]Object{
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
		Dict: map[string]Object{
			"Rect":    NewRectangle(250, 400, 280, 420),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"Parent":  *indRef,
			"T":       StringLiteral("Radio1"),
			"TU":      StringLiteral("Radio1"),
			"AS":      Name("card1"),
			"AP": PDFDict{
				Dict: map[string]Object{
					"N": PDFDict{
						Dict: map[string]Object{
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
		Dict: map[string]Object{
			"Rect":    NewRectangle(300, 400, 330, 420),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"Parent":  *indRef,
			"T":       StringLiteral("Radio2"),
			"TU":      StringLiteral("Radio2"),
			"AS":      Name("Off"),
			"AP": PDFDict{
				Dict: map[string]Object{
					"N": PDFDict{
						Dict: map[string]Object{
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

	d.Insert("Kids", Array{*indRefR1, *indRefR2})

	*pageAnnots = append(*pageAnnots, *indRefR1)
	*pageAnnots = append(*pageAnnots, *indRefR2)

	return indRef, nil
}

func createResetButton(xRefTable *XRefTable, pageAnnots *Array) (*IndirectRef, error) {

	var flags uint32
	flags = setBit(flags, 17)

	fN, err := createNormalAppearanceForFormField(xRefTable, 20, 20)
	if err != nil {
		return nil, err
	}

	resetFormActionDict := PDFDict{
		Dict: map[string]Object{
			"Type":   Name("Action"),
			"S":      Name("ResetForm"),
			"Fields": NewStringArray("inputField"),
			"Flags":  Integer(0),
		},
	}

	d := PDFDict{
		Dict: map[string]Object{
			"FT":      Name("Btn"),
			"Ff":      Integer(flags),
			"Rect":    NewRectangle(100, 400, 120, 420),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"AP":      PDFDict{Dict: map[string]Object{"N": *fN}},
			"T":       StringLiteral("Reset"),
			"TU":      StringLiteral("Reset"),
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

func createSubmitButton(xRefTable *XRefTable, pageAnnots *Array) (*IndirectRef, error) {

	var flags uint32
	flags = setBit(flags, 17)

	fN, err := createNormalAppearanceForFormField(xRefTable, 20, 20)
	if err != nil {
		return nil, err
	}

	urlSpec := PDFDict{
		Dict: map[string]Object{
			"FS": Name("URL"),
			"F":  StringLiteral("http://www.me.com"),
		},
	}

	submitFormActionDict := PDFDict{
		Dict: map[string]Object{
			"Type":   Name("Action"),
			"S":      Name("SubmitForm"),
			"F":      urlSpec,
			"Fields": NewStringArray("inputField"),
			"Flags":  Integer(0),
		},
	}

	d := PDFDict{
		Dict: map[string]Object{
			"FT":      Name("Btn"),
			"Ff":      Integer(flags),
			"Rect":    NewRectangle(140, 400, 160, 420),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"AP":      PDFDict{Dict: map[string]Object{"N": *fN}},
			"T":       StringLiteral("Submit"),
			"TU":      StringLiteral("Submit"),
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

func streamObjForXFAElement(xRefTable *XRefTable, s string) (*IndirectRef, error) {

	sd := &StreamDict{
		PDFDict: PDFDict{
			Dict: map[string]Object{},
		},
		Content: []byte(s),
	}

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createXFAArray(xRefTable *XRefTable) (*Array, error) {

	sd1, err := streamObjForXFAElement(xRefTable, "<xdp:xdp xmlns:xdp=\"http://ns.adobe.com/xdp/\">")
	if err != nil {
		return nil, err
	}

	sd3, err := streamObjForXFAElement(xRefTable, "</xdp:xdp>")
	if err != nil {
		return nil, err
	}

	return &Array{
		StringLiteral("xdp:xdp"), *sd1,
		StringLiteral("/xdp:xdp"), *sd3,
	}, nil
}

func createAcroFormDict(xRefTable *XRefTable) (*PDFDict, *Array, error) {

	pageAnnots := Array{}

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
		Dict: map[string]Object{
			"Fields":          Array{*text, *checkBox, *radioButton, *resetButton, *submitButton}, // indRefs of fieldDicts
			"NeedAppearances": Boolean(true),
			"CO":              Array{*text},
			"XFA":             *xfaArr,
		},
	}

	return &d, &pageAnnots, nil
}

// CreateAcroFormDemoXRef creates a PDF file with an AcroForm example.
func CreateAcroFormDemoXRef() (*XRefTable, error) {

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
			Dict: map[string]Object{
				"FitWindow":    Boolean(true),
				"CenterWindow": Boolean(true),
			},
		},
	)

	return xRefTable, nil
}

// CreatePDF creates a PDF file for an xRefTable.
func CreatePDF(xRefTable *XRefTable, dirName, fileName string) error {

	config := NewDefaultConfiguration()

	ctx := &PDFContext{
		Configuration: config,
		XRefTable:     xRefTable,
		Write:         NewWriteContext(config.Eol),
	}

	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	return WritePDFFile(ctx)
}
