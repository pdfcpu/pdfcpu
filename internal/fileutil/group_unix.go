//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

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
	"errors"
	"fmt"
	"os"
	"syscall"
)

// PreserveGroup gives a staged source the existing destination's group without changing ordinary permission bits.
// A missing destination leaves the source's inherited group unchanged.
func PreserveGroup(source, destination string) error {
	return preserveGroup(source, destination, os.Chown)
}

func preserveGroup(source, destination string, chown func(string, int, int) error) error {
	dst, err := os.Stat(destination)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	src, err := os.Stat(source)
	if err != nil {
		return err
	}
	srcStat, srcOK := src.Sys().(*syscall.Stat_t)
	dstStat, dstOK := dst.Sys().(*syscall.Stat_t)
	if !srcOK || !dstOK {
		return fmt.Errorf("preserve group for %s: unavailable filesystem ownership", destination)
	}
	if srcStat.Gid == dstStat.Gid {
		return nil
	}
	if err := chown(source, -1, int(dstStat.Gid)); err != nil {
		return fmt.Errorf("preserve group for %s: %w", destination, err)
	}
	return nil
}
