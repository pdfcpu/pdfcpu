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
	"encoding/hex"
	"errors"
	"io"
)

type asciiHexDecode struct {
	baseFilter
}

const eodHexDecode = '>'

// Encode implements encoding for an ASCIIHexDecode filter.
func (f asciiHexDecode) Encode(r io.Reader) (io.Reader, error) {
	bb, err := getReaderBytes(r)
	if err != nil {
		return nil, err
	}

	if len(bb) > maxInt/2 {
		return nil, errors.New("ASCIIHex encode: length overflow")
	}

	dst := make([]byte, hex.EncodedLen(len(bb)))
	hex.Encode(dst, bb)

	// eod marker
	dst = append(dst, eodHexDecode)

	return bytes.NewBuffer(dst), nil
}

// Decode implements decoding for an ASCIIHexDecode filter.
func (f asciiHexDecode) Decode(r io.Reader) (io.Reader, error) {
	return f.DecodeLength(r, -1)
}

// DecodeLength implements decoding for an ASCIIHexDecode filter with a maximum output length.
func (f asciiHexDecode) DecodeLength(r io.Reader, maxLen int64) (io.Reader, error) {
	bb, err := getReaderBytes(r)
	if err != nil {
		return nil, err
	}

	var p []byte

	// Remove any white space and cut off on eod
	for i := range bb {
		if bb[i] == eodHexDecode {
			break
		}
		if !bytes.ContainsRune([]byte{0x09, 0x0A, 0x0C, 0x0D, 0x20}, rune(bb[i])) {
			p = append(p, bb[i])
		}
	}

	// if len == odd add "0"
	if len(p)%2 == 1 {
		p = append(p, '0')
	}

	decodedLen := int64(hex.DecodedLen(len(p)))
	if maxLen < 0 {
		maxLen = decodedLen
		if limit := f.decodeLimit(-1); limit >= 0 && maxLen > limit {
			return nil, ErrDecodeLimitExceeded
		}
	} else if maxLen > decodedLen {
		return nil, io.ErrUnexpectedEOF
	}

	if maxLen > int64(maxInt) || maxLen > maxInt64/2 {
		return nil, errors.New("ASCIIHex decode: length overflow")
	}
	dst := make([]byte, maxLen)

	if _, err := hex.Decode(dst, p[:maxLen*2]); err != nil {
		return nil, err
	}

	return bytes.NewBuffer(dst), nil
}
