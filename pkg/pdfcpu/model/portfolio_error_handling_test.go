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
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func portfolioXRefTable() *XRefTable {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	size := 1
	xRefTable.Size = &size
	xRefTable.Table[0] = NewFreeHeadXRefTableEntry()
	return xRefTable
}

func TestEnsureCollectionWrapsCatalogAccess(t *testing.T) {
	wantErr := errors.New("catalog decode failed")
	xRefTable := portfolioXRefTable()
	xRefTable.Root = types.NewIndirectRef(1, 0)
	osd := &types.ObjectStreamDict{
		StreamDict: types.StreamDict{Content: []byte("catalog")},
	}
	lazyObject := types.NewLazyObjectStreamObject(osd, 0, -1, func(context.Context, string) (types.Object, error) {
		return nil, wantErr
	})
	xRefTable.Table[1] = NewXRefTableEntryGen0(lazyObject)

	err := xRefTable.EnsureCollection()

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "portfolio: catalog") {
		t.Fatalf("expected portfolio catalog context in %q", err)
	}
}

func TestEnsureCollectionGuardsNilCatalog(t *testing.T) {
	xRefTable := portfolioXRefTable()
	xRefTable.Root = types.NewIndirectRef(42, 0)

	err := xRefTable.EnsureCollection()

	if err == nil || !strings.Contains(err.Error(), "portfolio: catalog: missing dictionary") {
		t.Fatalf("expected missing portfolio catalog dictionary, got %v", err)
	}
}

func TestEnsureCollectionWrapsSchemaObjectInsertion(t *testing.T) {
	xRefTable := portfolioXRefTable()
	xRefTable.RootDict = types.NewDict()
	xRefTable.Table[0].Free = false

	err := xRefTable.EnsureCollection()

	if err == nil || !strings.Contains(err.Error(), "portfolio: insert schema object") {
		t.Fatalf("expected schema insertion context, got %v", err)
	}
}

func TestEnsureCollectionWrapsCollectionObjectInsertion(t *testing.T) {
	xRefTable := portfolioXRefTable()
	xRefTable.RootDict = types.NewDict()
	firstFreeObject := int64(1)
	missingFreeObject := int64(2)
	generation := 0
	xRefTable.Table[0].Offset = &firstFreeObject
	xRefTable.Table[1] = &XRefTableEntry{
		Free:       true,
		Offset:     &missingFreeObject,
		Generation: &generation,
	}

	err := xRefTable.EnsureCollection()

	if err == nil || !strings.Contains(err.Error(), "portfolio: insert Collection object") {
		t.Fatalf("expected Collection insertion context, got %v", err)
	}
}

func TestRemoveCollectionWrapsCatalogAccess(t *testing.T) {
	wantErr := errors.New("catalog decode failed")
	xRefTable := portfolioXRefTable()
	xRefTable.Root = types.NewIndirectRef(1, 0)
	xRefTable.Table[1] = NewXRefTableEntryGen0(failingRemovalObject(wantErr))

	err := xRefTable.RemoveCollection()

	requireRemovalError(t, err, wantErr, "portfolio: catalog: catalog decode failed")
}

func TestRemoveCollectionGuardsNilCatalog(t *testing.T) {
	xRefTable := portfolioXRefTable()
	xRefTable.Root = types.NewIndirectRef(42, 0)

	err := xRefTable.RemoveCollection()

	if err == nil || !strings.Contains(err.Error(), "portfolio: catalog: missing dictionary") {
		t.Fatalf("expected missing portfolio catalog dictionary, got %v", err)
	}
}

func TestRemoveCollectionWrapsCollectionEntryDeletion(t *testing.T) {
	wantErr := errors.New("Collection decode failed")
	xRefTable := portfolioXRefTable()
	xRefTable.RootDict = types.Dict{
		"Collection": *types.NewIndirectRef(1, 0),
	}
	xRefTable.Table[1] = NewXRefTableEntryGen0(failingRemovalObject(wantErr))

	err := xRefTable.RemoveCollection()

	requireRemovalError(
		t,
		err,
		wantErr,
		"portfolio: delete Collection entry: Collection decode failed",
	)
}
