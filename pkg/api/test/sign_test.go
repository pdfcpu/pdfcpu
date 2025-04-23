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

	// You may provide your signed PDFs in this dir.
	dir := filepath.Join(samplesDir, "signatures", "adbe.x509.rsa_sha1")

	for _, fn := range AllPDFs(t, dir) {
		inFile := filepath.Join(dir, fn)
		fmt.Println("\nvalidate signatures of " + inFile)
		all := true
		full := false
		ss, err := api.ValidateSignaturesFile(inFile, all, full, conf)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		logResults(ss)
	}
}

func TestValidateSignature_PKCS7_SHA1(t *testing.T) {
	msg := "ValidateSignature_PKCS7_SHA1"

	// You may provide your signed PDFs in this dir.
	dir := filepath.Join(samplesDir, "signatures", "adbe.pkcs7.sha1")

	for _, fn := range AllPDFs(t, dir) {
		inFile := filepath.Join(dir, fn)
		fmt.Println("validate signatures of " + inFile)
		all := true
		full := false
		ss, err := api.ValidateSignaturesFile(inFile, all, full, conf)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		logResults(ss)
	}
}

func TestValidateSignature_PKCS7_Detached(t *testing.T) {
	msg := "ValidateSignature_PKCS7_Detached"

	// You may provide your signed PDFs in this dir.
	dir := filepath.Join(samplesDir, "signatures", "adbe.pkcs7.detached")

	for _, fn := range AllPDFs(t, dir) {
		inFile := filepath.Join(dir, fn)
		fmt.Println("\nvalidate signatures of " + inFile)
		all := true
		full := true
		ss, err := api.ValidateSignaturesFile(inFile, all, full, conf)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		logResults(ss)
	}
}

func TestValidateSignature_ETSI_CAdES_Detached(t *testing.T) {
	msg := "ValidateSignature_ETSI_CAdES_Detached"

	// You may provide your signed PDFs in this dir.
	dir := filepath.Join(samplesDir, "signatures", "ETSI.CAdES.detached")

	for _, fn := range AllPDFs(t, dir) {
		inFile := filepath.Join(dir, fn)
		fmt.Println("\nvalidate signatures of " + inFile)
		all := true
		full := true
		ss, err := api.ValidateSignaturesFile(inFile, all, full, conf)
		if err != nil {
			t.Fatalf("%s: %v\n", msg, err)
		}
		logResults(ss)
	}
}
