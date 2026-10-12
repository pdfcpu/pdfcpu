package pdfcpu

import (
	"testing"

	"github.com/umats/pdfcpu/pkg/pdfcpu/model"
)

// Malformed nil/compressed identities are not all representable through the
// parser. Exercise the internal classification without changing model repair.
func TestBUG013EmptyFreeHeadClassification(t *testing.T) {
	entry := func(free bool, offset int64, generation int) *model.XRefTableEntry {
		return &model.XRefTableEntry{Free: free, Offset: &offset, Generation: &generation}
	}
	for _, name := range []string{"accepted", "missing-head", "nil-head", "in-use-head", "compressed-head", "nil-offset", "nil-generation", "bad-link", "generation-one", "canonical-generation", "neighbor-free", "valid-linked-neighbor", "nil-live", "nil-live-offset", "nil-live-generation", "compressed-live", "ambiguous-head-container"} {
		t.Run(name, func(t *testing.T) {
			table := map[int]*model.XRefTableEntry{0: entry(true, 0, 0), 1: entry(false, 100, 0)}
			switch name {
			case "missing-head":
				delete(table, 0)
			case "nil-head":
				table[0] = nil
			case "in-use-head":
				table[0].Free = false
			case "compressed-head":
				table[0].Compressed = true
			case "nil-offset":
				table[0].Offset = nil
			case "nil-generation":
				table[0].Generation = nil
			case "bad-link":
				*table[0].Offset = 99
			case "generation-one":
				*table[0].Generation = 1
			case "canonical-generation":
				*table[0].Generation = 65535
			case "neighbor-free":
				table[2] = entry(true, 0, 1)
			case "valid-linked-neighbor":
				*table[0].Offset = 2
				table[2] = entry(true, 0, 1)
			case "nil-live":
				table[1] = nil
			case "nil-live-offset":
				table[1].Offset = nil
			case "nil-live-generation":
				table[1].Generation = nil
			case "compressed-live":
				table[1].Compressed = true
			case "ambiguous-head-container":
				v := 1
				table[0].ObjectStream = &v
			}
			before := emptyFreeHeadSnapshot(table)
			if (before != nil) != (name == "accepted") {
				t.Fatalf("unexpected classification: %v", before)
			}
			if name != "accepted" {
				return
			}
			x := &model.XRefTable{Table: table}
			if err := x.EnsureValidFreeList(); err != nil {
				t.Fatal(err)
			}
			if !unchangedEmptyFreeHead(table, before) {
				t.Fatal("sole canonicalization rejected")
			}
			for _, mutation := range []string{"live-offset", "live-generation", "live-free", "head-free", "head-compressed", "head-link", "head-generation", "new-entry", "missing-entry", "nil-entry", "container"} {
				t.Run(mutation, func(t *testing.T) {
					after := map[int]*model.XRefTableEntry{0: entry(true, 0, 65535), 1: entry(false, 100, 0)}
					switch mutation {
					case "live-offset":
						*after[1].Offset++
					case "live-generation":
						*after[1].Generation++
					case "live-free":
						after[1].Free = true
					case "head-free":
						after[0].Free = false
					case "head-compressed":
						after[0].Compressed = true
					case "head-link":
						*after[0].Offset = 1
					case "head-generation":
						*after[0].Generation = 1
					case "new-entry":
						after[2] = entry(false, 200, 0)
					case "missing-entry":
						delete(after, 1)
					case "nil-entry":
						after[1] = nil
					case "container":
						v := 1
						after[1].ObjectStreamInd = &v
					}
					if unchangedEmptyFreeHead(after, before) {
						t.Fatal("additional mutation accepted")
					}
				})
			}
		})
	}
}
