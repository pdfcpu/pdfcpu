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
	"compress/zlib"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
)

type failingDecodedContentReader struct {
	err error
}

func (r failingDecodedContentReader) Read([]byte) (int, error) {
	return 0, r.err
}

func requireStreamFilterContext(t *testing.T, err error, index int, name, phase string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{
		"stream filter[",
		name,
		phase,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
	if !strings.Contains(err.Error(), "["+strconv.Itoa(index)+"]") {
		t.Fatalf("expected filter index %d in %q", index, err)
	}
}

func TestDecodeWrapsFilterParameterPreparation(t *testing.T) {
	sd := StreamDict{
		Dict:           NewDict(),
		Raw:            []byte("data"),
		FilterPipeline: []PDFFilter{{Name: filter.CCITTFax}},
	}

	err := sd.Decode()

	requireStreamFilterContext(t, err, 0, filter.CCITTFax, "prepare parameters")
}

func TestDecodeWrapsFilterConstruction(t *testing.T) {
	const filterName = "UnknownDecode"
	sd := StreamDict{
		Dict:           NewDict(),
		Raw:            []byte("data"),
		FilterPipeline: []PDFFilter{{Name: filterName}},
	}

	err := sd.Decode()

	requireStreamFilterContext(t, err, 0, filterName, "construct")
}

func TestDecodeWrapsFilterExecution(t *testing.T) {
	sd := StreamDict{
		Dict:           NewDict(),
		Raw:            []byte("not zlib"),
		FilterPipeline: []PDFFilter{{Name: filter.Flate}},
	}

	err := sd.Decode()

	if !errors.Is(err, zlib.ErrHeader) {
		t.Fatalf("expected %v, got %v", zlib.ErrHeader, err)
	}
	requireStreamFilterContext(t, err, 0, filter.Flate, "decode")
}

func TestDecodePreservesUnsupportedFilterSentinel(t *testing.T) {
	sd := StreamDict{
		Dict: NewDict(),
		Raw:  []byte("data"),
		FilterPipeline: []PDFFilter{
			{Name: filter.JPX},
			{Name: filter.ASCIIHex},
		},
	}

	err := sd.Decode()

	if !errors.Is(err, filter.ErrUnsupportedFilter) {
		t.Fatalf("expected %v, got %v", filter.ErrUnsupportedFilter, err)
	}
	requireStreamFilterContext(t, err, 0, filter.JPX, "decode")
}

func TestDecodedContentCopyPreservesCause(t *testing.T) {
	wantErr := errors.New("decoded content read failed")

	_, err := decodedContent(failingDecodedContentReader{err: wantErr})

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "copy decoded content") {
		t.Fatalf("expected decoded content copy context in %q", err)
	}
}
