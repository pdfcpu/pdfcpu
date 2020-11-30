package filter

import (
	"bytes"
	"io"
	"io/ioutil"
	"math/rand"
	"testing"
)

func TestReacher(t *testing.T) {
	input := []byte("789456zesd45679998989")
	rd := bytes.NewReader(input)
	r := newReacher(rd, []byte("456"))
	_, err := io.Copy(ioutil.Discard, r)
	if err != nil {
		t.Fatal(err)
	}
	nbRead := len(input) - rd.Len()
	if nbRead != 6 {
		t.Error()
	}

	rd = bytes.NewReader(input)
	r = newReacher(rd, []byte("998"))
	_, err = io.Copy(ioutil.Discard, r)
	if err != nil {
		t.Fatal(err)
	}
	nbRead = len(input) - rd.Len()
	if nbRead != len("789456zesd45679998") {
		t.Error()
	}
}

func TestDontPassEOD(t *testing.T) {
	for _, fi := range []string{
		ASCII85,
		ASCIIHex,
		RunLength,
		LZW,
		Flate,
	} {
		input := make([]byte, 1000)
		_, _ = rand.Read(input)
		fil, err := NewFilter(fi, nil)
		if err != nil {
			t.Fatal(err)
		}
		r, err := fil.Encode(bytes.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}
		filtered, err := ioutil.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}

		// add data passed EOD
		additionalBytes := []byte("')(à'(ààç454658")
		filtered = append(filtered, additionalBytes...)

		re := bytes.NewReader(filtered)
		toRead, err := fil.Decode(re)
		if err != nil {
			t.Fatal(err)
		}
		_, err = ioutil.ReadAll(toRead)
		if err != nil {
			t.Fatal(err)
		}
		// we want to use the number of byte read from the
		// filtered stream to detect EOD
		if re.Len() != len(additionalBytes) {
			t.Errorf("invalid number of bytes read with filter %s", fi)
		}
	}
}
