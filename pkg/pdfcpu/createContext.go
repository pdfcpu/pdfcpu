/*
Copyright 2026 The pdfcpu Authors.

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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// CreateXRefTableWithRootDict creates an xref table containing a catalog.
func CreateXRefTableWithRootDict() (*model.XRefTable, error) {
	xRefTable := &model.XRefTable{
		Table:             map[int]*model.XRefTableEntry{},
		Names:             map[string]*model.Node{},
		NameRefs:          map[string]model.NameMap{},
		KeywordList:       types.StringSet{},
		Properties:        map[string]string{},
		LinearizationObjs: types.IntSet{},
		PageAnnots:        map[int]model.PgAnnots{},
		PageThumbs:        map[int]types.IndirectRef{},
		Signatures:        map[int]map[int]model.Signature{},
		Stats:             model.NewPDFStats(),
		ValidationMode:    model.ValidationRelaxed,
		ValidateLinks:     false,
		URIs:              map[int]map[string]string{},
		UsedGIDs:          map[string]map[uint16]bool{},
		FillFonts:         map[string]types.IndirectRef{},
	}

	xRefTable.Table[0] = model.NewFreeHeadXRefTableEntry()

	one := 1
	xRefTable.Size = &one

	v := model.V17
	xRefTable.HeaderVersion = &v

	rootDict := types.NewDict()
	rootDict.InsertName("Type", "Catalog")

	ir, err := xRefTable.IndRefForNewObject(rootDict)
	if err != nil {
		return nil, err
	}

	xRefTable.Root = ir

	return xRefTable, nil
}

func addPageTreeWithoutPage(xRefTable *model.XRefTable, rootDict types.Dict, d *types.Dim) error {
	mediaBox := types.RectForDim(d.Width, d.Height)
	pagesDict := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Pages"),
			"Count":    types.Integer(0),
			"MediaBox": mediaBox.Array(),
		},
	)
	pagesDict.Insert("Kids", types.Array{})

	pagesRootIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}

	rootDict.Insert("Pages", *pagesRootIndRef)

	return nil
}

// CreateContext creates a context for the given xref table and configuration.
func CreateContext(xRefTable *model.XRefTable, conf *model.Configuration) *model.Context {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	xRefTable.Conf = conf
	xRefTable.ValidationMode = conf.ValidationMode
	return &model.Context{
		Configuration: conf,
		XRefTable:     xRefTable,
		Write:         model.NewWriteContext(conf.Eol),
	}
}

// CreateContextWithXRefTable creates a context with an xref table without pages for the given configuration.
func CreateContextWithXRefTable(conf *model.Configuration, pageDim *types.Dim) (*model.Context, error) {
	xRefTable, err := CreateXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	if err = addPageTreeWithoutPage(xRefTable, rootDict, pageDim); err != nil {
		return nil, err
	}

	return CreateContext(xRefTable, conf), nil
}
