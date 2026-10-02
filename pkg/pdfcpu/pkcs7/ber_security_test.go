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

import (
	"bytes"
	"strings"
	"testing"
)

func (o failingASN1Object) encodedLen() (int, error) {
	return 2, nil
}

func nestedSequenceBER(t *testing.T, depth int, indefinite bool) []byte {
	t.Helper()
	ber := []byte{0x02, 0x01, 0x01}
	for range depth {
		var outer bytes.Buffer
		outer.WriteByte(0x30)
		if indefinite {
			outer.WriteByte(0x80)
		} else {
			if err := encodeLength(&outer, len(ber)); err != nil {
				t.Fatal(err)
			}
		}
		outer.Write(ber)
		if indefinite {
			outer.Write([]byte{0, 0})
		}
		ber = outer.Bytes()
	}
	return ber
}

func TestBERConstructedDepthLimit(t *testing.T) {
	for _, tt := range []struct {
		name       string
		indefinite bool
	}{
		{"definite", false},
		{"indefinite", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ber2der(nestedSequenceBER(t, maxBERDepth, tt.indefinite)); err != nil {
				t.Fatalf("depth %d: %v", maxBERDepth, err)
			}
			_, err := ber2der(nestedSequenceBER(t, maxBERDepth+1, tt.indefinite))
			if err == nil || !strings.Contains(err.Error(), "BER depth exceeds limit 64") {
				t.Fatalf("got %v, want BER depth limit error", err)
			}
		})
	}
}

func TestBERObjectCountLimit(t *testing.T) {
	ber := make([]byte, 0, 4+maxBERObjects*2)
	ber = append(ber, 0x30, 0x80)
	for range maxBERObjects {
		ber = append(ber, 0x05, 0x00)
	}
	ber = append(ber, 0, 0)
	_, err := ber2der(ber)
	if err == nil || !strings.Contains(err.Error(), "BER object count exceeds limit") {
		t.Fatalf("got %v, want BER object count limit error", err)
	}
}

func TestDEROutputSizeLimit(t *testing.T) {
	if n, err := encodedObjectLength(1, maxBERBytes-6); err != nil || n != maxBERBytes {
		t.Fatalf("got length %d, error %v, want limit %d", n, err, maxBERBytes)
	}
	if _, err := encodedObjectLength(1, maxBERBytes-5); err == nil {
		t.Fatal("expected DER output size limit error")
	}
}
