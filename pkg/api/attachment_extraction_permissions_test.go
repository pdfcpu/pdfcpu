//go:build !windows

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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestWriteAttachmentPreservesExistingOutputPermissions(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "attachment.bin")
	if err := os.WriteFile(fileName, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := writeAttachmentToPath(fileName, model.Attachment{Reader: strings.NewReader("replacement")})
	if err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected permissions 0600, got %04o", got)
	}
}
