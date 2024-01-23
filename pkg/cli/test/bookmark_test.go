/*
	Copyright 2023 The pdfcpu Authors.

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
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
)

func TestListBookmarks(t *testing.T) {
	msg := "TestListBookmarks"
	inDir := filepath.Join("..", "..", "samples", "bookmarks")
	inFile := filepath.Join(inDir, "bookmarkTree.pdf")

	cmd := cli.ListBookmarksCommand(inFile, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestExportBookmarks(t *testing.T) {
	msg := "TestExportBookmarks"
	inDir := filepath.Join("..", "..", "samples", "bookmarks")
	inFile := filepath.Join(inDir, "bookmarkTree.pdf")
	outFile := filepath.Join(outDir, "bookmarkTree.json")

	cmd := cli.ExportBookmarksCommand(inFile, outFile, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestImportBookmarks(t *testing.T) {
	msg := "TestImportBookmarks"
	inDir := filepath.Join("..", "..", "samples", "bookmarks")
	inFile := filepath.Join(inDir, "bookmarkTree.pdf")
	inFileJSON := filepath.Join(inDir, "bookmarkTree.json")
	outFile := filepath.Join(outDir, "bookmarkTreeImported.pdf")

	replace := true
	cmd := cli.ImportBookmarksCommand(inFile, inFileJSON, outFile, replace, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := api.ImportBookmarksFile(inFile, inFileJSON, outFile, replace, nil); err != nil {
		t.Fatalf("%s importBookmarks: %v\n", msg, err)
	}

	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestRemoveBookmarks(t *testing.T) {
	msg := "TestRemoveBookmarks"
	inDir := filepath.Join("..", "..", "samples", "bookmarks")
	inFile := filepath.Join(inDir, "bookmarkTree.pdf")
	outFile := filepath.Join(outDir, "bookmarkTreeNoBookmarks.pdf")

	cmd := cli.RemoveBookmarksCommand(inFile, outFile, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}
