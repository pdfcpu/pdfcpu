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
	"io/ioutil"
)

type runLengthDecode struct {
	baseFilter
}

func (f runLengthDecode) decode(w io.ByteWriter, src []byte) {

	for i := 0; i < len(src); {
		b := src[i]
		if b == 0x80 {
			// eod
			break
		}
		i++
		if b < 0x80 {
			c := int(b) + 1
			for j := 0; j < c; j++ {
				w.WriteByte(src[i])
				i++
			}
			continue
		}
		c := 257 - int(b)
		for j := 0; j < c; j++ {
			w.WriteByte(src[i])
		}
		i++
	}

	return
}

func (f runLengthDecode) encode(w io.ByteWriter, src []byte) {

	const maxLen = 0x80
	const eod = 0x80

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
				w.WriteByte(0x80)
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
				w.WriteByte(0x80)
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

	p, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	f.decode(&b, p)

	return &b, nil
}
