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
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func testMergeContext(t *testing.T) *model.Context {
	t.Helper()
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	ctx, err := ReadFile(filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf"), conf)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestMergeXRefTablesPageTreeErrorIncludesMergeContext(t *testing.T) {
	src := testMergeContext(t)
	dest := testMergeContext(t)
	dest.PageCount++

	err := MergeXRefTables("source.pdf", src, dest, false, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "merge page tree: append source pages") {
		t.Fatalf("expected merge page tree context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "corrupt page node") {
		t.Fatalf("expected lower page tree context, got %q", err.Error())
	}
}

func TestMergeXRefTablesZipPageTreeErrorIncludesMergeContext(t *testing.T) {
	src := testMergeContext(t)
	dest := testMergeContext(t)
	dest.RootDict = nil
	dest.Root = nil

	err := MergeXRefTables("source.pdf", src, dest, true, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "merge page tree: zip source pages") {
		t.Fatalf("expected zip merge page tree context, got %q", err.Error())
	}
}

func TestMergeXRefTablesNilContextErrors(t *testing.T) {
	err := MergeXRefTables("source.pdf", nil, testMergeContext(t), false, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "merge: missing context") {
		t.Fatalf("expected missing context error, got %q", err.Error())
	}

	src := testMergeContext(t)
	src.Root = nil
	err = MergeXRefTables("source.pdf", src, testMergeContext(t), false, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "missing source root") {
		t.Fatalf("expected missing source root error, got %q", err.Error())
	}
}

func TestEnsureOutlinesNilContextReturnsError(t *testing.T) {
	err := EnsureOutlines(nil, "source.pdf", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "ensure outlines: missing context") {
		t.Fatalf("expected missing context error, got %q", err.Error())
	}
}
