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

package test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func confForAlgorithm(aes bool, keyLength int) *pdfcpu.Configuration {
	c := pdfcpu.NewDefaultConfiguration()
	c.EncryptUsingAES = aes
	c.EncryptKeyLength = keyLength
	return c
}

func testEncryptDecryptUseCase1(t *testing.T, fileName string, aes bool, keyLength int) {
	t.Helper()
	msg := "testEncryptDecryptUseCase1"

	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, "test.pdf")
	t.Log(inFile)

	// Encrypt opw and upw
	t.Log("Encrypt")
	conf := confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	conf.OwnerPW = "opw"

	cmd := cli.EncryptCommand(inFile, outFile, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: encrypt to %s: %v\n", msg, outFile, err)
	}

	// Validate wrong opw
	t.Log("Validate wrong opw fails")
	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opwWrong"
	if err := validateFile(t, outFile, conf); err == nil {
		t.Fatalf("%s: validate %s using wrong opw should fail!\n", msg, outFile)
	}

	// Validate opw
	t.Log("Validate opw")
	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: validate %s using opw: %v\n", msg, outFile, err)
	}

	// Validate wrong upw
	t.Log("Validate wrong upw fails")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upwWrong"
	if err := validateFile(t, outFile, conf); err == nil {
		t.Fatalf("%s: validate %s using wrong upw should fail!\n", msg, outFile)
	}

	// Validate upw
	t.Log("Validate upw")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: validate %s using upw: %v\n", msg, outFile, err)
	}

	// Change upw to "" = remove document open password.
	t.Log("ChangeUserPW to \"\"")
	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	pwOld := "upw"
	pwNew := ""
	cmd = cli.ChangeUserPWCommand(outFile, "", &pwOld, &pwNew, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %s change userPW to \"\": %v\n", msg, outFile, err)
	}

	// Validate upw
	t.Log("Validate upw")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = ""
	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: validate %s using upw: %v\n", msg, outFile, err)
	}

	// Validate no pw
	t.Log("Validate upw")
	conf = confForAlgorithm(aes, keyLength)
	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: validate %s: %v\n", msg, outFile, err)
	}

	// Change opw
	t.Log("ChangeOwnerPW")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = ""
	pwOld = "opw"
	pwNew = "opwNew"
	cmd = cli.ChangeOwnerPWCommand(outFile, "", &pwOld, &pwNew, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %s change opw: %v\n", msg, outFile, err)
	}

	// Decrypt wrong upw
	t.Log("Decrypt wrong upw fails")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upwWrong"
	cmd = cli.DecryptCommand(outFile, "", conf)
	if _, err := cli.Process(cmd); err == nil {
		t.Fatalf("%s: %s decrypt using wrong upw should fail\n", msg, outFile)
	}

	// Decrypt wrong opw succeeds on empty upw
	t.Log("Decrypt wrong opw succeeds on empty upw")
	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opwWrong"
	cmd = cli.DecryptCommand(outFile, "", conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %s decrypt wrong opw, empty upw: %v\n", msg, outFile, err)
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

func testEncryptDecryptUseCase2(t *testing.T, fileName string, aes bool, keyLength int) {
	t.Helper()
	msg := "testEncryptDecryptUseCase2"

	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, "test.pdf")
	t.Log(inFile)

	// Encrypt
	t.Log("Encrypt")
	conf := confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	conf.OwnerPW = "opw"
	cmd := cli.EncryptCommand(inFile, outFile, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: encrypt to %s: %v\n", msg, outFile, err)
	}

	// Encrypt already encrypted
	t.Log("Encrypt already encrypted")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	conf.OwnerPW = "opw"
	cmd = cli.EncryptCommand(outFile, "", conf)
	if _, err := cli.Process(cmd); err == nil {
		t.Fatalf("%s encrypt encrypted %s\n", msg, outFile)
	}

	// Validate using wrong owner pw
	t.Log("Validate wrong ownerPW")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	conf.OwnerPW = "opwWrong"
	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: validate %s using wrong ownerPW: %v\n", msg, outFile, err)
	}

	// Optimize using wrong owner pw
	t.Log("Optimize wrong ownerPW")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	conf.OwnerPW = "opwWrong"
	if err := optimizeFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: optimize %s using wrong ownerPW: %v\n", msg, outFile, err)
	}

	// Trim using wrong owner pw, falls back to upw and fails with insufficient permissions.
	t.Log("Trim wrong ownerPW, fallback to upw and fail with insufficient permissions.")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	conf.OwnerPW = "opwWrong"
	selectedPages := []string(nil) // writes w/o trimming anything, but sufficient for testing.
	cmd = cli.TrimCommand(outFile, "", selectedPages, conf)
	if _, err := cli.Process(cmd); err == nil {
		t.Fatalf("%s: trim %s using wrong ownerPW should fail: \n", msg, outFile)
	}

	// Set permissions
	t.Log("Add user access permissions")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	conf.OwnerPW = "opw"
	conf.Permissions = pdfcpu.PermissionsAll
	cmd = cli.SetPermissionsCommand(outFile, "", conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %s add permissions: %v\n", msg, outFile, err)
	}

	// List permissions
	conf = pdfcpu.NewDefaultConfiguration()
	conf.OwnerPW = "opw"
	cmd = cli.ListPermissionsCommand(outFile, conf)
	list, err := cli.Process(cmd)
	if err != nil {
		t.Fatalf("%s: list permissions for %s: %v\n", msg, outFile, err)
	}
	ensurePermissionsAll(t, list)

	// Split using wrong owner pw, falls back to upw
	t.Log("Split wrong ownerPW")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	conf.OwnerPW = "opwWrong"
	cmd = cli.SplitCommand(outFile, outDir, 1, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: trim %s using wrong ownerPW falls back to upw: \n", msg, outFile)
	}

	// Validate
	t.Log("Validate")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	conf.OwnerPW = "opw"
	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: validate %s: %v\n", msg, outFile, err)
	}

	// ChangeUserPW using wrong userpw
	t.Log("ChangeUserPW wrong userpw")
	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	pwOld := "upwWrong"
	pwNew := "upwNew"
	cmd = cli.ChangeUserPWCommand(outFile, "", &pwOld, &pwNew, conf)
	if _, err := cli.Process(cmd); err == nil {
		t.Fatalf("%s: %s change userPW using wrong userPW should fail:\n", msg, outFile)
	}

	// ChangeUserPW
	t.Log("ChangeUserPW")
	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	pwOld = "upw"
	pwNew = "upwNew"
	cmd = cli.ChangeUserPWCommand(outFile, "", &pwOld, &pwNew, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %s change upw: %v\n", msg, outFile, err)
	}

	// ChangeOwnerPW
	t.Log("ChangeOwnerPW")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upwNew"
	pwOld = "opw"
	pwNew = "opwNew"
	cmd = cli.ChangeOwnerPWCommand(outFile, "", &pwOld, &pwNew, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %s change opw: %v\n", msg, outFile, err)
	}

	// Decrypt using wrong pw
	t.Log("\nDecrypt using wrong pw")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upwWrong"
	conf.OwnerPW = "opwWrong"
	cmd = cli.DecryptCommand(outFile, "", conf)
	if _, err := cli.Process(cmd); err == nil {
		t.Fatalf("%s: decrypt using wrong pw %s\n", msg, outFile)
	}

	// Decrypt
	t.Log("\nDecrypt")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upwNew"
	conf.OwnerPW = "opwNew"
	cmd = cli.DecryptCommand(outFile, "", conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: decrypt %s: %v\n", msg, outFile, err)
	}
}

func testEncryptDecryptUseCase3(t *testing.T, fileName string, aes bool, keyLength int) {
	// Test for setting only the owner password.
	t.Helper()
	msg := "testEncryptDecryptUseCase3"

	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, "test.pdf")
	t.Log(inFile)

	// Encrypt opw only
	t.Log("Encrypt opw only")
	conf := confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	cmd := cli.EncryptCommand(inFile, outFile, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: encrypt with opw only to to %s: %v\n", msg, outFile, err)
	}

	// Validate wrong opw succeeds with fallback to empty upw
	t.Log("Validate wrong opw succeeds with fallback to empty upw")
	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opwWrong"
	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: validate %s using wrong opw succeeds falling back to empty upw: %v\n", msg, outFile, err)
	}

	// Validate opw
	t.Log("Validate opw")
	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: validate %s using opw: %v\n", msg, outFile, err)
	}

	// Validate wrong upw
	t.Log("Validate wrong upw fails")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upwWrong"
	if err := validateFile(t, outFile, conf); err == nil {
		t.Fatalf("%s: validate %s using wrong upw should fail!\n", msg, outFile)
	}

	// Validate no pw using empty upw
	t.Log("Validate no pw using empty upw")
	conf = confForAlgorithm(aes, keyLength)
	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s validate %s no pw using empty upw: %v\n", msg, outFile, err)
	}

	// Optimize wrong opw, succeeds with fallback to empty upw
	t.Log("Optimize wrong opw succeeds with fallback to empty upw")
	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opwWrong"
	if err := optimizeFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: optimize %s using wrong opw succeeds falling back to empty upw: %v\n", msg, outFile, err)
	}

	// Optimize opw
	t.Log("Optimize opw")
	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	if err := optimizeFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: optimize %s using opw: %v\n", msg, outFile, err)
	}

	// Optimize wrong upw
	t.Log("Optimize wrong upw fails")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upwWrong"
	if err := optimizeFile(t, outFile, conf); err == nil {
		t.Fatalf("%s: optimize %s using wrong upw should fail!\n", msg, outFile)
	}

	// Optimize empty upw
	t.Log("Optimize empty upw")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = ""
	if err := optimizeFile(t, outFile, conf); err != nil {
		t.Fatalf("TestEncrypt%s: optimize %s using upw: %v\n", msg, outFile, err)
	}

	// Change opw wrong upw
	t.Log("ChangeOwnerPW wrong upw fails")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	pwOld := "opw"
	pwNew := "opwNew"
	cmd = cli.ChangeOwnerPWCommand(outFile, "", &pwOld, &pwNew, conf)
	if _, err := cli.Process(cmd); err == nil {
		t.Fatalf("%s: %s change opw using wrong upw should fail\n", msg, outFile)
	}

	// Change opw wrong opwOld
	t.Log("ChangeOwnerPW wrong opwOld fails")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = ""
	pwOld = "opwOldWrong"
	pwNew = "opwNew"
	cmd = cli.ChangeOwnerPWCommand(outFile, "", &pwOld, &pwNew, conf)
	if _, err := cli.Process(cmd); err == nil {
		t.Fatalf("%s: %s change opw using wrong upwOld should fail\n", msg, outFile)
	}

	// Change opw
	t.Log("ChangeOwnerPW")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = ""
	pwOld = "opw"
	pwNew = "opwNew"
	cmd = cli.ChangeOwnerPWCommand(outFile, "", &pwOld, &pwNew, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %s change opw: %v\n", msg, outFile, err)
	}

	// Decrypt wrong upw
	t.Log("Decrypt wrong upw fails")
	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upwWrong"
	cmd = cli.DecryptCommand(outFile, "", conf)
	if _, err := cli.Process(cmd); err == nil {
		t.Fatalf("%s: %s decrypt using wrong upw should fail \n", msg, outFile)
	}

	// Decrypt wrong opw succeeds because of fallback to empty upw.
	t.Log("Decrypt wrong opw succeeds because of fallback to empty upw")
	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	cmd = cli.DecryptCommand(outFile, outFile, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %s decrypt using opw: %v\n", msg, outFile, err)
	}
}

func testPermissionsOPWOnly(t *testing.T, fileName string, aes bool, keyLength int) {
	t.Helper()
	msg := "TestPermissionsOPWOnly"

	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, "test.pdf")
	t.Log(inFile)

	cmd := cli.ListPermissionsCommand(inFile, nil)
	list, err := cli.Process(cmd)
	if err != nil {
		t.Fatalf("%s: list permissions %s: %v\n", msg, inFile, err)
	}
	if len(list) == 0 || list[0] != "Full access" {
		t.Fail()
	}

	conf := confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	cmd = cli.EncryptCommand(inFile, outFile, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: encrypt %s: %v\n", msg, outFile, err)
	}

	cmd = cli.ListPermissionsCommand(outFile, nil)
	if list, err = cli.Process(cmd); err != nil {
		t.Fatalf("%s: list permissions %s: %v\n", msg, outFile, err)
	}
	ensurePermissionsNone(t, list)

	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	conf.Permissions = pdfcpu.PermissionsAll
	cmd = cli.SetPermissionsCommand(outFile, "", conf)
	if _, err = cli.Process(cmd); err != nil {
		t.Fatalf("%s: set all permissions for %s: %v\n", msg, outFile, err)
	}

	cmd = cli.ListPermissionsCommand(outFile, nil)
	if list, err = cli.Process(cmd); err != nil {
		t.Fatalf("%s: list permissions for %s: %v\n", msg, outFile, err)
	}
	ensurePermissionsAll(t, list)

	conf = confForAlgorithm(aes, keyLength)
	conf.Permissions = pdfcpu.PermissionsNone
	cmd = cli.SetPermissionsCommand(outFile, "", conf)
	if _, err = cli.Process(cmd); err == nil {
		t.Fatalf("%s: clear all permissions w/o opw for %s\n", msg, outFile)
	}

	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	conf.Permissions = pdfcpu.PermissionsNone
	cmd = cli.SetPermissionsCommand(outFile, "", conf)
	if _, err = cli.Process(cmd); err != nil {
		t.Fatalf("%s: clear all permissions for %s: %v\n", msg, outFile, err)
	}
}

func testPermissions(t *testing.T, fileName string, aes bool, keyLength int) {
	t.Helper()
	msg := "TestPermissions"

	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, "test.pdf")
	t.Log(inFile)

	cmd := cli.ListPermissionsCommand(inFile, nil)
	list, err := cli.Process(cmd)
	if err != nil {
		t.Fatalf("%s: list permissions %s: %v\n", msg, inFile, err)
	}
	if len(list) == 0 || list[0] != "Full access" {
		t.Fail()
	}

	conf := confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	conf.OwnerPW = "opw"
	cmd = cli.EncryptCommand(inFile, outFile, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: encrypt %s: %v\n", msg, outFile, err)
	}

	cmd = cli.ListPermissionsCommand(outFile, nil)
	if _, err = cli.Process(cmd); err == nil {
		t.Fatalf("%s: list permissions w/o pw %s\n", msg, outFile)
	}

	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	cmd = cli.ListPermissionsCommand(outFile, conf)
	if list, err = cli.Process(cmd); err != nil {
		t.Fatalf("%s: list permissions %s: %v\n", msg, outFile, err)
	}
	ensurePermissionsNone(t, list)

	conf = pdfcpu.NewDefaultConfiguration()
	conf.OwnerPW = "opw"
	cmd = cli.ListPermissionsCommand(outFile, conf)
	if list, err = cli.Process(cmd); err != nil {
		t.Fatalf("%s: list permissions %s: %v\n", msg, outFile, err)
	}
	ensurePermissionsNone(t, list)

	conf = confForAlgorithm(aes, keyLength)
	conf.Permissions = pdfcpu.PermissionsAll
	cmd = cli.SetPermissionsCommand(outFile, "", conf)
	if _, err = cli.Process(cmd); err == nil {
		t.Fatalf("%s: set all permissions w/o pw for %s\n", msg, outFile)
	}

	conf = confForAlgorithm(aes, keyLength)
	conf.UserPW = "upw"
	conf.Permissions = pdfcpu.PermissionsAll
	cmd = cli.SetPermissionsCommand(outFile, "", conf)
	if _, err = cli.Process(cmd); err == nil {
		t.Fatalf("%s: set all permissions w/o opw for %s\n", msg, outFile)
	}

	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	conf.Permissions = pdfcpu.PermissionsAll
	cmd = cli.SetPermissionsCommand(outFile, "", conf)
	if _, err = cli.Process(cmd); err == nil {
		t.Fatalf("%s: set all permissions w/o both pws for %s\n", msg, outFile)
	}

	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	conf.UserPW = "upw"
	conf.Permissions = pdfcpu.PermissionsAll
	cmd = cli.SetPermissionsCommand(outFile, "", conf)
	if _, err = cli.Process(cmd); err != nil {
		t.Fatalf("%s: set all permissions for %s: %v\n", msg, outFile, err)
	}

	cmd = cli.ListPermissionsCommand(outFile, nil)
	if _, err = cli.Process(cmd); err == nil {
		t.Fatalf("%s: list permissions w/o pw %s\n", msg, outFile)
	}

	conf = confForAlgorithm(aes, keyLength)
	conf.OwnerPW = "opw"
	cmd = cli.ListPermissionsCommand(outFile, conf)
	if list, err = cli.Process(cmd); err != nil {
		t.Fatalf("%s: list permissions for %s: %v\n", msg, outFile, err)
	}
	ensurePermissionsAll(t, list)
}

func testEncryptDecryptFile(t *testing.T, fileName string, mode string, keyLength int) {
	t.Helper()
	testEncryptDecryptUseCase1(t, fileName, mode == "aes", keyLength)
	testEncryptDecryptUseCase2(t, fileName, mode == "aes", keyLength)
	testEncryptDecryptUseCase3(t, fileName, mode == "aes", keyLength)
	testPermissionsOPWOnly(t, fileName, mode == "aes", keyLength)
	testPermissions(t, fileName, mode == "aes", keyLength)
}

func TestEncryptDecrypt(t *testing.T) {
	for _, fileName := range []string{
		"5116.DCT_Filter.pdf",
		"networkProgr.pdf",
	} {
		testEncryptDecryptFile(t, fileName, "rc4", 40)
		testEncryptDecryptFile(t, fileName, "rc4", 128)
		testEncryptDecryptFile(t, fileName, "aes", 40)
		testEncryptDecryptFile(t, fileName, "aes", 128)
		testEncryptDecryptFile(t, fileName, "aes", 256)
	}
}
