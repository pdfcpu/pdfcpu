/*
Copyright 2024 The pdfcpu Authors.

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
	"fmt"
	"strings"
	"testing"
)

type pageOrderResults struct {
	id                 string
	nup                int
	pageCount          int
	expectedPageOrder  []int
	papersize          string
	bookletType        string
	binding            string
	useSignatures      bool
	nPagesPerSignature int
}

var bookletTestCases = []pageOrderResults{
	{
		id:        "2up",
		nup:       2,
		pageCount: 16,
		expectedPageOrder: []int{
			16, 1,
			15, 2,
			14, 3,
			13, 4,
			12, 5,
			11, 6,
			10, 7,
			9, 8,
		},
		papersize:   "A6",
		bookletType: "booklet",
		binding:     "long",
	},
	{
		id:        "2up with trailing blank pages",
		nup:       2,
		pageCount: 10,
		expectedPageOrder: []int{
			0, 1,
			0, 2,
			10, 3,
			9, 4,
			8, 5,
			7, 6,
		},
		papersize:   "A6",
		bookletType: "booklet",
		binding:     "long",
	},
	// basic booklet sidefold test cases
	{
		id:        "booklet portrait long edge",
		nup:       4,
		pageCount: 16,
		expectedPageOrder: []int{
			16, 1, 3, 14,
			2, 15, 13, 4,
			12, 5, 7, 10,
			6, 11, 9, 8,
		},
		papersize:   "A5", // portrait, long-edge binding
		bookletType: "booklet",
		binding:     "long",
	},
	{
		id:        "booklet landscape short edge",
		nup:       4,
		pageCount: 8,
		expectedPageOrder: []int{
			8, 1, 3, 6,
			4, 5, 7, 2, // this is ordered differently from the portrait layout (because of differences in how duplexing works)
		},
		papersize:   "A5L", // landscape, short-edge binding
		bookletType: "booklet",
		binding:     "short",
	},
	// basic booklet topfold test cases
	{
		id:        "booklet topfold portrait",
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
	{
		id:        "booklet topfold landscape",
		nup:       4,
		pageCount: 8,
		expectedPageOrder: []int{
			8, 3, 1, 6,
			2, 5, 7, 4, // this is 180degrees flipped from the portrait layout (because of differences in how duplexing works)
		},
		papersize:   "A5L", // landscape, long-edge binding
		bookletType: "booklet",
		binding:     "long",
	},
	// advanced booklet sidefold test cases
	{
		id:        "advanced portrait long edge",
		nup:       4,
		pageCount: 8,
		expectedPageOrder: []int{
			8, 1, 5, 4,
			2, 7, 3, 6,
		},
		papersize:   "A5", // portrait, long-edge binding
		bookletType: "bookletadvanced",
		binding:     "long",
	},
	{
		id:        "advanced landscape short edge",
		nup:       4,
		pageCount: 8,
		expectedPageOrder: []int{
			8, 1, 5, 4,
			6, 3, 7, 2, // this is ordered differently from the portrait layout (because of differences in how duplexing works)
		},
		papersize:   "A5L", // landscape, short-edge binding
		bookletType: "bookletadvanced",
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
	// 8up test
	{
		id:        "8up",
		nup:       8,
		pageCount: 32,
		expectedPageOrder: []int{
			1, 30, 32, 3, 5, 26, 28, 7,
			29, 2, 4, 31, 25, 6, 8, 27,
			9, 22, 24, 11, 13, 18, 20, 15,
			21, 10, 12, 23, 17, 14, 16, 19,
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
		id:        "perfect bound 8up",
		nup:       8,
		pageCount: 16,
		expectedPageOrder: []int{
			1, 3, 5, 7, 9, 11, 13, 15,
			4, 2, 8, 6, 12, 10, 16, 14,
		},
		papersize:   "A6", // portrait, long-edge binding
		bookletType: "perfectbound",
		binding:     "long",
	},
	// signatures
	{
		id:        "signatures 2up",
		nup:       2,
		pageCount: 16,
		expectedPageOrder: []int{
			12, 1, // signature 1
			11, 2,
			10, 3,
			9, 4,
			8, 5,
			7, 6,
			16, 13, // signature 2, incomplete
			15, 14,
		},
		papersize:          "A6",
		bookletType:        "booklet",
		binding:            "long",
		useSignatures:      true,
		nPagesPerSignature: 12,
	},
	{
		id:        "signatures 4up",
		nup:       4,
		pageCount: 24,
		expectedPageOrder: []int{
			16, 1, 3, 14, // signature 1
			2, 15, 13, 4,
			12, 5, 7, 10,
			6, 11, 9, 8,
			24, 17, 19, 22, // signature 2, incomplete
			18, 23, 21, 20,
		},
		papersize:          "A5",
		bookletType:        "booklet",
		binding:            "long",
		useSignatures:      true,
		nPagesPerSignature: 16,
	},
	{
		id:        "signatures 2up with trailing blank pages",
		nup:       2,
		pageCount: 18,
		expectedPageOrder: []int{
			12, 1, // signature 1
			11, 2,
			10, 3,
			9, 4,
			8, 5,
			7, 6,
			0, 13, // signature 2, incomplete, with blanks
			0, 14,
			18, 15,
			17, 16,
		},
		papersize:          "A6",
		bookletType:        "booklet",
		binding:            "long",
		useSignatures:      true,
		nPagesPerSignature: 12,
	},
}

func TestBookletPageOrder(t *testing.T) {
	for _, test := range bookletTestCases {
		t.Run(test.id, func(tt *testing.T) {
			desc := fmt.Sprintf("papersize:%s, btype:%s, binding: %s", test.papersize, test.bookletType, test.binding)
			if test.useSignatures {
				desc += fmt.Sprintf(", multifolio:on, foliosize:%d", test.nPagesPerSignature/4)
			}
			nup, err := PDFBookletConfig(test.nup, desc, nil)
			if err != nil {
				tt.Fatal(err)
			}
			pageNumbers := make(map[int]bool)
			for i := 0; i < test.pageCount; i++ {
				pageNumbers[i+1] = true
			}
			pageOrder := make([]int, len(test.expectedPageOrder))
			out := GetBookletOrdering(pageNumbers, nup)
			if len(test.expectedPageOrder) != len(out) {
				tt.Fatalf("page order output has the wrong length, expected %d but got %d", len(test.expectedPageOrder), len(out))
			}
			for i, p := range out {
				pageOrder[i] = p.Number
			}
			for i, expected := range test.expectedPageOrder {
				if pageOrder[i] != expected {
					tt.Fatal("incorrect page order\nexpected:", arrayToString(test.expectedPageOrder), "\n     got:", arrayToString(pageOrder))
				}
			}
		})
	}
}

func arrayToString(arr []int) string {
	out := make([]string, len(arr))
	for i, n := range arr {
		out[i] = fmt.Sprintf("%02d", n)
	}
	return fmt.Sprintf("[%s]", strings.Join(out, " "))
}
