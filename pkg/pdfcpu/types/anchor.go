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

package types

// Anchor represents symbolic positions within a rectangular region.
type Anchor int

func (a Anchor) String() string {

	switch a {

	case TopLeft:
		return "top left"

	case TopCenter:
		return "top center"

	case TopRight:
		return "top right"

	case Left:
		return "left"

	case Center:
		return "center"

	case Right:
		return "right"

	case BottomLeft:
		return "bottom left"

	case BottomCenter:
		return "bottom center"

	case BottomRight:
		return "bottom right"

	case Full:
		return "full"

	}

	return ""
}

// These are the defined anchors for relative positioning.
const (
	TopLeft Anchor = iota
	TopCenter
	TopRight
	Left
	Center // default
	Right
	BottomLeft
	BottomCenter
	BottomRight
	Full // special case, no anchor needed, imageSize = pageSize
)
