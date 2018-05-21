package filter

import (
	"bytes"
	"encoding/ascii85"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
)

type ascii85Decode struct {
	baseFilter
}

const eodASCII85 = "~>"

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

	// Add eod sequence
	buf.WriteString(eodASCII85)

	return buf, nil
}

// Decode implements decoding for an ASCII85Decode filter.
func (f ascii85Decode) Decode(r io.Reader) (*bytes.Buffer, error) {

	p, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if !bytes.HasSuffix(p, []byte(eodASCII85)) {
		return nil, errors.New("Decode: missing eod marker")
	}

	// Strip eod sequence: "~>"
	p = p[:len(p)-2]

	decoder := ascii85.NewDecoder(bytes.NewReader(p))

	buf, err := ioutil.ReadAll(decoder)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(buf), nil
}
