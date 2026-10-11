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

package model

import (
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestConsolidateResourcesOmitsMissingCategoryReference(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	resources := types.Dict{
		"Font":       types.Dict{},
		"Properties": *types.NewIndirectRef(9, 0),
	}
	attrs := &InheritedPageAttrs{}
	if err := xRefTable.consolidateResources(resources, attrs); err != nil {
		t.Fatal(err)
	}
	if _, found := attrs.Resources["Properties"]; found {
		t.Fatalf("missing resource category survived consolidation: %v", attrs.Resources["Properties"])
	}
	if _, found := attrs.Resources["Font"]; !found {
		t.Fatal("valid resource category was removed")
	}
}

func TestConsolidateResourceSubDictDereferencesCategory(t *testing.T) {
	tests := []struct {
		name         string
		object       types.Object
		required     bool
		wantMissing  bool
		wantTypeErr  bool
		wantRetained bool
	}{
		{name: "missing required", required: true, wantMissing: true},
		{name: "missing unused"},
		{name: "wrong type required", object: types.Integer(1), required: true, wantTypeErr: true, wantRetained: true},
		{name: "wrong type unused", object: types.Integer(1)},
		{name: "valid dictionary", object: types.Dict{"P1": types.Integer(1)}, required: true, wantRetained: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			xRefTable := newXRefTable(NewDefaultConfiguration())
			xRefTable.ValidationMode = ValidationStrict
			if tc.object != nil {
				xRefTable.Table[9] = NewXRefTableEntryGen0(tc.object)
			}
			d := types.Dict{"Properties": *types.NewIndirectRef(9, 0)}
			prn := NewPageResourceNames()
			if tc.required {
				prn["Properties"]["P1"] = true
			}
			err := xRefTable.consolidateResourceSubDict(d, "Properties", prn, 1)
			if tc.wantMissing && (err == nil || !strings.Contains(err.Error(), "missing required Properties resource dictionary")) {
				t.Fatalf("expected missing resource error, got %v", err)
			}
			if tc.wantTypeErr && !errors.Is(err, ErrExpectedDict) {
				t.Fatalf("expected dictionary type error, got %v", err)
			}
			if !tc.wantMissing && !tc.wantTypeErr && err != nil {
				t.Fatal(err)
			}
			_, found := d["Properties"]
			if found != tc.wantRetained {
				t.Fatalf("resource category retained: got %t, want %t", found, tc.wantRetained)
			}
		})
	}
}
