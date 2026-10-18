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

package filter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
)

type predictorRowReader struct {
	*bytes.Reader
	last []byte
}

func (r *predictorRowReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if n > 0 {
		r.last = p[:n]
	}
	return n, err
}

// TestFlatePredictorLimitBeforeReconstruction checks that a rejected row is never reconstructed in place.
func TestFlatePredictorLimitBeforeReconstruction(t *testing.T) {
	for _, tc := range []struct {
		name      string
		predictor int
		row       []byte
	}{
		{"TIFF", PredictorTIFF, []byte{1, 2, 3}},
		{"PNG", PredictorSub, []byte{PNGSub, 1, 2, 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &predictorRowReader{Reader: bytes.NewReader(bytes.Repeat(tc.row, 2))}
			f := flate{baseFilter{
				parms:          map[string]int{"Predictor": tc.predictor, "Columns": 3},
				maxDecodeBytes: 5,
			}}
			if _, err := f.decodePostProcess(r, -1); !errors.Is(err, ErrDecodeLimitExceeded) {
				t.Fatalf("got %v, want decode limit error", err)
			}
			if !bytes.Equal(r.last, tc.row) {
				t.Fatalf("oversized row reconstructed before rejection: got %v, want %v", r.last, tc.row)
			}
		})
	}
}

// TestFlatePredictorDecodeLimit checks cumulative output limits and prefixes for TIFF and every PNG predictor.
func TestFlatePredictorDecodeLimit(t *testing.T) {
	const content = "abcdef"
	for _, predictor := range []int{
		PredictorTIFF, PredictorNone, PredictorSub, PredictorUp, PredictorAverage, PredictorPaeth, PredictorOptimum,
	} {
		parms := map[string]int{"Predictor": predictor, "Columns": 3}
		data := encodedFilterData(t, Flate, parms, content)
		for _, limit := range []int64{5, 6, 7, 0, -1} {
			for _, prefix := range []int64{-1, 0, 1, 5, 6} {
				t.Run(fmt.Sprintf("predictor=%d/limit=%d/prefix=%d", predictor, limit, prefix), func(t *testing.T) {
					checkPredictorDecodeLimit(t, parms, data, content, limit, prefix)
				})
			}
		}
	}
}

func checkPredictorDecodeLimit(t *testing.T, parms map[string]int, data []byte, content string, limit, prefix int64) {
	t.Helper()
	f, err := NewFilter(Flate, parms, limit)
	if err != nil {
		t.Fatal(err)
	}
	want := content
	if prefix >= 0 {
		want = want[:prefix]
	}
	r, err := f.DecodeLength(bytes.NewReader(data), prefix)
	if limit > 0 && int64(len(want)) > limit {
		if !errors.Is(err, ErrDecodeLimitExceeded) {
			t.Fatalf("got %v, want decode limit error", err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
