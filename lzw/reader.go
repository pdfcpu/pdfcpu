// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lzw is an enhanced version of compress/lzw.
//
// It implements Adobe's PDF lzw compression as defined for the LZWDecode filter
// and is also compatible with the TIFF file format.
//
// See the golang proposal: https://github.com/golang/go/issues/25409.
//
// More information: https://github.com/hhrutter/pdfcpu/tree/master/lzw
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
	// Other valid codes are in the range [lo, hi] where lo := clear + 2,
	// with the upper bound incrementing on each code seen.
	//
	// overflow is the code at which hi overflows the code width. It always
	// equals 1 << width.
	//
	// last is the most recently seen code, or decoderInvalidCode.
	//
	// An invariant is that
	// (hi < overflow) || (hi == overflow && last == decoderInvalidCode)
	clear, eof, hi, overflow, last uint16

	// Each code c in [lo, hi] expands to two or more bytes. For c != hi:
	//   suffix[c] is the last of these bytes.
	//   prefix[c] is the code for all but the last byte.
	//   This code can either be a literal code or another code in [lo, c).
	// The c == hi case is a special case.
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

func (d *decoder) handleOverflow() {
	ui := d.hi
	if d.oneOff {
		ui++
	}
	if ui >= d.overflow {
		if d.width == maxWidth {
			d.last = decoderInvalidCode
			// Undo the d.hi++ a few lines above, so that (1) we maintain
			// the invariant that d.hi <= d.overflow, and (2) d.hi does not
			// eventually overflow a uint16.
			if !d.oneOff {
				d.hi--
			}
		} else {
			d.width++
			d.overflow <<= 1
		}
	}
}

// decode decompresses bytes from r and leaves them in d.toRead.
// read specifies how to decode bytes into codes.
// litWidth is the width in bits of literal codes.
func (d *decoder) decode() {
	i := 0
	// Loop over the code stream, converting codes into decompressed bytes.
loop:
	for {
		code, err := d.read(d)
		i++
		if err != nil {
			// Some PDF Writers write an EOD some don't.
			// Don't insist on EOD marker.
			// Don't return an unexpected EOF error.
			// if err == io.EOF {
			// 	err = io.ErrUnexpectedEOF
			// }
			d.err = err
			break
		}
		switch {
		case code < d.clear:
			// We have a literal code.
			d.output[d.o] = uint8(code)
			d.o++
			if d.last != decoderInvalidCode {
				// Save what the hi code expands to.
				d.suffix[d.hi] = uint8(code)
				d.prefix[d.hi] = d.last
			}
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
			if d.last != decoderInvalidCode {
				// Save what the hi code expands to.
				d.suffix[d.hi] = uint8(c)
				d.prefix[d.hi] = d.last
			}
		default:
			d.err = errors.New("lzw: invalid code")
			break loop
		}
		d.last, d.hi = code, d.hi+1
		d.handleOverflow()
		if d.o >= flushBuffer {
			break
		}
	}
	// Flush pending output.
	d.toRead = d.output[:d.o]
	d.o = 0
}

var errClosed = errors.New("lzw: reader/writer is closed")

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
