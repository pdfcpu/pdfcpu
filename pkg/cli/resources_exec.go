/*
Copyright 2026 The pdfcpu Authors.

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

package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ImportImages appends PDF pages containing images to outFile which will be created if necessary.
// ImportImages turns image files into a page sequence and writes the result to outFile.
// In its simplest form this operation converts an image into a PDF.
func ImportImages(cmd *Command) ([]string, error) {
	stdinImage, err := hasStdinImage(cmd.InFiles)
	if err != nil {
		return nil, err
	}
	if *cmd.OutFile != "-" && !stdinImage {
		return nil, api.ImportImagesFile(cmd.InFiles, *cmd.OutFile, cmd.Import, cmd.Conf)
	}

	readers, closers, err := importImageReaders(cmd.InFiles)
	if err != nil {
		return nil, err
	}
	defer closeAll(closers)

	if *cmd.OutFile == "-" {
		log.SetCLILogger(nil)
		return nil, api.ImportImages(nil, os.Stdout, readers, cmd.Import, cmd.Conf)
	}

	return nil, importImagesToFile(*cmd.OutFile, readers, cmd.Import, cmd.Conf)
}

func hasStdinImage(inFiles []string) (bool, error) {
	stdinImage := false
	for _, fn := range inFiles {
		if fn == "-" {
			if stdinImage {
				return false, fmt.Errorf("pdfcpu: only one imageFile may read from stdin")
			}
			stdinImage = true
		}
	}
	return stdinImage, nil
}

func importImageReader(fn string) (io.Reader, io.Closer, error) {
	if fn != "-" {
		f, err := os.Open(fn)
		if err != nil {
			return nil, nil, err
		}
		return f, f, nil
	}

	bb, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, nil, err
	}
	if len(bb) == 0 {
		return nil, nil, fmt.Errorf("pdfcpu: stdin is empty")
	}
	return bytes.NewReader(bb), nil, nil
}

func importImageReaders(inFiles []string) ([]io.Reader, []io.Closer, error) {
	readers := make([]io.Reader, 0, len(inFiles))
	closers := []io.Closer{}
	for _, fn := range inFiles {
		r, c, err := importImageReader(fn)
		if err != nil {
			closeAll(closers)
			return nil, nil, err
		}
		readers = append(readers, r)
		if c != nil {
			closers = append(closers, c)
		}
	}
	return readers, closers, nil
}

func closeAll(closers []io.Closer) {
	for _, c := range closers {
		_ = c.Close()
	}
}

func importImagesDestination(outFile string) (io.ReadSeeker, string, *os.File, error) {
	var rs io.ReadSeeker
	tmpFile := outFile
	if _, err := os.Stat(outFile); err == nil {
		f, err := os.Open(outFile)
		if err != nil {
			return nil, "", nil, err
		}
		rs = f
		tmpFile += ".tmp"
		return rs, tmpFile, f, nil
	} else if !os.IsNotExist(err) {
		return nil, "", nil, err
	}
	return rs, tmpFile, nil, nil
}

func importImagesToFile(outFile string, readers []io.Reader, imp *pdfcpu.Import, conf *model.Configuration) error {
	rs, tmpFile, f1, err := importImagesDestination(outFile)
	if err != nil {
		return err
	}
	if f1 != nil {
		defer f1.Close()
	}
	f, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok && tmpFile != outFile {
			_ = os.Remove(tmpFile)
		}
	}()

	if err := api.ImportImages(rs, f, readers, imp, conf); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	ok = true
	if tmpFile != outFile {
		return os.Rename(tmpFile, outFile)
	}
	return nil
}

// CreateCheatSheetsFonts creates single page PDF cheat sheets for user fonts in current dir.
func CreateCheatSheetsFonts(cmd *Command) ([]string, error) {
	return nil, api.CreateCheatSheetsUserFonts(cmd.InFiles)
}

// ListFonts gathers information about supported fonts and returns the result as []string.
func ListFonts(cmd *Command) ([]string, error) {
	return api.ListFonts()
}

// InstallFonts installs True Type fonts into the pdfcpu pconfig dir.
func InstallFonts(cmd *Command) ([]string, error) {
	return nil, api.InstallFonts(cmd.InFiles)
}

func listImages(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: listImages: Please provide rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTIMAGES

	ctx, err := api.ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, err
	}

	pages, err := api.PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return nil, err
	}

	return pdfcpu.ListImages(ctx, pages)
}

// ListImagesFile returns a formatted list of embedded images of inFile.
func ListImagesFile(inFiles []string, selectedPages []string, conf *model.Configuration) ([]string, error) {
	if len(selectedPages) == 0 {
		log.CLI.Printf("pages: all\n")
	}

	log.SetCLILogger(nil)

	ss := []string{}
	var errs []error

	for _, fn := range inFiles {
		f, err := os.Open(fn)
		if err != nil {
			if len(inFiles) > 1 {
				errs = append(errs, fmt.Errorf("%s: %w", fn, err))
				continue
			}
			return nil, err
		}
		defer f.Close()
		output, err := listImages(f, selectedPages, conf)
		if err != nil {
			if len(inFiles) > 1 {
				errs = append(errs, fmt.Errorf("%s: %w", fn, err))
				continue
			}
			return nil, err
		}
		ss = append(ss, "\n"+fn+":")
		ss = append(ss, output...)
	}

	return ss, errors.Join(errs...)
}

// ListImages returns inFiles embedded images.
func ListImages(cmd *Command) ([]string, error) {
	stdin := false
	for _, fn := range cmd.InFiles {
		if fn == "-" {
			stdin = true
			break
		}
	}
	if !stdin {
		return ListImagesFile(cmd.InFiles, cmd.PageSelection, cmd.Conf)
	}

	log.SetCLILogger(nil)
	var ss []string
	var errs []error
	for _, fn := range cmd.InFiles {
		var rs io.ReadSeeker
		var err error
		if fn == "-" {
			rs, err = readSeekerFromStdin()
		} else {
			rs, err = os.Open(fn)
		}
		if err != nil {
			if len(cmd.InFiles) == 1 {
				return nil, err
			}
			errs = append(errs, fmt.Errorf("%s: %w", fn, err))
			continue
		}
		if f, ok := rs.(*os.File); ok {
			defer f.Close()
		}

		output, err := listImages(rs, cmd.PageSelection, cmd.Conf)
		if err != nil {
			if len(cmd.InFiles) == 1 {
				return nil, err
			}
			errs = append(errs, fmt.Errorf("%s: %w", fn, err))
			continue
		}
		label := fn
		if label == "-" {
			label = "stdin"
		}
		ss = append(ss, "\n"+label+":")
		ss = append(ss, output...)
	}

	return ss, errors.Join(errs...)
}

func updateImageParams(cmd *Command) (objNr, pageNr int, id string) {
	if cmd.IntVal > 0 {
		if cmd.StringVal != "" {
			pageNr = cmd.IntVal
			id = cmd.StringVal
		} else {
			objNr = cmd.IntVal
		}
	}
	return objNr, pageNr, id
}

func updateImagesInOut(cmd *Command, objNr, pageNr int, id string) ([]string, error) {
	if cmd.InFiles[0] != "-" && *cmd.OutFile != "-" {
		return nil, api.UpdateImagesFile(cmd.InFiles[0], cmd.InFiles[1], *cmd.OutFile, objNr, pageNr, id, cmd.Conf)
	}

	rs, w, cleanup, err := streamInOut(cmd.InFiles[0], *cmd.OutFile)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	f, err := os.Open(cmd.InFiles[1])
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return nil, api.UpdateImages(rs, f, w, objNr, pageNr, id, cmd.Conf)
}

// UpdateImages replaces image objects.
func UpdateImages(cmd *Command) ([]string, error) {
	objNr, pageNr, id := updateImageParams(cmd)
	return updateImagesInOut(cmd, objNr, pageNr, id)
}
func listAttachments(rs io.ReadSeeker, conf *model.Configuration, withDesc, sorted bool) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: listAttachments: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTATTACHMENTS

	ctx, err := api.ReadAndValidate(rs, conf)
	if err != nil {
		return nil, err
	}

	aa, err := ctx.ListAttachments()
	if err != nil {
		return nil, err
	}

	var ss []string
	for _, a := range aa {
		s := a.FileName
		if withDesc && a.Desc != "" {
			s = fmt.Sprintf("%s (%s)", s, a.Desc)
		}
		ss = append(ss, s)
	}
	if sorted {
		sort.Strings(ss)
	}

	return ss, nil
}

// ListAttachmentsFile returns a list of embedded file attachments of inFile with optional description.
func ListAttachmentsFile(inFile string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return listAttachments(f, conf, true, true)
}

// ListAttachmentsCompactFile returns a list of embedded file attachments of inFile w/o optional description.
func ListAttachmentsCompactFile(inFile string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return listAttachments(f, conf, false, false)
}

// ListAttachments returns a list of embedded file attachments for inFile.
func ListAttachments(cmd *Command) ([]string, error) {
	if *cmd.InFile == "-" {
		rs, err := readSeekerFromStdin()
		if err != nil {
			return nil, err
		}
		return listAttachments(rs, cmd.Conf, true, true)
	}

	return ListAttachmentsFile(*cmd.InFile, cmd.Conf)
}

// AddAttachments embeds inFiles into a PDF context read from inFile and writes the result to outFile.
func AddAttachments(cmd *Command) ([]string, error) {
	if *cmd.InFile == "-" || *cmd.OutFile == "-" {
		rs, w, cleanup, err := streamInOut(*cmd.InFile, *cmd.OutFile)
		if err != nil {
			return nil, err
		}
		if cleanup != nil {
			defer cleanup()
		}
		return nil, api.AddAttachments(rs, w, cmd.InFiles, cmd.Mode == model.ADDATTACHMENTSPORTFOLIO, cmd.Conf)
	}

	return nil, api.AddAttachmentsFile(*cmd.InFile, *cmd.OutFile, cmd.InFiles, cmd.Mode == model.ADDATTACHMENTSPORTFOLIO, cmd.Conf)
}

// RemoveAttachments deletes inFiles from a PDF context read from inFile and writes the result to outFile.
func RemoveAttachments(cmd *Command) ([]string, error) {
	if *cmd.InFile == "-" || *cmd.OutFile == "-" {
		rs, w, cleanup, err := streamInOut(*cmd.InFile, *cmd.OutFile)
		if err != nil {
			return nil, err
		}
		if cleanup != nil {
			defer cleanup()
		}
		return nil, api.RemoveAttachments(rs, w, cmd.InFiles, cmd.Conf)
	}

	return nil, api.RemoveAttachmentsFile(*cmd.InFile, *cmd.OutFile, cmd.InFiles, cmd.Conf)
}

// ExtractAttachments extracts inFiles from a PDF context read from inFile and writes the result to outFile.
func ExtractAttachments(cmd *Command) ([]string, error) {
	if *cmd.InFile == "-" {
		rs, err := readSeekerFromStdin()
		if err != nil {
			return nil, err
		}
		return nil, api.ExtractAttachments(rs, *cmd.OutDir, cmd.InFiles, cmd.Conf)
	}

	return nil, api.ExtractAttachmentsFile(*cmd.InFile, *cmd.OutDir, cmd.InFiles, cmd.Conf)
}

// ListKeywordsFile returns the keyword list of inFile.
func ListKeywordsFile(inFile string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return api.Keywords(f, conf)
}

// ListKeywords returns a list of keywords for inFile.
func ListKeywords(cmd *Command) ([]string, error) {
	if *cmd.InFile == "-" {
		rs, err := readSeekerFromStdin()
		if err != nil {
			return nil, err
		}
		return api.Keywords(rs, cmd.Conf)
	}

	return ListKeywordsFile(*cmd.InFile, cmd.Conf)
}

// AddKeywords adds keywords to inFile's document info dict and writes the result to outFile.
func AddKeywords(cmd *Command) ([]string, error) {
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.AddKeywordsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
	}

	rs, w, cleanup, err := streamInOut(*cmd.InFile, *cmd.OutFile)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return nil, api.AddKeywords(rs, w, cmd.StringVals, cmd.Conf)
}

// RemoveKeywords deletes keywords from inFile's document info dict and writes the result to outFile.
func RemoveKeywords(cmd *Command) ([]string, error) {
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.RemoveKeywordsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
	}

	rs, w, cleanup, err := streamInOut(*cmd.InFile, *cmd.OutFile)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return nil, api.RemoveKeywords(rs, w, cmd.StringVals, cmd.Conf)
}

func listProperties(rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: listProperties: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTPROPERTIES

	ctx, err := api.ReadAndValidate(rs, conf)
	if err != nil {
		return nil, err
	}

	return pdfcpu.PropertiesList(ctx)
}

// ListPropertiesFile returns the property list of inFile.
func ListPropertiesFile(inFile string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return listProperties(f, conf)
}

// ListProperties returns inFile's properties.
func ListProperties(cmd *Command) ([]string, error) {
	if *cmd.InFile == "-" {
		rs, err := readSeekerFromStdin()
		if err != nil {
			return nil, err
		}
		return listProperties(rs, cmd.Conf)
	}

	return ListPropertiesFile(*cmd.InFile, cmd.Conf)
}

// AddProperties adds properties to inFile's document info dict and writes the result to outFile.
func AddProperties(cmd *Command) ([]string, error) {
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.AddPropertiesFile(*cmd.InFile, *cmd.OutFile, cmd.StringMap, cmd.Conf)
	}

	rs, w, cleanup, err := streamInOut(*cmd.InFile, *cmd.OutFile)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return nil, api.AddProperties(rs, w, cmd.StringMap, cmd.Conf)
}

// RemoveProperties deletes properties from inFile's document info dict and writes the result to outFile.
func RemoveProperties(cmd *Command) ([]string, error) {
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.RemovePropertiesFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
	}

	rs, w, cleanup, err := streamInOut(*cmd.InFile, *cmd.OutFile)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return nil, api.RemoveProperties(rs, w, cmd.StringVals, cmd.Conf)
}
