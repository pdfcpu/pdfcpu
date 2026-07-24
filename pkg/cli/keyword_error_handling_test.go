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
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestKeywordCommandBoundaryGuards verifies public CLI entry points preserve missing-file sentinels.
func TestKeywordCommandBoundaryGuards(t *testing.T) {
	empty := ""
	inFile := "in.pdf"
	outFile := "out.pdf"
	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{name: "list nil command", run: func() error {
			_, err := ListKeywords(nil)
			return err
		}, want: ErrMissingCommand},
		{name: "list missing input", run: func() error {
			_, err := ListKeywords(&Command{})
			return err
		}, want: api.ErrMissingPDFInput},
		{name: "list empty input", run: func() error {
			_, err := ListKeywords(&Command{InFile: &empty})
			return err
		}, want: api.ErrMissingPDFInput},
		{name: "add nil command", run: func() error {
			_, err := AddKeywords(nil)
			return err
		}, want: ErrMissingCommand},
		{name: "add missing input", run: func() error {
			_, err := AddKeywords(&Command{OutFile: &outFile})
			return err
		}, want: api.ErrMissingPDFInput},
		{name: "add empty input", run: func() error {
			_, err := AddKeywords(&Command{InFile: &empty, OutFile: &outFile})
			return err
		}, want: api.ErrMissingPDFInput},
		{name: "add missing output", run: func() error {
			_, err := AddKeywords(&Command{InFile: &inFile})
			return err
		}, want: api.ErrMissingPDFOutput},
		{name: "add empty output", run: func() error {
			_, err := AddKeywords(&Command{InFile: &inFile, OutFile: &empty})
			return err
		}, want: api.ErrMissingPDFOutput},
		{name: "remove nil command", run: func() error {
			_, err := RemoveKeywords(nil)
			return err
		}, want: ErrMissingCommand},
		{name: "remove missing input", run: func() error {
			_, err := RemoveKeywords(&Command{OutFile: &outFile})
			return err
		}, want: api.ErrMissingPDFInput},
		{name: "remove empty input", run: func() error {
			_, err := RemoveKeywords(&Command{InFile: &empty, OutFile: &outFile})
			return err
		}, want: api.ErrMissingPDFInput},
		{name: "remove missing output", run: func() error {
			_, err := RemoveKeywords(&Command{InFile: &inFile})
			return err
		}, want: api.ErrMissingPDFOutput},
		{name: "remove empty output", run: func() error {
			_, err := RemoveKeywords(&Command{InFile: &inFile, OutFile: &empty})
			return err
		}, want: api.ErrMissingPDFOutput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

// TestKeywordCommandConstructorsDefaultOutput verifies optional output becomes an explicit in-place target.
func TestKeywordCommandConstructorsDefaultOutput(t *testing.T) {
	inFile := "in.pdf"
	tests := []struct {
		name string
		cmd  *Command
	}{
		{name: "add", cmd: AddKeywordsCommand(inFile, "", nil, nil)},
		{name: "remove", cmd: RemoveKeywordsCommand(inFile, "", nil, nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.OutFile == nil || *tt.cmd.OutFile != inFile {
				t.Fatalf("expected in-place output %q, got %v", inFile, tt.cmd.OutFile)
			}
		})
	}
}

// TestKeywordCommandConstructorsOwnKeywordSlices verifies commands do not retain caller-owned slices.
func TestKeywordCommandConstructorsOwnKeywordSlices(t *testing.T) {
	tests := []struct {
		name string
		new  func([]string) *Command
	}{
		{name: "add", new: func(keywords []string) *Command {
			return AddKeywordsCommand("in.pdf", "out.pdf", keywords, nil)
		}},
		{name: "remove", new: func(keywords []string) *Command {
			return RemoveKeywordsCommand("in.pdf", "out.pdf", keywords, nil)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keywords := []string{"alpha", "beta"}
			cmd := tt.new(keywords)
			keywords[0] = "changed"
			keywords = append(keywords, "gamma")

			want := []string{"alpha", "beta"}
			if !slices.Equal(cmd.StringVals, want) {
				t.Fatalf("command keywords changed: got %v, want %v", cmd.StringVals, want)
			}
		})
	}
}

// TestDispatchKeywordsRejectsInvalidState verifies direct keyword dispatch rejects malformed state.
func TestDispatchKeywordsRejectsInvalidState(t *testing.T) {
	if _, err := dispatchKeywords(nil); !errors.Is(err, ErrMissingCommand) ||
		!strings.Contains(err.Error(), "dispatch keywords") {
		t.Fatalf("expected missing-command context, got %v", err)
	}

	mode := model.VALIDATE
	_, err := dispatchKeywords(&Command{Mode: mode})
	want := fmt.Sprintf("mode %d", mode)
	if !errors.Is(err, ErrUnsupportedCommandMode) || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q context, got %v", want, err)
	}
}

// TestDispatchRejectsInvalidState verifies public dispatch preserves command-state sentinels.
func TestDispatchRejectsInvalidState(t *testing.T) {
	if _, err := Dispatch(nil); !errors.Is(err, ErrMissingCommand) ||
		!strings.Contains(err.Error(), "pdfcpu: dispatch") {
		t.Fatalf("expected missing-command context, got %v", err)
	}

	mode := model.CommandMode(-1)
	_, err := Dispatch(&Command{Mode: mode})
	want := fmt.Sprintf("mode %d", mode)
	if !errors.Is(err, ErrUnsupportedCommandMode) || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q context, got %v", want, err)
	}
}

// TestKeywordCLIMalformedPDFErrorsIncludeReadPhase verifies add/remove retain operation and read context.
func TestKeywordCLIMalformedPDFErrorsIncludeReadPhase(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "malformed.pdf")
	if err := os.WriteFile(inFile, []byte("not a PDF"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		run  func(string) error
		op   string
	}{
		{name: "add", run: func(outFile string) error {
			_, err := Dispatch(AddKeywordsCommand(inFile, outFile, []string{"keyword"}, nil))
			return err
		}, op: "add keywords"},
		{name: "remove", run: func(outFile string) error {
			_, err := Dispatch(RemoveKeywordsCommand(inFile, outFile, []string{"keyword"}, nil))
			return err
		}, op: "remove keywords"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			err := tt.run(outFile)
			if !errors.Is(err, pdfcpu.ErrCorruptHeader) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrCorruptHeader, err)
			}
			for _, want := range []string{tt.op, "read context"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected %q context, got %v", want, err)
				}
			}
		})
	}
}

// TestKeywordCLINoMatchPreservesSentinel verifies dispatch preserves the API no-op contract.
func TestKeywordCLINoMatchPreservesSentinel(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	_, err := Dispatch(RemoveKeywordsCommand(
		inFile,
		outFile,
		[]string{"pdfcpu-keyword-that-does-not-exist"},
		nil,
	))
	if !errors.Is(err, api.ErrNoKeywordRemoved) {
		t.Fatalf("expected %v, got %v", api.ErrNoKeywordRemoved, err)
	}
}

// TestKeywordCLIRejectsEmptyKeywordsBeforeFileIO verifies keyword validation precedes input access.
func TestKeywordCLIRejectsEmptyKeywordsBeforeFileIO(t *testing.T) {
	tests := []struct {
		name string
		run  func(string) error
		op   string
	}{
		{name: "add", run: func(outFile string) error {
			_, err := Dispatch(AddKeywordsCommand("-", outFile, []string{" "}, nil))
			return err
		}, op: "add keywords: validate keywords"},
		{name: "remove", run: func(outFile string) error {
			_, err := Dispatch(RemoveKeywordsCommand("-", outFile, []string{" "}, nil))
			return err
		}, op: "remove keywords: validate keywords"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := errors.New("temporary input creation should not run")
			create := createTemporaryInputFile
			createTemporaryInputFile = func(string, string) (*os.File, error) {
				return nil, wantErr
			}
			t.Cleanup(func() {
				createTemporaryInputFile = create
			})

			outFile := filepath.Join(t.TempDir(), "out.pdf")
			err := tt.run(outFile)
			if err == nil || !strings.Contains(err.Error(), tt.op) {
				t.Fatalf("expected %q context, got %v", tt.op, err)
			}
			if errors.Is(err, wantErr) {
				t.Fatalf("temporary input was created before keyword validation: %v", err)
			}
			if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("unexpected output file: %v", statErr)
			}
		})
	}
}

func setClosedKeywordStdin(t *testing.T) {
	t.Helper()

	stdin := os.Stdin
	f, err := os.CreateTemp(t.TempDir(), "closed-stdin-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = stdin
	})
}

func runKeywordStdinCommand(add bool, outFile string) error {
	if add {
		_, err := Dispatch(AddKeywordsCommand("-", outFile, []string{"keyword"}, nil))
		return err
	}
	_, err := Dispatch(RemoveKeywordsCommand("-", outFile, []string{"keyword"}, nil))
	return err
}

// TestKeywordCLIStdinTemporaryFileFailuresRetainOperationContext verifies setup failures identify add/remove.
func TestKeywordCLIStdinTemporaryFileFailuresRetainOperationContext(t *testing.T) {
	tests := []struct {
		name string
		add  bool
		op   string
	}{
		{name: "add", add: true, op: "add keywords"},
		{name: "remove", op: "remove keywords"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := errors.New("create temporary input failed")
			create := createTemporaryInputFile
			createTemporaryInputFile = func(string, string) (*os.File, error) {
				return nil, wantErr
			}
			t.Cleanup(func() {
				createTemporaryInputFile = create
			})

			outFile := filepath.Join(t.TempDir(), "out.pdf")
			err := runKeywordStdinCommand(tt.add, outFile)
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.op+": create temporary input") {
				t.Fatalf("expected %q setup context, got %v", tt.op, err)
			}
		})
	}
}

// TestKeywordCLIStdinReadFailuresRetainOperationContext verifies stdin read failures identify add/remove.
func TestKeywordCLIStdinReadFailuresRetainOperationContext(t *testing.T) {
	tests := []struct {
		name string
		add  bool
		op   string
	}{
		{name: "add", add: true, op: "add keywords"},
		{name: "remove", op: "remove keywords"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setClosedKeywordStdin(t)
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			err := runKeywordStdinCommand(tt.add, outFile)
			if !errors.Is(err, os.ErrClosed) {
				t.Fatalf("expected %v, got %v", os.ErrClosed, err)
			}
			if !strings.Contains(err.Error(), tt.op+": read stdin") {
				t.Fatalf("expected %q read context, got %v", tt.op, err)
			}
		})
	}
}

// TestKeywordCLIStdinRewindFailuresRetainOperationContext verifies temporary-input failures identify add/remove.
func TestKeywordCLIStdinRewindFailuresRetainOperationContext(t *testing.T) {
	tests := []struct {
		name string
		add  bool
		op   string
	}{
		{name: "add", add: true, op: "add keywords"},
		{name: "remove", op: "remove keywords"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useStdin(t, "%PDF-1.7\n")
			wantErr := errors.New("rewind failed")
			rewind := rewindTemporaryInputFile
			rewindTemporaryInputFile = func(*os.File) error { return wantErr }
			t.Cleanup(func() {
				rewindTemporaryInputFile = rewind
			})

			outFile := filepath.Join(t.TempDir(), "out.pdf")
			err := runKeywordStdinCommand(tt.add, outFile)
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.op+": rewind temporary input") {
				t.Fatalf("expected %q rewind context, got %v", tt.op, err)
			}
		})
	}
}

// TestListKeywordsFileOpenError verifies input-open failures preserve their cause and filename.
func TestListKeywordsFileOpenError(t *testing.T) {
	if _, err := ListKeywordsFile("", nil); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}

	inFile := filepath.Join(t.TempDir(), "missing.pdf")
	_, err := ListKeywordsFile(inFile, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	want := "list keywords: open input " + inFile
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q context, got %v", want, err)
	}
}

// TestListKeywordsFileJoinsCloseError verifies operation and close failures remain discoverable.
func TestListKeywordsFileJoinsCloseError(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "empty.pdf")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("close failed")
	closeInput := closeListKeywordsInput
	closeListKeywordsInput = func(f *os.File) error {
		if err := closeInput(f); err != nil {
			t.Fatal(err)
		}
		return wantErr
	}
	t.Cleanup(func() {
		closeListKeywordsInput = closeInput
	})

	_, err := ListKeywordsFile(inFile, nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	want := "list keywords: close input " + inFile
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q context, got %v", want, err)
	}
}

// TestKeywordStreamingOpenErrorsUseExactOperation verifies operation-aware stream setup context.
func TestKeywordStreamingOpenErrorsUseExactOperation(t *testing.T) {
	outFile := "-"
	tests := []struct {
		name string
		run  func(*Command) error
		op   string
	}{
		{name: "add", run: func(cmd *Command) error {
			_, err := AddKeywords(cmd)
			return err
		}, op: "add keywords"},
		{name: "remove", run: func(cmd *Command) error {
			_, err := RemoveKeywords(cmd)
			return err
		}, op: "remove keywords"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inFile := filepath.Join(t.TempDir(), "missing.pdf")
			err := tt.run(&Command{InFile: &inFile, OutFile: &outFile})
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			want := tt.op + ": open input " + inFile
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q context, got %v", want, err)
			}
		})
	}
}

// TestRunKeywordStreamOperationPreservesOutput verifies finalization removes failed temporary output.
func TestRunKeywordStreamOperationPreservesOutput(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	wantOutput := []byte("existing output")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outFile, wantOutput, 0o600); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("keyword operation failed")

	err := runKeywordStreamOperation(inFile, outFile, "add keywords", func(io.ReadSeeker, io.Writer) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, wantOutput) {
		t.Fatalf("existing output changed: got %q, want %q", got, wantOutput)
	}
}

// TestKeywordStreamingFailurePreservesExistingOutput verifies protected output survives read failure.
func TestKeywordStreamingFailurePreservesExistingOutput(t *testing.T) {
	useStdin(t, "not a PDF")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := AddKeywords(AddKeywordsCommand("-", outFile, []string{"keyword"}, nil))
	if err == nil {
		t.Fatal("expected read failure")
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

// TestKeywordStreamingFailureRemovesNewOutput verifies failed streaming removes a new destination.
func TestKeywordStreamingFailureRemovesNewOutput(t *testing.T) {
	useStdin(t, "not a PDF")
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	_, err := RemoveKeywords(RemoveKeywordsCommand("-", outFile, nil, nil))
	if err == nil {
		t.Fatal("expected read failure")
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected failed output cleanup, got %v", statErr)
	}
}

// TestKeywordStreamingSuccessReplacesExistingOutput verifies successful delayed replacement.
func TestKeywordStreamingSuccessReplacesExistingOutput(t *testing.T) {
	bb, err := os.ReadFile(filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	useStdinBytes(t, bb)
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("existing output"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := AddKeywords(AddKeywordsCommand("-", outFile, []string{"keyword"}, nil)); err != nil {
		t.Fatal(err)
	}
	keywords, err := ListKeywordsFile(outFile, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(keywords, "keyword") {
		t.Fatalf("replacement output is missing keyword: %v", keywords)
	}
}

// TestKeywordFinalizerCloseFailurePreservesCause verifies output-close failures retain context and causes.
func TestKeywordFinalizerCloseFailurePreservesCause(t *testing.T) {
	output, err := os.CreateTemp(t.TempDir(), "keyword-output-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	outFile := output.Name()
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("keyword operation failed")
	finalizer := &streamInOutFinalizer{output: output, outFile: outFile}

	err = finalizer.finalize("add keywords", wantErr)
	if !errors.Is(err, wantErr) || !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected operation and close causes, got %v", err)
	}
	if !strings.Contains(err.Error(), "add keywords: close output") {
		t.Fatalf("expected close-output context, got %v", err)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected failed output cleanup, got %v", statErr)
	}
}

// TestKeywordFinalizerRemoveFailurePreservesCause verifies output-removal failures retain context and causes.
func TestKeywordFinalizerRemoveFailurePreservesCause(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "output")
	if err := os.Mkdir(outFile, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outFile, "keep"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	probeErr := os.Remove(outFile)
	if probeErr == nil {
		t.Fatal("expected non-empty directory removal failure")
	}
	removeCause := errors.Unwrap(probeErr)
	wantErr := errors.New("keyword operation failed")
	finalizer := &streamInOutFinalizer{outFile: outFile}

	err := finalizer.finalize("remove keywords", wantErr)
	if !errors.Is(err, wantErr) || !errors.Is(err, removeCause) {
		t.Fatalf("expected operation and remove causes, got %v", err)
	}
	if !strings.Contains(err.Error(), "remove keywords: remove output") {
		t.Fatalf("expected remove-output context, got %v", err)
	}
}

// TestKeywordFinalizerRenameFailurePreservesCause verifies replacement failures retain context and causes.
func TestKeywordFinalizerRenameFailurePreservesCause(t *testing.T) {
	output, err := os.CreateTemp(t.TempDir(), "keyword-output-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	outFile := output.Name()
	replaceOut := filepath.Join(t.TempDir(), "output")
	if err := os.Mkdir(replaceOut, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(replaceOut, "keep"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	finalizer := &streamInOutFinalizer{
		output:     output,
		outFile:    outFile,
		replaceOut: replaceOut,
	}

	err = finalizer.finalize("add keywords", nil)
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) || !errors.Is(err, linkErr.Err) {
		t.Fatalf("expected rename cause, got %v", err)
	}
	if !strings.Contains(err.Error(), "add keywords: replace output") {
		t.Fatalf("expected replace-output context, got %v", err)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected temporary output cleanup, got %v", statErr)
	}
}
