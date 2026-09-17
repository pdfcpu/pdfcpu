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
	"errors"
	"log"
	"strings"
	"testing"

	pdfcpuLog "github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type pageResourcesTestCase struct {
	name               string
	parentResources    types.Object
	parentHasResources bool
	pageResources      types.Object
	pageHasResources   bool
	contents           bool
}

func validatePageResourcesTestCase(t *testing.T, tc pageResourcesTestCase, mode int) error {
	t.Helper()
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = mode
	ctx, err := model.NewContext(strings.NewReader(""), conf)
	if err != nil {
		t.Fatal(err)
	}
	version := model.V17
	ctx.HeaderVersion = &version
	page := types.Dict{
		"Type":   types.Name("Page"),
		"Parent": *types.NewIndirectRef(2, 0),
	}
	if tc.pageHasResources {
		page["Resources"] = tc.pageResources
	}
	if tc.contents {
		page["Contents"] = types.StreamDict{Dict: types.NewDict()}
	}
	ctx.Table[3] = model.NewXRefTableEntryGen0(page)

	pages := types.Dict{
		"Type":     types.Name("Pages"),
		"Count":    types.Integer(1),
		"Kids":     types.Array{*types.NewIndirectRef(3, 0)},
		"MediaBox": types.Array{types.Integer(0), types.Integer(0), types.Integer(100), types.Integer(100)},
	}
	if tc.parentHasResources {
		pages["Resources"] = tc.parentResources
	}
	pageCount := 0
	return validatePagesDict(t.Context(), ctx.XRefTable, pages, 2, false, nil, &pageCount)
}

func TestPageResourcesStrictRequiresLocalOrInheritedDictionary(t *testing.T) {
	for _, tc := range []struct {
		pageResourcesTestCase
		wantErr bool
	}{
		{pageResourcesTestCase: pageResourcesTestCase{name: "local", pageHasResources: true, pageResources: types.Dict{}}},
		{pageResourcesTestCase: pageResourcesTestCase{name: "inherited", parentHasResources: true, parentResources: types.Dict{}}},
		{pageResourcesTestCase: pageResourcesTestCase{
			name: "local overrides inherited null", parentHasResources: true, pageHasResources: true, pageResources: types.Dict{},
		}},
		{pageResourcesTestCase: pageResourcesTestCase{name: "missing", contents: true}, wantErr: true},
		{pageResourcesTestCase: pageResourcesTestCase{name: "missing without contents"}, wantErr: true},
		{pageResourcesTestCase: pageResourcesTestCase{name: "local null", pageHasResources: true}, wantErr: true},
		{pageResourcesTestCase: pageResourcesTestCase{name: "inherited null", parentHasResources: true}, wantErr: true},
		{pageResourcesTestCase: pageResourcesTestCase{
			name:               "local null overrides inherited",
			parentHasResources: true,
			parentResources:    types.Dict{},
			pageHasResources:   true,
		}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePageResourcesTestCase(t, tc.pageResourcesTestCase, model.ValidationStrict)
			if !tc.wantErr {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), `page dict: missing required entry "Resources"`) {
				t.Fatalf("got %v", err)
			}
			var validationErr *model.ValidationError
			if !errors.As(err, &validationErr) || validationErr.ObjectNumber() != 3 {
				t.Fatalf("attribution: %v", err)
			}
		})
	}
}

func TestPageResourcesRelaxedDigestsMissingDictionary(t *testing.T) {
	for _, tc := range []struct {
		pageResourcesTestCase
		wantDiagnostic bool
	}{
		{pageResourcesTestCase: pageResourcesTestCase{name: "local", pageHasResources: true, pageResources: types.Dict{}}},
		{pageResourcesTestCase: pageResourcesTestCase{name: "inherited", parentHasResources: true, parentResources: types.Dict{}}},
		{pageResourcesTestCase: pageResourcesTestCase{
			name: "local overrides inherited null", parentHasResources: true, pageHasResources: true, pageResources: types.Dict{},
		}},
		{pageResourcesTestCase: pageResourcesTestCase{name: "missing", contents: true}, wantDiagnostic: true},
		{pageResourcesTestCase: pageResourcesTestCase{name: "missing without contents"}, wantDiagnostic: true},
		{pageResourcesTestCase: pageResourcesTestCase{name: "local null", pageHasResources: true}, wantDiagnostic: true},
		{pageResourcesTestCase: pageResourcesTestCase{name: "inherited null", parentHasResources: true}, wantDiagnostic: true},
		{pageResourcesTestCase: pageResourcesTestCase{
			name:               "local null overrides inherited",
			parentHasResources: true,
			parentResources:    types.Dict{},
			pageHasResources:   true,
		}, wantDiagnostic: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			pdfcpuLog.SetCLILogger(log.New(&buf, "", 0))
			defer pdfcpuLog.SetCLILogger(nil)
			if err := validatePageResourcesTestCase(t, tc.pageResourcesTestCase, model.ValidationRelaxed); err != nil {
				t.Fatal(err)
			}
			want := ""
			if tc.wantDiagnostic {
				want = "pdfcpu digested: spec violation (obj#:3): page dict: missing required entry \"Resources\"\n"
			}
			if got := buf.String(); got != want {
				t.Fatalf("diagnostic: got %q, want %q", got, want)
			}
		})
	}
}

func TestPageResourcesRejectsInvalidType(t *testing.T) {
	for _, tc := range []struct {
		name       string
		testCase   pageResourcesTestCase
		ownerObjNr int
	}{
		{"local", pageResourcesTestCase{pageHasResources: true, pageResources: types.Integer(7)}, 3},
		{"inherited", pageResourcesTestCase{parentHasResources: true, parentResources: types.Integer(7)}, 2},
	} {
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			t.Run(tc.name, func(t *testing.T) {
				err := validatePageResourcesTestCase(t, tc.testCase, mode)
				if err == nil || !strings.Contains(err.Error(), "expected types.Dict, got types.Integer") {
					t.Fatalf("got %v", err)
				}
				var validationErr *model.ValidationError
				if !errors.As(err, &validationErr) || validationErr.ObjectNumber() != tc.ownerObjNr {
					t.Fatalf("attribution: %v", err)
				}
			})
		}
	}
}
