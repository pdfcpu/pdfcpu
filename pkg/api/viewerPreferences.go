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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ViewerPreferences returns rs's viewer preferences.
func ViewerPreferences(rs io.ReadSeeker, conf *model.Configuration) (vp *model.ViewerPreferences, v *model.Version, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTVIEWERPREFERENCES

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, nil, fmt.Errorf("list viewer preferences: prepare PDF context: %w", err)
	}

	version := ctx.XRefTable.Version()

	return ctx.ViewerPref, &version, nil
}

func viewerPreferencesForListing(vp *model.ViewerPreferences, version model.Version, all bool) (*model.ViewerPreferences, error) {
	if !all {
		return vp, nil
	}

	vp, err := model.ViewerPreferencesWithDefaults(vp, version)
	if err != nil {
		return nil, fmt.Errorf("list viewer preferences: apply defaults: %w", err)
	}
	return vp, nil
}

func marshalViewerPreferencesJSON(vp *model.ViewerPreferences) (string, error) {
	s := struct {
		Header     pdfcpu.Header            `json:"header"`
		ViewerPref *model.ViewerPreferences `json:"viewerPreferences"`
	}{
		Header:     pdfcpu.Header{Version: "pdfcpu " + model.VersionStr, Creation: time.Now().Format("2006-01-02 15:04:05 MST")},
		ViewerPref: vp,
	}

	bb, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return "", fmt.Errorf("list viewer preferences: encode JSON: %w", err)
	}
	return string(bb), nil
}

// ViewerPreferencesFile returns inFile's viewer preferences.
func ViewerPreferencesFile(inFile string, all bool, conf *model.Configuration) (vp *model.ViewerPreferences, err error) {
	if inFile == "" {
		return nil, ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list viewer preferences: open input %s: %w", inFile, err)
	}
	defer func() {
		err = closeViewerPreferencesInput(err, f, "list viewer preferences: close input")
	}()

	vp, version, err := ViewerPreferences(f, conf)
	if err != nil {
		return nil, err
	}

	return viewerPreferencesForListing(vp, *version, all)
}

func closeViewerPreferencesInput(err error, f *os.File, context string) error {
	closeErr := closeFile(f, context)
	if err != nil && closeErr != nil {
		return errors.Join(err, closeErr)
	}
	if err != nil {
		return err
	}
	return closeErr
}

// ListViewerPreferences returns rs's viewer preferences.
func ListViewerPreferences(rs io.ReadSeeker, all bool, conf *model.Configuration) (ss []string, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTVIEWERPREFERENCES

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("list viewer preferences: prepare PDF context: %w", err)
	}

	vp, err := viewerPreferencesForListing(ctx.ViewerPref, ctx.XRefTable.Version(), all)
	if err != nil {
		return nil, err
	}
	if vp == nil {
		return []string{"No viewer preferences available."}, nil
	}
	return vp.List(), nil
}

// ListViewerPreferencesJSON returns rs's viewer preferences in JSON.
func ListViewerPreferencesJSON(rs io.ReadSeeker, all bool, conf *model.Configuration) ([]string, error) {
	vp, version, err := ViewerPreferences(rs, conf)
	if err != nil {
		return nil, err
	}

	vp, err = viewerPreferencesForListing(vp, *version, all)
	if err != nil {
		return nil, err
	}

	s, err := marshalViewerPreferencesJSON(vp)
	if err != nil {
		return nil, err
	}
	return []string{s}, nil
}

// ListViewerPreferencesFileJSON lists inFile's viewer preferences in JSON.
func ListViewerPreferencesFileJSON(inFile string, all bool, conf *model.Configuration) (ss []string, err error) {
	if inFile == "" {
		return nil, ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list viewer preferences: open input %s: %w", inFile, err)
	}
	defer func() {
		err = closeViewerPreferencesInput(err, f, "list viewer preferences: close input")
	}()

	return ListViewerPreferencesJSON(f, all, conf)
}

// ListViewerPreferencesFile lists inFile's viewer preferences.
func ListViewerPreferencesFile(inFile string, all, json bool, conf *model.Configuration) (ss []string, err error) {
	if inFile == "" {
		return nil, ErrMissingPDFInput
	}

	if json {
		return ListViewerPreferencesFileJSON(inFile, all, conf)
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list viewer preferences: open input %s: %w", inFile, err)
	}
	defer func() {
		err = closeViewerPreferencesInput(err, f, "list viewer preferences: close input")
	}()

	return ListViewerPreferences(f, all, conf)
}

// SetViewerPreferences sets rs's viewer preferences and writes the result to w.
func SetViewerPreferences(rs io.ReadSeeker, w io.Writer, vp model.ViewerPreferences, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.SETVIEWERPREFERENCES

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return fmt.Errorf("set viewer preferences: prepare PDF context: %w", err)
	}

	version := ctx.XRefTable.Version()

	if err := vp.Validate(version); err != nil {
		return fmt.Errorf("set viewer preferences: validate: %w", err)
	}

	if ctx.ViewerPref == nil {
		ctx.ViewerPref = &vp
	} else {
		ctx.ViewerPref.Populate(&vp)
	}

	ctx.XRefTable.BindViewerPreferences()

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("set viewer preferences: write output: %w", err)
	}
	return nil
}

// SetViewerPreferencesFromJSONBytes sets rs's viewer preferences corresponding to jsonBytes and writes the result to w.
func SetViewerPreferencesFromJSONBytes(rs io.ReadSeeker, w io.Writer, jsonBytes []byte, conf *model.Configuration) error {
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	vp := model.ViewerPreferences{}
	if err := json.Unmarshal(jsonBytes, &vp); err != nil {
		return fmt.Errorf("set viewer preferences: decode JSON: %w", errors.Join(ErrInvalidJSON, err))
	}

	return SetViewerPreferences(rs, w, vp, conf)
}

// SetViewerPreferencesFromJSONReader sets rs's viewer preferences corresponding to rd and writes the result to w.
func SetViewerPreferencesFromJSONReader(rs io.ReadSeeker, w io.Writer, rd io.Reader, conf *model.Configuration) error {
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if rd == nil {
		return fmt.Errorf("set viewer preferences: read JSON: %w", ErrMissingJSONReader)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rd); err != nil {
		return fmt.Errorf("set viewer preferences: read JSON: %w", err)
	}

	return SetViewerPreferencesFromJSONBytes(rs, w, buf.Bytes(), conf)
}

// SetViewerPreferencesFile sets inFile's viewer preferences and writes the result to outFile.
func SetViewerPreferencesFile(inFile, outFile string, vp model.ViewerPreferences, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("set viewer preferences: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "set viewer preferences")
	if err != nil {
		return errors.Join(
			fmt.Errorf("set viewer preferences: create output: %w", err),
			closeFile(f1, "set viewer preferences: close input"),
		)
	}
	f2 = staged.output.file

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = SetViewerPreferences(f1, f2, vp, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// SetViewerPreferencesFileFromJSONBytes sets inFile's viewer preferences corresponding to jsonBytes and writes the result to outFile.
func SetViewerPreferencesFileFromJSONBytes(inFile, outFile string, jsonBytes []byte, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("set viewer preferences: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "set viewer preferences")
	if err != nil {
		return errors.Join(
			fmt.Errorf("set viewer preferences: create output: %w", err),
			closeFile(f1, "set viewer preferences: close input"),
		)
	}
	f2 = staged.output.file

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = SetViewerPreferencesFromJSONBytes(f1, f2, jsonBytes, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// SetViewerPreferencesFileFromJSONFile sets inFile's viewer preferences corresponding to inFileJSON and writes the result to outFile.
func SetViewerPreferencesFileFromJSONFile(inFilePDF, outFilePDF, inFileJSON string, conf *model.Configuration) error {
	if inFilePDF == "" {
		return ErrMissingPDFInput
	}

	if inFileJSON == "" {
		return ErrMissingJSONInput
	}

	bb, err := os.ReadFile(inFileJSON)
	if err != nil {
		return fmt.Errorf("set viewer preferences: read JSON %s: %w", inFileJSON, err)
	}

	return SetViewerPreferencesFileFromJSONBytes(inFilePDF, outFilePDF, bb, conf)
}

// ResetViewerPreferences resets rs's viewer preferences and writes the result to w.
// If rs has no viewer preferences, it still writes the unchanged PDF and returns success.
func ResetViewerPreferences(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.RESETVIEWERPREFERENCES

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return fmt.Errorf("reset viewer preferences: prepare PDF context: %w", err)
	}

	if ctx.ViewerPref != nil {
		delete(ctx.RootDict, "ViewerPreferences")
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("reset viewer preferences: write output: %w", err)
	}
	return nil
}

// ResetViewerPreferencesFile resets inFile's viewer preferences and writes the result to outFile.
// If inFile has no viewer preferences, it still writes the unchanged PDF and returns success.
func ResetViewerPreferencesFile(inFile, outFile string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("reset viewer preferences: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "reset viewer preferences")
	if err != nil {
		return errors.Join(
			fmt.Errorf("reset viewer preferences: create output: %w", err),
			closeFile(f1, "reset viewer preferences: close input"),
		)
	}
	f2 = staged.output.file

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = ResetViewerPreferences(f1, f2, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
