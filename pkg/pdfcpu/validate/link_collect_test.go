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
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func externalReferenceContext(t *testing.T) *model.Context {
	t.Helper()
	ctx, err := model.NewContext(strings.NewReader(""), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	v := model.V17
	ctx.HeaderVersion = &v
	ctx.XRefTable.ValidationMode = model.ValidationStrict
	ctx.XRefTable.ValidateLinks = true
	ctx.CurPage = 3
	ctx.URIs = map[int]map[string]string{}
	return ctx
}

func setLinkTestTarget(xRefTable *model.XRefTable, page int, uri string, target linkTarget) {
	if xRefTable.URIs == nil {
		xRefTable.URIs = map[int]map[string]string{}
	}
	if xRefTable.URIs[page] == nil {
		xRefTable.URIs[page] = map[string]string{}
	}
	xRefTable.URIs[page][uri] = encodeLinkTarget(target)
}

func encodeLinkTestURIs(xRefTable *model.XRefTable) {
	for page, uris := range xRefTable.URIs {
		for uri := range uris {
			setLinkTestTarget(xRefTable, page, uri, linkTarget{kind: linkTargetURI, source: linkSourceURIAction})
		}
	}
}

func linkTestTarget(xRefTable *model.XRefTable, page int, uri string) (linkTarget, bool) {
	return decodeLinkTarget(xRefTable.URIs[page][uri])
}

func TestLinkTargetMetadataRoundTrip(t *testing.T) {
	want := linkTarget{kind: linkTargetURIReference, source: linkSourceMediaClipBaseURL}
	encoded := encodeLinkTarget(want)
	got, ok := decodeLinkTarget(encoded)
	if !ok || got != want {
		t.Fatalf("decoded target: got %+v, %t; want %+v, true", got, ok, want)
	}
}

func TestDecodeLinkTargetRejectsMalformedMetadata(t *testing.T) {
	for _, encoded := range []string{
		"", "b", linkTargetMetadataPrefix, linkTargetMetadataPrefix + "1",
		linkTargetMetadataPrefix + "0:1", linkTargetMetadataPrefix + "1:0",
		linkTargetMetadataPrefix + "5:1", linkTargetMetadataPrefix + "1:20",
		linkTargetMetadataPrefix + "01:1", linkTargetMetadataPrefix + "1:01",
		linkTargetMetadataPrefix + "1:1:1",
	} {
		if target, ok := decodeLinkTarget(encoded); ok {
			t.Fatalf("decoded malformed metadata %q as %+v", encoded, target)
		}
	}
}

func TestCollectLinkTargetReplacesFinalStatusAndPreservesPriority(t *testing.T) {
	xRefTable := &model.XRefTable{ValidateLinks: true, URIs: map[int]map[string]string{1: {"target": "b"}}}
	collectLinkTargetForPage(xRefTable, 1, "target", linkTargetFile, linkSourceRemoteGoTo)
	if got, ok := linkTestTarget(xRefTable, 1, "target"); !ok || got.kind != linkTargetFile {
		t.Fatalf("status replacement: got %+v, %t", got, ok)
	}

	collectLinkTargetForPage(xRefTable, 1, "target", linkTargetURIReference, linkSourceMediaClipBaseURL)
	collectLinkTargetForPage(xRefTable, 1, "target", linkTargetFile, linkSourceRemoteGoTo)
	if got, ok := linkTestTarget(xRefTable, 1, "target"); !ok ||
		got != (linkTarget{kind: linkTargetURIReference, source: linkSourceMediaClipBaseURL}) {
		t.Fatalf("priority result: got %+v, %t", got, ok)
	}
}
