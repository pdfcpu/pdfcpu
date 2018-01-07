package pdfcpu

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/hhrutter/pdfcpu/types"
)

const outputDir = "testdata/out"

func ExampleProcess_validate() {

	config := types.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	// Set relaxed validation mode.
	config.SetValidationRelaxed()

	cmd := ValidateCommand("in.pdf", config)

	_, err := Process(&cmd)
	if err != nil {
		return
	}

}

func ExampleProcess_optimize() {

	config := types.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	// Generate optional stats.
	config.StatsFileName = "stats.csv"

	// Configure end of line sequence for writing.
	config.Eol = types.EolLF

	cmd := OptimizeCommand("in.pdf", "out.pdf", config)

	_, err := Process(&cmd)
	if err != nil {
		return
	}

}

func ExampleProcess_merge() {

	// Concatenate this sequence of PDF files:
	filenamesIn := []string{"in1.pdf", "in2.pdf", "in3.pdf"}

	cmd := MergeCommand(filenamesIn, "out.pdf", types.NewDefaultConfiguration())

	_, err := Process(&cmd)
	if err != nil {
		return
	}

}
func ExampleProcess_split() {

	config := types.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	// Split into single-page PDFs.
	cmd := SplitCommand("in.pdf", "outDir", config)

	_, err := Process(&cmd)
	if err != nil {
		return
	}

}

func ExampleProcess_trim() {

	// Trim to first three pages.
	selectedPages := []string{"-3"}

	config := types.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	cmd := TrimCommand("in.pdf", "out.pdf", selectedPages, config)

	_, err := Process(&cmd)
	if err != nil {
		return
	}

}

func ExampleProcess_extractPages() {

	// Extract single-page PDFs for pages 3, 4 and 5.
	selectedPages := []string{"3..5"}

	config := types.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	cmd := ExtractPagesCommand("in.pdf", "dirOut", selectedPages, config)

	_, err := Process(&cmd)
	if err != nil {
		return
	}

}

func ExampleProcess_extractImages() {

	// Extract all embedded images for first 5 and last 5 pages but not for page 4.
	selectedPages := []string{"-5", "5-", "!4"}

	config := types.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	cmd := ExtractImagesCommand("in.pdf", "dirOut", selectedPages, config)

	_, err := Process(&cmd)
	if err != nil {
		return
	}

}

func ExampleProcess_listAttachments() {

	config := types.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = opw"

	cmd := ListAttachmentsCommand("in.pdf", config)

	list, err := Process(&cmd)
	if err != nil {
		return
	}

	// Print attachment list.
	for _, l := range list {
		fmt.Println(l)
	}

}

func ExampleProcess_addAttachments() {

	config := types.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	cmd := AddAttachmentsCommand("in.pdf", []string{"a.csv", "b.jpg", "c.pdf"}, config)

	_, err := Process(&cmd)
	if err != nil {
		return
	}
}

func ExampleProcess_removeAttachments() {

	config := types.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	// Not to be confused with the ExtractAttachmentsCommand!

	// Remove all attachments.
	cmd := RemoveAttachmentsCommand("in.pdf", nil, config)
	_, err := Process(&cmd)
	if err != nil {
		return
	}

	// Remove specific attachments.
	cmd = RemoveAttachmentsCommand("in.pdf", []string{"a.csv", "b.jpg"}, config)
	_, err = Process(&cmd)
	if err != nil {
		return
	}

}

func ExampleProcess_extractAttachments() {

	config := types.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	// Extract all attachments.
	cmd := ExtractAttachmentsCommand("in.pdf", "dirOut", nil, config)
	_, err := Process(&cmd)
	if err != nil {
		return
	}

	// Extract specific attachments.
	cmd = ExtractAttachmentsCommand("in.pdf", "dirOut", []string{"a.csv", "b.pdf"}, config)
	_, err = Process(&cmd)
	if err != nil {
		return
	}
}

func ExampleProcess_encrypt() {

	config := types.NewDefaultConfiguration()

	config.UserPW = "upw"
	config.OwnerPW = "opw"

	cmd := EncryptCommand("in.pdf", "out.pdf", config)

	_, err := Process(&cmd)
	if err != nil {
		return
	}
}

func ExampleProcess_decrypt() {

	config := types.NewDefaultConfiguration()

	config.UserPW = "upw"
	config.OwnerPW = "opw"

	cmd := DecryptCommand("in.pdf", "out.pdf", config)

	_, err := Process(&cmd)
	if err != nil {
		return
	}
}

func ExampleProcess_changeUserPW() {

	config := types.NewDefaultConfiguration()

	// supply existing owner pw like so
	config.OwnerPW = "opw"

	pwOld := "pwOld"
	pwNew := "pwNew"

	cmd := ChangeUserPWCommand("in.pdf", "out.pdf", config, &pwOld, &pwNew)

	_, err := Process(&cmd)
	if err != nil {
		return
	}
}

func ExampleProcess_changeOwnerPW() {

	config := types.NewDefaultConfiguration()

	// supply existing user pw like so
	config.UserPW = "upw"

	// old and new owner pw
	pwOld := "pwOld"
	pwNew := "pwNew"

	cmd := ChangeOwnerPWCommand("in.pdf", "out.pdf", config, &pwOld, &pwNew)

	_, err := Process(&cmd)
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

	config := types.NewDefaultConfiguration()
	config.SetValidationRelaxed()

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			cmd := ValidateCommand("testdata/"+file.Name(), config)
			_, err = Process(&cmd)
			if err != nil {
				t.Fatalf("TestValidateCommand: %v\n", err)
			}
		}
	}

}

func TestValidateOneFile(t *testing.T) {

	config := types.NewDefaultConfiguration()
	config.SetValidationRelaxed()

	cmd := ValidateCommand("testdata/gobook.0.pdf", config)
	_, err := Process(&cmd)
	if err != nil {
		t.Fatalf("TestValidateOneFile: %v\n", err)
	}

}

func BenchmarkValidateCommand(b *testing.B) {

	config := types.NewDefaultConfiguration()
	config.SetValidationRelaxed()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cmd := ValidateCommand("testdata/gobook.0.pdf", config)
		_, err := Process(&cmd)
		if err != nil {
			b.Fatalf("BenchmarkValidateCommand: %v\n", err)
		}
	}
}

// Optimize all PDFs in testdata and write with (default) end of line sequence "\n".
func TestOptimizeCommandWithLF(t *testing.T) {

	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatalf("TestOptimizeCommmand: %v\n", err)
	}

	config := types.NewDefaultConfiguration()

	// this is not necessary but to make it clearer.
	config.Eol = types.EolLF

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			cmd := OptimizeCommand("testdata/"+file.Name(), outputDir+"/test.pdf", config)
			_, err = Process(&cmd)
			if err != nil {
				t.Fatalf("TestOptimizeCommand: %v\n", err)
			}
		}
	}

}

// Optimize all PDFs in testdata and write with end of line sequence "\r".
func TestOptimizeCommandWithCR(t *testing.T) {

	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatalf("TestOptimizeCommmand: %v\n", err)
	}

	config := types.NewDefaultConfiguration()
	config.Eol = types.EolCR

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			cmd := OptimizeCommand("testdata/"+file.Name(), outputDir+"/test.pdf", config)
			_, err = Process(&cmd)
			if err != nil {
				t.Fatalf("TestOptimizeCommand: %v\n", err)
			}
		}
	}

}

// Optimize all PDFs in testdata and write with end of line sequence "\r".
// This test writes out the cross reference table the old way without using object streams and an xref stream.
func TestOptimizeCommandWithCRAndNoXrefStream(t *testing.T) {

	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatalf("TestOptimizeCommmand: %v\n", err)
	}

	config := types.NewDefaultConfiguration()
	config.Eol = types.EolCR
	config.WriteObjectStream = false
	config.WriteXRefStream = false

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			cmd := OptimizeCommand("testdata/"+file.Name(), outputDir+"/test.pdf", config)
			_, err = Process(&cmd)
			if err != nil {
				t.Fatalf("TestOptimizeCommand: %v\n", err)
			}
		}
	}

}

// Optimize all PDFs in testdata and write with end of line sequence "\r\n".
func TestOptimizeCommandWithCRLF(t *testing.T) {

	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatalf("TestOptimizeCommmand: %v\n", err)
	}

	config := types.NewDefaultConfiguration()
	config.Eol = types.EolCRLF
	config.StatsFileName = outputDir + "/testStats.csv"

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "pdf") {
			cmd := OptimizeCommand("testdata/"+file.Name(), outputDir+"/test.pdf", config)
			_, err = Process(&cmd)
			if err != nil {
				t.Fatalf("TestOptimizeCommand: %v\n", err)
			}
		}
	}

}

// Split a test PDF file up into single page PDFs.
func TestSplitCommand(t *testing.T) {

	cmd := SplitCommand("testdata/Acroforms2.pdf", outputDir, types.NewDefaultConfiguration())

	_, err := Process(&cmd)
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

	cmd := MergeCommand(inFiles, outputDir+"/test.pdf", types.NewDefaultConfiguration())
	_, err = Process(&cmd)
	if err != nil {
		t.Fatalf("TestMergeCommand: %v\n", err)
	}

}

// Trim test PDF file so that only the first two pages are rendered.
func TestTrimCommand(t *testing.T) {

	cmd := TrimCommand("testdata/pike-stanford.pdf", outputDir+"/test.pdf", []string{"-2"}, types.NewDefaultConfiguration())

	_, err := Process(&cmd)
	if err != nil {
		t.Fatalf("TestTrimCommand: %v\n", err)
	}

}

func TestExtractImagesCommand(t *testing.T) {

	cmd := ExtractImagesCommand("", outputDir, nil, types.NewDefaultConfiguration())
	var err error

	for _, fn := range []string{"go.pdf", "golang.pdf", "Wonderwall.pdf", "testImage.pdf"} {
		fn = "testdata/" + fn
		cmd.InFile = &fn
		_, err = Process(&cmd)
		if err != nil {
			t.Fatalf("TestExtractImageCommand: %v\n", err)
		}
	}

	cmd = ExtractImagesCommand("testdata/testImage.pdf", outputDir, []string{"1-"}, types.NewDefaultConfiguration())
	_, err = Process(&cmd)
	if err != nil {
		t.Fatalf("TestExtractImageCommand: %v\n", err)
	}

}

func TestExtractFontsCommand(t *testing.T) {

	cmd := ExtractFontsCommand("", outputDir, nil, types.NewDefaultConfiguration())
	var err error

	for _, fn := range []string{"5116.DCT_Filter.pdf", "testImage.pdf", "go.pdf"} {
		fn = "testdata/" + fn
		cmd.InFile = &fn
		_, err = Process(&cmd)
		if err != nil {
			t.Fatalf("TestExtractFontsCommand: %v\n", err)
		}
	}

	cmd = ExtractFontsCommand("testdata/go.pdf", outputDir, []string{"1-3"}, types.NewDefaultConfiguration())
	_, err = Process(&cmd)
	if err != nil {
		t.Fatalf("TestExtractFontsCommand: %v\n", err)
	}

}

func TestExtractContentCommand(t *testing.T) {

	cmd := ExtractContentCommand("testdata/5116.DCT_Filter.pdf", outputDir, nil, types.NewDefaultConfiguration())
	_, err := Process(&cmd)
	if err != nil {
		t.Fatalf("TestExtractContentCommand: %v\n", err)
	}

}

func TestExtractPagesCommand(t *testing.T) {

	cmd := ExtractPagesCommand("testdata/TheGoProgrammingLanguageCh1.pdf", outputDir, []string{"1"}, types.NewDefaultConfiguration())
	_, err := Process(&cmd)
	if err != nil {
		t.Fatalf("TestExtractPagesCommand: %v\n", err)
	}

}

func TestEncryptDecrypt(t *testing.T) {

	for _, fileName := range []string{"5116.DCT_Filter.pdf", "networkProgr.pdf"} {

		fin := "testdata/" + fileName
		fmt.Println("\n" + fin)
		f := outputDir + "/test.pdf"

		// Encrypt
		fmt.Println("\nEncrypt")
		config := types.NewDefaultConfiguration()
		config.UserPW = "upw"
		config.OwnerPW = "opw"
		if fileName == "networkProgr.pdf" {
			config.EncryptUsingAES = false
			config.EncryptUsing128BitKey = false
		}
		cmd := EncryptCommand(fin, f, config)
		_, err := Process(&cmd)
		if err != nil {
			t.Fatalf("TestEncryptDecrypt - encrypt %s: %v\n", f, err)
		}

		// Encrypt already encrypted
		fmt.Println("\nEncrypt already encrypted")
		config = types.NewDefaultConfiguration()
		config.UserPW = "upw"
		config.OwnerPW = "opw"
		cmd = EncryptCommand(f, f, config)
		_, err = Process(&cmd)
		if err == nil {
			t.Fatalf("TestEncryptDecrypt - encrypt encrypted %s\n", f)
		}

		// Validate using wrong owner pw
		fmt.Println("\nValidate wrong ownerPW")
		config = types.NewDefaultConfiguration()
		config.UserPW = "upw"
		config.OwnerPW = "opwWrong"
		cmd = ValidateCommand(f, config)
		_, err = Process(&cmd)
		if err != nil {
			t.Fatalf("TestEncryptDecrypt - validate %s using wrong ownerPW: %v\n", f, err)
		}

		// Optimize using wrong owner pw
		//fmt.Println("\nOptimize wrong ownerPW")
		config = types.NewDefaultConfiguration()
		config.UserPW = "upw"
		config.OwnerPW = "opwWrong"
		cmd = OptimizeCommand(f, f, config)
		_, err = Process(&cmd)
		if err != nil {
			t.Fatalf("TestEncryptDecrypt - optimize %s using wrong ownerPW: %v\n", f, err)
		}

		// Split using wrong owner pw
		//fmt.Println("\nSplit wrong ownerPW")
		config = types.NewDefaultConfiguration()
		config.UserPW = "upw"
		config.OwnerPW = "opwWrong"
		cmd = SplitCommand(f, outputDir, config)
		_, err = Process(&cmd)
		if err != nil {
			t.Fatalf("TestEncryptDecrypt - split %s using wrong ownerPW: %v\n", f, err)
		}

		// Validate
		//fmt.Println("\nValidate")
		config = types.NewDefaultConfiguration()
		config.UserPW = "upw"
		config.OwnerPW = "opw"
		cmd = ValidateCommand(f, config)
		_, err = Process(&cmd)
		if err != nil {
			t.Fatalf("TestEncryptDecrypt - validate %s: %v\n", f, err)
		}

		// ChangeUserPW using wrong userpw
		//fmt.Println("\nChangeUserPW wrong userpw")
		// config = types.NewDefaultConfiguration()
		// config.OwnerPW = "opw"
		// pwOld := "upwWrong"
		// pwNew := "upwNew"
		// cmd = ChangeUserPWCommand(f, f, config, &pwOld, &pwNew)
		// _, err = Process(&cmd)
		// if err == nil {
		// 	t.Fatalf("TestEncryption - change userPW using wrong userPW%s:\n", f)
		// }

		// ChangeUserPW
		//fmt.Println("\nChangeUserPW")
		config = types.NewDefaultConfiguration()
		config.OwnerPW = "opw"
		pwOld := "upw"
		pwNew := "upwNew"
		cmd = ChangeUserPWCommand(f, f, config, &pwOld, &pwNew)
		_, err = Process(&cmd)
		if err != nil {
			t.Fatalf("TestEncryption - change userPW %s: %v\n", f, err)
		}

		// ChangeOwnerPW
		//fmt.Println("\nChangeOwnerPW")
		config = types.NewDefaultConfiguration()
		config.UserPW = "upwNew"
		pwOld = "opw"
		pwNew = "opwNew"
		cmd = ChangeOwnerPWCommand(f, f, config, &pwOld, &pwNew)
		_, err = Process(&cmd)
		if err != nil {
			t.Fatalf("TestEncryption - change ownerPW %s: %v\n", f, err)
		}

		// Decrypt using wrong pw
		//fmt.Println("\nDecrypt using wrong pw")
		config = types.NewDefaultConfiguration()
		config.UserPW = "upwWrong"
		config.OwnerPW = "opwWrong"
		cmd = DecryptCommand(f, f, config)
		_, err = Process(&cmd)
		if err == nil {
			t.Fatalf("TestEncryptDecrypt - decrypt using wrong pw %s\n", f)
		}

		// Decrypt
		//fmt.Println("\nDecrypt")
		config = types.NewDefaultConfiguration()
		config.UserPW = "upwNew"
		config.OwnerPW = "opwNew"
		cmd = DecryptCommand(f, f, config)
		_, err = Process(&cmd)
		if err != nil {
			t.Fatalf("TestEncryptDecrypt - decrypt %s: %v\n", f, err)
		}

		// Validate
		//fmt.Println("\nValidate")
		config = types.NewDefaultConfiguration()
		cmd = ValidateCommand(f, config)
		_, err = Process(&cmd)
		if err != nil {
			t.Fatalf("TestEncryption - validate %s: %v\n", f, err)
		}

	}
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

func prepareForAttachmentTest(testDir string) (err error) {

	testFile := testDir + "/go.pdf"
	err = copyFile(testFile, outputDir+"/go.pdf")
	if err != nil {
		return
	}

	testFile = testDir + "/golang.pdf"
	err = copyFile(testFile, outputDir+"/golang.pdf")
	if err != nil {
		return
	}

	testFile = testDir + "/T4.pdf"
	err = copyFile(testFile, outputDir+"/T4.pdf")
	if err != nil {
		return
	}

	return
}
func TestAttachments(t *testing.T) {

	testDir := "testdata"

	err := prepareForAttachmentTest(testDir)
	if err != nil {
		t.Fatalf("prepare for attachments: %v\n", err)
	}

	config := types.NewDefaultConfiguration()

	fileName := outputDir + "/go.pdf"

	// attach list must be 0
	cmd := ListAttachmentsCommand(fileName, config)
	list, err := Process(&cmd)
	if err != nil {
		t.Fatalf("TestAttachments - list attachments %s: %v\n", fileName, err)
	}
	if len(list) > 0 {
		t.Fatalf("TestAttachments - list attachments %s: should have 0 attachments\n", fileName)
	}

	// attach add 2 files
	cmd = AddAttachmentsCommand(fileName, []string{outputDir + "/golang.pdf", outputDir + "/T4.pdf"}, config)
	_, err = Process(&cmd)
	if err != nil {
		t.Fatalf("TestAttachments - add attachments to %s: %v\n", fileName, err)
	}

	// attach list must be 2
	cmd = ListAttachmentsCommand(fileName, config)
	list, err = Process(&cmd)
	if err != nil {
		t.Fatalf("TestAttachments - list attachments %s: %v\n", fileName, err)
	}
	if len(list) != 2 {
		t.Fatalf("TestAttachments - list attachments %s: should have 0 attachments\n", fileName)
	}

	// attach extract all
	cmd = ExtractAttachmentsCommand(fileName, ".", nil, config)
	_, err = Process(&cmd)
	if err != nil {
		t.Fatalf("TestAttachments - extract all attachments from %s to %s: %v\n", fileName, ".", err)
	}

	// attach extract 1 file
	cmd = ExtractAttachmentsCommand(fileName, ".", []string{outputDir + "/golang.pdf"}, config)
	_, err = Process(&cmd)
	if err != nil {
		t.Fatalf("TestAttachments - extract 1 attachment from %s to %s: %v\n", fileName, ".", err)
	}

	// attach remove 1 file
	cmd = RemoveAttachmentsCommand(fileName, []string{outputDir + "/golang.pdf"}, config)
	_, err = Process(&cmd)
	if err != nil {
		t.Fatalf("TestAttachments - remove attachment from %s: %v\n", fileName, err)
	}

	// attach list must be 1
	cmd = ListAttachmentsCommand(fileName, config)
	list, err = Process(&cmd)
	if err != nil {
		t.Fatalf("TestAttachments - list attachments %s: %v\n", fileName, err)
	}
	if len(list) != 1 {
		t.Fatalf("TestAttachments - list attachments %s: should have 0 attachments\n", fileName)
	}

	// attach remove all
	cmd = RemoveAttachmentsCommand(fileName, nil, config)
	_, err = Process(&cmd)
	if err != nil {
		t.Fatalf("TestAttachments - remove all attachment from %s: %v\n", fileName, err)
	}

	// attach list must be 0.
	cmd = ListAttachmentsCommand(fileName, config)
	list, err = Process(&cmd)
	if err != nil {
		t.Fatalf("TestAttachments - list attachments %s: %v\n", fileName, err)
	}
	if len(list) > 0 {
		t.Fatalf("TestAttachments - list attachments %s: should have 0 attachments\n", fileName)
	}
}

func TestUnknownCommand(t *testing.T) {

	config := types.NewDefaultConfiguration()
	fileName := outputDir + "/go.pdf"

	cmd := &Command{
		Mode:   99,
		InFile: &fileName,
		Config: config}

	_, err := Process(cmd)
	if err == nil {
		t.Fatal("TestUnknowncommand - should have failed")
	}

}
