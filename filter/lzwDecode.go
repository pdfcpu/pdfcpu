package filter

import (
	"bytes"
	"compress/lzw"
	"io"

	"github.com/hhrutter/pdfcpu/log"
)

type lzwDecode struct {
	baseFilter
}

// Encode implements encoding for an LZWDecode filter.
func (f lzwDecode) Encode(r io.Reader) (*bytes.Buffer, error) {

	log.Debug.Println("EncodeLZW begin")

	var b bytes.Buffer

	wc := lzw.NewWriter(&b, lzw.MSB, 8)
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

	rc := lzw.NewReader(r, lzw.MSB, 8)
	defer rc.Close()

	var b bytes.Buffer
	written, err := io.Copy(&b, rc)
	if err != nil {
		return nil, err
	}
	log.Debug.Printf("DecodeLZW: decoded %d bytes.\n", written)

	return &b, nil
}
