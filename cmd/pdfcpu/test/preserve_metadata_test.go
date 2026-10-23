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

package test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type documentInfo struct {
	Producer     string `json:"producer"`
	CreationDate string `json:"creationDate"`
}

func documentInfoFor(t *testing.T, file string) documentInfo {
	t.Helper()
	stdout, stderr, err := runPDFCPUWithConfig(t, "disable", "info", "--json", file)
	if err != nil {
		t.Fatalf("info --json %s: %v\nstdout:\n%s\nstderr:\n%s", file, err, stdout, stderr)
	}
	var decoded struct {
		Infos []documentInfo `json:"infos"`
	}
	if err := json.Unmarshal(stdout, &decoded); err != nil {
		t.Fatalf("decode info JSON for %s: %v\nstdout:\n%s", file, err, stdout)
	}
	if len(decoded.Infos) != 1 {
		t.Fatalf("info JSON for %s has %d documents, want 1\nstdout:\n%s", file, len(decoded.Infos), stdout)
	}
	return decoded.Infos[0]
}

func TestPreserveMetadataFlag(t *testing.T) {
	inFile := repoFile(t, "pkg", "testdata", "test.pdf")
	original := documentInfoFor(t, inFile)

	t.Run("default", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "out.pdf")
		out, err := runPDFCPU(t, "optimize", inFile, outFile)
		if err != nil {
			t.Fatalf("optimize: %v\n%s", err, out)
		}
		got := documentInfoFor(t, outFile)
		want := "pdfcpu " + model.VersionStr
		if got.Producer != want {
			t.Fatalf("producer = %q, want %q", got.Producer, want)
		}
	})

	t.Run("preserve", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "out.pdf")
		out, err := runPDFCPU(t, "optimize", "--preserve-metadata", inFile, outFile)
		if err != nil {
			t.Fatalf("optimize --preserve-metadata: %v\n%s", err, out)
		}
		got := documentInfoFor(t, outFile)
		if got.Producer != original.Producer {
			t.Fatalf("producer = %q, want %q", got.Producer, original.Producer)
		}
		if got.CreationDate != original.CreationDate {
			t.Fatalf("creationDate = %q, want %q", got.CreationDate, original.CreationDate)
		}
	})
}
