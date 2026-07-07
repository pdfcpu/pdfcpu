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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func noOpFontAPIOperations() fontAPIOperations {
	return fontAPIOperations{
		userFontDir:     "/fonts",
		reloadUserFonts: func() error { return nil },
		installTrueTypeFont: func(_ string, fileName string) (font.InstallResult, error) {
			return font.InstallResult{PostScriptName: strings.TrimSuffix(fileName, filepath.Ext(fileName))}, nil
		},
		installTrueTypeCollection: func(_ string, fileName string) ([]font.InstallResult, error) {
			return []font.InstallResult{{PostScriptName: strings.TrimSuffix(fileName, filepath.Ext(fileName)), Member: 1}}, nil
		},
		createStagingDir: func(string) (string, error) { return "/staging", nil },
		createInputDir:   func(string, int) (string, error) { return "/input", nil },
		commitStagedFonts: func(string, string) (fontInstallCommit, error) {
			return fontInstallCommit{rollback: func() error { return nil }, finalize: func() error { return nil }}, nil
		},
		removeAll:            func(string) error { return nil },
		rename:               func(string, string) error { return nil },
		reportCleanupWarning: func(error) {},
	}
}

func TestListFontsPreservesLoadError(t *testing.T) {
	wantErr := errors.New("load user fonts")
	_, err := listFonts(func() ([]string, error) { return nil, wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "list fonts: load user fonts") {
		t.Fatalf("expected list/load context, got %q", err)
	}
}

func TestFontErrorSentinelsAliasLowerLayer(t *testing.T) {
	if ErrUnknownFont != font.ErrUnknownFont {
		t.Fatal("ErrUnknownFont is not an alias of font.ErrUnknownFont")
	}
	if ErrUserFontNotFound != font.ErrUnknownFont {
		t.Fatal("ErrUserFontNotFound is not a compatibility alias of font.ErrUnknownFont")
	}
	if ErrDuplicatePostScriptName != font.ErrDuplicatePostScriptName {
		t.Fatal("ErrDuplicatePostScriptName is not an alias of font.ErrDuplicatePostScriptName")
	}
}

func TestInstallFontsRejectsInvalidInputsBeforeInstalling(t *testing.T) {
	tests := []struct {
		name      string
		fileNames []string
		wantErr   error
		wantText  string
	}{
		{name: "missing slice", wantErr: ErrMissingFontInput, wantText: "install fonts"},
		{name: "empty filename", fileNames: []string{"font.ttf", " "}, wantErr: ErrMissingFontInput, wantText: "input 2"},
		{name: "unsupported extension", fileNames: []string{"font.ttf", "font.otf"}, wantErr: ErrUnsupportedFontFile, wantText: "font.otf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installCalls := 0
			ops := noOpFontAPIOperations()
			ops.installTrueTypeFont = func(string, string) (font.InstallResult, error) {
				installCalls++
				return font.InstallResult{}, nil
			}
			err := installFonts(tt.fileNames, ops)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.wantText) {
				t.Fatalf("expected context %q, got %q", tt.wantText, err)
			}
			if installCalls != 0 {
				t.Fatalf("expected validation before installation, got %d install calls", installCalls)
			}
		})
	}
}

func TestInstallFontsRequiresUserFontDirectory(t *testing.T) {
	ops := noOpFontAPIOperations()
	ops.userFontDir = ""
	err := installFonts([]string{"font.ttf"}, ops)
	if !errors.Is(err, ErrMissingConfiguration) {
		t.Fatalf("expected %v, got %v", ErrMissingConfiguration, err)
	}
	if !strings.Contains(err.Error(), "install fonts: user font directory") {
		t.Fatalf("expected directory context, got %q", err)
	}
}

func TestInstallFontsDispatchesSupportedFiles(t *testing.T) {
	var installed []string
	ops := noOpFontAPIOperations()
	ops.installTrueTypeFont = func(_, fn string) (font.InstallResult, error) {
		installed = append(installed, "ttf:"+fn)
		return font.InstallResult{PostScriptName: "One"}, nil
	}
	ops.installTrueTypeCollection = func(_, fn string) ([]font.InstallResult, error) {
		installed = append(installed, "ttc:"+fn)
		return []font.InstallResult{{PostScriptName: "Two", Member: 1}}, nil
	}

	fileNames := []string{"one.TTF", "two.ttc"}
	if err := installFonts(fileNames, ops); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(installed, ","), "ttf:one.TTF,ttc:two.ttc"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if got, want := strings.Join(fileNames, ","), "one.TTF,two.ttc"; got != want {
		t.Fatalf("caller-owned input changed: expected %q, got %q", want, got)
	}
}

func TestInstallFontsPreservesOperationErrors(t *testing.T) {
	tests := []struct {
		name      string
		fileName  string
		configure func(*fontAPIOperations, error)
		wantPhase string
	}{
		{
			name:     "true type font",
			fileName: "font.ttf",
			configure: func(ops *fontAPIOperations, wantErr error) {
				ops.installTrueTypeFont = func(string, string) (font.InstallResult, error) {
					return font.InstallResult{}, wantErr
				}
			},
			wantPhase: "install fonts",
		},
		{
			name:     "true type collection",
			fileName: "fonts.ttc",
			configure: func(ops *fontAPIOperations, wantErr error) {
				ops.installTrueTypeCollection = func(string, string) ([]font.InstallResult, error) {
					return nil, wantErr
				}
			},
			wantPhase: "install fonts",
		},
		{
			name:     "reload installed fonts",
			fileName: "font.ttf",
			configure: func(ops *fontAPIOperations, wantErr error) {
				ops.reloadUserFonts = func() error { return wantErr }
			},
			wantPhase: "install fonts: reload user fonts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := errors.New("underlying failure")
			ops := noOpFontAPIOperations()
			tt.configure(&ops, wantErr)
			err := installFonts([]string{tt.fileName}, ops)
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.wantPhase) {
				t.Fatalf("expected phase %q, got %q", tt.wantPhase, err)
			}
			if count := strings.Count(err.Error(), wantErr.Error()); count != 1 {
				t.Fatalf("expected cause once, got %d in %q", count, err)
			}
		})
	}
}

func transactionalFontAPIOperations(t *testing.T) fontAPIOperations {
	t.Helper()
	ops := noOpFontAPIOperations()
	ops.userFontDir = t.TempDir()
	ops.createStagingDir = func(fontDir string) (string, error) {
		return os.MkdirTemp(fontDir, ".pdfcpu-font-install-")
	}
	ops.createInputDir = func(stagingDir string, input int) (string, error) {
		return os.MkdirTemp(stagingDir, fmt.Sprintf(".input-%d-", input))
	}
	ops.commitStagedFonts = commitStagedFonts
	ops.removeAll = os.RemoveAll
	ops.rename = os.Rename
	return ops
}

func writeStagedFont(dir, name, content string) error {
	return os.WriteFile(filepath.Join(dir, name+".gob"), []byte(content), 0600)
}

func TestInstallFontsStagesEntireBatchBeforeCommit(t *testing.T) {
	wantErr := errors.New("second font malformed")
	ops := transactionalFontAPIOperations(t)
	reloadCalls := 0
	ops.reloadUserFonts = func() error {
		reloadCalls++
		return nil
	}
	ops.installTrueTypeFont = func(stagingDir, fileName string) (font.InstallResult, error) {
		if fileName == "two.ttf" {
			return font.InstallResult{}, wantErr
		}
		err := writeStagedFont(stagingDir, "One", "new")
		return font.InstallResult{PostScriptName: "One"}, err
	}

	err := installFonts([]string{"one.ttf", "two.ttf"}, ops)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "install fonts: input 2:") {
		t.Fatalf("expected batch identity, got %q", err)
	}
	if _, err := os.Stat(filepath.Join(ops.userFontDir, "One.gob")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no partial installation, got %v", err)
	}
	if reloadCalls != 0 {
		t.Fatalf("expected no metrics reload before commit, got %d", reloadCalls)
	}
}

func TestInstallFontsDoesNotReplaceExistingFontUntilBatchIsValid(t *testing.T) {
	target := filepath.Join(t.TempDir(), "Same.gob")
	if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	ops := transactionalFontAPIOperations(t)
	ops.userFontDir = filepath.Dir(target)
	wantErr := errors.New("invalid second input")
	ops.installTrueTypeFont = func(stagingDir, fileName string) (font.InstallResult, error) {
		if fileName == "two.ttf" {
			return font.InstallResult{}, wantErr
		}
		err := writeStagedFont(stagingDir, "Same", "new")
		return font.InstallResult{PostScriptName: "Same"}, err
	}

	err := installFonts([]string{"one.ttf", "two.ttf"}, ops)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	bb, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(bb) != "old" {
		t.Fatalf("expected existing font to remain unchanged, got %q", bb)
	}
}

func TestInstallFontsRollsBackCommittedFilesWhenReloadFails(t *testing.T) {
	ops := transactionalFontAPIOperations(t)
	target := filepath.Join(ops.userFontDir, "Same.gob")
	if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	ops.installTrueTypeFont = func(stagingDir, _ string) (font.InstallResult, error) {
		err := writeStagedFont(stagingDir, "Same", "new")
		return font.InstallResult{PostScriptName: "Same"}, err
	}
	wantErr := errors.New("reload failed")
	ops.reloadUserFonts = func() error { return wantErr }

	err := installFonts([]string{"one.ttf"}, ops)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	bb, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(bb) != "old" {
		t.Fatalf("expected rollback to restore old font, got %q", bb)
	}
}

func TestInstallFontsCommitsBatchBeforeReload(t *testing.T) {
	ops := transactionalFontAPIOperations(t)
	ops.installTrueTypeFont = func(stagingDir, fileName string) (font.InstallResult, error) {
		name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		err := writeStagedFont(stagingDir, name, fileName)
		return font.InstallResult{PostScriptName: name}, err
	}
	ops.reloadUserFonts = func() error {
		for _, name := range []string{"one", "two"} {
			if _, err := os.Stat(filepath.Join(ops.userFontDir, name+".gob")); err != nil {
				return fmt.Errorf("font %s unavailable during reload: %w", name, err)
			}
		}
		return nil
	}

	if err := installFonts([]string{"one.ttf", "two.ttf"}, ops); err != nil {
		t.Fatal(err)
	}
}

func TestInstallFontsRejectsDuplicateStagedPostScriptNames(t *testing.T) {
	ops := transactionalFontAPIOperations(t)
	ops.installTrueTypeFont = func(inputDir, fileName string) (font.InstallResult, error) {
		err := writeStagedFont(inputDir, "Same", fileName)
		return font.InstallResult{PostScriptName: "Same"}, err
	}
	err := installFonts([]string{"one.ttf", "two.ttf"}, ops)
	if err == nil {
		t.Fatal("expected duplicate PostScript name")
	}
	if !errors.Is(err, ErrDuplicatePostScriptName) {
		t.Fatalf("expected %v, got %v", ErrDuplicatePostScriptName, err)
	}
	for _, context := range []string{"duplicate PostScript name Same", "input 2 two.ttf", "input 1 one.ttf"} {
		if !strings.Contains(err.Error(), context) {
			t.Fatalf("expected context %q, got %q", context, err)
		}
	}
	if _, statErr := os.Stat(filepath.Join(ops.userFontDir, "Same.gob")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected duplicate batch to remain uncommitted, got %v", statErr)
	}
}

func TestInstallFontsReportsDuplicateCollectionMembers(t *testing.T) {
	ops := transactionalFontAPIOperations(t)
	ops.installTrueTypeCollection = func(inputDir, _ string) ([]font.InstallResult, error) {
		if err := writeStagedFont(inputDir, "Same", "member"); err != nil {
			return nil, err
		}
		return []font.InstallResult{
			{PostScriptName: "Same", Member: 1},
			{PostScriptName: "Same", Member: 2},
		}, nil
	}
	err := installFonts([]string{"fonts.ttc"}, ops)
	if err == nil {
		t.Fatal("expected duplicate collection member")
	}
	if !errors.Is(err, ErrDuplicatePostScriptName) {
		t.Fatalf("expected %v, got %v", ErrDuplicatePostScriptName, err)
	}
	for _, context := range []string{"duplicate PostScript name Same", "input 1 fonts.ttc member 2", "input 1 fonts.ttc member 1"} {
		if !strings.Contains(err.Error(), context) {
			t.Fatalf("expected context %q, got %q", context, err)
		}
	}
}

func TestInstallFontsJoinsReloadAndRollbackFailures(t *testing.T) {
	reloadErr := errors.New("reload failed")
	rollbackErr := errors.New("restore failed")
	ops := noOpFontAPIOperations()
	ops.reloadUserFonts = func() error { return reloadErr }
	ops.commitStagedFonts = func(string, string) (fontInstallCommit, error) {
		return fontInstallCommit{
			rollback: func() error { return rollbackErr },
			finalize: func() error { return nil },
		}, nil
	}
	err := installFonts([]string{"one.ttf"}, ops)
	if !errors.Is(err, reloadErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("expected joined reload and rollback failures, got %v", err)
	}
	for _, context := range []string{"reload user fonts", "rollback batch"} {
		if !strings.Contains(err.Error(), context) {
			t.Fatalf("expected context %q, got %q", context, err)
		}
	}
}

func TestInstallFontsTreatsFinalizeFailureAsCleanupWarning(t *testing.T) {
	finalizeErr := errors.New("remove backup failed")
	var warning error
	ops := noOpFontAPIOperations()
	ops.commitStagedFonts = func(string, string) (fontInstallCommit, error) {
		return fontInstallCommit{
			rollback: func() error { return nil },
			finalize: func() error { return finalizeErr },
		}, nil
	}
	ops.reportCleanupWarning = func(err error) { warning = err }
	if err := installFonts([]string{"one.ttf"}, ops); err != nil {
		t.Fatalf("installation succeeded and cleanup failure must be a warning, got %v", err)
	}
	if !errors.Is(warning, finalizeErr) {
		t.Fatalf("expected cleanup warning %v, got %v", finalizeErr, warning)
	}
	if !strings.Contains(warning.Error(), "finalize batch") {
		t.Fatalf("expected finalize context, got %q", warning)
	}
}

func TestCommitStagedFontsSyncFailureJoinsRollbackFailure(t *testing.T) {
	fontDir := t.TempDir()
	stagingDir := t.TempDir()
	if err := writeStagedFont(fontDir, "Same", "old"); err != nil {
		t.Fatal(err)
	}
	if err := writeStagedFont(stagingDir, "Same", "new"); err != nil {
		t.Fatal(err)
	}
	syncErr := errors.New("directory sync failed")
	rollbackErr := errors.New("restore rename failed")
	fs := defaultFontInstallFileOperations()
	rename := fs.rename
	fs.syncDir = func(string) error { return syncErr }
	fs.rename = func(source, target string) error {
		if strings.Contains(source, ".pdfcpu-font-backup-") {
			return rollbackErr
		}
		return rename(source, target)
	}
	_, err := commitStagedFontsWithOperations(fontDir, stagingDir, fs)
	if !errors.Is(err, syncErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("expected joined sync and rollback errors, got %v", err)
	}
	if !strings.Contains(err.Error(), "sync directory") || !strings.Contains(err.Error(), "font backup retained") {
		t.Fatalf("expected sync and retained-backup context, got %q", err)
	}
}

func TestRollbackCommittedFontsSyncFailurePreservesCause(t *testing.T) {
	fontDir := t.TempDir()
	stagingDir := t.TempDir()
	if err := writeStagedFont(fontDir, "Same", "old"); err != nil {
		t.Fatal(err)
	}
	if err := writeStagedFont(stagingDir, "Same", "new"); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("rollback directory sync failed")
	failSync := false
	fs := defaultFontInstallFileOperations()
	fs.syncDir = func(string) error {
		if failSync {
			return wantErr
		}
		return nil
	}
	commit, err := commitStagedFontsWithOperations(fontDir, stagingDir, fs)
	if err != nil {
		t.Fatal(err)
	}
	failSync = true
	err = commit.rollback()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "sync directory") || !strings.Contains(err.Error(), "font backup retained") {
		t.Fatalf("expected rollback-sync and retained-backup context, got %q", err)
	}
}

func TestCommitStagedFontsFilesystemFailureSeams(t *testing.T) {
	t.Run("rollback", func(t *testing.T) {
		fontDir := t.TempDir()
		stagingDir := t.TempDir()
		if err := writeStagedFont(fontDir, "Same", "old"); err != nil {
			t.Fatal(err)
		}
		if err := writeStagedFont(stagingDir, "Same", "new"); err != nil {
			t.Fatal(err)
		}
		wantErr := errors.New("restore rename failed")
		fs := defaultFontInstallFileOperations()
		rename := fs.rename
		fs.rename = func(source, target string) error {
			if strings.Contains(source, ".pdfcpu-font-backup-") {
				return wantErr
			}
			return rename(source, target)
		}
		commit, err := commitStagedFontsWithOperations(fontDir, stagingDir, fs)
		if err != nil {
			t.Fatal(err)
		}
		err = commit.rollback()
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
		if !strings.Contains(err.Error(), "font backup retained") {
			t.Fatalf("expected retained-backup context, got %q", err)
		}
	})

	t.Run("finalize", func(t *testing.T) {
		fontDir := t.TempDir()
		stagingDir := t.TempDir()
		if err := writeStagedFont(stagingDir, "Demo", "new"); err != nil {
			t.Fatal(err)
		}
		wantErr := errors.New("backup cleanup failed")
		fs := defaultFontInstallFileOperations()
		removeAll := fs.removeAll
		fs.removeAll = func(path string) error {
			if strings.Contains(path, ".pdfcpu-font-backup-") {
				return wantErr
			}
			return removeAll(path)
		}
		commit, err := commitStagedFontsWithOperations(fontDir, stagingDir, fs)
		if err != nil {
			t.Fatal(err)
		}
		if err := commit.finalize(); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}

func noOpFontDemoOperations() fontDemoOperations {
	return fontDemoOperations{
		loadUserFonts: func() error { return nil },
		userFont: func(string) (font.TTFLight, bool, error) {
			return font.TTFLight{Planes: map[int]bool{2: true}}, true, nil
		},
		userFontNames: func() ([]string, error) { return nil, nil },
		createXRef: func() (*model.XRefTable, error) {
			return nil, nil
		},
		createPage: func(*model.XRefTable, int, int, int, string) (model.Page, error) {
			return model.Page{}, nil
		},
		catalog: func(*model.XRefTable) (types.Dict, error) {
			return types.Dict{}, nil
		},
		addPageTree: func(*model.XRefTable, types.Dict, model.Page) error {
			return nil
		},
		createPDFFile: func(*model.XRefTable, string, *model.Configuration) error {
			return nil
		},
		files: cheatSheetFileOperations{
			mkdirTemp: func(string, string) (string, error) { return "/staging", nil },
			lstat:     func(string) (os.FileInfo, error) { return nil, os.ErrNotExist },
			syncDir:   func(string) error { return nil },
			rename:    func(string, string) error { return nil },
			remove:    func(string) error { return nil },
			removeAll: func(string) error { return nil },
		},
	}
}

func TestUnicodePlaneSuffixesAreValidAndUnique(t *testing.T) {
	suffixes := map[string]int{}
	for plane := range 17 {
		suffix, err := planeString(plane)
		if err != nil {
			t.Fatalf("plane %d: %v", plane, err)
		}
		if suffix == "" {
			t.Fatalf("plane %d has an empty suffix", plane)
		}
		if other, ok := suffixes[suffix]; ok {
			t.Fatalf("planes %d and %d share suffix %q", other, plane, suffix)
		}
		suffixes[suffix] = plane
	}
	for _, plane := range []int{-1, 17} {
		_, err := planeString(plane)
		if !errors.Is(err, ErrInvalidUnicodePlane) {
			t.Fatalf("plane %d: expected %v, got %v", plane, ErrInvalidUnicodePlane, err)
		}
	}
}

func TestCreateUserFontDemoFilesOrdersPlanesAndRejectsInvalidPlane(t *testing.T) {
	t.Run("ordered filenames", func(t *testing.T) {
		ops := noOpFontDemoOperations()
		ops.userFont = func(string) (font.TTFLight, bool, error) {
			return font.TTFLight{Planes: map[int]bool{2: true, 0: true, 1: true}}, true, nil
		}
		var fileNames []string
		ops.createPDFFile = func(_ *model.XRefTable, fileName string, _ *model.Configuration) error {
			fileNames = append(fileNames, filepath.Base(fileName))
			return nil
		}
		if err := createUserFontDemoFiles(t.TempDir(), "Demo", ops); err != nil {
			t.Fatal(err)
		}
		if got, want := strings.Join(fileNames, ","), "Demo_BMP.pdf,Demo_SMP.pdf,Demo_SIP.pdf"; got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("invalid plane before output", func(t *testing.T) {
		ops := noOpFontDemoOperations()
		ops.userFont = func(string) (font.TTFLight, bool, error) {
			return font.TTFLight{Planes: map[int]bool{17: true}}, true, nil
		}
		createCalls := 0
		ops.createXRef = func() (*model.XRefTable, error) {
			createCalls++
			return nil, nil
		}
		err := createUserFontDemoFiles(t.TempDir(), "Demo", ops)
		if !errors.Is(err, ErrInvalidUnicodePlane) {
			t.Fatalf("expected %v, got %v", ErrInvalidUnicodePlane, err)
		}
		if !strings.Contains(err.Error(), "font Demo plane") {
			t.Fatalf("expected font/plane context, got %q", err)
		}
		if createCalls != 0 {
			t.Fatalf("expected validation before PDF creation, got %d calls", createCalls)
		}
	})
}

func TestCreateUserFontDemoFilesValidatesAndLoadsFont(t *testing.T) {
	t.Run("missing name", func(t *testing.T) {
		loadCalls := 0
		ops := noOpFontDemoOperations()
		ops.loadUserFonts = func() error {
			loadCalls++
			return nil
		}
		err := createUserFontDemoFiles(t.TempDir(), " ", ops)
		if err == nil || !strings.Contains(err.Error(), "font name must not be empty") {
			t.Fatalf("expected missing font name error, got %v", err)
		}
		if loadCalls != 0 {
			t.Fatalf("expected validation before loading, got %d load calls", loadCalls)
		}
	})

	t.Run("load error", func(t *testing.T) {
		wantErr := errors.New("load fonts")
		ops := noOpFontDemoOperations()
		ops.loadUserFonts = func() error { return wantErr }
		err := createUserFontDemoFiles(t.TempDir(), "Demo", ops)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
		if !strings.Contains(err.Error(), "create font cheat sheet: load user fonts") {
			t.Fatalf("expected load context, got %q", err)
		}
	})

	t.Run("default name error", func(t *testing.T) {
		wantErr := errors.New("list fonts")
		ops := noOpFontDemoOperations()
		ops.userFontNames = func() ([]string, error) { return nil, wantErr }
		err := createCheatSheetsUserFonts(nil, ops)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
		if !strings.Contains(err.Error(), "create font cheat sheets: list user fonts") {
			t.Fatalf("expected list context, got %q", err)
		}
	})

	t.Run("unavailable font", func(t *testing.T) {
		ops := noOpFontDemoOperations()
		ops.userFont = func(string) (font.TTFLight, bool, error) {
			return font.TTFLight{}, false, nil
		}
		err := createUserFontDemoFiles(t.TempDir(), "Missing", ops)
		if !errors.Is(err, ErrUserFontNotFound) {
			t.Fatalf("expected %v, got %v", ErrUserFontNotFound, err)
		}
		if !strings.Contains(err.Error(), "font Missing") {
			t.Fatalf("expected font context, got %q", err)
		}
	})
}

func TestCreateUserFontDemoFilesPreservesPhaseErrors(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*fontDemoOperations, error)
		wantPhase string
	}{
		{
			name: "create PDF context",
			configure: func(ops *fontDemoOperations, wantErr error) {
				ops.createXRef = func() (*model.XRefTable, error) { return nil, wantErr }
			},
			wantPhase: "create PDF context",
		},
		{
			name: "render page",
			configure: func(ops *fontDemoOperations, wantErr error) {
				ops.createPage = func(*model.XRefTable, int, int, int, string) (model.Page, error) {
					return model.Page{}, wantErr
				}
			},
			wantPhase: "render page",
		},
		{
			name: "access catalog",
			configure: func(ops *fontDemoOperations, wantErr error) {
				ops.catalog = func(*model.XRefTable) (types.Dict, error) {
					return nil, wantErr
				}
			},
			wantPhase: "access catalog",
		},
		{
			name: "build page tree",
			configure: func(ops *fontDemoOperations, wantErr error) {
				ops.addPageTree = func(*model.XRefTable, types.Dict, model.Page) error {
					return wantErr
				}
			},
			wantPhase: "build page tree",
		},
		{
			name: "write output",
			configure: func(ops *fontDemoOperations, wantErr error) {
				ops.createPDFFile = func(*model.XRefTable, string, *model.Configuration) error {
					return fmt.Errorf("create: write output: %w", wantErr)
				}
			},
			wantPhase: "create: write output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := errors.New("underlying failure")
			ops := noOpFontDemoOperations()
			tt.configure(&ops, wantErr)
			err := createUserFontDemoFiles(t.TempDir(), "Demo", ops)
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			for _, context := range []string{"font Demo plane 2", tt.wantPhase} {
				if !strings.Contains(err.Error(), context) {
					t.Fatalf("expected context %q, got %q", context, err)
				}
			}
			if count := strings.Count(err.Error(), wantErr.Error()); count != 1 {
				t.Fatalf("expected cause once, got %d in %q", count, err)
			}
		})
	}
}

func TestCreateCheatSheetsUserFontsDoesNotMutateInputAndSortsWork(t *testing.T) {
	ops := noOpFontDemoOperations()
	ops.userFont = func(string) (font.TTFLight, bool, error) {
		return font.TTFLight{Planes: map[int]bool{0: true}}, true, nil
	}
	var fileNames []string
	ops.createPDFFile = func(_ *model.XRefTable, fileName string, _ *model.Configuration) error {
		fileNames = append(fileNames, filepath.Base(fileName))
		return nil
	}

	fontNames := []string{"Zulu", "Alpha"}
	if err := createCheatSheetsUserFonts(fontNames, ops); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(fontNames, ","), "Zulu,Alpha"; got != want {
		t.Fatalf("caller-owned input changed: expected %q, got %q", want, got)
	}
	if got, want := strings.Join(fileNames, ","), "Alpha_BMP.pdf,Zulu_BMP.pdf"; got != want {
		t.Fatalf("expected deterministic work order %q, got %q", want, got)
	}
}

func TestCheatSheetBatchGenerationFailureLeavesOutputsUntouched(t *testing.T) {
	dir := t.TempDir()
	alpha := filepath.Join(dir, "Alpha_BMP.pdf")
	if err := os.WriteFile(alpha, []byte("old alpha"), 0600); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("generate Zulu")
	ops := noOpFontDemoOperations()
	ops.files = defaultCheatSheetFileOperations()
	ops.createPDFFile = func(_ *model.XRefTable, fileName string, _ *model.Configuration) error {
		if filepath.Base(fileName) == "Zulu_BMP.pdf" {
			return wantErr
		}
		return os.WriteFile(fileName, []byte("new "+filepath.Base(fileName)), 0600)
	}
	fonts := map[string]font.TTFLight{
		"Alpha": {Planes: map[int]bool{0: true}},
		"Zulu":  {Planes: map[int]bool{0: true}},
	}
	err := createUserFontDemoBatch(dir, []string{"Alpha", "Zulu"}, fonts, ops)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	bb, readErr := os.ReadFile(alpha)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(bb) != "old alpha" {
		t.Fatalf("expected existing output untouched, got %q", bb)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "Zulu_BMP.pdf")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected no Zulu output, got %v", statErr)
	}
}

func TestCheatSheetPublicationFailureRollsBackEarlierFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"Alpha_BMP.pdf", "Zulu_BMP.pdf"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("old "+name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	wantErr := errors.New("publish Zulu")
	ops := noOpFontDemoOperations()
	ops.files = defaultCheatSheetFileOperations()
	rename := ops.files.rename
	ops.files.rename = func(source, target string) error {
		if filepath.Base(source) == "Zulu_BMP.pdf" && strings.Contains(source, ".pdfcpu-font-cheatsheets-") {
			return wantErr
		}
		return rename(source, target)
	}
	ops.createPDFFile = func(_ *model.XRefTable, fileName string, _ *model.Configuration) error {
		return os.WriteFile(fileName, []byte("new "+filepath.Base(fileName)), 0600)
	}
	fonts := map[string]font.TTFLight{
		"Alpha": {Planes: map[int]bool{0: true}},
		"Zulu":  {Planes: map[int]bool{0: true}},
	}
	err := createUserFontDemoBatch(dir, []string{"Alpha", "Zulu"}, fonts, ops)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	for _, name := range []string{"Alpha_BMP.pdf", "Zulu_BMP.pdf"} {
		bb, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(bb) != "old "+name {
			t.Fatalf("%s: expected rollback to old content, got %q", name, bb)
		}
	}
}

func TestCheatSheetPublicationFailureJoinsRollbackFailure(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"Alpha_BMP.pdf", "Zulu_BMP.pdf"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("old "+name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	publishErr := errors.New("publish Zulu")
	rollbackErr := errors.New("restore Alpha")
	ops := noOpFontDemoOperations()
	ops.files = defaultCheatSheetFileOperations()
	rename := ops.files.rename
	ops.files.rename = func(source, target string) error {
		switch {
		case filepath.Base(source) == "Zulu_BMP.pdf" && strings.Contains(source, ".pdfcpu-font-cheatsheets-"):
			return publishErr
		case filepath.Base(source) == "Alpha_BMP.pdf" && strings.Contains(source, ".pdfcpu-font-cheatsheet-backup-"):
			return rollbackErr
		default:
			return rename(source, target)
		}
	}
	ops.createPDFFile = func(_ *model.XRefTable, fileName string, _ *model.Configuration) error {
		return os.WriteFile(fileName, []byte("new "+filepath.Base(fileName)), 0600)
	}
	fonts := map[string]font.TTFLight{
		"Alpha": {Planes: map[int]bool{0: true}},
		"Zulu":  {Planes: map[int]bool{0: true}},
	}
	err := createUserFontDemoBatch(dir, []string{"Alpha", "Zulu"}, fonts, ops)
	if !errors.Is(err, publishErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("expected joined publication and rollback errors, got %v", err)
	}
	if !strings.Contains(err.Error(), "cheat-sheet backup retained") {
		t.Fatalf("expected retained-backup context, got %q", err)
	}
}

func TestCheatSheetCleanupFailureAfterPublicationReturnsError(t *testing.T) {
	dir := t.TempDir()
	wantErr := errors.New("backup cleanup failed")
	ops := noOpFontDemoOperations()
	ops.files = defaultCheatSheetFileOperations()
	removeAll := ops.files.removeAll
	ops.files.removeAll = func(path string) error {
		if strings.Contains(path, ".pdfcpu-font-cheatsheet-backup-") {
			return wantErr
		}
		return removeAll(path)
	}
	ops.createPDFFile = func(_ *model.XRefTable, fileName string, _ *model.Configuration) error {
		return os.WriteFile(fileName, []byte("published"), 0600)
	}
	fonts := map[string]font.TTFLight{"Demo": {Planes: map[int]bool{0: true}}}
	err := createUserFontDemoBatch(dir, []string{"Demo"}, fonts, ops)
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "published cheat sheets") {
		t.Fatalf("expected published cleanup error, got %v", err)
	}
	bb, readErr := os.ReadFile(filepath.Join(dir, "Demo_BMP.pdf"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(bb) != "published" {
		t.Fatalf("expected published output to remain, got %q", bb)
	}
}

func TestCreateCheatSheetsUserFontsLoadsDefaultsAndRejectsUnknownFonts(t *testing.T) {
	t.Run("all installed fonts", func(t *testing.T) {
		ops := noOpFontDemoOperations()
		loadCalls := 0
		ops.loadUserFonts = func() error {
			loadCalls++
			return nil
		}
		ops.userFontNames = func() ([]string, error) {
			return []string{"Zulu", "Alpha"}, nil
		}
		ops.userFont = func(string) (font.TTFLight, bool, error) {
			return font.TTFLight{}, true, nil
		}
		if err := createCheatSheetsUserFonts(nil, ops); err != nil {
			t.Fatal(err)
		}
		if loadCalls != 1 {
			t.Fatalf("expected one explicit load, got %d", loadCalls)
		}
	})

	t.Run("load error", func(t *testing.T) {
		wantErr := errors.New("load fonts")
		ops := noOpFontDemoOperations()
		ops.loadUserFonts = func() error { return wantErr }
		err := createCheatSheetsUserFonts([]string{"Demo"}, ops)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
		if !strings.Contains(err.Error(), "create font cheat sheets: load user fonts") {
			t.Fatalf("expected load context, got %q", err)
		}
	})

	t.Run("unknown explicit font", func(t *testing.T) {
		ops := noOpFontDemoOperations()
		ops.userFont = func(fontName string) (font.TTFLight, bool, error) {
			return font.TTFLight{}, fontName != "Missing", nil
		}
		createCalls := 0
		ops.createXRef = func() (*model.XRefTable, error) {
			createCalls++
			return nil, nil
		}
		err := createCheatSheetsUserFonts([]string{"Known", "Missing"}, ops)
		if !errors.Is(err, ErrUserFontNotFound) {
			t.Fatalf("expected %v, got %v", ErrUserFontNotFound, err)
		}
		if !strings.Contains(err.Error(), "font Missing") {
			t.Fatalf("expected missing font context, got %q", err)
		}
		if createCalls != 0 {
			t.Fatalf("expected all names validated before output, got %d create calls", createCalls)
		}
	})

	t.Run("delegated output error", func(t *testing.T) {
		wantErr := errors.New("underlying failure")
		ops := noOpFontDemoOperations()
		ops.userFont = func(string) (font.TTFLight, bool, error) {
			return font.TTFLight{Planes: map[int]bool{0: true}}, true, nil
		}
		ops.createPDFFile = func(*model.XRefTable, string, *model.Configuration) error {
			return fmt.Errorf("create: write output: %w", wantErr)
		}
		err := createCheatSheetsUserFonts([]string{"Demo"}, ops)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
		for _, context := range []string{"create font cheat sheets", "font Demo plane 0", "create: write output"} {
			if !strings.Contains(err.Error(), context) {
				t.Fatalf("expected context %q, got %q", context, err)
			}
		}
		if count := strings.Count(err.Error(), wantErr.Error()); count != 1 {
			t.Fatalf("expected cause once, got %d in %q", count, err)
		}
		if strings.Contains(err.Error(), "create font cheat sheets: create font cheat sheet") {
			t.Fatalf("expected normalized operation context, got %q", err)
		}
	})
}
