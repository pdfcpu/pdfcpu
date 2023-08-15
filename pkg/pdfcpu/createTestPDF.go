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

	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var (
	testDir          = "../../testdata"
	testAudioFileWAV = filepath.Join(testDir, "resources", "test.wav")
)

func CreateXRefTableWithRootDict() (*model.XRefTable, error) {
	xRefTable := &model.XRefTable{
		Table:      map[int]*model.XRefTableEntry{},
		Names:      map[string]*model.Node{},
		PageAnnots: map[int]model.PgAnnots{},
		Stats:      model.NewPDFStats(),
		URIs:       map[int]map[string]string{},
		UsedGIDs:   map[string]map[uint16]bool{},
	}

	xRefTable.Table[0] = model.NewFreeHeadXRefTableEntry()

	one := 1
	xRefTable.Size = &one

	v := model.V17
	xRefTable.HeaderVersion = &v

	xRefTable.PageCount = 0

	// Optional infoDict.
	xRefTable.Info = nil

	// Additional streams not implemented.
	xRefTable.AdditionalStreams = nil

	rootDict := types.NewDict()
	rootDict.InsertName("Type", "Catalog")

	ir, err := xRefTable.IndRefForNewObject(rootDict)
	if err != nil {
		return nil, err
	}

	xRefTable.Root = ir

	return xRefTable, nil
}

// CreateDemoXRef creates a minimal single page PDF file for demo purposes.
func CreateDemoXRef() (*model.XRefTable, error) {
	xRefTable, err := CreateXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	return xRefTable, nil
}

func addPageTreeForResourceDictInheritanceDemo(xRefTable *model.XRefTable, rootDict types.Dict) error {

	// Create root page node.

	fIndRef, err := pdffont.EnsureFontDict(xRefTable, "Courier", "", "", false, false, nil)
	if err != nil {
		return err
	}

	rootPagesDict := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Pages"),
			"Count":    types.Integer(1),
			"MediaBox": types.RectForFormat("A4").Array(),
			"Resources": types.Dict(
				map[string]types.Object{
					"Font": types.Dict(
						map[string]types.Object{
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

	f100IndRef, err := pdffont.EnsureFontDict(xRefTable, "Courier-Bold", "", "", false, false, nil)
	if err != nil {
		return err
	}

	pagesDict := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Pages"),
			"Count":    types.Integer(1),
			"MediaBox": types.RectForFormat("A4").Array(),
			"Resources": types.Dict(
				map[string]types.Object{
					"Font": types.Dict(
						map[string]types.Object{
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

	p := model.Page{MediaBox: types.RectForFormat("A4"), Fm: model.FontMap{}, Buf: new(bytes.Buffer)}

	fontName := "Times-Roman"
	k := p.Fm.EnsureKey(fontName)
	td := model.TextDescriptor{
		Text:     "This font is Times-Roman and it is defined in the resource dict of this page dict.",
		FontName: fontName,
		FontKey:  k,
		FontSize: 12,
		Scale:    1.,
		ScaleAbs: true,
		X:        300,
		Y:        400,
	}

	model.WriteMultiLine(xRefTable, p.Buf, p.MediaBox, nil, td)

	fontName = "Courier"
	td = model.TextDescriptor{
		Text:     "This font is Courier and it is inherited from the page root.",
		FontName: fontName,
		FontKey:  "F99",
		FontSize: 12,
		Scale:    1.,
		ScaleAbs: true,
		X:        300,
		Y:        300,
	}

	model.WriteMultiLine(xRefTable, p.Buf, p.MediaBox, nil, td)

	fontName = "Courier-Bold"
	td = model.TextDescriptor{
		Text:     "This font is Courier-Bold and it is inherited from an intermediate page node.",
		FontName: fontName,
		FontKey:  "F100",
		FontSize: 12,
		Scale:    1.,
		ScaleAbs: true,
		X:        300,
		Y:        350,
	}

	model.WriteMultiLine(xRefTable, p.Buf, p.MediaBox, nil, td)

	pageIndRef, err := createDemoPage(xRefTable, *pagesIndRef, p)
	if err != nil {
		return err
	}

	pagesDict.Insert("Kids", types.Array{*pageIndRef})
	pagesDict.Insert("Parent", *rootPageIndRef)

	rootPagesDict.Insert("Kids", types.Array{*pagesIndRef})
	rootDict.Insert("Pages", *rootPageIndRef)

	return nil
}

// CreateResourceDictInheritanceDemoXRef creates a page tree for testing resource dict inheritance.
func CreateResourceDictInheritanceDemoXRef() (*model.XRefTable, error) {
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

func createFunctionalShadingDict(xRefTable *model.XRefTable) types.Dict {
	f := types.Dict(
		map[string]types.Object{
			"FunctionType": types.Integer(2),
			"Domain":       types.NewNumberArray(1.0, 1.2, 1.4, 1.6, 1.8, 2.0),
			"N":            types.Float(1),
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"ShadingType": types.Integer(1),
			"Function":    types.Array{f},
		},
	)

	return d
}

func createRadialShadingDict(xRefTable *model.XRefTable) types.Dict {
	f := types.Dict(
		map[string]types.Object{
			"FunctionType": types.Integer(2),
			"Domain":       types.NewNumberArray(1.0, 1.2, 1.4, 1.6, 1.8, 2.0),
			"N":            types.Float(1),
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"ShadingType": types.Integer(3),
			"Coords":      types.NewNumberArray(0, 0, 50, 10, 10, 100),
			"Function":    types.Array{f},
		},
	)

	return d
}

func createStreamObjForHalftoneDictType6(xRefTable *model.XRefTable) (*types.IndirectRef, error) {
	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":             types.Name("Halftone"),
				"HalftoneType":     types.Integer(6),
				"Width":            types.Integer(100),
				"Height":           types.Integer(100),
				"TransferFunction": types.Name("Identity"),
			},
		),
		Content: []byte{},
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createStreamObjForHalftoneDictType10(xRefTable *model.XRefTable) (*types.IndirectRef, error) {
	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":         types.Name("Halftone"),
				"HalftoneType": types.Integer(10),
				"Xsquare":      types.Integer(100),
				"Ysquare":      types.Integer(100),
			},
		),
		Content: []byte{},
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createStreamObjForHalftoneDictType16(xRefTable *model.XRefTable) (*types.IndirectRef, error) {
	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":         types.Name("Halftone"),
				"HalftoneType": types.Integer(16),
				"Width":        types.Integer(100),
				"Height":       types.Integer(100),
			},
		),
		Content: []byte{},
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createPostScriptCalculatorFunctionStreamDict(xRefTable *model.XRefTable) (*types.IndirectRef, error) {
	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"FunctionType": types.Integer(4),
				"Domain":       types.NewNumberArray(100.),
				"Range":        types.NewNumberArray(100.),
			},
		),
		Content: []byte{},
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func addResources(xRefTable *model.XRefTable, pageDict types.Dict, fontName string) error {
	fIndRef, err := pdffont.EnsureFontDict(xRefTable, fontName, "", "", true, false, nil)
	if err != nil {
		return err
	}

	functionalBasedShDict := createFunctionalShadingDict(xRefTable)

	radialShDict := createRadialShadingDict(xRefTable)

	f := types.Dict(
		map[string]types.Object{
			"FunctionType": types.Integer(2),
			"Domain":       types.NewNumberArray(0.0, 1.0),
			"C0":           types.NewNumberArray(0.0),
			"C1":           types.NewNumberArray(1.0),
			"N":            types.Float(1),
		},
	)

	fontResources := types.Dict(
		map[string]types.Object{
			"F1": *fIndRef,
		},
	)

	shadingResources := types.Dict(
		map[string]types.Object{
			"S1": functionalBasedShDict,
			"S3": radialShDict,
		},
	)

	colorSpaceResources := types.Dict(
		map[string]types.Object{
			"CSCalGray": types.Array{
				types.Name("CalGray"),
				types.Dict(
					map[string]types.Object{
						"WhitePoint": types.NewNumberArray(0.9505, 1.0000, 1.0890),
					},
				),
			},
			"CSCalRGB": types.Array{
				types.Name("CalRGB"),
				types.Dict(
					map[string]types.Object{
						"WhitePoint": types.NewNumberArray(0.9505, 1.0000, 1.0890),
					},
				),
			},
			"CSLab": types.Array{
				types.Name("Lab"),
				types.Dict(
					map[string]types.Object{
						"WhitePoint": types.NewNumberArray(0.9505, 1.0000, 1.0890),
					},
				),
			},
			"CS4DeviceN": types.Array{
				types.Name("DeviceN"),
				types.NewNameArray("Orange", "Green", "None"),
				types.Name("DeviceCMYK"),
				f,
				types.Dict(
					map[string]types.Object{
						"Subtype": types.Name("DeviceN"),
					},
				),
			},
			"CS6DeviceN": types.Array{
				types.Name("DeviceN"),
				types.NewNameArray("L", "a", "b", "Spot1"),
				types.Name("DeviceCMYK"),
				f,
				types.Dict(
					map[string]types.Object{
						"Subtype": types.Name("NChannel"),
						"Process": types.Dict(
							map[string]types.Object{
								"ColorSpace": types.Array{
									types.Name("Lab"),
									types.Dict(
										map[string]types.Object{
											"WhitePoint": types.NewNumberArray(0.9505, 1.0000, 1.0890),
										},
									),
								},
								"Components": types.NewNameArray("L", "a", "b"),
							},
						),
						"Colorants": types.Dict(
							map[string]types.Object{
								"Spot1": types.Array{
									types.Name("Separation"),
									types.Name("Spot1"),
									types.Name("DeviceCMYK"),
									f,
								},
							},
						),
						"MixingHints": types.Dict(
							map[string]types.Object{
								"Solidities": types.Dict(
									map[string]types.Object{
										"Spot1": types.Float(1.0),
									},
								),
								"DotGain": types.Dict(
									map[string]types.Object{
										"Spot1":   f,
										"Magenta": f,
										"Yellow":  f,
									},
								),
								"PrintingOrder": types.NewNameArray("Magenta", "Yellow", "Spot1"),
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

	graphicStateResources := types.Dict(
		map[string]types.Object{
			"GS1": types.Dict(
				map[string]types.Object{
					"Type": types.Name("ExtGState"),
					"HT": types.Dict(
						map[string]types.Object{
							"Type":             types.Name("Halftone"),
							"HalftoneType":     types.Integer(1),
							"Frequency":        types.Integer(120),
							"Angle":            types.Integer(30),
							"SpotFunction":     types.Name("CosineDot"),
							"TransferFunction": types.Name("Identity"),
						},
					),
					"BM": types.NewNameArray("Overlay", "Darken", "Normal"),
					"SMask": types.Dict(
						map[string]types.Object{
							"Type": types.Name("Mask"),
							"S":    types.Name("Alpha"),
							"G":    *anyXObject,
							"TR":   f,
						},
					),
					"TR":  f,
					"TR2": f,
				},
			),
			"GS2": types.Dict(
				map[string]types.Object{
					"Type": types.Name("ExtGState"),
					"HT": types.Dict(
						map[string]types.Object{
							"Type":         types.Name("Halftone"),
							"HalftoneType": types.Integer(5),
							"Default": types.Dict(
								map[string]types.Object{
									"Type":             types.Name("Halftone"),
									"HalftoneType":     types.Integer(1),
									"Frequency":        types.Integer(120),
									"Angle":            types.Integer(30),
									"SpotFunction":     types.Name("CosineDot"),
									"TransferFunction": types.Name("Identity"),
								},
							),
						},
					),
					"BM": types.NewNameArray("Overlay", "Darken", "Normal"),
					"SMask": types.Dict(
						map[string]types.Object{
							"Type": types.Name("Mask"),
							"S":    types.Name("Alpha"),
							"G":    *anyXObject,
							"TR":   types.Name("Identity"),
						},
					),
					"TR":   types.Array{f, f, f, f},
					"TR2":  types.Array{f, f, f, f},
					"BG2":  f,
					"UCR2": f,
					"D":    types.Array{types.Array{}, types.Integer(0)},
				},
			),
			"GS3": types.Dict(
				map[string]types.Object{
					"Type": types.Name("ExtGState"),
					"HT":   *indRefHalfToneType6,
					"SMask": types.Dict(
						map[string]types.Object{
							"Type": types.Name("Mask"),
							"S":    types.Name("Alpha"),
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
			"GS4": types.Dict(
				map[string]types.Object{
					"Type": types.Name("ExtGState"),
					"HT":   *indRefHalfToneType10,
				},
			),
			"GS5": types.Dict(
				map[string]types.Object{
					"Type": types.Name("ExtGState"),
					"HT":   *indRefHalfToneType16,
				},
			),
			"GS6": types.Dict(
				map[string]types.Object{
					"Type": types.Name("ExtGState"),
					"HT": types.Dict(
						map[string]types.Object{
							"Type":         types.Name("Halftone"),
							"HalftoneType": types.Integer(1),
							"Frequency":    types.Integer(120),
							"Angle":        types.Integer(30),
							"SpotFunction": *indRefFunctionStream,
						},
					),
				},
			),
			"GS7": types.Dict(
				map[string]types.Object{
					"Type": types.Name("ExtGState"),
					"HT": types.Dict(
						map[string]types.Object{
							"Type":         types.Name("Halftone"),
							"HalftoneType": types.Integer(1),
							"Frequency":    types.Integer(120),
							"Angle":        types.Integer(30),
							"SpotFunction": f,
						},
					),
				},
			),
		},
	)

	resourceDict := types.Dict(
		map[string]types.Object{
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
func CreateTestPageContent(p model.Page) {
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

func addContents(xRefTable *model.XRefTable, pageDict types.Dict, p model.Page) error {
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

func createBoxColorDict() types.Dict {
	cropBoxColorInfoDict := types.Dict(
		map[string]types.Object{
			"C": types.NewNumberArray(1.0, 1.0, 0.0),
			"W": types.Float(1.0),
			"S": types.Name("D"),
			"D": types.NewIntegerArray(3, 2),
		},
	)
	bleedBoxColorInfoDict := types.Dict(
		map[string]types.Object{
			"C": types.NewNumberArray(1.0, 0.0, 0.0),
			"W": types.Float(3.0),
			"S": types.Name("S"),
		},
	)
	trimBoxColorInfoDict := types.Dict(
		map[string]types.Object{
			"C": types.NewNumberArray(0.0, 1.0, 0.0),
			"W": types.Float(1.0),
			"S": types.Name("D"),
			"D": types.NewIntegerArray(3, 2),
		},
	)
	artBoxColorInfoDict := types.Dict(
		map[string]types.Object{
			"C": types.NewNumberArray(0.0, 0.0, 1.0),
			"W": types.Float(1.0),
			"S": types.Name("S"),
		},
	)
	d := types.Dict(
		map[string]types.Object{
			"CropBox":  cropBoxColorInfoDict,
			"BleedBox": bleedBoxColorInfoDict,
			"Trim":     trimBoxColorInfoDict,
			"ArtBox":   artBoxColorInfoDict,
		},
	)
	return d
}

func addViewportDict(pageDict types.Dict) {
	measureDict := types.Dict(
		map[string]types.Object{
			"Type":    types.Name("Measure"),
			"Subtype": types.Name("RL"),
			"R":       types.StringLiteral("1in = 0.1m"),
			"X": types.Array{
				types.Dict(
					map[string]types.Object{
						"Type": types.Name("NumberFormat"),
						"U":    types.StringLiteral("mi"),
						"C":    types.Float(0.00139),
						"D":    types.Integer(100000),
					},
				),
			},
			"D": types.Array{
				types.Dict(
					map[string]types.Object{
						"Type": types.Name("NumberFormat"),
						"U":    types.StringLiteral("mi"),
						"C":    types.Float(1),
					},
				),
				types.Dict(
					map[string]types.Object{
						"Type": types.Name("NumberFormat"),
						"U":    types.StringLiteral("feet"),
						"C":    types.Float(5280),
					},
				),
				types.Dict(
					map[string]types.Object{
						"Type": types.Name("NumberFormat"),
						"U":    types.StringLiteral("inch"),
						"C":    types.Float(12),
						"F":    types.Name("F"),
						"D":    types.Integer(8),
					},
				),
			},
			"A": types.Array{
				types.Dict(
					map[string]types.Object{
						"Type": types.Name("NumberFormat"),
						"U":    types.StringLiteral("acres"),
						"C":    types.Float(640),
					},
				),
			},
			"O": types.NewIntegerArray(0, 1),
		},
	)

	bbox := types.RectForDim(10, 60)

	vpDict := types.Dict(
		map[string]types.Object{
			"Type":    types.Name("Viewport"),
			"BBox":    bbox.Array(),
			"Name":    types.StringLiteral("viewPort"),
			"Measure": measureDict,
		},
	)

	pageDict.Insert("VP", types.Array{vpDict})
}

func annotRect(i int, w, h, d, l float64) *types.Rectangle {
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

// createAnnotsArray generates side by side lined up annotations starting in the lower left corner of the page.
func createAnnotsArray(xRefTable *model.XRefTable, pageIndRef types.IndirectRef, mediaBox types.Array) (types.Array, error) {
	pageWidth := mediaBox[2].(types.Float)
	pageHeight := mediaBox[3].(types.Float)

	a := types.Array{}

	for i, f := range []func(*model.XRefTable, types.IndirectRef, types.Array) (*types.IndirectRef, error){
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

func createPageWithAnnotations(xRefTable *model.XRefTable, parentPageIndRef types.IndirectRef, mediaBox *types.Rectangle, fontName string) (*types.IndirectRef, error) {
	mba := mediaBox.Array()

	pageDict := types.Dict(
		map[string]types.Object{
			"Type":         types.Name("Page"),
			"Parent":       parentPageIndRef,
			"BleedBox":     mba,
			"TrimBox":      mba,
			"ArtBox":       mba,
			"BoxColorInfo": createBoxColorDict(),
			"UserUnit":     types.Float(1.5)}, // Note: not honoured by Apple Preview
	)

	err := addResources(xRefTable, pageDict, fontName)
	if err != nil {
		return nil, err
	}

	p := model.Page{MediaBox: mediaBox, Buf: new(bytes.Buffer)}
	err = addContents(xRefTable, pageDict, p)
	if err != nil {
		return nil, err
	}

	pageIndRef, err := xRefTable.IndRefForNewObject(pageDict)
	if err != nil {
		return nil, err
	}

	// Fake SeparationInfo related to a single page only.
	separationInfoDict := types.Dict(
		map[string]types.Object{
			"Pages":          types.Array{*pageIndRef},
			"DeviceColorant": types.Name("Cyan"),
			"ColorSpace": types.Array{
				types.Name("Separation"),
				types.Name("Green"),
				types.Name("DeviceCMYK"),
				types.Dict(
					map[string]types.Object{
						"FunctionType": types.Integer(2),
						"Domain":       types.NewNumberArray(0.0, 1.0),
						"C0":           types.NewNumberArray(0.0),
						"C1":           types.NewNumberArray(1.0),
						"N":            types.Float(1),
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

func createPageWithForm(xRefTable *model.XRefTable, parentPageIndRef types.IndirectRef, annotsArray types.Array, mediaBox *types.Rectangle, fontName string) (*types.IndirectRef, error) {
	mba := mediaBox.Array()

	pageDict := types.Dict(
		map[string]types.Object{
			"Type":         types.Name("Page"),
			"Parent":       parentPageIndRef,
			"BleedBox":     mba,
			"TrimBox":      mba,
			"ArtBox":       mba,
			"BoxColorInfo": createBoxColorDict(),
			"UserUnit":     types.Float(1.0), // Note: not honoured by Apple Preview
		},
	)

	err := addResources(xRefTable, pageDict, fontName)
	if err != nil {
		return nil, err
	}

	p := model.Page{MediaBox: mediaBox, Buf: new(bytes.Buffer)}
	err = addContents(xRefTable, pageDict, p)
	if err != nil {
		return nil, err
	}

	pageDict.Insert("Annots", annotsArray)

	return xRefTable.IndRefForNewObject(pageDict)
}

func addPageTreeWithoutPage(xRefTable *model.XRefTable, rootDict types.Dict, d *types.Dim) error {
	// May be modified later on.
	mediaBox := types.RectForDim(d.Width, d.Height)

	pagesDict := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Pages"),
			"Count":    types.Integer(0),
			"MediaBox": mediaBox.Array(),
		},
	)

	pagesDict.Insert("Kids", types.Array{})

	pagesRootIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}

	rootDict.Insert("Pages", *pagesRootIndRef)

	return nil
}

func AddPageTreeWithSamplePage(xRefTable *model.XRefTable, rootDict types.Dict, p model.Page) error {

	// mediabox = physical page dimensions
	mba := p.MediaBox.Array()

	pagesDict := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Pages"),
			"Count":    types.Integer(1),
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

	pagesDict.Insert("Kids", types.Array{*pageIndRef})
	rootDict.Insert("Pages", *parentPageIndRef)

	return nil
}

func addPageTreeWithAnnotations(xRefTable *model.XRefTable, rootDict types.Dict, fontName string) (*types.IndirectRef, error) {
	// mediabox = physical page dimensions
	mediaBox := types.RectForFormat("A4")
	mba := mediaBox.Array()

	pagesDict := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Pages"),
			"Count":    types.Integer(1),
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

	pagesDict.Insert("Kids", types.Array{*pageIndRef})
	rootDict.Insert("Pages", *parentPageIndRef)

	return pageIndRef, nil
}

func addPageTreeWithFormFields(xRefTable *model.XRefTable, rootDict types.Dict, annotsArray types.Array, fontName string) (*types.IndirectRef, error) {
	// mediabox = physical page dimensions
	mediaBox := types.RectForFormat("A4")
	mba := mediaBox.Array()

	pagesDict := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Pages"),
			"Count":    types.Integer(1),
			"MediaBox": mba,
			"CropBox":  mba,
		},
	)

	parentPageIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return nil, err
	}

	pageIndRef, err := createPageWithForm(xRefTable, *parentPageIndRef, annotsArray, mediaBox, fontName)
	if err != nil {
		return nil, err
	}

	pagesDict.Insert("Kids", types.Array{*pageIndRef})

	rootDict.Insert("Pages", *parentPageIndRef)

	return pageIndRef, nil
}

// create a thread with 2 beads.
func createThreadDict(xRefTable *model.XRefTable, pageIndRef types.IndirectRef) (*types.IndirectRef, error) {
	infoDict := types.NewDict()
	infoDict.InsertString("Title", "DummyArticle")

	d := types.Dict(
		map[string]types.Object{
			"Type": types.Name("Thread"),
			"I":    infoDict,
		},
	)

	dIndRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	// create first bead
	d1 := types.Dict(
		map[string]types.Object{
			"Type": types.Name("Bead"),
			"T":    *dIndRef,
			"P":    pageIndRef,
			"R":    types.NewNumberArray(0, 0, 100, 100),
		},
	)

	d1IndRef, err := xRefTable.IndRefForNewObject(d1)
	if err != nil {
		return nil, err
	}

	d.Insert("F", *d1IndRef)

	// create last bead
	d2 := types.Dict(
		map[string]types.Object{
			"Type": types.Name("Bead"),
			"T":    *dIndRef,
			"N":    *d1IndRef,
			"V":    *d1IndRef,
			"P":    pageIndRef,
			"R":    types.NewNumberArray(0, 100, 200, 100),
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

func addThreads(xRefTable *model.XRefTable, rootDict types.Dict, pageIndRef types.IndirectRef) error {
	ir, err := createThreadDict(xRefTable, pageIndRef)
	if err != nil {
		return err
	}

	ir, err = xRefTable.IndRefForNewObject(types.Array{*ir})
	if err != nil {
		return err
	}

	rootDict.Insert("Threads", *ir)

	return nil
}

func addOpenAction(xRefTable *model.XRefTable, rootDict types.Dict) error {
	nextActionDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("Action"),
			"S":    types.Name("Movie"),
			"T":    types.StringLiteral("Sample Movie"),
		},
	)

	script := `app.alert('Hello Gopher!');`

	d := types.Dict(
		map[string]types.Object{
			"Type": types.Name("Action"),
			"S":    types.Name("JavaScript"),
			"JS":   types.StringLiteral(script),
			"Next": nextActionDict,
		},
	)

	rootDict.Insert("OpenAction", d)

	return nil
}

func addURI(xRefTable *model.XRefTable, rootDict types.Dict) {
	d := types.NewDict()
	d.InsertString("Base", "http://www.adobe.com")

	rootDict.Insert("URI", d)
}

func addSpiderInfo(xRefTable *model.XRefTable, rootDict types.Dict) error {
	// webCaptureInfoDict
	webCaptureInfoDict := types.NewDict()
	webCaptureInfoDict.InsertInt("V", 1.0)

	a := types.Array{}
	captureCmdDict := types.NewDict()
	captureCmdDict.InsertString("URL", (""))

	cmdSettingsDict := types.NewDict()
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

func addOCProperties(xRefTable *model.XRefTable, rootDict types.Dict) error {
	usageAppDict := types.Dict(
		map[string]types.Object{
			"Event":    types.Name("View"),
			"OCGs":     types.Array{}, // of indRefs
			"Category": types.NewNameArray("Language"),
		},
	)

	optionalContentConfigDict := types.Dict(
		map[string]types.Object{
			"Name":      types.StringLiteral("OCConf"),
			"Creator":   types.StringLiteral("Horst Rutter"),
			"BaseState": types.Name("ON"),
			"OFF":       types.Array{},
			"Intent":    types.Name("Design"),
			"AS":        types.Array{usageAppDict},
			"Order":     types.Array{},
			"ListMode":  types.Name("AllPages"),
			"RBGroups":  types.Array{},
			"Locked":    types.Array{},
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"OCGs":    types.Array{}, // of indRefs
			"D":       optionalContentConfigDict,
			"Configs": types.Array{optionalContentConfigDict},
		},
	)

	rootDict.Insert("OCProperties", d)

	return nil
}

func addRequirements(xRefTable *model.XRefTable, rootDict types.Dict) {
	d := types.NewDict()
	d.InsertName("Type", "Requirement")
	d.InsertName("S", "EnableJavaScripts")

	rootDict.Insert("Requirements", types.Array{d})
}

// CreateAnnotationDemoXRef creates a PDF file with examples of annotations and actions.
func CreateAnnotationDemoXRef() (*model.XRefTable, error) {
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

func createNormalAppearanceForFormField(xRefTable *model.XRefTable, w, h float64) (*types.IndirectRef, error) {
	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":     types.Name("XObject"),
				"Subtype":  types.Name("Form"),
				"FormType": types.Integer(1),
				"BBox":     types.NewNumberArray(0, 0, w, h),
				"Matrix":   types.NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		),
		Content: b.Bytes(),
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createRolloverAppearanceForFormField(xRefTable *model.XRefTable, w, h float64) (*types.IndirectRef, error) {
	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "1 0 0 RG 0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":     types.Name("XObject"),
				"Subtype":  types.Name("Form"),
				"FormType": types.Integer(1),
				"BBox":     types.NewNumberArray(0, 0, w, h),
				"Matrix":   types.NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		),
		Content: b.Bytes(),
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createDownAppearanceForFormField(xRefTable *model.XRefTable, w, h float64) (*types.IndirectRef, error) {
	// stroke outline path
	var b bytes.Buffer
	fmt.Fprintf(&b, "0 0 m 0 %f l %f %f l %f 0 l s", h, w, h, w)

	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":     types.Name("XObject"),
				"Subtype":  types.Name("Form"),
				"FormType": types.Integer(1),
				"BBox":     types.NewNumberArray(0, 0, w, h),
				"Matrix":   types.NewIntegerArray(1, 0, 0, 1, 0, 0),
			},
		),
		Content: b.Bytes(),
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createFormTextField(xRefTable *model.XRefTable, pageAnnots *types.Array, fontName string) (*types.IndirectRef, error) {
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

	fontDict, err := pdffont.EnsureFontDict(xRefTable, fontName, "", "", true, false, nil)
	if err != nil {
		return nil, err
	}

	resourceDict := types.Dict(
		map[string]types.Object{
			"Font": types.Dict(
				map[string]types.Object{
					fontName: *fontDict,
				},
			),
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"AP": types.Dict(
				map[string]types.Object{
					"N": *fN,
					"R": *fR,
					"D": *fD,
				},
			),
			"DA":      types.StringLiteral("/" + fontName + " 12 Tf 0 g"),
			"DR":      resourceDict,
			"FT":      types.Name("Tx"),
			"Rect":    types.NewNumberArray(x, y, x+w, y+h),
			"Border":  types.NewIntegerArray(0, 0, 1),
			"Type":    types.Name("Annot"),
			"Subtype": types.Name("Widget"),
			"T":       types.StringLiteral("inputField"),
			"TU":      types.StringLiteral("inputField"),
			"DV":      types.StringLiteral("Default value"),
			"V":       types.StringLiteral("Default value"),
		},
	)

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	*pageAnnots = append(*pageAnnots, *ir)

	return ir, nil
}

func createYesAppearance(xRefTable *model.XRefTable, resourceDict types.Dict, w, h float64) (*types.IndirectRef, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "q 0 0 1 rg BT /ZaDb 12 Tf 0 0 Td (8) Tj ET Q")

	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Resources": resourceDict,
				"Subtype":   types.Name("Form"),
				"BBox":      types.NewNumberArray(0, 0, w, h),
				"OPI": types.Dict(
					map[string]types.Object{
						"2.0": types.Dict(
							map[string]types.Object{
								"Type":    types.Name("OPI"),
								"Version": types.Float(2.0),
								"F":       types.StringLiteral("Proxy"),
								"Inks":    types.Name("full_color"),
							},
						),
					},
				),
				"Ref": types.Dict(
					map[string]types.Object{
						"F":    types.StringLiteral("Proxy"),
						"Page": types.Integer(1),
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

func createOffAppearance(xRefTable *model.XRefTable, resourceDict types.Dict, w, h float64) (*types.IndirectRef, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "q 0 0 1 rg BT /ZaDb 12 Tf 0 0 Td (4) Tj ET Q")

	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Resources": resourceDict,
				"Subtype":   types.Name("Form"),
				"BBox":      types.NewNumberArray(0, 0, w, h),
				"OPI": types.Dict(
					map[string]types.Object{
						"1.3": types.Dict(
							map[string]types.Object{
								"Type":     types.Name("OPI"),
								"Version":  types.Float(1.3),
								"F":        types.StringLiteral("Proxy"),
								"Size":     types.NewIntegerArray(400, 400),
								"CropRect": types.NewIntegerArray(0, 400, 400, 0),
								"Position": types.NewNumberArray(0, 0, 0, 400, 400, 400, 400, 0),
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

func createCheckBoxButtonField(xRefTable *model.XRefTable, pageAnnots *types.Array) (*types.IndirectRef, error) {
	fontDict, err := pdffont.EnsureFontDict(xRefTable, "ZapfDingbats", "", "", false, false, nil)
	if err != nil {
		return nil, err
	}

	resDict := types.Dict(
		map[string]types.Object{
			"Font": types.Dict(
				map[string]types.Object{
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

	apDict := types.Dict(
		map[string]types.Object{
			"N": types.Dict(
				map[string]types.Object{
					"Yes": *yesForm,
					"Off": *offForm,
				},
			),
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"FT":      types.Name("Btn"),
			"Rect":    types.NewNumberArray(250, 300, 270, 320),
			"Type":    types.Name("Annot"),
			"Subtype": types.Name("Widget"),
			"T":       types.StringLiteral("CheckBox"),
			"TU":      types.StringLiteral("CheckBox"),
			"V":       types.Name("Yes"),
			"AS":      types.Name("Yes"),
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

func createRadioButtonField(xRefTable *model.XRefTable, pageAnnots *types.Array) (*types.IndirectRef, error) {
	var flags uint32
	flags = setBit(flags, 16)

	d := types.Dict(
		map[string]types.Object{
			"FT":   types.Name("Btn"),
			"Ff":   types.Integer(flags),
			"Rect": types.NewNumberArray(250, 400, 280, 420),
			//"Type":    Name("Annot"),
			//"Subtype": Name("Widget"),
			"T": types.StringLiteral("Credit card"),
			"V": types.Name("card1"),
		},
	)

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	fontDict, err := pdffont.EnsureFontDict(xRefTable, "ZapfDingbats", "", "", false, false, nil)
	if err != nil {
		return nil, err
	}

	resDict := types.Dict(
		map[string]types.Object{
			"Font": types.Dict(
				map[string]types.Object{
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

	r1 := types.Dict(
		map[string]types.Object{
			"Rect":    types.NewNumberArray(250, 400, 280, 420),
			"Type":    types.Name("Annot"),
			"Subtype": types.Name("Widget"),
			"Parent":  *indRef,
			"T":       types.StringLiteral("Radio1"),
			"TU":      types.StringLiteral("Radio1"),
			"AS":      types.Name("card1"),
			"AP": types.Dict(
				map[string]types.Object{
					"N": types.Dict(
						map[string]types.Object{
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

	r2 := types.Dict(
		map[string]types.Object{
			"Rect":    types.NewNumberArray(300, 400, 330, 420),
			"Type":    types.Name("Annot"),
			"Subtype": types.Name("Widget"),
			"Parent":  *indRef,
			"T":       types.StringLiteral("Radio2"),
			"TU":      types.StringLiteral("Radio2"),
			"AS":      types.Name("Off"),
			"AP": types.Dict(
				map[string]types.Object{
					"N": types.Dict(
						map[string]types.Object{
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

	d.Insert("Kids", types.Array{*indRefR1, *indRefR2})

	*pageAnnots = append(*pageAnnots, *indRefR1)
	*pageAnnots = append(*pageAnnots, *indRefR2)

	return indRef, nil
}

func createResetButton(xRefTable *model.XRefTable, pageAnnots *types.Array) (*types.IndirectRef, error) {
	var flags uint32
	flags = setBit(flags, 17)

	fN, err := createNormalAppearanceForFormField(xRefTable, 20, 20)
	if err != nil {
		return nil, err
	}

	resetFormActionDict := types.Dict(
		map[string]types.Object{
			"Type":   types.Name("Action"),
			"S":      types.Name("ResetForm"),
			"Fields": types.NewStringLiteralArray("inputField"),
			"Flags":  types.Integer(0),
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"FT":      types.Name("Btn"),
			"Ff":      types.Integer(flags),
			"Rect":    types.NewNumberArray(100, 400, 120, 420),
			"Type":    types.Name("Annot"),
			"Subtype": types.Name("Widget"),
			"AP":      types.Dict(map[string]types.Object{"N": *fN}),
			"T":       types.StringLiteral("Reset"),
			"TU":      types.StringLiteral("Reset"),
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

func createSubmitButton(xRefTable *model.XRefTable, pageAnnots *types.Array) (*types.IndirectRef, error) {
	var flags uint32
	flags = setBit(flags, 17)

	fN, err := createNormalAppearanceForFormField(xRefTable, 20, 20)
	if err != nil {
		return nil, err
	}

	urlSpec := types.Dict(
		map[string]types.Object{
			"FS": types.Name("URL"),
			"F":  types.StringLiteral("http://www.me.com"),
		},
	)

	submitFormActionDict := types.Dict(
		map[string]types.Object{
			"Type":   types.Name("Action"),
			"S":      types.Name("SubmitForm"),
			"F":      urlSpec,
			"Fields": types.NewStringLiteralArray("inputField"),
			"Flags":  types.Integer(0),
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"FT":      types.Name("Btn"),
			"Ff":      types.Integer(flags),
			"Rect":    types.NewNumberArray(140, 400, 160, 420),
			"Type":    types.Name("Annot"),
			"Subtype": types.Name("Widget"),
			"AP":      types.Dict(map[string]types.Object{"N": *fN}),
			"T":       types.StringLiteral("Submit"),
			"TU":      types.StringLiteral("Submit"),
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

func streamObjForXFAElement(xRefTable *model.XRefTable, s string) (*types.IndirectRef, error) {
	sd := &types.StreamDict{
		Dict:    types.Dict(map[string]types.Object{}),
		Content: []byte(s),
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createXFAArray(xRefTable *model.XRefTable) (types.Array, error) {
	sd1, err := streamObjForXFAElement(xRefTable, "<xdp:xdp xmlns:xdp=\"http://ns.adobe.com/xdp/\">")
	if err != nil {
		return nil, err
	}

	sd3, err := streamObjForXFAElement(xRefTable, "</xdp:xdp>")
	if err != nil {
		return nil, err
	}

	return types.Array{
		types.StringLiteral("xdp:xdp"), *sd1,
		types.StringLiteral("/xdp:xdp"), *sd3,
	}, nil
}

func createFormDict(xRefTable *model.XRefTable, fontName string) (types.Dict, types.Array, error) {
	pageAnnots := types.Array{}

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

	d := types.Dict(
		map[string]types.Object{
			"Fields":          types.Array{*text, *checkBox, *radioButton, *resetButton, *submitButton}, // indRefs of fieldDicts
			"NeedAppearances": types.Boolean(true),
			"CO":              types.Array{*text},
			"XFA":             xfaArr,
		},
	)

	return d, pageAnnots, nil
}

// CreateFormDemoXRef creates an xRefTable with an AcroForm example.
func CreateFormDemoXRef() (*model.XRefTable, error) {
	fontName := "Helvetica"

	xRefTable, err := CreateXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	formDict, annotsArray, err := createFormDict(xRefTable, fontName)
	if err != nil {
		return nil, err
	}

	rootDict.Insert("AcroForm", formDict)

	_, err = addPageTreeWithFormFields(xRefTable, rootDict, annotsArray, fontName)
	if err != nil {
		return nil, err
	}

	rootDict.Insert("ViewerPreferences",
		types.Dict(
			map[string]types.Object{
				"FitWindow":    types.Boolean(true),
				"CenterWindow": types.Boolean(true),
			},
		),
	)

	return xRefTable, nil
}

// CreateContext creates a Context for given cross reference table and configuration.
func CreateContext(xRefTable *model.XRefTable, conf *model.Configuration) *model.Context {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	xRefTable.ValidationMode = conf.ValidationMode
	return &model.Context{
		Configuration: conf,
		XRefTable:     xRefTable,
		Write:         model.NewWriteContext(conf.Eol),
	}
}

// CreateContextWithXRefTable creates a Context with an xRefTable without pages for given configuration.
func CreateContextWithXRefTable(conf *model.Configuration, pageDim *types.Dim) (*model.Context, error) {
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

func createDemoContentStreamDict(xRefTable *model.XRefTable, pageDict types.Dict, b []byte) (*types.IndirectRef, error) {
	sd, _ := xRefTable.NewStreamDictForBuf(b)
	if err := sd.Encode(); err != nil {
		return nil, err
	}
	return xRefTable.IndRefForNewObject(*sd)
}

func createDemoPage(xRefTable *model.XRefTable, parentPageIndRef types.IndirectRef, p model.Page) (*types.IndirectRef, error) {

	pageDict := types.Dict(
		map[string]types.Object{
			"Type":   types.Name("Page"),
			"Parent": parentPageIndRef,
		},
	)

	fontRes, err := pdffont.FontResources(xRefTable, p.Fm)
	if err != nil {
		return nil, err
	}

	if len(fontRes) > 0 {
		resDict := types.Dict(
			map[string]types.Object{
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
