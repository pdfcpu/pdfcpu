package pdfcpu

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// objStmBomb builds a small PDF with k Flate object streams of n entries each.
// Every stream passes the per-stream MaxObjectStreamCount check on its own.
func objStmBomb(n, k int) []byte {
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	w.Write([]byte(strings.Repeat("7 0 ", n) + "0"))
	w.Close()
	first := 4 * n

	var out bytes.Buffer
	offs := map[int]int{}
	obj := func(nr int, body string) {
		offs[nr] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", nr, body)
	}
	out.WriteString("%PDF-1.5\n")
	obj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	obj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	obj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 10 10] >>")
	for s := 9; s < 9+k; s++ {
		obj(s, fmt.Sprintf("<< /Type /ObjStm /N %d /First %d /Filter /FlateDecode /Length %d >>\nstream\n%s\nendstream", n, first, z.Len(), z.Bytes()))
	}
	x := 9 + k
	size := x + 1 + k
	xo := out.Len()
	var rows bytes.Buffer
	for i := 0; i < size; i++ {
		t, f2, f3 := byte(1), uint32(offs[i]), uint16(0)
		switch {
		case i == 0:
			t, f2, f3 = 0, 0, 65535
		case i >= 4 && i < 9:
			t, f2 = 0, 0
		case i == x:
			f2 = uint32(xo)
		case i > x:
			t, f2 = 2, uint32(9+i-x-1)
		}
		rows.WriteByte(t)
		binary.Write(&rows, binary.BigEndian, f2)
		binary.Write(&rows, binary.BigEndian, f3)
	}
	fmt.Fprintf(&out, "%d 0 obj\n<< /Type /XRef /Size %d /W [1 4 2] /Root 1 0 R /Length %d >>\nstream\n", x, size, rows.Len())
	out.Write(rows.Bytes())
	fmt.Fprintf(&out, "\nendstream\nendobj\nstartxref\n%d\n%%%%EOF\n", xo)
	return out.Bytes()
}

func readBomb(t *testing.T, mutate func(*model.ResourceLimits)) error {
	t.Helper()
	conf := model.NewDefaultConfiguration()
	mutate(&conf.Limits)
	_, err := Read(context.Background(), bytes.NewReader(objStmBomb(15000, 300)), conf)
	return err
}

func TestObjectStreamTotalCountLimit(t *testing.T) {
	// 300 streams x 15 000 entries = 4.5M entries from a ~60 KB file.
	if err := readBomb(t, func(l *model.ResourceLimits) {}); err != nil {
		t.Fatalf("limit unset: %v", err)
	}
	err := readBomb(t, func(l *model.ResourceLimits) { l.MaxObjectStreamTotalCount = 100_000 })
	if !errors.Is(err, model.ErrLimitExceeded) {
		t.Fatalf("got %v, want ErrLimitExceeded", err)
	}
}

func TestTotalDecodeBytesLimitObjectStreams(t *testing.T) {
	sd := func() *types.StreamDict {
		return &types.StreamDict{
			Raw:            []byte("31323334353637383930>"),
			FilterPipeline: []types.PDFFilter{{Name: filter.ASCIIHex}},
		}
	}
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxTotalDecodeBytes = 15
	b := newObjectStreamDecodeBudget(conf)
	if err := b.decode(sd(), conf.DecodeLimit()); err != nil {
		t.Fatalf("first decode: %v", err)
	}
	if err := b.decode(sd(), conf.DecodeLimit()); !errors.Is(err, model.ErrLimitExceeded) {
		t.Fatalf("second decode: got %v, want ErrLimitExceeded", err)
	}
	conf.Limits.MaxTotalDecodeBytes = 0
	b = newObjectStreamDecodeBudget(conf)
	for i := 0; i < 3; i++ {
		if err := b.decode(sd(), conf.DecodeLimit()); err != nil {
			t.Fatalf("limit unset, decode %d: %v", i, err)
		}
	}
}
