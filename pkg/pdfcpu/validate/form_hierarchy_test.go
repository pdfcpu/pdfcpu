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
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestFormHierarchyCachedNodes verifies structure is checked even when semantic validity is cached.
func TestFormHierarchyCachedNodes(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ir := *types.NewIndirectRef(1, 0)
	ctx.Table[1] = model.NewXRefTableEntryGen0(types.Dict{})
	if err := ctx.SetValid(ir); err != nil {
		t.Fatal(err)
	}
	err = validateFormHierarchyAndFields(t.Context(), ctx.XRefTable, types.Array{ir, ir}, 20, false)
	if !errors.Is(err, model.ErrFormFieldDuplicate) {
		t.Fatalf("got %v", err)
	}
	var attributed *model.ValidationError
	if !errors.As(err, &attributed) || attributed.ObjectNumber() != 1 {
		t.Fatalf("attribution: %v", err)
	}
}

// TestFormHierarchyCancellation verifies the preflight observes cancellation before semantic cache reuse.
func TestFormHierarchyCancellation(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	c, cancel := context.WithCancel(t.Context())
	cancel()
	err = validateFormHierarchyAndFields(c, ctx.XRefTable, nil, 20, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}
