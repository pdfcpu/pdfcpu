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

package types

import (
	"bytes"
	"errors"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
)

func TestDecodePreservesSoleJBIG2Stream(t *testing.T) {
	raw := []byte{0x97, 0x4a, 0x42, 0x32}
	sd := StreamDict{
		Dict:           NewDict(),
		Raw:            raw,
		FilterPipeline: []PDFFilter{{Name: filter.JBIG2}},
	}

	if err := sd.Decode(); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(sd.Content, raw) {
		t.Fatalf("got %x, want %x", sd.Content, raw)
	}
}

func TestDecodePreservesTerminalOpaqueImageStream(t *testing.T) {
	for _, tt := range []struct {
		name   string
		filter string
	}{
		{name: "JPX", filter: filter.JPX},
		{name: "JBIG2", filter: filter.JBIG2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sd := StreamDict{
				Dict: NewDict(),
				Raw:  []byte("616263>"),
				FilterPipeline: []PDFFilter{
					{Name: filter.ASCIIHex},
					{Name: tt.filter},
				},
			}

			if err := sd.Decode(); err != nil {
				t.Fatal(err)
			}

			if got, want := string(sd.Content), "abc"; got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

func TestDecodeRejectsNonTerminalOpaqueImageStream(t *testing.T) {
	sd := StreamDict{
		Dict: NewDict(),
		Raw:  []byte("abc"),
		FilterPipeline: []PDFFilter{
			{Name: filter.JPX},
			{Name: filter.ASCIIHex},
		},
	}

	if err := sd.Decode(); !errors.Is(err, filter.ErrUnsupportedFilter) {
		t.Fatalf("got %v, want %v", err, filter.ErrUnsupportedFilter)
	}
}

// TestStreamDictReencodesFlatePredictor verifies edits retain the original predictor parameters.
func TestStreamDictReencodesFlatePredictor(t *testing.T) {
	sd := StreamDict{
		Dict:    NewDict(),
		Content: []byte("abcdef"),
		FilterPipeline: []PDFFilter{{
			Name: filter.Flate,
			DecodeParms: Dict{
				"Predictor": Integer(filter.PredictorUp),
				"Columns":   Integer(3),
			},
		}},
	}
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	sd.Content = nil
	if err := sd.Decode(); err != nil {
		t.Fatal(err)
	}
	sd.Content = append(sd.Content, []byte("ghi")...)
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	sd.Content = nil
	if err := sd.Decode(); err != nil {
		t.Fatal(err)
	}
	if got, want := string(sd.Content), "abcdefghi"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestDecodeLengthWithLimitRejectsOversizedPrefixWithoutFilters covers raw and previously decoded streams.
func TestDecodeLengthWithLimitRejectsOversizedPrefixWithoutFilters(t *testing.T) {
	for _, tc := range []struct {
		name string
		sd   StreamDict
	}{
		{"raw", StreamDict{Dict: NewDict(), Raw: []byte("hello")}},
		{"cached", StreamDict{Dict: NewDict(), Content: []byte("hello")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.sd.DecodeLengthWithLimit(5, 4); !errors.Is(err, filter.ErrDecodeLimitExceeded) {
				t.Fatalf("got %v, want %v", err, filter.ErrDecodeLimitExceeded)
			}
		})
	}
}

// TestCachedFilterContentHonorsDecodeLimit checks stricter limits on previously decoded streams.
func TestCachedFilterContentHonorsDecodeLimit(t *testing.T) {
	for _, terminal := range []string{"", filter.JPX, filter.JBIG2} {
		sd := StreamDict{Dict: NewDict(), Raw: []byte("68656c6c6f>"), FilterPipeline: []PDFFilter{{Name: filter.ASCIIHex}}}
		if terminal != "" {
			sd.FilterPipeline = append(sd.FilterPipeline, PDFFilter{Name: terminal})
		}
		if err := sd.DecodeWithLimit(5); err != nil {
			t.Fatal(err)
		}
		if err := sd.DecodeWithLimit(4); !errors.Is(err, filter.ErrDecodeLimitExceeded) {
			t.Fatalf("terminal=%q: got %v", terminal, err)
		}
		got, err := sd.DecodeLengthWithLimit(4, 4)
		if err != nil || string(got) != "hell" {
			t.Fatalf("terminal=%q: prefix=%q, err=%v", terminal, got, err)
		}
	}
}

// TestPreservedStreamContentDoesNotConsumeDecodeLimit keeps raw and opaque image passthrough unchanged.
func TestPreservedStreamContentDoesNotConsumeDecodeLimit(t *testing.T) {
	for _, name := range []string{"", "Crypt", filter.DCT, filter.JPX, filter.JBIG2} {
		sd := StreamDict{Dict: NewDict(), Raw: []byte("hello")}
		if name != "" {
			sd.FilterPipeline = []PDFFilter{{Name: name}}
		}
		for range 2 {
			if err := sd.DecodeWithLimit(4); err != nil || string(sd.Content) != "hello" {
				t.Fatalf("filter=%q: content=%q, err=%v", name, sd.Content, err)
			}
		}
	}
}

func TestLazyObjectStreamGetDataPreservesDecodeErrorText(t *testing.T) {
	sd := StreamDict{
		Dict:           NewDict(),
		Raw:            []byte("not flate data"),
		FilterPipeline: []PDFFilter{{Name: filter.Flate}},
	}
	wantSD := sd
	wantErr := wantSD.DecodeWithLimit(1 << 20)
	if wantErr == nil {
		t.Fatal("corrupt Flate stream decoded without error")
	}

	osd := &ObjectStreamDict{StreamDict: sd, MaxDecodeBytes: 1 << 20}
	o := NewLazyObjectStreamObject(osd, 0, -1, nil)
	lazy := o.(LazyObjectStreamObject)
	if _, err := lazy.GetData(); err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("got %v, want unchanged decode error %v", err, wantErr)
	}
}

func TestLazyObjectStreamGetDataCompactsDecodedContent(t *testing.T) {
	// An odd length that is not an allocation size class, so a rounded-up copy would leave len < cap.
	sd := StreamDict{
		Dict:           NewDict(),
		Content:        bytes.Repeat([]byte{'0'}, 256<<10+3),
		FilterPipeline: []PDFFilter{{Name: filter.Flate}},
	}
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	sd.Content = nil
	osd := &ObjectStreamDict{StreamDict: sd, MaxDecodeBytes: 1 << 20}
	lazy := NewLazyObjectStreamObject(osd, 0, -1, nil).(LazyObjectStreamObject)
	if _, err := lazy.GetData(); err != nil {
		t.Fatal(err)
	}
	if len(osd.Content) != cap(osd.Content) {
		t.Fatalf("decoded object stream length=%d capacity=%d", len(osd.Content), cap(osd.Content))
	}
	if allocs := testing.AllocsPerRun(10, func() { _, _ = lazy.GetData() }); allocs != 0 {
		t.Fatalf("repeated GetData allocated %.0f times, want 0", allocs)
	}
}

func TestLazyObjectStreamGetDataPreservesRawAlias(t *testing.T) {
	raw := make([]byte, 4, 1<<20)
	osd := &ObjectStreamDict{StreamDict: StreamDict{Dict: NewDict(), Raw: raw}, MaxDecodeBytes: 1 << 20}
	lazy := NewLazyObjectStreamObject(osd, 0, -1, nil).(LazyObjectStreamObject)
	if _, err := lazy.GetData(); err != nil {
		t.Fatal(err)
	}
	if &osd.Content[0] != &osd.Raw[0] {
		t.Fatal("unfiltered object stream content no longer aliases raw data")
	}
}

func TestLazyObjectStreamGetDataUsesDecoder(t *testing.T) {
	wantErr := errors.New("shared decode budget")
	osd := &ObjectStreamDict{StreamDict: StreamDict{Dict: NewDict(), Raw: []byte("encoded")}}
	called := 0
	decodeStream := func(sd *StreamDict) error {
		called++
		if sd != &osd.StreamDict {
			t.Fatalf("got stream dict %p, want %p", sd, &osd.StreamDict)
		}
		return wantErr
	}
	lazy := NewLazyObjectStreamObjectWithDecoder(osd, 0, -1, nil, decodeStream).(LazyObjectStreamObject)
	if _, err := lazy.GetData(); !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want %v", err, wantErr)
	}
	if called != 1 {
		t.Fatalf("decoder called %d times, want 1", called)
	}
}
