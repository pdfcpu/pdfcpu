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

package color

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// Some popular colors.
var (
	Black     = SimpleColor{}
	White     = SimpleColor{R: 1, G: 1, B: 1}
	LightGray = SimpleColor{.9, .9, .9}
	Gray      = SimpleColor{.5, .5, .5}
	DarkGray  = SimpleColor{.3, .3, .3}
	Red       = SimpleColor{1, 0, 0}
	Green     = SimpleColor{0, 1, 0}
	Blue      = SimpleColor{0, 0, 1}
)

var ErrInvalidColor = errors.New("pdfcpu: invalid color constant")

// SimpleColor is a simple rgb wrapper.
type SimpleColor struct {
	R, G, B float32 // intensities between 0 and 1.
}

func (sc SimpleColor) String() string {
	return fmt.Sprintf("r=%1.1f g=%1.1f b=%1.1f", sc.R, sc.G, sc.B)
}

func (sc SimpleColor) Array() types.Array {
	return types.NewNumberArray(float64(sc.R), float64(sc.G), float64(sc.B))
}

// NewSimpleColor returns a SimpleColor for rgb in the form 0x00RRGGBB
func NewSimpleColor(rgb uint32) SimpleColor {
	r := float32((rgb>>16)&0xFF) / 255
	g := float32((rgb>>8)&0xFF) / 255
	b := float32(rgb&0xFF) / 255
	return SimpleColor{r, g, b}
}

// NewSimpleColorForArray returns a SimpleColor for an r,g,b array.
func NewSimpleColorForArray(arr types.Array) SimpleColor {
	var r, g, b float32

	if f, ok := arr[0].(types.Float); ok {
		r = float32(f.Value())
	} else {
		r = float32(arr[0].(types.Integer))
	}

	if f, ok := arr[1].(types.Float); ok {
		g = float32(f.Value())
	} else {
		g = float32(arr[1].(types.Integer))
	}

	if f, ok := arr[2].(types.Float); ok {
		b = float32(f.Value())
	} else {
		b = float32(arr[2].(types.Integer))
	}

	return SimpleColor{r, g, b}
}

// NewSimpleColorForHexCode returns a SimpleColor for a #FFFFFF type hexadecimal rgb color representation.
func NewSimpleColorForHexCode(hexCol string) (SimpleColor, error) {
	var sc SimpleColor
	if len(hexCol) != 7 || hexCol[0] != '#' {
		return sc, errors.Errorf("pdfcpu: invalid hex color string: #FFFFFF, %s\n", hexCol)
	}
	b, err := hex.DecodeString(hexCol[1:])
	if err != nil || len(b) != 3 {
		return sc, errors.Errorf("pdfcpu: invalid hex color string: #FFFFFF, %s\n", hexCol)
	}
	return SimpleColor{float32(b[0]) / 255, float32(b[1]) / 255, float32(b[2]) / 255}, nil
}

func internalSimpleColor(s string) (SimpleColor, error) {
	var (
		sc  SimpleColor
		err error
	)
	switch strings.ToLower(s) {
	case "black":
		sc = Black
	case "darkgray":
		sc = DarkGray
	case "gray":
		sc = Gray
	case "lightgray":
		sc = LightGray
	case "white":
		sc = White
	case "red":
		sc = Red
	case "green":
		sc = Green
	case "blue":
		sc = Blue
	default:
		err = ErrInvalidColor
	}
	return sc, err
}

// ParseColor turns a color string into a SimpleColor.
func ParseColor(s string) (SimpleColor, error) {
	var sc SimpleColor

	cs := strings.Split(s, " ")
	if len(cs) != 1 && len(cs) != 3 {
		return sc, errors.Errorf("pdfcpu: illegal color string: 3 intensities 0.0 <= i <= 1.0 or #FFFFFF, %s\n", s)
	}

	if len(cs) == 1 {
		if len(cs[0]) == 7 && cs[0][0] == '#' {
			// #FFFFFF to uint32
			return NewSimpleColorForHexCode(cs[0])
		}
		return internalSimpleColor(cs[0])
	}

	r, err := strconv.ParseFloat(cs[0], 32)
	if err != nil {
		return sc, errors.Errorf("red must be a float value: %s\n", cs[0])
	}
	if r < 0 || r > 1 {
		return sc, errors.New("pdfcpu: red: a color value is an intensity between 0.0 and 1.0")
	}
	sc.R = float32(r)

	g, err := strconv.ParseFloat(cs[1], 32)
	if err != nil {
		return sc, errors.Errorf("pdfcpu: green must be a float value: %s\n", cs[1])
	}
	if g < 0 || g > 1 {
		return sc, errors.New("pdfcpu: green: a color value is an intensity between 0.0 and 1.0")
	}
	sc.G = float32(g)

	b, err := strconv.ParseFloat(cs[2], 32)
	if err != nil {
		return sc, errors.Errorf("pdfcpu: blue must be a float value: %s\n", cs[2])
	}
	if b < 0 || b > 1 {
		return sc, errors.New("pdfcpu: blue: a color value is an intensity between 0.0 and 1.0")
	}
	sc.B = float32(b)

	return sc, nil
}
