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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Keywords returns the keywords of rs's info dict.
func Keywords(rs io.ReadSeeker, conf *model.Configuration) (ss []string, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTKEYWORDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, fmt.Errorf("list keywords: %w", err)
	}

	ss, err = pdfcpu.KeywordsList(ctx)
	if err != nil {
		return nil, fmt.Errorf("list keywords: collect keywords: %w", err)
	}
	return ss, nil
}

// AddKeywords adds keywords to rs's infodict and writes the result to w.
func AddKeywords(rs io.ReadSeeker, w io.Writer, keywords []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if err := validateNoEmptyStrings(keywords, "keyword"); err != nil {
		return fmt.Errorf("add keywords: validate keywords: %w", err)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.ADDKEYWORDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("add keywords: %w", err)
	}

	if err = pdfcpu.KeywordsAdd(ctx, keywords); err != nil {
		return fmt.Errorf("add keywords: update document keywords: %w", err)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("add keywords: write output: %w", err)
	}
	return nil
}

type keywordMutation func(io.ReadSeeker, io.Writer, []string, *model.Configuration) error

func mutateKeywordsFile(
	inFile, outFile string,
	keywords []string,
	conf *model.Configuration,
	op string,
	mutate keywordMutation,
) (err error) {
	var f1, f2 *os.File
	ok := false

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("%s: open input %s: %w", op, inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, op)
	if err != nil {
		return errors.Join(
			fmt.Errorf("%s: create output: %w", op, err),
			closeFile(f1, op+": close input"),
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

	if err = mutate(f1, f2, keywords, conf); err != nil {
		return err
	}

	ok = true
	return nil
}

// AddKeywordsFile adds keywords to inFile's infodict and writes the result to outFile.
func AddKeywordsFile(inFile, outFile string, keywords []string, conf *model.Configuration) error {
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if err := validateNoEmptyStrings(keywords, "keyword"); err != nil {
		return fmt.Errorf("add keywords: validate keywords: %w", err)
	}
	return mutateKeywordsFile(inFile, outFile, keywords, conf, "add keywords", AddKeywords)
}

// RemoveKeywords deletes keywords from rs's infodict and writes the result to w.
func RemoveKeywords(rs io.ReadSeeker, w io.Writer, keywords []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if err := validateNoEmptyStrings(keywords, "keyword"); err != nil {
		return fmt.Errorf("remove keywords: validate keywords: %w", err)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.REMOVEKEYWORDS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("remove keywords: %w", err)
	}

	var ok bool
	if ok, err = pdfcpu.KeywordsRemove(ctx, keywords); err != nil {
		return fmt.Errorf("remove keywords: update document keywords: %w", err)
	}
	if !ok {
		return fmt.Errorf("remove keywords: %w", ErrNoKeywordRemoved)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("remove keywords: write output: %w", err)
	}
	return nil
}

// RemoveKeywordsFile deletes keywords from inFile's infodict and writes the result to outFile.
func RemoveKeywordsFile(inFile, outFile string, keywords []string, conf *model.Configuration) error {
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if err := validateNoEmptyStrings(keywords, "keyword"); err != nil {
		return fmt.Errorf("remove keywords: validate keywords: %w", err)
	}
	return mutateKeywordsFile(inFile, outFile, keywords, conf, "remove keywords", RemoveKeywords)
}
