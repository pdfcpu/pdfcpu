/*
Copyright 2025 The pdf Authors.

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
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func logResults(ss []string) {
	for _, s := range ss {
		fmt.Println(s)
	}
}

func TestValidateSignature_X509_RSA_SHA1(t *testing.T) {
	msg := "ValidateSignature_X509_RSA_SHA1"

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	inDir := filepath.Join(homeDir, "Documents/pdfcpu/SignatureSamples")

	dir := filepath.Join(inDir, "signed", "adbe.x509.rsa_sha1")
	for _, fn := range AllPDFs(t, dir) {
		//inFile := filepath.Join(dir, "sample01.pdf")
		inFile := filepath.Join(dir, fn)
		fmt.Println("\nvalidate signatures of " + inFile)
		all := true
		full := true
		ss, err := api.ValidateSignaturesFile(inFile, all, full, nil)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		logResults(ss)
	}
}

func TestValidateSignature_PKCS7_SHA1(t *testing.T) {
	msg := "ValidateSignature_PKCS7_SHA1"

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	inDir := filepath.Join(homeDir, "Documents/pdfcpu/SignatureSamples")
	dir := filepath.Join(inDir, "signed", "adbe.pkcs7.sha1")

	for _, fn := range AllPDFs(t, dir) {
		inFile := filepath.Join(dir, fn)
		fmt.Println("validate signatures of " + inFile)
		all := true
		full := true
		ss, err := api.ValidateSignaturesFile(inFile, all, full, nil)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		logResults(ss)
	}
}

func TestValidateSignature_PKCS7_Detached(t *testing.T) {
	msg := "ValidateSignature_PKCS7_Detached"

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	inDir := filepath.Join(homeDir, "Documents/pdfcpu/SignatureSamples")
	dir := filepath.Join(inDir, "signed", "adbe.pkcs7.detached")
	//for _, fn := range AllPDFs(t, dir) {
	//inFile := filepath.Join(dir, "blank_signed.pdf")
	inFile := filepath.Join(dir, "digitalsignature.pdf") // LTVenabled and valid !!!
	//inFile := filepath.Join(dir, "i264.pdf")
	//inFile := filepath.Join(dir, "i677.pdf")
	//inFile := filepath.Join(dir, "inv.pdf")
	//inFile := filepath.Join(dir, "pdf_digital_signature_timestamp.pdf")
	//inFile := filepath.Join(dir, "Rechnung1.pdf")
	//inFile := filepath.Join(dir, "samplecertified.pdf") // LTVenabled and valid !!!
	//inFile := filepath.Join(dir, "WienEnergie2.pdf")
	//inFile := filepath.Join(dir, "erste.pdf")

	//inFile := filepath.Join(dir, "i767b.pdf")

	//inFile := filepath.Join(dir, fn)

	fmt.Println("\nvalidate signatures of " + inFile)
	all := true
	full := true
	ss, err := api.ValidateSignaturesFile(inFile, all, full, nil)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	logResults(ss)
	//}
}

func TestValidateSignature_ETSI_CAdES_Detached(t *testing.T) {
	msg := "ValidateSignature_ETSI_CAdES_Detached"

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	inDir := filepath.Join(homeDir, "Documents/pdfcpu/SignatureSamples")
	dir := filepath.Join(inDir, "signed", "ETSI.CAdES.detached")
	//for _, fn := range AllPDFs(t, dir) {
	//inFile := filepath.Join(dir, "1069910twice.pdf") // timestamp token signature verification failed: pkcs7: unsupported algorithm "1.2.840.113549.1.1.10"
	//inFile := filepath.Join(dir, "Beitragsgrundlage.pdf")
	//inFile := filepath.Join(dir, "Bescheid.pdf") // signature verification failure: crypto/rsa: verification error
	//inFile := filepath.Join(dir, "Bestaetigung.pdf")
	//inFile := filepath.Join(dir, "BGBLA.pdf")
	//inFile := filepath.Join(dir, "ContratoCurso.pdf")
	//inFile := filepath.Join(dir, "Datenauszug.pdf")
	//inFile := filepath.Join(dir, "dringliche.pdf")
	//inFile := filepath.Join(dir, "Horst_AtrustSig.pdf")
	//inFile := filepath.Join(dir, "signed_annot_sign.pdf")
	//inFile := filepath.Join(dir, "test_signiert_problem.pdf")
	//inFile := filepath.Join(dir, "tw.pdf")
	//inFile := filepath.Join(dir, "unterstuetzung.pdf") // pkcs7: signature verification failure: crypto/rsa: verification error
	//inFile := filepath.Join(dir, "vorschreibung.pdf")
	//inFile := filepath.Join(dir, "Wahlkartenantrag.pdf")
	//inFile := filepath.Join(dir, "testPAdES_BB.pdf")
	//inFile := filepath.Join(dir, "testPAdES_BT.pdf")
	//inFile := filepath.Join(dir, "testPAdES_BLT.pdf")
	inFile := filepath.Join(dir, "testPAdES_BLTA.pdf")

	//inFile := filepath.Join(dir, fn)

	fmt.Println("\nvalidate signatures of " + inFile)
	all := true
	full := true
	ss, err := api.ValidateSignaturesFile(inFile, all, full, nil)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	logResults(ss)
	//}
}
