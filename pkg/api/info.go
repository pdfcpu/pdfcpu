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

package api

import (
	"io"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// PDFInfo returns information about rs.
func PDFInfo(rs io.ReadSeeker, fileName string, selectedPages []string, conf *model.Configuration) (*pdfcpu.PDFInfo, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: PDFInfo: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		// Validation loads infodict.
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTINFO

	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return nil, err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, false, true)
	if err != nil {
		return nil, err
	}

	if err := pdfcpu.DetectWatermarks(ctx); err != nil {
		return nil, err
	}

	return pdfcpu.Info(ctx, fileName, pages)
}
