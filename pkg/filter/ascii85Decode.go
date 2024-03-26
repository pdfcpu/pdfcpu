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

	"github.com/pkg/errors"
)

type ascii85Decode struct {
	baseFilter
}

const eodASCII85 = "~>"

// Encode implements encoding for an ASCII85Decode filter.
func (f ascii85Decode) Encode(r io.Reader) (io.Reader, error) {

	b2 := &bytes.Buffer{}
	encoder := ascii85.NewEncoder(b2)
	if _, err := io.Copy(encoder, r); err != nil {
		return nil, err
	}
	encoder.Close()

	// Add eod sequence
	b2.WriteString(eodASCII85)

	return b2, nil
}

// Decode implements decoding for an ASCII85Decode filter.
func (f ascii85Decode) Decode(r io.Reader) (io.Reader, error) {
	return f.DecodeLength(r, -1)
}

func (f ascii85Decode) DecodeLength(r io.Reader, maxLen int64) (io.Reader, error) {

	bb, err := getReaderBytes(r)
	if err != nil {
		return nil, err
	}

	// fmt.Printf("dump:\n%s", hex.Dump(bb))

	l := len(bb)
	if bb[l-1] == 0x0A || bb[l-1] == 0x0D {
		bb = bb[:l-1]
	}

	if !bytes.HasSuffix(bb, []byte(eodASCII85)) {
		return nil, errors.New("pdfcpu: Decode: missing eod marker")
	}

	// Strip eod sequence: "~>"
	bb = bb[:len(bb)-2]

	decoder := ascii85.NewDecoder(bytes.NewReader(bb))

	var b2 bytes.Buffer
	if maxLen < 0 {
		if _, err := io.Copy(&b2, decoder); err != nil {
			return nil, err
		}
	} else {
		if _, err := io.CopyN(&b2, decoder, maxLen); err != nil {
			return nil, err
		}
	}

	return &b2, nil
}
