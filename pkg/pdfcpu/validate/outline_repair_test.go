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

package validate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestOutlinePreviousChain checks backward cycles and terminating repair paths in both modes.
func TestOutlinePreviousChain(t *testing.T) {
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		for _, scenario := range []string{"self", "pair", "duplicate", "predecessor", "end", "canceled"} {
			t.Run(fmt.Sprintf("%s/mode%d", scenario, mode), func(t *testing.T) {
				conf := model.NewDefaultConfiguration()
				conf.ValidationMode = mode
				x, err := model.NewContext(strings.NewReader(""), conf)
				if err != nil {
					t.Fatal(err)
				}
				last := types.NewIndirectRef(7, 0)
				d := types.Dict{"Title": types.StringLiteral("item"), "Prev": *types.NewIndirectRef(8, 0)}
				x.Table[7] = model.NewXRefTableEntryGen0(d)
				x.Table[8] = model.NewXRefTableEntryGen0(types.Dict{"Prev": *types.NewIndirectRef(7, 0)})
				switch scenario {
				case "self":
					d["Prev"] = *last
				case "duplicate":
					d["Prev"] = *types.NewIndirectRef(6, 0)
				case "predecessor":
					d["Prev"] = *types.NewIndirectRef(5, 0)
				case "end":
					delete(d, "Prev")
				}
				c, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				if scenario == "canceled" {
					cancel()
				}
				n, _, err := firstOfRemainder(c, x.XRefTable, last, 6, 5)
				checkOutlinePreviousResult(t, scenario, n, d, err)
			})
		}
	}
}

func checkOutlinePreviousResult(t *testing.T, scenario string, n int, d types.Dict, err error) {
	t.Helper()
	switch scenario {
	case "self", "pair":
		var v *model.ValidationError
		if err == nil || !strings.Contains(err.Error(), "previous chain: cycle detected") || !errors.As(err, &v) || v.ObjectNumber() != 7 {
			t.Fatalf("got %v", err)
		}
	case "canceled":
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v", err)
		}
	case "end":
		if err != nil || n != 0 {
			t.Fatalf("got %d, %v", n, err)
		}
	default:
		if err != nil || n != 7 || d.IndirectRefEntry("Prev").ObjectNumber.Value() != 5 {
			t.Fatalf("repair: %d, %v, %v", n, d, err)
		}
	}
}
