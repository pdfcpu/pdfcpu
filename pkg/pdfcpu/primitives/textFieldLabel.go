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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

type TextFieldLabel struct {
	TextField
	Width    int
	Gap      int    // horizontal space between textfield and label
	Position string `json:"pos"` // relative to textfield
	relPos   pdfcpu.RelPosition
}

func (tfl *TextFieldLabel) validate() error {

	if tfl.Value == "" {
		return errors.New("pdfcpu: missing label value")
	}

	if tfl.Width <= 0 {
		// only for pos left align left or pos right align right!
		return errors.Errorf("pdfcpu: invalid label width: %d", tfl.Width)
	}

	tfl.relPos = pdfcpu.RelPosLeft
	if tfl.Position != "" {
		rp, err := pdfcpu.ParseRelPosition(tfl.Position)
		if err != nil {
			return err
		}
		tfl.relPos = rp
	}

	if tfl.Font != nil {
		tfl.Font.pdf = tfl.pdf
		if err := tfl.Font.validate(); err != nil {
			return err
		}
	}

	if tfl.Border != nil {
		tfl.Border.pdf = tfl.pdf
		if err := tfl.Border.validate(); err != nil {
			return err
		}
	}

	if tfl.BackgroundColor != "" {
		sc, err := tfl.pdf.parseColor(tfl.BackgroundColor)
		if err != nil {
			return err
		}
		tfl.bgCol = sc
	}

	tfl.horAlign = pdfcpu.AlignLeft
	if tfl.Alignment != "" {
		ha, err := pdfcpu.ParseHorAlignment(tfl.Alignment)
		if err != nil {
			return err
		}
		tfl.horAlign = ha
	}

	return nil
}
