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

package validate

import (
	"bytes"
	"log"
	"strings"
	"testing"

	pdfcpuLog "github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validationOrderXRefTable() *model.XRefTable {
	v := model.V17
	return &model.XRefTable{
		Table:          map[int]*model.XRefTableEntry{},
		Conf:           model.NewDefaultConfiguration(),
		HeaderVersion:  &v,
		ValidationMode: model.ValidationStrict,
	}
}

func requireDeterministicValidationError(t *testing.T, validate func() error, want string) {
	t.Helper()

	for range 100 {
		err := validate()
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("got %v, want error containing %q", err, want)
		}
	}
}

// TestValidateResourceDictUsesFixedCategoryOrder verifies resource errors follow the declared category order.
func TestValidateResourceDictUsesFixedCategoryOrder(t *testing.T) {
	xRefTable := validationOrderXRefTable()
	d := types.Dict{
		"Font":      types.Integer(1),
		"ExtGState": types.Integer(1),
	}

	requireDeterministicValidationError(t, func() error {
		_, err := validateResourceDict(t.Context(), xRefTable, d)
		return err
	}, "ExtGState resource dict")
}

// TestValidateAdditionalActionsUsesSortedKeys verifies additional-action errors use lexical key order.
func TestValidateAdditionalActionsUsesSortedKeys(t *testing.T) {
	xRefTable := validationOrderXRefTable()
	d := types.Dict{
		"AA": types.Dict{
			"Zulu":  types.Dict{},
			"Alpha": types.Dict{},
		},
	}

	requireDeterministicValidationError(t, func() error {
		return validateAdditionalActions(t.Context(), xRefTable, d, "rootDict", "AA", REQUIRED, model.V10, "root")
	}, "action Alpha not allowed")
}

// TestValidateAppearanceSubDictUsesSortedKeys verifies appearance errors use lexical key order.
func TestValidateAppearanceSubDictUsesSortedKeys(t *testing.T) {
	xRefTable := validationOrderXRefTable()
	d := types.Dict{
		"Zulu":  types.Integer(1),
		"Alpha": types.Integer(1),
	}

	requireDeterministicValidationError(t, func() error {
		return validateAppearanceSubDict(t.Context(), xRefTable, d)
	}, "appearance subdict entry Alpha")
}

// TestValidateCharProcsDictUsesSortedKeys verifies CharProcs errors use lexical key order.
func TestValidateCharProcsDictUsesSortedKeys(t *testing.T) {
	xRefTable := validationOrderXRefTable()
	d := types.Dict{"CharProcs": types.Dict{
		"Zulu":  types.Integer(1),
		"Alpha": types.Integer(1),
	}}

	requireDeterministicValidationError(t, func() error {
		return validateCharProcsDict(xRefTable, d, "fontDict", REQUIRED, model.V10)
	}, "CharProcs entry Alpha")
}

// TestValidateNamedDestinationsUsesSortedKeys verifies named-destination errors use lexical key order.
func TestValidateNamedDestinationsUsesSortedKeys(t *testing.T) {
	xRefTable := validationOrderXRefTable()
	rootDict := types.Dict{
		"Dests": types.Dict{
			"Zulu":  types.Integer(1),
			"Alpha": types.Integer(1),
		},
	}

	requireDeterministicValidationError(t, func() error {
		return validateNamedDestinations(xRefTable, rootDict, REQUIRED, model.V10)
	}, "named destination Alpha")
}

// TestFixFontObjNrReportsRepairsInKeyOrder verifies font repair messages use lexical resource order.
func TestFixFontObjNrReportsRepairsInKeyOrder(t *testing.T) {
	var buf bytes.Buffer
	pdfcpuLog.SetCLILogger(log.New(&buf, "", 0))
	defer pdfcpuLog.SetCLILogger(nil)

	mappings := map[string]string{"Zulu": "F2", "Alpha": "F1"}
	fontRefs := map[string]types.IndirectRef{
		"F1": *types.NewIndirectRef(1, 0),
		"F2": *types.NewIndirectRef(2, 0),
	}
	fixFontObjNr(mappings, fontRefs, types.Dict{})

	s := buf.String()
	alpha := strings.Index(s, "font Alpha mapped")
	zulu := strings.Index(s, "font Zulu mapped")
	if alpha < 0 || zulu < 0 || alpha >= zulu {
		t.Fatalf("expected Alpha repair before Zulu repair, got %q", s)
	}
}

// TestLogURIErrorReportsURIsInKeyOrder verifies URI warnings use lexical URI order.
func TestLogURIErrorReportsURIsInKeyOrder(t *testing.T) {
	var buf bytes.Buffer
	pdfcpuLog.SetCLILogger(log.New(&buf, "", 0))
	defer pdfcpuLog.SetCLILogger(nil)

	xRefTable := validationOrderXRefTable()
	xRefTable.URIs = map[int]map[string]string{
		1: {
			"https://z.example": "i",
			"https://a.example": "i",
		},
	}
	logURIError(xRefTable, []int{1})

	s := buf.String()
	alpha := strings.Index(s, "https://a.example")
	zulu := strings.Index(s, "https://z.example")
	if alpha < 0 || zulu < 0 || alpha >= zulu {
		t.Fatalf("expected a.example warning before z.example warning, got %q", s)
	}
}

// TestLocateAnnForAPAndRectSelectsLowestPageAndObject verifies deterministic fallback annotation selection.
func TestLocateAnnForAPAndRectSelectsLowestPageAndObject(t *testing.T) {
	r := types.NewRectangle(0, 0, 10, 10)
	ann := model.Annotation{SubType: model.AnnWidget, Rect: *r, APObjNr: 9}
	pageAnnots := map[int]model.PgAnnots{
		2: {model.AnnWidget: {Map: model.AnnotMap{20: ann}}},
		1: {model.AnnWidget: {Map: model.AnnotMap{12: ann, 8: ann}}},
	}
	d := types.Dict{"AP": *types.NewIndirectRef(9, 0)}

	for range 100 {
		got := locateAnnForAPAndRect(d, r, pageAnnots)
		if got == nil || got.ObjectNumber.Value() != 8 {
			t.Fatalf("got %v, want 8 0 R", got)
		}
	}
}
