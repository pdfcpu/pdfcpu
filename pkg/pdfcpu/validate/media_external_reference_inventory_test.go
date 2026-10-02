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

func TestURLSNameTreeExternalReferenceInventory(t *testing.T) {
	ctx := externalReferenceContext(t)
	treeKey := "https://example.invalid/captured/page.html"
	sourceURL := "https://origin.example.invalid/page.html"
	aliasURL := "https://alias.example.invalid/page.html"
	commandURL := "https://capture.example.invalid/page.html"
	ctx.XRefTable.Table[11] = model.NewXRefTableEntryGen0(types.Dict{"URL": types.StringLiteral(commandURL)})
	contentSet := types.Dict{
		"Type": types.Name("SpiderContentSet"),
		"S":    types.Name("SPS"),
		"ID":   types.StringLiteral("content-id"),
		"O":    types.Array{*types.NewIndirectRef(10, 0)},
		"SI": types.Array{
			types.Dict{"AU": types.StringLiteral(sourceURL)},
			types.Dict{
				"AU": types.Dict{"U": types.StringLiteral(aliasURL)},
				"C":  *types.NewIndirectRef(11, 0),
			},
		},
	}
	tree := types.Dict{"Names": types.Array{types.StringLiteral(treeKey), contentSet}}
	first, last, _, err := validateNameTree(t.Context(), ctx.XRefTable, "URLS", tree, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if first != treeKey || last != treeKey {
		t.Fatalf("URLS bounds: got %q..%q, want %q", first, last, treeKey)
	}
	want := map[string]linkTarget{
		treeKey:    {kind: linkTargetURI, source: linkSourceWebCaptureURL},
		sourceURL:  {kind: linkTargetURI, source: linkSourceWebCaptureSource},
		aliasURL:   {kind: linkTargetURI, source: linkSourceWebCaptureSource},
		commandURL: {kind: linkTargetURI, source: linkSourceWebCaptureCommand},
	}
	for target, wantTarget := range want {
		got, found := inventoryTargetAtPage(ctx, 0, target)
		if !found || got != wantTarget {
			t.Fatalf("URLS target %q: got %+v, found=%t, want %+v", target, got, found, wantTarget)
		}
	}
	tr := &successfulLinkTransport{}
	failed, err := checkLinks(t.Context(), ctx.XRefTable, http.Client{Transport: tr}, []int{0})
	if err != nil {
		t.Fatal(err)
	}
	if failed || tr.calls != len(want) {
		t.Fatalf("URLS checks: failed=%t, transport calls=%d, want %d", failed, tr.calls, len(want))
	}
	for target := range want {
		if status := ctx.URIs[0][target]; status != "" {
			t.Fatalf("URLS target %q: got status %q, want success", target, status)
		}
	}
}
