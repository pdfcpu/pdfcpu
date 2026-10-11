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
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func encodedNameTreeLeaf(t *testing.T, keys ...types.HexLiteral) *Node {
	t.Helper()
	a := types.Array{}
	for i, key := range keys {
		a = append(a, key, types.Integer(i+1))
	}
	n := &Node{D: types.Dict{"Names": a}}
	for i, key := range keys {
		text, err := types.StringOrHexLiteral(key)
		if err != nil {
			t.Fatal(err)
		}
		n.AppendToNames(*text, types.Integer(i+1))
		if i == 0 {
			n.Kmin = *text
		}
		n.Kmax = *text
	}
	return n
}

func checkNameTreeTextValue(t *testing.T, n *Node, key string, want types.Object) {
	t.Helper()
	got, found, err := n.Value(t.Context(), key)
	if err != nil || !found || got != want {
		t.Fatalf("lookup %q: value=%v found=%t error=%v; want %v", key, got, found, err, want)
	}
}

func checkNameTreeSerializedKeys(t *testing.T, n *Node, want map[string]string) {
	t.Helper()
	stack := []*Node{n}
	previous := ""
	count := 0
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for i := len(node.Kids) - 1; i >= 0; i-- {
			stack = append(stack, node.Kids[i])
		}
		if !node.leaf() {
			continue
		}
		d := types.NewDict()
		bindNameTreeLeafNode(d, node)
		a := d.ArrayEntry("Names")
		for i, e := range node.Names {
			b, err := a[2*i].(types.HexLiteral).Bytes()
			if err != nil {
				t.Fatal(err)
			}
			raw := string(b)
			if raw != want[e.k] || raw < previous {
				t.Fatalf("key %q: bytes=%x previous=%x want=%x", e.k, raw, previous, want[e.k])
			}
			previous = raw
			count++
		}
	}
	if count != len(want) {
		t.Fatalf("got %d keys, want %d", count, len(want))
	}
}

// TestNameTreeEncodedKeysPreserveTextAPI verifies decoded lookup and byte-preserving serialization.
func TestNameTreeEncodedKeysPreserveTextAPI(t *testing.T) {
	for _, keys := range [][]types.HexLiteral{
		{"FEFFD800DC00", "FEFFE000"},
		{"80", "81"},
	} {
		t.Run(fmt.Sprint(keys), func(t *testing.T) {
			n := encodedNameTreeLeaf(t, keys...)
			want := map[string]string{}
			for i, key := range keys {
				text, err := types.StringOrHexLiteral(key)
				if err != nil {
					t.Fatal(err)
				}
				b, err := key.Bytes()
				if err != nil {
					t.Fatal(err)
				}
				want[*text] = string(b)
				checkNameTreeTextValue(t, n, *text, types.Integer(i+1))
			}
			checkNameTreeSerializedKeys(t, n, want)
			c, cancel := context.WithCancel(t.Context())
			cancel()
			if _, _, err := n.Value(c, n.Kmin); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}

// TestNameTreeEncodedKeyMutations preserves key bytes through splits, merging, and removal.
func TestNameTreeEncodedKeyMutations(t *testing.T) {
	n := encodedNameTreeLeaf(t, "FEFFD800DC00", "FEFFE000")
	want := map[string]string{
		"\U00010000": types.EncodeUTF16String("\U00010000"),
		"\uE000":     types.EncodeUTF16String("\uE000"),
	}
	for i := 0; i < 32; i++ {
		key := fmt.Sprintf("ascii%02d", i)
		if err := n.Add(t.Context(), nil, key, types.Integer(i), nil, nil); err != nil {
			t.Fatal(err)
		}
		want[key] = key
	}
	src := encodedNameTreeLeaf(t, "FEFF0078")
	if err := n.AddTree(t.Context(), nil, src, nil, nil); err != nil {
		t.Fatal(err)
	}
	want["x"] = types.EncodeUTF16String("x")
	checkNameTreeTextValue(t, n, "\U00010000", types.Integer(1))
	checkNameTreeTextValue(t, n, "\uE000", types.Integer(2))
	checkNameTreeTextValue(t, n, "x", types.Integer(1))
	checkNameTreeSerializedKeys(t, n, want)
	for _, key := range []string{"\U00010000", "x", "ascii00"} {
		if _, found, err := n.Remove(t.Context(), nil, key); err != nil || !found {
			t.Fatalf("remove %q: found=%t error=%v", key, found, err)
		}
		delete(want, key)
		checkNameTreeSerializedKeys(t, n, want)
	}
	checkNameTreeTextValue(t, n, "\uE000", types.Integer(2))
}

// TestNameTreeEncodedKeyCollision preserves decoded-name collision handling and matching reference bytes.
func TestNameTreeEncodedKeyCollision(t *testing.T) {
	for _, merge := range []bool{false, true} {
		t.Run(fmt.Sprintf("merge=%t", merge), func(t *testing.T) {
			n := encodedNameTreeLeaf(t, "FEFFD800DC00")
			key := "\U00010000"
			// Reserve the first suffix using different PDF bytes but the same public text representation.
			if err := n.Add(t.Context(), nil, key+"\x01", types.Integer(3), nil, nil); err != nil {
				t.Fatal(err)
			}
			d := types.Dict{"Dest": types.NewHexLiteral([]byte(types.EncodeUTF16String(key)))}
			x := locateNameTreeXRefTable()
			refs := NameMap{key: {d}}
			if merge {
				source := &Node{}
				if err := source.Add(t.Context(), nil, key, types.Integer(2), nil, nil); err != nil {
					t.Fatal(err)
				}
				if err := n.AddTree(t.Context(), x, source, refs, []string{"Dest"}); err != nil {
					t.Fatal(err)
				}
			} else if err := n.Add(t.Context(), x, key, types.Integer(2), refs, []string{"Dest"}); err != nil {
				t.Fatal(err)
			}
			renamed := key + "\x01\x01"
			checkNameTreeTextValue(t, n, key, types.Integer(1))
			checkNameTreeTextValue(t, n, key+"\x01", types.Integer(3))
			checkNameTreeTextValue(t, n, renamed, types.Integer(2))
			want := map[string]string{
				key: types.EncodeUTF16String(key), key + "\x01": key + "\x01", renamed: types.EncodeUTF16String(renamed),
			}
			checkNameTreeSerializedKeys(t, n, want)
			raw, err := d["Dest"].(types.HexLiteral).Bytes()
			if err != nil || string(raw) != want[renamed] {
				t.Fatalf("reference=%x error=%v", raw, err)
			}
		})
	}
}

// TestNameTreeChildMutationsPreserveCollisionHandling checks cached key lookup follows public subtree mutations.
func TestNameTreeChildMutationsPreserveCollisionHandling(t *testing.T) {
	n := &Node{}
	for _, key := range []string{"a", "c", "e", "g"} {
		if err := n.Add(t.Context(), nil, key, types.Integer(1), nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	child := n.Kids[1]
	if err := child.Add(t.Context(), nil, "f", types.Integer(2), nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := n.Add(t.Context(), nil, "f", types.Integer(3), nil, nil); err != nil {
		t.Fatal(err)
	}
	checkNameTreeTextValue(t, n, "f", types.Integer(2))
	if _, found, err := child.Remove(t.Context(), nil, "f"); err != nil || !found {
		t.Fatalf("remove: found=%t error=%v", found, err)
	}
	if err := n.Add(t.Context(), nil, "f", types.Integer(3), nil, nil); err != nil {
		t.Fatal(err)
	}
	checkNameTreeTextValue(t, n, "f", types.Integer(3))
	checkNameTreeSerializedKeys(t, n, map[string]string{"a": "a", "c": "c", "e": "e", "f": "f", "g": "g"})
}
