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

	"github.com/hhrutter/pdfcpu/ccitt"
	"github.com/hhrutter/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

type ccittDecode struct {
	baseFilter
}

// Encode implements encoding for an CCITTDecode filter.
func (f ccittDecode) Encode(r io.Reader) (*bytes.Buffer, error) {
	// TODO
	return nil, nil
}

// Decode implements decoding for a CCITTDecode filter.
func (f ccittDecode) Decode(r io.Reader) (*bytes.Buffer, error) {

	log.Trace.Println("DecodeCCITT begin")

	var ok bool

	// <0 : Pure two-dimensional encoding (Group 4)
	// =0 : Pure one-dimensional encoding (Group 3, 1-D)
	// >0 : Mixed one- and two-dimensional encoding (Group 3, 2-D)
	k := 0
	k, ok = f.parms["K"]
	if ok && k > 0 {
		return nil, errors.New("DecodeCCITT: K > 0 currently unsupported")
	}

	columns := 1728
	col, ok := f.parms["Columns"]
	if ok {
		columns = col
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

	mode := ccitt.Group3
	if k < 0 {
		mode = ccitt.Group4
	}
	rc := ccitt.NewReader(r, mode, columns, blackIs1, encodedByteAlign, false)
	defer rc.Close()

	var b bytes.Buffer
	written, err := io.Copy(&b, rc)
	if err != nil {
		return nil, err
	}
	log.Trace.Printf("DecodeCCITT: decoded %d bytes.\n", written)

	return &b, nil
}
