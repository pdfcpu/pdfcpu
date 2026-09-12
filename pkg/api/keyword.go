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
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Keywords returns the keywords of rs's info dict and supports cancellation.
func Keywords(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (ss []string, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	conf = operationConfiguration(conf, model.LISTKEYWORDS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return nil, fmt.Errorf("list keywords: %w", err)
	}

	ss, err = pdfcpu.KeywordsList(c, ctx)
	if err != nil {
		return nil, fmt.Errorf("list keywords: collect keywords: %w", err)
	}
	return ss, nil
}

// AddKeywords adds keywords to rs's infodict, writes the result to w and supports cancellation.
func AddKeywords(c context.Context, rs io.ReadSeeker, w io.Writer, keywords []string, conf *model.Configuration) (err error) {
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

	if err := validateNoEmptyStrings(keywords, "keyword"); err != nil {
		return fmt.Errorf("add keywords: validate keywords: %w", err)
	}

	conf = operationConfiguration(conf, model.ADDKEYWORDS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("add keywords: %w", err)
	}

	if err = pdfcpu.KeywordsAdd(c, ctx, keywords); err != nil {
		return fmt.Errorf("add keywords: update document keywords: %w", err)
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("add keywords: write output: %w", err)
	}
	return nil
}

// AddKeywordsFile adds keywords to inFile's infodict, writes the result to outFile and supports cancellation.
func AddKeywordsFile(c context.Context, inFile, outFile string, keywords []string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if err := validateNoEmptyStrings(keywords, "keyword"); err != nil {
		return fmt.Errorf("add keywords: validate keywords: %w", err)
	}

	f1, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("add keywords: open input %s: %w", inFile, err)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "add keywords")
	if err != nil {
		return errors.Join(
			fmt.Errorf("add keywords: create output: %w", err),
			closeFile(f1, "add keywords: close input"),
		)
	}
	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = AddKeywords(c, f1, staged.output.file, keywords, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}

// RemoveKeywords deletes keywords from rs's infodict, writes the result to w and supports cancellation.
func RemoveKeywords(c context.Context, rs io.ReadSeeker, w io.Writer, keywords []string, conf *model.Configuration) (err error) {
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

	if err := validateNoEmptyStrings(keywords, "keyword"); err != nil {
		return fmt.Errorf("remove keywords: validate keywords: %w", err)
	}

	conf = operationConfiguration(conf, model.REMOVEKEYWORDS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("remove keywords: %w", err)
	}

	var ok bool
	if ok, err = pdfcpu.KeywordsRemove(c, ctx, keywords); err != nil {
		return fmt.Errorf("remove keywords: update document keywords: %w", err)
	}
	if !ok {
		return fmt.Errorf("remove keywords: %w", ErrNoKeywordRemoved)
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("remove keywords: write output: %w", err)
	}
	return nil
}

// RemoveKeywordsFile deletes keywords from inFile's infodict, writes the result to outFile and supports cancellation.
func RemoveKeywordsFile(c context.Context, inFile, outFile string, keywords []string, conf *model.Configuration) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if err := validateNoEmptyStrings(keywords, "keyword"); err != nil {
		return fmt.Errorf("remove keywords: validate keywords: %w", err)
	}

	f1, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("remove keywords: open input %s: %w", inFile, err)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "remove keywords")
	if err != nil {
		return errors.Join(
			fmt.Errorf("remove keywords: create output: %w", err),
			closeFile(f1, "remove keywords: close input"),
		)
	}
	ok := false
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = RemoveKeywords(c, f1, staged.output.file, keywords, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}
	ok = true
	return nil
}
