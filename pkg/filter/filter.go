// Package filter contains PDF filter implementations.
package filter

// See 7.4 for a list of defined filter pdfcpu.

import (
	"bytes"
	"io"

	"github.com/trussworks/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// PDF defines the following filters.
const (
	ASCII85   = "ASCII85Decode"
	ASCIIHex  = "ASCIIHexDecode"
	RunLength = "RunLengthDecode"
	LZW       = "LZWDecode"
	Flate     = "FlateDecode"
	CCITTFax  = "CCITTFaxDecode"
	JBIG2     = "JBIG2Decode"
	DCT       = "DCTDecode"
	JPX       = "JPXDecode"
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

	case ASCII85:
		filter = ascii85Decode{baseFilter{}}

	case ASCIIHex:
		filter = asciiHexDecode{baseFilter{}}

	case RunLength:
		filter = runLengthDecode{baseFilter{parms}}

	case LZW:
		filter = lzwDecode{baseFilter{parms}}

	case Flate:
		filter = flate{baseFilter{parms}}

	// CCITTFax
	// JBIG2
	// DCT
	// JPX

	default:
		log.Info.Printf("Filter not supported: <%s>", filterName)
		err = ErrUnsupportedFilter
	}

	return filter, err
}

// List return the list of all supported PDF filters.
func List() []string {
	return []string{ASCII85, ASCIIHex, RunLength, LZW, Flate}
}

type baseFilter struct {
	parms map[string]int
}
