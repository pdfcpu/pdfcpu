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

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

type PageMode int

const (
	PageModeUseNone PageMode = iota
	PageModeUseOutlines
	PageModeUseThumbs
	PageModeFullScreen
	PageModeUseOC
	PageModeUseAttachments
)

func PageModeFor(s string) *PageMode {
	if s == "" {
		return nil
	}
	var pm PageMode
	switch strings.ToLower(s) {
	case "usenone":
		pm = PageModeUseNone
	case "useoutlines":
		pm = PageModeUseOutlines
	case "usethumbs":
		pm = PageModeUseThumbs
	case "fullscreen":
		pm = PageModeFullScreen
	case "useoc":
		pm = PageModeUseOC
	case "useattachments":
		pm = PageModeUseAttachments
	default:
		return nil
	}
	return &pm
}

func (pm *PageMode) String() string {
	if pm == nil {
		return ""
	}
	switch *pm {
	case PageModeUseNone:
		return "UseNone" // = default
	case PageModeUseOutlines:
		return "UseOutlines"
	case PageModeUseThumbs:
		return "UseThumbs"
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

type PageLayout int

const (
	PageLayoutSinglePage PageLayout = iota
	PageLayoutTwoColumnLeft
	PageLayoutTwoColumnRight
	PageLayoutTwoPageLeft
	PageLayoutTwoPageRight
)

func PageLayoutFor(s string) *PageLayout {
	if s == "" {
		return nil
	}
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
	default:
		return nil
	}
	return &pl
}

func (pl *PageLayout) String() string {
	if pl == nil {
		return ""
	}
	switch *pl {
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

type NonFullScreenPageMode PageMode

const (
	NFSPageModeUseNone NonFullScreenPageMode = iota
	NFSPageModeUseOutlines
	NFSPageModeUseThumb
	NFSPageModeUseOC
)

type PageBoundary int

const (
	MediaBox PageBoundary = iota
	CropBox
	TrimBox
	BleedBox
	ArtBox
)

func PageBoundaryFor(s string) *PageBoundary {
	if s == "" {
		return nil
	}
	var pb PageBoundary
	switch strings.ToLower(s) {
	case "mediabox":
		pb = MediaBox
	case "cropbox":
		pb = CropBox
	case "trimbox":
		pb = TrimBox
	case "bleedbox":
		pb = BleedBox
	case "artbox":
		pb = ArtBox
	default:
		return nil
	}
	return &pb
}

func (pb *PageBoundary) String() string {
	if pb == nil {
		return ""
	}
	switch *pb {
	case MediaBox:
		return "MediaBox"
	case CropBox:
		return "CropBox"
	case TrimBox:
		return "TrimBox"
	case BleedBox:
		return "BleedBox"
	case ArtBox:
		return "ArtBox"
	default:
		return "?"
	}
}

type PrintScaling int

const (
	PrintScalingNone PrintScaling = iota
	PrintScalingAppDefault
)

func PrintScalingFor(s string) *PrintScaling {
	if s == "" {
		return nil
	}
	var ps PrintScaling
	switch strings.ToLower(s) {
	case "none":
		ps = PrintScalingNone
	case "appdefault":
		ps = PrintScalingAppDefault
	default:
		return nil
	}
	return &ps
}

func (ps *PrintScaling) String() string {
	if ps == nil {
		return ""
	}
	switch *ps {
	case PrintScalingNone:
		return "None"
	case PrintScalingAppDefault:
		return "AppDefault"
	default:
		return "?"
	}
}

type Direction int

const (
	L2R Direction = iota
	R2L
)

func DirectionFor(s string) *Direction {
	if s == "" {
		return nil
	}
	var d Direction
	switch strings.ToLower(s) {
	case "l2r":
		d = L2R
	case "r2l":
		d = R2L
	default:
		return nil
	}
	return &d
}

func (d *Direction) String() string {
	if d == nil {
		return ""
	}
	switch *d {
	case L2R:
		return "L2R"
	case R2L:
		return "R2L"
	default:
		return "?"
	}
}

type PaperHandling int

const (
	Simplex PaperHandling = iota
	DuplexFlipShortEdge
	DuplexFlipLongEdge
)

func PaperHandlingFor(s string) *PaperHandling {
	if s == "" {
		return nil
	}
	var ph PaperHandling
	switch strings.ToLower(s) {
	case "simplex":
		ph = Simplex
	case "duplexflipshortedge":
		ph = DuplexFlipShortEdge
	case "duplexfliplongedge":
		ph = DuplexFlipLongEdge
	default:
		return nil
	}
	return &ph
}

func (ph *PaperHandling) String() string {
	if ph == nil {
		return ""
	}
	switch *ph {
	case Simplex:
		return "Simplex"
	case DuplexFlipShortEdge:
		return "DuplexFlipShortEdge"
	case DuplexFlipLongEdge:
		return "DuplexFlipLongEdge"
	default:
		return "?"
	}
}

// ViewerPreferences see 12.2 Table 147
type ViewerPreferences struct {
	HideToolbar           *bool
	HideMenubar           *bool
	HideWindowUI          *bool
	FitWindow             *bool
	CenterWindow          *bool
	DisplayDocTitle       *bool // since 1.4
	NonFullScreenPageMode *NonFullScreenPageMode
	Direction             *Direction     // since 1.3
	ViewArea              *PageBoundary  // since 1.4 to 1.7
	ViewClip              *PageBoundary  // since 1.4 to 1.7
	PrintArea             *PageBoundary  // since 1.4 to 1.7
	PrintClip             *PageBoundary  // since 1.4 to 1.7
	PrintScaling          *PrintScaling  // since 1.6
	Duplex                *PaperHandling // since 1.7
	PickTrayByPDFSize     *bool          // since 1.7
	PrintPageRange        types.Array    // since 1.7
	NumCopies             *types.Integer // since 1.7
	Enforce               types.Array    // since 2.0
}

func (vp *ViewerPreferences) validatePrinterPreferences(version Version) error {
	if vp.PrintScaling != nil && version < V16 {
		return errors.Errorf("pdfcpu: invalid viewer preference \"PrintScaling\" - since PDF 1.6, got: %v\n", version)
	}
	if vp.Duplex != nil && version < V17 {
		return errors.Errorf("pdfcpu: invalid viewer preference \"Duplex\" - since PDF 1.7, got: %v\n", version)
	}
	if vp.PickTrayByPDFSize != nil && version < V17 {
		return errors.Errorf("pdfcpu: invalid viewer preference \"PickTrayByPDFSize\" - since PDF 1.7, got: %v\n", version)
	}
	if len(vp.PrintPageRange) > 0 && version < V17 {
		return errors.Errorf("pdfcpu: invalid viewer preference \"PrintPageRange\" - since PDF 1.7, got: %v\n", version)
	}
	if vp.NumCopies != nil && version < V17 {
		return errors.Errorf("pdfcpu: invalid viewer preference \"NumCopies\" - since PDF 1.7, got: %v\n", version)
	}
	if len(vp.Enforce) > 0 && version < V20 {
		return errors.Errorf("pdfcpu: invalid viewer preference \"Enforce\" - since PDF 2.0, got: %v\n", version)
	}

	return nil
}

func (vp *ViewerPreferences) Validate(version Version) error {
	if vp.Direction != nil && version < V13 {
		return errors.Errorf("pdfcpu: invalid viewer preference \"Direction\" - since PDF 1.3, got: %v\n", version)
	}
	if vp.ViewArea != nil && (version < V14 || version > V17) {
		return errors.Errorf("pdfcpu: invalid viewer preference \"ViewArea\" - since PDF 1.4 until PDF 1.7, got: %v\n", version)
	}
	if vp.ViewClip != nil && (version < V14 || version > V17) {
		return errors.Errorf("pdfcpu: invalid viewer preference \"ViewClip\" - since PDF 1.4 until PDF 1.7, got: %v\n", version)
	}
	if vp.PrintArea != nil && (version < V14 || version > V17) {
		return errors.Errorf("pdfcpu: invalid viewer preference \"PrintArea\" - since PDF 1.4 until PDF 1.7, got: %v\n", version)
	}
	if vp.PrintClip != nil && (version < V14 || version > V17) {
		return errors.Errorf("pdfcpu: invalid viewer preference \"PrintClip\" - since PDF 1.4 until PDF 1.7, got: %v\n", version)
	}

	return vp.validatePrinterPreferences(version)
}

func (vp *ViewerPreferences) SetHideToolBar(val bool) {
	vp.HideToolbar = &val
}

func (vp *ViewerPreferences) SetHideMenuBar(val bool) {
	vp.HideMenubar = &val
}

func (vp *ViewerPreferences) SetHideWindowUI(val bool) {
	vp.HideWindowUI = &val
}

func (vp *ViewerPreferences) SetFitWindow(val bool) {
	vp.FitWindow = &val
}

func (vp *ViewerPreferences) SetCenterWindow(val bool) {
	vp.CenterWindow = &val
}

func (vp *ViewerPreferences) SetDisplayDocTitle(val bool) {
	vp.DisplayDocTitle = &val
}

func (vp *ViewerPreferences) SetPickTrayByPDFSize(val bool) {
	vp.PickTrayByPDFSize = &val
}

func (vp *ViewerPreferences) SetNumCopies(i int) {
	vp.NumCopies = (*types.Integer)(&i)
}

func (vp *ViewerPreferences) populatePrinterPreferences(vp1 *ViewerPreferences) {
	if vp1.PrintArea != nil {
		vp.PrintArea = vp1.PrintArea
	}
	if vp1.PrintClip != nil {
		vp.PrintClip = vp1.PrintClip
	}
	if vp1.PrintScaling != nil {
		vp.PrintScaling = vp1.PrintScaling
	}
	if vp1.Duplex != nil {
		vp.Duplex = vp1.Duplex
	}
	if vp1.PickTrayByPDFSize != nil {
		vp.PickTrayByPDFSize = vp1.PickTrayByPDFSize
	}
	if len(vp1.PrintPageRange) > 0 {
		vp.PrintPageRange = vp1.PrintPageRange
	}
	if vp1.NumCopies != nil {
		vp.NumCopies = vp1.NumCopies
	}
	if len(vp1.Enforce) > 0 {
		vp.Enforce = vp1.Enforce
	}
}

func (vp *ViewerPreferences) Populate(vp1 *ViewerPreferences) {
	if vp1.HideToolbar != nil {
		vp.HideToolbar = vp1.HideToolbar
	}
	if vp1.HideMenubar != nil {
		vp.HideMenubar = vp1.HideMenubar
	}
	if vp1.HideWindowUI != nil {
		vp.HideWindowUI = vp1.HideWindowUI
	}
	if vp1.FitWindow != nil {
		vp.FitWindow = vp1.FitWindow
	}
	if vp1.CenterWindow != nil {
		vp.CenterWindow = vp1.CenterWindow
	}
	if vp1.DisplayDocTitle != nil {
		vp.DisplayDocTitle = vp1.DisplayDocTitle
	}
	if vp1.NonFullScreenPageMode != nil {
		vp.NonFullScreenPageMode = vp1.NonFullScreenPageMode
	}
	if vp1.Direction != nil {
		vp.Direction = vp1.Direction
	}
	if vp1.ViewArea != nil {
		vp.ViewArea = vp1.ViewArea
	}
	if vp1.ViewClip != nil {
		vp.ViewClip = vp1.ViewClip
	}

	vp.populatePrinterPreferences(vp1)
}

func DefaultViewerPreferences(version Version) *ViewerPreferences {
	vp := ViewerPreferences{}
	vp.SetHideToolBar(false)
	vp.SetHideMenuBar(false)
	vp.SetHideWindowUI(false)
	vp.SetFitWindow(false)
	vp.SetCenterWindow(false)
	if version >= V14 {
		vp.SetDisplayDocTitle(false)
	}
	vp.NonFullScreenPageMode = (*NonFullScreenPageMode)(PageModeFor("UseNone"))
	if version >= V13 {
		vp.Direction = DirectionFor("L2R")
	}
	if version >= V14 && version < V20 {
		vp.ViewArea = PageBoundaryFor("CropBox")
		vp.ViewClip = PageBoundaryFor("CropBox")
		vp.PrintArea = PageBoundaryFor("CropBox")
		vp.PrintClip = PageBoundaryFor("CropBox")
	}
	if version >= V16 {
		vp.PrintScaling = PrintScalingFor("AppDefault")
	}
	if version >= V17 {
		vp.SetNumCopies(1)
	}

	return &vp
}

func ViewerPreferencesWithDefaults(vp *ViewerPreferences, version Version) (*ViewerPreferences, error) {
	vp1 := DefaultViewerPreferences(version)

	if vp == nil {
		return vp1, nil
	}

	vp1.Populate(vp)

	return vp1, nil
}

type ViewerPrefJSON struct {
	HideToolbar           *bool    `json:"hideToolbar,omitempty"`
	HideMenubar           *bool    `json:"hideMenubar,omitempty"`
	HideWindowUI          *bool    `json:"hideWindowUI,omitempty"`
	FitWindow             *bool    `json:"fitWindow,omitempty"`
	CenterWindow          *bool    `json:"centerWindow,omitempty"`
	DisplayDocTitle       *bool    `json:"displayDocTitle,omitempty"`
	NonFullScreenPageMode string   `json:"nonFullScreenPageMode,omitempty"`
	Direction             string   `json:"direction,omitempty"`
	ViewArea              string   `json:"viewArea,omitempty"`
	ViewClip              string   `json:"viewClip,omitempty"`
	PrintArea             string   `json:"printArea,omitempty"`
	PrintClip             string   `json:"printClip,omitempty"`
	PrintScaling          string   `json:"printScaling,omitempty"`
	Duplex                string   `json:"duplex,omitempty"`
	PickTrayByPDFSize     *bool    `json:"pickTrayByPDFSize,omitempty"`
	PrintPageRange        []int    `json:"printPageRange,omitempty"`
	NumCopies             *int     `json:"numCopies,omitempty"`
	Enforce               []string `json:"enforce,omitempty"`
}

func (vp *ViewerPreferences) MarshalJSON() ([]byte, error) {
	vpJSON := ViewerPrefJSON{
		HideToolbar:           vp.HideToolbar,
		HideMenubar:           vp.HideMenubar,
		HideWindowUI:          vp.HideWindowUI,
		FitWindow:             vp.FitWindow,
		CenterWindow:          vp.CenterWindow,
		DisplayDocTitle:       vp.DisplayDocTitle,
		NonFullScreenPageMode: (*PageMode)(vp.NonFullScreenPageMode).String(),
		Direction:             vp.Direction.String(),
		ViewArea:              vp.ViewArea.String(),
		ViewClip:              vp.ViewClip.String(),
		PrintArea:             vp.PrintArea.String(),
		PrintClip:             vp.PrintClip.String(),
		PrintScaling:          vp.PrintScaling.String(),
		Duplex:                vp.Duplex.String(),
		PickTrayByPDFSize:     vp.PickTrayByPDFSize,
		NumCopies:             (*int)(vp.NumCopies),
	}

	if len(vp.PrintPageRange) > 0 {
		var ii []int
		for _, v := range vp.PrintPageRange {
			ii = append(ii, v.(types.Integer).Value())
		}
		vpJSON.PrintPageRange = ii
	}

	if len(vp.Enforce) > 0 {
		var ss []string
		for _, v := range vp.Enforce {
			ss = append(ss, v.(types.Name).Value())
		}
		vpJSON.Enforce = ss
	}

	return json.Marshal(&vpJSON)
}

func (vp *ViewerPreferences) unmarshalPrintPageRange(vpJSON ViewerPrefJSON) error {
	if len(vpJSON.PrintPageRange) > 0 {
		arr := vpJSON.PrintPageRange
		if len(arr)%2 > 0 {
			return errors.New("pdfcpu: invalid \"PrintPageRange\" - expecting pairs of ascending page numbers\n")
		}
		for i := 0; i < len(arr); i += 2 {
			if arr[i] >= arr[i+1] {
				// TODO validate ascending, non overlapping int intervals.
				return errors.New("pdfcpu: invalid \"PrintPageRange\" - expecting pairs of ascending page numbers\n")
			}
		}
		vp.PrintPageRange = types.NewIntegerArray(arr...)
	}

	return nil
}

func (vp *ViewerPreferences) unmarshalPrinterPreferences(vpJSON ViewerPrefJSON) error {
	vp.PrintArea = PageBoundaryFor(vpJSON.PrintArea)
	if vpJSON.PrintArea != "" && vp.PrintArea == nil {
		return errors.Errorf("pdfcpu: unknown \"PrintArea\", got: %s want one of: MediaBox, CropBox, TrimBox, BleedBox, ArtBox\n", vpJSON.PrintArea)
	}

	vp.PrintClip = PageBoundaryFor(vpJSON.PrintClip)
	if vpJSON.PrintClip != "" && vp.PrintClip == nil {
		return errors.Errorf("pdfcpu: unknown \"PrintClip\", got: %s want one of: MediaBox, CropBox, TrimBox, BleedBox, ArtBox\n", vpJSON.PrintClip)
	}

	vp.PrintScaling = PrintScalingFor(vpJSON.PrintScaling)
	if vpJSON.PrintScaling != "" && vp.PrintScaling == nil {
		return errors.Errorf("pdfcpu: unknown \"PrintScaling\", got: %s, want one of: None, AppDefault", vpJSON.PrintScaling)
	}

	vp.Duplex = PaperHandlingFor(vpJSON.Duplex)
	if vpJSON.Duplex != "" && vp.Duplex == nil {
		return errors.Errorf("pdfcpu: unknown \"Duplex\", got: %s, want one of: Simplex, DuplexFlipShortEdge, DuplexFlipLongEdge", vpJSON.Duplex)
	}

	if err := vp.unmarshalPrintPageRange(vpJSON); err != nil {
		return err
	}

	if len(vpJSON.Enforce) > 1 {
		return errors.New("pdfcpu: \"Enforce\" must be array with one element: \"PrintScaling\"\n")
	}

	if len(vpJSON.Enforce) > 0 {
		if vpJSON.Enforce[0] != "PrintScaling" {
			return errors.New("pdfcpu: \"Enforce\" must be array with one element: \"PrintScaling\"\n")
		}
		vp.Enforce = types.NewNameArray("PrintScaling")
	}

	return nil
}

func (vp *ViewerPreferences) UnmarshalJSON(data []byte) error {

	vpJSON := ViewerPrefJSON{}

	if err := json.Unmarshal(data, &vpJSON); err != nil {
		return err
	}

	*vp = ViewerPreferences{
		HideToolbar:       vpJSON.HideToolbar,
		HideMenubar:       vpJSON.HideMenubar,
		HideWindowUI:      vpJSON.HideWindowUI,
		FitWindow:         vpJSON.FitWindow,
		CenterWindow:      vpJSON.CenterWindow,
		DisplayDocTitle:   vpJSON.DisplayDocTitle,
		PickTrayByPDFSize: vpJSON.PickTrayByPDFSize,
		NumCopies:         (*types.Integer)(vpJSON.NumCopies),
	}

	if vp.NumCopies != nil && *vp.NumCopies < 1 {
		return errors.Errorf("pdfcpu: invalid \"NumCopies\", got: %d, want a numerical value > 0", *vp.NumCopies)
	}

	vp.NonFullScreenPageMode = (*NonFullScreenPageMode)(PageModeFor(vpJSON.NonFullScreenPageMode))
	if vpJSON.NonFullScreenPageMode != "" {
		if vp.NonFullScreenPageMode == nil {
			return errors.Errorf("pdfcpu: unknown \"NonFullScreenPageMode\", got: %s want one of: UseNone, UseOutlines, UseThumbs, UseOC\n", vpJSON.NonFullScreenPageMode)
		}
		pm := (PageMode)(*vp.NonFullScreenPageMode)
		if pm == PageModeFullScreen || pm == PageModeUseAttachments {
			return errors.Errorf("pdfcpu: unknown \"NonFullScreenPageMode\", got: %s want one of: UseNone, UseOutlines, UseThumbs, UseOC\n", vpJSON.NonFullScreenPageMode)
		}
	}

	vp.Direction = DirectionFor(vpJSON.Direction)
	if vpJSON.Direction != "" && vp.Direction == nil {
		return errors.Errorf("pdfcpu: unknown \"Direction\", got: %s want one of: L2R, R2L\n", vpJSON.Direction)
	}

	vp.ViewArea = PageBoundaryFor(vpJSON.ViewArea)
	if vpJSON.ViewArea != "" && vp.ViewArea == nil {
		return errors.Errorf("pdfcpu: unknown \"ViewArea\", got: %s want one of: MediaBox, CropBox, TrimBox, BleedBox, ArtBox\n", vpJSON.ViewArea)
	}

	vp.ViewClip = PageBoundaryFor(vpJSON.ViewClip)
	if vpJSON.ViewClip != "" && vp.ViewClip == nil {
		return errors.Errorf("pdfcpu: unknown \"ViewClip\", got: %s want one of: MediaBox, CropBox, TrimBox, BleedBox, ArtBox\n", vpJSON.ViewClip)
	}

	return vp.unmarshalPrinterPreferences(vpJSON)
}

func renderViewerFlags(vp ViewerPreferences, ss *[]string) {
	if vp.HideToolbar != nil {
		*ss = append(*ss, fmt.Sprintf("%22s%s = %t", "", "HideToolbar", *vp.HideToolbar))
	}

	if vp.HideMenubar != nil {
		*ss = append(*ss, fmt.Sprintf("%22s%s = %t", "", "HideMenubar", *vp.HideMenubar))
	}

	if vp.HideWindowUI != nil {
		*ss = append(*ss, fmt.Sprintf("%22s%s = %t", "", "HideWindowUI", *vp.HideWindowUI))
	}

	if vp.FitWindow != nil {
		*ss = append(*ss, fmt.Sprintf("%22s%s = %t", "", "FitWindow", *vp.FitWindow))
	}

	if vp.CenterWindow != nil {
		*ss = append(*ss, fmt.Sprintf("%22s%s = %t", "", "CenterWindow", *vp.CenterWindow))
	}

	if vp.DisplayDocTitle != nil {
		*ss = append(*ss, fmt.Sprintf("%22s%s = %t", "", "DisplayDocTitle", *vp.DisplayDocTitle))
	}

	if vp.NonFullScreenPageMode != nil {
		pm := PageMode(*vp.NonFullScreenPageMode)
		*ss = append(*ss, fmt.Sprintf("%22s%s = %s", "", "NonFullScreenPageMode", pm.String()))
	}
}

func listViewerFlags(vp ViewerPreferences, ss *[]string) {
	if vp.HideToolbar != nil {
		*ss = append(*ss, fmt.Sprintf("%s = %t", "HideToolbar", *vp.HideToolbar))
	}

	if vp.HideMenubar != nil {
		*ss = append(*ss, fmt.Sprintf("%s = %t", "HideMenubar", *vp.HideMenubar))
	}

	if vp.HideWindowUI != nil {
		*ss = append(*ss, fmt.Sprintf("%s = %t", "HideWindowUI", *vp.HideWindowUI))
	}

	if vp.FitWindow != nil {
		*ss = append(*ss, fmt.Sprintf("%s = %t", "FitWindow", *vp.FitWindow))
	}

	if vp.CenterWindow != nil {
		*ss = append(*ss, fmt.Sprintf("%s = %t", "CenterWindow", *vp.CenterWindow))
	}

	if vp.DisplayDocTitle != nil {
		*ss = append(*ss, fmt.Sprintf("%s = %t", "DisplayDocTitle", *vp.DisplayDocTitle))
	}

	if vp.NonFullScreenPageMode != nil {
		pm := PageMode(*vp.NonFullScreenPageMode)
		*ss = append(*ss, fmt.Sprintf("%s = %s", "NonFullScreenPageMode", pm.String()))
	}
}

func (vp ViewerPreferences) listPrinterPreferences() []string {
	var ss []string

	if vp.PrintArea != nil {
		ss = append(ss, fmt.Sprintf("%s = %s", "PrintArea", vp.PrintArea))
	}

	if vp.PrintClip != nil {
		ss = append(ss, fmt.Sprintf("%s = %s", "PrintClip", vp.PrintClip))
	}

	if vp.PrintScaling != nil {
		ss = append(ss, fmt.Sprintf("%s = %s", "PrintScaling", vp.PrintScaling))
	}

	if vp.Duplex != nil {
		ss = append(ss, fmt.Sprintf("%s = %s", "Duplex", vp.Duplex))
	}

	if vp.PickTrayByPDFSize != nil {
		ss = append(ss, fmt.Sprintf("%s = %t", "PickTrayByPDFSize", *vp.PickTrayByPDFSize))
	}

	if len(vp.PrintPageRange) > 0 {
		var ss1 []string
		for i := 0; i < len(vp.PrintPageRange); i += 2 {
			ss1 = append(ss1, fmt.Sprintf("%d-%d", vp.PrintPageRange[i].(types.Integer), vp.PrintPageRange[i+1].(types.Integer)))
		}
		ss = append(ss, fmt.Sprintf("%s = %s", "PrintPageRange", strings.Join(ss1, ",")))
	}

	if vp.NumCopies != nil {
		ss = append(ss, fmt.Sprintf("%s = %d", "NumCopies", *vp.NumCopies))
	}

	if len(vp.Enforce) > 0 {
		var ss1 []string
		for _, v := range vp.Enforce {
			ss1 = append(ss1, v.String())
		}
		ss = append(ss, fmt.Sprintf("%s = %s", "Enforce", strings.Join(ss1, ",")))
	}

	return ss
}

// List generates output for the viewer pref command.
func (vp ViewerPreferences) List() []string {
	var ss []string

	listViewerFlags(vp, &ss)

	if vp.Direction != nil {
		ss = append(ss, fmt.Sprintf("%s = %s", "Direction", vp.Direction))
	}

	if vp.ViewArea != nil {
		ss = append(ss, fmt.Sprintf("%s = %s", "ViewArea", vp.ViewArea))
	}

	if vp.ViewClip != nil {
		ss = append(ss, fmt.Sprintf("%s = %s", "ViewClip", vp.ViewClip))
	}

	ss = append(ss, vp.listPrinterPreferences()...)

	if len(ss) > 0 {
		ss1 := []string{"Viewer preferences:"}
		for _, s := range ss {
			ss1 = append(ss1, "   "+s)
		}
		ss = ss1
	} else {
		ss = append(ss, "No viewer preferences available.")
	}

	return ss
}

// String generates output for the info command.
func (vp ViewerPreferences) String() string {
	var ss []string

	renderViewerFlags(vp, &ss)

	if vp.Direction != nil {
		ss = append(ss, fmt.Sprintf("%22s%s = %s", "", "Direction", vp.Direction))
	}

	if vp.ViewArea != nil {
		ss = append(ss, fmt.Sprintf("%22s%s = %s", "", "ViewArea", vp.ViewArea))
	}

	if vp.ViewClip != nil {
		ss = append(ss, fmt.Sprintf("%22s%s = %s", "", "ViewClip", vp.ViewClip))
	}

	if vp.PrintArea != nil {
		ss = append(ss, fmt.Sprintf("%22s%s = %s", "", "PrintArea", vp.PrintArea))
	}

	if vp.PrintClip != nil {
		ss = append(ss, fmt.Sprintf("%22s%s = %s", "", "PrintClip", vp.PrintClip))
	}

	if vp.PrintScaling != nil {
		ss = append(ss, fmt.Sprintf("%22s%s = %s", "", "PrintScaling", vp.PrintScaling))
	}

	if vp.Duplex != nil {
		ss = append(ss, fmt.Sprintf("%22s%s = %s", "", "Duplex", vp.Duplex))
	}

	if vp.PickTrayByPDFSize != nil {
		ss = append(ss, fmt.Sprintf("%22s%s = %t", "", "PickTrayByPDFSize", *vp.PickTrayByPDFSize))
	}

	if len(vp.PrintPageRange) > 0 {
		var ss1 []string
		for i := 0; i < len(vp.PrintPageRange); i += 2 {
			ss1 = append(ss1, fmt.Sprintf("%d-%d", vp.PrintPageRange[i].(types.Integer), vp.PrintPageRange[i+1].(types.Integer)))
		}
		ss = append(ss, fmt.Sprintf("%22s%s = %s", "", "PrintPageRange", strings.Join(ss1, ",")))
	}

	if vp.NumCopies != nil {
		ss = append(ss, fmt.Sprintf("%22s%s = %d", "", "NumCopies", *vp.NumCopies))
	}

	if len(vp.Enforce) > 0 {
		var ss1 []string
		for _, v := range vp.Enforce {
			ss1 = append(ss1, v.String())
		}
		ss = append(ss, fmt.Sprintf("%22s%s = %s", "", "Enforce", strings.Join(ss1, ",")))
	}

	return strings.TrimSpace(strings.Join(ss, "\n"))
}
