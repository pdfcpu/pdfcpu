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

package filter

import (
	"bytes"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
	"golang.org/x/image/ccitt"
)

type ccittDecode struct {
	baseFilter
}

// Encode implements encoding for a CCITTDecode filter.
func (f ccittDecode) Encode(r io.Reader) (io.Reader, error) {
	// TODO
	return nil, nil
}

// Decode implements decoding for a CCITTDecode filter.
func (f ccittDecode) Decode(r io.Reader) (io.Reader, error) {

	log.Trace.Println("DecodeCCITT begin")

	var ok bool

	// <0 : Pure two-dimensional encoding (Group 4)
	// =0 : Pure one-dimensional encoding (Group 3, 1-D)
	// >0 : Mixed one- and two-dimensional encoding (Group 3, 2-D)
	k := 0
	k, ok = f.parms["K"]
	if ok && k > 0 {
		return nil, errors.New("pdfcpu: filter CCITTFax k > 0 currently unsupported")
	}

	cols := 1728
	col, ok := f.parms["Columns"]
	if ok {
		cols = col
	}

	rows, ok := f.parms["Rows"]
	if !ok {
		return nil, errors.New("pdfcpu: ccitt: missing DecodeParam \"Rows\"")
	}

	blackIs1 := false
	v, ok := f.parms["BlackIs1"]
	if ok && v == 1 {
		blackIs1 = true
	}

	encodedByteAlign := false
	v, ok = f.parms["EncodedByteAlign"]
	if ok && v == 1 {
		encodedByteAlign = true
	}

	opts := &ccitt.Options{Invert: blackIs1, Align: encodedByteAlign}

	mode := ccitt.Group3
	if k < 0 {
		mode = ccitt.Group4
	}
	rd := ccitt.NewReader(r, ccitt.MSB, mode, cols, rows, opts)

	var b bytes.Buffer
	written, err := io.Copy(&b, rd)
	if err != nil {
		return nil, err
	}
	log.Trace.Printf("DecodeCCITT: decoded %d bytes.\n", written)

	return &b, nil
}
