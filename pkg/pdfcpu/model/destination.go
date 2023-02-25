/*
Copyright 2023 The pdfcpu Authors.

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

import "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

// DestinationType represents the various PDF destination types.
type DestinationType int

// See table 151
const (
	DestXYZ   DestinationType = iota // [page /XYZ left top zoom]
	DestFit                          // [page /Fit]
	DestFitH                         // [page /FitH top]
	DestFitV                         // [page /FitV left]
	DestFitR                         // [page /FitR left bottom right top]
	DestFitB                         // [page /FitB]
	DestFitBH                        // [page /FitBH top]
	DestFitBV                        // [page /FitBV left]
)

// DestinationTypeStrings manages string representations for destination types.
var DestinationTypeStrings = map[DestinationType]string{
	DestXYZ:   "XYZ",   // Position (left, top) at upper-left corner of window.
	DestFit:   "Fit",   // Fit entire page within window.
	DestFitH:  "FitH",  // Position with (top) at top edge of window.
	DestFitV:  "FitV",  // Position with (left) positioned at left edge of window.
	DestFitR:  "FitR",  // Fit (left, bottom, right, top) entirely within window.
	DestFitB:  "FitB",  // Magnify content just enough to fit its bounding box entirely within window.
	DestFitBH: "FitBH", // Position with (top) at top edge of window and contents fit bounding box width within window.
	DestFitBV: "FitBV", // Position with (left) at left edge of window and contents fit bounding box height within window.
}

// Destination represents a PDF destination.
type Destination struct {
	Typ                      DestinationType
	PageNr                   int
	Left, Bottom, Right, Top int
	Zoom                     float32
}

func (dest Destination) String() string {
	return DestinationTypeStrings[dest.Typ]
}

func (dest Destination) Name() types.Name {
	return types.Name(DestinationTypeStrings[dest.Typ])
}

func (dest Destination) Array(indRef types.IndirectRef) types.Array {
	arr := types.Array{indRef, dest.Name()}
	switch dest.Typ {
	case DestXYZ:
		arr = append(arr, types.Integer(dest.Left), types.Integer(dest.Top), types.Float(dest.Zoom))
	case DestFitH:
		arr = append(arr, types.Integer(dest.Top))
	case DestFitV:
		arr = append(arr, types.Integer(dest.Left))
	case DestFitR:
		arr = append(arr, types.Integer(dest.Left), types.Integer(dest.Bottom), types.Integer(dest.Right), types.Integer(dest.Top))
	case DestFitBH:
		arr = append(arr, types.Integer(dest.Top))
	case DestFitBV:
		arr = append(arr, types.Integer(dest.Left))
	}
	return arr
}
