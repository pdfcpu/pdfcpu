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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func attachmentTestInputFile() string {
	return filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
}

func copyAttachmentTestInput(t *testing.T) string {
	t.Helper()

	b, err := os.ReadFile(attachmentTestInputFile())
	if err != nil {
		t.Fatal(err)
	}
	inFile := filepath.Join(t.TempDir(), "in.pdf")
	if err := os.WriteFile(inFile, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return inFile
}

type attachmentIOTracker struct {
	used      bool
	readCalls int
	seekCalls int
}

func (r *attachmentIOTracker) Read(_ []byte) (int, error) {
	r.used = true
	r.readCalls++
	return 0, errors.New("unexpected attachment read")
}

func (r *attachmentIOTracker) Seek(_ int64, _ int) (int64, error) {
	r.used = true
	r.seekCalls++
	return 0, errors.New("unexpected attachment seek")
}

func TestAttachmentAPIArgumentErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "list reader", err: func() error {
			_, err := Attachments(nil, nil)
			return err
		}(), want: ErrMissingPDFReadSeeker},
		{name: "add reader", err: AddAttachments(nil, io.Discard, nil, false, nil), want: ErrMissingPDFReadSeeker},
		{name: "add writer", err: AddAttachments(bytes.NewReader(nil), nil, nil, false, nil), want: ErrMissingPDFWriter},
		{name: "remove reader", err: RemoveAttachments(nil, io.Discard, nil, nil), want: ErrMissingPDFReadSeeker},
		{name: "remove writer", err: RemoveAttachments(bytes.NewReader(nil), nil, nil, nil), want: ErrMissingPDFWriter},
		{name: "extract reader", err: ExtractAttachments(nil, "", nil, nil), want: ErrMissingPDFReadSeeker},
		{
			name: "extract output",
			err:  ExtractAttachments(bytes.NewReader(nil), "", nil, nil),
			want: ErrMissingPDFOutput,
		},
		{name: "add file input", err: AddAttachmentsFile("", "", nil, false, nil), want: ErrMissingPDFInput},
		{name: "add portfolio file input", err: AddAttachmentsFile("", "", nil, true, nil), want: ErrMissingPDFInput},
		{name: "remove file input", err: RemoveAttachmentsFile("", "", nil, nil), want: ErrMissingPDFInput},
		{name: "extract file input", err: ExtractAttachmentsFile("", "", nil, nil), want: ErrMissingPDFInput},
		{
			name: "extract file output",
			err:  ExtractAttachmentsFile("input.pdf", "", nil, nil),
			want: ErrMissingPDFOutput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, tt.err)
			}
		})
	}
}

func TestAttachmentAPIsValidateFileNamesBeforeReading(t *testing.T) {
	tests := []struct {
		name string
		run  func(*attachmentIOTracker) error
		want string
	}{
		{
			name: "add attachments",
			run: func(rs *attachmentIOTracker) error {
				return AddAttachments(rs, io.Discard, []string{",description"}, false, nil)
			},
			want: "add attachments: validate attachment filenames",
		},
		{
			name: "add portfolio attachments",
			run: func(rs *attachmentIOTracker) error {
				return AddAttachments(rs, io.Discard, []string{",description"}, true, nil)
			},
			want: "add portfolio attachments: validate attachment filenames",
		},
		{
			name: "remove attachments",
			run: func(rs *attachmentIOTracker) error {
				return RemoveAttachments(rs, io.Discard, []string{""}, nil)
			},
			want: "remove attachments: validate attachment filenames",
		},
		{
			name: "extract attachments",
			run: func(rs *attachmentIOTracker) error {
				_, err := ExtractAttachmentsRaw(rs, "", []string{""}, nil)
				return err
			},
			want: "extract attachments: validate attachment filenames",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &attachmentIOTracker{}
			err := tt.run(rs)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
			if rs.used {
				t.Fatal("validation error reached PDF input")
			}
		})
	}
}

func TestAttachmentAPIsSetCommandMode(t *testing.T) {
	tests := []struct {
		name string
		run  func(*model.Configuration) error
		want model.CommandMode
	}{
		{
			name: "add attachments",
			run: func(conf *model.Configuration) error {
				return AddAttachments(bytes.NewReader(nil), io.Discard, []string{"attachment.pdf"}, false, conf)
			},
			want: model.ADDATTACHMENTS,
		},
		{
			name: "add portfolio attachments",
			run: func(conf *model.Configuration) error {
				return AddAttachments(bytes.NewReader(nil), io.Discard, []string{"attachment.pdf"}, true, conf)
			},
			want: model.ADDATTACHMENTSPORTFOLIO,
		},
		{
			name: "remove attachments",
			run: func(conf *model.Configuration) error {
				return RemoveAttachments(bytes.NewReader(nil), io.Discard, nil, conf)
			},
			want: model.REMOVEATTACHMENTS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := model.NewDefaultConfiguration()
			_ = tt.run(conf)
			if conf.Cmd != tt.want {
				t.Fatalf("expected command %d, got %d", tt.want, conf.Cmd)
			}
		})
	}
}

func TestAttachmentAPIReadErrorsIncludeOperationContext(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "list attachments",
			run: func() error {
				_, err := Attachments(bytes.NewReader(nil), nil)
				return err
			},
			want: "list attachments: prepare PDF context: read context",
		},
		{
			name: "add attachments",
			run: func() error {
				return AddAttachments(bytes.NewReader(nil), io.Discard, []string{"attachment.pdf"}, false, nil)
			},
			want: "add attachments: prepare PDF context: read context",
		},
		{
			name: "add portfolio attachments",
			run: func() error {
				return AddAttachments(bytes.NewReader(nil), io.Discard, []string{"attachment.pdf"}, true, nil)
			},
			want: "add portfolio attachments: prepare PDF context: read context",
		},
		{
			name: "remove attachments",
			run: func() error {
				return RemoveAttachments(bytes.NewReader(nil), io.Discard, nil, nil)
			},
			want: "remove attachments: prepare PDF context: read context",
		},
		{
			name: "extract attachments",
			run: func() error {
				_, err := ExtractAttachmentsRaw(bytes.NewReader(nil), "", nil, nil)
				return err
			},
			want: "extract attachments: read context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func attachmentTestPDFWithAttachment(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	err := AddAttachments(
		openAPITestPDF(t, attachmentTestInputFile()),
		&buf,
		[]string{attachmentTestInputFile()},
		false,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestAttachmentAPIWriteErrorsIncludeOperationContext(t *testing.T) {
	wantErr := errors.New("attachment write failed")
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "add attachments",
			run: func() error {
				return AddAttachments(
					openAPITestPDF(t, attachmentTestInputFile()),
					failingWriter{err: wantErr},
					[]string{attachmentTestInputFile()},
					false,
					nil,
				)
			},
			want: "add attachments: write output",
		},
		{
			name: "add portfolio attachments",
			run: func() error {
				return AddAttachments(
					openAPITestPDF(t, attachmentTestInputFile()),
					failingWriter{err: wantErr},
					[]string{attachmentTestInputFile()},
					true,
					nil,
				)
			},
			want: "add portfolio attachments: write output",
		},
		{
			name: "remove attachments",
			run: func() error {
				return RemoveAttachments(
					bytes.NewReader(attachmentTestPDFWithAttachment(t)),
					failingWriter{err: wantErr},
					[]string{filepath.Base(attachmentTestInputFile())},
					nil,
				)
			},
			want: "remove attachments: write output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestAddAttachmentsNoFilesStopsBeforeConfigurationOrPDF(t *testing.T) {
	for _, tt := range []struct {
		name string
		coll bool
		want string
	}{
		{name: "attachments", want: "add attachments: no attachment added"},
		{name: "portfolio", coll: true, want: "add portfolio attachments: no attachment added"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rs := &attachmentIOTracker{}
			conf := model.NewDefaultConfiguration()
			cmd := conf.Cmd

			err := AddAttachments(rs, io.Discard, nil, tt.coll, conf)

			if !errors.Is(err, ErrNoAttachmentAdded) {
				t.Fatalf("expected %v, got %v", ErrNoAttachmentAdded, err)
			}
			if err.Error() != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, err)
			}
			if rs.used {
				t.Fatal("empty attachment list reached PDF input")
			}
			if rs.readCalls != 0 || rs.seekCalls != 0 {
				t.Fatalf("expected no PDF access, got %d reads and %d seeks", rs.readCalls, rs.seekCalls)
			}
			if conf.Cmd != cmd {
				t.Fatalf("configuration command changed from %d to %d", cmd, conf.Cmd)
			}
		})
	}
}

func TestAddAttachmentsFileNoFilesStopsBeforeFileIO(t *testing.T) {
	for _, tt := range []struct {
		name string
		coll bool
		want string
	}{
		{name: "attachments", want: "add attachments: no attachment added"},
		{name: "portfolio", coll: true, want: "add portfolio attachments: no attachment added"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			conf := model.NewDefaultConfiguration()
			cmd := conf.Cmd

			err := AddAttachmentsFile("input.pdf", "output.pdf", nil, tt.coll, conf)

			if !errors.Is(err, ErrNoAttachmentAdded) {
				t.Fatalf("expected %v, got %v", ErrNoAttachmentAdded, err)
			}
			if err.Error() != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, err)
			}
			if conf.Cmd != cmd {
				t.Fatalf("configuration command changed from %d to %d", cmd, conf.Cmd)
			}
		})
	}
}

func TestExtractAttachmentsMissingOutputStopsBeforePDF(t *testing.T) {
	rs := &attachmentIOTracker{}

	err := ExtractAttachments(rs, "", []string{""}, nil)

	if !errors.Is(err, ErrMissingPDFOutput) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFOutput, err)
	}
	if strings.Contains(err.Error(), "validate attachment filenames") {
		t.Fatalf("filename validation preceded missing output: %v", err)
	}
	if rs.readCalls != 0 || rs.seekCalls != 0 {
		t.Fatalf("expected no PDF access, got %d reads and %d seeks", rs.readCalls, rs.seekCalls)
	}
}

func TestExtractAttachmentsFileMissingOutputStopsBeforeOpen(t *testing.T) {
	err := ExtractAttachmentsFile("input.pdf", "", []string{""}, nil)

	if !errors.Is(err, ErrMissingPDFOutput) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFOutput, err)
	}
	if strings.Contains(err.Error(), "validate attachment filenames") {
		t.Fatalf("filename validation preceded missing output: %v", err)
	}
}

func TestRemoveAttachmentsNoMatchPreservesSentinel(t *testing.T) {
	err := RemoveAttachments(
		bytes.NewReader(attachmentTestPDFWithAttachment(t)),
		io.Discard,
		[]string{"missing-attachment"},
		nil,
	)
	if !errors.Is(err, ErrNoAttachmentRemoved) {
		t.Fatalf("expected %v, got %v", ErrNoAttachmentRemoved, err)
	}
	if !strings.Contains(err.Error(), "remove attachments: no attachment removed") {
		t.Fatalf("expected remove operation context, got %v", err)
	}
}

func TestRemoveAttachmentsWithoutAttachmentsPreservesSentinel(t *testing.T) {
	err := RemoveAttachments(openAPITestPDF(t, attachmentTestInputFile()), io.Discard, nil, nil)
	if !errors.Is(err, ErrNoAttachmentRemoved) {
		t.Fatalf("expected %v, got %v", ErrNoAttachmentRemoved, err)
	}
	if !strings.Contains(err.Error(), "remove attachments: no attachment removed") {
		t.Fatalf("expected remove operation context, got %v", err)
	}
}

func TestAddAttachmentsPreservesCommaInDescription(t *testing.T) {
	const desc = "description,with,commas"

	var buf bytes.Buffer
	err := AddAttachments(
		openAPITestPDF(t, attachmentTestInputFile()),
		&buf,
		[]string{attachmentTestInputFile() + "," + desc},
		false,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	attachments, err := Attachments(bytes.NewReader(buf.Bytes()), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected one attachment, got %d", len(attachments))
	}
	if attachments[0].Desc != desc {
		t.Fatalf("expected description %q, got %q", desc, attachments[0].Desc)
	}
}

func TestAddAttachmentsSourceErrorsIncludeContext(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.bin")
	tests := []struct {
		name string
		coll bool
		want string
	}{
		{name: "attachment", want: "add attachments: open attachment " + missing},
		{name: "portfolio", coll: true, want: "add portfolio attachments: open attachment " + missing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddAttachments(openAPITestPDF(t, attachmentTestInputFile()), io.Discard, []string{missing}, tt.coll, nil)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestAttachmentMutationFileErrorsIncludeContext(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.pdf")
	inFile := copyAttachmentTestInput(t)
	outFile := filepath.Join(t.TempDir(), "missing", "out.pdf")
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "add open input",
			run: func() error {
				return AddAttachmentsFile(missing, "", []string{attachmentTestInputFile()}, false, nil)
			},
			want: "add attachments: open input " + missing,
		},
		{
			name: "portfolio open input",
			run: func() error {
				return AddAttachmentsFile(missing, "", []string{attachmentTestInputFile()}, true, nil)
			},
			want: "add portfolio attachments: open input " + missing,
		},
		{
			name: "remove open input",
			run: func() error {
				return RemoveAttachmentsFile(missing, "", nil, nil)
			},
			want: "remove attachments: open input " + missing,
		},
		{
			name: "add create output",
			run: func() error {
				return AddAttachmentsFile(inFile, outFile, []string{attachmentTestInputFile()}, false, nil)
			},
			want: "add attachments: create output",
		},
		{
			name: "remove create output",
			run: func() error {
				return RemoveAttachmentsFile(inFile, outFile, nil, nil)
			},
			want: "remove attachments: create output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestAttachmentMutationFailurePreservesExistingOutput(t *testing.T) {
	inFile := copyAttachmentTestInput(t)
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(outFile, original, 0o640); err != nil {
		t.Fatal(err)
	}

	missing := filepath.Join(t.TempDir(), "missing.bin")
	err := AddAttachmentsFile(inFile, outFile, []string{missing}, false, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("existing output changed: got %q", got)
	}
}

func TestAttachmentMutationReplacesExistingOutputAfterSuccess(t *testing.T) {
	inFile := copyAttachmentTestInput(t)
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("existing output"), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := AddAttachmentsFile(inFile, outFile, []string{attachmentTestInputFile()}, false, nil); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(outFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	attachments, err := Attachments(f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected one attachment, got %d", len(attachments))
	}

	input, err := os.Open(inFile)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	attachments, err = Attachments(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(attachments) != 0 {
		t.Fatalf("input changed: got %d attachments", len(attachments))
	}
}

type failingAttachmentReader struct {
	err error
}

func (r failingAttachmentReader) Read([]byte) (int, error) {
	return 0, r.err
}

func attachmentExtractionOutput(t *testing.T) (string, []byte) {
	t.Helper()
	fileName := filepath.Join(t.TempDir(), "attachment.bin")
	original := []byte("existing attachment")
	if err := os.WriteFile(fileName, original, 0o644); err != nil {
		t.Fatal(err)
	}
	return fileName, original
}

func requireAttachmentOutput(t *testing.T, fileName string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q", got)
	}
	requireNoAttachmentTempFiles(t, fileName)
}

func requireNoAttachmentTempFiles(t *testing.T, fileName string) {
	t.Helper()
	pattern := filepath.Join(filepath.Dir(fileName), "."+filepath.Base(fileName)+".tmp-*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary attachment outputs remain: %v", matches)
	}
}

func TestWriteAttachmentCopyFailurePreservesOutput(t *testing.T) {
	wantErr := errors.New("attachment copy failed")
	fileName, original := attachmentExtractionOutput(t)
	a := model.Attachment{
		Reader:   failingAttachmentReader{err: wantErr},
		FileName: filepath.Base(fileName),
	}

	err := writeAttachment(filepath.Dir(fileName), 0, a)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "extract attachments: write output "+fileName) {
		t.Fatalf("expected extraction write context, got %v", err)
	}
	requireAttachmentOutput(t, fileName, original)
}

func TestWriteAttachmentReplacesOutputAfterCopyAndClose(t *testing.T) {
	fileName, _ := attachmentExtractionOutput(t)

	err := writeAttachmentToPath(fileName, model.Attachment{Reader: strings.NewReader("replacement")})
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "replacement" {
		t.Fatalf("expected replacement output, got %q", got)
	}
	requireNoAttachmentTempFiles(t, fileName)
}

func TestExtractAttachmentsFileOpenErrorIncludesContext(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.pdf")
	err := ExtractAttachmentsFile(missing, t.TempDir(), nil, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "extract attachments: open input "+missing) {
		t.Fatalf("expected extraction open context, got %v", err)
	}
}
