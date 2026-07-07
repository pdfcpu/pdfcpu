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
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func keywordTestInputFile() string {
	return filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
}

func copyKeywordTestInput(t *testing.T) string {
	t.Helper()

	b, err := os.ReadFile(keywordTestInputFile())
	if err != nil {
		t.Fatal(err)
	}
	inFile := filepath.Join(t.TempDir(), "in.pdf")
	if err := os.WriteFile(inFile, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return inFile
}

func readKeywordTestFile(t *testing.T, fileName string) []string {
	t.Helper()

	f, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	keywords, err := Keywords(f, nil)
	if err != nil {
		t.Fatal(err)
	}
	return keywords
}

// TestKeywordAPIArgumentErrors verifies public keyword boundary sentinels.
func TestKeywordAPIArgumentErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "list reader", err: func() error {
			_, err := Keywords(nil, nil)
			return err
		}(), want: ErrMissingPDFReadSeeker},
		{name: "add reader", err: AddKeywords(nil, io.Discard, nil, nil), want: ErrMissingPDFReadSeeker},
		{name: "add writer", err: AddKeywords(bytes.NewReader(nil), nil, nil, nil), want: ErrMissingPDFWriter},
		{name: "remove reader", err: RemoveKeywords(nil, io.Discard, nil, nil), want: ErrMissingPDFReadSeeker},
		{name: "remove writer", err: RemoveKeywords(bytes.NewReader(nil), nil, nil, nil), want: ErrMissingPDFWriter},
		{name: "add file input", err: AddKeywordsFile("", "", nil, nil), want: ErrMissingPDFInput},
		{name: "remove file input", err: RemoveKeywordsFile("", "", nil, nil), want: ErrMissingPDFInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, tt.err)
			}
		})
	}
}

// TestRemoveKeywordsNoMatchPreservesSentinel verifies callers can detect a no-op removal.
func TestRemoveKeywordsNoMatchPreservesSentinel(t *testing.T) {
	err := RemoveKeywords(
		openAPITestPDF(t, keywordTestInputFile()),
		io.Discard,
		[]string{"pdfcpu-keyword-that-does-not-exist"},
		nil,
	)
	if !errors.Is(err, ErrNoKeywordRemoved) {
		t.Fatalf("expected %v, got %v", ErrNoKeywordRemoved, err)
	}
	if !strings.Contains(err.Error(), "remove keywords: no keyword removed") {
		t.Fatalf("expected remove operation context, got %v", err)
	}
}

// TestKeywordAPIValidatesKeywordsBeforeReading verifies caller data is rejected before file I/O.
func TestKeywordAPIValidatesKeywordsBeforeReading(t *testing.T) {
	tests := []struct {
		name string
		run  func(string) error
		want string
	}{
		{name: "add stream", run: func(_ string) error {
			return AddKeywords(bytes.NewReader(nil), io.Discard, []string{" "}, nil)
		}, want: "add keywords: validate keywords"},
		{name: "remove stream", run: func(_ string) error {
			return RemoveKeywords(bytes.NewReader(nil), io.Discard, []string{" "}, nil)
		}, want: "remove keywords: validate keywords"},
		{name: "add file", run: func(inFile string) error {
			return AddKeywordsFile(inFile, filepath.Join(t.TempDir(), "out.pdf"), []string{" "}, nil)
		}, want: "add keywords: validate keywords"},
		{name: "remove file", run: func(inFile string) error {
			return RemoveKeywordsFile(inFile, filepath.Join(t.TempDir(), "out.pdf"), []string{" "}, nil)
		}, want: "remove keywords: validate keywords"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(filepath.Join(t.TempDir(), "missing.pdf"))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %v", tt.want, err)
			}
			if errors.Is(err, os.ErrNotExist) {
				t.Fatalf("file I/O occurred before keyword validation: %v", err)
			}
		})
	}
}

// TestKeywordAPIPrepareContextErrors verifies each stream entry point adds operation context.
func TestKeywordAPIPrepareContextErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "list", err: func() error {
			_, err := Keywords(bytes.NewReader(nil), nil)
			return err
		}(), want: "list keywords: prepare PDF context"},
		{name: "add", err: AddKeywords(bytes.NewReader(nil), io.Discard, nil, nil), want: "add keywords: prepare PDF context"},
		{name: "remove", err: RemoveKeywords(bytes.NewReader(nil), io.Discard, nil, nil), want: "remove keywords: prepare PDF context"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil || !strings.Contains(tt.err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %v", tt.want, tt.err)
			}
		})
	}
}

// TestAddKeywordsWriteErrorPreservesCause verifies write failures retain their cause and phase.
func TestAddKeywordsWriteErrorPreservesCause(t *testing.T) {
	want := errors.New("keyword writer failed")
	err := AddKeywords(
		openAPITestPDF(t, keywordTestInputFile()),
		failingWriter{err: want},
		[]string{"keyword"},
		nil,
	)
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
	if !strings.Contains(err.Error(), "add keywords: write output") {
		t.Fatalf("expected write context, got %v", err)
	}
}

// TestKeywordFileOpenAndCreateErrors verifies file setup failures retain their causes and phases.
func TestKeywordFileOpenAndCreateErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.pdf")
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "add open", err: AddKeywordsFile(missing, "", nil, nil), want: "add keywords: open input"},
		{name: "remove open", err: RemoveKeywordsFile(missing, "", nil, nil), want: "remove keywords: open input"},
		{name: "add create", err: AddKeywordsFile(keywordTestInputFile(), filepath.Join(t.TempDir(), "missing", "out.pdf"), nil, nil), want: "add keywords: create output"},
		{name: "remove create", err: RemoveKeywordsFile(keywordTestInputFile(), filepath.Join(t.TempDir(), "missing", "out.pdf"), nil, nil), want: "remove keywords: create output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, tt.err)
			}
			if !strings.Contains(tt.err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %v", tt.want, tt.err)
			}
		})
	}
}

// TestKeywordFileFailurePreservesExistingOutput verifies delayed replacement on processing failure.
func TestKeywordFileFailurePreservesExistingOutput(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "bad.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outFile, want, 0o600); err != nil {
		t.Fatal(err)
	}

	err := AddKeywordsFile(inFile, outFile, []string{"keyword"}, nil)
	if err == nil || !strings.Contains(err.Error(), "add keywords: prepare PDF context") {
		t.Fatalf("expected prepare-context error, got %v", err)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

// TestKeywordFileSuccessReplacesExistingOutput verifies successful delayed destination replacement.
func TestKeywordFileSuccessReplacesExistingOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("existing output"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AddKeywordsFile(keywordTestInputFile(), outFile, []string{"keyword"}, nil); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Fatalf("expected replacement PDF, got %q", b)
	}
}

// TestKeywordFileFailureRemovesNewOutput verifies failed processing removes a newly created destination.
func TestKeywordFileFailureRemovesNewOutput(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "bad.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	err := AddKeywordsFile(inFile, outFile, []string{"keyword"}, nil)
	if err == nil || !strings.Contains(err.Error(), "add keywords: prepare PDF context") {
		t.Fatalf("expected prepare-context error, got %v", err)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected failed output cleanup, got %v", statErr)
	}
}

// TestAddKeywordsFileReplacesInput verifies successful in-place finalization.
func TestAddKeywordsFileReplacesInput(t *testing.T) {
	inFile := copyKeywordTestInput(t)
	if err := AddKeywordsFile(inFile, "", []string{"keyword"}, nil); err != nil {
		t.Fatal(err)
	}
	if keywords := readKeywordTestFile(t, inFile); !slices.Contains(keywords, "keyword") {
		t.Fatalf("updated input is missing keyword: %v", keywords)
	}
}

// TestRemoveKeywordsFileReplacesExistingOutput verifies successful delayed replacement for removal.
func TestRemoveKeywordsFileReplacesExistingOutput(t *testing.T) {
	inFile := copyKeywordTestInput(t)
	if err := AddKeywordsFile(inFile, "", []string{"keyword"}, nil); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("existing output"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RemoveKeywordsFile(inFile, outFile, []string{"keyword"}, nil); err != nil {
		t.Fatal(err)
	}
	if keywords := readKeywordTestFile(t, outFile); slices.Contains(keywords, "keyword") {
		t.Fatalf("output still contains removed keyword: %v", keywords)
	}
}

// TestKeywordFileAliasDoesNotOverwriteInput verifies hard-linked output does not truncate the input.
func TestKeywordFileAliasDoesNotOverwriteInput(t *testing.T) {
	inFile := copyKeywordTestInput(t)
	outFile := filepath.Join(filepath.Dir(inFile), "alias.pdf")
	if err := os.Link(inFile, outFile); err != nil {
		t.Fatal(err)
	}
	if err := AddKeywordsFile(inFile, outFile, []string{"keyword"}, nil); err != nil {
		t.Fatal(err)
	}

	inKeywords := readKeywordTestFile(t, inFile)
	if slices.Contains(inKeywords, "keyword") {
		t.Fatalf("input was overwritten through output alias: %v", inKeywords)
	}

	outKeywords := readKeywordTestFile(t, outFile)
	if !slices.Contains(outKeywords, "keyword") {
		t.Fatalf("output is missing keyword: %v", outKeywords)
	}
}

// TestAddKeywordsDoesNotMutateCallerSlice verifies caller-owned keyword data remains unchanged.
func TestAddKeywordsDoesNotMutateCallerSlice(t *testing.T) {
	keywords := []string{" alpha ", "beta"}
	want := slices.Clone(keywords)
	var out bytes.Buffer
	if err := AddKeywords(openAPITestPDF(t, keywordTestInputFile()), &out, keywords, nil); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(keywords, want) {
		t.Fatalf("keywords mutated: got %q, want %q", keywords, want)
	}
}

// TestFinalizeKeywordFilesReportsCloseErrors verifies close failures retain phase context.
// TestFinalizeKeywordFilesReportsRenameError verifies replacement failures retain context and remove temporary output.
