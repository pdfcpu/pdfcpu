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
)

// Image is a Reader representing an image resource.
type Image struct {
	io.Reader
	Name        string // Resource name
	FileType    string
	pageNr      int
	objNr       int
	width       int    // "Width"
	height      int    // "Height"
	bpc         int    // "BitsPerComponent"
	cs          string // "ColorSpace"
	comp        int    // color component count
	sMask       bool   // "SMask"
	imgMask     bool   // "ImageMask"
	thumb       bool   // "Thumbnail"
	interpol    bool   // "Interpolate"
	size        int64  // "Length"
	filter      string // filter pipeline
	decodeParms string
}

func (ctx *Context) listImages(iii [][]Image, maxLenID, maxLenSize int) ([]string, int, error) {
	ss := []string{}
	first := true
	j := 0
	for _, ii := range iii {
		if first {
			s1 := ("page  obj# ")
			s2 := fmt.Sprintf("%%%ds", maxLenID)
			s3 := "  type width height colorspace comp bpc interp "
			s4 := fmt.Sprintf("%%%ds", maxLenSize)
			s5 := " filters"
			s := fmt.Sprintf(s1+s2+s3+s4+s5, "id", "size")
			ss = append(ss, s)
			ss = append(ss, strings.Repeat("=", len(s)))
			first = false
		}
		newPage := true
		for _, img := range ii {
			pageNr := ""
			if newPage {
				pageNr = strconv.Itoa(img.pageNr)
				newPage = false
			}
			t := "image"
			if img.sMask {
				t = "smask"
			}
			if img.imgMask {
				t = "imask"
			}
			if img.thumb {
				t = "thumb"
			}
			bpc := "-"
			if img.bpc > 0 {
				bpc = strconv.Itoa(img.bpc)
			}

			interp := " "
			if img.interpol {
				interp = "*"
			}

			sID := fmt.Sprintf("%%%ds", maxLenID)
			sSize := fmt.Sprintf("%%%ds", maxLenSize)

			ss = append(ss, fmt.Sprintf("%4s %5d "+sID+" %s %5d  %5d %10s    %d   %s    %s   "+sSize+" %s",
				pageNr, img.objNr, img.Name, t, img.width, img.height, img.cs, img.comp, bpc, interp, ByteSize(img.size), img.filter))
			j++
		}
	}
	return ss, j, nil
}

// ListImages returns a list of embedded images.
func (ctx *Context) ListImages(selectedPages IntSet) ([]string, error) {
	pageNrs := []int{}
	for k, v := range selectedPages {
		if !v {
			continue
		}
		pageNrs = append(pageNrs, k)
	}
	sort.Ints(pageNrs)

	iii := [][]Image{}
	var (
		maxLenID, maxLenSize int
	)

	for _, i := range pageNrs {
		ii, err := ctx.ExtractPageImages(i, true)
		if err != nil {
			return nil, err
		}
		if len(ii) == 0 {
			continue
		}
		for _, i := range ii {
			if len(i.Name) > maxLenID {
				maxLenID = len(i.Name)
			}
			lenSize := len(ByteSize(i.size).String())
			if lenSize > maxLenSize {
				maxLenSize = lenSize
			}
		}
		iii = append(iii, ii)
	}

	ss, j, err := ctx.listImages(iii, maxLenID, maxLenSize)
	if err != nil {
		return nil, err
	}

	return append([]string{fmt.Sprintf("%d images available", j)}, ss...), nil
}

// WriteImageToDisk returns a closure for writing img to disk.
func WriteImageToDisk(outDir, fileName string) func(Image, bool, int) error {
	return func(img Image, singleImgPerPage bool, maxPageDigits int) error {
		s := "%s_%" + fmt.Sprintf("0%dd", maxPageDigits)
		qual := img.Name
		if img.thumb {
			qual = "thumb"
		}
		f := fmt.Sprintf(s+"_%s.%s", fileName, img.pageNr, qual, img.FileType)
		if singleImgPerPage {
			if img.thumb {
				s += "_" + qual
			}
			f = fmt.Sprintf(s+".%s", fileName, img.pageNr, img.FileType)
		}
		outFile := filepath.Join(outDir, f)
		log.CLI.Printf("writing %s\n", outFile)
		return WriteReader(outFile, img)
	}
}
