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

package pdfcpu

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestWriteContextReplacesExistingDestination verifies direct filename publication.
func TestWriteContextReplacesExistingDestination(t *testing.T) {
	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(outFile, original, 0640); err != nil {
		t.Fatal(err)
	}
	ctx.Write.DirName = dir
	ctx.Write.FileName = filepath.Base(outFile)

	if err := WriteContext(ctx); err != nil {
		t.Fatal(err)
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if bytes.Equal(got, original) || !bytes.HasPrefix(got, []byte("%PDF-")) {
		t.Fatalf("unexpected PDF output prefix: %q", got[:min(len(got), 16)])
	}
	info, statErr := os.Stat(outFile)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0640); got != want {
		t.Fatalf("output permissions: got %o, want %o", got, want)
	}
}

// TestWriteContextRejectsInvalidInput verifies stable write precondition errors.
func TestWriteContextRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *model.Context
		wantErr error
	}{
		{
			name:    "missing context",
			wantErr: ErrMissingPDFContext,
		},
		{
			name:    "missing write context",
			ctx:     &model.Context{},
			wantErr: ErrMissingWriteContext,
		},
		{
			name:    "missing xref table",
			ctx:     &model.Context{Write: model.NewWriteContext("")},
			wantErr: ErrMissingXRefTable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WriteContext(tt.ctx)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

// TestWriteIncrementRejectsInvalidInput verifies stable increment precondition errors.
func TestWriteIncrementRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *model.Context
		wantErr error
	}{
		{
			name:    "missing context",
			wantErr: ErrMissingPDFContext,
		},
		{
			name:    "missing write context",
			ctx:     &model.Context{},
			wantErr: ErrMissingWriteContext,
		},
		{
			name:    "missing xref table",
			ctx:     &model.Context{Write: model.NewWriteContext("")},
			wantErr: ErrMissingXRefTable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WriteIncrement(tt.ctx)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
