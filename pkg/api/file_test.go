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
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// TestCloseFile verifies nil handling and contextual close errors.
func TestCloseFile(t *testing.T) {
	if err := closeFile(nil, "close nil"); err != nil {
		t.Fatal(err)
	}
	f, err := os.CreateTemp(t.TempDir(), "close-*")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	err = closeFile(f, "close output")
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected %v, got %v", os.ErrClosed, err)
	}
	if !strings.Contains(err.Error(), "close output") {
		t.Fatalf("expected close context, got %v", err)
	}
}

// TestRemoveFile verifies empty, missing, successful, and failed cleanup.
func TestRemoveFile(t *testing.T) {
	if err := removeFile("", "remove empty"); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := removeFile(filepath.Join(dir, "missing"), "remove missing"); err != nil {
		t.Fatal(err)
	}
	fileName := filepath.Join(dir, "output")
	if err := os.WriteFile(fileName, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := removeFile(fileName, "remove output"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(fileName); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removed output: got %v, want not exist", err)
	}

	nonEmptyDir := filepath.Join(dir, "non-empty")
	if err := os.Mkdir(nonEmptyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "entry"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	err := removeFile(nonEmptyDir, "remove temporary output")
	if err == nil {
		t.Fatal("expected remove failure")
	}
	if !strings.Contains(err.Error(), "remove temporary output") {
		t.Fatalf("expected remove context, got %v", err)
	}
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("expected path error, got %T", err)
	}
}

// TestReplaceFile verifies contextual replacement and cause preservation.
func TestReplaceFile(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "destination")
	if err := os.WriteFile(destination, []byte("previous"), 0644); err != nil {
		t.Fatal(err)
	}

	err := replaceFile(filepath.Join(dir, "missing"), destination, "replace output")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "replace output") {
		t.Fatalf("expected replace context, got %v", err)
	}
	bb, readErr := os.ReadFile(destination)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got, want := string(bb), "previous"; got != want {
		t.Fatalf("destination after failure: got %q, want %q", got, want)
	}
}

// TestStagedOutputCommitOrder verifies process, close, and publication ordering.
func TestStagedOutputCommitOrder(t *testing.T) {
	input := &os.File{}
	dataInput := &os.File{}
	output := &os.File{}
	events := []string{"process"}

	staged := newStagedOutput(input, output, "temporary", "input.pdf", "", "", "create").
		withInput(dataInput, "create: close JSON input")
	staged.operations = fileOperations{
		closeFn: func(f *os.File) error {
			switch f {
			case output:
				events = append(events, "close output")
			case input:
				events = append(events, "close input")
			case dataInput:
				events = append(events, "close JSON input")
			default:
				t.Fatalf("unexpected file: %p", f)
			}
			return nil
		},
		removeFn: func(string) error {
			events = append(events, "remove")
			return nil
		},
		replaceFn: func(oldName, newName string) error {
			if oldName != "temporary" || newName != "input.pdf" {
				t.Fatalf("commit: got (%q, %q)", oldName, newName)
			}
			events = append(events, "commit")
			return nil
		},
	}

	if err := staged.commit(); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(events, ","), "process,close output,close input,close JSON input,commit"; got != want {
		t.Fatalf("lifecycle: got %q, want %q", got, want)
	}
}

// TestStagedOutputCloseFailureSkipsCommit verifies publication requires successful closes.
func TestStagedOutputCloseFailureSkipsCommit(t *testing.T) {
	input := &os.File{}
	output := &os.File{}
	closeErr := errors.New("close output")
	events := []string{"process"}

	staged := newStagedOutput(input, output, "temporary", "input.pdf", "", "", "trim")
	staged.operations = fileOperations{
		closeFn: func(f *os.File) error {
			if f == output {
				events = append(events, "close output")
				return closeErr
			}
			events = append(events, "close input")
			return nil
		},
		removeFn: func(string) error {
			events = append(events, "remove")
			return nil
		},
		replaceFn: func(string, string) error {
			events = append(events, "commit")
			return nil
		},
	}

	err := staged.commit()
	if !errors.Is(err, closeErr) {
		t.Fatalf("expected %v, got %v", closeErr, err)
	}
	if got, want := strings.Join(events, ","), "process,close output,close input,remove"; got != want {
		t.Fatalf("failed lifecycle: got %q, want %q", got, want)
	}
}

// TestStagedOutputPublicationFailureCleansUp verifies failed publication preserves
// the replacement error and removes the temporary output.
func TestStagedOutputPublicationFailureCleansUp(t *testing.T) {
	input := &os.File{}
	output := &os.File{}
	replaceErr := errors.New("replace failed")
	removeErr := errors.New("remove failed")
	events := []string{"process"}

	staged := newStagedOutput(input, output, "temporary", "input.pdf", "", "", "trim")
	staged.operations = fileOperations{
		closeFn: func(f *os.File) error {
			switch f {
			case output:
				events = append(events, "close output")
			case input:
				events = append(events, "close input")
			default:
				t.Fatalf("unexpected file: %p", f)
			}
			return nil
		},
		removeFn: func(string) error {
			events = append(events, "remove")
			return removeErr
		},
		replaceFn: func(string, string) error {
			events = append(events, "commit")
			return replaceErr
		},
	}

	err := staged.commit()
	if !errors.Is(err, replaceErr) || !errors.Is(err, removeErr) {
		t.Fatalf("expected replacement and cleanup failures, got %v", err)
	}
	if got, want := strings.Join(events, ","), "process,close output,close input,commit,remove"; got != want {
		t.Fatalf("failed lifecycle: got %q, want %q", got, want)
	}
}

// TestOpenStagedOutputFilesystemContract verifies constructor failures through
// the shared filesystem operation contract.
func TestOpenStagedOutputFilesystemContract(t *testing.T) {
	failure := errors.New("filesystem failure")

	t.Run("open", func(t *testing.T) {
		ops := defaultFileOperations()
		ops.openExclusiveFn = func(string, int, os.FileMode) (*os.File, error) {
			return nil, failure
		}
		_, err := openStagedOutputWithOperations(nil, "input.pdf", "output.pdf", "test", ops)
		if !errors.Is(err, failure) {
			t.Fatalf("expected %v, got %v", failure, err)
		}
	})

	t.Run("stat", func(t *testing.T) {
		ops := defaultFileOperations()
		ops.statFn = func(string) (os.FileInfo, error) {
			return nil, failure
		}
		_, err := openStagedOutputWithOperations(nil, "input.pdf", "", "test", ops)
		if !errors.Is(err, failure) {
			t.Fatalf("expected %v, got %v", failure, err)
		}
	})

	t.Run("create temporary", func(t *testing.T) {
		dir := t.TempDir()
		inFile := filepath.Join(dir, "input.pdf")
		if err := os.WriteFile(inFile, nil, 0600); err != nil {
			t.Fatal(err)
		}
		ops := defaultFileOperations()
		ops.createTempFn = func(string, string) (*os.File, error) {
			return nil, failure
		}
		_, err := openStagedOutputWithOperations(nil, inFile, "", "test", ops)
		if !errors.Is(err, failure) {
			t.Fatalf("expected %v, got %v", failure, err)
		}
	})

	t.Run("set permissions", func(t *testing.T) {
		dir := t.TempDir()
		inFile := filepath.Join(dir, "input.pdf")
		if err := os.WriteFile(inFile, nil, 0600); err != nil {
			t.Fatal(err)
		}
		ops := defaultFileOperations()
		var temporaryFile string
		createTemp := ops.createTempFn
		ops.createTempFn = func(dir, pattern string) (*os.File, error) {
			f, err := createTemp(dir, pattern)
			if err == nil {
				temporaryFile = f.Name()
			}
			return f, err
		}
		ops.chmodFn = func(*os.File, os.FileMode) error {
			return failure
		}
		_, err := openStagedOutputWithOperations(nil, inFile, "", "test", ops)
		if !errors.Is(err, failure) {
			t.Fatalf("expected %v, got %v", failure, err)
		}
		if _, err := os.Stat(temporaryFile); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("temporary output after failure: got %v, want not exist", err)
		}
	})
}

func validateTemporaryOutput(t *testing.T, staged stagedOutput, dir string) {
	t.Helper()
	if staged.destination == "" {
		t.Fatal("missing replacement destination")
	}
	f := staged.output.file
	name := staged.temporaryFile
	if filepath.Dir(name) != dir {
		t.Fatalf("temporary file not created beside input: %q", name)
	}
	fi, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := fi.Mode().Perm(), os.FileMode(0640); got != want {
		t.Fatalf("got mode %v, want %v", got, want)
	}
}

// TestOpenStagedOutputInPlace verifies secure temporary output creation.
func TestOpenStagedOutputInPlace(t *testing.T) {
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

	staged1, err := openStagedOutput(nil, inFile, "", "test")
	if err != nil {
		t.Fatal(err)
	}
	f1, name1 := staged1.output.file, staged1.temporaryFile
	defer closeAndRemove(t, f1, name1)
	validateTemporaryOutput(t, staged1, dir)

	staged2, err := openStagedOutput(nil, inFile, inFile, "test")
	if err != nil {
		t.Fatal(err)
	}
	f2, name2 := staged2.output.file, staged2.temporaryFile
	defer closeAndRemove(t, f2, name2)
	validateTemporaryOutput(t, staged2, dir)

	if name1 == name2 || name1 == inFile+".tmp" || name2 == inFile+".tmp" {
		t.Fatalf("insecure temporary names: %q, %q", name1, name2)
	}

	bb, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(bb), "untouched"; got != want {
		t.Fatalf("symlink target got %q, want %q", got, want)
	}
}

// TestOpenStagedOutputExplicit verifies explicit output path handling.
func TestOpenStagedOutputExplicit(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(inFile, []byte("input"), 0600); err != nil {
		t.Fatal(err)
	}

	staged, err := openStagedOutput(nil, inFile, outFile, "test")
	if err != nil {
		t.Fatal(err)
	}
	f, name := staged.output.file, staged.temporaryFile
	if name != outFile {
		t.Fatalf("got %q, want %q", name, outFile)
	}
	if staged.destination != "" {
		t.Fatalf("unexpected replacement destination %q", staged.destination)
	}
	closeAndRemove(t, f, name)
}

// TestOpenStagedOutputExistingExplicitStagesReplacement verifies non-destructive output publication.
func TestOpenStagedOutputExistingExplicitStagesReplacement(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(inFile, []byte("input"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outFile, []byte("previous"), 0640); err != nil {
		t.Fatal(err)
	}

	staged, err := openStagedOutput(nil, inFile, outFile, "test")
	if err != nil {
		t.Fatal(err)
	}
	f, temporaryFile := staged.output.file, staged.temporaryFile
	if temporaryFile == outFile || filepath.Dir(temporaryFile) != dir {
		t.Fatalf("temporary output %q is not a sibling of %q", temporaryFile, outFile)
	}
	if staged.destination != outFile {
		t.Fatalf("replacement destination: got %q, want %q", staged.destination, outFile)
	}
	if bb, err := os.ReadFile(outFile); err != nil || string(bb) != "previous" {
		t.Fatalf("output before commit: got %q, err=%v", bb, err)
	}
	if _, err := f.Write([]byte("replacement")); err != nil {
		t.Fatal(err)
	}
	if err := staged.commit(); err != nil {
		t.Fatal(err)
	}
	if bb, err := os.ReadFile(outFile); err != nil || string(bb) != "replacement" {
		t.Fatalf("output after commit: got %q, err=%v", bb, err)
	}
	fi, err := os.Stat(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := fi.Mode().Perm(), os.FileMode(0640); got != want {
		t.Fatalf("output mode: got %v, want %v", got, want)
	}
}

// TestOpenStagedOutputExistingExplicitSurvivesProcessingFailure verifies rollback.
func TestOpenStagedOutputExistingExplicitSurvivesProcessingFailure(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(inFile, []byte("input"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outFile, []byte("previous"), 0640); err != nil {
		t.Fatal(err)
	}

	staged, err := openStagedOutput(nil, inFile, outFile, "test")
	if err != nil {
		t.Fatal(err)
	}
	f, temporaryFile := staged.output.file, staged.temporaryFile
	if _, err := f.Write([]byte("partial")); err != nil {
		t.Fatal(err)
	}
	processErr := errors.New("process failed")
	err = staged.cleanup(processErr)
	if !errors.Is(err, processErr) {
		t.Fatalf("expected %v, got %v", processErr, err)
	}
	if bb, err := os.ReadFile(outFile); err != nil || string(bb) != "previous" {
		t.Fatalf("output after rollback: got %q, err=%v", bb, err)
	}
	if _, err := os.Stat(temporaryFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary output after rollback: got %v, want not exist", err)
	}
}
