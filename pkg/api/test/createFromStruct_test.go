/*
Copyright 2023 The pdf Authors.

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

package test

import (
	"io"
	"os"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/primitives"
)

func TestCreatePDF(t *testing.T) {
	t.Helper()

	// if err := api.CreateFile("", "../../testdata/json/create/textAndAlignment.json", "textAndAlignmentFromJson.pdf", conf); err != nil {
	// 	t.Fatalf("TestTextAlignment CreateFile: %v\n", err)
	// }

	// if err := api.ValidateFile("textAndAlignmentFromJson.pdf", nil); err != nil {
	// 	t.Fatalf("TestTextAlignment ValidateFile: %v\n", err)
	// }

	pdf := &primitives.PDF{
		Paper:      "A4P",
		Crop:       "10",
		Origin:     "LowerLeft",
		Debug:      false,
		ContentBox: true,
		Guides:     true,
		Colors: map[string]string{
			"DarkOrange":   "#FF8C00",
			"DarkSeaGreen": "#8FBC8F",
			"LimeGreen":    "#BEDED9",
		},
		DirNames: map[string]string{
			"images": "../../testdata/resources",
		},
		FileNames: map[string]string{
			"logo1": "$images/logoVerySmall.png",
			"logo2": "$images/github.png",
		},
		Fonts: map[string]*primitives.FormFont{
			"myCourier": {
				Name: "Courier",
				Size: 12,
			},
			"myCourierBold": {
				Name: "Courier-Bold",
				Size: 12,
			},
			"input": {
				Name:  "Helvetica",
				Size:  12,
				Color: "#222222",
			},
			"label": {
				Name: "Helvetica",
				Size: 12,
			},
		},
		Margin: &primitives.Margin{
			Width: 10,
		},
		Header: &primitives.HorizontalBand{
			Font: &primitives.FormFont{
				Name:  "$myCourierBold",
				Size:  24,
				Color: "#C00000",
			},
			Left:   "$logo1",
			Center: "Textboxes and Alignment",
			Right:  "$logo2",
			Height: 40,
			Dx:     5,
			Dy:     5,
			Border: false,
		},
		Footer: &primitives.HorizontalBand{
			Font: &primitives.FormFont{
				Name: "Courier",
				Size: 9,
			},
			Left:   "pdfcpu: %v\nCreated: %t",
			Center: "Page %p of %P",
			Right:  "create from structure",
			Height: 30,
			Dx:     5,
			Dy:     5,
			Border: false,
		},
		ImageBoxPool: map[string]*primitives.ImageBox{
			"logo1": {
				Src: "$logo1",
				Url: "https://pdfcpu.io",
				Margin: &primitives.Margin{
					Width: 5,
				},
			},
			"logo2": {
				Src: "$logo2",
				Url: "https://github.com/pdfcpu/pdfcpu",
				Margin: &primitives.Margin{
					Width: 5,
				},
			},
		},
		Pages: map[string]*primitives.PDFPage{
			"1": {
				BackgroundColor: "LightGray",
				Content: &primitives.Content{
					Guides: []*primitives.Guide{
						{
							Position: [2]float64{
								-1,
								650,
							},
						},
						{
							Position: [2]float64{
								-1,
								600,
							},
						},
						{
							Position: [2]float64{
								-1,
								550,
							},
						},
						{
							Position: [2]float64{
								-1,
								485,
							},
						},

						{
							Position: [2]float64{
								-1,
								420,
							},
						},
						{
							Position: [2]float64{
								-1,
								370,
							},
						},
						{
							Position: [2]float64{
								-1,
								320,
							},
						},
						{
							Position: [2]float64{
								-1,
								260,
							},
						},

						{
							Position: [2]float64{
								-1,
								190,
							},
						},
						{
							Position: [2]float64{
								-1,
								140,
							},
						},
						{
							Position: [2]float64{
								-1,
								90,
							},
						},
						{
							Position: [2]float64{
								-1,
								3,
							},
						},
					},
					TextBoxes: []*primitives.TextBox{
						{
							Hide:  false,
							Value: "Textboxes without width:",
							Position: [2]float64{
								1,
								680,
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
						},
						{
							Hide:  false,
							Value: "A left aligned text\nwith border and padding.",
							Position: [2]float64{
								-1,
								650,
							},
							Alignment:       "left",
							BackgroundColor: "$LimeGreen",
							Border: &primitives.Border{
								Width: 2,
								Color: "Black",
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
							Padding: &primitives.Padding{
								Width: 5,
							},
						},
						{
							Hide:  false,
							Value: "A center aligned text\nwith border and padding.",
							Position: [2]float64{
								-1,
								600,
							},
							Alignment:       "center",
							BackgroundColor: "$LimeGreen",
							Border: &primitives.Border{
								Width: 2,
								Color: "Black",
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
							Padding: &primitives.Padding{
								Width: 5,
							},
						},
						{
							Hide:  false,
							Value: "A right aligned text\nwith border and padding.",
							Position: [2]float64{
								-1,
								550,
							},
							Alignment:       "right",
							BackgroundColor: "$LimeGreen",
							Border: &primitives.Border{
								Width: 2,
								Color: "Black",
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
							Padding: &primitives.Padding{
								Width: 5,
							},
						},
						{
							Hide:  false,
							Value: "A justified aligned text\nwith border and padding.",
							Position: [2]float64{
								-1,
								485,
							},
							Alignment:       "justify",
							BackgroundColor: "$LimeGreen",
							Border: &primitives.Border{
								Width: 2,
								Color: "Black",
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
							Padding: &primitives.Padding{
								Width: 5,
							},
						},

						{
							Hide:  false,
							Value: "Textboxes using width: 200",
							Position: [2]float64{
								1,
								450,
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
						},
						{
							Hide:  false,
							Value: "A left aligned text\nwith border and padding.",
							Position: [2]float64{
								-1,
								420,
							},
							Width:           200,
							Alignment:       "left",
							BackgroundColor: "$DarkSeaGreen",
							Border: &primitives.Border{
								Width: 2,
								Color: "Black",
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
							Padding: &primitives.Padding{
								Width: 5,
							},
						},
						{
							Hide:  false,
							Value: "A center aligned text\nwith border and padding.",
							Position: [2]float64{
								-1,
								370,
							},
							Width:           200,
							Alignment:       "center",
							BackgroundColor: "$DarkSeaGreen",
							Border: &primitives.Border{
								Width: 2,
								Color: "Black",
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
							Padding: &primitives.Padding{
								Width: 5,
							},
						},
						{
							Hide:  false,
							Value: "A right aligned text\nwith border and padding.",
							Position: [2]float64{
								-1,
								320,
							},
							Width:           200,
							Alignment:       "right",
							BackgroundColor: "$DarkSeaGreen",
							Border: &primitives.Border{
								Width: 2,
								Color: "Black",
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
							Padding: &primitives.Padding{
								Width: 5,
							},
						},
						{
							Hide:  false,
							Value: "A justified aligned text with border and padding.",
							Position: [2]float64{
								-1,
								260,
							},
							Width:           200,
							Alignment:       "justify",
							BackgroundColor: "$DarkSeaGreen",
							Border: &primitives.Border{
								Width: 2,
								Color: "Black",
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
							Padding: &primitives.Padding{
								Width: 5,
							},
						},

						{
							Hide:  false,
							Value: "Textboxes using width: 100",
							Position: [2]float64{
								1,
								220,
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
						},
						{
							Hide:  false,
							Value: "A left aligned text\nwith border and padding.",
							Position: [2]float64{
								-1,
								190,
							},
							Width:           100,
							Alignment:       "left",
							BackgroundColor: "$DarkOrange",
							Border: &primitives.Border{
								Width: 2,
								Color: "Black",
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
							Padding: &primitives.Padding{
								Width: 5,
							},
						},
						{
							Hide:  false,
							Value: "A center aligned text\nwith border and padding.",
							Position: [2]float64{
								-1,
								140,
							},
							Width:           100,
							Alignment:       "center",
							BackgroundColor: "$DarkOrange",
							Border: &primitives.Border{
								Width: 2,
								Color: "Black",
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
							Padding: &primitives.Padding{
								Width: 5,
							},
						},
						{
							Hide:  false,
							Value: "A right aligned text\nwith border and padding.",
							Position: [2]float64{
								-1,
								90,
							},
							Width:           100,
							Alignment:       "right",
							BackgroundColor: "$DarkOrange",
							Border: &primitives.Border{
								Width: 2,
								Color: "Black",
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
							Padding: &primitives.Padding{
								Width: 5,
							},
						},
						{
							Hide:  false,
							Value: "A justified aligned text with border and padding.",
							Position: [2]float64{
								-1,
								3,
							},
							Width:           100,
							Alignment:       "justify",
							BackgroundColor: "$DarkOrange",
							Border: &primitives.Border{
								Width: 2,
								Color: "Black",
							},
							Font: &primitives.FormFont{
								Name: "$myCourier",
							},
							Padding: &primitives.Padding{
								Width: 5,
							},
						},
					},
				},
			},
		},
	}

	outFile, _ := os.Create("textAndAlignment.pdf")
	conf := api.LoadConfiguration()
	err := api.CreatePDF(io.ReadSeeker(nil), pdf, outFile, conf)
	if err != nil {
		t.Fatalf("TestTextAlignmentFromStruct CreatePDF: %v\n", err)
	}

	if err := api.ValidateFile("textAndAlignment.pdf", nil); err != nil {
		t.Fatalf("TestTextAlignmentFromStruct ValidateFile: %v\n", err)
	}
}
