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

import (
	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

type FormFont struct {
	pdf   *PDF
	Name  string
	Size  int
	Color string `json:"col"`
	col   *pdfcpu.SimpleColor
}

func (f *FormFont) validate() error {

	if f.Name == "$" {
		return errors.New("pdfcpu: invalid font reference $")
	}

	if f.Name != "" && f.Name[0] != '$' {
		if !font.SupportedFont(f.Name) {
			return errors.Errorf("pdfcpu: font %s is unsupported, please refer to \"pdfcpu fonts list\".\n", f.Name)
		}
		if f.Size <= 0 {
			return errors.Errorf("pdfcpu: invalid font size: %d", f.Size)
		}
	}

	if f.Color != "" {
		sc, err := f.pdf.parseColor(f.Color)
		if err != nil {
			return err
		}
		f.col = sc
	}

	return nil
}

func (f *FormFont) mergeIn(f0 *FormFont) {
	if f.Name == "" {
		f.Name = f0.Name
	}
	if f.Size == 0 {
		f.Size = f0.Size
	}
	if f.col == nil {
		f.col = f0.col
	}
}
