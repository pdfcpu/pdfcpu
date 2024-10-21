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
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// Images returns all embedded images of rs.
func Images(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) ([]map[int]model.Image, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: ListImages: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTIMAGES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return nil, err
	}

	ii, _, err := pdfcpu.Images(ctx, pages)

	return ii, err
}

// UpdateImages replaces the XObject identified by objNr or (pageNr and resourceId).
func UpdateImages(rs io.ReadSeeker, rd io.Reader, w io.Writer, objNr, pageNr int, id string, conf *model.Configuration) error {

	if rs == nil {
		return errors.New("pdfcpu: UpdateImages: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.UPDATEIMAGES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	if objNr > 0 {
		if err := pdfcpu.UpdateImagesByObjNr(ctx, rd, objNr); err != nil {
			return err
		}

		return Write(ctx, w, conf)
	}

	if pageNr == 0 || id == "" {
		return errors.New("pdfcpu: UpdateImages: missing pageNr or id ")
	}

	if err := pdfcpu.UpdateImagesByPageNrAndId(ctx, rd, pageNr, id); err != nil {
		return err
	}

	return Write(ctx, w, conf)
}

func ensurePageNrAndId(pageNr *int, id *string, imageFile string) (err error) {
	// If objNr and pageNr and id are not set, we assume an image filename produced by "pdfcpu image list" and parse this info.
	// eg. mountain_1_Im0.png => pageNr:1, id:Im0

	if *pageNr > 0 && *id != "" {
		return nil
	}

	s := strings.TrimSuffix(imageFile, filepath.Ext(imageFile))

	ss := strings.Split(s, "_")

	if len(ss) < 3 {
		return errors.Errorf("pdfcpu: invalid image filename:%s - must conform to output filename of \"pdfcpu extract\"", imageFile)
	}

	*id = ss[len(ss)-1]

	*pageNr, err = strconv.Atoi(ss[len(ss)-2])
	if err != nil {
		return err
	}

	return nil
}

// UpdateImagesFile replaces the XObject identified by objNr or (pageNr and resourceId).
func UpdateImagesFile(inFile, imageFile, outFile string, objNr, pageNr int, id string, conf *model.Configuration) (err error) {

	if objNr < 1 {
		if err = ensurePageNrAndId(&pageNr, &id, imageFile); err != nil {
			return err
		}
	}

	var f0, f1, f2 *os.File

	if f0, err = os.Open(inFile); err != nil {
		return err
	}

	if f1, err = os.Open(imageFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			f0.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if err = f0.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	return UpdateImages(f0, f1, f2, objNr, pageNr, id, conf)
}
