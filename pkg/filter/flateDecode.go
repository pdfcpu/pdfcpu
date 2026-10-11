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
	stdflate "compress/flate"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/safemath"
)

// Portions of this code are based on ideas of image/png: reader.go:readImagePass
// PNG is documented here: www.w3.org/TR/PNG-Filters.html

// PDF allows a prediction step prior to compression applying TIFF or PNG prediction.
// Predictor algorithm.
const (
	// PredictorNo disables prediction.
	PredictorNo = 1

	// PredictorTIFF applies TIFF prediction to every row.
	PredictorTIFF = 2

	// PredictorNone applies the PNG none filter to every row.
	PredictorNone = 10

	// PredictorSub applies the PNG sub filter to every row.
	PredictorSub = 11

	// PredictorUp applies the PNG up filter to every row.
	PredictorUp = 12

	// PredictorAverage applies the PNG average filter to every row.
	PredictorAverage = 13

	// PredictorPaeth applies the PNG Paeth filter to every row.
	PredictorPaeth = 14

	// PredictorOptimum selects the optimum PNG filter for each row.
	PredictorOptimum = 15
)

// For predictor > 2 PNG filters (see RFC 2083) get applied and the first byte of each pixelrow defines
// the prediction algorithm used for all pixels of this row.
const (
	// PNGNone identifies the PNG none filter.
	PNGNone = 0x00

	// PNGSub identifies the PNG sub filter.
	PNGSub = 0x01

	// PNGUp identifies the PNG up filter.
	PNGUp = 0x02

	// PNGAverage identifies the PNG average filter.
	PNGAverage = 0x03

	// PNGPaeth identifies the PNG Paeth filter.
	PNGPaeth = 0x04
)

type flate struct {
	baseFilter
}

type zlibReadResetCloser interface {
	io.ReadCloser
	zlib.Resetter
}

type zlibErrorTrackingReader struct {
	r   io.Reader
	err error
}

var (
	zlibWriterPool sync.Pool
	zlibReaderPool sync.Pool
)

func (r *zlibErrorTrackingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if err != nil && err != io.EOF && r.err == nil {
		r.err = err
	}
	return n, err
}

func acquireZlibWriter(w io.Writer) *zlib.Writer {
	if v := zlibWriterPool.Get(); v != nil {
		zw := v.(*zlib.Writer)
		zw.Reset(w)
		return zw
	}
	return zlib.NewWriter(w)
}

func releaseZlibWriter(w *zlib.Writer, reusable bool) error {
	err := w.Close()
	if reusable && err == nil {
		zlibWriterPool.Put(w)
	}
	return err
}

func acquireZlibReader(r io.Reader) (zlibReadResetCloser, error) {
	if v := zlibReaderPool.Get(); v != nil {
		rc := v.(zlibReadResetCloser)
		if err := rc.Reset(r, nil); err != nil {
			_ = rc.Close()
			return nil, err
		}
		return rc, nil
	}

	rc, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}
	rrc, ok := rc.(zlibReadResetCloser)
	if !ok {
		_ = rc.Close()
		return nil, errors.New("flate decode: zlib reader does not support reset")
	}
	return rrc, nil
}

func releaseZlibReader(r zlibReadResetCloser, reusable bool) {
	err := r.Close()
	if reusable && err == nil {
		zlibReaderPool.Put(r)
	}
}

// IsCorruptFlateInput reports whether err contains a Flate corrupt-input error.
func IsCorruptFlateInput(err error) bool {
	var corruptInputErr stdflate.CorruptInputError
	return errors.As(err, &corruptInputErr)
}

// Encode implements encoding for a Flate filter.
func (f flate) Encode(r io.Reader) (io.Reader, error) {
	if log.TraceEnabled() {
		log.Trace.Println("EncodeFlate begin")
	}

	var b bytes.Buffer
	w := acquireZlibWriter(&b)

	written, err := f.encodePreProcess(w, r)
	if err != nil {
		_ = releaseZlibWriter(w, false)
		return nil, err
	}
	if err := releaseZlibWriter(w, true); err != nil {
		return nil, err
	}

	if log.TraceEnabled() {
		log.Trace.Printf("EncodeFlate end: %d bytes written\n", written)
	}

	return &b, nil
}

// encodePreProcess applies the forward predictor before Flate compression (see 7.4.4.4).
func (f flate) encodePreProcess(w io.Writer, r io.Reader) (int64, error) {
	predictor, found := f.parms["Predictor"]
	if !found || predictor == PredictorNo {
		return io.Copy(w, r)
	}
	if err := validatePredictor(predictor); err != nil {
		return 0, err
	}
	colors, bpc, columns, err := f.parameters()
	if err != nil {
		return 0, err
	}
	if predictor == PredictorTIFF && bpc != 8 {
		return 0, fmt.Errorf("flate encode: TIFF predictor with BitsPerComponent %d unsupported", bpc)
	}
	rowSize, rowLen, bytesPerPixel, err := predictorRowParams(predictor, colors, bpc, columns)
	if err != nil {
		return 0, err
	}
	if limit := f.decodeLimit(-1); limit >= 0 && int64(rowLen) > limit {
		return 0, ErrDecodeLimitExceeded
	}
	return encodePredictorRows(w, r, predictor, colors, rowSize, rowLen, bytesPerPixel)
}

func encodePredictorRows(w io.Writer, r io.Reader, predictor, colors, rowSize, rowLen, bytesPerPixel int) (int64, error) {
	current := make([]byte, rowSize)
	previous := make([]byte, rowSize)
	encoded := make([]byte, rowLen)
	var candidate []byte
	if predictor == PredictorOptimum {
		candidate = make([]byte, rowSize)
	}
	var written int64
	for {
		n, err := io.ReadFull(r, current)
		if err == io.EOF {
			return written, nil
		}
		if err != nil {
			return written, fmt.Errorf("flate encode: incomplete predictor row: %w", err)
		}
		written += int64(n)
		if predictor == PredictorTIFF {
			encodeTIFFRow(encoded, current, colors)
		} else {
			filter := byte(predictor - PredictorNone)
			if predictor == PredictorOptimum {
				filter = bestPNGFilter(encoded[1:], candidate, current, previous, bytesPerPixel)
			} else {
				filterPNGRow(encoded[1:], current, previous, filter, bytesPerPixel)
			}
			encoded[0] = filter
		}
		n, err = w.Write(encoded)
		if err != nil {
			return written, err
		}
		if n != len(encoded) {
			return written, io.ErrShortWrite
		}
		current, previous = previous, current
	}
}

func encodeTIFFRow(dst, src []byte, colors int) {
	copy(dst, src)
	for i := len(src) - 1; i >= colors; i-- {
		dst[i] -= src[i-colors]
	}
}

func bestPNGFilter(dst, candidate, current, previous []byte, bytesPerPixel int) byte {
	var best byte
	bestScore := int64(-1)
	for filter := byte(PNGNone); filter <= PNGPaeth; filter++ {
		filterPNGRow(candidate, current, previous, filter, bytesPerPixel)
		var score int64
		for _, b := range candidate {
			score += int64(abs(int(int8(b))))
		}
		if bestScore < 0 || score < bestScore {
			best, bestScore = filter, score
			copy(dst, candidate)
		}
	}
	return best
}

func filterPNGRow(dst, current, previous []byte, filter byte, bytesPerPixel int) {
	for i, b := range current {
		var left, upperLeft byte
		if i >= bytesPerPixel {
			left = current[i-bytesPerPixel]
			upperLeft = previous[i-bytesPerPixel]
		}
		above := previous[i]
		switch filter {
		case PNGNone:
			dst[i] = b
		case PNGSub:
			dst[i] = b - left
		case PNGUp:
			dst[i] = b - above
		case PNGAverage:
			dst[i] = b - byte((int(left)+int(above))/2)
		case PNGPaeth:
			dst[i] = b - paeth(left, above, upperLeft)
		}
	}
}

// Decode implements decoding for a Flate filter.
func (f flate) Decode(r io.Reader) (io.Reader, error) {
	return f.DecodeLength(r, -1)
}

// DecodeLength implements decoding for a Flate filter with a maximum output length.
func (f flate) DecodeLength(r io.Reader, maxLen int64) (io.Reader, error) {
	if err := f.checkDecodeLength(maxLen); err != nil {
		return nil, err
	}
	if log.TraceEnabled() {
		log.Trace.Println("DecodeFlate begin")
	}

	rc, err := acquireZlibReader(r)
	if err != nil {
		return nil, err
	}

	// Optional decode parameters need postprocessing.
	tr := zlibErrorTrackingReader{r: rc}
	out, err := f.decodePostProcess(&tr, maxLen)
	releaseZlibReader(rc, err == nil && tr.err == nil)
	return out, err
}

func (f flate) passThru(rin io.Reader, maxLen int64) (*bytes.Buffer, error) {
	b, err := f.copyDecoded(rin, maxLen)
	if errors.Is(err, zlib.ErrChecksum) {
		return b, fmt.Errorf("flate decode: %w", err)
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
		log.Read.Println("flate decode: ignoring unexpected EOF")
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

	return fmt.Errorf("flate decode: undefined \"Predictor\" %d", predictor)
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

// applyHorDiff undoes TIFF predictor 2 (horizontal differencing) on one row.
// Each colour component of a sample is the difference from the same component
// of the preceding sample, taken modulo 2^bpc on whole bpc-bit components
// (see 7.4.4.4). 16-bit components are big-endian;
// components narrower than a byte are packed MSB first and any padding bits at
// the end of the row are left untouched.
func applyHorDiff(row []byte, colors, bpc, columns int) ([]byte, error) {
	switch bpc {

	case 8:
		for i := colors; i < len(row); i++ {
			row[i] += row[i-colors]
		}

	case 16:
		for i := 2 * colors; i+1 < len(row); i += 2 {
			v := uint16(row[i])<<8 | uint16(row[i+1])
			v += uint16(row[i-2*colors])<<8 | uint16(row[i-2*colors+1])
			row[i], row[i+1] = byte(v>>8), byte(v)
		}

	default:
		// bpc 1, 2 or 4: components are packed within bytes.
		mask := byte(1<<bpc - 1)
		get := func(k int) byte {
			bit := k * bpc
			return row[bit/8] >> (8 - bpc - bit%8) & mask
		}
		n := colors * columns
		for k := colors; k < n; k++ {
			bit := k * bpc
			shift := 8 - bpc - bit%8
			v := (get(k) + get(k-colors)) & mask
			row[bit/8] = row[bit/8]&^(mask<<shift) | v<<shift
		}
	}

	return row, nil
}

func processRow(pr, cr []byte, p, colors, bpc, columns, bytesPerPixel int) ([]byte, error) {
	if p == PredictorTIFF {
		return applyHorDiff(cr, colors, bpc, columns)
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
		return nil, fmt.Errorf("flate decode: unexpected PNG predictor %d", f)
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
		return 0, 0, 0, fmt.Errorf("flate decode: \"Colors\" must be > 0")
	}

	// BitsPerComponent, int
	// The number of bits used to represent each colour component in a sample.
	// Valid values are 1, 2, 4, 8, and (PDF 1.5) 16. Default value: 8.
	// Used by PredictorTIFF only.
	bpc, found = f.parms["BitsPerComponent"]
	if !found {
		bpc = 8
	} else if !intMemberOf(bpc, []int{1, 2, 4, 8, 16}) {
		return 0, 0, 0, fmt.Errorf("flate decode: unexpected \"BitsPerComponent\": %d", bpc)
	}

	// Columns, int
	// The number of samples in each row. Default value: 1.
	columns, found = f.parms["Columns"]
	if !found {
		columns = 1
	} else if columns <= 0 {
		return 0, 0, 0, fmt.Errorf("flate decode: \"Columns\" must be > 0")
	}

	return colors, bpc, columns, nil
}

func checkBufLen(b bytes.Buffer, maxLen int64) bool {
	return maxLen < 0 || int64(b.Len()) < maxLen
}

func process(w io.Writer, pr, cr []byte, predictor, colors, bpc, columns, bytesPerPixel int, remaining int64) error {
	d, err := processRow(pr, cr, predictor, colors, bpc, columns, bytesPerPixel)
	if err != nil {
		return err
	}
	if remaining >= 0 && int64(len(d)) > remaining {
		d = d[:remaining]
	}

	_, err = w.Write(d)

	return err
}

func (f flate) decodePostProcessRows(r io.Reader, maxLen int64, m, predictor, colors, bpc, columns, bytesPerPixel int) (*bytes.Buffer, error) {
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
			return nil, fmt.Errorf("flate decode: read error, expected %d bytes, got: %d", m, n)
		}

		remaining := int64(-1)
		if maxLen >= 0 {
			remaining = maxLen - int64(b.Len())
		}
		if err := process(&b, pr, cr, predictor, colors, bpc, columns, bytesPerPixel, remaining); err != nil {
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

	b, err := f.decodePostProcessRows(r, maxLen, rowLen, predictor, colors, bpc, columns, bytesPerPixel)
	if err != nil {
		return nil, err
	}

	if maxLen < 0 && b.Len()%rowSize > 0 {
		log.Info.Printf("failed postprocessing: %d %d\n", b.Len(), rowSize)
		return nil, errors.New("flate decode: postprocessing failed")
	}

	return b, nil
}
