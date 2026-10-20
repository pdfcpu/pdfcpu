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

// GHSA-prvw-v9vv-6966: image masks may omit BitsPerComponent and must default to one bit per component.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-prvw-v9vv-6966

import (
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestGHSA_prvw_v9vv_6966_ImageMaskDefaultsBitsPerComponent verifies image masks use the implicit one-bit component
// depth.
func TestGHSA_prvw_v9vv_6966_ImageMaskDefaultsBitsPerComponent(t *testing.T) {
	sd := &types.StreamDict{Dict: types.Dict{
		"ColorSpace": types.Name(model.DeviceGrayCS),
		"Height":     types.Integer(1),
		"ImageMask":  types.Boolean(true),
		"Length":     types.Integer(1),
		"Width":      types.Integer(1),
	}}

	im, err := pdfImage(&model.XRefTable{}, sd, false, 7)
	if err != nil {
		t.Fatal(err)
	}
	if im.bpc != 1 || im.comp != 1 {
		t.Fatalf("got components/bpc %d/%d, want 1/1", im.comp, im.bpc)
	}
}
