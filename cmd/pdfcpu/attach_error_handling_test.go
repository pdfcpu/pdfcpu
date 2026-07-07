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

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestAttachmentFilesPreserveDescriptionDuringGlobExpansion(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		filepath.Join(dir, "one.pdf"),
		filepath.Join(dir, "two.pdf"),
	}
	for _, fileName := range files {
		if err := os.WriteFile(fileName, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	const desc = "description,with,commas"
	got, err := attachmentFiles([]string{filepath.Join(dir, "*.pdf") + "," + desc}, true, "add attachments")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(files) {
		t.Fatalf("expected %d files, got %d", len(files), len(got))
	}
	for i, fileName := range files {
		want := fileName + "," + desc
		if got[i] != want {
			t.Fatalf("expected %q, got %q", want, got[i])
		}
	}
}

func TestAttachmentFilesRecognizeAllGlobMetacharacters(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		filepath.Join(dir, "question1.txt"),
		filepath.Join(dir, "classa.txt"),
	}
	for _, fileName := range files {
		if err := os.WriteFile(fileName, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{name: "question", pattern: filepath.Join(dir, "question?.txt"), want: files[0]},
		{name: "character class", pattern: filepath.Join(dir, "class[ab].txt"), want: files[1]},
	}

	if os.PathSeparator != '\\' {
		fileName := filepath.Join(dir, "escaped?.txt")
		if err := os.WriteFile(fileName, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		tests = append(tests, struct {
			name    string
			pattern string
			want    string
		}{
			name:    "escape",
			pattern: filepath.Join(dir, `escaped\?.txt`),
			want:    fileName,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := attachmentFiles([]string{tt.pattern}, true, "add attachments")
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0] != tt.want {
				t.Fatalf("expected [%q], got %q", tt.want, got)
			}
		})
	}
}

func TestAttachmentFilesReportsGlobWithoutMatches(t *testing.T) {
	pattern := filepath.Join(t.TempDir(), "missing?.pdf")

	_, err := attachmentFiles([]string{pattern + ",description,with,commas"}, true, "add attachments")

	if !errors.Is(err, errNoAttachmentGlobMatches) {
		t.Fatalf("expected %v, got %v", errNoAttachmentGlobMatches, err)
	}
	want := `add attachments: expand attachment glob "` + pattern + `": no matches`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q in %q", want, err)
	}
}

func TestAttachmentFilesGlobErrorsIncludeContext(t *testing.T) {
	_, err := attachmentFiles([]string{"[*"}, true, "add portfolio attachments")
	if !errors.Is(err, filepath.ErrBadPattern) {
		t.Fatalf("expected %v, got %v", filepath.ErrBadPattern, err)
	}
	if !strings.Contains(err.Error(), `add portfolio attachments: expand attachment glob "[*"`) {
		t.Fatalf("expected glob context, got %v", err)
	}
}

func TestAttachmentCLIHandlerBoundaryGuards(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{name: "list configuration", run: func() error {
			return handleListAttachmentsCommand(nil, []string{"in.pdf"})
		}, want: api.ErrMissingConfiguration},
		{name: "list input", run: func() error {
			return handleListAttachmentsCommand(conf, nil)
		}, want: api.ErrMissingPDFInput},
		{name: "add input", run: func() error {
			return handleAddAttachmentsCommand(conf, nil)
		}, want: api.ErrMissingPDFInput},
		{name: "add attachment", run: func() error {
			return handleAddAttachmentsCommand(conf, []string{"in.pdf"})
		}, want: api.ErrNoAttachmentAdded},
		{name: "portfolio attachment", run: func() error {
			return handleAddAttachmentsPortfolioCommand(conf, []string{"in.pdf"})
		}, want: api.ErrNoAttachmentAdded},
		{name: "remove input", run: func() error {
			return handleRemoveAttachmentsCommand(conf, nil)
		}, want: api.ErrMissingPDFInput},
		{name: "extract input", run: func() error {
			return handleExtractAttachmentsCommand(conf, nil)
		}, want: api.ErrMissingPDFInput},
		{name: "extract output", run: func() error {
			return handleExtractAttachmentsCommand(conf, []string{"in.pdf"})
		}, want: api.ErrMissingPDFOutput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestExtractAttachmentsHandlerReportsOutputDirectoryContext(t *testing.T) {
	forceSave := force
	force = false
	t.Cleanup(func() {
		force = forceSave
	})

	outDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outDir, "existing"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	err := handleExtractAttachmentsCommand(model.NewDefaultConfiguration(), []string{"in.pdf", outDir})
	if err == nil || !strings.Contains(err.Error(), "extract attachments: prepare output directory") {
		t.Fatalf("expected output directory context, got %v", err)
	}
}
