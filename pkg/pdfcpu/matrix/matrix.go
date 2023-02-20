/*
Copyright 2022 The pdfcpu Authors.

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

package matrix

import (
	"fmt"
	"math"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const (
	DegToRad = math.Pi / 180
	RadToDeg = 180 / math.Pi
)

type Matrix [3][3]float64

var IdentMatrix = Matrix{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}

// Multiply calculates the product of two matrices.
func (m Matrix) Multiply(n Matrix) Matrix {
	var p Matrix
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				p[i][j] += m[i][k] * n[k][j]
			}
		}
	}
	return p
}

// Transform applies m to p.
func (m Matrix) Transform(p types.Point) types.Point {
	x := p.X*m[0][0] + p.Y*m[1][0] + m[2][0]
	y := p.X*m[0][1] + p.Y*m[1][1] + m[2][1]
	return types.Point{X: x, Y: y}
}

func (m Matrix) String() string {
	return fmt.Sprintf("%3.2f %3.2f %3.2f\n%3.2f %3.2f %3.2f\n%3.2f %3.2f %3.2f\n",
		m[0][0], m[0][1], m[0][2],
		m[1][0], m[1][1], m[1][2],
		m[2][0], m[2][1], m[2][2])
}

// CalcTransformMatrix returns a full transform matrix.
func CalcTransformMatrix(sx, sy, sin, cos, dx, dy float64) Matrix {
	// Scale
	m1 := IdentMatrix
	m1[0][0] = sx
	m1[1][1] = sy
	// Rotate
	m2 := IdentMatrix
	m2[0][0] = cos
	m2[0][1] = sin
	m2[1][0] = -sin
	m2[1][1] = cos
	// Translate
	m3 := IdentMatrix
	m3[2][0] = dx
	m3[2][1] = dy
	return m1.Multiply(m2).Multiply(m3)
}

// CalcRotateAndTranslateTransformMatrix returns a transform matrix that rotates and translates.
func CalcRotateAndTranslateTransformMatrix(r, dx, dy float64) Matrix {
	sin := math.Sin(float64(r) * float64(DegToRad))
	cos := math.Cos(float64(r) * float64(DegToRad))
	return CalcTransformMatrix(1, 1, sin, cos, dx, dy)
}

// CalcRotateTransformMatrix returns a transform matrix that rotates only.
func CalcRotateTransformMatrix(rot float64, bb *types.Rectangle) Matrix {
	sin := math.Sin(float64(rot) * float64(DegToRad))
	cos := math.Cos(float64(rot) * float64(DegToRad))
	dx := bb.LL.X + bb.Width()/2 + sin*(bb.Height()/2) - cos*bb.Width()/2
	dy := bb.LL.Y + bb.Height()/2 - cos*(bb.Height()/2) - sin*bb.Width()/2
	return CalcTransformMatrix(1, 1, sin, cos, dx, dy)
}
