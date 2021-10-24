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

import "github.com/pkg/errors"

type Padding struct {
	Name                     string
	Width                    float64
	Top, Right, Bottom, Left float64
}

func (p *Padding) validate() error {

	if p.Name == "$" {
		return errors.New("pdfcpu: invalid padding reference $")
	}

	if p.Width < 0 {
		if p.Top > 0 || p.Right > 0 || p.Bottom > 0 || p.Left > 0 {
			return errors.Errorf("pdfcpu: invalid padding width: %f", p.Width)
		}
	}

	if p.Width > 0 {
		p.Top, p.Right, p.Bottom, p.Left = p.Width, p.Width, p.Width, p.Width
		return nil
	}

	return nil
}

func (p *Padding) mergeIn(p0 *Padding) {
	if p.Width > 0 {
		return
	}
	if p.Width < 0 {
		p.Top, p.Right, p.Bottom, p.Left = 0, 0, 0, 0
		return
	}

	if p.Top == 0 {
		p.Top = p0.Top
	} else if p.Top < 0 {
		p.Top = 0.
	}

	if p.Right == 0 {
		p.Right = p0.Right
	} else if p.Right < 0 {
		p.Right = 0.
	}

	if p.Bottom == 0 {
		p.Bottom = p0.Bottom
	} else if p.Bottom < 0 {
		p.Bottom = 0.
	}

	if p.Left == 0 {
		p.Left = p0.Left
	} else if p.Left < 0 {
		p.Left = 0.
	}
}
