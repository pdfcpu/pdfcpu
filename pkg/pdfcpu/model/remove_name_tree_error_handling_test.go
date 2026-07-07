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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func lazyRemovalObject(
	xRefTable *XRefTable,
	f func(*XRefTable),
	o types.Object,
) types.Object {
	osd := &types.ObjectStreamDict{
		StreamDict: types.StreamDict{Content: []byte("object")},
	}
	return types.NewLazyObjectStreamObject(osd, 0, -1, func(context.Context, string) (types.Object, error) {
		if f != nil {
			f(xRefTable)
		}
		return o, nil
	})
}

func TestRemoveNameTreeWrapsNamesDictionaryAccess(t *testing.T) {
	wantErr := errors.New("Names dictionary decode failed")
	xRefTable := portfolioXRefTable()
	xRefTable.Root = types.NewIndirectRef(1, 0)
	xRefTable.Table[1] = NewXRefTableEntryGen0(failingRemovalObject(wantErr))

	err := xRefTable.RemoveNameTree("JavaScript")

	requireRemovalError(
		t,
		err,
		wantErr,
		`name tree "JavaScript": obtain Names dictionary: Names dictionary decode failed`,
	)
}

func TestRemoveNameTreeWrapsTreeEntryDeletion(t *testing.T) {
	wantErr := errors.New("tree entry decode failed")
	xRefTable := portfolioXRefTable()
	xRefTable.RootDict = types.Dict{
		"Names": types.Dict{
			"JavaScript": *types.NewIndirectRef(1, 0),
		},
	}
	xRefTable.Table[1] = NewXRefTableEntryGen0(failingRemovalObject(wantErr))

	err := xRefTable.RemoveNameTree("JavaScript")

	requireRemovalError(
		t,
		err,
		wantErr,
		`name tree "JavaScript": delete tree entry: tree entry decode failed`,
	)
}

func TestRemoveNameTreeWrapsCatalogAccess(t *testing.T) {
	wantErr := errors.New("catalog decode failed")
	xRefTable := portfolioXRefTable()
	xRefTable.RootDict = types.Dict{
		"Names": types.Dict{
			"JavaScript": *types.NewIndirectRef(1, 0),
		},
	}
	xRefTable.Table[1] = NewXRefTableEntryGen0(lazyRemovalObject(
		xRefTable,
		func(xRefTable *XRefTable) {
			xRefTable.RootDict = nil
			xRefTable.Root = types.NewIndirectRef(2, 0)
		},
		types.StringLiteral("tree"),
	))
	xRefTable.Table[2] = NewXRefTableEntryGen0(failingRemovalObject(wantErr))

	err := xRefTable.RemoveNameTree("JavaScript")

	requireRemovalError(
		t,
		err,
		wantErr,
		`name tree "JavaScript": obtain catalog: catalog decode failed`,
	)
}

func TestRemoveNameTreeWrapsEmptyNamesEntryDeletion(t *testing.T) {
	wantErr := errors.New("catalog Names entry decode failed")
	xRefTable := portfolioXRefTable()
	namesDict := types.Dict{
		"JavaScript": *types.NewIndirectRef(1, 0),
	}
	xRefTable.RootDict = types.Dict{
		"Names": namesDict,
	}
	xRefTable.Table[1] = NewXRefTableEntryGen0(lazyRemovalObject(
		xRefTable,
		func(xRefTable *XRefTable) {
			xRefTable.RootDict["Names"] = *types.NewIndirectRef(2, 0)
		},
		types.StringLiteral("tree"),
	))
	xRefTable.Table[2] = NewXRefTableEntryGen0(failingRemovalObject(wantErr))

	err := xRefTable.RemoveNameTree("JavaScript")

	requireRemovalError(
		t,
		err,
		wantErr,
		`name tree "JavaScript": delete empty catalog Names entry: catalog Names entry decode failed`,
	)
}
