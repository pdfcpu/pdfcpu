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

package pdfcpu

import (
	"io"
	"os"
	"testing"
)

type smallbufferreader struct {
	rs io.ReadSeeker
}

// Read reads up to 10 bytes into p. It returns the number of bytes read (0 <= n <= 10) and any error encountered.
func (sm *smallbufferreader) Read(p []byte) (int, error) {
	b := make([]byte, 10)
	n, err := sm.rs.Read(b)
	copy(p, b)
	return n, err
}

func (sm *smallbufferreader) Seek(offset int64, whence int) (int64, error) {
	return sm.rs.Seek(offset, whence)
}

func TestReadBufferNotFilled(t *testing.T) {
	fp := "../testdata/go.pdf"
	f, err := os.Open(fp)
	if err != nil {
		t.Fatal("couldn't open test pdf: ", fp)
	}
	defer f.Close()

	s := &smallbufferreader{f}

	_, err = Read(s, nil)
	if err != nil {
		t.Fatal("couldn't create pdf context: ", err)
	}
}
