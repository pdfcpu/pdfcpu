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
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestReaderFullDictionary(t *testing.T) {
	_, expected := fullDictionaryCodes()
	if len(expected) != 3842 || !bytes.HasSuffix(expected, []byte(" \n%BTBTB")) {
		t.Fatal("invalid full-dictionary fixture")
	}
	for _, earlyChange := range []bool{false, true} {
		for _, continuation := range []string{"end", "literals", "dictionary", "clear"} {
			t.Run(fmt.Sprintf("EarlyChange=%t/%s", earlyChange, continuation), func(t *testing.T) {
				codes, want := fullDictionaryCodes()
				switch continuation {
				case "literals":
					for range 70000 {
						codes = append(codes, '!')
						want = append(want, '!')
					}
				case "dictionary":
					for range 70000 {
						codes = append(codes, 4095)
						want = append(want, "BTB"...)
					}
				case "clear":
					codes = append(codes, 256, 'A', 258)
					want = append(want, "AAA"...)
				}
				codes = append(codes, 257)
				assertDecodedCodes(t, codes, earlyChange, want)
			})
		}
	}
}

func fullDictionaryCodes() ([]uint16, []byte) {
	// The first three literals define entry 259 as "BT". After 3837 literals and entry 259,
	// entry 4095 is the next free slot and must expand the previous "BT" to "BTB".
	want := append([]byte("%BT"), bytes.Repeat([]byte{' '}, 3832)...)
	want = append(want, '\n', '%')
	codes := []uint16{256}
	for _, b := range want {
		codes = append(codes, uint16(b))
	}
	codes = append(codes, 259, 4095)
	want = append(want, "BTBTB"...)
	return codes, want
}

func TestReaderCodeWidthTransitions(t *testing.T) {
	for _, earlyChange := range []bool{false, true} {
		for _, count := range []int{253, 254, 255, 765, 766, 767, 1789, 1790, 1791} {
			t.Run(fmt.Sprintf("EarlyChange=%t/literals=%d", earlyChange, count), func(t *testing.T) {
				codes := []uint16{256}
				want := make([]byte, count)
				for i := range want {
					want[i] = byte(i)
					codes = append(codes, uint16(want[i]))
				}
				codes = append(codes, 257)
				assertDecodedCodes(t, codes, earlyChange, want)
			})
		}
	}
}

func assertDecodedCodes(t *testing.T, codes []uint16, earlyChange bool, want []byte) {
	t.Helper()
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("decoder panicked: %v", p)
		}
	}()
	rc := NewReader(bytes.NewReader(packPDFCodes(codes, earlyChange)), earlyChange)
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("decoded %d bytes, tail %q; want %d bytes, tail %q",
			len(got), got[max(0, len(got)-8):], len(want), want[max(0, len(want)-8):])
	}
}

func packPDFCodes(codes []uint16, earlyChange bool) []byte {
	// Pack independently of the production writer, which clears before filling the table.
	// Count data codes since clear; each code after the first defines one dictionary entry.
	var out []byte
	var bits uint32
	var nBits uint
	count, adjustment := 0, 0
	if earlyChange {
		adjustment = 1
	}
	for _, code := range codes {
		width := pdfCodeWidth(count + adjustment)
		bits = bits<<width | uint32(code)
		nBits += width
		for nBits >= 8 {
			nBits -= 8
			out = append(out, byte(bits>>nBits))
		}
		if code == 256 {
			count = 0
		} else if code != 257 {
			count++
		}
	}
	if nBits > 0 {
		out = append(out, byte(bits<<(8-nBits)))
	}
	return out
}

func pdfCodeWidth(count int) uint {
	switch {
	case count < 255:
		return 9
	case count < 767:
		return 10
	case count < 1791:
		return 11
	default:
		return 12
	}
}

func TestReaderDecodesPDFMSB(t *testing.T) {
	rc := NewReader(strings.NewReader("\x80\x0b\x60\x50\x22\x0c\x0c\x85\x01"), true)
	defer rc.Close()

	var b bytes.Buffer
	if _, err := io.Copy(&b, rc); err != nil {
		t.Fatal(err)
	}
	if got, want := b.String(), "-----A---B"; got != want {
		t.Fatalf("decoded bytes = %q, want %q", got, want)
	}
}

func TestEarlyChangeRoundTrip(t *testing.T) {
	payload := bytes.Repeat([]byte("pdfcpu LZW EarlyChange regression data "), 512)
	for _, earlyChange := range []bool{false, true} {
		testEarlyChangeRoundTrip(t, payload, earlyChange)
	}
}

func testEarlyChangeRoundTrip(t *testing.T, payload []byte, earlyChange bool) {
	t.Helper()

	var enc bytes.Buffer
	wc := NewWriter(&enc, earlyChange)
	if _, err := wc.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := wc.Close(); err != nil {
		t.Fatal(err)
	}

	rc := NewReader(&enc, earlyChange)
	defer rc.Close()

	var dec bytes.Buffer
	if _, err := io.Copy(&dec, rc); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dec.Bytes(), payload) {
		t.Fatalf("earlyChange=%t round trip mismatch", earlyChange)
	}
}
