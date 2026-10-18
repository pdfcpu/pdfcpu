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

import (
	"bytes"
	"errors"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestReadStreamDecodeLimitFallbacks checks reader decoding with missing, zero, explicit, and disabled limits.
func TestReadStreamDecodeLimitFallbacks(t *testing.T) {
	for _, tc := range []struct {
		name string
		ctx  *model.Context
		fail bool
	}{
		{"nil context", nil, false},
		{"nil configuration", &model.Context{}, false},
		{"zero", &model.Context{Configuration: &model.Configuration{}}, false},
		{"below size", &model.Context{Configuration: &model.Configuration{Limits: model.ResourceLimits{MaxDecodeBytes: 3}}}, true},
		{"exact size", &model.Context{Configuration: &model.Configuration{Limits: model.ResourceLimits{MaxDecodeBytes: 4}}}, false},
		{"disabled", &model.Context{Configuration: &model.Configuration{Limits: model.ResourceLimits{MaxDecodeBytes: -1}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sd := types.StreamDict{Dict: types.NewDict(), Content: []byte("q Q\n"), FilterPipeline: []types.PDFFilter{{Name: filter.Flate}}}
			if err := sd.Encode(); err != nil {
				t.Fatal(err)
			}
			sd.Content = nil
			err := decodeStream(tc.ctx, &sd)
			if tc.fail {
				if !errors.Is(err, filter.ErrDecodeLimitExceeded) {
					t.Fatalf("got %v, want decode limit error", err)
				}
				return
			}
			if err != nil || string(sd.Content) != "q Q\n" {
				t.Fatalf("decoded=%q error=%v", sd.Content, err)
			}
		})
	}
}

// TestWatermarkDecodesUseConfiguredLimit covers insertion, removal, and detection.
func TestWatermarkDecodesUseConfiguredLimit(t *testing.T) {
	conf := model.NewStatelessConfiguration()
	conf.Limits.MaxDecodeBytes = 31
	x := &model.XRefTable{Conf: conf}
	for _, tc := range []struct {
		name string
		run  func(*types.StreamDict) error
	}{
		{"insert", func(sd *types.StreamDict) error {
			return patchFirstContentStreamForWatermark(x, sd, "GS0", "Fm0", nil, true)
		}},
		{"remove", func(sd *types.StreamDict) error { _, _, _, err := removeArtifacts(t.Context(), x, sd, 1); return err }},
		{"detect", func(sd *types.StreamDict) error { _, err := detectArtifacts(x, sd); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sd := types.StreamDict{Dict: types.NewDict(), Content: bytes.Repeat([]byte("q Q\n"), 8), FilterPipeline: []types.PDFFilter{{Name: filter.Flate}}}
			if err := sd.Encode(); err != nil {
				t.Fatal(err)
			}
			sd.Content = nil
			if err := tc.run(&sd); !errors.Is(err, filter.ErrDecodeLimitExceeded) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

// TestKeywordMetadataUsesConfiguredDecodeLimit bounds metadata before keyword processing.
func TestKeywordMetadataUsesConfiguredDecodeLimit(t *testing.T) {
	ctx := keywordOperationTestContext(t)
	ctx.Conf.Limits.MaxDecodeBytes = 8
	sd := types.StreamDict{
		Dict:           types.NewDict(),
		Content:        keywordMetadataContent(),
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate}},
	}
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	sd.Content = nil
	installIndirectKeywordMetadata(t, ctx, sd)
	if _, err := removeKeywordsFromMetadata(t.Context(), ctx); !errors.Is(err, filter.ErrDecodeLimitExceeded) {
		t.Fatalf("got %v", err)
	}
}

// TestLazyObjectStreamUsesConfiguredDecodeLimit accepts a bounded prolog but rejects an oversized body.
func TestLazyObjectStreamUsesConfiguredDecodeLimit(t *testing.T) {
	limits := model.DefaultResourceLimits()
	limits.MaxDecodeBytes = 8
	sd := types.StreamDict{
		Dict:           types.Dict{"N": types.Integer(1), "First": types.Integer(4)},
		Content:        []byte("1 0 (long string)"),
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate}},
	}
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	sd.Content = nil
	n, first := types.Integer(1), types.Integer(4)
	osd, err := model.ObjectStreamDictWithResolvedIntegers(&sd, limits, &n, &first)
	if err != nil {
		t.Fatal(err)
	}
	if err := parseObjectStream(t.Context(), osd, limits, model.ValidationStrict); err != nil {
		t.Fatalf("prolog within limit: %v", err)
	}
	obj := osd.ObjArray[0].(types.LazyObjectStreamObject)
	if _, err := obj.GetData(); !errors.Is(err, filter.ErrDecodeLimitExceeded) {
		t.Fatalf("lazy decode: got %v", err)
	}
}
