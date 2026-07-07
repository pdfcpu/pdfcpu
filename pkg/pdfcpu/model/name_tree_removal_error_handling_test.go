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

package model

import (
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func multiLevelRemovalTree(value types.Object, childDict, parentDict types.Dict) *Node {
	return &Node{
		Kmin: "target",
		Kmax: "z",
		D:    parentDict,
		Kids: []*Node{
			{
				Kmin:  "target",
				Kmax:  "target",
				D:     childDict,
				Names: []entry{{k: "target", v: value}},
			},
			{
				Kmin:  "z",
				Kmax:  "z",
				D:     types.NewDict(),
				Names: []entry{{k: "z", v: types.StringLiteral("value")}},
			},
		},
	}
}

func requireNameTreeRemovalError(t *testing.T, err error, want ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	for _, s := range want {
		if !strings.Contains(err.Error(), s) {
			t.Fatalf("expected %q in %q", s, err)
		}
	}
}

func TestRemoveAttachmentIdentifiesValueGraphDeletionInMultiLevelTree(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	xRefTable.Names["EmbeddedFiles"] = multiLevelRemovalTree(
		*types.NewIndirectRef(91, 0),
		types.NewDict(),
		types.NewDict(),
	)
	ctx := &Context{XRefTable: xRefTable}

	_, err := ctx.removeAttachment("target")

	requireNameTreeRemovalError(
		t,
		err,
		`remove attachment "target"`,
		`name tree key "target"`,
		"delete value graph",
		"obj #91",
	)
}

func TestRemoveIdentifiesChildNodeDeletionInMultiLevelTree(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	tree := multiLevelRemovalTree(
		types.StringLiteral("value"),
		types.Dict{"Broken": *types.NewIndirectRef(92, 0)},
		types.NewDict(),
	)

	_, _, err := tree.Remove(xRefTable, "target")

	requireNameTreeRemovalError(t, err, `name tree key "target"`, "delete child node", "obj #92")
}

func TestRemoveIdentifiesParentNodeDeletionInMultiLevelTree(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	tree := multiLevelRemovalTree(
		types.StringLiteral("value"),
		types.NewDict(),
		types.Dict{"Broken": *types.NewIndirectRef(93, 0)},
	)

	_, _, err := tree.Remove(xRefTable, "target")

	requireNameTreeRemovalError(t, err, `name tree key "target"`, "delete parent node", "obj #93")
}
