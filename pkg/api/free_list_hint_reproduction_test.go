package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/umats/pdfcpu/pkg/filter"
	"github.com/umats/pdfcpu/pkg/pdfcpu/model"
	"github.com/umats/pdfcpu/pkg/pdfcpu/types"
)

// bug013HeadGeneration changes only five-byte generation fields in both classic
// xrefs. Object bytes, live offsets, linearization ranges and file size are kept.
func bug013HeadGeneration(t *testing.T, pdf []byte) []byte {
	t.Helper()
	before := []byte("0000000000 65535 f \n")
	after := []byte("0000000000 00000 f \n")
	if bytes.Count(pdf, before) != 2 {
		t.Fatal("fixture must have exactly two canonical object-0 records")
	}
	out := bytes.ReplaceAll(pdf, before, after)
	if len(out) != len(pdf) || !bytes.Equal(bytes.ReplaceAll(out, after, before), pdf) {
		t.Fatal("head generation mutation changed unrelated bytes")
	}
	return out
}

// Originally RED on release 495db4e0: narrow relaxed-mode compatibility
// preserves metadata and decrypt for this input.
// This is synthetic reader-order coverage, not Annex F or private-sample proof.
func TestBUG013HeadGenerationCompatibilityReproduction(t *testing.T) {
	pdf, _, content := plaintextPrimaryHintPDF(t)
	pdf = bug013HeadGeneration(t, pdf)
	t.Run("metadata", func(t *testing.T) {
		info, err := PDFInfo(t.Context(), bytes.NewReader(pdf), "synthetic.pdf", nil, false, encryptedXRefReadConfig("user"))
		if err != nil {
			t.Fatalf("generation-only head normalization rejected metadata: %v", err)
		}
		if info.Title != encryptedXRefTitle {
			t.Fatalf("unexpected metadata title %q", info.Title)
		}
	})
	t.Run("decrypt", func(t *testing.T) {
		var out bytes.Buffer
		if err := Decrypt(t.Context(), bytes.NewReader(pdf), &out, encryptedXRefReadConfig("user")); err != nil {
			t.Fatalf("generation-only head normalization rejected decrypt: %v", err)
		}
		assertEncryptedXRefState(t, readEncryptedXRefState(t, out.Bytes(), ""), content, false)
	})
}

// Head normalization must not mask independently unsafe origins or identities.
func TestBUG013UnsafeNormalizationControls(t *testing.T) {
	for _, tc := range []struct {
		name string
		opt  primaryHintFixtureOptions
	}{
		{"live-offset-rebuild", primaryHintFixtureOptions{corruptOffset: true}},
		{"skipped-entry", primaryHintFixtureOptions{unreliableXRef: "skipped-entry"}},
		{"duplicate-identity", primaryHintFixtureOptions{unreliableXRef: "duplicate-classic"}},
		{"ambiguous-syntax", primaryHintFixtureOptions{ambiguousOrigin: "ordinary"}},
		{"referenced-hint", primaryHintFixtureOptions{referenced: true}},
		{"in-use-head", primaryHintFixtureOptions{}},
		{"malformed-head-link", primaryHintFixtureOptions{}},
		{"generation-one", primaryHintFixtureOptions{}},
		{"wrong-hint-generation", primaryHintFixtureOptions{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pdf, _, _ := plaintextPrimaryHintPDF(t, tc.opt)
			pdf = bug013HeadGeneration(t, pdf)
			switch tc.name {
			case "in-use-head":
				pdf = bytes.ReplaceAll(pdf, []byte("0000000000 00000 f \n"), []byte("0000000000 00000 n \n"))
			case "malformed-head-link":
				pdf = bytes.ReplaceAll(pdf, []byte("0000000000 00000 f \n"), []byte("0000000001 00000 f \n"))
			case "generation-one":
				pdf = bytes.ReplaceAll(pdf, []byte("0000000000 00000 f \n"), []byte("0000000000 00001 f \n"))
			case "wrong-hint-generation":
				if bytes.Count(pdf, []byte("8 0 obj")) != 1 {
					t.Fatal("missing hint header")
				}
				pdf = bytes.Replace(pdf, []byte("8 0 obj"), []byte("8 1 obj"), 1)
			}
			_, err := ReadContext(t.Context(), bytes.NewReader(pdf), encryptedXRefReadConfig("user"))
			if err == nil || !strings.Contains(err.Error(), "ciphertext not a multiple of block size") {
				t.Fatalf("unsafe origin should retain terminal ordinary AES error, got %v", err)
			}
		})
	}
}

// Ordinary encrypted hints allow the public reader to expose the normalization
// result without changing cipher policy or bypassing the eligibility flag.
func TestBUG013HeadGenerationCipherControl(t *testing.T) {
	pdf, _, content := plaintextPrimaryHintPDF(t, primaryHintFixtureOptions{encrypted: true})
	before, err := ReadContext(t.Context(), bytes.NewReader(pdf), encryptedXRefReadConfig("user"))
	if err != nil {
		t.Fatal(err)
	}
	after, err := ReadContext(t.Context(), bytes.NewReader(bug013HeadGeneration(t, pdf)), encryptedXRefReadConfig("user"))
	if err != nil {
		t.Fatal(err)
	}
	head := after.Table[0]
	if before.Read.RepairedXRef || after.Read.RepairedXRef || after.Read.RepairOffset != 0 ||
		!head.Free || *head.Offset != 0 || *head.Generation != types.FreeHeadGeneration {
		t.Fatal("did not isolate head-generation normalization with zero repair offset")
	}
	for nr, e := range before.Table {
		if nr == 0 {
			continue
		}
		other := after.Table[nr]
		if other == nil || e.Free != other.Free || e.Compressed != other.Compressed ||
			*e.Offset != *other.Offset || *e.Generation != *other.Generation {
			t.Fatalf("non-head identity changed for object %d", nr)
		}
		if e.Free {
			t.Fatalf("unexpected neighboring free object %d in minimal fixture", nr)
		}
	}
	if len(before.Table) != len(after.Table) || len(after.LinearizationObjs) == 0 || len(before.LinearizationObjs) != len(after.LinearizationObjs) {
		t.Fatal("normalization changed the verified hint inventory")
	}
	for nr, disposable := range before.LinearizationObjs {
		if after.LinearizationObjs[nr] != disposable {
			t.Fatalf("verified hint inventory changed for object %d", nr)
		}
	}
	assertEncryptedXRefState(t, readEncryptedXRefState(t, bug013HeadGeneration(t, pdf), "user"), content, true)
}

func TestBUG013HeadGenerationLimitsAndPolicy(t *testing.T) {
	pdf, _, _ := plaintextPrimaryHintPDF(t)
	pdf = bug013HeadGeneration(t, pdf)
	conf := encryptedXRefReadConfig("user")
	conf.Limits.MaxDecodeBytes = 100
	if _, err := ReadContext(t.Context(), bytes.NewReader(pdf), conf); !errors.Is(err, filter.ErrDecodeLimitExceeded) {
		t.Fatalf("decode limit: %v", err)
	}
	conf = encryptedXRefReadConfig("user")
	conf.Limits.MaxStreamBytes = 100
	if _, err := ReadContext(t.Context(), bytes.NewReader(pdf), conf); err == nil {
		t.Fatal("stream limit ignored")
	}
	c, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := ReadContext(c, bytes.NewReader(pdf), encryptedXRefReadConfig("user")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	conf = encryptedXRefReadConfig("user")
	conf.ValidationMode = model.ValidationStrict
	if _, err := ReadContext(t.Context(), bytes.NewReader(pdf), conf); err == nil {
		t.Fatal("strict plaintext hint accepted")
	}
}

// Shorten only the declared length of an ordinary encrypted stream, preserving
// all physical offsets. A successfully discovered hint cannot authorize it.
func TestBUG013OrdinaryNonalignedAES(t *testing.T) {
	pdf, _, _ := plaintextPrimaryHintPDF(t)
	pdf = bug013HeadGeneration(t, pdf)
	re := regexp.MustCompile(`/Length ([0-9]+)[^\n]*>>\nstream\n`)
	matches := re.FindAllSubmatchIndex(pdf, -1)
	changed := false
	for _, m := range matches {
		n, err := strconv.Atoi(string(pdf[m[2]:m[3]]))
		if err != nil {
			t.Fatal(err)
		}
		// The plaintext hint is nonaligned; ordinary AES includes its IV and
		// padded ciphertext, and therefore has a block-aligned serialized length.
		if n < 32 || n%16 != 0 {
			continue
		}
		replacement := fmt.Sprintf("%*d", m[3]-m[2], n-1)
		if len(replacement) != m[3]-m[2] {
			t.Fatal("length width changed")
		}
		copy(pdf[m[2]:m[3]], replacement)
		changed = true
		break
	}
	if !changed {
		t.Fatal("missing ordinary encrypted stream")
	}
	_, err := ReadContext(t.Context(), bytes.NewReader(pdf), encryptedXRefReadConfig("user"))
	if err == nil || !strings.Contains(err.Error(), "ciphertext not a multiple of block size") {
		t.Fatalf("ordinary AES was not terminal: %v", err)
	}
}

func TestBUG013LaterRepairInvalidatesInventory(t *testing.T) {
	for _, overflow := range []bool{false, true} {
		pdf, _, content := plaintextPrimaryHintPDF(t, primaryHintFixtureOptions{encrypted: true, overflow: overflow, corruptOffset: true})
		pdf = bug013HeadGeneration(t, pdf)
		ctx, err := ReadContext(t.Context(), bytes.NewReader(pdf), encryptedXRefReadConfig("user"))
		if err != nil {
			t.Fatal(err)
		}
		if !ctx.Read.RepairedXRef || len(ctx.LinearizationObjs) != 0 {
			t.Fatal("later live-offset rebuild retained authority")
		}
		assertEncryptedXRefState(t, readEncryptedXRefState(t, pdf, "user"), content, true)
	}
}
