/*
Copyright 2019 The pdf Authors.

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

package api

import (
	"path/filepath"
	"testing"

	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func createAndValidate(t *testing.T, xRefTable *pdf.XRefTable, outFile, msg string) {
	t.Helper()
	if err := CreatePDFFile(xRefTable, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestCreateDemoPDF(t *testing.T) {
	msg := "TestCreateDemoPDF"
	outFile := filepath.Join(outDir, "demo.pdf")
	xRefTable, err := pdf.CreateDemoXRef()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	createAndValidate(t, xRefTable, outFile, msg)
}

func TestAnnotationDemoPDF(t *testing.T) {
	msg := "TestAnnotationDemoPDF"
	outFile := filepath.Join(outDir, "annotationDemo.pdf")
	xRefTable, err := pdf.CreateAnnotationDemoXRef()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	createAndValidate(t, xRefTable, outFile, msg)
}

func TestAcroformDemoPDF(t *testing.T) {
	msg := "TestAcroformDemoPDF"
	outFile := filepath.Join(outDir, "acroFormDemo.pdf")
	xRefTable, err := pdf.CreateAcroFormDemoXRef()
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	createAndValidate(t, xRefTable, outFile, msg)
}
