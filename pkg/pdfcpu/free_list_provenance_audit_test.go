package pdfcpu

import (
	"testing"

	"github.com/umats/pdfcpu/pkg/pdfcpu/model"
)

// Characterizes existing model transformations without authorizing hint repair.
// Offset is a free link for free entries, but a physical origin for live ones.
func TestBUG013FreeListTransformations(t *testing.T) {
	type identity struct {
		free       bool
		offset     int64
		generation int
	}
	entry := func(i identity) *model.XRefTableEntry {
		return &model.XRefTableEntry{Free: i.free, Offset: &i.offset, Generation: &i.generation}
	}
	for _, tc := range []struct {
		name string
		in   map[int]identity
		want map[int]identity
	}{
		{"head-generation-only", map[int]identity{0: {true, 0, 0}}, map[int]identity{0: {true, 0, 65535}}},
		{"valid-neighbor", map[int]identity{0: {true, 2, 0}, 2: {true, 0, 1}}, map[int]identity{0: {true, 2, 65535}, 2: {true, 0, 1}}},
		{"missing-head", map[int]identity{}, map[int]identity{0: {true, 0, 65535}}},
		{"in-use-head", map[int]identity{0: {false, 100, 0}}, map[int]identity{0: {false, 0, 65535}}},
		{"bad-head-link", map[int]identity{0: {true, 99, 65535}}, map[int]identity{0: {true, 0, 65535}}},
		{"link-to-live", map[int]identity{0: {true, 1, 65535}, 2: {true, 0, 1}}, map[int]identity{0: {true, 2, 65535}, 2: {true, 0, 1}}},
		{"dangling-neighbor", map[int]identity{0: {true, 0, 65535}, 2: {true, 99, 1}}, map[int]identity{0: {true, 2, 65535}, 2: {true, 0, 1}}},
		{"cycle", map[int]identity{0: {true, 2, 65535}, 2: {true, 2, 1}}, map[int]identity{0: {true, 2, 65535}, 2: {true, 0, 1}}},
		{"dead-dangling", map[int]identity{0: {true, 0, 65535}, 2: {true, 99, 65535}}, map[int]identity{0: {true, 0, 65535}, 2: {true, 0, 65535}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := &model.XRefTable{Table: map[int]*model.XRefTableEntry{1: entry(identity{false, 100, 0})}}
			for nr, i := range tc.in {
				x.Table[nr] = entry(i)
			}
			if err := x.EnsureValidFreeList(); err != nil {
				t.Fatal(err)
			}
			for nr, want := range tc.want {
				e := x.Table[nr]
				got := identity{e.Free, *e.Offset, *e.Generation}
				if got != want {
					t.Fatalf("object %d: got %+v, want %+v", nr, got, want)
				}
			}
			live := x.Table[1]
			if live.Free || *live.Offset != 100 || *live.Generation != 0 {
				t.Fatal("neighboring live identity mutated")
			}
		})
	}
}
