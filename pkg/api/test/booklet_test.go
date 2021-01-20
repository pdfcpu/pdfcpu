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
		// Booklet (4up on ledger) on PDF
		{"TestBookletLedger",
			[]string{filepath.Join(inDir, "demo-booklet-input-statement.pdf")},
			filepath.Join(outDir, "booklet-ledger.pdf"),
			[]string{"1-24"},
			"pagesize:Statement, sheetsize:LedgerP",
		},

		// Booklet (2up with rotation) on PDF
		{"TestBookletLetter",
			[]string{filepath.Join(inDir, "demo-booklet-input-statement.pdf")},
			filepath.Join(outDir, "booklet-letter.pdf"),
			[]string{"1-16"},
			"pagesize:Statement, sheetsize:LetterP",
		},
	} {
		testBooklet(t, tt)
	}
}
