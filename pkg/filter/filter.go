/*
Copyright 2018 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package filter contains PDF filter implementations.
package filter

import (
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// PDF defines the following filters. See also 7.4 in the PDF spec.
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

// ErrUnsupportedFilter signals unsupported filter encountered.
var ErrUnsupportedFilter = errors.New("pdfcpu: filter not supported")

// Filter defines an interface for encoding/decoding PDF object streams.
type Filter interface {
	Encode(r io.Reader) (io.Reader, error)
	Decode(r io.Reader) (io.Reader, error)
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

	case CCITTFax:
		filter = ccittDecode{baseFilter{parms}}

	case DCT:
		filter = dctDecode{baseFilter{parms}}

	case JBIG2:
		// Unsupported
		fallthrough

	case JPX:
		// Unsupported
		log.Info.Printf("Filter not supported: <%s>", filterName)
		err = ErrUnsupportedFilter

	default:
		err = errors.Errorf("Invalid filter: <%s>", filterName)
	}

	return filter, err
}

// List return the list of all supported PDF filters.
func List() []string {
	// Exclude CCITTFax, DCT, JBIG2 & JPX since they only makes sense in the context of image processing.
	return []string{ASCII85, ASCIIHex, RunLength, LZW, Flate}
}

type baseFilter struct {
	parms map[string]int
}
