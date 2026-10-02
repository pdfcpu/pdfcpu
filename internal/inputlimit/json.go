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

// Package inputlimit provides fixed limits for non-PDF input consumed by pdfcpu.
package inputlimit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

const maxJSONBytes int64 = 64 << 20

// ErrJSONSize reports a JSON input exceeding the fixed byte limit.
var ErrJSONSize = errors.New("JSON input size limit exceeded")

// CheckJSONSize checks a JSON input size against the fixed byte limit.
func CheckJSONSize(size int64) error {
	return checkJSONSize(size, maxJSONBytes)
}

// ReadJSON reads a JSON input while enforcing the fixed byte limit.
func ReadJSON(r io.Reader) ([]byte, error) {
	return readJSON(r, maxJSONBytes)
}

func checkJSONSize(size, limit int64) error {
	if size > limit {
		return fmt.Errorf("maximum %d bytes: %w", limit, ErrJSONSize)
	}
	return nil
}

func readJSON(r io.Reader, limit int64) ([]byte, error) {
	var buf bytes.Buffer
	_, err := io.CopyN(&buf, r, limit)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return buf.Bytes(), nil
		}
		return nil, err
	}

	var probe [1]byte
	n, err := io.ReadFull(r, probe[:])
	if n > 0 {
		return nil, fmt.Errorf("maximum %d bytes: %w", limit, ErrJSONSize)
	}
	if errors.Is(err, io.EOF) {
		return buf.Bytes(), nil
	}
	return nil, err
}
