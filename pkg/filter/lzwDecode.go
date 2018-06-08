package filter

import (
	"bytes"
	"io"

	"github.com/hhrutter/pdfcpu/pkg/compress/lzw"
	"github.com/hhrutter/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

type lzwDecode struct {
	baseFilter
}

// Encode implements encoding for an LZWDecode filter.
func (f lzwDecode) Encode(r io.Reader) (*bytes.Buffer, error) {

	log.Debug.Println("EncodeLZW begin")

	var b bytes.Buffer

	ec, ok := f.parms["EarlyChange"]
	if !ok {
		ec = 1
	}

	wc := lzw.NewWriter(&b, ec == 1)
	defer wc.Close()

	written, err := io.Copy(wc, r)
	if err != nil {
		return nil, err
	}
	log.Debug.Printf("EncodeLZW end: %d bytes written\n", written)

	return &b, nil
}

// Decode implements decoding for an LZWDecode filter.
func (f lzwDecode) Decode(r io.Reader) (*bytes.Buffer, error) {

	log.Debug.Println("DecodeLZW begin")

	p, found := f.parms["Predictor"]
	if found && p > 1 {
		return nil, errors.Errorf("DecodeLZW: unsupported predictor %d", p)
	}

	ec, ok := f.parms["EarlyChange"]
	if !ok {
		ec = 1
	}

	rc := lzw.NewReader(r, ec == 1)
	defer rc.Close()

	var b bytes.Buffer
	written, err := io.Copy(&b, rc)
	if err != nil {
		return nil, err
	}
	log.Debug.Printf("DecodeLZW: decoded %d bytes.\n", written)

	return &b, nil
}
