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

package ccitt

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeImgToPNG(filename string, img image.Image) (string, error) {

	filename += ".png"

	f, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return filename, png.Encode(f, img)
}

func readImgFromPNG(fileName string) (image.Image, error) {

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return png.Decode(f)
}

func imgForBuf(buf []byte, w, h int) image.Image {

	img := image.NewGray(image.Rect(0, 0, w, h))

	i := 0

	for y := 0; y < h; y++ {
		for x := 0; x < w; {
			p := buf[i]
			for j := 0; j < 8 && x < w; j++ {
				v := p >> 7
				v = v * 0xff
				img.Set(x, y, color.Gray{v})
				p <<= 1
				x++
			}
			i++
		}
	}

	return img
}

func compare(t *testing.T, fileName string, img1, img2 image.Image) {

	if img1.Bounds().Max.X != img2.Bounds().Max.X ||
		img1.Bounds().Max.Y != img2.Bounds().Max.Y {
		t.Errorf("dimension mismatch: %s", fileName)
		return
	}

	w := img1.Bounds().Max.X
	h := img1.Bounds().Max.Y

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if img1.At(x, y) != img2.At(x, y) {
				t.Errorf("pixel mismatch: %s (%d,%d)", fileName, x, y)
				return
			}
		}
	}
}

func testFile(t *testing.T, fileName string, mode, w, h int, inverse, align bool) {

	f, err := os.Open(fileName)
	if err != nil {
		t.Errorf("%s: %v", fileName, err)
		return
	}
	defer f.Close()

	// Read a CCITT encoded file and decode it into buf.
	r := NewReader(f, mode, w, inverse, align, false)
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		t.Errorf("%s: %v", fileName, err)
		return
	}

	r.Close()

	// Generate image from buf.
	img1 := imgForBuf(buf, w, h)

	// Read reference PNG image from buf.
	fnNoExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	img2, err := readImgFromPNG(fnNoExt + ".png")
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	// Compare images.
	compare(t, fnNoExt, img1, img2)
}

func TestCCITT(t *testing.T) {

	for _, tt := range []struct {
		fileName string
		mode     int
		w, h     int
		inverse  bool
		align    bool
	}{

		// Test Group 3 decoding
		{"testdata/scan1.gr3", Group3, 2480, 3508, false, false},
		{"testdata/scan2.gr3", Group3, 1656, 2339, false, false},

		// Test Group 4 decoding
		{"testdata/amt.gr4", Group4, 43, 38, false, false},
		{"testdata/lc.gr4", Group4, 154, 154, false, false},
		{"testdata/do.gr4", Group4, 613, 373, true, false}, // <BlackIs1, true>
		{"testdata/t6diagram.gr4", Group4, 1163, 2433, false, false},
		{"testdata/Wonderwall.gr4", Group4, 2312, 3307, false, false},
		{"testdata/hoare.gr4", Group4, 2550, 3300, false, false},
		{"testdata/jphys.gr4", Group4, 3440, 5200, false, false},
		{"testdata/hl.gr4", Group4, 2548, 3300, false, true}, // <EncodedByteAlign, true>
		{"testdata/ho2.gr4", Group4, 2040, 2640, false, false},
		{"testdata/sie.gr4", Group4, 3310, 8672, false, false},
	} {
		testFile(t, tt.fileName, tt.mode, tt.w, tt.h, tt.inverse, tt.align)
	}

}

func TestBitBuf(t *testing.T) {

	var b bitBuf
	b = 0xa0000000

	fmt.Printf("%032b\n", b)

	code := "10100000000"
	if b.hasPrefix(code) {
		fmt.Printf("<%s> is prefix", code)
	} else {
		fmt.Printf("<%s> is no prefix", code)
	}

}

func printBufBinary(b []byte) {
	for _, v := range b {
		fmt.Printf("%08b|", v)
	}
	fmt.Println()
}

func TestGetBitBuf(t *testing.T) {

	var b buf
	b = []uint8{0xC0, 0xF0, 0xCA, 0x35, 0xB7}
	printBufBinary(b)

	for i := 0; i <= 39; i++ {
		bb, err := b.getBitBuf(i)
		if err != nil {
			t.Errorf("%v", err)
			return
		}
		fmt.Printf("%02d: %032b\n", i, *bb)
	}

}
