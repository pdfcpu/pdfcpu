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
	"os"
	"path/filepath"
	"testing"
)

func closeAndRemove(t *testing.T, f *os.File, name string) {
	t.Helper()
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
}

// TestCreateOutputFileInPlace verifies secure temporary output creation.
func TestCreateOutputFileInPlace(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(inFile, []byte("input"), 0640); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, inFile+".tmp"); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	f1, name1, err := createOutputFile(inFile, "")
	if err != nil {
		t.Fatal(err)
	}
	defer closeAndRemove(t, f1, name1)

	f2, name2, err := createOutputFile(inFile, inFile)
	if err != nil {
		t.Fatal(err)
	}
	defer closeAndRemove(t, f2, name2)

	if name1 == name2 || name1 == inFile+".tmp" || name2 == inFile+".tmp" {
		t.Fatalf("insecure temporary names: %q, %q", name1, name2)
	}
	if filepath.Dir(name1) != dir || filepath.Dir(name2) != dir {
		t.Fatalf("temporary files not created beside input: %q, %q", name1, name2)
	}

	fi, err := f1.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := fi.Mode().Perm(), os.FileMode(0640); got != want {
		t.Fatalf("got mode %v, want %v", got, want)
	}

	bb, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(bb), "untouched"; got != want {
		t.Fatalf("symlink target got %q, want %q", got, want)
	}
}

// TestCreateOutputFileExplicit verifies explicit output path handling.
func TestCreateOutputFileExplicit(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(inFile, []byte("input"), 0600); err != nil {
		t.Fatal(err)
	}

	f, name, err := createOutputFile(inFile, outFile)
	if err != nil {
		t.Fatal(err)
	}
	if name != outFile {
		t.Fatalf("got %q, want %q", name, outFile)
	}
	closeAndRemove(t, f, name)
}
