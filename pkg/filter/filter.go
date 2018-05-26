// Package filter contains PDF filter implementations.
package filter

// See 7.4 for a list of defined filter pdfcpu.

import (
	"bytes"
	"io"

	"github.com/hhrutter/pdfcpu/pkg/log"
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

	case "ASCII85Decode":
		filter = ascii85Decode{baseFilter{}}

	case "ASCIIHexDecode":
		filter = asciiHexDecode{baseFilter{}}

	case "RunLengthDecode":
		filter = runLengthDecode{baseFilter{parms}}

	case "LZWDecode":
		filter = lzwDecode{baseFilter{parms}}

	case "FlateDecode":
		filter = flate{baseFilter{parms}}

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
	return []string{"ASCII85Decode", "ASCIIHexDecode", "RunLengthDecode", "LZWDecode", "FlateDecode"}
}

type baseFilter struct {
	parms map[string]int
}
