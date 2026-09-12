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
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestWriteContextRejectsNilContext(t *testing.T) {
	if err := WriteContext(nil, nil); !errors.Is(err, ErrMissingContext) {
		t.Fatalf("got %v, want ErrMissingContext", err)
	}
}

func TestWriteContextReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	if err := WriteContext(c, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestWriteIncrementRejectsNilContext(t *testing.T) {
	if err := WriteIncrement(nil, nil); !errors.Is(err, ErrMissingContext) {
		t.Fatalf("got %v, want ErrMissingContext", err)
	}
}

func TestWriteIncrementReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	if err := WriteIncrement(c, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestWriteFlatObjectReturnsCancellationBeforeObjectProcessing(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	if err := writeFlatObject(c, nil, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestWriteKidsReturnsCancellationBeforeKidProcessing(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := writeKids(
		c,
		&model.Context{},
		types.Array{types.Integer(1)},
		new(int),
		0,
		model.NewPageTreeVisit(),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestWriteXRefSubsectionReturnsCancellationBeforeEntryProcessing(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	if err := writeXRefSubsection(c, nil, 0, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestCreateXRefStreamReturnsCancellationBeforeEntryProcessing(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := createXRefStream(c, &model.Context{}, 1, 1, 1, []int{0})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestWriteDeepObjectReturnsCancellationBeforeObjectProcessing(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := writeDeepObject(c, &model.Context{}, types.IndirectRef{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

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

	if err := WriteContext(t.Context(), ctx); err != nil {
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
	if runtime.GOOS != "windows" {
		if got, want := info.Mode().Perm(), os.FileMode(0640); got != want {
			t.Fatalf("output permissions: got %o, want %o", got, want)
		}
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
			err := WriteContext(t.Context(), tt.ctx)
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
			err := WriteIncrement(t.Context(), tt.ctx)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
