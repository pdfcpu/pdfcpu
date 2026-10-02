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
	"net/http"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestReferenceXObjectExternalReferenceInventory(t *testing.T) {
	for _, tt := range []struct {
		name          string
		fileSpec      types.Object
		target        string
		kind          linkTargetKind
		category      linkResultCategory
		transportCall int
	}{
		{
			name:     "plain file",
			fileSpec: types.StringLiteral("reference.pdf"),
			target:   "reference.pdf",
			kind:     linkTargetFile,
			category: linkResultSkipped,
		},
		{
			name:          "URL",
			fileSpec:      urlSpecification("https://example.invalid/reference.pdf"),
			target:        "https://example.invalid/reference.pdf",
			kind:          linkTargetURI,
			category:      linkResultOK,
			transportCall: 1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := externalReferenceContext(t)
			d := types.Dict{"F": tt.fileSpec, "Page": types.Integer(0)}
			if err := validateReferenceDict(ctx.XRefTable, d); err != nil {
				t.Fatal(err)
			}
			want := linkTarget{kind: tt.kind, source: linkSourceReferenceXObject}
			if got, found := linkTestTarget(ctx.XRefTable, ctx.CurPage, tt.target); !found || got != want {
				t.Fatalf("Reference XObject target: got %+v, found=%t, want %+v", got, found, want)
			}
			tr := &successfulLinkTransport{}
			failed, err := checkLinks(t.Context(), ctx.XRefTable, http.Client{Transport: tr}, []int{ctx.CurPage})
			if err != nil {
				t.Fatal(err)
			}
			if failed != (tt.category != linkResultOK) || tr.calls != tt.transportCall {
				t.Fatalf("Reference XObject check: failed=%t, transport calls=%d", failed, tr.calls)
			}
			wantStatus := ""
			if tt.category == linkResultSkipped {
				wantStatus = "k"
			}
			if got := ctx.URIs[ctx.CurPage][tt.target]; got != wantStatus {
				t.Fatalf("Reference XObject status: got %q, want %q", got, wantStatus)
			}
		})
	}

	t.Run("embedded file", func(t *testing.T) {
		ctx := externalReferenceContext(t)
		fileName := "reference.pdf"
		ctx.XRefTable.Table[20] = model.NewXRefTableEntryGen0(types.StreamDict{
			Dict: types.Dict{"Type": types.Name("EmbeddedFile")},
		})
		fileSpec := types.Dict{
			"Type": types.Name("Filespec"),
			"F":    types.StringLiteral(fileName),
			"EF":   types.Dict{"F": *types.NewIndirectRef(20, 0)},
		}
		if err := validateReferenceDict(ctx.XRefTable, types.Dict{"F": fileSpec, "Page": types.Integer(0)}); err != nil {
			t.Fatal(err)
		}
		if target, found := linkTestTarget(ctx.XRefTable, ctx.CurPage, fileName); found {
			t.Fatalf("embedded Reference XObject unexpectedly collected as external: %+v", target)
		}
	})
}

func TestOPIExternalReferenceInventory(t *testing.T) {
	opiV13 := func(fileSpec types.Object) types.Dict {
		return types.Dict{
			"Version":  types.Float(1.3),
			"F":        fileSpec,
			"Size":     types.Array{types.Integer(100), types.Integer(100)},
			"CropRect": types.Array{types.Integer(0), types.Integer(0), types.Integer(100), types.Integer(100)},
			"Position": types.Array{
				types.Integer(0), types.Integer(0), types.Integer(100), types.Integer(0),
				types.Integer(100), types.Integer(100), types.Integer(0), types.Integer(100),
			},
		}
	}
	opiV20 := func(fileSpec types.Object) types.Dict {
		return types.Dict{"Version": types.Float(2.0), "F": fileSpec}
	}

	for _, tt := range []struct {
		name          string
		dict          types.Dict
		validate      func(*model.XRefTable, types.Dict) error
		target        string
		kind          linkTargetKind
		category      linkResultCategory
		transportCall int
	}{
		{
			name:     "1.3 plain file",
			dict:     opiV13(types.StringLiteral("proxy-13.tif")),
			validate: validateOPIDictV13,
			target:   "proxy-13.tif",
			kind:     linkTargetFile,
			category: linkResultSkipped,
		},
		{
			name:          "1.3 URL",
			dict:          opiV13(urlSpecification("https://example.invalid/proxy-13.tif")),
			validate:      validateOPIDictV13,
			target:        "https://example.invalid/proxy-13.tif",
			kind:          linkTargetURI,
			category:      linkResultOK,
			transportCall: 1,
		},
		{
			name:     "2.0 plain file",
			dict:     opiV20(types.StringLiteral("proxy-20.tif")),
			validate: validateOPIDictV20,
			target:   "proxy-20.tif",
			kind:     linkTargetFile,
			category: linkResultSkipped,
		},
		{
			name:          "2.0 URL",
			dict:          opiV20(urlSpecification("https://example.invalid/proxy-20.tif")),
			validate:      validateOPIDictV20,
			target:        "https://example.invalid/proxy-20.tif",
			kind:          linkTargetURI,
			category:      linkResultOK,
			transportCall: 1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := externalReferenceContext(t)
			if err := tt.validate(ctx.XRefTable, tt.dict); err != nil {
				t.Fatal(err)
			}
			want := linkTarget{kind: tt.kind, source: linkSourceOPI}
			if got, found := linkTestTarget(ctx.XRefTable, ctx.CurPage, tt.target); !found || got != want {
				t.Fatalf("OPI target: got %+v, found=%t, want %+v", got, found, want)
			}
			tr := &successfulLinkTransport{}
			failed, err := checkLinks(t.Context(), ctx.XRefTable, http.Client{Transport: tr}, []int{ctx.CurPage})
			if err != nil {
				t.Fatal(err)
			}
			if failed != (tt.category != linkResultOK) || tr.calls != tt.transportCall {
				t.Fatalf("OPI check: failed=%t, transport calls=%d", failed, tr.calls)
			}
			wantStatus := ""
			if tt.category == linkResultSkipped {
				wantStatus = "k"
			}
			if got := ctx.URIs[ctx.CurPage][tt.target]; got != wantStatus {
				t.Fatalf("OPI status: got %q, want %q", got, wantStatus)
			}
		})
	}

	for _, tt := range []struct {
		name     string
		dict     func(types.Object) types.Dict
		validate func(*model.XRefTable, types.Dict) error
	}{
		{name: "1.3 embedded file", dict: opiV13, validate: validateOPIDictV13},
		{name: "2.0 embedded file", dict: opiV20, validate: validateOPIDictV20},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := externalReferenceContext(t)
			fileName := "embedded-proxy.tif"
			ctx.XRefTable.Table[20] = model.NewXRefTableEntryGen0(types.StreamDict{
				Dict: types.Dict{"Type": types.Name("EmbeddedFile")},
			})
			fileSpec := types.Dict{
				"Type": types.Name("Filespec"),
				"F":    types.StringLiteral(fileName),
				"EF":   types.Dict{"F": *types.NewIndirectRef(20, 0)},
			}
			if err := tt.validate(ctx.XRefTable, tt.dict(fileSpec)); err != nil {
				t.Fatal(err)
			}
			if target, found := linkTestTarget(ctx.XRefTable, ctx.CurPage, fileName); found {
				t.Fatalf("embedded OPI target unexpectedly collected as external: %+v", target)
			}
		})
	}
}
