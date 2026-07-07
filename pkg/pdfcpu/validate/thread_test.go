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

func TestValidateThreadDictReportsObjectContext(t *testing.T) {
	err := validateThreadDict(testXRef(t, model.ValidationStrict), types.Integer(1), model.V11)
	if err == nil {
		t.Fatal("expected invalid thread object error")
	}
	if got := err.Error(); got != "threadDict: expected indirect ref, got types.Integer" {
		t.Fatalf("got %q, want thread object context", got)
	}
}

func TestValidateThreadDictReportsMissingFirstBeadContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	threadRef, err := xRefTable.IndRefForObject(1, types.Dict{
		"Type": types.Name("Thread"),
	})
	if err != nil {
		t.Fatal(err)
	}

	err = validateThreadDict(xRefTable, *threadRef, model.V11)
	requireErrContains(t, err, "thread obj#1 F: missing required indirect entry")
}

func TestValidateThreadsReportsArrayIndexContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	threadsRef, err := xRefTable.IndRefForObject(1, types.Array{types.Integer(1)})
	if err != nil {
		t.Fatal(err)
	}

	err = validateThreads(xRefTable, types.Dict{"Threads": *threadsRef}, OPTIONAL, model.V11)
	requireErrContains(t, err, "rootDict.Threads[0]")
}

func TestValidateThreadsReportsArrayObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	threadRef, err := xRefTable.IndRefForObject(2, types.Dict{
		"Type": types.Name("Thread"),
	})
	if err != nil {
		t.Fatal(err)
	}
	threadsRef, err := xRefTable.IndRefForObject(1, types.Array{*threadRef})
	if err != nil {
		t.Fatal(err)
	}

	err = validateThreads(xRefTable, types.Dict{"Threads": *threadsRef}, OPTIONAL, model.V11)
	requireErrChainContains(t, err, "rootDict.Threads[0] obj#2", "thread obj#2 F")
}

func TestValidateFirstBeadDictReportsBackpointerContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	threadRef, err := xRefTable.IndRefForObject(1, types.Dict{"Type": types.Name("Thread")})
	if err != nil {
		t.Fatal(err)
	}
	otherThreadRef, err := xRefTable.IndRefForObject(2, types.Dict{"Type": types.Name("Thread")})
	if err != nil {
		t.Fatal(err)
	}
	beadRef, err := xRefTable.IndRefForObject(3, types.Dict{
		"T": *otherThreadRef,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = validateFirstBeadDict(xRefTable, beadRef, threadRef)
	requireErrContains(t, err, "first bead obj#3: invalid T backpointer")
}
