/*
Copyright 2020 The pdfcpu Authors.

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

	"github.com/mechiko/pdfcpu/pkg/api"
	"github.com/mechiko/pdfcpu/pkg/cli"
	"github.com/mechiko/pdfcpu/pkg/pdfcpu/model"
)

func testGrid(t *testing.T, msg string, inFiles []string, outFile string, selectedPages []string, desc string, rows, cols int, isImg bool, conf *model.Configuration) {
	t.Helper()

	var (
		nup *model.NUp
		err error
	)

	if isImg {
		if nup, err = api.ImageGridConfig(rows, cols, desc, conf); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	} else {
		if nup, err = api.PDFGridConfig(rows, cols, desc, conf); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	}

	cmd := cli.NUpCommand(inFiles, outFile, selectedPages, nup, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestGridCommand(t *testing.T) {
	for _, tt := range []struct {
		msg           string
		inFiles       []string
		outFile       string
		selectedPages []string
		desc          string
		unit          string
		rows, cols    int
		isImg         bool
	}{
		{"TestGridFromPDF",
			[]string{filepath.Join(inDir, "Acroforms2.pdf")},
			filepath.Join(outDir, "testGridFromPDF.pdf"),
			nil, "form:LegalL", "points", 1, 3, false},

		{"TestGridFromImages",
			[]string{
				filepath.Join(resDir, "pdfchip3.png"),
				filepath.Join(resDir, "demo.png"),
				filepath.Join(resDir, "snow.jpg"),
			},
			filepath.Join(outDir, "testGridFromImages.pdf"),
			nil, "d:500 500, margin:20, border:off", "points", 1, 3, true},
	} {
		conf := model.NewDefaultConfiguration()
		conf.SetUnit(tt.unit)
		testGrid(t, tt.msg, tt.inFiles, tt.outFile, tt.selectedPages, tt.desc, tt.rows, tt.cols, tt.isImg, conf)
	}
}
