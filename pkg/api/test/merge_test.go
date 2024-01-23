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
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func TestMergeCreateNew(t *testing.T) {
	msg := "TestMergeCreate"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}

	// Merge inFiles by concatenation in the order specified and write the result to outFile.
	// outFile will be overwritten.

	// Bookmarks for the merged document will be created/preserved per default (see config.yaml)

	outFile := filepath.Join(outDir, "out.pdf")
	if err := api.MergeCreateFile(inFiles, outFile, false, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Insert an empty page between merged files.
	outFile = filepath.Join(outDir, "outWithDivider.pdf")
	dividerPage := true
	if err := api.MergeCreateFile(inFiles, outFile, dividerPage, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestMergeCreateZipped(t *testing.T) {
	msg := "TestMergeCreateZipped"

	// The actual usecase for this is the recombination of 2 PDF files representing even and odd pages of some PDF source.
	// See #716
	inFile2 := filepath.Join(inDir, "adobe_errata.pdf")
	inFile1 := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "out.pdf")

	if err := api.MergeCreateZipFile(inFile1, inFile2, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestMergeAppendNew(t *testing.T) {
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

	// Bookmarks for the merged document will be created/preserved per default (see config.yaml)

	if err := api.MergeAppendFile(inFiles, outFile, false, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	anotherFile := filepath.Join(inDir, "testRot.pdf")
	err := api.MergeAppendFile([]string{anotherFile}, outFile, false, nil)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestMergeToBufNew(t *testing.T) {
	msg := "TestMergeToBuf"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}
	outFile := filepath.Join(outDir, "test.pdf")

	destFile := inFiles[0]
	inFiles = inFiles[1:]

	buf := &bytes.Buffer{}
	if err := api.Merge(destFile, inFiles, buf, nil, false); err != nil {
		t.Fatalf("%s: merge: %v\n", msg, err)
	}

	if err := os.WriteFile(outFile, buf.Bytes(), 0644); err != nil {
		t.Fatalf("%s: write: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestMergeRaw(t *testing.T) {
	msg := "TestMergeRaw"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}
	outFile := filepath.Join(outDir, "test.pdf")

	var rsc []io.ReadSeeker = make([]io.ReadSeeker, 2)

	f0, err := os.Open(inFiles[0])
	if err != nil {
		t.Fatalf("%s: open file1: %v\n", msg, err)
	}
	defer f0.Close()
	rsc[0] = f0

	f1, err := os.Open(inFiles[1])
	if err != nil {
		t.Fatalf("%s: open file2: %v\n", msg, err)
	}
	defer f1.Close()
	rsc[1] = f1

	buf := &bytes.Buffer{}
	if err := api.MergeRaw(rsc, buf, false, nil); err != nil {
		t.Fatalf("%s: merge: %v\n", msg, err)
	}

	if err := os.WriteFile(outFile, buf.Bytes(), 0644); err != nil {
		t.Fatalf("%s: write: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}
