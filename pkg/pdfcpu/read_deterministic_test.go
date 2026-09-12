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

package pdfcpu

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	pdfcpuLog "github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func failingCompressedEntry(objectStreamObjNr int) *model.XRefTableEntry {
	objectStreamIndex := 0
	return &model.XRefTableEntry{
		Compressed:      true,
		ObjectStream:    &objectStreamObjNr,
		ObjectStreamInd: &objectStreamIndex,
	}
}

func denseDereferenceTestContext() *model.Context {
	table := map[int]*model.XRefTableEntry{}
	for objNr := range 8 {
		table[objNr] = model.NewFreeHeadXRefTableEntry()
	}
	table[3] = failingCompressedEntry(30)
	table[7] = failingCompressedEntry(70)
	return &model.Context{XRefTable: &model.XRefTable{Table: table}}
}

func sparseDereferenceTestContext() *model.Context {
	return &model.Context{XRefTable: &model.XRefTable{Table: map[int]*model.XRefTableEntry{
		3:  failingCompressedEntry(30),
		99: failingCompressedEntry(990),
	}}}
}

func TestDereferenceObjectsAscendingSelectsLowestDenseObjectError(t *testing.T) {
	for range 25 {
		ctx := denseDereferenceTestContext()
		maxObjNr := maxObjectNumber(ctx.Table)
		if !denseXRefTable(ctx.Table, maxObjNr) {
			t.Fatal("expected dense xref traversal")
		}
		err := dereferenceObjectsAscending(t.Context(), ctx)
		if err == nil || !strings.Contains(err.Error(), "object stream 30") {
			t.Fatalf("expected lowest object error for obj #3, got %v", err)
		}
		if !errors.Is(err, errMissingObjectStreamEntry) {
			t.Fatalf("expected missing object stream entry cause, got %v", err)
		}
	}
}

func TestDereferenceObjectsAscendingSelectsLowestSparseObjectError(t *testing.T) {
	for range 25 {
		ctx := sparseDereferenceTestContext()
		maxObjNr := maxObjectNumber(ctx.Table)
		if denseXRefTable(ctx.Table, maxObjNr) {
			t.Fatal("expected sparse xref traversal")
		}
		err := dereferenceObjectsAscending(t.Context(), ctx)
		if err == nil || !strings.Contains(err.Error(), "object stream 30") {
			t.Fatalf("expected lowest object error for obj #3, got %v", err)
		}
		if !errors.Is(err, errMissingObjectStreamEntry) {
			t.Fatalf("expected missing object stream entry cause, got %v", err)
		}
	}
}

func dereferenceObjectsErrorWithStats(t *testing.T, statsEnabled bool) error {
	t.Helper()
	pdfcpuLog.SetStatsLogger(nil)
	if statsEnabled {
		pdfcpuLog.SetStatsLogger(log.New(&bytes.Buffer{}, "", 0))
	}
	return dereferenceObjects(t.Context(), denseDereferenceTestContext())
}

func TestDereferenceObjectsErrorIndependentOfStatsLogging(t *testing.T) {
	defer pdfcpuLog.SetStatsLogger(nil)

	for range 25 {
		withoutStats := dereferenceObjectsErrorWithStats(t, false)
		withStats := dereferenceObjectsErrorWithStats(t, true)
		if withoutStats == nil || withStats == nil {
			t.Fatalf("expected errors, got without stats %v and with stats %v", withoutStats, withStats)
		}
		if withoutStats.Error() != withStats.Error() {
			t.Fatalf("error depends on stats logging:\nwithout: %v\nwith:    %v", withoutStats, withStats)
		}
		if !errors.Is(withoutStats, errMissingObjectStreamEntry) || !errors.Is(withStats, errMissingObjectStreamEntry) {
			t.Fatalf("expected missing object stream causes, got without stats %v and with stats %v", withoutStats, withStats)
		}
	}
}

func reconstructionRetryTestContext(t *testing.T) *model.Context {
	t.Helper()

	pdf := []byte(`%PDF-1.7
% bad bad obj
1 0 obj
1
endobj
2 0 obj
2
endobj
3 0 obj
[[[1]]]
endobj
4 0 obj
4
endobj
5 0 obj
5
endobj
6 0 obj
6
endobj
7 0 obj
invalid
endobj
`)
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationStrict
	conf.Limits.MaxRecursionDepth = 1
	ctx, err := model.NewContext(bytes.NewReader(pdf), conf)
	if err != nil {
		t.Fatal(err)
	}

	ctx.XRefTable.ValidationMode = model.ValidationStrict
	offset := int64(bytes.Index(pdf, []byte("bad bad obj")))
	generation := 0
	ctx.Table = map[int]*model.XRefTableEntry{
		0: model.NewFreeHeadXRefTableEntry(),
		9: {Offset: &offset, Generation: &generation},
	}
	return ctx
}

func dereferenceObjectsAfterReconstructionError(t *testing.T, statsEnabled bool) (error, *model.Context) {
	t.Helper()

	pdfcpuLog.SetStatsLogger(nil)
	if statsEnabled {
		pdfcpuLog.SetStatsLogger(log.New(&bytes.Buffer{}, "", 0))
	}
	ctx := reconstructionRetryTestContext(t)
	return dereferenceObjects(t.Context(), ctx), ctx
}

// TestDereferenceObjectsReconstructionRetryIsDeterministic verifies that xref reconstruction preserves the same
// first dereference error with normal and debug traversal.
func TestDereferenceObjectsReconstructionRetryIsDeterministic(t *testing.T) {
	defer pdfcpuLog.SetStatsLogger(nil)

	for range 25 {
		withoutStats, normalCtx := dereferenceObjectsAfterReconstructionError(t, false)
		withStats, debugCtx := dereferenceObjectsAfterReconstructionError(t, true)
		if withoutStats == nil || withStats == nil {
			t.Fatalf("expected errors, got without stats %v and with stats %v", withoutStats, withStats)
		}
		if withoutStats.Error() != withStats.Error() {
			t.Fatalf("reconstruction retry error depends on stats logging:\nwithout: %v\nwith:    %v", withoutStats, withStats)
		}
		if !errors.Is(withoutStats, model.ErrMaxRecursionDepthExceeded) ||
			!errors.Is(withStats, model.ErrMaxRecursionDepthExceeded) {
			t.Fatalf("expected recursion depth causes, got without stats %v and with stats %v", withoutStats, withStats)
		}
		for _, ctx := range []*model.Context{normalCtx, debugCtx} {
			if _, found := ctx.Table[3]; !found {
				t.Fatal("expected reconstructed xref entry for object 3")
			}
			if _, found := ctx.Table[7]; !found {
				t.Fatal("expected reconstructed xref entry for object 7")
			}
			if _, found := ctx.Table[9]; found {
				t.Fatal("unexpected stale xref entry for object 9 after reconstruction")
			}
		}
	}
}

func TestMaterializeEncryptionIntegersSelectsLowestCryptFilter(t *testing.T) {
	for range 25 {
		ctx, err := model.NewContext(bytes.NewReader(nil), nil)
		if err != nil {
			t.Fatal(err)
		}
		ctx.Table[3] = model.NewXRefTableEntryGen0(types.Name("wrong"))
		ctx.Table[9] = model.NewXRefTableEntryGen0(types.Name("wrong"))
		d := types.Dict{"CF": types.Dict{
			"Zulu":  types.Dict{"Length": *types.NewIndirectRef(9, 0)},
			"Alpha": types.Dict{"Length": *types.NewIndirectRef(3, 0)},
		}}

		err = materializeEncryptionIntegers(t.Context(), ctx, d)
		if err == nil || !strings.Contains(err.Error(), "crypt filter Alpha") {
			t.Fatalf("expected Alpha crypt filter error, got %v", err)
		}
		if !errors.Is(err, errCorruptIntegerObject) {
			t.Fatalf("expected corrupt integer cause, got %v", err)
		}
	}
}

func TestValidateLinearizationDirectEntriesSelectsLowestKey(t *testing.T) {
	d := types.Dict{
		"Zulu":  *types.NewIndirectRef(9, 0),
		"Alpha": *types.NewIndirectRef(3, 0),
	}
	for range 25 {
		err := validateLinearizationDirectEntries(d, 1)
		if err == nil || !strings.Contains(err.Error(), "entry Alpha") {
			t.Fatalf("expected Alpha entry error, got %v", err)
		}
	}
}

func TestValidateHintStreamDictSelectsLowestKey(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	d := types.Dict{
		"Zulu":  *types.NewIndirectRef(9, 0),
		"Alpha": *types.NewIndirectRef(3, 0),
	}
	for range 25 {
		err := validateHintStreamDict(ctx, d, 1)
		if err == nil || !strings.Contains(err.Error(), "entry Alpha") {
			t.Fatalf("expected Alpha entry error, got %v", err)
		}
	}
}

func TestValidateHintStreamNestedDictSelectsLowestKey(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	d := types.Dict{"Container": types.Dict{
		"Zulu":  *types.NewIndirectRef(9, 0),
		"Alpha": *types.NewIndirectRef(3, 0),
	}}
	for range 25 {
		err := validateHintStreamDict(ctx, d, 1)
		if err == nil || !strings.Contains(err.Error(), "entry Container key Alpha") {
			t.Fatalf("expected nested Alpha key error, got %v", err)
		}
	}
}

func TestValidateHintStreamDictionariesSelectsLowestObject(t *testing.T) {
	for range 25 {
		ctx, err := model.NewContext(bytes.NewReader(nil), nil)
		if err != nil {
			t.Fatal(err)
		}
		ctx.Read.Linearized = true
		offset := int64(100)
		ctx.OffsetPrimaryHintTable = &offset
		for _, objNr := range []int{9, 3} {
			entry := model.NewXRefTableEntryGen0(types.StreamDict{Dict: types.Dict{
				"Alpha": *types.NewIndirectRef(objNr, 0),
			}})
			entry.Offset = &offset
			ctx.Table[objNr] = entry
		}

		err = validateHintStreamDictionaries(ctx)
		if err == nil || !strings.Contains(err.Error(), "hint stream obj#3") {
			t.Fatalf("expected lowest hint stream object error, got %v", err)
		}
	}
}

func TestDecryptDictSelectsLowestKey(t *testing.T) {
	for range 25 {
		d := types.Dict{
			"Zulu":  types.HexLiteral("invalid-zulu"),
			"Alpha": types.HexLiteral("invalid-alpha"),
		}
		err := decryptDict(d, 1, 0, []byte("key"), false, 2)
		if err == nil || !strings.Contains(err.Error(), "entry Alpha") {
			t.Fatalf("expected Alpha decryption error, got %v", err)
		}
	}
}
