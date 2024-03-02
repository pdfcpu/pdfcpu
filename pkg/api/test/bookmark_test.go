/*
	Copyright 2020 The pdfcpu Authors.

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
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Acrobat Reader "Bookmarks" = Mac Preview "Table of Contents".
// Mac Preview limitations: does not render color, style, outline tree collapsed by default.

func listBookmarksFile(t *testing.T, fileName string, conf *model.Configuration) ([]string, error) {
	t.Helper()

	msg := "listBookmarks"

	f, err := os.Open(fileName)
	if err != nil {
		t.Fatalf("%s open: %v\n", msg, err)
	}
	defer f.Close()

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTBOOKMARKS

	ctx, err := api.ReadValidateAndOptimize(f, conf)
	if err != nil {
		t.Fatalf("%s ReadValidateAndOptimize: %v\n", msg, err)
	}

	return pdfcpu.BookmarkList(ctx)
}

func TestListBookmarks(t *testing.T) {
	msg := "TestListBookmarks"
	inDir := filepath.Join("..", "..", "samples", "bookmarks")
	inFile := filepath.Join(inDir, "bookmarkTree.pdf")

	if _, err := listBookmarksFile(t, inFile, nil); err != nil {
		t.Fatalf("%s list bookmarks: %v\n", msg, err)
	}
}

func InactiveTestAddDuplicateBookmarks(t *testing.T) {
	msg := "TestAddDuplicateBookmarks"
	inFile := filepath.Join(inDir, "CenterOfWhy.pdf")
	outFile := filepath.Join("..", "..", "samples", "bookmarks", "bookmarkDuplicates.pdf")

	bms := []pdfcpu.Bookmark{
		{PageFrom: 2, Title: "Duplicate Name"},
		{PageFrom: 3, Title: "Duplicate Name"},
		{PageFrom: 5, Title: "Duplicate Name"},
	}

	replace := true // Replace existing bookmarks.
	if err := api.AddBookmarksFile(inFile, outFile, bms, replace, nil); err != nil {
		t.Fatalf("%s addBookmarks: %v\n", msg, err)
	}
	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestAddSimpleBookmarks(t *testing.T) {
	msg := "TestAddSimpleBookmarks"
	inFile := filepath.Join(inDir, "CenterOfWhy.pdf")
	outFile := filepath.Join("..", "..", "samples", "bookmarks", "bookmarkSimple.pdf")

	bookmarkColor := color.NewSimpleColor(0xab6f30)

	// TODO Emoji support!

	bms := []pdfcpu.Bookmark{
		{PageFrom: 1, Title: "Page 1: Applicant’s Form"},
		{PageFrom: 2, Title: "Page 2: Bold 这是一个测试", Bold: true},
		{PageFrom: 3, Title: "Page 3: Italic 测试 尾巴", Italic: true, Bold: true},
		{PageFrom: 4, Title: "Page 4: Bold & Italic", Bold: true, Italic: true},
		{PageFrom: 16, Title: "Page 16: The birthday of Smalltalk", Color: &bookmarkColor},
		{PageFrom: 17, Title: "Page 17: Gray", Color: &color.Gray},
		{PageFrom: 18, Title: "Page 18: Red", Color: &color.Red},
		{PageFrom: 19, Title: "Page 19: Bold Red", Color: &color.Red, Bold: true},
	}

	replace := true // Replace existing bookmarks.
	if err := api.AddBookmarksFile(inFile, outFile, bms, replace, nil); err != nil {
		t.Fatalf("%s addBookmarks: %v\n", msg, err)
	}
	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestAddBookmarkTree2Levels(t *testing.T) {
	msg := "TestAddBookmarkTree2Levels"
	inFile := filepath.Join(inDir, "CenterOfWhy.pdf")
	outFile := filepath.Join("..", "..", "samples", "bookmarks", "bookmarkTree.pdf")

	bms := []pdfcpu.Bookmark{
		{PageFrom: 1, Title: "Page 1: Level 1", Color: &color.Green,
			Kids: []pdfcpu.Bookmark{
				{PageFrom: 2, Title: "Page 2: Level 1.1"},
				{PageFrom: 3, Title: "Page 3: Level 1.2",
					Kids: []pdfcpu.Bookmark{
						{PageFrom: 4, Title: "Page 4: Level 1.2.1"},
					}},
			}},
		{PageFrom: 5, Title: "Page 5: Level 2", Color: &color.Blue,
			Kids: []pdfcpu.Bookmark{
				{PageFrom: 6, Title: "Page 6: Level 2.1"},
				{PageFrom: 7, Title: "Page 7: Level 2.2"},
				{PageFrom: 8, Title: "Page 8: Level 2.3"},
			}},
	}

	if err := api.AddBookmarksFile(inFile, outFile, bms, false, nil); err != nil {
		t.Fatalf("%s addBookmarks: %v\n", msg, err)
	}
	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestRemoveBookmarks(t *testing.T) {
	msg := "TestRemoveBookmarks"
	inDir := filepath.Join("..", "..", "samples", "bookmarks")
	inFile := filepath.Join(inDir, "bookmarkTree.pdf")
	outFile := filepath.Join(inDir, "bookmarkTreeNoBookmarks.pdf")

	if err := api.RemoveBookmarksFile(inFile, outFile, nil); err != nil {
		t.Fatalf("%s removeBookmarks: %v\n", msg, err)
	}
	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestExportBookmarks(t *testing.T) {
	msg := "TestExportBookmarks"
	inDir := filepath.Join("..", "..", "samples", "bookmarks")
	inFile := filepath.Join(inDir, "bookmarkTree.pdf")
	outFile := filepath.Join(inDir, "bookmarkTree.json")

	if err := api.ExportBookmarksFile(inFile, outFile, nil); err != nil {
		t.Fatalf("%s export bookmarks: %v\n", msg, err)
	}
}

func TestImportBookmarks(t *testing.T) {
	msg := "TestImportBookmarks"
	inDir := filepath.Join("..", "..", "samples", "bookmarks")
	inFile := filepath.Join(inDir, "bookmarkTree.pdf")
	inFileJSON := filepath.Join(inDir, "bookmarkTree.json")
	outFile := filepath.Join(inDir, "bookmarkTreeImported.pdf")

	replace := true
	if err := api.ImportBookmarksFile(inFile, inFileJSON, outFile, replace, nil); err != nil {
		t.Fatalf("%s importBookmarks: %v\n", msg, err)
	}
	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}
