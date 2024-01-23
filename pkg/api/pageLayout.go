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

// PageLayout returns rs's page layout.
func PageLayout(rs io.ReadSeeker, conf *model.Configuration) (*model.PageLayout, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: PageLayout: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTPAGELAYOUT

	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	return ctx.PageLayout, nil
}

// PageLayoutFile returns inFile's page layout.
func PageLayoutFile(inFile string, conf *model.Configuration) (*model.PageLayout, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return PageLayout(f, conf)
}

// ListPageLayout lists rs's page layout.
func ListPageLayout(rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: ListPageLayout: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTPAGELAYOUT

	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	if ctx.PageLayout != nil {
		return []string{ctx.PageLayout.String()}, nil
	}

	return []string{"No page layout set, PDF viewers will default to \"SinglePage\""}, nil
}

// ListPageLayoutFile lists inFile's page layout.
func ListPageLayoutFile(inFile string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ListPageLayout(f, conf)
}

// SetPageLayout sets rs's page layout and writes the result to w.
func SetPageLayout(rs io.ReadSeeker, w io.Writer, val model.PageLayout, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: SetPageLayout: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.SETPAGELAYOUT

	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if ctx.Version() == model.V20 {
		return pdfcpu.ErrUnsupportedVersion
	}

	ctx.RootDict["PageLayout"] = types.Name(val.String())

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	return nil
}

// SetPageLayoutFile sets inFile's page layout and writes the result to outFile.
func SetPageLayoutFile(inFile, outFile string, val model.PageLayout, conf *model.Configuration) (err error) {
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

	return SetPageLayout(f1, f2, val, conf)
}

// ResetPageLayout resets rs's page layout and writes the result to w.
func ResetPageLayout(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: ResetPageLayout: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.RESETPAGELAYOUT

	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return err
	}

	if ctx.Version() == model.V20 {
		return pdfcpu.ErrUnsupportedVersion
	}

	delete(ctx.RootDict, "PageLayout")

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	return nil
}

// ResetPageLayoutFile resets inFile's page layout and writes the result to outFile.
func ResetPageLayoutFile(inFile, outFile string, conf *model.Configuration) (err error) {
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

	return ResetPageLayout(f1, f2, conf)
}
