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

func freeListTestEntry(next int64) *XRefTableEntry {
	generation := 0
	return &XRefTableEntry{Free: true, Offset: &next, Generation: &generation}
}

func TestEnsureValidFreeListRepairsDeterministically(t *testing.T) {
	for range 25 {
		xRefTable := newXRefTable(NewDefaultConfiguration())
		xRefTable.Table[0] = NewFreeHeadXRefTableEntry()
		*xRefTable.Table[0].Offset = 99
		xRefTable.Table[11] = freeListTestEntry(0)
		xRefTable.Table[7] = freeListTestEntry(0)
		xRefTable.Table[3] = freeListTestEntry(0)

		if err := xRefTable.EnsureValidFreeList(); err != nil {
			t.Fatal(err)
		}

		for objNr, want := range map[int]int64{0: 11, 11: 7, 7: 3, 3: 0} {
			if got := *xRefTable.Table[objNr].Offset; got != want {
				t.Fatalf("free object %d: expected successor %d, got %d", objNr, want, got)
			}
		}
	}
}

func TestHandleDanglingFreeSelectsLowestObjectError(t *testing.T) {
	for range 25 {
		xRefTable := newXRefTable(NewDefaultConfiguration())
		xRefTable.Table[3] = NewXRefTableEntryGen0(types.Integer(1))
		xRefTable.Table[9] = NewXRefTableEntryGen0(types.Integer(2))

		err := xRefTable.handleDanglingFree(types.IntSet{9: true, 3: true}, NewFreeHeadXRefTableEntry())
		if err == nil || !strings.Contains(err.Error(), "not free for obj #3") {
			t.Fatalf("expected lowest object error for obj #3, got %v", err)
		}
	}
}
