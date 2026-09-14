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

type annotationCancelContext struct {
	context.Context
	cancel   context.CancelFunc
	cancelAt int
	checks   int
}

func (c *annotationCancelContext) Err() error {
	c.checks++
	if c.checks == c.cancelAt {
		c.cancel()
	}
	return c.Context.Err()
}

func annotationCancellationXRefTable() (*model.XRefTable, types.Array) {
	conf := model.NewDefaultConfiguration()
	v := model.V17
	xRefTable := &model.XRefTable{
		Table:          map[int]*model.XRefTableEntry{},
		PageAnnots:     map[int]model.PgAnnots{},
		Conf:           conf,
		HeaderVersion:  &v,
		ValidationMode: model.ValidationStrict,
		CurPage:        1,
	}
	a := types.Array{}
	for objNumber := 1; objNumber <= 4; objNumber++ {
		xRefTable.Table[objNumber] = model.NewXRefTableEntryGen0(types.Dict{
			"Type":     types.Name("Annot"),
			"Subtype":  types.Name("Text"),
			"Rect":     types.NewNumberArray(0, 0, 1, 1),
			"Contents": types.StringLiteral("annotation"),
		})
		a = append(a, *types.NewIndirectRef(objNumber, 0))
	}
	return xRefTable, a
}

func TestValidateAnnotationsArrayCancellation(t *testing.T) {
	xRefTable, a := annotationCancellationXRefTable()
	base, cancel := context.WithCancel(t.Context())
	defer cancel()
	c := &annotationCancelContext{Context: base, cancel: cancel, cancelAt: 2}
	_, err := validateAnnotationsArray(c, xRefTable, a, 10)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if c.checks != 2 {
		t.Fatalf("continued traversal after cancellation: %d checks", c.checks)
	}
}
