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

	"github.com/hhrutter/lzw"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

type lzwDecode struct {
	baseFilter
}

// Encode implements encoding for an LZWDecode filter.
func (f lzwDecode) Encode(r io.Reader) (io.Reader, error) {

	log.Trace.Println("EncodeLZW begin")

	var b bytes.Buffer

	ec, ok := f.parms["EarlyChange"]
	if !ok {
		ec = 1
	}

	wc := lzw.NewWriter(&b, ec == 1)
	defer wc.Close()

	written, err := io.Copy(wc, r)
	if err != nil {
		return nil, err
	}
	log.Trace.Printf("EncodeLZW end: %d bytes written\n", written)

	return &b, nil
}

// Decode implements decoding for an LZWDecode filter.
func (f lzwDecode) Decode(r io.Reader) (io.Reader, error) {

	log.Trace.Println("DecodeLZW begin")

	p, found := f.parms["Predictor"]
	if found && p > 1 {
		return nil, errors.Errorf("DecodeLZW: unsupported predictor %d", p)
	}

	ec, ok := f.parms["EarlyChange"]
	if !ok {
		ec = 1
	}

	rc := lzw.NewReader(r, ec == 1)
	defer rc.Close()

	var b bytes.Buffer
	written, err := io.Copy(&b, rc)
	if err != nil {
		return nil, err
	}
	log.Trace.Printf("DecodeLZW: decoded %d bytes.\n", written)

	return &b, nil
}
