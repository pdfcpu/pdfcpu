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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// PageLayout returns rs's page layout.
func PageLayout(rs io.ReadSeeker, conf *model.Configuration) (pl *model.PageLayout, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTPAGELAYOUT

	ctx, err := ReadAndValidate(rs, conf)
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
func ListPageLayout(rs io.ReadSeeker, conf *model.Configuration) (ss []string, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTPAGELAYOUT

	ctx, err := ReadAndValidate(rs, conf)
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
func SetPageLayout(rs io.ReadSeeker, w io.Writer, val model.PageLayout, conf *model.Configuration) (err error) {
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
	conf.Cmd = model.SETPAGELAYOUT

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return err
	}

	ctx.RootDict["PageLayout"] = types.Name(val.String())

	return Write(ctx, w, conf)
}

// SetPageLayoutFile sets inFile's page layout and writes the result to outFile.
func SetPageLayoutFile(inFile, outFile string, val model.PageLayout, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
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

	if err = SetPageLayout(f1, f2, val, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// ResetPageLayout resets rs's page layout and writes the result to w.
func ResetPageLayout(rs io.ReadSeeker, w io.Writer, conf *model.Configuration) (err error) {
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
	conf.Cmd = model.RESETPAGELAYOUT

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return err
	}

	delete(ctx.RootDict, "PageLayout")

	return Write(ctx, w, conf)
}

// ResetPageLayoutFile resets inFile's page layout and writes the result to outFile.
func ResetPageLayoutFile(inFile, outFile string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
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

	if err = ResetPageLayout(f1, f2, conf); err != nil {
		return err
	}

	ok = true

	return nil
}
