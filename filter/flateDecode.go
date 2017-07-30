package filter

import (
	"bytes"
	"compress/zlib"
	"io"
	"io/ioutil"

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

	logDebugFilter.Println("EncodeFlate begin")

	// TODO EncodeParams preprocessing missing.

	var b bytes.Buffer
	w := zlib.NewWriter(&b)

	p, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	logDebugFilter.Printf("EncodeFlate: read %d bytes\n", len(p))

	n, err := w.Write(p)
	if err != nil {
		return nil, err
	}
	logDebugFilter.Printf("EncodeFlate: %d bytes written\n", n)

	w.Close()

	logDebugFilter.Println("EncodeFlate end")

	return &b, nil
}

// Decode implements decoding for a Flate filter.
func (f flate) Decode(r io.Reader) (*bytes.Buffer, error) {

	logDebugFilter.Println("DecodeFlate begin")

	rc, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	written, err := io.Copy(&b, rc)
	if err != nil {
		return nil, err
	}
	logDebugFilter.Printf("DecodeFlate: decoded %d bytes.\n", written)

	rc.Close()

	if f.decodeParms == nil {
		logDebugFilter.Println("DecodeFlate end w/o decodeParms")
		return &b, nil
	}

	logDebugFilter.Println("DecodeFlate end w/o decodeParms")

	// If we have decode parms we need to do some postprocessing.
	return f.decodePostProcess(&b)
}

// decodePostProcess
// TODO Post processing for other predictors than PredictorUp.
// TODO Process parameters
// "Colors"(>=1), check Version, default=1
// "BitsPerComponent"(1,2,4,8,16). check Version, default=8
func (f flate) decodePostProcess(rin io.Reader) (a *bytes.Buffer, err error) {

	const PredictorNo = 1       // Don't use prediction - missing.
	const PredictorTIFF = 2     // all lines use TIFF predictor - missing.
	const PredictorNone = 10    // all lines have Png predictor 0x00 - missing.
	const PredictorSub = 11     // all lines have Png predictor 0x01 - missing.
	const PredictorUp = 12      // all lines have Png predictor 0x02 - implemented.
	const PredictorAverage = 13 // all lines have Png predictor 0x03 - missing.
	const PredictorPaeth = 14   // all lines have Png predictor 0x04 - missing.
	const PredictorOptimum = 15 // each line has individual predictor - missing.

	const PngNone = 0x00
	const PngSub = 0x01
	const PngUp = 0x02
	const PngAverage = 0x03
	const PngPaeth = 0x04

	c := f.decodeParms.IntEntry("Columns")
	if c == nil {
		err = errFlateMissingDecodeParmColumn
		return
	}

	columns := *c

	p := f.decodeParms.IntEntry("Predictor")
	if p == nil {
		err = errFlateMissingDecodeParmPredictor
		return
	}

	predictor := *p

	if predictor != PredictorUp {
		err = errors.Errorf("Filter FlateDecode: Predictor %d unsupported", predictor)
		return
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(rin)
	if err != nil {
		return
	}

	b := buf.Bytes()

	if len(b)%(columns+1) > 0 {
		return nil, errFlatePostProcessing
	}

	var fbuf []byte
	j := 0
	for i := 0; i < len(b); i += columns + 1 {
		if b[i] != PngUp {
			err = errFlatePostProcessing
			return
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
