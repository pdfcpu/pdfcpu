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

package pdfcpu

import (
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// ParseCutConfigForPoster parses a Cut command string into an internal structure.
// formsize(=papersize) or dimensions, optionally: scalefactor, border, margin, bgcolor
func ParseCutConfigForPoster(s string, u types.DisplayUnit) (*model.Cut, error) {

	if s == "" {
		return nil, errors.New("pdfcpu: missing poster configuration string")
	}

	cut := &model.Cut{Unit: u}

	ss := strings.Split(s, ",")

	for _, s := range ss {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, errors.New("pdfcpu: Invalid poster configuration string. Please consult pdfcpu help poster")
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := model.CutParamMap.Handle(paramPrefix, paramValueStr, cut); err != nil {
			return nil, err
		}
	}

	if cut.UserDim && cut.PageSize != "" {
		return nil, errors.New("pdfcpu: poster - please supply either dimensions or form size ")
	}

	// Calc cut.Hor, cut.Vert based on current page cropbox only

	return cut, nil
}

// ParseCutConfigForN parses a NDown command string into an internal structure.
// n, Optionally: border, margin, bgcolor
func ParseCutConfigForN(n int, s string, u types.DisplayUnit) (*model.Cut, error) {

	if s == "" {
		return nil, errors.New("pdfcpu: missing ndown configuration string")
	}

	cut := &model.Cut{Unit: u}

	ss := strings.Split(s, ",")

	for _, s := range ss {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, errors.New("pdfcpu: Invalid ndown configuration string. Please consult pdfcpu help ndown")
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := model.CutParamMap.Handle(paramPrefix, paramValueStr, cut); err != nil {
			return nil, err
		}
	}

	cut.Hor, cut.Vert = []float64{}, []float64{}

	// Assuming current page in portrait mode

	switch n {
	case 2:
		cut.Hor = append(cut.Hor, .5)
	case 3:
		cut.Hor = append(cut.Hor, .33, .66)
	case 4:
		cut.Hor = append(cut.Hor, .5)
		cut.Vert = append(cut.Vert, .5)
	case 6:
		cut.Hor = append(cut.Hor, .33, .66)
		cut.Vert = append(cut.Vert, .5)
	case 8:
		cut.Hor = append(cut.Hor, .25, .5, .75)
		cut.Vert = append(cut.Vert, .5)
	case 9:
		cut.Hor = append(cut.Hor, .33, .66)
		cut.Vert = append(cut.Vert, .33, .66)
	case 12:
		cut.Hor = append(cut.Hor, .25, .5, .75)
		cut.Vert = append(cut.Vert, .33, .66)
	case 16:
		cut.Hor = append(cut.Hor, .25, .5, .75)
		cut.Vert = append(cut.Vert, .25, .5, .75)
	default:
		return nil, errors.New("pdfcpu: Invalid ndown value. Must be one of: 2,3,4,6,8,9,12,16")
	}

	return cut, nil
}

// ParseCutConfig parses a Cut command string into an internal structure.
// optionally: horizontalCut, verticalCut, bgcolor, border, margin, origin
func ParseCutConfig(s string, u types.DisplayUnit) (*model.Cut, error) {

	if s == "" {
		return nil, errors.New("pdfcpu: missing cut configuration string")
	}

	cut := &model.Cut{Unit: u}

	ss := strings.Split(s, ",")

	for _, s := range ss {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, errors.New("pdfcpu: Invalid cut configuration string. Please consult pdfcpu help cut")
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := model.CutParamMap.Handle(paramPrefix, paramValueStr, cut); err != nil {
			return nil, err
		}
	}

	if len(cut.Hor) == 0 && len(cut.Vert) == 0 {
		return nil, errors.New("pdfcpu: Invalid cut configuration string: missing hor/ver cutpoints. Please consult pdfcpu help cut")
	}

	return cut, nil
}

func CutPage(ctx *model.Context, i int) (*model.Context, error) {
	return nil, nil
}
