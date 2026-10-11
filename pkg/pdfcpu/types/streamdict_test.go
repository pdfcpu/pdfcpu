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

package types

import (
	"bytes"
	"errors"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
)

func TestDecodePreservesSoleJBIG2Stream(t *testing.T) {
	raw := []byte{0x97, 0x4a, 0x42, 0x32}
	sd := StreamDict{
		Dict:           NewDict(),
		Raw:            raw,
		FilterPipeline: []PDFFilter{{Name: filter.JBIG2}},
	}

	if err := sd.Decode(); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(sd.Content, raw) {
		t.Fatalf("got %x, want %x", sd.Content, raw)
	}
}

func TestDecodePreservesTerminalOpaqueImageStream(t *testing.T) {
	for _, tt := range []struct {
		name   string
		filter string
	}{
		{name: "JPX", filter: filter.JPX},
		{name: "JBIG2", filter: filter.JBIG2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sd := StreamDict{
				Dict: NewDict(),
				Raw:  []byte("616263>"),
				FilterPipeline: []PDFFilter{
					{Name: filter.ASCIIHex},
					{Name: tt.filter},
				},
			}

			if err := sd.Decode(); err != nil {
				t.Fatal(err)
			}

			if got, want := string(sd.Content), "abc"; got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

func TestDecodeRejectsNonTerminalOpaqueImageStream(t *testing.T) {
	sd := StreamDict{
		Dict: NewDict(),
		Raw:  []byte("abc"),
		FilterPipeline: []PDFFilter{
			{Name: filter.JPX},
			{Name: filter.ASCIIHex},
		},
	}

	if err := sd.Decode(); !errors.Is(err, filter.ErrUnsupportedFilter) {
		t.Fatalf("got %v, want %v", err, filter.ErrUnsupportedFilter)
	}
}

// TestStreamDictReencodesFlatePredictor verifies edits retain the original predictor parameters.
func TestStreamDictReencodesFlatePredictor(t *testing.T) {
	sd := StreamDict{
		Dict:    NewDict(),
		Content: []byte("abcdef"),
		FilterPipeline: []PDFFilter{{
			Name: filter.Flate,
			DecodeParms: Dict{
				"Predictor": Integer(filter.PredictorUp),
				"Columns":   Integer(3),
			},
		}},
	}
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	sd.Content = nil
	if err := sd.Decode(); err != nil {
		t.Fatal(err)
	}
	sd.Content = append(sd.Content, []byte("ghi")...)
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	sd.Content = nil
	if err := sd.Decode(); err != nil {
		t.Fatal(err)
	}
	if got, want := string(sd.Content), "abcdefghi"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestDecodeLengthWithLimitRejectsOversizedPrefixWithoutFilters covers raw and previously decoded streams.
func TestDecodeLengthWithLimitRejectsOversizedPrefixWithoutFilters(t *testing.T) {
	for _, tc := range []struct {
		name string
		sd   StreamDict
	}{
		{"raw", StreamDict{Dict: NewDict(), Raw: []byte("hello")}},
		{"cached", StreamDict{Dict: NewDict(), Content: []byte("hello")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.sd.DecodeLengthWithLimit(5, 4); !errors.Is(err, filter.ErrDecodeLimitExceeded) {
				t.Fatalf("got %v, want %v", err, filter.ErrDecodeLimitExceeded)
			}
		})
	}
}
