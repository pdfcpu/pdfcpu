/*
Copyright 2025 The pdfcpu Authors.

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
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func logResults(ss []string) {
	for _, s := range ss {
		fmt.Println(s)
	}
}

// TestValidateSignature_X509_RSA_SHA1 exercises validation-only compatibility
// for legacy adbe.x509.rsa_sha1 signatures.
func TestValidateSignature_X509_RSA_SHA1(t *testing.T) {
	msg := "ValidateSignature_X509_RSA_SHA1"

	// You may provide your signed PDFs in this dir.
	dir := filepath.Join(samplesDir, "signatures", "adbe.x509.rsa_sha1")

	for _, fn := range AllPDFs(t, dir) {
		inFile := filepath.Join(dir, fn)
		fmt.Println("\nvalidate signatures in " + inFile)
		all, full := true, true
		ss, err := api.ValidateSignaturesFile(inFile, all, full, conf)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		logResults(ss)
	}
}

// TestValidateSignature_PKCS7_SHA1 exercises validation-only compatibility
// for legacy adbe.pkcs7.sha1 signatures.
func TestValidateSignature_PKCS7_SHA1(t *testing.T) {
	msg := "ValidateSignature_PKCS7_SHA1"

	// You may provide your signed PDFs in this dir.
	dir := filepath.Join(samplesDir, "signatures", "adbe.pkcs7.sha1")

	for _, fn := range AllPDFs(t, dir) {
		inFile := filepath.Join(dir, fn)
		fmt.Println("validate signatures in " + inFile)
		all, full := true, true
		ss, err := api.ValidateSignaturesFile(inFile, all, full, conf)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		logResults(ss)
	}
}

// TestValidateSignature_PKCS7_Detached exercises validation of
// adbe.pkcs7.detached signatures.
func TestValidateSignature_PKCS7_Detached(t *testing.T) {
	msg := "ValidateSignature_PKCS7_Detached"

	// You may provide your signed PDFs in this dir.
	dir := filepath.Join(samplesDir, "signatures", "adbe.pkcs7.detached")

	for _, fn := range AllPDFs(t, dir) {
		inFile := filepath.Join(dir, fn)
		fmt.Println("\nvalidate signatures in " + inFile)
		all, full := false, true
		ss, err := api.ValidateSignaturesFile(inFile, all, full, conf)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		logResults(ss)
	}
}

// TestValidateSignature_ETSI_CAdES_Detached exercises supported
// ETSI.CAdES.detached signature validation.
func TestValidateSignature_ETSI_CAdES_Detached(t *testing.T) {
	msg := "ValidateSignature_ETSI_CAdES_Detached"

	// You may provide your signed PDFs in this dir.
	dir := filepath.Join(samplesDir, "signatures", "ETSI.CAdES.detached")

	for _, fn := range AllPDFs(t, dir) {
		inFile := filepath.Join(dir, fn)
		fmt.Println("\nvalidate signatures in " + inFile)
		all, full := true, true
		ss, err := api.ValidateSignaturesFile(inFile, all, full, conf)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		logResults(ss)
	}
}

// TestRemoveSignatures verifies signatures can be removed from a signed PDF.
func TestRemoveSignatures(t *testing.T) {
	msg := "TestRemoveSignatures"

	inDir := filepath.Join(samplesDir, "signatures", "ETSI.CAdES.detached")
	inFile := filepath.Join(inDir, "testPAdES_BB.pdf")
	outFile := filepath.Join(outDir, "testPAdES_BB_noSigs.pdf")

	//conf := model.NewDefaultConfiguration()
	//conf.RemoveEncryption = true
	if err := api.RemoveSignaturesFile(inFile, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}
