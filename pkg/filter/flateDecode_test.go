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
	"sync"
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

func flateDecodedString(t *testing.T, f flate, r io.Reader) string {
	t.Helper()

	decoded, err := f.Decode(r)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(decoded)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type flateFailingReader struct {
	r   io.Reader
	err error
}

func (r *flateFailingReader) Read(p []byte) (int, error) {
	if r.r != nil {
		n, err := r.r.Read(p)
		if err != io.EOF {
			return n, err
		}
		r.r = nil
		if n > 0 {
			return n, nil
		}
	}
	return 0, r.err
}

// TestFlateRepeatedReuse verifies repeated encoding and decoding remain independent.
func TestFlateRepeatedReuse(t *testing.T) {
	f := flate{}
	for i := range 100 {
		want := fmt.Sprintf("payload-%d-%s", i, strings.Repeat("x", i%31))
		encoded, err := f.Encode(strings.NewReader(want))
		if err != nil {
			t.Fatal(err)
		}
		if got := flateDecodedString(t, f, encoded); got != want {
			t.Fatalf("iteration %d: got %q, want %q", i, got, want)
		}
	}
}

// TestFlateConcurrentReuse verifies concurrent encoding and decoding through the shared pools.
func TestFlateConcurrentReuse(t *testing.T) {
	const (
		workers    = 8
		iterations = 50
	)

	errCh := make(chan error, workers)
	var wg sync.WaitGroup
	for worker := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f := flate{}
			for iteration := range iterations {
				want := fmt.Sprintf("worker-%d-iteration-%d", worker, iteration)
				encoded, err := f.Encode(strings.NewReader(want))
				if err != nil {
					errCh <- err
					return
				}
				decoded, err := f.Decode(encoded)
				if err != nil {
					errCh <- err
					return
				}
				got, err := io.ReadAll(decoded)
				if err != nil {
					errCh <- err
					return
				}
				if string(got) != want {
					errCh <- fmt.Errorf("got %q, want %q", got, want)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}

// TestFlateReaderFailuresDoNotAffectValidDecoding verifies failed readers are not reused with stale state.
func TestFlateReaderFailuresDoNotAffectValidDecoding(t *testing.T) {
	f := flate{}
	tests := []struct {
		name    string
		data    io.Reader
		wantErr error
	}{
		{name: "malformed header", data: strings.NewReader("not a zlib stream"), wantErr: zlib.ErrHeader},
		{
			name:    "checksum failure",
			data:    flateChecksumFailureTestData(t, []byte("checksum")),
			wantErr: zlib.ErrChecksum,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := f.Decode(tt.data); !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
			if got := flateDecodedString(t, f, flateTestData(t, "valid")); got != "valid" {
				t.Fatalf("got %q, want %q", got, "valid")
			}
		})
	}
}

// TestFlateWriterFailureDoesNotAffectValidEncoding verifies a failed input read does not poison later writers.
func TestFlateWriterFailureDoesNotAffectValidEncoding(t *testing.T) {
	wantErr := errors.New("read failure")
	f := flate{}
	_, err := f.Encode(&flateFailingReader{r: strings.NewReader("partial"), err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want %v", err, wantErr)
	}

	encoded, err := f.Encode(strings.NewReader("valid"))
	if err != nil {
		t.Fatal(err)
	}
	if got := flateDecodedString(t, f, encoded); got != "valid" {
		t.Fatalf("got %q, want %q", got, "valid")
	}
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

	if got := flateDecodedString(t, f, flateTestData(t, string([]byte{PNGNone, 'b', 'a', 'r'}))); got != "bar" {
		t.Fatalf("got %q, want %q", got, "bar")
	}
}

// BenchmarkFlateWriterReuse compares pooled writers with newly allocated writers.
func BenchmarkFlateWriterReuse(b *testing.B) {
	data := bytes.Repeat([]byte("pdfcpu flate writer benchmark "), 32)

	b.Run("pooled", func(b *testing.B) {
		var dst bytes.Buffer
		b.ReportAllocs()
		for b.Loop() {
			dst.Reset()
			w := acquireZlibWriter(&dst)
			if _, err := w.Write(data); err != nil {
				b.Fatal(err)
			}
			if err := releaseZlibWriter(w, true); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("new", func(b *testing.B) {
		var dst bytes.Buffer
		b.ReportAllocs()
		for b.Loop() {
			dst.Reset()
			w := zlib.NewWriter(&dst)
			if _, err := w.Write(data); err != nil {
				b.Fatal(err)
			}
			if err := w.Close(); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkFlateReaderReuse compares pooled readers with newly allocated readers.
func BenchmarkFlateReaderReuse(b *testing.B) {
	data := bytes.Repeat([]byte("pdfcpu flate reader benchmark "), 32)
	var encoded bytes.Buffer
	w := zlib.NewWriter(&encoded)
	if _, err := w.Write(data); err != nil {
		b.Fatal(err)
	}
	if err := w.Close(); err != nil {
		b.Fatal(err)
	}

	b.Run("pooled", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			r, err := acquireZlibReader(bytes.NewReader(encoded.Bytes()))
			if err != nil {
				b.Fatal(err)
			}
			_, err = io.Copy(io.Discard, r)
			releaseZlibReader(r, err == nil)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("new", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			r, err := zlib.NewReader(bytes.NewReader(encoded.Bytes()))
			if err != nil {
				b.Fatal(err)
			}
			if _, err := io.Copy(io.Discard, r); err != nil {
				_ = r.Close()
				b.Fatal(err)
			}
			if err := r.Close(); err != nil {
				b.Fatal(err)
			}
		}
	})
}
