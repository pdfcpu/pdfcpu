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

package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReplaceFileReplacesExistingDestination verifies successful replacement publication.
func TestReplaceFileReplacesExistingDestination(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	destination := filepath.Join(dir, "destination")
	if err := os.WriteFile(source, []byte("replacement"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("previous"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ReplaceFile(source, destination); err != nil {
		t.Fatal(err)
	}
	bb, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(bb), "replacement"; got != want {
		t.Fatalf("destination: got %q, want %q", got, want)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source after replacement: got %v, want not exist", err)
	}
}

// TestReplaceFileFailurePreservesDestination verifies failed publication leaves the destination intact.
func TestReplaceFileFailurePreservesDestination(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "destination")
	if err := os.WriteFile(destination, []byte("previous"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ReplaceFile(filepath.Join(dir, "missing"), destination); err == nil {
		t.Fatal("expected replacement failure")
	}
	bb, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(bb), "previous"; got != want {
		t.Fatalf("destination after failure: got %q, want %q", got, want)
	}
}

// TestRemoveFileIgnoresMissingPath verifies idempotent transaction cleanup.
func TestRemoveFileIgnoresMissingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact")
	if err := RemoveFile(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("artifact"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RemoveFile(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected removed artifact, got %v", err)
	}
}

// TestSyncDirectory verifies transaction metadata can be flushed.
func TestSyncDirectory(t *testing.T) {
	if err := SyncDirectory(t.TempDir()); err != nil {
		t.Fatal(err)
	}
}
