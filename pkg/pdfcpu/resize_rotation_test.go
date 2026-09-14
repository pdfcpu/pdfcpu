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
	"math"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func assertResizeNumbers(t *testing.T, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length: got %d, want %d", len(got), len(want))
	}
	for i, v := range got {
		if math.Abs(v-want[i]) > 0.00001 {
			t.Fatalf("value %d: got %g, want %g", i, v, want[i])
		}
	}
}

// TestResizeDefaultGeometry characterizes the existing fit and destination-orientation contract.
func TestResizeDefaultGeometry(t *testing.T) {
	tests := []struct {
		name           string
		sw, sh, dw, dh float64
		enforce        bool
		want           []float64
	}{
		{"portrait", 100, 200, 300, 400, true, []float64{300, 400, 2, 0, 1, 50, 0}},
		{"landscape", 200, 100, 400, 300, true, []float64{400, 300, 2, 0, 1, 0, 50}},
		{"landscape to portrait", 200, 100, 300, 400, true, []float64{300, 400, 2, 1, 0, 250, 0}},
		{"portrait to landscape", 100, 200, 400, 300, true, []float64{400, 300, 2, 1, 0, 400, 50}},
		{"keep source orientation", 200, 100, 300, 400, false, []float64{400, 300, 2, 0, 1, 0, 50}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := &model.Resize{PageDim: &types.Dim{Width: tt.dw, Height: tt.dh}, EnforceOrient: tt.enforce}
			r, sc, sin, cos, dx, dy := prepResize(res, types.RectForDim(tt.sw, tt.sh))
			assertResizeNumbers(t, []float64{r.Width(), r.Height(), sc, sin, cos, dx, dy}, tt.want)
		})
	}
}

// TestResizeRotationOption verifies boolean spellings, prefix matching and contextual failures.
func TestResizeRotationOption(t *testing.T) {
	for _, v := range []string{"on", "true", "t", "OFF", "false", "f"} {
		res, err := ParseResizeConfig("form:A4P, rot:"+v, types.POINTS)
		if err != nil {
			t.Fatal(err)
		}
		want := v == "OFF" || v == "false" || v == "f"
		if res.DisableContentRotation != want || !res.EnforceOrientation() {
			t.Fatalf("unexpected policy for %s: %+v", v, res)
		}
	}
	_, err := ParseResizeConfig("form:A4P, rotate:maybe", types.POINTS)
	if err == nil || !strings.Contains(err.Error(), `clause 2: resize parameter "rotate"`) {
		t.Fatalf("got %v", err)
	}
}

// TestResizeWithoutContentRotation verifies independent orientation and proportional centering.
func TestResizeWithoutContentRotation(t *testing.T) {
	tests := []struct {
		config string
		sw, sh float64
		want   []float64
	}{
		{"dim:300 400, enforce:on, rotate:off", 200, 100, []float64{300, 400, 1.5, 0, 1, 0, 125}},
		{"dim:400 300, enforce:on, rotate:off", 100, 200, []float64{400, 300, 1.5, 0, 1, 125, 0}},
		{"dim:300 400, rotate:off", 200, 100, []float64{400, 300, 2, 0, 1, 0, 50}},
		{"dim:300 400, enforce:on, rotate:off", 100, 200, []float64{300, 400, 2, 0, 1, 50, 0}},
		{"dim:300 0, rotate:off", 200, 100, []float64{300, 150, 1.5, 0, 1, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.config, func(t *testing.T) {
			res, err := ParseResizeConfig(tt.config, types.POINTS)
			if err != nil {
				t.Fatal(err)
			}
			r, sc, sin, cos, dx, dy := prepResize(res, types.RectForDim(tt.sw, tt.sh))
			assertResizeNumbers(t, []float64{r.Width(), r.Height(), sc, sin, cos, dx, dy}, tt.want)
		})
	}
}

// TestResizeSourceRotation checks visible coordinates without mutating the inherited source rectangle.
func TestResizeSourceRotation(t *testing.T) {
	tests := []struct {
		rotation int
		want     []float64
	}{
		{0, []float64{100, 200, 10, 20}},
		{90, []float64{200, 100, 20, 90}},
		{-270, []float64{200, 100, 20, 90}},
		{180, []float64{100, 200, 90, 180}},
		{-180, []float64{100, 200, 90, 180}},
		{270, []float64{200, 100, 180, 10}},
		{-90, []float64{200, 100, 180, 10}},
	}
	for _, tt := range tests {
		src := types.NewRectangle(10, 20, 110, 220)
		box, m := resizeSourceGeometry(src, tt.rotation)
		p := m.Transform(types.Point{X: 20, Y: 40})
		assertResizeNumbers(t, []float64{box.Width(), box.Height(), p.X, p.Y}, tt.want)
		assertResizeNumbers(t, []float64{src.LL.X, src.LL.Y, src.UR.X, src.UR.Y}, []float64{10, 20, 110, 220})
	}
}
