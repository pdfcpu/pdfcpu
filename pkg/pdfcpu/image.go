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

package pdfcpu

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// Images returns all embedded images of ctx.
func Images(ctx *model.Context, selectedPages types.IntSet) ([]map[int]model.Image, *ImageListMaxLengths, error) {
	pageNrs := []int{}
	for k, v := range selectedPages {
		if !v {
			continue
		}
		pageNrs = append(pageNrs, k)
	}
	sort.Ints(pageNrs)

	mm := []map[int]model.Image{}
	var (
		maxLenObjNr, maxLenID, maxLenSize, maxLenFilters int
	)

	for _, i := range pageNrs {
		m, err := ExtractPageImages(ctx, i, true)
		if err != nil {
			return nil, nil, err
		}
		if len(m) == 0 {
			continue
		}
		for _, i := range m {
			s := strconv.Itoa(i.ObjNr)
			if len(s) > maxLenObjNr {
				maxLenObjNr = len(s)
			}
			if len(i.Name) > maxLenID {
				maxLenID = len(i.Name)
			}
			lenSize := len(types.ByteSize(i.Size).String())
			if lenSize > maxLenSize {
				maxLenSize = lenSize
			}
			if len(i.Filter) > maxLenFilters {
				maxLenFilters = len(i.Filter)
			}
		}
		mm = append(mm, m)
	}

	maxLen := &ImageListMaxLengths{ObjNr: maxLenObjNr, ID: maxLenID, Size: maxLenSize, Filters: maxLenFilters}

	return mm, maxLen, nil
}

func prepHorSep(horSep *[]int, maxLen *ImageListMaxLengths) string {
	s := "Page Obj# "
	if maxLen.ObjNr > 4 {
		s += strings.Repeat(" ", maxLen.ObjNr-4)
		*horSep = append(*horSep, 10+maxLen.ObjNr-4)
	} else {
		*horSep = append(*horSep, 10)
	}

	s += draw.VBar + " Id "
	if maxLen.ID > 2 {
		s += strings.Repeat(" ", maxLen.ID-2)
		*horSep = append(*horSep, 4+maxLen.ID-2)
	} else {
		*horSep = append(*horSep, 4)
	}

	s += draw.VBar + " Type  SoftMask ImgMask "
	*horSep = append(*horSep, 24)

	s += draw.VBar + " Width " + draw.VBar + " Height " + draw.VBar + " ColorSpace Comp bpc Interp "
	*horSep = append(*horSep, 7, 8, 28)

	s += draw.VBar + " "
	if maxLen.Size > 4 {
		s += strings.Repeat(" ", maxLen.Size-4)
		*horSep = append(*horSep, 6+maxLen.Size-4)
	} else {
		*horSep = append(*horSep, 6)
	}
	s += "Size " + draw.VBar + " Filters"
	if maxLen.Filters > 7 {
		*horSep = append(*horSep, 8+maxLen.Filters-7)
	} else {
		*horSep = append(*horSep, 8)
	}

	return s
}

func sortedObjNrs(ii map[int]model.Image) []int {
	objNrs := []int{}
	for k := range ii {
		objNrs = append(objNrs, k)
	}
	sort.Ints(objNrs)
	return objNrs
}

func listImages(mm []map[int]model.Image, maxLen *ImageListMaxLengths) ([]string, int, int64, error) {
	ss := []string{}
	first := true
	j, size := 0, int64(0)
	m := map[int]bool{}
	horSep := []int{}
	for _, ii := range mm {
		if first {
			s := prepHorSep(&horSep, maxLen)
			ss = append(ss, s)
			first = false
		}
		ss = append(ss, draw.HorSepLine(horSep))

		newPage := true

		for _, objNr := range sortedObjNrs(ii) {
			img := ii[objNr]
			pageNr := ""
			if newPage {
				pageNr = strconv.Itoa(img.PageNr)
				newPage = false
			}
			t := "image"
			if img.IsImgMask {
				t = "imask"
			}
			if img.Thumb {
				t = "thumb"
			}

			sm := " "
			if img.HasSMask {
				sm = "*"
			}

			im := " "
			if img.HasImgMask {
				im = "*"
			}

			bpc := "-"
			if img.Bpc > 0 {
				bpc = strconv.Itoa(img.Bpc)
			}

			interp := " "
			if img.Interpol {
				interp = "*"
			}

			s := strconv.Itoa(img.ObjNr)
			fill1 := strings.Repeat(" ", maxLen.ObjNr-len(s))
			if maxLen.ObjNr < 4 {
				fill1 += strings.Repeat(" ", 4-maxLen.ObjNr)
			}

			fill2 := strings.Repeat(" ", maxLen.ID-len(img.Name))
			if maxLen.ID < 2 {
				fill2 += strings.Repeat(" ", 2-maxLen.ID-len(img.Name))
			}

			sizeStr := types.ByteSize(img.Size).String()
			fill3 := strings.Repeat(" ", maxLen.Size-len(sizeStr))
			if maxLen.Size < 4 {
				fill3 = strings.Repeat(" ", 4-maxLen.Size)
			}

			ss = append(ss, fmt.Sprintf("%4s %s%s %s %s%s %s %s    %s        %s    %s %5d %s  %5d %s %10s    %d   %s    %s   %s %s%s %s %s",
				pageNr, fill1, strconv.Itoa(img.ObjNr), draw.VBar,
				fill2, img.Name, draw.VBar,
				t, sm, im, draw.VBar,
				img.Width, draw.VBar,
				img.Height, draw.VBar,
				img.Cs, img.Comp, bpc, interp, draw.VBar,
				fill3, sizeStr, draw.VBar, img.Filter))

			if !m[img.ObjNr] {
				m[img.ObjNr] = true
				j++
				size += img.Size
			}
		}
	}
	return ss, j, size, nil
}

type ImageListMaxLengths struct {
	ObjNr, ID, Size, Filters int
}

// ListImages returns a formatted list of embedded images.
func ListImages(ctx *model.Context, selectedPages types.IntSet) ([]string, error) {

	mm, maxLen, err := Images(ctx, selectedPages)
	if err != nil {
		return nil, err
	}

	ss, j, size, err := listImages(mm, maxLen)
	if err != nil {
		return nil, err
	}

	s := fmt.Sprintf("%d images available", j)

	if j > 0 {
		s += fmt.Sprintf(" (%s)", types.ByteSize(size))
	}

	return append([]string{s}, ss...), nil
}

// WriteImageToDisk returns a closure for writing img to disk.
func WriteImageToDisk(outDir, fileName string) func(model.Image, bool, int) error {
	return func(img model.Image, singleImgPerPage bool, maxPageDigits int) error {
		if img.Reader == nil {
			return nil
		}
		s := "%s_%" + fmt.Sprintf("0%dd", maxPageDigits)
		qual := img.Name
		if img.Thumb {
			qual = "thumb"
		}
		f := fmt.Sprintf(s+"_%s.%s", fileName, img.PageNr, qual, img.FileType)
		outFile := filepath.Join(outDir, f)
		log.CLI.Printf("writing %s\n", outFile)
		return WriteReader(outFile, img)
	}
}

func validateImageDimensions(ctx *model.Context, objNr, w, h int) error {
	imgObj := ctx.Optimize.ImageObjects[objNr]
	if imgObj == nil {
		return errors.Errorf("pdfcpu: unknown image object for objNr=%d", objNr)
	}

	d := imgObj.ImageDict

	width := d.IntEntry("Width")
	height := d.IntEntry("Height")

	if width == nil || height == nil {
		return errors.New("pdfcpu: corrupt image dict")
	}

	if *width != w || *height != h {
		return errors.Errorf("pdfcpu: invalid image dimensions, want(%d,%d), got(%d,%d)", w, h, *width, *height)
	}

	return nil
}

// UpdateImagesByObjNr replaces an XObject.
func UpdateImagesByObjNr(ctx *model.Context, rd io.Reader, objNr int) error {

	sd, w, h, err := model.CreateImageStreamDict(ctx.XRefTable, rd, false, false)
	if err != nil {
		return err
	}

	if err := validateImageDimensions(ctx, objNr, w, h); err != nil {
		return err
	}

	genNr := 0
	entry, ok := ctx.FindTableEntry(objNr, genNr)
	if !ok {
		errors.Errorf("pdfcpu: invalid objNr=%d", objNr)
	}

	entry.Object = *sd

	return nil
}

func isInheritedXObjectResource(inhRes types.Dict, id string) bool {
	if inhRes == nil {
		return false
	}

	d := inhRes.DictEntry("XObject")
	if d == nil {
		return false
	}

	for resId := range d {
		if resId == id {
			return true
		}
	}

	return false
}

// UpdateImagesByPageNrAndId replaces the XObject referenced by pageNr and id.
func UpdateImagesByPageNrAndId(ctx *model.Context, rd io.Reader, pageNr int, id string) error {

	imgIndRef, w, h, err := model.CreateImageResource(ctx.XRefTable, rd, false, false)
	if err != nil {
		return err
	}

	d, _, inhPAttrs, err := ctx.PageDict(pageNr, false)
	if err != nil {
		return err
	}

	obj, found := d.Find("Resources")
	if !found {
		if isInheritedXObjectResource(inhPAttrs.Resources, id) {
			d1 := types.NewDict()
			d1[id] = *imgIndRef
			d2 := types.NewDict()
			d2["XObject"] = d1
			d["Resources"] = d2
			return nil
		}
		return errors.Errorf("pdfcpu: page %d: unknown resource %s\n", pageNr, id)
	}

	resDict, err := ctx.DereferenceDict(obj)
	if err != nil {
		return err
	}

	obj1, ok := resDict.Find("XObject")
	if !ok {
		if isInheritedXObjectResource(inhPAttrs.Resources, id) {
			d := types.NewDict()
			d[id] = *imgIndRef
			resDict["XObject"] = d
			return nil
		}
		return errors.Errorf("pdfcpu: page %d: unknown resource %s\n", pageNr, id)
	}

	imgResDict, err := ctx.DereferenceDict(obj1)
	if err != nil {
		return err
	}

	for resId, indRef := range imgResDict {
		if resId == id {

			ir := indRef.(types.IndirectRef)
			if err := validateImageDimensions(ctx, ir.ObjectNumber.Value(), w, h); err != nil {
				return err
			}

			imgResDict[id] = *imgIndRef
			return nil
		}
	}

	if isInheritedXObjectResource(inhPAttrs.Resources, id) {
		imgResDict[id] = *imgIndRef
		return nil
	}

	return errors.Errorf("pdfcpu: page %d: unknown resource %s\n", pageNr, id)
}
