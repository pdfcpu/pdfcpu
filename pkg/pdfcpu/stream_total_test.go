package pdfcpu

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// dupStreamPDF is a classic-xref PDF with k xref entries pointing at one unreferenced p-byte stream.
func dupStreamPDF(k, p int) []byte {
	var out bytes.Buffer
	offs := map[int]int{}
	out.WriteString("%PDF-1.4\n")
	for i, b := range []string{
		"<</Type/Catalog/Pages 2 0 R>>",
		"<</Type/Pages/Kids[3 0 R]/Count 1>>",
		"<</Type/Page/Parent 2 0 R/MediaBox[0 0 10 10]>>",
	} {
		offs[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, b)
	}
	s := out.Len()
	fmt.Fprintf(&out, "4 0 obj\n<</Length %d>>\nstream\n%s\nendstream\nendobj\n", p, strings.Repeat("A", p))
	xo := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", 4+k)
	for i := 1; i < 4+k; i++ {
		o := s
		if i < 4 {
			o = offs[i]
		}
		fmt.Fprintf(&out, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&out, "trailer\n<</Size %d/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", 4+k, xo)
	return out.Bytes()
}

// TestMaxStreamTotalBytes: one stream named by 8 xref entries is read 8 times and held 8 times; the cumulative
// cap sees that, while the per-stream cap and the read budget (8 x file size) do not. Zero keeps today's behaviour.
func TestMaxStreamTotalBytes(t *testing.T) {
	const p = 1 << 20
	pdf := dupStreamPDF(8, p)
	read := func(max int64) error {
		conf := model.NewDefaultConfiguration()
		conf.Limits.MaxStreamTotalBytes = max
		_, err := Read(context.Background(), bytes.NewReader(pdf), conf)
		return err
	}
	if err := read(0); err != nil {
		t.Fatalf("limit unset: %v", err)
	}
	if err := read(2 * p); !errors.Is(err, model.ErrStreamTotalLimitExceeded) || !errors.Is(err, model.ErrLimitExceeded) {
		t.Fatalf("got %v, want ErrStreamTotalLimitExceeded", err)
	}
	if err := read(10 * p); err != nil {
		t.Fatalf("limit above the streams: %v", err)
	}
}
