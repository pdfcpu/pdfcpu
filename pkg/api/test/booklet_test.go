package test

import (
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

type testBookletCfg struct {
	msg           string
	inFiles       []string
	outFile       string
	selectedPages []string
	desc          string
}

func testBooklet(t *testing.T, cfg *testBookletCfg) {
	t.Helper()

	booklet, err := pdfcpu.BookletConfig(cfg.desc)
	if err != nil {
		t.Fatalf("%s %s: %v\n", cfg.msg, cfg.outFile, err)
	}
	if err := api.PDFBooklet(cfg.inFiles, cfg.outFile, cfg.selectedPages, booklet, nil); err != nil {
		t.Fatalf("%s %s: %v\n", cfg.msg, cfg.outFile, err)
	}
	if err := api.ValidateFile(cfg.outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", cfg.msg, err)
	}
}

func TestBooklet(t *testing.T) {
	outDir := "../../samples/booklet"
	for _, tt := range []*testBookletCfg{
		// 4-up booklet
		{"TestBookletLedger",
			[]string{filepath.Join(inDir, "demo-booklet-input-statement.pdf")},
			filepath.Join(outDir, "booklet-ledger.pdf"),
			[]string{"1-24"},
			"pagesize:Statement, sheetsize:LedgerP",
		},

		// 2-up booklet
		{"TestBookletLetter",
			[]string{filepath.Join(inDir, "demo-booklet-input-statement.pdf")},
			filepath.Join(outDir, "booklet-letter.pdf"),
			[]string{"1-16"},
			"pagesize:Statement, sheetsize:LetterP",
		},

		// 2-up booklet where the number of pages don't fill the whole sheet
		{"TestBookletBlankPages",
			[]string{filepath.Join(inDir, "demo-booklet-input-statement.pdf")},
			filepath.Join(outDir, "booklet-letter-with-blank-pages.pdf"),
			[]string{"1-14"},
			"pagesize:Statement, sheetsize:LetterP",
		},

		// 4-up booklet where the number of pages don't fill the whole sheet
		{"TestBookletBlankPagesLedger",
			[]string{filepath.Join(inDir, "demo-booklet-input-statement.pdf")},
			filepath.Join(outDir, "booklet-ledger-with-blank-pages.pdf"),
			[]string{"1-21"},
			"pagesize:Statement, sheetsize:LedgerP",
		},
	} {
		testBooklet(t, tt)
	}
}
