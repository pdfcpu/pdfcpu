/*
Copyright 2026 The pdfcpu Authors.

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

package pkcs7

import "fmt"

// scanBERContent checks whether a constructed object's content can be copied without rewriting lengths.
// It validates every nested object without allocating a tree, preserving the parser's depth and boundary checks.
func scanBERContent(ber []byte, offset, contentEnd, depth int) (bool, error) {
	for offset < contentEnd {
		next, unchanged, err := scanBERObject(ber, offset, contentEnd, depth)
		if err != nil {
			return false, fmt.Errorf("read child at offset %d: %w", offset, err)
		}
		if !unchanged {
			return false, nil
		}
		offset = next
	}
	return true, nil
}

func scanBERObject(ber []byte, offset, parentEnd, depth int) (int, bool, error) {
	start := offset
	tagStart, tagEnd, constructed, next, err := readTag(ber, offset)
	if err != nil {
		return 0, false, fmt.Errorf("read tag at offset %d: %w", start, err)
	}
	length, indefinite, offset, err := readLength(ber, next)
	if err != nil {
		return 0, false, fmt.Errorf("read length at offset %d: %w", next, err)
	}
	if err := validateBERContentBounds(ber, tagStart, tagEnd, offset, length, indefinite, constructed); err != nil {
		return 0, false, err
	}
	if indefinite {
		return 0, false, nil
	}
	end := offset + length
	if end > parentEnd {
		return 0, false, fmt.Errorf("child at offset %d exceeds parent boundary %d", start, parentEnd)
	}
	if constructed && depth >= maxBERDepth {
		return 0, false, fmt.Errorf("BER depth exceeds limit %d", maxBERDepth)
	}
	// readLength already rejects leading zeros in long lengths; only short values can still need rewriting.
	if length < 128 && offset-next != 1 {
		return end, false, nil
	}
	if constructed {
		unchanged, err := scanBERContent(ber, offset, end, depth+1)
		return end, unchanged, err
	}
	return end, true, nil
}
