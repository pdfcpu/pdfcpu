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

package filter

import (
	"bytes"
	stdflate "compress/flate"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/log"
)

// TestIsCorruptFlateInput verifies the corresponding behavior.
func TestIsCorruptFlateInput(t *testing.T) {
	err := fmt.Errorf("decode: %w", stdflate.CorruptInputError(7))
	if !IsCorruptFlateInput(err) {
		t.Fatalf("expected corrupt Flate input, got %v", err)
	}

	err = errors.New("flate: corrupt input before offset 7")
	if IsCorruptFlateInput(err) {
		t.Fatalf("expected text-only error not to match, got %v", err)
	}
}

func flateTestData(t *testing.T, s string) *bytes.Buffer {
	t.Helper()

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	if _, err := w.Write([]byte(s)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return &b
}

func flatePartialFlushTestData(t *testing.T, data []byte) *bytes.Buffer {
	t.Helper()

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	return &b
}

func flateChecksumFailureTestData(t *testing.T, data []byte) *bytes.Buffer {
	t.Helper()

	b := flateTestData(t, string(data))
	bb := append([]byte(nil), b.Bytes()...)
	bb[len(bb)-1] ^= 0xff
	return bytes.NewBuffer(bb)
}

// TestFlateChecksumFailurePreservesSentinelWithoutCLILogging verifies checksum errors are returned to the caller.
func TestFlateChecksumFailurePreservesSentinelWithoutCLILogging(t *testing.T) {
	var cliLog bytes.Buffer
	log.SetCLILogger(stdlog.New(&cliLog, "", 0))
	t.Cleanup(func() { log.SetCLILogger(nil) })

	_, err := (flate{}).Decode(flateChecksumFailureTestData(t, []byte("checksum")))
	if !errors.Is(err, zlib.ErrChecksum) {
		t.Fatalf("expected %v, got %v", zlib.ErrChecksum, err)
	}
	if !strings.Contains(err.Error(), "flate decode") {
		t.Fatalf("expected filter operation context, got %q", err.Error())
	}
	if cliLog.Len() != 0 {
		t.Fatalf("filter must not write user-facing output, got %q", cliLog.String())
	}
}

// TestFilterErrorsUseOperationContext verifies returned errors do not expose filter implementation identifiers.
func TestFilterErrorsUseOperationContext(t *testing.T) {
	tests := []struct {
		name       string
		want       string
		mechanical string
		decode     func() error
	}{
		{
			name:       "ASCII85",
			want:       "ASCII85 decode",
			mechanical: "ascii85Decode",
			decode: func() error {
				_, err := (ascii85Decode{}).Decode(bytes.NewReader(nil))
				return err
			},
		},
		{
			name:       "CCITT",
			want:       "CCITT decode",
			mechanical: "ccittDecode",
			decode: func() error {
				_, err := (ccittDecode{baseFilter{parms: map[string]int{}}}).Decode(bytes.NewReader(nil))
				return err
			},
		},
		{
			name:       "LZW",
			want:       "LZW decode",
			mechanical: "decodeLZW",
			decode: func() error {
				f := lzwDecode{baseFilter{parms: map[string]int{"Predictor": 2}}}
				_, err := f.Decode(bytes.NewReader(nil))
				return err
			},
		},
		{
			name:       "Flate",
			want:       "flate decode",
			mechanical: "flateDecode",
			decode: func() error {
				return validatePredictor(3)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.decode()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
			if strings.Contains(err.Error(), tt.mechanical) {
				t.Fatalf("unexpected implementation context %q in %q", tt.mechanical, err.Error())
			}
		})
	}
}

// TestFlatePredictorRejectsInvalidColors verifies invalid color counts are rejected.
func TestFlatePredictorRejectsInvalidColors(t *testing.T) {
	f := flate{baseFilter{parms: map[string]int{
		"Predictor": PredictorNone,
		"Colors":    -1,
	}}}

	_, err := f.Decode(flateTestData(t, ""))
	if err == nil || !strings.Contains(err.Error(), "Colors") {
		t.Fatalf("got %v, want Colors validation error", err)
	}
}

// TestFlatePredictorRejectsInvalidColumns verifies invalid column counts are rejected.
func TestFlatePredictorRejectsInvalidColumns(t *testing.T) {
	f := flate{baseFilter{parms: map[string]int{
		"Predictor": PredictorNone,
		"Columns":   -1,
	}}}

	_, err := f.Decode(flateTestData(t, ""))
	if err == nil || !strings.Contains(err.Error(), "Columns") {
		t.Fatalf("got %v, want Columns validation error", err)
	}
}

// TestFlatePredictorRejectsOverflowingRowSize verifies overflowing row sizes are rejected.
func TestFlatePredictorRejectsOverflowingRowSize(t *testing.T) {
	f := flate{baseFilter{parms: map[string]int{
		"Predictor": PredictorNone,
		"Colors":    maxInt,
		"Columns":   2,
	}}}

	_, err := f.Decode(flateTestData(t, ""))
	if err == nil || !strings.Contains(err.Error(), "integer overflow") {
		t.Fatalf("got %v, want integer overflow error", err)
	}
}

// TestFlatePredictorRejectsRowLargerThanDecodeLimit verifies row decode limits are enforced.
func TestFlatePredictorRejectsRowLargerThanDecodeLimit(t *testing.T) {
	f := flate{baseFilter{
		parms: map[string]int{
			"Predictor": PredictorNone,
			"Columns":   8,
		},
		maxDecodeBytes: 4,
	}}

	_, err := f.Decode(flateTestData(t, ""))
	if err != ErrDecodeLimitExceeded {
		t.Fatalf("got %v, want %v", err, ErrDecodeLimitExceeded)
	}
}

// TestFlatePredictorIgnoresUnexpectedEOF verifies predictor postprocessing tolerates partial flushes.
func TestFlatePredictorIgnoresUnexpectedEOF(t *testing.T) {
	f := flate{baseFilter{parms: map[string]int{
		"Predictor": PredictorNone,
		"Columns":   3,
	}}}

	r, err := f.Decode(flatePartialFlushTestData(t, []byte{PNGNone, 'f', 'o', 'o'}))
	if err != nil {
		t.Fatal(err)
	}

	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "foo" {
		t.Fatalf("got %q, want %q", b, "foo")
	}
}
