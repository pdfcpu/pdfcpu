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
	stdlog "log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

type extractErrorWriter struct {
	err error
}

// Write implements io.Writer.
func (w extractErrorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func extractTestPDF(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "testdata", "Hybrid-PDF.pdf")
}

// TestExtractPagesStdoutReadErrorHasPhaseContext verifies the corresponding behavior.
func TestExtractPagesStdoutReadErrorHasPhaseContext(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "invalid.pdf")
	if err := os.WriteFile(inFile, []byte("not a PDF"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := ExtractPages(ExtractPagesCommand(inFile, "-", []string{"1"}, nil))
	if err == nil || !strings.Contains(err.Error(), "extract pages: read") {
		t.Fatalf("expected read phase context, got %v", err)
	}
}

// TestExtractCommandsContextualizeStdinFailures verifies the corresponding behavior.
func TestExtractCommandsContextualizeStdinFailures(t *testing.T) {
	stdin := os.Stdin
	f, err := os.CreateTemp(t.TempDir(), "empty-stdin-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = stdin
		_ = f.Close()
	})

	tests := []struct {
		name string
		op   string
		cmd  *Command
		fn   func(*Command) ([]string, error)
	}{
		{name: "images", op: "extract images", cmd: ExtractImagesCommand("-", t.TempDir(), nil, nil), fn: ExtractImages},
		{name: "fonts", op: "extract fonts", cmd: ExtractFontsCommand("-", t.TempDir(), nil, nil), fn: ExtractFonts},
		{name: "pages", op: "extract pages", cmd: ExtractPagesCommand("-", t.TempDir(), nil, nil), fn: ExtractPages},
		{name: "content", op: "extract content", cmd: ExtractContentCommand("-", t.TempDir(), nil, nil), fn: ExtractContent},
		{name: "metadata", op: "extract metadata", cmd: ExtractMetadataCommand("-", t.TempDir(), nil), fn: ExtractMetadata},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			_, err := tt.fn(tt.cmd)
			want := tt.op + ": read stdin"
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %v", want, err)
			}
		})
	}
}

// TestExtractPagesStdoutSelectionErrorHasPhaseContext verifies the corresponding behavior.
func TestExtractPagesStdoutSelectionErrorHasPhaseContext(t *testing.T) {
	_, err := ExtractPages(ExtractPagesCommand(extractTestPDF(t), "-", []string{"invalid"}, nil))
	if err == nil || !strings.Contains(err.Error(), "extract pages: selection") {
		t.Fatalf("expected selection phase context, got %v", err)
	}
}

// TestExtractPagesStdoutExtractionErrorHasPhaseContext verifies the corresponding behavior.
func TestExtractPagesStdoutExtractionErrorHasPhaseContext(t *testing.T) {
	err := writeExtractedPageToStdout(nil, 1, io.Discard)
	if !errors.Is(err, pdfcpu.ErrMissingPDFContext) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrMissingPDFContext, err)
	}
	if !strings.Contains(err.Error(), "extract pages: extraction") {
		t.Fatalf("expected extraction phase context, got %q", err.Error())
	}
}

// TestExtractPagesStdoutCopyErrorHasPhaseContext verifies the corresponding behavior.
func TestExtractPagesStdoutCopyErrorHasPhaseContext(t *testing.T) {
	f, err := os.Open(extractTestPDF(t))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	wantErr := errors.New("copy failed")
	cmd := ExtractPagesCommand("unused.pdf", "-", []string{"1"}, nil)
	err = extractSelectedPageToStdout(f, extractErrorWriter{err: wantErr}, cmd)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "extract pages: stdout copy") {
		t.Fatalf("expected stdout copy phase context, got %q", err.Error())
	}
}

// TestExtractPagesStdoutSuccessIsPurePDF verifies the corresponding behavior.
func TestExtractPagesStdoutSuccessIsPurePDF(t *testing.T) {
	stdout := os.Stdout
	out, err := os.CreateTemp(t.TempDir(), "stdout-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = out
	t.Cleanup(func() {
		os.Stdout = stdout
		_ = out.Close()
	})

	_, extractErr := ExtractPages(ExtractPagesCommand(extractTestPDF(t), "-", []string{"1"}, nil))
	os.Stdout = stdout
	if extractErr != nil {
		t.Fatal(extractErr)
	}
	if _, err := out.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	bb, err := io.ReadAll(out)
	if err != nil {
		t.Fatal(err)
	}
	pdf := bytes.TrimSpace(bb)
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("stdout does not start with a PDF header: %q", pdf)
	}
	if !bytes.HasSuffix(pdf, []byte("%%EOF")) {
		t.Fatalf("stdout contains data after the PDF trailer: %q", pdf[len(pdf)-min(len(pdf), 80):])
	}
	if err := api.Validate(bytes.NewReader(bb), nil); err != nil {
		t.Fatalf("stdout is not a valid PDF: %v", err)
	}
}

// TestExtractPagesStdoutBrokenPipe verifies the corresponding behavior.
func TestExtractPagesStdoutBrokenPipe(t *testing.T) {
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := pipeReader.Close(); err != nil {
		t.Fatal(err)
	}
	_, pipeErr := pipeWriter.Write([]byte("probe"))
	if pipeErr == nil {
		t.Fatal("expected broken-pipe probe to fail")
	}
	var pathErr *os.PathError
	if errors.As(pipeErr, &pathErr) {
		pipeErr = pathErr.Err
	}

	stdout := os.Stdout
	os.Stdout = pipeWriter
	t.Cleanup(func() {
		os.Stdout = stdout
		_ = pipeWriter.Close()
	})
	_, extractErr := ExtractPages(ExtractPagesCommand(extractTestPDF(t), "-", []string{"1"}, nil))
	os.Stdout = stdout

	if !errors.Is(extractErr, pipeErr) {
		t.Fatalf("expected broken-pipe error %v, got %v", pipeErr, extractErr)
	}
	if !strings.Contains(extractErr.Error(), "extract pages: stdout copy") {
		t.Fatalf("expected stdout copy phase context, got %q", extractErr)
	}
}

// TestExtractPagesStdoutCleanupPreservesErrors verifies the corresponding behavior.
func TestExtractPagesStdoutCleanupPreservesErrors(t *testing.T) {
	rs, _, finalize, err := streamInOutForOperation(extractTestPDF(t), "-", extractPagesOperation)
	if err != nil {
		t.Fatal(err)
	}
	f := rs.(*os.File)
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	wantErr := errors.New("operation failed")
	err = finalize(wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "extract pages: close input") {
		t.Fatalf("expected cleanup phase context, got %q", err.Error())
	}
}

// TestReportUnsupportedResourceSkipsDoesNotHideFailures verifies the corresponding behavior.
func TestReportUnsupportedResourceSkipsDoesNotHideFailures(t *testing.T) {
	stdout := os.Stdout
	stdoutFile, err := os.CreateTemp(t.TempDir(), "stdout-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = stdoutFile
	t.Cleanup(func() {
		os.Stdout = stdout
		_ = stdoutFile.Close()
	})

	var stderr bytes.Buffer
	log.SetCLILogger(stdlog.New(&stderr, "", 0))
	t.Cleanup(func() { log.SetCLILogger(nil) })

	skipCause := fmt.Errorf("image obj#7: %w", pdfcpu.ErrUnsupportedResource)
	skipErr := &api.UnsupportedResourceError{Err: skipCause}
	if err := reportUnsupportedResourceSkips(skipErr); err != nil {
		t.Fatalf("expected pure skip to be reported, got %v", err)
	}
	if got, want := stderr.String(), skipErr.Error()+"\n"; got != want {
		t.Fatalf("stderr: got %q, want %q", got, want)
	}

	stderr.Reset()
	wantErr := errors.New("close failed")
	err = reportUnsupportedResourceSkips(errors.Join(skipCause, wantErr))
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected mixed error not to be logged as a skip, got %q", stderr.String())
	}
	fi, err := stdoutFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() != 0 {
		t.Fatalf("expected stdout to remain empty, got %d bytes", fi.Size())
	}
}
