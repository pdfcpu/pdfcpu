// Copyright 2026 The pdfcpu Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lzw

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestReaderDecodesPDFMSB(t *testing.T) {
	rc := NewReader(strings.NewReader("\x80\x0b\x60\x50\x22\x0c\x0c\x85\x01"), true)
	defer rc.Close()

	var b bytes.Buffer
	if _, err := io.Copy(&b, rc); err != nil {
		t.Fatal(err)
	}
	if got, want := b.String(), "-----A---B"; got != want {
		t.Fatalf("decoded bytes = %q, want %q", got, want)
	}
}

func TestEarlyChangeRoundTrip(t *testing.T) {
	payload := bytes.Repeat([]byte("pdfcpu LZW EarlyChange regression data "), 512)
	for _, earlyChange := range []bool{false, true} {
		testEarlyChangeRoundTrip(t, payload, earlyChange)
	}
}

func testEarlyChangeRoundTrip(t *testing.T, payload []byte, earlyChange bool) {
	t.Helper()

	var enc bytes.Buffer
	wc := NewWriter(&enc, earlyChange)
	if _, err := wc.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := wc.Close(); err != nil {
		t.Fatal(err)
	}

	rc := NewReader(&enc, earlyChange)
	defer rc.Close()

	var dec bytes.Buffer
	if _, err := io.Copy(&dec, rc); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dec.Bytes(), payload) {
		t.Fatalf("earlyChange=%t round trip mismatch", earlyChange)
	}
}
