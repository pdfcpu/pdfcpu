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

// Package ccitt implements a CCITT Fax image decoder for Group3/1d and Group4.
package ccitt

import (
	"errors"
	"io"
	"io/ioutil"
	"math/bits"
)

// The supported CCITT encoding modes.
const (
	Group3 = iota
	Group4
)

var errMissingTermCode = errors.New("ccitt: missing terminating code")

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
	//fmt.Printf("setPix(%d,%d): off=%d mask=%d (bit=%d)\n", row, x, off, mask, bit)
	pb.buf[off] |= mask
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

// getBitBuf provides the next 24 bits starting at pos
func (b buf) getBitBuf(pos int) (*bitBuf, error) {

	bit := pos % 8
	off := pos / 8

	//fmt.Printf("getBitBuf(%d) off=%d bit=%d\n", pos, off, bit)

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
	r        io.Reader
	raw      buf
	pos      int
	err      error
	pb       *pixelBuf
	mode     int    // Group3 or Group4
	w        int    // image width = row length
	white    bool   // current color, true for set pixel bit.
	align    bool   // pad last row byte with 0s
	LSBToMSB bool   // fill order is lsb to msb
	inv      bool   // invert all pixels
	row      int    // current row number 0..h-1
	a0       int    // current horizontal scan position -1..w
	toRead   []byte // bytes to return from Read prepared after decoding
}

func (d *decoder) nextMode() (mode string, err error) {

	//fmt.Printf("nextMode: d.align=%t\n", d.align)

	if d.align && d.a0 < 0 {
		// Skip bits until next byte boundary.
		if d.pos%8 > 0 {
			d.pos += 8 - d.pos%8
		}
	}

	// Get next 24 encoded bits.
	bb, err := d.raw.getBitBuf(d.pos)
	if err != nil {
		return "", err
	}
	//fmt.Printf("%032b\n", *bb)

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
		//fmt.Printf("%032b\n", *bb)

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
	//fmt.Printf("%032b\n", *bb)

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
	//fmt.Printf("%032b\n", *bb)

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
		err = errMissingTermCode
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
	//fmt.Printf("begin Pass\n")
	b2 := d.calcb2(d.a0)
	d.pb.addRun(d.row, d.a0, b2-d.a0, d.white)
	d.a0 = b2
	//fmt.Printf("end Pass a0=%d (b2=%d)\n", d.a0, b2)
}

func (d *decoder) handleHorizontal() {

	//fmt.Printf("begin Horiz\n")

	if d.a0 == -1 {
		d.a0++
	}

	// One run using the current color.
	//fmt.Println("nextRunLength")
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
		// Note: group3 encoding skips trailing black runs for a line.
		if err != nil && (d.mode == Group4 || d.mode == Group3 && err != errMissingTermCode) {
			d.err = err
			return
		}
		r2 = d.w - d.a0
	}
	d.pb.addRun(d.row, d.a0, r2, d.white)
	d.a0 += r2

	d.white = !d.white

	//fmt.Printf("end Horiz(%d,%d) a0=%d\n", r1, r2, d.a0)
}

func (d *decoder) handleVertical(offa1b1 int) {
	b1 := d.calcb1(d.a0)
	//fmt.Printf("handleVertical: b1=%d\n", b1)
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

func (d *decoder) handleV0() {
	//fmt.Println("V(0)")
	d.handleVertical(0)
}

func (d *decoder) handleVL1() {
	//fmt.Println("VL(1)")
	d.handleVertical(-1)
}

func (d *decoder) handleVL2() {
	//fmt.Println("VL(2)")
	d.handleVertical(-2)
}

func (d *decoder) handleVL3() {
	//fmt.Println("VL(3)")
	d.handleVertical(-3)
}

func (d *decoder) handleVR1() {
	//fmt.Println("VR(1)")
	d.handleVertical(1)
}

func (d *decoder) handleVR2() {
	//fmt.Println("VR(2)")
	d.handleVertical(2)
}

func (d *decoder) handleVR3() {
	//fmt.Println("VR(3)")
	d.handleVertical(3)
}

func (d *decoder) handleExt() {
	d.err = errors.New("ccitt: uncompressed mode not supported")
}

func (d *decoder) initBufs() {

	// Prepare raw buf.
	var err error
	d.raw, err = ioutil.ReadAll(d.r)
	if err != nil {
		d.err = err
		return
	}

	if d.LSBToMSB {
		for i := 0; i < len(d.raw); i++ {
			d.raw[i] = bits.Reverse8(d.raw[i])
		}
	}

	// Prepare pixelbuf.
	stride := d.w / 8
	if d.w%8 > 0 {
		stride++
	}
	d.pb = &pixelBuf{buf: make([]uint8, stride), stride: stride, w: d.w}
	//fmt.Printf("len(d.pb.buf)=%d stride=%d\n", len(d.pb.buf), stride)
}

func (d *decoder) readEOL() (bool, error) {

	// Get next 24 compressed Bits.
	bb, err := d.raw.getBitBuf(d.pos)
	if err != nil {
		return false, err
	}

	if !bb.hasPrefix(eol) {
		return false, nil
	}
	d.pos += len(eol)

	return true, nil
}

func (d *decoder) decodeGroup3OneDimensional() {

	d.initBufs()

	//fmt.Printf("%d(#%02x)bytes: \n%s\n", len(d.raw), len(d.raw), hex.Dump(d.raw))

	// Check for leading eol.

	ok, err := d.readEOL()
	if err != nil {
		d.err = err
		return
	}
	if !ok {
		d.err = errors.New("group3: missing eol page prefix")
		return
	}

	for {

		//fmt.Printf("\nrow=%d a0=%d white=%t\n", d.row, d.a0, d.white)

		if d.a0 == d.w {
			d.a0 = -1
			d.row++
			d.pb.addRow()
			d.white = true

			// Check for eol.

			ok, err := d.readEOL()
			if err != nil {
				d.err = err
				return
			}
			if !ok {
				d.err = errors.New("group3: missing eol")
				return
			}

			// Check for rtc.

			ok, err = d.readEOL()
			if err != nil {
				d.err = err
				return
			}

			if ok {
				// Processed 2 eol's.
				// Need another 4 eol's to complete rtc sequence.
				for i := 0; i < 4; i++ {
					ok, err = d.readEOL()
					if err != nil {
						d.err = err
						return
					}
					if !ok {
						d.err = errors.New("group3: corrupt rtc")
						return
					}
				}
				// Successfully parsed rtc.
				break
			}

			// Move on to process next line.
		}

		d.handleHorizontal()
		if d.err != nil {
			return
		}

	}

	if d.inv {
		d.pb.invertBuf()
	}

	d.toRead = d.pb.buf
	d.err = io.EOF
}

// decode decompresses bytes from r and leaves them in d.toRead.
func (d *decoder) decodeGroup4() {

	d.initBufs()

	//fmt.Printf("%d(#%02x)bytes: \n%s\n", len(d.raw), len(d.raw), hex.Dump(d.raw))

	for {

		//fmt.Printf("\nrow=%d a0=%d white=%t\n", d.row, d.a0, d.white)

		if d.a0 == d.w {
			d.a0 = -1
			d.row++
			d.pb.addRow()
			d.white = true
			//fmt.Printf("row=%d a0=%d white=%t\n", d.row, d.a0, d.white)
		}

		mode, err := d.nextMode()
		if err != nil {
			d.err = err
			return
		}

		for k, v := range map[string]func(){
			mP:   d.handlePass,
			mH:   d.handleHorizontal,
			mV0:  d.handleV0,
			mVL1: d.handleVL1,
			mVL2: d.handleVL2,
			mVL3: d.handleVL3,
			mVR1: d.handleVR1,
			mVR2: d.handleVR2,
			mVR3: d.handleVR3,
			ext:  d.handleExt,
		} {
			if k == mode {
				v()
				break
			}
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

func (d *decoder) decode() {
	switch d.mode {
	case Group3:
		d.decodeGroup3OneDimensional()
	case Group4:
		d.decodeGroup4()
	}
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
func NewReader(r io.Reader, mode, width int, inverse, align, LSBToMSB bool) io.ReadCloser {
	return &decoder{r: r, mode: mode, white: true, w: width, inv: inverse, align: align, LSBToMSB: LSBToMSB}
}
