package filter

import (
	"bytes"
	"compress/zlib"
	"io"

	"github.com/hhrutter/pdfcpu/log"
	"github.com/pkg/errors"
)

var (
	errFlateMissingDecodeParmColumn    = errors.New("filter FlateDecode: missing decode parm: Columns")
	errFlateMissingDecodeParmPredictor = errors.New("filter FlateDecode: missing decode parm: Predictor")
	errFlatePostProcessing             = errors.New("filter FlateDecode: postprocessing failed")
)

type flate struct {
	baseFilter
}

// Encode implements encoding for a Flate filter.
func (f flate) Encode(r io.Reader) (*bytes.Buffer, error) {

	log.Debug.Println("EncodeFlate begin")

	// Optional decode parameters need preprocessing
	// but this filter implementation is used for object streams
	// and xref streams only and does not use decode parameters.

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	defer w.Close()

	written, err := io.Copy(w, r)
	if err != nil {
		return nil, err
	}
	log.Debug.Printf("EncodeFlate end: %d bytes written\n", written)

	return &b, nil
}

// Decode implements decoding for a Flate filter.
func (f flate) Decode(r io.Reader) (*bytes.Buffer, error) {

	log.Debug.Println("DecodeFlate begin")

	rc, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var b bytes.Buffer
	written, err := io.Copy(&b, rc)
	if err != nil {
		return nil, err
	}
	log.Debug.Printf("DecodeFlate: decoded %d bytes.\n", written)

	if len(f.parms) == 0 {
		log.Debug.Println("DecodeFlate end w/o decodeParms")
		return &b, nil
	}

	log.Debug.Println("DecodeFlate with decodeParms")

	// Optional decode parameters need postprocessing.
	return f.decodePostProcess(&b)
}

func passThru(rin io.Reader) (*bytes.Buffer, error) {
	var b bytes.Buffer
	_, err := io.Copy(&b, rin)
	return &b, err
}

// decodePostProcess
func (f flate) decodePostProcess(rin io.Reader) (*bytes.Buffer, error) {

	// The only postprocessing needed (for decoding object streams) is: PredictorUp with PngUp.

	const PredictorNo = 1
	const PredictorTIFF = 2
	const PredictorNone = 10 // implemented
	const PredictorSub = 11
	const PredictorUp = 12 // implemented
	const PredictorAverage = 13
	const PredictorPaeth = 14
	const PredictorOptimum = 15

	const PngNone = 0x00
	const PngSub = 0x01
	const PngUp = 0x02 // implemented
	const PngAverage = 0x03
	const PngPaeth = 0x04

	columns, found := f.parms["Columns"]
	if !found {
		return nil, errFlateMissingDecodeParmColumn
	}

	predictor, found := f.parms["Predictor"]
	if !found {
		// eg.
		// <BitsPerComponent, 8>
		// <Colors, 3>
		// <Columns, 1000>
		// TODO implement
		return passThru(rin)
		//return nil, errFlateMissingDecodeParmPredictor
	}

	if predictor == PredictorNone {
		return passThru(rin)
	}

	// PredictorUp is a popular predictor used for flate encoded stream dicts.
	if predictor != PredictorUp {
		return nil, errors.Errorf("Filter FlateDecode: Predictor %d unsupported", predictor)
	}

	// BitsPerComponent optional, integer: 1,2,4,8,16 (Default:8)
	// The number of bits used to represent each colour component in a sample.
	bpc, found := f.parms["BitsPerComponents"]
	if found {
		return nil, errors.Errorf("Filter FlateDecode: Unexpected \"BitsPerComponent\": %d", bpc)
	}

	// Colors, optional, integer: 1,2,3,4 (Default:1)
	// The number of interleaved colour components per sample.
	colors, found := f.parms["Colors"]
	if found {
		return nil, errors.Errorf("Filter FlateDecode: Unexpected \"Colors\": %d", colors)
	}

	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(rin)
	if err != nil {
		return nil, err
	}

	b := buf.Bytes()

	if len(b)%(columns+1) > 0 {
		return nil, errFlatePostProcessing
	}

	var fbuf []byte
	j := 0
	for i := 0; i < len(b); i += columns + 1 {
		if b[i] != PngUp {
			return nil, errFlatePostProcessing
		}
		fbuf = append(fbuf, b[i+1:i+columns+1]...)
		j++
	}

	bufOut := make([]byte, len(fbuf))
	for i := 0; i < len(fbuf); i += columns {
		for j := 0; j < columns; j++ {
			from := i - columns + j
			if from >= 0 {
				bufOut[i+j] = fbuf[i+j] + bufOut[i-columns+j]
			} else {
				bufOut[j] = fbuf[j]
			}
		}
	}

	return bytes.NewBuffer(bufOut), nil
}
