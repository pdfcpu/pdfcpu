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
	"bytes"
	"context"
	"errors"
	"io"
	stdlog "log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
	pdfcpuLog "github.com/pdfcpu/pdfcpu/pkg/log"
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
	dispatch := func(context.Context, *cli.Command) ([]string, error) {
		return []string{"first", "second", "third"}, dispatchErr
	}
	err := runCommandWithOutput(t.Context(), &cli.Command{}, w, dispatch, false)
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

func TestRunCommandConfiguresValidationNoticeOutput(t *testing.T) {
	quietSave := quiet
	defer func() {
		quiet = quietSave
	}()

	quiet = false
	cmd := &cli.Command{}
	_ = runCommand(t.Context(), cmd)
	if cmd.NoticeOutput == nil {
		t.Fatal("expected validation notice output in normal mode")
	}

	quiet = true
	_ = runCommand(t.Context(), cmd)
	if cmd.NoticeOutput != nil {
		t.Fatal("expected validation notice output suppression in quiet mode")
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
	if err := handleResizeCommand(t.Context(), nil, nil); !errors.Is(err, api.ErrMissingConfiguration) {
		t.Fatalf("expected %v, got %v", api.ErrMissingConfiguration, err)
	}
	conf := model.NewDefaultConfiguration()
	if err := handleResizeCommand(t.Context(), conf, nil); !errors.Is(err, api.ErrMissingResizeConfiguration) {
		t.Fatalf("expected %v, got %v", api.ErrMissingResizeConfiguration, err)
	}
	if err := handleResizeCommand(
		t.Context(), conf, []string{"sc:.5"},
	); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}
	err := handleResizeCommand(t.Context(), conf, []string{"bad", "missing.pdf"})
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
	err := handleGridCommand(
		t.Context(), model.NewDefaultConfiguration(), []string{"out.pdf", "x", "3", "in.pdf"},
	)
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
	err := handleValidateCommand(t.Context(), model.NewDefaultConfiguration(), []string{"missing.txt"}, opts)
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
	err := handleValidateCommand(t.Context(), model.NewDefaultConfiguration(), []string{"[*"}, opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "validate: expand input") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestHandleValidateCommandPropagatesContextCancellation(t *testing.T) {
	inFile := filepath.Join("..", "..", "pkg", "testdata", "test.pdf")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleValidateCommand(
		ctx,
		model.NewDefaultConfiguration(),
		[]string{inFile},
		&validateOptions{mode: "relaxed"},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleOptimizeCommandPropagatesContextCancellation(t *testing.T) {
	inFile := filepath.Join("..", "..", "pkg", "testdata", "test.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleOptimizeCommand(
		c,
		model.NewDefaultConfiguration(),
		[]string{inFile, outFile},
		&optimizeCommandOptions{},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleMergeCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleMergeCommand(
		c,
		model.NewDefaultConfiguration(),
		[]string{"out.pdf", "ignored.pdf"},
		&mergeOptions{mode: "create"},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleSplitCommandPropagatesContextCancellation(t *testing.T) {
	inFile := filepath.Join("..", "..", "pkg", "testdata", "test.pdf")
	tests := []struct {
		name string
		mode string
		args func(string) []string
	}{
		{"span", "span", func(outDir string) []string { return []string{inFile, outDir} }},
		{"page", "page", func(outDir string) []string { return []string{inFile, outDir, "2"} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, cancel := context.WithCancel(t.Context())
			cancel()
			err := handleSplitCommand(
				c,
				model.NewDefaultConfiguration(),
				tt.args(t.TempDir()),
				&splitOptions{mode: tt.mode},
			)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestSplitCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd := splitCmd()
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"ignored.pdf", "out"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleTrimCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleTrimCommand(c, model.NewDefaultConfiguration(), []string{"ignored.pdf"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestTrimCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd := trimCmd()
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"ignored.pdf"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleCollectCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleCollectCommand(c, model.NewDefaultConfiguration(), []string{"ignored.pdf"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestCollectCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd := collectCmd()
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"ignored.pdf"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleRotateCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleRotateCommand(c, model.NewDefaultConfiguration(), []string{"ignored.pdf", "90"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestRotateCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd := rotateCmd()
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"ignored.pdf", "90"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleInsertPagesCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleInsertPagesCommand(
		c, model.NewDefaultConfiguration(), []string{"ignored.pdf"}, &pagesInsertOptions{mode: "before"},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestInsertPagesCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd, _, err := pagesCmd().Find([]string{"insert"})
	if err != nil {
		t.Fatal(err)
	}
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"ignored.pdf"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleRemovePagesCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleRemovePagesCommand(c, model.NewDefaultConfiguration(), []string{"ignored.pdf"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestRemovePagesCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd, _, err := pagesCmd().Find([]string{"remove"})
	if err != nil {
		t.Fatal(err)
	}
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"ignored.pdf"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleCropCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleCropCommand(c, model.NewDefaultConfiguration(), []string{"10", "ignored.pdf"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestCropCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd := cropCmd()
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"10", "ignored.pdf"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleAddBoxesCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleAddBoxesCommand(c, model.NewDefaultConfiguration(), []string{"media:dim:100 100", "ignored.pdf"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleListBoxesCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleListBoxesCommand(c, model.NewDefaultConfiguration(), []string{"ignored.pdf"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestListBoxesCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd, _, err := boxesCmd().Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"ignored.pdf"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestAddBoxesCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd, _, err := boxesCmd().Find([]string{"add"})
	if err != nil {
		t.Fatal(err)
	}
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"media:dim:100 100", "ignored.pdf"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleRemoveBoxesCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleRemoveBoxesCommand(c, model.NewDefaultConfiguration(), []string{"crop", "ignored.pdf"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestRemoveBoxesCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd, _, err := boxesCmd().Find([]string{"remove"})
	if err != nil {
		t.Fatal(err)
	}
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"crop", "ignored.pdf"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleZoomCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleZoomCommand(c, model.NewDefaultConfiguration(), []string{"factor:.5", "ignored.pdf"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestZoomCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd := zoomCmd()
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"factor:.5", "ignored.pdf"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleResizeCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleResizeCommand(c, model.NewDefaultConfiguration(), []string{"sc:.5", "ignored.pdf"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestResizeCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd := resizeCmd()
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"sc:.5", "ignored.pdf"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleNUpCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleNUpCommand(c, model.NewDefaultConfiguration(), []string{"out.pdf", "2", "ignored.pdf"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestNUpCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd := nupCmd()
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"out.pdf", "2", "ignored.pdf"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleGridCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleGridCommand(c, model.NewDefaultConfiguration(), []string{"out.pdf", "2", "2", "ignored.pdf"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestGridCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd := gridCmd()
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"out.pdf", "2", "2", "ignored.pdf"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleBookletCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleBookletCommand(c, model.NewDefaultConfiguration(), []string{"out.pdf", "2", "ignored.pdf"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestBookletCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd := bookletCmd()
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"out.pdf", "2", "ignored.pdf"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandlePosterCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handlePosterCommand(
		c, model.NewDefaultConfiguration(), []string{"dim:100 100", "ignored.pdf", "out"},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestPosterCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd := posterCmd()
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"dim:100 100", "ignored.pdf", "out"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleNDownCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleNDownCommand(c, model.NewDefaultConfiguration(), []string{"2", "ignored.pdf", "out"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestNDownCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd := ndownCmd()
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"2", "ignored.pdf", "out"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestHandleCutCommandPropagatesContextCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	err := handleCutCommand(c, model.NewDefaultConfiguration(), []string{"hor:.5", "ignored.pdf", "out"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestCutCommandUsesCobraContext(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	cmd := cutCmd()
	cmd.SetContext(c)

	if err := cmd.RunE(cmd, []string{"hor:.5", "ignored.pdf", "out"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestExecuteRejectsNilContext(t *testing.T) {
	if err := execute(nil); !errors.Is(err, cli.ErrMissingContext) {
		t.Fatalf("got %v, want cli.ErrMissingContext", err)
	}
}

func TestRestoreDefaultSignalHandlingAfterCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	restored := make(chan struct{})
	go restoreDefaultSignalHandling(c, func() { close(restored) })

	cancel()
	select {
	case <-restored:
	case <-time.After(time.Second):
		t.Fatal("default signal handling was not restored")
	}
}

func TestCollectInFilesExpandsRecursivePDFPattern(t *testing.T) {
	dir := t.TempDir()
	nestedDir := filepath.Join(dir, "nested")
	if err := os.Mkdir(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	rootFile := filepath.Join(dir, "root.pdf")
	nestedFile := filepath.Join(nestedDir, "nested.pdf")
	for _, fn := range []string{rootFile, nestedFile} {
		if err := os.WriteFile(fn, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := collectInFiles(model.NewDefaultConfiguration(), []string{filepath.Join(dir, "**", "*.pdf")})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{nestedFile, rootFile}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestHandleValidateCommandStreamsFailuresInQuietMode(t *testing.T) {
	quietSave := quiet
	quiet = true
	defer func() {
		quiet = quietSave
	}()

	var validationErr error
	stderr := captureStderr(t, func() {
		validationErr = handleValidateCommand(
			t.Context(),
			model.NewDefaultConfiguration(),
			[]string{"missing1.pdf", "missing2.pdf"},
			&validateOptions{mode: "relaxed"},
		)
	})

	if validationErr == nil {
		t.Fatal("expected error")
	}
	if validationErr.Error() != "validation failed: 2 of 2 files invalid" {
		t.Fatalf("got %q", validationErr.Error())
	}
	for _, fn := range []string{"missing1.pdf", "missing2.pdf"} {
		if !strings.Contains(stderr, fn) {
			t.Fatalf("expected %q in stderr, got %q", fn, stderr)
		}
	}
}

func TestValidateCommandDefinesProgressFlag(t *testing.T) {
	if flag := validateCmd().Flags().Lookup("progress"); flag == nil {
		t.Fatal("expected progress flag")
	}
}

func TestHandleValidateCommandReportsProgressInQuietMode(t *testing.T) {
	quietSave := quiet
	quiet = true
	defer func() {
		quiet = quietSave
	}()

	var validationErr error
	stderr := captureStderr(t, func() {
		validationErr = handleValidateCommand(
			t.Context(),
			model.NewDefaultConfiguration(),
			[]string{"missing1.pdf", "missing2.pdf"},
			&validateOptions{mode: "relaxed", progress: true},
		)
	})

	if validationErr == nil {
		t.Fatal("expected error")
	}
	for _, fn := range []string{"missing1.pdf", "missing2.pdf"} {
		progress := "validating(mode=relaxed) " + fn + " ..."
		progressIndex := strings.Index(stderr, progress)
		failureIndex := strings.Index(stderr, "validate: open "+fn)
		if progressIndex < 0 || failureIndex < 0 {
			t.Fatalf("expected progress and failure for %q, got %q", fn, stderr)
		}
		if progressIndex >= failureIndex {
			t.Fatalf("expected progress before failure for %q, got %q", fn, stderr)
		}
	}
	if strings.Contains(stderr, "validation ok") {
		t.Fatalf("expected quiet progress without success output, got %q", stderr)
	}
}

func TestHandleValidateCommandQuietProgressForValidInput(t *testing.T) {
	quietSave := quiet
	quiet = true
	pdfcpuLog.SetCLILogger(nil)
	defer func() {
		quiet = quietSave
		pdfcpuLog.SetCLILogger(nil)
	}()

	inFile := filepath.Join("..", "..", "pkg", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name     string
		progress bool
		optimize bool
		want     string
	}{
		{name: "quiet", progress: false},
		{
			name:     "quiet progress",
			progress: true,
			want:     "validating(mode=relaxed) " + inFile + " ...\n",
		},
		{
			name:     "quiet progress with optimization",
			progress: true,
			optimize: true,
			want:     "validating(mode=relaxed) " + inFile + " ...\noptimizing...\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var validationErr error
			stderr := captureStderr(t, func() {
				validationErr = handleValidateCommand(
					t.Context(),
					model.NewDefaultConfiguration(),
					[]string{inFile},
					&validateOptions{mode: "relaxed", progress: tt.progress, optimize: tt.optimize},
				)
			})
			if validationErr != nil {
				t.Fatal(validationErr)
			}
			if stderr != tt.want {
				t.Fatalf("got %q, want %q", stderr, tt.want)
			}
		})
	}
}

func TestHandleValidateCommandRoutesNonQuietProgressThroughCommandOutput(t *testing.T) {
	quietSave := quiet
	quiet = false
	var cliOutput bytes.Buffer
	pdfcpuLog.SetCLILogger(stdlog.New(&cliOutput, "", 0))
	defer func() {
		quiet = quietSave
		pdfcpuLog.SetCLILogger(nil)
	}()

	inFile := filepath.Join("..", "..", "pkg", "samples", "create", "primitives", "textAndAlignment.pdf")
	var validationErr error
	stderr := captureStderr(t, func() {
		validationErr = handleValidateCommand(
			t.Context(),
			model.NewDefaultConfiguration(),
			[]string{inFile},
			&validateOptions{mode: "relaxed", progress: true},
		)
	})
	if validationErr != nil {
		t.Fatal(validationErr)
	}
	if got := strings.Count(stderr, "validating(mode=relaxed)"); got != 1 {
		t.Fatalf("got %d progress lines, want 1: %q", got, stderr)
	}
	if strings.Contains(cliOutput.String(), "validating(mode=relaxed)") {
		t.Fatalf("validation progress bypassed command output: %q", cliOutput.String())
	}
}

func TestHandleValidateCommandReportsRecursiveProgressInTraversalOrder(t *testing.T) {
	quietSave := quiet
	quiet = true
	defer func() {
		quiet = quietSave
	}()

	dir := t.TempDir()
	nestedDir := filepath.Join(dir, "nested")
	if err := os.Mkdir(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	nestedFile := filepath.Join(nestedDir, "nested.pdf")
	rootFile := filepath.Join(dir, "root.pdf")
	for _, fn := range []string{nestedFile, rootFile} {
		if err := os.WriteFile(fn, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var validationErr error
	stderr := captureStderr(t, func() {
		validationErr = handleValidateCommand(
			t.Context(),
			model.NewDefaultConfiguration(),
			[]string{filepath.Join(dir, "**", "*.pdf")},
			&validateOptions{mode: "relaxed", progress: true},
		)
	})
	if validationErr == nil {
		t.Fatal("expected error")
	}

	nestedProgress := strings.Index(stderr, "validating(mode=relaxed) "+nestedFile+" ...")
	rootProgress := strings.Index(stderr, "validating(mode=relaxed) "+rootFile+" ...")
	if nestedProgress < 0 || rootProgress < 0 || nestedProgress >= rootProgress {
		t.Fatalf("expected recursive progress in traversal order, got %q", stderr)
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
	err := handleSplitCommand(
		t.Context(),
		model.NewDefaultConfiguration(),
		[]string{"missing.pdf", t.TempDir(), "2"},
		opts,
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bookmark mode does not accept") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestHandleExportBookmarksAcceptsStdout(t *testing.T) {
	err := handleExportBookmarksCommand(
		t.Context(), model.NewDefaultConfiguration(), []string{"missing.pdf", "-"},
	)
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
