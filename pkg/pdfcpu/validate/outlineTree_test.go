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

func newOutlineTestRoot(t *testing.T, xRefTable *model.XRefTable, objNr *int) *types.IndirectRef {
	t.Helper()
	root := types.Dict{"Type": types.Name("Outlines")}
	rootRef, err := xRefTable.IndRefForObject(*objNr, root)
	if err != nil {
		t.Fatal(err)
	}
	*objNr = *objNr + 1
	return rootRef
}

func newOutlineTestItem(t *testing.T, xRefTable *model.XRefTable, objNr *int, parent *types.IndirectRef, count *int) (*types.IndirectRef, types.Dict) {
	t.Helper()
	d := types.Dict{
		"Parent": *parent,
		"Title":  types.StringLiteral("bookmark"),
	}
	if count != nil {
		d["Count"] = types.Integer(*count)
	}
	itemRef, err := xRefTable.IndRefForObject(*objNr, d)
	if err != nil {
		t.Fatal(err)
	}
	*objNr = *objNr + 1
	return itemRef, d
}

func TestValidateOutlineTreeRejectsLeafCountStrict(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	objNr := 1
	rootRef := newOutlineTestRoot(t, xRefTable, &objNr)
	count := 1
	itemRef, _ := newOutlineTestItem(t, xRefTable, &objNr, rootRef, &count)
	fixed := false

	_, _, err := validateOutlineTree(xRefTable, itemRef, itemRef, map[int]bool{}, &fixed)
	requireErrContains(t, err, "outline item obj#2: leaf Count must be 0")
}

func TestValidateOutlineTreeReportsNestedChildContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	objNr := 1
	rootRef := newOutlineTestRoot(t, xRefTable, &objNr)
	count := 1
	parentRef, parentDict := newOutlineTestItem(t, xRefTable, &objNr, rootRef, &count)
	childRef, childDict := newOutlineTestItem(t, xRefTable, &objNr, parentRef, nil)
	parentDict["First"] = *childRef
	parentDict["Last"] = *childRef
	childDict["Count"] = types.Integer(1)
	fixed := false

	_, _, err := validateOutlineTree(xRefTable, parentRef, parentRef, map[int]bool{}, &fixed)
	requireErrChainContains(t, err, "outline item obj#2", "validate children", "outline item obj#3", "leaf Count must be 0")
}

func TestValidateOutlineTreeRepairsLeafCountRelaxed(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationRelaxed)
	objNr := 1
	rootRef := newOutlineTestRoot(t, xRefTable, &objNr)
	count := 1
	itemRef, itemDict := newOutlineTestItem(t, xRefTable, &objNr, rootRef, &count)
	fixed := false

	if _, _, err := validateOutlineTree(xRefTable, itemRef, itemRef, map[int]bool{}, &fixed); err != nil {
		t.Fatalf("relaxed validation: %v", err)
	}
	if _, ok := itemDict["Count"]; ok {
		t.Fatal("relaxed validation preserved stale leaf Count")
	}
	if !fixed {
		t.Fatal("relaxed validation did not mark stale leaf Count as repaired")
	}
}

func TestValidateOutlineTreeRepairsNonLeafCountRelaxed(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationRelaxed)
	objNr := 1
	rootRef := newOutlineTestRoot(t, xRefTable, &objNr)
	count := 99
	parentRef, parentDict := newOutlineTestItem(t, xRefTable, &objNr, rootRef, &count)
	firstRef, firstDict := newOutlineTestItem(t, xRefTable, &objNr, parentRef, nil)
	lastRef, lastDict := newOutlineTestItem(t, xRefTable, &objNr, parentRef, nil)
	firstDict["Next"] = *lastRef
	lastDict["Prev"] = *firstRef
	parentDict["First"] = *firstRef
	parentDict["Last"] = *lastRef
	fixed := false

	if _, _, err := validateOutlineTree(xRefTable, parentRef, parentRef, map[int]bool{}, &fixed); err != nil {
		t.Fatalf("relaxed validation: %v", err)
	}
	if got := parentDict.IntEntry("Count"); got == nil || *got != 2 {
		t.Fatalf("relaxed validation repaired Count to %v, want 2", got)
	}
	if !fixed {
		t.Fatal("relaxed validation did not mark non-leaf Count as repaired")
	}
}

func TestValidateOutlineTreeAcceptsCollapsedCountStrict(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	objNr := 1
	rootRef := newOutlineTestRoot(t, xRefTable, &objNr)
	count := -2
	parentRef, parentDict := newOutlineTestItem(t, xRefTable, &objNr, rootRef, &count)
	firstRef, firstDict := newOutlineTestItem(t, xRefTable, &objNr, parentRef, nil)
	lastRef, lastDict := newOutlineTestItem(t, xRefTable, &objNr, parentRef, nil)
	firstDict["Next"] = *lastRef
	lastDict["Prev"] = *firstRef
	parentDict["First"] = *firstRef
	parentDict["Last"] = *lastRef
	fixed := false

	if _, _, err := validateOutlineTree(xRefTable, parentRef, parentRef, map[int]bool{}, &fixed); err != nil {
		t.Fatalf("strict validation rejected collapsed Count: %v", err)
	}
}

func TestValidateOutlinesReportsRootContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	objNr := 1
	outlineDict := types.Dict{"Type": types.Name("Outlines")}
	outlineRef, err := xRefTable.IndRefForObject(objNr, outlineDict)
	if err != nil {
		t.Fatal(err)
	}
	objNr++
	itemRef, _ := newOutlineTestItem(t, xRefTable, &objNr, outlineRef, nil)
	outlineDict["First"] = *itemRef

	rootDict := types.Dict{"Outlines": *outlineRef}

	err = validateOutlines(xRefTable, rootDict, OPTIONAL, model.V10)
	requireErrContains(t, err, "outline root: missing Last")
	requireErrNotContains(t, err, "outline root: outline root")
}

func TestValidateOutlinesReportsRootObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	osd := types.NewObjectStreamDict()
	lazy := types.NewLazyObjectStreamObject(osd, 1, -1, nil)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(lazy)

	err := validateOutlines(xRefTable, types.Dict{"Outlines": *types.NewIndirectRef(2, 0)}, OPTIONAL, model.V10)
	requireErrChainContains(t, err, "outline root obj#2", "dereference", "unexpected EOF")
}
