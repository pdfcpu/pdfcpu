/*
Copyright 2019 The pdf Authors.

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

	"github.com/pdfcpu/pdfcpu/pkg/cli"
)

func TestPortfolioCommand(t *testing.T) {
	msg := "TestPortfolioCommand"

	if err := prepareForAttachmentTest(t); err != nil {
		t.Fatalf("%s prepare for attachments: %v\n", msg, err)
	}

	fileName := filepath.Join(outDir, "go.pdf")

	// # of portfolio entries must be 0
	listAttachments(t, msg, fileName, 0)

	// attach add 4 portfolio entries including descriptions.
	files := []string{
		filepath.Join(outDir, "golang.pdf"),
		filepath.Join(outDir, "T4.pdf") + ", CCITT spec",
		filepath.Join(outDir, "go-lecture.pdf"),
		filepath.Join(outDir, "test.wav") + ", test audio file"}

	cmd := cli.AddAttachmentsPortfolioCommand(fileName, "", files, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s add portfolio entries: %v\n", msg, err)
	}

	// List portfolio entries.
	list := listAttachments(t, msg, fileName, 4)
	for _, s := range list {
		t.Log(s)
	}

	// Extract all portfolio entries.
	cmd = cli.ExtractAttachmentsCommand(fileName, outDir, nil, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s extract all portfolio entries: %v\n", msg, err)
	}

	// Extract 1 portfolio entry.
	cmd = cli.ExtractAttachmentsCommand(fileName, outDir, []string{"golang.pdf"}, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s extract one portfolio entry: %v\n", msg, err)
	}

	// Remove 1 portfolio entry.
	cmd = cli.RemoveAttachmentsCommand(fileName, "", []string{"golang.pdf"}, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s remove one portfolio entry: %v\n", msg, err)
	}
	listAttachments(t, msg, fileName, 3)

	// Remove all portfolio entries.
	cmd = cli.RemoveAttachmentsCommand(fileName, "", nil, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s remove all portfolio entries: %v\n", msg, err)
	}
	listAttachments(t, msg, fileName, 0)

	// Validate the processed file.
	if err := validateFile(t, fileName, nil); err != nil {
		t.Fatalf("%s: validate: %v\n", msg, err)
	}
}
