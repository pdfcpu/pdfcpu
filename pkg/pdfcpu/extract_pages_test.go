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
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestExtractPagesReturnsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := ExtractPages(ctx, nil, nil, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestAddPagesReturnsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := AddPages(ctx, nil, nil, nil, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func appendExtractionTestPage(t *testing.T, ctx *model.Context) types.IndirectRef {
	t.Helper()

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		t.Fatal(err)
	}
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		t.Fatal(err)
	}
	pageIndRef, err := ctx.EmptyPage(pagesIndRef, types.RectForFormat("A4"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := model.AppendPageTree(pageIndRef, 1, pagesDict); err != nil {
		t.Fatal(err)
	}
	ctx.PageCount++

	return *pageIndRef
}

func textFieldExtractionContext(t *testing.T) (*model.Context, types.IndirectRef, types.IndirectRef) {
	t.Helper()

	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	pageIndRef := appendExtractionTestPage(t, ctx)

	parent := types.Dict{
		"FT": types.Name("Tx"),
		"T":  types.StringLiteral("customer_name"),
	}
	parentIndRef, err := ctx.IndRefForNewObject(parent)
	if err != nil {
		t.Fatal(err)
	}
	widget := types.Dict{
		"P":       pageIndRef,
		"Parent":  *parentIndRef,
		"Rect":    types.NewNumberArray(10, 10, 110, 30),
		"Subtype": types.Name("Widget"),
		"Type":    types.Name("Annot"),
	}
	widgetIndRef, err := ctx.IndRefForNewObject(widget)
	if err != nil {
		t.Fatal(err)
	}
	parent["Kids"] = types.Array{*widgetIndRef}

	pageDict, err := ctx.DereferenceDict(pageIndRef)
	if err != nil {
		t.Fatal(err)
	}
	pageDict["Annots"] = types.Array{*widgetIndRef}
	form := types.Dict{"Fields": types.Array{*parentIndRef}}
	ctx.Form = form
	ctx.RootDict["AcroForm"] = form

	return ctx, *parentIndRef, *widgetIndRef
}

func multiPageTextFieldExtractionContext(t *testing.T) *model.Context {
	t.Helper()

	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	pageIndRefs := []types.IndirectRef{
		appendExtractionTestPage(t, ctx),
		appendExtractionTestPage(t, ctx),
	}
	parent := types.Dict{
		"FT": types.Name("Tx"),
		"T":  types.StringLiteral("customer_name"),
	}
	parentIndRef, err := ctx.IndRefForNewObject(parent)
	if err != nil {
		t.Fatal(err)
	}

	kids := types.Array{}
	for i, pageIndRef := range pageIndRefs {
		widget := types.Dict{
			"NM":      types.StringLiteral(fmt.Sprintf("page%d", i+1)),
			"P":       pageIndRef,
			"Parent":  *parentIndRef,
			"Rect":    types.NewNumberArray(10, 10, 110, 30),
			"Subtype": types.Name("Widget"),
			"Type":    types.Name("Annot"),
		}
		widgetIndRef, err := ctx.IndRefForNewObject(widget)
		if err != nil {
			t.Fatal(err)
		}
		kids = append(kids, *widgetIndRef)
		pageDict, err := ctx.DereferenceDict(pageIndRef)
		if err != nil {
			t.Fatal(err)
		}
		pageDict["Annots"] = types.Array{*widgetIndRef}
	}
	parent["Kids"] = kids
	form := types.Dict{"Fields": types.Array{*parentIndRef}}
	ctx.Form = form
	ctx.RootDict["AcroForm"] = form

	return ctx
}

func nestedButtonFieldExtractionContext(t *testing.T) (*model.Context, types.IndirectRef) {
	t.Helper()

	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	pageIndRef := appendExtractionTestPage(t, ctx)
	top := types.Dict{"T": types.StringLiteral("account")}
	topIndRef, err := ctx.IndRefForNewObject(top)
	if err != nil {
		t.Fatal(err)
	}
	button := types.Dict{
		"FT":     types.Name("Btn"),
		"Parent": *topIndRef,
		"T":      types.StringLiteral("choice"),
	}
	buttonIndRef, err := ctx.IndRefForNewObject(button)
	if err != nil {
		t.Fatal(err)
	}
	widget := types.Dict{
		"P":       pageIndRef,
		"Parent":  *buttonIndRef,
		"Rect":    types.NewNumberArray(10, 10, 30, 30),
		"Subtype": types.Name("Widget"),
		"Type":    types.Name("Annot"),
	}
	widgetIndRef, err := ctx.IndRefForNewObject(widget)
	if err != nil {
		t.Fatal(err)
	}
	top["Kids"] = types.Array{*buttonIndRef}
	button["Kids"] = types.Array{*widgetIndRef}

	pageDict, err := ctx.DereferenceDict(pageIndRef)
	if err != nil {
		t.Fatal(err)
	}
	pageDict["Annots"] = types.Array{*widgetIndRef}
	form := types.Dict{"Fields": types.Array{*topIndRef}}
	ctx.Form = form
	ctx.RootDict["AcroForm"] = form

	return ctx, *buttonIndRef
}

func combinedFieldWidgetExtractionContext(t *testing.T) *model.Context {
	t.Helper()

	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	pageIndRef := appendExtractionTestPage(t, ctx)
	field := types.Dict{
		"FT":      types.Name("Tx"),
		"P":       pageIndRef,
		"Rect":    types.NewNumberArray(10, 10, 110, 30),
		"Subtype": types.Name("Widget"),
		"T":       types.StringLiteral("standalone"),
		"Type":    types.Name("Annot"),
	}
	fieldIndRef, err := ctx.IndRefForNewObject(field)
	if err != nil {
		t.Fatal(err)
	}
	pageDict, err := ctx.DereferenceDict(pageIndRef)
	if err != nil {
		t.Fatal(err)
	}
	pageDict["Annots"] = types.Array{*fieldIndRef}
	form := types.Dict{"Fields": types.Array{*fieldIndRef}}
	ctx.Form = form
	ctx.RootDict["AcroForm"] = form

	return ctx
}

func checkboxExtractionContext(t *testing.T) *model.Context {
	t.Helper()

	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	pageIndRef := appendExtractionTestPage(t, ctx)
	field := types.Dict{
		"AS":      types.Name("Yes"),
		"FT":      types.Name("Btn"),
		"P":       pageIndRef,
		"Rect":    types.NewNumberArray(10, 10, 30, 30),
		"Subtype": types.Name("Widget"),
		"T":       types.StringLiteral("consent"),
		"Type":    types.Name("Annot"),
		"V":       types.Name("Yes"),
	}
	fieldIndRef, err := ctx.IndRefForNewObject(field)
	if err != nil {
		t.Fatal(err)
	}
	pageDict, err := ctx.DereferenceDict(pageIndRef)
	if err != nil {
		t.Fatal(err)
	}
	pageDict["Annots"] = types.Array{*fieldIndRef}
	form := types.Dict{"Fields": types.Array{*fieldIndRef}}
	ctx.Form = form
	ctx.RootDict["AcroForm"] = form

	return ctx
}

func radioButtonExtractionContext(t *testing.T) *model.Context {
	t.Helper()

	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	pageIndRef := appendExtractionTestPage(t, ctx)
	parent := types.Dict{
		"FT": types.Name("Btn"),
		"Ff": types.Integer(1 << 15),
		"T":  types.StringLiteral("payment"),
		"V":  types.Name("card1"),
	}
	parentIndRef, err := ctx.IndRefForNewObject(parent)
	if err != nil {
		t.Fatal(err)
	}
	kids := types.Array{}
	for i, state := range []string{"card1", "Off"} {
		widget := types.Dict{
			"AS":      types.Name(state),
			"P":       pageIndRef,
			"Parent":  *parentIndRef,
			"Rect":    types.NewNumberArray(float64(10+i*30), 10, float64(30+i*30), 30),
			"Subtype": types.Name("Widget"),
			"Type":    types.Name("Annot"),
		}
		widgetIndRef, err := ctx.IndRefForNewObject(widget)
		if err != nil {
			t.Fatal(err)
		}
		kids = append(kids, *widgetIndRef)
	}
	parent["Kids"] = kids
	pageDict, err := ctx.DereferenceDict(pageIndRef)
	if err != nil {
		t.Fatal(err)
	}
	pageDict["Annots"] = kids.Clone().(types.Array)
	form := types.Dict{"Fields": types.Array{*parentIndRef}}
	ctx.Form = form
	ctx.RootDict["AcroForm"] = form

	return ctx
}

func destinationFields(t *testing.T, ctx *model.Context) types.Array {
	t.Helper()

	form, err := ctx.DereferenceDict(ctx.RootDict["AcroForm"])
	if err != nil {
		t.Fatal(err)
	}
	fields, err := ctx.DereferenceArray(form["Fields"])
	if err != nil {
		t.Fatal(err)
	}
	return fields
}

func destinationField(t *testing.T, ctx *model.Context) (types.IndirectRef, types.Dict) {
	t.Helper()

	fields := destinationFields(t, ctx)
	if len(fields) != 1 {
		t.Fatalf("destination field count: got %d, want 1", len(fields))
	}
	indRef, ok := fields[0].(types.IndirectRef)
	if !ok {
		t.Fatalf("destination field: got %T, want indirect reference", fields[0])
	}
	d, err := ctx.DereferenceDict(indRef)
	if err != nil {
		t.Fatal(err)
	}
	return indRef, d
}

func destinationPage(t *testing.T, ctx *model.Context) types.Dict {
	t.Helper()

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		t.Fatal(err)
	}
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		t.Fatal(err)
	}
	pageKids, err := ctx.DereferenceArray(pagesDict["Kids"])
	if err != nil {
		t.Fatal(err)
	}
	if len(pageKids) != 1 {
		t.Fatalf("destination page count: got %d, want 1", len(pageKids))
	}
	pageIndRef, ok := pageKids[0].(types.IndirectRef)
	if !ok {
		t.Fatalf("destination page: got %T, want indirect reference", pageKids[0])
	}
	pageDict, err := ctx.DereferenceDict(pageIndRef)
	if err != nil {
		t.Fatal(err)
	}
	return pageDict
}

func singleFieldKid(t *testing.T, ctx *model.Context, d types.Dict, label string) (types.IndirectRef, types.Dict) {
	t.Helper()

	kids, err := ctx.DereferenceArray(d["Kids"])
	if err != nil {
		t.Fatal(err)
	}
	if len(kids) != 1 {
		t.Fatalf("%s kid count: got %d, want 1", label, len(kids))
	}
	indRef, ok := kids[0].(types.IndirectRef)
	if !ok {
		t.Fatalf("%s kid: got %T, want indirect reference", label, kids[0])
	}
	kid, err := ctx.DereferenceDict(indRef)
	if err != nil {
		t.Fatal(err)
	}

	return indRef, kid
}

func assertNestedButtonHierarchy(t *testing.T, ctx *model.Context) {
	t.Helper()

	topIndRef, top := destinationField(t, ctx)
	if name := top.StringEntry("T"); name == nil || *name != "account" {
		t.Fatalf("destination top-level name: got %v, want account", name)
	}
	buttonIndRef, button := singleFieldKid(t, ctx, top, "top-level field")
	if parent := button.IndirectRefEntry("Parent"); parent == nil || *parent != topIndRef {
		t.Fatalf("destination button parent: got %v, want %v", parent, topIndRef)
	}
	if ft := button.NameEntry("FT"); ft == nil || *ft != "Btn" {
		t.Fatalf("destination button type: got %v, want Btn", ft)
	}
	_, widget := singleFieldKid(t, ctx, button, "button field")
	if parent := widget.IndirectRefEntry("Parent"); parent == nil || *parent != buttonIndRef {
		t.Fatalf("destination widget parent: got %v, want %v", parent, buttonIndRef)
	}
	if subtype := widget.Subtype(); subtype == nil || *subtype != "Widget" {
		t.Fatalf("destination widget subtype: got %v, want Widget", subtype)
	}
}

func assertTextFieldHierarchy(t *testing.T, ctx *model.Context) types.IndirectRef {
	t.Helper()

	parentIndRef, parent := destinationField(t, ctx)
	if ft := parent.NameEntry("FT"); ft == nil || *ft != "Tx" {
		t.Fatalf("destination field type: got %v, want Tx", ft)
	}
	if name := parent.StringEntry("T"); name == nil || *name != "customer_name" {
		t.Fatalf("destination field name: got %v, want customer_name", name)
	}
	widgetIndRef, widget := singleFieldKid(t, ctx, parent, "text field")
	if parent := widget.IndirectRefEntry("Parent"); parent == nil || *parent != parentIndRef {
		t.Fatalf("destination widget parent: got %v, want %v", parent, parentIndRef)
	}

	return widgetIndRef
}

// TestExtractPagesPreservesTextFieldParent verifies extraction retains a separate parent field dictionary.
func TestExtractPagesPreservesTextFieldParent(t *testing.T) {
	ctxSrc, parentIndRef, widgetIndRef := textFieldExtractionContext(t)
	parentBefore, err := ctxSrc.DereferenceDict(parentIndRef)
	if err != nil {
		t.Fatal(err)
	}
	widgetBefore, err := ctxSrc.DereferenceDict(widgetIndRef)
	if err != nil {
		t.Fatal(err)
	}
	parentBefore = parentBefore.Clone().(types.Dict)
	widgetBefore = widgetBefore.Clone().(types.Dict)

	ctxDest, err := ExtractPages(t.Context(), ctxSrc, []int{1}, false)
	if err != nil {
		t.Fatal(err)
	}
	assertTextFieldHierarchy(t, ctxDest)

	parentAfter, err := ctxSrc.DereferenceDict(parentIndRef)
	if err != nil {
		t.Fatal(err)
	}
	widgetAfter, err := ctxSrc.DereferenceDict(widgetIndRef)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parentAfter, parentBefore) {
		t.Fatalf("source parent changed:\n got: %#v\nwant: %#v", parentAfter, parentBefore)
	}
	if !reflect.DeepEqual(widgetAfter, widgetBefore) {
		t.Fatalf("source widget changed:\n got: %#v\nwant: %#v", widgetAfter, widgetBefore)
	}
}

// TestExtractPagesPrunesUnselectedWidgets verifies extraction rebuilds Kids from widgets on selected pages only.
func TestExtractPagesPrunesUnselectedWidgets(t *testing.T) {
	ctxDest, err := ExtractPages(t.Context(), multiPageTextFieldExtractionContext(t), []int{1}, false)
	if err != nil {
		t.Fatal(err)
	}
	widgetIndRef := assertTextFieldHierarchy(t, ctxDest)
	widget, err := ctxDest.DereferenceDict(widgetIndRef)
	if err != nil {
		t.Fatal(err)
	}
	if name := widget.StringEntry("NM"); name == nil || *name != "page1" {
		t.Fatalf("destination widget name: got %v, want page1", name)
	}
}

// TestExtractPagesPreservesNestedButtonField verifies nested ancestors survive without mutating the source hierarchy.
func TestExtractPagesPreservesNestedButtonField(t *testing.T) {
	ctxSrc, buttonIndRef := nestedButtonFieldExtractionContext(t)
	buttonBefore, err := ctxSrc.DereferenceDict(buttonIndRef)
	if err != nil {
		t.Fatal(err)
	}
	buttonBefore = buttonBefore.Clone().(types.Dict)

	ctxDest, err := ExtractPages(t.Context(), ctxSrc, []int{1}, false)
	if err != nil {
		t.Fatal(err)
	}
	assertNestedButtonHierarchy(t, ctxDest)

	buttonAfter, err := ctxSrc.DereferenceDict(buttonIndRef)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(buttonAfter, buttonBefore) {
		t.Fatalf("source button field changed:\n got: %#v\nwant: %#v", buttonAfter, buttonBefore)
	}
}

// TestExtractPagesPreservesCombinedFieldWidget verifies a terminal field/widget remains a top-level form field.
func TestExtractPagesPreservesCombinedFieldWidget(t *testing.T) {
	ctxDest, err := ExtractPages(t.Context(), combinedFieldWidgetExtractionContext(t), []int{1}, false)
	if err != nil {
		t.Fatal(err)
	}
	fieldIndRef, field := destinationField(t, ctxDest)
	if ft := field.NameEntry("FT"); ft == nil || *ft != "Tx" {
		t.Fatalf("destination field type: got %v, want Tx", ft)
	}
	if name := field.StringEntry("T"); name == nil || *name != "standalone" {
		t.Fatalf("destination field name: got %v, want standalone", name)
	}
	if parent := field.IndirectRefEntry("Parent"); parent != nil {
		t.Fatalf("destination combined field/widget parent: got %v, want none", parent)
	}
	pageDict := destinationPage(t, ctxDest)
	annots, err := ctxDest.DereferenceArray(pageDict["Annots"])
	if err != nil {
		t.Fatal(err)
	}
	if len(annots) != 1 || annots[0] != fieldIndRef {
		t.Fatalf("destination annotations: got %v, want [%v]", annots, fieldIndRef)
	}
}

// TestExtractPagesPreservesCombinedCheckbox verifies extraction retains a combined checkbox field/widget.
func TestExtractPagesPreservesCombinedCheckbox(t *testing.T) {
	ctxDest, err := ExtractPages(t.Context(), checkboxExtractionContext(t), []int{1}, false)
	if err != nil {
		t.Fatal(err)
	}
	_, field := destinationField(t, ctxDest)
	if ft := field.NameEntry("FT"); ft == nil || *ft != "Btn" {
		t.Fatalf("destination checkbox type: got %v, want Btn", ft)
	}
	if name := field.StringEntry("T"); name == nil || *name != "consent" {
		t.Fatalf("destination checkbox name: got %v, want consent", name)
	}
	if value := field.NameEntry("V"); value == nil || *value != "Yes" {
		t.Fatalf("destination checkbox value: got %v, want Yes", value)
	}
	if state := field.NameEntry("AS"); state == nil || *state != "Yes" {
		t.Fatalf("destination checkbox appearance state: got %v, want Yes", state)
	}
}

// TestExtractPagesPreservesRadioButtonGroup verifies extraction retains radio flags, values, kids, and parent links.
func TestExtractPagesPreservesRadioButtonGroup(t *testing.T) {
	ctxDest, err := ExtractPages(t.Context(), radioButtonExtractionContext(t), []int{1}, false)
	if err != nil {
		t.Fatal(err)
	}
	parentIndRef, parent := destinationField(t, ctxDest)
	if flags := parent.IntEntry("Ff"); flags == nil || *flags != 1<<15 {
		t.Fatalf("destination radio flags: got %v, want %d", flags, 1<<15)
	}
	if value := parent.NameEntry("V"); value == nil || *value != "card1" {
		t.Fatalf("destination radio value: got %v, want card1", value)
	}
	kids, err := ctxDest.DereferenceArray(parent["Kids"])
	if err != nil {
		t.Fatal(err)
	}
	if len(kids) != 2 {
		t.Fatalf("destination radio kid count: got %d, want 2", len(kids))
	}
	for i, wantState := range []string{"card1", "Off"} {
		kidIndRef, ok := kids[i].(types.IndirectRef)
		if !ok {
			t.Fatalf("destination radio kid %d: got %T, want indirect reference", i, kids[i])
		}
		kid, err := ctxDest.DereferenceDict(kidIndRef)
		if err != nil {
			t.Fatal(err)
		}
		if parent := kid.IndirectRefEntry("Parent"); parent == nil || *parent != parentIndRef {
			t.Fatalf("destination radio kid %d parent: got %v, want %v", i, parent, parentIndRef)
		}
		if state := kid.NameEntry("AS"); state == nil || *state != wantState {
			t.Fatalf("destination radio kid %d state: got %v, want %s", i, state, wantState)
		}
	}
}

// TestExtractPagesRejectsInvalidInput verifies stable extraction errors.
func TestExtractPagesRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *model.Context
		pageNrs []int
		wantErr error
	}{
		{
			name:    "missing context",
			pageNrs: []int{1},
			wantErr: ErrMissingPDFContext,
		},
		{
			name:    "missing page numbers",
			ctx:     &model.Context{XRefTable: &model.XRefTable{}},
			wantErr: ErrMissingPageNumbers,
		},
		{
			name:    "missing xref table",
			ctx:     &model.Context{},
			pageNrs: []int{1},
			wantErr: ErrMissingXRefTable,
		},
		{
			name:    "page number below lower bound",
			ctx:     &model.Context{XRefTable: &model.XRefTable{PageCount: 2}},
			pageNrs: []int{0},
			wantErr: ErrInvalidPageNumber,
		},
		{
			name:    "page number above upper bound",
			ctx:     &model.Context{XRefTable: &model.XRefTable{PageCount: 2}},
			pageNrs: []int{3},
			wantErr: ErrInvalidPageNumber,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractPages(t.Context(), tt.ctx, tt.pageNrs, false)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestExtractPagesAddPagesErrorsIncludePageContext(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{PageCount: 1}}

	_, err := ExtractPages(t.Context(), ctx, []int{1}, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "extract pages [1]") {
		t.Fatalf("expected extract pages context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "add pages") {
		t.Fatalf("expected add pages context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "page 1") {
		t.Fatalf("expected page context, got %q", err.Error())
	}
}

func TestAddPagesRejectsMissingContexts(t *testing.T) {
	tests := []struct {
		name string
		src  *model.Context
		dest *model.Context
		want string
	}{
		{
			name: "missing source",
			dest: &model.Context{XRefTable: &model.XRefTable{}},
			want: "add pages: source context",
		},
		{
			name: "missing destination",
			src:  &model.Context{XRefTable: &model.XRefTable{}},
			want: "add pages: destination context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddPages(t.Context(), tt.src, tt.dest, []int{1}, false)
			if !errors.Is(err, ErrMissingPDFContext) {
				t.Fatalf("expected %v, got %v", ErrMissingPDFContext, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestAddPagesRejectsMissingDestinationPageTree verifies missing roots retain destination page-tree error context.
func TestAddPagesRejectsMissingDestinationPageTree(t *testing.T) {
	src, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	dest, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	dest.RootDict.Delete("Pages")

	err = AddPages(t.Context(), src, dest, []int{1}, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "add pages: read destination page tree: missing pages root") {
		t.Fatalf("expected missing destination page tree context, got %q", err.Error())
	}
}

func TestAddPagesInternalGuardsPreventPanics(t *testing.T) {
	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "missing page tree dict",
			fn: func() error {
				return addPages(
					t.Context(), ctx, ctx, []int{1}, false, *types.NewIndirectRef(1, 0), nil,
					&types.Array{}, &types.Array{}, map[int]int{},
				)
			},
			want: "missing destination page tree dict",
		},
		{
			name: "missing source fields",
			fn: func() error {
				return addPages(
					t.Context(), ctx, ctx, []int{1}, false, *types.NewIndirectRef(1, 0), types.Dict{}, nil,
					&types.Array{}, map[int]int{},
				)
			},
			want: "missing source form fields",
		},
		{
			name: "missing migration map",
			fn: func() error {
				return addPages(
					t.Context(), ctx, ctx, []int{1}, false, *types.NewIndirectRef(1, 0), types.Dict{},
					&types.Array{}, &types.Array{}, nil,
				)
			},
			want: "missing migration map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("expected error, got panic: %v", r)
				}
			}()

			err := tt.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestMigratePageDictErrorsIncludeEntryContext(t *testing.T) {
	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}

	d := types.Dict{
		"Annots": types.Integer(1),
	}

	err = migratePageDict(t.Context(), d, *types.NewIndirectRef(1, 0), ctx, ctx, map[int]int{}, newFormFieldSelection())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "page dict entry Annots") {
		t.Fatalf("expected page dict entry context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "migrate annotations") {
		t.Fatalf("expected migrate annotations context, got %q", err.Error())
	}
}

func TestMigrateNamedDestsErrorsIncludeKeyContext(t *testing.T) {
	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}

	n := &model.Node{}
	n.AppendToNames("badDest", types.Integer(1))

	err = migrateNamedDests(t.Context(), ctx, n, map[int]int{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "named destination \"badDest\"") {
		t.Fatalf("expected named destination key context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "process named destinations") {
		t.Fatalf("expected process named destinations context, got %q", err.Error())
	}
}
