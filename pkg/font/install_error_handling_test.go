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

package font

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallTrueTypeInputsPreserveOpenErrors(t *testing.T) {
	for _, tt := range []struct {
		name string
		fn   func(string) error
		want string
	}{
		{
			name: "font",
			fn: func(fileName string) error {
				_, err := InstallTrueTypeFont(t.TempDir(), fileName)
				return err
			},
			want: "font ",
		},
		{
			name: "collection",
			fn: func(fileName string) error {
				_, err := InstallTrueTypeCollection(t.TempDir(), fileName)
				return err
			},
			want: "font collection ",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fileName := filepath.Join(t.TempDir(), "missing")
			err := tt.fn(fileName)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), "open") {
				t.Fatalf("expected font/open context, got %q", err)
			}
		})
	}
}

func TestExportedInstallersGuardRequiredInputs(t *testing.T) {
	fontDir := t.TempDir()
	tests := []struct {
		name string
		fn   func() error
		want error
	}{
		{"font directory", func() error {
			_, err := InstallTrueTypeFont("", "missing.ttf")
			return err
		}, ErrMissingFontDir},
		{"font name", func() error {
			_, err := InstallTrueTypeFont(fontDir, " ")
			return err
		}, ErrMissingFontName},
		{"font result directory", func() error {
			_, err := InstallTrueTypeFontResult("", "missing.ttf")
			return err
		}, ErrMissingFontDir},
		{"font result name", func() error {
			_, err := InstallTrueTypeFontResult(fontDir, " ")
			return err
		}, ErrMissingFontName},
		{"collection directory", func() error {
			_, err := InstallTrueTypeCollection("", "missing.ttc")
			return err
		}, ErrMissingFontDir},
		{"collection name", func() error {
			_, err := InstallTrueTypeCollection(fontDir, "")
			return err
		}, ErrMissingFontName},
		{"collection results directory", func() error {
			_, err := InstallTrueTypeCollectionResults("", "missing.ttc")
			return err
		}, ErrMissingFontDir},
		{"collection results name", func() error {
			_, err := InstallTrueTypeCollectionResults(fontDir, "")
			return err
		}, ErrMissingFontName},
		{"bytes directory", func() error { return InstallFontFromBytes("", "demo.ttf", []byte{1}) }, ErrMissingFontDir},
		{"bytes name", func() error { return InstallFontFromBytes(fontDir, "", []byte{1}) }, ErrMissingFontName},
		{"bytes data", func() error { return InstallFontFromBytes(fontDir, "demo.ttf", nil) }, ErrMissingFontData},
		{"quiet directory", func() error { return InstallFontFromBytesQuiet(" ", "demo.ttf", []byte{1}) }, ErrMissingFontDir},
		{"quiet name", func() error { return InstallFontFromBytesQuiet(fontDir, " ", []byte{1}) }, ErrMissingFontName},
		{"quiet data", func() error { return InstallFontFromBytesQuiet(fontDir, "demo.ttf", nil) }, ErrMissingFontData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestInstallRejectsUnsupportedFontFormat(t *testing.T) {
	bb := make([]byte, 12)
	copy(bb, sfntVersionCFF)
	err := InstallFontFromBytes(t.TempDir(), "cff.otf", bb)
	if !errors.Is(err, ErrUnsupportedFontFormat) {
		t.Fatalf("expected %v, got %v", ErrUnsupportedFontFormat, err)
	}
	if errors.Is(err, ErrInvalidFontData) {
		t.Fatalf("unsupported format must remain distinct from %v", ErrInvalidFontData)
	}
}

func TestInstallTrueTypeCollectionPreservesHeaderReadError(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "short.ttc")
	if err := os.WriteFile(fileName, []byte(ttcTag), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := InstallTrueTypeCollection(t.TempDir(), fileName)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected preserved short-read error, got %v", err)
	}
	if !errors.Is(err, ErrInvalidFontData) {
		t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
	}
	if !strings.Contains(err.Error(), "read header") {
		t.Fatalf("expected header context, got %q", err)
	}
}

func TestInstallTrueTypeCollectionNormalizesStructuralErrors(t *testing.T) {
	for _, tt := range []struct {
		name    string
		version uint32
		count   uint32
		want    string
	}{
		{name: "invalid version", version: 3, count: 1, want: "header version"},
		{name: "excessive member count", version: 0x00010000, count: maxFontCollectionFonts + 1, want: "font count"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bb := make([]byte, 12)
			copy(bb, ttcTag)
			binary.BigEndian.PutUint32(bb[4:], tt.version)
			binary.BigEndian.PutUint32(bb[8:], tt.count)
			fileName := filepath.Join(t.TempDir(), "malformed.ttc")
			if err := os.WriteFile(fileName, bb, 0600); err != nil {
				t.Fatal(err)
			}
			_, err := InstallTrueTypeCollection(t.TempDir(), fileName)
			requireInvalidFontData(t, err, tt.want)
		})
	}
}

func TestCollectionCommitRollbackRestoresExistingFonts(t *testing.T) {
	fontDir := t.TempDir()
	stagingDir := t.TempDir()
	for _, file := range []struct {
		dir, name, value string
	}{
		{fontDir, "One.gob", "old one"},
		{fontDir, "Two.gob", "old two"},
		{stagingDir, "One.gob", "new one"},
		{stagingDir, "Two.gob", "new two"},
	} {
		if err := os.WriteFile(filepath.Join(file.dir, file.name), []byte(file.value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	wantErr := errors.New("commit failed")
	ops := defaultCollectionInstallFileOperations()
	rename := ops.rename
	ops.rename = func(oldPath, newPath string) error {
		if oldPath == filepath.Join(stagingDir, "Two.gob") {
			return wantErr
		}
		return rename(oldPath, newPath)
	}
	err := commitCollectionFonts(fontDir, stagingDir, []InstallResult{{PostScriptName: "One"}, {PostScriptName: "Two"}}, ops)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	for _, file := range []struct {
		name, value string
	}{
		{"One.gob", "old one"},
		{"Two.gob", "old two"},
	} {
		bb, readErr := os.ReadFile(filepath.Join(fontDir, file.name))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(bb) != file.value {
			t.Fatalf("%s: expected restored content %q, got %q", file.name, file.value, bb)
		}
	}
}

func TestCollectionCommitSyncFailureJoinsRollbackFailure(t *testing.T) {
	fontDir := t.TempDir()
	stagingDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(fontDir, "Same.gob"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "Same.gob"), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	syncErr := errors.New("directory sync failed")
	rollbackErr := errors.New("restore rename failed")
	ops := defaultCollectionInstallFileOperations()
	rename := ops.rename
	ops.syncDir = func(string) error { return syncErr }
	ops.rename = func(source, target string) error {
		if strings.Contains(source, ".pdfcpu-font-backup-") {
			return rollbackErr
		}
		return rename(source, target)
	}
	err := commitCollectionFonts(fontDir, stagingDir, []InstallResult{{PostScriptName: "Same"}}, ops)
	if !errors.Is(err, syncErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("expected joined sync and rollback errors, got %v", err)
	}
	if !strings.Contains(err.Error(), "sync directory") || !strings.Contains(err.Error(), "font backup retained") {
		t.Fatalf("expected sync and retained-backup context, got %q", err)
	}
}

func TestCollectionCommitFailureJoinsRollbackSyncFailure(t *testing.T) {
	fontDir := t.TempDir()
	stagingDir := t.TempDir()
	for _, file := range []struct {
		dir, name, value string
	}{
		{fontDir, "One.gob", "old one"},
		{fontDir, "Two.gob", "old two"},
		{stagingDir, "One.gob", "new one"},
		{stagingDir, "Two.gob", "new two"},
	} {
		if err := os.WriteFile(filepath.Join(file.dir, file.name), []byte(file.value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	commitErr := errors.New("commit rename failed")
	rollbackSyncErr := errors.New("rollback directory sync failed")
	failSync := false
	ops := defaultCollectionInstallFileOperations()
	rename := ops.rename
	ops.syncDir = func(string) error {
		if failSync {
			return rollbackSyncErr
		}
		return nil
	}
	ops.rename = func(source, target string) error {
		if source == filepath.Join(stagingDir, "Two.gob") {
			failSync = true
			return commitErr
		}
		return rename(source, target)
	}
	err := commitCollectionFonts(fontDir, stagingDir, []InstallResult{{PostScriptName: "One"}, {PostScriptName: "Two"}}, ops)
	if !errors.Is(err, commitErr) || !errors.Is(err, rollbackSyncErr) {
		t.Fatalf("expected joined commit and rollback-sync errors, got %v", err)
	}
	if !strings.Contains(err.Error(), "font backup retained") {
		t.Fatalf("expected retained-backup context, got %q", err)
	}
}

func TestTrueTypeSourceCloseFailurePreventsPublication(t *testing.T) {
	fontDir := t.TempDir()
	source := filepath.Join(t.TempDir(), "source.ttf")
	if err := os.WriteFile(source, nil, 0600); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("close failed")
	installCalled := false
	ops := defaultTrueTypeFontInstallOperations()
	ops.headerAndTables = func(io.ReaderAt, int64, int64) ([]byte, map[string]*table, error) {
		return []byte("header"), map[string]*table{}, nil
	}
	ops.close = func(f *os.File) error {
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		return wantErr
	}
	ops.installRep = func(string, string, []byte, map[string]*table, bool, func(string) error) (string, error) {
		installCalled = true
		return "Published", os.WriteFile(filepath.Join(fontDir, "Published.gob"), []byte("published"), 0600)
	}

	result, err := installTrueTypeFontResult(fontDir, source, ops)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if len(result.Fonts) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("expected no result, got %+v", result)
	}
	if installCalled {
		t.Fatal("install representation called after source close failure")
	}
	if _, statErr := os.Stat(filepath.Join(fontDir, "Published.gob")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected no published font, got %v", statErr)
	}
}

func TestCollectionSourceCloseFailurePreventsPublication(t *testing.T) {
	fontDir := t.TempDir()
	source := filepath.Join(t.TempDir(), "source.ttc")
	bb := make([]byte, 28)
	copy(bb, ttcTag)
	binary.BigEndian.PutUint32(bb[4:], 0x00010000)
	binary.BigEndian.PutUint32(bb[8:], 1)
	binary.BigEndian.PutUint32(bb[12:], 16)
	if err := os.WriteFile(source, bb, 0600); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("close failed")
	ops := defaultCollectionInstallFileOperations()
	ops.stageMembers = func(_ *os.File, stagingDir, _ string, _ int64, _ uint32, _ int64) ([]InstallResult, error) {
		if err := os.WriteFile(filepath.Join(stagingDir, "Staged.gob"), []byte("staged"), 0600); err != nil {
			return nil, err
		}
		return []InstallResult{{PostScriptName: "Staged", Member: 1}}, nil
	}
	ops.close = func(f *os.File) error {
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		return wantErr
	}

	results, err := installTrueTypeCollectionResults(fontDir, source, ops)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if len(results.Fonts) != 0 {
		t.Fatalf("expected no partial results, got %+v", results)
	}
	if _, statErr := os.Stat(filepath.Join(fontDir, "Staged.gob")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected no published font, got %v", statErr)
	}
}

func TestCollectionInstallerReturnsCleanupWarnings(t *testing.T) {
	fontDir := t.TempDir()
	source := filepath.Join(t.TempDir(), "source.ttc")
	bb := make([]byte, 28)
	copy(bb, ttcTag)
	binary.BigEndian.PutUint32(bb[4:], 0x00010000)
	binary.BigEndian.PutUint32(bb[8:], 1)
	binary.BigEndian.PutUint32(bb[12:], 16)
	if err := os.WriteFile(source, bb, 0600); err != nil {
		t.Fatal(err)
	}
	backupErr := errors.New("backup cleanup failed")
	stagingErr := errors.New("staging cleanup failed")
	ops := defaultCollectionInstallFileOperations()
	removeAll := ops.removeAll
	ops.stageMembers = func(_ *os.File, stagingDir, _ string, _ int64, _ uint32, _ int64) ([]InstallResult, error) {
		if err := os.WriteFile(filepath.Join(stagingDir, "Staged.gob"), []byte("staged"), 0600); err != nil {
			return nil, err
		}
		return []InstallResult{{PostScriptName: "Staged", Member: 1}}, nil
	}
	ops.removeAll = func(path string) error {
		switch {
		case strings.Contains(path, ".pdfcpu-font-backup-"):
			return backupErr
		case strings.Contains(path, ".pdfcpu-ttc-install-"):
			return stagingErr
		default:
			return removeAll(path)
		}
	}
	report, err := installTrueTypeCollectionResults(fontDir, source, ops)
	if err != nil {
		t.Fatalf("cleanup warnings must not replace installation success: %v", err)
	}
	if len(report.Fonts) != 1 || report.Fonts[0].PostScriptName != "Staged" {
		t.Fatalf("expected installed font result, got %+v", report.Fonts)
	}
	if len(report.Warnings) != 2 {
		t.Fatalf("expected two cleanup warnings, got %+v", report.Warnings)
	}
	if !errors.Is(report.Warnings[0], backupErr) || !errors.Is(report.Warnings[1], stagingErr) {
		t.Fatalf("expected ordered backup and staging warnings, got %+v", report.Warnings)
	}
}

func TestReserveCollectionPostScriptNamePreservesDuplicateSentinel(t *testing.T) {
	members := map[string]int{}
	if err := reserveCollectionPostScriptName(members, "Same", 1); err != nil {
		t.Fatal(err)
	}
	err := reserveCollectionPostScriptName(members, "Same", 2)
	if !errors.Is(err, ErrDuplicatePostScriptName) {
		t.Fatalf("expected %v, got %v", ErrDuplicatePostScriptName, err)
	}
	for _, context := range []string{"Same", "member 2", "member 1"} {
		if !strings.Contains(err.Error(), context) {
			t.Fatalf("expected context %q, got %q", context, err)
		}
	}
}

func TestInstallFontFromBytesRejectsOversizedTableWithoutPanic(t *testing.T) {
	bb := make([]byte, 28)
	copy(bb, sfntVersionTrueType)
	binary.BigEndian.PutUint16(bb[4:], 1)
	copy(bb[12:], "name")
	binary.BigEndian.PutUint32(bb[20:], 28)
	binary.BigEndian.PutUint32(bb[24:], math.MaxUint32)

	err := InstallFontFromBytes(t.TempDir(), "oversized.ttf", bb)
	if !errors.Is(err, ErrInvalidFontData) {
		t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
	}
	if !strings.Contains(err.Error(), "table name") || !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("expected table size context, got %q", err)
	}
}

func TestInstallFontFromBytesExposesMalformedHeaderSentinel(t *testing.T) {
	for _, tt := range []struct {
		name string
		data []byte
	}{
		{name: "truncated", data: []byte{0, 1}},
		{name: "invalid signature", data: make([]byte, 12)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := InstallFontFromBytes(t.TempDir(), "malformed.ttf", tt.data)
			if !errors.Is(err, ErrInvalidFontData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
			}
			if !strings.Contains(err.Error(), "parse tables") {
				t.Fatalf("expected parse context, got %q", err)
			}
		})
	}
}

func TestInstallRejectsDuplicateTableTagsBeforeReadingTableData(t *testing.T) {
	bb := make([]byte, 44)
	copy(bb, testTTFHeader(2))
	for i := range 2 {
		off := 12 + i*16
		copy(bb[off:], "name")
		binary.BigEndian.PutUint32(bb[off+8:], uint32(len(bb)))
	}
	err := InstallFontFromBytes(t.TempDir(), "duplicate.ttf", bb)
	if !errors.Is(err, ErrInvalidFontData) {
		t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
	}
	if !strings.Contains(err.Error(), `duplicate table tag "name"`) {
		t.Fatalf("expected duplicate-tag context, got %q", err)
	}
}

func TestTTFTablesRejectsDuplicateTags(t *testing.T) {
	bb := make([]byte, 44)
	copy(bb, testTTFHeader(2))
	copy(bb[12:], "name")
	copy(bb[28:], "name")
	for _, off := range []int{12, 28} {
		binary.BigEndian.PutUint32(bb[off+8:], uint32(len(bb)))
	}
	_, err := ttfTables(2, bb)
	if !errors.Is(err, ErrInvalidFontData) {
		t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
	}
	if !strings.Contains(err.Error(), "duplicate table tag") {
		t.Fatalf("expected duplicate-tag context, got %q", err)
	}
}

func TestFontAllocationLimitsRejectBeforeAllocation(t *testing.T) {
	header := testTTFHeader(maxFontTableCount + 1)
	_, _, err := readFontHeader(bytes.NewReader(header), 0, int64(len(header)))
	requireInvalidFontData(t, err, "table count")

	_, _, err = readFontHeader(bytes.NewReader(testTTFHeader(1)), 0, maxFontFileSize+1)
	requireInvalidFontData(t, err, "font size")

	bb := make([]byte, 28)
	copy(bb, testTTFHeader(1))
	copy(bb[12:], "name")
	binary.BigEndian.PutUint32(bb[20:], uint32(len(bb)))
	binary.BigEndian.PutUint32(bb[24:], maxFontTableSize+1)
	err = InstallFontFromBytes(t.TempDir(), "oversized-table.ttf", bb)
	requireInvalidFontData(t, err, "exceeds limit")

	directory := make([]byte, 12+3*16)
	copy(directory, testTTFHeader(3))
	for i, tag := range []string{"one1", "two2", "tre3"} {
		off := 12 + i*16
		copy(directory[off:], tag)
		binary.BigEndian.PutUint32(directory[off+8:], uint32(len(directory)))
		binary.BigEndian.PutUint32(directory[off+12:], maxFontTableSize)
	}
	_, _, err = headerAndTables(bytes.NewReader(directory), 0, maxFontFileSize)
	requireInvalidFontData(t, err, "table data size")
}

func TestWriteGobPublishesVerifiedTemporaryFile(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "Demo.gob")
	if err := os.WriteFile(fileName, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	want := validInstalledTTF("Demo", minimalSubsetFont(t))
	var mode os.FileMode
	ops := defaultGobPersistenceOperations()
	chmod := ops.chmod
	ops.chmod = func(f *os.File, fileMode os.FileMode) error {
		mode = fileMode
		return chmod(f, fileMode)
	}
	if err := writeGobWithOperations(fileName, want, ops); err != nil {
		t.Fatal(err)
	}
	if mode != installedFontMode {
		t.Fatalf("published font mode: got %04o, want %04o", mode, installedFontMode)
	}
	got := ttf{}
	if err := readGob(fileName, &got); err != nil {
		t.Fatal(err)
	}
	if !ttfEqual(want, got) {
		t.Fatal("persisted font differs from source")
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(fileName), ".Demo.gob.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected temporary files to be cleaned up, got %v", matches)
	}
}

func TestWriteGobCleansTemporaryFileWhenPublishFails(t *testing.T) {
	dir := t.TempDir()
	fileName := filepath.Join(dir, "Demo.gob")
	if err := os.Mkdir(fileName, 0700); err != nil {
		t.Fatal(err)
	}
	err := writeGob(fileName, validInstalledTTF("Demo", minimalSubsetFont(t)))
	if err == nil || !strings.Contains(err.Error(), "publish installed font") {
		t.Fatalf("expected publish error, got %v", err)
	}
	matches, globErr := filepath.Glob(filepath.Join(dir, ".Demo.gob.tmp-*"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("expected temporary file cleanup, got %v", matches)
	}
}

func TestWriteGobFailureSeamsPreserveCauseAndCleanup(t *testing.T) {
	tests := []struct {
		name  string
		phase string
		edit  func(*gobPersistenceOperations, error)
	}{
		{"encode", "encode", func(ops *gobPersistenceOperations, wantErr error) {
			ops.encode = func(*os.File, ttf) error { return wantErr }
		}},
		{"permissions", "set permissions", func(ops *gobPersistenceOperations, wantErr error) {
			ops.chmod = func(*os.File, os.FileMode) error { return wantErr }
		}},
		{"sync", "sync", func(ops *gobPersistenceOperations, wantErr error) {
			ops.sync = func(*os.File) error { return wantErr }
		}},
		{"close", "close temporary font", func(ops *gobPersistenceOperations, wantErr error) {
			ops.close = func(f *os.File) error {
				return errors.Join(f.Close(), wantErr)
			}
		}},
		{"rename", "publish installed font", func(ops *gobPersistenceOperations, wantErr error) {
			ops.rename = func(string, string) error { return wantErr }
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			fileName := filepath.Join(dir, "Demo.gob")
			if err := os.WriteFile(fileName, []byte("old"), 0600); err != nil {
				t.Fatal(err)
			}
			wantErr := errors.New(tt.name + " failed")
			ops := defaultGobPersistenceOperations()
			tt.edit(&ops, wantErr)
			err := writeGobWithOperations(fileName, validInstalledTTF("Demo", minimalSubsetFont(t)), ops)
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.phase) {
				t.Fatalf("expected phase %q, got %q", tt.phase, err)
			}
			bb, readErr := os.ReadFile(fileName)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(bb) != "old" {
				t.Fatalf("expected original file to remain intact, got %q", bb)
			}
			matches, globErr := filepath.Glob(filepath.Join(dir, ".Demo.gob.tmp-*"))
			if globErr != nil {
				t.Fatal(globErr)
			}
			if len(matches) != 0 {
				t.Fatalf("expected temporary cleanup, got %v", matches)
			}
		})
	}
}

func TestWriteGobDirectorySyncFailurePreservesCause(t *testing.T) {
	dir := t.TempDir()
	fileName := filepath.Join(dir, "Demo.gob")
	wantErr := errors.New("directory sync failed")
	want := validInstalledTTF("Demo", minimalSubsetFont(t))
	ops := defaultGobPersistenceOperations()
	ops.syncDir = func(string) error { return wantErr }
	err := writeGobWithOperations(fileName, want, ops)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "sync installed font directory") {
		t.Fatalf("expected directory-sync context, got %q", err)
	}
	got := ttf{}
	if err := readGob(fileName, &got); err != nil {
		t.Fatalf("expected published gob to remain readable: %v", err)
	}
	if !ttfEqual(want, got) {
		t.Fatal("published font differs from source")
	}
	matches, globErr := filepath.Glob(filepath.Join(dir, ".Demo.gob.tmp-*"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("expected temporary cleanup, got %v", matches)
	}
}

func TestReadGobRejectsOversizedRepresentationBeforeDecode(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "Oversized.gob")
	if err := os.WriteFile(fileName, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(fileName, maxInstalledFontSize+1); err != nil {
		t.Fatal(err)
	}
	err := readGob(fileName, &ttf{})
	requireInvalidFontData(t, err, "exceeds limit")
}

func TestValidateDecodedFontFileStructure(t *testing.T) {
	if err := validateDecodedFontFile(nil); !errors.Is(err, ErrMissingFontData) {
		t.Fatalf("expected %v, got %v", ErrMissingFontData, err)
	}

	if err := validateDecodedFontFile(testTTFHeader(1)); err == nil || !strings.Contains(err.Error(), "table directory") {
		t.Fatalf("expected missing table-directory error, got %v", err)
	}

	bb := make([]byte, 28)
	copy(bb, testTTFHeader(1))
	copy(bb[12:], "head")
	binary.BigEndian.PutUint32(bb[20:], uint32(len(bb)))
	binary.BigEndian.PutUint32(bb[24:], maxFontTableSize+1)
	if err := validateDecodedFontFile(bb); err == nil || !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("expected bounded table-directory error, got %v", err)
	}

	if err := validateDecodedFontFile(minimalSubsetFont(t)); err != nil {
		t.Fatalf("expected valid embedded SFNT, got %v", err)
	}
}

func testTTFHeader(tableCount uint16) []byte {
	header := make([]byte, 12)
	copy(header, sfntVersionTrueType)
	binary.BigEndian.PutUint16(header[4:], tableCount)
	return header
}

func validTestTable(size int) *table {
	data := make([]byte, size)
	data = pad(data)
	return &table{size: uint32(size), padded: uint32(len(data)), data: data}
}

func TestCreateTTFRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
		tables map[string]*table
		want   string
	}{
		{
			name:   "short header",
			header: make([]byte, 11),
			tables: map[string]*table{"name": validTestTable(4)},
			want:   "font header",
		},
		{
			name:   "unsupported header",
			header: make([]byte, 12),
			tables: map[string]*table{"name": validTestTable(4)},
			want:   "unsupported version",
		},
		{
			name:   "missing tables",
			header: testTTFHeader(0),
			tables: nil,
			want:   "missing tables",
		},
		{
			name:   "table count mismatch",
			header: testTTFHeader(2),
			tables: map[string]*table{"name": validTestTable(4)},
			want:   "table count",
		},
		{
			name:   "invalid tag",
			header: testTTFHeader(1),
			tables: map[string]*table{"bad": validTestTable(4)},
			want:   "expected 4 bytes",
		},
		{
			name:   "nil table",
			header: testTTFHeader(1),
			tables: map[string]*table{"name": nil},
			want:   "missing table data",
		},
		{
			name:   "invalid padded size",
			header: testTTFHeader(1),
			tables: map[string]*table{"name": {size: 3, padded: 3, data: make([]byte, 3)}},
			want:   "padded size",
		},
		{
			name:   "invalid data length",
			header: testTTFHeader(1),
			tables: map[string]*table{"name": {size: 3, padded: 4, data: make([]byte, 3)}},
			want:   "data length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := createTTF(tt.header, tt.tables)
			if !errors.Is(err, ErrInvalidFontData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected context %q, got %q", tt.want, err)
			}
		})
	}
}

func TestCreateTTFRejectsOffsetOverflow(t *testing.T) {
	_, err := nextTTFOffset(uint64(math.MaxUint32)-1, 4)
	if !errors.Is(err, ErrInvalidFontData) {
		t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
	}
	if !strings.Contains(err.Error(), "exceeds uint32") {
		t.Fatalf("expected overflow context, got %q", err)
	}
}

func TestCreateTTFDoesNotMutateTablesOnValidationError(t *testing.T) {
	glyf := validTestTable(4)
	glyf.off = 99
	glyf.chksum = 77
	tables := map[string]*table{
		"glyf": glyf,
		"name": {size: 3, padded: 3, data: make([]byte, 3)},
	}

	if _, err := createTTF(testTTFHeader(2), tables); err == nil {
		t.Fatal("expected validation error")
	}
	if glyf.off != 99 || glyf.chksum != 77 {
		t.Fatalf("table mutated on error: offset=%d checksum=%d", glyf.off, glyf.chksum)
	}
}

func TestCreateTTFCommitsMetadataAfterSuccess(t *testing.T) {
	name := validTestTable(4)
	name.off = 99
	bb, err := createTTF(testTTFHeader(1), map[string]*table{"name": name})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(bb), 32; got != want {
		t.Fatalf("expected output length %d, got %d", want, got)
	}
	if got, want := name.off, uint32(28); got != want {
		t.Fatalf("expected table offset %d, got %d", want, got)
	}
	if got := binary.BigEndian.Uint32(bb[20:]); got != name.off {
		t.Fatalf("expected directory offset %d, got %d", name.off, got)
	}
}

func validInstalledTTF(fontName string, fontData []byte) ttf {
	return ttf{
		PostscriptName:  fontName,
		UnitsPerEm:      1000,
		LLx:             -100,
		LLy:             -200,
		URx:             1000,
		URy:             800,
		HorMetricsCount: 1,
		GlyphCount:      1,
		GlyphWidths:     []int{500},
		Chars:           map[uint32]uint16{},
		ToUnicode:       map[uint16]uint32{},
		Planes:          map[int]bool{0: true},
		FontFile:        fontData,
	}
}

func writeSubsetFont(t *testing.T, fontName string, fontData []byte) {
	t.Helper()
	f, err := os.Create(filepath.Join(UserFontDir, fontName+".gob"))
	if err != nil {
		t.Fatal(err)
	}
	if err := gob.NewEncoder(f).Encode(validInstalledTTF(fontName, fontData)); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func minimalSubsetFont(t *testing.T) []byte {
	t.Helper()
	head := validTestTable(52)
	maxp := validTestTable(6)
	binary.BigEndian.PutUint16(maxp.data[4:], 1)
	glyf := validTestTable(0)
	loca := validTestTable(4)
	bb, err := createTTF(testTTFHeader(4), map[string]*table{
		"glyf": glyf,
		"head": head,
		"loca": loca,
		"maxp": maxp,
	})
	if err != nil {
		t.Fatal(err)
	}
	return bb
}

func TestSubsetGuardsAndPreservesPhaseErrors(t *testing.T) {
	if _, err := Subset(" ", nil); !errors.Is(err, ErrMissingFontName) {
		t.Fatalf("expected %v, got %v", ErrMissingFontName, err)
	}

	originalDir := UserFontDir
	UserFontDir = t.TempDir()
	t.Cleanup(func() { UserFontDir = originalDir })

	_, err := Subset("Missing", nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "subset font Missing: read installed font") {
		t.Fatalf("expected read context, got %q", err)
	}

	writeSubsetFont(t, "Short", []byte{0x00, 0x01})
	_, err = Subset("Short", nil)
	if !errors.Is(err, ErrInvalidFontData) {
		t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
	}
	if !strings.Contains(err.Error(), "subset font Short: read installed font") || !strings.Contains(err.Error(), "embedded font file") {
		t.Fatalf("expected header context, got %q", err)
	}

	writeSubsetFont(t, "Minimal", minimalSubsetFont(t))
	if _, err := Subset("Minimal", nil); err != nil {
		t.Fatalf("expected nil glyph map to be accepted, got %v", err)
	}
}

func tableForTestData(data []byte) *table {
	size := len(data)
	data = pad(append([]byte(nil), data...))
	return &table{size: uint32(size), padded: uint32(len(data)), data: data}
}

func glyphTablesForTest(numGlyphs, format uint16, locaData, glyfData []byte) map[string]*table {
	head := tableForTestData(make([]byte, 52))
	binary.BigEndian.PutUint16(head.data[50:], format)
	maxp := tableForTestData(make([]byte, 6))
	binary.BigEndian.PutUint16(maxp.data[4:], numGlyphs)
	return map[string]*table{
		"head": head,
		"maxp": maxp,
		"loca": tableForTestData(locaData),
		"glyf": tableForTestData(glyfData),
	}
}

func shortLoca(offsets ...uint16) []byte {
	bb := make([]byte, len(offsets)*2)
	for i, off := range offsets {
		binary.BigEndian.PutUint16(bb[i*2:], off/2)
	}
	return bb
}

func longLoca(offsets ...uint32) []byte {
	bb := make([]byte, len(offsets)*4)
	for i, off := range offsets {
		binary.BigEndian.PutUint32(bb[i*4:], off)
	}
	return bb
}

func TestCheckedTableReadsRejectInvalidOffsets(t *testing.T) {
	tab := &table{data: []byte{0, 1, 2}}
	for _, tt := range []struct {
		name string
		fn   func() error
	}{
		{name: "nil uint16", fn: func() error { _, err := tableUint16At(nil, 0); return err }},
		{name: "negative uint16", fn: func() error { _, err := tableUint16At(tab, -1); return err }},
		{name: "short uint16", fn: func() error { _, err := tableUint16At(tab, 2); return err }},
		{name: "short uint32", fn: func() error { _, err := tableUint32At(tab, 0); return err }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, ErrInvalidFontData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
			}
		})
	}
}

func TestSubsetGlyphTableInfoRejectsMalformedTables(t *testing.T) {
	valid := func() map[string]*table {
		return glyphTablesForTest(1, 0, shortLoca(0, 0), nil)
	}
	tests := []struct {
		name      string
		configure func(map[string]*table)
		want      string
	}{
		{name: "missing head", configure: func(m map[string]*table) { delete(m, "head") }, want: "missing head table"},
		{name: "short head", configure: func(m map[string]*table) { m["head"] = tableForTestData(make([]byte, 10)) }, want: "head table"},
		{name: "short maxp", configure: func(m map[string]*table) { m["maxp"] = tableForTestData(make([]byte, 4)) }, want: "maxp table"},
		{name: "invalid format", configure: func(m map[string]*table) { binary.BigEndian.PutUint16(m["head"].data[50:], 2) }, want: "indexToLocFormat"},
		{name: "zero glyphs", configure: func(m map[string]*table) { binary.BigEndian.PutUint16(m["maxp"].data[4:], 0) }, want: "numGlyphs"},
		{name: "short loca", configure: func(m map[string]*table) { m["loca"] = tableForTestData(shortLoca(0)) }, want: "loca table"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tables := valid()
			tt.configure(tables)
			_, err := subsetGlyphTableInfo(tables)
			if !errors.Is(err, ErrInvalidFontData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected context %q, got %q", tt.want, err)
			}
		})
	}
}

func TestGlyphRangeRejectsInvalidGlyphsAndOffsets(t *testing.T) {
	tests := []struct {
		name      string
		numGlyphs uint16
		offsets   []uint16
		glyfSize  int
		gid       int
		want      string
	}{
		{name: "invalid glyph ID", numGlyphs: 1, offsets: []uint16{0, 0}, gid: 1, want: "glyph ID 1"},
		{name: "descending offsets", numGlyphs: 2, offsets: []uint16{0, 4, 2}, glyfSize: 4, gid: 1, want: "descending offsets"},
		{name: "offset beyond glyf", numGlyphs: 1, offsets: []uint16{0, 6}, glyfSize: 4, gid: 0, want: "exceeds glyf size"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tables := glyphTablesForTest(tt.numGlyphs, 0, shortLoca(tt.offsets...), make([]byte, tt.glyfSize))
			info, err := subsetGlyphTableInfo(tables)
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = glyphRange(info, tt.gid)
			if !errors.Is(err, ErrInvalidFontData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected context %q, got %q", tt.want, err)
			}
		})
	}
}

func compoundGlyphForTest(flags, componentGID uint16) []byte {
	bb := make([]byte, 10)
	binary.BigEndian.PutUint16(bb, math.MaxUint16)
	component := make([]byte, 4)
	binary.BigEndian.PutUint16(component, flags)
	binary.BigEndian.PutUint16(component[2:], componentGID)
	bb = append(bb, component...)
	argSize := 2
	if flags&compoundArgWords != 0 {
		argSize = 4
	}
	bb = append(bb, make([]byte, argSize)...)
	transformSize, _ := compoundTransformSize(flags)
	bb = append(bb, make([]byte, transformSize)...)
	if flags&compoundInstructions != 0 {
		bb = append(bb, 0, 0)
	}
	return bb
}

func compoundGlyphInfo(t *testing.T, compound []byte) subsetGlyphTables {
	t.Helper()
	simple := make([]byte, 10)
	glyfData := append(append([]byte(nil), compound...), simple...)
	tables := glyphTablesForTest(2, 0, shortLoca(0, uint16(len(compound)), uint16(len(glyfData))), glyfData)
	info, err := subsetGlyphTableInfo(tables)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestResolveCompoundGlyphAddsComponentsSafely(t *testing.T) {
	compound := compoundGlyphForTest(0, 1)
	info := compoundGlyphInfo(t, compound)
	usedGIDs := map[uint16]bool{}
	if err := resolveCompoundGlyphs(usedGIDs, info); err != nil {
		t.Fatal(err)
	}
	if !usedGIDs[1] {
		t.Fatal("expected compound component glyph 1")
	}
}

func TestResolveCompoundGlyphRejectsMalformedRecords(t *testing.T) {
	validCompound := compoundGlyphForTest(0, 1)
	info := compoundGlyphInfo(t, validCompound)
	tests := []struct {
		name string
		bb   []byte
		info subsetGlyphTables
		want string
	}{
		{name: "short header", bb: make([]byte, 9), info: info, want: "glyph header"},
		{name: "missing component", bb: validCompound[:10], info: info, want: "need 4 bytes"},
		{name: "truncated arguments", bb: compoundGlyphForTest(compoundArgWords, 1)[:14], info: info, want: "record length"},
		{name: "conflicting transforms", bb: compoundGlyphForTest(compoundScale|compoundXYScale, 1), info: info, want: "conflicting transform"},
		{name: "early instructions", bb: compoundGlyphForTest(compoundInstructions|compoundMoreComponents, 1), info: info, want: "before final component"},
		{name: "invalid component glyph", bb: compoundGlyphForTest(0, 2), info: info, want: "glyph ID 2"},
		{name: "truncated instructions", bb: compoundGlyphForTest(compoundInstructions, 1)[:16], info: info, want: "missing length"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := resolveCompoundGlyph(0, tt.bb, map[uint16]bool{}, tt.info, 0)
			if !errors.Is(err, ErrInvalidFontData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected context %q, got %q", tt.want, err)
			}
		})
	}

	err := resolveCompoundGlyph(0, validCompound, map[uint16]bool{}, info, maxCompoundGlyphDepth)
	if !errors.Is(err, ErrInvalidFontData) || !strings.Contains(err.Error(), "nesting exceeds") {
		t.Fatalf("expected nesting error, got %v", err)
	}
}

func TestSortedUsedGlyphIDsIncludesZeroOnce(t *testing.T) {
	gids := sortedUsedGlyphIDs(map[uint16]bool{0: true, 2: true, 5: true, 7: false})
	if got, want := fmt.Sprint(gids), "[0 2 5]"; got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestEncodeGlyfOffsetChecksRepresentability(t *testing.T) {
	tests := []struct {
		name   string
		off    uint64
		format int
		want   string
	}{
		{name: "unaligned short offset", off: 1, format: 0, want: "not 2-byte aligned"},
		{name: "short offset overflow", off: uint64(^uint16(0))*2 + 2, format: 0, want: "exceeds uint16"},
		{name: "long offset overflow", off: uint64(^uint32(0)) + 1, format: 1, want: "exceeds uint32"},
		{name: "invalid format", format: 2, want: "expected 0 or 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encodeGlyfOffset(tt.off, tt.format)
			if !errors.Is(err, ErrInvalidFontData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected context %q, got %q", tt.want, err)
			}
		})
	}

	if _, err := encodeGlyfOffset(uint64(^uint16(0))*2, 0); err != nil {
		t.Fatalf("expected maximum short offset to succeed, got %v", err)
	}
	if _, err := encodeGlyfOffset(uint64(^uint32(0)), 1); err != nil {
		t.Fatalf("expected maximum long offset to succeed, got %v", err)
	}
}

func TestNextGlyfOffsetChecksAlignmentAndOverflow(t *testing.T) {
	if _, err := nextGlyfOffset(0, 1, 0); !errors.Is(err, ErrInvalidFontData) {
		t.Fatalf("expected short alignment error, got %v", err)
	}
	if _, err := nextGlyfOffset(uint64(^uint16(0))*2, 2, 0); !errors.Is(err, ErrInvalidFontData) {
		t.Fatalf("expected short offset overflow, got %v", err)
	}
	if _, err := nextGlyfOffset(^uint64(0), 1, 1); !errors.Is(err, ErrInvalidFontData) {
		t.Fatalf("expected uint64 overflow, got %v", err)
	}
	if _, err := nextGlyfOffset(0, -1, 1); !errors.Is(err, ErrInvalidFontData) {
		t.Fatalf("expected negative length error, got %v", err)
	}
}

func TestRebuiltTableDataPreservesLogicalAndPaddedLengths(t *testing.T) {
	data, size, padded, err := rebuiltTableData("glyf", []byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	if size != 3 || padded != 4 || len(data) != 4 {
		t.Fatalf("expected sizes 3/4 and 4 bytes, got %d/%d and %d bytes", size, padded, len(data))
	}
	if data[3] != 0 {
		t.Fatalf("expected zero padding, got %#x", data[3])
	}
}

func TestGlyfAndLocaRebuildsLongOffsets(t *testing.T) {
	glyph := make([]byte, 10)
	tables := glyphTablesForTest(2, 1, longLoca(0, 0, uint32(len(glyph))), glyph)
	if err := glyfAndLoca(tables, map[uint16]bool{1: true}); err != nil {
		t.Fatal(err)
	}
	loca := tables["loca"]
	if got := binary.BigEndian.Uint32(loca.data[8:]); got != uint32(len(glyph)) {
		t.Fatalf("expected final loca offset %d, got %d", len(glyph), got)
	}
	if tables["glyf"].size != uint32(len(glyph)) {
		t.Fatalf("expected glyf size %d, got %d", len(glyph), tables["glyf"].size)
	}
}

func TestSubsetPreservesMalformedGlyphErrorChain(t *testing.T) {
	originalDir := UserFontDir
	UserFontDir = t.TempDir()
	t.Cleanup(func() { UserFontDir = originalDir })

	tables := glyphTablesForTest(1, 0, shortLoca(0, 2), []byte{0, 0})
	bb, err := createTTF(testTTFHeader(uint16(len(tables))), tables)
	if err != nil {
		t.Fatal(err)
	}
	writeSubsetFont(t, "MalformedGlyph", bb)

	_, err = Subset("MalformedGlyph", nil)
	if !errors.Is(err, ErrInvalidFontData) {
		t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
	}
	for _, context := range []string{"subset font MalformedGlyph", "subset glyphs", "resolve compound glyphs", "glyph header"} {
		if !strings.Contains(err.Error(), context) {
			t.Fatalf("expected context %q, got %q", context, err)
		}
	}
}

func TestSubsetDoesNotMutateCallerGlyphMap(t *testing.T) {
	originalDir := UserFontDir
	UserFontDir = t.TempDir()
	t.Cleanup(func() { UserFontDir = originalDir })

	compound := compoundGlyphForTest(0, 1)
	simple := make([]byte, 10)
	glyfData := append(append([]byte(nil), compound...), simple...)
	tables := glyphTablesForTest(2, 0, shortLoca(0, uint16(len(compound)), uint16(len(glyfData))), glyfData)
	bb, err := createTTF(testTTFHeader(uint16(len(tables))), tables)
	if err != nil {
		t.Fatal(err)
	}
	writeSubsetFont(t, "Compound", bb)

	usedGIDs := map[uint16]bool{}
	if _, err := Subset("Compound", usedGIDs); err != nil {
		t.Fatal(err)
	}
	if len(usedGIDs) != 0 {
		t.Fatalf("expected caller glyph map to remain unchanged, got %#v", usedGIDs)
	}
}

func requireInvalidFontData(t *testing.T, err error, context string) {
	t.Helper()
	if !errors.Is(err, ErrInvalidFontData) {
		t.Fatalf("expected %v, got %v", ErrInvalidFontData, err)
	}
	if context != "" && !strings.Contains(err.Error(), context) {
		t.Fatalf("expected context %q, got %q", context, err)
	}
}

func TestParseRejectsShortTablesWithoutPanicking(t *testing.T) {
	for _, tag := range []string{"head", "OS/2", "post", "name", "hhea", "maxp", "hmtx", "cmap"} {
		t.Run(tag, func(t *testing.T) {
			fd := &ttf{UnitsPerEm: 1000, HorMetricsCount: 1, GlyphCount: 1}
			err := parse(map[string]*table{tag: tableForTestData([]byte{0})}, tag, fd)
			requireInvalidFontData(t, err, tag)
		})
	}
}

func TestParseGuardsMissingDataDestinationAndParser(t *testing.T) {
	requireInvalidFontData(t, parse(nil, "head", &ttf{}), "missing head table")
	requireInvalidFontData(t, parse(map[string]*table{"head": nil}, "head", &ttf{}), "missing data")
	if err := parse(map[string]*table{}, "OS/2", &ttf{}); err != nil {
		t.Fatalf("expected missing optional OS/2 table to be accepted, got %v", err)
	}
	requireInvalidFontData(t, parse(map[string]*table{"head": tableForTestData(make([]byte, 44))}, "head", nil), "missing font destination")
	requireInvalidFontData(t, parse(map[string]*table{"kern": tableForTestData([]byte{0})}, "kern", &ttf{}), "unsupported table parser")
}

func TestParseFontHeaderRejectsInvalidCoreFields(t *testing.T) {
	head := tableForTestData(make([]byte, 44))
	binary.BigEndian.PutUint32(head.data[12:], ttfHeadMagicNumber)
	requireInvalidFontData(t, head.parseFontHeaderTable(&ttf{}), "unitsPerEm")

	binary.BigEndian.PutUint16(head.data[18:], 1000)
	binary.BigEndian.PutUint32(head.data[12:], 0)
	requireInvalidFontData(t, head.parseFontHeaderTable(&ttf{}), "magic number")
}

func TestParsePostScriptReadsSignedAngleAndFixedPitch(t *testing.T) {
	post := tableForTestData(make([]byte, 16))
	angle := int32(-98304)
	binary.BigEndian.PutUint32(post.data[4:], uint32(angle))
	binary.BigEndian.PutUint32(post.data[12:], 1)
	fd := &ttf{}
	if err := post.parsePostScriptTable(fd); err != nil {
		t.Fatal(err)
	}
	if fd.ItalicAngle != -1.5 {
		t.Fatalf("expected italic angle -1.5, got %f", fd.ItalicAngle)
	}
	if !fd.FixedPitch {
		t.Fatal("expected fixed-pitch font")
	}
}

func TestParseWindowsMetricsChecksVersionSpecificSize(t *testing.T) {
	os2 := tableForTestData(make([]byte, 72))
	binary.BigEndian.PutUint16(os2.data, 2)
	requireInvalidFontData(t, os2.parseWindowsMetricsTable(&ttf{UnitsPerEm: 1000}), "90 bytes")
}

func nameTableForTest(length uint16, nameBytes []byte) *table {
	bb := make([]byte, 18)
	binary.BigEndian.PutUint16(bb[2:], 1)
	binary.BigEndian.PutUint16(bb[4:], 18)
	binary.BigEndian.PutUint16(bb[6:], 3)
	binary.BigEndian.PutUint16(bb[8:], 1)
	binary.BigEndian.PutUint16(bb[10:], 0x0409)
	binary.BigEndian.PutUint16(bb[12:], 6)
	binary.BigEndian.PutUint16(bb[14:], length)
	bb = append(bb, nameBytes...)
	return tableForTestData(bb)
}

func TestParseNamingTableChecksRecordsAndStrings(t *testing.T) {
	truncatedRecords := tableForTestData(make([]byte, 6))
	binary.BigEndian.PutUint16(truncatedRecords.data[2:], 1)
	requireInvalidFontData(t, truncatedRecords.parseNamingTable(&ttf{}), "name records")

	requireInvalidFontData(t, nameTableForTest(4, []byte{0, 'A'}).parseNamingTable(&ttf{}), "PostScript name record")
	requireInvalidFontData(t, nameTableForTest(1, []byte{'A'}).parseNamingTable(&ttf{}), "odd UTF-16")

	fd := &ttf{}
	if err := nameTableForTest(2, []byte{0, 'A'}).parseNamingTable(fd); err != nil {
		t.Fatal(err)
	}
	if fd.PostscriptName != "A" {
		t.Fatalf("expected PostScript name A, got %q", fd.PostscriptName)
	}
}

func TestParseMetricTablesCheckCountsAndLengths(t *testing.T) {
	hhea := tableForTestData(make([]byte, 36))
	requireInvalidFontData(t, hhea.parseHorizontalHeaderTable(&ttf{UnitsPerEm: 1000}), "numOfLongHorMetrics")

	maxp := tableForTestData(make([]byte, 6))
	requireInvalidFontData(t, maxp.parseMaximumProfile(&ttf{}), "numGlyphs")

	hmtx := tableForTestData(make([]byte, 4))
	requireInvalidFontData(t, hmtx.parseHorizontalMetricsTable(&ttf{UnitsPerEm: 1000, GlyphCount: 2}), "metrics count")
	requireInvalidFontData(t, hmtx.parseHorizontalMetricsTable(&ttf{UnitsPerEm: 1000, GlyphCount: 2, HorMetricsCount: 3}), "outside")
	requireInvalidFontData(t, hmtx.parseHorizontalMetricsTable(&ttf{UnitsPerEm: 1000, GlyphCount: 2, HorMetricsCount: 1}), "expected at least 6 bytes")
}

func cmapFormat4ForTest(start, end, delta, rangeOffset uint16, glyphs ...uint16) table {
	length := 24 + len(glyphs)*2
	bb := make([]byte, length)
	binary.BigEndian.PutUint16(bb, 4)
	binary.BigEndian.PutUint16(bb[2:], uint16(length))
	binary.BigEndian.PutUint16(bb[6:], 2)
	binary.BigEndian.PutUint16(bb[14:], end)
	binary.BigEndian.PutUint16(bb[18:], start)
	binary.BigEndian.PutUint16(bb[20:], delta)
	binary.BigEndian.PutUint16(bb[22:], rangeOffset)
	for i, glyph := range glyphs {
		binary.BigEndian.PutUint16(bb[24+i*2:], glyph)
	}
	return *tableForTestData(bb)
}

func cmapTestFont(glyphCount int) *ttf {
	return &ttf{
		GlyphCount: glyphCount,
		Chars:      map[uint32]uint16{},
		ToUnicode:  map[uint16]uint32{},
		Planes:     map[int]bool{},
	}
}

func TestParseCMapFormat4ChecksDynamicRanges(t *testing.T) {
	fd := cmapTestFont(2)
	delta := int16(1 - 'A')
	valid := cmapFormat4ForTest('A', 'A', uint16(delta), 0)
	if err := valid.parseCMapFormat4(fd); err != nil {
		t.Fatal(err)
	}
	if fd.Chars['A'] != 1 {
		t.Fatalf("expected glyph 1 for A, got %d", fd.Chars['A'])
	}

	descending := cmapFormat4ForTest('B', 'A', 0, 0)
	requireInvalidFontData(t, descending.parseCMapFormat4(cmapTestFont(2)), "precedes start")

	missingGlyph := cmapFormat4ForTest('A', 'A', 0, 2)
	requireInvalidFontData(t, missingGlyph.parseCMapFormat4(cmapTestFont(2)), "glyph for code point")

	badGlyph := cmapFormat4ForTest('A', 'A', 0, 2, 2)
	requireInvalidFontData(t, badGlyph.parseCMapFormat4(cmapTestFont(2)), "exceeds glyph count")
}

func cmapFormat12ForTest(start, end, glyph uint32) table {
	bb := make([]byte, 28)
	binary.BigEndian.PutUint16(bb, 12)
	binary.BigEndian.PutUint32(bb[4:], uint32(len(bb)))
	binary.BigEndian.PutUint32(bb[12:], 1)
	binary.BigEndian.PutUint32(bb[16:], start)
	binary.BigEndian.PutUint32(bb[20:], end)
	binary.BigEndian.PutUint32(bb[24:], glyph)
	return *tableForTestData(bb)
}

func TestParseCMapFormat12ChecksGroups(t *testing.T) {
	bmp := cmapFormat12ForTest('A', 'A', 1)
	bmpFD := cmapTestFont(2)
	if err := bmp.parseCMapFormat12(bmpFD); err != nil {
		t.Fatal(err)
	}
	if !bmpFD.Planes[0] {
		t.Fatal("expected plane 0 to be recorded for the first group")
	}

	valid := cmapFormat12ForTest(0x10000, 0x10001, 1)
	fd := cmapTestFont(3)
	if err := valid.parseCMapFormat12(fd); err != nil {
		t.Fatal(err)
	}
	if fd.Chars[0x10001] != 2 || !fd.Planes[1] {
		t.Fatalf("expected plane 1 glyph mapping, got %#v and planes %#v", fd.Chars, fd.Planes)
	}

	descending := cmapFormat12ForTest(2, 1, 0)
	requireInvalidFontData(t, descending.parseCMapFormat12(cmapTestFont(2)), "precedes start")
	beyondUnicode := cmapFormat12ForTest(0x10FFFF, 0x110000, 0)
	requireInvalidFontData(t, beyondUnicode.parseCMapFormat12(cmapTestFont(2)), "exceeds Unicode")
	badGlyphs := cmapFormat12ForTest(1, 2, 1)
	requireInvalidFontData(t, badGlyphs.parseCMapFormat12(cmapTestFont(2)), "exceeds glyph count")
}

func cmapTableForTest(platform, encoding uint16, subtable table) table {
	bb := make([]byte, 12)
	binary.BigEndian.PutUint16(bb[2:], 1)
	binary.BigEndian.PutUint16(bb[4:], platform)
	binary.BigEndian.PutUint16(bb[6:], encoding)
	binary.BigEndian.PutUint32(bb[8:], 12)
	bb = append(bb, subtable.data[:subtable.size]...)
	return *tableForTestData(bb)
}

func TestParseCMapDirectoryChecksOffsetsAndPreservesCause(t *testing.T) {
	cmap := tableForTestData(make([]byte, 12))
	binary.BigEndian.PutUint16(cmap.data[2:], 1)
	binary.BigEndian.PutUint32(cmap.data[8:], 12)
	requireInvalidFontData(t, cmap.parseCharToGlyphMappingTable(cmapTestFont(2)), "encoding record 0")

	valid := cmapTableForTest(0, 10, cmapFormat12ForTest(0x10000, 0x10000, 1))
	fd := cmapTestFont(2)
	if err := valid.parseCharToGlyphMappingTable(fd); err != nil {
		t.Fatal(err)
	}
	if fd.Chars[0x10000] != 1 {
		t.Fatalf("expected glyph 1, got %d", fd.Chars[0x10000])
	}
}

func TestInstallTrueTypeRepPreservesParserErrorChain(t *testing.T) {
	tables := map[string]*table{"head": tableForTestData([]byte{0})}
	_, err := installTrueTypeRep(t.TempDir(), "broken.ttf", testTTFHeader(1), tables, false, nil)
	requireInvalidFontData(t, err, "parse head table")
}
