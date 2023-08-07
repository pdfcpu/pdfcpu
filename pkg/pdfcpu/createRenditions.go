/*
Copyright 2018 The pdfcpu Authors.

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

// Functions needed to create a test.pdf that gets used for validation testing (see process_test.go)

func createMHBEDict() *types.Dict {

	softwareIdentDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("SoftwareIdentifier"),
			"U":    types.StringLiteral("vnd.adobe.swname:ADBE_Acrobat"),
			"L":    types.NewIntegerArray(0),
			"H":    types.NewIntegerArray(),
			"OS":   types.NewStringLiteralArray(),
		},
	)

	mediaCriteriaDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("MediaCriteria"),
			"A":    types.Boolean(false),
			"C":    types.Boolean(false),
			"O":    types.Boolean(false),
			"S":    types.Boolean(false),
			"R":    types.Integer(0),
			"D": types.Dict(
				map[string]types.Object{
					"Type": types.Name("MinBitDepth"),
					"V":    types.Integer(0),
					"M":    types.Integer(0),
				},
			),
			"V": types.Array{softwareIdentDict},
			"Z": types.Dict(
				map[string]types.Object{
					"Type": types.Name("MinScreenSize"),
					"V":    types.NewIntegerArray(640, 480),
					"M":    types.Integer(0),
				},
			),
			"P": types.NewNameArray("1.3"),
			"L": types.NewStringLiteralArray("en-US"),
		},
	)

	mhbe := types.NewDict()
	mhbe.Insert("C", mediaCriteriaDict)

	return &mhbe
}

func createMediaPlayersDict() *types.Dict {

	softwareIdentDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("SoftwareIdentifier"),
			"U":    types.StringLiteral("vnd.adobe.swname:ADBE_Acrobat"),
			"L":    types.NewIntegerArray(0),
			"H":    types.NewIntegerArray(),
			"OS":   types.NewStringLiteralArray(),
		},
	)

	mediaPlayerInfoDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("MediaPlayerInfo"),
			"PID":  softwareIdentDict,
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"Type": types.Name("MediaPlayers"),
			"MU":   types.Array{mediaPlayerInfoDict},
		},
	)

	return &d
}

func createMediaOffsetDict() *types.Dict {

	timeSpanDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("Timespan"),
			"S":    types.Name("S"),
			"V":    types.Integer(1),
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"Type": types.Name("MediaOffset"),
			"S":    types.Name("T"),
			"T":    timeSpanDict,
		},
	)

	return &d
}

func createSectionMHBEDict() *types.Dict {

	d := createMediaOffsetDict()

	d1 := types.Dict(
		map[string]types.Object{
			"B": *d,
			"E": *d,
		},
	)

	return &d1
}

func createMediaClipDataDict(xRefTable *model.XRefTable) (*types.IndirectRef, error) {

	// not supported: mp3,mp4,m4a

	fileSpecDict, err := createFileSpecDict(xRefTable, testAudioFileWAV)
	if err != nil {
		return nil, err
	}

	mediaPermissionsDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("MediaPermissions"),
			"TF":   types.StringLiteral("TEMPNEVER"), //TEMPALWAYS
		},
	)

	mediaPlayersDict := createMediaPlayersDict()

	mhbe := types.Dict(map[string]types.Object{"BU": nil})

	d := types.Dict(
		map[string]types.Object{
			"Type": types.Name("MediaClip"),
			"S":    types.Name("MCD"), // media clip data
			"N":    types.StringLiteral("Sample Audio"),
			"D":    fileSpecDict,
			"CT":   types.StringLiteral("audio/x-wav"),
			//"CT": StringLiteral("audio/mp4"),
			//"CT":   StringLiteral("video/mp4"),
			"P":   mediaPermissionsDict,
			"Alt": types.NewStringLiteralArray("en-US", "My vacation", "de", "Mein Urlaub", "", "My vacation"),
			"PL":  *mediaPlayersDict,
			"MH":  mhbe,
			"BE":  mhbe,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createMediaPlayParamsMHBE() *types.Dict {

	timeSpanDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("Timespan"),
			"S":    types.Name("S"),
			"V":    types.Float(10.0),
		},
	)

	mediaDurationDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("MediaDuration"),
			"S":    types.Name("T"),
			"T":    timeSpanDict,
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"V":  types.Integer(100),
			"C":  types.Boolean(false),
			"F":  types.Integer(5),
			"D":  mediaDurationDict,
			"A":  types.Boolean(true),
			"RC": types.Float(1.0),
		},
	)

	return &d
}

func createMediaPlayParamsDict() *types.Dict {

	d := createMediaPlayersDict()
	mhbe := createMediaPlayParamsMHBE()

	d1 := types.Dict(
		map[string]types.Object{
			"Type": types.Name("MediaPlayParams"),
			"PL":   *d,
			"MH":   *mhbe,
			"BE":   *mhbe,
		},
	)

	return &d1
}

func createFloatingWindowsParamsDict() *types.Dict {

	d := types.Dict(
		map[string]types.Object{
			"Type": types.Name("FWParams"),
			"D":    types.NewIntegerArray(200, 200),
			"RT":   types.Integer(0),
			"P":    types.Integer(4),
			"O":    types.Integer(1),
			"T":    types.Boolean(true),
			"UC":   types.Boolean(true),
			"R":    types.Integer(0),
			"TT":   types.NewStringLiteralArray("en-US", "Special title", "de", "Spezieller Titel", "default title"),
		},
	)

	return &d
}

func createScreenParamsDict() *types.Dict {

	d := createFloatingWindowsParamsDict()

	mhbe := types.Dict(
		map[string]types.Object{
			"Type": types.Name("MediaScreenParams"),
			"W":    types.Integer(0),
			"B":    types.NewNumberArray(1.0, 0.0, 0.0),
			"O":    types.Float(1.0),
			"M":    types.Integer(0),
			"F":    *d,
		},
	)

	d1 := types.Dict(
		map[string]types.Object{
			"Type": types.Name("MediaScreenParams"),
			"MH":   mhbe,
			"BE":   mhbe,
		},
	)

	return &d1
}

func createMediaRendition(xRefTable *model.XRefTable, mediaClipDataDict *types.IndirectRef) *types.Dict {

	mhbe := createMHBEDict()

	d1 := createMediaPlayParamsDict()
	d2 := createScreenParamsDict()

	d3 := types.Dict(
		map[string]types.Object{
			"Type": types.Name("Rendition"),
			"S":    types.Name("MR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"C":    *mediaClipDataDict,
			"P":    *d1,
			"SP":   *d2,
		},
	)

	return &d3
}

func createSectionMediaRendition(mediaClipDataDict *types.IndirectRef) *types.Dict {

	mhbe := createSectionMHBEDict()

	mediaClipSectionDict := types.Dict(
		map[string]types.Object{
			"Type": types.Name("MediaClip"),
			"S":    types.Name("MCS"), // media clip section
			"N":    types.StringLiteral("Sample movie"),
			"D":    *mediaClipDataDict,
			"Alt":  types.NewStringLiteralArray("en-US", "My vacation", "de", "Mein Urlaub", "", "default vacation"),
			"MH":   *mhbe,
			"BE":   *mhbe,
		},
	)

	mhbe = createMHBEDict()

	d := types.Dict(
		map[string]types.Object{
			"Type": types.Name("Rendition"),
			"S":    types.Name("MR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"C":    mediaClipSectionDict,
		},
	)

	return &d
}

func createSelectorRendition(mediaClipDataDict *types.IndirectRef) *types.Dict {

	mhbe := createMHBEDict()

	r := createSectionMediaRendition(mediaClipDataDict)

	d := types.Dict(
		map[string]types.Object{
			"Type": types.Name("Rendition"),
			"S":    types.Name("SR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"R":    types.Array{*r},
		},
	)

	return &d
}
