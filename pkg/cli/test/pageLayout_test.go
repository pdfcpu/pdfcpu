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

func TestPageLayout(t *testing.T) {
	msg := "testPageLayout"

	pageLayout := "TwoColumnLeft"

	inFile := filepath.Join(inDir, "test.pdf")
	outFile := filepath.Join(outDir, "test.pdf")

	cmd := cli.ListPageLayoutCommand(inFile, conf)
	ss, err := cli.Process(cmd)
	if err != nil {
		t.Fatalf("%s %s: list pageLayout: %v\n", msg, inFile, err)
	}
	if len(ss) > 0 && !strings.HasPrefix(ss[0], "No page layout") {
		t.Fatalf("%s %s: list pageLayout, unexpected: %s\n", msg, inFile, ss[0])
	}

	cmd = cli.SetPageLayoutCommand(inFile, outFile, pageLayout, nil)
	if _, err = cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: set pageLayout: %v\n", msg, outFile, err)
	}

	cmd = cli.ListPageLayoutCommand(outFile, conf)
	ss, err = cli.Process(cmd)
	if err != nil {
		t.Fatalf("%s %s: list pageLayout: %v\n", msg, outFile, err)
	}
	if len(ss) == 0 {
		t.Fatalf("%s %s: list pageLayout, missing page layout\n", msg, outFile)
	}
	if ss[0] != pageLayout {
		t.Fatalf("%s %s: list pageLayout, want:%s, got:%s\n", msg, outFile, pageLayout, ss[0])
	}

	cmd = cli.ResetPageLayoutCommand(outFile, "", nil)
	if _, err = cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: reset pageLayout: %v\n", msg, outFile, err)
	}

	cmd = cli.ListPageLayoutCommand(outFile, conf)
	ss, err = cli.Process(cmd)
	if err != nil {
		t.Fatalf("%s %s: list pageLayout: %v\n", msg, outFile, err)
	}
	if len(ss) > 0 && !strings.HasPrefix(ss[0], "No page layout") {
		t.Fatalf("%s %s: list pageLayout, unexpected: %s\n", msg, outFile, ss[0])
	}
}
