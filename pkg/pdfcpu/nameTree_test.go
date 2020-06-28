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

package pdfcpu

import (
	"testing"
)

func checkAddResult(t *testing.T, r *Node, exp string, root bool) {

	l := r.String()

	if l != exp {
		t.Fatalf("Add b: %s != %s", l, exp)
	}

	if root {
		if !r.leaf() {
			t.Fatal("root only node should be a leaf node")
		}
		return
	}

	if r.leaf() {
		t.Fatal("root node with kids should not be a leaf node")
	}

}

func checkRemoveResult(t *testing.T, r *Node, k string, empty, ok bool, exp string, leaf bool) {

	if !ok {
		t.Fatalf("could not Remove %s\n", k)
	}

	if empty {
		t.Fatalf("r should not be empty after removing %s\n", k)
	}

	if leaf {
		if !r.leaf() {
			t.Fatalf("root node should be a leaf node after removing %s\n", k)
		}
	} else {
		if r.leaf() {
			t.Fatalf("root node with kids should not be a leaf node after removing %s\n", k)
		}
	}

	if empty {
		t.Fatalf("r should not be empty after removing %s\n", k)
	}

	l := r.String()
	if l != exp {
		t.Fatalf("Remove %s: %s != %s", k, l, exp)
	}
}

func buildNameTree(t *testing.T, r *Node) {

	r.Add(nil, "b", StringLiteral("bv"))
	checkAddResult(t, r, "[(b,(bv)){b,b}]", true)

	_, ok, _ := r.Remove(nil, "x")
	if ok {
		t.Fatal("should not be able to Remove x")
	}

	r.Add(nil, "f", StringLiteral("fv"))
	checkAddResult(t, r, "[(b,(bv))(f,(fv)){b,f}]", true)

	_, ok, _ = r.Remove(nil, "c")
	if ok {
		t.Fatal("should not be able to Remove c")
	}

	r.Add(nil, "d", StringLiteral("dv"))
	checkAddResult(t, r, "[(b,(bv))(d,(dv))(f,(fv)){b,f}]", true)

	_, ok = r.Value("c")
	if ok {
		t.Fatal("should not find Value for c")
	}

	r.Add(nil, "h", StringLiteral("hv"))
	checkAddResult(t, r, "{b,h},[(b,(bv))(d,(dv)){b,d}],[(f,(fv))(h,(hv)){f,h}]", false)

	r.Add(nil, "a", StringLiteral("av"))
	checkAddResult(t, r, "{a,h},[(a,(av))(b,(bv))(d,(dv)){a,d}],[(f,(fv))(h,(hv)){f,h}]", false)

	r.Add(nil, "i", StringLiteral("iv"))
	checkAddResult(t, r, "{a,i},[(a,(av))(b,(bv))(d,(dv)){a,d}],[(f,(fv))(h,(hv))(i,(iv)){f,i}]", false)

	r.Add(nil, "c", StringLiteral("cv"))
	checkAddResult(t, r, "{a,i},{a,d},[(a,(av))(b,(bv)){a,b}],[(c,(cv))(d,(dv)){c,d}],[(f,(fv))(h,(hv))(i,(iv)){f,i}]", false)
}

func destroyNameTree(t *testing.T, r *Node) {

	_, ok, _ := r.Remove(nil, "g")
	if ok {
		t.Fatal("should not be able to Remove g")
	}

	v, ok := r.Value("a")
	if !ok {
		t.Fatal("cannot find Value for a")
	}
	if v.String() != "(av)" {
		t.Fatalf("Value for a should be: %s but is %s", "av", v)
	}

	v, ok = r.Value("x")
	if ok {
		t.Fatal("should not find Value for x")
	}

	_, ok, _ = r.Remove(nil, "x")
	if ok {
		t.Fatal("should not be able to Remove x")
	}

	empty, ok, _ := r.Remove(nil, "b")
	checkRemoveResult(t, r, "b", empty, ok, "{a,i},{a,d},[(a,(av)){a,a}],[(c,(cv))(d,(dv)){c,d}],[(f,(fv))(h,(hv))(i,(iv)){f,i}]", false)

	empty, ok, _ = r.Remove(nil, "a")
	checkRemoveResult(t, r, "a", empty, ok, "{c,i},[(c,(cv))(d,(dv)){c,d}],[(f,(fv))(h,(hv))(i,(iv)){f,i}]", false)

	v, ok = r.Value("h")
	if !ok {
		t.Fatal("cannot find Value for h")
	}
	if v.String() != "(hv)" {
		t.Fatalf("Value for h should be: %s but is %s", "hv", v)
	}

	v, ok = r.Value("x")
	if ok {
		t.Fatal("should not find Value for x")
	}

	empty, ok, _ = r.Remove(nil, "h")
	checkRemoveResult(t, r, "h", empty, ok, "{c,i},[(c,(cv))(d,(dv)){c,d}],[(f,(fv))(i,(iv)){f,i}]", false)

	empty, ok, _ = r.Remove(nil, "i")
	checkRemoveResult(t, r, "i", empty, ok, "{c,f},[(c,(cv))(d,(dv)){c,d}],[(f,(fv)){f,f}]", false)

	empty, ok, _ = r.Remove(nil, "f")
	checkRemoveResult(t, r, "f", empty, ok, "[(c,(cv))(d,(dv)){c,d}]", true)

	v, ok = r.Value("x")
	if ok {
		t.Fatal("should not find Value for x")
	}

	r.Add(nil, "c", StringLiteral("cvv"))
	l := r.String()
	exp := "[(c,(cvv))(d,(dv)){c,d}]"
	if l != exp {
		t.Fatalf("update c: %s != %s", l, exp)
	}

	empty, ok, _ = r.Remove(nil, "c")
	checkRemoveResult(t, r, "c", empty, ok, "[(d,(dv)){d,d}]", true)

	empty, ok, _ = r.Remove(nil, "d")
	if !ok {
		t.Fatal("could not Remove d")
	}
	if !r.leaf() {
		t.Fatal("root node should be a leaf node after removing d")
	}
	if !empty {
		t.Fatal("r should be empty after removing f")
	}
	l = r.String()
	exp = "[{,}]"
	if l != exp {
		t.Fatalf("Remove d: %s != %s", l, exp)
	}
}

func TestNameTree(t *testing.T) {

	r := &Node{}
	buildNameTree(t, r)
	destroyNameTree(t, r)
}
