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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/primitives"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func buttonWidget(on string) types.Dict {
	return types.Dict{
		"AS": types.Name("Off"),
		"AP": types.Dict{"N": types.Dict{"Off": types.Dict{}, on: types.Dict{}}},
	}
}

func inheritedRadioField(t *testing.T) (*model.Context, types.Dict, types.Dict) {
	t.Helper()
	ctx := emptyFormContext(t)
	parent, err := ctx.IndRefForObject(1, types.Dict{"Ff": types.Integer(primitives.FieldRadio)})
	if err != nil {
		t.Fatal(err)
	}
	widget := buttonWidget("A")
	field := types.Dict{"Parent": *parent, "V": types.Name("Off"), "Kids": types.Array{widget}}
	return ctx, field, widget
}

// TestButtonKindUsesRadioFlag checks that the widget count does not determine a button field's kind.
func TestButtonKindUsesRadioFlag(t *testing.T) {
	ctx := emptyFormContext(t)
	for _, tc := range []struct {
		name string
		d    types.Dict
		want FieldType
		val  string
	}{
		{"two-widget checkbox", types.Dict{
			"V": types.Name("Yes"), "Kids": types.Array{buttonWidget("Yes"), buttonWidget("Yes")},
		}, FTCheckBox, "Yes"},
		{"one-widget radio", types.Dict{
			"Ff": types.Integer(primitives.FieldRadio), "V": types.Name("A"),
			"Kids": types.Array{buttonWidget("A")},
		}, FTRadioButtonGroup, "A"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &Field{}
			if err := collectBtn(ctx.XRefTable, tc.d, f, &FieldMeta{}); err != nil {
				t.Fatal(err)
			}
			if f.Typ != tc.want || f.V != tc.val {
				t.Fatalf("got kind %v, value %q; want kind %v, value %q", f.Typ, f.V, tc.want, tc.val)
			}
		})
	}
}

// TestButtonExportUsesRadioFlag checks that form export uses the field flag for both widget counts.
func TestButtonExportUsesRadioFlag(t *testing.T) {
	ctx := emptyFormContext(t)
	checkbox := types.Dict{"V": types.Name("Yes"), "Kids": types.Array{buttonWidget("Yes"), buttonWidget("Yes")}}
	radioFlag := types.Integer(primitives.FieldRadio)
	radio := types.Dict{"Ff": radioFlag, "V": types.Name("A"), "Kids": types.Array{buttonWidget("A")}}
	form := &Form{}
	ok := false
	if err := exportBtn(ctx.XRefTable, 1, form, checkbox, "cb", "cb", "", false, &ok, nil); err != nil {
		t.Fatal(err)
	}
	if err := exportBtn(ctx.XRefTable, 1, form, radio, "rb", "rb", "", false, &ok, &radioFlag); err != nil {
		t.Fatal(err)
	}
	if len(form.CheckBoxes) != 1 || len(form.RadioButtonGroups) != 1 ||
		form.CheckBoxes[0].ID != "cb" || form.RadioButtonGroups[0].ID != "rb" {
		t.Fatalf("wrong exported button kinds: checkboxes=%+v radios=%+v", form.CheckBoxes, form.RadioButtonGroups)
	}
}

// TestButtonFillUsesRadioFlag checks that all checkbox widgets and a single radio widget receive the right state.
func TestButtonFillUsesRadioFlag(t *testing.T) {
	ctx := emptyFormContext(t)
	first, second := buttonWidget("Yes"), buttonWidget("Yes")
	checkbox := types.Dict{"V": types.Name("Off"), "Kids": types.Array{first, second}}
	radioWidget := buttonWidget("A")
	radio := types.Dict{
		"Ff": types.Integer(primitives.FieldRadio), "V": types.Name("Off"),
		"Kids": types.Array{radioWidget},
	}
	fillDetails := func(_, _ string, fieldType FieldType, _ DataFormat) ([]string, bool, bool) {
		switch fieldType {
		case FTCheckBox:
			return []string{"true"}, false, true
		case FTRadioButtonGroup:
			return []string{"A"}, false, true
		}
		return nil, false, false
	}
	for _, tc := range []struct {
		name string
		d    types.Dict
		want types.Name
	}{
		{"checkbox", checkbox, "Yes"},
		{"radio", radio, "A"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ok := false
			if err := fillBtn(ctx, tc.d, tc.name, tc.name, false, JSON, fillDetails, &ok); err != nil {
				t.Fatal(err)
			}
			if !ok || tc.d["V"] != tc.want {
				t.Fatalf("got changed=%t, V=%v; want changed and V=%s", ok, tc.d["V"], tc.want)
			}
		})
	}
	for _, widget := range []types.Dict{first, second} {
		if widget["AS"] != types.Name("Yes") {
			t.Fatalf("checkbox widget AS = %v, want Yes", widget["AS"])
		}
	}
	if radioWidget["AS"] != types.Name("A") {
		t.Fatalf("radio widget AS = %v, want A", radioWidget["AS"])
	}
	ok := false
	if err := fillBtn(ctx, checkbox, "checkbox", "checkbox", false, JSON, fillCheckBoxValue(false), &ok); err != nil {
		t.Fatal(err)
	}
	if !ok || checkbox["V"] != types.Name("Off") || first["AS"] != types.Name("Off") || second["AS"] != types.Name("Off") {
		t.Fatalf("checkbox off state: changed=%t, V=%v, AS=%v/%v", ok, checkbox["V"], first["AS"], second["AS"])
	}
}

// TestButtonResetUpdatesEveryWidget covers checkboxes and radio groups with nonstandard widget counts.
func TestButtonResetUpdatesEveryWidget(t *testing.T) {
	ctx := emptyFormContext(t)
	for _, tc := range []struct {
		name string
		d    types.Dict
		want types.Name
	}{
		{"two-widget checkbox", types.Dict{
			"DV": types.Name("Yes"), "V": types.Name("Off"),
			"Kids": types.Array{buttonWidget("Yes"), buttonWidget("Yes")},
		}, "Yes"},
		{"one-widget radio", types.Dict{
			"Ff": types.Integer(primitives.FieldRadio), "DV": types.Name("A"), "V": types.Name("Off"),
			"Kids": types.Array{buttonWidget("A")},
		}, "A"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := resetBtn(ctx.XRefTable, tc.d); err != nil {
				t.Fatal(err)
			}
			if tc.d["V"] != tc.want {
				t.Fatalf("field V = %v, want %s", tc.d["V"], tc.want)
			}
			for i, o := range tc.d.ArrayEntry("Kids") {
				widget := o.(types.Dict)
				if widget["AS"] != tc.want {
					t.Fatalf("kid %d AS = %v, want %s", i+1, widget["AS"], tc.want)
				}
			}
		})
	}
}

// TestInheritedRadioFlagAcrossFormOperations verifies that button kind follows an inherited Ff value.
func TestInheritedRadioFlagAcrossFormOperations(t *testing.T) {
	ctx, field, widget := inheritedRadioField(t)
	ff, _, err := formFieldFlags(ctx.XRefTable, field)
	if err != nil {
		t.Fatal(err)
	}
	if !isRadioButtonGroup(ff) {
		t.Fatal("inherited Radio flag was lost")
	}
	f := &Field{}
	if err := collectBtn(ctx.XRefTable, field, f, &FieldMeta{}); err != nil {
		t.Fatal(err)
	}
	if f.Typ != FTRadioButtonGroup {
		t.Fatalf("listed kind = %v, want radio group", f.Typ)
	}
	form := &Form{}
	ok := false
	if err := exportBtn(ctx.XRefTable, 1, form, field, "rb", "rb", "", false, &ok, ff); err != nil {
		t.Fatal(err)
	}
	if len(form.RadioButtonGroups) != 1 || len(form.CheckBoxes) != 0 {
		t.Fatalf("exported button kind: radios=%d checkboxes=%d", len(form.RadioButtonGroups), len(form.CheckBoxes))
	}
	fillDetails := func(_, _ string, fieldType FieldType, _ DataFormat) ([]string, bool, bool) {
		return []string{"A"}, false, fieldType == FTRadioButtonGroup
	}
	if err := fillBtn(ctx, field, "rb", "rb", false, JSON, fillDetails, &ok); err != nil {
		t.Fatal(err)
	}
	if field["V"] != types.Name("A") || widget["AS"] != types.Name("A") {
		t.Fatalf("filled state: V=%v, AS=%v; want A", field["V"], widget["AS"])
	}
}

// TestInheritedRadioFlagSurvivesLocking verifies that a local ReadOnly flag retains the inherited button kind.
func TestInheritedRadioFlagSurvivesLocking(t *testing.T) {
	ctx, field, _ := inheritedRadioField(t)
	if err := lockFormField(ctx.XRefTable, field); err != nil {
		t.Fatal(err)
	}
	ff, _, err := ctx.DereferenceIntegerEntry(field, "Ff")
	if err != nil {
		t.Fatal(err)
	}
	flags := primitives.FieldFlags(ff.Value())
	if flags&primitives.FieldRadio == 0 || flags&primitives.FieldReadOnly == 0 {
		t.Fatalf("locked field flags = %d, want Radio and ReadOnly", flags)
	}
	if err := unlockFormField(ctx.XRefTable, field); err != nil {
		t.Fatal(err)
	}
	ff, _, err = ctx.DereferenceIntegerEntry(field, "Ff")
	if err != nil {
		t.Fatal(err)
	}
	flags = primitives.FieldFlags(ff.Value())
	if flags&primitives.FieldRadio == 0 || flags&primitives.FieldReadOnly != 0 {
		t.Fatalf("unlocked field flags = %d, want Radio without ReadOnly", flags)
	}
}

// TestInheritedRadioFlagRejectsParentCycle keeps flag lookup bounded on malformed field trees.
func TestInheritedRadioFlagRejectsParentCycle(t *testing.T) {
	ctx := emptyFormContext(t)
	parent := types.Dict{}
	ref, err := ctx.IndRefForObject(1, parent)
	if err != nil {
		t.Fatal(err)
	}
	parent["Parent"] = *ref
	field := types.Dict{"Parent": *ref}
	if _, err := inheritedFieldFlags(ctx.XRefTable, field); !errors.Is(err, model.ErrFormFieldCycle) {
		t.Fatalf("got %v, want form field cycle", err)
	}
}
