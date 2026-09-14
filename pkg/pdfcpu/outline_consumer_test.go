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
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type outlineCancelContext struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (c *outlineCancelContext) Err() error {
	c.checks++
	if c.checks == 2 {
		c.cancel()
	}
	return c.Context.Err()
}

func loadedOutlineContext(t *testing.T, scenario string) *model.Context {
	t.Helper()
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.RootDict = types.Dict{"Outlines": *types.NewIndirectRef(4, 0)}
	ctx.Table[4] = model.NewXRefTableEntryGen0(types.Dict{
		"First": *types.NewIndirectRef(5, 0),
		"Last":  *types.NewIndirectRef(6, 0),
		"Count": types.Integer(2),
	})
	first := types.Dict{"Next": *types.NewIndirectRef(6, 0)}
	last := types.Dict{}
	switch scenario {
	case "self":
		first["Next"] = *types.NewIndirectRef(5, 0)
	case "pair":
		last["Next"] = *types.NewIndirectRef(5, 0)
	}
	ctx.Table[5] = model.NewXRefTableEntryGen0(first)
	ctx.Table[6] = model.NewXRefTableEntryGen0(last)
	ctx.Table[9] = model.NewXRefTableEntryGen0(types.Dict{})
	return ctx
}

func runOutlineConsumer(name string, c context.Context, ctx *model.Context) (int, error) {
	parent := types.NewIndirectRef(9, 0)
	switch name {
	case "fold":
		return 0, foldExistingOutlines(c, ctx, ctx.RootDict, parent, false)
	case "attach":
		wrapper, err := ctx.DereferenceDict(*parent)
		if err != nil {
			return 0, err
		}
		return attachWrappedSourceOutlines(c, ctx, ctx.RootDict, wrapper, parent)
	case "reparent":
		return reparentOutlineItems(c, ctx, types.NewIndirectRef(5, 0), parent)
	}
	return 0, fmt.Errorf("unknown outline consumer %q", name)
}

func checkLoadedOutlineResult(t *testing.T, name string, count int, ctx *model.Context) {
	t.Helper()
	if name != "fold" && count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if name == "reparent" {
		return
	}
	wrapper, err := ctx.DereferenceDict(*types.NewIndirectRef(9, 0))
	if err != nil {
		t.Fatal(err)
	}
	first, last := wrapper.IndirectRefEntry("First"), wrapper.IndirectRefEntry("Last")
	if first == nil || first.ObjectNumber.Value() != 5 || last == nil || last.ObjectNumber.Value() != 6 {
		t.Fatalf("wrapper bounds = %v, %v", first, last)
	}
	wantCount := 2
	if name == "fold" {
		wantCount = -2
	}
	if count := wrapper.IntEntry("Count"); count == nil || *count != wantCount {
		t.Fatalf("wrapper Count = %v, want %d", count, wantCount)
	}
}

func checkLoadedOutlineParents(t *testing.T, ctx *model.Context) {
	t.Helper()
	for _, objNr := range []int{5, 6} {
		d, err := ctx.DereferenceDict(*types.NewIndirectRef(objNr, 0))
		if err != nil {
			t.Fatal(err)
		}
		parent := d.IndirectRefEntry("Parent")
		if parent == nil || parent.ObjectNumber.Value() != 9 {
			t.Fatalf("obj#%d Parent = %v, want obj#9", objNr, parent)
		}
	}
}

// TestLoadedOutlineConsumers verifies valid chains and local cycle tracking for all outline merge loops.
func TestLoadedOutlineConsumers(t *testing.T) {
	for _, name := range []string{"fold", "attach", "reparent"} {
		t.Run(name+"/valid", func(t *testing.T) {
			ctx := loadedOutlineContext(t, "valid")
			count, err := runOutlineConsumer(name, t.Context(), ctx)
			if err != nil {
				t.Fatal(err)
			}
			checkLoadedOutlineResult(t, name, count, ctx)
			checkLoadedOutlineParents(t, ctx)
		})
		for _, scenario := range []string{"self", "pair"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				ctx := loadedOutlineContext(t, scenario)
				for range 2 {
					_, err := runOutlineConsumer(name, t.Context(), ctx)
					if !errors.Is(err, ErrCircularBookmarks) || !strings.Contains(err.Error(), "obj#5") {
						t.Fatalf("got %v", err)
					}
				}
			})
		}
	}
}

// TestLoadedOutlineConsumersCancellation verifies every loop observes cancellation after traversal starts.
func TestLoadedOutlineConsumersCancellation(t *testing.T) {
	for _, name := range []string{"fold", "attach", "reparent"} {
		t.Run(name, func(t *testing.T) {
			base, cancel := context.WithCancel(t.Context())
			defer cancel()
			c := &outlineCancelContext{Context: base, cancel: cancel}
			_, err := runOutlineConsumer(name, c, loadedOutlineContext(t, "valid"))
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
			if c.checks != 2 {
				t.Fatalf("continued traversal after cancellation: %d checks", c.checks)
			}
		})
	}
}
