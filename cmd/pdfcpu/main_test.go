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
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type commandOutputErrorWriter struct {
	writes int
	err    error
}

func (w *commandOutputErrorWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == 2 {
		return 0, w.err
	}
	return len(p), nil
}

func TestRunCommandJoinsDispatchAndOutputErrors(t *testing.T) {
	dispatchErr := errors.New("dispatch failed")
	writeErr := errors.New("stdout failed")
	w := &commandOutputErrorWriter{err: writeErr}
	dispatch := func(*cli.Command) ([]string, error) {
		return []string{"first", "second", "third"}, dispatchErr
	}
	err := runCommandWithOutput(&cli.Command{}, w, dispatch, false)
	if !errors.Is(err, dispatchErr) || !errors.Is(err, writeErr) {
		t.Fatalf("expected joined dispatch and stdout errors, got %v", err)
	}
	if w.writes != 2 {
		t.Fatalf("expected writes to stop after checked failure, got %d", w.writes)
	}
	if !strings.Contains(err.Error(), "output line 2") {
		t.Fatalf("expected output line context, got %q", err)
	}
}

// TestRotation verifies command-line rotation parsing across signed integer boundaries.
func TestRotation(t *testing.T) {
	minInt := -int(^uint(0)>>1) - 1
	tests := []struct {
		input string
		want  int
		valid bool
	}{
		{input: "90", want: 90, valid: true},
		{input: "-90", want: -90, valid: true},
		{input: "0", want: 0, valid: true},
		{input: "45"},
		{input: "-45"},
		{input: strconv.Itoa(minInt)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := rotation(tt.input)
			if tt.valid {
				if err != nil {
					t.Fatal(err)
				}
				if got != tt.want {
					t.Fatalf("expected %d, got %d", tt.want, got)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "multiple of 90") {
				t.Fatalf("expected rotation error, got %v", err)
			}
		})
	}
}

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

func TestParseForGridUsesAPIBoundary(t *testing.T) {
	nup := model.DefaultNUpConfig()
	nup.PageGrid = true
	argInd := 0
	if err := parseForGrid([]string{"2", "3"}, nup, &argInd); err != nil {
		t.Fatal(err)
	}
	if argInd != 2 {
		t.Fatalf("expected argument index 2, got %d", argInd)
	}
	if nup.Grid == nil || nup.Grid.Width != 3 || nup.Grid.Height != 2 {
		t.Fatalf("expected 2x3 grid, got %v", nup.Grid)
	}
}

// TestHandleResizeCommandGuardsAndContext verifies resize command-handler boundaries.
func TestHandleResizeCommandGuardsAndContext(t *testing.T) {
	if err := handleResizeCommand(nil, nil); !errors.Is(err, api.ErrMissingConfiguration) {
		t.Fatalf("expected %v, got %v", api.ErrMissingConfiguration, err)
	}
	conf := model.NewDefaultConfiguration()
	if err := handleResizeCommand(conf, nil); !errors.Is(err, api.ErrMissingResizeConfiguration) {
		t.Fatalf("expected %v, got %v", api.ErrMissingResizeConfiguration, err)
	}
	if err := handleResizeCommand(conf, []string{"sc:.5"}); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}
	err := handleResizeCommand(conf, []string{"bad", "missing.pdf"})
	if err == nil || !strings.Contains(err.Error(), "resize: parse configuration") {
		t.Fatalf("expected resize configuration context, got %v", err)
	}
}

func TestRemoveBoxBoundariesDefersPolicyToAPI(t *testing.T) {
	pb, err := removeBoxBoundaries("")
	if err != nil {
		t.Fatal(err)
	}
	if pb == nil {
		t.Fatal("expected empty page boundaries request")
	}

	pb, err = removeBoxBoundaries("media")
	if err != nil {
		t.Fatal(err)
	}
	if pb.Media == nil {
		t.Fatal("expected MediaBox removal request")
	}
}

func TestParseForGridUsesGridParseDefinition(t *testing.T) {
	argInd := 0
	err := parseForGrid([]string{"2", "3"}, nil, &argInd)
	if !errors.Is(err, api.ErrMissingGridConfiguration) {
		t.Fatalf("expected %v, got %v", api.ErrMissingGridConfiguration, err)
	}
	if argInd != 0 {
		t.Fatalf("expected argument index to remain 0, got %d", argInd)
	}
}

func TestParseGridArgumentsRejectMissingInput(t *testing.T) {
	nup := model.DefaultNUpConfig()
	nup.PageGrid = true
	_, err := parseAfterNUpDetails([]string{"2", "3"}, nup, 0, nil, "out.pdf", true)
	if err == nil || !strings.Contains(err.Error(), "missing input file") {
		t.Fatalf("expected missing input error, got %v", err)
	}
}

func TestParseForGridErrorsIncludeDimensionContext(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{name: "missing", args: []string{"2"}, want: "missing grid dimensions"},
		{name: "rows", args: []string{"x", "3"}, want: `parse grid rows "x"`},
		{name: "columns", args: []string{"2", "x"}, want: `parse grid columns "x"`},
		{name: "dimensions", args: []string{"0", "3"}, want: "parse grid dimensions"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			nup := model.DefaultNUpConfig()
			nup.PageGrid = true
			argInd := 0
			err := parseForGrid(tt.args, nup, &argInd)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestHandleGridCommandAddsArgumentContext(t *testing.T) {
	err := handleGridCommand(model.NewDefaultConfiguration(), []string{"out.pdf", "x", "3", "in.pdf"})
	if err == nil || !strings.Contains(err.Error(), `grid: parse arguments: parse grid rows "x"`) {
		t.Fatalf("expected grid argument context, got %v", err)
	}
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

func TestHandleValidateCommandRejectsEmptyExpansion(t *testing.T) {
	opts := &validateOptions{mode: "relaxed"}
	err := handleValidateCommand(model.NewDefaultConfiguration(), []string{"missing.txt"}, opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}
	if !strings.Contains(err.Error(), "missing PDF input") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestHandleValidateCommandReturnsExpansionError(t *testing.T) {
	opts := &validateOptions{mode: "relaxed"}
	err := handleValidateCommand(model.NewDefaultConfiguration(), []string{"[*"}, opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "validate: expand input") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestSplitSpanRejectsExtraArgs(t *testing.T) {
	_, err := splitSpan([]string{"in.pdf", "out", "2", "4"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "span mode accepts at most one span") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestHandleSplitCommandRejectsBookmarkExtraArgs(t *testing.T) {
	opts := &splitOptions{mode: "bookmark"}
	err := handleSplitCommand(model.NewDefaultConfiguration(), []string{"missing.pdf", t.TempDir(), "2"}, opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bookmark mode does not accept") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestHandleExportBookmarksAcceptsStdout(t *testing.T) {
	err := handleExportBookmarksCommand(model.NewDefaultConfiguration(), []string{"missing.pdf", "-"})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "needs extension") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestSplitPageNumbersRejectsDuplicates(t *testing.T) {
	_, err := splitPageNumbers([]string{"in.pdf", "out", "2", "2"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unique") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestSplitPageNumbersRejectsOutOfOrder(t *testing.T) {
	_, err := splitPageNumbers([]string{"in.pdf", "out", "10", "2"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "sorted ascending") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestSplitPageNumbersPreservesInputOrder(t *testing.T) {
	pageNrs, err := splitPageNumbers([]string{"in.pdf", "out", "2", "10"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{2, 10}; !reflect.DeepEqual(pageNrs, want) {
		t.Fatalf("got %v, want %v", pageNrs, want)
	}
}
