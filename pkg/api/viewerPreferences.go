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
	"io"
	"os"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

var ErrNoOp = errors.New("pdfcpu: no operation")

// ViewerPreferences returns rs's viewer preferences.
func ViewerPreferences(rs io.ReadSeeker, conf *model.Configuration) (*model.ViewerPreferences, *model.Version, error) {
	if rs == nil {
		return nil, nil, errors.New("pdfcpu: ViewerPreferences: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTVIEWERPREFERENCES

	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return nil, nil, err
	}

	v := ctx.Version()

	return ctx.ViewerPref, &v, nil
}

// ViewerPreferences returns inFile's viewer preferences.
func ViewerPreferencesFile(inFile string, all bool, conf *model.Configuration) (*model.ViewerPreferences, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vp, version, err := ViewerPreferences(f, conf)
	if err != nil {
		return nil, err
	}

	if !all {
		return vp, nil
	}

	return model.ViewerPreferencesWithDefaults(vp, *version)
}

// ListViewerPreferences returns rs's viewer preferences.
func ListViewerPreferences(rs io.ReadSeeker, all bool, conf *model.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: ListViewerPreferences: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTVIEWERPREFERENCES

	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	if !all {
		if ctx.ViewerPref != nil {
			return ctx.ViewerPref.List(), nil
		}
		return []string{"No viewer preferences available."}, nil
	}

	vp1, err := model.ViewerPreferencesWithDefaults(ctx.ViewerPref, ctx.Version())
	if err != nil {
		return nil, err
	}

	return vp1.List(), nil
}

// ListViewerPreferencesFile lists inFile's viewer preferences in JSON.
func ListViewerPreferencesFileJSON(inFile string, all bool, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vp, version, err := ViewerPreferences(f, conf)
	if err != nil {
		return nil, err
	}

	if !all {
		if vp == nil {
			return []string{"No viewer preferences available."}, nil
		}
	} else {
		vp, err = model.ViewerPreferencesWithDefaults(vp, *version)
		if err != nil {
			return nil, err
		}
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
		return nil, err
	}

	return []string{string(bb)}, nil
}

// ListViewerPreferencesFile lists inFile's viewer preferences.
func ListViewerPreferencesFile(inFile string, all, json bool, conf *model.Configuration) ([]string, error) {
	if json {
		return ListViewerPreferencesFileJSON(inFile, all, conf)
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ListViewerPreferences(f, all, conf)
}

// SetViewerPreferences sets rs's viewer preferences and writes the result to w.
func SetViewerPreferences(rs io.ReadSeeker, w io.Writer, vp model.ViewerPreferences, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: SetViewerPreferences: missing rs")
	}

	if w == nil {
		return errors.New("pdfcpu: SetViewerPreferences: missing w")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.SETVIEWERPREFERENCES

	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if ctx.Version() == model.V20 {
		return pdfcpu.ErrUnsupportedVersion
	}

	version := ctx.Version()

	if err := vp.Validate(version); err != nil {
		return err
	}

	if ctx.ViewerPref == nil {
		ctx.ViewerPref = &vp
	} else {
		ctx.ViewerPref.Populate(&vp)
	}

	ctx.XRefTable.BindViewerPreferences()

	return WriteContext(ctx, w)
}

// SetViewerPreferencesFromJSONBytes sets rs's viewer preferences corresponding to jsonBytes and writes the result to w.
func SetViewerPreferencesFromJSONBytes(rs io.ReadSeeker, w io.Writer, jsonBytes []byte, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: SetViewerPreferencesFromJSONBytes: missing rs")
	}

	if w == nil {
		return errors.New("pdfcpu: SetViewerPreferencesFromJSONBytes: missing w")
	}

	if !json.Valid(jsonBytes) {
		return ErrInvalidJSON
	}

	vp := model.ViewerPreferences{}

	if err := json.Unmarshal(jsonBytes, &vp); err != nil {
		return err
	}

	return SetViewerPreferences(rs, w, vp, conf)
}

// SetViewerPreferencesFromJSONReader sets rs's viewer preferences corresponding to rd and writes the result to w.
func SetViewerPreferencesFromJSONReader(rs io.ReadSeeker, w io.Writer, rd io.Reader, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: SetViewerPreferencesFromJSONReader: missing rs")
	}

	if w == nil {
		return errors.New("pdfcpu: SetViewerPreferencesFromJSONReader: missing w")
	}

	if rd == nil {
		return errors.New("pdfcpu: SetViewerPreferencesFromJSONReader: missing rd")
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rd); err != nil {
		return err
	}

	return SetViewerPreferencesFromJSONBytes(rs, w, buf.Bytes(), conf)
}

// SetViewerPreferencesFile sets inFile's viewer preferences and writes the result to outFile.
func SetViewerPreferencesFile(inFile, outFile string, vp model.ViewerPreferences, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
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

	return SetViewerPreferences(f1, f2, vp, conf)
}

// SetViewerPreferencesFileFromJSONBytes sets inFile's viewer preferences corresponding to jsonBytes and writes the result to outFile.
func SetViewerPreferencesFileFromJSONBytes(inFile, outFile string, jsonBytes []byte, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
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

	return SetViewerPreferencesFromJSONBytes(f1, f2, jsonBytes, conf)
}

// SetViewerPreferencesFileFromJSONFile sets inFile's viewer preferences corresponding to inFileJSON and writes the result to outFile.
func SetViewerPreferencesFileFromJSONFile(inFilePDF, outFilePDF, inFileJSON string, conf *model.Configuration) error {
	if inFileJSON == "" {
		return errors.New("pdfcpu: SetViewerPreferencesFileFromJSONFile: missing inFileJSON")
	}

	bb, err := os.ReadFile(inFileJSON)
	if err != nil {
		return err
	}

	return SetViewerPreferencesFileFromJSONBytes(inFilePDF, outFilePDF, bb, conf)
}

// ResetViewerPreferences resets rs's viewer preferences and writes the result to w.
func ResetViewerPreferences(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: ResetViewerPreferences: missing rs")
	}

	if w == nil {
		return errors.New("pdfcpu: ResetViewerPreferences: missing w")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.RESETVIEWERPREFERENCES

	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if ctx.ViewerPref == nil {
		return ErrNoOp
	}

	if ctx.Version() == model.V20 {
		return pdfcpu.ErrUnsupportedVersion
	}

	delete(ctx.RootDict, "ViewerPreferences")

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	return nil
}

// ResetViewerPreferencesFile resets inFile's viewer preferences and writes the result to outFile.
func ResetViewerPreferencesFile(inFile, outFile string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			if err == ErrNoOp {
				err = nil
			}
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

	return ResetViewerPreferences(f1, f2, conf)
}
