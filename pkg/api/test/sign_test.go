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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func TestSign(t *testing.T) {
	msg := "TestSign"
	fileName := "ProgrammingInJava.pdf"
	inFile := filepath.Join(inDir, fileName)
	fnNoExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	outFile := filepath.Join(inDir, fnNoExt+"_signed.pdf")

	// Create a signed version of inFile.
	err := api.SignFile(inFile, outFile, nil)
	if err == pdfcpu.ErrHasAcroForm {
		t.Log("already has an AcroForm")
		return
	}
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

}
