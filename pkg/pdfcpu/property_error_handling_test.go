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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func propertyOperationTestContext(t *testing.T) *model.Context {
	t.Helper()

	f, err := os.Open(filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	ctx, err := Read(f, model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// TestPropertiesAddReportsInfoDictionaryDereference verifies malformed Info state does not panic.
func TestPropertiesAddReportsInfoDictionaryDereference(t *testing.T) {
	ctx := propertyOperationTestContext(t)
	indRef, err := ctx.IndRefForNewObject(types.Integer(1))
	if err != nil {
		t.Fatal(err)
	}
	ctx.Info = indRef
	version := model.V20
	ctx.HeaderVersion = &version
	ctx.RootVersion = nil

	err = PropertiesAdd(ctx, map[string]string{"name": "value"})
	if err == nil || !strings.Contains(err.Error(), "Info dictionary: dereference") {
		t.Fatalf("expected Info dictionary dereference context, got %v", err)
	}
}

// TestPropertiesAddReportsMissingInfoDictionary verifies PDF 2.0 missing Info state does not panic.
func TestPropertiesAddReportsMissingInfoDictionary(t *testing.T) {
	ctx := propertyOperationTestContext(t)
	version := model.V20
	ctx.HeaderVersion = &version
	ctx.RootVersion = nil
	ctx.Info = nil

	err := PropertiesAdd(ctx, map[string]string{"name": "value"})
	if err == nil || !strings.Contains(err.Error(), "Info dictionary: missing") {
		t.Fatalf("expected missing Info dictionary context, got %v", err)
	}
}

// TestPreparePropertiesInfoReportsDistinctPhases verifies preparation failures retain their owning phase.
func TestPreparePropertiesInfoReportsDistinctPhases(t *testing.T) {
	t.Run("Info dictionary ensure", func(t *testing.T) {
		ctx := propertyOperationTestContext(t)
		indRef, err := ctx.IndRefForNewObject(types.Integer(1))
		if err != nil {
			t.Fatal(err)
		}
		ctx.Info = indRef

		err = preparePropertiesInfo(ctx)
		if err == nil || !strings.Contains(err.Error(), "Info dictionary: ensure") {
			t.Fatalf("expected Info dictionary ensure context, got %v", err)
		}
		cause := errors.Unwrap(err)
		if cause == nil || !strings.Contains(cause.Error(), "wrong type") {
			t.Fatalf("expected preserved Info dictionary cause, got %v", err)
		}
	})

	t.Run("Info dictionary missing", func(t *testing.T) {
		ctx := propertyOperationTestContext(t)
		version := model.V20
		ctx.HeaderVersion = &version
		ctx.RootVersion = nil
		ctx.Info = nil

		err := preparePropertiesInfo(ctx)
		if err == nil || err.Error() != "Info dictionary: missing" {
			t.Fatalf("expected missing Info dictionary context, got %v", err)
		}
		if errors.Unwrap(err) != nil {
			t.Fatalf("expected cause-free missing Info dictionary error, got %v", err)
		}
	})

	t.Run("file ID ensure", func(t *testing.T) {
		ctx := propertyOperationTestContext(t)
		ctx.ID = types.Array{types.HexLiteral("one")}

		err := preparePropertiesInfo(ctx)
		if err == nil || !strings.Contains(err.Error(), "file ID: ensure") {
			t.Fatalf("expected file ID ensure context, got %v", err)
		}
		cause := errors.Unwrap(err)
		if cause == nil || !strings.Contains(cause.Error(), "id must be an array with 2 elements") {
			t.Fatalf("expected preserved file ID cause, got %v", err)
		}
	})
}

// TestPropertiesRemoveDeletesEveryMatch verifies named removal is not limited to the first match.
func TestPropertiesRemoveDeletesEveryMatch(t *testing.T) {
	ctx := propertyOperationTestContext(t)
	if err := PropertiesAdd(ctx, map[string]string{"alpha": "one", "beta": "two"}); err != nil {
		t.Fatal(err)
	}

	removed, err := PropertiesRemove(ctx, []string{"alpha", "beta"})
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Fatal("expected properties to be removed")
	}
	if _, found := ctx.Properties["alpha"]; found {
		t.Fatal("alpha was not removed")
	}
	if _, found := ctx.Properties["beta"]; found {
		t.Fatal("beta was not removed")
	}
}

// TestPropertiesRemoveReportsInfoDictionaryDereference verifies semantic remove context.
func TestPropertiesRemoveReportsInfoDictionaryDereference(t *testing.T) {
	ctx := propertyOperationTestContext(t)
	indRef, err := ctx.IndRefForNewObject(types.Integer(1))
	if err != nil {
		t.Fatal(err)
	}
	ctx.Info = indRef

	_, err = PropertiesRemove(ctx, []string{"name"})
	if err == nil || !strings.Contains(err.Error(), "Info dictionary: dereference") {
		t.Fatalf("expected Info dictionary dereference context, got %v", err)
	}
}

// TestPropertiesRemoveAllReportsCatalogContext verifies catalog failures retain domain context.
func TestPropertiesRemoveAllReportsCatalogContext(t *testing.T) {
	ctx := propertyOperationTestContext(t)
	ctx.Info = nil
	indRef, err := ctx.IndRefForNewObject(types.Integer(1))
	if err != nil {
		t.Fatal(err)
	}
	ctx.Root = indRef
	ctx.RootDict = nil

	_, err = PropertiesRemove(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "catalog: access") {
		t.Fatalf("expected catalog access context, got %v", err)
	}
}
