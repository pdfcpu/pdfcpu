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
	"io"
	"os"
)

// Write rd to filepath and respect overwrite.
func Write(rd io.Reader, filepath string, overwrite bool) (bool, error) {
	if !overwrite {
		if _, err := os.Stat(filepath); err == nil {
			return false, nil
		}
	}

	to, err := os.Create(filepath)
	if err != nil {
		return false, err
	}
	defer to.Close()

	_, err = io.Copy(to, rd)
	return true, err
}

// CopyFile copies srcFilename to destFilename
func CopyFile(srcFilename, destFilename string, overwrite bool) (bool, error) {
	if !overwrite {
		if _, err := os.Stat(destFilename); err == nil {
			//log.Printf("skipping: %s already exists", filepath)
			return false, nil
		}
	}

	from, err := os.Open(srcFilename)
	if err != nil {
		return false, err
	}
	defer from.Close()
	to, err := os.Create(destFilename)
	if err != nil {
		return false, err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	return true, err
}
