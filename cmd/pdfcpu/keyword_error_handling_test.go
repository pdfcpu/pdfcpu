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

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func renderKeywordCommandError(t *testing.T, err error) string {
	t.Helper()
	return captureStderr(t, func() {
		printError(commandError(err))
	})
}

func copyKeywordCommandTestPDF(t *testing.T) string {
	t.Helper()
	source := filepath.Join("..", "..", "pkg", "samples", "create", "primitives", "textAndAlignment.pdf")
	bb, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	inFile := filepath.Join(t.TempDir(), "in.pdf")
	if err := os.WriteFile(inFile, bb, 0o600); err != nil {
		t.Fatal(err)
	}
	return inFile
}

// TestKeywordCommandsMalformedInputPreservesCause verifies add/remove command error chains and rendering.
func TestKeywordCommandsMalformedInputPreservesCause(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "malformed.pdf")
	if err := os.WriteFile(inFile, []byte("not a PDF"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		op   string
		run  func(*model.Configuration, []string) error
	}{
		{name: "add", op: "add keywords", run: handleAddKeywordsCommand},
		{name: "remove", op: "remove keywords", run: handleRemoveKeywordsCommand},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(model.NewDefaultConfiguration(), []string{inFile, "keyword"})
			if !errors.Is(err, pdfcpu.ErrCorruptHeader) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrCorruptHeader, err)
			}
			rendered := renderKeywordCommandError(t, err)
			if got := strings.Count(rendered, tt.op+":"); got != 1 {
				t.Fatalf("operation prefix count: got %d, want 1 in %q", got, rendered)
			}
		})
	}
}

// TestRemoveKeywordsCommandNoMatchPreservesCause verifies command rendering retains the API sentinel.
func TestRemoveKeywordsCommandNoMatchPreservesCause(t *testing.T) {
	inFile := copyKeywordCommandTestPDF(t)
	err := handleRemoveKeywordsCommand(
		model.NewDefaultConfiguration(),
		[]string{inFile, "pdfcpu-keyword-that-does-not-exist"},
	)
	if !errors.Is(err, api.ErrNoKeywordRemoved) {
		t.Fatalf("expected %v, got %v", api.ErrNoKeywordRemoved, err)
	}
	rendered := renderKeywordCommandError(t, err)
	if got := strings.Count(rendered, "remove keywords:"); got != 1 {
		t.Fatalf("operation prefix count: got %d, want 1 in %q", got, rendered)
	}
}
