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

package api

import (
	"context"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func createFontXRefTable() (*model.XRefTable, error) {
	return pdfcpu.CreateXRefTableWithRootDict()
}

func addFontPageTree(c context.Context, xRefTable *model.XRefTable, rootDict types.Dict, p model.Page) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	pagesDict := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Pages"),
			"Count":    types.Integer(1),
			"MediaBox": p.MediaBox.Array(),
		},
	)

	parentPageIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}

	pageIndRef, err := createFontPageObject(c, xRefTable, *parentPageIndRef, p)
	if err != nil {
		return err
	}

	pagesDict.Insert("Kids", types.Array{*pageIndRef})
	rootDict.Insert("Pages", *parentPageIndRef)

	return nil
}

func createFontContentStream(xRefTable *model.XRefTable, b []byte) (*types.IndirectRef, error) {
	sd, _ := xRefTable.NewStreamDictForBuf(b)
	if err := sd.Encode(); err != nil {
		return nil, err
	}
	return xRefTable.IndRefForNewObject(*sd)
}

func createFontPageObject(c context.Context, xRefTable *model.XRefTable, parentPageIndRef types.IndirectRef, p model.Page) (*types.IndirectRef, error) {
	pageDict := types.Dict(
		map[string]types.Object{
			"Type":   types.Name("Page"),
			"Parent": parentPageIndRef,
		},
	)

	fontRes, err := pdffont.FontResources(c, xRefTable, p.Fm)
	if err != nil {
		return nil, err
	}

	if len(fontRes) > 0 {
		resDict := types.Dict(
			map[string]types.Object{
				"Font": fontRes,
			},
		)
		pageDict.Insert("Resources", resDict)
	}

	ir, err := createFontContentStream(xRefTable, p.Buf.Bytes())
	if err != nil {
		return nil, err
	}
	pageDict.Insert("Contents", *ir)

	return xRefTable.IndRefForNewObject(pageDict)
}
