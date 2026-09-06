//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

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
	"syscall"
	"testing"
)

func groupOf(t *testing.T, path string) uint32 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Sys().(*syscall.Stat_t).Gid
}

func groupReplacementFiles(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	source, destination := filepath.Join(dir, "staged"), filepath.Join(dir, "destination")
	for _, path := range []string{source, destination} {
		if err := os.WriteFile(path, []byte(filepath.Base(path)), 0640); err != nil {
			t.Fatal(err)
		}
	}
	return source, destination
}

func useDifferentGroup(t *testing.T, source, destination string) {
	t.Helper()
	groups, err := os.Getgroups()
	if err != nil {
		t.Fatal(err)
	}
	want := groupOf(t, destination)
	for _, gid := range groups {
		if uint32(gid) != want && os.Chown(source, -1, gid) == nil {
			return
		}
	}
	t.Skip("requires permission to assign a second group")
}

// TestReplaceFilePreservesGroup verifies replacement retains shared-group access and staged permission bits.
func TestReplaceFilePreservesGroup(t *testing.T) {
	source, destination := groupReplacementFiles(t)
	useDifferentGroup(t, source, destination)
	wantGroup := groupOf(t, destination)
	if err := os.Chmod(source, 0640); err != nil {
		t.Fatal(err)
	}
	if err := ReplaceFile(source, destination); err != nil {
		t.Fatal(err)
	}
	if got := groupOf(t, destination); got != wantGroup {
		t.Fatalf("group: got %d, want %d", got, wantGroup)
	}
	info, err := os.Stat(destination)
	if err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("replacement permissions: %v, %v", info, err)
	}
}

// TestPreserveGroupFailure verifies ownership errors are returned before destination mutation.
func TestPreserveGroupFailure(t *testing.T) {
	source, destination := groupReplacementFiles(t)
	useDifferentGroup(t, source, destination)
	err := preserveGroup(source, destination, func(string, int, int) error { return syscall.EPERM })
	if !errors.Is(err, syscall.EPERM) {
		t.Fatalf("expected ownership error, got %v", err)
	}
	bb, err := os.ReadFile(destination)
	if err != nil || string(bb) != "destination" {
		t.Fatalf("destination changed: %q, %v", bb, err)
	}
}

// TestPreserveGroupAvoidsUnnecessaryChown verifies ordinary replacements need no ownership privilege.
func TestPreserveGroupAvoidsUnnecessaryChown(t *testing.T) {
	source, destination := groupReplacementFiles(t)
	chown := func(string, int, int) error {
		t.Fatal("unexpected ownership change")
		return nil
	}
	if err := preserveGroup(source, destination, chown); err != nil {
		t.Fatal(err)
	}
	if err := preserveGroup(source, destination+"-missing", chown); err != nil {
		t.Fatal(err)
	}
}
