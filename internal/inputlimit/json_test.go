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

package inputlimit

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type countingReader struct {
	r io.Reader
	n int
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	r.n += n
	return n, err
}

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestReadJSONLimit(t *testing.T) {
	bb, err := readJSON(strings.NewReader("abc"), 3)
	if err != nil || string(bb) != "abc" {
		t.Fatalf("exact limit: got %q, %v", bb, err)
	}

	r := &countingReader{r: strings.NewReader("abcd")}
	if _, err := readJSON(r, 3); !errors.Is(err, ErrJSONSize) {
		t.Fatalf("overflow: got %v, want %v", err, ErrJSONSize)
	}
	if r.n != 4 {
		t.Fatalf("read %d bytes, want 4", r.n)
	}
}

func TestReadJSONPreservesReadError(t *testing.T) {
	wantErr := errors.New("read JSON")
	if _, err := readJSON(failingReader{err: wantErr}, 3); !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want %v", err, wantErr)
	}
}

func TestCheckJSONSize(t *testing.T) {
	if err := checkJSONSize(3, 3); err != nil {
		t.Fatalf("exact limit: %v", err)
	}
	if err := checkJSONSize(4, 3); !errors.Is(err, ErrJSONSize) {
		t.Fatalf("overflow: got %v, want %v", err, ErrJSONSize)
	}
}
