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
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/cli"
)

func TestViewerPreferences(t *testing.T) {
	msg := "testViewerPreferences"

	fileName := "Hybrid-PDF.pdf"
	inFile := filepath.Join(outDir, fileName)
	copyFile(t, filepath.Join(inDir, fileName), inFile)
	inFileJSON := filepath.Join(inDir, "json", "viewerPreferences.json")
	stringJSON := "{\"HideMenuBar\": true, \"CenterWindow\": true}"

	all, json := false, false
	cmd := cli.ListViewerPreferencesCommand(inFile, all, json, nil)
	ss, err := cli.Process(cmd)
	if err != nil {
		t.Fatalf("%s %s: list viewer preferences: %v\n", msg, inFile, err)
	}
	if len(ss) > 0 && strings.HasPrefix(ss[0], "No viewer preferences") {
		t.Fatalf("%s %s: missing viewer preferences\n", msg, inFile)
	}

	cmd = cli.ResetViewerPreferencesCommand(inFile, "", nil)
	if _, err = cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: reset viewer preferences: %v\n", msg, inFile, err)
	}

	cmd = cli.ListViewerPreferencesCommand(inFile, all, json, nil)
	ss, err = cli.Process(cmd)
	if err != nil {
		t.Fatalf("%s %s: list viewer preferences: %v\n", msg, inFile, err)
	}
	if len(ss) > 0 && !strings.HasPrefix(ss[0], "No viewer preferences") {
		t.Fatalf("%s %s: unexpected viewer preferences\n", msg, inFile)
	}

	cmd = cli.SetViewerPreferencesCommand(inFile, inFileJSON, "", "", nil)
	if _, err = cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: set viewer preferences from JSON file: %v\n", msg, inFile, err)
	}

	cmd = cli.ListViewerPreferencesCommand(inFile, all, json, nil)
	ss, err = cli.Process(cmd)
	if err != nil {
		t.Fatalf("%s %s: list viewer preferences: %v\n", msg, inFile, err)
	}
	if len(ss) > 0 && strings.HasPrefix(ss[0], "No viewer preferences") {
		t.Fatalf("%s %s: missing viewer preferences\n", msg, inFile)
	}

	cmd = cli.SetViewerPreferencesCommand(inFile, "", "", stringJSON, nil)
	if _, err = cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: set viewer preferences from JSON string: %v\n", msg, inFile, err)
	}
}
