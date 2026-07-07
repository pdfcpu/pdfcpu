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

package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestExportFormFieldsFailurePreservesExistingOutput verifies staged CLI JSON publication.
func TestExportFormFieldsFailurePreservesExistingOutput(t *testing.T) {
	useStdin(t, "not a pdf")
	outFile := filepath.Join(t.TempDir(), "form.json")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0640); err != nil {
		t.Fatal(err)
	}

	_, err := ExportFormFields(ExportFormCommand("-", outFile, nil))
	if err == nil {
		t.Fatal("expected export failure")
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

func TestListFormFieldsCLIUsesPreparedContextErrorChain(t *testing.T) {
	_, err := listFormFields(strings.NewReader("not a PDF"), nil)
	if err == nil || !strings.Contains(err.Error(), "list form fields: prepare PDF context") {
		t.Fatalf("expected prepared context error, got %v", err)
	}
	if got := strings.Count(err.Error(), "prepare PDF context"); got != 1 {
		t.Fatalf("expected prepare PDF context once, got %d in %q", got, err.Error())
	}
}

func TestListFormFieldsJSONErrorsNameInvokedCommand(t *testing.T) {
	_, err := exportFormGroup(strings.NewReader("not a PDF"), "source.pdf", nil)
	want := "list form fields: export data: export form: prepare PDF context"
	if err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("expected %q prefix, got %v", want, err)
	}
	if got := strings.Count(err.Error(), "list form fields"); got != 1 {
		t.Fatalf("expected list form fields once, got %d in %q", got, err.Error())
	}
}

func TestListFormFieldsFileErrorIncludesInputPath(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "missing.pdf")
	_, err := ListFormFieldsFile([]string{inFile}, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "list form fields: open input "+inFile) {
		t.Fatalf("expected input path context, got %v", err)
	}
}

func TestFillFormCLIDataOpenErrorIncludesPath(t *testing.T) {
	inFileData := filepath.Join(t.TempDir(), "missing.json")
	cmd := FillFormCommand("-", inFileData, "-", model.NewDefaultConfiguration())
	_, err := FillFormFields(cmd)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "fill form: open form data "+inFileData) {
		t.Fatalf("expected form data path context, got %v", err)
	}
}

func TestMultiFillFormCLIRejectsUnmergedStdout(t *testing.T) {
	cmd := MultiFillFormCommand("unused.pdf", "unused.json", "", "-", false, nil)
	_, err := multiFillFormFieldsToStdout(cmd, "unused.pdf")
	if err == nil || !strings.Contains(err.Error(), "multi-fill form: stdout requires merge mode") {
		t.Fatalf("expected merge mode error, got %v", err)
	}
}
