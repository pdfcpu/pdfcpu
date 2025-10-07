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

	"github.com/angel-one/pdfcpu/pkg/api"
	"github.com/angel-one/pdfcpu/pkg/cli"
	"github.com/angel-one/pdfcpu/pkg/pdfcpu/model"
)

func testNUp(t *testing.T, msg string, inFiles []string, outFile string, selectedPages []string, desc string, n int, isImg bool, conf *model.Configuration) {
	t.Helper()

	var (
		nup *model.NUp
		err error
	)

	if isImg {
		if nup, err = api.ImageNUpConfig(n, desc, conf); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	} else {
		if nup, err = api.PDFNUpConfig(n, desc, conf); err != nil {
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

func TestNUpCommand(t *testing.T) {
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
		{"TestNUpFromPDF",
			[]string{filepath.Join(inDir, "Acroforms2.pdf")},
			filepath.Join(outDir, "Acroforms2.pdf"),
			nil,
			"",
			"points",
			4,
			false},

		{"TestNUpFromSingleImage",
			[]string{filepath.Join(resDir, "pdfchip3.png")},
			filepath.Join(outDir, "out.pdf"),
			nil,
			"form:A3L",
			"points",
			9,
			true},

		{"TestNUpFromImages",
			[]string{
				filepath.Join(resDir, "pdfchip3.png"),
				filepath.Join(resDir, "demo.png"),
				filepath.Join(resDir, "snow.jpg"),
			},
			filepath.Join(outDir, "out1.pdf"),
			nil,
			"form:Tabloid, bo:off, ma:0, enforce:off",
			"points",
			6,
			true},
	} {
		conf := model.NewDefaultConfiguration()
		conf.SetUnit(tt.unit)
		testNUp(t, tt.msg, tt.inFiles, tt.outFile, tt.selectedPages, tt.desc, tt.n, tt.isImg, conf)
	}
}
