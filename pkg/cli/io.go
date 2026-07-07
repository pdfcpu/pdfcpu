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

// Package cli provides pdfcpu command line processing.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/internal/fileutil"
	"github.com/pdfcpu/pdfcpu/pkg/log"
)

type streamInOutFinalizer struct {
	input       *os.File
	temporaryIn *temporaryInput
	output      *os.File
	outFile     string
	replaceOut  string
}

type temporaryInput struct {
	file   *os.File
	path   string
	remove func(string) error
}

var (
	createTemporaryInputFile = os.CreateTemp
	rewindTemporaryInputFile = func(f *os.File) error {
		_, err := f.Seek(0, io.SeekStart)
		return err
	}
)

// Read implements io.Reader.
func (in *temporaryInput) Read(p []byte) (int, error) {
	return in.file.Read(p)
}

// Seek implements io.Seeker.
func (in *temporaryInput) Seek(offset int64, whence int) (int64, error) {
	return in.file.Seek(offset, whence)
}

func (in *temporaryInput) finalize(op string, opErr error) error {
	closeErr := in.file.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("%s: close temporary input: %w", op, closeErr)
	}
	removeErr := in.remove(in.path)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		removeErr = fmt.Errorf("%s: remove temporary input: %w", op, removeErr)
	} else {
		removeErr = nil
	}
	return errors.Join(opErr, closeErr, removeErr)
}

func readSeekerFromStdin(op string) (*temporaryInput, error) {
	f, err := createTemporaryInputFile("", "pdfcpu-stdin-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("%s: create temporary input: %w", op, err)
	}
	in := &temporaryInput{file: f, path: f.Name(), remove: os.Remove}

	n, copyErr := io.Copy(f, os.Stdin)
	if copyErr != nil {
		return nil, in.finalize(op, fmt.Errorf("%s: read stdin: %w", op, copyErr))
	}
	if n == 0 {
		return nil, in.finalize(op, fmt.Errorf("%s: read stdin: stdin is empty", op))
	}
	if err := rewindTemporaryInputFile(f); err != nil {
		return nil, in.finalize(op, fmt.Errorf("%s: rewind temporary input: %w", op, err))
	}
	return in, nil
}

func withStdinReadSeeker[T any](op string, fn func(io.ReadSeeker) (T, error)) (T, error) {
	var zero T
	in, err := readSeekerFromStdin(op)
	if err != nil {
		return zero, err
	}
	result, opErr := fn(in)
	return result, in.finalize(op, opErr)
}

func closeStreamFile(f *os.File, context string) error {
	if f == nil {
		return nil
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}
	return nil
}

func removeStreamOutput(fileName, context string) error {
	if fileName == "" || fileName == "-" {
		return nil
	}
	if err := os.Remove(fileName); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("%s: %w", context, err)
	}
	return nil
}

func (f *streamInOutFinalizer) finalize(op string, opErr error) error {
	closeErr := errors.Join(
		closeStreamFile(f.output, op+": close output"),
	)
	err := errors.Join(opErr, closeErr)
	if f.temporaryIn != nil {
		err = f.temporaryIn.finalize(op, err)
	} else {
		err = errors.Join(err, closeStreamFile(f.input, op+": close input"))
	}
	if err != nil {
		return errors.Join(err, removeStreamOutput(f.outFile, op+": remove output"))
	}
	if f.replaceOut == "" {
		return err
	}
	if replaceErr := fileutil.ReplaceFile(f.outFile, f.replaceOut); replaceErr != nil {
		err = fmt.Errorf("%s: replace output: %w", op, replaceErr)
	}
	if err == nil {
		return nil
	}
	return errors.Join(err, removeStreamOutput(f.outFile, op+": remove output"))
}

func createStreamOutput(fileName string) (*os.File, string, string, error) {
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err == nil {
		return f, fileName, "", nil
	}
	if !errors.Is(err, os.ErrExist) {
		return nil, "", "", err
	}
	fi, err := os.Stat(fileName)
	if err != nil {
		return nil, "", "", err
	}
	pattern := "." + filepath.Base(fileName) + ".tmp-*"
	f, err = os.CreateTemp(filepath.Dir(fileName), pattern)
	if err != nil {
		return nil, "", "", err
	}
	if err := f.Chmod(fi.Mode().Perm()); err != nil {
		name := f.Name()
		_ = f.Close()
		_ = os.Remove(name)
		return nil, "", "", err
	}
	return f, f.Name(), fileName, nil
}

func streamInOutForOperation(inFile, outFile, op string) (io.ReadSeeker, io.Writer, func(error) error, error) {
	if inFile == "-" && outFile == "" {
		outFile = "-"
	}

	var rs io.ReadSeeker
	finalizer := &streamInOutFinalizer{}
	if inFile != "" {
		if inFile == "-" {
			in, err := readSeekerFromStdin(op)
			if err != nil {
				return nil, nil, nil, err
			}
			rs = in
			finalizer.temporaryIn = in
		} else {
			f, err := os.Open(inFile)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("%s: open input %s: %w", op, inFile, err)
			}
			rs = f
			finalizer.input = f
		}
	}

	if outFile == "-" {
		log.SetCLILogger(nil)
		return rs, os.Stdout, func(err error) error { return finalizer.finalize(op, err) }, nil
	}

	f, tmpFile, replaceOut, err := createStreamOutput(outFile)
	if err != nil {
		err = finalizer.finalize(op, fmt.Errorf("%s: create output %s: %w", op, outFile, err))
		return nil, nil, nil, err
	}
	finalizer.output = f
	finalizer.outFile = tmpFile
	finalizer.replaceOut = replaceOut
	return rs, f, func(err error) error { return finalizer.finalize(op, err) }, nil
}
