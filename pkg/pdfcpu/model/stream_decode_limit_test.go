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

package model

import (
	"bytes"
	"errors"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func decodeLimitTestStream(t *testing.T, filtered, cached bool) types.StreamDict {
	t.Helper()
	sd := types.StreamDict{Dict: types.NewDict(), Content: bytes.Repeat([]byte("q Q\n"), 8)}
	if filtered {
		sd.FilterPipeline = []types.PDFFilter{{Name: filter.Flate}}
	}
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	if !cached {
		sd.Content = nil
	}
	return sd
}

// TestDecodeStreamLimitFallbacks checks default, explicit, and disabled limits through the xref decoding path.
func TestDecodeStreamLimitFallbacks(t *testing.T) {
	for _, tc := range []struct {
		name string
		x    *XRefTable
		fail bool
	}{
		{"nil table", nil, false},
		{"nil configuration", &XRefTable{}, false},
		{"zero", &XRefTable{Conf: &Configuration{}}, false},
		{"below size", &XRefTable{Conf: &Configuration{Limits: ResourceLimits{MaxDecodeBytes: 31}}}, true},
		{"exact size", &XRefTable{Conf: &Configuration{Limits: ResourceLimits{MaxDecodeBytes: 32}}}, false},
		{"disabled", &XRefTable{Conf: &Configuration{Limits: ResourceLimits{MaxDecodeBytes: -1}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sd := decodeLimitTestStream(t, true, false)
			err := tc.x.DecodeStream(&sd)
			if tc.fail {
				if !errors.Is(err, filter.ErrDecodeLimitExceeded) {
					t.Fatalf("got %v, want decode limit error", err)
				}
				return
			}
			if err != nil || len(sd.Content) != 32 {
				t.Fatalf("decoded=%d error=%v", len(sd.Content), err)
			}
		})
	}
}

// TestPageContentCombinedDecodeLimit bounds filtered, raw, and cached content arrays.
func TestPageContentCombinedDecodeLimit(t *testing.T) {
	for _, filtered := range []bool{false, true} {
		for _, cached := range []bool{false, true} {
			sd := decodeLimitTestStream(t, filtered, cached)
			for _, limit := range []int64{63, 64, 65, 0, -1} {
				conf := NewStatelessConfiguration()
				conf.Limits.MaxDecodeBytes = limit
				x := newXRefTable(conf)
				page := types.Dict{"Contents": types.Array{sd, nil, sd}}
				got, err := x.PageContent(page, 1)
				if limit == 63 {
					if !errors.Is(err, filter.ErrDecodeLimitExceeded) || got != nil {
						t.Fatalf("filtered=%v cached=%v limit=%d: got %d bytes, %v", filtered, cached, limit, len(got), err)
					}
				} else if err != nil || !bytes.Equal(got, bytes.Repeat([]byte("q Q\n"), 16)) {
					t.Fatalf("filtered=%v cached=%v limit=%d: got %d bytes, %v", filtered, cached, limit, len(got), err)
				}
			}
		}
	}
}

// TestDeferredModelDecodesUseConfiguredLimit covers decoding, dumping, and content appending.
func TestDeferredModelDecodesUseConfiguredLimit(t *testing.T) {
	conf := NewStatelessConfiguration()
	conf.Limits.MaxDecodeBytes = 31
	x := newXRefTable(conf)
	for _, tc := range []struct {
		name string
		run  func(*types.StreamDict) error
	}{
		{"decode", x.DecodeStream},
		{"dump", func(sd *types.StreamDict) error { _, err := x.dumpDecodedStream(t.Context(), *sd, 1, 1); return err }},
		{"append", func(sd *types.StreamDict) error { return x.appendToContentStream(sd, []byte("q Q")) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sd := decodeLimitTestStream(t, true, false)
			if err := tc.run(&sd); !errors.Is(err, filter.ErrDecodeLimitExceeded) {
				t.Fatalf("got %v", err)
			}
		})
	}
}
