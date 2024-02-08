/*
Copyright 2023 The pdfcpu Authors.

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

// Package cli provides pdfcpu command line processing.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

func listAttachments(rs io.ReadSeeker, conf *model.Configuration, withDesc, sorted bool) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: listAttachments: missing rs")
	}
	var err error
	if conf == nil {
		conf, err = model.NewDefaultConfiguration()
		if err != nil {
			return nil, err
		}
	}
	conf.Cmd = model.LISTATTACHMENTS

	ctx, _, _, _, err := api.ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

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

	return ss, nil
}

// ListAttachmentsFile returns a list of embedded file attachments of inFile with optional description.
func ListAttachmentsFile(inFile string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return listAttachments(f, conf, true, true)
}

// ListAttachmentsCompactFile returns a list of embedded file attachments of inFile w/o optional description.
func ListAttachmentsCompactFile(inFile string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return listAttachments(f, conf, false, false)
}

func listAnnotations(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (int, []string, error) {
	annots, err := api.Annotations(rs, selectedPages, conf)
	if err != nil {
		return 0, nil, err
	}

	return pdfcpu.ListAnnotations(annots)
}

// ListAnnotationsFile returns a list of page annotations of inFile.
func ListAnnotationsFile(inFile string, selectedPages []string, conf *model.Configuration) (int, []string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()

	return listAnnotations(f, selectedPages, conf)
}

func listBoxes(rs io.ReadSeeker, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: listBoxes: missing rs")
	}

	var err error
	if conf == nil {
		conf, err = model.NewDefaultConfiguration()
		if err != nil {
			return nil, err
		}
	}
	conf.Cmd = model.LISTBOXES

	ctx, _, _, _, err := api.ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return nil, err
	}

	pages, err := api.PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return nil, err
	}

	return ctx.ListPageBoundaries(pages, pb)
}

// ListBoxesFile returns a list of page boundaries for selected pages of inFile.
func ListBoxesFile(inFile string, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if pb == nil {
		pb = &model.PageBoundaries{}
		pb.SelectAll()
	}
	log.CLI.Printf("listing %s for %s\n", pb, inFile)

	return listBoxes(f, selectedPages, pb, conf)
}

func listFormFields(rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	var err error
	if conf == nil {
		conf, err = model.NewDefaultConfiguration()
		if err != nil {
			return nil, err
		}
	}
	conf.Cmd = model.LISTFORMFIELDS

	ctx, _, _, _, err := api.ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return nil, err
	}

	return form.ListFormFields(ctx)
}

// ListFormFieldsFile returns a list of form field ids in inFile.
func ListFormFieldsFile(inFiles []string, conf *model.Configuration) ([]string, error) {
	log.SetCLILogger(nil)

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

		output, err := listFormFields(f, conf)
		if err != nil {
			if len(inFiles) > 1 {
				ss = append(ss, fmt.Sprintf("\n%s:\n%v", fn, err))
				continue
			}
			return nil, err
		}

		ss = append(ss, "\n"+fn+":\n")
		ss = append(ss, output...)
	}

	return ss, nil
}

func listImages(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: listImages: Please provide rs")
	}

	var err error
	if conf == nil {
		conf, err = model.NewDefaultConfiguration()
		if err != nil {
			return nil, err
		}
	}
	conf.Cmd = model.LISTIMAGES

	ctx, _, _, _, err := api.ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return nil, err
	}

	pages, err := api.PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return nil, err
	}

	return pdfcpu.ListImages(ctx, pages)
}

// ListImagesFile returns a formatted list of embedded images of inFile.
func ListImagesFile(inFiles []string, selectedPages []string, conf *model.Configuration) ([]string, error) {
	if len(selectedPages) == 0 {
		log.CLI.Printf("pages: all\n")
	}

	log.SetCLILogger(nil)

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
		output, err := listImages(f, selectedPages, conf)
		if err != nil {
			if len(inFiles) > 1 {
				ss = append(ss, fmt.Sprintf("\n%s: %v", fn, err))
				continue
			}
			return nil, err
		}
		ss = append(ss, "\n"+fn+":")
		ss = append(ss, output...)
	}

	return ss, nil
}

// ListInfoFile returns formatted information about inFile.
func ListInfoFile(inFile string, selectedPages []string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := api.PDFInfo(f, inFile, selectedPages, conf)
	if err != nil {
		return nil, err
	}

	pages, err := api.PagesForPageSelection(info.PageCount, selectedPages, false, false)
	if err != nil {
		return nil, err
	}

	ss, err := pdfcpu.ListInfo(info, pages)
	if err != nil {
		return nil, err
	}

	return append([]string{inFile + ":"}, ss...), err
}

func listInfoFilesJSON(inFiles []string, selectedPages []string, conf *model.Configuration) ([]string, error) {
	var infos []*pdfcpu.PDFInfo

	for _, fn := range inFiles {

		f, err := os.Open(fn)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		info, err := api.PDFInfo(f, fn, selectedPages, conf)
		if err != nil {
			return nil, err
		}

		infos = append(infos, info)
	}

	s := struct {
		Header pdfcpu.Header `json:"header"`
		Infos  []*pdfcpu.PDFInfo
	}{
		Header: pdfcpu.Header{Version: "pdfcpu " + model.VersionStr, Creation: time.Now().Format("2006-01-02 15:04:05 MST")},
		Infos:  infos,
	}

	bb, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return nil, err
	}

	return []string{string(bb)}, nil
}

// ListInfoFiles returns formatted information about inFiles.
func ListInfoFiles(inFiles []string, selectedPages []string, json bool, conf *model.Configuration) ([]string, error) {

	if json {
		return listInfoFilesJSON(inFiles, selectedPages, conf)
	}

	var ss []string

	for i, fn := range inFiles {
		if i > 0 {
			ss = append(ss, "")
		}
		ssx, err := ListInfoFile(fn, selectedPages, conf)
		if err != nil {
			if len(inFiles) == 1 {
				return nil, err
			}
			fmt.Fprintf(os.Stderr, "%s: %v\n", fn, err)
		}
		ss = append(ss, ssx...)
	}

	return ss, nil
}

// ListKeywordsFile returns the keyword list of inFile.
func ListKeywordsFile(inFile string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return api.Keywords(f, conf)
}

func listPermissions(rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: listPermissions: missing rs")
	}

	var err error
	if conf == nil {
		conf, err = model.NewDefaultConfiguration()
		if err != nil {
			return nil, err
		}
	}
	conf.Cmd = model.LISTPERMISSIONS

	ctx, _, _, _, err := api.ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	return pdfcpu.Permissions(ctx), nil
}

// ListPermissionsFile returns a list of user access permissions for inFile.
func ListPermissionsFile(inFiles []string, conf *model.Configuration) ([]string, error) {
	log.SetCLILogger(nil)

	var ss []string

	for i, fn := range inFiles {
		if i > 0 {
			ss = append(ss, "")
		}
		f, err := os.Open(fn)
		if err != nil {
			return nil, err
		}
		defer func() {
			f.Close()
		}()
		ssx, err := listPermissions(f, conf)
		if err != nil {
			if len(inFiles) == 1 {
				return nil, err
			}
			fmt.Fprintf(os.Stderr, "%s: %v\n", fn, err)
		}
		ss = append(ss, fn+":")
		ss = append(ss, ssx...)
	}

	return ss, nil
}

func listProperties(rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: listProperties: missing rs")
	}

	var err error
	if conf == nil {
		conf, err = model.NewDefaultConfiguration()
		if err != nil {
			return nil, err
		}
	} else {
		// Validation loads infodict.
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTPROPERTIES

	ctx, _, _, _, err := api.ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	return pdfcpu.PropertiesList(ctx)
}

// ListPropertiesFile returns the property list of inFile.
func ListPropertiesFile(inFile string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return listProperties(f, conf)
}

func listBookmarks(rs io.ReadSeeker, conf *model.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: listBookmarks: missing rs")
	}

	var err error
	if conf == nil {
		conf, err = model.NewDefaultConfiguration()
		if err != nil {
			return nil, err
		}
	} else {
		// Validation loads infodict.
		conf.ValidationMode = model.ValidationRelaxed
	}
	conf.Cmd = model.LISTBOOKMARKS

	ctx, _, _, _, err := api.ReadValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}

	return pdfcpu.BookmarkList(ctx)
}

// ListBookmarksFile returns the bookmarks of inFile.
func ListBookmarksFile(inFile string, conf *model.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return listBookmarks(f, conf)
}
