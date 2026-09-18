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

package pdfcpu

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func malformedCatalogKeyPDF(t *testing.T) []byte {
	t.Helper()
	pdf := catalogEndobjPDF(false)
	valid := []byte("<< /Type /Catalog")
	malformed := []byte("<<@/Type /Catalog")
	if bytes.Count(pdf, valid) != 1 || len(valid) != len(malformed) {
		t.Fatal("unexpected catalog fixture")
	}
	return bytes.Replace(pdf, valid, malformed, 1)
}

func readMalformedCatalogKey(t *testing.T, mode int) (*model.Context, error) {
	t.Helper()
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = mode
	ctx, err := Read(t.Context(), bytes.NewReader(malformedCatalogKeyPDF(t)), conf)
	if err == nil {
		err = ctx.EnsurePageCount()
	}
	return ctx, err
}

func TestStrictReadRejectsMalformedDictionaryKey(t *testing.T) {
	ctx, err := readMalformedCatalogKey(t, model.ValidationStrict)
	if err == nil {
		t.Fatal("strict read unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "corrupt dictionary key") || !strings.Contains(err.Error(), "corrupt name object") {
		t.Fatalf("strict read error: got %q", err)
	}
	var validationErr *model.ValidationError
	if !errors.As(err, &validationErr) || validationErr.ObjectNumber() != 3 {
		t.Fatalf("strict read attribution: got %v", err)
	}
	if ctx != nil && !ctx.ValidationReport().Empty() {
		t.Fatalf("strict read reported accepted divergences: %+v", ctx.ValidationReport().Notices())
	}
}

func TestRelaxedReadReportsMalformedDictionaryKey(t *testing.T) {
	ctx, err := readMalformedCatalogKey(t, model.ValidationRelaxed)
	if err != nil {
		t.Fatalf("relaxed read: %v", err)
	}

	notices := ctx.ValidationReport().Notices()
	if len(notices) != 1 {
		t.Fatalf("notice count: got %d, want 1", len(notices))
	}
	notice := notices[0]
	if notice.Phase != model.NoticePhaseParse || notice.Disposition != model.NoticeDigested {
		t.Fatalf("notice classification: got %q/%q", notice.Phase, notice.Disposition)
	}
	if notice.ObjectNumber != 3 || notice.Message != "object dictionary contains a non-name key token" {
		t.Fatalf("notice: got %+v", notice)
	}
	if notice.Cause == nil || !strings.Contains(notice.Cause.Error(), "corrupt dictionary key") {
		t.Fatalf("notice cause: got %v", notice.Cause)
	}
}
