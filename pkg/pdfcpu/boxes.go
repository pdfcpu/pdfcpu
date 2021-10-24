/*
Copyright 2020 The pdfcpu Authors.

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
	"fmt"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/types"
	"github.com/pkg/errors"
)

// Box is a rectangular region in user space
// expressed either explicitly via Rect
// or implicitly via margins applied to the containing parent box.
// Media box serves as parent box for crop box.
// Crop box serves as parent box for trim, bleed and art box.
type Box struct {
	Rect      *Rectangle // Rectangle in user space.
	Inherited bool       // Media box and Crop box may be inherited.
	RefBox    string     // Use position of another box,
	// Margins to parent box in points.
	// Relative to parent box if 0 < x < 0.5
	MLeft, MRight float64
	MTop, MBot    float64
	// Relative position within parent box
	Dim    *Dim   // dimensions
	Pos    Anchor // position anchor within parent box, one of tl,tc,tr,l,c,r,bl,bc,br.
	Dx, Dy int    // anchor offset
}

// PageBoundaries represent the defined PDF page boundaries.
type PageBoundaries struct {
	Media *Box
	Crop  *Box
	Trim  *Box
	Bleed *Box
	Art   *Box
	Rot   int // The effective page rotation.
}

// SelectAll selects all page boundaries.
func (pb *PageBoundaries) SelectAll() {
	b := &Box{}
	pb.Media, pb.Crop, pb.Trim, pb.Bleed, pb.Art = b, b, b, b, b
}

func (pb PageBoundaries) String() string {
	ss := []string{}
	if pb.Media != nil {
		ss = append(ss, "mediaBox")
	}
	if pb.Crop != nil {
		ss = append(ss, "cropBox")
	}
	if pb.Trim != nil {
		ss = append(ss, "trimBox")
	}
	if pb.Bleed != nil {
		ss = append(ss, "bleedBox")
	}
	if pb.Art != nil {
		ss = append(ss, "artBox")
	}
	return strings.Join(ss, ", ")
}

// MediaBox returns the effective mediabox for pb.
func (pb PageBoundaries) MediaBox() *Rectangle {
	if pb.Media == nil {
		return nil
	}
	return pb.Media.Rect
}

// CropBox returns the effective cropbox for pb.
func (pb PageBoundaries) CropBox() *Rectangle {
	if pb.Crop == nil || pb.Crop.Rect == nil {
		return pb.MediaBox()
	}
	return pb.Crop.Rect
}

// TrimBox returns the effective trimbox for pb.
func (pb PageBoundaries) TrimBox() *Rectangle {
	if pb.Trim == nil || pb.Trim.Rect == nil {
		return pb.CropBox()
	}
	return pb.Trim.Rect
}

// BleedBox returns the effective bleedbox for pb.
func (pb PageBoundaries) BleedBox() *Rectangle {
	if pb.Bleed == nil || pb.Bleed.Rect == nil {
		return pb.CropBox()
	}
	return pb.Bleed.Rect
}

// ArtBox returns the effective artbox for pb.
func (pb PageBoundaries) ArtBox() *Rectangle {
	if pb.Art == nil || pb.Art.Rect == nil {
		return pb.CropBox()
	}
	return pb.Art.Rect
}

// ResolveBox resolves s and tries to assign an empty page boundary.
func (pb *PageBoundaries) ResolveBox(s string) error {
	for _, k := range []string{"media", "crop", "trim", "bleed", "art"} {
		b := &Box{}
		if strings.HasPrefix(k, s) {
			switch k {
			case "media":
				pb.Media = b
			case "crop":
				pb.Crop = b
			case "trim":
				pb.Trim = b
			case "bleed":
				pb.Bleed = b
			case "art":
				pb.Art = b
			}
			return nil
		}
	}
	return errors.Errorf("pdfcpu: invalid box prefix: %s", s)
}

// ParseBoxList parses a list of box types.
func ParseBoxList(s string) (*PageBoundaries, error) {
	// A comma separated, unsorted list of values:
	//
	// m(edia), c(rop), t(rim), b(leed), a(rt)

	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return nil, nil
	}
	pb := &PageBoundaries{}
	for _, s := range strings.Split(s, ",") {
		if err := pb.ResolveBox(strings.TrimSpace(s)); err != nil {
			return nil, err
		}
	}
	return pb, nil
}

func resolveBoxType(s string) (string, error) {
	for _, k := range []string{"media", "crop", "trim", "bleed", "art"} {
		if strings.HasPrefix(k, s) {
			return k, nil
		}
	}
	return "", errors.Errorf("pdfcpu: invalid box type: %s", s)
}

func processBox(b **Box, boxID, paramValueStr string, unit DisplayUnit) error {
	var err error
	if *b != nil {
		return errors.Errorf("pdfcpu: duplicate box definition: %s", boxID)
	}
	// process box assignment
	boxVal, err := resolveBoxType(paramValueStr)
	if err == nil {
		if boxVal == boxID {
			return errors.Errorf("pdfcpu: invalid box self assigment: %s", boxID)
		}
		*b = &Box{RefBox: boxVal}
		return nil
	}
	// process box definition
	*b, err = ParseBox(paramValueStr, unit)
	return err
}

// ParsePageBoundaries parses a list of box definitions and assignments.
func ParsePageBoundaries(s string, unit DisplayUnit) (*PageBoundaries, error) {
	// A sequence of box definitions/assignments:
	//
	// 	  m(edia): {box}
	//     c(rop): {box}
	//      a(rt): {box} | b(leed) | c(rop)  | m(edia) | t(rim)
	//    b(leed): {box} | a(rt)   | c(rop)  | m(edia) | t(rim)
	//     t(rim): {box} | a(rt)   | b(leed) | c(rop)  | m(edia)

	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return nil, errors.New("pdfcpu: missing page boundaries in the form of box definitions/assignments")
	}
	pb := &PageBoundaries{}
	for _, s := range strings.Split(s, ",") {

		s1 := strings.Split(s, ":")
		if len(s1) != 2 {
			return nil, errors.New("pdfcpu: invalid box assignment")
		}

		paramPrefix := strings.TrimSpace(s1[0])
		paramValueStr := strings.TrimSpace(s1[1])

		boxKey, err := resolveBoxType(paramPrefix)
		if err != nil {
			return nil, errors.New("pdfcpu: invalid box type")
		}

		// process box definition
		switch boxKey {
		case "media":
			if pb.Media != nil {
				return nil, errors.New("pdfcpu: duplicate box definition: media")
			}
			// process media box definition
			pb.Media, err = ParseBox(paramValueStr, unit)

		case "crop":
			if pb.Crop != nil {
				return nil, errors.New("pdfcpu: duplicate box definition: crop")
			}
			// process crop box definition
			pb.Crop, err = ParseBox(paramValueStr, unit)

		case "trim":
			err = processBox(&pb.Trim, "trim", paramValueStr, unit)

		case "bleed":
			err = processBox(&pb.Bleed, "bleed", paramValueStr, unit)

		case "art":
			err = processBox(&pb.Art, "art", paramValueStr, unit)

		}

		if err != nil {
			return nil, err
		}
	}
	return pb, nil
}

func parseBoxByRectangle(s string, u DisplayUnit) (*Box, error) {
	ss := strings.Fields(s)
	if len(ss) != 4 {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	f, err := strconv.ParseFloat(ss[0], 64)
	if err != nil {
		return nil, err
	}
	xmin := toUserSpace(f, u)

	f, err = strconv.ParseFloat(ss[1], 64)
	if err != nil {
		return nil, err
	}
	ymin := toUserSpace(f, u)

	f, err = strconv.ParseFloat(ss[2], 64)
	if err != nil {
		return nil, err
	}
	xmax := toUserSpace(f, u)

	f, err = strconv.ParseFloat(ss[3], 64)
	if err != nil {
		return nil, err
	}
	ymax := toUserSpace(f, u)

	if xmax < xmin {
		xmin, xmax = xmax, xmin
	}

	if ymax < ymin {
		ymin, ymax = ymax, ymin
	}

	return &Box{Rect: Rect(xmin, ymin, xmax, ymax)}, nil
}

func parseBoxPercentage(s string) (float64, error) {
	pct, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if pct <= -50 || pct >= 50 {
		return 0, errors.Errorf("pdfcpu: invalid margin percentage: %s must be < 50%%", s)
	}
	return pct / 100, nil
}

func parseBoxBySingleMarginVal(s, s1 string, abs bool, u DisplayUnit) (*Box, error) {
	if s1[len(s1)-1] == '%' {
		// margin percentage
		// 10.5%
		// % has higher precedence than abs/rel.
		s1 = s1[:len(s1)-1]
		if len(s1) == 0 {
			return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
		}
		m, err := parseBoxPercentage(s1)
		if err != nil {
			return nil, err
		}
		return &Box{MLeft: m, MRight: m, MTop: m, MBot: m}, nil
	}
	m, err := strconv.ParseFloat(s1, 64)
	if err != nil {
		return nil, err
	}
	if !abs {
		// 0.25 rel (=25%)
		if m <= 0 || m >= .5 {
			return nil, errors.Errorf("pdfcpu: invalid relative box margin: %f must be positive < 0.5", m)
		}
		return &Box{MLeft: m, MRight: m, MTop: m, MBot: m}, nil
	}
	// 10
	// 10 abs
	// .5
	// .5 abs
	m = toUserSpace(m, u)
	return &Box{MLeft: m, MRight: m, MTop: m, MBot: m}, nil
}

func parseBoxBy2Percentages(s, s1, s2 string) (*Box, error) {
	// 10% 40%
	// Parse vert margin.
	s1 = s1[:len(s1)-1]
	if len(s1) == 0 {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	vm, err := parseBoxPercentage(s1)
	if err != nil {
		return nil, err
	}

	if s2[len(s2)-1] != '%' {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	// Parse hor margin.
	s2 = s2[:len(s2)-1]
	if len(s2) == 0 {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	hm, err := parseBoxPercentage(s2)
	if err != nil {
		return nil, err
	}
	return &Box{MLeft: hm, MRight: hm, MTop: vm, MBot: vm}, nil
}

func parseBoxBy2MarginVals(s, s1, s2 string, abs bool, u DisplayUnit) (*Box, error) {
	if s1[len(s1)-1] == '%' {
		return parseBoxBy2Percentages(s, s1, s2)
	}

	// 10 5
	// 10 5 abs
	// .1 .5
	// .1 .5 abs
	// .1 .4 rel
	vm, err := strconv.ParseFloat(s1, 64)
	if err != nil {
		return nil, err
	}
	if !abs {
		// eg 0.25 rel (=25%)
		if vm <= 0 || vm >= .5 {
			return nil, errors.Errorf("pdfcpu: invalid relative vertical box margin: %f must be positive < 0.5", vm)
		}
	}
	hm, err := strconv.ParseFloat(s2, 64)
	if err != nil {
		return nil, err
	}
	if !abs {
		// eg 0.25 rel (=25%)
		if hm <= 0 || hm >= .5 {
			return nil, errors.Errorf("pdfcpu: invalid relative horizontal box margin: %f must be positive < 0.5", hm)
		}
	}
	if abs {
		vm = toUserSpace(vm, u)
		hm = toUserSpace(hm, u)
	}
	return &Box{MLeft: hm, MRight: hm, MTop: vm, MBot: vm}, nil
}

func parseBoxBy3Percentages(s, s1, s2, s3 string) (*Box, error) {
	// 10% 15.5% 10%
	// Parse top margin.
	s1 = s1[:len(s1)-1]
	if len(s1) == 0 {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	pct, err := strconv.ParseFloat(s1, 64)
	if err != nil {
		return nil, err
	}
	tm := pct / 100

	if s2[len(s2)-1] != '%' {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	// Parse hor margin.
	s2 = s2[:len(s2)-1]
	if len(s2) == 0 {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	hm, err := parseBoxPercentage(s2)
	if err != nil {
		return nil, err
	}

	if s3[len(s3)-1] != '%' {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	// Parse bottom margin.
	s3 = s3[:len(s3)-1]
	if len(s3) == 0 {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	pct, err = strconv.ParseFloat(s3, 64)
	if err != nil {
		return nil, err
	}
	bm := pct / 100
	if tm+bm >= 1 {
		return nil, errors.Errorf("pdfcpu: vertical margin overflow: %s", s)
	}

	return &Box{MLeft: hm, MRight: hm, MTop: tm, MBot: bm}, nil
}

func parseBoxBy3MarginVals(s, s1, s2, s3 string, abs bool, u DisplayUnit) (*Box, error) {
	if s1[len(s1)-1] == '%' {
		return parseBoxBy3Percentages(s, s1, s2, s3)
	}

	// 10 5 15 				... absolute, top:10 left,right:5 bottom:15
	// 10 5 15 abs			... absolute, top:10 left,right:5 bottom:15
	// .1 .155 .1			... absolute, top:.1 left,right:.155 bottom:.1
	// .1 .155 .1 abs		... absolute, top:.1 left,right:.155 bottom:.1
	// .1 .155 .1 rel 		... relative, top:.1 left,right:.155 bottom:.1
	tm, err := strconv.ParseFloat(s1, 64)
	if err != nil {
		return nil, err
	}

	hm, err := strconv.ParseFloat(s2, 64)
	if err != nil {
		return nil, err
	}
	if !abs {
		// eg 0.25 rel (=25%)
		if hm <= 0 || hm >= .5 {
			return nil, errors.Errorf("pdfcpu: invalid relative horizontal box margin: %f must be positive < 0.5", hm)
		}
	}

	bm, err := strconv.ParseFloat(s3, 64)
	if err != nil {
		return nil, err
	}
	if !abs && (tm+bm >= 1) {
		return nil, errors.Errorf("pdfcpu: vertical margin overflow: %s", s)
	}

	if abs {
		tm = toUserSpace(tm, u)
		hm = toUserSpace(hm, u)
		bm = toUserSpace(bm, u)
	}
	return &Box{MLeft: hm, MRight: hm, MTop: tm, MBot: bm}, nil
}

func parseBoxBy4Percentages(s, s1, s2, s3, s4 string) (*Box, error) {
	// 10% 15% 15% 10%
	// Parse top margin.
	s1 = s1[:len(s1)-1]
	if len(s1) == 0 {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	pct, err := strconv.ParseFloat(s1, 64)
	if err != nil {
		return nil, err
	}
	tm := pct / 100

	// Parse right margin.
	if s2[len(s2)-1] != '%' {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	s2 = s2[:len(s2)-1]
	if len(s2) == 0 {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	pct, err = strconv.ParseFloat(s1, 64)
	if err != nil {
		return nil, err
	}
	rm := pct / 100

	// Parse bottom margin.
	if s3[len(s3)-1] != '%' {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	s3 = s3[:len(s3)-1]
	if len(s3) == 0 {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	pct, err = strconv.ParseFloat(s3, 64)
	if err != nil {
		return nil, err
	}
	bm := pct / 100

	// Parse left margin.
	if s4[len(s4)-1] != '%' {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	s4 = s4[:len(s4)-1]
	if len(s4) == 0 {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	pct, err = strconv.ParseFloat(s3, 64)
	if err != nil {
		return nil, err
	}
	lm := pct / 100

	if tm+bm >= 1 {
		return nil, errors.Errorf("pdfcpu: vertical margin overflow: %s", s)
	}
	if rm+lm >= 1 {
		return nil, errors.Errorf("pdfcpu: horizontal margin overflow: %s", s)
	}

	return &Box{MLeft: lm, MRight: rm, MTop: tm, MBot: bm}, nil
}

func parseBoxBy4MarginVals(s, s1, s2, s3, s4 string, abs bool, u DisplayUnit) (*Box, error) {
	if s1[len(s1)-1] == '%' {
		return parseBoxBy4Percentages(s, s1, s2, s3, s4)
	}

	// 0.4 0.4 20 20		... absolute, top:.4 right:.4 bottom:20 left:20
	// 0.4 0.4 .1 .1		... absolute, top:.4 right:.4 bottom:.1 left:.1
	// 0.4 0.4 .1 .1 abs	... absolute, top:.4 right:.4 bottom:.1 left:.1
	// 0.4 0.4 .1 .1 rel  	... relative, top:.4 right:.4 bottom:.1 left:.1

	// Parse top margin.
	tm, err := strconv.ParseFloat(s1, 64)
	if err != nil {
		return nil, err
	}

	// Parse right margin.
	rm, err := strconv.ParseFloat(s2, 64)
	if err != nil {
		return nil, err
	}

	// Parse bottom margin.
	bm, err := strconv.ParseFloat(s3, 64)
	if err != nil {
		return nil, err
	}

	// Parse left margin.
	lm, err := strconv.ParseFloat(s4, 64)
	if err != nil {
		return nil, err
	}
	if !abs {
		if tm+bm >= 1 {
			return nil, errors.Errorf("pdfcpu: vertical margin overflow: %s", s)
		}
		if lm+rm >= 1 {
			return nil, errors.Errorf("pdfcpu: horizontal margin overflow: %s", s)
		}
	}

	if abs {
		tm = toUserSpace(tm, u)
		rm = toUserSpace(rm, u)
		bm = toUserSpace(bm, u)
		lm = toUserSpace(lm, u)
	}
	return &Box{MLeft: lm, MRight: rm, MTop: tm, MBot: bm}, nil
}

func parseBoxOffset(s string, b *Box, u DisplayUnit) error {
	d := strings.Split(s, " ")
	if len(d) != 2 {
		return errors.Errorf("pdfcpu: illegal position offset string: need 2 numeric values, %s\n", s)
	}

	f, err := strconv.ParseFloat(d[0], 64)
	if err != nil {
		return err
	}
	b.Dx = int(toUserSpace(f, u))

	f, err = strconv.ParseFloat(d[1], 64)
	if err != nil {
		return err
	}
	b.Dy = int(toUserSpace(f, u))

	return nil
}

func parseBoxDimByPercentage(s, s1, s2 string, b *Box) error {
	// 10% 40%
	// Parse width.
	s1 = s1[:len(s1)-1]
	if len(s1) == 0 {
		return errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	pct, err := strconv.ParseFloat(s1, 64)
	if err != nil {
		return err
	}
	if pct <= 0 || pct >= 100 {
		return errors.Errorf("pdfcpu: invalid percentage: %s", s)
	}
	w := pct / 100

	if s2[len(s2)-1] != '%' {
		return errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	// Parse height.
	s2 = s2[:len(s2)-1]
	if len(s2) == 0 {
		return errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	pct, err = strconv.ParseFloat(s2, 64)
	if err != nil {
		return err
	}
	if pct <= 0 || pct >= 100 {
		return errors.Errorf("pdfcpu: invalid percentage: %s", s)
	}
	h := pct / 100
	b.Dim = &Dim{w, h}
	return nil
}

func parseBoxDimWidthAndHeight(s1, s2 string, abs bool) (float64, float64, error) {
	var (
		w, h float64
		err  error
	)

	w, err = strconv.ParseFloat(s1, 64)
	if err != nil {
		return w, h, err
	}
	if !abs {
		// eg 0.25 rel (=25%)
		if w <= 0 || w >= 1 {
			return w, h, errors.Errorf("pdfcpu: invalid relative box width: %f must be positive < 1", w)
		}
	}

	h, err = strconv.ParseFloat(s2, 64)
	if err != nil {
		return w, h, err
	}
	if !abs {
		// eg 0.25 rel (=25%)
		if h <= 0 || h >= 1 {
			return w, h, errors.Errorf("pdfcpu: invalid relative box height: %f must be positive < 1", h)
		}
	}

	return w, h, nil
}

func parseBoxDim(s string, b *Box, u DisplayUnit) error {
	ss := strings.Fields(s)
	if len(ss) != 2 && len(ss) != 3 {
		return errors.Errorf("pdfcpu: illegal dimension string: need 2 positive numeric values, %s\n", s)
	}
	abs := true
	if len(ss) == 3 {
		s1 := ss[2]
		if s1 != "rel" && s1 != "abs" {
			return errors.New("pdfcpu: illegal dimension string")
		}
		abs = s1 == "abs"
	}

	s1, s2 := ss[0], ss[1]
	if s1[len(s1)-1] == '%' {
		return parseBoxDimByPercentage(s, s1, s2, b)
	}

	w, h, err := parseBoxDimWidthAndHeight(s1, s2, abs)
	if err != nil {
		return err
	}

	if abs {
		w = toUserSpace(w, u)
		h = toUserSpace(h, u)
	}
	b.Dim = &Dim{w, h}
	return nil
}

func parseBoxByPosWithinParent(s string, ss []string, u DisplayUnit) (*Box, error) {
	b := &Box{Pos: Center}
	for _, s := range ss {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		switch paramPrefix {
		case "dim":
			if err := parseBoxDim(paramValueStr, b, u); err != nil {
				return nil, err
			}

		case "pos":
			a, err := parsePositionAnchor(paramValueStr)
			if err != nil {
				return nil, err
			}
			b.Pos = a

		case "off":
			if err := parseBoxOffset(paramValueStr, b, u); err != nil {
				return nil, err
			}

		default:
			return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
		}
	}
	if b.Dim == nil {
		return nil, errors.New("pdfcpu: missing box definition attr dim")
	}
	return b, nil
}

func parseBoxByMarginVals(ss []string, s string, abs bool, u DisplayUnit) (*Box, error) {
	switch len(ss) {
	case 1:
		return parseBoxBySingleMarginVal(s, ss[0], abs, u)
	case 2:
		return parseBoxBy2MarginVals(s, ss[0], ss[1], abs, u)
	case 3:
		return parseBoxBy3MarginVals(s, ss[0], ss[1], ss[2], abs, u)
	case 4:
		return parseBoxBy4MarginVals(s, ss[0], ss[1], ss[2], ss[3], abs, u)
	case 5:
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	return nil, nil
}

// ParseBox parses a box definition.
func ParseBox(s string, u DisplayUnit) (*Box, error) {
	// A rectangular region in userspace expressed in terms of
	// a rectangle or margins relative to its parent box.
	// Media box serves as parent/default for crop box.
	// Crop box serves as parent/default for trim, bleed and art box:

	// [0 10 200 150]		... rectangle

	// 0.5 0.5 20 20		... absolute, top:.5 right:.5 bottom:20 left:20
	// 0.5 0.5 .1 .1 abs	... absolute, top:.5 right:.5 bottom:.1 left:.1
	// 0.5 0.5 .1 .1 rel  	... relative, top:.5 right:.5 bottom:20 left:20
	// 10                 	... absolute, top,right,bottom,left:10
	// 10 5               	... absolute, top,bottom:10  left,right:5
	// 10 5 15            	... absolute, top:10 left,right:5 bottom:15
	// 5%         <50%      ... relative, top,right,bottom,left:5% of parent box width/height
	// .1 .5              	... absolute, top,bottom:.1  left,right:.5
	// .1 .3 rel         	... relative, top,bottom:.1=10%  left,right:.3=30%
	// -10                	... absolute, top,right,bottom,left enlarging the parent box as well

	// dim:30 30			... 30 x 30 display units, anchored at center of parent box
	// dim:30 30 abs		... 30 x 30 display units, anchored at center of parent box
	// dim:.3 .3 rel  		... 0.3 x 0.3 relative width/height of parent box, anchored at center of parent box
	// dim:30% 30%			... 0.3 x 0.3 relative width/height of parent box, anchored at center of parent box
	// pos:tl, dim:30 30	... 0.3 x 0.3 relative width/height of parent box, anchored at top left corner of parent box
	// pos:bl, off: 5 5, dim:30 30			...30 x 30 display units with offset 5/5, anchored at bottom left corner of parent box
	// pos:bl, off: -5 -5, dim:.3 .3 rel 	...0.3 x 0.3 relative width/height and anchored at bottom left corner of parent box

	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return nil, nil
	}

	if s[0] == '[' && s[len(s)-1] == ']' {
		// Rectangle in PDF Array notation.
		return parseBoxByRectangle(s[1:len(s)-1], u)
	}

	// Via relative position within parent box.
	ss := strings.Split(s, ",")
	if len(ss) > 3 {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	if len(ss) > 1 || strings.HasPrefix(ss[0], "dim") {
		return parseBoxByPosWithinParent(s, ss, u)
	}

	// Via margins relative to parent box.
	ss = strings.Fields(s)
	if len(ss) > 5 {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}
	if len(ss) == 1 && (ss[0] == "abs" || ss[0] == "rel") {
		return nil, errors.Errorf("pdfcpu: invalid box definition: %s", s)
	}

	abs := true
	l := len(ss) - 1
	s1 := ss[l]
	if s1 == "rel" || s1 == "abs" {
		abs = s1 == "abs"
		ss = ss[:l]
	}

	return parseBoxByMarginVals(ss, s, abs, u)
}

func (ctx *Context) addPageBoundaryString(i int, pb PageBoundaries, wantPB *PageBoundaries) []string {
	unit := ctx.unit()
	ss := []string{}
	d := pb.CropBox().Dimensions()
	if pb.Rot%180 != 0 {
		d.Width, d.Height = d.Height, d.Width
	}
	or := "portrait"
	if d.Landscape() {
		or = "landscape"
	}

	s := fmt.Sprintf("rot=%+d orientation:%s", pb.Rot, or)
	ss = append(ss, fmt.Sprintf("Page %d: %s", i+1, s))
	if wantPB.Media != nil {
		s := ""
		if pb.Media.Inherited {
			s = "(inherited)"
		}
		ss = append(ss, fmt.Sprintf("  MediaBox (%s) %v %s", unit, pb.MediaBox().Format(ctx.Unit), s))
	}
	if wantPB.Crop != nil {
		s := ""
		if pb.Crop == nil {
			s = "(default)"
		} else if pb.Crop.Inherited {
			s = "(inherited)"
		}
		ss = append(ss, fmt.Sprintf("   CropBox (%s) %v %s", unit, pb.CropBox().Format(ctx.Unit), s))
	}
	if wantPB.Trim != nil {
		s := ""
		if pb.Trim == nil {
			s = "(default)"
		}
		ss = append(ss, fmt.Sprintf("   TrimBox (%s) %v %s", unit, pb.TrimBox().Format(ctx.Unit), s))
	}
	if wantPB.Bleed != nil {
		s := ""
		if pb.Bleed == nil {
			s = "(default)"
		}
		ss = append(ss, fmt.Sprintf("  BleedBox (%s) %v %s", unit, pb.BleedBox().Format(ctx.Unit), s))
	}
	if wantPB.Art != nil {
		s := ""
		if pb.Art == nil {
			s = "(default)"
		}
		ss = append(ss, fmt.Sprintf("    ArtBox (%s) %v %s", unit, pb.ArtBox().Format(ctx.Unit), s))
	}
	return append(ss, "")
}

// ListPageBoundaries lists page boundaries specified in wantPB for selected pages.
func (ctx *Context) ListPageBoundaries(selectedPages IntSet, wantPB *PageBoundaries) ([]string, error) {
	// TODO ctx.PageBoundaries(selectedPages)
	pbs, err := ctx.PageBoundaries()
	if err != nil {
		return nil, err
	}
	ss := []string{}
	for i, pb := range pbs {
		if _, found := selectedPages[i+1]; !found {
			continue
		}
		ss = append(ss, ctx.addPageBoundaryString(i, pb, wantPB)...)
	}

	return ss, nil
}

// RemovePageBoundaries removes page boundaries specified by pb for selected pages.
// The media box is mandatory (inherited or not) and can't be removed.
// A removed crop box defaults to the media box.
// Removed trim/bleed/art boxes default to the crop box.
func (ctx *Context) RemovePageBoundaries(selectedPages IntSet, pb *PageBoundaries) error {
	for k, v := range selectedPages {
		if !v {
			continue
		}
		d, _, inhPAttrs, err := ctx.PageDict(k, false)
		if err != nil {
			return err
		}
		if pb.Crop != nil {
			if oldVal := d.Delete("CropBox"); oldVal == nil {
				d.Insert("CropBox", inhPAttrs.MediaBox.Array())
			}
		}
		if pb.Trim != nil {
			d.Delete("TrimBox")
		}
		if pb.Bleed != nil {
			d.Delete("BleedBox")
		}
		if pb.Art != nil {
			d.Delete("ArtBox")
		}
	}
	return nil
}

func boxLowerLeftCorner(r *Rectangle, w, h float64, a Anchor) types.Point {
	var p types.Point

	switch a {

	case TopLeft:
		p.X = r.LL.X
		p.Y = r.UR.Y - h

	case TopCenter:
		p.X = r.UR.X - r.Width()/2 - w/2
		p.Y = r.UR.Y - h

	case TopRight:
		p.X = r.UR.X - w
		p.Y = r.UR.Y - h

	case Left:
		p.X = r.LL.X
		p.Y = r.UR.Y - r.Height()/2 - h/2

	case Center:
		p.X = r.UR.X - r.Width()/2 - w/2
		p.Y = r.UR.Y - r.Height()/2 - h/2

	case Right:
		p.X = r.UR.X - w
		p.Y = r.UR.Y - r.Height()/2 - h/2

	case BottomLeft:
		p.X = r.LL.X
		p.Y = r.LL.Y

	case BottomCenter:
		p.X = r.UR.X - r.Width()/2 - w/2
		p.Y = r.LL.Y

	case BottomRight:
		p.X = r.UR.X - w
		p.Y = r.LL.Y
	}

	return p
}

func boxByDim(boxName string, b *Box, d Dict, parent *Rectangle) *Rectangle {
	w := b.Dim.Width
	if w < 1 {
		w *= parent.Width()
	}
	h := b.Dim.Height
	if h < 1 {
		h *= parent.Height()
	}
	ll := boxLowerLeftCorner(parent, w, h, b.Pos)
	r := RectForWidthAndHeight(ll.X+float64(b.Dx), ll.Y+float64(b.Dy), w, h)
	if d != nil {
		d.Update(boxName, r.Array())
	}
	return r
}

func ApplyBox(boxName string, b *Box, d Dict, parent *Rectangle) *Rectangle {
	if b.Rect != nil {
		if d != nil {
			d.Update(boxName, b.Rect.Array())
		}
		return b.Rect
	}

	if b.Dim != nil {
		return boxByDim(boxName, b, d, parent)
	}

	mLeft, mRight, mTop, mBot := b.MLeft, b.MRight, b.MTop, b.MBot
	if b.MLeft != 0 && -1 < b.MLeft && b.MLeft < 1 {
		// Margins relative to media box
		mLeft *= parent.Width()
		mRight *= parent.Width()
		mBot *= parent.Height()
		mTop *= parent.Height()
	}
	xmin := parent.LL.X + mLeft
	ymin := parent.LL.Y + mBot
	xmax := parent.UR.X - mRight
	ymax := parent.UR.Y - mTop
	r := Rect(xmin, ymin, xmax, ymax)
	if d != nil {
		d.Update(boxName, r.Array())
	}
	if boxName != "CropBox" {
		return r
	}

	if xmin < parent.LL.X || ymin < parent.LL.Y || xmax > parent.UR.X || ymax > parent.UR.Y {
		// Expand media box.
		if xmin < parent.LL.X {
			parent.LL.X = xmin
		}
		if xmax > parent.UR.X {
			parent.UR.X = xmax
		}
		if ymin < parent.LL.Y {
			parent.LL.Y = ymin
		}
		if xmax > parent.UR.X {
			parent.UR.X = xmax
		}
		if ymax > parent.UR.Y {
			parent.UR.Y = ymax
		}
		if d != nil {
			d.Update("MediaBox", parent.Array())
		}
	}
	return r
}

type boxes struct {
	mediaBox, cropBox, trimBox, bleedBox, artBox *Rectangle
}

func applyBoxDefinitions(d Dict, pb *PageBoundaries, b *boxes) {
	parentBox := b.mediaBox
	if pb.Media != nil {
		//fmt.Println("add mb")
		b.mediaBox = ApplyBox("MediaBox", pb.Media, d, parentBox)
	}

	if pb.Crop != nil {
		//fmt.Println("add cb")
		b.cropBox = ApplyBox("CropBox", pb.Crop, d, parentBox)
	}

	if b.cropBox != nil {
		parentBox = b.cropBox
	}
	if pb.Trim != nil && pb.Trim.RefBox == "" {
		//fmt.Println("add tb")
		b.trimBox = ApplyBox("TrimBox", pb.Trim, d, parentBox)
	}

	if pb.Bleed != nil && pb.Bleed.RefBox == "" {
		//fmt.Println("add bb")
		b.bleedBox = ApplyBox("BleedBox", pb.Bleed, d, parentBox)
	}

	if pb.Art != nil && pb.Art.RefBox == "" {
		//fmt.Println("add ab")
		b.artBox = ApplyBox("ArtBox", pb.Art, d, parentBox)
	}
}

func updateTrimBox(d Dict, trimBox *Box, b *boxes) {
	var r *Rectangle
	switch trimBox.RefBox {
	case "media":
		r = b.mediaBox
	case "crop":
		r = b.cropBox
	case "bleed":
		r = b.bleedBox
		if r == nil {
			r = b.cropBox
		}
	case "art":
		r = b.artBox
		if r == nil {
			r = b.cropBox
		}
	}
	d.Update("TrimBox", r.Array())
	b.trimBox = r
}

func updateBleedBox(d Dict, bleedBox *Box, b *boxes) {
	var r *Rectangle
	switch bleedBox.RefBox {
	case "media":
		r = b.mediaBox
	case "crop":
		r = b.cropBox
	case "trim":
		r = b.trimBox
		if r == nil {
			r = b.cropBox
		}
	case "art":
		r = b.artBox
		if r == nil {
			r = b.cropBox
		}
	}
	d.Update("BleedBox", r.Array())
	b.bleedBox = r
}

func updateArtBox(d Dict, artBox *Box, b *boxes) {
	var r *Rectangle
	switch artBox.RefBox {
	case "media":
		r = b.mediaBox
	case "crop":
		r = b.cropBox
	case "trim":
		r = b.trimBox
		if r == nil {
			r = b.cropBox
		}
	case "bleed":
		r = b.bleedBox
		if r == nil {
			r = b.cropBox
		}
	}
	d.Update("ArtBox", r.Array())
	b.artBox = r
}

func applyBoxAssignments(d Dict, pb *PageBoundaries, b *boxes) {
	if pb.Trim != nil && pb.Trim.RefBox != "" {
		updateTrimBox(d, pb.Trim, b)
	}

	if pb.Bleed != nil && pb.Bleed.RefBox != "" {
		updateBleedBox(d, pb.Bleed, b)
	}

	if pb.Art != nil && pb.Art.RefBox != "" {
		updateArtBox(d, pb.Art, b)
	}
}

// AddPageBoundaries adds page boundaries specified by pb for selected pages.
func (ctx *Context) AddPageBoundaries(selectedPages IntSet, pb *PageBoundaries) error {
	for k, v := range selectedPages {
		if !v {
			continue
		}
		d, _, inhPAttrs, err := ctx.PageDict(k, false)
		if err != nil {
			return err
		}
		mediaBox := inhPAttrs.MediaBox
		cropBox := inhPAttrs.CropBox

		var trimBox *Rectangle
		obj, found := d.Find("TrimBox")
		if found {
			a, err := ctx.DereferenceArray(obj)
			if err != nil {
				return err
			}
			if trimBox, err = rect(ctx.XRefTable, a); err != nil {
				return err
			}
		}

		var bleedBox *Rectangle
		obj, found = d.Find("BleedBox")
		if found {
			a, err := ctx.DereferenceArray(obj)
			if err != nil {
				return err
			}
			if bleedBox, err = rect(ctx.XRefTable, a); err != nil {
				return err
			}
		}

		var artBox *Rectangle
		obj, found = d.Find("ArtBox")
		if found {
			a, err := ctx.DereferenceArray(obj)
			if err != nil {
				return err
			}
			if artBox, err = rect(ctx.XRefTable, a); err != nil {
				return err
			}
		}

		boxes := &boxes{mediaBox: mediaBox, cropBox: cropBox, trimBox: trimBox, bleedBox: bleedBox, artBox: artBox}
		applyBoxDefinitions(d, pb, boxes)
		applyBoxAssignments(d, pb, boxes)
	}
	return nil
}

// Crop sets crop box for selected pages to b.
func (ctx *Context) Crop(selectedPages IntSet, b *Box) error {
	for k, v := range selectedPages {
		if !v {
			continue
		}
		d, _, inhPAttrs, err := ctx.PageDict(k, false)
		if err != nil {
			return err
		}
		ApplyBox("CropBox", b, d, inhPAttrs.MediaBox)
	}
	return nil
}
