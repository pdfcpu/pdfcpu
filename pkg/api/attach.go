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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func listAttachments(rs io.ReadSeeker, conf *pdfcpu.Configuration, withDesc, sorted bool) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: ListAttachments: Please provide rs")
	}
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()

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

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdfcpu.TimingStats("list files", durRead, durVal, durOpt, durWrite, durTotal)

	return ss, nil
}

// ListAttachments returns a list of embedded file attachments of rs with optional description.
func ListAttachments(rs io.ReadSeeker, conf *pdfcpu.Configuration) ([]string, error) {
	return listAttachments(rs, conf, true, true)
}

// ListAttachmentsFile returns a list of embedded file attachments of inFile with optional description.
func ListAttachmentsFile(inFile string, conf *pdfcpu.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ListAttachments(f, conf)
}

// ListAttachmentsCompact returns a list of embedded file attachments of rs w/o optional description.
func ListAttachmentsCompact(rs io.ReadSeeker, conf *pdfcpu.Configuration) ([]string, error) {
	return listAttachments(rs, conf, false, false)
}

// ListAttachmentsCompactFile returns a list of embedded file attachments of inFile w/o optional description.
func ListAttachmentsCompactFile(inFile string, conf *pdfcpu.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ListAttachmentsCompact(f, conf)
}

// AddAttachments embeds files into a PDF context read from rs and writes the result to w.
// file is either a file name or a file name and a description separated by a comma.
func AddAttachments(rs io.ReadSeeker, w io.Writer, files []string, coll bool, conf *pdfcpu.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: AddAttachments: Please provide rs")
	}
	if w == nil {
		return errors.New("pdfcpu: AddAttachments: Please provide w")
	}
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	from := time.Now()
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

		log.CLI.Printf("adding %s\n", fileName)
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

		a := pdfcpu.Attachment{Reader: f, ID: filepath.Base(fileName), Desc: desc, ModTime: &mt}
		if err = ctx.AddAttachment(a, coll); err != nil {
			return err
		}
		ok = true
	}

	if !ok {
		return errors.New("no attachment added")
	}

	durAdd := time.Since(from).Seconds()
	fromWrite := time.Now()

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := durAdd + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "add attachment, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// AddAttachmentsFile embeds files into a PDF context read from inFile and writes the result to outFile.
func AddAttachmentsFile(inFile, outFile string, files []string, coll bool, conf *pdfcpu.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	if f2, err = os.Create(tmpFile); err != nil {
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
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return AddAttachments(f1, f2, files, coll, conf)
}

// RemoveAttachments deletes embedded files from a PDF context read from rs and writes the result to w.
func RemoveAttachments(rs io.ReadSeeker, w io.Writer, files []string, conf *pdfcpu.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: RemoveAttachments: Please provide rs")
	}
	if w == nil {
		return errors.New("pdfcpu: RemoveAttachments: Please provide w")
	}
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	from := time.Now()

	var ok bool
	if ok, err = ctx.RemoveAttachments(files); err != nil {
		return err
	}
	if !ok {
		return errors.New("no attachment removed")
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

// RemoveAttachmentsFile deletes embedded files from a PDF context read from inFile and writes the result to outFile.
func RemoveAttachmentsFile(inFile, outFile string, files []string, conf *pdfcpu.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	if f2, err = os.Create(tmpFile); err != nil {
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
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return RemoveAttachments(f1, f2, files, conf)
}

// ExtractAttachmentsRaw extracts embedded files from a PDF context read from rs.
func ExtractAttachmentsRaw(rs io.ReadSeeker, outDir string, fileNames []string, conf *pdfcpu.Configuration) ([]pdfcpu.Attachment, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: ExtractAttachmentsRaw: Please provide rs")
	}
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}

	ctx, _, _, _, err := readValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	return ctx.ExtractAttachments(fileNames)
}

// ExtractAttachments extracts embedded files from a PDF context read from rs into outDir.
func ExtractAttachments(rs io.ReadSeeker, outDir string, fileNames []string, conf *pdfcpu.Configuration) error {
	aa, err := ExtractAttachmentsRaw(rs, outDir, fileNames, conf)
	if err != nil {
		return err
	}

	for _, a := range aa {
		fileName := filepath.Join(outDir, a.FileName)
		log.CLI.Printf("writing %s\n", fileName)
		f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return err
		}
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
func ExtractAttachmentsFile(inFile, outDir string, files []string, conf *pdfcpu.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()
	return ExtractAttachments(f, outDir, files, conf)
}
