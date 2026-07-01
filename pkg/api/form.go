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
	"encoding/csv"
	"encoding/json"
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
	"github.com/pkg/errors"
)

var (
	ErrNoFormData           = errors.New("pdfcpu: missing form data")
	ErrNoFormFieldsAffected = errors.New("pdfcpu: no form fields affected")
	ErrInvalidCSV           = errors.New("pdfcpu: invalid csv input file")
	ErrInvalidJSON          = errors.New("pdfcpu: invalid JSON encoding")
)

// FormFields returns all form fields of rs.
func FormFields(rs io.ReadSeeker, conf *model.Configuration) (fields []form.Field, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, errors.New("pdfcpu: FormFields: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTFORMFIELDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, err
	}

	fields, _, err = form.FormFields(ctx)

	return fields, err
}

// RemoveFormFields deletes form fields in rs and writes the result to w.
func RemoveFormFields(rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: RemoveFormFields: missing rs")
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
		return err
	}

	ok, err := form.RemoveFormFields(ctx, fieldIDsOrNames)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoFormFieldsAffected
	}

	return Write(ctx, w, conf)
}

// RemoveFormFieldsFile deletes form fields in inFile and writes the result to outFile.
func RemoveFormFieldsFile(inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := validateNoEmptyStrings(fieldIDsOrNames, "form field ID or name"); err != nil {
		return err
	}

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	logWritingTo(outFile)

	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	if err = RemoveFormFields(f1, f2, fieldIDsOrNames, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// LockFormFields turns form fields in rs into read-only and writes the result to w.
func LockFormFields(rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: LockFormFields: missing rs")
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
		return err
	}

	ok, err := form.LockFormFields(ctx, fieldIDsOrNames)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoFormFieldsAffected
	}

	return Write(ctx, w, conf)
}

// LockFormFieldsFile turns form fields of inFile into read-only and writes the result to outFile.
func LockFormFieldsFile(inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := validateNoEmptyStrings(fieldIDsOrNames, "form field ID or name"); err != nil {
		return err
	}

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	logWritingTo(outFile)

	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	if err = LockFormFields(f1, f2, fieldIDsOrNames, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// UnlockFormFields makess form fields in rs writeable and writes the result to w.
func UnlockFormFields(rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: UnlockFormFields: missing rs")
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
		return err
	}

	ok, err := form.UnlockFormFields(ctx, fieldIDsOrNames)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoFormFieldsAffected
	}

	return Write(ctx, w, conf)
}

// UnlockFormFieldsFile makes form fields of inFile writeable and writes the result to outFile.
func UnlockFormFieldsFile(inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := true

	if err := validateNoEmptyStrings(fieldIDsOrNames, "form field ID or name"); err != nil {
		return err
	}

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	logWritingTo(outFile)

	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	if err = UnlockFormFields(f1, f2, fieldIDsOrNames, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// ResetFormFields resets form fields of rs and writes the result to w.
func ResetFormFields(rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: ResetFormFields: missing rs")
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
		return err
	}

	ok, err := form.ResetFormFields(ctx, fieldIDsOrNames)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoFormFieldsAffected
	}

	return Write(ctx, w, conf)
}

// ResetFormFieldsFile resets form fields of inFile and writes the result to outFile.
func ResetFormFieldsFile(inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := validateNoEmptyStrings(fieldIDsOrNames, "form field ID or name"); err != nil {
		return err
	}

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	logWritingTo(outFile)
	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	if err = ResetFormFields(f1, f2, fieldIDsOrNames, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// ExportForm extracts form data originating from source from rs.
func ExportForm(rs io.ReadSeeker, source string, conf *model.Configuration) (formGroup *form.FormGroup, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, errors.New("pdfcpu: ExportForm: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXPORTFORMFIELDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, err
	}

	formGroup, ok, err := form.ExportForm(ctx.XRefTable, source)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNoFormFieldsAffected
	}

	return formGroup, nil
}

// ExportFormJSON extracts form data originating from source from rs and writes the result to w.
func ExportFormJSON(rs io.ReadSeeker, w io.Writer, source string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: ExportFormJSON: missing rs")
	}

	if w == nil {
		return errors.New("pdfcpu: ExportFormJSON: missing w")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXPORTFORMFIELDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	ok, err := form.ExportFormJSON(ctx.XRefTable, source, w)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoFormFieldsAffected
	}

	return nil
}

// ExportFormFile extracts form data from inFilePDF and writes the result to outFileJSON.
func ExportFormFile(inFilePDF, outFileJSON string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if f1, err = os.Open(inFilePDF); err != nil {
		return err
	}

	if f2, err = os.Create(outFileJSON); err != nil {
		_ = f1.Close()
		return err
	}
	logWritingTo(outFileJSON)

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
	}()

	if err = ExportFormJSON(f1, f2, inFilePDF, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

func validateComboBoxValues(f form.Form) error {
	for _, cb := range f.ComboBoxes {
		if cb.Value == "" || cb.Editable {
			continue
		}
		if len(cb.Options) > 0 {
			if !types.MemberOf(cb.Value, cb.Options) {
				i, err := strconv.Atoi(cb.Value)
				if err == nil && i < len(cb.Options) {
					return nil
				}
				return errors.Errorf("pdfcpu: fill field name: \"%s\" unknown value: \"%s\" - options: [%v]\n", cb.Name, cb.Value, strings.Join(cb.Options, ", "))
			}
		}
	}
	return nil
}

func validateListBoxValues(f form.Form) error {
	for _, lb := range f.ListBoxes {
		if len(lb.Values) == 0 {
			continue
		}
		if len(lb.Options) > 0 {
			for _, v := range lb.Values {
				if !types.MemberOf(v, lb.Options) {
					i, err := strconv.Atoi(v)
					if err == nil && i < len(lb.Options) {
						return nil
					}
					return errors.Errorf("pdfcpu: fill field name: \"%s\" unknown value: \"%s\" - options: [%v]\n", lb.Name, v, strings.Join(lb.Options, ", "))
				}
			}
		}
	}
	return nil
}

func validateRadioButtonGroupValues(f form.Form) error {
	for _, rbg := range f.RadioButtonGroups {
		if rbg.Value == "" {
			continue
		}
		if len(rbg.Options) > 0 {
			if !types.MemberOf(rbg.Value, rbg.Options) {
				i, err := strconv.Atoi(rbg.Value)
				if err == nil && i < len(rbg.Options) {
					return nil
				}
				return errors.Errorf("pdfcpu: fill field name: \"%s\" unknown value: \"%s\" - options: [%v]\n", rbg.Name, rbg.Value, strings.Join(rbg.Options, ", "))
			}
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
		return err
	}

	return ValidateContext(ctx)
}

// FillForm populates the form rs with data from rd and writes the result to w.
func FillForm(rs io.ReadSeeker, rd io.Reader, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return errors.New("pdfcpu: FillForm: missing rs")
	}

	if rd == nil {
		return errors.New("pdfcpu: FillForm: missing rd")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.FILLFORMFIELDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	// TODO not necessarily so
	ctx.RemoveSignature()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rd); err != nil {
		return err
	}

	bb := buf.Bytes()

	if !json.Valid(bb) {
		return ErrInvalidJSON
	}

	formGroup := form.FormGroup{}

	if err := json.Unmarshal(bb, &formGroup); err != nil {
		return err
	}

	if len(formGroup.Forms) == 0 {
		return ErrNoFormData
	}

	f := formGroup.Forms[0]

	if err := validateOptionValues(f); err != nil {
		return err
	}

	ok, pp, err := form.FillForm(ctx, form.FillDetails(&f, nil), f.Pages, form.JSON)
	if err != nil {
		return err
	}
	if !ok {
		if log.CLIEnabled() {
			log.CLI.Println("nothing written")
		}
		return ErrNoFormFieldsAffected
	}

	if log.CLIEnabled() {
		log.CLI.Println("filling...")
	}

	if err := fillPostProc(ctx, pp); err != nil {
		return err
	}

	return Write(ctx, w, conf)
}

// FillFormFile populates the form inFilePDF with data from inFileJSON and writes the result to outFilePDF.
func FillFormFile(inFilePDF, inFileJSON, outFilePDF string, conf *model.Configuration) (err error) {
	var f0, f1, f2 *os.File
	ok := false

	if f0, err = os.Open(inFileJSON); err != nil {
		return err
	}

	if f1, err = os.Open(inFilePDF); err != nil {
		_ = f0.Close()
		return err
	}
	rs := f1

	tmpFile := ""
	if outFilePDF != "" && inFilePDF != outFilePDF {
		tmpFile = outFilePDF
	}
	logWritingTo(outFilePDF)
	if f2, tmpFile, err = createOutputFile(inFilePDF, tmpFile); err != nil {
		_ = f1.Close()
		_ = f0.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			_ = f0.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if err = f0.Close(); err != nil {
			return
		}
		if outFilePDF == "" || inFilePDF == outFilePDF {
			err = os.Rename(tmpFile, inFilePDF)
		}
	}()

	if err = FillForm(rs, f0, f2, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

func parseFormGroup(rd io.Reader) (*form.FormGroup, error) {
	formGroup := &form.FormGroup{}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rd); err != nil {
		return nil, err
	}

	bb := buf.Bytes()

	if !json.Valid(bb) {
		return nil, ErrInvalidJSON
	}

	if err := json.Unmarshal(bb, formGroup); err != nil {
		return nil, err
	}

	if len(formGroup.Forms) == 0 {
		return nil, ErrNoFormData
	}

	return formGroup, nil
}

func mergeForms(outDir, fileName string, outFiles []string, conf *model.Configuration) error {
	fileName = sanitizeFilenamePart(fileName, "form")
	outFile := filepath.Join(outDir, fileName+".pdf")
	if err := MergeCreateFile(outFiles, outFile, false, conf); err != nil {
		return err
	}
	if log.CLIEnabled() {
		log.CLI.Println("cleaning up...")
	}
	for _, fn := range outFiles {
		if err := os.Remove(fn); err != nil {
			return err
		}
	}
	return nil
}

func multiFillFormJSON(inFilePDF string, rd io.Reader, outDir, fileName string, merge bool, conf *model.Configuration) error {
	formGroup, err := parseFormGroup(rd)
	if err != nil {
		return err
	}

	var outFiles []string

	for i, f := range formGroup.Forms {

		rs, err := os.Open(inFilePDF)
		if err != nil {
			return err
		}
		defer rs.Close()

		ctx, err := ReadValidateAndOptimize(rs, conf)
		if err != nil {
			return err
		}

		ok, pp, err := form.FillForm(ctx, form.FillDetails(&f, nil), f.Pages, form.JSON)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNoFormFieldsAffected
		}

		if _, _, err := create.UpdatePageTree(ctx, pp, nil); err != nil {
			return err
		}

		if conf.PostProcessValidate {
			if err = ValidateContext(ctx); err != nil {
				return err
			}
		}

		outFile := f.FileName
		if outFile == "" {
			outFile = filepath.Join(outDir, fmt.Sprintf("%s_%02d.pdf", fileName, i+1))
		} else {
			outFile, err = sanitize.Path(outFile)
			if err != nil {
				outFile = fmt.Sprintf("form_%02d", i+1)
			}
			if !strings.HasSuffix(strings.ToLower(outFile), ".pdf") {
				outFile += ".pdf"
			}
			outFile = filepath.Join(outDir, fmt.Sprintf("%s", outFile))
		}

		logWritingTo(outFile)

		if err := WriteContextFile(ctx, outFile); err != nil {
			return err
		}
		outFiles = append(outFiles, outFile)
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
		return nil, err
	}

	if len(csvLines) < 2 {
		return nil, ErrInvalidCSV
	}

	fieldNames := csvLines[0]
	if len(fieldNames) == 0 {
		return nil, ErrInvalidCSV
	}

	return csvLines, nil
}

func multiFillFormCSV(inFilePDF string, rd io.Reader, outDir, fileName string, merge bool, conf *model.Configuration) error {
	csvLines, err := parseCSVLines(rd)
	if err != nil {
		return err
	}

	fieldNames := csvLines[0]
	var outFiles []string

	for i, formRecord := range csvLines[1:] {

		f, err := os.Open(inFilePDF)
		if err != nil {
			return err
		}
		defer f.Close()

		ctx, err := ReadValidateAndOptimize(f, conf)
		if err != nil {
			return err
		}

		fieldMap, imgPageMap, outFile, err := form.FieldMap(fieldNames, formRecord)
		if err != nil {
			return err
		}

		ok, pp, err := form.FillForm(ctx, form.FillDetails(nil, fieldMap), imgPageMap, form.CSV)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNoFormFieldsAffected
		}

		if _, _, err := create.UpdatePageTree(ctx, pp, nil); err != nil {
			return err
		}

		if conf.PostProcessValidate {
			if err = ValidateContext(ctx); err != nil {
				return err
			}
		}

		if outFile == "" {
			outFile = filepath.Join(outDir, fmt.Sprintf("%s_%02d.pdf", fileName, i+1))
		} else {
			outFile, err = sanitize.Path(outFile)
			if err != nil {
				outFile = fmt.Sprintf("form_%02d", i+1)
			}
			outFile = filepath.Join(outDir, fmt.Sprintf("%s", outFile))
		}

		logWritingTo(outFile)

		if err := WriteContextFile(ctx, outFile); err != nil {
			return err
		}
		outFiles = append(outFiles, outFile)
	}

	if merge {
		return mergeForms(outDir, fileName, outFiles, conf)
	}

	return nil
}

// MultiFillForm populates multiples instances of inFilePDF's form with data from rd and writes the result to outDir.
func MultiFillForm(inFilePDF string, rd io.Reader, outDir, fileName string, format form.DataFormat, merge bool, conf *model.Configuration) error {
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

// MultiFillFormFile populates multiples instances of inFilePDFs form with data from inFileData and writes the result to outDir.
// The output file will be written to outFilePDF with incrementing numerical suffix unless
// the input json uses "filename" or the input csv contains a @filename field.
func MultiFillFormFile(inFilePDF, inFileData, outDir, outFilePDF string, merge bool, conf *model.Configuration) (err error) {
	format := form.JSON
	if strings.HasSuffix(strings.ToLower(inFileData), ".csv") {
		format = form.CSV
	}

	var f *os.File

	if f, err = os.Open(inFileData); err != nil {
		return err
	}

	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()

	s := "JSON"
	if format == form.CSV {
		s = "CSV"
	}

	outFileBase := filepath.Base(outFilePDF)

	if log.CLIEnabled() {
		log.CLI.Printf("filling multiple forms via %s based on %s data from %s into %s/%s ...\n", inFilePDF, s, inFileData, outDir, outFileBase)
	}

	return MultiFillForm(inFilePDF, f, outDir, outFileBase, format, merge, conf)
}
