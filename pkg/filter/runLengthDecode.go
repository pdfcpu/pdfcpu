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
	"bufio"
	"bytes"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
)

func unexpectedEOF(err error) error {
	if err == io.EOF {
		return errors.New("missing EOD marker in encoded stream")
	}
	return err
}

type runLengthDecode struct {
	baseFilter
}

const eodRunLength = 0x80

func (f runLengthDecode) decode(w io.ByteWriter, src io.ByteReader) error {

	for b, err := src.ReadByte(); ; b, err = src.ReadByte() {
		// EOF is an error since we expect the EOD marker
		if err != nil {
			return unexpectedEOF(err)
		}
		if b == eodRunLength { // eod
			return nil
		}
		if b < 0x80 {
			c := int(b) + 1
			for j := 0; j < c; j++ {
				nextChar, err := src.ReadByte()
				if err != nil {
					return unexpectedEOF(err) // EOF here is an error
				}
				w.WriteByte(nextChar)
			}
			continue
		}
		c := 257 - int(b)
		nextChar, err := src.ReadByte()
		if err != nil {
			return unexpectedEOF(err) // EOF here is an error
		}
		for j := 0; j < c; j++ {
			w.WriteByte(nextChar)
		}
	}
}

func (f runLengthDecode) encode(w io.ByteWriter, src []byte) {

	const maxLen = 0x80

	i := 0
	b := src[i]
	start := i

	for {

		// Detect constant run eg. 0x1414141414141414
		for i < len(src) && src[i] == b && (i-start < maxLen) {
			i++
		}
		c := i - start
		if c > 1 {
			// Write constant run with length=c
			w.WriteByte(byte(257 - c))
			w.WriteByte(b)
			if i == len(src) {
				w.WriteByte(eodRunLength)
				return
			}
			b = src[i]
			start = i
			continue
		}

		// Detect variable run eg. 0x20FFD023335BCC12
		for i < len(src) && src[i] != b && (i-start < maxLen) {
			b = src[i]
			i++
		}
		if i == len(src) || i-start == maxLen {
			c = i - start
			w.WriteByte(byte(c - 1))
			for j := 0; j < c; j++ {
				w.WriteByte(src[start+j])
			}
			if i == len(src) {
				w.WriteByte(eodRunLength)
				return
			}
		} else {
			c = i - 1 - start
			// Write variable run with length=c
			w.WriteByte(byte(c - 1))
			for j := 0; j < c; j++ {
				w.WriteByte(src[start+j])
			}
			i--
		}
		b = src[i]
		start = i
	}

}

// Encode implements encoding for a RunLengthDecode filter.
func (f runLengthDecode) Encode(r io.Reader) (io.Reader, error) {

	p, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	f.encode(&b, p)

	return &b, nil
}

// Decode implements decoding for an RunLengthDecode filter.
func (f runLengthDecode) Decode(r io.Reader) (io.Reader, error) {
	var b bytes.Buffer

	// when possible, we make sure not to read passed EOD
	byteReader, ok := r.(io.ByteReader)
	if !ok {
		byteReader = bufio.NewReader(r)
	}

	err := f.decode(&b, byteReader)
	return &b, err
}
