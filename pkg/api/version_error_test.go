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
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestVersionErrorsAliasLowerLayers verifies the API exposes the model error contract unchanged.
func TestVersionErrorsAliasLowerLayers(t *testing.T) {
	if ErrVersionTooLow != pdfcpu.ErrVersionTooLow {
		t.Fatal("ErrVersionTooLow is not an alias of pdfcpu.ErrVersionTooLow")
	}
	if pdfcpu.ErrVersionTooLow != model.ErrVersionTooLow {
		t.Fatal("pdfcpu.ErrVersionTooLow is not an alias of model.ErrVersionTooLow")
	}

	err := error(&model.VersionRequirementError{
		Element:         "Example",
		ActualVersion:   model.V13,
		RequiredVersion: model.V14,
	})
	var apiErr *VersionRequirementError
	if !errors.As(err, &apiErr) {
		t.Fatal("VersionRequirementError is not an alias of model.VersionRequirementError")
	}
	if !errors.Is(err, ErrVersionTooLow) {
		t.Fatalf("got %v, want %v", err, ErrVersionTooLow)
	}
	if errors.Is(err, pdfcpu.ErrUnsupportedVersion) {
		t.Fatalf("version requirement error matches operation-level error %v", pdfcpu.ErrUnsupportedVersion)
	}
}

// TestValidationErrorPreservesVersionRequirement verifies API context wrapping preserves the model error contract.
func TestValidationErrorPreservesVersionRequirement(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationStrict
	versionCause := &model.VersionRequirementError{
		Element:         "dict=Example entry=Feature",
		ActualVersion:   model.V13,
		RequiredVersion: model.V14,
	}
	cause := model.WithValidationErrorObject(versionCause, 20)

	err := fmt.Errorf("validate example.pdf: %w", validationError(conf, cause))
	if !errors.Is(err, ErrVersionTooLow) {
		t.Fatalf("got %v, want %v", err, ErrVersionTooLow)
	}

	var versionErr *VersionRequirementError
	if !errors.As(err, &versionErr) {
		t.Fatalf("got %T, want *VersionRequirementError", err)
	}
	if versionErr.Element != versionCause.Element || versionErr.ActualVersion != versionCause.ActualVersion ||
		versionErr.RequiredVersion != versionCause.RequiredVersion {
		t.Fatalf("got %+v, want %+v", versionErr, versionCause)
	}

	for _, want := range []string{"validate example.pdf", "validation error (obj#:20)", "try --mode=relaxed"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("missing context %q in %q", want, err)
		}
	}
}

func TestValidationErrorOmitsUnknownObject(t *testing.T) {
	conf := model.NewDefaultConfiguration()

	err := validationError(conf, errors.New("validation failed"))
	if strings.Contains(err.Error(), "obj#:") {
		t.Fatalf("error uses last dereferenced object: %q", err)
	}
	if want := "validation error: validation failed"; err.Error() != want {
		t.Fatalf("got %q, want %q", err, want)
	}
}
