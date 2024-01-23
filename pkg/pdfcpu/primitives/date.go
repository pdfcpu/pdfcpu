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

package primitives

import (
	"strings"
	"time"

	"github.com/pkg/errors"
)

// DateFormat represents a supported date format.
// It consists of an internal and an external form.
type DateFormat struct {
	Int string
	Ext string
}

var dateFormats = []DateFormat{

	// based on separator '-'
	{"2006-1-2", "yyyy-m-d"},
	{"2006-2-1", "yyyy-d-m"},
	{"2006-01-02", "yyyy-mm-dd"},
	{"2006-02-01", "yyyy-dd-mm"},
	{"02-01-2006", "dd-mm-yyyy"},
	{"01-02-2006", "mm-dd-yyyy"},
	{"2-1-2006", "d-m-yyyy"},
	{"1-2-2006", "m-d-yyyy"},

	// based on separator '/'
	{"2006/1/2", "yyyy/m/d"},
	{"2006/2/1", "yyyy/d/m"},
	{"2006/01/02", "yyyy/mm/dd"},
	{"2006/02/01", "yyyy/dd/mm"},
	{"02/01/2006", "dd/mm/yyyy"},
	{"01/02/2006", "mm/dd/yyyy"},
	{"2/1/2006", "d/m/yyyy"},
	{"1/2/2006", "m/d/yyyy"},

	// based on separator '.'
	{"2006.1.2", "yyyy.m.d"},
	{"2006.2.1", "yyyy.d.m"},
	{"2006.01.02", "yyyy.mm.dd"},
	{"2006.02.01", "yyyy.dd.mm"},
	{"02.01.2006", "dd.mm.yyyy"},
	{"01.02.2006", "mm.dd.yyyy"},
	{"2.1.2006", "d.m.yyyy"},
	{"1.2.2006", "m.d.yyyy"},
}

// DateFormatForFmtInt returns the date format for an internal format string.
func DateFormatForFmtInt(fmtInt string) (*DateFormat, error) {
	for _, df := range dateFormats {
		if df.Int == fmtInt {
			return &df, nil
		}
	}
	return nil, errors.Errorf("pdfcpu: \"%s\": unknown internal date format", fmtInt)
}

// DateFormatForFmtInt returns the date format for an external format string.
func DateFormatForFmtExt(fmtExt string) (*DateFormat, error) {
	s := strings.ToLower(fmtExt)
	for _, df := range dateFormats {
		if df.Ext == s {
			return &df, nil
		}
	}
	return nil, errors.Errorf("pdfcpu: \"%s\": unknown external date format", fmtExt)
}

// DateFormatForDate returns the date format for given date string.
func DateFormatForDate(date string) (*DateFormat, error) {
	for _, df := range dateFormats {
		if _, err := time.Parse(df.Int, date); err == nil {
			return &df, nil
		}
	}
	return nil, errors.Errorf("pdfcpu: \"%s\": using unknown date format", date)
}

func (df DateFormat) validate(date string) error {
	_, err := time.Parse(df.Int, date)
	return err
}
