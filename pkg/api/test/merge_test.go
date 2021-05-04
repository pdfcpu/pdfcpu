/*
Copyright 2020 The pdf Authors.

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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func TestMergeCreate(t *testing.T) {
	msg := "TestMergeCreate"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}
	outFile := filepath.Join(outDir, "test.pdf")

	// Merge inFiles by concatenation in the order specified and write the result to outFile.
	// outFile will be overwritten.
	if err := api.MergeCreateFile(inFiles, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestMergeAppend(t *testing.T) {
	msg := "TestMergeAppend"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}
	outFile := filepath.Join(outDir, "test.pdf")
	if err := copyFile(t, filepath.Join(inDir, "test.pdf"), outFile); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Merge inFiles by concatenation in the order specified and write the result to outFile.
	// If outFile already exists its content will be preserved and serves as the beginning of the merge result.
	if err := api.MergeAppendFile(inFiles, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestMergeToBuf(t *testing.T) {
	msg := "TestMergeToBuf"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}
	outFile := filepath.Join(outDir, "test.pdf")

	ff := []*os.File(nil)
	for _, f := range inFiles {
		f, err := os.Open(f)
		if err != nil {
			t.Fatalf("%s: open: %v\n", msg, err)
		}
		ff = append(ff, f)
	}

	rs := make([]io.ReadSeeker, len(ff))
	for i, f := range ff {
		rs[i] = f
	}

	buf := &bytes.Buffer{}
	if err := api.Merge(rs, buf, nil); err != nil {
		t.Fatalf("%s: merge: %v\n", msg, err)
	}

	if err := ioutil.WriteFile(outFile, buf.Bytes(), 0644); err != nil {
		t.Fatalf("%s: write: %v\n", msg, err)
	}
}
