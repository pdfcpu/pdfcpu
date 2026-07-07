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
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestRemoveSignaturesFileFailurePreservesExistingOutput verifies staged publication.
func TestRemoveSignaturesFileFailurePreservesExistingOutput(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "invalid.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(inFile, []byte("not a PDF"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outFile, original, 0640); err != nil {
		t.Fatal(err)
	}

	if err := RemoveSignaturesFile(inFile, outFile, nil); err == nil {
		t.Fatal("expected signature-removal failure")
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bb, original) {
		t.Fatalf("existing output changed: got %q, want %q", bb, original)
	}
}
