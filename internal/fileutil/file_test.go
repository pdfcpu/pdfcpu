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
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
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

// TestWriteFileFailurePreservesDestination verifies a partially written replacement stays private.
func TestWriteFileFailurePreservesDestination(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("write failed")
	err := writeFile(path, 0600, func(f *os.File) error {
		if _, err := f.WriteString("partial"); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("expected write failure, got %v", err)
	}
	bb, err := os.ReadFile(path)
	if err != nil || string(bb) != "original" {
		t.Fatalf("destination changed: %q, %v", bb, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("staging cleanup: %v, %v", entries, err)
	}
	if err := WriteFile(path, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	bb, err = os.ReadFile(path)
	if err != nil || string(bb) != "replacement" {
		t.Fatalf("publication: %q, %v", bb, err)
	}
}

// TestWriteFileStorageFailures verifies partial writes and close failures preserve an existing destination.
func TestWriteFileStorageFailures(t *testing.T) {
	for _, closeFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "disk full", true: "close"}[closeFailure], func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yml")
			if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
				t.Fatal(err)
			}
			want := error(syscall.ENOSPC)
			if closeFailure {
				want = os.ErrClosed
			}
			err := writeFile(path, 0600, func(f *os.File) error {
				if _, err := f.WriteString("partial"); err != nil {
					return err
				}
				if closeFailure {
					return f.Close()
				}
				return &os.PathError{Op: "write", Path: f.Name(), Err: syscall.ENOSPC}
			})
			if !errors.Is(err, want) {
				t.Fatalf("expected %v, got %v", want, err)
			}
			assertUnchangedOutput(t, dir, path)
		})
	}
}

func assertUnchangedOutput(t *testing.T, dir, path string) {
	t.Helper()
	bb, err := os.ReadFile(path)
	if err != nil || string(bb) != "original" {
		t.Fatalf("destination changed: %q, %v", bb, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("staging cleanup: %v, %v", entries, err)
	}
}

// TestWriteFilePublicationFailureCleansStaging verifies a rejected rename removes only staged output.
func TestWriteFilePublicationFailureCleansStaging(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "destination")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(path, "original")
	if err := os.WriteFile(marker, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("replacement"), 0600); err == nil {
		t.Fatal("expected publication failure")
	}
	assertUnchangedOutput(t, dir, marker)
}

// TestWriteFileUnwritableDirectory verifies a real permission failure preserves the destination without staging debris.
func TestWriteFileUnwritableDirectory(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires Unix directory permissions enforced for a non-root user")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0700); err != nil {
			t.Error(err)
		}
	})
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("replacement"), 0600); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("expected permission error, got %v", err)
	}
	assertUnchangedOutput(t, dir, path)
}
