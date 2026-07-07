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

package primitives

import (
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestPDFValidateWrapsTopLevelPhase(t *testing.T) {
	pdf := &PDF{
		Paper: "bogus",
		Conf:  model.NewDefaultConfiguration(),
	}

	err := pdf.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "page boundaries") {
		t.Fatalf("expected page boundaries context, got %q", err.Error())
	}
}

func TestPDFValidateWrapsPagePhase(t *testing.T) {
	pdf := &PDF{
		Conf: model.NewDefaultConfiguration(),
		Pages: map[string]*PDFPage{
			"2": {},
		},
	}

	err := pdf.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "page 2: validate") {
		t.Fatalf("expected page validation context, got %q", err.Error())
	}
}
