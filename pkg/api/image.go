/*
Copyright 2021 The pdfcpu Authors.

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
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// ListImages returns a list of embedded images of rs.
func ListImages(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: ListImages: Please provide rs")
	}
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.LISTIMAGES
	}
	ctx, _, _, _, err := readValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}
	if err := ctx.EnsurePageCount(); err != nil {
		return nil, err
	}
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return nil, err
	}

	return pdfcpu.ListImages(ctx, pages)
}

// ListImagesFile returns a list of embedded images of inFile.
func ListImagesFile(inFiles []string, selectedPages []string, conf *model.Configuration) ([]string, error) {
	if len(selectedPages) == 0 {
		log.CLI.Printf("pages: all\n")
	}
	ss := []string{}
	for _, fn := range inFiles {
		f, err := os.Open(fn)
		if err != nil {
			if len(inFiles) > 1 {
				ss = append(ss, fmt.Sprintf("\ncan't open %s: %v", fn, err))
				continue
			}
			return nil, err
		}
		defer f.Close()
		output, err := ListImages(f, selectedPages, conf)
		if err != nil {
			if len(inFiles) > 1 {
				ss = append(ss, fmt.Sprintf("\n%s: %v", fn, err))
				continue
			}
			return nil, err
		}
		ss = append(ss, "\n"+fn)
		ss = append(ss, output...)
	}
	return ss, nil
}
