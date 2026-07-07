//go:build darwin

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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestWriteAttachmentsUsesDarwinFilesystemCaseSensitivity(t *testing.T) {
	outDir := t.TempDir()
	aa := []model.Attachment{
		{Reader: strings.NewReader("first"), ID: "attachment-1", FileName: "shared.txt"},
		{Reader: strings.NewReader("second"), ID: "attachment-2", FileName: "SHARED.TXT"},
	}

	err := writeAttachments(outDir, aa)

	if errors.Is(err, ErrAttachmentOutputCollision) {
		if _, statErr := os.Stat(filepath.Join(outDir, "shared.txt")); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("expected collision detection before writing, got %v", statErr)
		}
		requireNoAttachmentReservations(t, outDir)
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"shared.txt", "SHARED.TXT"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Fatal(err)
		}
	}
	requireNoAttachmentReservations(t, outDir)
}

func TestWriteAttachmentsUsesDarwinUnicodeNormalization(t *testing.T) {
	outDir := t.TempDir()
	names := []string{"caf\u00e9.txt", "cafe\u0301.txt"}
	aa := []model.Attachment{
		{Reader: strings.NewReader("first"), ID: "attachment-1", FileName: names[0]},
		{Reader: strings.NewReader("second"), ID: "attachment-2", FileName: names[1]},
	}

	err := writeAttachments(outDir, aa)

	if errors.Is(err, ErrAttachmentOutputCollision) {
		for _, name := range names {
			if _, statErr := os.Stat(filepath.Join(outDir, name)); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("expected collision detection before writing, got %v", statErr)
			}
		}
		requireNoAttachmentReservations(t, outDir)
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Fatal(err)
		}
	}
	requireNoAttachmentReservations(t, outDir)
}
