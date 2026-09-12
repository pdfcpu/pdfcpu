/*
Copyright 2026 The pdfcpu Authors.

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
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func relaxedOnlyValidationTestPDF() []byte {
	return validationModeTestPDF("D:")
}

func strictValidationTestPDF() []byte {
	return validationModeTestPDF("D:20200101000000Z")
}

func validationModeTestPDF(creationDate string) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n%\xFF\xFF\xFF\xFF\n")

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Count 1 /Kids [3 0 R] >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>",
		fmt.Sprintf("<< /CreationDate (%s) >>", creationDate),
	}
	offsets := make([]int, len(objects))
	for i, object := range objects {
		offsets[i] = appendValidationTestObject(&buf, i+1, object)
	}

	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(
		&buf,
		"trailer\n<< /Size 5 /Root 1 0 R /Info 4 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		xrefOffset,
	)
	return buf.Bytes()
}

func TestPDFInfoAlwaysUsesRelaxedValidation(t *testing.T) {
	pdf := relaxedOnlyValidationTestPDF()
	strict := model.NewDefaultConfiguration()
	strict.ValidationMode = model.ValidationStrict

	if _, err := ReadAndValidate(t.Context(), bytes.NewReader(pdf), strict); err == nil {
		t.Fatal("expected strict validation error")
	}
	info, err := PDFInfo(t.Context(), bytes.NewReader(pdf), "relaxed-only.pdf", nil, false, strict)
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal("expected PDF information")
	}
	if strict.ValidationMode != model.ValidationStrict {
		t.Fatalf("caller validation mode: got %d, want %d", strict.ValidationMode, model.ValidationStrict)
	}
}

type validationModeOperation struct {
	name string
	run  func(*model.Configuration) error
}

func configuredValidationModeOperations(c context.Context) []validationModeOperation {
	return []validationModeOperation{
		{"bookmarks", func(conf *model.Configuration) error {
			_, err := Bookmarks(c, bytes.NewReader(relaxedOnlyValidationTestPDF()), conf)
			return err
		}},
		{"keywords", func(conf *model.Configuration) error {
			_, err := Keywords(c, bytes.NewReader(relaxedOnlyValidationTestPDF()), conf)
			return err
		}},
		{"properties", func(conf *model.Configuration) error {
			_, err := Properties(c, bytes.NewReader(relaxedOnlyValidationTestPDF()), conf)
			return err
		}},
		{"page layout", func(conf *model.Configuration) error {
			_, err := PageLayout(c, bytes.NewReader(relaxedOnlyValidationTestPDF()), conf)
			return err
		}},
		{"page mode", func(conf *model.Configuration) error {
			_, err := PageMode(c, bytes.NewReader(relaxedOnlyValidationTestPDF()), conf)
			return err
		}},
		{"viewer preferences", func(conf *model.Configuration) error {
			_, _, err := ViewerPreferences(c, bytes.NewReader(relaxedOnlyValidationTestPDF()), conf)
			return err
		}},
	}
}

func TestConfiguredValidationModeIsHonored(t *testing.T) {
	for _, tt := range configuredValidationModeOperations(t.Context()) {
		t.Run(tt.name, func(t *testing.T) {
			strict := model.NewDefaultConfiguration()
			strict.ValidationMode = model.ValidationStrict
			err := tt.run(strict)
			var validationErr *model.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("strict mode: expected validation error, got %v", err)
			}
			if strict.ValidationMode != model.ValidationStrict {
				t.Fatalf("caller validation mode: got %d, want %d", strict.ValidationMode, model.ValidationStrict)
			}

			relaxed := model.NewDefaultConfiguration()
			relaxed.ValidationMode = model.ValidationRelaxed
			if err := tt.run(relaxed); err != nil {
				t.Fatalf("relaxed mode: %v", err)
			}
		})
	}
}
