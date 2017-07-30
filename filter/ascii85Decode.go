package filter

import (
	"bytes"
	"encoding/ascii85"
	"io"
	"io/ioutil"
)

type ascii85Decode struct {
	baseFilter
}

// Encode implements encoding for an ASCII85Decode filter.
func (f ascii85Decode) Encode(r io.Reader) (*bytes.Buffer, error) {

	p, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	encoder := ascii85.NewEncoder(buf)
	encoder.Write(p)
	encoder.Close()

	return buf, nil
}

// Decode implements decoding for an ASCII85Decode filter.
func (f ascii85Decode) Decode(r io.Reader) (*bytes.Buffer, error) {

	decoder := ascii85.NewDecoder(r)

	buf, err := ioutil.ReadAll(decoder)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(buf), nil
}
