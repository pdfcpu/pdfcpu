/*
Copyright 2021 The pdf Authors.

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
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func testBooklet(t *testing.T, msg string, inFiles []string, outFile string, selectedPages []string, desc string, n int, isImg bool, conf *model.Configuration) {
	t.Helper()

	var (
		booklet *model.NUp
		err     error
	)

	if isImg {
		if booklet, err = api.ImageBookletConfig(n, desc, conf); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	} else {
		if booklet, err = api.PDFBookletConfig(n, desc, conf); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	}

	if err := api.BookletFile(inFiles, outFile, selectedPages, booklet, conf); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestBooklet(t *testing.T) {
	outDir := filepath.Join("..", "..", "samples", "booklet")

	for _, tt := range []struct {
		msg           string
		inFiles       []string
		outFile       string
		selectedPages []string
		desc          string
		unit          string
		n             int
		isImg         bool
	}{
		// 2-up booklet from images on A4
		{"TestBookletFromImagesA42Up",
			imageFileNames(t, resDir),
			filepath.Join(outDir, "BookletFromImagesA4_2Up.pdf"),
			nil,
			"p:A4, border:false, g:on, ma:25, bgcol:#beded9",
			"points",
			2,
			true,
		},

		// 4-up booklet from images on A4
		{"TestBookletFromImagesA44Up",
			imageFileNames(t, resDir),
			filepath.Join(outDir, "BookletFromImagesA4_4Up.pdf"),
			nil,
			"p:A4, border:false, g:on, ma:25, bgcol:#beded9",
			"points",
			4,
			true,
		},

		// 2-up booklet from PDF on A4
		{"TestBookletFromPDF2UpA4",
			[]string{filepath.Join(inDir, "zineTest.pdf")},
			filepath.Join(outDir, "BookletFromPDFA4_2Up.pdf"),
			nil, // all pages
			"p:A4, border:false, g:on, ma:10, bgcol:#beded9",
			"points",
			2,
			false,
		},

		// 4-up booklet from PDF on A4
		{"TestBookletFromPDF4UpA4",
			[]string{filepath.Join(inDir, "zineTest.pdf")},
			filepath.Join(outDir, "BookletFromPDFA4_4Up.pdf"),
			[]string{"1-"}, // all pages
			"p:A4, border:off, guides:on, ma:10, bgcol:#beded9",
			"points",
			4,
			false,
		},

		// 4-up booklet from PDF on Ledger
		{"TestBookletFromPDF4UpLedger",
			[]string{filepath.Join(inDir, "bookletTest.pdf")},
			filepath.Join(outDir, "BookletFromPDFLedger_4Up.pdf"),
			[]string{"1-24"},
			"p:LedgerP, g:on, ma:10, bgcol:#f7e6c7",
			"points",
			4,
			false,
		},

		// 4-up booklet from PDF on Ledger where the number of pages don't fill the whole sheet
		{"TestBookletFromPDF4UpLedgerWithTrailingBlankPages",
			[]string{filepath.Join(inDir, "bookletTest.pdf")},
			filepath.Join(outDir, "BookletFromPDFLedger_4UpWithTrailingBlankPages.pdf"),
			[]string{"1-21"},
			"p:LedgerP, g:on, ma:10, bgcol:#f7e6c7",
			"points",
			4,
			false,
		},

		// 2-up booklet from PDF on Letter
		{"TestBookletFromPDF2UpLetter",
			[]string{filepath.Join(inDir, "bookletTest.pdf")},
			filepath.Join(outDir, "BookletFromPDFLetter_2Up.pdf"),
			[]string{"1-16"},
			"p:LetterP, g:on, ma:10, bgcol:#f7e6c7",
			"points",
			2,
			false,
		},

		// 2-up booklet from PDF on Letter where the number of pages don't fill the whole sheet
		{"TestBookletFromPDF2UpLetterWithTrailingBlankPages",
			[]string{filepath.Join(inDir, "bookletTest.pdf")},
			filepath.Join(outDir, "BookletFromPDFLetter_2UpWithTrailingBlankPages.pdf"),
			[]string{"1-14"},
			"p:LetterP, g:on, ma:10, bgcol:#f7e6c7",
			"points",
			2,
			false,
		},

		// more nup
		{"TestBookletFromPDF_2up_perfectbound",
			[]string{filepath.Join(inDir, "bookletTest.pdf")},
			filepath.Join(outDir, "BookletFromPDFLetter_2Up_perfectbound.pdf"),
			[]string{"1-24"},
			"p:LetterP, g:on, btype:perfectbound",
			"points",
			2,
			false,
		},
		{"TestBookletFromPDF_6up",
			[]string{filepath.Join(inDir, "bookletTest.pdf")},
			filepath.Join(outDir, "BookletFromPDFLedger_6Up.pdf"),
			[]string{"1-24"},
			"p:LedgerP, g:on",
			"points",
			6,
			false,
		},
		{"TestBookletFromPDF_8up",
			[]string{filepath.Join(inDir, "bookletTest.pdf")},
			filepath.Join(outDir, "BookletFromPDFLedger_8Up.pdf"),
			[]string{"1-32"},
			"p:LedgerP, g:on",
			"points",
			8,
			false,
		},

		// misc orientations and booklet types on 4-up
		{"TestBookletFromPDF_4up_portrait_short",
			[]string{filepath.Join(inDir, "bookletTest.pdf")},
			filepath.Join(outDir, "BookletFromPDFLedger_4Up_portrait_short.pdf"),
			[]string{"1-24"},
			"p:LedgerP, g:on, binding:short",
			"points",
			4,
			false,
		},
		{"TestBookletFromPDF_4up_landscape_long",
			[]string{filepath.Join(inDir, "bookletTestLandscape.pdf")},
			filepath.Join(outDir, "BookletFromPDFLedger_4Up_landscape_long.pdf"),
			[]string{"1-24"},
			"p:LedgerL, g:on",
			"points",
			4,
			false,
		},
		{"TestBookletFromPDF_4up_landscape_short",
			[]string{filepath.Join(inDir, "bookletTestLandscape.pdf")},
			filepath.Join(outDir, "BookletFromPDFLedger_4Up_landscape_short.pdf"),
			[]string{"1-24"},
			"p:LedgerL, g:on, binding:short",
			"points",
			4,
			false,
		},
		{"TestBookletFromPDF_4up-portrait_long_advanced",
			[]string{filepath.Join(inDir, "bookletTest.pdf")},
			filepath.Join(outDir, "BookletFromPDFLedger_4Up_portrait_long_advanced.pdf"),
			[]string{"1-24"},
			"p:LedgerP, g:on, btype:bookletadvanced",
			"points",
			4,
			false,
		},
		{"TestBookletFromPDF_4up_landscape_short_advanced",
			[]string{filepath.Join(inDir, "bookletTestLandscape.pdf")},
			filepath.Join(outDir, "BookletFromPDFLedger_4Up_landscape_short_advanced.pdf"),
			[]string{"1-24"},
			"p:LedgerL, g:on, binding:short, btype:bookletadvanced",
			"points",
			4,
			false,
		},
		{"TestBookletFromPDF_4up_perfectbound",
			[]string{filepath.Join(inDir, "bookletTest.pdf")},
			filepath.Join(outDir, "BookletFromPDFLedger_4Up_perfectbound.pdf"),
			[]string{"1-24"},
			"p:LedgerP, g:on, btype:perfectbound",
			"points",
			4,
			false,
		},

		// 2-up multi folio booklet from PDF on A4 using 8 sheets per folio
		// using the default foliosize:8
		// Here we print 2 complete folios (2 x 8 sheets) + 1 partial folio
		// multi folio only makes sense for n = 2
		// See also  https://www.instructables.com/How-to-bind-your-own-Hardback-Book/
		{"TestHardbackBookFromPDF",
			[]string{filepath.Join(inDir, "WaldenFull.pdf")},
			filepath.Join(outDir, "HardbackBookFromPDF.pdf"),
			[]string{"1-70"},
			"p:A4, multifolio:on, border:off, g:on, ma:10, bgcol:#beded9",
			"points",
			2,
			false,
		},
	} {
		conf := model.NewDefaultConfiguration()
		conf.SetUnit(tt.unit)
		testBooklet(t, tt.msg, tt.inFiles, tt.outFile, tt.selectedPages, tt.desc, tt.n, tt.isImg, conf)
	}
}
