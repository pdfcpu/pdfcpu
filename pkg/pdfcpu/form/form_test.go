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

package form

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestFormFieldHelpersRejectRecursionDepth verifies form field helpers respect recursion limits.
func TestFormFieldHelpersRejectRecursionDepth(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	xRefTable := ctx.XRefTable
	maxDepth := xRefTable.MaxRecursionDepth()
	ir := *types.NewIndirectRef(1, 0)
	id, name := "", ""
	fields := types.Array{}

	_, err = fullyQualifiedFieldNameDepth(xRefTable, ir, fields, &id, &name, maxDepth+1, model.NewFormFieldVisit())
	if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}

	_, err = annotIndRefsDepth(xRefTable, fields, maxDepth+1, model.NewFormFieldVisit())
	if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}

	_, err = annotIndRefForFieldDepth(xRefTable, fields, "1.2", maxDepth+1, model.NewFormFieldVisit())
	if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}

	err = removeFormFieldsDepth(xRefTable, nil, &fields, maxDepth+1, model.NewFormFieldVisit())
	if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}
}

func TestExtractRadioButtonGroupOptionsSingleAppearanceStream(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	widget := types.Dict{
		"AP": types.Dict{
			"N": types.StreamDict{},
		},
	}
	radioButtonGroup := types.Dict{
		"Kids": types.Array{widget},
		"V":    types.Name("Choice"),
	}

	opts, explicit, err := extractRadioButtonGroupOptions(ctx.XRefTable, radioButtonGroup)
	if err != nil {
		t.Fatal(err)
	}
	if explicit {
		t.Fatal("single appearance stream reported explicit options")
	}
	if len(opts) != 0 {
		t.Fatalf("got options %v, want none", opts)
	}

	rbg, err := extractRadioButtonGroup(ctx.XRefTable, 1, radioButtonGroup, "1", "choice", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if rbg.Value != "Choice" {
		t.Fatalf("got value %q, want %q", rbg.Value, "Choice")
	}
}

func TestLocateAPNAppearanceStateDict(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	states := types.Dict{
		"Off":    types.StreamDict{},
		"Choice": types.StreamDict{},
	}
	widget := types.Dict{
		"AP": types.Dict{
			"N": states,
		},
	}

	got, err := locateAPN(ctx.XRefTable, widget)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(states) {
		t.Fatalf("got %d appearance states, want %d", len(got), len(states))
	}
}

func cyclicFormContext(t *testing.T) (*model.XRefTable, types.Array) {
	t.Helper()

	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	ir := *types.NewIndirectRef(1, 0)
	d := types.Dict{
		"Kids":   types.Array{ir},
		"Parent": ir,
		"T":      types.StringLiteral("1"),
	}
	if _, err := ctx.IndRefForObject(1, d); err != nil {
		t.Fatal(err)
	}

	return ctx.XRefTable, types.Array{ir}
}

// TestFormFieldHelpersRejectCycle verifies form field helpers reject cycles.
func TestFormFieldHelpersRejectCycle(t *testing.T) {
	xRefTable, fields := cyclicFormContext(t)
	ir := fields[0].(types.IndirectRef)
	id, name := "", ""

	_, err := fullyQualifiedFieldNameDepth(xRefTable, ir, fields, &id, &name, 0, model.NewFormFieldVisit())
	if !errors.Is(err, model.ErrFormFieldCycle) {
		t.Fatalf("got %v, want ErrFormFieldCycle", err)
	}

	_, err = annotIndRefsDepth(xRefTable, fields, 0, model.NewFormFieldVisit())
	if !errors.Is(err, model.ErrFormFieldCycle) {
		t.Fatalf("got %v, want ErrFormFieldCycle", err)
	}

	_, err = annotIndRefForFieldDepth(xRefTable, fields, "1.1.2", 0, model.NewFormFieldVisit())
	if !errors.Is(err, model.ErrFormFieldCycle) {
		t.Fatalf("got %v, want ErrFormFieldCycle", err)
	}

	indRefs := []types.IndirectRef{ir}
	err = removeFormFieldsDepth(xRefTable, &indRefs, &fields, 0, model.NewFormFieldVisit())
	if !errors.Is(err, model.ErrFormFieldCycle) {
		t.Fatalf("got %v, want ErrFormFieldCycle", err)
	}
}

func emptyFormContext(t *testing.T) *model.Context {
	t.Helper()
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestFormOperationsIdentifyAcroFormFieldsPhase(t *testing.T) {
	ctx := emptyFormContext(t)
	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "list", fn: func() error { _, _, err := FormFields(ctx); return err }},
		{name: "export", fn: func() error { _, _, err := ExportForm(ctx.XRefTable, "source.pdf"); return err }},
		{name: "remove", fn: func() error { _, err := RemoveFormFields(ctx, nil); return err }},
		{name: "reset", fn: func() error { _, err := ResetFormFields(ctx, nil); return err }},
		{name: "lock", fn: func() error { _, err := LockFormFields(ctx, nil); return err }},
		{name: "unlock", fn: func() error { _, err := UnlockFormFields(ctx, nil); return err }},
		{name: "fill", fn: func() error {
			fillDetails := func(string, string, FieldType, DataFormat) ([]string, bool, bool) {
				return nil, false, false
			}
			_, _, err := FillForm(ctx, fillDetails, nil, JSON)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), "AcroForm Fields: no form available") {
				t.Fatalf("expected AcroForm Fields context, got %v", err)
			}
		})
	}
}

func TestListFormFieldsAddsCollectionPhase(t *testing.T) {
	ctx := emptyFormContext(t)
	_, err := ListFormFields(ctx)
	if err == nil || !strings.Contains(err.Error(), "collect fields: AcroForm Fields") {
		t.Fatalf("expected collection phase, got %v", err)
	}
	if got := strings.Count(err.Error(), "collect fields"); got != 1 {
		t.Fatalf("expected collect fields once, got %d in %q", got, err.Error())
	}
}

func TestIsFieldCallersDoNotRepeatWidgetIdentity(t *testing.T) {
	ctx := emptyFormContext(t)
	irRef, err := ctx.XRefTable.IndRefForObject(7, types.Integer(1))
	if err != nil {
		t.Fatal(err)
	}
	ir := *irRef
	indRefs := []types.IndirectRef{ir}
	wAnnots := model.Annot{IndRefs: &indRefs}
	fillDetails := func(string, string, FieldType, DataFormat) ([]string, bool, bool) {
		return nil, false, false
	}

	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "collect", fn: func() error {
			fields := []Field{}
			return collectPageFields(ctx.XRefTable, wAnnots, nil, 1, &FieldMeta{}, &fields, 0)
		}},
		{name: "reset", fn: func() error {
			ok := false
			return resetPageFields(ctx, nil, wAnnots, nil, map[string]types.IndirectRef{}, &ok)
		}},
		{name: "lock", fn: func() error {
			ok := false
			return lockPageFields(ctx, nil, nil, wAnnots, map[string]types.IndirectRef{}, &ok)
		}},
		{name: "unlock", fn: func() error {
			ok := false
			return unlockPageFields(ctx.XRefTable, nil, nil, wAnnots, &ok)
		}},
		{name: "fill", fn: func() error {
			ok := false
			return fillWidgetAnnots(ctx, nil, map[types.IndirectRef]bool{}, wAnnots, JSON, map[string]types.IndirectRef{}, fillDetails, &ok)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), "resolve field: widget obj#7: dereference") {
				t.Fatalf("expected normalized widget context, got %v", err)
			}
			if got := strings.Count(err.Error(), "widget obj#7"); got != 1 {
				t.Fatalf("expected widget identity once, got %d in %q", got, err.Error())
			}
		})
	}
}

func TestCollectButtonFieldIdentityAppearsOnce(t *testing.T) {
	tests := []struct {
		name  string
		entry string
	}{
		{name: "default value", entry: "DV"},
		{name: "value", entry: "V"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := emptyFormContext(t)
			d := types.Dict{
				"FT":     types.Name("Btn"),
				"T":      types.StringLiteral("customer.name"),
				tt.entry: types.Integer(1),
			}
			ir, err := ctx.XRefTable.IndRefForObject(7, d)
			if err != nil {
				t.Fatal(err)
			}
			fields := types.Array{*ir}
			indRefs := []types.IndirectRef{*ir}
			wAnnots := model.Annot{IndRefs: &indRefs}
			collected := []Field{}

			err = collectPageFields(ctx.XRefTable, wAnnots, fields, 1, &FieldMeta{}, &collected, 0)
			want := fmt.Sprintf(`field 7: entry %q`, tt.entry)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %v", want, err)
			}
			if got := strings.Count(err.Error(), "field 7"); got != 1 {
				t.Fatalf("expected field identity once, got %d in %q", got, err.Error())
			}
		})
	}
}

func TestLockAndUnlockPageFieldsUseOneBasedKidIndexes(t *testing.T) {
	ctx := emptyFormContext(t)
	d := types.Dict{
		"FT":   types.Name("Tx"),
		"T":    types.StringLiteral("field"),
		"Kids": types.Array{types.Integer(1)},
	}
	ir, err := ctx.XRefTable.IndRefForObject(1, d)
	if err != nil {
		t.Fatal(err)
	}
	fields := types.Array{*ir}
	indRefs := []types.IndirectRef{*ir}
	wAnnots := model.Annot{IndRefs: &indRefs}

	tests := []struct {
		name string
		fn   func(*bool) error
	}{
		{name: "lock", fn: func(ok *bool) error {
			return lockPageFields(ctx, nil, fields, wAnnots, map[string]types.IndirectRef{}, ok)
		}},
		{name: "unlock", fn: func(ok *bool) error {
			return unlockPageFields(ctx.XRefTable, nil, fields, wAnnots, ok)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok := false
			err := tt.fn(&ok)
			if err == nil || !strings.Contains(err.Error(), "kid 1: dereference") {
				t.Fatalf("expected one-based kid context, got %v", err)
			}
		})
	}
}

func TestFormTreeWalkersRejectDirectObjectsWithoutPanic(t *testing.T) {
	ctx := emptyFormContext(t)
	fields := types.Array{types.Integer(1)}
	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "collect annotations", fn: func() error {
			_, err := annotIndRefs(ctx.XRefTable, fields)
			return err
		}},
		{name: "resolve field", fn: func() error {
			_, err := annotIndRefForField(ctx.XRefTable, fields, "1")
			return err
		}},
		{name: "remove field", fn: func() error {
			indRefs := []types.IndirectRef{*types.NewIndirectRef(1, 0)}
			return removeFormFields(ctx.XRefTable, &indRefs, &fields)
		}},
		{name: "resolve page annotations", fn: func() error {
			_, err := fieldsForAnnots(ctx.XRefTable, fields, nil)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), "expected indirect reference") {
				t.Fatalf("expected indirect reference error, got %v", err)
			}
		})
	}
}
