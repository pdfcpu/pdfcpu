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

func locateNameTreeXRefTable() *XRefTable {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	size := 1
	xRefTable.Size = &size
	xRefTable.Table[0] = NewFreeHeadXRefTableEntry()
	return xRefTable
}

func requireLocateNameTreeError(t *testing.T, err error, want ...string) {
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

func TestLocateNameTreeCatalogContext(t *testing.T) {
	xRefTable := locateNameTreeXRefTable()

	err := xRefTable.LocateNameTree("JavaScript", false)

	requireLocateNameTreeError(t, err, `name tree "JavaScript"`, "catalog", "missing root dict")
}

func TestLocateNameTreeGuardsNilCatalog(t *testing.T) {
	xRefTable := locateNameTreeXRefTable()
	xRefTable.Root = types.NewIndirectRef(42, 0)

	err := xRefTable.LocateNameTree("JavaScript", false)

	requireLocateNameTreeError(t, err, `name tree "JavaScript"`, "catalog", "missing dictionary")
}

func TestLocateNameTreeNamesDictionaryCreationContext(t *testing.T) {
	xRefTable := locateNameTreeXRefTable()
	xRefTable.RootDict = types.NewDict()
	xRefTable.Table[0].Free = false

	err := xRefTable.LocateNameTree("JavaScript", true)

	requireLocateNameTreeError(t, err, `name tree "JavaScript"`, "create Names dictionary", "object #0 found, but not free")
}

func TestLocateNameTreeNamesDictionaryDereferenceContext(t *testing.T) {
	xRefTable := locateNameTreeXRefTable()
	xRefTable.RootDict = types.Dict{"Names": types.Integer(1)}

	err := xRefTable.LocateNameTree("JavaScript", false)

	requireLocateNameTreeError(t, err, `name tree "JavaScript"`, "dereference Names dictionary", "expected types.Dict")
}

func TestLocateNameTreeGuardsNilNamesDictionary(t *testing.T) {
	xRefTable := locateNameTreeXRefTable()
	xRefTable.RootDict = types.Dict{"Names": *types.NewIndirectRef(42, 0)}

	err := xRefTable.LocateNameTree("JavaScript", false)

	requireLocateNameTreeError(t, err, `name tree "JavaScript"`, "dereference Names dictionary", "missing dictionary")
}

func TestLocateNameTreeTreeCreationContext(t *testing.T) {
	xRefTable := locateNameTreeXRefTable()
	xRefTable.RootDict = types.Dict{"Names": types.NewDict()}
	xRefTable.Table[0].Free = false

	err := xRefTable.LocateNameTree("JavaScript", true)

	requireLocateNameTreeError(t, err, `name tree "JavaScript"`, "create tree", "object #0 found, but not free")
}

func TestLocateNameTreeTreeDereferenceContext(t *testing.T) {
	xRefTable := locateNameTreeXRefTable()
	xRefTable.RootDict = types.Dict{
		"Names": types.Dict{"JavaScript": types.Integer(1)},
	}

	err := xRefTable.LocateNameTree("JavaScript", false)

	requireLocateNameTreeError(t, err, `name tree "JavaScript"`, "dereference tree", "expected types.Dict")
}

func TestLocateNameTreeGuardsNilTreeDictionary(t *testing.T) {
	xRefTable := locateNameTreeXRefTable()
	xRefTable.RootDict = types.Dict{
		"Names": types.Dict{"JavaScript": *types.NewIndirectRef(42, 0)},
	}

	err := xRefTable.LocateNameTree("JavaScript", false)

	requireLocateNameTreeError(t, err, `name tree "JavaScript"`, "dereference tree", "missing dictionary")
}
