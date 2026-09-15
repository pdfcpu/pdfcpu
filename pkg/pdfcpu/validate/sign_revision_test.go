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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestCacheSignatureDictionaryRevision verifies field updates do not change the signature's revision.
func TestCacheSignatureDictionaryRevision(t *testing.T) {
	for _, revisions := range [][2]int{{1, 2}, {3, 1}} {
		fieldRevision, signatureRevision := revisions[0], revisions[1]
		entry := model.NewXRefTableEntryGen0(types.Dict{"Type": types.Name("Sig")})
		entry.Incr = signatureRevision
		version := model.V17
		xref := &model.XRefTable{
			HeaderVersion: &version,
			Table:         map[int]*model.XRefTableEntry{9: entry},
			Signatures:    map[int]map[int]model.Signature{},
		}
		field := types.Dict{
			"FT":   types.Name("Sig"),
			"V":    *types.NewIndirectRef(9, 0),
			"Rect": types.Array{types.Integer(0), types.Integer(0), types.Integer(10), types.Integer(10)},
		}
		if err := cacheSig(xref, field, "formFieldDict", true, 7, fieldRevision); err != nil {
			t.Fatal(err)
		}
		if _, ok := xref.Signatures[signatureRevision][7]; !ok {
			t.Fatalf("field revision %d: signature not cached under revision %d", fieldRevision, signatureRevision)
		}
	}
}
