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
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestMergeCreateNew(t *testing.T) {
	msg := "TestMergeCreate"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}
	outFile := filepath.Join(outDir, "out.pdf")

	// Merge inFiles by concatenation in the order specified and write the result to outFile.
	// outFile will be overwritten.

	// Bookmarks for the merged document will be created/preserved per default (see config.yaml)
	conf := model.NewDefaultConfiguration()
	//conf.CreateBookmarks = false

	if err := api.MergeCreateFile(inFiles, outFile, conf); err != nil {
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
	if err := api.MergeAppendFile(inFiles, outFile, nil); err != nil {
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
	if err := api.Merge(destFile, inFiles, buf, nil); err != nil {
		t.Fatalf("%s: merge: %v\n", msg, err)
	}

	if err := os.WriteFile(outFile, buf.Bytes(), 0644); err != nil {
		t.Fatalf("%s: write: %v\n", msg, err)
	}
}
