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

package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestCopyInputLimit verifies exact boundaries and keeps the overflow probe out of the spool.
func TestCopyInputLimit(t *testing.T) {
	for _, limit := range []int64{0, 3, 4, 5} {
		r := strings.NewReader("data")
		var out bytes.Buffer
		n, err := copyInput(&out, r, limit)
		if limit == 3 {
			if !errors.Is(err, model.ErrInputSizeLimit) || n != 3 || out.String() != "dat" || r.Len() != 0 {
				t.Fatalf("overflow: n=%d, output=%q, remaining=%d, err=%v", n, out.String(), r.Len(), err)
			}
		} else if err != nil || n != 4 || out.String() != "data" {
			t.Fatalf("limit %d: n=%d, output=%q, err=%v", limit, n, out.String(), err)
		}
	}
}

// TestStdinInputLimitCleansSpool verifies input overflow does not leave temporary files.
func TestStdinInputLimitCleansSpool(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	conf := model.NewStatelessConfiguration()
	conf.Limits.MaxInputBytes = 3
	in, err := readSeekerFromReader(t.Context(), conf, "limited input", strings.NewReader("data"))
	if in != nil || !errors.Is(err, model.ErrInputSizeLimit) {
		t.Fatalf("got %v, %v", in, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary files: %v, %v", entries, err)
	}
	conf.Limits.MaxInputBytes = -1
	in, err = readSeekerFromReader(t.Context(), conf, "invalid limit", strings.NewReader("data"))
	if in != nil || err == nil {
		t.Fatalf("accepted negative limit: %v, %v", in, err)
	}
	entries, err = os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary files: %v, %v", entries, err)
	}
}

// TestOptimizeStdinInputLimitPreservesDestination verifies command configuration reaches the spooler.
func TestOptimizeStdinInputLimitPreservesDestination(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	source, err := os.CreateTemp(dir, "source-*")
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if _, err := source.WriteString("data"); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = source
	t.Cleanup(func() { os.Stdin = old })
	out := filepath.Join(dir, "output.pdf")
	if err := os.WriteFile(out, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	conf := model.NewStatelessConfiguration()
	conf.Limits.MaxInputBytes = 3
	_, err = Dispatch(t.Context(), OptimizeCommand("-", out, conf))
	if !errors.Is(err, model.ErrInputSizeLimit) {
		t.Fatalf("got %v, want size limit", err)
	}
	got, err := os.ReadFile(out)
	if err != nil || string(got) != "original" {
		t.Fatalf("destination: %q, %v", got, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 2 {
		t.Fatalf("unexpected staging files: %v, %v", entries, err)
	}
}
