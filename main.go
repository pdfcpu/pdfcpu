package main

import (
	"fmt"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"strings"
)

func printBookmarkAndChildren(bookmark pdfcpu.Bookmark, level int) {
	fmt.Printf("%s%d-%d | %s\n", strings.Repeat("\t", level), bookmark.PageFrom, bookmark.PageThru, bookmark.Title)
	for _, child := range bookmark.Children {
		printBookmarkAndChildren(child, level+1)
	}
}

func main() {
	pdfCtx, err := pdfcpu.ReadFile("demo.pdf", pdfcpu.NewDefaultConfiguration())
	if err != nil {
		fmt.Print(err)
	}
	bookmarks, err := pdfCtx.BookmarksForOutline()
	if err != nil {
		fmt.Print(err)
	}
	for _, bookmark := range bookmarks {
		printBookmarkAndChildren(bookmark, 0)
	}
}

