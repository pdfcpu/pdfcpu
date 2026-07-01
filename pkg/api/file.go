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
	"os"
	"path/filepath"
)

func createOutputFile(inFile, outFile string) (*os.File, string, error) {
	if outFile != "" && inFile != outFile {
		f, err := os.Create(outFile)
		return f, outFile, err
	}

	fi, err := os.Stat(inFile)
	if err != nil {
		return nil, "", err
	}

	pattern := "." + filepath.Base(inFile) + ".tmp-*"
	f, err := os.CreateTemp(filepath.Dir(inFile), pattern)
	if err != nil {
		return nil, "", err
	}

	if err := f.Chmod(fi.Mode().Perm()); err != nil {
		name := f.Name()
		_ = f.Close()
		_ = os.Remove(name)
		return nil, "", err
	}

	return f, f.Name(), nil
}
