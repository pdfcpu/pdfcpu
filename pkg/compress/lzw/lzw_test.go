package lzw_test

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"

	"io"
	"os"
	"testing"

	"github.com/hhrutter/pdfcpu/pkg/compress/lzw"
)

func compareToGolden(t *testing.T, b []byte, fileName string) {

	golden, err := os.Open(fileName)
	if err != nil {
		t.Errorf("%s: %v", fileName, err)
		return
	}
	defer golden.Close()

	g, err := ioutil.ReadAll(golden)
	if err != nil {
		t.Errorf("%s: %v", fileName, err)
		return
	}

	if len(b) != len(g) {
		t.Errorf("%s: length mismatch after compression %d != %d", fileName, len(b), len(g))
		t.Logf("encodedBytes:\n%s\n", hex.Dump(b))
		t.Logf("goldenBytes:\n%s\n", hex.Dump(g))
		return
	}

	for i := 0; i < len(b); i++ {
		if b[i] != g[i] {
			t.Errorf("%s: mismatch at %d(0x%02x), 0x%02x != 0x%02x\n", fileName, i, i, b[i], g[i])
			t.Logf("encodedBytes:\n%s\n", hex.Dump(b))
			t.Logf("goldenBytes:\n%s\n", hex.Dump(g))
			return
		}
	}

}

// testFile tests that encoding and subsequent decoding of a given file
// yields byte streams that correspond to a golden file content at each stage.
func testFile(t *testing.T, filePrefix string, earlyChange bool) {

	t.Logf("testFile: %s\n", filePrefix)

	rawFileName := filePrefix + "Raw.lzw" // The golden file for decoded lzw.
	encFileName := filePrefix + "Enc.lzw" // The golden file for encoded lzw.

	// Read in some decompressed bytes.
	raw, err := os.Open(rawFileName)
	if err != nil {
		t.Errorf("%s: %v", rawFileName, err)
		return
	}
	defer raw.Close()

	// Compress.
	var b bytes.Buffer
	wc := lzw.NewWriter(&b, lzw.MSB, 8, earlyChange)
	written, err := io.Copy(wc, raw)
	if err != nil {
		t.Errorf("%s: %v", rawFileName, err)
		return
	}
	wc.Close()

	// The available test data implies some PDF Writers
	// do not write the final bits after eof during Close().
	// This is why do not compare these compressed results to known compressed bytes.
	// See extra step below.
	//
	// Compare compressed bytes with the corresponding golden files content.
	// compareToGolden(t, b.Bytes(), encFileName)

	// Decompress.
	rc := lzw.NewReader(&b, lzw.MSB, 8, earlyChange)
	defer rc.Close()

	var dec bytes.Buffer
	written, err = io.Copy(&dec, rc)
	if err != nil {
		t.Errorf("%s: %v", encFileName, err)
		return
	}
	t.Logf("%s: decompressed bytes:%d(%d)\n", encFileName, written, dec.Len())

	// Compare decompressed bytes with the corresponding golden files content.
	compareToGolden(t, dec.Bytes(), rawFileName)

	// The available test data implies some PDF Writers
	// do not write the final bits after eof during Close().
	// Here we take an extra step and decode known compressed bytes
	// and compare the result to known uncompressed bytes.

	// Read in encoded
	enc, err := os.Open(encFileName)
	if err != nil {
		t.Errorf("%s: %v", encFileName, err)
		return
	}
	defer enc.Close()

	// Decompress.
	rc = lzw.NewReader(&b, lzw.MSB, 8, earlyChange)

	written, err = io.Copy(&dec, rc)
	if err != nil {
		t.Errorf("%s: %v", encFileName, err)
		return
	}
	t.Logf("%s: decompressed bytes:%d(%d)\n", encFileName, written, dec.Len())

	// Compare raw/decoded bytes with the corresponding golden files content.
	compareToGolden(t, dec.Bytes(), rawFileName)
}

func TestLZW(t *testing.T) {
	for _, tt := range []struct {
		fileNamePrefix string
		earlyChange    bool
	}{
		{"testdata/earlyChange0", false}, // extracted from pdfcpu/testdata/Paclitaxel.pdf
		{"testdata/earlyChange1", true},  // extracted from pdfcpu/testdata/T6.pdf
	} {
		testFile(t, tt.fileNamePrefix, tt.earlyChange)
	}
}
