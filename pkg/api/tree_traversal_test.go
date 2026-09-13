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

package api

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func treePolicyLeaf(nameTree bool, key int, empty bool) string {
	if nameTree {
		if empty {
			return "<< /Limits [() ()] /Names [] >>"
		}
		return fmt.Sprintf("<< /Limits [(k%d) (k%d)] /Names [(k%d) 8 0 R] >>", key, key, key)
	}
	if empty {
		return "<< /Limits [0 0] /Nums [] >>"
	}
	return fmt.Sprintf("<< /Limits [%d %d] /Nums [%d 8 0 R] >>", key, key, key)
}

func treePolicyBranch(nameTree bool, kids string) string {
	limits := "[0 0]"
	if nameTree {
		limits = "[(k0) (k0)]"
	}
	return fmt.Sprintf("<< /Limits %s /Kids [%s] >>", limits, kids)
}

func treePolicyObjects(nameTree bool, scenario string) []string {
	entry := "/PageLabels 4 0 R"
	value := "<< /S /D >>"
	if nameTree {
		entry = "/Names << /JavaScript 4 0 R >>"
		value = "<< /S /JavaScript /JS () >>"
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R " + entry + " >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources << >> >>",
		"<< /Kids [5 0 R 6 0 R] >>",
		treePolicyLeaf(nameTree, 0, false), treePolicyLeaf(nameTree, 1, false),
		treePolicyLeaf(nameTree, 0, false), value,
	}
	switch scenario {
	case "self-cycle":
		objects[3] = treePolicyBranch(nameTree, "4 0 R")
	case "nested-cycle":
		objects[4] = treePolicyBranch(nameTree, "7 0 R")
		objects[6] = treePolicyBranch(nameTree, "5 0 R")
	case "sibling-duplicate":
		objects[3] = "<< /Kids [5 0 R 5 0 R] >>"
	case "cross-parent-duplicate":
		objects[4] = treePolicyBranch(nameTree, "7 0 R")
		objects[5] = treePolicyBranch(nameTree, "7 0 R")
	case "empty-duplicate":
		objects[3] = "<< /Kids [5 0 R 5 0 R] >>"
		objects[4] = treePolicyLeaf(nameTree, 0, true)
	case "direct-children":
		objects[3] = "<< /Kids [" + objects[4] + " " + objects[5] + "] >>"
	case "depth":
		objects[3] = "<< /Kids [5 0 R] >>"
		objects[4] = treePolicyBranch(nameTree, "6 0 R")
		objects[5] = treePolicyBranch(nameTree, "7 0 R")
	}
	return objects
}

func treePolicyPDF(nameTree bool, scenario string) []byte {
	objects := treePolicyObjects(nameTree, scenario)
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(objects))
	for i, body := range objects {
		offsets[i] = appendValidationTestObject(&b, i+1, body)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Root 1 0 R /Size %d >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return b.Bytes()
}

// TestTreeTraversalRejectsMalformedGraphs verifies structural errors survive both validation modes.
func TestTreeTraversalRejectsMalformedGraphs(t *testing.T) {
	for _, nameTree := range []bool{true, false} {
		cycle, duplicate := model.ErrNumberTreeCycle, model.ErrNumberTreeDuplicate
		if nameTree {
			cycle, duplicate = model.ErrNameTreeCycle, model.ErrNameTreeDuplicate
		}
		cases := []struct {
			name   string
			want   error
			object int
		}{
			{"self-cycle", cycle, 4}, {"nested-cycle", cycle, 5},
			{"sibling-duplicate", duplicate, 5}, {"cross-parent-duplicate", duplicate, 7},
			{"empty-duplicate", duplicate, 5}, {"depth", model.ErrMaxRecursionDepthExceeded, 7},
		}
		for _, tt := range cases {
			for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
				t.Run(fmt.Sprintf("name=%t/%s/mode=%d", nameTree, tt.name, mode), func(t *testing.T) {
					conf := model.NewStatelessConfiguration()
					conf.Offline, conf.ValidationMode = true, mode
					conf.Limits.MaxRecursionDepth = 2
					err := Validate(t.Context(), bytes.NewReader(treePolicyPDF(nameTree, tt.name)), conf, nil)
					if !errors.Is(err, tt.want) {
						t.Fatalf("got %v, want %v", err, tt.want)
					}
					var attributed *model.ValidationError
					if !errors.As(err, &attributed) || attributed.ObjectNumber() != tt.object {
						t.Fatalf("unexpected object attribution: %v", err)
					}
				})
			}
		}
	}
}

// TestTreeTraversalPreservesSharedValues verifies child identity checks do not conflate values or direct nodes.
func TestTreeTraversalPreservesSharedValues(t *testing.T) {
	for _, nameTree := range []bool{true, false} {
		for _, scenario := range []string{"shared-value", "direct-children"} {
			for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
				conf := model.NewStatelessConfiguration()
				conf.Offline, conf.ValidationMode = true, mode
				for i := 0; i < 2; i++ {
					err := Validate(t.Context(), bytes.NewReader(treePolicyPDF(nameTree, scenario)), conf, nil)
					if err != nil {
						t.Fatalf("name=%t %s mode=%d: %v", nameTree, scenario, mode, err)
					}
				}
			}
		}
	}
}
