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
	"compress/zlib"
	"io"
	"strconv"
	"testing"
)

// TestFlatePredictorEncodeRoundTrip verifies that encoding honors the predictor kept in DecodeParms.
func TestFlatePredictorEncodeRoundTrip(t *testing.T) {
	data := []byte{
		10, 20, 30, 11, 22, 33, 12, 24, 36,
		13, 26, 39, 14, 28, 42, 15, 30, 45,
	}
	for _, predictor := range []int{
		PredictorTIFF, PredictorNone, PredictorSub, PredictorUp,
		PredictorAverage, PredictorPaeth, PredictorOptimum,
	} {
		t.Run(strconv.Itoa(predictor), func(t *testing.T) {
			f, err := NewFilter(Flate, map[string]int{
				"Predictor": predictor, "Colors": 3, "Columns": 3,
			})
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := f.Encode(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := f.Decode(encoded)
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(decoded)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, data) {
				t.Fatalf("predictor %d: got %v, want %v", predictor, got, data)
			}
		})
	}
}

// TestFlatePredictorPackedSamples verifies PNG prediction with sub-byte and 16-bit components.
func TestFlatePredictorPackedSamples(t *testing.T) {
	for _, tc := range []struct {
		name    string
		bpc     int
		columns int
		data    []byte
	}{
		{"one-bit", 1, 9, []byte{0x85, 0x80, 0xc3, 0x00}},
		{"sixteen-bit", 16, 2, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := NewFilter(Flate, map[string]int{
				"Predictor": PredictorPaeth, "BitsPerComponent": tc.bpc, "Columns": tc.columns,
			})
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := f.Encode(bytes.NewReader(tc.data))
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := f.Decode(encoded)
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(decoded)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, tc.data) {
				t.Fatalf("got %v, want %v", got, tc.data)
			}
		})
	}
}

// TestFlatePredictorEncodingFormat checks row framing, TIFF differencing, and incomplete-row errors.
func TestFlatePredictorEncodingFormat(t *testing.T) {
	for _, tc := range []struct {
		name      string
		predictor int
		data      []byte
		want      []byte
	}{
		{"TIFF", PredictorTIFF, []byte{10, 12, 15, 20, 22, 25}, []byte{10, 2, 3, 20, 2, 3}},
		{"PNG-Up", PredictorUp, []byte{10, 12, 15, 20, 22, 25}, []byte{PNGUp, 10, 12, 15, PNGUp, 10, 10, 10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := NewFilter(Flate, map[string]int{"Predictor": tc.predictor, "Columns": 3})
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := f.Encode(bytes.NewReader(tc.data))
			if err != nil {
				t.Fatal(err)
			}
			zr, err := zlib.NewReader(encoded)
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(zr)
			if err != nil {
				t.Fatal(err)
			}
			if err := zr.Close(); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}

	for _, tc := range []struct {
		name  string
		parms map[string]int
	}{
		{"short row", map[string]int{"Predictor": PredictorUp, "Columns": 4}},
		{"unsupported TIFF depth", map[string]int{"Predictor": PredictorTIFF, "BitsPerComponent": 16, "Columns": 4}},
		{"invalid columns", map[string]int{"Predictor": PredictorUp, "Columns": 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := NewFilter(Flate, tc.parms)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.Encode(bytes.NewReader([]byte{1, 2, 3})); err == nil {
				t.Fatal("expected predictor encoding error")
			}
		})
	}
}
