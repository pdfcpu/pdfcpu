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
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type nopReadWriteSeeker struct {
	*bytes.Reader
}

func (n nopReadWriteSeeker) Write(p []byte) (int, error) {
	return len(p), nil
}

func TestAPIArgumentErrors(t *testing.T) {
	tests := []struct {
		name    string
		fn      func() error
		wantErr error
	}{
		{
			name: "optimize missing reader",
			fn: func() error {
				return Optimize(nil, io.Discard, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "encrypt missing configuration",
			fn: func() error {
				return Encrypt(bytes.NewReader(nil), io.Discard, nil)
			},
			wantErr: ErrMissingConfiguration,
		},
		{
			name: "merge raw missing inputs",
			fn: func() error {
				return MergeRaw(nil, io.Discard, false, nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "merge missing writer",
			fn: func() error {
				return Merge("", []string{"in.pdf"}, nil, nil, false)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "merge missing input",
			fn: func() error {
				return Merge("", nil, io.Discard, nil, false)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "merge zip missing first reader",
			fn: func() error {
				return MergeCreateZip(nil, bytes.NewReader(nil), io.Discard, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "merge zip missing second reader",
			fn: func() error {
				return MergeCreateZip(bytes.NewReader(nil), nil, io.Discard, nil)
			},
			wantErr: ErrMissingPDFReadSeeker,
		},
		{
			name: "booklet missing configuration",
			fn: func() error {
				return Booklet(bytes.NewReader(nil), io.Discard, nil, nil, nil, nil)
			},
			wantErr: ErrMissingBookletConfiguration,
		},
		{
			name: "booklet file missing input",
			fn: func() error {
				return BookletFile(nil, "out.pdf", nil, nil, nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "update images missing image input",
			fn: func() error {
				return UpdateImages(bytes.NewReader(nil), nil, io.Discard, 1, 0, "", nil)
			},
			wantErr: ErrMissingImageInput,
		},
		{
			name: "update images missing writer",
			fn: func() error {
				return UpdateImages(bytes.NewReader(nil), bytes.NewReader(nil), nil, 1, 0, "", nil)
			},
			wantErr: ErrMissingPDFWriter,
		},
		{
			name: "validate context missing context",
			fn: func() error {
				return ValidateContext(nil)
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "optimize context missing context",
			fn: func() error {
				return OptimizeContext(nil)
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "write context missing context",
			fn: func() error {
				return WriteContext(nil, io.Discard)
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "write increment missing context",
			fn: func() error {
				return WriteIncrement(nil, io.Discard)
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "write missing context",
			fn: func() error {
				return Write(nil, io.Discard, nil)
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "write incr missing read write seeker",
			fn: func() error {
				return WriteIncr(&model.Context{}, nil, model.NewDefaultConfiguration())
			},
			wantErr: ErrMissingPDFReadWriteSeeker,
		},
		{
			name: "write incr missing configuration",
			fn: func() error {
				return WriteIncr(&model.Context{}, nopReadWriteSeeker{bytes.NewReader(nil)}, nil)
			},
			wantErr: ErrMissingConfiguration,
		},
		{
			name: "extract page missing context",
			fn: func() error {
				_, err := ExtractPage(nil, 1)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "watermark context missing context",
			fn: func() error {
				return WatermarkContext(nil, nil, nil)
			},
			wantErr: ErrMissingPDFContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestMergeErrorsIncludeSourceContext(t *testing.T) {
	err := appendTo(nil, "1", nil, false)
	if !errors.Is(err, ErrMissingPDFReadSeeker) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFReadSeeker, err)
	}
	if !strings.Contains(err.Error(), "merge source 1: read source") {
		t.Fatalf("expected source context, got %q", err.Error())
	}

	err = MergeRaw([]io.ReadSeeker{bytes.NewReader(nil)}, io.Discard, false, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "merge source 0: read and validate") {
		t.Fatalf("expected raw source context, got %q", err.Error())
	}
}

func TestValidateFilesReturnsJoinedErrors(t *testing.T) {
	stderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
		r.Close()
	}()

	inFiles := []string{"missing1.pdf", "missing2.pdf"}
	err = ValidateFiles(inFiles, nil)
	w.Close()

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	for _, fn := range inFiles {
		if !strings.Contains(err.Error(), fn) {
			t.Fatalf("expected %q in error, got %q", fn, err.Error())
		}
	}

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	if buf.Len() > 0 {
		t.Fatalf("expected no stderr output, got %q", buf.String())
	}
}
