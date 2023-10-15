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

func TestViewerPreferences(t *testing.T) {
	msg := "testViewerPreferences"

	fileName := "Hybrid-PDF.pdf"
	inFile := filepath.Join(outDir, fileName)
	copyFile(t, filepath.Join(inDir, fileName), inFile)
	inFileJSON := filepath.Join(inDir, "json", "viewerPreferences.json")
	stringJSON := "{\"HideMenuBar\": true, \"CenterWindow\": true}"

	vp, err := api.ViewerPreferencesFile(inFile, false, nil)
	if err != nil {
		t.Fatalf("%s %s: viewerPref struct: %v\n", msg, inFile, err)
	}
	if vp == nil {
		t.Fatalf("%s %s: missing viewerPref struct\n", msg, inFile)
	}

	if err := api.ResetViewerPreferencesFile(inFile, "", nil); err != nil {
		t.Fatalf("%s %s: reset: %v\n", msg, inFile, err)
	}

	vp, err = api.ViewerPreferencesFile(inFile, false, nil)
	if err != nil {
		t.Fatalf("%s %s: viewerPref struct: %v\n", msg, inFile, err)
	}
	if vp != nil {
		t.Fatalf("%s %s: unexpected viewerPref struct: %v\n", msg, inFile, vp)
	}

	if err := api.SetViewerPreferencesFileFromJSONFile(inFile, "", inFileJSON, nil); err != nil {
		t.Fatalf("%s %s: set via JSON file: %v\n", msg, inFile, err)
	}

	vp, err = api.ViewerPreferencesFile(inFile, false, nil)
	if err != nil {
		t.Fatalf("%s %s: viewerPref struct: %v\n", msg, inFile, err)
	}
	if vp == nil {
		t.Fatalf("%s %s: missing viewerPref struct\n", msg, inFile)
	}

	vp = &model.ViewerPreferences{}
	vp.SetCenterWindow(true)
	vp.SetHideMenuBar(true)
	vp.SetNumCopies(5)

	if err := api.SetViewerPreferencesFile(inFile, "", *vp, nil); err != nil {
		t.Fatalf("%s %s: set: %v\n", msg, inFile, err)
	}

	if err := api.SetViewerPreferencesFileFromJSONBytes(inFile, "", []byte(stringJSON), nil); err != nil {
		t.Fatalf("%s %s: set via JSON string: %v\n", msg, inFile, err)
	}
}
