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

package pdfcpu

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var errInvalidNUpConfig = errors.New("Invalid nup configuration string. Please consult pdfcpu help nup")

type orientation int

func (o orientation) String() string {

	switch o {

	case RightDown:
		return "right down"

	case DownRight:
		return "down right"

	case LeftDown:
		return "left down"

	case DownLeft:
		return "down left"

	}

	return ""
}

// These are the defined anchors for relative positioning.
const (
	RightDown orientation = iota
	DownRight
	LeftDown
	DownLeft
)

func parseOrientation(s string) (orientation, error) {

	switch s {
	case "rd":
		return RightDown, nil
	case "dr":
		return DownRight, nil
	case "ld":
		return LeftDown, nil
	case "dl":
		return DownLeft, nil
	}

	return 0, errors.Errorf("unknown nUp orientation: %s", s)
}

// NUp represents the command details for the command "NUp".
type NUp struct {
	pageDim          dim         // page dimensions in user units.
	pageSize         string      // one of A0,A1,A2,A3,A4(=default),A5,A6,A7,A8,Letter,Legal,Ledger,Tabloid,Executive,ANSIC,ANSID,ANSIE.
	orient           orientation // one of rd(=default),dr,ld,dl
	grid             dim         // grid dimensions eg (2,2)
	preservePageSize bool        // for PDF inputfiles only
	ImgInputFile     bool
}

// DefaultNUpConfig returns the default configuration.
func DefaultNUpConfig() *NUp {
	return &NUp{
		pageDim:  PaperSize["A4"], // for image input file
		pageSize: "A4",            // for image input file
		orient:   RightDown,       // for pdf input file
	}
}

// ParseNUpGridDefinition parses NUp grid dimensions into an internal structure.
func ParseNUpGridDefinition(s string, nUp *NUp) error {

	n, err := strconv.Atoi(s)
	if err != nil || !IntMemberOf(n, []int{2, 4, 9, 16}) {
		if nUp.ImgInputFile {
			return errors.New("grid definition: one of 2, 4, 9 or 16")
		}
		ss := strings.Split(s, "x")
		if len(ss) != 2 {
			return errors.New("grid definition: one of 2, 4, 9, 16 or mxn")
		}
		m, err := strconv.Atoi(ss[0])
		if err != nil {
			return errors.New("grid definition: one of 2, 4, 9, 16 or mxn")
		}
		n, err := strconv.Atoi(ss[1])
		if err != nil {
			return errors.New("grid definition: one of 2, 4, 9, 16 or mxn")
		}
		nUp.grid = dim{m, n}
		return nil
	}

	var d dim
	switch n {
	case 2:
		d = dim{1, 2}
	case 4:
		d = dim{2, 2}
	case 9:
		d = dim{3, 3}
	case 16:
		d = dim{4, 4}
	}
	nUp.grid = d
	nUp.preservePageSize = true

	return nil
}

// ParseNUpDetails parses a NUp command string into an internal structure.
func ParseNUpDetails(s string, nup *NUp) error {

	if s == "" {
		return nil
	}

	ss := strings.Split(s, ",")

	var setDim, setFormat bool

	for _, s := range ss {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return errInvalidNUpConfig
		}

		k := strings.TrimSpace(ss1[0])
		v := strings.TrimSpace(ss1[1])
		//fmt.Printf("key:<%s> value<%s>\n", k, v)

		var err error

		switch k {
		case "f": // page format
			nup.pageDim, nup.pageSize, err = parsePageFormat(v, setDim)
			setFormat = true

		case "d": // page dimensions
			nup.pageDim, nup.pageSize, err = parsePageDim(v, setFormat)
			setDim = true

		case "o": // offset
			nup.orient, err = parseOrientation(v)

		default:
			err = errInvalidNUpConfig
		}

		if err != nil {
			return err
		}
	}

	return nil
}
