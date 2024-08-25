/*
Copyright 2022 The pdfcpu Authors.

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
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

type pagesParamMap map[string]func(string, *PageConfiguration) error

// Handle applies parameter completion and if successful
// parses the parameter values into pages.
func (m pagesParamMap) Handle(paramPrefix, paramValueStr string, pageConf *PageConfiguration) error {

	var param string

	// Completion support
	for k := range m {
		if !strings.HasPrefix(k, strings.ToLower(paramPrefix)) {
			continue
		}
		if len(param) > 0 {
			return errors.Errorf("pdfcpu: ambiguous parameter prefix \"%s\"", paramPrefix)
		}
		param = k
	}

	if param == "" {
		return errors.Errorf("pdfcpu: unknown parameter prefix \"%s\"", paramPrefix)
	}

	return m[param](paramValueStr, pageConf)
}

var pParamMap = pagesParamMap{
	"dimensions": parseDimensions,
	"formsize":   parsePageFormat,
	"papersize":  parsePageFormat,
}

// PageConfiguration represents the page config for the "pages insert" command.
type PageConfiguration struct {
	PageDim  *types.Dim        // page dimensions in display unit.
	PageSize string            // one of A0,A1,A2,A3,A4(=default),A5,A6,A7,A8,Letter,Legal,Ledger,Tabloid,Executive,ANSIC,ANSID,ANSIE.
	UserDim  bool              // true if one of dimensions or paperSize provided overriding the default.
	InpUnit  types.DisplayUnit // input display unit.
}

// DefaultPageConfiguration returns the default configuration.
func DefaultPageConfiguration() *PageConfiguration {
	return &PageConfiguration{
		PageDim:  types.PaperSize["A4"],
		PageSize: "A4",
		InpUnit:  types.POINTS,
	}
}

func (p PageConfiguration) String() string {
	return fmt.Sprintf("Page config: %s %s\n", p.PageSize, p.PageDim)
}

func parsePageFormat(s string, p *PageConfiguration) (err error) {
	if p.UserDim {
		return errors.New("pdfcpu: only one of formsize(papersize) or dimensions allowed")
	}
	p.PageDim, p.PageSize, err = types.ParsePageFormat(s)
	p.UserDim = true
	return err
}

func parseDimensions(s string, p *PageConfiguration) (err error) {
	if p.UserDim {
		return errors.New("pdfcpu: only one of formsize(papersize) or dimensions allowed")
	}
	p.PageDim, p.PageSize, err = ParsePageDim(s, p.InpUnit)
	p.UserDim = true
	return err
}

// ParsePageConfiguration parses a page configuration string into an internal structure.
func ParsePageConfiguration(s string, u types.DisplayUnit) (*PageConfiguration, error) {

	if s == "" {
		return nil, nil
	}

	pageConf := DefaultPageConfiguration()
	pageConf.InpUnit = u

	ss := strings.Split(s, ",")

	for _, s := range ss {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, errors.New("pdfcpu: Invalid page configuration string. Please consult pdfcpu help pages")
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := pParamMap.Handle(paramPrefix, paramValueStr, pageConf); err != nil {
			return nil, err
		}
	}

	return pageConf, nil
}

func addPages(
	ctxSrc, ctxDest *model.Context,
	pageNrs []int,
	usePgCache bool,
	pagesIndRef types.IndirectRef,
	pagesDict types.Dict,
	fieldsSrc, fieldsDest *types.Array,
	migrated map[int]int) error {

	// Used by collect, extractPages, split

	pageCache := map[int]*types.IndirectRef{}

	for _, i := range pageNrs {

		if usePgCache {
			if indRef, ok := pageCache[i]; ok {
				if err := model.AppendPageTree(indRef, 1, pagesDict); err != nil {
					return err
				}
				continue
			}
		}

		d, pageIndRef, inhPAttrs, err := ctxSrc.PageDict(i, true)
		if err != nil {
			return err
		}
		if d == nil {
			return errors.Errorf("pdfcpu: unknown page number: %d\n", i)
		}

		obj, err := migrateIndRef(pageIndRef, ctxSrc, ctxDest, migrated)
		if err != nil {
			return err
		}

		d = obj.(types.Dict)
		d["Resources"] = inhPAttrs.Resources.Clone()
		d["Parent"] = pagesIndRef
		d["MediaBox"] = inhPAttrs.MediaBox.Array()
		if inhPAttrs.Rotate%360 > 0 {
			d["Rotate"] = types.Integer(inhPAttrs.Rotate)
		}

		if err := migratePageDict(d, *pageIndRef, ctxSrc, ctxDest, migrated); err != nil {
			return err
		}

		if d["Annots"] != nil && len(*fieldsSrc) > 0 {
			if err := migrateFields(d, fieldsSrc, fieldsDest, ctxSrc, ctxDest, migrated); err != nil {
				return err
			}
		}

		if err := model.AppendPageTree(pageIndRef, 1, pagesDict); err != nil {
			return err
		}

		if usePgCache {
			pageCache[i] = pageIndRef
		}
	}

	return nil
}

func migrateNamedDests(ctxSrc *model.Context, n *model.Node, migrated map[int]int) error {
	patchValues := func(xRefTable *model.XRefTable, k string, v *types.Object) error {
		arr, err := xRefTable.DereferenceArray(*v)
		if err == nil {
			arr[0] = patchObject(arr[0], migrated)
			*v = arr
			return nil
		}
		d, err := xRefTable.DereferenceDict(*v)
		if err != nil {
			return err
		}
		arr = d.ArrayEntry("D")
		arr[0] = patchObject(arr[0], migrated)
		*v = d
		return nil
	}

	return n.Process(ctxSrc.XRefTable, patchValues)
}

// AddPages adds pages and corresponding resources from ctxSrc to ctxDest.
func AddPages(ctxSrc, ctxDest *model.Context, pageNrs []int, usePgCache bool) error {

	pagesIndRef, err := ctxDest.Pages()
	if err != nil {
		return err
	}

	pagesDict, err := ctxDest.DereferenceDict(*pagesIndRef)
	if err != nil {
		return err
	}

	fieldsSrc, fieldsDest := types.Array{}, types.Array{}

	if ctxSrc.Form != nil {
		o, _ := ctxSrc.Form.Find("Fields")
		fieldsSrc, err = ctxSrc.DereferenceArray(o)
		if err != nil {
			return err
		}
	}

	migrated := map[int]int{}

	if err := addPages(ctxSrc, ctxDest, pageNrs, usePgCache, *pagesIndRef, pagesDict, &fieldsSrc, &fieldsDest, migrated); err != nil {
		return err
	}

	if ctxSrc.Form != nil && len(fieldsDest) > 0 {
		d := ctxSrc.Form.Clone().(types.Dict)
		if err := migrateFormDict(d, fieldsDest, ctxSrc, ctxDest, migrated); err != nil {
			return err
		}
		ctxDest.RootDict["AcroForm"] = d
	}

	if n, ok := ctxSrc.Names["Dests"]; ok {
		// Carry over used named destinations.
		if err := migrateNamedDests(ctxSrc, n, migrated); err != nil {
			return err
		}
		ctxDest.Names = map[string]*model.Node{"Dests": n}
	}

	return nil
}
