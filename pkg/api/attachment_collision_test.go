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

func TestWriteAttachmentsRejectsSameRunOutputCollision(t *testing.T) {
	outDir := t.TempDir()
	aa := []model.Attachment{
		{Reader: strings.NewReader("first"), ID: "attachment-1", FileName: "shared.txt"},
		{Reader: strings.NewReader("second"), ID: "attachment-2", FileName: "shared.txt"},
	}

	err := writeAttachments(outDir, aa)

	if !errors.Is(err, ErrAttachmentOutputCollision) {
		t.Fatalf("expected %v, got %v", ErrAttachmentOutputCollision, err)
	}
	for _, want := range []string{"attachment-1", "attachment-2", filepath.Join(outDir, "shared.txt")} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "shared.txt")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected collision detection before writing, got %v", statErr)
	}
	requireNoAttachmentReservations(t, outDir)
}

func requireNoAttachmentReservations(t *testing.T, outDir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(outDir, ".*.pdfcpu-reservation-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("attachment output reservations remain: %v", matches)
	}
}
