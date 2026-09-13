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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestOptimizeFileInputLimitPreservesDestination verifies overflow cannot publish an output.
func TestOptimizeFileInputLimitPreservesDestination(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(out, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	conf := model.NewStatelessConfiguration()
	conf.Limits.MaxInputBytes = 1
	err := OptimizeFile(t.Context(), "../testdata/test.pdf", out, conf, nil)
	if !errors.Is(err, model.ErrInputSizeLimit) {
		t.Fatalf("got %v, want size limit", err)
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "original" {
		t.Fatalf("destination: %q, %v", data, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("unexpected staging files: %v, %v", entries, err)
	}
}

// TestMergeInputLimitIsPerSource verifies input sizes are not accumulated across merge sources.
func TestMergeInputLimitIsPerSource(t *testing.T) {
	data, err := os.ReadFile("../testdata/test.pdf")
	if err != nil {
		t.Fatal(err)
	}
	conf := model.NewStatelessConfiguration()
	conf.Limits.MaxInputBytes = int64(len(data))
	var out bytes.Buffer
	sources := []io.ReadSeeker{bytes.NewReader(data), bytes.NewReader(data)}
	if err := MergeRaw(t.Context(), sources, &out, false, conf); err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 {
		t.Fatal("missing merged PDF")
	}
	sources = []io.ReadSeeker{bytes.NewReader(data), bytes.NewReader(append(data, '\n'))}
	err = MergeRaw(t.Context(), sources, io.Discard, false, conf)
	if !errors.Is(err, model.ErrInputSizeLimit) {
		t.Fatalf("oversized second source: %v", err)
	}
}
