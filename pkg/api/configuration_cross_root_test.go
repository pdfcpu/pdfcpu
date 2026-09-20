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

package api_test

import (
	"bytes"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type configurationTreeEntry struct {
	mode    fs.FileMode
	modTime int64
	data    string
}

func copyConfigurationFixture(t *testing.T, source, destination string) {
	t.Helper()
	bb, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, bb, 0600); err != nil {
		t.Fatal(err)
	}
}

func replaceConfigurationSetting(t *testing.T, path, old, replacement string) {
	t.Helper()
	bb, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	bb = bytes.ReplaceAll(bb, []byte("\r\n"), []byte("\n"))
	updated := bytes.Replace(bb, []byte(old), []byte(replacement), 1)
	if bytes.Equal(updated, bb) {
		t.Fatalf("configuration does not contain %q", old)
	}
	if err := os.WriteFile(path, updated, 0600); err != nil {
		t.Fatal(err)
	}
}

func prepareReadOnlyConfigurationRoot(t *testing.T, root, fontFile, certificateFile string) string {
	t.Helper()
	configDir := filepath.Join(root, "pdfcpu")
	fontDir := filepath.Join(configDir, "fonts")
	certificateDir := filepath.Join(configDir, "certs")
	for _, dir := range []string{fontDir, certificateDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	copyConfigurationFixture(
		t,
		filepath.Join("..", "pdfcpu", "model", "resources", "config.yml"),
		filepath.Join(configDir, "config.yml"),
	)
	report, err := font.InstallTrueTypeFontResult(fontDir, fontFile)
	if err != nil {
		t.Fatalf("install user font: %v", err)
	}
	if len(report.Fonts) != 1 || len(report.Warnings) != 0 {
		t.Fatalf("install user font: unexpected report: %+v", report)
	}
	copyConfigurationFixture(t, certificateFile, filepath.Join(certificateDir, filepath.Base(certificateFile)))
	return report.Fonts[0].PostScriptName
}

func setConfigurationTreeMode(t *testing.T, root string, dirMode, fileMode fs.FileMode) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.Chmod(path, dirMode)
		}
		return os.Chmod(path, fileMode)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func makeConfigurationTreeReadOnly(t *testing.T, root string) {
	t.Helper()
	setConfigurationTreeMode(t, root, 0555, 0444)
	t.Cleanup(func() {
		setConfigurationTreeMode(t, root, 0755, 0644)
	})
}

func configurationTreeSnapshot(t *testing.T, root string) map[string]configurationTreeEntry {
	t.Helper()
	entries := map[string]configurationTreeEntry{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := configurationSnapshotInfo(path)
		if err != nil {
			return err
		}
		var data []byte
		if entry.Type().IsRegular() {
			data, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entries[relative] = configurationTreeEntry{
			mode:    info.Mode(),
			modTime: info.ModTime().UnixNano(),
			data:    string(data),
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func certificateFileSubjects(t *testing.T, path string) map[string]struct{} {
	t.Helper()
	certificates, err := pdfcpu.LoadCertificatesFile(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	subjects := make(map[string]struct{}, len(certificates))
	for _, certificate := range certificates {
		subjects[string(certificate.RawSubject)] = struct{}{}
	}
	return subjects
}

func configurationCertificateSubjects(t *testing.T, conf *model.Configuration) map[string]struct{} {
	t.Helper()
	pool, err := pdfcpu.CertificatePoolForConfiguration(t.Context(), conf)
	if err != nil {
		t.Fatal(err)
	}
	subjects := make(map[string]struct{}, len(pool.Subjects()))
	for _, subject := range pool.Subjects() {
		subjects[string(subject)] = struct{}{}
	}
	return subjects
}

func requireConfigurationFont(t *testing.T, conf *model.Configuration, name string, want bool) {
	t.Helper()
	repository := (&model.XRefTable{Conf: conf}).FontRepository()
	got, err := repository.IsUserFont(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("font %s: got %t, want %t", name, got, want)
	}
}

func requireReadOnlyConfiguration(t *testing.T, root string, conf *model.Configuration, validationMode int, fontName, otherFontName string, certificateSubjects map[string]struct{}) {
	t.Helper()
	configDir := filepath.Join(root, "pdfcpu")
	if conf.Path != filepath.Join(configDir, "config.yml") {
		t.Fatalf("configuration path: got %q, want root %q", conf.Path, configDir)
	}
	if conf.ValidationMode != validationMode {
		t.Fatalf("validation mode: got %d, want %d", conf.ValidationMode, validationMode)
	}
	fontDir, available := conf.UserFontStore()
	if !available || fontDir != filepath.Join(configDir, "fonts") {
		t.Fatalf("user-font store: got %q, available=%t", fontDir, available)
	}
	certificateDir, available := conf.TrustedCertificateStore()
	if !available || certificateDir != filepath.Join(configDir, "certs") {
		t.Fatalf("certificate store: got %q, available=%t", certificateDir, available)
	}
	requireConfigurationFont(t, conf, fontName, true)
	requireConfigurationFont(t, conf, otherFontName, false)
	if got := configurationCertificateSubjects(t, conf); !maps.Equal(got, certificateSubjects) {
		t.Fatalf("certificate subjects: got %d, want %d", len(got), len(certificateSubjects))
	}
}

func loadReadOnlyConfiguration(t *testing.T, root string) *model.Configuration {
	t.Helper()
	conf, err := api.LoadConfiguration(api.ConfigurationOptions{
		Root: root,
		Mode: api.ConfigurationModeReadOnly,
	})
	if err != nil {
		t.Fatalf("load read-only configuration: %v", err)
	}
	return conf
}

func TestLoadConfigurationIsolatesReadOnlyRoots(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	fontA := prepareReadOnlyConfigurationRoot(
		t,
		rootA,
		filepath.Join("..", "testdata", "fonts", "unifont-13.0.03.ttf"),
		filepath.Join("..", "pdfcpu", "model", "resources", "certs", "uk.p7c"),
	)
	fontB := prepareReadOnlyConfigurationRoot(
		t,
		rootB,
		filepath.Join("..", "testdata", "fonts", "unifont_jp-13.0.03.ttf"),
		filepath.Join("..", "pdfcpu", "model", "resources", "certs", "at.p7c"),
	)
	if fontA == fontB {
		t.Fatalf("font fixtures share PostScript name %q", fontA)
	}
	replaceConfigurationSetting(
		t,
		filepath.Join(rootA, "pdfcpu", "config.yml"),
		"validationMode: ValidationRelaxed",
		"validationMode: ValidationStrict",
	)

	subjectsA := certificateFileSubjects(
		t,
		filepath.Join("..", "pdfcpu", "model", "resources", "certs", "uk.p7c"),
	)
	subjectsB := certificateFileSubjects(
		t,
		filepath.Join("..", "pdfcpu", "model", "resources", "certs", "at.p7c"),
	)
	if maps.Equal(subjectsA, subjectsB) {
		t.Fatal("certificate fixtures have identical subjects")
	}

	makeConfigurationTreeReadOnly(t, rootA)
	makeConfigurationTreeReadOnly(t, rootB)
	beforeA := configurationTreeSnapshot(t, rootA)
	beforeB := configurationTreeSnapshot(t, rootB)

	confA := loadReadOnlyConfiguration(t, rootA)
	requireReadOnlyConfiguration(t, rootA, confA, model.ValidationStrict, fontA, fontB, subjectsA)
	confB := loadReadOnlyConfiguration(t, rootB)
	requireReadOnlyConfiguration(t, rootB, confB, model.ValidationRelaxed, fontB, fontA, subjectsB)
	confA = loadReadOnlyConfiguration(t, rootA)
	requireReadOnlyConfiguration(t, rootA, confA, model.ValidationStrict, fontA, fontB, subjectsA)

	if afterA := configurationTreeSnapshot(t, rootA); !maps.Equal(beforeA, afterA) {
		t.Fatal("read-only loading modified configuration root A")
	}
	if afterB := configurationTreeSnapshot(t, rootB); !maps.Equal(beforeB, afterB) {
		t.Fatal("read-only loading modified configuration root B")
	}
}

// configurationSnapshotInfo reads metadata from an open handle to avoid stale Windows directory enumeration metadata.
func configurationSnapshotInfo(path string) (fs.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Stat()
}
