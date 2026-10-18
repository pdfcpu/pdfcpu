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

package api

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Regression coverage for GHSA-jp42-qm45-2x6g: configured decoded-byte limit bypasses.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-jp42-qm45-2x6g
// Fixtures keep allocations modest while exercising the four reported failure cases.

func ghsaJP42Stream(t *testing.T, dict string, data []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return []byte(fmt.Sprintf("<< %s /Filter /FlateDecode /Length %d >>\nstream\n%s\nendstream", dict, b.Len(), b.Bytes()))
}

func ghsaJP42Objects(t *testing.T, metadata []byte) [][]byte {
	t.Helper()
	catalog := "<< /Type /Catalog /Pages 2 0 R"
	if metadata != nil {
		catalog += " /Metadata 5 0 R"
	}
	objects := [][]byte{
		[]byte(catalog + " >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 10 10] /Resources << >> /Contents 4 0 R >>"),
		ghsaJP42Stream(t, "", []byte("q Q\n")),
	}
	if metadata != nil {
		objects = append(objects, ghsaJP42Stream(t, "/Type /Metadata /Subtype /XML", metadata))
	}
	return objects
}

func ghsaJP42PDF(t *testing.T, objects [][]byte, xrefSize int) []byte {
	t.Helper()
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	start := b.Len()
	if xrefSize > 0 {
		offsets = append(offsets, start)
		xrefSize = max(xrefSize, len(offsets))
		data := make([]byte, xrefSize*7)
		for i, off := range offsets {
			data[i*7] = 1
			binary.BigEndian.PutUint32(data[i*7+1:], uint32(off))
		}
		data[0], data[5], data[6] = 0, 255, 255
		// Permanently free entries may link to object zero outside the free-list chain (7.5.4).
		for i := len(offsets); i < xrefSize; i++ {
			data[i*7+5], data[i*7+6] = 255, 255
		}
		dict := fmt.Sprintf("/Type /XRef /Size %d /Root 1 0 R /W [1 4 2]", xrefSize)
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", len(offsets)-1, ghsaJP42Stream(t, dict, data))
	} else {
		fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f\n", len(offsets))
		for _, off := range offsets[1:] {
			fmt.Fprintf(&b, "%010d 00000 n\n", off)
		}
		fmt.Fprintf(&b, "trailer\n<< /Root 1 0 R /Size %d >>\n", len(offsets))
	}
	fmt.Fprintf(&b, "startxref\n%d\n%%%%EOF\n", start)
	return b.Bytes()
}

// TestGHSAJP42ReadContextCompressedXRefDecodeLimit rejects excessive expansion before xref parsing or repair.
func TestGHSAJP42ReadContextCompressedXRefDecodeLimit(t *testing.T) {
	const entries = 16 * 1024
	const decodedBytes = entries * 7
	objects := ghsaJP42Objects(t, nil)
	data := ghsaJP42PDF(t, objects, entries)
	if len(data) >= decodedBytes/20 {
		t.Fatalf("fixture is not highly compressed: PDF=%d decoded xref=%d", len(data), decodedBytes)
	}
	t.Logf("PDF=%d bytes; decoded xref=%d bytes", len(data), decodedBytes)
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		for _, eager := range []bool{false, true} {
			for _, limit := range []int64{4 << 10, decodedBytes - 1, decodedBytes} {
				t.Run(fmt.Sprintf("mode=%d/eager=%v/limit=%d", mode, eager, limit), func(t *testing.T) {
					conf := model.NewStatelessConfiguration()
					conf.ValidationMode, conf.DecodeAllStreams = mode, eager
					conf.Limits.MaxDecodeBytes = limit
					ctx, err := ReadContext(t.Context(), bytes.NewReader(data), conf)
					if limit < decodedBytes {
						if !errors.Is(err, filter.ErrDecodeLimitExceeded) {
							t.Fatalf("got %v, want decode limit error without repair", err)
						}
						return
					}
					if err != nil {
						t.Fatalf("at exact limit: %v", err)
					}
					if ctx.Size == nil || *ctx.Size != entries || len(ctx.Table) != entries {
						t.Fatalf("incomplete xref: size=%v entries=%d", ctx.Size, len(ctx.Table))
					}
				})
			}
		}
	}
}

type ghsaJP42RepairWatchReader struct {
	*bytes.Reader
	xrefOffset int64
	xrefSeen   bool
	rewinds    int
}

// Seek records rewinds to the start after the reader has visited the xref stream.
func (r *ghsaJP42RepairWatchReader) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekStart {
		if offset == 0 && r.xrefSeen {
			r.rewinds++
		}
		if offset == r.xrefOffset {
			r.xrefSeen = true
		}
	}
	return r.Reader.Seek(offset, whence)
}

// TestGHSAJP42ReadContextXRefDecodeLimitStopsRepair rejects oversized xrefs without a repair scan.
func TestGHSAJP42ReadContextXRefDecodeLimitStopsRepair(t *testing.T) {
	objects := ghsaJP42Objects(t, nil)
	data := ghsaJP42PDF(t, objects, 16*1024)
	offset := bytes.Index(data, []byte(fmt.Sprintf("%d 0 obj\n", len(objects)+1)))
	if offset < 0 {
		t.Fatal("missing xref stream object")
	}
	// A same-length unknown filter forces the ordinary repair path without changing any offsets.
	corrupt := append(bytes.Clone(data[:offset]), bytes.Replace(data[offset:], []byte("/FlateDecode"), []byte("/BogusDecode"), 1)...)
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		for _, eager := range []bool{false, true} {
			t.Run(fmt.Sprintf("mode=%d/eager=%v", mode, eager), func(t *testing.T) {
				conf := model.NewStatelessConfiguration()
				conf.ValidationMode, conf.DecodeAllStreams = mode, eager
				conf.Limits.MaxDecodeBytes = 4 << 10
				r := &ghsaJP42RepairWatchReader{Reader: bytes.NewReader(data), xrefOffset: int64(offset)}
				_, err := ReadContext(t.Context(), r, conf)
				if !errors.Is(err, filter.ErrDecodeLimitExceeded) {
					t.Fatalf("got %v, want decode limit error", err)
				}
				if !r.xrefSeen || r.rewinds != 0 {
					t.Fatalf("xref seen=%v repair rewinds=%d, want true/0", r.xrefSeen, r.rewinds)
				}
				control := &ghsaJP42RepairWatchReader{Reader: bytes.NewReader(corrupt), xrefOffset: int64(offset)}
				_, _ = ReadContext(t.Context(), control, conf)
				if !control.xrefSeen || control.rewinds == 0 {
					t.Fatal("corrupt-xref control did not exercise the repair-scan detector")
				}
			})
		}
	}
}

// TestGHSAJP42ReadAndValidateCatalogMetadataDecodeLimit bounds catalog metadata through the direct read/validate API.
func TestGHSAJP42ReadAndValidateCatalogMetadataDecodeLimit(t *testing.T) {
	const decodedBytes = 128 << 10
	const prefix = "<x:xmpmeta xmlns:x=\"adobe:ns:meta/\">"
	const suffix = "</x:xmpmeta>"
	payload := append([]byte(prefix), bytes.Repeat([]byte(" "), decodedBytes-len(prefix)-len(suffix))...)
	payload = append(payload, suffix...)
	data := ghsaJP42PDF(t, ghsaJP42Objects(t, payload), 0)
	if len(data) >= decodedBytes/20 {
		t.Fatalf("fixture is not highly compressed: PDF=%d decoded metadata=%d", len(data), decodedBytes)
	}
	t.Logf("PDF=%d bytes; decoded metadata=%d bytes", len(data), decodedBytes)
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		for _, eager := range []bool{false, true} {
			for _, limit := range []int64{4 << 10, decodedBytes - 1, decodedBytes} {
				t.Run(fmt.Sprintf("mode=%d/eager=%v/limit=%d", mode, eager, limit), func(t *testing.T) {
					conf := model.NewStatelessConfiguration()
					conf.ValidationMode, conf.DecodeAllStreams = mode, eager
					conf.Limits.MaxDecodeBytes = limit
					ctx, err := ReadAndValidate(t.Context(), bytes.NewReader(data), conf)
					if limit < decodedBytes {
						if !errors.Is(err, filter.ErrDecodeLimitExceeded) {
							t.Fatalf("got %v, want catalog metadata decode limit error", err)
						}
						return
					}
					if err != nil {
						t.Fatalf("at exact limit: %v", err)
					}
					if !ctx.Valid || ctx.PageCount != 1 {
						t.Fatalf("valid=%v pages=%d", ctx.Valid, ctx.PageCount)
					}
				})
			}
		}
	}
}

func ghsaJP42DistinctContentPDF(t *testing.T, streams int) (data, content []byte) {
	t.Helper()
	objects := ghsaJP42Objects(t, nil)[:3]
	refs := ""
	for i := range streams {
		payload := bytes.Repeat([]byte(" "), 7<<10)
		copy(payload, fmt.Sprintf("q Q %% stream %d\n", i))
		content = append(content, payload...)
		objects = append(objects, ghsaJP42Stream(t, "", payload))
		refs += fmt.Sprintf("%d 0 R ", i+4)
	}
	objects[2] = bytes.Replace(objects[2], []byte("/Contents 4 0 R"), []byte("/Contents ["+refs+"]"), 1)
	return ghsaJP42PDF(t, objects, 0), content
}

// TestGHSAJP42PageContentDistinctStreamsDecodeLimit scales the six-stream PoC to 7 KiB streams and an 8 KiB limit.
func TestGHSAJP42PageContentDistinctStreamsDecodeLimit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		streams int
		limit   int64
		fail    bool
	}{
		{"single fits", 1, 8 << 10, false},
		{"six exceed combined limit", 6, 8 << 10, true},
		{"six fit exact limit", 6, 42 << 10, false},
	} {
		data, content := ghsaJP42DistinctContentPDF(t, tc.streams)
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			for _, eager := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/mode=%d/eager=%v", tc.name, mode, eager), func(t *testing.T) {
					conf := model.NewStatelessConfiguration()
					conf.ValidationMode, conf.DecodeAllStreams = mode, eager
					conf.Limits.MaxDecodeBytes = tc.limit
					calls := 0
					err := ExtractContent(t.Context(), bytes.NewReader(data), nil, func(r io.Reader, _ int) error {
						calls++
						got, err := io.ReadAll(r)
						if err == nil && !bytes.Equal(got, content) {
							t.Fatal("unexpected extracted content")
						}
						return err
					}, conf)
					if tc.fail {
						if !errors.Is(err, filter.ErrDecodeLimitExceeded) || calls != 0 {
							t.Fatalf("calls=%d error=%v, want combined limit error before extraction", calls, err)
						}
						return
					}
					if err != nil || calls != 1 {
						t.Fatalf("calls=%d error=%v", calls, err)
					}
				})
			}
		}
	}
}
