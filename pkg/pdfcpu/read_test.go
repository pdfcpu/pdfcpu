/*
Copyright 2024 The pdfcpu Authors.

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
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileContext(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "test.pdf")

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	if doc, err := ReadFileWithContext(ctx, inFile, nil); err == nil {
		t.Errorf("reading should have failed, got %+v", doc)
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("should have failed with timeout, got %s", err)
	}
}

func TestReadContext(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "test.pdf")

	fp, err := os.Open(inFile)
	if err != nil {
		t.Fatal(err)
	}
	defer fp.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	if doc, err := ReadWithContext(ctx, fp, nil); err == nil {
		t.Errorf("reading should have failed, got %+v", doc)
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("should have failed with timeout, got %s", err)
	}
}
