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
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/log"
)

// PDF defines the following filters. See also 7.4 in the PDF spec.
const (
	ASCII85   = "ASCII85Decode"
	ASCIIHex  = "ASCIIHexDecode"
	RunLength = "RunLengthDecode"
	LZW       = "LZWDecode"
	Flate     = "FlateDecode"
	CCITTFax  = "CCITTFaxDecode"
	JBIG2     = "JBIG2Decode" // TODO
	DCT       = "DCTDecode"
	JPX       = "JPXDecode" // TODO
)

// ErrUnsupportedFilter signals unsupported filter encountered.
var ErrUnsupportedFilter = errors.New("pdfcpu: filter not supported")

// ErrDecodeLimitExceeded signals that decoded filter output exceeds the configured decode limit.
var ErrDecodeLimitExceeded = errors.New("pdfcpu: filter decode limit exceeded")

const DefaultMaxDecodeBytes int64 = 512 << 20 // 512 MiB

const maxInt = int(^uint(0) >> 1)
const maxInt64 = int64(^uint64(0) >> 1)

// Filter defines an interface for encoding/decoding PDF object streams.
type Filter interface {
	Encode(r io.Reader) (io.Reader, error)
	Decode(r io.Reader) (io.Reader, error)
	// DecodeLength will decode at least maxLen bytes. For filters where decoding
	// parts doesn't make sense (e.g. DCT), the whole stream is decoded.
	// If maxLen < 0 is passed, the whole stream is decoded.
	DecodeLength(r io.Reader, maxLen int64) (io.Reader, error)
}

// NewFilter returns a filter for given filterName and an optional parameter dictionary.
func NewFilter(filterName string, parms map[string]int, maxDecodeBytes ...int64) (filter Filter, err error) {
	limit := DefaultMaxDecodeBytes
	if len(maxDecodeBytes) > 0 {
		limit = maxDecodeBytes[0]
	}
	switch filterName {

	case ASCII85:
		filter = ascii85Decode{baseFilter{maxDecodeBytes: limit}}

	case ASCIIHex:
		filter = asciiHexDecode{baseFilter{maxDecodeBytes: limit}}

	case RunLength:
		filter = runLengthDecode{baseFilter{parms: parms, maxDecodeBytes: limit}}

	case LZW:
		filter = lzwDecode{baseFilter{parms: parms, maxDecodeBytes: limit}}

	case Flate:
		filter = flate{baseFilter{parms: parms, maxDecodeBytes: limit}}

	case CCITTFax:
		filter = ccittDecode{baseFilter{parms: parms, maxDecodeBytes: limit}}

	case DCT:
		filter = dctDecode{baseFilter{parms: parms, maxDecodeBytes: limit}}

	case JBIG2:
		// Unsupported
		fallthrough

	case JPX:
		// Unsupported
		if log.InfoEnabled() {
			log.Info.Printf("Filter not supported: <%s>", filterName)
		}
		err = ErrUnsupportedFilter

	default:
		err = fmt.Errorf("Invalid filter: <%s>", filterName)
	}

	return filter, err
}

// List return the list of all supported PDF filters.
func List() []string {
	// Exclude CCITTFax, DCT, JBIG2 & JPX since they only makes sense in the context of image processing.
	return []string{ASCII85, ASCIIHex, RunLength, LZW, Flate}
}

type baseFilter struct {
	parms          map[string]int
	maxDecodeBytes int64
}

// SupportsDecodeParms returns true if filterName supports decode parameters.
func SupportsDecodeParms(f string) bool {
	return f == CCITTFax || f == LZW || f == Flate
}

func getReaderBytes(r io.Reader) ([]byte, error) {
	var bb []byte
	if buf, ok := r.(*bytes.Buffer); ok {
		bb = buf.Bytes()
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			return nil, err
		}

		bb = buf.Bytes()
	}

	return bb, nil
}

func (f baseFilter) decodeLimit(maxLen int64) int64 {
	if maxLen >= 0 {
		return maxLen
	}
	if f.maxDecodeBytes == 0 {
		return DefaultMaxDecodeBytes
	}
	return f.maxDecodeBytes
}

func (f baseFilter) copyDecoded(r io.Reader, maxLen int64) (*bytes.Buffer, error) {
	if maxLen >= 0 {
		var b bytes.Buffer
		_, err := io.CopyN(&b, r, maxLen)
		return &b, err
	}

	limit := f.decodeLimit(maxLen)
	if limit < 0 {
		var b bytes.Buffer
		_, err := io.Copy(&b, r)
		return &b, err
	}
	if limit == maxInt64 {
		var b bytes.Buffer
		_, err := io.Copy(&b, r)
		return &b, err
	}

	lr := &io.LimitedReader{R: r, N: limit + 1}
	var b bytes.Buffer
	if _, err := io.Copy(&b, lr); err != nil {
		return &b, err
	}
	if int64(b.Len()) > limit {
		return nil, ErrDecodeLimitExceeded
	}
	return &b, nil
}
