/*
Copyright 2021 The pdfcpu Authors.

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

	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

type testBookletCfg struct {
	msg           string
	inFiles       []string
	outFile       string
	selectedPages []string
	desc          string
}

func testBooklet(t *testing.T, cfg testBookletCfg) {
	t.Helper()

	booklet, err := pdfcpu.BookletConfig(cfg.desc)
	if err != nil {
		t.Fatalf("%s %s: %v\n", cfg.msg, cfg.outFile, err)
	}
	cmd := cli.BookletCommand(cfg.inFiles, cfg.outFile, cfg.selectedPages, booklet, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", cfg.msg, cfg.outFile, err)
	}

	if err := validateFile(t, cfg.outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", cfg.msg, err)
	}
}

func TestBookletCommand(t *testing.T) {
	for _, tt := range []testBookletCfg{
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
