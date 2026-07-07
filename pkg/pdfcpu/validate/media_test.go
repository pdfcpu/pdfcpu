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

package validate

import (
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestValidateMediaCriteriaDictEntryVReportsIndexContext(t *testing.T) {
	err := validateMediaCriteriaDictEntryV(testXRef(t, model.ValidationStrict), types.Dict{
		"V": types.Array{types.Integer(1)},
	}, "mediaCritDict", REQUIRED, model.V10)
	requireErrContains(t, err, "mediaCritDict.V[0]")
}

func TestValidateMediaPlayersDictReportsIndexContext(t *testing.T) {
	err := validateMediaPlayersDict(testXRef(t, model.ValidationStrict), types.Dict{
		"MU": types.Array{types.Integer(1)},
	}, model.V10)
	requireErrContains(t, err, "mediaPlayersDict.MU[0]")
}

func TestValidateSelectorRenditionDictReportsIndexContext(t *testing.T) {
	err := validateSelectorRenditionDict(testXRef(t, model.ValidationStrict), types.Dict{
		"R": types.Array{types.Integer(1)},
	}, model.V10)
	requireErrContains(t, err, "selectorRendDict.R[0]")
}

func TestValidateSelectorRenditionDictReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"S": types.Name("MR"),
		"C": types.Dict{"S": types.Name("Bogus")},
	})

	err := validateSelectorRenditionDict(xRefTable, types.Dict{
		"R": types.Array{*types.NewIndirectRef(2, 0)},
	}, model.V10)
	requireErrChainContains(t, err, "selectorRendDict.R[0] obj#2", "renditionDict media rendition", "mediaRendDict.C")
}

func TestValidateRenditionDictReportsSelectorContext(t *testing.T) {
	err := validateRenditionDict(testXRef(t, model.ValidationStrict), types.Dict{
		"S": types.Name("SR"),
		"R": types.Array{types.Integer(1)},
	}, model.V10)
	requireErrContains(t, err, "renditionDict selector rendition")
}

func TestValidateRenditionDictSkipsMissingOptionalBECriteria(t *testing.T) {
	if err := validateRenditionDict(testXRef(t, model.ValidationStrict), types.Dict{
		"S":  types.Name("MR"),
		"BE": types.Dict{},
	}, model.V10); err != nil {
		t.Fatal(err)
	}
}

func TestValidateMediaClipDataDictReportsFileSpecObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Type": types.Name("Filespec"),
	})

	err := validateMediaClipDataDict(xRefTable, types.Dict{
		"D": *types.NewIndirectRef(2, 0),
	}, model.V10)
	requireErrChainContains(t, err, "mediaClipDataDict.D obj#2", "file specification obj#2 dict", "required entry=F")
}

func TestValidateRenditionActionReportsMediaFileSpecContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"S": types.Name("MR"),
		"C": *types.NewIndirectRef(3, 0),
	})
	xRefTable.Table[3] = model.NewXRefTableEntryGen0(types.Dict{
		"S": types.Name("MCD"),
		"D": *types.NewIndirectRef(4, 0),
	})
	xRefTable.Table[4] = model.NewXRefTableEntryGen0(types.Dict{
		"Type": types.Name("Filespec"),
	})

	err := validateActionDict(xRefTable, types.Dict{
		"S":  types.Name("Rendition"),
		"OP": types.Integer(0),
		"R":  *types.NewIndirectRef(2, 0),
	})
	requireErrChainContains(
		t,
		err,
		"action Rendition",
		"Rendition.R obj#2",
		"renditionDict media rendition",
		"mediaRendDict.C obj#3",
		"mediaClipDict data",
		"mediaClipDataDict.D obj#4",
		"required entry=F",
	)
}
