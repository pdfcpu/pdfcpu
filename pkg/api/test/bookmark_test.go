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
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
)

// Acrobat Reader "Bookmarks" = Mac Preview "Table of Contents".
// Mac Preview limitations: does not render color, style, outline tree collapsed by default.

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
}

func TestAddBookmarkTree2Levels(t *testing.T) {
	msg := "TestAddBookmarkTree2Levels"
	inFile := filepath.Join(inDir, "CenterOfWhy.pdf")
	outFile := filepath.Join("..", "..", "samples", "bookmarks", "bookmarkTree.pdf")

	bms := []pdfcpu.Bookmark{
		{PageFrom: 1, Title: "Page 1: Level 1", Color: &color.Green,
			Children: []pdfcpu.Bookmark{
				{PageFrom: 2, Title: "Page 2: Level 1.1"},
				{PageFrom: 3, Title: "Page 3: Level 1.2",
					Children: []pdfcpu.Bookmark{
						{PageFrom: 4, Title: "Page 4: Level 1.2.1"},
					}},
			}},
		{PageFrom: 5, Title: "Page 5: Level 2", Color: &color.Blue,
			Children: []pdfcpu.Bookmark{
				{PageFrom: 6, Title: "Page 6: Level 2.1"},
				{PageFrom: 7, Title: "Page 7: Level 2.2"},
				{PageFrom: 8, Title: "Page 8: Level 2.3"},
			}},
	}

	if err := api.AddBookmarksFile(inFile, outFile, bms, false, nil); err != nil {
		t.Fatalf("%s addBookmarks: %v\n", msg, err)
	}
}
