package pdfcpu

// See 7.4 for a list of the defined filters.

import (
	"bytes"
	"io"

	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/log"
)

func parmsForFilter(d *PDFDict) map[string]int {

	m := map[string]int{}

	if d == nil {
		return m
	}

	for k, v := range d.Dict {

		i, ok := v.(PDFInteger)
		if !ok {
			continue
		}
		m[k] = i.Value()
	}

	return m
}

// encodeStream encodes stream dict data by applying its filter pipeline.
func encodeStream(streamDict *PDFStreamDict) error {

	log.Debug.Printf("encodeStream begin")

	// No filter specified, nothing to encode.
	if streamDict.FilterPipeline == nil {
		log.Debug.Println("encodeStream: returning uncompressed stream.")
		streamDict.Raw = streamDict.Content
		streamLength := int64(len(streamDict.Raw))
		streamDict.StreamLength = &streamLength
		streamDict.Insert("Length", PDFInteger(streamLength))
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

		// make parms map[string]int
		parms := parmsForFilter(f.DecodeParms)

		fi, err := filter.NewFilter(f.Name, parms)
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
	streamDict.Insert("Length", PDFInteger(streamLength))

	log.Debug.Printf("encodeStream end")

	return nil
}

// decodeStream decodes streamDict data by applying its filter pipeline.
func decodeStream(streamDict *PDFStreamDict) error {

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

		// make parms map[string]int
		parms := parmsForFilter(f.DecodeParms)

		fi, err := filter.NewFilter(f.Name, parms)
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
