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

package sign

import (
	"bytes"
	"errors"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestDSSEvidenceUsesConfiguredDecodeLimit bounds certificates, CRLs, and OCSP responses.
func TestDSSEvidenceUsesConfiguredDecodeLimit(t *testing.T) {
	conf := model.NewStatelessConfiguration()
	conf.Limits.MaxDecodeBytes = 8
	sd := types.StreamDict{Dict: types.NewDict(), Content: bytes.Repeat([]byte{0}, 9), FilterPipeline: []types.PDFFilter{{Name: filter.Flate}}}
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	sd.Content = nil
	ctx := &model.Context{XRefTable: &model.XRefTable{Conf: conf, DSS: types.Dict{
		"Certs": types.Array{sd}, "CRLs": types.Array{sd}, "OCSPs": types.Array{sd},
	}}}
	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"certificates", func() error { _, err := extractCertsFromDSS(ctx); return err }},
		{"crls", func() error { _, err := extractCRLsFromDSS(ctx); return err }},
		{"ocsp", func() error { _, err := extractOCSPsFromDSS(ctx); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, filter.ErrDecodeLimitExceeded) {
				t.Fatalf("got %v", err)
			}
		})
	}
}
