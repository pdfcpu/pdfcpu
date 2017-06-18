package filter

import (
	"bytes"
	"io"
)

/////////////////
// ASCIIHexDecode
/////////////////

type asciiHexDecode struct {
	baseFilter
}

// Encode implements encoding for an ASCIIHexDecode filter.
func (f asciiHexDecode) Encode(r io.Reader) (*bytes.Buffer, error) {
	return nil, nil
}

// Decode implements decoding for an ASCIIHexDecode filter.
func (f asciiHexDecode) Decode(r io.Reader) (*bytes.Buffer, error) {
	return nil, nil
}
