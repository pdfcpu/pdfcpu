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
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func restoreCertificateTestDir(t *testing.T, dir string) {
	t.Helper()
	original := model.TrustedCertDir
	model.TrustedCertDir = dir
	t.Cleanup(func() {
		model.TrustedCertDir = original
	})
}

func certificateImportTestInputs(t *testing.T, dir string, names ...string) []string {
	t.Helper()
	source := filepath.Join("..", "pdfcpu", "model", "resources", "certs", "uk.p7c")
	bb, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	inFiles := make([]string, 0, len(names))
	for _, name := range names {
		inFile := filepath.Join(dir, name)
		if err := os.WriteFile(inFile, bb, 0600); err != nil {
			t.Fatal(err)
		}
		inFiles = append(inFiles, inFile)
	}
	return inFiles
}

func assertCertificateDirectoryEntries(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, entry.Name())
	}
	if !slices.Equal(got, want) {
		t.Fatalf("directory entries: got %q, want %q", got, want)
	}
}

// TestCertificateInputValidation verifies certificate API boundary guards.
func TestCertificateInputValidation(t *testing.T) {
	operations := []struct {
		name      string
		operation string
		run       func([]string) error
	}{
		{
			name:      "import",
			operation: "import certificates",
			run: func(inFiles []string) error {
				_, err := ImportCertificates(inFiles)
				return err
			},
		},
		{
			name:      "inspect",
			operation: "inspect certificates",
			run: func(inFiles []string) error {
				_, err := InspectCertificates(inFiles)
				return err
			},
		},
	}

	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			for _, inFiles := range [][]string{nil, {}} {
				if err := op.run(inFiles); !errors.Is(err, ErrMissingCertificateInput) {
					t.Fatalf("expected %v, got %v", ErrMissingCertificateInput, err)
				}
			}

			inFiles := []string{"not-accessed.pem", " \t "}
			want := slices.Clone(inFiles)
			err := op.run(inFiles)
			if !errors.Is(err, ErrMissingCertificateInput) {
				t.Fatalf("expected %v, got %v", ErrMissingCertificateInput, err)
			}
			wantContext := op.operation + ": certificate input 2"
			if !strings.Contains(err.Error(), wantContext) {
				t.Fatalf("expected indexed certificate context, got %v", err)
			}
			if !slices.Equal(inFiles, want) {
				t.Fatalf("input mutated: got %q, want %q", inFiles, want)
			}
		})
	}
}

// TestCertificateInputErrorContext verifies API operation and input context preserve causes.
func TestCertificateInputErrorContext(t *testing.T) {
	missingFile := filepath.Join(t.TempDir(), "missing.pem")
	tests := []struct {
		name string
		run  func(string) error
		want string
	}{
		{
			name: "import",
			run: func(inFile string) error {
				_, err := ImportCertificates([]string{inFile})
				return err
			},
			want: fmt.Sprintf("import certificates: input 1 %q", missingFile),
		},
		{
			name: "inspect",
			run: func(inFile string) error {
				_, err := InspectCertificates([]string{inFile})
				return err
			},
			want: fmt.Sprintf("inspect certificates: input 1 %q: load certificates", missingFile),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(missingFile)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, err)
			}
		})
	}
}

// TestCertificateInputErrorContextPreservesSentinel verifies unsupported types remain discoverable.
func TestCertificateInputErrorContextPreservesSentinel(t *testing.T) {
	if ErrUnsupportedCertificateFile != pdfcpu.ErrUnsupportedCertificateFile {
		t.Fatal("API unsupported-certificate sentinel is not a core alias")
	}
	inFile := filepath.Join(t.TempDir(), "certificate.txt")
	_, err := InspectCertificates([]string{inFile})
	if !errors.Is(err, ErrUnsupportedCertificateFile) {
		t.Fatalf("expected %v, got %v", ErrUnsupportedCertificateFile, err)
	}
	if !errors.Is(err, pdfcpu.ErrUnknownFileType) {
		t.Fatalf("expected compatibility alias %v, got %v", pdfcpu.ErrUnknownFileType, err)
	}
	want := fmt.Sprintf("inspect certificates: input 1 %q: load certificates", inFile)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

// TestInspectCertificateErrorContext verifies individual certificate failures include both indexes.
func TestInspectCertificateErrorContext(t *testing.T) {
	inFile := "certificates.pem"
	_, err := inspectCertificate(inFile, 1, 2, nil)
	if !errors.Is(err, ErrMissingCertificate) {
		t.Fatalf("expected %v, got %v", ErrMissingCertificate, err)
	}
	want := `inspect certificates: input 2 "certificates.pem": inspect certificate 3`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

// TestCertificateMultiFileSummaries verifies successful batch output remains stable.
func TestCertificateMultiFileSummaries(t *testing.T) {
	dir := t.TempDir()
	trustedDir := filepath.Join(dir, "trusted")
	if err := os.Mkdir(trustedDir, 0700); err != nil {
		t.Fatal(err)
	}
	restoreCertificateTestDir(t, trustedDir)

	source := filepath.Join("..", "pdfcpu", "model", "resources", "certs", "uk.p7c")
	bb, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	inFiles := []string{filepath.Join(dir, "first.p7c"), filepath.Join(dir, "second.p7c")}
	for _, inFile := range inFiles {
		if err := os.WriteFile(inFile, bb, 0600); err != nil {
			t.Fatal(err)
		}
	}
	certs, err := pdfcpu.LoadCertificatesFile(source)
	if err != nil {
		t.Fatal(err)
	}
	count := len(certs)

	imported, err := ImportCertificates(inFiles)
	if err != nil {
		t.Fatal(err)
	}
	wantImported := []string{
		fmt.Sprintf("%s: %d certificates", inFiles[0], count),
		fmt.Sprintf("%s: %d certificates", inFiles[1], count),
		fmt.Sprintf("imported %d certificates", 2*count),
	}
	if !slices.Equal(imported, wantImported) {
		t.Fatalf("import summary: got %q, want %q", imported, wantImported)
	}

	inspected, err := InspectCertificates(inFiles)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := inspected[len(inspected)-1], fmt.Sprintf("inspected %d certificates", 2*count); got != want {
		t.Fatalf("inspect summary: got %q, want %q", got, want)
	}
}

// TestImportCertificatesPreflightPreventsPartialPublication verifies every source is checked before writing.
func TestImportCertificatesPreflightPreventsPartialPublication(t *testing.T) {
	dir := t.TempDir()
	trustedDir := filepath.Join(dir, "trusted")
	if err := os.Mkdir(trustedDir, 0700); err != nil {
		t.Fatal(err)
	}
	restoreCertificateTestDir(t, trustedDir)

	source := filepath.Join("..", "pdfcpu", "model", "resources", "certs", "uk.p7c")
	bb, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	validFile := filepath.Join(dir, "valid.p7c")
	if err := os.WriteFile(validFile, bb, 0600); err != nil {
		t.Fatal(err)
	}
	emptyFile := filepath.Join(dir, "empty.pem")
	if err := os.WriteFile(emptyFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	missingFile := filepath.Join(dir, "missing.crt")

	_, err = ImportCertificates([]string{validFile, emptyFile, missingFile})
	if !errors.Is(err, ErrNoCertificates) || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected joined preflight errors, got %v", err)
	}
	for _, fileName := range []string{emptyFile, missingFile} {
		if !strings.Contains(err.Error(), fmt.Sprintf("%q", fileName)) {
			t.Fatalf("expected preflight error for %q, got %v", fileName, err)
		}
	}
	entries, readErr := os.ReadDir(trustedDir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("preflight published certificate files: %v", entries)
	}
}

// TestImportCertificatesRejectsDuplicateDestinations verifies colliding target names fail preflight.
func TestImportCertificatesRejectsDuplicateDestinations(t *testing.T) {
	dir := t.TempDir()
	trustedDir := filepath.Join(dir, "trusted")
	restoreCertificateTestDir(t, trustedDir)

	source := filepath.Join("..", "pdfcpu", "model", "resources", "certs", "uk.p7c")
	bb, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	inFiles := make([]string, 0, 2)
	for _, subdir := range []string{"first", "second"} {
		inputDir := filepath.Join(dir, subdir)
		if err := os.Mkdir(inputDir, 0700); err != nil {
			t.Fatal(err)
		}
		inFile := filepath.Join(inputDir, "shared.p7c")
		if err := os.WriteFile(inFile, bb, 0600); err != nil {
			t.Fatal(err)
		}
		inFiles = append(inFiles, inFile)
	}

	_, err = ImportCertificates(inFiles)
	if !errors.Is(err, ErrDuplicateCertificateDestination) {
		t.Fatalf("expected %v, got %v", ErrDuplicateCertificateDestination, err)
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%q", filepath.Join(trustedDir, "shared.p7c"))) {
		t.Fatalf("expected duplicate destination context, got %v", err)
	}
	if _, statErr := os.Stat(trustedDir); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("duplicate preflight created trusted directory: %v", statErr)
	}
}

// TestImportCertificatesEnsuresTrustedDirectory verifies publication creates its destination directory.
func TestImportCertificatesEnsuresTrustedDirectory(t *testing.T) {
	dir := t.TempDir()
	trustedDir := filepath.Join(dir, "trusted")
	restoreCertificateTestDir(t, trustedDir)

	source := filepath.Join("..", "pdfcpu", "model", "resources", "certs", "uk.p7c")
	inFile := filepath.Join(dir, "valid.p7c")
	bb, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inFile, bb, 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := ImportCertificates([]string{inFile}); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(trustedDir, "valid.p7c")
	if _, err := pdfcpu.LoadCertificatesFile(outFile); err != nil {
		t.Fatalf("load published certificates: %v", err)
	}
}

// TestImportCertificatesRefreshesLoadedTrustPool verifies committed imports replace an already-loaded pool.
func TestImportCertificatesRefreshesLoadedTrustPool(t *testing.T) {
	trustedDir := t.TempDir()
	restoreCertificateTestDir(t, trustedDir)
	pdfcpu.InvalidateCertificatePool()
	t.Cleanup(pdfcpu.InvalidateCertificatePool)
	if err := pdfcpu.LoadCertificates(); err != nil {
		t.Fatal(err)
	}
	oldPool := model.UserCertPool
	if oldPool == nil || len(oldPool.Subjects()) != 0 {
		t.Fatal("expected an initially empty certificate pool")
	}

	inFile := certificateImportTestInputs(t, t.TempDir(), "imported.p7c")[0]
	if _, err := ImportCertificates([]string{inFile}); err != nil {
		t.Fatal(err)
	}
	if err := pdfcpu.LoadCertificates(); err != nil {
		t.Fatal(err)
	}
	if pool := model.UserCertPool; pool == nil || pool == oldPool || len(pool.Subjects()) == 0 {
		t.Fatal("successful import did not refresh the loaded certificate pool")
	}
}

// TestImportCertificatesReplacesExistingDestination verifies replacement is the public overwrite policy.
func TestImportCertificatesReplacesExistingDestination(t *testing.T) {
	dir := t.TempDir()
	trustedDir := filepath.Join(dir, "trusted")
	if err := os.Mkdir(trustedDir, 0700); err != nil {
		t.Fatal(err)
	}
	restoreCertificateTestDir(t, trustedDir)
	inFile := certificateImportTestInputs(t, dir, "existing.p7c")[0]
	outFile := filepath.Join(trustedDir, "existing.p7c")
	original := []byte("existing certificate data")
	if err := os.WriteFile(outFile, original, 0600); err != nil {
		t.Fatal(err)
	}

	out, err := ImportCertificates([]string{inFile})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || !strings.HasPrefix(out[0], inFile+": ") || !strings.HasPrefix(out[1], "imported ") {
		t.Fatalf("unexpected replacement summary: %q", out)
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Equal(bb, original) {
		t.Fatal("existing destination was not replaced")
	}
	if _, err := pdfcpu.LoadCertificatesFile(outFile); err != nil {
		t.Fatalf("load replacement certificate: %v", err)
	}
	assertCertificateDirectoryEntries(t, trustedDir, "existing.p7c")
}

// TestImportCertificatesStagesEntireBatchBeforePublication verifies staging failure leaves destinations untouched.
func TestImportCertificatesStagesEntireBatchBeforePublication(t *testing.T) {
	dir := t.TempDir()
	trustedDir := filepath.Join(dir, "trusted")
	if err := os.Mkdir(trustedDir, 0700); err != nil {
		t.Fatal(err)
	}
	restoreCertificateTestDir(t, trustedDir)
	inFiles := certificateImportTestInputs(t, dir, "one.p7c", "two.p7c")

	stageErr := errors.New("stage failed")
	ops := defaultCertificateImportOperations()
	saveCertificates := ops.saveCertificates
	saveCalls := 0
	ops.saveCertificates = func(certs []*x509.Certificate, fileName string) error {
		saveCalls++
		if filepath.Dir(fileName) != trustedDir {
			t.Fatalf("staging file outside destination directory: %s", fileName)
		}
		if saveCalls == 2 {
			return stageErr
		}
		return saveCertificates(certs, fileName)
	}
	replaceCalls := 0
	replace := ops.files.replaceFn
	ops.files.replaceFn = func(oldName, newName string) error {
		replaceCalls++
		return replace(oldName, newName)
	}

	out, err := importCertificates(inFiles, ops)
	if !errors.Is(err, stageErr) {
		t.Fatalf("expected %v, got %v", stageErr, err)
	}
	if out != nil {
		t.Fatalf("staging failure returned success output: %q", out)
	}
	if replaceCalls != 0 {
		t.Fatalf("staging failure reached publication: %d renames", replaceCalls)
	}
	assertCertificateDirectoryEntries(t, trustedDir)
}

// TestImportCertificatesSecondPublicationFailureLeavesNoPartialInstallation verifies new batches roll back completely.
func TestImportCertificatesSecondPublicationFailureLeavesNoPartialInstallation(t *testing.T) {
	dir := t.TempDir()
	trustedDir := filepath.Join(dir, "trusted")
	if err := os.Mkdir(trustedDir, 0700); err != nil {
		t.Fatal(err)
	}
	restoreCertificateTestDir(t, trustedDir)
	inFiles := certificateImportTestInputs(t, dir, "one.p7c", "two.p7c")

	publishErr := errors.New("second publication failed")
	ops := defaultCertificateImportOperations()
	replace := ops.files.replaceFn
	publishCalls := 0
	ops.files.replaceFn = func(oldName, newName string) error {
		if strings.Contains(filepath.Base(oldName), ".stage-") {
			publishCalls++
			if publishCalls == 2 {
				return publishErr
			}
		}
		return replace(oldName, newName)
	}

	out, err := importCertificates(inFiles, ops)
	if !errors.Is(err, publishErr) {
		t.Fatalf("expected %v, got %v", publishErr, err)
	}
	if out != nil {
		t.Fatalf("publication failure returned success output: %q", out)
	}
	if publishCalls != 2 {
		t.Fatalf("publication attempts: got %d, want 2", publishCalls)
	}
	assertCertificateDirectoryEntries(t, trustedDir)
}

// TestImportCertificatesRollsBackPublishedFiles verifies publication failure restores the whole batch.
func TestImportCertificatesRollsBackPublishedFiles(t *testing.T) {
	dir := t.TempDir()
	trustedDir := filepath.Join(dir, "trusted")
	if err := os.Mkdir(trustedDir, 0700); err != nil {
		t.Fatal(err)
	}
	restoreCertificateTestDir(t, trustedDir)
	inFiles := certificateImportTestInputs(t, dir, "one.p7c", "two.p7c")
	originals := map[string]string{
		"one.p7c": "original one",
		"two.p7c": "original two",
	}
	for name, content := range originals {
		if err := os.WriteFile(filepath.Join(trustedDir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	publishErr := errors.New("publish failed")
	ops := defaultCertificateImportOperations()
	replace := ops.files.replaceFn
	publishCalls := 0
	ops.files.replaceFn = func(oldName, newName string) error {
		if strings.Contains(filepath.Base(oldName), ".stage-") {
			publishCalls++
			if publishCalls == 2 {
				return publishErr
			}
		}
		return replace(oldName, newName)
	}

	out, err := importCertificates(inFiles, ops)
	if !errors.Is(err, publishErr) {
		t.Fatalf("expected %v, got %v", publishErr, err)
	}
	if out != nil {
		t.Fatalf("publication failure returned success output: %q", out)
	}
	for name, want := range originals {
		bb, readErr := os.ReadFile(filepath.Join(trustedDir, name))
		if readErr != nil || string(bb) != want {
			t.Fatalf("restored %s: got %q, err=%v, want %q", name, bb, readErr, want)
		}
	}
	assertCertificateDirectoryEntries(t, trustedDir, "one.p7c", "two.p7c")
}

// TestImportCertificatesJoinsPublicationRollbackAndCleanupErrors verifies transaction failures remain discoverable.
func TestImportCertificatesJoinsPublicationRollbackAndCleanupErrors(t *testing.T) {
	dir := t.TempDir()
	trustedDir := filepath.Join(dir, "trusted")
	if err := os.Mkdir(trustedDir, 0700); err != nil {
		t.Fatal(err)
	}
	restoreCertificateTestDir(t, trustedDir)
	inFiles := certificateImportTestInputs(t, dir, "one.p7c", "two.p7c")
	for _, name := range []string{"one.p7c", "two.p7c"} {
		if err := os.WriteFile(filepath.Join(trustedDir, name), []byte("original "+name), 0600); err != nil {
			t.Fatal(err)
		}
	}

	publishErr := errors.New("publish failed")
	rollbackErr := errors.New("rollback failed")
	cleanupErr := errors.New("cleanup failed")
	ops := defaultCertificateImportOperations()
	replace := ops.files.replaceFn
	publishCalls := 0
	failedStage := ""
	ops.files.replaceFn = func(oldName, newName string) error {
		if strings.Contains(filepath.Base(oldName), ".stage-") {
			publishCalls++
			if publishCalls == 2 {
				failedStage = oldName
				return publishErr
			}
		}
		return replace(oldName, newName)
	}
	remove := ops.files.removeFn
	ops.files.removeFn = func(path string) error {
		switch path {
		case filepath.Join(trustedDir, "one.p7c"):
			return errors.Join(rollbackErr, remove(path))
		case failedStage:
			return errors.Join(cleanupErr, remove(path))
		}
		return remove(path)
	}

	out, err := importCertificates(inFiles, ops)
	for _, want := range []error{publishErr, rollbackErr, cleanupErr} {
		if !errors.Is(err, want) {
			t.Fatalf("expected joined %v, got %v", want, err)
		}
	}
	if out != nil {
		t.Fatalf("transaction failure returned success output: %q", out)
	}
	assertCertificateDirectoryEntries(t, trustedDir, "one.p7c", "two.p7c")
}

// TestImportCertificatesReportsCleanupFailureAfterCommit verifies summaries require a clean commit.
func TestImportCertificatesReportsCleanupFailureAfterCommit(t *testing.T) {
	dir := t.TempDir()
	trustedDir := filepath.Join(dir, "trusted")
	if err := os.Mkdir(trustedDir, 0700); err != nil {
		t.Fatal(err)
	}
	restoreCertificateTestDir(t, trustedDir)
	inFiles := certificateImportTestInputs(t, dir, "one.p7c")
	if err := os.WriteFile(filepath.Join(trustedDir, "one.p7c"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}

	cleanupErr := errors.New("backup cleanup failed")
	ops := defaultCertificateImportOperations()
	remove := ops.files.removeFn
	backupRemovals := 0
	ops.files.removeFn = func(path string) error {
		if strings.Contains(filepath.Base(path), ".backup-") {
			backupRemovals++
			if backupRemovals == 2 {
				return errors.Join(cleanupErr, remove(path))
			}
		}
		return remove(path)
	}

	out, err := importCertificates(inFiles, ops)
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("expected %v, got %v", cleanupErr, err)
	}
	if out != nil {
		t.Fatalf("cleanup failure returned success output: %q", out)
	}
	if _, err := pdfcpu.LoadCertificatesFile(filepath.Join(trustedDir, "one.p7c")); err != nil {
		t.Fatalf("committed certificate missing after cleanup failure: %v", err)
	}
	assertCertificateDirectoryEntries(t, trustedDir, "one.p7c")
}

// TestImportCertificatesConcurrentWithTrustPoolLoad verifies import and lazy loads remain race-safe.
func TestImportCertificatesConcurrentWithTrustPoolLoad(t *testing.T) {
	trustedDir := t.TempDir()
	restoreCertificateTestDir(t, trustedDir)
	pdfcpu.InvalidateCertificatePool()
	t.Cleanup(pdfcpu.InvalidateCertificatePool)
	inFiles := certificateImportTestInputs(t, t.TempDir(), "one.p7c", "two.p7c")

	firstPublished := make(chan struct{})
	continuePublication := make(chan struct{})
	ops := defaultCertificateImportOperations()
	replace := ops.files.replaceFn
	publishCalls := 0
	ops.files.replaceFn = func(oldName, newName string) error {
		if err := replace(oldName, newName); err != nil {
			return err
		}
		if strings.Contains(filepath.Base(oldName), ".stage-") {
			publishCalls++
			if publishCalls == 1 {
				close(firstPublished)
				<-continuePublication
			}
		}
		return nil
	}

	importDone := make(chan error, 1)
	go func() {
		_, err := importCertificates(inFiles, ops)
		importDone <- err
	}()

	<-firstPublished
	loadErr := pdfcpu.LoadCertificates()
	partialPool := model.UserCertPool
	close(continuePublication)
	importErr := <-importDone
	if loadErr != nil {
		t.Fatalf("concurrent load: %v", loadErr)
	}
	if importErr != nil {
		t.Fatalf("concurrent import: %v", importErr)
	}
	if partialPool == nil || len(partialPool.Subjects()) == 0 {
		t.Fatal("concurrent load did not publish its intermediate pool")
	}

	if err := pdfcpu.LoadCertificates(); err != nil {
		t.Fatal(err)
	}
	if pool := model.UserCertPool; pool == nil || pool == partialPool || len(pool.Subjects()) == 0 {
		t.Fatal("committed import did not replace the intermediate certificate pool")
	}
	assertCertificateDirectoryEntries(t, trustedDir, "one.p7c", "two.p7c")
}

// TestListCertificatesText verifies API listing preserves partial output, errors, and certificate counts.
func TestListCertificatesText(t *testing.T) {
	dir := t.TempDir()
	restoreCertificateTestDir(t, dir)
	source := filepath.Join("..", "pdfcpu", "model", "resources", "certs", "uk.p7c")
	bb, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "uk.p7c"), bb, 0600); err != nil {
		t.Fatal(err)
	}
	invalidFiles := []string{filepath.Join(dir, "empty-1.pem"), filepath.Join(dir, "empty-2.pem")}
	for _, fileName := range invalidFiles {
		if err := os.WriteFile(fileName, nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	certs, err := pdfcpu.LoadCertificatesFile(source)
	if err != nil {
		t.Fatal(err)
	}

	out, err := ListCertificates(false)
	if !errors.Is(err, ErrNoCertificates) {
		t.Fatalf("expected %v, got %v", ErrNoCertificates, err)
	}
	for _, fileName := range invalidFiles {
		if !strings.Contains(err.Error(), fmt.Sprintf("%q", fileName)) {
			t.Fatalf("expected joined error for %q, got %v", fileName, err)
		}
	}
	want := fmt.Sprintf("total installed certs: %d", len(certs))
	if !slices.Contains(out, want) {
		t.Fatalf("expected %q, got %q", want, out)
	}
	for _, name := range []string{"uk.p7c:", "empty-1.pem:", "empty-2.pem:"} {
		found := false
		for _, line := range out {
			if strings.HasPrefix(line, string(filepath.Separator)+name) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected output for %q, got %q", name, out)
		}
	}
}

// TestListCertificatesJSONReportsFileErrors verifies JSON preserves partial results and error identity.
func TestListCertificatesJSONReportsFileErrors(t *testing.T) {
	dir := t.TempDir()
	restoreCertificateTestDir(t, dir)
	source := filepath.Join("..", "pdfcpu", "model", "resources", "certs", "uk.p7c")
	bb, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "valid.p7c"), bb, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "empty.pem"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	out, err := ListCertificates(true)
	if !errors.Is(err, ErrNoCertificates) {
		t.Fatalf("expected %v, got %v", ErrNoCertificates, err)
	}
	if len(out) != 1 {
		t.Fatalf("expected one JSON result, got %d", len(out))
	}
	var result struct {
		TotalInstalledCerts int `json:"totalInstalledCerts"`
		Files               []struct {
			Name         string            `json:"name"`
			Certificates []json.RawMessage `json:"certificates"`
			Error        string            `json:"error"`
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(out[0]), &result); err != nil {
		t.Fatal(err)
	}
	if result.TotalInstalledCerts == 0 || len(result.Files) != 2 {
		t.Fatalf("unexpected certificate result: %+v", result)
	}
	var foundValid, foundInvalid bool
	for _, file := range result.Files {
		switch {
		case strings.HasSuffix(file.Name, "valid.p7c"):
			foundValid = len(file.Certificates) > 0 && file.Error == ""
		case strings.HasSuffix(file.Name, "empty.pem"):
			foundInvalid = strings.Contains(file.Error, ErrNoCertificates.Error())
		}
	}
	if !foundValid || !foundInvalid {
		t.Fatalf("expected valid and invalid file results, got %+v", result.Files)
	}
}

// TestListCertificatesDirectoryErrorContext verifies API listing owns directory setup context.
func TestListCertificatesDirectoryErrorContext(t *testing.T) {
	dir := t.TempDir()
	fileName := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(fileName, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	restoreCertificateTestDir(t, filepath.Join(fileName, "certs"))

	_, err := ListCertificates(false)
	if err == nil {
		t.Fatal("expected directory creation error")
	}
	if !strings.Contains(err.Error(), "list certificates: create trusted certificate directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}
