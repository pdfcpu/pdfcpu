package filter

import (
	"bytes"
	"io"
)

// only read until delim is reached (including delim)
// so that the number of bytes read is a reliable way
// to detect End Of Data of filtered content.
type reacher struct {
	source io.ByteReader
	// the current matching bytes
	// the invariant is that
	// matched = delim[...:n] = (with n = length(matched))
	// and matched are the last n bytes read (possibly 0)
	matched []byte
	delim   []byte // the target to find
}

// delim must not be empty
func newReacher(source io.ByteReader, delim []byte) *reacher {
	return &reacher{source: source, delim: delim, matched: make([]byte, 0, len(delim))}
}

func (r *reacher) Read(out []byte) (int, error) {
	i := 0
	for ; i < len(out); i++ {
		// we have found the match, do no read anymore
		if len(r.matched) == len(r.delim) {
			return i, io.EOF
		}

		next, err := r.source.ReadByte()
		if err != nil {
			return 0, err
		}

		// check for overlappings, starting with the biggest possible
		r.matched = append(r.matched, next) // to avoid allocations
		for j := 0; j <= len(r.matched); j++ {
			if bytes.HasPrefix(r.delim, r.matched[j:]) {
				r.matched = r.matched[j:]
				break
			}
			// when i == len(r.matched) - 1 , HasPrefix is true
			// and r.matched = []
		}
		out[i] = next
	}
	return i, nil
}
