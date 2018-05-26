package filter

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func compare(t *testing.T, a, b []byte) {

	if len(a) != len(b) {
		t.Errorf("length mismatch %d != %d", len(a), len(b))
		t.Logf("a:\n%s\n", hex.Dump(a))
		t.Logf("b:\n%s\n", hex.Dump(b))
		return
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			t.Errorf("mismatch at %d(0x%02x), 0x%02x != 0x%02x\n", i, i, a[i], b[i])
			t.Logf("a:\n%s\n", hex.Dump(a))
			t.Logf("b:\n%s\n", hex.Dump(b))
			return
		}
	}

}

func TestRunLengthEncoding(t *testing.T) {

	f := runLengthDecode{baseFilter{}}

	for _, tt := range []struct {
		raw, enc string
	}{
		{"\x01", "\x00\x01\x80"},
		{"\x01\x01", "\xFF\x01\x80"},
		{"\x00\x00\x02\x02", "\xFF\x00\xFF\x02\x80"},
		{"\x00\x00\x00", "\xFE\x00\x80"},
		{"\x00\x00\x00\x01", "\xFE\x00\x00\x01\x80"},
		{"\x00\x00\x00\x00", "\xFD\x00\x80"},
		{"\x00\x00\x00\x00\x00", "\xFC\x00\x80"},
		{"\x00\x00\x01", "\xFF\x00\x00\x01\x80"},
		{"\x00\x01", "\x01\x00\x01\x80"},
		{"\x00\x01\x02", "\x02\x00\x01\x02\x80"},
		{"\x00\x01\x02\x03", "\x03\x00\x01\x02\x03\x80"},
		{"\x00\x01\x02\x03\x02", "\x04\x00\x01\x02\x03\x02\x80"},
		{"\x00\x01", "\x01\x00\x01\x80"},
		{"\x00\x01\x01", "\x00\x00\xFF\x01\x80"},
		{"\x00\x01\x01\x01", "\x00\x00\xFE\x01\x80"},
		{"\x00\x00\x01\x02\x00\x00", "\xFF\x00\x01\x01\x02\xFF\x00\x80"},
	} {
		var enc bytes.Buffer
		f.encode(&enc, []byte(tt.raw))
		compare(t, enc.Bytes(), []byte(tt.enc))

		var raw bytes.Buffer
		f.decode(&raw, enc.Bytes())
		compare(t, raw.Bytes(), []byte(tt.raw))
	}

}
