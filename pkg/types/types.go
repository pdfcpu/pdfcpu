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

// Translate modifies p's coordinates.
func (p *Point) Translate(dx, dy float64) {
	p.X += dx
	p.Y += dy
}

func (p Point) String() string {
	return fmt.Sprintf("(%.2f,%.2f)\n", p.X, p.Y)
}

// Rectangle represents a rectangular region in userspace.
type Rectangle struct {
	LL, UR Point
}

// Width returns the horizontal span of a rectangle in userspace.
func (r Rectangle) Width() float64 {
	return r.UR.X - r.LL.X
}

// Height returns the vertical span of a rectangle in userspace.
func (r Rectangle) Height() float64 {
	return r.UR.Y - r.LL.Y
}

// AspectRatio returns the relation between width and height of a rectangle.
func (r Rectangle) AspectRatio() float64 {
	return r.Width() / r.Height()
}

// Landscape returns true if r is in landscape mode.
func (r Rectangle) Landscape() bool {
	return r.AspectRatio() > 1
}

// Portrait returns true if r is in portrait mode.
func (r Rectangle) Portrait() bool {
	return r.AspectRatio() < 1
}

// Center returns the center point of a rectangle.
func (r Rectangle) Center() Point {
	return Point{(r.UR.X - r.Width()/2), (r.UR.Y - r.Height()/2)}
}

// Contains returns true if rectangle r contains point p.
func (r Rectangle) Contains(p Point) bool {
	return p.X >= r.LL.X && p.X <= r.UR.X && p.Y >= r.LL.Y && p.Y <= r.LL.Y
}

func (r Rectangle) String() string {
	return fmt.Sprintf("(%3.2f, %3.2f, %3.2f, %3.2f) w=%.2f h=%.2f ar=%.2f", r.LL.X, r.LL.Y, r.UR.X, r.UR.Y, r.Width(), r.Height(), r.AspectRatio())
}

// ShortString returns a compact string representation for r.
func (r Rectangle) ShortString() string {
	return fmt.Sprintf("(%3.0f, %3.0f, %3.0f, %3.0f)", r.LL.X, r.LL.Y, r.UR.X, r.UR.Y)
}

// NewRectangle returns a new rectangle for given corner coordinates.
func NewRectangle(llx, lly, urx, ury float64) *Rectangle {
	return &Rectangle{LL: Point{llx, lly}, UR: Point{urx, ury}}
}
