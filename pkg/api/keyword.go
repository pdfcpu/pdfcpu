/*
	Copyright 2020 The pdfcpu Authors.

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
	"github.com/pkg/errors"
)

// Keywords returns the keywords of rs's info dict.
func Keywords(rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: ListKeywords: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		// Validation loads infodict.
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTKEYWORDS

	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	return pdfcpu.KeywordsList(ctx.XRefTable)
}

// AddKeywords embeds files into a PDF context read from rs and writes the result to w.
func AddKeywords(rs io.ReadSeeker, w io.Writer, files []string, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: AddKeywords: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		// Validation loads infodict.
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.ADDKEYWORDS

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := ReadValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	from := time.Now()

	if err = pdfcpu.KeywordsAdd(ctx.XRefTable, files); err != nil {
		return err
	}

	durAdd := time.Since(from).Seconds()
	fromWrite := time.Now()

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := durAdd + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "add keyword, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// AddKeywordsFile embeds files into a PDF context read from inFile and writes the result to outFile.
func AddKeywordsFile(inFile, outFile string, files []string, conf *model.Configuration) (err error) {
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
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
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

	return AddKeywords(f1, f2, files, conf)
}

// RemoveKeywords deletes embedded files from a PDF context read from rs and writes the result to w.
func RemoveKeywords(rs io.ReadSeeker, w io.Writer, keywords []string, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: RemoveKeywords: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		// Validation loads infodict.
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.REMOVEKEYWORDS

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := ReadValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	from := time.Now()

	var ok bool
	if ok, err = pdfcpu.KeywordsRemove(ctx.XRefTable, keywords); err != nil {
		return err
	}
	if !ok {
		return errors.New("no keyword removed")
	}

	durRemove := time.Since(from).Seconds()
	fromWrite := time.Now()
	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := durRemove + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "remove att, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// RemoveKeywordsFile deletes embedded files from a PDF context read from inFile and writes the result to outFile.
func RemoveKeywordsFile(inFile, outFile string, keywords []string, conf *model.Configuration) (err error) {
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
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
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

	return RemoveKeywords(f1, f2, keywords, conf)
}
