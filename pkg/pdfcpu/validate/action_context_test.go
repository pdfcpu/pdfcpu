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

type actionCancelContext struct {
	context.Context
	cancel   context.CancelFunc
	cancelAt int
	checks   int
}

func (c *actionCancelContext) Err() error {
	c.checks++
	if c.checks == c.cancelAt {
		c.cancel()
	}
	return c.Context.Err()
}

func cancellationActionDict(next types.Object) types.Dict {
	d := types.Dict{
		"S": types.Name("Named"),
		"N": types.Name("FirstPage"),
	}
	if next != nil {
		d["Next"] = next
	}
	return d
}

func actionCancellationXRefTable(dicts map[int]types.Dict) *model.XRefTable {
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxRecursionDepth = 100
	table := map[int]*model.XRefTableEntry{}
	for objNr, d := range dicts {
		table[objNr] = model.NewXRefTableEntryGen0(d)
	}
	v := model.V17
	return &model.XRefTable{
		Table:          table,
		Conf:           conf,
		HeaderVersion:  &v,
		ValidationMode: model.ValidationStrict,
	}
}

func cancellationEmbeddedTarget(next types.Object) types.Dict {
	d := types.Dict{"R": types.Name("P")}
	if next != nil {
		d["T"] = next
	}
	return d
}

func cancellationEmbeddedGoToAction(target types.Object) types.Dict {
	return types.Dict{
		"S": types.Name("GoToE"),
		"D": types.StringLiteral("destination"),
		"T": target,
	}
}

func TestValidateActionNextArrayCancellation(t *testing.T) {
	dicts := map[int]types.Dict{}
	next := types.Array{}
	for objNr := 2; objNr <= 5; objNr++ {
		dicts[objNr] = cancellationActionDict(nil)
		next = append(next, *types.NewIndirectRef(objNr, 0))
	}
	dicts[1] = cancellationActionDict(next)

	base, cancel := context.WithCancel(t.Context())
	defer cancel()
	c := &actionCancelContext{Context: base, cancel: cancel, cancelAt: 4}
	err := validateActionDictObject(c, actionCancellationXRefTable(dicts), dicts[1], *types.NewIndirectRef(1, 0), "test action")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if c.checks != 4 {
		t.Fatalf("continued traversal after cancellation: %d checks", c.checks)
	}
}

func TestValidateEmbeddedTargetCancellation(t *testing.T) {
	ir5 := *types.NewIndirectRef(5, 0)
	ir6 := *types.NewIndirectRef(6, 0)
	dicts := map[int]types.Dict{
		5: cancellationEmbeddedTarget(ir6),
		6: cancellationEmbeddedTarget(nil),
	}

	base, cancel := context.WithCancel(t.Context())
	defer cancel()
	c := &actionCancelContext{Context: base, cancel: cancel, cancelAt: 2}
	err := validateGoToEActionDict(c, actionCancellationXRefTable(dicts), cancellationEmbeddedGoToAction(ir5), "GoToE")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if c.checks != 2 {
		t.Fatalf("continued traversal after cancellation: %d checks", c.checks)
	}
}
