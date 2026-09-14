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
	"context"
	"errors"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type xObjectCancelContext struct {
	context.Context
	cancel   context.CancelFunc
	cancelAt int
	checks   int
}

func (c *xObjectCancelContext) Err() error {
	c.checks++
	if c.checks == c.cancelAt {
		c.cancel()
	}
	return c.Context.Err()
}

func formXObject(resources types.Object) types.StreamDict {
	d := types.Dict{
		"Type":    types.Name("XObject"),
		"Subtype": types.Name("Form"),
		"BBox":    types.NewNumberArray(0, 0, 1, 1),
	}
	if resources != nil {
		d["Resources"] = resources
	}
	return types.StreamDict{Dict: d}
}

func xObjectCancellationXRefTable(objects map[int]types.Object) *model.XRefTable {
	table := map[int]*model.XRefTableEntry{}
	for objNr, o := range objects {
		table[objNr] = model.NewXRefTableEntryGen0(o)
	}
	v := model.V17
	return &model.XRefTable{
		Table:          table,
		Conf:           model.NewDefaultConfiguration(),
		HeaderVersion:  &v,
		ValidationMode: model.ValidationStrict,
	}
}

func TestValidateXObjectResourceCancellation(t *testing.T) {
	objects := map[int]types.Object{}
	resources := types.Dict{}
	for objNr := 1; objNr <= 4; objNr++ {
		objects[objNr] = formXObject(nil)
		resources[string(rune('A'+objNr-1))] = *types.NewIndirectRef(objNr, 0)
	}
	xRefTable := xObjectCancellationXRefTable(objects)
	base, cancel := context.WithCancel(t.Context())
	defer cancel()
	c := &xObjectCancelContext{Context: base, cancel: cancel, cancelAt: 2}
	err := validateXObjectResourceDict(c, xRefTable, resources, model.V10)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if c.checks != 2 {
		t.Fatalf("continued traversal after cancellation: %d checks", c.checks)
	}
	if !xRefTable.Table[1].Valid {
		t.Fatal("first XObject was not validated")
	}
	for objNr := 2; objNr <= 4; objNr++ {
		if xRefTable.Table[objNr].Valid {
			t.Fatalf("XObject %d validated after cancellation", objNr)
		}
	}
}

func TestValidateXObjectResourcePreservesRecursiveAndSharedForms(t *testing.T) {
	for _, tt := range []struct {
		name      string
		resources types.Dict
		nested    types.Object
	}{
		{
			name:      "recursive",
			resources: types.Dict{"Root": *types.NewIndirectRef(1, 0)},
			nested:    types.Dict{"XObject": types.Dict{"Self": *types.NewIndirectRef(1, 0)}},
		},
		{
			name:      "shared",
			resources: types.Dict{"First": *types.NewIndirectRef(1, 0), "Second": *types.NewIndirectRef(1, 0)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			xRefTable := xObjectCancellationXRefTable(map[int]types.Object{1: formXObject(tt.nested)})
			if err := validateXObjectResourceDict(t.Context(), xRefTable, tt.resources, model.V10); err != nil {
				t.Fatal(err)
			}
			if !xRefTable.Table[1].Valid {
				t.Fatal("Form XObject was not marked valid")
			}
		})
	}
}
