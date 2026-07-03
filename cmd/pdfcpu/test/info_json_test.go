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
	"strings"
	"testing"
)

func TestInfoJSONWithFreshConfigWritesOnlyJSONToStdout(t *testing.T) {
	stdout, stderr, err := runPDFCPUWithConfig(
		t,
		t.TempDir(),
		"info",
		"--json",
		repoFile(t, "pkg", "testdata", "test.pdf"),
	)
	if err != nil {
		t.Fatalf("info --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !json.Valid(stdout) {
		t.Fatalf("stdout is not valid JSON:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if strings.Contains(string(stdout), "installing user font") {
		t.Fatalf("stdout contains config bootstrap log:\n%s", stdout)
	}
}
