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
	"math/rand"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func checkBalancedNameTree(t *testing.T, root *Node, count int) map[*Node]bool {
	t.Helper()
	type frame struct {
		n     *Node
		depth int
	}
	stack := []frame{{root, 0}}
	nodes := map[*Node]bool{}
	leafDepth := -1
	entries := 0
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		nodes[f.n] = true
		if f.n.leaf() {
			entries += len(f.n.Names)
			if leafDepth < 0 {
				leafDepth = f.depth
			}
			if f.depth != leafDepth {
				t.Fatalf("unequal leaf depths: %d and %d", leafDepth, f.depth)
			}
			continue
		}
		if len(f.n.Kids) > nameTreeMaxKids {
			t.Fatalf("unexpected fanout: %d", len(f.n.Kids))
		}
		if f.n.Kmin != f.n.Kids[0].Kmin || f.n.Kmax != f.n.Kids[len(f.n.Kids)-1].Kmax {
			t.Fatal("incorrect branch limits")
		}
		for _, kid := range f.n.Kids {
			stack = append(stack, frame{kid, f.depth + 1})
		}
	}
	if entries != count || leafDepth > 12 {
		t.Fatalf("entries=%d depth=%d", entries, leafDepth)
	}
	for i := 0; i < count; i++ {
		value, ok, err := root.Value(t.Context(), fmt.Sprintf("key%04d", i))
		if err != nil || !ok || value != types.Integer(i) {
			t.Fatalf("key %d: %v %t %v", i, value, ok, err)
		}
	}
	return nodes
}

// TestNameTreeInsertionStaysBalanced verifies insertion order cannot create a deep chain or lose values.
func TestNameTreeInsertionStaysBalanced(t *testing.T) {
	const count = 2048
	for _, order := range []string{"ascending", "descending", "shuffled"} {
		t.Run(order, func(t *testing.T) {
			indices := make([]int, count)
			for i := range indices {
				indices[i] = i
			}
			if order == "descending" {
				for i := range indices {
					indices[i] = count - 1 - i
				}
			}
			if order == "shuffled" {
				rand.New(rand.NewSource(0)).Shuffle(count, func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })
			}
			root := &Node{}
			for _, i := range indices {
				if err := root.Add(t.Context(), nil, fmt.Sprintf("key%04d", i), types.Integer(i), nil, nil); err != nil {
					t.Fatal(err)
				}
			}
			checkBalancedNameTree(t, root, count)
		})
	}
}

// TestNameTreeSplitRetainsDictionaries verifies promoting a split preserves existing dictionary identity.
func TestNameTreeSplitRetainsDictionaries(t *testing.T) {
	rootDict := types.Dict{}
	root := &Node{D: rootDict}
	for i := 0; i < 4; i++ {
		if err := root.Add(t.Context(), nil, fmt.Sprintf("key%04d", i), types.Integer(i), nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	child := root.Kids[1]
	childDict := types.Dict{}
	child.D = childDict
	for i := 4; i < 64; i++ {
		if err := root.Add(t.Context(), nil, fmt.Sprintf("key%04d", i), types.Integer(i), nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	rootDict["Marker"] = types.Integer(1)
	childDict["Marker"] = types.Integer(2)
	if root.D["Marker"] != types.Integer(1) || child.D["Marker"] != types.Integer(2) {
		t.Fatal("dictionary identity changed")
	}
	nodes := checkBalancedNameTree(t, root, 64)
	if !nodes[child] {
		t.Fatal("original child was detached")
	}
}

// TestNameTreeInsertionCancellation verifies a canceled insertion leaves existing values available.
func TestNameTreeInsertionCancellation(t *testing.T) {
	root := &Node{}
	for i := 0; i < 204; i++ {
		if err := root.Add(t.Context(), nil, fmt.Sprintf("key%04d", i), types.Integer(i), nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	c, cancel := context.WithCancel(t.Context())
	cancel()
	if err := root.Add(c, nil, "new", types.Integer(204), nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	checkBalancedNameTree(t, root, 204)
}
