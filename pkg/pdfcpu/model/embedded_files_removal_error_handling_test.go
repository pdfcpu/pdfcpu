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

func failingRemovalObject(err error) types.Object {
	osd := &types.ObjectStreamDict{
		StreamDict: types.StreamDict{Content: []byte("object")},
	}
	return types.NewLazyObjectStreamObject(osd, 0, -1, func(context.Context, string) (types.Object, error) {
		return nil, err
	})
}

func requireRemovalError(t *testing.T, err, wantErr error, want string) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected cause %v, got %v", wantErr, err)
	}
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

func TestRemoveEmbeddedFilesNameTreeWrapsNameTreeRemoval(t *testing.T) {
	wantErr := errors.New("embedded files tree decode failed")
	xRefTable := portfolioXRefTable()
	xRefTable.RootDict = types.Dict{
		"Names": types.Dict{
			"EmbeddedFiles": *types.NewIndirectRef(1, 0),
		},
	}
	xRefTable.Table[1] = NewXRefTableEntryGen0(failingRemovalObject(wantErr))

	err := xRefTable.RemoveEmbeddedFilesNameTree()

	requireRemovalError(
		t,
		err,
		wantErr,
		`remove EmbeddedFiles name tree: name tree "EmbeddedFiles": delete tree entry: embedded files tree decode failed`,
	)
}

func TestRemoveEmbeddedFilesNameTreeWrapsCollectionRemoval(t *testing.T) {
	wantErr := errors.New("Collection decode failed")
	xRefTable := portfolioXRefTable()
	xRefTable.RootDict = types.Dict{
		"Names": types.Dict{
			"EmbeddedFiles": types.StringLiteral("attachment"),
		},
		"Collection": *types.NewIndirectRef(1, 0),
	}
	xRefTable.Table[1] = NewXRefTableEntryGen0(failingRemovalObject(wantErr))

	err := xRefTable.RemoveEmbeddedFilesNameTree()

	requireRemovalError(
		t,
		err,
		wantErr,
		"remove portfolio Collection: portfolio: delete Collection entry: Collection decode failed",
	)
}
