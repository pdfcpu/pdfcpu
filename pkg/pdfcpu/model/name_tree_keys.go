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

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func (e entry) key() string {
	if e.raw != nil {
		return *e.raw
	}
	return e.k
}

// nameTreeEntry retains PDF string bytes separately from the existing decoded-string interface (7.9.6).
func nameTreeEntry(k string, v types.Object, d types.Dict, index int) entry {
	e := entry{k: k, v: v}
	a := d.ArrayEntry("Names")
	if index >= len(a) {
		return e
	}
	text, err := types.StringOrHexLiteral(a[index])
	if err != nil || *text != k {
		return e
	}
	var b []byte
	switch o := a[index].(type) {
	case types.StringLiteral:
		b, err = types.Unescape(o.Value())
	case types.HexLiteral:
		b, err = o.Bytes()
	}
	if err == nil {
		raw := string(b)
		e.raw = &raw
	}
	return e
}

func (n Node) rawLimits() (string, string) {
	first, last := &n, &n
	for len(first.Kids) > 0 {
		first = first.Kids[0]
	}
	for len(last.Kids) > 0 {
		last = last.Kids[len(last.Kids)-1]
	}
	lo, hi := n.Kmin, n.Kmax
	if len(first.Names) > 0 {
		lo = first.Names[0].key()
	}
	if len(last.Names) > 0 {
		hi = last.Names[len(last.Names)-1].key()
	}
	return lo, hi
}

func (n Node) entryForRawKey(c context.Context, k string) (entry, bool, error) {
	if err := contextutil.Check(c); err != nil {
		return entry{}, false, err
	}
	if !n.withinLimits(k) {
		return entry{}, false, nil
	}
	for _, e := range n.Names {
		if err := contextutil.Check(c); err != nil {
			return entry{}, false, err
		}
		if e.key() == k {
			return e, true, nil
		}
	}
	for _, kid := range n.Kids {
		if err := contextutil.Check(c); err != nil {
			return entry{}, false, err
		}
		if kid.withinLimits(k) {
			return kid.entryForRawKey(c, k)
		}
	}
	return entry{}, false, nil
}

// entryForText preserves decoded-string lookup without assuming decoded text retains PDF byte order.
func (n Node) entryForText(c context.Context, k string) (entry, bool, error) {
	for _, raw := range []string{k, types.EncodeUTF16String(k)} {
		e, found, err := n.entryForRawKey(c, raw)
		if err != nil || found && e.k == k {
			return e, found, err
		}
	}
	// Other encodings and relaxed, unsorted input need an equality search through the cached entries.
	stack := []*Node{&n}
	for len(stack) > 0 {
		if err := contextutil.Check(c); err != nil {
			return entry{}, false, err
		}
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, e := range node.Names {
			if err := contextutil.Check(c); err != nil {
				return entry{}, false, err
			}
			if e.k == k {
				return e, true, nil
			}
		}
		for i := len(node.Kids) - 1; i >= 0; i-- {
			stack = append(stack, node.Kids[i])
		}
	}
	return entry{}, false, nil
}

// Entries share a mutation index for their containing subtree. Keeping it on private entries preserves Node's API.
// Mutating any indexed child invalidates the shared index; decoded text is never used for PDF ordering.
type nameTreeTextIndex struct {
	owner *Node
	keys  map[string]string
	valid bool
}

func (n *Node) firstEntry() *entry {
	for len(n.Kids) > 0 {
		n = n.Kids[0]
	}
	if len(n.Names) == 0 {
		return nil
	}
	return &n.Names[0]
}

func (n *Node) clearTextIndex() {
	if e := n.firstEntry(); e != nil && e.index != nil {
		e.index.valid = false
	}
}

func (n *Node) indexTextKeys(c context.Context) (*nameTreeTextIndex, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if e := n.firstEntry(); e != nil && e.index != nil && e.index.owner == n && e.index.valid {
		return e.index, nil
	}
	n.clearTextIndex()
	index := &nameTreeTextIndex{owner: n, keys: map[string]string{}}
	stack := []*Node{n}
	for len(stack) > 0 {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if err := index.indexLeaf(c, node); err != nil {
			return nil, err
		}
		for i := len(node.Kids) - 1; i >= 0; i-- {
			stack = append(stack, node.Kids[i])
		}
	}
	index.valid = true
	return index, nil
}

func (index *nameTreeTextIndex) indexLeaf(c context.Context, n *Node) error {
	for i := range n.Names {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		e := &n.Names[i]
		if _, found := index.keys[e.k]; !found {
			index.keys[e.k] = e.key()
		}
		if e.index != nil {
			e.index.valid = false
		}
		e.index = index
	}
	return nil
}

func (n *Node) entryToAdd(c context.Context, e entry) (entry, *nameTreeTextIndex, error) {
	index, err := n.indexTextKeys(c)
	if err != nil {
		return e, nil, err
	}
	e.index = index
	if raw, found := index.keys[e.k]; found {
		// Preserve the existing decoded-name collision policy, including across different encodings.
		e.raw = &raw
	}
	return e, index, nil
}

func (n *Node) addEntry(c context.Context, x *XRefTable, e entry, m NameMap, refKeys []string) error {
	e, index, err := n.entryToAdd(c, e)
	if err != nil {
		return err
	}
	e, skip, err := uniqueNameTreeEntry(c, x, e, index.keys, m, refKeys)
	if err != nil || skip {
		return err
	}
	if err := n.add(c, x, e, m, refKeys); err != nil {
		index.valid = false
		return err
	}
	index.keys[e.k] = e.key()
	index.valid = true
	return nil
}

func uniqueNameTreeEntry(c context.Context, x *XRefTable, e entry, keys map[string]string, m NameMap, refKeys []string) (entry, bool, error) {
	if _, duplicate := keys[e.k]; !duplicate {
		return e, false, nil
	}
	dd, found := m[e.k]
	if !found {
		return e, true, nil
	}
	next := e.renamed()
	for {
		if err := contextutil.Check(c); err != nil {
			return e, false, err
		}
		if _, duplicate := keys[next.k]; !duplicate {
			break
		}
		next = next.renamed()
	}
	err := updateNameRefDicts(c, x, dd, refKeys, e.k, next.k, next.key())
	return next, false, err
}

func (e entry) renamed() entry {
	raw := e.key()
	if types.IsUTF16BE([]byte(raw)) {
		raw += "\x00\x01"
	} else {
		raw += "\x01"
	}
	e.k += "\x01"
	e.raw = &raw
	return e
}
