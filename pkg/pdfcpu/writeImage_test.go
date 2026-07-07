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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type writeReaderErrorReader struct {
	err  error
	read bool
}

// Read implements io.Reader.
func (r *writeReaderErrorReader) Read(p []byte) (int, error) {
	if r.read {
		return 0, r.err
	}
	r.read = true
	return copy(p, "partial"), nil
}

type writeReaderErrorWriteCloser struct {
	copyErr  error
	closeErr error
	closed   bool
	name     string
}

// Write implements io.Writer.
func (w *writeReaderErrorWriteCloser) Write([]byte) (int, error) {
	return 0, w.copyErr
}

// Close implements io.Closer.
func (w *writeReaderErrorWriteCloser) Close() error {
	w.closed = true
	return w.closeErr
}

// Name returns the writer name.
func (w *writeReaderErrorWriteCloser) Name() string {
	return w.name
}

type writeReaderContractWriteCloser struct {
	bytes.Buffer
	closeErr error
	closed   bool
	name     string
}

func (w *writeReaderContractWriteCloser) Close() error {
	w.closed = true
	return w.closeErr
}

func (w *writeReaderContractWriteCloser) Name() string {
	return w.name
}

// TestValidatePDFImageDimensionsRejectsPixelLimit verifies pixel limits are enforced.
func TestValidatePDFImageDimensionsRejectsPixelLimit(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxImagePixels = 9
	conf.Limits.MaxImageBytes = 1 << 20

	xRefTable := &model.XRefTable{Conf: conf}
	err := validatePDFImageDimensions(xRefTable, 5, 2, 3, 8, 1)
	if err == nil || !strings.Contains(err.Error(), "pixel count") {
		t.Fatalf("got %v, want pixel count limit error", err)
	}
}

// TestValidatePDFImageDimensionsRejectsByteLimit verifies byte limits are enforced.
func TestValidatePDFImageDimensionsRejectsByteLimit(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxImagePixels = 100
	conf.Limits.MaxImageBytes = 8

	xRefTable := &model.XRefTable{Conf: conf}
	err := validatePDFImageDimensions(xRefTable, 2, 2, 3, 8, 1)
	if err == nil || !strings.Contains(err.Error(), "byte size") {
		t.Fatalf("got %v, want byte size limit error", err)
	}
}

// TestWriteReaderLabelsCreateFailure verifies the corresponding behavior.
func TestWriteReaderLabelsCreateFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "out.bin")
	err := WriteReader(path, strings.NewReader("data"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "create temporary output "+path) {
		t.Fatalf("expected create phase, got %q", err.Error())
	}
}

// TestWriteReaderRejectsNilReader verifies nil input is rejected before output creation.
func TestWriteReaderRejectsNilReader(t *testing.T) {
	var typedNil *strings.Reader
	tests := []struct {
		name   string
		reader io.Reader
	}{
		{"nil interface", nil},
		{"typed nil", typedNil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "out.bin")
			if err := os.WriteFile(path, []byte("previous"), 0644); err != nil {
				t.Fatal(err)
			}

			err := WriteReader(path, tt.reader)
			if !errors.Is(err, ErrMissingReader) {
				t.Fatalf("expected %v, got %v", ErrMissingReader, err)
			}
			bb, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if got, want := string(bb), "previous"; got != want {
				t.Fatalf("existing output: got %q, want %q", got, want)
			}
		})
	}
}

// TestWriteReaderRemovesPartialOutputAfterCopyFailure verifies the corresponding behavior.
func TestWriteReaderRemovesPartialOutputAfterCopyFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.bin")
	wantErr := errors.New("copy failed")

	err := WriteReader(path, &writeReaderErrorReader{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "copy output "+path) {
		t.Fatalf("expected copy phase, got %q", err.Error())
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected partial output removal, got %v", statErr)
	}
}

// TestWriteReaderPreservesExistingOutputAfterCopyFailure verifies the corresponding behavior.
func TestWriteReaderPreservesExistingOutputAfterCopyFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.bin")
	if err := os.WriteFile(path, []byte("previous"), 0644); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("copy failed")

	err := WriteReader(path, &writeReaderErrorReader{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	bb, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got, want := string(bb), "previous"; got != want {
		t.Fatalf("existing output: got %q, want %q", got, want)
	}
	matches, globErr := filepath.Glob(filepath.Join(filepath.Dir(path), ".out.bin.tmp-*"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("expected temporary output cleanup, got %v", matches)
	}
}

// TestWriteReaderJoinsCopyCloseAndCleanupFailures verifies the corresponding behavior.
func TestWriteReaderJoinsCopyCloseAndCleanupFailures(t *testing.T) {
	copyErr := errors.New("copy failed")
	closeErr := errors.New("close failed")
	removeErr := errors.New("remove failed")
	w := &writeReaderErrorWriteCloser{copyErr: copyErr, closeErr: closeErr, name: "out.bin.tmp"}
	removed := false
	renamed := false

	err := writeReader(
		"out.bin",
		strings.NewReader("data"),
		func(string) (namedWriteCloser, error) { return w, nil },
		func(string, string) error {
			renamed = true
			return nil
		},
		func(string) error {
			removed = true
			return removeErr
		},
	)

	for _, wantErr := range []error{copyErr, closeErr, removeErr} {
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	}
	for _, want := range []string{"copy output out.bin", "close output out.bin", "remove temporary output for out.bin"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q, got %q", want, err.Error())
		}
	}
	if !w.closed {
		t.Fatal("expected writer to close")
	}
	if !removed {
		t.Fatal("expected partial output removal")
	}
	if renamed {
		t.Fatal("expected rename to be skipped after copy and close failures")
	}
}

// TestWriteReaderCloseFailurePreservesDestination verifies close failure prevents publication.
func TestWriteReaderCloseFailurePreservesDestination(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.bin")
	if err := os.WriteFile(path, []byte("previous"), 0644); err != nil {
		t.Fatal(err)
	}
	closeErr := errors.New("close failed")
	w := &writeReaderContractWriteCloser{
		closeErr: closeErr,
		name:     filepath.Join(dir, ".out.bin.tmp-contract"),
	}
	removed := false

	err := writeReader(
		path,
		strings.NewReader("replacement"),
		func(string) (namedWriteCloser, error) { return w, nil },
		func(string, string) error {
			t.Fatal("rename called after close failure")
			return nil
		},
		func(name string) error {
			if name != w.name {
				t.Fatalf("remove: got %q, want %q", name, w.name)
			}
			removed = true
			return nil
		},
	)

	if !errors.Is(err, closeErr) {
		t.Fatalf("expected %v, got %v", closeErr, err)
	}
	if !w.closed {
		t.Fatal("expected temporary output to close")
	}
	if !removed {
		t.Fatal("expected temporary output cleanup")
	}
	bb, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got, want := string(bb), "previous"; got != want {
		t.Fatalf("destination after close failure: got %q, want %q", got, want)
	}
}

// TestWriteReaderRenameFailurePreservesDestination verifies publication failure preserves existing output.
func TestWriteReaderRenameFailurePreservesDestination(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.bin")
	if err := os.WriteFile(path, []byte("previous"), 0644); err != nil {
		t.Fatal(err)
	}
	renameErr := errors.New("rename failed")
	w := &writeReaderContractWriteCloser{name: filepath.Join(dir, ".out.bin.tmp-contract")}
	removed := false

	err := writeReader(
		path,
		strings.NewReader("replacement"),
		func(string) (namedWriteCloser, error) { return w, nil },
		func(oldName, newName string) error {
			if !w.closed {
				t.Fatal("rename called before close")
			}
			if oldName != w.name || newName != path {
				t.Fatalf("rename: got (%q, %q), want (%q, %q)", oldName, newName, w.name, path)
			}
			return renameErr
		},
		func(name string) error {
			if name != w.name {
				t.Fatalf("remove: got %q, want %q", name, w.name)
			}
			removed = true
			return nil
		},
	)

	if !errors.Is(err, renameErr) {
		t.Fatalf("expected %v, got %v", renameErr, err)
	}
	if !removed {
		t.Fatal("expected temporary output cleanup")
	}
	bb, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got, want := string(bb), "previous"; got != want {
		t.Fatalf("destination after rename failure: got %q, want %q", got, want)
	}
}

// TestWriteReaderSuccess verifies the corresponding behavior.
func TestWriteReaderSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.bin")
	if err := WriteReader(path, strings.NewReader("data")); err != nil {
		t.Fatal(err)
	}
	bb, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(bb) != "data" {
		t.Fatalf("expected %q, got %q", "data", bb)
	}
}

// TestWriteReaderNewOutputUsesCreatePermissions verifies the corresponding behavior.
func TestWriteReaderNewOutputUsesCreatePermissions(t *testing.T) {
	dir := t.TempDir()
	controlPath := filepath.Join(dir, "control.bin")
	if err := os.WriteFile(controlPath, nil, 0666); err != nil {
		t.Fatal(err)
	}
	controlInfo, err := os.Stat(controlPath)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "out.bin")
	if err := WriteReader(path, strings.NewReader("data")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), controlInfo.Mode().Perm(); got != want {
		t.Fatalf("new output permissions: got %04o, want %04o", got, want)
	}
}

// TestWriteReaderReplacesExistingOutput verifies the corresponding behavior.
func TestWriteReaderReplacesExistingOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.bin")
	if err := os.WriteFile(path, []byte("previous"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := WriteReader(path, strings.NewReader("replacement")); err != nil {
		t.Fatal(err)
	}
	bb, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(bb), "replacement"; got != want {
		t.Fatalf("existing output: got %q, want %q", got, want)
	}
}

// TestWriteReaderPreservesExistingOutputPermissions verifies replacement retains destination permissions.
func TestWriteReaderPreservesExistingOutputPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.bin")
	if err := os.WriteFile(path, []byte("previous"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := WriteReader(path, strings.NewReader("replacement")); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := after.Mode().Perm(), before.Mode().Perm(); got != want {
		t.Fatalf("replacement permissions: got %04o, want %04o", got, want)
	}
}

// TestUnsupportedImageRenderingReturnsSentinel verifies the corresponding behavior.
func TestUnsupportedImageRenderingReturnsSentinel(t *testing.T) {
	im := &PDFImage{
		objNr: 7,
		sd:    &types.StreamDict{Dict: types.Dict{}},
		comp:  5,
		bpc:   8,
	}
	xRefTable := &model.XRefTable{}
	tests := []struct {
		name string
		want string
		fn   func() error
	}{
		{
			name: "indexed name colorspace",
			want: "indexed base colorspace Lab",
			fn: func() error {
				_, _, err := renderIndexedNameCS(im, types.Name(model.LabCS), 0, nil)
				return err
			},
		},
		{
			name: "indexed array colorspace",
			want: "indexed base colorspace [Lab]",
			fn: func() error {
				_, _, err := renderIndexedArrayCS(xRefTable, im, types.Array{types.Name(model.LabCS)}, 0, nil)
				return err
			},
		},
		{
			name: "indexed colorspace type",
			want: "indexed base colorspace type types.Integer",
			fn: func() error {
				cs := types.Array{types.Name(model.IndexedCS), types.Integer(1), types.Integer(0), types.StringLiteral("x")}
				_, _, err := renderIndexed(xRefTable, im, cs)
				return err
			},
		},
		{
			name: "DeviceN alternate type",
			want: "DeviceN alternate colorspace type types.Integer",
			fn: func() error {
				_, _, err := renderDeviceN(im, types.Array{types.Name(model.DeviceNCS), types.Array{}, types.Integer(1)})
				return err
			},
		},
		{
			name: "DeviceN alternate colorspace",
			want: "DeviceN alternate colorspace Lab",
			fn: func() error {
				_, _, err := renderDeviceN(im, types.Array{types.Name(model.DeviceNCS), types.Array{}, types.Name(model.LabCS)})
				return err
			},
		},
		{
			name: "array colorspace",
			want: "colorspace [CalGray",
			fn: func() error {
				sd := &types.StreamDict{
					Dict: types.Dict{
						"BitsPerComponent": types.Integer(8),
						"ColorSpace":       types.Array{types.Name(model.CalGrayCS), types.Dict{}},
						"Height":           types.Integer(1),
						"Width":            types.Integer(1),
					},
					Content: []byte{0},
				}
				_, _, err := RenderImage(xRefTable, sd, false, "", 7)
				return err
			},
		},
		{
			name: "unknown filter",
			want: "filter Unsupported",
			fn: func() error {
				sd := &types.StreamDict{FilterPipeline: []types.PDFFilter{{Name: "Unsupported"}}}
				_, _, err := RenderImage(nil, sd, false, "", 7)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, ErrUnsupportedResource) {
				t.Fatalf("expected %v, got %v", ErrUnsupportedResource, err)
			}
			for _, want := range []string{"image obj#7 render", tt.want} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected %q, got %q", want, err)
				}
			}
		})
	}
}
