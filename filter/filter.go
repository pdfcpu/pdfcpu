// Package filter contains PDF filter implementations.
package filter

// See 7.4 for a list of defined filter pdfcpu.

import (
	"bytes"
	"io"

	"github.com/hhrutter/pdfcpu/log"
	"github.com/pkg/errors"
)

var (

	// ErrUnsupportedFilter signals an unsupported filter type.
	ErrUnsupportedFilter = errors.New("Filter not supported")
)

// Filter defines an interface for encoding/decoding buffers.
type Filter interface {
	Encode(r io.Reader) (*bytes.Buffer, error)
	Decode(r io.Reader) (*bytes.Buffer, error)
}

// NewFilter returns a filter for given filterName and an optional parameter dictionary.
func NewFilter(filterName string, parms map[string]int) (filter Filter, err error) {

	switch filterName {

	case "FlateDecode":
		filter = flate{baseFilter{parms}}

	case "ASCII85Decode":
		filter = ascii85Decode{baseFilter{}}

	case "ASCIIHexDecode":
		filter = asciiHexDecode{baseFilter{}}

	case "LZWDecode":
		filter = lzwDecode{baseFilter{parms}}

	// RunLengthDecode
	// CCITTFaxDecode
	// JBIG2Decode
	// DCTDecode
	// JPXDecode

	default:
		log.Info.Printf("Filter not supported: <%s>", filterName)
		err = ErrUnsupportedFilter
	}

	return filter, err
}

// List return the list of all supported PDF filters.
func List() []string {
	return []string{"FlateDecode", "ASCII85Decode", "ASCIIHexDecode", "LZWDecode"}
}

type baseFilter struct {
	parms map[string]int
}
