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
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type formJSONErrorWriter struct {
	err error
}

func (w formJSONErrorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

type formJSONShortWriter struct{}

func (formJSONShortWriter) Write(p []byte) (int, error) {
	return len(p) - 1, nil
}

// TestFieldMapSkipsEmptyImageValue verifies empty CSV image fields do not create image boxes.
func TestFieldMapSkipsEmptyImageValue(t *testing.T) {
	fieldNames := []string{"@img(page:1, pos:40 350, w:290, h:200)"}
	formRecord := []string{""}

	_, images, _, err := FieldMap(fieldNames, formRecord)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 0 {
		t.Fatalf("image map size = %d, want 0", len(images))
	}
}

func TestFormNameEntriesRejectWrongTypesWithoutPanic(t *testing.T) {
	ctx := emptyFormContext(t)
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "export checkbox value",
			fn: func() error {
				_, err := extractCheckBox(ctx.XRefTable, 1, types.Dict{"V": types.Integer(1)}, "7", "field", "", false)
				return err
			},
			want: `checkbox 7: entry "V": expected name`,
		},
		{
			name: "collect checkbox value",
			fn: func() error {
				return collectBtn(ctx.XRefTable, types.Dict{"V": types.Integer(1)}, &Field{ID: "7"}, &FieldMeta{})
			},
			want: `entry "V": expected name`,
		},
		{
			name: "reset checkbox default",
			fn: func() error {
				return resetBtn(ctx.XRefTable, types.Dict{"DV": types.Integer(1)})
			},
			want: `entry "DV": expected name`,
		},
		{
			name: "fill checkbox value",
			fn: func() error {
				fillDetails := func(string, string, FieldType, DataFormat) ([]string, bool, bool) {
					return []string{"true"}, false, true
				}
				return fillCheckBox(ctx, types.Dict{"V": types.Integer(1)}, "7", "field", false, JSON, fillDetails, new(bool))
			},
			want: `checkbox 7: entry "V": expected name`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestDictionaryEntryDecodeErrorsIncludeContext(t *testing.T) {
	ctx := emptyFormContext(t)
	badName := types.Name("#")
	badString := types.StringLiteral("\xfe\xff\xd8\x00")
	tests := []struct {
		name  string
		fn    func() error
		entry string
		phase string
	}{
		{name: "radio value", fn: func() error {
			return collectRadioButtonGroup(ctx.XRefTable, types.Dict{"V": badName}, &Field{}, &FieldMeta{})
		}, entry: "entry V", phase: "decode name"},
		{name: "button default", fn: func() error {
			return collectBtn(ctx.XRefTable, types.Dict{"DV": badName}, &Field{}, &FieldMeta{})
		}, entry: "entry DV", phase: "decode name"},
		{name: "reset button default", fn: func() error {
			return resetBtn(ctx.XRefTable, types.Dict{"DV": badName})
		}, entry: "entry DV", phase: "decode name"},
		{name: "reset text default", fn: func() error {
			return resetTx(ctx, types.Dict{"DV": badString}, map[string]types.IndirectRef{})
		}, entry: "entry DV", phase: "decode string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			want := tt.entry + ": " + tt.phase
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %v", want, err)
			}
			if got := strings.Count(err.Error(), tt.entry); got != 1 {
				t.Fatalf("expected exactly one %q context, got %d in %q", tt.entry, got, err)
			}
			if got := strings.Count(err.Error(), tt.phase); got != 1 {
				t.Fatalf("expected exactly one %q phase, got %d in %q", tt.phase, got, err)
			}
		})
	}
}

func TestResetTextDefaultDereferenceErrorIncludesContext(t *testing.T) {
	ctx := emptyFormContext(t)
	lazy := types.NewLazyObjectStreamObject(types.NewObjectStreamDict(), 1, -1, nil)
	ctx.XRefTable.Table[7] = model.NewXRefTableEntryGen0(lazy)
	d := types.Dict{"DV": *types.NewIndirectRef(7, 0)}

	err := resetTx(ctx, d, map[string]types.IndirectRef{})
	if err == nil || !strings.Contains(err.Error(), "entry DV: dereference") {
		t.Fatalf("expected default dereference context, got %v", err)
	}
	if got := strings.Count(err.Error(), "entry DV"); got != 1 {
		t.Fatalf("expected exactly one entry DV context, got %d in %q", got, err)
	}
	if got := strings.Count(err.Error(), "dereference"); got != 1 {
		t.Fatalf("expected exactly one dereference phase, got %d in %q", got, err)
	}
}

func TestCacheResourceDictRejectsWrongTypeWithoutPanic(t *testing.T) {
	err := cacheResourceDict(types.Dict{"Font": types.Integer(1)}, "Font", 3, map[int]types.Dict{})
	if err == nil || !strings.Contains(err.Error(), "page 3: Font resources: expected dictionary") {
		t.Fatalf("expected resource dictionary error, got %v", err)
	}
}

func TestFillFormRejectsInvalidCallbacksAndFormats(t *testing.T) {
	ctx := emptyFormContext(t)
	if _, _, err := FillForm(ctx, nil, nil, JSON); err == nil || !strings.Contains(err.Error(), "missing fill details") {
		t.Fatalf("expected missing fill details error, got %v", err)
	}
	fillDetails := func(string, string, FieldType, DataFormat) ([]string, bool, bool) {
		return nil, false, false
	}
	if _, _, err := FillForm(ctx, fillDetails, nil, DataFormat(99)); err == nil || !strings.Contains(err.Error(), "unsupported data format") {
		t.Fatalf("expected unsupported data format error, got %v", err)
	}
}

func TestFillDetailsHandlesNilForm(t *testing.T) {
	fillDetails := FillDetails(nil, nil)
	if values, _, found := fillDetails("1", "field", FTText, JSON); found || values != nil {
		t.Fatalf("expected missing field, got values=%v found=%t", values, found)
	}
}

func TestFormLookupsSkipNilFields(t *testing.T) {
	f := Form{
		TextFields:        []*TextField{nil},
		DateFields:        []*DateField{nil},
		CheckBoxes:        []*CheckBox{nil},
		RadioButtonGroups: []*RadioButtonGroup{nil},
		ComboBoxes:        []*ComboBox{nil},
		ListBoxes:         []*ListBox{nil},
	}
	if _, _, found := f.textFieldValueAndLock("1", "field"); found {
		t.Fatal("unexpected text field match")
	}
	if _, _, found := f.dateFieldValueAndLock("1", "field"); found {
		t.Fatal("unexpected date field match")
	}
	if _, _, found := f.checkBoxValueAndLock("1", "field"); found {
		t.Fatal("unexpected checkbox match")
	}
	if _, _, found := f.radioButtonGroupValueAndLock("1", "field"); found {
		t.Fatal("unexpected radio-button group match")
	}
	if _, _, found := f.comboBoxValueAndLock("1", "field"); found {
		t.Fatal("unexpected combo box match")
	}
	if _, _, found := f.listBoxValuesAndLock("1", "field"); found {
		t.Fatal("unexpected list box match")
	}
}

func TestFormStringArraysDereferenceIndirectValues(t *testing.T) {
	ctx := emptyFormContext(t)
	ir, err := ctx.IndRefForObject(1, types.Array{types.StringLiteral("one"), types.StringLiteral("two")})
	if err != nil {
		t.Fatal(err)
	}
	values, err := parseStringLiteralArray(ctx.XRefTable, types.Dict{"V": *ir}, "V")
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0] != "one" || values[1] != "two" {
		t.Fatalf("unexpected values: %v", values)
	}
}

func TestInheritedValueDereferencesIndirectString(t *testing.T) {
	ctx := emptyFormContext(t)
	ir, err := ctx.IndRefForObject(1, types.StringLiteral("value"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := getV(ctx.XRefTable, types.Dict{"V": *ir})
	if err != nil {
		t.Fatal(err)
	}
	if value != "value" {
		t.Fatalf("got %q, want value", value)
	}
}

func TestResolveOptionRejectsInvalidExplicitIndex(t *testing.T) {
	for _, value := range []string{"-1", "2", "not-an-index"} {
		t.Run(value, func(t *testing.T) {
			if _, err := resolveOption(value, []string{"one", "two"}, true); err == nil {
				t.Fatalf("expected invalid option index error for %q", value)
			}
		})
	}
}

func TestRadioButtonOptionsAreUnique(t *testing.T) {
	ctx := emptyFormContext(t)
	appearance := types.Dict{"Off": types.Dict{}, "Yes": types.Dict{}}
	kid := types.Dict{"AP": types.Dict{"N": appearance}}
	opts, _, err := extractRadioButtonGroupOptions(ctx.XRefTable, types.Dict{"Kids": types.Array{kid, kid}})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 1 || opts[0] != "Yes" {
		t.Fatalf("unexpected options: %v", opts)
	}
}

func TestChildWidgetHelpersAddKidDereferenceContext(t *testing.T) {
	ctx := emptyFormContext(t)
	badKids := types.Array{types.Integer(1)}
	fillDetails := func(string, string, FieldType, DataFormat) ([]string, bool, bool) {
		return []string{"true"}, false, true
	}
	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "collect radio options", fn: func() error {
			_, err := collectRadioButtonGroupOptions(ctx.XRefTable, types.Dict{"Kids": badKids})
			return err
		}},
		{name: "reset radio", fn: func() error {
			return resetBtn(ctx.XRefTable, types.Dict{"Kids": badKids})
		}},
		{name: "reset text", fn: func() error {
			return resetTx(ctx, types.Dict{"V": types.StringLiteral("old"), "Kids": badKids}, map[string]types.IndirectRef{})
		}},
		{name: "fill checkbox", fn: func() error {
			return fillCheckBox(ctx, types.Dict{"V": types.Name("Off"), "Kids": badKids}, "7", "field", false, JSON, fillDetails, new(bool))
		}},
		{name: "fill date", fn: func() error {
			return fillDateField(ctx, types.Dict{"Kids": badKids}, "7", "field", "", false, JSON, map[string]types.IndirectRef{}, fillDetails, new(bool))
		}},
		{name: "fill text", fn: func() error {
			return fillTextField(ctx, types.Dict{"Kids": badKids}, "7", "field", "", false, JSON, map[string]types.IndirectRef{}, fillDetails, nil, new(bool))
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), "kid 1: dereference") {
				t.Fatalf("expected child dereference context, got %v", err)
			}
			if got := strings.Count(err.Error(), "kid 1"); got != 1 {
				t.Fatalf("expected child identity once, got %d in %q", got, err.Error())
			}
		})
	}
}

func TestRootFillAppearanceErrorsIncludePhase(t *testing.T) {
	ctx := emptyFormContext(t)
	invalidAppearance := func() types.Dict {
		return types.Dict{"Rect": types.Array{types.StringLiteral("invalid"), types.Integer(0), types.Integer(10), types.Integer(10)}}
	}
	value := func(v string, lock bool) func(string, string, FieldType, DataFormat) ([]string, bool, bool) {
		return func(string, string, FieldType, DataFormat) ([]string, bool, bool) {
			return []string{v}, lock, true
		}
	}
	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "checkbox", fn: func() error {
			return fillCheckBox(ctx, types.Dict{"V": types.Name("Off"), "AS": types.Name("Off")}, "7", "field", false, JSON, value("true", false), new(bool))
		}},
		{name: "combo box", fn: func() error {
			return fillComboBox(ctx, invalidAppearance(), "7", "field", nil, false, JSON, map[string]types.IndirectRef{}, value("new", true), new(bool))
		}},
		{name: "list box", fn: func() error {
			return fillListBox(ctx, invalidAppearance(), "7", "field", []string{"one"}, false, JSON, map[string]types.IndirectRef{}, value("one", false), nil, new(bool))
		}},
		{name: "date field", fn: func() error {
			return fillDateField(ctx, invalidAppearance(), "7", "field", "", false, JSON, map[string]types.IndirectRef{}, value("2026-07-14", false), new(bool))
		}},
		{name: "text field", fn: func() error {
			return fillTextField(ctx, invalidAppearance(), "7", "field", "", false, JSON, map[string]types.IndirectRef{}, value("new", false), nil, new(bool))
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), "appearance:") {
				t.Fatalf("expected root appearance phase, got %v", err)
			}
			if got := strings.Count(err.Error(), "appearance:"); got != 1 {
				t.Fatalf("expected appearance phase once, got %d in %q", got, err.Error())
			}
		})
	}
}

func TestRootResetAppearanceErrorsIncludePhase(t *testing.T) {
	ctx := emptyFormContext(t)
	ff := types.Integer(0)
	invalidAppearance := func() types.Dict {
		return types.Dict{"Rect": types.Array{types.StringLiteral("invalid"), types.Integer(0), types.Integer(10), types.Integer(10)}}
	}
	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "list box", fn: func() error {
			d := invalidAppearance()
			d["Ff"] = ff
			return resetCh(ctx, d, map[string]types.IndirectRef{})
		}},
		{name: "text field", fn: func() error {
			d := invalidAppearance()
			d["V"] = types.StringLiteral("old")
			return resetTx(ctx, d, map[string]types.IndirectRef{})
		}},
		{name: "date field", fn: func() error {
			d := invalidAppearance()
			d["DV"] = types.StringLiteral("2026-07-14")
			return resetTx(ctx, d, map[string]types.IndirectRef{})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), "appearance:") {
				t.Fatalf("expected root appearance phase, got %v", err)
			}
			if got := strings.Count(err.Error(), "appearance:"); got != 1 {
				t.Fatalf("expected appearance phase once, got %d in %q", got, err.Error())
			}
		})
	}
}

func TestRadioChildHelpersAddAppearancePhases(t *testing.T) {
	ctx := emptyFormContext(t)
	tests := []struct {
		name string
		kid  types.Dict
		want string
	}{
		{name: "appearance", kid: types.Dict{}, want: "kid 1: appearance"},
		{name: "decode appearance state", kid: types.Dict{"AP": types.Dict{"N": types.Dict{"#": types.Dict{}}}}, want: "kid 1: decode appearance state"},
	}
	helpers := []struct {
		name string
		fn   func(types.Dict) error
	}{
		{name: "collect options", fn: func(d types.Dict) error {
			_, err := collectRadioButtonGroupOptions(ctx.XRefTable, d)
			return err
		}},
		{name: "reset", fn: func(d types.Dict) error { return resetBtn(ctx.XRefTable, d) }},
	}

	for _, helper := range helpers {
		for _, tt := range tests {
			t.Run(helper.name+"/"+tt.name, func(t *testing.T) {
				err := helper.fn(types.Dict{"Kids": types.Array{tt.kid}})
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("expected %q, got %v", tt.want, err)
				}
			})
		}
	}
}

func TestFillRadioButtonsAddsChildIndexAndPhase(t *testing.T) {
	ctx := emptyFormContext(t)
	validKid := types.Dict{"AP": types.Dict{"N": types.Dict{"Off": types.Dict{}}}}
	tests := []struct {
		name string
		kid  types.Object
		want string
	}{
		{name: "dereference", kid: types.Integer(1), want: "kid 2: dereference"},
		{name: "appearance", kid: types.Dict{}, want: "kid 2: appearance"},
		{name: "decode appearance state", kid: types.Dict{"AP": types.Dict{"N": types.Dict{"#": types.Dict{}}}}, want: "kid 2: decode appearance state"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := types.Dict{"Kids": types.Array{validKid, tt.kid}}
			err := fillRadioButtons(ctx, d, "Yes", types.Name("Yes"))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestDateFormatActionRejectsMissingQuoteWithoutPanic(t *testing.T) {
	d := types.Dict{"AA": types.Dict{"F": types.Dict{"JS": types.StringLiteral(`AFDate_FormatEx("broken`)}}}
	_, err := dateFormatFromJSAction(d)
	if err == nil || !strings.Contains(err.Error(), "missing closing quote") {
		t.Fatalf("expected malformed date format error, got %v", err)
	}
}

func TestDateStringEntryDereferencesValues(t *testing.T) {
	ctx := emptyFormContext(t)

	for _, key := range []string{"V", "DV"} {
		t.Run(key, func(t *testing.T) {
			ir, err := ctx.IndRefForObject(1, types.StringLiteral("2026-07-14"))
			if err != nil {
				t.Fatal(err)
			}

			s, found, err := dateStringEntry(ctx.XRefTable, types.Dict{key: *ir}, key)
			if err != nil {
				t.Fatal(err)
			}
			if !found || s != "2026-07-14" {
				t.Fatalf("got value %q, found=%t", s, found)
			}
		})
	}
}

func TestDateStringEntryErrorsIncludeContext(t *testing.T) {
	ctx := emptyFormContext(t)
	badString := types.StringLiteral("\xfe\xff\xd8\x00")
	lazy := types.NewLazyObjectStreamObject(types.NewObjectStreamDict(), 1, -1, nil)
	ctx.XRefTable.Table[7] = model.NewXRefTableEntryGen0(lazy)
	badRef := *types.NewIndirectRef(7, 0)
	tests := []struct {
		name string
		key  string
		obj  types.Object
		want string
	}{
		{name: "V decode", key: "V", obj: badString, want: "entry V: decode string"},
		{name: "DV decode", key: "DV", obj: badString, want: "entry DV: decode string"},
		{name: "V dereference", key: "V", obj: badRef, want: "entry V: dereference"},
		{name: "DV dereference", key: "DV", obj: badRef, want: "entry DV: dereference"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := dateStringEntry(ctx.XRefTable, types.Dict{tt.key: tt.obj}, tt.key)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
			if got := strings.Count(err.Error(), tt.want); got != 1 {
				t.Fatalf("expected context once, got %d in %q", got, err)
			}
		})
	}
}

func TestExportFormJSONRejectsNilWriter(t *testing.T) {
	ctx := emptyFormContext(t)
	if _, err := ExportFormJSON(ctx.XRefTable, "source.pdf", nil); !errors.Is(err, ErrMissingJSONWriter) {
		t.Fatalf("expected %v, got %v", ErrMissingJSONWriter, err)
	}
}

func TestExportFormJSONReturnsFalseWhenNothingWasExported(t *testing.T) {
	ctx := emptyFormContext(t)
	ctx.XRefTable.Form = types.Dict{"Fields": types.Array{types.Dict{}}}
	ok, err := ExportFormJSON(ctx.XRefTable, "source.pdf", io.Discard)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ok {
		t.Fatal("expected no exported form fields")
	}
}

func TestExportFormJSONPreservesCollectionFailure(t *testing.T) {
	wantErr := errors.New("collect form data")
	export := func(*model.XRefTable, string) (*FormGroup, bool, error) {
		return nil, false, wantErr
	}

	_, err := exportFormJSON(nil, "source.pdf", io.Discard, export, json.MarshalIndent)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "collect data") {
		t.Fatalf("expected collection context, got %v", err)
	}
}

func TestExportFormJSONPreservesEncodingFailure(t *testing.T) {
	wantErr := errors.New("encode form data")
	export := func(*model.XRefTable, string) (*FormGroup, bool, error) {
		return &FormGroup{}, true, nil
	}
	marshal := func(any, string, string) ([]byte, error) {
		return nil, wantErr
	}

	_, err := exportFormJSON(nil, "source.pdf", io.Discard, export, marshal)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "encode JSON") {
		t.Fatalf("expected encoding context, got %v", err)
	}
}

func TestExportFormJSONPreservesWriteFailures(t *testing.T) {
	wantErr := errors.New("write form data")
	export := func(*model.XRefTable, string) (*FormGroup, bool, error) {
		return &FormGroup{}, true, nil
	}

	tests := []struct {
		name    string
		writer  io.Writer
		wantErr error
	}{
		{name: "writer error", writer: formJSONErrorWriter{err: wantErr}, wantErr: wantErr},
		{name: "short write", writer: formJSONShortWriter{}, wantErr: io.ErrShortWrite},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := exportFormJSON(nil, "source.pdf", tt.writer, export, json.MarshalIndent)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), "write JSON") {
				t.Fatalf("expected write context, got %v", err)
			}
		})
	}
}

// TestFieldMapRejectsInvalidRecordsWithoutPanic verifies malformed CSV records return contextual errors.
func TestFieldMapRejectsInvalidRecordsWithoutPanic(t *testing.T) {
	tests := []struct {
		name        string
		fieldNames  []string
		formRecord  []string
		wantContext string
	}{
		{name: "short record", fieldNames: []string{"one", "two"}, formRecord: []string{"value"}, wantContext: "got 1 values, want 2"},
		{name: "empty field name", fieldNames: []string{""}, formRecord: []string{"value"}, wantContext: "column 1"},
		{name: "lock marker only", fieldNames: []string{"*"}, formRecord: []string{"value"}, wantContext: "column 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := FieldMap(tt.fieldNames, tt.formRecord)
			if err == nil || !strings.Contains(err.Error(), tt.wantContext) {
				t.Fatalf("expected %q, got %v", tt.wantContext, err)
			}
		})
	}
}
