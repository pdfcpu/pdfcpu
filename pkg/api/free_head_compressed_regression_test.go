package api

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/umats/pdfcpu/pkg/pdfcpu/model"
)

// This synthetic reader-order fixture is not an Annex F conformance assertion.
func TestBUG013CompressedHeadGenerationCompatibility(t *testing.T) {
	DisableConfigDir()
	pdf, err := os.ReadFile("testdata/free_head_compressed.pdf")
	if err != nil {
		t.Fatal(err)
	}
	conf := model.NewDefaultConfiguration()
	conf.UserPW, conf.OwnerPW = "user", "user"
	conf.ValidationMode = model.ValidationRelaxed
	info, err := PDFInfo(context.Background(), bytes.NewReader(pdf), "synthetic.pdf", nil, false, conf)
	if err != nil {
		t.Fatal(err)
	}
	if info.PageCount != 1 || !info.Encrypted {
		t.Fatal("source document identity changed")
	}
	var out bytes.Buffer
	if err := Decrypt(context.Background(), bytes.NewReader(pdf), &out, conf); err != nil {
		t.Fatal(err)
	}
	info, err = PDFInfo(context.Background(), bytes.NewReader(out.Bytes()), "", nil, false, nil)
	if err != nil || info.PageCount != 1 || info.Encrypted {
		t.Fatalf("decrypted output: %v", err)
	}
	if err := Validate(context.Background(), bytes.NewReader(out.Bytes()), nil, nil); err != nil {
		t.Fatal(err)
	}
}
