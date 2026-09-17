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
	"context"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type cancelAfterFileChecksContext struct {
	context.Context
	remaining int
}

func (c *cancelAfterFileChecksContext) Err() error {
	c.remaining--
	if c.remaining <= 0 {
		return context.Canceled
	}
	return nil
}

func progressTestPDF() string {
	return filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
}

func incorrectTrailerSizePDF(t *testing.T) string {
	t.Helper()
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources << >> >>",
	}
	offsets := make([]int, len(objects))
	for i, object := range objects {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xrefOffset := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects), xrefOffset)

	fileName := filepath.Join(t.TempDir(), "incorrect-trailer-size.pdf")
	if err := os.WriteFile(fileName, b.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return fileName
}

func fileCancellationPreservesDestination(
	t *testing.T,
	name string,
	call func(context.Context, context.CancelFunc, string) error,
) {
	t.Helper()
	dir := t.TempDir()
	outFile := filepath.Join(dir, name)
	original := []byte("original destination")
	if err := os.WriteFile(outFile, original, 0o644); err != nil {
		t.Fatal(err)
	}
	c, cancel := context.WithCancel(t.Context())
	err := call(c, cancel, outFile)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("destination changed: got %q, want %q", got, original)
	}
	temporaryFiles, err := filepath.Glob(filepath.Join(dir, "."+name+".tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("temporary output remains: %v", temporaryFiles)
	}
}

func captureStderr(t *testing.T, f func()) string {
	t.Helper()

	stderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.Stderr = stderr
		_ = r.Close()
		_ = w.Close()
	}()
	os.Stderr = w

	f()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestValidateReportsOrderedStages(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.Optimize = true
	var events []ProgressEvent
	options := ProgressOptions{
		Observer: func(event ProgressEvent) error {
			events = append(events, event)
			return nil
		},
		Item:  2,
		Total: 3,
	}

	inFile := progressTestPDF()
	if err := ValidateFile(t.Context(), inFile, conf, &options); err != nil {
		t.Fatal(err)
	}
	if options.Input != "" || options.Item != 2 || options.Total != 3 {
		t.Fatalf("validation modified caller options: %#v", options)
	}

	wantStages := []ProgressStage{
		ProgressStageReading,
		ProgressStageValidating,
		ProgressStageOptimizing,
	}
	if len(events) != len(wantStages) {
		t.Fatalf("got %d events, want %d: %#v", len(events), len(wantStages), events)
	}
	for i, event := range events {
		if event.Stage != wantStages[i] {
			t.Fatalf("event %d stage: got %q, want %q", i, event.Stage, wantStages[i])
		}
		if event.Input != inFile || event.Item != 2 || event.Total != 3 {
			t.Fatalf("event %d data: got %#v", i, event)
		}
	}
}

func TestValidateFilesAddsInputPosition(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.Optimize = false
	var events []ProgressEvent
	options := ProgressOptions{Observer: func(event ProgressEvent) error {
		if event.Stage == ProgressStageReading {
			events = append(events, event)
		}
		return nil
	}}
	inFile := progressTestPDF()

	if err := ValidateFiles(t.Context(), []string{inFile, inFile}, conf, &options); err != nil {
		t.Fatal(err)
	}
	if options.Input != "" || options.Item != 0 || options.Total != 0 {
		t.Fatalf("validation modified caller options: %#v", options)
	}

	want := []ProgressEvent{
		{Stage: ProgressStageReading, Input: inFile, Item: 1, Total: 2},
		{Stage: ProgressStageReading, Input: inFile, Item: 2, Total: 2},
	}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("got %#v, want %#v", events, want)
	}
}

func TestValidateNilOptionsIsSilent(t *testing.T) {
	var cliOutput bytes.Buffer
	log.SetCLILogger(stdlog.New(&cliOutput, "", 0))
	t.Cleanup(func() { log.SetCLILogger(nil) })

	stderr := captureStderr(t, func() {
		if err := ValidateFile(t.Context(), incorrectTrailerSizePDF(t), nil, nil); err != nil {
			t.Fatal(err)
		}
	})

	if cliOutput.Len() != 0 {
		t.Fatalf("validation wrote CLI output: %q", cliOutput.String())
	}
	if stderr != "" {
		t.Fatalf("validation wrote stderr: %q", stderr)
	}
}

func TestValidatePreservesObserverFailure(t *testing.T) {
	wantErr := errors.New("observer failed")
	var stages []ProgressStage
	options := ProgressOptions{Observer: func(event ProgressEvent) error {
		stages = append(stages, event.Stage)
		if event.Stage == ProgressStageValidating {
			return wantErr
		}
		return nil
	}}

	err := ValidateFile(t.Context(), progressTestPDF(), nil, &options)
	if !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want cause %v", err, wantErr)
	}
	var progressErr *ProgressError
	if !errors.As(err, &progressErr) {
		t.Fatalf("got %T, want *ProgressError", err)
	}
	if progressErr.Event.Stage != ProgressStageValidating {
		t.Fatalf("got stage %q, want %q", progressErr.Event.Stage, ProgressStageValidating)
	}
	wantStages := []ProgressStage{ProgressStageReading, ProgressStageValidating}
	if !reflect.DeepEqual(stages, wantStages) {
		t.Fatalf("got stages %#v, want %#v", stages, wantStages)
	}
}

func TestOptimizeFileObserverFailurePreservesDestination(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.pdf")
	original := []byte("original destination")
	if err := os.WriteFile(outFile, original, 0o644); err != nil {
		t.Fatal(err)
	}

	wantErr := errors.New("commit observer failed")
	var stages []ProgressStage
	options := ProgressOptions{Observer: func(event ProgressEvent) error {
		stages = append(stages, event.Stage)
		if event.Stage == ProgressStageCommitting {
			return wantErr
		}
		return nil
	}}
	err := OptimizeFile(t.Context(), progressTestPDF(), outFile, nil, &options)
	if options.Input != "" || options.Item != 0 || options.Total != 0 {
		t.Fatalf("optimization modified caller options: %#v", options)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want cause %v", err, wantErr)
	}

	wantStages := []ProgressStage{
		ProgressStageReading,
		ProgressStageValidating,
		ProgressStageOptimizing,
		ProgressStageWriting,
		ProgressStageCommitting,
	}
	if !reflect.DeepEqual(stages, wantStages) {
		t.Fatalf("got stages %#v, want %#v", stages, wantStages)
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("destination changed: got %q, want %q", got, original)
	}
	temporaryFiles, globErr := filepath.Glob(filepath.Join(dir, ".out.pdf.tmp-*"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("temporary output remains: %v", temporaryFiles)
	}
}

func TestOptimizeFileNilOptionsCancellationPreservesDestination(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.pdf")
	original := []byte("original destination")
	if err := os.WriteFile(outFile, original, 0o644); err != nil {
		t.Fatal(err)
	}

	c := &cancelAfterFileChecksContext{Context: t.Context(), remaining: 2}
	err := OptimizeFile(c, progressTestPDF(), outFile, nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}

	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("destination changed: got %q, want %q", got, original)
	}
	temporaryFiles, err := filepath.Glob(filepath.Join(dir, ".out.pdf.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("temporary output remains: %v", temporaryFiles)
	}
}

func TestOptimizeFileCancellationPreservesDestination(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.pdf")
	original := []byte("original destination")
	if err := os.WriteFile(outFile, original, 0o644); err != nil {
		t.Fatal(err)
	}

	c, cancel := context.WithCancel(t.Context())
	options := ProgressOptions{Observer: func(event ProgressEvent) error {
		if event.Stage == ProgressStageCommitting {
			cancel()
		}
		return nil
	}}
	err := OptimizeFile(c, progressTestPDF(), outFile, nil, &options)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}

	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("destination changed: got %q, want %q", got, original)
	}
	temporaryFiles, err := filepath.Glob(filepath.Join(dir, ".out.pdf.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("temporary output remains: %v", temporaryFiles)
	}
}
