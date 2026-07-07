/*
Copyright 2026 The pdfcpu Authors.

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

package api

import (
	"bytes"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func propertyTestInputFile() string {
	return filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
}

func copyPropertyTestInput(t *testing.T) string {
	t.Helper()

	b, err := os.ReadFile(propertyTestInputFile())
	if err != nil {
		t.Fatal(err)
	}
	inFile := filepath.Join(t.TempDir(), "in.pdf")
	if err := os.WriteFile(inFile, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return inFile
}

func readPropertyTestFile(t *testing.T, fileName string) map[string]string {
	t.Helper()

	f, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	properties, err := Properties(f, nil)
	if err != nil {
		t.Fatal(err)
	}
	return properties
}

type propertyIOTracker struct {
	used bool
}

func (r *propertyIOTracker) Read(_ []byte) (int, error) {
	r.used = true
	return 0, errors.New("unexpected property read")
}

func (r *propertyIOTracker) Seek(_ int64, _ int) (int64, error) {
	r.used = true
	return 0, errors.New("unexpected property seek")
}

var protectedPropertyNames = []string{"Keywords", "Producer", "CreationDate", "ModDate", "Trapped"}

// TestPropertyAPIArgumentErrors verifies public property boundary sentinels.
func TestPropertyAPIArgumentErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "list reader", err: func() error {
			_, err := Properties(nil, nil)
			return err
		}(), want: ErrMissingPDFReadSeeker},
		{name: "add reader", err: AddProperties(nil, io.Discard, nil, nil), want: ErrMissingPDFReadSeeker},
		{name: "add writer", err: AddProperties(bytes.NewReader(nil), nil, nil, nil), want: ErrMissingPDFWriter},
		{name: "remove reader", err: RemoveProperties(nil, io.Discard, nil, nil), want: ErrMissingPDFReadSeeker},
		{name: "remove writer", err: RemoveProperties(bytes.NewReader(nil), nil, nil, nil), want: ErrMissingPDFWriter},
		{name: "add file input", err: AddPropertiesFile("", "", nil, nil), want: ErrMissingPDFInput},
		{name: "remove file input", err: RemovePropertiesFile("", "", nil, nil), want: ErrMissingPDFInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, tt.err)
			}
		})
	}
}

// TestRemovePropertiesNoMatchPreservesSentinel verifies callers can detect a no-op removal.
func TestRemovePropertiesNoMatchPreservesSentinel(t *testing.T) {
	err := RemoveProperties(
		openAPITestPDF(t, propertyTestInputFile()),
		io.Discard,
		[]string{"pdfcpu-property-that-does-not-exist"},
		nil,
	)
	if !errors.Is(err, ErrNoPropertyRemoved) {
		t.Fatalf("expected %v, got %v", ErrNoPropertyRemoved, err)
	}
	if !strings.Contains(err.Error(), "remove properties: no property removed") {
		t.Fatalf("expected remove operation context, got %v", err)
	}
}

// TestPropertyAPIValidatesPropertiesBeforeReading verifies caller data is rejected before file I/O.
func TestPropertyAPIValidatesPropertiesBeforeReading(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "add stream", err: AddProperties(bytes.NewReader(nil), io.Discard, map[string]string{"": "value"}, nil), want: "add properties: validate properties"},
		{name: "remove stream", err: RemoveProperties(bytes.NewReader(nil), io.Discard, []string{" "}, nil), want: "remove properties: validate properties"},
		{name: "add file", err: AddPropertiesFile(filepath.Join(t.TempDir(), "missing.pdf"), "", map[string]string{"name": ""}, nil), want: "add properties: validate properties"},
		{name: "remove file", err: RemovePropertiesFile(filepath.Join(t.TempDir(), "missing.pdf"), "", []string{" "}, nil), want: "remove properties: validate properties"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil || !strings.Contains(tt.err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %v", tt.want, tt.err)
			}
			if errors.Is(tt.err, os.ErrNotExist) {
				t.Fatalf("file I/O occurred before property validation: %v", tt.err)
			}
		})
	}
}

// TestPropertyAPIRejectsProtectedNamesBeforeIO verifies every protected name is rejected at the API boundary.
func TestPropertyAPIRejectsProtectedNamesBeforeIO(t *testing.T) {
	for _, property := range protectedPropertyNames {
		t.Run(property, func(t *testing.T) {
			tests := []struct {
				name string
				run  func(*propertyIOTracker, string) error
			}{
				{name: "add stream", run: func(rs *propertyIOTracker, _ string) error {
					return AddProperties(rs, io.Discard, map[string]string{property: "value"}, nil)
				}},
				{name: "remove stream", run: func(rs *propertyIOTracker, _ string) error {
					return RemoveProperties(rs, io.Discard, []string{property}, nil)
				}},
				{name: "add file", run: func(_ *propertyIOTracker, missing string) error {
					return AddPropertiesFile(missing, "", map[string]string{property: "value"}, nil)
				}},
				{name: "remove file", run: func(_ *propertyIOTracker, missing string) error {
					return RemovePropertiesFile(missing, "", []string{property}, nil)
				}},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					rs := &propertyIOTracker{}
					err := tt.run(rs, filepath.Join(t.TempDir(), "missing.pdf"))
					if err == nil || !strings.Contains(err.Error(), "validate properties") ||
						!strings.Contains(err.Error(), "property name \""+property+"\" not allowed") {
						t.Fatalf("expected protected property validation for %q, got %v", property, err)
					}
					if rs.used {
						t.Fatal("stream I/O occurred before property validation")
					}
					if errors.Is(err, os.ErrNotExist) {
						t.Fatalf("file I/O occurred before property validation: %v", err)
					}
				})
			}
		})
	}
}

// TestPropertyAPIPrepareContextErrors verifies each stream entry point adds operation context.
func TestPropertyAPIPrepareContextErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "list", err: func() error {
			_, err := Properties(bytes.NewReader(nil), nil)
			return err
		}(), want: "list properties: prepare PDF context"},
		{name: "add", err: AddProperties(bytes.NewReader(nil), io.Discard, nil, nil), want: "add properties: prepare PDF context"},
		{name: "remove", err: RemoveProperties(bytes.NewReader(nil), io.Discard, nil, nil), want: "remove properties: prepare PDF context"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil || !strings.Contains(tt.err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %v", tt.want, tt.err)
			}
		})
	}
}

// TestAddPropertiesWriteErrorPreservesCause verifies write failures retain their cause and phase.
func TestAddPropertiesWriteErrorPreservesCause(t *testing.T) {
	want := errors.New("property writer failed")
	err := AddProperties(
		openAPITestPDF(t, propertyTestInputFile()),
		failingWriter{err: want},
		map[string]string{"name": "value"},
		nil,
	)
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
	if !strings.Contains(err.Error(), "add properties: write output") {
		t.Fatalf("expected write context, got %v", err)
	}
}

// TestRemovePropertiesWriteErrorPreservesCause verifies remove write failures retain their cause and phase.
func TestRemovePropertiesWriteErrorPreservesCause(t *testing.T) {
	inFile := copyPropertyTestInput(t)
	if err := AddPropertiesFile(inFile, "", map[string]string{"name": "value"}, nil); err != nil {
		t.Fatal(err)
	}

	want := errors.New("property writer failed")
	err := RemoveProperties(
		openAPITestPDF(t, inFile),
		failingWriter{err: want},
		[]string{"name"},
		nil,
	)
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
	if !strings.Contains(err.Error(), "remove properties: write output") {
		t.Fatalf("expected write context, got %v", err)
	}
}

// TestPropertyFileOpenAndCreateErrors verifies file setup failures retain their causes and phases.
func TestPropertyFileOpenAndCreateErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.pdf")
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "add open", err: AddPropertiesFile(missing, "", nil, nil), want: "add properties: open input"},
		{name: "remove open", err: RemovePropertiesFile(missing, "", nil, nil), want: "remove properties: open input"},
		{name: "add create", err: AddPropertiesFile(propertyTestInputFile(), filepath.Join(t.TempDir(), "missing", "out.pdf"), nil, nil), want: "add properties: create output"},
		{name: "remove create", err: RemovePropertiesFile(propertyTestInputFile(), filepath.Join(t.TempDir(), "missing", "out.pdf"), nil, nil), want: "remove properties: create output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, tt.err)
			}
			if !strings.Contains(tt.err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %v", tt.want, tt.err)
			}
		})
	}
}

// TestPropertyFileFailurePreservesExistingOutput verifies delayed replacement on processing failure.
func TestPropertyFileFailurePreservesExistingOutput(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "bad.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outFile, want, 0o600); err != nil {
		t.Fatal(err)
	}

	err := AddPropertiesFile(inFile, outFile, map[string]string{"name": "value"}, nil)
	if err == nil || !strings.Contains(err.Error(), "add properties: prepare PDF context") {
		t.Fatalf("expected prepare-context error, got %v", err)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

// TestPropertyFileSuccessReplacesExistingOutput verifies successful delayed destination replacement.
func TestPropertyFileSuccessReplacesExistingOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("existing output"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AddPropertiesFile(propertyTestInputFile(), outFile, map[string]string{"name": "value"}, nil); err != nil {
		t.Fatal(err)
	}
	if _, found := readPropertyTestFile(t, outFile)["name"]; !found {
		t.Fatal("replacement output is missing added property")
	}
}

// TestPropertyFileFailureRemovesNewOutput verifies failed processing removes a new destination.
func TestPropertyFileFailureRemovesNewOutput(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "bad.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(inFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	err := AddPropertiesFile(inFile, outFile, map[string]string{"name": "value"}, nil)
	if err == nil || !strings.Contains(err.Error(), "add properties: prepare PDF context") {
		t.Fatalf("expected prepare-context error, got %v", err)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected failed output cleanup, got %v", statErr)
	}
}

// TestAddPropertiesFileReplacesInput verifies successful in-place finalization.
func TestAddPropertiesFileReplacesInput(t *testing.T) {
	inFile := copyPropertyTestInput(t)
	if err := AddPropertiesFile(inFile, "", map[string]string{"name": "value"}, nil); err != nil {
		t.Fatal(err)
	}
	if _, found := readPropertyTestFile(t, inFile)["name"]; !found {
		t.Fatal("updated input is missing added property")
	}
}

// TestRemovePropertiesFileReplacesExistingOutput verifies delayed replacement for removal.
func TestRemovePropertiesFileReplacesExistingOutput(t *testing.T) {
	inFile := copyPropertyTestInput(t)
	if err := AddPropertiesFile(inFile, "", map[string]string{"name": "value"}, nil); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("existing output"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RemovePropertiesFile(inFile, outFile, []string{"name"}, nil); err != nil {
		t.Fatal(err)
	}
	if _, found := readPropertyTestFile(t, outFile)["name"]; found {
		t.Fatal("replacement output still contains removed property")
	}
}

// TestPropertyFileAliasDoesNotOverwriteInput verifies hard-linked output does not truncate the input.
func TestPropertyFileAliasDoesNotOverwriteInput(t *testing.T) {
	inFile := copyPropertyTestInput(t)
	outFile := filepath.Join(filepath.Dir(inFile), "alias.pdf")
	if err := os.Link(inFile, outFile); err != nil {
		t.Fatal(err)
	}
	if err := AddPropertiesFile(inFile, outFile, map[string]string{"name": "value"}, nil); err != nil {
		t.Fatal(err)
	}

	if _, found := readPropertyTestFile(t, inFile)["name"]; found {
		t.Fatal("input was overwritten through output alias")
	}
	if _, found := readPropertyTestFile(t, outFile)["name"]; !found {
		t.Fatal("output is missing added property")
	}
}

// TestAddPropertiesDoesNotMutateCallerMap verifies the API leaves caller-owned data unchanged.
func TestAddPropertiesDoesNotMutateCallerMap(t *testing.T) {
	properties := map[string]string{"name": " value "}
	want := maps.Clone(properties)
	var out bytes.Buffer
	if err := AddProperties(openAPITestPDF(t, propertyTestInputFile()), &out, properties, nil); err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(properties, want) {
		t.Fatalf("properties mutated: got %q, want %q", properties, want)
	}
}

// TestFinalizePropertyFilesReportsCloseErrors verifies close failures retain phase context.
// TestFinalizePropertyFilesReportsRenameError verifies replacement failures retain context.
