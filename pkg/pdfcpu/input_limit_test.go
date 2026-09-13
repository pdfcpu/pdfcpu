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

package pdfcpu

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestReadInputLimit verifies per-PDF size enforcement before parsing and inclusive boundaries.
func TestReadInputLimit(t *testing.T) {
	data, err := os.ReadFile("../testdata/test.pdf")
	if err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int64{0, int64(len(data)), int64(len(data)) + 1, int64(len(data)) - 1, -1} {
		conf := model.NewStatelessConfiguration()
		conf.Limits.MaxInputBytes = limit
		_, err := Read(t.Context(), bytes.NewReader(data), conf)
		switch {
		case limit < 0:
			if err == nil {
				t.Fatal("accepted negative input limit")
			}
		case limit == int64(len(data))-1:
			if !errors.Is(err, model.ErrInputSizeLimit) {
				t.Fatalf("got %v, want size limit", err)
			}
		default:
			if err != nil {
				t.Fatalf("limit %d: %v", limit, err)
			}
		}
	}
	conf := model.NewStatelessConfiguration()
	conf.Limits.MaxInputBytes = 1
	_, err = Read(t.Context(), bytes.NewReader([]byte("invalid PDF")), conf)
	if !errors.Is(err, model.ErrInputSizeLimit) {
		t.Fatalf("parsed oversized input: %v", err)
	}
}
