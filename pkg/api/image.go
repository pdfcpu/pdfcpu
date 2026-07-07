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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func prepareImagesContext(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (*model.Context, types.IntSet, error) {
	if rs == nil {
		return nil, nil, ErrMissingPDFReadSeeker
	}
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTIMAGES
	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, nil, fmt.Errorf("list images: %w", err)
	}
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return nil, nil, fmt.Errorf("list images: parse page selection: %w", err)
	}
	return ctx, pages, nil
}

// Images returns all embedded images of rs.
func Images(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (ii []map[int]model.Image, err error) {
	defer fault.Catch(&err)

	ctx, pages, err := prepareImagesContext(rs, selectedPages, conf)
	if err != nil {
		return nil, err
	}
	ii, _, err = pdfcpu.Images(ctx, pages)
	if err != nil {
		return nil, fmt.Errorf("list images: collect images: %w", err)
	}
	return ii, nil
}

// ListImages returns a formatted list of all embedded images of rs.
func ListImages(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (ss []string, err error) {
	defer fault.Catch(&err)

	ctx, pages, err := prepareImagesContext(rs, selectedPages, conf)
	if err != nil {
		return nil, err
	}
	ss, err = pdfcpu.ListImages(ctx, pages)
	if err != nil {
		return nil, fmt.Errorf("list images: format image list: %w", err)
	}
	return ss, nil
}

func validateImageSelection(objNr, pageNr int, id string) error {
	if objNr < 0 {
		return fmt.Errorf("negative object number %d: %w", objNr, ErrInvalidImageSelection)
	}
	if objNr > 0 {
		if pageNr != 0 || id != "" {
			return fmt.Errorf("object number conflicts with page resource: %w", ErrInvalidImageSelection)
		}
		return nil
	}
	if pageNr < 1 {
		return fmt.Errorf("missing page number: %w", ErrInvalidImageSelection)
	}
	if id == "" {
		return fmt.Errorf("missing resource id: %w", ErrInvalidImageSelection)
	}
	return nil
}

// UpdateImages replaces the XObject identified by objNr or (pageNr and resourceId).
func UpdateImages(rs io.ReadSeeker, rd io.Reader, w io.Writer, objNr, pageNr int, id string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if rd == nil {
		return ErrMissingImageInput
	}
	if w == nil {
		return ErrMissingPDFWriter
	}
	if err := validateImageSelection(objNr, pageNr, id); err != nil {
		return fmt.Errorf("update images: validate selection: %w", err)
	}
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.UPDATEIMAGES
	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("update images: %w", err)
	}
	if objNr > 0 {
		if err := pdfcpu.UpdateImagesByObjNr(ctx, rd, objNr); err != nil {
			return fmt.Errorf("update images: replace image object: %w", err)
		}
	} else if err := pdfcpu.UpdateImagesByPageNrAndId(ctx, rd, pageNr, id); err != nil {
		return fmt.Errorf("update images: replace page resource: %w", err)
	}
	if err := Write(ctx, w, conf); err != nil {
		return fmt.Errorf("update images: write output: %w", err)
	}
	return nil
}

func imageSelectionFromFile(imageFile string) (int, string, error) {
	base := filepath.Base(imageFile)
	s := strings.TrimSuffix(base, filepath.Ext(base))
	ss := strings.Split(s, "_")
	if len(ss) < 3 {
		return 0, "", fmt.Errorf(
			"parse image filename %s: expected at least three underscore-separated segments: %w",
			imageFile,
			ErrInvalidImageSelection,
		)
	}
	id := ss[len(ss)-1]
	pageToken := ss[len(ss)-2]
	pageNr, err := strconv.Atoi(pageToken)
	if err != nil {
		return 0, "", fmt.Errorf("parse image filename %s page number %q: %w", imageFile, pageToken, err)
	}
	return pageNr, id, nil
}

func resolveImageSelection(imageFile string, objNr, pageNr int, id string) (int, string, error) {
	if objNr == 0 && pageNr == 0 && id == "" {
		var err error
		if pageNr, id, err = imageSelectionFromFile(imageFile); err != nil {
			return 0, "", err
		}
	}
	if err := validateImageSelection(objNr, pageNr, id); err != nil {
		return 0, "", err
	}
	return pageNr, id, nil
}

// ValidateUpdateImagesOutput ensures outFile does not alias imageFile.
func ValidateUpdateImagesOutput(imageFile, outFile string) error {
	if imageFile == "" {
		return ErrMissingImageInput
	}
	if outFile == "" {
		return nil
	}
	aliases, err := outputAliasesInput(imageFile, outFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("update images: compare image input and output: %w", err)
	}
	if aliases {
		return fmt.Errorf("output %s aliases image input %s: %w", outFile, imageFile, ErrUpdateImagesOutputConflict)
	}
	return nil
}

// UpdateImagesFile replaces the XObject identified by objNr or (pageNr and resourceId).
func UpdateImagesFile(inFile, imageFile, outFile string, objNr, pageNr int, id string, conf *model.Configuration) (err error) {
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if imageFile == "" {
		return ErrMissingImageInput
	}
	if pageNr, id, err = resolveImageSelection(imageFile, objNr, pageNr, id); err != nil {
		return fmt.Errorf("update images: validate selection: %w", err)
	}
	if err := ValidateUpdateImagesOutput(imageFile, outFile); err != nil {
		return err
	}
	var f0, f1, f2 *os.File
	ok := false
	if f0, err = os.Open(inFile); err != nil {
		return fmt.Errorf("update images: open input %s: %w", inFile, err)
	}
	if f1, err = os.Open(imageFile); err != nil {
		return errors.Join(
			fmt.Errorf("update images: open image %s: %w", imageFile, err),
			closeFile(f0, "update images: close input"),
		)
	}
	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "update images")
	if err != nil {
		return errors.Join(
			fmt.Errorf("update images: create output: %w", err),
			closeFile(f1, "update images: close image"),
			closeFile(f0, "update images: close input"),
		)
	}
	f2 = staged.output.file
	staged.inputs[0].context = "update images: close image"
	staged = staged.withInput(f0, "update images: close input")
	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()
	if err = UpdateImages(f0, f1, f2, objNr, pageNr, id, conf); err != nil {
		return err
	}
	ok = true
	return nil
}
