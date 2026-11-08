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

// TestTotalDecodeBytesOneBudgetPerParse checks that object streams, other streams and images
// are charged to the same MaxTotalDecodeBytes budget.
func TestTotalDecodeBytesOneBudgetPerParse(t *testing.T) {
	sd := func() *types.StreamDict {
		sd := &types.StreamDict{
			Dict:           types.NewDict(),
			Content:        []byte("1234567890"),
			FilterPipeline: []types.PDFFilter{{Name: filter.Flate}},
		}
		if err := sd.Encode(); err != nil {
			t.Fatal(err)
		}
		sd.Content = nil
		return sd
	}
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxTotalDecodeBytes = 10
	x := &model.XRefTable{Conf: conf}

	if err := newObjectStreamDecodeBudget(x).decode(sd(), conf.DecodeLimit()); err != nil {
		t.Fatalf("object stream: %v", err)
	}
	// The object stream spent the whole budget (compacted to its 10 bytes), so other streams are refused too.
	if err := x.DecodeStream(sd()); !errors.Is(err, model.ErrLimitExceeded) {
		t.Fatalf("content stream: got %v, want ErrLimitExceeded", err)
	}
	if _, err := streamBytes(x, sd(), "image"); !errors.Is(err, model.ErrLimitExceeded) {
		t.Fatalf("image: got %v, want ErrLimitExceeded", err)
	}
}

// bigArrayPDF is a catalog whose /X is one array of n copies of elem, written in place
// or packed in a Flate object stream. "[]" parses into about 40 bytes per 2 bytes of input.
func bigArrayPDF(elem string, n int, packed bool) []byte {
	body := "[" + strings.Repeat(elem, n) + "]"
	var out bytes.Buffer
	offs := map[int]int{}
	obj := func(nr int, b string) {
		offs[nr] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", nr, b)
	}
	out.WriteString("%PDF-1.5\n")
	obj(1, "<< /Type /Catalog /Pages 2 0 R /X 5 0 R >>")
	obj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	obj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 10 10] >>")
	if !packed {
		obj(5, body)
		xo := out.Len()
		out.WriteString("xref\n0 6\n0000000000 65535 f \n")
		for i := 1; i < 6; i++ {
			if o, ok := offs[i]; ok {
				fmt.Fprintf(&out, "%010d 00000 n \n", o)
			} else {
				out.WriteString("0000000000 00000 f \n")
			}
		}
		fmt.Fprintf(&out, "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xo)
		return out.Bytes()
	}
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	w.Write([]byte("5 0 " + body))
	w.Close()
	obj(4, fmt.Sprintf("<< /Type /ObjStm /N 1 /First 4 /Filter /FlateDecode /Length %d >>\nstream\n%s\nendstream", z.Len(), z.Bytes()))
	var rows bytes.Buffer
	for i := 0; i < 7; i++ {
		t, f2 := byte(1), uint32(offs[i])
		switch i {
		case 0:
			t, f2 = 0, 0
		case 5:
			t, f2 = 2, 4
		case 6:
			f2 = uint32(out.Len())
		}
		rows.WriteByte(t)
		binary.Write(&rows, binary.BigEndian, f2)
		rows.Write([]byte{0, 0})
	}
	xo := out.Len()
	fmt.Fprintf(&out, "6 0 obj\n<< /Type /XRef /Size 7 /W [1 4 2] /Root 1 0 R /Length %d >>\nstream\n", rows.Len())
	out.Write(rows.Bytes())
	fmt.Fprintf(&out, "\nendstream\nendobj\nstartxref\n%d\n%%%%EOF\n", xo)
	return out.Bytes()
}

// TestMaxParsedBytesDictElements: a "<</a 1>>" element holds about 360 bytes live (measured), which the
// limit must charge, not the 136 of one array slot and one entry.
func TestMaxParsedBytesDictElements(t *testing.T) {
	pdf := bigArrayPDF("<</a 1>>", 10_000, false)
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxParsedBytes = 10_000 * 300
	if _, err := Read(context.Background(), bytes.NewReader(pdf), conf); !errors.Is(err, model.ErrParsedLimitExceeded) {
		t.Fatalf("got %v, want ErrParsedLimitExceeded", err)
	}
}

func TestMaxParsedBytesLimit(t *testing.T) {
	for _, packed := range []bool{false, true} {
		pdf := bigArrayPDF("[]", 100_000, packed)
		read := func(max int64) error {
			conf := model.NewDefaultConfiguration()
			conf.Limits.MaxParsedBytes = max
			ctx, err := Read(context.Background(), bytes.NewReader(pdf), conf)
			if err != nil {
				return err
			}
			// Packed objects are parsed when first read.
			if l, ok := ctx.Table[5].Object.(types.LazyObjectStreamObject); ok {
				_, err = l.DecodedObject(context.Background())
			}
			return err
		}
		if err := read(0); err != nil {
			t.Fatalf("packed=%v limit unset: %v", packed, err)
		}
		if err := read(1 << 20); !errors.Is(err, model.ErrParsedLimitExceeded) || !errors.Is(err, model.ErrLimitExceeded) {
			t.Fatalf("packed=%v: got %v, want ErrLimitExceeded", packed, err)
		}
		if err := read(100_000 * 40 * 2); err != nil {
			t.Fatalf("packed=%v limit above the estimate: %v", packed, err)
		}
	}
}
