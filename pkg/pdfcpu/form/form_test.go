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
