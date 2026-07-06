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

package filter

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/safemath"
)

// Portions of this code are based on ideas of image/png: reader.go:readImagePass
// PNG is documented here: www.w3.org/TR/PNG-Filters.html

// PDF allows a prediction step prior to compression applying TIFF or PNG prediction.
// Predictor algorithm.
const (
	PredictorNo      = 1  // No prediction.
	PredictorTIFF    = 2  // Use TIFF prediction for all rows.
	PredictorNone    = 10 // Use PNGNone for all rows.
	PredictorSub     = 11 // Use PNGSub for all rows.
	PredictorUp      = 12 // Use PNGUp for all rows.
	PredictorAverage = 13 // Use PNGAverage for all rows.
	PredictorPaeth   = 14 // Use PNGPaeth for all rows.
	PredictorOptimum = 15 // Use the optimum PNG prediction for each row.
)

// For predictor > 2 PNG filters (see RFC 2083) get applied and the first byte of each pixelrow defines
// the prediction algorithm used for all pixels of this row.
const (
	PNGNone    = 0x00
	PNGSub     = 0x01
	PNGUp      = 0x02
	PNGAverage = 0x03
	PNGPaeth   = 0x04
)

type flate struct {
	baseFilter
}

// Encode implements encoding for a Flate filter.
func (f flate) Encode(r io.Reader) (io.Reader, error) {
	if log.TraceEnabled() {
		log.Trace.Println("EncodeFlate begin")
	}

	// TODO Optional decode parameters may need predictor preprocessing.

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	defer w.Close()

	written, err := io.Copy(w, r)
	if err != nil {
		return nil, err
	}

	if log.TraceEnabled() {
		log.Trace.Printf("EncodeFlate end: %d bytes written\n", written)
	}

	return &b, nil
}

// Decode implements decoding for a Flate filter.
func (f flate) Decode(r io.Reader) (io.Reader, error) {
	return f.DecodeLength(r, -1)
}

// DecodeLength implements decoding for a Flate filter with a maximum output length.
func (f flate) DecodeLength(r io.Reader, maxLen int64) (io.Reader, error) {
	if log.TraceEnabled() {
		log.Trace.Println("DecodeFlate begin")
	}

	rc, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	// Optional decode parameters need postprocessing.
	return f.decodePostProcess(rc, maxLen)
}

func (f flate) passThru(rin io.Reader, maxLen int64) (*bytes.Buffer, error) {
	b, err := f.copyDecoded(rin, maxLen)
	if err != nil && strings.Contains(err.Error(), "invalid checksum") {
		if log.CLIEnabled() {
			log.CLI.Println("skipped: truncated zlib stream")
		}
		err = nil
	}
	if err == io.ErrUnexpectedEOF {
		logUnexpectedEOFFlateDecode()
		err = nil
	}
	return b, err
}

func logUnexpectedEOFFlateDecode() {
	// Workaround for missing support for partial flush in compress/flate.
	// See also https://github.com/golang/go/issues/31514
	if log.ReadEnabled() {
		log.Read.Println("flateDecode: ignoring unexpected EOF")
	}
}

func intMemberOf(i int, list []int) bool {
	return slices.Contains(list, i)
}

func validatePredictor(predictor int) error {
	if intMemberOf(
		predictor,
		[]int{PredictorTIFF,
			PredictorNone,
			PredictorSub,
			PredictorUp,
			PredictorAverage,
			PredictorPaeth,
			PredictorOptimum,
		}) {
		return nil
	}

	return fmt.Errorf("flateDecode: undefined \"Predictor\" %d", predictor)
}

func predictorRowParams(predictor, colors, bpc, columns int) (rowSize, rowLen, bytesPerPixel int, err error) {
	bitsPerPixel, err := safemath.MultiplyInt(bpc, colors)
	if err != nil {
		return 0, 0, 0, err
	}
	bitsPerPixelRounded, err := safemath.AddInt(bitsPerPixel, 7)
	if err != nil {
		return 0, 0, 0, err
	}
	bytesPerPixel = bitsPerPixelRounded / 8

	rowBits, err := safemath.MultiplyInt(bitsPerPixel, columns)
	if err != nil {
		return 0, 0, 0, err
	}
	rowBitsRounded, err := safemath.AddInt(rowBits, 7)
	if err != nil {
		return 0, 0, 0, err
	}
	rowSize = rowBitsRounded / 8

	rowLen = rowSize
	if predictor != PredictorTIFF {
		// PNG prediction uses a row filter byte prefixing the pixelbytes of a row.
		rowLen, err = safemath.AddInt(rowLen, 1)
		if err != nil {
			return 0, 0, 0, err
		}
	}

	return rowSize, rowLen, bytesPerPixel, nil
}

// Each prediction value implies (a) certain row filter(s).
// func validateRowFilter(f, p int) error {

// 	switch p {

// 	case PredictorNone:
// 		if !intMemberOf(f, []int{PNGNone, PNGSub, PNGUp, PNGAverage, PNGPaeth}) {
// 			return fmt.Errorf("pdfcpu: validateRowFilter: PredictorOptimum, unexpected row filter #%02x", f)
// 		}
// 		// if f != PNGNone {
// 		// 	return fmt.Errorf("validateRowFilter: expected row filter #%02x, got: #%02x", PNGNone, f)
// 		// }

// 	case PredictorSub:
// 		if f != PNGSub {
// 			return fmt.Errorf("pdfcpu: validateRowFilter: expected row filter #%02x, got: #%02x", PNGSub, f)
// 		}

// 	case PredictorUp:
// 		if f != PNGUp {
// 			return fmt.Errorf("pdfcpu: validateRowFilter: expected row filter #%02x, got: #%02x", PNGUp, f)
// 		}

// 	case PredictorAverage:
// 		if f != PNGAverage {
// 			return fmt.Errorf("pdfcpu: validateRowFilter: expected row filter #%02x, got: #%02x", PNGAverage, f)
// 		}

// 	case PredictorPaeth:
// 		if f != PNGPaeth {
// 			return fmt.Errorf("pdfcpu: validateRowFilter: expected row filter #%02x, got: #%02x", PNGPaeth, f)
// 		}

// 	case PredictorOptimum:
// 		if !intMemberOf(f, []int{PNGNone, PNGSub, PNGUp, PNGAverage, PNGPaeth}) {
// 			return fmt.Errorf("pdfcpu: validateRowFilter: PredictorOptimum, unexpected row filter #%02x", f)
// 		}

// 	default:
// 		return fmt.Errorf("pdfcpu: validateRowFilter: unexpected predictor #%02x", p)

// 	}

// 	return nil
// }

func applyHorDiff(row []byte, colors int) ([]byte, error) {
	// This works for 8 bits per color only.
	for i := 1; i < len(row)/colors; i++ {
		for j := range colors {
			row[i*colors+j] += row[(i-1)*colors+j]
		}
	}
	return row, nil
}

func processRow(pr, cr []byte, p, colors, bytesPerPixel int) ([]byte, error) {
	//fmt.Printf("pr(%v) =\n%s\n", &pr, hex.Dump(pr))
	//fmt.Printf("cr(%v) =\n%s\n", &cr, hex.Dump(cr))

	if p == PredictorTIFF {
		return applyHorDiff(cr, colors)
	}

	// Apply the filter.
	cdat := cr[1:]
	pdat := pr[1:]

	// Get row filter from 1st byte
	f := int(cr[0])

	// The value of Predictor supplied by the decoding filter need not match the value
	// used when the data was encoded if they are both greater than or equal to 10.

	switch f {

	case PNGNone:
		// No operation.

	case PNGSub:
		for i := bytesPerPixel; i < len(cdat); i++ {
			cdat[i] += cdat[i-bytesPerPixel]
		}

	case PNGUp:
		for i, p := range pdat {
			cdat[i] += p
		}

	case PNGAverage:
		// The average of the two neighboring pixels (left and above).
		// Raw(x) - floor((Raw(x-bpp)+Prior(x))/2)
		for i := range bytesPerPixel {
			cdat[i] += pdat[i] / 2
		}
		for i := bytesPerPixel; i < len(cdat); i++ {
			cdat[i] += uint8((int(cdat[i-bytesPerPixel]) + int(pdat[i])) / 2)
		}

	case PNGPaeth:
		filterPaeth(cdat, pdat, bytesPerPixel)

	default:
		return nil, fmt.Errorf("flateDecode: unexpected PNG predictor %d", f)
	}

	return cdat, nil
}

func (f flate) parameters() (colors, bpc, columns int, err error) {
	// Colors, int
	// The number of interleaved colour components per sample.
	// Valid values are 1 to 4 (PDF 1.0) and 1 or greater (PDF 1.3). Default value: 1.
	// Used by PredictorTIFF only.
	colors, found := f.parms["Colors"]
	if !found {
		colors = 1
	} else if colors <= 0 {
		return 0, 0, 0, fmt.Errorf("flateDecode: \"Colors\" must be > 0")
	}

	// BitsPerComponent, int
	// The number of bits used to represent each colour component in a sample.
	// Valid values are 1, 2, 4, 8, and (PDF 1.5) 16. Default value: 8.
	// Used by PredictorTIFF only.
	bpc, found = f.parms["BitsPerComponent"]
	if !found {
		bpc = 8
	} else if !intMemberOf(bpc, []int{1, 2, 4, 8, 16}) {
		return 0, 0, 0, fmt.Errorf("flateDecode: unexpected \"BitsPerComponent\": %d", bpc)
	}

	// Columns, int
	// The number of samples in each row. Default value: 1.
	columns, found = f.parms["Columns"]
	if !found {
		columns = 1
	} else if columns <= 0 {
		return 0, 0, 0, fmt.Errorf("flateDecode: \"Columns\" must be > 0")
	}

	return colors, bpc, columns, nil
}

func checkBufLen(b bytes.Buffer, maxLen int64) bool {
	return maxLen < 0 || int64(b.Len()) < maxLen
}

func process(w io.Writer, pr, cr []byte, predictor, colors, bytesPerPixel int) error {
	d, err := processRow(pr, cr, predictor, colors, bytesPerPixel)
	if err != nil {
		return err
	}

	_, err = w.Write(d)

	return err
}

func (f flate) decodePostProcessRows(r io.Reader, maxLen int64, m, predictor, colors, bytesPerPixel int) (*bytes.Buffer, error) {
	// cr and pr are the bytes for the current and previous row.
	cr := make([]byte, m)
	pr := make([]byte, m)

	// Output buffer
	var b bytes.Buffer

	for checkBufLen(b, maxLen) {

		// Read decompressed bytes for one pixel row.
		n, err := io.ReadFull(r, cr)
		if err != nil {
			if err == io.ErrUnexpectedEOF && n == 0 {
				logUnexpectedEOFFlateDecode()
				break
			}
			if err != io.EOF {
				return nil, err
			}
			// eof
			if n == 0 {
				break
			}
		}

		if n != m {
			return nil, fmt.Errorf("flateDecode: read error, expected %d bytes, got: %d", m, n)
		}

		if err := process(&b, pr, cr, predictor, colors, bytesPerPixel); err != nil {
			return nil, err
		}

		if maxLen < 0 {
			if limit := f.decodeLimit(maxLen); limit >= 0 && int64(b.Len()) > limit {
				return nil, ErrDecodeLimitExceeded
			}
		}

		if err == io.EOF {
			break
		}

		pr, cr = cr, pr
	}

	return &b, nil
}

func (f flate) decodePostProcess(r io.Reader, maxLen int64) (io.Reader, error) {
	predictor, found := f.parms["Predictor"]
	if !found || predictor == PredictorNo {
		return f.passThru(r, maxLen)
	}

	if err := validatePredictor(predictor); err != nil {
		return nil, err
	}

	colors, bpc, columns, err := f.parameters()
	if err != nil {
		return nil, err
	}

	rowSize, rowLen, bytesPerPixel, err := predictorRowParams(predictor, colors, bpc, columns)
	if err != nil {
		return nil, err
	}

	if limit := f.decodeLimit(-1); limit >= 0 && int64(rowLen) > limit {
		return nil, ErrDecodeLimitExceeded
	}

	b, err := f.decodePostProcessRows(r, maxLen, rowLen, predictor, colors, bytesPerPixel)
	if err != nil {
		return nil, err
	}

	if maxLen < 0 && b.Len()%rowSize > 0 {
		log.Info.Printf("failed postprocessing: %d %d\n", b.Len(), rowSize)
		return nil, errors.New("flateDecode: postprocessing failed")
	}

	return b, nil
}
