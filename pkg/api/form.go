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

package api

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/create"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/sanitize"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var (
	// ErrNoFormData signals missing form data for a form fill operation.
	ErrNoFormData = errors.New("missing form data")

	// ErrMissingFormInput signals a missing required form input reader or file.
	ErrMissingFormInput = errors.New("missing form input")

	// ErrNoFormFieldsAffected signals that a form operation did not change any fields.
	ErrNoFormFieldsAffected = errors.New("no form fields affected")

	// ErrUnsupportedFormDataFormat signals an unsupported form data format.
	ErrUnsupportedFormDataFormat = errors.New("unsupported data format")

	// ErrInvalidCSV signals malformed or incomplete CSV form data.
	ErrInvalidCSV = errors.New("invalid csv input file")

	// ErrInvalidJSON signals invalid JSON form data.
	ErrInvalidJSON = errors.New("invalid JSON encoding")

	// ErrInvalidFormData signals structurally invalid decoded form data.
	ErrInvalidFormData = errors.New("invalid form data")
)

// FormFields returns all form fields of rs.
func FormFields(rs io.ReadSeeker, conf *model.Configuration) (fields []form.Field, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTFORMFIELDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("list form fields: %w", err)
	}

	fields, _, err = form.FormFields(ctx)
	if err != nil {
		return nil, fmt.Errorf("list form fields: collect fields: %w", err)
	}

	return fields, nil
}

// ListFormFields returns a rendered list of all form fields in rs.
func ListFormFields(rs io.ReadSeeker, conf *model.Configuration) (fields []string, err error) {
	defer fault.Catch(&err)
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTFORMFIELDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("list form fields: %w", err)
	}

	fields, err = form.ListFormFields(ctx)
	if err != nil {
		return nil, fmt.Errorf("list form fields: %w", err)
	}
	return fields, nil
}

// RemoveFormFields deletes form fields in rs and writes the result to w.
func RemoveFormFields(rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if err := validateNoEmptyStrings(fieldIDsOrNames, "form field ID or name"); err != nil {
		return err
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEFORMFIELDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("remove form fields: %w", err)
	}

	ok, err := form.RemoveFormFields(ctx, fieldIDsOrNames)
	if err != nil {
		return fmt.Errorf("remove form fields: update fields: %w", err)
	}
	if !ok {
		return fmt.Errorf("remove form fields: %w", ErrNoFormFieldsAffected)
	}

	if err := Write(ctx, w, conf); err != nil {
		return fmt.Errorf("remove form fields: write output: %w", err)
	}
	return nil
}

type formFieldMutation func(io.ReadSeeker, io.Writer, []string, *model.Configuration) error

func mutateFormFieldsFile(inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration, operation string, mutate formFieldMutation) (err error) {
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if err := validateNoEmptyStrings(fieldIDsOrNames, "form field ID or name"); err != nil {
		return err
	}

	f1, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("%s: open input %s: %w", operation, inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}

	staged, err := openStagedOutput(f1, inFile, tmpFile, operation)
	if err != nil {
		return errors.Join(
			fmt.Errorf("%s: create output: %w", operation, err),
			closeFile(f1, operation+": close input"),
		)
	}
	f2 := staged.output.file

	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = mutate(f1, f2, fieldIDsOrNames, conf); err != nil {
		return err
	}
	ok = true
	return nil
}

// RemoveFormFieldsFile deletes form fields in inFile and writes the result to outFile.
func RemoveFormFieldsFile(inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	return mutateFormFieldsFile(inFile, outFile, fieldIDsOrNames, conf, "remove form fields", RemoveFormFields)
}

// LockFormFields turns form fields in rs into read-only and writes the result to w.
func LockFormFields(rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if err := validateNoEmptyStrings(fieldIDsOrNames, "form field ID or name"); err != nil {
		return err
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LOCKFORMFIELDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("lock form fields: %w", err)
	}

	ok, err := form.LockFormFields(ctx, fieldIDsOrNames)
	if err != nil {
		return fmt.Errorf("lock form fields: update fields: %w", err)
	}
	if !ok {
		return fmt.Errorf("lock form fields: %w", ErrNoFormFieldsAffected)
	}

	if err := Write(ctx, w, conf); err != nil {
		return fmt.Errorf("lock form fields: write output: %w", err)
	}
	return nil
}

// LockFormFieldsFile turns form fields of inFile into read-only and writes the result to outFile.
func LockFormFieldsFile(inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	return mutateFormFieldsFile(inFile, outFile, fieldIDsOrNames, conf, "lock form fields", LockFormFields)
}

// UnlockFormFields makes form fields in rs writable and writes the result to w.
func UnlockFormFields(rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if err := validateNoEmptyStrings(fieldIDsOrNames, "form field ID or name"); err != nil {
		return err
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.UNLOCKFORMFIELDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("unlock form fields: %w", err)
	}

	ok, err := form.UnlockFormFields(ctx, fieldIDsOrNames)
	if err != nil {
		return fmt.Errorf("unlock form fields: update fields: %w", err)
	}
	if !ok {
		return fmt.Errorf("unlock form fields: %w", ErrNoFormFieldsAffected)
	}

	if err := Write(ctx, w, conf); err != nil {
		return fmt.Errorf("unlock form fields: write output: %w", err)
	}
	return nil
}

// UnlockFormFieldsFile makes form fields of inFile writable and writes the result to outFile.
func UnlockFormFieldsFile(inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	return mutateFormFieldsFile(inFile, outFile, fieldIDsOrNames, conf, "unlock form fields", UnlockFormFields)
}

// ResetFormFields resets form fields of rs and writes the result to w.
func ResetFormFields(rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if err := validateNoEmptyStrings(fieldIDsOrNames, "form field ID or name"); err != nil {
		return err
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.RESETFORMFIELDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("reset form fields: %w", err)
	}

	ok, err := form.ResetFormFields(ctx, fieldIDsOrNames)
	if err != nil {
		return fmt.Errorf("reset form fields: update fields: %w", err)
	}
	if !ok {
		return fmt.Errorf("reset form fields: %w", ErrNoFormFieldsAffected)
	}

	if err := Write(ctx, w, conf); err != nil {
		return fmt.Errorf("reset form fields: write output: %w", err)
	}
	return nil
}

// ResetFormFieldsFile resets form fields of inFile and writes the result to outFile.
func ResetFormFieldsFile(inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	return mutateFormFieldsFile(inFile, outFile, fieldIDsOrNames, conf, "reset form fields", ResetFormFields)
}

// ExportForm extracts form data originating from source from rs.
func ExportForm(rs io.ReadSeeker, source string, conf *model.Configuration) (formGroup *form.FormGroup, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXPORTFORMFIELDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("export form: %w", err)
	}

	formGroup, err = exportedFormGroup(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("export form: collect data: %w", err)
	}

	return formGroup, nil
}

func exportedFormGroup(ctx *model.Context, source string) (*form.FormGroup, error) {
	formGroup, ok, err := form.ExportForm(ctx.XRefTable, source)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNoFormFieldsAffected
	}
	return formGroup, nil
}

type formJSONExporter func(*model.XRefTable, string, io.Writer) (bool, error)

func exportFormJSONResult(xRefTable *model.XRefTable, source string, w io.Writer, export formJSONExporter) error {
	ok, err := export(xRefTable, source, w)
	if err != nil {
		return fmt.Errorf("export form: %w", err)
	}
	if !ok {
		return fmt.Errorf("export form: collect data: %w", ErrNoFormFieldsAffected)
	}
	return nil
}

// ExportFormJSON extracts form data originating from source from rs and writes the result to w.
func ExportFormJSON(rs io.ReadSeeker, w io.Writer, source string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingJSONWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXPORTFORMFIELDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("export form: %w", err)
	}

	return exportFormJSONResult(ctx.XRefTable, source, w, form.ExportFormJSON)
}

// ExportFormFile extracts form data from inFilePDF and writes the result to outFileJSON.
func ExportFormFile(inFilePDF, outFileJSON string, conf *model.Configuration) (err error) {
	if inFilePDF == "" {
		return ErrMissingPDFInput
	}

	if outFileJSON == "" {
		return ErrMissingJSONOutput
	}

	f1, err := os.Open(inFilePDF)
	if err != nil {
		return fmt.Errorf("export form: open input %s: %w", inFilePDF, err)
	}

	staged, err := openStagedOutput(f1, inFilePDF, outFileJSON, "export form")
	if err != nil {
		return errors.Join(
			fmt.Errorf("export form: create output %s: %w", outFileJSON, err),
			closeFile(f1, "export form: close input"),
		)
	}
	f2 := staged.output.file
	logWritingTo(outFileJSON)

	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = ExportFormJSON(f1, f2, inFilePDF, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

func validOptionValue(value string, options []string) bool {
	if types.MemberOf(value, options) {
		return true
	}
	i, err := strconv.Atoi(value)
	return err == nil && i >= 0 && i < len(options)
}

func unknownOptionValueError(fieldType string, fieldIndex int, name, value string, options []string) error {
	return fmt.Errorf("%s %d name: %q invalid value: %q - options: [%s]: %w", fieldType, fieldIndex+1, name, value, strings.Join(options, ", "), ErrInvalidFormData)
}

func validateFormData(f form.Form) error {
	for i, field := range f.TextFields {
		if field == nil {
			return fmt.Errorf("text field %d: %w", i+1, ErrInvalidFormData)
		}
	}
	for i, field := range f.DateFields {
		if field == nil {
			return fmt.Errorf("date field %d: %w", i+1, ErrInvalidFormData)
		}
	}
	for i, field := range f.CheckBoxes {
		if field == nil {
			return fmt.Errorf("checkbox %d: %w", i+1, ErrInvalidFormData)
		}
	}
	for i, field := range f.RadioButtonGroups {
		if field == nil {
			return fmt.Errorf("radio-button group %d: %w", i+1, ErrInvalidFormData)
		}
	}
	for i, field := range f.ComboBoxes {
		if field == nil {
			return fmt.Errorf("combo box %d: %w", i+1, ErrInvalidFormData)
		}
	}
	for i, field := range f.ListBoxes {
		if field == nil {
			return fmt.Errorf("list box %d: %w", i+1, ErrInvalidFormData)
		}
	}
	return nil
}

func validateComboBoxValues(f form.Form) error {
	for i, cb := range f.ComboBoxes {
		if cb == nil {
			return fmt.Errorf("combo box %d: missing field: %w", i+1, ErrInvalidFormData)
		}
		if cb.Value == "" || cb.Editable {
			continue
		}
		if len(cb.Options) > 0 && !validOptionValue(cb.Value, cb.Options) {
			return unknownOptionValueError("combo box", i, cb.Name, cb.Value, cb.Options)
		}
	}
	return nil
}

func validateListBoxValues(f form.Form) error {
	for i, lb := range f.ListBoxes {
		if lb == nil {
			return fmt.Errorf("list box %d: missing field: %w", i+1, ErrInvalidFormData)
		}
		if len(lb.Values) == 0 {
			continue
		}
		for _, value := range lb.Values {
			if len(lb.Options) > 0 && !validOptionValue(value, lb.Options) {
				return unknownOptionValueError("list box", i, lb.Name, value, lb.Options)
			}
		}
	}
	return nil
}

func validateRadioButtonGroupValues(f form.Form) error {
	for i, rbg := range f.RadioButtonGroups {
		if rbg == nil {
			return fmt.Errorf("radio-button group %d: missing field: %w", i+1, ErrInvalidFormData)
		}
		if rbg.Value == "" {
			continue
		}
		if len(rbg.Options) > 0 && !validOptionValue(rbg.Value, rbg.Options) {
			return unknownOptionValueError("radio-button group", i, rbg.Name, rbg.Value, rbg.Options)
		}
	}
	return nil
}

func validateOptionValues(f form.Form) error {
	if err := validateRadioButtonGroupValues(f); err != nil {
		return err
	}

	if err := validateComboBoxValues(f); err != nil {
		return err
	}

	if err := validateListBoxValues(f); err != nil {
		return err
	}

	return nil
}

func fillPostProc(ctx *model.Context, pp []*model.Page) error {
	if _, _, err := create.UpdatePageTree(ctx, pp, nil); err != nil {
		return fmt.Errorf("fill form: update page tree: %w", err)
	}
	if err := ValidateContext(ctx); err != nil {
		return fmt.Errorf("fill form: validate output: %w", err)
	}
	return nil
}

func formGroupFromReader(rd io.Reader) (*form.FormGroup, error) {
	bb, err := io.ReadAll(rd)
	if err != nil {
		return nil, fmt.Errorf("fill form: read form data: %w", err)
	}

	formGroup := form.FormGroup{}
	if err := json.Unmarshal(bb, &formGroup); err != nil {
		return nil, fmt.Errorf("fill form: decode JSON: %w", errors.Join(ErrInvalidJSON, err))
	}
	return &formGroup, nil
}

func validatedFillForm(formGroup *form.FormGroup) (form.Form, error) {
	if len(formGroup.Forms) == 0 {
		return form.Form{}, fmt.Errorf("fill form: %w", ErrNoFormData)
	}
	f := formGroup.Forms[0]
	if err := validateFormData(f); err != nil {
		return form.Form{}, fmt.Errorf("fill form: validate form data: %w", err)
	}
	if err := validateOptionValues(f); err != nil {
		return form.Form{}, fmt.Errorf("fill form: validate option values: %w", err)
	}
	return f, nil
}

// FillForm populates the form rs with data from rd and writes the result to w.
func FillForm(rs io.ReadSeeker, rd io.Reader, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if rd == nil {
		return fmt.Errorf("fill form: %w", ErrMissingFormInput)
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.FILLFORMFIELDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("fill form: %w", err)
	}

	// TODO not necessarily so
	ctx.RemoveSignature()

	formGroup, err := formGroupFromReader(rd)
	if err != nil {
		return err
	}

	f, err := validatedFillForm(formGroup)
	if err != nil {
		return err
	}

	ok, pp, err := form.FillForm(ctx, form.FillDetails(&f, nil), f.Pages, form.JSON)
	if err != nil {
		return fmt.Errorf("fill form: fill fields: %w", err)
	}
	if !ok {
		if log.CLIEnabled() {
			log.CLI.Println("nothing written")
		}
		return fmt.Errorf("fill form: %w", ErrNoFormFieldsAffected)
	}

	if log.CLIEnabled() {
		log.CLI.Println("filling...")
	}

	if err := fillPostProc(ctx, pp); err != nil {
		return err
	}

	if err := Write(ctx, w, conf); err != nil {
		return fmt.Errorf("fill form: write output: %w", err)
	}
	return nil
}

// FillFormFile populates the form inFilePDF with data from inFileJSON and writes the result to outFilePDF.
func FillFormFile(inFilePDF, inFileJSON, outFilePDF string, conf *model.Configuration) (err error) {
	if inFilePDF == "" {
		return ErrMissingPDFInput
	}

	if inFileJSON == "" {
		return ErrMissingJSONInput
	}

	f0, err := os.Open(inFileJSON)
	if err != nil {
		return fmt.Errorf("fill form: open form data %s: %w", inFileJSON, err)
	}

	f1, err := os.Open(inFilePDF)
	if err != nil {
		return errors.Join(
			fmt.Errorf("fill form: open input %s: %w", inFilePDF, err),
			closeFile(f0, "fill form: close form data"),
		)
	}

	tmpFile := ""
	if outFilePDF != "" && inFilePDF != outFilePDF {
		tmpFile = outFilePDF
		logWritingTo(outFilePDF)
	} else {
		logWritingTo(inFilePDF)
	}
	staged, err := openStagedOutput(f1, inFilePDF, tmpFile, "fill form")
	if err != nil {
		return errors.Join(
			fmt.Errorf("fill form: create output: %w", err),
			closeFile(f1, "fill form: close input"),
			closeFile(f0, "fill form: close form data"),
		)
	}
	f2 := staged.output.file
	staged = staged.withInput(f0, "fill form: close form data")

	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = FillForm(f1, f0, f2, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

func parseFormGroup(rd io.Reader) (*form.FormGroup, error) {
	bb, err := io.ReadAll(rd)
	if err != nil {
		return nil, fmt.Errorf("multi-fill form: read form data: %w", err)
	}

	formGroup := &form.FormGroup{}
	if err := json.Unmarshal(bb, formGroup); err != nil {
		return nil, fmt.Errorf("multi-fill form: decode JSON: %w", errors.Join(ErrInvalidJSON, err))
	}

	if len(formGroup.Forms) == 0 {
		return nil, fmt.Errorf("multi-fill form: %w", ErrNoFormData)
	}
	return formGroup, nil
}

// rollbackMultiFillOutputs removes the intermediate files belonging to the
// multi-file form transaction. It is intentionally separate from stagedOutput.
func rollbackMultiFillOutputs(outFiles []string) error {
	var errs []error
	for i, fileName := range outFiles {
		context := fmt.Sprintf("multi-fill form: remove intermediate %d %s", i+1, fileName)
		errs = append(errs, removeFile(fileName, context))
	}
	return errors.Join(errs...)
}

func mergeForms(outDir, fileName string, outFiles []string, conf *model.Configuration) error {
	fileName = sanitizeFilenamePart(fileName, "form")
	outFile := filepath.Join(outDir, fileName+".pdf")
	if err := MergeCreateFile(outFiles, outFile, false, conf); err != nil {
		return fmt.Errorf("multi-fill form: merge outputs: %w", err)
	}
	return nil
}

func multiFillJSONOutputFile(outDir, fileName, requested string, formNr int) string {
	if requested == "" {
		return filepath.Join(outDir, fmt.Sprintf("%s_%02d.pdf", fileName, formNr))
	}

	outFile, err := sanitize.Path(requested)
	if err != nil {
		outFile = fmt.Sprintf("form_%02d", formNr)
	}
	if !strings.HasSuffix(strings.ToLower(outFile), ".pdf") {
		outFile += ".pdf"
	}
	return filepath.Join(outDir, outFile)
}

func multiFillCSVOutputFile(outDir, fileName, requested string, recordNr int) string {
	if requested == "" {
		return filepath.Join(outDir, fmt.Sprintf("%s_%02d.pdf", fileName, recordNr))
	}

	outFile, err := sanitize.Path(requested)
	if err != nil {
		outFile = fmt.Sprintf("form_%02d", recordNr)
	}
	return filepath.Join(outDir, outFile)
}

func multiFillPostProcess(ctx *model.Context, pp []*model.Page, context string, validate bool) error {
	if _, _, err := create.UpdatePageTree(ctx, pp, nil); err != nil {
		return fmt.Errorf("%s: update page tree: %w", context, err)
	}
	if validate {
		if err := ValidateContext(ctx); err != nil {
			return fmt.Errorf("%s: validate output: %w", context, err)
		}
	}
	return nil
}

func writeMultiFillOutput(ctx *model.Context, outFile, context string) error {
	return writeMultiFillOutputWith(ctx, outFile, context, WriteContext)
}

func writeMultiFillOutputWith(ctx *model.Context, outFile, context string, writeContext func(*model.Context, io.Writer) error) error {
	logWritingTo(outFile)
	staged, err := openStagedOutput(nil, "", outFile, context)
	if err != nil {
		return fmt.Errorf("%s: create output %s: %w", context, outFile, err)
	}

	f := staged.output.file
	staged.removeContext = context + ": remove temporary output"
	staged.replaceContext = fmt.Sprintf("%s: replace output %s", context, outFile)
	if err := writeContext(ctx, f); err != nil {
		return staged.cleanup(fmt.Errorf("%s: write output %s: %w", context, outFile, err))
	}
	return staged.commit()
}

func multiFillJSONForm(inFilePDF string, f form.Form, outDir, fileName string, formNr int, conf *model.Configuration, writeContext func(*model.Context, io.Writer) error) (outFile string, err error) {
	context := fmt.Sprintf("multi-fill form %d", formNr)
	if err := validateFormData(f); err != nil {
		return "", fmt.Errorf("%s: validate form data: %w", context, err)
	}
	rs, err := os.Open(inFilePDF)
	if err != nil {
		return "", fmt.Errorf("%s: open input %s: %w", context, inFilePDF, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(rs, context+": close input"))
	}()

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return "", fmt.Errorf("%s: %w", context, err)
	}

	if err := validateOptionValues(f); err != nil {
		return "", fmt.Errorf("%s: validate option values: %w", context, err)
	}

	ok, pp, err := form.FillForm(ctx, form.FillDetails(&f, nil), f.Pages, form.JSON)
	if err != nil {
		return "", fmt.Errorf("%s: fill fields: %w", context, err)
	}
	if !ok {
		return "", fmt.Errorf("%s: %w", context, ErrNoFormFieldsAffected)
	}

	if err := multiFillPostProcess(ctx, pp, context, conf.PostProcessValidate); err != nil {
		return "", err
	}

	outFile = multiFillJSONOutputFile(outDir, fileName, f.FileName, formNr)
	if err := writeMultiFillOutputWith(ctx, outFile, context, writeContext); err != nil {
		return "", err
	}
	return outFile, nil
}

func multiFillFormJSON(inFilePDF string, rd io.Reader, outDir, fileName string, merge bool, conf *model.Configuration) (err error) {
	return multiFillFormJSONWith(inFilePDF, rd, outDir, fileName, merge, conf, WriteContext)
}

func multiFillFormJSONWith(inFilePDF string, rd io.Reader, outDir, fileName string, merge bool, conf *model.Configuration, writeContext func(*model.Context, io.Writer) error) (err error) {
	formGroup, err := parseFormGroup(rd)
	if err != nil {
		return err
	}

	var outFiles []string
	if merge {
		defer func() {
			if log.CLIEnabled() {
				log.CLI.Println("cleaning up...")
			}
			err = errors.Join(err, rollbackMultiFillOutputs(outFiles))
		}()
	}

	for i, f := range formGroup.Forms {
		outFile, fillErr := multiFillJSONForm(inFilePDF, f, outDir, fileName, i+1, conf, writeContext)
		if outFile != "" {
			outFiles = append(outFiles, outFile)
		}
		if fillErr != nil {
			return fillErr
		}
	}

	if merge {
		return mergeForms(outDir, fileName, outFiles, conf)
	}
	return nil
}

func parseCSVLines(rd io.Reader) ([][]string, error) {
	// Does NOT do any fieldtype checking!
	// Don't use unless you know your form anatomy inside out!

	// The first row is expected to hold the fieldIDs/fieldNames of the fields to be filled - the only form metadata needed for this usecase.
	// The remaining rows are the corresponding data tuples.
	// Each row results in one separate PDF form written to outDir.

	// fieldName1	fieldName2	fieldName3	fieldName4
	// John			Doe			1.1.2000	male
	// Jane			Doe			1.1.2000	female
	// Jacky		Doe			1.1.2000	non-binary

	csvLines, err := csv.NewReader(rd).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("multi-fill form: parse CSV: %w", errors.Join(ErrInvalidCSV, err))
	}

	if len(csvLines) < 2 {
		return nil, fmt.Errorf("multi-fill form: parse CSV: %w", ErrInvalidCSV)
	}

	fieldNames := csvLines[0]
	if len(fieldNames) == 0 {
		return nil, fmt.Errorf("multi-fill form: parse CSV: %w", ErrInvalidCSV)
	}
	for i, fieldName := range fieldNames {
		if fieldName == "" || fieldName == "*" {
			return nil, fmt.Errorf("multi-fill form: parse CSV header column %d: %w", i+1, ErrInvalidCSV)
		}
	}

	return csvLines, nil
}

func multiFillCSVRecord(inFilePDF string, fieldNames, formRecord []string, outDir, fileName string, recordNr, rowNr int, conf *model.Configuration, writeContext func(*model.Context, io.Writer) error) (outFile string, err error) {
	context := fmt.Sprintf("multi-fill CSV row %d", rowNr)
	f, err := os.Open(inFilePDF)
	if err != nil {
		return "", fmt.Errorf("%s: open input %s: %w", context, inFilePDF, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(f, context+": close input"))
	}()

	ctx, err := ReadValidateAndOptimize(f, conf)
	if err != nil {
		return "", fmt.Errorf("%s: %w", context, err)
	}

	fieldMap, imgPageMap, requested, err := form.FieldMap(fieldNames, formRecord)
	if err != nil {
		return "", fmt.Errorf("%s: map fields: %w", context, errors.Join(ErrInvalidCSV, err))
	}

	ok, pp, err := form.FillForm(ctx, form.FillDetails(nil, fieldMap), imgPageMap, form.CSV)
	if err != nil {
		return "", fmt.Errorf("%s: fill fields: %w", context, err)
	}
	if !ok {
		return "", fmt.Errorf("%s: %w", context, ErrNoFormFieldsAffected)
	}

	if err := multiFillPostProcess(ctx, pp, context, conf.PostProcessValidate); err != nil {
		return "", err
	}

	outFile = multiFillCSVOutputFile(outDir, fileName, requested, recordNr)
	if err := writeMultiFillOutputWith(ctx, outFile, context, writeContext); err != nil {
		return "", err
	}
	return outFile, nil
}

func multiFillFormCSV(inFilePDF string, rd io.Reader, outDir, fileName string, merge bool, conf *model.Configuration) (err error) {
	return multiFillFormCSVWith(inFilePDF, rd, outDir, fileName, merge, conf, WriteContext)
}

func multiFillFormCSVWith(inFilePDF string, rd io.Reader, outDir, fileName string, merge bool, conf *model.Configuration, writeContext func(*model.Context, io.Writer) error) (err error) {
	csvLines, err := parseCSVLines(rd)
	if err != nil {
		return err
	}

	fieldNames := csvLines[0]
	var outFiles []string
	if merge {
		defer func() {
			if log.CLIEnabled() {
				log.CLI.Println("cleaning up...")
			}
			err = errors.Join(err, rollbackMultiFillOutputs(outFiles))
		}()
	}

	for i, formRecord := range csvLines[1:] {
		outFile, fillErr := multiFillCSVRecord(inFilePDF, fieldNames, formRecord, outDir, fileName, i+1, i+2, conf, writeContext)
		if outFile != "" {
			outFiles = append(outFiles, outFile)
		}
		if fillErr != nil {
			return fillErr
		}
	}

	if merge {
		return mergeForms(outDir, fileName, outFiles, conf)
	}
	return nil
}

// MultiFillForm populates multiple instances of inFilePDF's form with data from rd and writes the result to outDir.
func MultiFillForm(inFilePDF string, rd io.Reader, outDir, fileName string, format form.DataFormat, merge bool, conf *model.Configuration) error {
	if inFilePDF == "" {
		return ErrMissingPDFInput
	}

	if rd == nil {
		return fmt.Errorf("multi-fill form: %w", ErrMissingFormInput)
	}

	if format != form.JSON && format != form.CSV {
		return fmt.Errorf("multi-fill form: %w: %d", ErrUnsupportedFormDataFormat, format)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.MULTIFILLFORMFIELDS

	fileName = strings.TrimSuffix(filepath.Base(fileName), ".pdf")
	fileName = sanitizeFilenamePart(fileName, "form")

	if format == form.JSON {
		return multiFillFormJSON(inFilePDF, rd, outDir, fileName, merge, conf)
	}

	return multiFillFormCSV(inFilePDF, rd, outDir, fileName, merge, conf)
}

// MultiFillFormFile populates multiple instances of inFilePDF's form with data from inFileData and writes the result to outDir.
// The output file will be written to outFilePDF with incrementing numerical suffix unless
// the input JSON uses "filename" or the input CSV contains a @filename field.
func MultiFillFormFile(inFilePDF, inFileData, outDir, outFilePDF string, merge bool, conf *model.Configuration) (err error) {
	if inFilePDF == "" {
		return ErrMissingPDFInput
	}

	if inFileData == "" {
		return ErrMissingFormInput
	}

	format := form.JSON
	if strings.HasSuffix(strings.ToLower(inFileData), ".csv") {
		format = form.CSV
	}

	f, err := os.Open(inFileData)
	if err != nil {
		return fmt.Errorf("multi-fill form: open data %s: %w", inFileData, err)
	}

	defer func() {
		err = errors.Join(err, closeFile(f, "multi-fill form: close data"))
	}()

	s := "JSON"
	if format == form.CSV {
		s = "CSV"
	}

	outFileBase := filepath.Base(outFilePDF)

	if log.CLIEnabled() {
		log.CLI.Printf("filling multiple forms via %s based on %s data from %s into %s/%s ...\n", inFilePDF, s, inFileData, outDir, outFileBase)
	}

	err = MultiFillForm(inFilePDF, f, outDir, outFileBase, format, merge, conf)
	return err
}
