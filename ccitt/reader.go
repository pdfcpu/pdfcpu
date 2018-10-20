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

// Package ccitt implements a CCITT Fax image decoder and encoder.
package ccitt

import (
	"errors"
	"io"
	"io/ioutil"
)

// 1 bit deep pixel buffer
type pixelBuf struct {
	buf    []uint8 // size = w * h / 8 +1
	stride int     // stride (in bytes) between vertically adjacent pixels.
	w      int     // image width = row length
}

func (pb *pixelBuf) addRow() {
	pb.buf = append(pb.buf, make([]uint8, pb.stride)...)
}

func (pb *pixelBuf) whitePix(row, x int) bool {
	bit := uint8(7 - (x % 8))
	off := row*pb.stride + x/8
	mask := uint8(0x01 << bit)
	//fmt.Printf("whitePix(row=%d, x=%d) off=%d mask=x%02x (bit=%d) %08b\n", row, x, off, mask, bit, pb.buf[off])
	return pb.buf[off]&mask > 0
}

func (pb *pixelBuf) nextChangeElement(row, start int) int {
	white := true
	if start >= 0 {
		white = pb.whitePix(row, start)
	}
	//fmt.Printf("nextChangeElement starts at row=%d start=%d with white=%t (end=%d)\n", row, start, white, pb.w-1)

	i := start + 1
	for i < pb.w && pb.whitePix(row, i) == white {
		i++
	}
	//fmt.Printf("nextChangeElement: ends with changeElement at %d\n", i)
	return i
}

func (pb *pixelBuf) nextChangeElementWithColor(row, start int, white bool) int {
	//fmt.Printf("nextChangeElementWithColor begin at row=%d start=%d with white=%t (end=%d)\n", row, start, white, pb.w-1)
	b := pb.nextChangeElement(row, start)
	if b == 0 || b == pb.w || pb.whitePix(row, b) == white {
		return b
	}
	e := pb.nextChangeElement(row, b)
	//fmt.Printf("nextChangeElementWithColor returning %d\n", e)
	return e
}

func (pb *pixelBuf) setPix(row, x int) {
	bit := uint8(7 - (x % 8))
	off := row*pb.stride + x/8
	mask := uint8(0x01 << bit)
	// fmt.Printf("setPix(%d,%d): off=%d mask=%d (bit=%d)\n", row, x, off, mask, bit)
	pb.buf[off] |= mask
	//fmt.Printf("i=%d off=%d mask=%d (bit=%d) %08b\n", pb.i, off, mask, bit, pb.buf[off])
}

func (pb *pixelBuf) addRun(row, a0, l int, white bool) {
	//fmt.Printf("addRun: row=%d a0=%d l=%d white=%t\n", row, a0, l, white)
	if white {
		//fmt.Printf("addWhiteRun: row=%d a0=%d l=%d\n", row, a0, l)
		for i := 0; i < l; i++ {
			pb.setPix(row, a0+i)
		}
	}
}

func (pb *pixelBuf) invertBuf() {
	//fmt.Println("invertBuf")
	for i, b := range pb.buf {
		pb.buf[i] = b ^ 0xff
	}
}

type buf []uint8

func shiftFor(a, b int) uint8 {
	shift := a
	if b > 0 {
		shift = a - 8 + b
	}
	return uint8(shift)
}

// getBitBuf provides the next 24 Bits starting at pos
func (b buf) getBitBuf(pos int) (*bitBuf, error) {

	bit := pos % 8
	off := pos / 8

	// fmt.Printf("getBitBuf(%d) off=%d bit=%d\n", pos, off, bit)

	if off >= len(b) {
		// fmt.Printf("off=%d len=%d\n", off, len(b))
		return nil, errors.New("bitBuf overflow")
	}

	var bb bitBuf

	if bit > 0 {
		mask := uint8(0xff >> uint8(bit))
		bb |= bitBuf(b[off] & mask)
		if off == len(b)-1 {
			bb <<= shiftFor(32, bit)
			return &bb, nil
		}
		off++
		bb <<= 8
	}

	bb |= bitBuf(b[off])
	if off == len(b)-1 {
		bb <<= shiftFor(24, bit)
		return &bb, nil
	}
	off++
	bb <<= 8

	bb |= bitBuf(b[off])
	if off == len(b)-1 {
		bb <<= shiftFor(16, bit)
		return &bb, nil
	}
	off++
	bb <<= 8

	bb |= bitBuf(b[off])
	bb <<= shiftFor(8, bit)
	return &bb, nil
}

type bitBuf uint32

func (b bitBuf) hasPrefix(code string) bool {
	for i, bit := range code {
		mask := uint32(0x01 << uint8(31-i))
		k := b & bitBuf(mask)
		j := int(bit) - 48
		if j == 1 {
			if k == 0 {
				return false
			}
		} else {
			if k > 0 {
				return false
			}
		}
	}
	// fmt.Printf("has prefix = %s\n", code)
	return true
}

type decoder struct {
	r      io.Reader
	raw    buf
	pos    int
	err    error
	pb     *pixelBuf
	w      int    // image width = row length
	white  bool   // current color, true for set pixel bit.
	align  bool   // pad last row byte with 0s
	inv    bool   // invert all pixels
	row    int    // current row number 0..h-1
	a0     int    // current horizontal scan position -1..w
	toRead []byte // bytes to return from Read prepared after decoding
}

func (d *decoder) nextMode() (mode string, err error) {

	//fmt.Printf("d.align=%t\n", d.align)

	if d.align && d.a0 < 0 {
		// Skip bits until next byte boundary.
		if d.pos%8 > 0 {
			d.pos += 8 - d.pos%8
		}
	}

	bb, err := d.raw.getBitBuf(d.pos)
	if err != nil {
		return "", err
	}

	for _, v := range codes {
		if !bb.hasPrefix(v) {
			continue
		}
		d.pos += len(v)
		return v, nil
	}

	return "", errors.New("ccitt: corrupt group4 data")
}

func (d *decoder) nextRunLength() (int, error) {

	var err error

	c := 0

	// Process optional big markup codes.

	ok := true
	for ok {

		ok = false

		bb, err := d.raw.getBitBuf(d.pos)
		if err != nil {
			return c, err
		}

		for k, v := range makeupBig {
			if !bb.hasPrefix(k) {
				continue
			}
			d.pos += len(k)
			c += v
			ok = true
			break
		}
	}

	// Process optional markup code.

	makeup := makeupW
	if !d.white {
		makeup = makeupB
	}

	bb, err := d.raw.getBitBuf(d.pos)
	if err != nil {
		return c, err
	}

	for k, v := range makeup {
		if !bb.hasPrefix(k) {
			continue
		}
		d.pos += len(k)
		c += v
		break
	}

	// Process terminal code.

	term := termW
	if !d.white {
		term = termB
	}

	bb, err = d.raw.getBitBuf(d.pos)
	if err != nil {
		return c, err
	}

	ok = false
	for k, v := range term {
		if !bb.hasPrefix(k) {
			continue
		}
		d.pos += len(k)
		c += v
		ok = true
		break
	}

	if !ok {
		err = errors.New("group4: missing terminating code")
	}

	return c, err
}

func (d *decoder) calcb1(a0 int) int {
	if d.row == 0 {
		// b1 is 0 for first row
		return d.w
	}
	return d.pb.nextChangeElementWithColor(d.row-1, a0, !d.white)
}

func (d *decoder) calcb2(a0 int) int {
	b1 := d.calcb1(a0)
	//fmt.Printf("calcb2(%d) b1=%d eol=%d\n", a0, b1, d.pb.w-1)
	return d.pb.nextChangeElement(d.row-1, b1)
}

func (d *decoder) handlePass() {
	b2 := d.calcb2(d.a0)
	d.pb.addRun(d.row, d.a0, b2-d.a0, d.white)
	d.a0 = b2
	// fmt.Printf("end Pass a0=%d (b2=%d)\n", d.a0, b2)
}

func (d *decoder) handleHorizontal() {

	if d.a0 == -1 {
		d.a0++
	}

	// One run using the current color.
	// fmt.Println("nextRunLength")
	r1, err := d.nextRunLength()
	if err != nil {
		d.err = err
		return
	}
	d.pb.addRun(d.row, d.a0, r1, d.white)
	d.a0 += r1

	// Another run using the opposite color.
	// fmt.Println("nextRunLength")
	d.white = !d.white
	r2, err := d.nextRunLength()
	if err != nil {
		d.err = err
		return
	}
	d.pb.addRun(d.row, d.a0, r2, d.white)
	d.a0 += r2

	d.white = !d.white

	// fmt.Printf("end Horiz(%d,%d) a0=%d\n", r1, r2, d.a0)
}

func (d *decoder) handleVertical(offa1b1 int) {
	b1 := d.calcb1(d.a0)
	if b1 == 0 {
		// at row start
		a1 := b1 + offa1b1
		d.a0 = a1
		d.white = false
	} else {
		a1 := b1 + offa1b1
		if a1 > d.w {
			a1 = d.w
		}
		if d.a0 < 0 {
			d.a0 = 0
		}
		d.pb.addRun(d.row, d.a0, a1-d.a0, d.white)
		d.a0 = a1
		d.white = !d.white
	}
}

func (d *decoder) initBufs() {

	// Prepare raw buf.
	var err error
	d.raw, err = ioutil.ReadAll(d.r)
	if err != nil {
		d.err = err
		return
	}

	// Prepare pixelbuf.
	stride := d.w / 8
	if d.w%8 > 0 {
		stride++
	}
	d.pb = &pixelBuf{buf: make([]uint8, stride), stride: stride, w: d.w}
	// fmt.Printf("len(d.pb.buf)=%d stride=%d\n", len(d.pb.buf), stride)
}

// decode decompresses bytes from r and leaves them in d.toRead.
func (d *decoder) decode() {

	d.initBufs()

	for {

		// fmt.Printf("\nrow=%d a0=%d white=%t\n", d.row, d.a0, d.white)

		if d.a0 == d.w {
			d.a0 = -1
			d.row++
			d.pb.addRow()
			d.white = true
		}

		mode, err := d.nextMode()
		if err != nil {
			d.err = err
			return
		}

		switch mode {

		case mP: // a0 to b2 keeps color, a0 = b2
			// fmt.Printf("begin Pass\n")
			d.handlePass()

		case mH: // a0 to a1-1 keeps color, change color on a1, a1-a2-1 keeps color, change color on a2, a0 = a2
			// fmt.Printf("begin Horiz\n")
			d.handleHorizontal()

		case mV0: // a0 to a1-1 keeps color, change color on a1, a0 = a1
			// fmt.Println("V(0)")
			d.handleVertical(0)

		case mVL1:
			// fmt.Println("VL(1)")
			d.handleVertical(-1)

		case mVL2:
			// fmt.Println("VL(2)")
			d.handleVertical(-2)

		case mVL3:
			// fmt.Println("VL(3)")
			d.handleVertical(-3)

		case mVR1:
			// fmt.Println("VR(1)")
			d.handleVertical(1)

		case mVR2:
			// fmt.Println("VR(2)")
			d.handleVertical(2)

		case mVR3:
			// fmt.Println("VR(3)")
			d.handleVertical(3)

		case eofb:
			//fmt.Println("eofb")

		case ext:
			//fmt.Println("ccitt: uncompressed mode not supported")
			d.err = errors.New("ccitt: uncompressed mode not supported")

		}

		if d.err != nil {
			return
		}

		if mode == eofb {
			break
		}

	}

	if d.inv {
		d.pb.invertBuf()
	}

	d.toRead = d.pb.buf
	d.err = io.EOF
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

var errClosed = errors.New("ccitt: reader/writer is closed")

func (d *decoder) Close() error {
	d.err = errClosed // in case any Reads come along
	return nil
}

// NewReader creates a new ReadCloser
// Reads from the returned io.ReadCloser read and decompress data from r.
// If r does not also implement io.ByteReader,
// the decompressor may read more data than necessary from r.
// It is the caller's responsibility to call Close on the ReadCloser when
// finished reading.
func NewReader(r io.Reader, w int, inverse, align bool) io.ReadCloser {
	return &decoder{r: r, white: true, w: w, inv: inverse, align: align}
}
