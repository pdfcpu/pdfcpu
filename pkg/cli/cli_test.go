/*
Copyright 2019 The pdfcpu Authors.

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

package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	PDFCPULog "github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

var inDir, outDir, resDir, fontDir string

func TestMain(m *testing.M) {
	inDir = "../testdata"
	resDir = filepath.Join(inDir, "resources")
	fontDir = filepath.Join(inDir, "fonts")
	var err error

	if outDir, err = ioutil.TempDir("", "pdfcpu_cli_tests"); err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
	//fmt.Printf("outDir = %s\n", outDir)

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

func isPDF(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".pdf")
}

func allPDFs(t *testing.T, dir string) []string {
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

func validateFile(t *testing.T, fileName string, conf *pdf.Configuration) error {
	t.Helper()
	cmd := ValidateCommand(fileName, conf)
	_, err := Process(cmd)
	return err
}

func optimizeFile(t *testing.T, fileName string, conf *pdf.Configuration) error {
	t.Helper()
	cmd := OptimizeCommand(fileName, "", conf)
	_, err := Process(cmd)
	return err
}

func TestValidate(t *testing.T) {
	msg := "TestValidateCommand"
	for _, f := range allPDFs(t, inDir) {
		inFile := filepath.Join(inDir, f)
		if err := validateFile(t, inFile, nil); err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
	}
}

func testOptimizeFile(t *testing.T, inFile, outFile string) {
	t.Helper()
	msg := "testOptimizeFile"

	// Optimize inFile and write result to outFile.
	cmd := OptimizeCommand(inFile, outFile, nil)
	if _, err := Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}

	// Optimize outFile and write result to outFile.
	cmd = OptimizeCommand(outFile, "", nil)
	if _, err := Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Optimize outFile and write result to outFile.
	// Also skip validation.
	c := pdf.NewDefaultConfiguration()
	c.ValidationMode = pdf.ValidationNone
	cmd = OptimizeCommand(outFile, outFile, c)
	if _, err := Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestOptimize(t *testing.T) {
	for _, f := range allPDFs(t, inDir) {
		inFile := filepath.Join(inDir, f)
		outFile := filepath.Join(outDir, f)
		testOptimizeFile(t, inFile, outFile)
	}
}

// Split a test PDF file up into single page PDFs (using a split span of 1).
func TestSplitCommand(t *testing.T) {
	msg := "TestSplitCommand"
	fileName := "Acroforms2.pdf"
	inFile := filepath.Join(inDir, fileName)
	span := 1

	// Skip validation to boost processing.
	conf := pdf.NewDefaultConfiguration()
	conf.ValidationMode = pdf.ValidationNone

	cmd := SplitCommand(inFile, outDir, span, conf)
	if _, err := Process(cmd); err != nil {
		t.Fatalf("%s span=%d %s: %v\n", msg, span, inFile, err)
	}
}

// Split a test PDF file up into PDFs with 3 pages each (using a split span of 3).
func TestSplitSpanCommand(t *testing.T) {
	msg := "TestSplitCommand"
	fileName := "CenterOfWhy.pdf"
	inFile := filepath.Join(inDir, fileName)
	span := 3

	cmd := SplitCommand(inFile, outDir, span, nil)
	if _, err := Process(cmd); err != nil {
		t.Fatalf("%s span=%d %s: %v\n", msg, span, inFile, err)
	}
}

// Trim test PDF file so that only the first two pages are rendered.
func TestTrim(t *testing.T) {
	msg := "TestTrim"
	inFile := filepath.Join(inDir, "pike-stanford.pdf")
	outFile := filepath.Join(outDir, "test.pdf")

	if _, err := Process(TrimCommand(inFile, outFile, []string{"-2"}, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	if err := validateFile(t, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

// Rotate first 2 pages clockwise by 90 degrees.
func TestRotate(t *testing.T) {
	msg := "TestRotate"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "test.pdf")
	rotation := 90

	if _, err := Process(RotateCommand(inFile, outFile, rotation, []string{"-2"}, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	if err := validateFile(t, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func testAddWatermarks(t *testing.T, msg, inFile, outFile string, selectedPages []string, mode, modeParm, desc string, onTop bool) {
	t.Helper()
	inFile = filepath.Join(inDir, inFile)
	outFile = filepath.Join(outDir, outFile)

	var (
		wm  *pdfcpu.Watermark
		err error
	)
	switch mode {
	case "text":
		wm, err = pdf.ParseTextWatermarkDetails(modeParm, desc, onTop)
	case "image":
		wm, err = pdf.ParseImageWatermarkDetails(modeParm, desc, onTop)
	case "pdf":
		wm, err = pdf.ParsePDFWatermarkDetails(modeParm, desc, onTop)
	}
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	if _, err := Process(AddWatermarksCommand(inFile, outFile, selectedPages, wm, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := validateFile(t, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestAddWatermarks(t *testing.T) {
	for _, tt := range []struct {
		msg             string
		inFile, outFile string
		selectedPages   []string
		onTop           bool
		mode            string
		modeParm        string
		wmConf          string
	}{
		// Add text watermark to all pages of inFile starting at page 1 using a rotation angle of 20 degrees.
		{"TestWatermarkText",
			"Acroforms2.pdf",
			"testwm.pdf",
			[]string{"1-"},
			false,
			"text",
			"Draft",
			"s:0.7, rot:20"},

		// Add a greenish, slightly transparent stroked and filled text stamp to all odd pages of inFile other than page 1
		// using the default rotation which is aligned along the first diagonal running from lower left to upper right corner.
		{"TestStampText",
			"pike-stanford.pdf",
			"testStampText1.pdf",
			[]string{"odd", "!1"},
			true,
			"text",
			"Demo",
			"font:Courier, c: 0 .8 0, op:0.8, m:2"},

		// Add a red filled text stamp to all odd pages of inFile other than page 1 using a font size of 48 points
		// and the default rotation which is aligned along the first diagonal running from lower left to upper right corner.
		{"TestStampTextUsingFontsize",
			"pike-stanford.pdf",
			"testStampText2.pdf",
			[]string{"odd", "!1"},
			true,
			"text",
			"Demo",
			"font:Courier, c: 1 0 0, op:1, s:1 abs, points:48"},

		// Add image watermark to inFile starting at page 1 using no rotation.
		{"TestWatermarkImage",
			"Acroforms2.pdf", "testWMImageRel.pdf",
			[]string{"1-"},
			false,
			"image",
			filepath.Join(resDir, "pdfchip3.png"),
			"rot:0"},

		// Add image stamp to inFile using absolute scaling and a negative rotation of 90 degrees.
		{"TestStampImageAbsScaling",
			"Acroforms2.pdf",
			"testWMImageAbs.pdf",
			[]string{"1-"},
			true,
			"image",
			filepath.Join(resDir, "pdfchip3.png"),
			"s:.5 a, rot:-90"},

		// Add a PDF stamp to all pages of inFile using the 3rd page of pdfFile
		// and rotate along the 2nd diagonal running from upper left to lower right corner.
		{"TestWatermarkText",
			"Acroforms2.pdf",
			"testStampPDF.pdf",
			nil,
			true,
			"pdf",
			filepath.Join(inDir, "Wonderwall.pdf:3"),
			"d:2"},

		// Add a PDF multistamp to all pages of inFile
		// and rotate along the 2nd diagonal running from upper left to lower right corner.
		{"TestWatermarkText",
			"Acroforms2.pdf",
			"testMultistampPDF.pdf",
			nil,
			true,
			"pdf",
			filepath.Join(inDir, "Wonderwall.pdf"),
			"d:2"},
	} {
		testAddWatermarks(t, tt.msg, tt.inFile, tt.outFile, tt.selectedPages, tt.mode, tt.modeParm, tt.wmConf, tt.onTop)
	}
}

func TestStampingLifecyle(t *testing.T) {
	msg := "TestStampingLifecyle"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "stampLC.pdf")
	onTop := true // we are testing stamps

	// Stamp all pages.
	wm, err := pdf.ParseTextWatermarkDetails("Demo", "", onTop)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if _, err := Process(AddWatermarksCommand(inFile, outFile, nil, wm, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// // Update stamp on page 1.
	wm, err = pdf.ParseTextWatermarkDetails("Confidential", "", onTop)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	wm.Update = true
	if _, err := Process(AddWatermarksCommand(outFile, "", []string{"1"}, wm, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Add another stamp on top for all pages.
	// This is a redish transparent footer.
	wm, err = pdf.ParseTextWatermarkDetails("Footer", "pos:bc, c:0.8 0 0, op:.6, rot:0", onTop)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if _, err := Process(AddWatermarksCommand(outFile, "", nil, wm, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Remove stamp on page 1.
	if _, err := Process(RemoveWatermarksCommand(outFile, "", []string{"1"}, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Remove all stamps.
	if _, err := Process(RemoveWatermarksCommand(outFile, "", nil, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Validate the result.
	if err := validateFile(t, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
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

	if _, err := Process(NUpCommand(inFiles, outFile, selectedPages, nup, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := validateFile(t, outFile, nil); err != nil {
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
		{"TestNUpFromPDF",
			[]string{filepath.Join(inDir, "Acroforms2.pdf")},
			filepath.Join(outDir, "Acroforms2.pdf"),
			nil,
			"",
			4,
			false},

		{"TestNUpFromSingleImage",
			[]string{filepath.Join(resDir, "pdfchip3.png")},
			filepath.Join(outDir, "out.pdf"),
			nil,
			"f:A3L",
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

	if _, err := Process(NUpCommand(inFiles, outFile, selectedPages, nup, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := validateFile(t, outFile, nil); err != nil {
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
	if _, err := Process(ImportImagesCommand(imgFiles, outFile, imp, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := validateFile(t, outFile, nil); err != nil {
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

func TestGetPageCount(t *testing.T) {
	msg := "TestInsertRemovePages"
	inFile := filepath.Join(inDir, "CenterOfWhy.pdf")

	n, err := api.PageCountFile(inFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
	if n != 25 {
		t.Fatalf("%s %s: pageCount want:%d got:%d\n", msg, inFile, 25, n)
	}
}

func TestInsertRemovePages(t *testing.T) {
	msg := "TestInsertRemovePages"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "test.pdf")

	n1, err := api.PageCountFile(inFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}

	// Insert an empty page before pages 1 and 2.
	if _, err := Process(InsertPagesCommand(inFile, outFile, []string{"-2"}, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := validateFile(t, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	n2, err := api.PageCountFile(outFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
	if n2 != n1+2 {
		t.Fatalf("%s %s: pageCount want:%d got:%d\n", msg, inFile, n1+2, n2)
	}

	// Remove pages 1 and 2.
	if _, err := Process(RemovePagesCommand(outFile, "", []string{"-2"}, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := validateFile(t, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	n2, err = api.PageCountFile(outFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
	if n1 != n2 {
		t.Fatalf("%s %s: pageCount want:%d got:%d\n", msg, inFile, n1, n2)
	}
}

// Merge all PDFs in testdir into out/test.pdf.
func TestMergeCommand(t *testing.T) {
	msg := "TestMergeCommand"

	inFiles := []string(nil)
	for _, f := range allPDFs(t, inDir) {
		inFile := filepath.Join(inDir, f)
		inFiles = append(inFiles, inFile)
	}

	outFile := filepath.Join(outDir, "test.pdf")
	if _, err := Process(MergeCommand(inFiles, outFile, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	if _, err := Process(ValidateCommand(outFile, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestExtractImagesCommand(t *testing.T) {
	msg := "TestExtractImagesCommand"

	// Extract all images for each PDF file into outDir.
	cmd := ExtractImagesCommand("", outDir, nil, nil)
	for _, f := range allPDFs(t, inDir) {
		inFile := filepath.Join(inDir, f)
		cmd.InFile = &inFile
		// Extract all images.
		if _, err := Process(cmd); err != nil {
			t.Fatalf("%s %s: %v\n", msg, inFile, err)
		}
	}

	// Extract all images for inFile starting with page 1 into outDir.
	inFile := filepath.Join(inDir, "testImage.pdf")
	if _, err := Process(ExtractImagesCommand(inFile, outDir, []string{"1-"}, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractFontsCommand(t *testing.T) {
	msg := "TestExtractFontsCommand"

	// Extract fonts for all pages into outDir.
	cmd := ExtractFontsCommand("", outDir, nil, nil)
	for _, fn := range []string{"5116.DCT_Filter.pdf", "testImage.pdf", "go.pdf"} {
		fn = filepath.Join(inDir, fn)
		cmd.InFile = &fn
		if _, err := Process(cmd); err != nil {
			t.Fatalf("%s %s: %v\n", msg, fn, err)
		}
	}

	// Extract fonts for pages 1-3 into outDir.
	inFile := filepath.Join(inDir, "go.pdf")
	if _, err := Process(ExtractFontsCommand(inFile, outDir, []string{"1-3"}, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractContentCommand(t *testing.T) {
	msg := "TestExtractContentCommand"

	// Extract content of all pages into outDir.
	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")
	if _, err := Process(ExtractContentCommand(inFile, outDir, nil, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractPagesCommand(t *testing.T) {
	msg := "TestExtractPagesCommand"

	// Extract page #1 into outDir.
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	if _, err := Process(ExtractPagesCommand(inFile, outDir, []string{"1"}, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractMetadataCommand(t *testing.T) {
	msg := "TestExtractMetadataCommand"

	// Extract metadata into outDir.
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	if _, err := Process(ExtractMetadataCommand(inFile, outDir, nil)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestUnknownCommand(t *testing.T) {
	msg := "TestUnknownCommand"
	conf := pdf.NewDefaultConfiguration()
	inFile := filepath.Join(outDir, "go.pdf")

	cmd := &Command{
		Mode:   99,
		InFile: &inFile,
		Conf:   conf}

	if _, err := Process(cmd); err == nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestInstallFontsCommand(t *testing.T) {
	msg := "TestInstallFontsCommand"
	userFontName := filepath.Join(fontDir, "Geneva.ttf")
	_, err := Process(InstallFontsCommand([]string{userFontName}, nil))
	if err != nil {
		t.Fatalf("%s install fonts: %v\n", msg, err)
	}
}

func TestListFontsCommand(t *testing.T) {
	msg := "TestListFontsCommand"
	_, err := Process(ListFontsCommand(nil))
	if err != nil {
		t.Fatalf("%s list fonts: %v\n", msg, err)
	}
}

// Enable this test for debugging of a specific file.
func XTestSomeCommand(t *testing.T) {
	msg := "TestSomeCommand"

	PDFCPULog.SetDefaultTraceLogger()
	//PDFCPULog.SetDefaultParseLogger()
	PDFCPULog.SetDefaultReadLogger()
	PDFCPULog.SetDefaultValidateLogger()
	PDFCPULog.SetDefaultOptimizeLogger()
	PDFCPULog.SetDefaultWriteLogger()

	conf := pdf.NewDefaultConfiguration()
	inFile := filepath.Join(inDir, "test.pdf")

	if _, err := Process(ValidateCommand(inFile, conf)); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}
