/*
Copyright 2025 The pdfcpu Authors.

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
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/pdfcpu/pdfcpu/internal/fileutil"
)

type namedWriteCloser interface {
	io.WriteCloser
	Name() string
}

func removeStagedFile(path, tmpPath string, remove func(string) error) error {
	if err := remove(tmpPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove temporary output for %s: %w", path, err)
	}
	return nil
}

func finishStagedFile(
	path string,
	w namedWriteCloser,
	writeErr error,
	closeInput func() error,
	replace func(string, string) error,
	remove func(string) error,
) error {
	tmpPath := w.Name()
	closeErr := w.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("close output %s: %w", path, closeErr)
	}
	var inputErr error
	if closeInput != nil {
		inputErr = closeInput()
	}
	if err := errors.Join(writeErr, closeErr, inputErr); err != nil {
		return errors.Join(err, removeStagedFile(path, tmpPath, remove))
	}
	if err := replace(tmpPath, path); err != nil {
		return errors.Join(
			fmt.Errorf("replace output %s: %w", path, err),
			removeStagedFile(path, tmpPath, remove),
		)
	}
	return nil
}

func writeReader(
	path string,
	r io.Reader,
	createTemp func(string) (namedWriteCloser, error),
	replace func(string, string) error,
	remove func(string) error,
) error {
	w, err := createTemp(path)
	if err != nil {
		return fmt.Errorf("create temporary output %s: %w", path, err)
	}
	_, writeErr := io.Copy(w, r)
	if writeErr != nil {
		writeErr = fmt.Errorf("copy output %s: %w", path, writeErr)
	}
	return finishStagedFile(path, w, writeErr, nil, replace, remove)
}

func createStagedFile(path string) (*os.File, error) {
	dir, prefix := filepath.Dir(path), "."+filepath.Base(path)+".tmp-"
	f, err := openStagedFile(dir, prefix)
	if err != nil {
		return nil, err
	}
	if fi, err := os.Stat(path); err == nil {
		if err := f.Chmod(fi.Mode().Perm()); err != nil {
			name := f.Name()
			return nil, errors.Join(err, f.Close(), os.Remove(name))
		}
	}
	return f, nil
}

func createWriteReaderTemp(path string) (namedWriteCloser, error) {
	return createStagedFile(path)
}

func openStagedFile(dir, prefix string) (*os.File, error) {
	const attempts = 100
	for range attempts {
		var suffix [8]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return nil, err
		}
		name := filepath.Join(dir, fmt.Sprintf("%s%x", prefix, suffix))
		f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return f, err
	}
	return nil, fmt.Errorf("exhausted temporary output name attempts")
}

func isNilReader(r io.Reader) bool {
	if r == nil {
		return true
	}
	v := reflect.ValueOf(r)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func writeNewFile(rd io.Reader, filePath string) (bool, error) {
	to, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, copyErr := io.Copy(to, rd)
	closeErr := to.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return false, errors.Join(err, os.Remove(filePath))
	}
	return true, nil
}

// WriteReader consumes r's content by writing it to a file at path.
func WriteReader(path string, r io.Reader) error {
	if isNilReader(r) {
		return ErrMissingReader
	}
	return writeReader(path, r, createWriteReaderTemp, fileutil.ReplaceFile, os.Remove)
}

// Write rd to filepath and respect overwrite.
func Write(rd io.Reader, filePath string, overwrite bool) (bool, error) {
	if isNilReader(rd) {
		return false, ErrMissingReader
	}
	if overwrite {
		return true, WriteReader(filePath, rd)
	}
	return writeNewFile(rd, filePath)
}

// CopyFile copies srcFilename to destFilename.
func CopyFile(srcFilename, destFilename string, overwrite bool) (bool, error) {
	from, err := os.Open(srcFilename)
	if err != nil {
		return false, err
	}
	if !overwrite {
		defer from.Close()
		return Write(from, destFilename, false)
	}
	sourceInfo, sourceErr := from.Stat()
	destinationInfo, destinationErr := os.Stat(destFilename)
	if sourceErr == nil && destinationErr == nil && os.SameFile(sourceInfo, destinationInfo) {
		return true, from.Close()
	}
	to, err := createStagedFile(destFilename)
	if err != nil {
		return false, errors.Join(err, from.Close())
	}
	_, copyErr := io.Copy(to, from)
	if copyErr != nil {
		copyErr = fmt.Errorf("copy output %s: %w", destFilename, copyErr)
	}
	closeInput := func() error {
		if err := from.Close(); err != nil {
			return fmt.Errorf("close input %s: %w", srcFilename, err)
		}
		return nil
	}
	return true, finishStagedFile(destFilename, to, copyErr, closeInput, fileutil.ReplaceFile, os.Remove)
}
