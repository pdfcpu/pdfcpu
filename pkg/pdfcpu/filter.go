package pdfcpu

// See 7.4 for a list of the defined filters.

import (
	"bytes"
	"io"

	"github.com/trussworks/pdfcpu/pkg/filter"
	"github.com/trussworks/pdfcpu/pkg/log"
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
func encodeStream(sd *PDFStreamDict) error {

	log.Debug.Printf("encodeStream begin")

	// No filter specified, nothing to encode.
	if sd.FilterPipeline == nil {
		log.Debug.Println("encodeStream: returning uncompressed stream.")
		sd.Raw = sd.Content
		streamLength := int64(len(sd.Raw))
		sd.StreamLength = &streamLength
		sd.Insert("Length", PDFInteger(streamLength))
		return nil
	}

	var b io.Reader
	b = bytes.NewReader(sd.Content)

	var c *bytes.Buffer

	// Apply each filter in the pipeline to result of preceding filter.
	for _, f := range sd.FilterPipeline {

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

	sd.Raw = c.Bytes()

	streamLength := int64(len(sd.Raw))
	sd.StreamLength = &streamLength
	sd.Insert("Length", PDFInteger(streamLength))

	log.Debug.Printf("encodeStream end")

	return nil
}

// decodeStream decodes streamDict data by applying its filter pipeline.
func decodeStream(sd *PDFStreamDict) error {

	log.Debug.Printf("decodeStream begin \n%s\n", sd)

	if sd.Content != nil {
		// This stream has already been decoded.
		return nil
	}

	// No filter specified, nothing to decode.
	if sd.FilterPipeline == nil {
		sd.Content = sd.Raw
		//fmt.Printf("decodedStream returning %d(#%02x)bytes: \n%s\n", len(sd.Content), len(sd.Content), hex.Dump(sd.Content))
		return nil
	}

	var b io.Reader
	b = bytes.NewReader(sd.Raw)

	//fmt.Printf("decodedStream before:\n%s\n", hex.Dump(sd.Raw))

	var c *bytes.Buffer

	// Apply each filter in the pipeline to result of preceding filter.
	for _, f := range sd.FilterPipeline {

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

		//fmt.Printf("decodedStream after:%s\n%s\n", f.Name, hex.Dump(c.Bytes()))

		b = c
	}

	sd.Content = c.Bytes()

	//fmt.Printf("decodedStream returning %d(#%02x)bytes: \n%s\n", len(sd.Content), len(sd.Content), hex.Dump(c.Bytes()))

	log.Debug.Printf("decodeStream end")

	return nil
}
