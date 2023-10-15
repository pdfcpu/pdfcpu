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

func TestPageMode(t *testing.T) {
	msg := "testPageMode"

	fileName := "test.pdf"
	inFile := filepath.Join(outDir, fileName)
	copyFile(t, filepath.Join(inDir, fileName), inFile)

	pageMode := model.PageModeUseOutlines

	pl, err := api.PageModeFile(inFile, nil)
	if err != nil {
		t.Fatalf("%s %s: list pageMode: %v\n", msg, inFile, err)
	}
	if pl != nil {
		t.Fatalf("%s %s: list pageMode, unexpected: %s\n", msg, inFile, pl)
	}

	if err := api.SetPageModeFile(inFile, "", pageMode, nil); err != nil {
		t.Fatalf("%s %s: set pageMode: %v\n", msg, inFile, err)
	}

	pm, err := api.PageModeFile(inFile, nil)
	if err != nil {
		t.Fatalf("%s %s: list pageMode: %v\n", msg, inFile, err)
	}
	if pm == nil {
		t.Fatalf("%s %s: list pageMode, missing page mode\n", msg, inFile)
	}
	if *pm != pageMode {
		t.Fatalf("%s %s: list pageMode, want:%s, got:%s\n", msg, inFile, pageMode.String(), pm.String())
	}

	if err := api.ResetPageModeFile(inFile, "", nil); err != nil {
		t.Fatalf("%s %s: reset pageMode: %v\n", msg, inFile, err)
	}

	pl, err = api.PageModeFile(inFile, nil)
	if err != nil {
		t.Fatalf("%s %s: list pageMode: %v\n", msg, inFile, err)
	}
	if pl != nil {
		t.Fatalf("%s %s: list pageMode, unexpected: %s\n", msg, inFile, pl)
	}
}
