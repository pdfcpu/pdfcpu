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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ViewerPreferences returns rs's viewer preferences and supports cancellation.
func ViewerPreferences(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (vp *model.ViewerPreferences, v *model.Version, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, nil, err
	}
	if rs == nil {
		return nil, nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.LISTVIEWERPREFERENCES)

	ctx, err := ReadAndValidate(c, rs, conf)
	if err != nil {
		return nil, nil, fmt.Errorf("list viewer preferences: prepare PDF context: %w", err)
	}

	version := ctx.XRefTable.Version()

	return ctx.ViewerPref, &version, contextutil.Check(c)
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

func marshalViewerPreferencesJSON(c context.Context, vp *model.ViewerPreferences) (string, error) {
	if err := contextutil.Check(c); err != nil {
		return "", err
	}
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
	return string(bb), contextutil.Check(c)
}

// ViewerPreferencesFile returns inFile's viewer preferences and supports cancellation.
func ViewerPreferencesFile(c context.Context, inFile string, all bool, conf *model.Configuration) (vp *model.ViewerPreferences, err error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
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

	vp, version, err := ViewerPreferences(c, f, conf)
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

// ListViewerPreferences returns rs's viewer preferences and supports cancellation.
func ListViewerPreferences(c context.Context, rs io.ReadSeeker, all bool, conf *model.Configuration) (ss []string, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.LISTVIEWERPREFERENCES)

	ctx, err := ReadAndValidate(c, rs, conf)
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
	return vp.List(), contextutil.Check(c)
}

// ListViewerPreferencesJSON returns rs's viewer preferences in JSON and supports cancellation.
func ListViewerPreferencesJSON(c context.Context, rs io.ReadSeeker, all bool, conf *model.Configuration) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	vp, version, err := ViewerPreferences(c, rs, conf)
	if err != nil {
		return nil, err
	}

	vp, err = viewerPreferencesForListing(vp, *version, all)
	if err != nil {
		return nil, err
	}

	s, err := marshalViewerPreferencesJSON(c, vp)
	if err != nil {
		return nil, err
	}
	return []string{s}, nil
}

// ListViewerPreferencesFileJSON lists inFile's viewer preferences in JSON and supports cancellation.
func ListViewerPreferencesFileJSON(c context.Context, inFile string, all bool, conf *model.Configuration) (ss []string, err error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
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

	return ListViewerPreferencesJSON(c, f, all, conf)
}

// ListViewerPreferencesFile lists inFile's viewer preferences and supports cancellation.
func ListViewerPreferencesFile(c context.Context, inFile string, all, json bool, conf *model.Configuration) (ss []string, err error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if inFile == "" {
		return nil, ErrMissingPDFInput
	}

	if json {
		return ListViewerPreferencesFileJSON(c, inFile, all, conf)
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list viewer preferences: open input %s: %w", inFile, err)
	}
	defer func() {
		err = closeViewerPreferencesInput(err, f, "list viewer preferences: close input")
	}()

	return ListViewerPreferences(c, f, all, conf)
}

// SetViewerPreferences sets rs's viewer preferences,
// writes the result to w and supports cancellation.
func SetViewerPreferences(c context.Context, rs io.ReadSeeker, w io.Writer, vp model.ViewerPreferences, conf *model.Configuration) (err error) {
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

	conf = operationConfiguration(conf, model.SETVIEWERPREFERENCES)

	ctx, err := ReadAndValidate(c, rs, conf)
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

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("set viewer preferences: write output: %w", err)
	}
	return nil
}

// SetViewerPreferencesFromJSONBytes sets rs's viewer preferences corresponding to jsonBytes,
// writes the result to w and supports cancellation.
func SetViewerPreferencesFromJSONBytes(c context.Context, rs io.ReadSeeker, w io.Writer, jsonBytes []byte, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	if err := contextutil.Check(c); err != nil {
		return err
	}
	return SetViewerPreferences(c, rs, w, vp, conf)
}

// SetViewerPreferencesFromJSONReader sets rs's viewer preferences corresponding to rd,
// writes the result to w and supports cancellation.
func SetViewerPreferencesFromJSONReader(c context.Context, rs io.ReadSeeker, w io.Writer, rd io.Reader, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
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
	if err := copyStream(c, &buf, rd); err != nil {
		return fmt.Errorf("set viewer preferences: read JSON: %w", err)
	}

	return SetViewerPreferencesFromJSONBytes(c, rs, w, buf.Bytes(), conf)
}

// SetViewerPreferencesFile sets inFile's viewer preferences,
// writes the result to outFile and supports cancellation.
func SetViewerPreferencesFile(c context.Context, inFile, outFile string, vp model.ViewerPreferences, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	if err = SetViewerPreferences(c, f1, f2, vp, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

// SetViewerPreferencesFileFromJSONBytes sets inFile's viewer preferences corresponding to jsonBytes,
// writes the result to outFile and supports cancellation.
func SetViewerPreferencesFileFromJSONBytes(c context.Context, inFile, outFile string, jsonBytes []byte, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	if err = SetViewerPreferencesFromJSONBytes(c, f1, f2, jsonBytes, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

func readViewerPreferencesJSON(c context.Context, fileName string) (bb []byte, err error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}

	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("set viewer preferences: read JSON %s: %w", fileName, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(f, "set viewer preferences: close JSON input"))
	}()

	var buf bytes.Buffer
	if err := copyStream(c, &buf, f); err != nil {
		return nil, fmt.Errorf("set viewer preferences: read JSON %s: %w", fileName, err)
	}

	return buf.Bytes(), contextutil.Check(c)
}

// SetViewerPreferencesFileFromJSONFile sets inFile's viewer preferences corresponding to inFileJSON,
// writes the result to outFile and supports cancellation.
func SetViewerPreferencesFileFromJSONFile(c context.Context, inFilePDF, outFilePDF, inFileJSON string, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFilePDF == "" {
		return ErrMissingPDFInput
	}

	if inFileJSON == "" {
		return ErrMissingJSONInput
	}

	bb, err := readViewerPreferencesJSON(c, inFileJSON)
	if err != nil {
		return err
	}

	return SetViewerPreferencesFileFromJSONBytes(c, inFilePDF, outFilePDF, bb, conf)
}

// ResetViewerPreferences resets rs's viewer preferences and writes the result to w.
// If rs has no viewer preferences, it still writes the unchanged PDF and returns success.
func ResetViewerPreferences(c context.Context, rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
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

	conf = operationConfiguration(conf, model.RESETVIEWERPREFERENCES)

	ctx, err := ReadAndValidate(c, rs, conf)
	if err != nil {
		return fmt.Errorf("reset viewer preferences: prepare PDF context: %w", err)
	}

	if ctx.ViewerPref != nil {
		delete(ctx.RootDict, "ViewerPreferences")
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("reset viewer preferences: write output: %w", err)
	}
	return nil
}

// ResetViewerPreferencesFile resets inFile's viewer preferences and writes the result to outFile.
// If inFile has no viewer preferences, it still writes the unchanged PDF and returns success.
func ResetViewerPreferencesFile(c context.Context, inFile, outFile string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	if err = ResetViewerPreferences(c, f1, f2, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}
