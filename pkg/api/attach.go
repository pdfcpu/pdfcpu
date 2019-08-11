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
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// ListAttachments returns a list of embedded file attachments of rs.
func ListAttachments(rs io.ReadSeeker, conf *pdf.Configuration) ([]string, error) {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()
	list, err := pdf.AttachList(ctx.XRefTable)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats("list files", durRead, durVal, durOpt, durWrite, durTotal)

	return list, nil
}

// ListAttachmentsFile returns a list of embedded file attachments of inFile.
func ListAttachmentsFile(inFile string, conf *pdf.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ListAttachments(f, conf)
}

// AddAttachments embeds files into a PDF context read from rs and writes the result to w.
func AddAttachments(rs io.ReadSeeker, w io.Writer, files []string, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	from := time.Now()
	var ok bool

	if ok, err = pdf.AttachAdd(ctx.XRefTable, stringSet(files)); err != nil {
		return err
	}
	if !ok {
		log.CLI.Println("no attachment added.")
		return nil
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
func AddAttachmentsFile(inFile, outFile string, files []string, conf *pdf.Configuration) (err error) {
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

	return AddAttachments(f1, f2, files, conf)
}

// RemoveAttachments deletes embedded files from a PDF context read from rs and writes the result to w.
func RemoveAttachments(rs io.ReadSeeker, w io.Writer, fileNames []string, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	from := time.Now()

	var ok bool
	if ok, err = pdf.AttachRemove(ctx.XRefTable, stringSet(fileNames)); err != nil {
		return err
	}
	if !ok {
		log.CLI.Println("no attachment removed.")
		return nil
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
func RemoveAttachmentsFile(inFile, outFile string, files []string, conf *pdf.Configuration) (err error) {
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

// ExtractAttachments extracts embedded files from a PDF context read from rs into outDir.
func ExtractAttachments(rs io.ReadSeeker, outDir string, fileNames []string, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	fromWrite := time.Now()

	ctx.Write.DirName = outDir
	if err = pdf.AttachExtract(ctx, stringSet(fileNames)); err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats("write files", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// ExtractAttachmentsFile extracts embedded files from a PDF context read from inFile into outDir.
func ExtractAttachmentsFile(inFile, outDir string, fileNames []string, conf *pdf.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()
	return ExtractAttachments(f, outDir, fileNames, conf)
}
