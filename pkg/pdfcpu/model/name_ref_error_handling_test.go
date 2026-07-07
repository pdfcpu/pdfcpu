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
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func requireNameRefUpdateContext(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{`entry "Dest"`, `key "old"`, `"new"`} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
}

func TestUpdateNameRefPreservesStringParsingError(t *testing.T) {
	sl := types.StringLiteral(string([]byte{0xFE, 0xFF, 0xD8, 0x00}))
	_, cause := types.StringLiteralToString(sl)
	if cause == nil {
		t.Fatal("expected string parsing error")
	}
	d := types.Dict{
		"Dest": sl,
	}

	err := updateNameRef(d, []string{"Dest"}, "old", "new")

	root := err
	for errors.Unwrap(root) != nil {
		root = errors.Unwrap(root)
	}
	if root.Error() != cause.Error() {
		t.Fatalf("expected root cause %q, got %q", cause, root)
	}
	requireNameRefUpdateContext(t, err)
}

func TestUpdateNameRefPreservesHexParsingError(t *testing.T) {
	d := types.Dict{
		"Dest": types.HexLiteral("zz"),
	}

	err := updateNameRef(d, []string{"Dest"}, "old", "new")

	var invalidByte hex.InvalidByteError
	if !errors.As(err, &invalidByte) {
		t.Fatalf("expected hex.InvalidByteError, got %v", err)
	}
	requireNameRefUpdateContext(t, err)
}

func TestUpdateNameRefMismatchIncludesEntryAndKeys(t *testing.T) {
	d := types.Dict{
		"Dest": types.StringLiteral("different"),
	}

	err := updateNameRef(d, []string{"Dest"}, "old", "new")

	requireNameRefUpdateContext(t, err)
}

func TestInsertUniqueIntoLeafUsesDuplicateKeySentinel(t *testing.T) {
	d := types.Dict{"Dest": types.StringLiteral("old")}
	n := &Node{
		Kmin:  "old",
		Kmax:  "old",
		Names: []entry{{k: "old", v: types.StringLiteral("value")}},
	}

	duplicate, err := n.insertUniqueIntoLeaf(
		"old",
		types.StringLiteral("replacement"),
		NameMap{"old": []types.Dict{d}},
		[]string{"Dest"},
	)

	if err != nil {
		t.Fatal(err)
	}
	if duplicate {
		t.Fatal("expected duplicate key to be renamed")
	}
	if _, found := n.Value("old\x01"); !found {
		t.Fatal("expected renamed name-tree key")
	}
	s, err := d.StringOrHexLiteralEntry("Dest")
	if err != nil {
		t.Fatal(err)
	}
	if s == nil || *s != "old\x01" {
		t.Fatalf("expected updated name reference, got %v", s)
	}
}
