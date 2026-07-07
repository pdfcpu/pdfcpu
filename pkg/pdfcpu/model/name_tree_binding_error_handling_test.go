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

func requireNameTreeBindingError(t *testing.T, err error, want ...string) {
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

func TestBindNameTreesAddsTreeNameAndRootDictionaryContext(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	xRefTable.Names["JavaScript"] = &Node{
		D:     types.NewDict(),
		Kmin:  "script",
		Kmax:  "script",
		Names: []entry{{k: "script", v: types.StringLiteral("value")}},
	}

	err := xRefTable.BindNameTrees()

	requireNameTreeBindingError(t, err, `name tree "JavaScript"`, "root dictionary", "missing root dict")
}

func TestBindNameTreesDistinguishesIntermediateAndChildInsertion(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	xRefTable.Table[0] = NewXRefTableEntryGen0(nil)
	xRefTable.Names["JavaScript"] = &Node{
		Kmin: "a",
		Kmax: "z",
		Kids: []*Node{
			{Kmin: "a", Kmax: "a", Names: []entry{{k: "a", v: types.StringLiteral("a")}}},
			{Kmin: "z", Kmax: "z", Names: []entry{{k: "z", v: types.StringLiteral("z")}}},
		},
	}

	err := xRefTable.BindNameTrees()

	requireNameTreeBindingError(
		t,
		err,
		`name tree "JavaScript"`,
		"intermediate node",
		"child object 0",
		"insert",
		"object #0 found, but not free",
	)
}
