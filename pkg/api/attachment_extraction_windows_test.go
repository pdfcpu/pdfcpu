//go:build windows

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

func attachmentCountWindows(t *testing.T, fileName string) int {
	t.Helper()
	f, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	aa, err := Attachments(f, nil)
	if err != nil {
		t.Fatal(err)
	}
	return len(aa)
}

func TestWriteAttachmentReplacesExistingOutputWindows(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "attachment.bin")
	if err := os.WriteFile(fileName, []byte("existing attachment"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := writeAttachmentToPath(fileName, model.Attachment{Reader: strings.NewReader("replacement")})
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "replacement" {
		t.Fatalf("expected replacement output, got %q", got)
	}
	requireNoAttachmentTempFiles(t, fileName)
}

func TestAddAttachmentsFileInPlaceWindows(t *testing.T) {
	inFile := copyAttachmentTestInput(t)

	err := AddAttachmentsFile(inFile, "", []string{attachmentTestInputFile()}, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	if got := attachmentCountWindows(t, inFile); got != 1 {
		t.Fatalf("expected one attachment, got %d", got)
	}
}

func TestRemoveAttachmentsFileInPlaceWindows(t *testing.T) {
	inFile := copyAttachmentTestInput(t)
	attachment := attachmentTestInputFile()
	if err := AddAttachmentsFile(inFile, "", []string{attachment}, false, nil); err != nil {
		t.Fatal(err)
	}

	err := RemoveAttachmentsFile(inFile, "", []string{filepath.Base(attachment)}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if got := attachmentCountWindows(t, inFile); got != 0 {
		t.Fatalf("expected no attachments, got %d", got)
	}
}

func TestAddAttachmentsFileReplacesExistingDistinctOutputWindows(t *testing.T) {
	inFile := copyAttachmentTestInput(t)
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("existing output"), 0o640); err != nil {
		t.Fatal(err)
	}

	err := AddAttachmentsFile(inFile, outFile, []string{attachmentTestInputFile()}, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	if got := attachmentCountWindows(t, outFile); got != 1 {
		t.Fatalf("expected one attachment, got %d", got)
	}
	if got := attachmentCountWindows(t, inFile); got != 0 {
		t.Fatalf("input changed: got %d attachments", got)
	}
}

func TestWriteAttachmentsRejectsCaseEquivalentOutputsWindows(t *testing.T) {
	outDir := t.TempDir()
	aa := []model.Attachment{
		{Reader: strings.NewReader("first"), ID: "attachment-1", FileName: "shared.txt"},
		{Reader: strings.NewReader("second"), ID: "attachment-2", FileName: "SHARED.TXT"},
	}

	err := writeAttachments(outDir, aa)

	if !errors.Is(err, ErrAttachmentOutputCollision) {
		t.Fatalf("expected %v, got %v", ErrAttachmentOutputCollision, err)
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "shared.txt")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected collision detection before writing, got %v", statErr)
	}
}
