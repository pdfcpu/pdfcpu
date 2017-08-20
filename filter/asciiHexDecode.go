package filter

import (
	"bytes"
	"encoding/hex"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
)

type asciiHexDecode struct {
	baseFilter
}

// EOD represents the end of data marker.
const EOD = '>'

// Encode implements encoding for an ASCIIHexDecode filter.
func (f asciiHexDecode) Encode(r io.Reader) (*bytes.Buffer, error) {

	p, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	dst := make([]byte, hex.EncodedLen(len(p)))
	hex.Encode(dst, p)

	// eod marker
	dst = append(dst, EOD)

	return bytes.NewBuffer(dst), nil
}

// Decode implements decoding for an ASCIIHexDecode filter.
func (f asciiHexDecode) Decode(r io.Reader) (*bytes.Buffer, error) {

	p, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// if no eod then err
	if p[len(p)-1] != EOD {
		return nil, errors.New("Decode: missing eod marker")
	}

	// remove eod,
	p = p[:len(p)-1]

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
