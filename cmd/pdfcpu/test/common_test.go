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

package test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var pdfcpuBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "pdfcpu-cmd-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	pdfcpuBin = filepath.Join(dir, "pdfcpu")
	if runtime.GOOS == "windows" {
		pdfcpuBin += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", pdfcpuBin, "..")
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		os.Stderr.Write(out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func runPDFCPU(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command(pdfcpuBin, append([]string{"--conf", "disable"}, args...)...)
	cmd.Env = os.Environ()
	return cmd.CombinedOutput()
}

func runPDFCPUWithConfig(t *testing.T, configDir string, args ...string) ([]byte, []byte, error) {
	t.Helper()
	cmd := exec.Command(pdfcpuBin, append([]string{"--conf", configDir}, args...)...)
	cmd.Env = os.Environ()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func repoFile(t *testing.T, elems ...string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	args := append([]string{wd, "..", "..", ".."}, elems...)
	return filepath.Clean(filepath.Join(args...))
}
