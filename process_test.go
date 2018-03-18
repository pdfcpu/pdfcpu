package pdfcpu

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/hhrutter/pdfcpu/create"
	"github.com/hhrutter/pdfcpu/log"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/hhrutter/pdfcpu/validate"
)

const outputDir = "testdata/out"

func ExampleProcess_validate() {

	config := types.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	// Set relaxed validation mode.
	config.SetValidationRelaxed()

	_, err := Process(ValidateCommand("in.pdf", config))
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

	_, err := Process(OptimizeCommand("in.pdf", "out.pdf", config))
	if err != nil {
		return
	}

}

func ExampleProcess_merge() {

	// Concatenate this sequence of PDF files:
	filenamesIn := []string{"in1.pdf", "in2.pdf", "in3.pdf"}

	_, err := Process(MergeCommand(filenamesIn, "out.pdf", types.NewDefaultConfiguration()))
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

	_, err := Process(SplitCommand("in.pdf", "outDir", config))
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

	_, err := Process(TrimCommand("in.pdf", "out.pdf", selectedPages, config))
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

	_, err := Process(ExtractPagesCommand("in.pdf", "dirOut", selectedPages, config))
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

	_, err := Process(ExtractImagesCommand("in.pdf", "dirOut", selectedPages, config))
	if err != nil {
		return
	}

}

func ExampleProcess_listAttachments() {

	config := types.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = opw"

	list, err := Process(ListAttachmentsCommand("in.pdf", config))
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

	_, err := Process(AddAttachmentsCommand("in.pdf", []string{"a.csv", "b.jpg", "c.pdf"}, config))
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
	_, err := Process(RemoveAttachmentsCommand("in.pdf", nil, config))
	if err != nil {
		return
	}

	// Remove specific attachments.
	_, err = Process(RemoveAttachmentsCommand("in.pdf", []string{"a.csv", "b.jpg"}, config))
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
	_, err := Process(ExtractAttachmentsCommand("in.pdf", "dirOut", nil, config))
	if err != nil {
		return
	}

	// Extract specific attachments.
	_, err = Process(ExtractAttachmentsCommand("in.pdf", "dirOut", []string{"a.csv", "b.pdf"}, config))
	if err != nil {
		return
	}
}

func ExampleProcess_encrypt() {

	config := types.NewDefaultConfiguration()

	config.UserPW = "upw"
	config.OwnerPW = "opw"

	_, err := Process(EncryptCommand("in.pdf", "out.pdf", config))
	if err != nil {
		return
	}
}

func ExampleProcess_decrypt() {

	config := types.NewDefaultConfiguration()

	config.UserPW = "upw"
	config.OwnerPW = "opw"

	_, err := Process(DecryptCommand("in.pdf", "out.pdf", config))
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

	_, err := Process(ChangeUserPWCommand("in.pdf", "out.pdf", config, &pwOld, &pwNew))
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

	_, err := Process(ChangeOwnerPWCommand("in.pdf", "out.pdf", config, &pwOld, &pwNew))
	if err != nil {
		return
	}
}

func ExampleProcess_listPermissions() {

	config := types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"

	list, err := Process(ListPermissionsCommand("in.pdf", config))
	if err != nil {
		return
	}

	// Print permissions list.
	for _, l := range list {
		fmt.Println(l)
	}
}

func ExampleProcess_addPermissions() {

	config := types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"

	config.UserAccessPermissions = types.PermissionsAll

	_, err := Process(AddPermissionsCommand("in.pdf", config))
	if err != nil {
		return
	}

}

func TestMain(m *testing.M) {

	os.Mkdir(outputDir, 0777)

	log.SetDefaultLoggers()

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
			_, err = Process(ValidateCommand("testdata/"+file.Name(), config))
			if err != nil {
				t.Fatalf("TestValidateCommand: %v\n", err)
			}
		}
	}

}

func TestValidateOneFile(t *testing.T) {

	config := types.NewDefaultConfiguration()
	config.SetValidationRelaxed()

	_, err := Process(ValidateCommand("testdata/gobook.0.pdf", config))
	if err != nil {
		t.Fatalf("TestValidateOneFile: %v\n", err)
	}

}

func BenchmarkValidateCommand(b *testing.B) {

	config := types.NewDefaultConfiguration()
	config.SetValidationRelaxed()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := Process(ValidateCommand("testdata/gobook.0.pdf", config))
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
			_, err = Process(OptimizeCommand("testdata/"+file.Name(), outputDir+"/test.pdf", config))
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
			_, err = Process(OptimizeCommand("testdata/"+file.Name(), outputDir+"/test.pdf", config))
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
			_, err = Process(OptimizeCommand("testdata/"+file.Name(), outputDir+"/test.pdf", config))
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
			_, err = Process(OptimizeCommand("testdata/"+file.Name(), outputDir+"/test.pdf", config))
			if err != nil {
				t.Fatalf("TestOptimizeCommand: %v\n", err)
			}
		}
	}

}

// Split a test PDF file up into single page PDFs.
func TestSplitCommand(t *testing.T) {

	_, err := Process(SplitCommand("testdata/Acroforms2.pdf", outputDir, types.NewDefaultConfiguration()))
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

	_, err = Process(MergeCommand(inFiles, outputDir+"/test.pdf", types.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestMergeCommand: %v\n", err)
	}

}

// Trim test PDF file so that only the first two pages are rendered.
func TestTrimCommand(t *testing.T) {

	_, err := Process(TrimCommand("testdata/pike-stanford.pdf", outputDir+"/test.pdf", []string{"-2"}, types.NewDefaultConfiguration()))
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
		_, err = Process(cmd)
		if err != nil {
			t.Fatalf("TestExtractImageCommand: %v\n", err)
		}
	}

	_, err = Process(ExtractImagesCommand("testdata/testImage.pdf", outputDir, []string{"1-"}, types.NewDefaultConfiguration()))
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
		_, err = Process(cmd)
		if err != nil {
			t.Fatalf("TestExtractFontsCommand: %v\n", err)
		}
	}

	_, err = Process(ExtractFontsCommand("testdata/go.pdf", outputDir, []string{"1-3"}, types.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestExtractFontsCommand: %v\n", err)
	}

}

func TestExtractContentCommand(t *testing.T) {

	_, err := Process(ExtractContentCommand("testdata/5116.DCT_Filter.pdf", outputDir, nil, types.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestExtractContentCommand: %v\n", err)
	}

}

func TestExtractPagesCommand(t *testing.T) {

	_, err := Process(ExtractPagesCommand("testdata/TheGoProgrammingLanguageCh1.pdf", outputDir, []string{"1"}, types.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestExtractPagesCommand: %v\n", err)
	}

}

func TestEncryptUPWOnly(t *testing.T) {
	t.Log("running TestEncryptUPWOnly..")

	f := outputDir + "/test.pdf"

	// Encrypt upw only
	t.Log("Encrypt upw only")
	config := types.NewDefaultConfiguration()
	config.UserPW = "upw"
	_, err := Process(EncryptCommand("testdata/5116.DCT_Filter.pdf", f, config))
	if err != nil {
		t.Fatalf("TestEncryptUPWOnly - encrypt with upw only to %s: %v\n", f, err)
	}

	// Validate wrong upw
	t.Log("Validate wrong upw fails")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(ValidateCommand(f, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - validate %s using wrong upw should fail!\n", f)
	}

	// Validate wrong opw
	t.Log("Validate wrong opw fails")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(ValidateCommand(f, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - validate %s using wrong opw should fail!\n", f)
	}

	// Validate default opw=upw (if there is no ownerpw set)
	t.Log("Validate default opw")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "upw"
	_, err = Process(ValidateCommand(f, config))
	if err != nil {
		t.Fatalf("TestEncryptUPWOnly - validate %s using default opw: %s!\n", f, err)
	}

	// Validate upw
	t.Log("Validate upw")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	_, err = Process(ValidateCommand(f, config))
	if err != nil {
		t.Fatalf("TestEncryptUPWOnly - validate %s using upw: %v\n", f, err)
	}

	// Optimize wrong opw
	t.Log("Optimize wrong opw fails")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(OptimizeCommand(f, f, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - optimize %s using wrong opw should fail!\n", f)
	}

	// Optimize empty opw
	t.Log("Optimize empty opw fails")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = ""
	_, err = Process(OptimizeCommand(f, f, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - optimize %s using empty opw should fail!\n", f)
	}

	// Optimize wrong upw
	t.Log("Optimize wrong upw fails")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(OptimizeCommand(f, f, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - optimize %s using wrong upw should fail!\n", f)
	}

	// Optimize upw
	t.Log("Optimize upw")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	_, err = Process(OptimizeCommand(f, f, config))
	if err != nil {
		t.Fatalf("TestEncryptUPWOnly - optimize %s using upw: %v\n", f, err)
	}

	//Change upw wrong upwOld
	t.Log("ChangeUserPW wrong upwOld fails")
	config = types.NewDefaultConfiguration()
	pwOld := "upwWrong"
	pwNew := "upwNew"
	_, err = Process(ChangeUserPWCommand(f, f, config, &pwOld, &pwNew))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - %s change userPW using wrong upwOld should fail\n", f)
	}

	// Change upw
	t.Log("ChangeUserPW")
	config = types.NewDefaultConfiguration()
	pwOld = "upw"
	pwNew = "upwNew"
	_, err = Process(ChangeUserPWCommand(f, f, config, &pwOld, &pwNew))
	if err != nil {
		t.Fatalf("TestEncryptUPWOnly - %s change userPW: %v\n", f, err)
	}

	// Decrypt wrong opw
	t.Log("Decrypt wrong opw fails")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(DecryptCommand(f, f, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - %s decrypt using wrong opw should fail\n", f)
	}

	// Decrypt wrong upw
	t.Log("Decrypt wrong upw fails")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	_, err = Process(DecryptCommand(f, f, config))
	if err == nil {
		t.Fatalf("TestEncryptUPWOnly - %s decrypt using wrong upw should fail\n", f)
	}

	// Decrypt upw
	t.Log("Decrypt upw")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upwNew"
	_, err = Process(DecryptCommand(f, f, config))
	if err != nil {
		t.Fatalf("TestEncryptUPWOnly - %s decrypt using upw: %v\n", f, err)
	}

}

func TestEncryptOPWOnly(t *testing.T) {

	t.Log("running TestEncryptOPWOnly..")

	f := outputDir + "/test.pdf"

	// Encrypt opw only
	t.Log("Encrypt opw only")
	config := types.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	_, err := Process(EncryptCommand("testdata/5116.DCT_Filter.pdf", f, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - encrypt with opw only to %s: %v\n", f, err)
	}

	// Validate wrong opw succeeds with fallback to empty upw
	t.Log("Validate wrong opw succeeds with fallback to empty upw")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(ValidateCommand(f, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - validate %s using wrong opw succeeds falling back to empty upw!: %v\n", f, err)
	}

	// Validate opw
	t.Log("Validate opw")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	_, err = Process(ValidateCommand(f, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - validate %s using opw: %v\n", f, err)
	}

	// Validate wrong upw
	t.Log("Validate wrong upw fails")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(ValidateCommand(f, config))
	if err == nil {
		t.Fatalf("TestEncryptOPWOnly - validate %s using wrong upw should fail!\n", f)
	}

	// Validate no pw using empty upw
	t.Log("Validate no pw using empty upw")
	config = types.NewDefaultConfiguration()
	_, err = Process(ValidateCommand(f, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - validate %s no pw using empty upw: %v\n", f, err)
	}

	// Optimize wrong opw, succeeds with fallback to empty upw
	t.Log("Optimize wrong opw succeeds with fallback to empty upw")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(OptimizeCommand(f, f, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - optimize %s using wrong opw succeeds falling back to empty upw: %v\n", f, err)
	}

	// Optimize opw
	t.Log("Optimize opw")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	_, err = Process(OptimizeCommand(f, f, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - optimize %s using opw: %v\n", f, err)
	}

	// Optimize wrong upw
	t.Log("Optimize wrong upw fails")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(OptimizeCommand(f, f, config))
	if err == nil {
		t.Fatalf("TestEncryptOPWOnly - optimize %s using wrong upw should fail!\n", f)
	}

	// Optimize empty upw
	t.Log("Optimize empty upw")
	config = types.NewDefaultConfiguration()
	config.UserPW = ""
	_, err = Process(OptimizeCommand(f, f, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - optimize %s using upw: %v\n", f, err)
	}

	// Change opw wrong upw
	t.Log("ChangeOwnerPW wrong upw fails")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	pwOld := "opw"
	pwNew := "opwNew"
	_, err = Process(ChangeOwnerPWCommand(f, f, config, &pwOld, &pwNew))
	if err == nil {
		t.Fatalf("TestEncryptOPWOnly - %s change opw using wrong upw should fail\n", f)
	}

	// Change opw wrong opwOld
	t.Log("ChangeOwnerPW wrong opwOld fails")
	config = types.NewDefaultConfiguration()
	config.UserPW = ""
	pwOld = "opwOldWrong"
	pwNew = "opwNew"
	_, err = Process(ChangeOwnerPWCommand(f, f, config, &pwOld, &pwNew))
	if err == nil {
		t.Fatalf("TestEncryptOPWOnly - %s change opw using wrong opwOld should fail\n", f)
	}

	// Change opw
	t.Log("ChangeOwnerPW")
	config = types.NewDefaultConfiguration()
	config.UserPW = ""
	pwOld = "opw"
	pwNew = "opwNew"
	_, err = Process(ChangeOwnerPWCommand(f, f, config, &pwOld, &pwNew))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - %s change opw: %v\n", f, err)
	}

	// Decrypt wrong upw
	t.Log("Decrypt wrong upw fails")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(DecryptCommand(f, f, config))
	if err == nil {
		t.Fatalf("TestEncryptOPWOnly - %s decrypt using wrong upw should fail \n", f)
	}

	// Decrypt wrong opw succeeds because of fallback to empty upw.
	t.Log("Decrypt wrong opw succeeds because of fallback to empty upw")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	_, err = Process(DecryptCommand(f, f, config))
	if err != nil {
		t.Fatalf("TestEncryptOPWOnly - %s decrypt using opw: %v\n", f, err)
	}

}

func TestEncrypt(t *testing.T) {

	t.Log("running TestEncrypt..")

	f := outputDir + "/test.pdf"

	// Encrypt opw and upw
	t.Log("Encrypt")
	config := types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	_, err := Process(EncryptCommand("testdata/5116.DCT_Filter.pdf", f, config))
	if err != nil {
		t.Fatalf("TestEncrypt - encrypt to %s: %v\n", f, err)
	}

	// Validate wrong opw
	t.Log("Validate wrong opw fails")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(ValidateCommand(f, config))
	if err == nil {
		t.Fatalf("TestEncrypt - validate %s using wrong opw should fail!\n", f)
	}

	// Validate opw
	t.Log("Validate opw")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	_, err = Process(ValidateCommand(f, config))
	if err != nil {
		t.Fatalf("TestEncrypt - validate %s using opw: %v\n", f, err)
	}

	// Validate wrong upw
	t.Log("Validate wrong upw fails")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(ValidateCommand(f, config))
	if err == nil {
		t.Fatalf("TestEncrypt - validate %s using wrong upw should fail!\n", f)
	}

	// Validate upw
	t.Log("Validate upw")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	_, err = Process(ValidateCommand(f, config))
	if err != nil {
		t.Fatalf("TestEncrypt - validate %s using upw: %v\n", f, err)
	}

	// Change upw to "" = remove document open password.
	t.Log("ChangeUserPW to \"\"")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	pwOld := "upw"
	pwNew := ""
	_, err = Process(ChangeUserPWCommand(f, f, config, &pwOld, &pwNew))
	if err != nil {
		t.Fatalf("TestEncrypt - %s change userPW to \"\": %v\n", f, err)
	}

	// Validate upw
	t.Log("Validate upw")
	config = types.NewDefaultConfiguration()
	config.UserPW = ""
	_, err = Process(ValidateCommand(f, config))
	if err != nil {
		t.Fatalf("TestEncrypt - validate %s using upw: %v\n", f, err)
	}

	// Validate no pw
	t.Log("Validate upw")
	config = types.NewDefaultConfiguration()
	_, err = Process(ValidateCommand(f, config))
	if err != nil {
		t.Fatalf("TestEncrypt - validate %s: %v\n", f, err)
	}

	// Change opw
	t.Log("ChangeOwnerPW")
	config = types.NewDefaultConfiguration()
	config.UserPW = ""
	pwOld = "opw"
	pwNew = "opwNew"
	_, err = Process(ChangeOwnerPWCommand(f, f, config, &pwOld, &pwNew))
	if err != nil {
		t.Fatalf("TestEncrypt - %s change opw: %v\n", f, err)
	}

	// Decrypt wrong upw
	t.Log("Decrypt wrong upw fails")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	_, err = Process(DecryptCommand(f, f, config))
	if err == nil {
		t.Fatalf("TestEncrypt - %s decrypt using wrong upw should fail\n", f)
	}

	// Decrypt wrong opw succeeds on empty upw
	t.Log("Decrypt wrong opw succeeds on empty upw")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opwWrong"
	_, err = Process(DecryptCommand(f, f, config))
	if err != nil {
		t.Fatalf("TestEncrypt - %s decrypt wrong opw, empty upw: %v\n", f, err)
	}
}

func encryptDecrypt(fileName string, config *types.Configuration, t *testing.T) {

	fin := "testdata/" + fileName
	t.Log(fin)
	f := outputDir + "/test.pdf"

	// Encrypt
	t.Log("Encrypt")
	_, err := Process(EncryptCommand(fin, f, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - encrypt %s: %v\n", f, err)
	}

	// Encrypt already encrypted
	t.Log("Encrypt already encrypted")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	_, err = Process(EncryptCommand(f, f, config))
	if err == nil {
		t.Fatalf("TestEncryptDecrypt - encrypt encrypted %s\n", f)
	}

	// Validate using wrong owner pw
	t.Log("Validate wrong ownerPW")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opwWrong"
	_, err = Process(ValidateCommand(f, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - validate %s using wrong ownerPW: %v\n", f, err)
	}

	// Optimize using wrong owner pw
	t.Log("Optimize wrong ownerPW")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opwWrong"
	_, err = Process(OptimizeCommand(f, f, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - optimize %s using wrong ownerPW: %v\n", f, err)
	}

	// Trim using wrong owner pw, falls back to upw and fails with insufficient permissions.
	t.Log("Trim wrong ownerPW, fallback to upw and fail with insufficient permissions.")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opwWrong"
	_, err = Process(TrimCommand(f, f, nil, config))
	if err == nil {
		t.Fatalf("TestEncryptDecrypt - trim %s using wrong ownerPW should fail: \n", f)
	}

	// Split using wrong owner pw, falls back to upw and fails with insufficient permissions.
	t.Log("Split wrong ownerPW, fallback to upw and fail with insufficient permissions.")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opwWrong"
	_, err = Process(SplitCommand(f, outputDir, config))
	if err == nil {
		t.Fatalf("TestEncryptDecrypt - split %s using wrong ownerPW should fail: \n", f)
	}

	// Add permissions
	t.Log("Add user access permissions")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	config.UserAccessPermissions = types.PermissionsAll
	_, err = Process(AddPermissionsCommand(f, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - %s add permissions: %v\n", f, err)
	}

	// Split using wrong owner pw, falls back to upw
	t.Log("Split wrong ownerPW")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opwWrong"
	_, err = Process(SplitCommand(f, outputDir, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - split %s using wrong ownerPW: %v\n", f, err)
	}

	// Validate
	t.Log("Validate")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	_, err = Process(ValidateCommand(f, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - validate %s: %v\n", f, err)
	}

	// ChangeUserPW using wrong userpw
	t.Log("ChangeUserPW wrong userpw")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	pwOld := "upwWrong"
	pwNew := "upwNew"
	_, err = Process(ChangeUserPWCommand(f, f, config, &pwOld, &pwNew))
	if err == nil {
		t.Fatalf("TestEncryption - %s change userPW using wrong userPW should fail:\n", f)
	}

	// ChangeUserPW
	t.Log("ChangeUserPW")
	config = types.NewDefaultConfiguration()
	config.OwnerPW = "opw"
	pwOld = "upw"
	pwNew = "upwNew"
	_, err = Process(ChangeUserPWCommand(f, f, config, &pwOld, &pwNew))
	if err != nil {
		t.Fatalf("TestEncryption - change userPW %s: %v\n", f, err)
	}

	// ChangeOwnerPW
	t.Log("ChangeOwnerPW")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upwNew"
	pwOld = "opw"
	pwNew = "opwNew"
	_, err = Process(ChangeOwnerPWCommand(f, f, config, &pwOld, &pwNew))
	if err != nil {
		t.Fatalf("TestEncryption - change ownerPW %s: %v\n", f, err)
	}

	// Decrypt using wrong pw
	t.Log("\nDecrypt using wrong pw")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upwWrong"
	config.OwnerPW = "opwWrong"
	_, err = Process(DecryptCommand(f, f, config))
	if err == nil {
		t.Fatalf("TestEncryptDecrypt - decrypt using wrong pw %s\n", f)
	}

	// Decrypt
	t.Log("\nDecrypt")
	config = types.NewDefaultConfiguration()
	config.UserPW = "upwNew"
	config.OwnerPW = "opwNew"
	_, err = Process(DecryptCommand(f, f, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - decrypt %s: %v\n", f, err)
	}

}

func TestEncryptDecrypt(t *testing.T) {

	config := types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	encryptDecrypt("5116.DCT_Filter.pdf", config, t)

	config = types.NewDefaultConfiguration()
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
	list, err := Process(ListAttachmentsCommand(fileName, config))
	if err != nil {
		t.Fatalf("TestAttachments - list attachments %s: %v\n", fileName, err)
	}
	if len(list) > 0 {
		t.Fatalf("TestAttachments - list attachments %s: should have 0 attachments\n", fileName)
	}

	// attach add 2 files
	_, err = Process(AddAttachmentsCommand(fileName, []string{outputDir + "/golang.pdf", outputDir + "/T4.pdf"}, config))
	if err != nil {
		t.Fatalf("TestAttachments - add attachments to %s: %v\n", fileName, err)
	}

	// attach list must be 2
	list, err = Process(ListAttachmentsCommand(fileName, config))
	if err != nil {
		t.Fatalf("TestAttachments - list attachments %s: %v\n", fileName, err)
	}
	if len(list) != 2 {
		t.Fatalf("TestAttachments - list attachments %s: should have 0 attachments\n", fileName)
	}

	// attach extract all
	_, err = Process(ExtractAttachmentsCommand(fileName, ".", nil, config))
	if err != nil {
		t.Fatalf("TestAttachments - extract all attachments from %s to %s: %v\n", fileName, ".", err)
	}

	// attach extract 1 file
	_, err = Process(ExtractAttachmentsCommand(fileName, ".", []string{outputDir + "/golang.pdf"}, config))
	if err != nil {
		t.Fatalf("TestAttachments - extract 1 attachment from %s to %s: %v\n", fileName, ".", err)
	}

	// attach remove 1 file
	_, err = Process(RemoveAttachmentsCommand(fileName, []string{outputDir + "/golang.pdf"}, config))
	if err != nil {
		t.Fatalf("TestAttachments - remove attachment from %s: %v\n", fileName, err)
	}

	// attach list must be 1
	list, err = Process(ListAttachmentsCommand(fileName, config))
	if err != nil {
		t.Fatalf("TestAttachments - list attachments %s: %v\n", fileName, err)
	}
	if len(list) != 1 {
		t.Fatalf("TestAttachments - list attachments %s: should have 0 attachments\n", fileName)
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

func TestListPermissionsCommand(t *testing.T) {

	fin := "testdata/" + "5116.DCT_Filter.pdf"

	_, err := Process(ListPermissionsCommand(fin, types.NewDefaultConfiguration()))
	if err != nil {
		t.Fatalf("TestListPermissionsCommand: for unencrypted %s: %v\n", fin, err)
	}

	config := types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	f := outputDir + "/test.pdf"
	_, err = Process(EncryptCommand(fin, f, config))
	if err != nil {
		t.Fatalf("TestEncryptDecrypt - encrypt %s: %v\n", f, err)
	}

	config = types.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"
	_, err = Process(ListPermissionsCommand(f, config))
	if err != nil {
		t.Fatalf("TestListPermissionsCommand: for encrypted %s: %v\n", f, err)
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

func xxxTestDemoXRef(t *testing.T) {

	xRefTable, err := create.AnnotationDemoXRef()
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

func TestAnnotationDemoPDF(t *testing.T) {

	//dir := "testdata"
	dir := "testdata/out"

	xRefTable, err := create.AnnotationDemoXRef()
	if err != nil {
		t.Fatalf("testAnnotationDemoPDF %v\n", err)
	}

	err = create.DemoPDF(xRefTable, dir+"/", "annotationDemo.pdf")
	if err != nil {
		t.Fatalf("testAnnotationDemoPDF %v\n", err)
	}

	config := types.NewDefaultConfiguration()
	config.SetValidationRelaxed()

	_, err = Process(ValidateCommand(dir+"/"+"annotationDemo.pdf", config))
	if err != nil {
		t.Fatalf("testAnnotationDemoPDF %v\n", err)
	}

}

func TestAcroformDemoPDF(t *testing.T) {

	//dir := "testdata"
	dir := "testdata/out"

	xRefTable, err := create.AcroFormDemoXRef()
	if err != nil {
		t.Fatalf("testAcroformDemoPDF %v\n", err)
	}

	err = create.DemoPDF(xRefTable, dir+"/", "acroFormDemo.pdf")
	if err != nil {
		t.Fatalf("testAcroformDemoPDF %v\n", err)
	}

	config := types.NewDefaultConfiguration()
	config.SetValidationRelaxed()

	_, err = Process(ValidateCommand(dir+"/"+"acroFormDemo.pdf", config))
	if err != nil {
		t.Fatalf("testAcroformDemoPDF %v\n", err)
	}

}
