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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// TextWatermark returns a text watermark configuration.
func TextWatermark(text, desc string, onTop, update bool) (*pdfcpu.Watermark, error) {
	wm, err := pdfcpu.ParseTextWatermarkDetails(text, desc, onTop)
	if err != nil {
		return nil, err
	}
	wm.Update = update
	return wm, nil
}

// ImageWatermark returns an image watermark configuration.
func ImageWatermark(fileName, desc string, onTop, update bool) (*pdfcpu.Watermark, error) {
	wm, err := pdfcpu.ParseImageWatermarkDetails(fileName, desc, onTop)
	if err != nil {
		return nil, err
	}
	wm.Update = update
	return wm, nil
}

// PDFWatermark returns a PDF watermark configuration.
func PDFWatermark(fileName, desc string, onTop, update bool) (*pdfcpu.Watermark, error) {
	wm, err := pdfcpu.ParsePDFWatermarkDetails(fileName, desc, onTop)
	if err != nil {
		return nil, err
	}
	wm.Update = update
	return wm, nil
}

// AddTextWatermarksFile adds text stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func AddTextWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, text, desc string, conf *pdfcpu.Configuration) error {
	wm, err := TextWatermark(text, desc, onTop, false)
	if err != nil {
		return err
	}
	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// AddImageWatermarksFile adds image stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func AddImageWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *pdfcpu.Configuration) error {
	wm, err := ImageWatermark(fileName, desc, onTop, false)
	if err != nil {
		return err
	}
	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// AddPDFWatermarksFile adds PDF stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func AddPDFWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *pdfcpu.Configuration) error {
	wm, err := PDFWatermark(fileName, desc, onTop, false)
	if err != nil {
		return err
	}
	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// UpdateTextWatermarksFile adds text stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func UpdateTextWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, text, desc string, conf *pdfcpu.Configuration) error {
	wm, err := TextWatermark(text, desc, onTop, true)
	if err != nil {
		return err
	}
	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// UpdateImageWatermarksFile adds image stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func UpdateImageWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *pdfcpu.Configuration) error {
	wm, err := ImageWatermark(fileName, desc, onTop, true)
	if err != nil {
		return err
	}
	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// UpdatePDFWatermarksFile adds PDF stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func UpdatePDFWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *pdfcpu.Configuration) error {
	wm, err := PDFWatermark(fileName, desc, onTop, true)
	if err != nil {
		return err
	}
	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}
