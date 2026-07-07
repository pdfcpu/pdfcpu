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
	"math"
	"strings"
	"testing"
)

// TestEnsureZoomFactorAndMarginsSupportsNegativeHorizontalMargin verifies symmetric margin handling.
func TestEnsureZoomFactorAndMarginsSupportsNegativeHorizontalMargin(t *testing.T) {
	zoom := &Zoom{HMargin: -10}
	if err := zoom.EnsureFactorAndMargins(100, 200); err != nil {
		t.Fatal(err)
	}
	if math.Abs(zoom.Factor-1.2) > 0.0001 || zoom.HMargin != -10 || math.Abs(zoom.VMargin-(-20)) > 0.0001 {
		t.Fatalf("unexpected derived zoom: %+v", zoom)
	}
}

// TestEnsureZoomFactorAndMarginsRejectsOversizedMargin verifies non-positive scale rejection.
func TestEnsureZoomFactorAndMarginsRejectsOversizedMargin(t *testing.T) {
	tests := []struct {
		name string
		zoom *Zoom
		want string
	}{
		{name: "horizontal", zoom: &Zoom{HMargin: 50}, want: "horizontal margin"},
		{name: "vertical", zoom: &Zoom{VMargin: 100}, want: "vertical margin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.zoom.EnsureFactorAndMargins(100, 200)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
		})
	}
}
