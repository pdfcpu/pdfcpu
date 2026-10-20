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

package pdfcpu

// GHSA-w2hj-54mc-76r4: overflowing xref stream /W widths must return an error instead of panicking.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-w2hj-54mc-76r4

import (
	"bytes"
	"errors"
	"math"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestGHSA_w2hj_54mc_76r4_XRefStreamEntryLenRejectsOverflow(t *testing.T) {
	for _, tc := range []struct {
		name string
		w    [3]int
	}{
		{"first addition", [3]int{math.MaxInt, 1, 0}},
		{"second addition", [3]int{math.MaxInt - 1, 1, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := xRefStreamEntryLen(tc.w); !errors.Is(err, errInvalidXRefStreamWArray) {
				t.Fatalf("got %v, want %v", err, errInvalidXRefStreamWArray)
			}
		})
	}
}

func TestGHSA_w2hj_54mc_76r4_ExtractXRefStreamEntriesRejectsWidthOverflow(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	w := 2*(math.MaxInt/3) + 1
	xsd := &types.XRefStreamDict{
		Objects: []int{0, 1, 2},
		W:       [3]int{w, w, w + 2},
	}
	err = extractXRefTableEntriesFromXRefStream(make([]byte, 3), 0, xsd, ctx, 0)
	if !errors.Is(err, errInvalidXRefStreamWArray) {
		t.Fatalf("got %v, want %v", err, errInvalidXRefStreamWArray)
	}
}

func TestGHSA_w2hj_54mc_76r4_ExtractXRefStreamEntriesRejectsImpossibleObjectCount(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}

	xsd := &types.XRefStreamDict{
		Objects: []int{0, 1},
		W:       [3]int{math.MaxInt/2 + 1, 0, 0},
	}
	err = extractXRefTableEntriesFromXRefStream(nil, 0, xsd, ctx, 0)
	if !errors.Is(err, errCorruptXRefStream) {
		t.Fatalf("got %v, want %v", err, errCorruptXRefStream)
	}
}
