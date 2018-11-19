/*
Copyright 2018 The pdfcpu Authors.

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

// Package types provides pdfcpu's base types.
package types

import "fmt"

// Point represents a user space location.
type Point struct {
	X, Y float64
}

// Rectangle represents a rectangular region in userspace.
type Rectangle struct {
	LL, UR Point
}

// Width returns the horizontal span of a rectangle in userspace.
func (r Rectangle) Width() float64 {
	return r.UR.X - r.LL.X
}

// Height returns the vertical span of a rectangle un userspace.
func (r Rectangle) Height() float64 {
	return r.UR.Y - r.LL.Y
}

// AspectRatio returns the relation between width and height of a rectangle.
func (r Rectangle) AspectRatio() float64 {
	return r.Width() / r.Height()
}

func (r Rectangle) String() string {
	return fmt.Sprintf("(%3.2f, %3.2f, %3.2f, %3.2f) w=%.2f h=%.2f ar=%.2f", r.LL.X, r.LL.Y, r.UR.X, r.UR.Y, r.Width(), r.Height(), r.AspectRatio())
}

// NewRectangle returns a new rectangle for given corner coordinates.
func NewRectangle(llx, lly, urx, ury float64) Rectangle {
	return Rectangle{LL: Point{llx, lly}, UR: Point{urx, ury}}
}
