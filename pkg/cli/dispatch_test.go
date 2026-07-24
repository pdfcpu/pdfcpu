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

package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestDispatchRecoversUnexpectedPanicWithStackMetadata(t *testing.T) {
	mode := model.CommandMode(-1)
	dispatchTable[mode] = func(*Command) ([]string, error) {
		panic("boom")
	}
	defer delete(dispatchTable, mode)

	_, err := Dispatch(&Command{Mode: mode, Conf: model.NewDefaultConfiguration()})
	if err == nil {
		t.Fatal("expected error")
	}

	var p fault.Panic
	if !errors.As(err, &p) {
		t.Fatalf("got %T, want fault.Panic", err)
	}
	if got := err.Error(); !strings.Contains(got, "unexpected panic attack: boom") {
		t.Fatalf("got %q, want unexpected panic message", got)
	}
	if strings.Contains(err.Error(), "goroutine ") {
		t.Fatalf("error string includes stack trace: %q", err.Error())
	}
	if !strings.Contains(string(p.Stack), "TestDispatchRecoversUnexpectedPanicWithStackMetadata") {
		t.Fatalf("stack trace does not include test frame:\n%s", p.Stack)
	}
}

// TestDispatchRejectsAddSignature verifies the unimplemented command mode cannot report false success.
func TestDispatchRejectsAddSignature(t *testing.T) {
	_, err := Dispatch(&Command{Mode: model.ADDSIGNATURE})
	if !errors.Is(err, ErrUnsupportedCommandMode) {
		t.Fatalf("expected unsupported command mode, got %v", err)
	}
	if !strings.Contains(err.Error(), "mode") {
		t.Fatalf("expected command mode context, got %v", err)
	}
}
