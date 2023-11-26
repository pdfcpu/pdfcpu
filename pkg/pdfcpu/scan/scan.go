/*
Copyright 2023 The pdfcpu Authors.

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

package scan

import "bytes"

// Lines is a split function for a Scanner that returns each line of
// text, stripped of any trailing end-of-line marker. The returned line may
// be empty. The end-of-line marker is one carriage return followed
// by one newline or one carriage return or one newline.
// The last non-empty line of input will be returned even if it has no newline.
func Lines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	indCR := bytes.IndexByte(data, '\r')
	indLF := bytes.IndexByte(data, '\n')

	switch {

	case indCR >= 0 && indLF >= 0:
		if indCR < indLF {
			if indLF == indCR+1 {
				// 0x0D0A
				return indLF + 1, data[0:indCR], nil
			}
			// 0x0D ... 0x0A
			return indCR + 1, data[0:indCR], nil
		}
		// 0x0A ... 0x0D
		return indLF + 1, data[0:indLF], nil

	case indCR >= 0:
		// We have a full carriage return terminated line.
		return indCR + 1, data[0:indCR], nil

	case indLF >= 0:
		// We have a full newline-terminated line.
		return indLF + 1, data[0:indLF], nil

	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}

	// Request more data.
	return 0, nil, nil
}

func LinesForSingleEol(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	indCR := bytes.IndexByte(data, '\r')
	indLF := bytes.IndexByte(data, '\n')

	switch {

	case indCR >= 0 && indLF >= 0:
		if indCR < indLF {
			// 0x0D ... 0x0A
			return indCR + 2, data[0:indCR], nil
		}
		// 0x0A ... 0x0D
		return indLF + 2, data[0:indLF], nil

	case indCR >= 0:
		// We have a full carriage return terminated line.
		return indCR + 1, data[0:indCR], nil

	case indLF >= 0:
		// We have a full newline-terminated line.
		return indLF + 1, data[0:indLF], nil

	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}

	// Request more data.
	return 0, nil, nil
}
