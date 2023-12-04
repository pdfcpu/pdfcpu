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
	"io"
	"os"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// PageMode returns rs's page mode.
func PageMode(rs io.ReadSeeker, conf *model.Configuration) (*model.PageMode, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: PageMode: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTPAGEMODE

	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	return ctx.PageMode, nil
}

// PageModeFile returns inFile's page mode.
func PageModeFile(inFile string, conf *model.Configuration) (*model.PageMode, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return PageMode(f, conf)
}

// ListPageMode lists rs's page mode.
func ListPageMode(rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: ListPageMode: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTPAGEMODE

	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	if ctx.PageMode != nil {
		return []string{ctx.PageMode.String()}, nil
	}

	return []string{"No page mode set, PDF viewers will default to \"UseNone\""}, nil
}

// ListPageModeFile lists inFile's page mode.
func ListPageModeFile(inFile string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ListPageMode(f, conf)
}

// SetPageMode sets rs's page mode and writes the result to w.
func SetPageMode(rs io.ReadSeeker, w io.Writer, val model.PageMode, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: SetPageMode: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.SETPAGEMODE

	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if ctx.Version() == model.V20 {
		return pdfcpu.ErrUnsupportedVersion
	}

	ctx.RootDict["PageMode"] = types.Name(val.String())

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	return nil
}

// SetPageModeFile sets inFile's page mode and writes the result to outFile.
func SetPageModeFile(inFile, outFile string, val model.PageMode, conf *model.Configuration) (err error) {
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

	return SetPageMode(f1, f2, val, conf)
}

// ResetPageMode resets rs's page mode and writes the result to w.
func ResetPageMode(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: ResetPageMode: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.RESETPAGEMODE

	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if ctx.Version() == model.V20 {
		return pdfcpu.ErrUnsupportedVersion
	}

	delete(ctx.RootDict, "PageMode")

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	return nil
}

// ResetPageModeFile resets inFile's page mode and writes the result to outFile.
func ResetPageModeFile(inFile, outFile string, conf *model.Configuration) (err error) {
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

	return ResetPageMode(f1, f2, conf)
}
