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

// See 7.4 for a list of defined filter pdfcpu.

import (
	"bytes"
	"io"

	"github.com/hhrutter/pdfcpu/pkg/log"
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
	//Encode(r io.Reader, w io.Writer) error
	//Decode(r io.Reader, w io.Writer) error
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

	// DCT
	// JBIG2
	// JPX

	default:
		log.Info.Printf("Filter not supported: <%s>", filterName)
		err = ErrUnsupportedFilter
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
