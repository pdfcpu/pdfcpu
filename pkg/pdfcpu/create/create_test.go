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

package create

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/primitives"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func useMissingGlobalFontDirectory(t *testing.T) {
	t.Helper()
	originalDir := font.UserFontDir
	font.UserFontDir = filepath.Join(t.TempDir(), "missing")
	if err := font.ReloadUserFonts(t.Context()); err == nil {
		t.Fatal("expected missing global font directory error")
	}
	t.Cleanup(func() {
		font.UserFontDir = originalDir
		if err := font.ReloadUserFonts(context.WithoutCancel(t.Context())); err != nil {
			t.Errorf("restore global font directory: %v", err)
		}
	})
}

type failingJSONReader struct {
	err error
}

func (r failingJSONReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

func newCreateTestContext(t *testing.T) *model.Context {
	t.Helper()

	ctx, err := pdfcpu.CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestEnsureFontIndRefUsesStatelessRepository(t *testing.T) {
	useMissingGlobalFontDirectory(t)
	indRef := types.NewIndirectRef(7, 0)
	xRefTable := &model.XRefTable{Conf: model.NewStatelessConfiguration()}
	fontResource := model.FontResource{
		Res:      model.Resource{IndRef: indRef},
		FontFile: indRef,
	}

	got, err := ensureFontIndRef(t.Context(), xRefTable, "Demo", fontResource, model.FontMap{"Demo": fontResource})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != *indRef {
		t.Fatalf("expected existing font reference %v, got %v", indRef, got)
	}
}

const createTestBlankPageJSON = `{
	"pages": {
		"1": {
			"content": {}
		}
	}
}`

func TestFromJSONBlankPageIncludesEmptyResources(t *testing.T) {
	ctx := newCreateTestContext(t)
	if err := FromJSON(t.Context(), ctx, strings.NewReader(createTestBlankPageJSON)); err != nil {
		t.Fatal(err)
	}

	pageDict, _, _, err := ctx.PageDict(t.Context(), 1, false)
	if err != nil {
		t.Fatal(err)
	}
	resources, found := pageDict.Find("Resources")
	if !found {
		t.Fatal("page dict: missing Resources")
	}
	if d, ok := resources.(types.Dict); !ok || len(d) != 0 {
		t.Fatalf("page dict: expected empty Resources dict, got %T %v", resources, resources)
	}
}

func TestFromJSONBoundaryErrors(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *model.Context
		rd      io.Reader
		wantErr error
	}{
		{
			name:    "missing context",
			wantErr: model.ErrMissingPDFContext,
		},
		{
			name:    "missing xref table",
			ctx:     &model.Context{},
			wantErr: model.ErrMissingXRefTable,
		},
		{
			name:    "missing JSON reader",
			ctx:     newCreateTestContext(t),
			wantErr: ErrMissingJSONReader,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FromJSON(t.Context(), tt.ctx, tt.rd)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestFromJSONReadErrorIncludesPhaseContext(t *testing.T) {
	wantErr := errors.New("read failed")

	err := FromJSON(t.Context(), newCreateTestContext(t), failingJSONReader{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "read JSON") {
		t.Fatalf("expected read JSON context, got %q", err.Error())
	}
}

func TestFromJSONParseErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "invalid JSON",
			in:   "{",
			want: "parse JSON: invalid JSON encoding detected",
		},
		{
			name: "missing pages",
			in:   "{}",
			want: "parse JSON: validate JSON model: please supply \"pages\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FromJSON(t.Context(), newCreateTestContext(t), strings.NewReader(tt.in))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
		})
	}
}

func TestFromJSONRenderPagesErrorIncludesPrimitiveContext(t *testing.T) {
	const input = `{
		"pages": {
			"1": {
				"content": {
					"text": [
						{"name": "$missing"}
					]
				}
			}
		}
	}`

	err := FromJSON(t.Context(), newCreateTestContext(t), strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error")
	}

	for _, want := range []string{
		"render pages",
		"page 1",
		"content",
		"text boxes",
		"text 1",
		"unknown named text missing",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestFromJSONUpdatePageTreeErrorIncludesPageContext(t *testing.T) {
	ctx := newCreateTestContext(t)
	if err := FromJSON(t.Context(), ctx, strings.NewReader(createTestBlankPageJSON)); err != nil {
		t.Fatal(err)
	}

	pageDict, _, _, err := ctx.PageDict(t.Context(), 1, false)
	if err != nil {
		t.Fatal(err)
	}
	pageDict["Contents"] = types.Name("broken")

	err = FromJSON(t.Context(), ctx, strings.NewReader(createTestBlankPageJSON))
	if err == nil {
		t.Fatal("expected error")
	}

	for _, want := range []string{
		"update page tree",
		"page 1",
		"append content",
		"corrupt page",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestFromJSONUpdateFormErrorIncludesDefaultResourceContext(t *testing.T) {
	ctx := newCreateTestContext(t)
	form := types.Dict{
		"Fields": types.Array{},
		"DR":     types.Dict{},
	}
	ctx.Form = form
	ctx.RootDict["AcroForm"] = form

	pdf := &primitives.PDF{
		HasForm: true,
		FormFonts: map[string]*primitives.FormFont{
			"F0": {
				Name: "Helvetica",
				Size: 12,
			},
		},
		Optimize: &model.OptimizationContext{
			FormFontObjects: map[int]*model.FontObject{},
		},
		XRefTable: ctx.XRefTable,
	}

	err := handleForm(t.Context(), ctx, pdf, types.Array{}, model.FontMap{})
	if err == nil {
		t.Fatal("expected error")
	}

	for _, want := range []string{
		"form fields",
		"default resources",
		"missing font dict",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestFromJSONContentValidateErrorIncludesPhaseContext(t *testing.T) {
	const input = `{
		"pages": {
			"1": {
				"content": {
					"bgCol": "bogus"
				}
			}
		}
	}`

	err := FromJSON(t.Context(), newCreateTestContext(t), strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error")
	}

	for _, want := range []string{
		"parse JSON",
		"validate JSON model",
		"page 1: validate",
		"content",
		"background color",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestFromJSONValidationContext(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "malformed page number includes JSON model context",
			in: `{
				"pages": {
					"bogus": {
						"content": {}
					}
				}
			}`,
			want: []string{
				"parse JSON",
				"validate JSON model",
				"invalid page number: bogus",
			},
		},
		{
			name: "invalid page content includes page context",
			in: `{
				"pages": {
					"2": {
						"content": {
							"bgCol": "bogus"
						}
					}
				}
			}`,
			want: []string{
				"parse JSON",
				"validate JSON model",
				"page 2: validate",
				"content",
				"background color",
			},
		},
		{
			name: "missing content includes page context",
			in: `{
				"pages": {
					"3": {}
				}
			}`,
			want: []string{
				"parse JSON",
				"validate JSON model",
				"page 3: validate",
				"please supply page \"content\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FromJSON(t.Context(), newCreateTestContext(t), strings.NewReader(tt.in))
			if err == nil {
				t.Fatal("expected error")
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected %q in %q", want, err.Error())
				}
			}
		})
	}
}
