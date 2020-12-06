package filter

import (
	"bytes"
	"errors"
	"io"
)

// ByteReader allows both one byte and multi byte efficient
// reads.
type ByteReader interface {
	io.Reader
	io.ByteReader
}

// LimitedDCTDecoder return a Reader which do not read passed
// the end of the image.
// DCT is not among the supported decoders (since it would produce an Image)
// but this function provides an alternative to parse inline image data.
func LimitedDCTDecoder(input ByteReader) io.Reader {
	return &jpegLimitedReader{source: input, scratch: &bytes.Buffer{}}
}

// implement a reader which don't read passed the EOD
// we jump from markers to markers and save the data
// into a temporary buffer
// we follow the sketch described at
// https://stackoverflow.com/questions/4585527/detect-eof-for-jpg-images
type jpegLimitedReader struct {
	source  ByteReader
	scratch *bytes.Buffer // nil means EOD is reached
}

// read one byte from the source and stores it back to the buffer
func (j jpegLimitedReader) read() (byte, error) {
	c, err := j.source.ReadByte()
	if err != nil {
		return 0, unexpectedEOF(err)
	}
	j.scratch.WriteByte(c)
	return c, nil
}

func (j *jpegLimitedReader) Read(p []byte) (int, error) {
	if j.scratch == nil { // we have reached EOD
		return 0, io.EOF
	}

	// if our internal buffer is large enough, just return the data
	// and update the buffer
	n := len(p)
	if j.scratch.Len() >= n {
		return j.scratch.Read(p)
	}

	// start reading from the source
	c, err := j.read()
	if err != nil {
		return 0, err
	}
	for {
		if c == 0xff { // start of a marker or fill bytes
			next, err := j.read()
			if err != nil {
				return 0, err
			}

			if next == 0xD9 {
				// end of data: return the remaining buffer
				// and mark EOD by setting the buffer to nil
				out, err := j.scratch.Read(p)
				j.scratch = nil
				return out, err
			}

			if next == 0xff {
				// fill byte; we dont want to read another byte
				// since `next` could be the start of a marker
				c = next
				continue
			}

			if next == 0 || next == 1 || (0xD0 <= next && next <= 0xD8) {
				// standalone marker just, ignore it
			} else {
				lb1, err := j.read()
				if err != nil {
					return 0, err
				}

				lb2, err := j.read()
				if err != nil {
					return 0, err
				}

				segmentLength := int(uint16(lb2) | uint16(lb1)<<8)
				// the length includes the two-byte length
				if segmentLength < 2 {
					return 0, errors.New("corrupted JPEG data")
				}

				// jump to the next segment and store the data
				_, err = io.CopyN(j.scratch, j.source, int64(segmentLength-2))
				if err != nil {
					return 0, unexpectedEOF(err)
				}
			}
		}

		// if we now have enough bytes, return them ...
		if j.scratch.Len() >= n {
			return j.scratch.Read(p)
		}

		// ... else advance and loop
		c, err = j.read()
		if err != nil {
			return 0, err
		}
	}
}
