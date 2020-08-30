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
	"encoding/ascii85"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
)

type ascii85Decode struct {
	baseFilter
}

const eodASCII85 = "~>"

// Encode implements encoding for an ASCII85Decode filter.
func (f ascii85Decode) Encode(r io.Reader) (io.Reader, error) {

	p, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	encoder := ascii85.NewEncoder(buf)
	encoder.Write(p)
	encoder.Close()

	// Add eod sequence
	buf.WriteString(eodASCII85)

	return buf, nil
}

// Decode implements decoding for an ASCII85Decode filter.
func (f ascii85Decode) Decode(r io.Reader) (io.Reader, error) {

	p, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if !bytes.HasSuffix(p, []byte(eodASCII85)) {
		return nil, errors.New("pdfcpu: Decode: missing eod marker")
	}

	// Strip eod sequence: "~>"
	p = p[:len(p)-2]

	decoder := ascii85.NewDecoder(bytes.NewReader(p))

	buf, err := ioutil.ReadAll(decoder)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(buf), nil
}
