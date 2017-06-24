package filter

import (
	"bytes"
	"testing"
)

// Encode a test string twice with same filter
// then decode the result twice to get to the original string.
func TestEncodeDecode(t *testing.T) {

	filter, err := NewFilter("FlateDecode", nil, nil)
	if err != nil {
		t.Fatalf("Problem: %v\n", err)
	}

	input := "Hello, Gopher!"
	//t.Logf("encoding: len:%d % X <%s>\n", len(input), input, input)

	r := bytes.NewReader([]byte(input))

	b1, err := filter.Encode(r)
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

	if input != c2.String() {
		t.Fatal("original content != decoded content")
	}

}
