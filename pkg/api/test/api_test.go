/*
Copyright 2018 The pdf Authors.

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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

var inDir, outDir, resDir string

func TestMain(m *testing.M) {
	inDir = "../../testdata"
	resDir = filepath.Join(inDir, "resources")
	var err error

	if outDir, err = ioutil.TempDir("", "pdfcpu_api_tests"); err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
	// fmt.Printf("outDir = %s\n", outDir)

	exitCode := m.Run()

	if err = os.RemoveAll(outDir); err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

func copyFile(t *testing.T, srcFileName, destFileName string) error {
	t.Helper()
	from, err := os.Open(srcFileName)
	if err != nil {
		return err
	}
	defer from.Close()
	to, err := os.Create(destFileName)
	if err != nil {
		return err
	}
	defer to.Close()
	_, err = io.Copy(to, from)
	return err
}
func BenchmarkValidateCommand(b *testing.B) {
	msg := "BenchmarkValidateCommand"
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		f, err := os.Open(filepath.Join(inDir, "gobook.0.pdf"))
		if err != nil {
			b.Fatalf("%s: %v\n", msg, err)
		}
		if err = api.Validate(f, nil); err != nil {
			b.Fatalf("%s: %v\n", msg, err)
		}
		if err = f.Close(); err != nil {
			b.Fatalf("%s: %v\n", msg, err)
		}
	}
}

func isPDF(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".pdf")
}

func AllPDFs(t *testing.T, dir string) []string {
	t.Helper()
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("pdfFiles from %s: %v\n", dir, err)
	}
	ff := []string(nil)
	for _, f := range files {
		if isPDF(f.Name()) {
			ff = append(ff, f.Name())
		}
	}
	return ff
}

func TestPageCount(t *testing.T) {
	msg := "TestPageCount"

	fn := "5116.DCT_Filter.pdf"
	wantPageCount := 52
	inFile := filepath.Join(inDir, fn)

	// Retrieve page count for inFile.
	gotPageCount, err := api.PageCountFile(inFile)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if wantPageCount != gotPageCount {
		t.Fatalf("%s %s: pageCount want:%d got:%d\n", msg, inFile, wantPageCount, gotPageCount)
	}
}

func TestPageDimensions(t *testing.T) {
	msg := "TestPageDimensions"
	for _, fn := range AllPDFs(t, inDir) {
		inFile := filepath.Join(inDir, fn)

		// Retrieve page dimensions for inFile.
		if _, err := api.PageDimsFile(inFile); err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
	}
}

func TestValidate(t *testing.T) {
	msg := "TestValidate"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")

	// Validate inFile.
	if err := api.ValidateFile(inFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestManipulateContext(t *testing.T) {
	msg := "TestManipulateContext"
	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")
	outFile := filepath.Join(outDir, "abc.pdf")

	// Read a PDF Context from inFile.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s: ReadContextFile %s: %v\n", msg, inFile, err)
	}

	// Manipulate the PDF Context.
	// Eg. Let's stamp all pages with pageCount and current timestamp.
	text := fmt.Sprintf("Pages: %d \n Current time: %v", ctx.PageCount, time.Now())
	wm, err := pdf.ParseTextWatermarkDetails(text, "font:Times-Italic, scale:.9", true)
	if err != nil {
		t.Fatalf("%s: ParseTextWatermarkDetails: %v\n", msg, err)
	}
	if err := api.WatermarkContext(ctx, nil, wm); err != nil {
		t.Fatalf("%s: WatermarkContext: %v\n", msg, err)
	}

	// Write the manipulated PDF context to outFile.
	if err := api.WriteContextFile(ctx, outFile); err != nil {
		t.Fatalf("%s: WriteContextFile %s: %v\n", msg, outFile, err)
	}
}

func TestInfo(t *testing.T) {
	msg := "TestInfo"
	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")

	if _, err := api.InfoFile(inFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}
