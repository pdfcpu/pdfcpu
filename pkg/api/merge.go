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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// appendTo appends inFile to ctxDest's page tree.
func appendTo(rs io.ReadSeeker, ctxDest *model.Context) error {
	ctxSource, _, _, err := readAndValidate(rs, ctxDest.Configuration, time.Now())
	if err != nil {
		return err
	}

	// Merge source context into dest context.
	return pdfcpu.MergeXRefTables(ctxSource, ctxDest)
}

func Merge(destFile string, inFiles []string, w io.Writer, conf *model.Configuration) error {

	if w == nil {
		return errors.New("pdfcpu: Merge: Please provide w")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.MERGECREATE

	if destFile != "" {
		conf.Cmd = model.MERGEAPPEND
	}
	if destFile == "" {
		destFile = inFiles[0]
		inFiles = inFiles[1:]
	}

	f, err := os.Open(destFile)
	if err != nil {
		return err
	}
	defer f.Close()

	log.CLI.Println("merging into " + destFile)

	ctxDest, _, _, err := readAndValidate(f, conf, time.Now())
	if err != nil {
		return err
	}

	ctxDest.EnsureVersionForWriting()

	for _, fName := range inFiles {
		if err := func() error {
			f, err := os.Open(fName)
			if err != nil {
				return err
			}
			defer f.Close()

			log.CLI.Println(fName)
			if err = appendTo(f, ctxDest); err != nil {
				return err
			}

			return nil

		}(); err != nil {
			return err
		}
	}

	if err := OptimizeContext(ctxDest); err != nil {
		return err
	}

	if conf.ValidationMode != model.ValidationNone {
		if err := ValidateContext(ctxDest); err != nil {
			return err
		}
	}

	return WriteContext(ctxDest, w)
}

func MergeCreateFile(inFiles []string, outFile string, conf *model.Configuration) (err error) {

	f, err := os.Create(outFile)
	if err != nil {
		return err
	}

	defer func() {
		err = f.Close()
	}()

	log.CLI.Printf("writing %s...\n", outFile)
	return Merge("", inFiles, f, conf)
}

func MergeAppendFile(inFiles []string, outFile string, conf *model.Configuration) (err error) {

	tmpFile := outFile
	overWrite := false
	destFile := ""

	if fileExists(outFile) {
		overWrite = true
		destFile = outFile
		tmpFile += ".tmp"
		log.CLI.Printf("appending to %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", outFile)
	}

	f, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if err = f.Close(); err != nil {
				return
			}
			if overWrite {
				err = os.Remove(tmpFile)
			}
			return
		}
		if err = f.Close(); err != nil {
			return
		}
		if overWrite {
			err = os.Rename(tmpFile, outFile)
		}
	}()

	return Merge(destFile, inFiles, f, conf)
}

// Merge merges a sequence of PDF streams and writes the result to w.
func MergeOld(rsc []io.ReadSeeker, w io.Writer, conf *model.Configuration) error {

	if rsc == nil {
		return errors.New("pdfcpu: Merge: Please provide rsc")
	}

	if w == nil {
		return errors.New("pdfcpu: Merge: Please provide w")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.MERGECREATE

	ctxDest, _, _, err := readAndValidate(rsc[0], conf, time.Now())
	if err != nil {
		return err
	}

	ctxDest.EnsureVersionForWriting()

	for _, f := range rsc[1:] {
		if err = appendTo(f, ctxDest); err != nil {
			return err
		}
	}

	if err = OptimizeContext(ctxDest); err != nil {
		return err
	}

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctxDest); err != nil {
			return err
		}
	}

	return WriteContext(ctxDest, w)
}

// MergeCreateFile merges a sequence of inFiles and writes the result to outFile.
// This operation corresponds to file concatenation in the order specified by inFiles.
// The first entry of inFiles serves as the destination context where all remaining files get merged into.
func MergeCreateFileOld(inFiles []string, outFile string, conf *model.Configuration) error {
	ff := []*os.File(nil)
	for _, f := range inFiles {
		log.CLI.Println(f)
		f, err := os.Open(f)
		if err != nil {
			return err
		}
		ff = append(ff, f)
	}
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f.Close()
			for _, f := range ff {
				f.Close()
			}
			return
		}
		if err = f.Close(); err != nil {
			return
		}
		for _, f := range ff {
			if err = f.Close(); err != nil {
				return
			}
		}
	}()

	rs := make([]io.ReadSeeker, len(ff))
	for i, f := range ff {
		rs[i] = f
	}

	log.CLI.Printf("writing %s...\n", outFile)
	return MergeOld(rs, f, conf)
}

func prepareReadSeekers(ff []*os.File) []io.ReadSeeker {
	rss := make([]io.ReadSeeker, len(ff))
	for i, f := range ff {
		rss[i] = f
	}
	return rss
}

// MergeAppendFile merges a sequence of inFiles and writes the result to outFile.
// This operation corresponds to file concatenation in the order specified by inFiles.
// If outFile already exists, inFiles will be appended.
func MergeAppendFileOld(inFiles []string, outFile string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	tmpFile := outFile
	if fileExists(outFile) {
		if f1, err = os.Open(outFile); err != nil {
			return err
		}
		tmpFile += ".tmp"
		log.CLI.Printf("appending to %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", outFile)
	}

	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	ff := []*os.File(nil)
	if f1 != nil {
		ff = append(ff, f1)
	}
	for _, f := range inFiles {
		log.CLI.Println(f)
		f, err := os.Open(f)
		if err != nil {
			return err
		}
		ff = append(ff, f)
	}

	defer func() {
		if err != nil {
			f2.Close()
			if f1 != nil {
				os.Remove(tmpFile)
			}
			for _, f := range ff {
				f.Close()
			}
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		for _, f := range ff {
			if err = f.Close(); err != nil {
				return
			}
		}
		if f1 != nil {
			if err = os.Rename(tmpFile, outFile); err != nil {
				return
			}
		}
	}()

	return MergeOld(prepareReadSeekers(ff), f2, conf)
}
