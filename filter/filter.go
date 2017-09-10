// Package filter contains implementations for PDF filters.
package filter

// See 7.4 for a list of defined filter types.

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

var (
	logDebugFilter, logInfoFilter, logWarningFilter, logErrorFilter *log.Logger

	// ErrUnsupportedFilter signals an unsupported filter type.
	ErrUnsupportedFilter = errors.New("Filter not supported")
)

func init() {

	logDebugFilter = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	//logDebugFilter = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	logInfoFilter = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logWarningFilter = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	logErrorFilter = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

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
		logWarningFilter.Printf("Filter not supported: %s", filterName)
		err = ErrUnsupportedFilter
	}

	return
}

type baseFilter struct {
	decodeParms *types.PDFDict
	encodeParms *types.PDFDict
}

// EncodeStream encodes stream dict data by applying its filter pipeline.
func EncodeStream(streamDict *types.PDFStreamDict) (err error) {

	logDebugFilter.Printf("encodeStream begin")

	// No filter specified, nothing to encode.
	if streamDict.FilterPipeline == nil {
		logDebugFilter.Println("encodeStream: returning uncompressed stream.")
		streamDict.Raw = streamDict.Content
		streamLength := int64(len(streamDict.Raw))
		streamDict.StreamLength = &streamLength
		streamDict.Insert("Length", types.PDFInteger(streamLength))
		return
	}

	var b io.Reader
	b = bytes.NewReader(streamDict.Content)

	var c *bytes.Buffer

	// Apply each filter in the pipeline to result of preceding filter.
	for _, f := range streamDict.FilterPipeline {

		if f.DecodeParms != nil {
			logDebugFilter.Printf("encodeStream: encoding filter:%s\ndecodeParms:%s\n", f.Name, f.DecodeParms)
		} else {
			logDebugFilter.Printf("encodeStream: encoding filter:%s\n", f.Name)
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

	logDebugFilter.Printf("encodeStream end")

	return
}

// DecodeStream decodes streamDict data by applying its filter pipeline.
func DecodeStream(streamDict *types.PDFStreamDict) (err error) {

	logDebugFilter.Printf("decodeStream begin")

	// No filter specified, nothing to decode.
	if streamDict.FilterPipeline == nil {
		logDebugFilter.Println("decodeStream: returning uncompressed stream.")
		streamDict.Content = streamDict.Raw
		return
	}

	var b io.Reader
	b = bytes.NewReader(streamDict.Raw)

	var c *bytes.Buffer

	// Apply each filter in the pipeline to result of preceding filter.
	for _, f := range streamDict.FilterPipeline {

		if f.DecodeParms != nil {
			logDebugFilter.Printf("decodeStream: decoding filter:%s\ndecodeParms:%s\n", f.Name, f.DecodeParms)
		} else {
			logDebugFilter.Printf("decodeStream: decoding filter:%s\n", f.Name)
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

	logDebugFilter.Printf("decodeStream end")

	return
}
