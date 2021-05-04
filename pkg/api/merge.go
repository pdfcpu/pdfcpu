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
	"github.com/pkg/errors"
)

// appendTo appends inFile to ctxDest's page tree.
func appendTo(rs io.ReadSeeker, ctxDest *pdfcpu.Context) error {
	ctxSource, _, _, err := readAndValidate(rs, ctxDest.Configuration, time.Now())
	if err != nil {
		return err
	}

	// Merge the source context into the dest context.
	return pdfcpu.MergeXRefTables(ctxSource, ctxDest)
}

// ReadSeekerCloser combines io.ReadSeeker and io.Closer
type ReadSeekerCloser interface {
	io.ReadSeeker
	io.Closer
}

// Merge merges a sequence of PDF streams and writes the result to w.
func Merge(rsc []io.ReadSeeker, w io.Writer, conf *pdfcpu.Configuration) error {
	if rsc == nil {
		return errors.New("pdfcpu: Merge: Please provide rsc")
	}
	if w == nil {
		return errors.New("pdfcpu: Merge: Please provide w")
	}
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.MERGECREATE

	ctxDest, _, _, err := readAndValidate(rsc[0], conf, time.Now())
	if err != nil {
		return err
	}

	ctxDest.EnsureVersionForWriting()

	// Repeatedly merge files into fileDest's xref table.
	for _, f := range rsc[1:] {
		if err = appendTo(f, ctxDest); err != nil {
			return err
		}
	}

	if err = OptimizeContext(ctxDest); err != nil {
		return err
	}

	if conf.ValidationMode != pdfcpu.ValidationNone {
		if err = ValidateContext(ctxDest); err != nil {
			return err
		}
	}

	return WriteContext(ctxDest, w)
}

// MergeCreateFile merges a sequence of inFiles and writes the result to outFile.
// This operation corresponds to file concatenation in the order specified by inFiles.
// The first entry of inFiles serves as the destination context where all remaining files get merged into.
func MergeCreateFile(inFiles []string, outFile string, conf *pdfcpu.Configuration) error {
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
	return Merge(rs, f, conf)
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
func MergeAppendFile(inFiles []string, outFile string, conf *pdfcpu.Configuration) (err error) {
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

	return Merge(prepareReadSeekers(ff), f2, conf)
}
