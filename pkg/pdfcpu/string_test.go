package pdfcpu

import (
	"testing"
)

func TestByteForOctalString(t *testing.T) {
	tests := []struct {
		input    string
		expected byte
	}{
		{
			"001",
			0x1,
		},
		{
			"01",
			0x1,
		},
		{
			"1",
			0x1,
		},
		{
			"010",
			0x8,
		},
		{
			"020",
			0x10,
		},
		{
			"377",
			0xff,
		},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			actual := byteForOctalString(test.input)
			if actual != test.expected {
				t.Errorf("got %x; want %x", test.expected, actual)
			}
		})
	}
}
