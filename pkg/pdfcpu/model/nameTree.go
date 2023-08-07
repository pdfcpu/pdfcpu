/*
Copyright 2018 The pdfcpu Authors.

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
	"fmt"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

var errNameTreeDuplicateKey = errors.New("pdfcpu: name: duplicate key")

const maxEntries = 3

// Node is an opinionated implementation of the PDF name tree.
// pdfcpu caches all name trees found in the PDF catalog with this data structure.
// The PDF spec does not impose any rules regarding a strategy for the creation of nodes.
// A binary tree was chosen where each leaf node has a limited number of entries (maxEntries).
// Once maxEntries has been reached a leaf node turns into an intermediary node with two kids,
// which are leaf nodes each of them holding half of the sorted entries of the original leaf node.
type Node struct {
	Kids       []*Node    // Mirror of the name tree's Kids array, an array of indirect references.
	Names      []entry    // Mirror of the name tree's Names array.
	Kmin, Kmax string     // Mirror of the name tree's Limit array[Kmin,Kmax].
	D          types.Dict // The PDF dict representing this name tree node.
}

// entry is a key value pair.
type entry struct {
	k string
	v types.Object
}

func (n Node) leaf() bool {
	return n.Kids == nil
}

func keyLess(k, s string) bool {
	return k < s
}

func keyLessOrEqual(k, s string) bool {
	return k == s || keyLess(k, s)
}

func (n Node) withinLimits(k string) bool {
	return keyLessOrEqual(n.Kmin, k) && keyLessOrEqual(k, n.Kmax)
}

// Value returns the value for given key
func (n Node) Value(k string) (types.Object, bool) {

	if !n.withinLimits(k) {
		return nil, false
	}

	if n.leaf() {

		// names are sorted by key.
		for _, v := range n.Names {

			if v.k < k {
				continue
			}

			if v.k == k {
				return v.v, true
			}

			return nil, false
		}

		return nil, false
	}

	// kids are sorted by key ranges.
	for _, v := range n.Kids {
		if v.withinLimits(k) {
			return v.Value(k)
		}
	}

	return nil, false
}

// AppendToNames adds an entry to a leaf node (for internalizing name trees).
func (n *Node) AppendToNames(k string, v types.Object) {

	//fmt.Printf("AddToLeaf: %s %v (%v)\n\n", k, v, &v)

	if n.Names == nil {
		n.Names = make([]entry, 0, maxEntries)
	}

	arr, ok := v.(types.Array)
	if ok {
		arr1 := make(types.Array, len(arr))
		for i, v := range arr {
			if indRef, ok := v.(types.IndirectRef); ok {
				arr1[i] = *types.NewIndirectRef(indRef.ObjectNumber.Value(), indRef.GenerationNumber.Value())
			} else {
				arr1[i] = v
			}
		}
		n.Names = append(n.Names, entry{k, arr1})
	} else {
		n.Names = append(n.Names, entry{k, v})
	}
}

type NameMap map[string][]types.Dict

func (m NameMap) Add(k string, d types.Dict) {
	dd, ok := m[k]
	if !ok {
		m[k] = []types.Dict{d}
		return
	}
	m[k] = append(dd, d)
}

func (n *Node) insertIntoLeaf(k string, v types.Object, m NameMap) error {
	log.Debug.Printf("Insert k:%s in the middle\n", k)
	for i, e := range n.Names {
		if keyLess(e.k, k) {
			continue
		}
		if e.k == k {
			return errNameTreeDuplicateKey
		}
		// Insert entry(k,v) at i
		n.Names = append(n.Names, entry{})
		copy(n.Names[i+1:], n.Names[i:])
		n.Names[i] = entry{k, v}
		return nil
	}
	log.Debug.Printf("Insert k:%s at end\n", k)
	n.Kmax = k
	n.Names = append(n.Names, entry{k, v})
	return nil
}

func updateNameRef(d types.Dict, keys []string, nameOld, nameNew string) error {
	for _, k := range keys {
		s, err := d.StringOrHexLiteralEntry(k)
		if err != nil {
			return err
		}
		if s != nil {
			if *s != nameOld {
				return errors.Errorf("invalid Name ref detected for: %s", nameOld)
			}
			d[k] = types.NewHexLiteral([]byte(nameNew))
		}
	}
	return nil
}

func updateNameRefDicts(dd []types.Dict, nameRefDictKeys []string, nameOld, nameNew string) error {
	// eg.
	// "Dests": "D", "Dest"    		[]string{"D", "Dest"}
	// "EmbeddedFiles": F", "UF"	[]string{"F", "UF"}

	for _, d := range dd {
		if err := updateNameRef(d, nameRefDictKeys, nameOld, nameNew); err != nil {
			return err
		}
	}

	return nil
}

// HandleLeaf processes a leaf node.
func (n *Node) HandleLeaf(xRefTable *XRefTable, k string, v types.Object, m NameMap, nameRefDictKeys []string) error {
	// A leaf node contains up to maxEntries names.
	// Any number of entries greater than maxEntries will be delegated to kid nodes.

	//fmt.Printf("HandleLeaf: %s %v\n\n", k, v)

	if len(n.Names) == 0 {
		n.Names = append(n.Names, entry{k, v})
		n.Kmin, n.Kmax = k, k
		log.Debug.Printf("first key=%s\n", k)
		return nil
	}

	log.Debug.Printf("kmin=%s kmax=%s\n", n.Kmin, n.Kmax)

	if keyLess(k, n.Kmin) {
		// Prepend (k,v).
		log.Debug.Printf("Insert k:%s at beginning\n", k)
		n.Kmin = k
		n.Names = append(n.Names, entry{})
		copy(n.Names[1:], n.Names[0:])
		n.Names[0] = entry{k, v}
	} else if keyLess(n.Kmax, k) {
		// Append (k,v).
		log.Debug.Printf("Insert k:%s at end\n", k)
		n.Kmax = k
		n.Names = append(n.Names, entry{k, v})
	} else {
		// Insert (k,v) while ensuring unique k.
		var err error
		kOrig := k
		for first := true; first || err == errNameTreeDuplicateKey; first = false {
			err = n.insertIntoLeaf(k, v, m)
			if err == nil {
				break
			}
			if len(m) == 0 {
				return err
			}
			if err != errNameTreeDuplicateKey {
				return err
			}
			kNew := k + "\x01"
			dd, ok := m[kOrig]
			if !ok {
				return nil
				//return errors.Errorf("unreferenced Name detected for: %s", k)
			}
			if err := updateNameRefDicts(dd, nameRefDictKeys, k, kNew); err != nil {
				return err
			}
			k = kNew
		}
	}

	// if len was already > maxEntries we know we are dealing with somebody elses name tree.
	// In that case we do not know the branching strategy and therefore just add to Names and do not create kids.
	// If len is within maxEntries we do not create kids either way.
	if len(n.Names) != maxEntries+1 {
		return nil
	}

	// turn leaf into intermediate node with 2 kids/leafs (binary tree)
	c := maxEntries + 1
	k1 := &Node{Names: make([]entry, c/2, maxEntries)}
	copy(k1.Names, n.Names[:c/2])
	k1.Kmin = n.Names[0].k
	k1.Kmax = n.Names[c/2-1].k

	k2 := &Node{Names: make([]entry, len(n.Names)-c/2, maxEntries)}
	copy(k2.Names, n.Names[c/2:])
	k2.Kmin = n.Names[c/2].k
	k2.Kmax = n.Names[c-1].k

	n.Kids = []*Node{k1, k2}
	n.Names = nil

	return nil
}

// Add adds an entry to a name tree.
func (n *Node) Add(xRefTable *XRefTable, k string, v types.Object, m NameMap, nameRefDictKeys []string) error {

	//fmt.Printf("Add: %s %v\n", k, v)

	// The values associated with the keys may be objects of any type.
	// Stream objects shall be specified by indirect object references.
	// Dictionary, array, and string objects should be specified by indirect object references.
	// Other PDF objects (null, number, boolean and name) should be specified as direct objects.

	if n.Names == nil {
		n.Names = make([]entry, 0, maxEntries)
	}

	if n.leaf() {
		return n.HandleLeaf(xRefTable, k, v, m, nameRefDictKeys)
	}

	if k == n.Kmin || k == n.Kmax {
		return nil
	}

	if keyLess(k, n.Kmin) {
		n.Kmin = k
	} else if keyLess(n.Kmax, k) {
		n.Kmax = k
	}

	// For intermediary nodes we delegate to the corresponding subtree.
	for _, a := range n.Kids {
		if keyLess(k, a.Kmin) || a.withinLimits(k) {
			return a.Add(xRefTable, k, v, m, nameRefDictKeys)
		}
	}

	// Insert k into last (right most) subtree.
	last := n.Kids[len(n.Kids)-1]
	return last.Add(xRefTable, k, v, m, nameRefDictKeys)
}

// AddTree adds a name tree to a name tree.
func (n *Node) AddTree(xRefTable *XRefTable, tree *Node, m NameMap, nameRefDictKeys []string) error {

	if !tree.leaf() {
		for _, v := range tree.Kids {
			if err := n.AddTree(xRefTable, v, m, nameRefDictKeys); err != nil {
				return err
			}
		}
		return nil
	}

	for _, e := range tree.Names {
		if err := n.Add(xRefTable, e.k, e.v, m, nameRefDictKeys); err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) removeFromNames(xRefTable *XRefTable, k string) (ok bool, err error) {

	for i, v := range n.Names {

		if v.k < k {
			continue
		}

		if v.k == k {

			if xRefTable != nil {
				// Remove object graph of value.
				log.Debug.Println("removeFromNames: deleting object graph of v")
				err := xRefTable.DeleteObjectGraph(v.v)
				if err != nil {
					return false, err
				}
			}

			n.Names = append(n.Names[:i], n.Names[i+1:]...)
			return true, nil
		}

	}

	return false, nil
}

func (n *Node) removeFromLeaf(xRefTable *XRefTable, k string) (empty, ok bool, err error) {

	if keyLess(k, n.Kmin) || keyLess(n.Kmax, k) {
		return false, false, nil
	}

	// kmin < k < kmax

	// If sole entry gets deleted, remove this node from parent.
	if len(n.Names) == 1 {
		if xRefTable != nil {
			// Remove object graph of value.
			log.Debug.Println("removeFromLeaf: deleting object graph of v")
			err := xRefTable.DeleteObjectGraph(n.Names[0].v)
			if err != nil {
				return false, false, err
			}
		}
		n.Kmin, n.Kmax = "", ""
		n.Names = nil
		return true, true, nil
	}

	if k == n.Kmin {

		if xRefTable != nil {
			// Remove object graph of value.
			log.Debug.Println("removeFromLeaf: deleting object graph of v")
			err := xRefTable.DeleteObjectGraph(n.Names[0].v)
			if err != nil {
				return false, false, err
			}
		}

		n.Names = n.Names[1:]
		n.Kmin = n.Names[0].k
		return false, true, nil
	}

	if k == n.Kmax {

		if xRefTable != nil {
			// Remove object graph of value.
			log.Debug.Println("removeFromLeaf: deleting object graph of v")
			err := xRefTable.DeleteObjectGraph(n.Names[len(n.Names)-1].v)
			if err != nil {
				return false, false, err
			}
		}

		n.Names = n.Names[:len(n.Names)-1]
		n.Kmax = n.Names[len(n.Names)-1].k
		return false, true, nil
	}

	ok, err = n.removeFromNames(xRefTable, k)
	if err != nil {
		return false, false, err
	}

	return false, ok, nil
}

func (n *Node) removeFromKids(xRefTable *XRefTable, k string) (ok bool, err error) {

	// Locate the kid to recurse into, then remove k from that subtree.
	for i, kid := range n.Kids {

		if !kid.withinLimits(k) {
			continue
		}

		empty, ok, err := kid.Remove(xRefTable, k)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}

		if empty {

			// This kid is now empty and needs to be removed.

			if xRefTable != nil {
				err = xRefTable.DeleteObject(kid.D)
				if err != nil {
					return false, err
				}
			}

			if i == 0 {
				// Remove first kid.
				log.Debug.Println("removeFromKids: remove first kid.")
				n.Kids = n.Kids[1:]
			} else if i == len(n.Kids)-1 {
				log.Debug.Println("removeFromKids: remove last kid.")
				// Remove last kid.
				n.Kids = n.Kids[:len(n.Kids)-1]
			} else {
				// Remove kid from the middle.
				log.Debug.Println("removeFromKids: remove kid form the middle.")
				n.Kids = append(n.Kids[:i], n.Kids[i+1:]...)
			}

			if len(n.Kids) == 1 {

				// If a single kid remains we can merge it with its parent.
				// By doing this we get rid of a redundant intermediary node.
				log.Debug.Println("removeFromKids: only 1 kid")

				if xRefTable != nil {
					err = xRefTable.DeleteObject(n.D)
					if err != nil {
						return false, err
					}
				}

				*n = *n.Kids[0]

				log.Debug.Printf("removeFromKids: new n = %s\n", n)

				return true, nil
			}

		}

		// Update kMin, kMax for n.
		n.Kmin = n.Kids[0].Kmin
		n.Kmax = n.Kids[len(n.Kids)-1].Kmax

		return true, nil
	}

	return false, nil
}

// Remove removes an entry from a name tree.
// empty returns true if this node is an empty leaf node after removal.
// ok returns true if removal was successful.
func (n *Node) Remove(xRefTable *XRefTable, k string) (empty, ok bool, err error) {

	if n.leaf() {
		return n.removeFromLeaf(xRefTable, k)
	}

	ok, err = n.removeFromKids(xRefTable, k)
	if err != nil {
		return false, false, err
	}

	return false, ok, nil

}

// Process traverses the nametree applying a handler to each entry (key-value pair).
func (n *Node) Process(xRefTable *XRefTable, handler func(*XRefTable, string, *types.Object) error) error {

	if !n.leaf() {
		for _, v := range n.Kids {
			if err := v.Process(xRefTable, handler); err != nil {
				return err
			}
		}
		return nil
	}

	for k, e := range n.Names {
		if err := handler(xRefTable, e.k, &e.v); err != nil {
			return err
		}
		n.Names[k] = e
	}

	return nil
}

// KeyList returns a sorted list of all keys.
func (n Node) KeyList() ([]string, error) {

	list := []string{}

	keys := func(xRefTable *XRefTable, k string, v *types.Object) error {
		list = append(list, fmt.Sprintf("%s %v", k, *v))
		return nil
	}

	err := n.Process(nil, keys)
	if err != nil {
		return nil, err
	}

	return list, nil

}

func (n Node) String() string {

	a := []string{}

	if n.leaf() {
		a = append(a, "[")
		for _, n := range n.Names {
			a = append(a, fmt.Sprintf("(%s,%s)", n.k, n.v))
		}
		a = append(a, fmt.Sprintf("{%s,%s}]", n.Kmin, n.Kmax))
		return strings.Join(a, "")
	}

	a = append(a, fmt.Sprintf("{%s,%s}", n.Kmin, n.Kmax))

	for _, v := range n.Kids {
		a = append(a, v.String())
	}

	return strings.Join(a, ",")
}
