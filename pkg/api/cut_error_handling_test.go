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
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func cutErrorTestInput() string {
	return filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
}

func validCutConfiguration() *model.Cut {
	return &model.Cut{Hor: []float64{0.5}}
}

func validNDownConfiguration() *model.Cut {
	return &model.Cut{}
}

func validPosterConfiguration() *model.Cut {
	return &model.Cut{Scale: 1, PageDim: &types.Dim{Width: 100, Height: 100}, UserDim: true}
}

func snapshotCutConfiguration(cut *model.Cut) model.Cut {
	snapshot := *cut
	snapshot.Hor = append([]float64(nil), cut.Hor...)
	snapshot.Vert = append([]float64(nil), cut.Vert...)
	if cut.PageDim != nil {
		dim := *cut.PageDim
		snapshot.PageDim = &dim
	}
	if cut.BgColor != nil {
		bgColor := *cut.BgColor
		snapshot.BgColor = &bgColor
	}
	return snapshot
}

func assertCutConfigurationUnchanged(t *testing.T, cut *model.Cut, want model.Cut) {
	t.Helper()
	if !reflect.DeepEqual(*cut, want) {
		t.Fatalf("cut configuration changed:\n got: %#v\nwant: %#v", *cut, want)
	}
}

type cutOutputTestFile struct {
	bytes.Buffer
	name     string
	closeErr error
	closed   bool
	mode     os.FileMode
}

func (f *cutOutputTestFile) Close() error {
	f.closed = true
	return f.closeErr
}

func (f *cutOutputTestFile) Name() string {
	return f.name
}

func (f *cutOutputTestFile) Chmod(mode os.FileMode) error {
	f.mode = mode
	return nil
}

type cutOutputOSFile struct {
	*os.File
	closeErr error
}

func (f *cutOutputOSFile) Close() error {
	return errors.Join(f.File.Close(), f.closeErr)
}

func cutMutationTestConfiguration(operation string) *model.Cut {
	bgColor := color.SimpleColor{R: 0.1, G: 0.2, B: 0.3}
	cut := &model.Cut{
		Hor:     []float64{0.75, 0.25},
		Vert:    []float64{0.8, 0.2},
		PageDim: &types.Dim{Width: 100, Height: 100},
		BgColor: &bgColor,
	}
	if operation == "poster" {
		cut.Scale = 1
		cut.UserDim = true
	}
	return cut
}

// TestCloneCutConfigurationDeepCopiesOwnedData verifies deep ownership at the API boundary.
func TestCloneCutConfigurationDeepCopiesOwnedData(t *testing.T) {
	cut := cutMutationTestConfiguration("poster")
	want := snapshotCutConfiguration(cut)
	clone := cloneCutConfiguration(cut)
	clone.Hor[0] = 0.1
	clone.Vert[0] = 0.1
	clone.PageDim.Width = 200
	clone.BgColor.R = 0.9
	assertCutConfigurationUnchanged(t, cut, want)
}

// TestSelectedCutPagesAreSorted verifies deterministic processing order for all cut operations.
func TestSelectedCutPagesAreSorted(t *testing.T) {
	got, err := selectedCutPages(5, []string{"5", "1", "3"}, "cut")
	if err != nil {
		t.Fatal(err)
	}
	want := []int{1, 3, 5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selected pages: got %v, want %v", got, want)
	}
}

// TestCutAPIRejectsDuplicateCutPoints verifies duplicate detection on cloned, sorted cut points.
func TestCutAPIRejectsDuplicateCutPoints(t *testing.T) {
	tests := []struct {
		name string
		cut  *model.Cut
		axis string
	}{
		{name: "horizontal", cut: &model.Cut{Hor: []float64{0.5, 0.5}}, axis: "horizontal"},
		{name: "vertical", cut: &model.Cut{Vert: []float64{0.25, 0.25}}, axis: "vertical"},
		{name: "repeated zero", cut: &model.Cut{Hor: []float64{0, 0}}, axis: "horizontal"},
		{name: "unsorted duplicates", cut: &model.Cut{Vert: []float64{0.75, 0.25, 0.75}}, axis: "vertical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := snapshotCutConfiguration(tt.cut)
			err := Cut(bytes.NewReader(nil), "", "", nil, tt.cut, nil)
			if !errors.Is(err, ErrInvalidCutConfiguration) {
				t.Fatalf("expected %v, got %v", ErrInvalidCutConfiguration, err)
			}
			if !strings.Contains(err.Error(), "duplicate "+tt.axis+" cut point") {
				t.Fatalf("expected duplicate %s context, got %q", tt.axis, err.Error())
			}
			assertCutConfigurationUnchanged(t, tt.cut, want)
		})
	}
}

// TestCutAPIsPreserveCallerConfiguration verifies successful and failed calls never mutate caller-owned data.
func TestCutAPIsPreserveCallerConfiguration(t *testing.T) {
	type runCutAPI func(*testing.T, string, *model.Cut) error
	operations := []struct {
		name   string
		reader runCutAPI
		file   runCutAPI
	}{
		{name: "cut", reader: func(t *testing.T, outDir string, cut *model.Cut) error {
			return Cut(openAPITestPDF(t, cutErrorTestInput()), outDir, "out", []string{"1"}, cut, nil)
		}, file: func(_ *testing.T, outDir string, cut *model.Cut) error {
			return CutFile(cutErrorTestInput(), outDir, "out", []string{"1"}, cut, nil)
		}},
		{name: "ndown", reader: func(t *testing.T, outDir string, cut *model.Cut) error {
			return NDown(openAPITestPDF(t, cutErrorTestInput()), outDir, "out", []string{"1"}, 2, cut, nil)
		}, file: func(_ *testing.T, outDir string, cut *model.Cut) error {
			return NDownFile(cutErrorTestInput(), outDir, "out", []string{"1"}, 2, cut, nil)
		}},
		{name: "poster", reader: func(t *testing.T, outDir string, cut *model.Cut) error {
			return Poster(openAPITestPDF(t, cutErrorTestInput()), outDir, "out", []string{"1"}, cut, nil)
		}, file: func(_ *testing.T, outDir string, cut *model.Cut) error {
			return PosterFile(cutErrorTestInput(), outDir, "out", []string{"1"}, cut, nil)
		}},
	}

	for _, operation := range operations {
		for _, boundary := range []struct {
			name string
			run  runCutAPI
		}{{name: "reader", run: operation.reader}, {name: "file", run: operation.file}} {
			t.Run(operation.name+"/"+boundary.name, func(t *testing.T) {
				cut := cutMutationTestConfiguration(operation.name)
				want := snapshotCutConfiguration(cut)
				if err := boundary.run(t, t.TempDir(), cut); err != nil {
					t.Fatal(err)
				}
				assertCutConfigurationUnchanged(t, cut, want)

				cut = cutMutationTestConfiguration(operation.name)
				want = snapshotCutConfiguration(cut)
				missingDir := filepath.Join(t.TempDir(), "missing")
				if err := boundary.run(t, missingDir, cut); err == nil {
					t.Fatal("expected failure")
				}
				assertCutConfigurationUnchanged(t, cut, want)
			})
		}
	}
}

// TestWriteCutOutputJoinsPhaseAndCleanupFailures verifies protected-output failure aggregation.
func TestWriteCutOutputJoinsPhaseAndCleanupFailures(t *testing.T) {
	writeErr := errors.New("write failure")
	flushErr := errors.New("flush failure")
	closeErr := errors.New("close failure")
	cleanupErr := errors.New("cleanup failure")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	tmpFile := filepath.Join(filepath.Dir(outFile), ".out.pdf.tmp-test")
	f := &cutOutputTestFile{name: tmpFile, closeErr: closeErr}
	renameCalled := false
	removeCalled := false
	ops := cutOutputOperations{
		createTemp: func(dir, pattern string) (cutOutputFile, error) {
			if dir != filepath.Dir(outFile) || !strings.HasPrefix(pattern, ".out.pdf.tmp-") {
				t.Fatalf("unexpected temporary output location: dir=%q pattern=%q", dir, pattern)
			}
			return f, nil
		},
		writeAndFlush: func(*model.Context, io.Writer) (error, error) {
			return writeErr, flushErr
		},
		rename: func(string, string) error {
			renameCalled = true
			return nil
		},
		remove: func(name string) error {
			removeCalled = true
			if name != tmpFile {
				t.Fatalf("removed %q, want %q", name, tmpFile)
			}
			return cleanupErr
		},
	}

	err := writeCutOutputWith(nil, outFile, "cut", ops)
	for _, want := range []error{writeErr, flushErr, closeErr, cleanupErr} {
		if !errors.Is(err, want) {
			t.Fatalf("expected joined error %v, got %v", want, err)
		}
	}
	if renameCalled {
		t.Fatal("rename called after unsuccessful write")
	}
	if !removeCalled {
		t.Fatal("temporary output was not removed")
	}
}

// TestWriteCutOutputCleansUpAfterRenameFailure verifies rename and cleanup errors remain visible.
func TestWriteCutOutputCleansUpAfterRenameFailure(t *testing.T) {
	renameErr := errors.New("rename failure")
	cleanupErr := errors.New("cleanup failure")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	f := &cutOutputTestFile{name: filepath.Join(filepath.Dir(outFile), ".out.pdf.tmp-test")}
	removeCalled := false
	ops := cutOutputOperations{
		createTemp: func(string, string) (cutOutputFile, error) { return f, nil },
		writeAndFlush: func(*model.Context, io.Writer) (error, error) {
			return nil, nil
		},
		rename: func(oldName, newName string) error {
			if oldName != f.Name() || newName != outFile {
				t.Fatalf("rename (%q, %q), want (%q, %q)", oldName, newName, f.Name(), outFile)
			}
			if !f.closed {
				t.Fatal("rename called before close")
			}
			return renameErr
		},
		remove: func(string) error {
			removeCalled = true
			return cleanupErr
		},
	}

	err := writeCutOutputWith(nil, outFile, "cut", ops)
	for _, want := range []error{renameErr, cleanupErr} {
		if !errors.Is(err, want) {
			t.Fatalf("expected joined error %v, got %v", want, err)
		}
	}
	if !removeCalled {
		t.Fatal("temporary output was not removed")
	}
}

// TestWriteCutOutputRecoversWritePanicBeforeCleanup verifies panic recovery and cleanup error joining.
func TestWriteCutOutputRecoversWritePanicBeforeCleanup(t *testing.T) {
	panicErr := errors.New("write panic")
	closeErr := errors.New("close failure")
	cleanupErr := errors.New("cleanup failure")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	f := &cutOutputTestFile{name: filepath.Join(filepath.Dir(outFile), ".out.pdf.tmp-test"), closeErr: closeErr}
	removeCalled := false
	renameCalled := false
	ops := cutOutputOperations{
		createTemp: func(string, string) (cutOutputFile, error) { return f, nil },
		writeAndFlush: func(*model.Context, io.Writer) (error, error) {
			fault.Fail("write output: %w", panicErr)
			return nil, nil
		},
		rename: func(string, string) error {
			renameCalled = true
			return nil
		},
		remove: func(string) error {
			removeCalled = true
			return cleanupErr
		},
	}

	err := writeCutOutputWith(nil, outFile, "cut", ops)
	if err == nil {
		t.Fatal("expected recovered panic error")
	}
	for _, want := range []error{panicErr, closeErr, cleanupErr} {
		if !errors.Is(err, want) {
			t.Fatalf("expected joined error %v, got %v", want, err)
		}
	}
	var recovered fault.Panic
	if !errors.As(err, &recovered) {
		t.Fatalf("expected recovered fault panic, got %T", err)
	}
	if !f.closed || !removeCalled {
		t.Fatalf("cleanup incomplete: closed=%t removed=%t", f.closed, removeCalled)
	}
	if renameCalled {
		t.Fatal("rename called after write panic")
	}
}

// TestWriteCutOutputAppliesDestinationPermissions verifies replacement and new-output modes.
func TestWriteCutOutputAppliesDestinationPermissions(t *testing.T) {
	tests := []struct {
		name     string
		existing bool
	}{
		{name: "existing destination", existing: true},
		{name: "new destination"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			outFile := filepath.Join(dir, "out.pdf")
			var wantMode os.FileMode
			if tt.existing {
				if err := os.WriteFile(outFile, []byte("existing"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(outFile, 0640); err != nil {
					t.Fatal(err)
				}
				wantMode = 0640
			} else {
				reference := filepath.Join(dir, "reference.pdf")
				f, err := os.OpenFile(reference, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
				if err != nil {
					t.Fatal(err)
				}
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
				info, err := os.Stat(reference)
				if err != nil {
					t.Fatal(err)
				}
				wantMode = info.Mode().Perm()
			}

			statCalled := false
			ops := defaultCutOutputOperations()
			ops.stat = func(name string) (os.FileInfo, error) {
				statCalled = true
				return os.Stat(name)
			}
			ops.createTemp = func(dir, pattern string) (cutOutputFile, error) {
				if !statCalled {
					t.Fatal("temporary output created before destination stat")
				}
				return createCutTemporaryOutput(dir, pattern)
			}
			ops.writeAndFlush = func(_ *model.Context, w io.Writer) (error, error) {
				_, err := w.Write([]byte("replacement"))
				return err, nil
			}
			if err := writeCutOutputWith(nil, outFile, "cut", ops); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(outFile)
			if err != nil {
				t.Fatal(err)
			}
			if runtime.GOOS != "windows" {
				if got := info.Mode().Perm(); got != wantMode {
					t.Fatalf("output permissions: got %04o, want %04o", got, wantMode)
				}
			}
		})
	}
}

// TestWriteCutOutputSuccessfullyReplacesDestination verifies a committed replacement and cleanup.
func TestWriteCutOutputSuccessfullyReplacesDestination(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(outFile, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(outFile, 0640); err != nil {
		t.Fatal(err)
	}
	replacement := []byte("replacement")
	ops := defaultCutOutputOperations()
	ops.writeAndFlush = func(_ *model.Context, w io.Writer) (error, error) {
		_, err := w.Write(replacement)
		return err, nil
	}
	if err := writeCutOutputWith(nil, outFile, "cut", ops); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, replacement) {
		t.Fatalf("destination content: got %q, want %q", got, replacement)
	}
	info, err := os.Stat(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if got, want := info.Mode().Perm(), os.FileMode(0640); got != want {
			t.Fatalf("destination permissions: got %04o, want %04o", got, want)
		}
	}
	tmpFiles, err := filepath.Glob(filepath.Join(dir, ".out.pdf.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tmpFiles) != 0 {
		t.Fatalf("temporary outputs remain: %v", tmpFiles)
	}
}

// TestWriteCutOutputFailurePhases verifies focused lifecycle failures and phase ownership.
func TestWriteCutOutputFailurePhases(t *testing.T) {
	tests := []struct {
		name       string
		createErr  error
		writeErr   error
		flushErr   error
		closeErr   error
		renameErr  error
		removeErr  error
		wantPhase  string
		wantRemove bool
		wantRename bool
	}{
		{name: "output creation", createErr: errors.New("create failure"), wantPhase: "create temporary output"},
		{name: "write", writeErr: errors.New("write failure"), wantPhase: "write", wantRemove: true},
		{name: "flush", flushErr: errors.New("flush failure"), wantPhase: "flush", wantRemove: true},
		{name: "close", closeErr: errors.New("close failure"), wantPhase: "close", wantRemove: true},
		{name: "cleanup", writeErr: errors.New("write failure"), removeErr: errors.New("cleanup failure"), wantPhase: "remove temporary output", wantRemove: true},
		{name: "rename", renameErr: errors.New("rename failure"), wantPhase: "rename temporary output", wantRemove: true, wantRename: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			f := &cutOutputTestFile{name: filepath.Join(filepath.Dir(outFile), ".out.pdf.tmp-test"), closeErr: tt.closeErr}
			removeCalled := false
			renameCalled := false
			ops := cutOutputOperations{
				createTemp: func(string, string) (cutOutputFile, error) {
					if tt.createErr != nil {
						return nil, tt.createErr
					}
					return f, nil
				},
				writeAndFlush: func(*model.Context, io.Writer) (error, error) {
					return tt.writeErr, tt.flushErr
				},
				rename: func(string, string) error {
					renameCalled = true
					return tt.renameErr
				},
				remove: func(string) error {
					removeCalled = true
					return tt.removeErr
				},
			}

			err := writeCutOutputWith(nil, outFile, "cut", ops)
			if err == nil || !strings.Contains(err.Error(), "cut: write output "+outFile+": "+tt.wantPhase) {
				t.Fatalf("expected %q phase, got %v", tt.wantPhase, err)
			}
			if removeCalled != tt.wantRemove {
				t.Fatalf("remove called=%t, want %t", removeCalled, tt.wantRemove)
			}
			if renameCalled != tt.wantRename {
				t.Fatalf("rename called=%t, want %t", renameCalled, tt.wantRename)
			}
		})
	}
}

// TestWriteCutOutputPreservesExistingDestination verifies failed replacement never damages prior output.
func TestWriteCutOutputPreservesExistingDestination(t *testing.T) {
	primaryErr := errors.New("output failure")
	tests := []struct {
		name          string
		writeErr      error
		closeErr      error
		renameFailure bool
	}{
		{name: "write", writeErr: primaryErr},
		{name: "close", closeErr: primaryErr},
		{name: "rename", renameFailure: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			outFile := filepath.Join(dir, "out.pdf")
			original := []byte("original output")
			if err := os.WriteFile(outFile, original, 0600); err != nil {
				t.Fatal(err)
			}
			ops := defaultCutOutputOperations()
			ops.createTemp = func(dir, pattern string) (cutOutputFile, error) {
				f, err := os.CreateTemp(dir, pattern)
				if err != nil {
					return nil, err
				}
				return &cutOutputOSFile{File: f, closeErr: tt.closeErr}, nil
			}
			ops.writeAndFlush = func(_ *model.Context, w io.Writer) (error, error) {
				if _, err := w.Write([]byte("replacement output")); err != nil {
					return err, nil
				}
				return tt.writeErr, nil
			}
			if tt.renameFailure {
				ops.rename = func(string, string) error { return primaryErr }
			}

			err := writeCutOutputWith(nil, outFile, "cut", ops)
			if !errors.Is(err, primaryErr) {
				t.Fatalf("expected %v, got %v", primaryErr, err)
			}
			got, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, original) {
				t.Fatalf("destination changed: got %q, want %q", got, original)
			}
			tmpFiles, err := filepath.Glob(filepath.Join(dir, ".out.pdf.tmp-*"))
			if err != nil {
				t.Fatal(err)
			}
			if len(tmpFiles) != 0 {
				t.Fatalf("temporary output remains: %v", tmpFiles)
			}
		})
	}
}

// TestCutAPIMissingArguments verifies public slice boundary guards.
func TestCutAPIMissingArguments(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "cut reader", err: Cut(nil, "", "", nil, validCutConfiguration(), nil), want: ErrMissingPDFReadSeeker},
		{name: "ndown reader", err: NDown(nil, "", "", nil, 2, validNDownConfiguration(), nil), want: ErrMissingPDFReadSeeker},
		{name: "poster reader", err: Poster(nil, "", "", nil, validPosterConfiguration(), nil), want: ErrMissingPDFReadSeeker},
		{name: "cut configuration", err: Cut(bytes.NewReader(nil), "", "", nil, nil, nil), want: ErrMissingCutConfiguration},
		{name: "ndown configuration", err: NDown(bytes.NewReader(nil), "", "", nil, 2, nil, nil), want: ErrMissingCutConfiguration},
		{name: "poster configuration", err: Poster(bytes.NewReader(nil), "", "", nil, nil, nil), want: ErrMissingCutConfiguration},
		{name: "cut input", err: CutFile("", "", "", nil, nil, nil), want: ErrMissingPDFInput},
		{name: "ndown input", err: NDownFile("", "", "", nil, 2, nil, nil), want: ErrMissingPDFInput},
		{name: "poster input", err: PosterFile("", "", "", nil, nil, nil), want: ErrMissingPDFInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, tt.err)
			}
		})
	}
}

// TestCutAPIRejectsInvalidConfigurationsBeforeIO verifies direct configuration validation order.
func TestCutAPIRejectsInvalidConfigurationsBeforeIO(t *testing.T) {
	missingFile := filepath.Join(t.TempDir(), "missing.pdf")
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "cut points missing", err: Cut(bytes.NewReader(nil), "", "", nil, &model.Cut{}, nil), want: "missing horizontal or vertical cut points"},
		{name: "cut point NaN", err: Cut(bytes.NewReader(nil), "", "", nil, &model.Cut{Hor: []float64{math.NaN()}}, nil), want: "horizontal cut point 1"},
		{name: "cut point infinity", err: Cut(bytes.NewReader(nil), "", "", nil, &model.Cut{Vert: []float64{math.Inf(1)}}, nil), want: "vertical cut point 1"},
		{name: "cut margin", err: Cut(bytes.NewReader(nil), "", "", nil, &model.Cut{Hor: []float64{0.5}, Margin: -1}, nil), want: "margin"},
		{name: "ndown value", err: NDown(bytes.NewReader(nil), "", "", nil, 5, validNDownConfiguration(), nil), want: "n-down value 5"},
		{name: "ndown margin", err: NDown(bytes.NewReader(nil), "", "", nil, 2, &model.Cut{Margin: math.Inf(1)}, nil), want: "margin"},
		{name: "poster source missing", err: Poster(bytes.NewReader(nil), "", "", nil, &model.Cut{}, nil), want: "missing dimensions or form size"},
		{name: "poster scale", err: Poster(bytes.NewReader(nil), "", "", nil, &model.Cut{Scale: math.NaN(), PageDim: &types.Dim{Width: 100, Height: 100}, UserDim: true}, nil), want: "scale factor"},
		{name: "poster dimensions missing", err: Poster(bytes.NewReader(nil), "", "", nil, &model.Cut{Scale: 1, UserDim: true}, nil), want: "missing dimensions"},
		{name: "poster dimensions zero", err: Poster(bytes.NewReader(nil), "", "", nil, &model.Cut{Scale: 1, PageDim: &types.Dim{}, UserDim: true}, nil), want: "dimensions must be finite and > 0"},
		{name: "cut file before open", err: CutFile(missingFile, "", "", nil, &model.Cut{}, nil), want: "missing horizontal or vertical cut points"},
		{name: "ndown file before open", err: NDownFile(missingFile, "", "", nil, 5, validNDownConfiguration(), nil), want: "n-down value 5"},
		{name: "poster file before open", err: PosterFile(missingFile, "", "", nil, &model.Cut{}, nil), want: "missing dimensions or form size"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, ErrInvalidCutConfiguration) {
				t.Fatalf("expected %v, got %v", ErrInvalidCutConfiguration, tt.err)
			}
			if !strings.Contains(tt.err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %q", tt.want, tt.err.Error())
			}
			operation := strings.SplitN(tt.name, " ", 2)[0]
			if !strings.Contains(tt.err.Error(), operation+": validate configuration") {
				t.Fatalf("expected %s validation context, got %q", operation, tt.err.Error())
			}
			if errors.Is(tt.err, pdfcpu.ErrEmptyInput) || errors.Is(tt.err, os.ErrNotExist) {
				t.Fatalf("configuration validation must precede PDF I/O: %v", tt.err)
			}
		})
	}
}

// TestCutAPIReadErrorsIncludeOperationContext verifies PDF preparation phases.
func TestCutAPIReadErrorsIncludeOperationContext(t *testing.T) {
	tests := []struct {
		operation string
		run       func() error
	}{
		{operation: "cut", run: func() error { return Cut(bytes.NewReader(nil), "", "", nil, validCutConfiguration(), nil) }},
		{operation: "ndown", run: func() error { return NDown(bytes.NewReader(nil), "", "", nil, 2, validNDownConfiguration(), nil) }},
		{operation: "poster", run: func() error { return Poster(bytes.NewReader(nil), "", "", nil, validPosterConfiguration(), nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			err := tt.run()
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if !strings.Contains(err.Error(), tt.operation+": prepare PDF context") {
				t.Fatalf("expected preparation context, got %q", err.Error())
			}
		})
	}
}

// TestCutAPIPageSelectionErrorsIncludeOperationContext verifies selection phases.
func TestCutAPIPageSelectionErrorsIncludeOperationContext(t *testing.T) {
	tests := []struct {
		operation string
		run       func() error
	}{
		{operation: "cut", run: func() error {
			return Cut(openAPITestPDF(t, cutErrorTestInput()), t.TempDir(), "out", []string{"foo"}, validCutConfiguration(), nil)
		}},
		{operation: "ndown", run: func() error {
			return NDown(openAPITestPDF(t, cutErrorTestInput()), t.TempDir(), "out", []string{"foo"}, 2, validNDownConfiguration(), nil)
		}},
		{operation: "poster", run: func() error {
			return Poster(openAPITestPDF(t, cutErrorTestInput()), t.TempDir(), "out", []string{"foo"}, validPosterConfiguration(), nil)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), tt.operation+": parse page selection") {
				t.Fatalf("expected page-selection context, got %v", err)
			}
		})
	}
}

// TestCutAPIPageAndWriteErrorsIncludeOperationContext verifies per-page phases.
func TestCutAPIPageAndWriteErrorsIncludeOperationContext(t *testing.T) {
	err := Poster(
		openAPITestPDF(t, cutErrorTestInput()),
		t.TempDir(),
		"out",
		[]string{"1"},
		&model.Cut{Scale: 1, PageDim: &types.Dim{Width: 10000, Height: 10000}, UserDim: true},
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "poster: process page 1") {
		t.Fatalf("expected poster page context, got %v", err)
	}

	missingDir := filepath.Join(t.TempDir(), "missing")
	tests := []struct {
		operation string
		run       func() error
	}{
		{operation: "cut", run: func() error {
			return Cut(openAPITestPDF(t, cutErrorTestInput()), missingDir, "out", []string{"1"}, validCutConfiguration(), nil)
		}},
		{operation: "ndown", run: func() error {
			return NDown(openAPITestPDF(t, cutErrorTestInput()), missingDir, "out", []string{"1"}, 2, validNDownConfiguration(), nil)
		}},
		{operation: "poster", run: func() error {
			return Poster(openAPITestPDF(t, cutErrorTestInput()), missingDir, "out", []string{"1"}, validPosterConfiguration(), nil)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			err := tt.run()
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.operation+": write output ") {
				t.Fatalf("expected write context, got %q", err.Error())
			}
		})
	}
}

// TestCutAPIFileErrorsIncludeOperationAndInput verifies file-entry context.
func TestCutAPIFileErrorsIncludeOperationAndInput(t *testing.T) {
	missingFile := filepath.Join(t.TempDir(), "missing.pdf")
	tests := []struct {
		operation string
		run       func() error
	}{
		{operation: "cut", run: func() error { return CutFile(missingFile, "", "", nil, validCutConfiguration(), nil) }},
		{operation: "ndown", run: func() error { return NDownFile(missingFile, "", "", nil, 2, validNDownConfiguration(), nil) }},
		{operation: "poster", run: func() error { return PosterFile(missingFile, "", "", nil, validPosterConfiguration(), nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			err := tt.run()
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.operation+": open input "+missingFile) {
				t.Fatalf("expected input context, got %q", err.Error())
			}
		})
	}
}

// TestCloseCutInputFilePreservesCause verifies input close context and cause preservation.
func TestCloseCutInputFilePreservesCause(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "cut-close-*")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	err = closeFile(f, "cut: close input")
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected %v, got %v", os.ErrClosed, err)
	}
	if !strings.Contains(err.Error(), "cut: close input") {
		t.Fatalf("expected close context, got %q", err.Error())
	}
}
