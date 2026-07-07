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
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestEncryptDictRejectsMissingReferencedObject verifies undefined references return an explicit cause.
func TestEncryptDictRejectsMissingReferencedObject(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	xRefTable.Encrypt = types.NewIndirectRef(7, 0)

	d, err := xRefTable.EncryptDict()
	if d != nil {
		t.Fatalf("expected no encryption dictionary, got %v", d)
	}
	if !errors.Is(err, ErrMissingEncryptDictObject) {
		t.Fatalf("got %v, want %v", err, ErrMissingEncryptDictObject)
	}
}

// TestEncryptDictRejectsWrongTypeObject verifies wrong-type objects retain a distinct cause.
func TestEncryptDictRejectsWrongTypeObject(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	xRefTable.Encrypt = types.NewIndirectRef(7, 0)
	xRefTable.Table[7] = NewXRefTableEntryGen0(types.Integer(1))

	d, err := xRefTable.EncryptDict()
	if d != nil {
		t.Fatalf("expected no encryption dictionary, got %v", d)
	}
	if !errors.Is(err, ErrWrongTypeEncryptDictObject) {
		t.Fatalf("got %v, want %v", err, ErrWrongTypeEncryptDictObject)
	}
	if !strings.Contains(err.Error(), "types.Integer") {
		t.Fatalf("missing resolved object type: %v", err)
	}
}
