//go:build aix || android || darwin || dragonfly || freebsd || ios || linux || netbsd || openbsd || solaris

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

package font

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteGobPublishesSharedReadableMode(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "Demo.gob")
	fd := validInstalledTTF("Demo", minimalSubsetFont(t))
	if err := writeGob(fileName, fd); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != installedFontMode {
		t.Fatalf("published font mode: got %04o, want %04o", got, installedFontMode)
	}
}
