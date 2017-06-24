package filter

import (
	"bytes"
	"io"
)

/////////////////
// ASCII85Decode
/////////////////

type ascii85Decode struct {
	baseFilter
}

// Encode implements encoding for an ASCII85Decode filter.
func (f ascii85Decode) Encode(r io.Reader) (*bytes.Buffer, error) {
	return nil, nil
}

// Decode implements decoding for an ASCII85Decode filter.
func (f ascii85Decode) Decode(r io.Reader) (*bytes.Buffer, error) {
	return nil, nil
}
