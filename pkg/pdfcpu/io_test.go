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

package pdfcpu

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteOverwriteFailurePreservesDestination verifies staged overwrite publication.
func TestWriteOverwriteFailurePreservesDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.bin")
	original := []byte("existing output")
	if err := os.WriteFile(path, original, 0640); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("read failed")

	ok, err := Write(&writeReaderErrorReader{err: wantErr}, path, true)
	if !ok {
		t.Fatal("expected overwrite attempt")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("existing output changed: got %q, want %q", got, original)
	}
}

// TestWriteWithoutOverwriteLeavesDestination verifies exclusive publication.
func TestWriteWithoutOverwriteLeavesDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.bin")
	original := []byte("existing output")
	if err := os.WriteFile(path, original, 0640); err != nil {
		t.Fatal(err)
	}

	ok, err := Write(strings.NewReader("replacement"), path, false)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected existing destination to reject publication")
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("existing output changed: got %q, want %q", got, original)
	}
}

// TestCopyFileOverwriteReplacesDestination verifies copy publication.
func TestCopyFileOverwriteReplacesDestination(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.bin")
	destination := filepath.Join(dir, "destination.bin")
	if err := os.WriteFile(source, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("destination"), 0640); err != nil {
		t.Fatal(err)
	}

	ok, err := CopyFile(source, destination, true)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected destination replacement")
	}
	got, readErr := os.ReadFile(destination)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "source" {
		t.Fatalf("got %q, want source", got)
	}
}
