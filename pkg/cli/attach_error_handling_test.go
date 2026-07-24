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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func cliAttachmentTestInputFile() string {
	return filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
}

func TestListAttachmentsUsesAPIBoundarySentinel(t *testing.T) {
	_, err := listAttachments(nil, nil, true, true)
	if !errors.Is(err, api.ErrMissingPDFReadSeeker) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFReadSeeker, err)
	}
}

func TestListAttachmentsUsesAPIOperationContext(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "empty-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = ListAttachmentsFile(f.Name(), nil)
	if err == nil || !strings.Contains(err.Error(), "list attachments: prepare PDF context: read context") {
		t.Fatalf("expected API operation context, got %v", err)
	}
}

func TestAttachmentCommandBoundaryGuards(t *testing.T) {
	empty := ""
	inFile := "in.pdf"
	outFile := "out.pdf"
	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{name: "list nil command", run: func() error {
			_, err := ListAttachments(nil)
			return err
		}, want: ErrMissingCommand},
		{name: "list missing input", run: func() error {
			_, err := ListAttachments(&Command{})
			return err
		}, want: api.ErrMissingPDFInput},
		{name: "add nil command", run: func() error {
			_, err := AddAttachments(nil)
			return err
		}, want: ErrMissingCommand},
		{name: "add missing output", run: func() error {
			_, err := AddAttachments(&Command{InFile: &inFile})
			return err
		}, want: api.ErrMissingPDFOutput},
		{name: "remove nil command", run: func() error {
			_, err := RemoveAttachments(nil)
			return err
		}, want: ErrMissingCommand},
		{name: "remove missing output", run: func() error {
			_, err := RemoveAttachments(&Command{InFile: &inFile})
			return err
		}, want: api.ErrMissingPDFOutput},
		{name: "extract nil command", run: func() error {
			_, err := ExtractAttachments(nil)
			return err
		}, want: ErrMissingCommand},
		{name: "extract missing directory", run: func() error {
			_, err := ExtractAttachments(&Command{InFile: &inFile, OutFile: &outFile})
			return err
		}, want: api.ErrMissingPDFOutput},
		{name: "extract empty directory", run: func() error {
			_, err := ExtractAttachments(&Command{InFile: &inFile, OutDir: &empty})
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

func TestListAttachmentsFileErrorsIncludeContext(t *testing.T) {
	if _, err := ListAttachmentsFile("", nil); !errors.Is(err, api.ErrMissingPDFInput) {
		t.Fatalf("expected %v, got %v", api.ErrMissingPDFInput, err)
	}

	missing := filepath.Join(t.TempDir(), "missing.pdf")
	_, err := ListAttachmentsFile(missing, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "list attachments: open input "+missing) {
		t.Fatalf("expected list open context, got %v", err)
	}
}

func TestListAttachmentsFileReportsCloseError(t *testing.T) {
	wantErr := errors.New("list attachments close failed")
	closeInput := closeListAttachmentsInput
	closeListAttachmentsInput = func(f *os.File) error {
		_ = closeInput(f)
		return wantErr
	}
	t.Cleanup(func() {
		closeListAttachmentsInput = closeInput
	})

	_, err := ListAttachmentsFile(cliAttachmentTestInputFile(), nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "list attachments: close input "+cliAttachmentTestInputFile()) {
		t.Fatalf("expected list close context, got %v", err)
	}
}

func TestAttachmentStreamingOpenErrorsIncludeOperationContext(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.pdf")
	out := "-"
	tests := []struct {
		name string
		cmd  *Command
		run  func(*Command) error
		want string
	}{
		{
			name: "add",
			cmd:  &Command{InFile: &missing, OutFile: &out},
			run: func(cmd *Command) error {
				_, err := AddAttachments(cmd)
				return err
			},
			want: "add attachments: open input " + missing,
		},
		{
			name: "portfolio",
			cmd:  &Command{Mode: model.ADDATTACHMENTSPORTFOLIO, InFile: &missing, OutFile: &out},
			run: func(cmd *Command) error {
				_, err := AddAttachments(cmd)
				return err
			},
			want: "add portfolio attachments: open input " + missing,
		},
		{
			name: "remove",
			cmd:  &Command{InFile: &missing, OutFile: &out},
			run: func(cmd *Command) error {
				_, err := RemoveAttachments(cmd)
				return err
			},
			want: "remove attachments: open input " + missing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(tt.cmd)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}
