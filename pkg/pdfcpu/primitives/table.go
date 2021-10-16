/*
Copyright 2021 The pdfcpu Authors.

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

package primitives

import "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"

type Table struct {
	pdf             *pdfcpu.PDF
	content         *pdfcpu.Content
	Name            string
	Position        [2]float64 `json:"pos"` // x,y
	x, y            float64
	Dx, Dy          float64
	Anchor          string
	anchor          pdfcpu.Anchor
	anchored        bool
	Width           float64
	Rows, Cols      int
	colWidths       []int
	LineHeight      int
	Margin          *pdfcpu.Margin
	Border          *pdfcpu.Border
	Padding         *pdfcpu.Padding
	BackgroundColor string `json:"bgCol"`
	OddColor        string `json:"oddCol"`
	EvenColor       string `json:"evenCol"`
	HeaderColor     string `json:"headerCol"`
	bgCol           *pdfcpu.SimpleColor
	oddCol          *pdfcpu.SimpleColor
	evenCol         *pdfcpu.SimpleColor
	headerCol       *pdfcpu.SimpleColor
	Rotation        float64 `json:"rot"`
	Hide            bool
}

func (t *Table) validate() error {
	return nil
}

func (t *Table) render() error {
	return nil
}
