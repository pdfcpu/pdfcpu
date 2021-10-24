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
	"path/filepath"
)

var (
	testDir          = "../../testdata"
	testAudioFileWAV = filepath.Join(testDir, "resources", "test.wav")
)

func CreateXRefTableWithRootDict() (*XRefTable, error) {
	xRefTable := &XRefTable{
		Table:      map[int]*XRefTableEntry{},
		Names:      map[string]*Node{},
		PageAnnots: map[int]PgAnnots{},
		Stats:      NewPDFStats(),
	}

	xRefTable.Table[0] = NewFreeHeadXRefTableEntry()

	one := 1
	xRefTable.Size = &one

	v := V17
	xRefTable.HeaderVersion = &v

	xRefTable.PageCount = 0

	// Optional infoDict.
	xRefTable.Info = nil

	// Additional streams not implemented.
	xRefTable.AdditionalStreams = nil

	rootDict := NewDict()
	rootDict.InsertName("Type", "Catalog")

	ir, err := xRefTable.IndRefForNewObject(rootDict)
	if err != nil {
		return nil, err
	}

	xRefTable.Root = ir

	return xRefTable, nil
}

// CreateDemoXRef creates a minimal single page PDF file for demo purposes.
func CreateDemoXRef(p Page) (*XRefTable, error) {
	xRefTable, err := CreateXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	if err = addPageTreeWithSamplePage(xRefTable, rootDict, p); err != nil {
		return nil, err
	}

	return xRefTable, nil
}

func addPageTreeForResourceDictInheritanceDemo(xRefTable *XRefTable, rootDict Dict) error {

	// Create root page node.

	fIndRef, err := EnsureFontDict(xRefTable, "Courier", false, nil)
	if err != nil {
		return err
	}

	rootPagesDict := Dict(
		map[string]Object{
			"Type":     Name("Pages"),
			"Count":    Integer(1),
			"MediaBox": RectForFormat("A4").Array(),
			"Resources": Dict(
				map[string]Object{
					"Font": Dict(
						map[string]Object{
							"F99": *fIndRef,
						},
					),
				},
			),
		},
	)

	rootPageIndRef, err := xRefTable.IndRefForNewObject(rootPagesDict)
	if err != nil {
		return err
	}

	// Create intermediate page node.

	f100IndRef, err := EnsureFontDict(xRefTable, "Courier-Bold", false, nil)
	if err != nil {
		return err
	}

	pagesDict := Dict(
		map[string]Object{
			"Type":     Name("Pages"),
			"Count":    Integer(1),
			"MediaBox": RectForFormat("A4").Array(),
			"Resources": Dict(
				map[string]Object{
					"Font": Dict(
						map[string]Object{
							"F100": *f100IndRef,
						},
					),
				},
			),
		},
	)

	pagesIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}

	// Create leaf page node.

	p := Page{MediaBox: RectForFormat("A4"), Fm: FontMap{}, Buf: new(bytes.Buffer)}

	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := TextDescriptor{
		Text:     "This font is Times-Roman and it is defined in the resource dict of this page dict.",
		FontName: fontName,
		FontKey:  k,
		FontSize: 12,
		Scale:    1.,
		ScaleAbs: true,
		X:        300,
		Y:        400,
	}

	WriteMultiLine(p.Buf, p.MediaBox, nil, td)

	fontName = "Courier"
	td = TextDescriptor{
		Text:     "This font is Courier and it is inherited from the page root.",
		FontName: fontName,
		FontKey:  "F99",
		FontSize: 12,
		Scale:    1.,
		ScaleAbs: true,
		X:        300,
		Y:        300,
	}

	WriteMultiLine(p.Buf, p.MediaBox, nil, td)

	fontName = "Courier-Bold"
	td = TextDescriptor{
		Text:     "This font is Courier-Bold and it is inherited from an intermediate page node.",
		FontName: fontName,
		FontKey:  "F100",
		FontSize: 12,
		Scale:    1.,
		ScaleAbs: true,
		X:        300,
		Y:        350,
	}

	WriteMultiLine(p.Buf, p.MediaBox, nil, td)

	pageIndRef, err := createDemoPage(xRefTable, *pagesIndRef, p)
	if err != nil {
		return err
	}

	pagesDict.Insert("Kids", Array{*pageIndRef})
	pagesDict.Insert("Parent", *rootPageIndRef)

	rootPagesDict.Insert("Kids", Array{*pagesIndRef})
	rootDict.Insert("Pages", *rootPageIndRef)

	return nil
}

// CreateResourceDictInheritanceDemoXRef creates a page tree for testing resource dict inheritance.
func CreateResourceDictInheritanceDemoXRef() (*XRefTable, error) {
	xRefTable, err := CreateXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	if err = addPageTreeForResourceDictInheritanceDemo(xRefTable, rootDict); err != nil {
		return nil, err
	}

	return xRefTable, nil
}

func createFunctionalShadingDict(xRefTable *XRefTable) Dict {
	f := Dict(
		map[string]Object{
			"FunctionType": Integer(2),
			"Domain":       NewNumberArray(1.0, 1.2, 1.4, 1.6, 1.8, 2.0),
			"N":            Float(1),
		},
	)

	d := Dict(
		map[string]Object{
			"ShadingType": Integer(1),
			"Function":    Array{f},
		},
	)

	return d
}

func createRadialShadingDict(xRefTable *XRefTable) Dict {
	f := Dict(
		map[string]Object{
			"FunctionType": Integer(2),
			"Domain":       NewNumberArray(1.0, 1.2, 1.4, 1.6, 1.8, 2.0),
			"N":            Float(1),
		},
	)

	d := Dict(
		map[string]Object{
			"ShadingType": Integer(3),
			"Coords":      NewNumberArray(0, 0, 50, 10, 10, 100),
			"Function":    Array{f},
		},
	)

	return d
}

func createStreamObjForHalftoneDictType6(xRefTable *XRefTable) (*IndirectRef, error) {
	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":             Name("Halftone"),
				"HalftoneType":     Integer(6),
				"Width":            Integer(100),
				"Height":           Integer(100),
				"TransferFunction": Name("Identity"),
			},
		),
		Content: []byte{},
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createStreamObjForHalftoneDictType10(xRefTable *XRefTable) (*IndirectRef, error) {
	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":         Name("Halftone"),
				"HalftoneType": Integer(10),
				"Xsquare":      Integer(100),
				"Ysquare":      Integer(100),
			},
		),
		Content: []byte{},
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createStreamObjForHalftoneDictType16(xRefTable *XRefTable) (*IndirectRef, error) {
	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":         Name("Halftone"),
				"HalftoneType": Integer(16),
				"Width":        Integer(100),
				"Height":       Integer(100),
			},
		),
		Content: []byte{},
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createPostScriptCalculatorFunctionStreamDict(xRefTable *XRefTable) (*IndirectRef, error) {
	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"FunctionType": Integer(4),
				"Domain":       NewNumberArray(100.),
				"Range":        NewNumberArray(100.),
			},
		),
		Content: []byte{},
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func addResources(xRefTable *XRefTable, pageDict Dict, fontName string) error {
	fIndRef, err := EnsureFontDict(xRefTable, fontName, true, nil)
	if err != nil {
		return err
	}

	functionalBasedShDict := createFunctionalShadingDict(xRefTable)

	radialShDict := createRadialShadingDict(xRefTable)

	f := Dict(
		map[string]Object{
			"FunctionType": Integer(2),
			"Domain":       NewNumberArray(0.0, 1.0),
			"C0":           NewNumberArray(0.0),
			"C1":           NewNumberArray(1.0),
			"N":            Float(1),
		},
	)

	fontResources := Dict(
		map[string]Object{
			"F1": *fIndRef,
		},
	)

	shadingResources := Dict(
		map[string]Object{
			"S1": functionalBasedShDict,
			"S3": radialShDict,
		},
	)

	colorSpaceResources := Dict(
		map[string]Object{
			"CSCalGray": Array{
				Name("CalGray"),
				Dict(
					map[string]Object{
						"WhitePoint": NewNumberArray(0.9505, 1.0000, 1.0890),
					},
				),
			},
			"CSCalRGB": Array{
				Name("CalRGB"),
				Dict(
					map[string]Object{
						"WhitePoint": NewNumberArray(0.9505, 1.0000, 1.0890),
					},
				),
			},
			"CSLab": Array{
				Name("Lab"),
				Dict(
					map[string]Object{
						"WhitePoint": NewNumberArray(0.9505, 1.0000, 1.0890),
					},
				),
			},
			"CS4DeviceN": Array{
				Name("DeviceN"),
				NewNameArray("Orange", "Green", "None"),
				Name("DeviceCMYK"),
				f,
				Dict(
					map[string]Object{
						"Subtype": Name("DeviceN"),
					},
				),
			},
			"CS6DeviceN": Array{
				Name("DeviceN"),
				NewNameArray("L", "a", "b", "Spot1"),
				Name("DeviceCMYK"),
				f,
				Dict(
					map[string]Object{
						"Subtype": Name("NChannel"),
						"Process": Dict(
							map[string]Object{
								"ColorSpace": Array{
									Name("Lab"),
									Dict(
										map[string]Object{
											"WhitePoint": NewNumberArray(0.9505, 1.0000, 1.0890),
										},
									),
								},
								"Components": NewNameArray("L", "a", "b"),
							},
						),
						"Colorants": Dict(
							map[string]Object{
								"Spot1": Array{
									Name("Separation"),
									Name("Spot1"),
									Name("DeviceCMYK"),
									f,
								},
							},
						),
						"MixingHints": Dict(
							map[string]Object{
								"Solidities": Dict(
									map[string]Object{
										"Spot1": Float(1.0),
									},
								),
								"DotGain": Dict(
									map[string]Object{
										"Spot1":   f,
										"Magenta": f,
										"Yellow":  f,
									},
								),
								"PrintingOrder": NewNameArray("Magenta", "Yellow", "Spot1"),
							},
						),
					},
				),
			},
		},
	)

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

	graphicStateResources := Dict(
		map[string]Object{
			"GS1": Dict(
				map[string]Object{
					"Type": Name("ExtGState"),
					"HT": Dict(
						map[string]Object{
							"Type":             Name("Halftone"),
							"HalftoneType":     Integer(1),
							"Frequency":        Integer(120),
							"Angle":            Integer(30),
							"SpotFunction":     Name("CosineDot"),
							"TransferFunction": Name("Identity"),
						},
					),
					"BM": NewNameArray("Overlay", "Darken", "Normal"),
					"SMask": Dict(
						map[string]Object{
							"Type": Name("Mask"),
							"S":    Name("Alpha"),
							"G":    *anyXObject,
							"TR":   f,
						},
					),
					"TR":  f,
					"TR2": f,
				},
			),
			"GS2": Dict(
				map[string]Object{
					"Type": Name("ExtGState"),
					"HT": Dict(
						map[string]Object{
							"Type":         Name("Halftone"),
							"HalftoneType": Integer(5),
							"Default": Dict(
								map[string]Object{
									"Type":             Name("Halftone"),
									"HalftoneType":     Integer(1),
									"Frequency":        Integer(120),
									"Angle":            Integer(30),
									"SpotFunction":     Name("CosineDot"),
									"TransferFunction": Name("Identity"),
								},
							),
						},
					),
					"BM": NewNameArray("Overlay", "Darken", "Normal"),
					"SMask": Dict(
						map[string]Object{
							"Type": Name("Mask"),
							"S":    Name("Alpha"),
							"G":    *anyXObject,
							"TR":   Name("Identity"),
						},
					),
					"TR":   Array{f, f, f, f},
					"TR2":  Array{f, f, f, f},
					"BG2":  f,
					"UCR2": f,
					"D":    Array{Array{}, Integer(0)},
				},
			),
			"GS3": Dict(
				map[string]Object{
					"Type": Name("ExtGState"),
					"HT":   *indRefHalfToneType6,
					"SMask": Dict(
						map[string]Object{
							"Type": Name("Mask"),
							"S":    Name("Alpha"),
							"G":    *anyXObject,
							"TR":   *indRefFunctionStream,
						},
					),
					"BG2":  *indRefFunctionStream,
					"UCR2": *indRefFunctionStream,
					"TR":   *indRefFunctionStream,
					"TR2":  *indRefFunctionStream,
				},
			),
			"GS4": Dict(
				map[string]Object{
					"Type": Name("ExtGState"),
					"HT":   *indRefHalfToneType10,
				},
			),
			"GS5": Dict(
				map[string]Object{
					"Type": Name("ExtGState"),
					"HT":   *indRefHalfToneType16,
				},
			),
			"GS6": Dict(
				map[string]Object{
					"Type": Name("ExtGState"),
					"HT": Dict(
						map[string]Object{
							"Type":         Name("Halftone"),
							"HalftoneType": Integer(1),
							"Frequency":    Integer(120),
							"Angle":        Integer(30),
							"SpotFunction": *indRefFunctionStream,
						},
					),
				},
			),
			"GS7": Dict(
				map[string]Object{
					"Type": Name("ExtGState"),
					"HT": Dict(
						map[string]Object{
							"Type":         Name("Halftone"),
							"HalftoneType": Integer(1),
							"Frequency":    Integer(120),
							"Angle":        Integer(30),
							"SpotFunction": f,
						},
					),
				},
			),
		},
	)

	resourceDict := Dict(
		map[string]Object{
			"Font":       fontResources,
			"Shading":    shadingResources,
			"ColorSpace": colorSpaceResources,
			"ExtGState":  graphicStateResources,
		},
	)

	pageDict.Insert("Resources", resourceDict)

	return nil
}

// CreateTestPageContent draws a test grid.
func CreateTestPageContent(p Page) {
	b := p.Buf
	mb := p.MediaBox

	b.WriteString("[3]0 d 0 w ")

	// X
	fmt.Fprintf(b, "0 0 m %f %f l s %f 0 m 0 %f l s ",
		mb.Width(), mb.Height(), mb.Width(), mb.Height())

	// Horizontal guides
	c := 6
	if mb.Landscape() {
		c = 4
	}
	j := mb.Height() / float64(c)
	for i := 1; i < c; i++ {
		k := mb.Height() - float64(i)*j
		s := fmt.Sprintf("0 %f m %f %f l s ", k, mb.Width(), k)
		b.WriteString(s)
	}

	// Vertical guides
	c = 4
	if mb.Landscape() {
		c = 6
	}
	j = mb.Width() / float64(c)
	for i := 1; i < c; i++ {
		k := float64(i) * j
		s := fmt.Sprintf("%f 0 m %f %f l s ", k, k, mb.Height())
		b.WriteString(s)
	}
}

func addContents(xRefTable *XRefTable, pageDict Dict, p Page) error {
	CreateTestPageContent(p)
	sd, _ := xRefTable.NewStreamDictForBuf(p.Buf.Bytes())

	if err := sd.Encode(); err != nil {
		return err
	}

	ir, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	pageDict.Insert("Contents", *ir)

	return nil
}

func createBoxColorDict() Dict {
	cropBoxColorInfoDict := Dict(
		map[string]Object{
			"C": NewNumberArray(1.0, 1.0, 0.0),
			"W": Float(1.0),
			"S": Name("D"),
			"D": NewIntegerArray(3, 2),
		},
	)
	bleedBoxColorInfoDict := Dict(
		map[string]Object{
			"C": NewNumberArray(1.0, 0.0, 0.0),
			"W": Float(3.0),
			"S": Name("S"),
		},
	)
	trimBoxColorInfoDict := Dict(
		map[string]Object{
			"C": NewNumberArray(0.0, 1.0, 0.0),
			"W": Float(1.0),
			"S": Name("D"),
			"D": NewIntegerArray(3, 2),
		},
	)
	artBoxColorInfoDict := Dict(
		map[string]Object{
			"C": NewNumberArray(0.0, 0.0, 1.0),
			"W": Float(1.0),
			"S": Name("S"),
		},
	)
	d := Dict(
		map[string]Object{
			"CropBox":  cropBoxColorInfoDict,
			"BleedBox": bleedBoxColorInfoDict,
			"Trim":     trimBoxColorInfoDict,
			"ArtBox":   artBoxColorInfoDict,
		},
	)
	return d
}

func addViewportDict(pageDict Dict) {
	measureDict := Dict(
		map[string]Object{
			"Type":    Name("Measure"),
			"Subtype": Name("RL"),
			"R":       StringLiteral("1in = 0.1m"),
			"X": Array{
				Dict(
					map[string]Object{
						"Type": Name("NumberFormat"),
						"U":    StringLiteral("mi"),
						"C":    Float(0.00139),
						"D":    Integer(100000),
					},
				),
			},
			"D": Array{
				Dict(
					map[string]Object{
						"Type": Name("NumberFormat"),
						"U":    StringLiteral("mi"),
						"C":    Float(1),
					},
				),
				Dict(
					map[string]Object{
						"Type": Name("NumberFormat"),
						"U":    StringLiteral("feet"),
						"C":    Float(5280),
					},
				),
				Dict(
					map[string]Object{
						"Type": Name("NumberFormat"),
						"U":    StringLiteral("inch"),
						"C":    Float(12),
						"F":    Name("F"),
						"D":    Integer(8),
					},
				),
			},
			"A": Array{
				Dict(
					map[string]Object{
						"Type": Name("NumberFormat"),
						"U":    StringLiteral("acres"),
						"C":    Float(640),
					},
				),
			},
			"O": NewIntegerArray(0, 1),
		},
	)

	bbox := RectForDim(10, 60)

	vpDict := Dict(
		map[string]Object{
			"Type":    Name("Viewport"),
			"BBox":    bbox.Array(),
			"Name":    StringLiteral("viewPort"),
			"Measure": measureDict,
		},
	)

	pageDict.Insert("VP", Array{vpDict})
}

func annotRect(i int, w, h, d, l float64) *Rectangle {
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

	return Rect(llx, lly, urx, ury)
}

// createAnnotsArray generates side by side lined up annotations starting in the lower left corner of the page.
func createAnnotsArray(xRefTable *XRefTable, pageIndRef IndirectRef, mediaBox Array) (Array, error) {
	pageWidth := mediaBox[2].(Float)
	pageHeight := mediaBox[3].(Float)

	a := Array{}

	for i, f := range []func(*XRefTable, IndirectRef, Array) (*IndirectRef, error){
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

		ir, err := f(xRefTable, pageIndRef, r.Array())
		if err != nil {
			return nil, err
		}

		a = append(a, *ir)
	}

	return a, nil
}

func createPageWithAnnotations(xRefTable *XRefTable, parentPageIndRef IndirectRef, mediaBox *Rectangle, fontName string) (*IndirectRef, error) {
	mba := mediaBox.Array()

	pageDict := Dict(
		map[string]Object{
			"Type":         Name("Page"),
			"Parent":       parentPageIndRef,
			"BleedBox":     mba,
			"TrimBox":      mba,
			"ArtBox":       mba,
			"BoxColorInfo": createBoxColorDict(),
			"UserUnit":     Float(1.5)}, // Note: not honored by Apple Preview
	)

	err := addResources(xRefTable, pageDict, fontName)
	if err != nil {
		return nil, err
	}

	p := Page{MediaBox: mediaBox, Buf: new(bytes.Buffer)}
	err = addContents(xRefTable, pageDict, p)
	if err != nil {
		return nil, err
	}

	pageIndRef, err := xRefTable.IndRefForNewObject(pageDict)
	if err != nil {
		return nil, err
	}

	// Fake SeparationInfo related to a single page only.
	separationInfoDict := Dict(
		map[string]Object{
			"Pages":          Array{*pageIndRef},
			"DeviceColorant": Name("Cyan"),
			"ColorSpace": Array{
				Name("Separation"),
				Name("Green"),
				Name("DeviceCMYK"),
				Dict(
					map[string]Object{
						"FunctionType": Integer(2),
						"Domain":       NewNumberArray(0.0, 1.0),
						"C0":           NewNumberArray(0.0),
						"C1":           NewNumberArray(1.0),
						"N":            Float(1),
					},
				),
			},
		},
	)
	pageDict.Insert("SeparationInfo", separationInfoDict)

	annotsArray, err := createAnnotsArray(xRefTable, *pageIndRef, mba)
	if err != nil {
		return nil, err
	}
	pageDict.Insert("Annots", annotsArray)

	addViewportDict(pageDict)

	return pageIndRef, nil
}

func createPageWithAcroForm(xRefTable *XRefTable, parentPageIndRef IndirectRef, annotsArray Array, mediaBox *Rectangle, fontName string) (*IndirectRef, error) {
	mba := mediaBox.Array()

	pageDict := Dict(
		map[string]Object{
			"Type":         Name("Page"),
			"Parent":       parentPageIndRef,
			"BleedBox":     mba,
			"TrimBox":      mba,
			"ArtBox":       mba,
			"BoxColorInfo": createBoxColorDict(),
			"UserUnit":     Float(1.0), // Note: not honored by Apple Preview
		},
	)

	err := addResources(xRefTable, pageDict, fontName)
	if err != nil {
		return nil, err
	}

	p := Page{MediaBox: mediaBox, Buf: new(bytes.Buffer)}
	err = addContents(xRefTable, pageDict, p)
	if err != nil {
		return nil, err
	}

	pageDict.Insert("Annots", annotsArray)

	return xRefTable.IndRefForNewObject(pageDict)
}

func addPageTreeWithoutPage(xRefTable *XRefTable, rootDict Dict, d *Dim) error {
	// May be modified later on.
	mediaBox := RectForDim(d.Width, d.Height)

	pagesDict := Dict(
		map[string]Object{
			"Type":     Name("Pages"),
			"Count":    Integer(0),
			"MediaBox": mediaBox.Array(),
		},
	)

	pagesDict.Insert("Kids", Array{})

	pagesRootIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}

	rootDict.Insert("Pages", *pagesRootIndRef)

	return nil
}

func addPageTreeWithSamplePage(xRefTable *XRefTable, rootDict Dict, p Page) error {

	// mediabox = physical page dimensions
	mba := p.MediaBox.Array()

	pagesDict := Dict(
		map[string]Object{
			"Type":     Name("Pages"),
			"Count":    Integer(1),
			"MediaBox": mba,
		},
	)

	parentPageIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}

	pageIndRef, err := createDemoPage(xRefTable, *parentPageIndRef, p)
	if err != nil {
		return err
	}

	pagesDict.Insert("Kids", Array{*pageIndRef})
	rootDict.Insert("Pages", *parentPageIndRef)

	return nil
}

func addPageTreeWithAnnotations(xRefTable *XRefTable, rootDict Dict, fontName string) (*IndirectRef, error) {
	// mediabox = physical page dimensions
	mediaBox := RectForFormat("A4")
	mba := mediaBox.Array()

	pagesDict := Dict(
		map[string]Object{
			"Type":     Name("Pages"),
			"Count":    Integer(1),
			"MediaBox": mba,
			"CropBox":  mba,
		},
	)

	parentPageIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return nil, err
	}

	pageIndRef, err := createPageWithAnnotations(xRefTable, *parentPageIndRef, mediaBox, fontName)
	if err != nil {
		return nil, err
	}

	pagesDict.Insert("Kids", Array{*pageIndRef})
	rootDict.Insert("Pages", *parentPageIndRef)

	return pageIndRef, nil
}

func addPageTreeWithAcroFields(xRefTable *XRefTable, rootDict Dict, annotsArray Array, fontName string) (*IndirectRef, error) {
	// mediabox = physical page dimensions
	mediaBox := RectForFormat("A4")
	mba := mediaBox.Array()

	pagesDict := Dict(
		map[string]Object{
			"Type":     Name("Pages"),
			"Count":    Integer(1),
			"MediaBox": mba,
			"CropBox":  mba,
		},
	)

	parentPageIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return nil, err
	}

	pageIndRef, err := createPageWithAcroForm(xRefTable, *parentPageIndRef, annotsArray, mediaBox, fontName)
	if err != nil {
		return nil, err
	}

	pagesDict.Insert("Kids", Array{*pageIndRef})

	rootDict.Insert("Pages", *parentPageIndRef)

	return pageIndRef, nil
}

// create a thread with 2 beads.
func createThreadDict(xRefTable *XRefTable, pageIndRef IndirectRef) (*IndirectRef, error) {
	infoDict := NewDict()
	infoDict.InsertString("Title", "DummyArticle")

	d := Dict(
		map[string]Object{
			"Type": Name("Thread"),
			"I":    infoDict,
		},
	)

	dIndRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	// create first bead
	d1 := Dict(
		map[string]Object{
			"Type": Name("Bead"),
			"T":    *dIndRef,
			"P":    pageIndRef,
			"R":    NewNumberArray(0, 0, 100, 100),
		},
	)

	d1IndRef, err := xRefTable.IndRefForNewObject(d1)
	if err != nil {
		return nil, err
	}

	d.Insert("F", *d1IndRef)

	// create last bead
	d2 := Dict(
		map[string]Object{
			"Type": Name("Bead"),
			"T":    *dIndRef,
			"N":    *d1IndRef,
			"V":    *d1IndRef,
			"P":    pageIndRef,
			"R":    NewNumberArray(0, 100, 200, 100),
		},
	)

	d2IndRef, err := xRefTable.IndRefForNewObject(d2)
	if err != nil {
		return nil, err
	}

	d1.Insert("N", *d2IndRef)
	d1.Insert("V", *d2IndRef)

	return dIndRef, nil
}

func addThreads(xRefTable *XRefTable, rootDict Dict, pageIndRef IndirectRef) error {
	ir, err := createThreadDict(xRefTable, pageIndRef)
	if err != nil {
		return err
	}

	ir, err = xRefTable.IndRefForNewObject(Array{*ir})
	if err != nil {
		return err
	}

	rootDict.Insert("Threads", *ir)

	return nil
}

func addOpenAction(xRefTable *XRefTable, rootDict Dict) error {
	nextActionDict := Dict(
		map[string]Object{
			"Type": Name("Action"),
			"S":    Name("Movie"),
			"T":    StringLiteral("Sample Movie"),
		},
	)

	script := `app.alert('Hello Gopher!');`

	d := Dict(
		map[string]Object{
			"Type": Name("Action"),
			"S":    Name("JavaScript"),
			"JS":   StringLiteral(script),
			"Next": nextActionDict,
		},
	)

	rootDict.Insert("OpenAction", d)

	return nil
}

func addURI(xRefTable *XRefTable, rootDict Dict) {
	d := NewDict()
	d.InsertString("Base", "http://www.adobe.com")

	rootDict.Insert("URI", d)
}

func addSpiderInfo(xRefTable *XRefTable, rootDict Dict) error {
	// webCaptureInfoDict
	webCaptureInfoDict := NewDict()
	webCaptureInfoDict.InsertInt("V", 1.0)

	a := Array{}
	captureCmdDict := NewDict()
	captureCmdDict.InsertString("URL", (""))

	cmdSettingsDict := NewDict()
	captureCmdDict.Insert("S", cmdSettingsDict)

	ir, err := xRefTable.IndRefForNewObject(captureCmdDict)
	if err != nil {
		return err
	}

	a = append(a, *ir)

	webCaptureInfoDict.Insert("C", a)

	ir, err = xRefTable.IndRefForNewObject(webCaptureInfoDict)
	if err != nil {
		return err
	}

	rootDict.Insert("SpiderInfo", *ir)

	return nil
}

func addOCProperties(xRefTable *XRefTable, rootDict Dict) error {
	usageAppDict := Dict(
		map[string]Object{
			"Event":    Name("View"),
			"OCGs":     Array{}, // of indRefs
			"Category": NewNameArray("Language"),
		},
	)

	optionalContentConfigDict := Dict(
		map[string]Object{
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
	)

	d := Dict(
		map[string]Object{
			"OCGs":    Array{}, // of indRefs
			"D":       optionalContentConfigDict,
			"Configs": Array{optionalContentConfigDict},
		},
	)

	rootDict.Insert("OCProperties", d)

	return nil
}

func addRequirements(xRefTable *XRefTable, rootDict Dict) {
	d := NewDict()
	d.InsertName("Type", "Requirement")
	d.InsertName("S", "EnableJavaScripts")

	rootDict.Insert("Requirements", Array{d})
}

// CreateAnnotationDemoXRef creates a PDF file with examples of annotations and actions.
func CreateAnnotationDemoXRef() (*XRefTable, error) {
	fontName := "Helvetica"

	xRefTable, err := CreateXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	pageIndRef, err := addPageTreeWithAnnotations(xRefTable, rootDict, fontName)
	if err != nil {
		return nil, err
	}

	err = addThreads(xRefTable, rootDict, *pageIndRef)
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
		Dict: Dict(
			map[string]Object{
				"Type":     Name("XObject"),
				"Subtype":  Name("Form"),
				"FormType": Integer(1),
				"BBox":     NewNumberArray(0, 0, w, h),
				"Matrix":   NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		),
		Content: b.Bytes(),
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createRolloverAppearanceForFormField(xRefTable *XRefTable, w, h float64) (*IndirectRef, error) {
	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "1 0 0 RG 0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":     Name("XObject"),
				"Subtype":  Name("Form"),
				"FormType": Integer(1),
				"BBox":     NewNumberArray(0, 0, w, h),
				"Matrix":   NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		),
		Content: b.Bytes(),
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createDownAppearanceForFormField(xRefTable *XRefTable, w, h float64) (*IndirectRef, error) {
	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":     Name("XObject"),
				"Subtype":  Name("Form"),
				"FormType": Integer(1),
				"BBox":     NewNumberArray(0, 0, w, h),
				"Matrix":   NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		),
		Content: b.Bytes(),
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createFormTextField(xRefTable *XRefTable, pageAnnots *Array, fontName string) (*IndirectRef, error) {
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

	fontDict, err := EnsureFontDict(xRefTable, fontName, true, nil)
	if err != nil {
		return nil, err
	}

	resourceDict := Dict(
		map[string]Object{
			"Font": Dict(
				map[string]Object{
					fontName: *fontDict,
				},
			),
		},
	)

	d := Dict(
		map[string]Object{
			"AP": Dict(
				map[string]Object{
					"N": *fN,
					"R": *fR,
					"D": *fD,
				},
			),
			"DA":      StringLiteral("/" + fontName + " 12 Tf 0 g"),
			"DR":      resourceDict,
			"FT":      Name("Tx"),
			"Rect":    NewNumberArray(x, y, x+w, y+h),
			"Border":  NewIntegerArray(0, 0, 1),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"T":       StringLiteral("inputField"),
			"TU":      StringLiteral("inputField"),
			"DV":      StringLiteral("Default value"),
			"V":       StringLiteral("Default value"),
		},
	)

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	*pageAnnots = append(*pageAnnots, *ir)

	return ir, nil
}

func createYesAppearance(xRefTable *XRefTable, resourceDict Dict, w, h float64) (*IndirectRef, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "q 0 0 1 rg BT /ZaDb 12 Tf 0 0 Td (8) Tj ET Q")

	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Resources": resourceDict,
				"Subtype":   Name("Form"),
				"BBox":      NewNumberArray(0, 0, w, h),
				"OPI": Dict(
					map[string]Object{
						"2.0": Dict(
							map[string]Object{
								"Type":    Name("OPI"),
								"Version": Float(2.0),
								"F":       StringLiteral("Proxy"),
								"Inks":    Name("full_color"),
							},
						),
					},
				),
				"Ref": Dict(
					map[string]Object{
						"F":    StringLiteral("Proxy"),
						"Page": Integer(1),
					},
				),
			},
		),
		Content: b.Bytes(),
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createOffAppearance(xRefTable *XRefTable, resourceDict Dict, w, h float64) (*IndirectRef, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "q 0 0 1 rg BT /ZaDb 12 Tf 0 0 Td (4) Tj ET Q")

	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Resources": resourceDict,
				"Subtype":   Name("Form"),
				"BBox":      NewNumberArray(0, 0, w, h),
				"OPI": Dict(
					map[string]Object{
						"1.3": Dict(
							map[string]Object{
								"Type":     Name("OPI"),
								"Version":  Float(1.3),
								"F":        StringLiteral("Proxy"),
								"Size":     NewIntegerArray(400, 400),
								"CropRect": NewIntegerArray(0, 400, 400, 0),
								"Position": NewNumberArray(0, 0, 0, 400, 400, 400, 400, 0),
							},
						),
					},
				),
			},
		),
		Content: b.Bytes(),
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createCheckBoxButtonField(xRefTable *XRefTable, pageAnnots *Array) (*IndirectRef, error) {
	fontDict, err := EnsureFontDict(xRefTable, "ZapfDingbats", false, nil)
	if err != nil {
		return nil, err
	}

	resDict := Dict(
		map[string]Object{
			"Font": Dict(
				map[string]Object{
					"ZaDb": *fontDict,
				},
			),
		},
	)

	yesForm, err := createYesAppearance(xRefTable, resDict, 20.0, 20.0)
	if err != nil {
		return nil, err
	}

	offForm, err := createOffAppearance(xRefTable, resDict, 20.0, 20.0)
	if err != nil {
		return nil, err
	}

	apDict := Dict(
		map[string]Object{
			"N": Dict(
				map[string]Object{
					"Yes": *yesForm,
					"Off": *offForm,
				},
			),
		},
	)

	d := Dict(
		map[string]Object{
			"FT":      Name("Btn"),
			"Rect":    NewNumberArray(250, 300, 270, 320),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"T":       StringLiteral("CheckBox"),
			"TU":      StringLiteral("CheckBox"),
			"V":       Name("Yes"),
			"AS":      Name("Yes"),
			"AP":      apDict,
		},
	)

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	*pageAnnots = append(*pageAnnots, *ir)

	return ir, nil
}

func createRadioButtonField(xRefTable *XRefTable, pageAnnots *Array) (*IndirectRef, error) {
	var flags uint32
	flags = setBit(flags, 16)

	d := Dict(
		map[string]Object{
			"FT":   Name("Btn"),
			"Ff":   Integer(flags),
			"Rect": NewNumberArray(250, 400, 280, 420),
			//"Type":    Name("Annot"),
			//"Subtype": Name("Widget"),
			"T": StringLiteral("Credit card"),
			"V": Name("card1"),
		},
	)

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	fontDict, err := EnsureFontDict(xRefTable, "ZapfDingbats", false, nil)
	if err != nil {
		return nil, err
	}

	resDict := Dict(
		map[string]Object{
			"Font": Dict(
				map[string]Object{
					"ZaDb": *fontDict,
				},
			),
		},
	)

	selectedForm, err := createYesAppearance(xRefTable, resDict, 20.0, 20.0)
	if err != nil {
		return nil, err
	}

	offForm, err := createOffAppearance(xRefTable, resDict, 20.0, 20.0)
	if err != nil {
		return nil, err
	}

	r1 := Dict(
		map[string]Object{
			"Rect":    NewNumberArray(250, 400, 280, 420),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"Parent":  *indRef,
			"T":       StringLiteral("Radio1"),
			"TU":      StringLiteral("Radio1"),
			"AS":      Name("card1"),
			"AP": Dict(
				map[string]Object{
					"N": Dict(
						map[string]Object{
							"card1": *selectedForm,
							"Off":   *offForm,
						},
					),
				},
			),
		},
	)

	indRefR1, err := xRefTable.IndRefForNewObject(r1)
	if err != nil {
		return nil, err
	}

	r2 := Dict(
		map[string]Object{
			"Rect":    NewNumberArray(300, 400, 330, 420),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"Parent":  *indRef,
			"T":       StringLiteral("Radio2"),
			"TU":      StringLiteral("Radio2"),
			"AS":      Name("Off"),
			"AP": Dict(
				map[string]Object{
					"N": Dict(
						map[string]Object{
							"card2": *selectedForm,
							"Off":   *offForm,
						},
					),
				},
			),
		},
	)

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

	resetFormActionDict := Dict(
		map[string]Object{
			"Type":   Name("Action"),
			"S":      Name("ResetForm"),
			"Fields": NewStringArray("inputField"),
			"Flags":  Integer(0),
		},
	)

	d := Dict(
		map[string]Object{
			"FT":      Name("Btn"),
			"Ff":      Integer(flags),
			"Rect":    NewNumberArray(100, 400, 120, 420),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"AP":      Dict(map[string]Object{"N": *fN}),
			"T":       StringLiteral("Reset"),
			"TU":      StringLiteral("Reset"),
			"A":       resetFormActionDict,
		},
	)

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	*pageAnnots = append(*pageAnnots, *ir)

	return ir, nil
}

func createSubmitButton(xRefTable *XRefTable, pageAnnots *Array) (*IndirectRef, error) {
	var flags uint32
	flags = setBit(flags, 17)

	fN, err := createNormalAppearanceForFormField(xRefTable, 20, 20)
	if err != nil {
		return nil, err
	}

	urlSpec := Dict(
		map[string]Object{
			"FS": Name("URL"),
			"F":  StringLiteral("http://www.me.com"),
		},
	)

	submitFormActionDict := Dict(
		map[string]Object{
			"Type":   Name("Action"),
			"S":      Name("SubmitForm"),
			"F":      urlSpec,
			"Fields": NewStringArray("inputField"),
			"Flags":  Integer(0),
		},
	)

	d := Dict(
		map[string]Object{
			"FT":      Name("Btn"),
			"Ff":      Integer(flags),
			"Rect":    NewNumberArray(140, 400, 160, 420),
			"Type":    Name("Annot"),
			"Subtype": Name("Widget"),
			"AP":      Dict(map[string]Object{"N": *fN}),
			"T":       StringLiteral("Submit"),
			"TU":      StringLiteral("Submit"),
			"A":       submitFormActionDict,
		},
	)

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	*pageAnnots = append(*pageAnnots, *ir)

	return ir, nil
}

func streamObjForXFAElement(xRefTable *XRefTable, s string) (*IndirectRef, error) {
	sd := &StreamDict{
		Dict:    Dict(map[string]Object{}),
		Content: []byte(s),
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createXFAArray(xRefTable *XRefTable) (Array, error) {
	sd1, err := streamObjForXFAElement(xRefTable, "<xdp:xdp xmlns:xdp=\"http://ns.adobe.com/xdp/\">")
	if err != nil {
		return nil, err
	}

	sd3, err := streamObjForXFAElement(xRefTable, "</xdp:xdp>")
	if err != nil {
		return nil, err
	}

	return Array{
		StringLiteral("xdp:xdp"), *sd1,
		StringLiteral("/xdp:xdp"), *sd3,
	}, nil
}

func createAcroFormDict(xRefTable *XRefTable, fontName string) (Dict, Array, error) {
	pageAnnots := Array{}

	text, err := createFormTextField(xRefTable, &pageAnnots, fontName)
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

	d := Dict(
		map[string]Object{
			"Fields":          Array{*text, *checkBox, *radioButton, *resetButton, *submitButton}, // indRefs of fieldDicts
			"NeedAppearances": Boolean(true),
			"CO":              Array{*text},
			"XFA":             xfaArr,
		},
	)

	return d, pageAnnots, nil
}

// CreateAcroFormDemoXRef creates an xRefTable with an AcroForm example.
func CreateAcroFormDemoXRef() (*XRefTable, error) {
	fontName := "Helvetica"

	xRefTable, err := CreateXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	acroFormDict, annotsArray, err := createAcroFormDict(xRefTable, fontName)
	if err != nil {
		return nil, err
	}

	rootDict.Insert("AcroForm", acroFormDict)

	_, err = addPageTreeWithAcroFields(xRefTable, rootDict, annotsArray, fontName)
	if err != nil {
		return nil, err
	}

	rootDict.Insert("ViewerPreferences",
		Dict(
			map[string]Object{
				"FitWindow":    Boolean(true),
				"CenterWindow": Boolean(true),
			},
		),
	)

	return xRefTable, nil
}

// CreateContext creates a Context for given cross reference table and configuration.
func CreateContext(xRefTable *XRefTable, conf *Configuration) *Context {
	if conf == nil {
		conf = NewDefaultConfiguration()
	}
	xRefTable.ValidationMode = conf.ValidationMode
	return &Context{
		Configuration: conf,
		XRefTable:     xRefTable,
		Write:         NewWriteContext(conf.Eol),
	}
}

// CreateContextWithXRefTable creates a Context with an xRefTable without pages for given configuration.
func CreateContextWithXRefTable(conf *Configuration, pageDim *Dim) (*Context, error) {
	xRefTable, err := CreateXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	if err = addPageTreeWithoutPage(xRefTable, rootDict, pageDim); err != nil {
		return nil, err
	}

	return CreateContext(xRefTable, conf), nil
}

func createDemoContentStreamDict(xRefTable *XRefTable, pageDict Dict, b []byte) (*IndirectRef, error) {
	sd, _ := xRefTable.NewStreamDictForBuf(b)
	if err := sd.Encode(); err != nil {
		return nil, err
	}
	return xRefTable.IndRefForNewObject(*sd)
}

func fontResources(xRefTable *XRefTable, fm FontMap) (Dict, error) {

	d := Dict{}

	for fontName, font := range fm {
		ir, err := EnsureFontDict(xRefTable, fontName, true, nil)
		if err != nil {
			return nil, err
		}
		d.Insert(font.Res.ID, *ir)
	}

	return d, nil
}

func createDemoPage(xRefTable *XRefTable, parentPageIndRef IndirectRef, p Page) (*IndirectRef, error) {

	pageDict := Dict(
		map[string]Object{
			"Type":   Name("Page"),
			"Parent": parentPageIndRef,
		},
	)

	fontRes, err := fontResources(xRefTable, p.Fm)
	if err != nil {
		return nil, err
	}

	if len(fontRes) > 0 {
		resDict := Dict(
			map[string]Object{
				"Font": fontRes,
			},
		)
		pageDict.Insert("Resources", resDict)
	}

	ir, err := createDemoContentStreamDict(xRefTable, pageDict, p.Buf.Bytes())
	if err != nil {
		return nil, err
	}
	pageDict.Insert("Contents", *ir)

	return xRefTable.IndRefForNewObject(pageDict)
}
