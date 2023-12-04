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
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/create"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
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
func FormFields(rs io.ReadSeeker, conf *model.Configuration) ([]form.Field, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: FormFields: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTFORMFIELDS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return nil, err
	}

	fields, _, err := form.FormFields(ctx)

	return fields, err
}

// RemoveFormFields deletes form fields in rs and writes the result to w.
func RemoveFormFields(rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: RemoveFormFields: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEFORMFIELDS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	ok, err := form.RemoveFormFields(ctx, fieldIDsOrNames)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoFormFieldsAffected
	}

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// RemoveFormFieldsFile deletes form fields in inFile and writes the result to outFile.
func RemoveFormFieldsFile(inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	logWritingTo(outFile)

	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
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

	return RemoveFormFields(f1, f2, fieldIDsOrNames, conf)
}

// LockFormFields turns form fields in rs into read-only and writes the result to w.
func LockFormFields(rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: LockFormFields: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LOCKFORMFIELDS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	ok, err := form.LockFormFields(ctx, fieldIDsOrNames)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoFormFieldsAffected
	}

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// LockFormFieldsFile turns form fields of inFile into read-only and writes the result to outFile.
func LockFormFieldsFile(inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	logWritingTo(outFile)

	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
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

	return LockFormFields(f1, f2, fieldIDsOrNames, conf)
}

// UnlockFormFields makess form fields in rs writeable and writes the result to w.
func UnlockFormFields(rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: UnlockFormFields: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.UNLOCKFORMFIELDS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	ok, err := form.UnlockFormFields(ctx, fieldIDsOrNames)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoFormFieldsAffected
	}

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// UnlockFormFieldsFile makes form fields of inFile writeable and writes the result to outFile.
func UnlockFormFieldsFile(inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	logWritingTo(outFile)

	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
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

	return UnlockFormFields(f1, f2, fieldIDsOrNames, conf)
}

// ResetFormFields resets form fields of rs and writes the result to w.
func ResetFormFields(rs io.ReadSeeker, w io.Writer, fieldIDsOrNames []string, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: ResetFormFields: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.RESETFORMFIELDS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	ok, err := form.ResetFormFields(ctx, fieldIDsOrNames)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoFormFieldsAffected
	}

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	return WriteContext(ctx, w)
}

// ResetFormFieldsFile resets form fields of inFile and writes the result to outFile.
func ResetFormFieldsFile(inFile, outFile string, fieldIDsOrNames []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	logWritingTo(outFile)

	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
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

	return ResetFormFields(f1, f2, fieldIDsOrNames, conf)
}

// ExportForm extracts form data originating from source from rs.
func ExportForm(rs io.ReadSeeker, source string, conf *model.Configuration) (*form.FormGroup, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: ExportForm: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXPORTFORMFIELDS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	if err := ctx.EnsurePageCount(); err != nil {
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
func ExportFormJSON(rs io.ReadSeeker, w io.Writer, source string, conf *model.Configuration) error {
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

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
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

	if f1, err = os.Open(inFilePDF); err != nil {
		return err
	}

	if f2, err = os.Create(outFileJSON); err != nil {
		f1.Close()
		return err
	}
	logWritingTo(outFileJSON)

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
	}()

	return ExportFormJSON(f1, f2, inFilePDF, conf)
}

func validateComboBoxValues(f form.Form) error {
	for _, cb := range f.ComboBoxes {
		if cb.Value == "" || cb.Editable {
			continue
		}
		if len(cb.Options) > 0 {
			if !types.MemberOf(cb.Value, cb.Options) {
				return errors.Errorf("pdfcpu: fill field name: \"%s\" unknown value: \"%s\" - options: %v\n", cb.Name, cb.Value, cb.Options)
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
					return errors.Errorf("pdfcpu: fill field name: \"%s\" unknown value: \"%s\" - options: %v\n", lb.Name, v, lb.Options)
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
				return errors.Errorf("pdfcpu: fill field name: \"%s\" unknown value: \"%s\" - options: %v\n", rbg.Name, rbg.Value, rbg.Options)
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

func fillPostProc(ctx *model.Context, pp []*model.Page, conf *model.Configuration) error {
	if _, _, err := create.UpdatePageTree(ctx, pp, nil); err != nil {
		return err
	}

	if conf.ValidationMode != model.ValidationNone {
		return ValidateContext(ctx)
	}

	return nil
}

// FillForm populates the form rs with data from rd and writes the result to w.
func FillForm(rs io.ReadSeeker, rd io.Reader, w io.Writer, conf *model.Configuration) error {
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

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return err
	}

	ctx.RemoveSignature()

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

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

	if log.CLIEnabled() {
		log.CLI.Println("filling...")
	}

	ok, pp, err := form.FillForm(ctx, form.FillDetails(&f, nil), f.Pages, form.JSON)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoFormFieldsAffected
	}

	if err := fillPostProc(ctx, pp, conf); err != nil {
		return err
	}

	return WriteContext(ctx, w)
}

// FillFormFile populates the form inFilePDF with data from inFileJSON and writes the result to outFilePDF.
func FillFormFile(inFilePDF, inFileJSON, outFilePDF string, conf *model.Configuration) (err error) {
	var f0, f1, f2 *os.File

	if f0, err = os.Open(inFileJSON); err != nil {
		return err
	}

	if f1, err = os.Open(inFilePDF); err != nil {
		f0.Close()
		return err
	}
	rs := f1

	tmpFile := inFilePDF + ".tmp"
	if outFilePDF != "" && inFilePDF != outFilePDF {
		tmpFile = outFilePDF
	}
	logWritingTo(outFilePDF)

	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		f0.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			f0.Close()
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

	return FillForm(rs, f0, f2, conf)
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

		ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
		if err != nil {
			return err
		}

		if err := ctx.EnsurePageCount(); err != nil {
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

		if conf.ValidationMode != model.ValidationNone {
			if err = ValidateContext(ctx); err != nil {
				return err
			}
		}

		outFile := filepath.Join(outDir, fmt.Sprintf("%s_%02d.pdf", fileName, i+1))
		if log.CLIEnabled() {
			log.CLI.Printf("writing %s\n", outFile)
		}

		if err := WriteContextFile(ctx, outFile); err != nil {
			return err
		}
		outFiles = append(outFiles, outFile)
	}

	if merge {
		if err := mergeForms(outDir, fileName, outFiles, conf); err != nil {
			return err
		}
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

		ctx, _, _, _, err := ReadValidateAndOptimize(f, conf, time.Now())
		if err != nil {
			return err
		}

		if err := ctx.EnsurePageCount(); err != nil {
			return err
		}

		fieldMap, imgPageMap, err := form.FieldMap(fieldNames, formRecord)
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

		if conf.ValidationMode != model.ValidationNone {
			if err = ValidateContext(ctx); err != nil {
				return err
			}
		}

		outFile := filepath.Join(outDir, fmt.Sprintf("%s_%02d.pdf", fileName, i+1))
		logWritingTo(outFile)
		if err := WriteContextFile(ctx, outFile); err != nil {
			return err
		}
		outFiles = append(outFiles, outFile)
	}

	if merge {
		if err := mergeForms(outDir, fileName, outFiles, conf); err != nil {
			return err
		}
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

	if format == form.JSON {
		return multiFillFormJSON(inFilePDF, rd, outDir, fileName, merge, conf)
	}

	return multiFillFormCSV(inFilePDF, rd, outDir, fileName, merge, conf)
}

// MultiFillFormFile populates multiples instances of inFilePDFs form with data from inFileData and writes the result to outDir.
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
