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

package fileutil

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile writes data beside name and replaces name only after writing and closing succeed.
// Existing permission bits are preserved; perm applies to new files subject to the process umask.
func WriteFile(name string, data []byte, perm os.FileMode) error {
	return writeFile(name, perm, func(f *os.File) error {
		_, err := f.Write(data)
		return err
	})
}

func createReplacement(name string, perm os.FileMode) (*os.File, error) {
	info, err := os.Stat(name)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	tmp := filepath.Join(filepath.Dir(name), "."+filepath.Base(name)+".tmp-"+rand.Text())
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return nil, err
	}
	if info != nil {
		if err := f.Chmod(info.Mode().Perm()); err != nil {
			return nil, errors.Join(err, f.Close(), RemoveFile(tmp))
		}
	}
	return f, nil
}

func writeFile(name string, perm os.FileMode, write func(*os.File) error) error {
	f, err := createReplacement(name, perm)
	if err != nil {
		return fmt.Errorf("create temporary output for %s: %w", name, err)
	}
	writeErr := write(f)
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return errors.Join(fmt.Errorf("write output %s: %w", name, err), RemoveFile(f.Name()))
	}
	if err := ReplaceFile(f.Name(), name); err != nil {
		return errors.Join(fmt.Errorf("replace output %s: %w", name, err), RemoveFile(f.Name()))
	}
	return nil
}
