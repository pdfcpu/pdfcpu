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
	"strings"
	"testing"
)

func TestVersionRejectsArgs(t *testing.T) {
	out, err := runPDFCPU(t, "version", "extra")
	if err == nil {
		t.Fatalf("expected version to reject extra args, output:\n%s", out)
	}
}

func TestVersionOutput(t *testing.T) {
	out, err := runPDFCPU(t, "version")
	if err != nil {
		t.Fatalf("version failed: %v\n%s", err, out)
	}
	for _, want := range []string{"version:", "commit:", "date:", "go:"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("expected version output to contain %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "config:") {
		t.Fatalf("version output contains configuration state:\n%s", out)
	}
}
