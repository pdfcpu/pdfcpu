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

package primitives

// GHSA-44h9-p34h-2w4p: prevent local image path and symlink escapes before reading file contents.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-44h9-p34h-2w4p

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestGHSA44h9P34h2w4pSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	for _, dir := range []string{root, outside} {
		if err := os.WriteFile(filepath.Join(dir, "image.bin"), []byte("harmless"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for name, target := range map[string]string{
		"inside":    "image.bin",
		"outside":   filepath.Join(outside, "image.bin"),
		"directory": outside,
	} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}
	conf := model.NewStatelessConfiguration()
	conf.LocalImageRoot = root
	for _, tc := range []struct {
		name string
		deny bool
	}{{"inside", false}, {"outside", true}, {"directory/image.bin", true}} {
		t.Run(tc.name, func(t *testing.T) {
			ib := ImageBox{Src: tc.name, pdf: &PDF{Conf: conf}}
			f, err := ib.resource()
			if tc.deny {
				if f != nil {
					f.Close()
					t.Fatal("denied resource returned an open file")
				}
				if !errors.Is(err, os.ErrPermission) {
					t.Fatalf("want denial, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			data, err := io.ReadAll(f)
			if err != nil || string(data) != "harmless" {
				t.Fatalf("read: %q, %v", data, err)
			}
		})
	}
}
