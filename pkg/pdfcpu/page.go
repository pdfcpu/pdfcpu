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
	"errors"
	"fmt"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var pParamMap = parameterMap[PageConfiguration]{
	"dimensions": parseDimensions,
	"formsize":   parsePageFormat,
	"papersize":  parsePageFormat,
}

// PageConfiguration represents the page config for the "pages insert" command.
type PageConfiguration struct {
	PageDim  *types.Dim        // page dimensions in display unit; nil inherits each selected page's effective MediaBox.
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

// String returns the string value of p.
func (p PageConfiguration) String() string {
	return fmt.Sprintf("Page config: %s %s\n", p.PageSize, p.PageDim)
}

func parsePageFormat(s string, p *PageConfiguration) (err error) {
	if p.UserDim {
		return errAmbiguousPageDim
	}
	p.PageDim, p.PageSize, err = types.ParsePageFormat(s)
	p.UserDim = true
	return err
}

func parseDimensions(s string, p *PageConfiguration) (err error) {
	if p.UserDim {
		return errAmbiguousPageDim
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

	for s := range strings.SplitSeq(s, ",") {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, errors.New("invalid page configuration string")
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := handleParameter(pParamMap, paramPrefix, paramValueStr, pageConf); err != nil {
			return nil, err
		}
	}

	return pageConf, nil
}

func validateAddPagesInputs(
	ctxSrc, ctxDest *model.Context,
	pagesDict types.Dict,
	fieldsSrc, fieldsDest *types.Array,
	migrated map[int]int) error {
	if err := requireContextWithXRefTable(ctxSrc); err != nil {
		return fmt.Errorf("add pages: source context: %w", err)
	}
	if err := requireContextWithXRefTable(ctxDest); err != nil {
		return fmt.Errorf("add pages: destination context: %w", err)
	}
	if pagesDict == nil {
		return errors.New("add pages: missing destination page tree dict")
	}
	if fieldsSrc == nil {
		return errors.New("add pages: missing source form fields")
	}
	if fieldsDest == nil {
		return errors.New("add pages: missing destination form fields")
	}
	if migrated == nil {
		return errors.New("add pages: missing migration map")
	}
	return nil
}

func migratedPageDict(ctxSrc, ctxDest *model.Context, pageNr int, migrated map[int]int) (types.Dict, *types.IndirectRef, *model.InheritedPageAttrs, error) {
	d, pageIndRef, inhPAttrs, err := ctxSrc.PageDict(pageNr, true)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read page dict: %w", err)
	}
	if d == nil {
		return nil, nil, nil, fmt.Errorf("unknown page number: %d", pageNr)
	}

	obj, err := migrateIndRef(pageIndRef, ctxSrc, ctxDest, migrated)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("migrate page object: %w", err)
	}

	pageDict, ok := obj.(types.Dict)
	if !ok {
		return nil, nil, nil, fmt.Errorf("migrated page object is %T, want types.Dict", obj)
	}
	return pageDict, pageIndRef, inhPAttrs, nil
}

func addPage(
	ctxSrc, ctxDest *model.Context,
	pageNr int,
	pagesIndRef types.IndirectRef,
	pagesDict types.Dict,
	fieldsSrc, fieldsDest *types.Array,
	migrated map[int]int) (*types.IndirectRef, error) {
	d, pageIndRef, inhPAttrs, err := migratedPageDict(ctxSrc, ctxDest, pageNr, migrated)
	if err != nil {
		return nil, fmt.Errorf("page %d: %w", pageNr, err)
	}

	d["Resources"] = inhPAttrs.Resources.Clone()
	d["Parent"] = pagesIndRef
	d["MediaBox"] = inhPAttrs.MediaBox.Array()
	if inhPAttrs.Rotate%360 > 0 {
		d["Rotate"] = types.Integer(inhPAttrs.Rotate)
	}

	if err := migratePageDict(d, *pageIndRef, ctxSrc, ctxDest, migrated); err != nil {
		return nil, fmt.Errorf("page %d: migrate page dict: %w", pageNr, err)
	}

	if d["Annots"] != nil && len(*fieldsSrc) > 0 {
		if err := migrateFields(d, fieldsSrc, fieldsDest, ctxSrc, ctxDest, migrated); err != nil {
			return nil, fmt.Errorf("page %d: migrate fields: %w", pageNr, err)
		}
	}

	if err := model.AppendPageTree(pageIndRef, 1, pagesDict); err != nil {
		return nil, fmt.Errorf("page %d: append page tree: %w", pageNr, err)
	}
	return pageIndRef, nil
}

func addPages(
	ctxSrc, ctxDest *model.Context,
	pageNrs []int,
	usePgCache bool,
	pagesIndRef types.IndirectRef,
	pagesDict types.Dict,
	fieldsSrc, fieldsDest *types.Array,
	migrated map[int]int) error {
	if err := validateAddPagesInputs(ctxSrc, ctxDest, pagesDict, fieldsSrc, fieldsDest, migrated); err != nil {
		return err
	}
	// Used by collect, extractPages, split
	pageCache := map[int]*types.IndirectRef{}

	for _, i := range pageNrs {
		if usePgCache {
			if indRef, ok := pageCache[i]; ok {
				if err := model.AppendPageTree(indRef, 1, pagesDict); err != nil {
					return fmt.Errorf("page %d: append cached page tree: %w", i, err)
				}
				continue
			}
		}

		pageIndRef, err := addPage(ctxSrc, ctxDest, i, pagesIndRef, pagesDict, fieldsSrc, fieldsDest, migrated)
		if err != nil {
			return err
		}

		if usePgCache {
			pageCache[i] = pageIndRef
		}
	}

	return nil
}

func destinationPageMigrated(arr types.Array, migrated map[int]int) bool {
	indRef, ok := arr[0].(types.IndirectRef)
	return !ok || migrated[indRef.ObjectNumber.Value()] > 0
}

func migrateNamedDestArray(arr types.Array, migrated map[int]int) bool {
	if len(arr) == 0 || !destinationPageMigrated(arr, migrated) {
		return false
	}
	arr[0] = patchObject(arr[0], migrated)
	return true
}

func migrateNamedDestValue(xRefTable *model.XRefTable, v *types.Object, migrated map[int]int) (bool, error) {
	// destination array
	arr, err := xRefTable.DereferenceArray(*v)
	if err == nil {
		if !migrateNamedDestArray(arr, migrated) {
			return false, nil
		}
		*v = arr
		return true, nil
	}

	// destination dict with a D array.
	d, err := xRefTable.DereferenceDict(*v)
	if err != nil {
		return false, err
	}

	arr = d.ArrayEntry("D")
	if !migrateNamedDestArray(arr, migrated) {
		return false, nil
	}
	*v = d
	return true, nil
}

func migrateNamedDests(ctxSrc *model.Context, n *model.Node, migrated map[int]int) error {
	if err := requireContextWithXRefTable(ctxSrc); err != nil {
		return fmt.Errorf("source context: %w", err)
	}
	if n == nil {
		return errors.New("missing named destinations")
	}
	if migrated == nil {
		return errors.New("missing migration map")
	}

	var remove []string

	patchValues := func(xRefTable *model.XRefTable, k string, v *types.Object) error {
		if v == nil {
			return fmt.Errorf("named destination %q: missing value", k)
		}
		if *v == nil {
			// Skip corrupt node.
			return nil
		}
		keep, err := migrateNamedDestValue(xRefTable, v, migrated)
		if err != nil {
			return fmt.Errorf("named destination %q: %w", k, err)
		}
		if !keep {
			remove = append(remove, k)
		}
		return nil
	}

	if err := n.Process(ctxSrc.XRefTable, patchValues); err != nil {
		return fmt.Errorf("process named destinations: %w", err)
	}

	for _, k := range remove {
		if _, _, err := n.Remove(ctxSrc.XRefTable, k); err != nil {
			return fmt.Errorf("remove named destination %q: %w", k, err)
		}
	}

	return nil
}

// AddPages adds pages and corresponding resources from ctxSrc to ctxDest.
func AddPages(ctxSrc, ctxDest *model.Context, pageNrs []int, usePgCache bool) error {
	if err := requireContextWithXRefTable(ctxSrc); err != nil {
		return fmt.Errorf("add pages: source context: %w", err)
	}
	if err := requireContextWithXRefTable(ctxDest); err != nil {
		return fmt.Errorf("add pages: destination context: %w", err)
	}

	pagesIndRef, err := ctxDest.Pages()
	if err != nil {
		return fmt.Errorf("add pages: read destination page tree: %w", err)
	}
	if pagesIndRef == nil {
		return errors.New("add pages: missing destination page tree")
	}

	pagesDict, err := ctxDest.DereferenceDict(*pagesIndRef)
	if err != nil {
		return fmt.Errorf("add pages: dereference destination page tree: %w", err)
	}

	fieldsSrc, fieldsDest := types.Array{}, types.Array{}

	if ctxSrc.Form != nil {
		o, _ := ctxSrc.Form.Find("Fields")
		fieldsSrc, err = ctxSrc.DereferenceArray(o)
		if err != nil {
			return fmt.Errorf("add pages: read source form fields: %w", err)
		}
	}

	migrated := map[int]int{}

	if err := addPages(ctxSrc, ctxDest, pageNrs, usePgCache, *pagesIndRef, pagesDict, &fieldsSrc, &fieldsDest, migrated); err != nil {
		return fmt.Errorf("add pages: %w", err)
	}

	if ctxSrc.Form != nil && len(fieldsDest) > 0 {
		d := ctxSrc.Form.Clone().(types.Dict)
		if err := migrateFormDict(d, fieldsDest, ctxSrc, ctxDest, migrated); err != nil {
			return fmt.Errorf("add pages: migrate form: %w", err)
		}
		ctxDest.RootDict["AcroForm"] = d
	}

	if n, ok := ctxSrc.Names["Dests"]; ok {
		// Carry over used named destinations.
		if err := migrateNamedDests(ctxSrc, n, migrated); err != nil {
			return fmt.Errorf("add pages: migrate named destinations: %w", err)
		}
		ctxDest.Names = map[string]*model.Node{"Dests": n}
	}

	return nil
}
