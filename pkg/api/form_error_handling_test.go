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
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func formTestInputFile() string {
	return filepath.Join("..", "samples", "form", "demoSinglePage", "english.pdf")
}

func formMutationTestInputFile() string {
	return filepath.Join("..", "samples", "form", "primitives", "textfield.pdf")
}

func changedFormTestJSON(t *testing.T) []byte {
	t.Helper()
	formGroup, err := ExportForm(openAPITestPDF(t, formMutationTestInputFile()), "textfield.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(formGroup.Forms) == 0 || len(formGroup.Forms[0].TextFields) == 0 {
		t.Fatal("expected exported text field")
	}
	formGroup.Forms[0].TextFields[0].Value += " changed"
	bb, err := json.Marshal(formGroup)
	if err != nil {
		t.Fatal(err)
	}
	return bb
}

func changedFormTestCSV(t *testing.T) string {
	t.Helper()
	formGroup, err := ExportForm(openAPITestPDF(t, formMutationTestInputFile()), "textfield.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(formGroup.Forms) == 0 || len(formGroup.Forms[0].TextFields) == 0 {
		t.Fatal("expected exported text field")
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write([]string{formGroup.Forms[0].TextFields[0].ID}); err != nil {
		t.Fatal(err)
	}
	if err := w.Write([]string{"changed"}); err != nil {
		t.Fatal(err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func multiFillFormWithWriteContext(inFilePDF string, rd io.Reader, outDir, fileName string, format form.DataFormat, merge bool, writeContext func(*model.Context, io.Writer) error) error {
	conf := model.NewDefaultConfiguration()
	conf.Cmd = model.MULTIFILLFORMFIELDS
	fileName = strings.TrimSuffix(filepath.Base(fileName), ".pdf")
	fileName = sanitizeFilenamePart(fileName, "form")
	if format == form.JSON {
		return multiFillFormJSONWith(inFilePDF, rd, outDir, fileName, merge, conf, writeContext)
	}
	return multiFillFormCSVWith(inFilePDF, rd, outDir, fileName, merge, conf, writeContext)
}

func formMutationFileFunctions() []struct {
	name string
	fn   func(string, string) error
} {
	return []struct {
		name string
		fn   func(string, string) error
	}{
		{name: "remove form fields", fn: func(inFile, outFile string) error {
			return RemoveFormFieldsFile(inFile, outFile, nil, nil)
		}},
		{name: "lock form fields", fn: func(inFile, outFile string) error {
			return LockFormFieldsFile(inFile, outFile, nil, nil)
		}},
		{name: "unlock form fields", fn: func(inFile, outFile string) error {
			return UnlockFormFieldsFile(inFile, outFile, nil, nil)
		}},
		{name: "reset form fields", fn: func(inFile, outFile string) error {
			return ResetFormFieldsFile(inFile, outFile, nil, nil)
		}},
	}
}

func callFormAPI(fn func() error) (err error, panicValue any) {
	defer func() {
		panicValue = recover()
	}()
	err = fn()
	return err, nil
}

func TestListFormFieldsRejectsNilReader(t *testing.T) {
	_, err := ListFormFields(nil, nil)
	if !errors.Is(err, ErrMissingPDFReadSeeker) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFReadSeeker, err)
	}
}

func TestFormFileAPIsRejectMissingPDFInput(t *testing.T) {
	dir := t.TempDir()
	dataFile := filepath.Join(dir, "form.json")
	if err := os.WriteFile(dataFile, []byte(`{"forms":[{}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "remove", fn: func() error { return RemoveFormFieldsFile("", filepath.Join(dir, "remove.pdf"), nil, nil) }},
		{name: "lock", fn: func() error { return LockFormFieldsFile("", filepath.Join(dir, "lock.pdf"), nil, nil) }},
		{name: "unlock", fn: func() error { return UnlockFormFieldsFile("", filepath.Join(dir, "unlock.pdf"), nil, nil) }},
		{name: "reset", fn: func() error { return ResetFormFieldsFile("", filepath.Join(dir, "reset.pdf"), nil, nil) }},
		{name: "export", fn: func() error { return ExportFormFile("", filepath.Join(dir, "form-out.json"), nil) }},
		{name: "fill", fn: func() error { return FillFormFile("", dataFile, filepath.Join(dir, "fill.pdf"), nil) }},
		{name: "multi-fill", fn: func() error {
			return MultiFillFormFile("", dataFile, dir, "multi.pdf", false, nil)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, ErrMissingPDFInput) {
				t.Fatalf("expected %v, got %v", ErrMissingPDFInput, err)
			}
		})
	}
}

func TestMultiFillFormRejectsInvalidInputWithoutPanic(t *testing.T) {
	tests := []struct {
		name    string
		fn      func() error
		wantErr error
		want    string
	}{
		{
			name: "missing form reader",
			fn: func() error {
				return MultiFillForm("unused.pdf", nil, t.TempDir(), "out.pdf", form.JSON, false, nil)
			},
			wantErr: ErrMissingFormInput,
			want:    "multi-fill form: missing form input",
		},
		{
			name: "unsupported format",
			fn: func() error {
				return MultiFillForm("unused.pdf", strings.NewReader("a,b\n1,2\n"), t.TempDir(), "out.pdf", form.DataFormat(99), false, nil)
			},
			wantErr: ErrUnsupportedFormDataFormat,
			want:    "multi-fill form: unsupported data format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, panicValue := callFormAPI(tt.fn)
			if panicValue != nil {
				t.Fatalf("unexpected panic: %v", panicValue)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestFormAdditionalBoundaryErrors(t *testing.T) {
	tests := []struct {
		name        string
		fn          func() error
		wantErr     error
		wantContext string
	}{
		{
			name: "export missing JSON writer",
			fn: func() error {
				return ExportFormJSON(bytes.NewReader(nil), nil, "source.pdf", nil)
			},
			wantErr: ErrMissingJSONWriter,
		},
		{
			name: "export missing JSON output",
			fn: func() error {
				return ExportFormFile("unused.pdf", "", nil)
			},
			wantErr: ErrMissingJSONOutput,
		},
		{
			name: "fill missing form reader",
			fn: func() error {
				return FillForm(bytes.NewReader(nil), nil, io.Discard, nil)
			},
			wantErr:     ErrMissingFormInput,
			wantContext: "fill form",
		},
		{
			name: "fill file missing JSON input",
			fn: func() error {
				return FillFormFile("unused.pdf", "", "out.pdf", nil)
			},
			wantErr: ErrMissingJSONInput,
		},
		{
			name: "multi-fill missing PDF input",
			fn: func() error {
				return MultiFillForm("", strings.NewReader(`{"forms":[{}]}`), t.TempDir(), "out.pdf", form.JSON, false, nil)
			},
			wantErr: ErrMissingPDFInput,
		},
		{
			name: "multi-fill file missing form input",
			fn: func() error {
				return MultiFillFormFile("unused.pdf", "", t.TempDir(), "out.pdf", false, nil)
			},
			wantErr: ErrMissingFormInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if tt.wantContext != "" && !strings.Contains(err.Error(), tt.wantContext) {
				t.Fatalf("expected %q in %q", tt.wantContext, err.Error())
			}
		})
	}
}

func TestFormReadErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{name: "list", fn: func() error {
			_, err := FormFields(bytes.NewReader(nil), nil)
			return err
		}, want: "list form fields: prepare PDF context"},
		{name: "rendered list", fn: func() error {
			_, err := ListFormFields(bytes.NewReader(nil), nil)
			return err
		}, want: "list form fields: prepare PDF context"},
		{name: "remove", fn: func() error {
			return RemoveFormFields(bytes.NewReader(nil), io.Discard, nil, nil)
		}, want: "remove form fields: prepare PDF context"},
		{name: "lock", fn: func() error {
			return LockFormFields(bytes.NewReader(nil), io.Discard, nil, nil)
		}, want: "lock form fields: prepare PDF context"},
		{name: "unlock", fn: func() error {
			return UnlockFormFields(bytes.NewReader(nil), io.Discard, nil, nil)
		}, want: "unlock form fields: prepare PDF context"},
		{name: "reset", fn: func() error {
			return ResetFormFields(bytes.NewReader(nil), io.Discard, nil, nil)
		}, want: "reset form fields: prepare PDF context"},
		{name: "export", fn: func() error {
			_, err := ExportForm(bytes.NewReader(nil), "source.pdf", nil)
			return err
		}, want: "export form: prepare PDF context"},
		{name: "export JSON", fn: func() error {
			return ExportFormJSON(bytes.NewReader(nil), io.Discard, "source.pdf", nil)
		}, want: "export form: prepare PDF context"},
		{name: "fill", fn: func() error {
			return FillForm(bytes.NewReader(nil), strings.NewReader(`{"forms":[{}]}`), io.Discard, nil)
		}, want: "fill form: prepare PDF context"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
			if strings.Count(err.Error(), tt.want) != 1 {
				t.Fatalf("expected exactly one %q in %q", tt.want, err.Error())
			}
			if got := strings.Count(err.Error(), "prepare PDF context"); got != 1 {
				t.Fatalf("expected prepare PDF context once, got %d in %q", got, err.Error())
			}
			if !strings.Contains(err.Error(), "read context") {
				t.Fatalf("expected reader context in %q", err.Error())
			}
		})
	}
}

func TestFormMutationOperationErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{name: "remove", fn: func() error {
			return RemoveFormFields(openAPITestPDF(t, inFile), io.Discard, nil, nil)
		}, want: "remove form fields: update fields"},
		{name: "lock", fn: func() error {
			return LockFormFields(openAPITestPDF(t, inFile), io.Discard, nil, nil)
		}, want: "lock form fields: update fields"},
		{name: "unlock", fn: func() error {
			return UnlockFormFields(openAPITestPDF(t, inFile), io.Discard, nil, nil)
		}, want: "unlock form fields: update fields"},
		{name: "reset", fn: func() error {
			return ResetFormFields(openAPITestPDF(t, inFile), io.Discard, nil, nil)
		}, want: "reset form fields: update fields"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestFormMutationNoFieldsAffectedPreservesSentinel(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{name: "remove", fn: func() error {
			return RemoveFormFields(openAPITestPDF(t, formTestInputFile()), io.Discard, []string{"missing"}, nil)
		}, want: "remove form fields"},
		{name: "lock", fn: func() error {
			return LockFormFields(openAPITestPDF(t, formTestInputFile()), io.Discard, []string{"missing"}, nil)
		}, want: "lock form fields"},
		{name: "unlock", fn: func() error {
			return UnlockFormFields(openAPITestPDF(t, formTestInputFile()), io.Discard, []string{"missing"}, nil)
		}, want: "unlock form fields"},
		{name: "reset", fn: func() error {
			return ResetFormFields(openAPITestPDF(t, formTestInputFile()), io.Discard, []string{"missing"}, nil)
		}, want: "reset form fields"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, ErrNoFormFieldsAffected) {
				t.Fatalf("expected %v, got %v", ErrNoFormFieldsAffected, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
		})
	}
}

func TestFormMutationWriteErrorsPreserveCauseAndContext(t *testing.T) {
	wantErr := errors.New("write form output")
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{name: "remove", fn: func() error {
			return RemoveFormFields(openAPITestPDF(t, formMutationTestInputFile()), failingWriter{err: wantErr}, nil, nil)
		}, want: "remove form fields: write output"},
		{name: "lock", fn: func() error {
			return LockFormFields(openAPITestPDF(t, formMutationTestInputFile()), failingWriter{err: wantErr}, nil, nil)
		}, want: "lock form fields: write output"},
		{name: "unlock", fn: func() error {
			return UnlockFormFields(openAPITestPDF(t, formMutationTestInputFile()), failingWriter{err: wantErr}, nil, nil)
		}, want: "unlock form fields: write output"},
		{name: "reset", fn: func() error {
			return ResetFormFields(openAPITestPDF(t, formMutationTestInputFile()), failingWriter{err: wantErr}, nil, nil)
		}, want: "reset form fields: write output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
		})
	}
}

func TestFormListAndExportOperationErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name  string
		fn    func() error
		want  string
		phase string
	}{
		{
			name: "list fields",
			fn: func() error {
				_, err := FormFields(openAPITestPDF(t, inFile), nil)
				return err
			},
			want:  "list form fields: collect fields",
			phase: "collect fields",
		},
		{
			name: "rendered list fields",
			fn: func() error {
				_, err := ListFormFields(openAPITestPDF(t, inFile), nil)
				return err
			},
			want:  "list form fields: collect fields",
			phase: "collect fields",
		},
		{
			name: "export",
			fn: func() error {
				_, err := ExportForm(openAPITestPDF(t, inFile), "source.pdf", nil)
				return err
			},
			want:  "export form: collect data",
			phase: "collect data",
		},
		{
			name: "export JSON",
			fn: func() error {
				return ExportFormJSON(openAPITestPDF(t, inFile), io.Discard, "source.pdf", nil)
			},
			want:  "export form: collect data",
			phase: "collect data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
			if got := strings.Count(err.Error(), tt.phase); got != 1 {
				t.Fatalf("expected %s once, got %d in %q", tt.phase, got, err.Error())
			}
		})
	}
}

func TestExportFormJSONWriteErrorsPreserveCauseAndContext(t *testing.T) {
	wantErr := errors.New("write form JSON")
	tests := []struct {
		name    string
		w       io.Writer
		wantErr error
	}{
		{name: "writer error", w: failingWriter{err: wantErr}, wantErr: wantErr},
		{name: "short write", w: shortFormWriter{}, wantErr: io.ErrShortWrite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExportFormJSON(openAPITestPDF(t, formTestInputFile()), tt.w, "source.pdf", nil)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), "export form: write JSON") {
				t.Fatalf("expected write JSON context, got %q", err.Error())
			}
			if strings.Count(err.Error(), "export form") != 1 || strings.Count(err.Error(), "write JSON") != 1 {
				t.Fatalf("expected unduplicated export and write context, got %q", err.Error())
			}
		})
	}
}

func TestExportFormJSONTranslatesOperationResults(t *testing.T) {
	wantCollectErr := errors.New("collect form data")
	wantEncodeErr := errors.New("encode form data")
	tests := []struct {
		name        string
		export      func(*model.XRefTable, string, io.Writer) (bool, error)
		wantErr     error
		wantContext string
	}{
		{
			name: "collection failure",
			export: func(*model.XRefTable, string, io.Writer) (bool, error) {
				return false, fmt.Errorf("collect data: %w", wantCollectErr)
			},
			wantErr:     wantCollectErr,
			wantContext: "collect data",
		},
		{
			name: "encoding failure",
			export: func(*model.XRefTable, string, io.Writer) (bool, error) {
				return false, fmt.Errorf("encode JSON: %w", wantEncodeErr)
			},
			wantErr:     wantEncodeErr,
			wantContext: "encode JSON",
		},
		{
			name: "nothing exported",
			export: func(*model.XRefTable, string, io.Writer) (bool, error) {
				return false, nil
			},
			wantErr:     ErrNoFormFieldsAffected,
			wantContext: "collect data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exportFormJSONResult(nil, "source.pdf", io.Discard, tt.export)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), "export form: "+tt.wantContext) {
				t.Fatalf("expected %q context, got %v", tt.wantContext, err)
			}
			if strings.Count(err.Error(), "export form") != 1 || strings.Count(err.Error(), tt.wantContext) != 1 {
				t.Fatalf("expected unduplicated export and phase context, got %q", err.Error())
			}
		})
	}
}

func TestFormJSONErrorsPreserveSentinelAndCause(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		wantContext string
		wantCause   any
	}{
		{name: "fill syntax", data: "{", wantContext: "fill form: decode JSON", wantCause: new(*json.SyntaxError)},
		{name: "fill type", data: `{"forms":"bad"}`, wantContext: "fill form: decode JSON", wantCause: new(*json.UnmarshalTypeError)},
		{name: "multi-fill syntax", data: "{", wantContext: "multi-fill form: decode JSON", wantCause: new(*json.SyntaxError)},
		{name: "multi-fill type", data: `{"forms":"bad"}`, wantContext: "multi-fill form: decode JSON", wantCause: new(*json.UnmarshalTypeError)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if strings.HasPrefix(tt.name, "fill ") {
				err = FillForm(openAPITestPDF(t, formTestInputFile()), strings.NewReader(tt.data), io.Discard, nil)
			} else {
				err = MultiFillForm("unused.pdf", strings.NewReader(tt.data), t.TempDir(), "out.pdf", form.JSON, false, nil)
			}
			if !errors.Is(err, ErrInvalidJSON) {
				t.Fatalf("expected %v, got %v", ErrInvalidJSON, err)
			}
			if !errors.As(err, tt.wantCause) {
				t.Fatalf("expected cause %T, got %v", tt.wantCause, err)
			}
			if !strings.Contains(err.Error(), tt.wantContext) {
				t.Fatalf("expected %q in %q", tt.wantContext, err.Error())
			}
		})
	}
}

func TestFillFormDataErrorsPreserveSentinelsAndContext(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr error
	}{
		{name: "missing forms", data: `{}`, wantErr: ErrNoFormData},
		{name: "no fields affected", data: `{"forms":[{}]}`, wantErr: ErrNoFormFieldsAffected},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FillForm(openAPITestPDF(t, formMutationTestInputFile()), strings.NewReader(tt.data), io.Discard, nil)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), "fill form") {
				t.Fatalf("expected fill form context in %q", err.Error())
			}
		})
	}
}

func TestFillFormOptionErrorsIncludePhaseContext(t *testing.T) {
	data := `{"forms":[{"combobox":[{"name":"choice","options":["one"],"value":"invalid"}]}]}`
	err := FillForm(openAPITestPDF(t, formMutationTestInputFile()), strings.NewReader(data), io.Discard, nil)
	if !errors.Is(err, ErrInvalidFormData) {
		t.Fatalf("expected %v, got %v", ErrInvalidFormData, err)
	}
	if err == nil || !strings.Contains(err.Error(), `fill form: validate option values: combo box 1 name: "choice" invalid value: "invalid"`) {
		t.Fatalf("expected option validation context, got %v", err)
	}
}

func TestFillFormOperationErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	err := FillForm(openAPITestPDF(t, inFile), strings.NewReader(`{"forms":[{}]}`), io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "fill form: fill fields") {
		t.Fatalf("expected fill fields context, got %v", err)
	}
	if got := strings.Count(err.Error(), "fill fields"); got != 1 {
		t.Fatalf("expected fill fields once, got %d in %q", got, err.Error())
	}
}

func TestFillFormWriteErrorPreservesCauseAndContext(t *testing.T) {
	wantErr := errors.New("write filled form")
	err := FillForm(openAPITestPDF(t, formMutationTestInputFile()), bytes.NewReader(changedFormTestJSON(t)), failingWriter{err: wantErr}, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "fill form: write output") {
		t.Fatalf("expected write output context in %q", err.Error())
	}
}

func TestMultiFillFormJSONErrorsIncludeFormIndex(t *testing.T) {
	formGroup := form.FormGroup{}
	if err := json.Unmarshal(changedFormTestJSON(t), &formGroup); err != nil {
		t.Fatal(err)
	}
	second := formGroup.Forms[0]
	second.ComboBoxes = []*form.ComboBox{nil}
	formGroup.Forms = append(formGroup.Forms, second)
	bb, err := json.Marshal(formGroup)
	if err != nil {
		t.Fatal(err)
	}

	err = MultiFillForm(formMutationTestInputFile(), bytes.NewReader(bb), t.TempDir(), "batch.pdf", form.JSON, false, nil)
	if !errors.Is(err, ErrInvalidFormData) {
		t.Fatalf("expected %v, got %v", ErrInvalidFormData, err)
	}
	if err == nil || !strings.Contains(err.Error(), "multi-fill form 2: validate form data: combo box 1") {
		t.Fatalf("expected second form context, got %v", err)
	}
}

func TestMultiFillFormJSONOptionErrorsIncludeFormIndex(t *testing.T) {
	formGroup := form.FormGroup{}
	if err := json.Unmarshal(changedFormTestJSON(t), &formGroup); err != nil {
		t.Fatal(err)
	}
	second := formGroup.Forms[0]
	second.ComboBoxes = []*form.ComboBox{{Name: "choice", Options: []string{"one"}, Value: "invalid"}}
	formGroup.Forms = append(formGroup.Forms, second)
	bb, err := json.Marshal(formGroup)
	if err != nil {
		t.Fatal(err)
	}

	err = MultiFillForm(formMutationTestInputFile(), bytes.NewReader(bb), t.TempDir(), "batch.pdf", form.JSON, false, nil)
	if !errors.Is(err, ErrInvalidFormData) {
		t.Fatalf("expected %v, got %v", ErrInvalidFormData, err)
	}
	if err == nil || !strings.Contains(err.Error(), `multi-fill form 2: validate option values: combo box 1 name: "choice" invalid value: "invalid"`) {
		t.Fatalf("expected second form option context, got %v", err)
	}
}

func TestMultiFillFormCSVParseErrorsPreserveSentinelAndCause(t *testing.T) {
	err := MultiFillForm("unused.pdf", strings.NewReader("field\none,two\n"), t.TempDir(), "batch.pdf", form.CSV, false, nil)
	if !errors.Is(err, ErrInvalidCSV) {
		t.Fatalf("expected %v, got %v", ErrInvalidCSV, err)
	}
	var parseErr *csv.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected CSV parse error, got %v", err)
	}
	if !strings.Contains(err.Error(), "multi-fill form: parse CSV") {
		t.Fatalf("expected CSV parse context in %q", err.Error())
	}
}

func TestMultiFillFormCSVErrorsIncludeRowContext(t *testing.T) {
	tests := []struct {
		name    string
		inFile  string
		data    string
		wantErr error
		want    string
	}{
		{
			name:   "open input",
			inFile: filepath.Join(t.TempDir(), "missing.pdf"),
			data:   "field\nvalue\n",
			want:   "multi-fill CSV row 2: open input",
		},
		{
			name:    "no fields affected",
			inFile:  formMutationTestInputFile(),
			data:    "missing\nvalue\n",
			wantErr: ErrNoFormFieldsAffected,
			want:    "multi-fill CSV row 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MultiFillForm(tt.inFile, strings.NewReader(tt.data), t.TempDir(), "batch.pdf", form.CSV, false, nil)
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestMultiFillFormWriteErrorsIncludeFormAndPathContext(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "missing")
	err := MultiFillForm(formMutationTestInputFile(), bytes.NewReader(changedFormTestJSON(t)), outDir, "batch.pdf", form.JSON, false, nil)
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("expected path error, got %v", err)
	}
	if !strings.Contains(err.Error(), "multi-fill form 1: create output "+filepath.Join(outDir, "batch_01.pdf")) {
		t.Fatalf("expected form and output path context in %q", err.Error())
	}
}

func TestMultiFillFormRemovesPartialOutputs(t *testing.T) {
	wantErr := errors.New("write multifill output")
	writeContext := func(_ *model.Context, w io.Writer) error {
		if _, err := w.Write([]byte("partial PDF")); err != nil {
			return err
		}
		return wantErr
	}

	tests := []struct {
		name   string
		format form.DataFormat
		merge  bool
		data   func(*testing.T) io.Reader
	}{
		{name: "JSON", format: form.JSON, data: func(t *testing.T) io.Reader { return bytes.NewReader(changedFormTestJSON(t)) }},
		{name: "JSON merge", format: form.JSON, merge: true, data: func(t *testing.T) io.Reader { return bytes.NewReader(changedFormTestJSON(t)) }},
		{name: "CSV", format: form.CSV, data: func(t *testing.T) io.Reader { return strings.NewReader(changedFormTestCSV(t)) }},
		{name: "CSV merge", format: form.CSV, merge: true, data: func(t *testing.T) io.Reader { return strings.NewReader(changedFormTestCSV(t)) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outDir := t.TempDir()
			err := multiFillFormWithWriteContext(formMutationTestInputFile(), tt.data(t), outDir, "batch.pdf", tt.format, tt.merge, writeContext)
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if err == nil || !strings.Contains(err.Error(), "write output "+filepath.Join(outDir, "batch_01.pdf")) {
				t.Fatalf("expected output path context, got %v", err)
			}
			for _, fileName := range []string{"batch_01.pdf", "batch.pdf"} {
				if _, statErr := os.Stat(filepath.Join(outDir, fileName)); !errors.Is(statErr, os.ErrNotExist) {
					t.Fatalf("expected %s to be absent, got %v", fileName, statErr)
				}
			}
			tmpFiles, globErr := filepath.Glob(filepath.Join(outDir, ".batch_01.pdf.tmp-*"))
			if globErr != nil {
				t.Fatal(globErr)
			}
			if len(tmpFiles) != 0 {
				t.Fatalf("expected temporary output cleanup, got %v", tmpFiles)
			}
		})
	}
}

func TestWriteMultiFillOutputRemovesPartialOutputAfterCloseFailure(t *testing.T) {
	writeContext := func(_ *model.Context, w io.Writer) error {
		f := w.(*os.File)
		if _, err := f.Write([]byte("partial PDF")); err != nil {
			return err
		}
		return f.Close()
	}

	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "batch_01.pdf")
	err := writeMultiFillOutputWith(nil, outFile, "multi-fill form 1", writeContext)
	if err == nil || !strings.Contains(err.Error(), "multi-fill form 1: close output") {
		t.Fatalf("expected close output error, got %v", err)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output to be absent, got %v", statErr)
	}
	tmpFiles, globErr := filepath.Glob(filepath.Join(outDir, ".batch_01.pdf.tmp-*"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(tmpFiles) != 0 {
		t.Fatalf("expected temporary output cleanup, got %v", tmpFiles)
	}
}

func TestMultiFillFormWriteFailurePreservesExistingOutput(t *testing.T) {
	wantErr := errors.New("write multifill output")
	writeContext := func(_ *model.Context, w io.Writer) error {
		if _, err := w.Write([]byte("partial PDF")); err != nil {
			return err
		}
		return wantErr
	}

	tests := []struct {
		name   string
		format form.DataFormat
		merge  bool
		data   func(*testing.T) io.Reader
	}{
		{name: "JSON", format: form.JSON, data: func(t *testing.T) io.Reader { return bytes.NewReader(changedFormTestJSON(t)) }},
		{name: "JSON merge", format: form.JSON, merge: true, data: func(t *testing.T) io.Reader { return bytes.NewReader(changedFormTestJSON(t)) }},
		{name: "CSV", format: form.CSV, data: func(t *testing.T) io.Reader { return strings.NewReader(changedFormTestCSV(t)) }},
		{name: "CSV merge", format: form.CSV, merge: true, data: func(t *testing.T) io.Reader { return strings.NewReader(changedFormTestCSV(t)) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outDir := t.TempDir()
			outFile := filepath.Join(outDir, "batch_01.pdf")
			original := []byte("existing output")
			if err := os.WriteFile(outFile, original, 0o640); err != nil {
				t.Fatal(err)
			}

			err := multiFillFormWithWriteContext(formMutationTestInputFile(), tt.data(t), outDir, "batch.pdf", tt.format, tt.merge, writeContext)
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			bb, readErr := os.ReadFile(outFile)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(bb, original) {
				t.Fatalf("existing output changed: %q", bb)
			}
			tmpFiles, globErr := filepath.Glob(filepath.Join(outDir, ".batch_01.pdf.tmp-*"))
			if globErr != nil {
				t.Fatal(globErr)
			}
			if len(tmpFiles) != 0 {
				t.Fatalf("expected temporary output cleanup, got %v", tmpFiles)
			}
		})
	}
}

func TestMultiFillFormMergeErrorsCleanIntermediateOutputs(t *testing.T) {
	outDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(outDir, "batch.pdf"), 0o700); err != nil {
		t.Fatal(err)
	}

	err := MultiFillForm(formMutationTestInputFile(), bytes.NewReader(changedFormTestJSON(t)), outDir, "batch.pdf", form.JSON, true, nil)
	if err == nil || !strings.Contains(err.Error(), "multi-fill form: merge outputs") {
		t.Fatalf("expected merge context, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "batch_01.pdf")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected intermediate output cleanup, got %v", statErr)
	}
}

func TestMultiFillFormFileOpenDataErrorsIncludeContext(t *testing.T) {
	inFileData := filepath.Join(t.TempDir(), "missing.json")
	err := MultiFillFormFile("unused.pdf", inFileData, t.TempDir(), "batch.pdf", false, nil)
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("expected path error, got %v", err)
	}
	if !strings.Contains(err.Error(), "multi-fill form: open data "+inFileData) {
		t.Fatalf("expected data path context in %q", err.Error())
	}
}

func TestValidateOptionValuesChecksEveryValue(t *testing.T) {
	tests := []struct {
		name string
		f    form.Form
		want string
	}{
		{
			name: "combo box after numeric option",
			f: form.Form{ComboBoxes: []*form.ComboBox{
				{Name: "first", Options: []string{"one"}, Value: "0"},
				{Name: "second", Options: []string{"two"}, Value: "invalid"},
			}},
			want: `combo box 2 name: "second" invalid value: "invalid"`,
		},
		{
			name: "list box after numeric option",
			f: form.Form{ListBoxes: []*form.ListBox{
				{Name: "second", Options: []string{"one"}, Values: []string{"0", "invalid"}},
			}},
			want: `list box 1 name: "second" invalid value: "invalid"`,
		},
		{
			name: "radio group after numeric option",
			f: form.Form{RadioButtonGroups: []*form.RadioButtonGroup{
				{Name: "first", Options: []string{"one"}, Value: "0"},
				{Name: "second", Options: []string{"two"}, Value: "invalid"},
			}},
			want: `radio-button group 2 name: "second" invalid value: "invalid"`,
		},
		{
			name: "negative numeric option",
			f: form.Form{ComboBoxes: []*form.ComboBox{
				{Name: "negative", Options: []string{"one"}, Value: "-1"},
			}},
			want: `combo box 1 name: "negative" invalid value: "-1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptionValues(tt.f)
			if !errors.Is(err, ErrInvalidFormData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFormData, err)
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestValidateOptionValuesRejectsNilFieldsWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		f    form.Form
		want string
	}{
		{name: "combo box", f: form.Form{ComboBoxes: []*form.ComboBox{nil}}, want: "combo box 1: missing field"},
		{name: "list box", f: form.Form{ListBoxes: []*form.ListBox{nil}}, want: "list box 1: missing field"},
		{name: "radio group", f: form.Form{RadioButtonGroups: []*form.RadioButtonGroup{nil}}, want: "radio-button group 1: missing field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, panicValue := callFormAPI(func() error { return validateOptionValues(tt.f) })
			if panicValue != nil {
				t.Fatalf("unexpected panic: %v", panicValue)
			}
			if !errors.Is(err, ErrInvalidFormData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFormData, err)
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestValidateOptionValuesAcceptsValidValues(t *testing.T) {
	tests := []struct {
		name string
		f    form.Form
	}{
		{name: "combo box numeric index", f: form.Form{ComboBoxes: []*form.ComboBox{
			{Name: "choice", Options: []string{"one", "two"}, Value: "1"},
		}}},
		{name: "list box values", f: form.Form{ListBoxes: []*form.ListBox{
			{Name: "choices", Options: []string{"one", "two"}, Values: []string{"one", "1"}},
		}}},
		{name: "radio button value", f: form.Form{RadioButtonGroups: []*form.RadioButtonGroup{
			{Name: "choice", Options: []string{"one"}, Value: "one"},
		}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateOptionValues(tt.f); err != nil {
				t.Fatalf("expected valid option values, got %v", err)
			}
		})
	}
}

func TestFillFormRejectsNilFieldsWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "text field", data: `{"forms":[{"textfield":[null]}]}`, want: "text field 1"},
		{name: "date field", data: `{"forms":[{"datefield":[null]}]}`, want: "date field 1"},
		{name: "checkbox", data: `{"forms":[{"checkbox":[null]}]}`, want: "checkbox 1"},
		{name: "radio group", data: `{"forms":[{"radiobuttongroup":[null]}]}`, want: "radio-button group 1"},
		{name: "combo box", data: `{"forms":[{"combobox":[null]}]}`, want: "combo box 1"},
		{name: "list box", data: `{"forms":[{"listbox":[null]}]}`, want: "list box 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, panicValue := callFormAPI(func() error {
				return FillForm(openAPITestPDF(t, formMutationTestInputFile()), strings.NewReader(tt.data), io.Discard, nil)
			})
			if panicValue != nil {
				t.Fatalf("unexpected panic: %v", panicValue)
			}
			if !errors.Is(err, ErrInvalidFormData) {
				t.Fatalf("expected %v, got %v", ErrInvalidFormData, err)
			}
			want := "fill form: validate form data: " + tt.want
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %v", want, err)
			}
		})
	}
}

func TestFormMutationFileOpenErrorsIncludeContext(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "missing.pdf")
	for _, tt := range formMutationFileFunctions() {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(inFile, filepath.Join(t.TempDir(), "out.pdf"))
			var pathErr *os.PathError
			if !errors.As(err, &pathErr) {
				t.Fatalf("expected path error, got %v", err)
			}
			want := tt.name + ": open input " + inFile
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q in %q", want, err.Error())
			}
		})
	}
}

func TestFormMutationFileCreateErrorsIncludeContext(t *testing.T) {
	for _, tt := range formMutationFileFunctions() {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "missing", "out.pdf")
			err := tt.fn(formMutationTestInputFile(), outFile)
			var pathErr *os.PathError
			if !errors.As(err, &pathErr) {
				t.Fatalf("expected path error, got %v", err)
			}
			want := tt.name + ": create output"
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q in %q", want, err.Error())
			}
		})
	}
}

func TestExportAndFillFormFileOpenErrorsIncludeContext(t *testing.T) {
	dir := t.TempDir()
	missingPDF := filepath.Join(dir, "missing.pdf")
	missingJSON := filepath.Join(dir, "missing.json")
	dataFile := filepath.Join(dir, "form.json")
	if err := os.WriteFile(dataFile, []byte(`{"forms":[{}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "export PDF input",
			fn:   func() error { return ExportFormFile(missingPDF, filepath.Join(dir, "form-out.json"), nil) },
			want: "export form: open input " + missingPDF,
		},
		{
			name: "fill form data",
			fn:   func() error { return FillFormFile("unused.pdf", missingJSON, filepath.Join(dir, "out.pdf"), nil) },
			want: "fill form: open form data " + missingJSON,
		},
		{
			name: "fill PDF input",
			fn:   func() error { return FillFormFile(missingPDF, dataFile, filepath.Join(dir, "out.pdf"), nil) },
			want: "fill form: open input " + missingPDF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected not exist error, got %v", err)
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
		})
	}
}

func TestExportAndFillFormFileCreateErrorsIncludeContext(t *testing.T) {
	dir := t.TempDir()
	inFilePDF := filepath.Join(dir, "input.pdf")
	inFileJSON := filepath.Join(dir, "form.json")
	if err := os.WriteFile(inFilePDF, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inFileJSON, []byte(`{"forms":[{}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(dir, "missing", "out")

	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "export",
			fn:   func() error { return ExportFormFile(inFilePDF, outFile+".json", nil) },
			want: "export form: create output " + outFile + ".json",
		},
		{
			name: "fill",
			fn:   func() error { return FillFormFile(inFilePDF, inFileJSON, outFile+".pdf", nil) },
			want: "fill form: create output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected not exist error, got %v", err)
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %q", tt.want, err.Error())
			}
		})
	}
}

func TestExportAndFillFormFilesRemoveOutputAfterFailure(t *testing.T) {
	dir := t.TempDir()
	brokenPDF := filepath.Join(dir, "broken.pdf")
	if err := os.WriteFile(brokenPDF, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	dataFile := filepath.Join(dir, "form.json")
	if err := os.WriteFile(dataFile, []byte(`{"forms":[{}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		outFile string
		fn      func(string) error
	}{
		{
			name:    "export",
			outFile: filepath.Join(dir, "form-out.json"),
			fn:      func(outFile string) error { return ExportFormFile(brokenPDF, outFile, nil) },
		},
		{
			name:    "fill",
			outFile: filepath.Join(dir, "filled.pdf"),
			fn:      func(outFile string) error { return FillFormFile(brokenPDF, dataFile, outFile, nil) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(tt.outFile)
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if _, statErr := os.Stat(tt.outFile); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("expected failed output to be removed, got %v", statErr)
			}
		})
	}
}

func TestFormMutationFilesRemoveOutputAfterFailure(t *testing.T) {
	for _, tt := range formMutationFileFunctions() {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			inFile := filepath.Join(dir, "broken.pdf")
			outFile := filepath.Join(dir, "out.pdf")
			if err := os.WriteFile(inFile, nil, 0o600); err != nil {
				t.Fatal(err)
			}

			err := tt.fn(inFile, outFile)
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if !strings.Contains(err.Error(), tt.name+": prepare PDF context") {
				t.Fatalf("expected operation context in %q", err.Error())
			}
			if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("expected failed output to be removed, got %v", statErr)
			}
		})
	}
}

func TestCleanupFormMutationFilesPreservesAllErrors(t *testing.T) {
	dir := t.TempDir()
	f1, err := os.CreateTemp(dir, "input")
	if err != nil {
		t.Fatal(err)
	}
	f2, err := os.CreateTemp(dir, "output")
	if err != nil {
		t.Fatal(err)
	}
	if err := f1.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f2.Close(); err != nil {
		t.Fatal(err)
	}
	nonEmptyDir := filepath.Join(dir, "non-empty")
	if err := os.Mkdir(nonEmptyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "keep"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	wantErr := errors.New("mutation failed")
	err = newStagedOutput(f1, f2, nonEmptyDir, "", "", "", "lock form fields").cleanup(wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	for _, want := range []string{"close output", "close input", "remove output"} {
		if !strings.Contains(err.Error(), "lock form fields: "+want) {
			t.Fatalf("expected %q context in %q", want, err.Error())
		}
	}
}

func TestFormDataReadErrorPreservesCause(t *testing.T) {
	wantErr := errors.New("read form data")
	rd := errorReader{err: wantErr}

	_, err := formGroupFromReader(rd)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if err == nil || !strings.Contains(err.Error(), "fill form: read form data") {
		t.Fatalf("expected read phase context, got %v", err)
	}
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

type shortFormWriter struct{}

func (shortFormWriter) Write(p []byte) (int, error) {
	return len(p) - 1, nil
}

var _ io.Reader = errorReader{}
var _ io.Writer = shortFormWriter{}
