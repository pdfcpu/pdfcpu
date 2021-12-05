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

package filter_test

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
)

// Encode a test string twice with same filter
// then decode the result twice to get to the original string.
func encodeDecodeUsingFilterNamed(t *testing.T, filterName string) {

	filter, err := filter.NewFilter(filterName, nil)
	if err != nil {
		t.Fatalf("Problem: %v\n", err)
	}

	want := "Hello, Gopher!"
	t.Logf("encoding using filter %s: len:%d % X <%s>\n", filterName, len(want), want, want)

	b1, err := filter.Encode(strings.NewReader(want))
	if err != nil {
		t.Fatalf("Problem encoding 1: %v\n", err)
	}
	//t.Logf("encoded 1:  len:%d % X <%s>\n", b1.Len(), b1.Bytes(), b1.Bytes())

	b2, err := filter.Encode(b1)
	if err != nil {
		t.Fatalf("Problem encoding 2: %v\n", err)
	}
	//t.Logf("encoded 2:  len:%d % X <%s>\n", b2.Len(), b2.Bytes(), b2.Bytes())

	c1, err := filter.Decode(b2)
	if err != nil {
		t.Fatalf("Problem decoding 2: %v\n", err)
	}
	//t.Logf("decoded 2:  len:%d % X <%s>\n", c1.Len(), c1.Bytes(), c1.Bytes())

	c2, err := filter.Decode(c1)
	if err != nil {
		t.Fatalf("Problem decoding 1: %v\n", err)
	}
	//t.Logf("decoded 1:  len:%d % X <%s>\n", c2.Len(), c2.Bytes(), c2.Bytes())

	bb, err := io.ReadAll(c2)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	got := string(bb)
	if got != want {
		t.Fatalf("got:%s want:%s\n", got, want)
	}
}

func TestFilterSupport(t *testing.T) {
	var filtersTests = []struct {
		filterName string
		expected   error
	}{
		{filter.ASCII85, nil},
		{filter.ASCIIHex, nil},
		{filter.RunLength, nil},
		{filter.LZW, nil},
		{filter.Flate, nil},
		{filter.CCITTFax, nil},
		{filter.DCT, nil},
		{filter.JBIG2, filter.ErrUnsupportedFilter},
		{filter.JPX, filter.ErrUnsupportedFilter},
		{"INVALID_FILTER", errors.New("Invalid filter: <INVALID_FILTER>")},
	}
	for _, tt := range filtersTests {
		_, err := filter.NewFilter(tt.filterName, nil)
		if (tt.expected != nil && err != nil && err.Error() != tt.expected.Error()) ||
			((err == nil || tt.expected == nil) && err != tt.expected) {
			t.Errorf("Problem: '%s' (expected '%s')\n", err.Error(), tt.expected.Error())
		}
	}
}

func TestEncodeDecode(t *testing.T) {
	for _, f := range filter.List() {
		encodeDecodeUsingFilterNamed(t, f)
	}
}

var filenames = []string{
	"testdata/gettysburg.txt",
	"testdata/e.txt",
	"testdata/pi.txt",
	"testdata/Mark.Twain-Tom.Sawyer.txt",
}

// testFile tests that encoding and then decoding the given file with
// the given filter yields a file that is an exact match with the original file.
func testFile(t *testing.T, filterName, fileName string) {

	t.Logf("testFile: %s with filter:%s\n", fileName, filterName)

	f, err := filter.NewFilter(filterName, nil)
	if err != nil {
		t.Errorf("Problem: %v\n", err)
	}

	raw, err := os.Open(fileName)
	if err != nil {
		t.Errorf("%s: %v", fileName, err)
		return
	}
	defer raw.Close()

	enc, err := f.Encode(raw)
	if err != nil {
		t.Errorf("Problem encoding: %v\n", err)
	}

	dec, err := f.Decode(enc)
	if err != nil {
		t.Errorf("Problem decoding: %v\n", err)
	}

	// Compare decoded bytes with original bytes.
	golden, err := os.Open(fileName)
	if err != nil {
		t.Errorf("%s: %v", fileName, err)
		return
	}
	defer golden.Close()

	g, err := io.ReadAll(golden)
	if err != nil {
		t.Errorf("%s: %v", fileName, err)
		return
	}

	d, err := io.ReadAll(dec)
	if err != nil {
		t.Errorf("%s: %v", fileName, err)
		return
	}

	if len(d) != len(g) {
		t.Errorf("%s: length mismatch %d != %d", fileName, len(d), len(g))
		return
	}

	for i := 0; i < len(d); i++ {
		if d[i] != g[i] {
			t.Errorf("%s: mismatch at %d, 0x%02x != 0x%02x\n", fileName, i, d[i], g[i])
			return
		}
	}

}

func TestWriter(t *testing.T) {
	for _, filterName := range filter.List() {
		for _, filename := range filenames {
			testFile(t, filterName, filename)
		}
	}
}
