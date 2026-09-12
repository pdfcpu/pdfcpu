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
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/create"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/sanitize"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// FormFields returns all form fields of rs and supports cancellation.
func FormFields(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (fields []form.Field, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.LISTFORMFIELDS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return nil, fmt.Errorf("list form fields: %w", err)
	}

	fields, _, err = form.FormFields(c, ctx)
	if err != nil {
		return nil, fmt.Errorf("list form fields: collect fields: %w", err)
	}

	return fields, nil
}

// ListFormFields returns a rendered list of all form fields in rs and supports cancellation.
func ListFormFields(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (fields []string, err error) {
	defer fault.Catch(&err)
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.LISTFORMFIELDS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return nil, fmt.Errorf("list form fields: %w", err)
	}

	fields, err = form.ListFormFields(c, ctx)
	if err != nil {
		return nil, fmt.Errorf("list form fields: %w", err)
	}
	return fields, nil
}

// RemoveFormFields deletes form fields in rs, writes the result to w and supports cancellation.
func RemoveFormFields(c context.Context, rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if err := validateNoEmptyStrings(fieldIDsOrNames, "form field ID or name"); err != nil {
		return err
	}

	conf = operationConfiguration(conf, model.REMOVEFORMFIELDS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("remove form fields: %w", err)
	}

	ok, err := form.RemoveFormFields(c, ctx, fieldIDsOrNames)
	if err != nil {
		return fmt.Errorf("remove form fields: update fields: %w", err)
	}
	if !ok {
		return fmt.Errorf("remove form fields: %w", ErrNoFormFieldsAffected)
	}

	if err := Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("remove form fields: write output: %w", err)
	}
	return nil
}

type formFieldMutation func(
	context.Context,
	io.ReadSeeker,
	io.Writer,
	[]string,
	*model.Configuration,
) error

func mutateFormFieldsFile(c context.Context, inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration, operation string, mutate formFieldMutation) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	if err = mutate(c, f1, f2, fieldIDsOrNames, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}

// RemoveFormFieldsFile deletes form fields in inFile, writes the result to outFile and supports cancellation.
func RemoveFormFieldsFile(c context.Context, inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) error {
	return mutateFormFieldsFile(
		c, inFile, outFile, fieldIDsOrNames, conf, "remove form fields", RemoveFormFields,
	)
}

// LockFormFields turns form fields in rs into read-only, writes the result to w and supports cancellation.
func LockFormFields(c context.Context, rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if err := validateNoEmptyStrings(fieldIDsOrNames, "form field ID or name"); err != nil {
		return err
	}

	conf = operationConfiguration(conf, model.LOCKFORMFIELDS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("lock form fields: %w", err)
	}

	ok, err := form.LockFormFields(c, ctx, fieldIDsOrNames)
	if err != nil {
		return fmt.Errorf("lock form fields: update fields: %w", err)
	}
	if !ok {
		return fmt.Errorf("lock form fields: %w", ErrNoFormFieldsAffected)
	}

	if err := Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("lock form fields: write output: %w", err)
	}
	return nil
}

// LockFormFieldsFile turns form fields of inFile into read-only, writes the result to outFile
// and supports cancellation.
func LockFormFieldsFile(c context.Context, inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) error {
	return mutateFormFieldsFile(
		c, inFile, outFile, fieldIDsOrNames, conf, "lock form fields", LockFormFields,
	)
}

// UnlockFormFields makes form fields in rs writable, writes the result to w and supports cancellation.
func UnlockFormFields(c context.Context, rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if err := validateNoEmptyStrings(fieldIDsOrNames, "form field ID or name"); err != nil {
		return err
	}

	conf = operationConfiguration(conf, model.UNLOCKFORMFIELDS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("unlock form fields: %w", err)
	}

	ok, err := form.UnlockFormFields(c, ctx, fieldIDsOrNames)
	if err != nil {
		return fmt.Errorf("unlock form fields: update fields: %w", err)
	}
	if !ok {
		return fmt.Errorf("unlock form fields: %w", ErrNoFormFieldsAffected)
	}

	if err := Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("unlock form fields: write output: %w", err)
	}
	return nil
}

// UnlockFormFieldsFile makes form fields of inFile writable, writes the result to outFile and supports cancellation.
func UnlockFormFieldsFile(c context.Context, inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) error {
	return mutateFormFieldsFile(
		c, inFile, outFile, fieldIDsOrNames, conf, "unlock form fields", UnlockFormFields,
	)
}

// ResetFormFields resets form fields of rs, writes the result to w and supports cancellation.
func ResetFormFields(c context.Context, rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if err := validateNoEmptyStrings(fieldIDsOrNames, "form field ID or name"); err != nil {
		return err
	}

	conf = operationConfiguration(conf, model.RESETFORMFIELDS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("reset form fields: %w", err)
	}

	ok, err := form.ResetFormFields(c, ctx, fieldIDsOrNames)
	if err != nil {
		return fmt.Errorf("reset form fields: update fields: %w", err)
	}
	if !ok {
		return fmt.Errorf("reset form fields: %w", ErrNoFormFieldsAffected)
	}

	if err := Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("reset form fields: write output: %w", err)
	}
	return nil
}

// ResetFormFieldsFile resets form fields of inFile, writes the result to outFile and supports cancellation.
func ResetFormFieldsFile(c context.Context, inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) error {
	return mutateFormFieldsFile(
		c, inFile, outFile, fieldIDsOrNames, conf, "reset form fields", ResetFormFields,
	)
}

// ExportForm extracts form data originating from source from rs and supports cancellation.
func ExportForm(c context.Context, rs io.ReadSeeker, source string, conf *model.Configuration) (formGroup *form.FormGroup, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.EXPORTFORMFIELDS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return nil, fmt.Errorf("export form: %w", err)
	}

	formGroup, err = exportedFormGroup(c, ctx, source)
	if err != nil {
		return nil, fmt.Errorf("export form: collect data: %w", err)
	}

	return formGroup, nil
}

func exportedFormGroup(c context.Context, ctx *model.Context, source string) (*form.FormGroup, error) {
	formGroup, ok, err := form.ExportForm(c, ctx.XRefTable, source)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNoFormFieldsAffected
	}
	return formGroup, nil
}

type formJSONExporter func(context.Context, *model.XRefTable, string, io.Writer) (bool, error)

func exportFormJSONResultUsing(c context.Context, xRefTable *model.XRefTable, source string, w io.Writer, export formJSONExporter) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	ok, err := export(c, xRefTable, source, w)
	if err != nil {
		return fmt.Errorf("export form: %w", err)
	}
	if !ok {
		return fmt.Errorf("export form: collect data: %w", ErrNoFormFieldsAffected)
	}
	return contextutil.Check(c)
}

// ExportFormJSON extracts form data originating from source from rs, writes the result to w and supports cancellation.
func ExportFormJSON(c context.Context, rs io.ReadSeeker, w io.Writer, source string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingJSONWriter
	}

	conf = operationConfiguration(conf, model.EXPORTFORMFIELDS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("export form: %w", err)
	}

	return exportFormJSONResultUsing(c, ctx.XRefTable, source, w, form.ExportFormJSON)
}

// ExportFormFile extracts form data from inFilePDF, writes the result to outFileJSON and supports cancellation.
func ExportFormFile(c context.Context, inFilePDF, outFileJSON string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = ExportFormJSON(c, f1, f2, inFilePDF, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
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

func fillPostProc(c context.Context, ctx *model.Context, pp []*model.Page) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if _, _, err := create.UpdatePageTree(c, ctx, pp, nil); err != nil {
		return fmt.Errorf("fill form: update page tree: %w", err)
	}
	if err := ValidateContext(c, ctx); err != nil {
		return fmt.Errorf("fill form: validate output: %w", err)
	}
	return nil
}

func formGroupFromReader(c context.Context, rd io.Reader) (*form.FormGroup, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := copyStream(c, &buf, rd); err != nil {
		return nil, fmt.Errorf("fill form: read form data: %w", err)
	}
	bb := buf.Bytes()

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

// FillForm populates the form rs with data from rd, writes the result to w and supports cancellation.
func FillForm(c context.Context, rs io.ReadSeeker, rd io.Reader, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if rd == nil {
		return fmt.Errorf("fill form: %w", ErrMissingFormInput)
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	conf = operationConfiguration(conf, model.FILLFORMFIELDS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("fill form: %w", err)
	}

	// TODO not necessarily so
	ctx.RemoveSignature()

	formGroup, err := formGroupFromReader(c, rd)
	if err != nil {
		return err
	}

	f, err := validatedFillForm(formGroup)
	if err != nil {
		return err
	}
	if err := contextutil.Check(c); err != nil {
		return err
	}

	ok, pp, err := form.FillForm(c, ctx, form.FillDetails(&f, nil), f.Pages, form.JSON)
	if err != nil {
		return fmt.Errorf("fill form: fill fields: %w", err)
	}
	if !ok {
		return fmt.Errorf("fill form: %w", ErrNoFormFieldsAffected)
	}

	if err := fillPostProc(c, ctx, pp); err != nil {
		return err
	}

	if err := Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("fill form: write output: %w", err)
	}
	return nil
}

// FillFormFile populates the form inFilePDF with data from inFileJSON,
// writes the result to outFilePDF and supports cancellation.
func FillFormFile(c context.Context, inFilePDF, inFileJSON, outFilePDF string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	if err = FillForm(c, f1, f0, f2, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

func parseFormGroup(c context.Context, rd io.Reader) (*form.FormGroup, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := copyStream(c, &buf, rd); err != nil {
		return nil, fmt.Errorf("multi-fill form: read form data: %w", err)
	}
	bb := buf.Bytes()

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
func rollbackMultiFillOutputs(c context.Context, outFiles []string) error {
	var errs []error
	for i, fileName := range outFiles {
		op := fmt.Sprintf("multi-fill form: remove intermediate %d %s", i+1, fileName)
		errs = append(errs, removeFile(fileName, op))
	}
	return errors.Join(contextutil.Check(c), errors.Join(errs...))
}

func mergeForms(c context.Context, outDir, fileName string, outFiles []string, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	fileName = sanitizeFilenamePart(fileName, "form")
	outFile := filepath.Join(outDir, fileName+".pdf")
	if err := MergeCreateFile(c, outFiles, outFile, false, conf); err != nil {
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

func multiFillPostProcess(c context.Context, ctx *model.Context, pp []*model.Page, op string, validate bool) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if _, _, err := create.UpdatePageTree(c, ctx, pp, nil); err != nil {
		return fmt.Errorf("%s: update page tree: %w", op, err)
	}
	if validate {
		if err := ValidateContext(c, ctx); err != nil {
			return fmt.Errorf("%s: validate output: %w", op, err)
		}
	}
	return contextutil.Check(c)
}

type formContextWriter func(context.Context, *model.Context, io.Writer) error

func writeMultiFillOutputUsing(c context.Context, ctx *model.Context, outFile, op string, writeContext formContextWriter) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	staged, err := openStagedOutput(nil, "", outFile, op)
	if err != nil {
		return fmt.Errorf("%s: create output %s: %w", op, outFile, err)
	}

	f := staged.output.file
	staged.removeContext = op + ": remove temporary output"
	staged.replaceContext = fmt.Sprintf("%s: replace output %s", op, outFile)
	if err := writeContext(c, ctx, f); err != nil {
		return staged.cleanup(fmt.Errorf("%s: write output %s: %w", op, outFile, err))
	}
	if err := contextutil.Check(c); err != nil {
		return staged.cleanup(err)
	}
	return staged.commit()
}

func multiFillJSONForm(c context.Context, inFilePDF string, f form.Form, outDir, fileName string, formNr int, conf *model.Configuration, writeContext formContextWriter) (outFile string, err error) {
	if err := contextutil.Check(c); err != nil {
		return "", err
	}
	op := fmt.Sprintf("multi-fill form %d", formNr)
	if err := validateFormData(f); err != nil {
		return "", fmt.Errorf("%s: validate form data: %w", op, err)
	}
	rs, err := os.Open(inFilePDF)
	if err != nil {
		return "", fmt.Errorf("%s: open input %s: %w", op, inFilePDF, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(rs, op+": close input"))
	}()

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if err := validateOptionValues(f); err != nil {
		return "", fmt.Errorf("%s: validate option values: %w", op, err)
	}
	if err := contextutil.Check(c); err != nil {
		return "", err
	}

	ok, pp, err := form.FillForm(c, ctx, form.FillDetails(&f, nil), f.Pages, form.JSON)
	if err != nil {
		return "", fmt.Errorf("%s: fill fields: %w", op, err)
	}
	if !ok {
		return "", fmt.Errorf("%s: %w", op, ErrNoFormFieldsAffected)
	}

	if err := multiFillPostProcess(c, ctx, pp, op, conf.PostProcessValidate); err != nil {
		return "", err
	}

	outFile = multiFillJSONOutputFile(outDir, fileName, f.FileName, formNr)
	if err := writeMultiFillOutputUsing(c, ctx, outFile, op, writeContext); err != nil {
		return "", err
	}
	return outFile, nil
}

func multiFillFormJSONUsing(c context.Context, inFilePDF string, rd io.Reader, outDir, fileName string, merge bool, conf *model.Configuration, writeContext formContextWriter) (err error) {
	formGroup, err := parseFormGroup(c, rd)
	if err != nil {
		return err
	}

	var outFiles []string
	if merge {
		defer func() {
			err = errors.Join(err, rollbackMultiFillOutputs(c, outFiles))
		}()
	}

	for i, f := range formGroup.Forms {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		outFile, fillErr := multiFillJSONForm(
			c, inFilePDF, f, outDir, fileName, i+1, conf, writeContext,
		)
		if outFile != "" {
			outFiles = append(outFiles, outFile)
		}
		if fillErr != nil {
			return fillErr
		}
	}

	if merge {
		return mergeForms(c, outDir, fileName, outFiles, conf)
	}
	return contextutil.Check(c)
}

func parseCSVLines(c context.Context, rd io.Reader) ([][]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	// Does NOT do any fieldtype checking!
	// Don't use unless you know your form anatomy inside out!

	// The first row is expected to hold the fieldIDs/fieldNames of the fields to be filled - the only form metadata needed
	// for this usecase.
	// The remaining rows are the corresponding data tuples.
	// Each row results in one separate PDF form written to outDir.

	// fieldName1	fieldName2	fieldName3	fieldName4
	// John			Doe			1.1.2000	male
	// Jane			Doe			1.1.2000	female
	// Jacky		Doe			1.1.2000	non-binary

	var buf bytes.Buffer
	if err := copyStream(c, &buf, rd); err != nil {
		return nil, fmt.Errorf("multi-fill form: read CSV: %w", err)
	}
	csvLines, err := csv.NewReader(&buf).ReadAll()
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
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		if fieldName == "" || fieldName == "*" {
			return nil, fmt.Errorf("multi-fill form: parse CSV header column %d: %w", i+1, ErrInvalidCSV)
		}
	}

	return csvLines, nil
}

func multiFillCSVRecord(c context.Context, inFilePDF string, fieldNames, formRecord []string, outDir, fileName string, recordNr, rowNr int, conf *model.Configuration, writeContext formContextWriter) (outFile string, err error) {
	if err := contextutil.Check(c); err != nil {
		return "", err
	}
	op := fmt.Sprintf("multi-fill CSV row %d", rowNr)
	f, err := os.Open(inFilePDF)
	if err != nil {
		return "", fmt.Errorf("%s: open input %s: %w", op, inFilePDF, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(f, op+": close input"))
	}()

	ctx, err := ReadValidateAndOptimize(c, f, conf, nil)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	fieldMap, imgPageMap, requested, err := form.FieldMap(fieldNames, formRecord)
	if err != nil {
		return "", fmt.Errorf("%s: map fields: %w", op, errors.Join(ErrInvalidCSV, err))
	}
	if err := contextutil.Check(c); err != nil {
		return "", err
	}

	ok, pp, err := form.FillForm(c, ctx, form.FillDetails(nil, fieldMap), imgPageMap, form.CSV)
	if err != nil {
		return "", fmt.Errorf("%s: fill fields: %w", op, err)
	}
	if !ok {
		return "", fmt.Errorf("%s: %w", op, ErrNoFormFieldsAffected)
	}

	if err := multiFillPostProcess(c, ctx, pp, op, conf.PostProcessValidate); err != nil {
		return "", err
	}

	outFile = multiFillCSVOutputFile(outDir, fileName, requested, recordNr)
	if err := writeMultiFillOutputUsing(c, ctx, outFile, op, writeContext); err != nil {
		return "", err
	}
	return outFile, nil
}

func multiFillFormCSVUsing(c context.Context, inFilePDF string, rd io.Reader, outDir, fileName string, merge bool, conf *model.Configuration, writeContext formContextWriter) (err error) {
	csvLines, err := parseCSVLines(c, rd)
	if err != nil {
		return err
	}

	fieldNames := csvLines[0]
	var outFiles []string
	if merge {
		defer func() {
			err = errors.Join(err, rollbackMultiFillOutputs(c, outFiles))
		}()
	}

	for i, formRecord := range csvLines[1:] {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		outFile, fillErr := multiFillCSVRecord(
			c, inFilePDF, fieldNames, formRecord, outDir, fileName, i+1, i+2, conf, writeContext,
		)
		if outFile != "" {
			outFiles = append(outFiles, outFile)
		}
		if fillErr != nil {
			return fillErr
		}
	}

	if merge {
		return mergeForms(c, outDir, fileName, outFiles, conf)
	}
	return contextutil.Check(c)
}

// MultiFillForm populates multiple instances of inFilePDF's form with data from rd, writes the result to outDir
// and supports cancellation.
func MultiFillForm(c context.Context, inFilePDF string, rd io.Reader, outDir, fileName string, format form.DataFormat, merge bool, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFilePDF == "" {
		return ErrMissingPDFInput
	}

	if rd == nil {
		return fmt.Errorf("multi-fill form: %w", ErrMissingFormInput)
	}

	if format != form.JSON && format != form.CSV {
		return fmt.Errorf("multi-fill form: %w: %d", ErrUnsupportedFormDataFormat, format)
	}

	conf = operationConfiguration(conf, model.MULTIFILLFORMFIELDS)

	fileName = strings.TrimSuffix(filepath.Base(fileName), ".pdf")
	fileName = sanitizeFilenamePart(fileName, "form")

	if format == form.JSON {
		return multiFillFormJSONUsing(c, inFilePDF, rd, outDir, fileName, merge, conf, WriteContext)
	}

	return multiFillFormCSVUsing(c, inFilePDF, rd, outDir, fileName, merge, conf, WriteContext)
}

// MultiFillFormFile populates multiple instances of inFilePDF's form with data from inFileData
// and writes the result to outDir.
// The output file will be written to outFilePDF with incrementing numerical suffix unless
// the input JSON uses "filename" or the input CSV contains a @filename field.
// MultiFillFormFile supports cancellation.
func MultiFillFormFile(c context.Context, inFilePDF, inFileData, outDir, outFilePDF string, merge bool, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	outFileBase := filepath.Base(outFilePDF)

	err = MultiFillForm(c, inFilePDF, f, outDir, outFileBase, format, merge, conf)
	return err
}
