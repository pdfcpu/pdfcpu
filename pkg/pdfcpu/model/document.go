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

package model

import "strings"

type PageMode int

const (
	PageModeUseNone PageMode = iota
	PageModeUseOutlines
	PageModeUseThumb
	PageModeFullScreen
	PageModeUseOC
	PageModeUseAttachments
)

func (pm PageMode) String() string {
	switch pm {
	case PageModeUseNone:
		return "UseNone" // = default
	case PageModeUseOutlines:
		return "UseOutlines"
	case PageModeUseThumb:
		return "UseThumb"
	case PageModeFullScreen:
		return "FullScreen"
	case PageModeUseOC:
		return "UseOC"
	case PageModeUseAttachments:
		return "UseAttachments"
	default:
		return "?"
	}
}

func PageModeFor(s string) *PageMode {
	var pm PageMode
	switch strings.ToLower(s) {
	case "usenone":
		pm = PageModeUseNone
	case "useoutlines":
		pm = PageModeUseOutlines
	case "usethumb":
		pm = PageModeUseThumb
	case "fullscreen":
		pm = PageModeFullScreen
	case "useoc":
		pm = PageModeUseOC
	case "useattachments":
		pm = PageModeUseAttachments
	}
	return &pm
}

type PageLayout int

const (
	PageLayoutSinglePage PageLayout = iota
	PageLayoutTwoColumnLeft
	PageLayoutTwoColumnRight
	PageLayoutTwoPageLeft
	PageLayoutTwoPageRight
)

func (pl PageLayout) String() string {
	switch pl {
	case PageLayoutSinglePage:
		return "SinglePage" // = default
	case PageLayoutTwoColumnLeft:
		return "TwoColumnLeft"
	case PageLayoutTwoColumnRight:
		return "TwoColumnRight"
	case PageLayoutTwoPageLeft:
		return "TwoPageLeft"
	case PageLayoutTwoPageRight:
		return "TwoPageRight"
	default:
		return "?"
	}
}

func PageLayoutFor(s string) *PageLayout {
	var pl PageLayout
	switch strings.ToLower(s) {
	case "singlepage":
		pl = PageLayoutSinglePage
	case "twocolumnleft":
		pl = PageLayoutTwoColumnLeft
	case "twocolumnright":
		pl = PageLayoutTwoColumnRight
	case "twopageleft":
		pl = PageLayoutTwoPageLeft
	case "twopageright":
		pl = PageLayoutTwoPageRight
	}
	return &pl
}
