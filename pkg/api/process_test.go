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

package api

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hhrutter/pdfcpu/pkg/pdfcpu"
	"github.com/hhrutter/pdfcpu/pkg/pdfcpu/validate"
)

var inDir, outDir string

func TestMain(m *testing.M) {

	inDir = "testdata"

	var err error

	outDir, err = ioutil.TempDir("", "pdfcpu_apiTests")
	//fmt.Printf("outDir = %s\n", outDir)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	exitCode := m.Run()

	err = os.RemoveAll(outDir)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

func TestGetPageCount(t *testing.T) {

	config := pdfcpu.NewDefaultConfiguration()
	inFile := filepath.Join(inDir, "CenterOfWhy.pdf")

	ctx, err := Read(inFile, config)
	if err != nil {
		t.Fatalf("TestGetPageCount:  %v\n", err)
	}

	err = validate.XRefTable(ctx.XRefTable)
	if err != nil {
		t.Fatalf("TestGetPageCount: %v\n", err)
	}

	if ctx.PageCount != 25 {
		t.Fatalf("TestGetPageCount: pageCount should be %d but is %d\n", 25, ctx.PageCount)
	}

}

// Validate all PDFs in testdata.
func TestValidateCommand(t *testing.T) {

	files, err := ioutil.ReadDir(inDir)
	if err != nil {
		t.Fatalf("TestValidateCommand: %v\n", err)
	}

	config := pdfcpu.NewDefaultConfiguration()
	config.ValidationMode = pdfcpu.ValidationRelaxed

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			inFile := filepath.Join(inDir, file.Name())
			_, err = Process(ValidateCommand(inFile, config))
			if err != nil {
				t.Fatalf("TestValidateCommand: %v\n", err)
			}
		}
	}

}

func TestValidateOneFile(t *testing.T) {

	config := pdfcpu.NewDefaultConfiguration()
	config.ValidationMode = pdfcpu.ValidationRelaxed

	inFile := filepath.Join(inDir, "gobook.0.pdf")
	_, err := Process(ValidateCommand(inFile, config))
	if err != nil {
		t.Fatalf("TestValidateOneFile: %v\n", err)
	}

}

func BenchmarkValidateCommand(b *testing.B) {

	config := pdfcpu.NewDefaultConfiguration()
	config.ValidationMode = pdfcpu.ValidationRelaxed

	inFile := filepath.Join(inDir, "gobook.0.pdf")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := Process(ValidateCommand(inFile, config))
		if err != nil {
			b.Fatalf("BenchmarkValidateCommand: %v\n", err)
		}
	}
}

// Optimize all PDFs in testdata and write with (default) end of line sequence "\n".
func TestOptimizeCommandWithLF(t *testing.T) {

	files, err := ioutil.ReadDir(inDir)
	if err != nil {
		t.Fatalf("TestOptimizeCommandWithLF: %v\n", err)
	}

	config := pdfcpu.NewDefaultConfiguration()

	// this is not necessary but to make it clearer.
	config.Eol = pdfcpu.EolLF
	outFile := filepath.Join(outDir, "test.pdf")

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {

			inFile := filepath.Join(inDir, file.Name())

			_, err = Process(OptimizeCommand(inFile, outFile, config))
			if err != nil {
				t.Fatalf("TestOptimizeCommandWithLF: %v\n", err)
			}

		}
	}

}

// Optimize all PDFs in testdata and write with end of line sequence "\r".
func TestOptimizeCommandWithCR(t *testing.T) {

	files, err := ioutil.ReadDir(inDir)
	if err != nil {
		t.Fatalf("TestOptimizeCommandWithCR: %v\n", err)
	}

	config := pdfcpu.NewDefaultConfiguration()
	config.Eol = pdfcpu.EolCR

	outFile := filepath.Join(outDir, "test.pdf")

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {

			inFile := filepath.Join(inDir, file.Name())

			_, err = Process(OptimizeCommand(inFile, outFile, config))
			if err != nil {
				t.Fatalf("TestOptimizeCommandWithCR: %v\n", err)
			}

			_, err = Process(ValidateCommand(outFile, config))
			if err != nil {
				t.Fatalf("TestOptimizeCommandWithCR validation: %v\n", err)
			}

		}
	}

}

// Optimize all PDFs in testdata and write with end of line sequence "\r".
// This test writes out the cross reference table the old way without using object streams and an xref stream.
func TestOptimizeCommandWithCRAndNoXrefStream(t *testing.T) {

	files, err := ioutil.ReadDir(inDir)
	if err != nil {
		t.Fatalf("TestOptimizeCommandWithCRAndNoXrefStream: %v\n", err)
	}

	config := pdfcpu.NewDefaultConfiguration()
	config.Eol = pdfcpu.EolCR
	config.WriteObjectStream = false
	config.WriteXRefStream = false

	outFile := filepath.Join(outDir, "test.pdf")

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			inFile := filepath.Join(inDir, file.Name())
			_, err = Process(OptimizeCommand(inFile, outFile, config))
			if err != nil {
				t.Fatalf("TestOptimizeCommandWithCRAndNoXrefStream: %v\n", err)
			}
		}
	}

}

// Optimize all PDFs in testdata and write with end of line sequence "\r\n".
func TestOptimizeCommandWithCRLF(t *testing.T) {

	files, err := ioutil.ReadDir(inDir)
	if err != nil {
		t.Fatalf("TestOptimizeCommmand: %v\n", err)
	}

	config := pdfcpu.NewDefaultConfiguration()
	config.Eol = pdfcpu.EolCRLF
	config.StatsFileName = outDir + "/testStats.csv"

	outFile := filepath.Join(outDir, "test.pdf")

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			inFile := filepath.Join(inDir, file.Name())
			_, err = Process(OptimizeCommand(inFile, outFile, config))
			if err != nil {
				t.Fatalf("TestOptimizeCommand: %v\n", err)
			}
		}
	}

}

// Split a test PDF file up into single page PDFs.
func TestSplitCommand(t *testing.T) {

	_, err := Process(SplitCommand("testdata/Acroforms2.pdf", outDir, pdfcpu.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestSplitCommand: %v\n", err)
	}
}

// Merge all PDFs in testdir into out/test.pdf.
func TestMergeCommand(t *testing.T) {

	files, err := ioutil.ReadDir(inDir)
	if err != nil {
		t.Fatalf("TestMergeCommmand: %v\n", err)
	}

	inFiles := []string{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			inFile := filepath.Join(inDir, file.Name())
			inFiles = append(inFiles, inFile)
		}
	}

	outFile := filepath.Join(outDir, "test.pdf")
	_, err = Process(MergeCommand(inFiles, outFile, pdfcpu.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestMergeCommand: %v\n", err)
	}

}

// Trim test PDF file so that only the first two pages are rendered.
func TestTrimCommand(t *testing.T) {

	inFile := filepath.Join(inDir, "pike-stanford.pdf")
	outFile := filepath.Join(outDir, "test.pdf")

	_, err := Process(TrimCommand(inFile, outFile, []string{"-2"}, pdfcpu.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestTrimCommand: %v\n", err)
	}

}

func TestWatermark(t *testing.T) {

	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "testwm.pdf")

	onTop := false
	wm, err := pdfcpu.ParseWatermarkDetails("Draft, s:0.7, r:20", onTop)
	if err != nil {
		t.Fatalf("TestWatermark: %v\n", err)
	}

	_, err = Process(AddWatermarksCommand(inFile, outFile, []string{"1-"}, wm, pdfcpu.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestWatermark: %v\n", err)
	}

}

// Stamp all but page 1.
func TestStampCommand(t *testing.T) {

	inFile := filepath.Join(inDir, "pike-stanford.pdf")
	outFile := filepath.Join(outDir, "teststamp.pdf")

	onTop := true
	wm, err := pdfcpu.ParseWatermarkDetails("Demo, f:Courier, c: 0 .8 0, o:0.8", onTop)
	if err != nil {
		t.Fatalf("TestStampCommand: %v\n", err)
	}

	_, err = Process(AddWatermarksCommand(inFile, outFile, []string{"1-"}, wm, pdfcpu.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestStampCommand: %v\n", err)
	}

}

func TestWatermarkImage(t *testing.T) {

	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "testWMImage.pdf")

	onTop := false
	wm, err := pdfcpu.ParseWatermarkDetails("../../resources/pdfchip3.png, r:0", onTop)
	if err != nil {
		t.Fatalf("TestWatermarkImage: %v\n", err)
	}

	_, err = Process(AddWatermarksCommand(inFile, outFile, []string{"1-"}, wm, pdfcpu.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestWatermarkImage: %v\n", err)
	}

}

func TestExtractImagesCommand(t *testing.T) {

	files, err := ioutil.ReadDir(inDir)
	if err != nil {
		t.Fatalf("TestExtractImagesCommand: %v\n", err)
	}

	c := pdfcpu.NewDefaultConfiguration()

	cmd := ExtractImagesCommand("", outDir, nil, c)

	for _, file := range files {

		if !strings.HasSuffix(file.Name(), "pdf") {
			continue
		}

		inFile := filepath.Join(inDir, file.Name())
		cmd.InFile = &inFile

		// Extract all images.
		_, err := Process(cmd)
		if err != nil {
			t.Fatalf("TestExtractImageCommand: %v\n", err)
		}

	}

	// Extract images starting with page 1.
	inFile := filepath.Join(inDir, "testImage.pdf")
	_, err = Process(ExtractImagesCommand(inFile, outDir, []string{"1-"}, pdfcpu.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestExtractImageCommand: %v\n", err)
	}

}

func TestExtractFontsCommand(t *testing.T) {

	cmd := ExtractFontsCommand("", outDir, nil, pdfcpu.NewDefaultConfiguration())

	for _, fn := range []string{"5116.DCT_Filter.pdf", "testImage.pdf", "go.pdf"} {

		fn = filepath.Join(inDir, fn)
		cmd.InFile = &fn

		_, err := Process(cmd)
		if err != nil {
			t.Fatalf("TestExtractFontsCommand: %v\n", err)
		}

	}

	inFile := filepath.Join(inDir, "go.pdf")
	_, err := Process(ExtractFontsCommand(inFile, outDir, []string{"1-3"}, pdfcpu.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestExtractFontsCommand: %v\n", err)
	}

}

func TestExtractContentCommand(t *testing.T) {

	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")

	_, err := Process(ExtractContentCommand(inFile, outDir, nil, pdfcpu.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestExtractContentCommand: %v\n", err)
	}

}

func TestExtractPagesCommand(t *testing.T) {

	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")

	_, err := Process(ExtractPagesCommand(inFile, outDir, []string{"1"}, pdfcpu.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestExtractPagesCommand: %v\n", err)
	}

}

func TestEncryptUPWOnly(t *testing.T) {

	// Test for setting only the user password.

	t.Log("running TestEncryptUPWOnly..")

	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")
	outFile := filepath.Join(outDir, "test.pdf")

	// Encrypt upw only
	t.Log("Encrypt upw only")
	config := pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	_, err := Process(EncryptCommand(inFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptUPWOnly - encrypt with upw only to %s: %v\n", outFile, err)
	}

	// Validate wrong upw
	t.Log("Validate wrong upw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(ValidateCommand(outFile, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - validate %s using wrong upw should fail!\n", outFile)
	}

	// Validate wrong opw
	t.Log("Validate wrong opw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(ValidateCommand(outFile, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - validate %s using wrong opw should fail!\n", outFile)
	}

	// Validate default opw=upw (if there is no ownerpw set)
	t.Log("Validate default opw")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "upw"
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptUPWOnly - validate %s using default opw: %s!\n", outFile, err)
	}

	// Validate upw
	t.Log("Validate upw")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptUPWOnly - validate %s using upw: %v\n", outFile, err)
	}

	// Optimize wrong opw
	t.Log("Optimize wrong opw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(OptimizeCommand(outFile, outFile, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - optimize %s using wrong opw should fail!\n", outFile)
	}

	// Optimize empty opw
	t.Log("Optimize empty opw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = ""
	_, err = Process(OptimizeCommand(outFile, outFile, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - optimize %s using empty opw should fail!\n", outFile)
	}

	// Optimize wrong upw
	t.Log("Optimize wrong upw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(OptimizeCommand(outFile, outFile, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - optimize %s using wrong upw should fail!\n", outFile)
	}

	// Optimize upw
	t.Log("Optimize upw")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	_, err = Process(OptimizeCommand(outFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptUPWOnly - optimize %s using upw: %v\n", outFile, err)
	}

	//Change upw wrong upwOld
	t.Log("ChangeUserPW wrong upwOld fails")
	config = pdfcpu.NewDefaultConfiguration()
	pwOld := "upwWrong"
	pwNew := "upwNew"
	_, err = Process(ChangeUserPWCommand(outFile, outFile, config, &pwOld, &pwNew))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - %s change userPW using wrong upwOld should fail\n", outFile)
	}

	// Change upw
	t.Log("ChangeUserPW")
	config = pdfcpu.NewDefaultConfiguration()
	pwOld = "upw"
	pwNew = "upwNew"
	_, err = Process(ChangeUserPWCommand(outFile, outFile, config, &pwOld, &pwNew))
	if err != nil {
		t.Fatalf("TestEncryptUPWOnly - %s change userPW: %v\n", outFile, err)
	}

	// Decrypt wrong opw
	t.Log("Decrypt wrong opw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(DecryptCommand(outFile, outFile, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - %s decrypt using wrong opw should fail\n", outFile)
	}

	// Decrypt wrong upw
	t.Log("Decrypt wrong upw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	_, err = Process(DecryptCommand(outFile, outFile, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - %s decrypt using wrong upw should fail\n", outFile)
	}

	// Decrypt upw
	t.Log("Decrypt upw")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upwNew"
	_, err = Process(DecryptCommand(outFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptUPWOnly - %s decrypt using upw: %v\n", outFile, err)
	}

}

func TestEncryptOPWOnly(t *testing.T) {

	// Test for setting only the owner password.

	t.Log("running TestEncryptOPWOnly..")

	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")
	outFile := filepath.Join(outDir, "test.pdf")

	// Encrypt opw only
	t.Log("Encrypt opw only")
	config := pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	_, err := Process(EncryptCommand(inFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - encrypt with opw only to %s: %v\n", outFile, err)
	}

	// Validate wrong opw succeeds with fallback to empty upw
	t.Log("Validate wrong opw succeeds with fallback to empty upw")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - validate %s using wrong opw succeeds falling back to empty upw!: %v\n", outFile, err)
	}

	// Validate opw
	t.Log("Validate opw")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - validate %s using opw: %v\n", outFile, err)
	}

	// Validate wrong upw
	t.Log("Validate wrong upw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(ValidateCommand(outFile, config))
	if err == nil {
		t.Fatalf("TestEncryptOPWOnly - validate %s using wrong upw should fail!\n", outFile)
	}

	// Validate no pw using empty upw
	t.Log("Validate no pw using empty upw")
	config = pdfcpu.NewDefaultConfiguration()
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - validate %s no pw using empty upw: %v\n", outFile, err)
	}

	// Optimize wrong opw, succeeds with fallback to empty upw
	t.Log("Optimize wrong opw succeeds with fallback to empty upw")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(OptimizeCommand(outFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - optimize %s using wrong opw succeeds falling back to empty upw: %v\n", outFile, err)
	}

	// Optimize opw
	t.Log("Optimize opw")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	_, err = Process(OptimizeCommand(outFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - optimize %s using opw: %v\n", outFile, err)
	}

	// Optimize wrong upw
	t.Log("Optimize wrong upw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(OptimizeCommand(outFile, outFile, config))
	if err == nil {
		t.Fatalf("TestEncryptOPWOnly - optimize %s using wrong upw should fail!\n", outFile)
	}

	// Optimize empty upw
	t.Log("Optimize empty upw")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = ""
	_, err = Process(OptimizeCommand(outFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - optimize %s using upw: %v\n", outFile, err)
	}

	// Change opw wrong upw
	t.Log("ChangeOwnerPW wrong upw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	pwOld := "opw"
	pwNew := "opwNew"
	_, err = Process(ChangeOwnerPWCommand(outFile, outFile, config, &pwOld, &pwNew))
	if err == nil {
		t.Fatalf("TestEncryptOPWOnly - %s change opw using wrong upw should fail\n", outFile)
	}

	// Change opw wrong opwOld
	t.Log("ChangeOwnerPW wrong opwOld fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = ""
	pwOld = "opwOldWrong"
	pwNew = "opwNew"
	_, err = Process(ChangeOwnerPWCommand(outFile, outFile, config, &pwOld, &pwNew))
	if err == nil {
		t.Fatalf("TestEncryptOPWOnly - %s change opw using wrong opwOld should fail\n", outFile)
	}

	// Change opw
	t.Log("ChangeOwnerPW")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = ""
	pwOld = "opw"
	pwNew = "opwNew"
	_, err = Process(ChangeOwnerPWCommand(outFile, outFile, config, &pwOld, &pwNew))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - %s change opw: %v\n", outFile, err)
	}

	// Decrypt wrong upw
	t.Log("Decrypt wrong upw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(DecryptCommand(outFile, outFile, config))
	if err == nil {
		t.Fatalf("TestEncryptOPWOnly - %s decrypt using wrong upw should fail \n", outFile)
	}

	// Decrypt wrong opw succeeds because of fallback to empty upw.
	t.Log("Decrypt wrong opw succeeds because of fallback to empty upw")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	_, err = Process(DecryptCommand(outFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - %s decrypt using opw: %v\n", outFile, err)
	}

}

func TestEncrypt(t *testing.T) {

	t.Log("running TestEncrypt..")

	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")
	outFile := filepath.Join(outDir, "test.pdf")

	// Encrypt opw and upw
	t.Log("Encrypt")
	config := pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	_, err := Process(EncryptCommand(inFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncrypt - encrypt to %s: %v\n", outFile, err)
	}

	// Validate wrong opw
	t.Log("Validate wrong opw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(ValidateCommand(outFile, config))
	if err == nil {
		t.Fatalf("TestEncrypt - validate %s using wrong opw should fail!\n", outFile)
	}

	// Validate opw
	t.Log("Validate opw")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("TestEncrypt - validate %s using opw: %v\n", outFile, err)
	}

	// Validate wrong upw
	t.Log("Validate wrong upw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(ValidateCommand(outFile, config))
	if err == nil {
		t.Fatalf("TestEncrypt - validate %s using wrong upw should fail!\n", outFile)
	}

	// Validate upw
	t.Log("Validate upw")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("TestEncrypt - validate %s using upw: %v\n", outFile, err)
	}

	// Change upw to "" = remove document open password.
	t.Log("ChangeUserPW to \"\"")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	pwOld := "upw"
	pwNew := ""
	_, err = Process(ChangeUserPWCommand(outFile, outFile, config, &pwOld, &pwNew))
	if err != nil {
		t.Fatalf("TestEncrypt - %s change userPW to \"\": %v\n", outFile, err)
	}

	// Validate upw
	t.Log("Validate upw")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = ""
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("TestEncrypt - validate %s using upw: %v\n", outFile, err)
	}

	// Validate no pw
	t.Log("Validate upw")
	config = pdfcpu.NewDefaultConfiguration()
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("TestEncrypt - validate %s: %v\n", outFile, err)
	}

	// Change opw
	t.Log("ChangeOwnerPW")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = ""
	pwOld = "opw"
	pwNew = "opwNew"
	_, err = Process(ChangeOwnerPWCommand(outFile, outFile, config, &pwOld, &pwNew))
	if err != nil {
		t.Fatalf("TestEncrypt - %s change opw: %v\n", outFile, err)
	}

	// Decrypt wrong upw
	t.Log("Decrypt wrong upw fails")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(DecryptCommand(outFile, outFile, config))
	if err == nil {
		t.Fatalf("TestEncrypt - %s decrypt using wrong upw should fail\n", outFile)
	}

	// Decrypt wrong opw succeeds on empty upw
	t.Log("Decrypt wrong opw succeeds on empty upw")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(DecryptCommand(outFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncrypt - %s decrypt wrong opw, empty upw: %v\n", outFile, err)
	}
}

func encryptDecrypt(fileName string, config *pdfcpu.Configuration, t *testing.T) {

	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, "test.pdf")

	t.Log(inFile)

	// Encrypt
	t.Log("Encrypt")
	_, err := Process(EncryptCommand(inFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - encrypt %s: %v\n", outFile, err)
	}

	// Encrypt already encrypted
	t.Log("Encrypt already encrypted")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	_, err = Process(EncryptCommand(outFile, outFile, config))
	if err == nil {
		t.Fatalf("TestEncryptDecrypt - encrypt encrypted %s\n", outFile)
	}

	// Validate using wrong owner pw
	t.Log("Validate wrong ownerPW")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opwWrong"
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - validate %s using wrong ownerPW: %v\n", outFile, err)
	}

	// Optimize using wrong owner pw
	t.Log("Optimize wrong ownerPW")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opwWrong"
	_, err = Process(OptimizeCommand(outFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - optimize %s using wrong ownerPW: %v\n", outFile, err)
	}

	// Trim using wrong owner pw, falls back to upw and fails with insufficient permissions.
	t.Log("Trim wrong ownerPW, fallback to upw and fail with insufficient permissions.")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opwWrong"
	// pageSelection = nil, writes w/o trimming anything, but sufficient for testing.
	_, err = Process(TrimCommand(outFile, outFile, nil, config))
	if err == nil {
		t.Fatalf("TestEncryptDecrypt - trim %s using wrong ownerPW should fail: \n", outFile)
	}

	// Split using wrong owner pw, falls back to upw and fails with insufficient permissions.
	t.Log("Split wrong ownerPW, fallback to upw and fail with insufficient permissions.")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opwWrong"
	_, err = Process(SplitCommand(outFile, outDir, config))
	if err == nil {
		t.Fatalf("TestEncryptDecrypt - split %s using wrong ownerPW should fail: \n", outFile)
	}

	// Add permissions
	t.Log("Add user access permissions")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	config.UserAccessPermissions = pdfcpu.PermissionsAll
	_, err = Process(AddPermissionsCommand(outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - %s add permissions: %v\n", outFile, err)
	}

	// Split using wrong owner pw, falls back to upw
	t.Log("Split wrong ownerPW")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opwWrong"
	_, err = Process(SplitCommand(outFile, outDir, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - split %s using wrong ownerPW: %v\n", outFile, err)
	}

	// Validate
	t.Log("Validate")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - validate %s: %v\n", outFile, err)
	}

	// ChangeUserPW using wrong userpw
	t.Log("ChangeUserPW wrong userpw")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	pwOld := "upwWrong"
	pwNew := "upwNew"
	_, err = Process(ChangeUserPWCommand(outFile, outFile, config, &pwOld, &pwNew))
	if err == nil {
		t.Fatalf("TestEncryption - %s change userPW using wrong userPW should fail:\n", outFile)
	}

	// ChangeUserPW
	t.Log("ChangeUserPW")
	config = pdfcpu.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	pwOld = "upw"
	pwNew = "upwNew"
	_, err = Process(ChangeUserPWCommand(outFile, outFile, config, &pwOld, &pwNew))
	if err != nil {
		t.Fatalf("TestEncryption - change userPW %s: %v\n", outFile, err)
	}

	// ChangeOwnerPW
	t.Log("ChangeOwnerPW")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upwNew"
	pwOld = "opw"
	pwNew = "opwNew"
	_, err = Process(ChangeOwnerPWCommand(outFile, outFile, config, &pwOld, &pwNew))
	if err != nil {
		t.Fatalf("TestEncryption - change ownerPW %s: %v\n", outFile, err)
	}

	// Decrypt using wrong pw
	t.Log("\nDecrypt using wrong pw")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	config.OwnerPW = "opwWrong"
	_, err = Process(DecryptCommand(outFile, outFile, config))
	if err == nil {
		t.Fatalf("TestEncryptDecrypt - decrypt using wrong pw %s\n", outFile)
	}

	// Decrypt
	t.Log("\nDecrypt")
	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upwNew"
	config.OwnerPW = "opwNew"
	_, err = Process(DecryptCommand(outFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - decrypt %s: %v\n", outFile, err)
	}

}

func TestEncryptDecrypt(t *testing.T) {

	config := pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	encryptDecrypt("5116.DCT_Filter.pdf", config, t)

	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	config.EncryptUsingAES = false
	config.EncryptUsing128BitKey = false
	encryptDecrypt("networkProgr.pdf", config, t)
}

func copyFile(srcFileName, destFileName string) (err error) {

	from, err := os.Open(srcFileName)
	if err != nil {
		return
	}
	defer from.Close()

	to, err := os.OpenFile(destFileName, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return
	}
	defer to.Close()

	_, err = io.Copy(to, from)

	return
}

func prepareForAttachmentTest() (err error) {

	for _, fileName := range []string{"go.pdf", "golang.pdf", "T4.pdf", "go-lecture.pdf", "test.wav"} {
		inFile := filepath.Join(inDir, fileName)
		outFile := filepath.Join(outDir, fileName)
		err = copyFile(inFile, outFile)
		if err != nil {
			return
		}
	}

	return
}

func testAttachmentsStage1(fileName string, config *pdfcpu.Configuration, t *testing.T) {

	// attach list must be 0
	list, err := Process(ListAttachmentsCommand(fileName, config))
	if err != nil {
		t.Fatalf("TestAttachments - list attachments %s: %v\n", fileName, err)
	}
	if len(list) > 0 {
		t.Fatalf("TestAttachments - list attachments %s: should have 0 attachments\n", fileName)
	}

	// attach add 4 files
	_, err = Process(AddAttachmentsCommand(fileName,
		[]string{outDir + "/golang.pdf",
			outDir + "/T4.pdf",
			outDir + "/go-lecture.pdf",
			outDir + "/test.wav"},
		config))

	if err != nil {
		t.Fatalf("TestAttachments - add attachments to %s: %v\n", fileName, err)
	}

	// attach list must be 4
	list, err = Process(ListAttachmentsCommand(fileName, config))
	if err != nil {
		t.Fatalf("TestAttachments - list attachments %s: %v\n", fileName, err)
	}
	if len(list) != 4 {
		t.Fatalf("TestAttachments - list attachments %s: should have 4 attachments\n", fileName)
	}
	for _, s := range list {
		t.Log(s)
	}

}

func testAttachmentsStage2(fileName string, config *pdfcpu.Configuration, t *testing.T) {

	// attach extract all
	_, err := Process(ExtractAttachmentsCommand(fileName, outDir, nil, config))
	if err != nil {
		t.Fatalf("TestAttachments - extract all attachments from %s to %s: %v\n", fileName, ".", err)
	}

	// attach extract 1 file
	_, err = Process(ExtractAttachmentsCommand(fileName, outDir, []string{"golang.pdf"}, config))
	if err != nil {
		t.Fatalf("TestAttachments - extract 1 attachment from %s to %s: %v\n", fileName, ".", err)
	}

	// attach remove 1 file
	_, err = Process(RemoveAttachmentsCommand(fileName, []string{"golang.pdf"}, config))
	if err != nil {
		t.Fatalf("TestAttachments - remove attachment from %s: %v\n", fileName, err)
	}

	// attach list must be 3
	list, err := Process(ListAttachmentsCommand(fileName, config))
	if err != nil {
		t.Fatalf("TestAttachments - list attachments %s: %v\n", fileName, err)
	}
	if len(list) != 3 {
		t.Fatalf("TestAttachments - list attachments %s: should have 3 attachments\n", fileName)
	}

	// attach remove all
	_, err = Process(RemoveAttachmentsCommand(fileName, nil, config))
	if err != nil {
		t.Fatalf("TestAttachments - remove all attachment from %s: %v\n", fileName, err)
	}

	// attach list must be 0.
	list, err = Process(ListAttachmentsCommand(fileName, config))
	if err != nil {
		t.Fatalf("TestAttachments - list attachments %s: %v\n", fileName, err)
	}
	t.Log(list)
	if len(list) > 0 {
		t.Fatalf("TestAttachments - list attachments %s: should have 0 attachments\n", fileName)
	}
}

func TestAttachments(t *testing.T) {

	err := prepareForAttachmentTest()
	if err != nil {
		t.Fatalf("prepare for attachments: %v\n", err)
	}

	config := pdfcpu.NewDefaultConfiguration()

	fileName := filepath.Join(outDir, "go.pdf")

	testAttachmentsStage1(fileName, config, t)
	testAttachmentsStage2(fileName, config, t)
}

func TestListPermissionsCommand(t *testing.T) {

	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")
	outFile := filepath.Join(outDir, "test.pdf")

	_, err := Process(ListPermissionsCommand(inFile, pdfcpu.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestListPermissionsCommand: for unencrypted %s: %v\n", inFile, err)
	}

	config := pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	_, err = Process(EncryptCommand(inFile, outFile, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - encrypt %s: %v\n", outFile, err)
	}

	config = pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	_, err = Process(ListPermissionsCommand(outFile, config))
	if err != nil {
		t.Fatalf("TestListPermissionsCommand: for encrypted %s: %v\n", outFile, err)
	}

}

func TestUnknownCommand(t *testing.T) {

	config := pdfcpu.NewDefaultConfiguration()
	inFile := filepath.Join(outDir, "go.pdf")

	cmd := &Command{
		Mode:   99,
		InFile: &inFile,
		Config: config}

	_, err := Process(cmd)
	if err == nil {
		t.Fatal("TestUnknowncommand - should have failed")
	}

}

func xxxTestDemoXRef(t *testing.T) {

	xRefTable, err := pdfcpu.CreateAnnotationDemoXRef()
	if err != nil {
		t.Fatalf("testDemoXRef %v\n", err)
	}
	if xRefTable == nil {
		t.Fatal("testDemoXRef: xRefTable == nil")
	}

	err = validate.XRefTable(xRefTable)
	if err != nil {
		t.Fatalf("testDemoXRef %v\n", err)
	}

	// var logStr []string
	// logStr = xRefTable.List(logStr)
	// t.Logf("XRefTable:\n%s\n", strings.Join(logStr, ""))
}

func TestCreateDemoPDF(t *testing.T) {

	xRefTable, err := pdfcpu.CreateDemoXRef()
	if err != nil {
		t.Fatalf("testCreateDemoPDF %v\n", err)
	}

	err = pdfcpu.CreatePDF(xRefTable, outDir+"/", "demo.pdf")
	if err != nil {
		t.Fatalf("testCreateDemoPDF %v\n", err)
	}

	config := pdfcpu.NewDefaultConfiguration()
	config.ValidationMode = pdfcpu.ValidationRelaxed

	outFile := filepath.Join(outDir, "demo.pdf")
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("testCreateDemoPDF %v\n", err)
	}
}

func TestAnnotationDemoPDF(t *testing.T) {

	xRefTable, err := pdfcpu.CreateAnnotationDemoXRef()
	if err != nil {
		t.Fatalf("testAnnotationDemoPDF %v\n", err)
	}

	err = pdfcpu.CreatePDF(xRefTable, outDir+"/", "annotationDemo.pdf")
	if err != nil {
		t.Fatalf("testAnnotationDemoPDF %v\n", err)
	}

	config := pdfcpu.NewDefaultConfiguration()
	config.ValidationMode = pdfcpu.ValidationRelaxed

	outFile := filepath.Join(outDir, "annotationDemo.pdf")
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("testAnnotationDemoPDF %v\n", err)
	}

}

func TestAcroformDemoPDF(t *testing.T) {

	xRefTable, err := pdfcpu.CreateAcroFormDemoXRef()
	if err != nil {
		t.Fatalf("testAcroformDemoPDF %v\n", err)
	}

	err = pdfcpu.CreatePDF(xRefTable, outDir+"/", "acroFormDemo.pdf")
	if err != nil {
		t.Fatalf("testAcroformDemoPDF %v\n", err)
	}

	config := pdfcpu.NewDefaultConfiguration()
	config.ValidationMode = pdfcpu.ValidationRelaxed

	outFile := filepath.Join(outDir, "acroFormDemo.pdf")
	_, err = Process(ValidateCommand(outFile, config))
	if err != nil {
		t.Fatalf("testAcroformDemoPDF %v\n", err)
	}

}
