/*
Copyright 2020 The pdf Authors.

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

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

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

func setPermissions(t *testing.T, aes bool, keyLength int, msg, outFile string) {
	t.Helper()
	// Set all permissions of encrypted file w/o passwords should fail.
	conf := confForAlgorithm(aes, keyLength, "", "")
	conf.Permissions = pdf.PermissionsAll
	if err := api.SetPermissionsFile(outFile, "", conf); err == nil {
		t.Fatalf("%s: set all permissions w/o pw for %s\n", msg, outFile)
	}

	// Set all permissions of encrypted file with user password should fail.
	conf = confForAlgorithm(aes, keyLength, "upw", "")
	conf.Permissions = pdf.PermissionsAll
	if err := api.SetPermissionsFile(outFile, "", conf); err == nil {
		t.Fatalf("%s: set all permissions w/o opw for %s\n", msg, outFile)
	}

	// Set all permissions of encrypted file with owner password should fail.
	conf = confForAlgorithm(aes, keyLength, "", "opw")
	conf.Permissions = pdf.PermissionsAll
	if err := api.SetPermissionsFile(outFile, "", conf); err == nil {
		t.Fatalf("%s: set all permissions w/o both pws for %s\n", msg, outFile)
	}

	// Set all permissions of encrypted file using both passwords.
	conf = confForAlgorithm(aes, keyLength, "upw", "opw")
	conf.Permissions = pdf.PermissionsAll
	if err := api.SetPermissionsFile(outFile, "", conf); err != nil {
		t.Fatalf("%s: set all permissions for %s: %v\n", msg, outFile, err)
	}

	// List permissions using the owner password.
	conf = confForAlgorithm(aes, keyLength, "", "opw")
	p, err := api.GetPermissionsFile(outFile, conf)
	if err != nil {
		t.Fatalf("%s: get permissions %s: %v\n", msg, outFile, err)
	}
	// Ensure permissions all.
	if p == nil || *p != pdf.PermissionsAll {
		t.Fatal()
	}

}

func testEncryption(t *testing.T, fileName string, alg string, keyLength int) {
	t.Helper()
	msg := "testEncryption"

	aes := alg == "aes"
	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, "test.pdf")
	t.Log(inFile)

	p, err := api.GetPermissionsFile(inFile, nil)
	if err != nil {
		t.Fatalf("%s: get permissions %s: %v\n", msg, inFile, err)
	}
	// Ensure full access.
	if p != nil {
		t.Fatal()
	}

	// Encrypt file.
	conf := confForAlgorithm(aes, keyLength, "upw", "opw")
	if err := api.EncryptFile(inFile, outFile, conf); err != nil {
		t.Fatalf("%s: encrypt %s: %v\n", msg, outFile, err)
	}

	// List permissions of encrypted file w/o passwords should fail.
	if list, err := api.ListPermissionsFile(outFile, nil); err == nil {
		t.Fatalf("%s: list permissions w/o pw %s: %v\n", msg, outFile, list)
	}

	// List permissions of encrypted file using the user password.
	conf = confForAlgorithm(aes, keyLength, "upw", "")
	p, err = api.GetPermissionsFile(outFile, conf)
	if err != nil {
		t.Fatalf("%s: get permissions %s: %v\n", msg, inFile, err)
	}
	// Ensure permissions none.
	if p == nil || *p != pdf.PermissionsNone {
		t.Fatal()
	}

	// List permissions of encrypted file using the owner password.
	conf = confForAlgorithm(aes, keyLength, "", "opw")
	p, err = api.GetPermissionsFile(outFile, conf)
	if err != nil {
		t.Fatalf("%s: get permissions %s: %v\n", msg, inFile, err)
	}
	// Ensure permissions none.
	if p == nil || *p != pdf.PermissionsNone {
		t.Fatal()
	}

	setPermissions(t, aes, keyLength, msg, outFile)

	// Change user password.
	conf = confForAlgorithm(aes, keyLength, "upw", "opw")
	if err = api.ChangeUserPasswordFile(outFile, "", "upw", "upwNew", conf); err != nil {
		t.Fatalf("%s: change upw %s: %v\n", msg, outFile, err)
	}

	// Change owner password.
	conf = confForAlgorithm(aes, keyLength, "upwNew", "opw")
	if err = api.ChangeOwnerPasswordFile(outFile, "", "opw", "opwNew", conf); err != nil {
		t.Fatalf("%s: change opw %s: %v\n", msg, outFile, err)
	}

	// Decrypt file using both passwords.
	conf = confForAlgorithm(aes, keyLength, "upwNew", "opwNew")
	if err = api.DecryptFile(outFile, "", conf); err != nil {
		t.Fatalf("%s: decrypt %s: %v\n", msg, outFile, err)
	}

	// Validate decrypted file.
	if err = api.ValidateFile(outFile, nil); err != nil {
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
