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

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func localImageName(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(name, "/") || strings.Contains(name, ":") || strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("local image %q: expected a relative path: %w", name, os.ErrPermission)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return "", fmt.Errorf("local image %q: parent-directory components are forbidden: %w", name, os.ErrPermission)
		}
	}
	return filepath.FromSlash(name), nil
}

func localImageError(name string, err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return err
	}
	return fmt.Errorf("local image %q: access denied by LocalImageRoot: %w: %w", name, os.ErrPermission, err)
}

func openLocalImage(root *os.Root, name string) (*os.File, error) {
	info, err := root.Stat(name)
	if err != nil {
		return nil, localImageError(name, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("local image %q: expected a regular file: %w", name, os.ErrPermission)
	}
	// Nonblocking opens on Unix prevent a replacement FIFO from blocking between Stat and OpenFile.
	f, err := root.OpenFile(name, localImageOpenFlags, 0)
	if err != nil {
		return nil, localImageError(name, err)
	}
	info, err = f.Stat()
	if err != nil {
		return nil, errors.Join(localImageError(name, err), f.Close())
	}
	if !info.Mode().IsRegular() {
		return nil, errors.Join(
			fmt.Errorf("local image %q: expected a regular file: %w", name, os.ErrPermission),
			f.Close(),
		)
	}
	return f, nil
}

func (ib *ImageBox) localResource() (io.ReadCloser, error) {
	if ib.Src == "" {
		return nil, nil
	}
	if ib.pdf == nil || ib.pdf.Conf == nil || ib.pdf.Conf.LocalImageRoot == "" {
		return nil, fmt.Errorf("local images require LocalImageRoot: %w", os.ErrPermission)
	}
	// Go's js implementation of os.Root cannot prevent symlink races.
	if runtime.GOOS == "js" {
		return nil, fmt.Errorf("confined local images are unavailable on js: %w", os.ErrPermission)
	}
	name, err := localImageName(ib.Src)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(ib.pdf.Conf.LocalImageRoot)
	if err != nil {
		return nil, fmt.Errorf("open LocalImageRoot: %w: %w", os.ErrPermission, err)
	}
	defer root.Close()
	f, err := openLocalImage(root, name)
	if err != nil {
		return nil, err
	}
	return f, nil
}
