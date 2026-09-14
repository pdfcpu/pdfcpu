// Copyright 2011 The Go Authors. All rights reserved.
// Copyright 2026 The pdfcpu Authors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lzw is an enhanced version of compress/lzw.
//
// It implements Adobe's PDF lzw compression as defined for the LZWDecode filter
// and is also compatible with the TIFF file format.
//
// See the golang proposal: https://github.com/golang/go/issues/25409.

package lzw

import (
	"bufio"
	"errors"
	"io"
)

const (
	maxWidth           = 12
	decoderInvalidCode = 0xffff
	flushBuffer        = 1 << maxWidth
)

// decoder is the state from which the readXxx method converts a byte
// stream into a code stream.
type decoder struct {
	r        io.ByteReader
	bits     uint32
	nBits    uint
	width    uint
	read     func(*decoder) (uint16, error) // readMSB always for PDF and TIFF
	litWidth uint                           // width in bits of literal codes
	err      error

	// The first 1<<litWidth codes are literal codes.
	// The next two codes mean clear and EOF.
	// Dictionary entries start at clear + 2. After the first literal, hi is the next free index,
	// capped at len(prefix) when the dictionary is full.
	// overflow is the next code-width boundary; oneOff makes width changes occur one code early.
	// last is the most recently seen code, or decoderInvalidCode after initialization or clear.
	clear, eof, hi, overflow, last uint16

	// Each stored dictionary code c in [clear + 2, hi) expands to two or more bytes:
	//   suffix[c] is the last of these bytes.
	//   prefix[c] is the code for all but the last byte.
	//   This code can either be a literal code or another code in [clear + 2, c).
	// When hi is below capacity, c == hi expands the previous code followed by its first byte.
	suffix [1 << maxWidth]uint8
	prefix [1 << maxWidth]uint16

	// output is the temporary output buffer.
	// Literal codes are accumulated from the start of the buffer.
	// Non-literal codes decode to a sequence of suffixes that are first
	// written right-to-left from the end of the buffer before being copied
	// to the start of the buffer.
	// It is flushed when it contains >= 1<<maxWidth bytes,
	// so that there is always room to decode an entire code.
	output [2 * 1 << maxWidth]byte
	o      int    // write index into output
	toRead []byte // bytes to return from Read
	// oneOff makes code length increases occur one code early.
	oneOff bool
}

// readMSB returns the next code for "Most Significant Bits first" data.
func (d *decoder) readMSB() (uint16, error) {
	for d.nBits < d.width {
		x, err := d.r.ReadByte()
		if err != nil {
			return 0, err
		}
		d.bits |= uint32(x) << (24 - d.nBits)
		d.nBits += 8
	}
	code := uint16(d.bits >> (32 - d.width))
	d.bits <<= d.width
	d.nBits -= d.width
	return code, nil
}

// Read decompresses data into b.
func (d *decoder) Read(b []byte) (int, error) {
	for {
		if len(d.toRead) > 0 {
			n := copy(b, d.toRead)
			d.toRead = d.toRead[n:]
			return n, nil
		}
		if d.err != nil {
			return 0, d.err
		}
		d.decode()
	}
}

func (d *decoder) advanceCode() {
	if d.hi < uint16(len(d.prefix)) {
		d.hi++
	}
	ui := d.hi
	if d.oneOff {
		ui++
	}
	if ui >= d.overflow && d.width < maxWidth {
		d.width++
		d.overflow <<= 1
	}
}

func (d *decoder) saveEntry(head uint8) {
	if d.last == decoderInvalidCode || d.hi >= uint16(len(d.prefix)) {
		return
	}
	d.suffix[d.hi] = head
	d.prefix[d.hi] = d.last
}

// decode decompresses bytes from r and leaves them in d.toRead.
// read specifies how to decode bytes into codes.
// litWidth is the width in bits of literal codes.
func (d *decoder) decode() {
	// Loop over the code stream, converting codes into decompressed bytes.
loop:
	for {
		code, err := d.read(d)
		if err != nil {
			// Some PDF Writers write an EOD some don't.
			// Don't insist on EOD marker.
			// Don't return an unexpected EOF error.
			d.err = err
			break
		}
		switch {
		case code < d.clear:
			// We have a literal code.
			d.output[d.o] = uint8(code)
			d.o++
			d.saveEntry(uint8(code))
		case code == d.clear:
			d.width = 1 + d.litWidth
			d.hi = d.eof
			d.overflow = 1 << d.width
			d.last = decoderInvalidCode
			continue
		case code == d.eof:
			d.err = io.EOF
			break loop
		case code <= d.hi:
			c, i := code, len(d.output)-1
			if code == d.hi && d.last != decoderInvalidCode {
				// code == hi is a special case which expands to the last expansion
				// followed by the head of the last expansion. To find the head, we walk
				// the prefix chain until we find a literal code.
				c = d.last
				for c >= d.clear {
					c = d.prefix[c]
				}
				d.output[i] = uint8(c)
				i--
				c = d.last
			}
			// Copy the suffix chain into output and then write that to w.
			for c >= d.clear {
				d.output[i] = d.suffix[c]
				i--
				c = d.prefix[c]
			}
			d.output[i] = uint8(c)
			d.o += copy(d.output[d.o:], d.output[i:])
			d.saveEntry(uint8(c))
		default:
			d.err = errors.New("lzw: invalid code")
			break loop
		}
		d.last = code
		d.advanceCode()
		if d.o >= flushBuffer {
			break
		}
	}
	// Flush pending output.
	d.toRead = d.output[:d.o]
	d.o = 0
}

var errClosed = errors.New("lzw: reader/writer is closed")

// Close prevents further reads without closing the underlying reader.
func (d *decoder) Close() error {
	d.err = errClosed // in case any Reads come along
	return nil
}

// NewReader creates a new io.ReadCloser.
// Reads from the returned io.ReadCloser read and decompress data from r.
// If r does not also implement io.ByteReader,
// the decompressor may read more data than necessary from r.
// It is the caller's responsibility to call Close on the ReadCloser when
// finished reading.
// oneOff makes code length increases occur one code early. It should be true
// for LZWDecode filters with earlyChange=1 which is also the default.
func NewReader(r io.Reader, oneOff bool) io.ReadCloser {
	br, ok := r.(io.ByteReader)
	if !ok {
		br = bufio.NewReader(r)
	}

	lw := uint(8)
	clear := uint16(1) << lw
	width := 1 + lw

	return &decoder{
		r:        br,
		read:     (*decoder).readMSB,
		litWidth: lw,
		width:    width,
		clear:    clear,
		eof:      clear + 1,
		hi:       clear + 1,
		overflow: uint16(1) << width,
		last:     decoderInvalidCode,
		oneOff:   oneOff,
	}
}
