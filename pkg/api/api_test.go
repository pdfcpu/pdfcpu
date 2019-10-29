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

package api

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

var inDir, outDir, resDir string

func TestMain(m *testing.M) {
	inDir = "../testdata"
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

func copyFile(srcFileName, destFileName string) error {

	from, err := os.Open(srcFileName)
	if err != nil {
		return err
	}
	defer from.Close()

	to, err := os.OpenFile(destFileName, os.O_RDWR|os.O_CREATE, 0666)
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
		if err = Validate(f, nil); err != nil {
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
	gotPageCount, err := PageCountFile(inFile)
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
		_, err := PageDimsFile(inFile)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
	}
}

func TestValidate(t *testing.T) {
	msg := "TestValidate"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")

	// Validate inFile.
	if err := ValidateFile(inFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestOptimize(t *testing.T) {
	msg := "TestOptimize"
	fileName := "Acroforms2.pdf"
	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, fileName)

	// Create an optimized version of inFile.
	if err := OptimizeFile(inFile, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Create an optimized version of inFile.
	// If you want to modify the original file, pass an empty string for outFile.
	inFile = outFile
	if err := OptimizeFile(inFile, "", nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestTrim(t *testing.T) {
	msg := "TestTrim"
	fileName := "adobe_errata.pdf"
	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, fileName)

	// Create a trimmed version of inFile containing odd page numbers only.
	if err := TrimFile(inFile, outFile, []string{"odd"}, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Create a trimmed version of inFile containing the first two pages only.
	// If you want to modify the original file, pass an empty string for outFile.
	inFile = outFile
	if err := TrimFile(inFile, "", []string{"1-2"}, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestSplit(t *testing.T) {
	msg := "TestSplit"
	fileName := "Acroforms2.pdf"
	inFile := filepath.Join(inDir, fileName)

	// Create single page files of inFile in outDir.
	if err := SplitFile(inFile, outDir, 1, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestRotate(t *testing.T) {
	msg := "TestRotate"
	fileName := "Acroforms2.pdf"
	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, fileName)

	// Rotate all pages of inFile, clockwise by 90 degrees and write the result to outFile.
	if err := RotateFile(inFile, outFile, 90, nil, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Rotate the first page of inFile by 180 degrees.
	// If you want to modify the original file, pass an empty string for outFile.
	inFile = outFile
	if err := RotateFile(inFile, "", 180, []string{"1"}, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestMerge(t *testing.T) {
	msg := "TestMerge"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}
	outFile := filepath.Join(outDir, "test.pdf")

	// Merge inFiles by concatenation in the order specified and write the result to outFile.
	if err := MergeFile(inFiles, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestInsertRemovePages(t *testing.T) {
	msg := "TestInsertRemovePages"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "test.pdf")

	n1, err := PageCountFile(inFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}

	// Insert an empty page before pages 1 and 2.
	if err := InsertPagesFile(inFile, outFile, []string{"-2"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	n2, err := PageCountFile(outFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if n2 != n1+2 {
		t.Fatalf("%s %s: pageCount want:%d got:%d\n", msg, inFile, n1+2, n2)
	}

	// 	// Remove pages 1 and 2.
	if err := RemovePagesFile(outFile, "", []string{"-2"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	n2, err = PageCountFile(outFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
	if n1 != n2 {
		t.Fatalf("%s %s: pageCount want:%d got:%d\n", msg, inFile, n1, n2)
	}
}

func testAddWatermarks(t *testing.T, msg, inFile, outFile string, selectedPages []string, wmConf string, onTop bool) {
	t.Helper()
	inFile = filepath.Join(inDir, inFile)
	outFile = filepath.Join(outDir, outFile)
	wm, err := pdf.ParseWatermarkDetails(wmConf, onTop)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := AddWatermarksFile(inFile, outFile, selectedPages, wm, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestAddWatermarks(t *testing.T) {
	for _, tt := range []struct {
		msg             string
		inFile, outFile string
		selectedPages   []string
		onTop           bool
		wmConf          string
	}{
		// Add text watermark to all pages of inFile starting at page 1 using a rotation angle of 20 degrees.
		{"TestWatermarkText",
			"Acroforms2.pdf",
			"testwm.pdf",
			[]string{"1-"},
			false,
			"Draft, s:0.7, rot:20"},

		// Add a greenish, slightly transparent stroked and filled text stamp to all odd pages of inFile other than page 1
		// using the default rotation which is aligned along the first diagonal running from lower left to upper right corner.
		{"TestStampText",
			"pike-stanford.pdf",
			"testStampText1.pdf",
			[]string{"odd", "!1"},
			true,
			"Demo, f:Courier, c: 0 .8 0, op:0.8, m:2"},

		// Add a red filled text stamp to all odd pages of inFile other than page 1 using a font size of 48 points
		// and the default rotation which is aligned along the first diagonal running from lower left to upper right corner.
		{"TestStampTextUsingFontsize",
			"pike-stanford.pdf",
			"testStampText2.pdf",
			[]string{"odd", "!1"},
			true,
			"Demo, font:Courier, c: 1 0 0, op:1, s:1 abs, points:48"},

		// Add image watermark to inFile starting at page 1 using no rotation.
		{"TestWatermarkImage",
			"Acroforms2.pdf", "testWMImageRel.pdf",
			[]string{"1-"},
			false,
			filepath.Join(resDir, "pdfchip3.png") + ", rot:0"},

		// Add image stamp to inFile using absolute scaling and a negative rotation of 90 degrees.
		{"TestStampImageAbsScaling",
			"Acroforms2.pdf",
			"testWMImageAbs.pdf",
			[]string{"1-"},
			true,
			filepath.Join(resDir, "pdfchip3.png") + ", s:.5 a, rot:-90"},

		// Add a PDF stamp to all pages of inFile using the 2nd page of pdfFile
		// and rotate along the 2nd diagonal running from upper left to lower right corner.
		{"TestWatermarkText",
			"Acroforms2.pdf",
			"testStampPDF.pdf",
			nil,
			true,
			filepath.Join(inDir, "Wonderwall.pdf") + ":2, d:2"},
	} {
		testAddWatermarks(t, tt.msg, tt.inFile, tt.outFile, tt.selectedPages, tt.wmConf, tt.onTop)
	}
}

func TestStampingLifecyle(t *testing.T) {
	msg := "TestStampingLifecyle"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "stampLC.pdf")
	onTop := true // we are testing stamps

	// Stamp all pages.
	wm, err := pdf.ParseWatermarkDetails("Demo", onTop)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := AddWatermarksFile(inFile, outFile, nil, wm, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// // Update stamp on page 1.
	wm, err = pdf.ParseWatermarkDetails("Confidential", onTop)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	wm.Update = true
	if err := AddWatermarksFile(outFile, "", []string{"1"}, wm, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Add another stamp on top for all pages.
	// This is a redish transparent footer.
	wm, err = pdf.ParseWatermarkDetails("Footer, pos:bc, c:0.8 0 0, op:.6, rot:0", onTop)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := AddWatermarksFile(outFile, "", nil, wm, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Remove stamp on page 1.
	if err := RemoveWatermarksFile(outFile, "", []string{"1"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Remove all stamps.
	if err := RemoveWatermarksFile(outFile, "", nil, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Validate the result.
	if err := ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func testImportImages(t *testing.T, msg string, imgFiles []string, outFile, impConf string, ensureOutFile bool) {
	t.Helper()
	var err error

	outFile = filepath.Join(outDir, outFile)
	if ensureOutFile {
		// We want to test appending to an existing PDF.
		copyFile(filepath.Join(inDir, outFile), outFile)
	}

	// The default import conf uses the special pos:full argument
	// which overrides all other import conf parms.
	imp := pdf.DefaultImportConfig()
	if impConf != "" {
		if imp, err = pdf.ParseImportDetails(impConf); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	}
	if err := ImportImagesFile(imgFiles, outFile, imp, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestImportImages(t *testing.T) {
	for _, tt := range []struct {
		msg           string
		imgFiles      []string
		outFile       string
		impConf       string
		ensureOutFile bool
	}{
		// Convert an image into a single page PDF.
		// The page dimensions will match the image dimensions.
		{"TestConvertImageToPDF",
			[]string{filepath.Join(resDir, "pdfchip3.png")},
			"testConvertImage.pdf",
			"",
			false},

		// Import an image as a new page of the existing output file.
		{"TestImportImage",
			[]string{filepath.Join(resDir, "pdfchip3.png")},
			"Acroforms2.pdf",
			"",
			true},

		// Import images by creating an A3 page for each image.
		// Images are page centered with 1.0 relative scaling.
		// Import an image as a new page of the existing output file.
		{"TestCenteredImportImage",
			[]string{
				filepath.Join(resDir, "pdfchip3.png"),
				filepath.Join(resDir, "demo.png"),
				filepath.Join(resDir, "snow.jpg"),
			},
			"Acroforms2.pdf",
			"f:A3, pos:c, s:1.0",
			true},
	} {
		testImportImages(t, tt.msg, tt.imgFiles, tt.outFile, tt.impConf, tt.ensureOutFile)
	}
}

func testNUp(t *testing.T, msg string, inFiles []string, outFile string, selectedPages []string, desc string, n int, isImg bool) {
	t.Helper()

	var (
		nup *pdf.NUp
		err error
	)

	if isImg {
		if nup, err = pdf.ImageNUpConfig(n, desc); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	} else {
		if nup, err = pdf.PDFNUpConfig(n, desc); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	}

	if err := NUpFile(inFiles, outFile, selectedPages, nup, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestNUp(t *testing.T) {
	for _, tt := range []struct {
		msg           string
		inFiles       []string
		outFile       string
		selectedPages []string
		desc          string
		n             int
		isImg         bool
	}{
		// 4-Up a PDF
		{"TestNUpFromPDF",
			[]string{filepath.Join(inDir, "Acroforms2.pdf")},
			filepath.Join(outDir, "Acroforms2.pdf"),
			nil,
			"",
			4,
			false},
		// 9-Up an image
		{"TestNUpFromSingleImage",
			[]string{filepath.Join(resDir, "pdfchip3.png")},
			filepath.Join(outDir, "out.pdf"),
			nil,
			"f:A3L",
			9,
			true},
		// 6-Up a sequence of images.
		{"TestNUpFromImages",
			[]string{
				filepath.Join(resDir, "pdfchip3.png"),
				filepath.Join(resDir, "demo.png"),
				filepath.Join(resDir, "snow.jpg"),
			},
			filepath.Join(outDir, "out1.pdf"),
			nil,
			"f:Tabloid, b:off, m:0",
			6,
			true},
	} {
		testNUp(t, tt.msg, tt.inFiles, tt.outFile, tt.selectedPages, tt.desc, tt.n, tt.isImg)
	}
}

func testGrid(t *testing.T, msg string, inFiles []string, outFile string, selectedPages []string, desc string, rows, cols int, isImg bool) {
	t.Helper()

	var (
		nup *pdf.NUp
		err error
	)

	if isImg {
		if nup, err = pdf.ImageGridConfig(rows, cols, desc); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	} else {
		if nup, err = pdf.PDFGridConfig(rows, cols, desc); err != nil {
			t.Fatalf("%s %s: %v\n", msg, outFile, err)
		}
	}

	if err := NUpFile(inFiles, outFile, selectedPages, nup, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestGrid(t *testing.T) {
	for _, tt := range []struct {
		msg           string
		inFiles       []string
		outFile       string
		selectedPages []string
		desc          string
		rows, cols    int
		isImg         bool
	}{
		{"TestGridFromPDF",
			[]string{filepath.Join(inDir, "Acroforms2.pdf")},
			filepath.Join(outDir, "testGridFromPDF.pdf"),
			nil, "f:LegalL", 1, 3, false},

		{"TestGridFromImages",
			[]string{
				filepath.Join(resDir, "pdfchip3.png"),
				filepath.Join(resDir, "demo.png"),
				filepath.Join(resDir, "snow.jpg"),
			},
			filepath.Join(outDir, "testGridFromImages.pdf"),
			nil, "d:500 500, m:20, b:off", 1, 3, true},
	} {
		testGrid(t, tt.msg, tt.inFiles, tt.outFile, tt.selectedPages, tt.desc, tt.rows, tt.cols, tt.isImg)
	}
}

func confForAlgorithm(aes bool, keyLength int, upw, opw string) *pdf.Configuration {
	if aes {
		return pdf.NewAESConfiguration(upw, opw, keyLength)
	}
	return pdf.NewRC4Configuration(upw, opw, keyLength)
}

func ensureFullAccess(t *testing.T, listPermOutput []string) {
	t.Helper()
	if len(listPermOutput) == 0 || listPermOutput[0] != "Full access" {
		t.Fail()
	}
}

func ensurePermissionsNone(t *testing.T, listPermOutput []string) {
	t.Helper()
	if len(listPermOutput) == 0 || !strings.HasPrefix(listPermOutput[0], "permission bits:            0") {
		t.Fail()
	}
}

func ensurePermissionsAll(t *testing.T, listPermOutput []string) {
	t.Helper()
	if len(listPermOutput) == 0 || listPermOutput[0] != "permission bits: 111100111100" {
		t.Fail()
	}
}

func testEncryption(t *testing.T, fileName string, alg string, keyLength int) {
	msg := "testEncryption"

	aes := alg == "aes"
	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, "test.pdf")
	t.Log(inFile)

	// List permissions of unencrypted file.
	list, err := ListPermissionsFile(inFile, nil)
	if err != nil {
		t.Fatalf("%s: list permissions %s: %v\n", msg, inFile, err)
	}
	ensureFullAccess(t, list)

	// Encrypt file.
	conf := confForAlgorithm(aes, keyLength, "upw", "opw")
	if err := EncryptFile(inFile, outFile, conf); err != nil {
		t.Fatalf("%s: encrypt %s: %v\n", msg, outFile, err)
	}

	// List permissions of encrypted file w/o passwords should fail.
	if list, err = ListPermissionsFile(outFile, nil); err == nil {
		t.Fatalf("%s: list permissions w/o pw %s: %v\n", msg, outFile, list)
	}

	// List permissions of encrypted file using the user password.
	conf = confForAlgorithm(aes, keyLength, "upw", "")
	if list, err = ListPermissionsFile(outFile, conf); err != nil {
		t.Fatalf("%s: list permissions %s: %v\n", msg, outFile, err)
	}
	ensurePermissionsNone(t, list)

	// List permissions of encrypted file using the owner password.
	conf = confForAlgorithm(aes, keyLength, "", "opw")
	if list, err = ListPermissionsFile(outFile, conf); err != nil {
		t.Fatalf("%s: list permissions %s: %v\n", msg, outFile, err)
	}
	ensurePermissionsNone(t, list)

	// Set all permissions of encrypted file w/o passwords should fail.
	conf = confForAlgorithm(aes, keyLength, "", "")
	conf.Permissions = pdfcpu.PermissionsAll
	if err = SetPermissionsFile(outFile, "", conf); err == nil {
		t.Fatalf("%s: set all permissions w/o pw for %s\n", msg, outFile)
	}

	// Set all permissions of encrypted file with user password should fail.
	conf = confForAlgorithm(aes, keyLength, "upw", "")
	conf.Permissions = pdfcpu.PermissionsAll
	if err = SetPermissionsFile(outFile, "", conf); err == nil {
		t.Fatalf("%s: set all permissions w/o opw for %s\n", msg, outFile)
	}

	// Set all permissions of encrypted file with owner password should fail.
	conf = confForAlgorithm(aes, keyLength, "", "opw")
	conf.Permissions = pdfcpu.PermissionsAll
	if err = SetPermissionsFile(outFile, "", conf); err == nil {
		t.Fatalf("%s: set all permissions w/o both pws for %s\n", msg, outFile)
	}

	// Set all permissions of encrypted file using both passwords.
	conf = confForAlgorithm(aes, keyLength, "upw", "opw")
	conf.Permissions = pdfcpu.PermissionsAll
	if err = SetPermissionsFile(outFile, "", conf); err != nil {
		t.Fatalf("%s: set all permissions for %s: %v\n", msg, outFile, err)
	}

	// List permissions using the owner password.
	conf = confForAlgorithm(aes, keyLength, "", "opw")
	if list, err = ListPermissionsFile(outFile, conf); err != nil {
		t.Fatalf("%s: list permissions for %s: %v\n", msg, outFile, err)
	}
	ensurePermissionsAll(t, list)

	// Change user password.
	conf = confForAlgorithm(aes, keyLength, "upw", "opw")
	if err = ChangeUserPasswordFile(outFile, "", "upw", "upwNew", conf); err != nil {
		t.Fatalf("%s: change upw %s: %v\n", msg, outFile, err)
	}

	// Change owner password.
	conf = confForAlgorithm(aes, keyLength, "upwNew", "opw")
	if err = ChangeOwnerPasswordFile(outFile, "", "opw", "opwNew", conf); err != nil {
		t.Fatalf("%s: change opw %s: %v\n", msg, outFile, err)
	}

	// Decrypt file using both passwords.
	conf = confForAlgorithm(aes, keyLength, "upwNew", "opwNew")
	if err = DecryptFile(outFile, "", conf); err != nil {
		t.Fatalf("%s: decrypt %s: %v\n", msg, outFile, err)
	}

	// Validate decrypted file.
	if err = ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: validate %s: %v\n", msg, outFile, err)
	}
}

func TestEncryption(t *testing.T) {
	for _, fileName := range []string{
		"5116.DCT_Filter.pdf",
		"networkProgr.pdf",
	} {
		testEncryption(t, fileName, "rc4", 40)
		testEncryption(t, fileName, "rc4", 128)
		testEncryption(t, fileName, "aes", 40)
		testEncryption(t, fileName, "aes", 128)
		testEncryption(t, fileName, "aes", 256)
	}
}

func TestExtractImagesCommand(t *testing.T) {
	msg := "TestExtractImagesCommand"

	// Extract images for all pages into outDir.
	for _, fn := range []string{"5116.DCT_Filter.pdf", "testImage.pdf", "go.pdf"} {
		fn = filepath.Join(inDir, fn)
		if err := ExtractImagesFile(fn, outDir, nil, nil); err != nil {
			t.Fatalf("%s %s: %v\n", msg, fn, err)
		}
	}

	// Extract images for inFile starting with page 1 into outDir.
	inFile := filepath.Join(inDir, "testImage.pdf")
	if err := ExtractImagesFile(inFile, outDir, []string{"1-"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractFontsCommand(t *testing.T) {
	msg := "TestExtractFontsCommand"

	// Extract fonts for all pages into outDir.
	for _, fn := range []string{"5116.DCT_Filter.pdf", "testImage.pdf", "go.pdf"} {
		fn = filepath.Join(inDir, fn)
		if err := ExtractFontsFile(fn, outDir, nil, nil); err != nil {
			t.Fatalf("%s %s: %v\n", msg, fn, err)
		}
	}

	// Extract fonts for inFile for pages 1-3 into outDir.
	inFile := filepath.Join(inDir, "go.pdf")
	if err := ExtractFontsFile(inFile, outDir, []string{"1-3"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractContentCommand(t *testing.T) {
	msg := "TestExtractContentCommand"

	// Extract content of all pages into outDir.
	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")
	if err := ExtractContentFile(inFile, outDir, nil, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractPagesCommand(t *testing.T) {
	msg := "TestExtractPagesCommand"

	// Extract page #1 into outDir.
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	if err := ExtractPagesFile(inFile, outDir, []string{"1"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractMetadataCommand(t *testing.T) {
	msg := "TestExtractMetadataCommand"

	// Extract metadata into outDir.
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	if err := ExtractMetadataFile(inFile, outDir, nil, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}
