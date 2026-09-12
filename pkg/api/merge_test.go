/*
Copyright 2026 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func mergeContextAPICalls(c context.Context) []struct {
	name string
	call func() error
} {
	return []struct {
		name string
		call func() error
	}{
		{"merge raw", func() error { return MergeRaw(c, nil, nil, false, nil) }},
		{"merge", func() error { return Merge(c, "", nil, nil, nil, false) }},
		{"merge create file", func() error { return MergeCreateFile(c, nil, "", false, nil) }},
		{"merge append file", func() error { return MergeAppendFile(c, nil, "", false, nil) }},
		{"merge create zip", func() error { return MergeCreateZip(c, nil, nil, nil, nil) }},
		{"merge create zip file", func() error { return MergeCreateZipFile(c, "", "", "", nil) }},
	}
}

func TestMergeContextAPIsRejectNilContext(t *testing.T) {
	for _, tt := range mergeContextAPICalls(nil) {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, ErrMissingContext) {
				t.Fatalf("got %v, want ErrMissingContext", err)
			}
		})
	}
}

func TestMergeContextAPIsReturnCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	for _, tt := range mergeContextAPICalls(c) {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestMergeCreateFileCancellationPreservesDestination(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(outFile, original, 0o640); err != nil {
		t.Fatal(err)
	}
	c, cancel := context.WithCancel(t.Context())
	cancel()

	if err := MergeCreateFile(c, nil, outFile, false, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bb, original) {
		t.Fatalf("existing output changed: got %q, want %q", bb, original)
	}
}

type mergeValidationModeTest struct {
	name string
	run  func(*model.Configuration) error
}

func mergeValidationModeTests(c context.Context, validFile, relaxedOnlyFile string) []mergeValidationModeTest {
	valid := strictValidationTestPDF()
	relaxedOnly := relaxedOnlyValidationTestPDF()

	return []mergeValidationModeTest{
		{"raw destination", func(conf *model.Configuration) error {
			return MergeRaw(
				c,
				[]io.ReadSeeker{bytes.NewReader(relaxedOnly), bytes.NewReader(valid)},
				io.Discard,
				false,
				conf,
			)
		}},
		{"raw source", func(conf *model.Configuration) error {
			return MergeRaw(
				c,
				[]io.ReadSeeker{bytes.NewReader(valid), bytes.NewReader(relaxedOnly)},
				io.Discard,
				false,
				conf,
			)
		}},
		{"create destination", func(conf *model.Configuration) error {
			return Merge(c, "", []string{relaxedOnlyFile, validFile}, io.Discard, conf, false)
		}},
		{"create source", func(conf *model.Configuration) error {
			return Merge(c, "", []string{validFile, relaxedOnlyFile}, io.Discard, conf, false)
		}},
		{"append destination", func(conf *model.Configuration) error {
			return Merge(c, relaxedOnlyFile, []string{validFile}, io.Discard, conf, false)
		}},
		{"append source", func(conf *model.Configuration) error {
			return Merge(c, validFile, []string{relaxedOnlyFile}, io.Discard, conf, false)
		}},
		{"zip destination", func(conf *model.Configuration) error {
			return MergeCreateZip(c, bytes.NewReader(relaxedOnly), bytes.NewReader(valid), io.Discard, conf)
		}},
		{"zip source", func(conf *model.Configuration) error {
			return MergeCreateZip(c, bytes.NewReader(valid), bytes.NewReader(relaxedOnly), io.Discard, conf)
		}},
	}
}

func mergeValidationConfiguration(mode int) *model.Configuration {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = mode
	conf.CreateBookmarks = false
	conf.OptimizeBeforeWriting = false
	return conf
}

func TestMergeHonorsValidationModeForEverySubject(t *testing.T) {
	tmpDir := t.TempDir()
	validFile := filepath.Join(tmpDir, "valid.pdf")
	relaxedOnlyFile := filepath.Join(tmpDir, "relaxed-only.pdf")
	if err := os.WriteFile(validFile, strictValidationTestPDF(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(relaxedOnlyFile, relaxedOnlyValidationTestPDF(), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tt := range mergeValidationModeTests(t.Context(), validFile, relaxedOnlyFile) {
		t.Run(tt.name, func(t *testing.T) {
			strict := mergeValidationConfiguration(model.ValidationStrict)
			err := tt.run(strict)
			var validationErr *model.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("strict mode: expected validation error, got %v", err)
			}
			if strict.ValidationMode != model.ValidationStrict {
				t.Fatalf("caller validation mode: got %d, want %d", strict.ValidationMode, model.ValidationStrict)
			}

			relaxed := mergeValidationConfiguration(model.ValidationRelaxed)
			if err := tt.run(relaxed); err != nil {
				t.Fatalf("relaxed mode: %v", err)
			}
			if relaxed.ValidationMode != model.ValidationRelaxed {
				t.Fatalf("caller validation mode: got %d, want %d", relaxed.ValidationMode, model.ValidationRelaxed)
			}
		})
	}
}

// TestMergeAppendFileFailurePreservesExistingOutput verifies staged append publication.
func TestMergeAppendFileFailurePreservesExistingOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "append.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(outFile, original, 0640); err != nil {
		t.Fatal(err)
	}

	if err := MergeAppendFile(t.Context(), nil, outFile, false, nil); err == nil {
		t.Fatal("expected merge-append failure")
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bb, original) {
		t.Fatalf("existing output changed: got %q, want %q", bb, original)
	}
}

func TestMergeAppendFileCancellationPreservesDestination(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "append.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(outFile, original, 0o640); err != nil {
		t.Fatal(err)
	}
	c, cancel := context.WithCancel(t.Context())
	cancel()

	if err := MergeAppendFile(c, nil, outFile, false, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bb, original) {
		t.Fatalf("existing output changed: got %q, want %q", bb, original)
	}
}

func TestMergeCreateZipFileCancellationPreservesDestination(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "zip.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(outFile, original, 0o640); err != nil {
		t.Fatal(err)
	}
	c, cancel := context.WithCancel(t.Context())
	cancel()

	if err := MergeCreateZipFile(c, "", "", outFile, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bb, original) {
		t.Fatalf("existing output changed: got %q, want %q", bb, original)
	}
}
