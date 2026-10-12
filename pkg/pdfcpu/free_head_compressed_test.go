package pdfcpu

import (
	"github.com/umats/pdfcpu/pkg/pdfcpu/model"
	"testing"
)

func TestFreeHeadCanonicalizationWithCompressedMembers(t *testing.T) {
	entry := func(free bool, offset int64, generation int) *model.XRefTableEntry {
		return &model.XRefTableEntry{Free: free, Offset: &offset, Generation: &generation}
	}
	makeTable := func() map[int]*model.XRefTableEntry {
		container, index := 1, 0
		return map[int]*model.XRefTableEntry{0: entry(true, 0, 0), 1: entry(false, 100, 0), 2: {Compressed: true, ObjectStream: &container, ObjectStreamInd: &index}}
	}
	table := makeTable()
	before := emptyFreeHeadSnapshot(table)
	if before == nil {
		t.Fatal("valid compressed member disqualified free-head normalization")
	}
	x := &model.XRefTable{Table: table}
	if err := x.EnsureValidFreeList(); err != nil {
		t.Fatal(err)
	}
	if !unchangedEmptyFreeHead(table, before) {
		t.Fatal("generation-only normalization rejected")
	}
	for _, name := range []string{"index", "container", "generation", "free", "offset", "missing-container", "compressed-container", "duplicate-member"} {
		t.Run(name, func(t *testing.T) {
			table := makeTable()
			switch name {
			case "index":
				*table[2].ObjectStreamInd = -1
			case "container":
				*table[2].ObjectStream = 2
			case "generation":
				v := 1
				table[2].Generation = &v
			case "free":
				table[2].Free = true
			case "offset":
				v := int64(99)
				table[2].Offset = &v
			case "missing-container":
				delete(table, 1)
			case "compressed-container":
				table[1].Compressed = true
			case "duplicate-member":
				e := *table[2]
				table[3] = &e
			}
			if emptyFreeHeadSnapshot(table) != nil {
				t.Fatal("ambiguous compressed identity accepted")
			}
		})
	}
	for _, name := range []string{"index", "container", "offset", "generation", "free"} {
		t.Run("mutation-"+name, func(t *testing.T) {
			table := makeTable()
			before := emptyFreeHeadSnapshot(table)
			*table[0].Generation = 65535
			switch name {
			case "index":
				*table[2].ObjectStreamInd = 1
			case "container":
				*table[2].ObjectStream = 9
			case "offset":
				*table[1].Offset = 101
			case "generation":
				*table[1].Generation = 1
			case "free":
				table[2].Free = true
			}
			if unchangedEmptyFreeHead(table, before) {
				t.Fatal("live identity mutation accepted")
			}
		})
	}
}
