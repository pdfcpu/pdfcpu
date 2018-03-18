// Package filter contains PDF filter implementations.
package filter

// See 7.4 for a list of defined filter types.

import (
	"bytes"
	"io"

	"github.com/hhrutter/pdfcpu/log"
	"github.com/hhrutter/pdfcpu/types"
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
func NewFilter(filterName string, decodeParms, encodeParms *types.PDFDict) (filter Filter, err error) {

	switch filterName {

	case "FlateDecode":
		filter = flate{baseFilter{decodeParms, encodeParms}}

	case "ASCII85Decode":
		filter = ascii85Decode{baseFilter{decodeParms, encodeParms}}

	case "ASCIIHexDecode":
		filter = asciiHexDecode{baseFilter{decodeParms, encodeParms}}

	// LZWDecode
	// RunLengthDecode
	// CCITTFaxDecode
	// JBIG2Decode
	// DCTDecode
	// JPXDecode

	default:
		log.Info.Printf("Filter not supported: %s", filterName)
		err = ErrUnsupportedFilter
	}

	return filter, err
}

type baseFilter struct {
	decodeParms *types.PDFDict
	encodeParms *types.PDFDict
}

// EncodeStream encodes stream dict data by applying its filter pipeline.
func EncodeStream(streamDict *types.PDFStreamDict) error {

	log.Debug.Printf("encodeStream begin")

	// No filter specified, nothing to encode.
	if streamDict.FilterPipeline == nil {
		log.Debug.Println("encodeStream: returning uncompressed stream.")
		streamDict.Raw = streamDict.Content
		streamLength := int64(len(streamDict.Raw))
		streamDict.StreamLength = &streamLength
		streamDict.Insert("Length", types.PDFInteger(streamLength))
		return nil
	}

	var b io.Reader
	b = bytes.NewReader(streamDict.Content)

	var c *bytes.Buffer

	// Apply each filter in the pipeline to result of preceding filter.
	for _, f := range streamDict.FilterPipeline {

		if f.DecodeParms != nil {
			log.Debug.Printf("encodeStream: encoding filter:%s\ndecodeParms:%s\n", f.Name, f.DecodeParms)
		} else {
			log.Debug.Printf("encodeStream: encoding filter:%s\n", f.Name)
		}

		fi, err := NewFilter(f.Name, f.DecodeParms, nil)
		if err != nil {
			return err
		}

		c, err = fi.Encode(b)
		if err != nil {
			return err
		}

		b = c
	}

	streamDict.Raw = c.Bytes()

	//DumpBuf(c.Bytes(), 32, "decodedStream returning:")

	streamLength := int64(len(streamDict.Raw))
	streamDict.StreamLength = &streamLength
	streamDict.Insert("Length", types.PDFInteger(streamLength))

	log.Debug.Printf("encodeStream end")

	return nil
}

// DecodeStream decodes streamDict data by applying its filter pipeline.
func DecodeStream(streamDict *types.PDFStreamDict) error {

	log.Debug.Printf("decodeStream begin")

	// No filter specified, nothing to decode.
	if streamDict.FilterPipeline == nil {
		log.Debug.Println("decodeStream: returning uncompressed stream.")
		streamDict.Content = streamDict.Raw
		return nil
	}

	var b io.Reader
	b = bytes.NewReader(streamDict.Raw)

	var c *bytes.Buffer

	// Apply each filter in the pipeline to result of preceding filter.
	for _, f := range streamDict.FilterPipeline {

		if f.DecodeParms != nil {
			log.Debug.Printf("decodeStream: decoding filter:%s\ndecodeParms:%s\n", f.Name, f.DecodeParms)
		} else {
			log.Debug.Printf("decodeStream: decoding filter:%s\n", f.Name)
		}

		fi, err := NewFilter(f.Name, f.DecodeParms, nil)
		if err != nil {
			return err
		}

		c, err = fi.Decode(b)
		if err != nil {
			return err
		}

		b = c
	}

	streamDict.Content = c.Bytes()

	//DumpBuf(c.Bytes(), 32, "decodedStream returning:")

	log.Debug.Printf("decodeStream end")

	return nil
}
