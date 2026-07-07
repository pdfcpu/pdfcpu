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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/internal/fileutil"
)

type fileOperations struct {
	openExclusiveFn func(string, int, os.FileMode) (*os.File, error)
	createTempFn    func(string, string) (*os.File, error)
	statFn          func(string) (os.FileInfo, error)
	chmodFn         func(*os.File, os.FileMode) error
	closeFn         func(*os.File) error
	removeFn        func(string) error
	replaceFn       func(string, string) error
}

func (ops fileOperations) closeFile(f *os.File, context string) error {
	if f == nil {
		return nil
	}
	if err := ops.closeFn(f); err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}
	return nil
}

func (ops fileOperations) removeFile(fileName, context string) error {
	if fileName == "" {
		return nil
	}
	if err := ops.removeFn(fileName); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s: %w", context, err)
	}
	return nil
}

func (ops fileOperations) replaceFile(oldName, newName, context string) error {
	if err := ops.replaceFn(oldName, newName); err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}
	return nil
}

func defaultFileOperations() fileOperations {
	return fileOperations{
		openExclusiveFn: func(name string, flag int, perm os.FileMode) (*os.File, error) {
			return os.OpenFile(name, flag, perm)
		},
		createTempFn: os.CreateTemp,
		statFn:       os.Stat,
		chmodFn:      func(f *os.File, mode os.FileMode) error { return f.Chmod(mode) },
		closeFn:      (*os.File).Close,
		removeFn:     os.Remove,
		replaceFn:    fileutil.ReplaceFile,
	}
}

func closeFile(f *os.File, context string) error {
	return defaultFileOperations().closeFile(f, context)
}

func removeFile(fileName, context string) error {
	return defaultFileOperations().removeFile(fileName, context)
}

func replaceFile(oldName, newName, context string) error {
	return defaultFileOperations().replaceFile(oldName, newName, context)
}

type stagedFile struct {
	file    *os.File
	context string
}

type stagedOutput struct {
	output         stagedFile
	inputs         []stagedFile
	extraClosers   []func() error
	temporaryFile  string
	destination    string
	removeContext  string
	replaceContext string
	operations     fileOperations
}

func newStagedOutput(
	input, output *os.File,
	temporaryFile, inFile, outFile, replaceOut, operation string,
) stagedOutput {
	destination := replaceOut
	replaceContext := operation + ": replace output"
	if destination == "" && (outFile == "" || inFile == outFile) {
		destination = inFile
		replaceContext = operation + ": replace input"
	}
	return stagedOutput{
		output:         stagedFile{output, operation + ": close output"},
		inputs:         []stagedFile{{input, operation + ": close input"}},
		temporaryFile:  temporaryFile,
		destination:    destination,
		removeContext:  operation + ": remove output",
		replaceContext: replaceContext,
		operations:     defaultFileOperations(),
	}
}

func openStagedOutput(input *os.File, inFile, outFile, operation string) (stagedOutput, error) {
	return openStagedOutputWithOperations(input, inFile, outFile, operation, defaultFileOperations())
}

func openStagedOutputWithOperations(
	input *os.File,
	inFile, outFile, operation string,
	ops fileOperations,
) (stagedOutput, error) {
	target := inFile
	replaceOut := ""
	if outFile != "" && inFile != outFile {
		f, err := ops.openExclusiveFn(outFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		if err == nil {
			staged := newStagedOutput(input, f, outFile, inFile, outFile, "", operation)
			staged.operations = ops
			return staged, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return stagedOutput{}, err
		}
		target = outFile
		replaceOut = outFile
	}

	fi, err := ops.statFn(target)
	if err != nil {
		return stagedOutput{}, err
	}
	pattern := "." + filepath.Base(target) + ".tmp-*"
	f, err := ops.createTempFn(filepath.Dir(target), pattern)
	if err != nil {
		return stagedOutput{}, err
	}
	if err := ops.chmodFn(f, fi.Mode().Perm()); err != nil {
		name := f.Name()
		return stagedOutput{}, errors.Join(
			err,
			ops.closeFile(f, operation+": close temporary output"),
			ops.removeFile(name, operation+": remove temporary output"),
		)
	}
	staged := newStagedOutput(input, f, f.Name(), inFile, outFile, replaceOut, operation)
	staged.operations = ops
	return staged, nil
}

func (s stagedOutput) cleanup(processErr error) error {
	return errors.Join(
		processErr,
		s.operations.closeFile(s.output.file, s.output.context),
		s.closeInputs(),
		s.operations.removeFile(s.temporaryFile, s.removeContext),
	)
}

func (s stagedOutput) withInput(file *os.File, context string) stagedOutput {
	s.inputs = append(s.inputs, stagedFile{file, context})
	return s
}

func (s stagedOutput) withCloser(closeFn func() error) stagedOutput {
	s.extraClosers = append(s.extraClosers, closeFn)
	return s
}

func (s stagedOutput) closeInputs() error {
	errs := make([]error, 0, len(s.inputs)+len(s.extraClosers))
	for _, input := range s.inputs {
		errs = append(errs, s.operations.closeFile(input.file, input.context))
	}
	for _, closeFn := range s.extraClosers {
		errs = append(errs, closeFn())
	}
	return errors.Join(errs...)
}

func (s stagedOutput) commit() error {
	if err := s.operations.closeFile(s.output.file, s.output.context); err != nil {
		return errors.Join(err, s.closeInputs(), s.operations.removeFile(s.temporaryFile, s.removeContext))
	}
	if err := s.closeInputs(); err != nil {
		if s.destination != "" {
			return errors.Join(err, s.operations.removeFile(s.temporaryFile, s.removeContext))
		}
		return err
	}
	if s.destination == "" {
		return nil
	}
	if err := s.operations.replaceFile(s.temporaryFile, s.destination, s.replaceContext); err != nil {
		return errors.Join(err, s.operations.removeFile(s.temporaryFile, s.removeContext))
	}
	return nil
}

func outputAliasesInput(inFile, outFile string) (bool, error) {
	return outputAliasesInputWith(inFile, outFile, filepath.Abs, os.Stat)
}

func outputAliasesInputWith(
	inFile, outFile string,
	abs func(string) (string, error),
	stat func(string) (os.FileInfo, error),
) (bool, error) {
	inAbs, err := abs(inFile)
	if err != nil {
		return false, fmt.Errorf("resolve input path: %w", err)
	}
	outAbs, err := abs(outFile)
	if err != nil {
		return false, fmt.Errorf("resolve output path: %w", err)
	}
	if inAbs == outAbs {
		return true, nil
	}
	outInfo, err := stat(outFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat output: %w", err)
	}
	inInfo, err := stat(inFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat input: %w", err)
	}
	return os.SameFile(inInfo, outInfo), nil
}
