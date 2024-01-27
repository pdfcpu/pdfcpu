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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestReadFileContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	conf := model.NewDefaultConfiguration()
	if doc, err := ReadFileWithContext(ctx, "../samples/basic/test.pdf", conf); err == nil {
		t.Errorf("reading should have failed, got %+v", doc)
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("should have failed with timeout, got %s", err)
	}
}

func TestReadContext(t *testing.T) {
	fp, err := os.Open("../samples/basic/test.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer fp.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	conf := model.NewDefaultConfiguration()
	if doc, err := ReadWithContext(ctx, fp, conf); err == nil {
		t.Errorf("reading should have failed, got %+v", doc)
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("should have failed with timeout, got %s", err)
	}
}
