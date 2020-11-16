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

	"github.com/pdfcpu/pdfcpu/pkg/log"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// ListProperties returns the property list of rs.
func ListProperties(rs io.ReadSeeker, conf *pdf.Configuration) ([]string, error) {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	} else {
		// Validation loads infodict.
		conf.ValidationMode = pdf.ValidationRelaxed
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()
	list, err := pdf.PropertiesList(ctx.XRefTable)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats("list files", durRead, durVal, durOpt, durWrite, durTotal)

	return list, nil
}

// ListPropertiesFile returns the property list of inFile.
func ListPropertiesFile(inFile string, conf *pdf.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ListProperties(f, conf)
}

// AddProperties embeds files into a PDF context read from rs and writes the result to w.
func AddProperties(rs io.ReadSeeker, w io.Writer, properties map[string]string, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	} else {
		// Validation loads infodict.
		conf.ValidationMode = pdf.ValidationRelaxed
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	from := time.Now()

	if err = pdf.PropertiesAdd(ctx.XRefTable, properties); err != nil {
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

// AddPropertiesFile embeds files into a PDF context read from inFile and writes the result to outFile.
func AddPropertiesFile(inFile, outFile string, properties map[string]string, conf *pdf.Configuration) (err error) {
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

	return AddProperties(f1, f2, properties, conf)
}

// RemoveProperties deletes embedded files from a PDF context read from rs and writes the result to w.
func RemoveProperties(rs io.ReadSeeker, w io.Writer, properties []string, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	} else {
		// Validation loads infodict.
		conf.ValidationMode = pdf.ValidationRelaxed
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	from := time.Now()

	var ok bool
	if ok, err = pdf.PropertiesRemove(ctx.XRefTable, properties); err != nil {
		return err
	}
	if !ok {
		return errors.New("no property removed")
	}

	durRemove := time.Since(from).Seconds()
	fromWrite := time.Now()
	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := durRemove + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "remove prop, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// RemovePropertiesFile deletes embedded files from a PDF context read from inFile and writes the result to outFile.
func RemovePropertiesFile(inFile, outFile string, properties []string, conf *pdf.Configuration) (err error) {
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

	return RemoveProperties(f1, f2, properties, conf)
}
