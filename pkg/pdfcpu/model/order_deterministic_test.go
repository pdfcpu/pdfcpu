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
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	pdfcpuLog "github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func invalidLazyObject() types.Object {
	osd := types.NewObjectStreamDict()
	osd.Content = []byte{}
	return types.NewLazyObjectStreamObject(osd, 1, -1, nil)
}

func TestProcessRefCountsSelectsLowestDictionaryKey(t *testing.T) {
	conf := NewDefaultConfiguration()
	conf.Limits.MaxRecursionDepth = 1
	xRefTable := newXRefTable(conf)
	nested := types.Dict{"Nested": types.Dict{"Deep": types.Integer(1)}}
	d := types.Dict{"Zulu": nested, "Alpha": nested}

	for range 25 {
		err := ProcessRefCountsWithError(xRefTable, d)
		if err == nil || !strings.Contains(err.Error(), "dict entry Alpha") {
			t.Fatalf("expected Alpha reference-count error, got %v", err)
		}
		if !errors.Is(err, ErrMaxRecursionDepthExceeded) {
			t.Fatalf("expected recursion-depth cause, got %v", err)
		}
	}
}

func TestEqualObjectsSelectsLowestDictionaryKey(t *testing.T) {
	lazy := invalidLazyObject()
	d1 := types.Dict{"Zulu": lazy, "Alpha": lazy}
	d2 := types.Dict{"Zulu": lazy, "Alpha": lazy}
	xRefTable := newXRefTable(NewDefaultConfiguration())

	for range 25 {
		_, err := EqualObjects(t.Context(), d1, d2, xRefTable, nil)
		if err == nil || !strings.Contains(err.Error(), "dict entry Alpha") {
			t.Fatalf("expected Alpha comparison error, got %v", err)
		}
	}
}

func TestConsolidateResourcesSelectsLowestDictionaryKey(t *testing.T) {
	for range 25 {
		xRefTable := newXRefTable(NewDefaultConfiguration())
		xRefTable.Table[3] = NewXRefTableEntryGen0(invalidLazyObject())
		xRefTable.Table[9] = NewXRefTableEntryGen0(invalidLazyObject())
		d := types.Dict{
			"Zulu":  *types.NewIndirectRef(9, 0),
			"Alpha": *types.NewIndirectRef(3, 0),
		}

		err := xRefTable.consolidateResources(d, &InheritedPageAttrs{})
		if err == nil || !strings.Contains(err.Error(), "resource Alpha") {
			t.Fatalf("expected Alpha resource error, got %v", err)
		}
	}
}

func missingResourceNames() PageResourceNames {
	prn := NewPageResourceNames()
	prn["Font"]["Zulu"] = true
	prn["Font"]["Alpha"] = true
	return prn
}

func TestConsolidateResourceDictSelectsLowestResourceName(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	xRefTable.ValidationMode = ValidationStrict
	d := types.Dict{"Font": types.Dict{}}
	for range 25 {
		err := xRefTable.consolidateResourceDict(d, missingResourceNames(), 1)
		if err == nil || !strings.Contains(err.Error(), "missing required Font: Alpha") {
			t.Fatalf("expected Alpha resource error, got %v", err)
		}
	}
}

func TestConsolidateResourceDictPreservesCategoryPriority(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	xRefTable.ValidationMode = ValidationStrict
	prn := NewPageResourceNames()
	prn["Font"]["F1"] = true
	prn["ColorSpace"]["CS1"] = true

	err := xRefTable.consolidateResourceDict(types.Dict{}, prn, 1)
	if err == nil || !strings.Contains(err.Error(), "missing required ColorSpace resource dictionary") {
		t.Fatalf("expected ColorSpace category error, got %v", err)
	}
}

func TestConsolidateResourceDictReportsWarningsDeterministically(t *testing.T) {
	var buf bytes.Buffer
	pdfcpuLog.SetCLILogger(log.New(&buf, "", 0))
	defer pdfcpuLog.SetCLILogger(nil)

	xRefTable := newXRefTable(NewDefaultConfiguration())
	xRefTable.ValidationMode = ValidationRelaxed
	if err := xRefTable.consolidateResourceDict(types.Dict{"Font": types.Dict{}}, missingResourceNames(), 1); err != nil {
		t.Fatal(err)
	}

	s := buf.String()
	alpha := strings.Index(s, "missing required Font: Alpha")
	zulu := strings.Index(s, "missing required Font: Zulu")
	if alpha < 0 || zulu < 0 || alpha >= zulu {
		t.Fatalf("expected Alpha warning before Zulu warning, got %q", s)
	}
}
