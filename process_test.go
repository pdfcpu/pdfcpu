package pdflib

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/hhrutter/pdflib/types"
)

const outputDir = "testdata/out"

func ExampleProcess_validate() {

	mode := "relaxed"

	cmd := ValidateCommand("in.pdf", mode)

	err := Process(&cmd)
	if err != nil {
		return
	}

}

func ExampleProcess_optimize() {

	// Generate optional stats.
	statsFile := "stats.csv"

	cmd := OptimizeCommand("in.pdf", "out.pdf", statsFile, types.EolLF)

	err := Process(&cmd)
	if err != nil {
		return
	}

}

func ExampleProcess_merge() {

	// Concatenate this sequence of PDF files:
	filenamesIn := []string{"in1.pdf", "in2.pdf", "in3.pdf"}

	cmd := MergeCommand(filenamesIn, "out.pdf")

	err := Process(&cmd)
	if err != nil {
		return
	}

}
func ExampleProcess_split() {

	// Split into single-page PDFs.
	cmd := SplitCommand("in.pdf", "outDir")

	err := Process(&cmd)
	if err != nil {
		return
	}

}

func ExampleProcess_trim() {

	// Trim to first three pages.
	selectedPages := []string{"-3"}

	cmd := TrimCommand("in.pdf", "out.pdf", selectedPages)

	err := Process(&cmd)
	if err != nil {
		return
	}

}

func ExampleProcess_extractPages() {

	// Extract single-page PDFs for pages 3, 4 and 5.
	selectedPages := []string{"3..5"}

	cmd := ExtractPagesCommand("in.pdf", "dirOut", selectedPages)

	err := Process(&cmd)
	if err != nil {
		return
	}

}

func ExampleProcess_extractImages() {

	// Extract all embedded images for first 5 and last 5 pages but not for page 4.
	selectedPages := []string{"-5", "5-", "!4"}

	cmd := ExtractImagesCommand("in.pdf", "dirOut", selectedPages)

	err := Process(&cmd)
	if err != nil {
		return
	}

}

func TestMain(m *testing.M) {
	os.Mkdir(outputDir, 0777)

	exitCode := m.Run()

	err := os.RemoveAll(outputDir)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

// Validate all PDFs in testdata.
func TestValidateCommand(t *testing.T) {

	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatalf("TestValidateCommand: %v\n", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			cmd := ValidateCommand("testdata/"+file.Name(), "relaxed")
			err = Process(&cmd)
			if err != nil {
				t.Fatalf("TestValidateCommand: %v\n", err)
			}
		}
	}

}

// Optimize all PDFs in testdata.
func TestOptimizeCommandWithLF(t *testing.T) {

	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatalf("TestOptimizeCommmand: %v\n", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			cmd := OptimizeCommand("testdata/"+file.Name(), outputDir+"/test.pdf", "", types.EolCR)
			err = Process(&cmd)
			if err != nil {
				t.Fatalf("TestOptimizeCommand: %v\n", err)
			}
		}
	}

}

func TestOptimizeCommandWithCRLF(t *testing.T) {

	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatalf("TestOptimizeCommmand: %v\n", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			cmd := OptimizeCommand("testdata/"+file.Name(), outputDir+"/test.pdf", "", types.EolCRLF)
			err = Process(&cmd)
			if err != nil {
				t.Fatalf("TestOptimizeCommand: %v\n", err)
			}
		}
	}

}

func TestOptimizeCommandWithCR(t *testing.T) {

	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatalf("TestOptimizeCommmand: %v\n", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			cmd := OptimizeCommand("testdata/"+file.Name(), outputDir+"/test.pdf", "", types.EolCR)
			err = Process(&cmd)
			if err != nil {
				t.Fatalf("TestOptimizeCommand: %v\n", err)
			}
		}
	}

}

// Split a test PDF file up into single page PDFs.
func TestSplitCommand(t *testing.T) {

	cmd := SplitCommand("testdata/Acroforms2.pdf", outputDir)

	err := Process(&cmd)
	if err != nil {
		t.Fatalf("TestSplitCommand: %v\n", err)
	}
}

// Merge all PDFs in testdir into out/test.pdf.
func TestMergeCommand(t *testing.T) {

	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatalf("TestMergeCommmand: %v\n", err)
	}

	inFiles := []string{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			inFiles = append(inFiles, "testdata/"+file.Name())
		}
	}

	cmd := MergeCommand(inFiles, outputDir+"/test.pdf")
	err = Process(&cmd)
	if err != nil {
		t.Fatalf("TestMergeCommand: %v\n", err)
	}

}

// Trim test PDF file so that only the first two pages are rendered.
func TestTrimCommand(t *testing.T) {

	cmd := TrimCommand("testdata/pike-stanford.pdf", outputDir+"/test.pdf", []string{"-2"})

	err := Process(&cmd)
	if err != nil {
		t.Fatalf("TestTrimCommand: %v\n", err)
	}

}

func TestExtractImagesCommand(t *testing.T) {

	cmd := ExtractImagesCommand("testdata/TheGoProgrammingLanguageCh1.pdf", outputDir, nil)
	err := Process(&cmd)
	if err != nil {
		t.Fatalf("TestExtractImageCommand: %v\n", err)
	}

}

func TestExtractFontsCommand(t *testing.T) {

	cmd := ExtractFontsCommand("testdata/TheGoProgrammingLanguageCh1.pdf", outputDir, nil)
	err := Process(&cmd)
	if err != nil {
		t.Fatalf("TestExtractFontsCommand: %v\n", err)
	}

}

func TestExtractContentCommand(t *testing.T) {

	cmd := ExtractContentCommand("testdata/TheGoProgrammingLanguageCh1.pdf", outputDir, nil)
	err := Process(&cmd)
	if err != nil {
		t.Fatalf("TestExtractContentCommand: %v\n", err)
	}

}

func TestExtractPagesCommand(t *testing.T) {

	cmd := ExtractPagesCommand("testdata/TheGoProgrammingLanguageCh1.pdf", outputDir, []string{"1"})
	err := Process(&cmd)
	if err != nil {
		t.Fatalf("TestExtractPagesCommand: %v\n", err)
	}

}
