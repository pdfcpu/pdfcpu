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

type Regions struct {
	page        *PDFPage
	parent      *Content
	Name        string // unique
	Orientation string `json:"orient"`
	horizontal  bool
	Divider     *Divider `json:"div"`
	Left, Right *Content // 2 horizontal regions or
	Top, Bottom *Content // 2 vertical regions
	mediaBox    *pdfcpu.Rectangle
}

func (r *Regions) validate() error {

	pdf := r.page.pdf

	// trim json string necessary?
	if r.Orientation == "" {
		return errors.Errorf("pdfcpu: region is missing orientation")
	}
	o, err := pdfcpu.ParseRegionOrientation(r.Orientation)
	if err != nil {
		return err
	}
	r.horizontal = o == pdfcpu.Horizontal

	if r.Divider == nil {
		return errors.New("pdfcpu: region is missing divider")
	}
	r.Divider.pdf = pdf
	if err := r.Divider.validate(); err != nil {
		return err
	}

	if r.horizontal {
		if r.Left == nil {
			return errors.Errorf("pdfcpu: regions %s is missing Left", r.Name)
		}
		r.Left.page = r.page
		r.Left.parent = r.parent
		if err := r.Left.validate(); err != nil {
			return err
		}
		if r.Right == nil {
			return errors.Errorf("pdfcpu: regions %s is missing Right", r.Name)
		}
		r.Right.page = r.page
		r.Right.parent = r.parent
		return r.Right.validate()
	}

	if r.Top == nil {
		return errors.Errorf("pdfcpu: regions %s is missing Top", r.Name)
	}
	r.Top.page = r.page
	r.Top.parent = r.parent
	if err := r.Top.validate(); err != nil {
		return err
	}
	if r.Bottom == nil {
		return errors.Errorf("pdfcpu: regions %s is missing Bottom", r.Name)
	}
	r.Bottom.page = r.page
	r.Bottom.parent = r.parent
	if err := r.Bottom.validate(); err != nil {
		return err
	}

	return nil
}

func (r *Regions) render(p *pdfcpu.Page, pageNr int, fonts pdfcpu.FontMap, images pdfcpu.ImageMap, fields *pdfcpu.Array) error {

	if r.horizontal {

		// Calc divider.
		dx := r.mediaBox.Width() * r.Divider.At
		r.Divider.p.X, r.Divider.p.Y = pdfcpu.NormalizeCoord(dx, 0, r.mediaBox, r.page.pdf.origin, true)
		r.Divider.q.X, r.Divider.q.Y = pdfcpu.NormalizeCoord(dx, r.mediaBox.Height(), r.mediaBox, r.page.pdf.origin, true)

		// Render left region.
		r.Left.mediaBox = r.mediaBox.CroppedCopy(0)
		r.Left.mediaBox.UR.X = r.Divider.p.X - float64(r.Divider.Width)/2
		r.Left.page = r.page
		if err := r.Left.render(p, pageNr, fonts, images, fields); err != nil {
			return err
		}

		// Render right region.
		r.Right.mediaBox = r.mediaBox.CroppedCopy(0)
		r.Right.mediaBox.LL.X = r.Divider.p.X + float64(r.Divider.Width)/2
		r.Right.page = r.page
		if err := r.Right.render(p, pageNr, fonts, images, fields); err != nil {
			return err
		}

	} else {

		// Calc divider.
		dy := r.mediaBox.Height() * r.Divider.At
		r.Divider.p.X, r.Divider.p.Y = pdfcpu.NormalizeCoord(0, dy, r.mediaBox, r.page.pdf.origin, true)
		r.Divider.q.X, r.Divider.q.Y = pdfcpu.NormalizeCoord(r.mediaBox.Width(), dy, r.mediaBox, r.page.pdf.origin, true)

		// Render top region.
		r.Top.mediaBox = r.mediaBox.CroppedCopy(0)
		r.Top.mediaBox.LL.Y = r.Divider.p.Y + float64(r.Divider.Width)/2
		r.Top.page = r.page
		if err := r.Top.render(p, pageNr, fonts, images, fields); err != nil {
			return err
		}

		// Render bottom region.
		r.Bottom.mediaBox = r.mediaBox.CroppedCopy(0)
		r.Bottom.mediaBox.UR.Y = r.Divider.p.Y - float64(r.Divider.Width)/2
		r.Bottom.page = r.page
		if err := r.Bottom.render(p, pageNr, fonts, images, fields); err != nil {
			return err
		}

	}

	return r.Divider.render(p)
}
