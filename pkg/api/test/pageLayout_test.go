/*
Copyright 2023 The pdf Authors.

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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestPageLayout(t *testing.T) {
	msg := "testPageLayout"

	fileName := "test.pdf"
	inFile := filepath.Join(outDir, fileName)
	copyFile(t, filepath.Join(inDir, fileName), inFile)

	pageLayout := model.PageLayoutTwoColumnLeft

	pl, err := api.PageLayoutFile(inFile, nil)
	if err != nil {
		t.Fatalf("%s %s: list pageLayout: %v\n", msg, inFile, err)
	}
	if pl != nil {
		t.Fatalf("%s %s: list pageLayout, unexpected: %s\n", msg, inFile, pl)
	}

	if err := api.SetPageLayoutFile(inFile, "", pageLayout, nil); err != nil {
		t.Fatalf("%s %s: set pageLayout: %v\n", msg, inFile, err)
	}

	pl, err = api.PageLayoutFile(inFile, nil)
	if err != nil {
		t.Fatalf("%s %s: list pageLayout: %v\n", msg, inFile, err)
	}
	if pl == nil {
		t.Fatalf("%s %s: list pageLayout, missing page layout\n", msg, inFile)
	}
	if *pl != pageLayout {
		t.Fatalf("%s %s: list pageLayout, want:%s, got:%s\n", msg, inFile, pageLayout.String(), pl.String())
	}

	if err := api.ResetPageLayoutFile(inFile, "", nil); err != nil {
		t.Fatalf("%s %s: reset pageLayout: %v\n", msg, inFile, err)
	}

	pl, err = api.PageLayoutFile(inFile, nil)
	if err != nil {
		t.Fatalf("%s %s: list page layout: %v\n", msg, inFile, err)
	}
	if pl != nil {
		t.Fatalf("%s %s: list page layout, unexpected: %s\n", msg, inFile, pl)
	}
}
