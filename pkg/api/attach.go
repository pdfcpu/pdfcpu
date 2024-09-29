/*
	Copyright 2019 The pdfcpu Authors.

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
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// Attachments returns rs's attachments.
func Attachments(rs io.ReadSeeker, conf *model.Configuration) ([]model.Attachment, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: Attachments: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTATTACHMENTS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, err
	}

	return ctx.ListAttachments()
}

// AddAttachments embeds files into a PDF context read from rs and writes the result to w.
// file is either a file name or a file name and a description separated by a comma.
func AddAttachments(rs io.ReadSeeker, w io.Writer, files []string, coll bool, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: AddAttachments: missing rs")
	}

	if w == nil {
		return errors.New("pdfcpu: AddAttachments: missing w")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDATTACHMENTS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	var ok bool

	for _, fn := range files {
		s := strings.Split(fn, ",")
		if len(s) == 0 || len(s) > 2 {
			continue
		}

		fileName := s[0]
		desc := ""
		if len(s) == 2 {
			desc = s[1]
		}

		if log.CLIEnabled() {
			log.CLI.Printf("adding %s\n", fileName)
		}
		f, err := os.Open(fileName)
		if err != nil {
			return err
		}
		defer f.Close()

		fi, err := f.Stat()
		if err != nil {
			return err
		}
		mt := fi.ModTime()

		a := model.Attachment{Reader: f, ID: filepath.Base(fileName), Desc: desc, ModTime: &mt}
		if err = ctx.AddAttachment(a, coll); err != nil {
			return err
		}
		ok = true
	}

	if !ok {
		return errors.New("pdfcpu: AddAttachments: No attachment added")
	}

	return Write(ctx, w, conf)
}

// AddAttachmentsFile embeds files into a PDF context read from inFile and writes the result to outFile.
func AddAttachmentsFile(inFile, outFile string, files []string, coll bool, conf *model.Configuration) (err error) {
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

	return AddAttachments(f1, f2, files, coll, conf)
}

// RemoveAttachments deletes embedded files from a PDF context read from rs and writes the result to w.
func RemoveAttachments(rs io.ReadSeeker, w io.Writer, files []string, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: RemoveAttachments: missing rs")
	}

	if w == nil {
		return errors.New("pdfcpu: RemoveAttachments: missing w")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDATTACHMENTS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	var ok bool
	if ok, err = ctx.RemoveAttachments(files); err != nil {
		return err
	}
	if !ok {
		return errors.New("pdfcpu: RemoveAttachments: No attachment removed")
	}

	return Write(ctx, w, conf)
}

// RemoveAttachmentsFile deletes embedded files from a PDF context read from inFile and writes the result to outFile.
func RemoveAttachmentsFile(inFile, outFile string, files []string, conf *model.Configuration) (err error) {
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

	return RemoveAttachments(f1, f2, files, conf)
}

// ExtractAttachmentsRaw extracts embedded files from a PDF context read from rs.
func ExtractAttachmentsRaw(rs io.ReadSeeker, outDir string, fileNames []string, conf *model.Configuration) ([]model.Attachment, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: ExtractAttachmentsRaw: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTATTACHMENTS

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, err
	}

	return ctx.ExtractAttachments(fileNames)
}

// ExtractAttachments extracts embedded files from a PDF context read from rs into outDir.
func ExtractAttachments(rs io.ReadSeeker, outDir string, fileNames []string, conf *model.Configuration) error {
	aa, err := ExtractAttachmentsRaw(rs, outDir, fileNames, conf)
	if err != nil {
		return err
	}

	for _, a := range aa {
		fileName := filepath.Join(outDir, a.FileName)
		f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
		if err != nil {
			fileName = filepath.Base(a.FileName)
			f, err = os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
			if err != nil {
				return err
			}
		}
		logWritingTo(fileName)
		if _, err = io.Copy(f, a); err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}

	return nil
}

// ExtractAttachmentsFile extracts embedded files from a PDF context read from inFile into outDir.
func ExtractAttachmentsFile(inFile, outDir string, files []string, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return ExtractAttachments(f, outDir, files, conf)
}
