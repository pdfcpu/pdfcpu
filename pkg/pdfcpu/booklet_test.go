package pdfcpu

import (
	"fmt"
	"testing"
)

type pageOrderResults struct {
	id                string
	nup               int
	pageCount         int
	expectedPageOrder []int
	papersize         string
	bookletType       string
	binding           string
}

var bookletTestCases = []pageOrderResults{
	// classic (booklet) test cases
	{
		id:        "portrait long edge",
		nup:       4,
		pageCount: 8,
		expectedPageOrder: []int{
			8, 1, 5, 4,
			2, 7, 3, 6,
		},
		papersize:   "A5", // portrait, long-edge binding
		bookletType: "booklet",
		binding:     "long",
	},
	{
		id:        "landscape short edge",
		nup:       4,
		pageCount: 8,
		expectedPageOrder: []int{
			8, 1, 5, 4,
			6, 3, 7, 2, // this is 180degrees flipped from the portrait layout (because of differences in how duplexing works)
		},
		papersize:   "A5L", // landscape, short-edge binding
		bookletType: "booklet",
		binding:     "short",
	},
	// topfold test cases
	{
		id:        "topfold portrait",
		nup:       4,
		pageCount: 8,
		expectedPageOrder: []int{
			8, 3, 1, 6,
			4, 7, 5, 2,
		},
		papersize:   "A5", // portrait, short-edge binding
		bookletType: "booklet",
		binding:     "short",
	},
	{
		id:        "topfold landscape",
		nup:       4,
		pageCount: 8,
		expectedPageOrder: []int{
			8, 3, 1, 6,
			2, 5, 7, 4,
		},
		papersize:   "A5L", // landscape, long-edge binding
		bookletType: "booklet",
		binding:     "long",
	},
	{
		id:        "topfold portrait multisheet",
		nup:       4,
		pageCount: 16,
		expectedPageOrder: []int{
			16, 3, 1, 14,
			4, 15, 13, 2,
			12, 7, 5, 10,
			8, 11, 9, 6,
		},
		papersize:   "A5", // portrait, short-edge binding
		bookletType: "booklet",
		binding:     "short",
	},
	// 6up test
	{
		id:        "6up",
		nup:       6,
		pageCount: 12,
		expectedPageOrder: []int{
			12, 1, 10, 3, 8, 5,
			2, 11, 4, 9, 6, 7,
		},
		papersize:   "A6", // portrait, long-edge binding
		bookletType: "booklet",
		binding:     "long",
	},
	{
		id:        "6up multisheet",
		nup:       6,
		pageCount: 24,
		expectedPageOrder: []int{
			24, 1, 22, 3, 20, 5,
			2, 23, 4, 21, 6, 19,
			18, 7, 16, 9, 14, 11,
			8, 17, 10, 15, 12, 13,
		},
		papersize:   "A6", // portrait, long-edge binding
		bookletType: "booklet",
		binding:     "long",
	},
	// perfect bound
	{
		id:        "perfect bound 2up",
		nup:       2,
		pageCount: 8,
		expectedPageOrder: []int{
			1, 3,
			2, 4,
			5, 7,
			6, 8,
		},
		papersize:   "A6", // portrait, long-edge binding
		bookletType: "perfectbound",
		binding:     "long",
	},
	{
		id:        "perfect bound 4up",
		nup:       4,
		pageCount: 16,
		expectedPageOrder: []int{
			1, 3, 5, 7,
			4, 2, 8, 6,
			9, 11, 13, 15,
			12, 10, 16, 14,
		},
		papersize:   "A6", // portrait, long-edge binding
		bookletType: "perfectbound",
		binding:     "long",
	},
	{
		id:        "perfect bound 4up landscape short-edge",
		nup:       4,
		pageCount: 16,
		expectedPageOrder: []int{
			1, 3, 5, 7,
			6, 8, 2, 4,
			9, 11, 13, 15,
			14, 16, 10, 12,
		},
		papersize:   "A6L", // landscape, short-edge binding
		bookletType: "perfectbound",
		binding:     "short",
	},
	{
		id:        "perfect bound 6up",
		nup:       6,
		pageCount: 12,
		expectedPageOrder: []int{
			1, 3, 5, 7, 9, 11,
			4, 2, 8, 6, 12, 10,
		},
		papersize:   "A6", // portrait, long-edge binding
		bookletType: "perfectbound",
		binding:     "long",
	},
}

func TestBookletPageOrder(t *testing.T) {
	for _, test := range bookletTestCases {
		t.Run(test.id, func(t *testing.T) {
			nup, err := PDFBookletConfig(test.nup, fmt.Sprintf("papersize:%s, btype:%s, binding: %s", test.papersize, test.bookletType, test.binding))
			if err != nil {
				t.Fatal(err)
			}
			pageNumbers := make(map[int]bool)
			for i := 0; i < test.pageCount; i++ {
				pageNumbers[i+1] = true
			}
			pageOrder := make([]int, test.pageCount)
			for i, p := range sortSelectedPagesForBooklet(pageNumbers, nup) {
				pageOrder[i] = p.number
			}
			for i, expected := range test.expectedPageOrder {
				if pageOrder[i] != expected {
					t.Fatal("incorrect page order\nexpected:", test.expectedPageOrder, "\ngot:", pageOrder)
				}
			}
		})
	}
}
