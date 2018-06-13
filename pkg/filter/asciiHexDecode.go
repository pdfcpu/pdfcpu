package filter

import (
	"bytes"
	"encoding/hex"
	"io"
	"io/ioutil"
)

type asciiHexDecode struct {
	baseFilter
}

const eodHexDecode = '>'

// Encode implements encoding for an ASCIIHexDecode filter.
func (f asciiHexDecode) Encode(r io.Reader) (*bytes.Buffer, error) {

	bb, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	dst := make([]byte, hex.EncodedLen(len(bb)))
	hex.Encode(dst, bb)

	// eod marker
	dst = append(dst, eodHexDecode)

	return bytes.NewBuffer(dst), nil
}

// Decode implements decoding for an ASCIIHexDecode filter.
func (f asciiHexDecode) Decode(r io.Reader) (*bytes.Buffer, error) {

	bb, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var p []byte

	// Remove any white space and cut off on eod
	for i := 0; i < len(bb); i++ {
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

	dst := make([]byte, hex.DecodedLen(len(p)))

	_, err = hex.Decode(dst, p)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(dst), nil
}
