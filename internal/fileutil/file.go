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

// Package fileutil provides internal cross-platform filesystem operations.
package fileutil

import (
	"errors"
	"os"
)

// ReplaceFile renames source over destination.
func ReplaceFile(source, destination string) error {
	return os.Rename(source, destination)
}

// RemoveFile removes path and succeeds if path does not exist.
func RemoveFile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
