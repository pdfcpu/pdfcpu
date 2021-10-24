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

type Margin struct {
	Name                     string
	Width                    float64
	Top, Right, Bottom, Left float64
}

func (m *Margin) validate() error {

	if m.Name == "$" {
		return errors.New("pdfcpu: invalid margin reference $")
	}

	if m.Width < 0 {
		if m.Top > 0 || m.Right > 0 || m.Bottom > 0 || m.Left > 0 {
			return errors.Errorf("pdfcpu: individual margins not allowed for width: %f", m.Width)
		}
	}

	if m.Width > 0 {
		m.Top, m.Right, m.Bottom, m.Left = m.Width, m.Width, m.Width, m.Width
		return nil
	}

	return nil
}

func (m *Margin) mergeIn(m0 *Margin) {
	if m.Width > 0 {
		return
	}
	if m.Width < 0 {
		m.Top, m.Right, m.Bottom, m.Left = 0, 0, 0, 0
		return
	}

	if m.Top == 0 {
		m.Top = m0.Top
	} else if m.Top < 0 {
		m.Top = 0.
	}

	if m.Right == 0 {
		m.Right = m0.Right
	} else if m.Right < 0 {
		m.Right = 0.
	}

	if m.Bottom == 0 {
		m.Bottom = m0.Bottom
	} else if m.Bottom < 0 {
		m.Bottom = 0.
	}

	if m.Left == 0 {
		m.Left = m0.Left
	} else if m.Left < 0 {
		m.Left = 0.
	}
}
