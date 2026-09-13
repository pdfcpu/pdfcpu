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

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
)

const nameTreeMaxKids = 4

func insertNameTreeSibling(c context.Context, parent, child, sibling *Node) error {
	for i, kid := range parent.Kids {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if kid != child {
			continue
		}
		parent.Kids = append(parent.Kids, nil)
		copy(parent.Kids[i+2:], parent.Kids[i+1:])
		parent.Kids[i+1] = sibling
		return nil
	}
	return errors.New("name tree split: parent does not contain child")
}

func nameTreeBranch(kids []*Node) *Node {
	return &Node{Kids: kids, Kmin: kids[0].Kmin, Kmax: kids[len(kids)-1].Kmax}
}

func promoteNameTreeSplit(c context.Context, path []*Node, n *Node) error {
	for i := len(path) - 1; i >= 0; i-- {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		parent := path[i]
		left, right := n.Kids[0], n.Kids[1]
		if err := insertNameTreeSibling(c, parent, n, right); err != nil {
			return err
		}
		// Retain the original node and its PDF dictionary as the left half.
		n.Kids, n.Names = left.Kids, left.Names
		n.Kmin, n.Kmax = left.Kmin, left.Kmax
		parent.Kmin, parent.Kmax = parent.Kids[0].Kmin, parent.Kids[len(parent.Kids)-1].Kmax
		if len(parent.Kids) != nameTreeMaxKids+1 {
			return nil
		}
		middle := len(parent.Kids) / 2
		left = nameTreeBranch(parent.Kids[:middle:middle])
		right = nameTreeBranch(parent.Kids[middle:])
		parent.Kids = []*Node{left, right}
		parent.Names = nil
		n = parent
	}
	return nil
}
