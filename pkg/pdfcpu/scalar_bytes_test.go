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

// scalarPDF is a classic-xref PDF whose catalog references object 5, whose body is the hostile scalar.
func scalarPDF(body string) []byte {
	var out bytes.Buffer
	offs := map[int]int{}
	obj := func(nr int, b string) {
		offs[nr] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", nr, b)
	}
	out.WriteString("%PDF-1.4\n")
	obj(1, "<< /Type /Catalog /Pages 2 0 R /X 5 0 R >>")
	obj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	obj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 10 10] >>")
	obj(4, "null")
	obj(5, body)
	xo := out.Len()
	out.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i < 6; i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offs[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xo)
	return out.Bytes()
}

// TestMaxParsedBytesScalars: strings, hex strings, names and dictionary keys are charged by length, so a few
// megabytes of one scalar are refused under a budget far below them, while zero keeps today's behaviour.
func TestMaxParsedBytesScalars(t *testing.T) {
	const n = 1 << 20
	shapes := map[string]string{
		"string": "(" + strings.Repeat("A", n) + ")",
		"hex":    "<" + strings.Repeat("41", n/2) + ">",
		"name":   "/" + strings.Repeat("a", n),
		"key":    "<</" + strings.Repeat("a", n) + " 1>>",
		"keys": func() string {
			var b strings.Builder
			b.WriteString("<<")
			for i := 0; i < n/1000; i++ {
				fmt.Fprintf(&b, "/k%07d%s 1", i, strings.Repeat("a", 990))
			}
			b.WriteString(">>")
			return b.String()
		}(),
	}
	for name, body := range shapes {
		t.Run(name, func(t *testing.T) {
			pdf := scalarPDF(body)
			read := func(max int64) error {
				conf := model.NewDefaultConfiguration()
				conf.Limits.MaxParsedBytes = max
				_, err := Read(context.Background(), bytes.NewReader(pdf), conf)
				return err
			}
			if err := read(0); err != nil {
				t.Fatalf("limit unset: %v", err)
			}
			if err := read(n / 2); !errors.Is(err, model.ErrParsedLimitExceeded) {
				t.Fatalf("got %v, want ErrParsedLimitExceeded", err)
			}
			if err := read(8 * n); err != nil {
				t.Fatalf("limit above the object: %v", err)
			}
		})
	}
}

// TestStrictObjectBytesRefusal: with StrictObjectBytes an object over MaxObjectBytes is a limit error.
func TestStrictObjectBytesRefusal(t *testing.T) {
	pdf := scalarPDF("(" + strings.Repeat("A", 1<<20) + ")")
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxObjectBytes = 256 << 10
	conf.Limits.StrictObjectBytes = true
	if _, err := Read(context.Background(), bytes.NewReader(pdf), conf); !errors.Is(err, model.ErrObjectBytesExceeded) || !errors.Is(err, model.ErrLimitExceeded) {
		t.Fatalf("got %v, want ErrObjectBytesExceeded", err)
	}
}

// TestObjectBytesStaysSkippable: without StrictObjectBytes, a caller-set MaxObjectBytes still skips the object.
func TestObjectBytesStaysSkippable(t *testing.T) {
	pdf := scalarPDF("(" + strings.Repeat("A", 1<<20) + ")")
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxObjectBytes = 256 << 10
	if _, err := Read(context.Background(), bytes.NewReader(pdf), conf); errors.Is(err, model.ErrLimitExceeded) {
		t.Fatalf("got %v, want no limit error", err)
	}
}
