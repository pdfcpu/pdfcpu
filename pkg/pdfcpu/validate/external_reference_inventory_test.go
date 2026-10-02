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
