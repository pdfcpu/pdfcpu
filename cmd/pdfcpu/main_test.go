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
	"io"
	"os"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
)

func captureStderr(t *testing.T, f func()) string {
	t.Helper()

	stderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
	}()

	f()
	w.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestPrintErrorOmitsStackTraceByDefault(t *testing.T) {
	needStackTraceSave := needStackTrace
	needStackTrace = false
	defer func() {
		needStackTrace = needStackTraceSave
	}()
	err := fault.Panic{
		Err:   errors.New("boom"),
		Stack: []byte("goroutine 1 [running]:\nstack frame"),
	}

	out := captureStderr(t, func() {
		printError(err)
	})

	if out != "boom\n" {
		t.Fatalf("got %q, want terse error", out)
	}
}

func TestPrintErrorIncludesStackTraceWhenRequested(t *testing.T) {
	needStackTraceSave := needStackTrace
	needStackTrace = true
	defer func() {
		needStackTrace = needStackTraceSave
	}()
	err := fault.Panic{
		Err:   errors.New("boom"),
		Stack: []byte("goroutine 1 [running]:\nstack frame"),
	}

	out := captureStderr(t, func() {
		printError(err)
	})

	if !strings.Contains(out, "Fatal: boom\n") {
		t.Fatalf("got %q, want fatal error", out)
	}
	if !strings.Contains(out, "Stack Trace:\ngoroutine 1 [running]:") {
		t.Fatalf("got %q, want stack trace", out)
	}
}
