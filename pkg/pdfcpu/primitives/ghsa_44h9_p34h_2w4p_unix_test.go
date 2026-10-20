//go:build unix

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

// GHSA-44h9-p34h-2w4p: reject FIFOs before a blocking image open.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-44h9-p34h-2w4p

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestGHSA_44h9_p34h_2w4p_FIFO(t *testing.T) {
	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "pipe"), 0600); err != nil {
		t.Fatal(err)
	}
	conf := model.NewStatelessConfiguration()
	conf.LocalImageRoot = root
	done := make(chan error, 1)
	go func() {
		ib := ImageBox{Src: "pipe", pdf: &PDF{Conf: conf}}
		f, err := ib.resource()
		if f != nil {
			f.Close()
		}
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, os.ErrPermission) {
			t.Fatalf("want denial, got %v", err)
		}
	case <-time.After(3 * time.Second):
		// Unblock a regressed FIFO open so the test does not leave a blocked goroutine.
		fd, err := syscall.Open(filepath.Join(root, "pipe"), syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err == nil {
			syscall.Close(fd)
		}
		t.Fatal("local image open blocked on FIFO")
	}
}
