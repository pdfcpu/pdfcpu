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

// Functions needed to create a test.pdf that gets used for validation testing (see process_test.go)

func createMHBEDict() *Dict {

	softwareIdentDict := Dict(
		map[string]Object{
			"Type": Name("SoftwareIdentifier"),
			"U":    StringLiteral("vnd.adobe.swname:ADBE_Acrobat"),
			"L":    NewIntegerArray(0),
			"H":    NewIntegerArray(),
			"OS":   NewStringArray(),
		},
	)

	mediaCriteriaDict := Dict(
		map[string]Object{
			"Type": Name("MediaCriteria"),
			"A":    Boolean(false),
			"C":    Boolean(false),
			"O":    Boolean(false),
			"S":    Boolean(false),
			"R":    Integer(0),
			"D": Dict(
				map[string]Object{
					"Type": Name("MinBitDepth"),
					"V":    Integer(0),
					"M":    Integer(0),
				},
			),
			"V": Array{softwareIdentDict},
			"Z": Dict(
				map[string]Object{
					"Type": Name("MinScreenSize"),
					"V":    NewIntegerArray(640, 480),
					"M":    Integer(0),
				},
			),
			"P": NewNameArray("1.3"),
			"L": NewStringArray("en-US"),
		},
	)

	mhbe := NewDict()
	mhbe.Insert("C", mediaCriteriaDict)

	return &mhbe
}

func createMediaPlayersDict() *Dict {

	softwareIdentDict := Dict(
		map[string]Object{
			"Type": Name("SoftwareIdentifier"),
			"U":    StringLiteral("vnd.adobe.swname:ADBE_Acrobat"),
			"L":    NewIntegerArray(0),
			"H":    NewIntegerArray(),
			"OS":   NewStringArray(),
		},
	)

	mediaPlayerInfoDict := Dict(
		map[string]Object{
			"Type": Name("MediaPlayerInfo"),
			"PID":  softwareIdentDict,
		},
	)

	d := Dict(
		map[string]Object{
			"Type": Name("MediaPlayers"),
			"MU":   Array{mediaPlayerInfoDict},
		},
	)

	return &d
}

func createMediaOffsetDict() *Dict {

	timeSpanDict := Dict(
		map[string]Object{
			"Type": Name("Timespan"),
			"S":    Name("S"),
			"V":    Integer(1),
		},
	)

	d := Dict(
		map[string]Object{
			"Type": Name("MediaOffset"),
			"S":    Name("T"),
			"T":    timeSpanDict,
		},
	)

	return &d
}

func createSectionMHBEDict() *Dict {

	d := createMediaOffsetDict()

	d1 := Dict(
		map[string]Object{
			"B": *d,
			"E": *d,
		},
	)

	return &d1
}

func createMediaClipDataDict(xRefTable *XRefTable) (*IndirectRef, error) {

	// not supported: mp3,mp4,m4a

	fileSpecDict, err := createFileSpecDict(xRefTable, testAudioFileWAV)
	if err != nil {
		return nil, err
	}

	mediaPermissionsDict := Dict(
		map[string]Object{
			"Type": Name("MediaPermissions"),
			"TF":   StringLiteral("TEMPNEVER"), //TEMPALWAYS
		},
	)

	mediaPlayersDict := createMediaPlayersDict()

	mhbe := Dict(map[string]Object{"BU": nil})

	d := Dict(
		map[string]Object{
			"Type": Name("MediaClip"),
			"S":    Name("MCD"), // media clip data
			"N":    StringLiteral("Sample Audio"),
			"D":    fileSpecDict,
			"CT":   StringLiteral("audio/x-wav"),
			//"CT": StringLiteral("audio/mp4"),
			//"CT":   StringLiteral("video/mp4"),
			"P":   mediaPermissionsDict,
			"Alt": NewStringArray("en-US", "My vacation", "de", "Mein Urlaub", "", "My vacation"),
			"PL":  *mediaPlayersDict,
			"MH":  mhbe,
			"BE":  mhbe,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createMediaPlayParamsMHBE() *Dict {

	timeSpanDict := Dict(
		map[string]Object{
			"Type": Name("Timespan"),
			"S":    Name("S"),
			"V":    Float(10.0),
		},
	)

	mediaDurationDict := Dict(
		map[string]Object{
			"Type": Name("MediaDuration"),
			"S":    Name("T"),
			"T":    timeSpanDict,
		},
	)

	d := Dict(
		map[string]Object{
			"V":  Integer(100),
			"C":  Boolean(false),
			"F":  Integer(5),
			"D":  mediaDurationDict,
			"A":  Boolean(true),
			"RC": Float(1.0),
		},
	)

	return &d
}

func createMediaPlayParamsDict() *Dict {

	d := createMediaPlayersDict()
	mhbe := createMediaPlayParamsMHBE()

	d1 := Dict(
		map[string]Object{
			"Type": Name("MediaPlayParams"),
			"PL":   *d,
			"MH":   *mhbe,
			"BE":   *mhbe,
		},
	)

	return &d1
}

func createFloatingWindowsParamsDict() *Dict {

	d := Dict(
		map[string]Object{
			"Type": Name("FWParams"),
			"D":    NewIntegerArray(200, 200),
			"RT":   Integer(0),
			"P":    Integer(4),
			"O":    Integer(1),
			"T":    Boolean(true),
			"UC":   Boolean(true),
			"R":    Integer(0),
			"TT":   NewStringArray("en-US", "Special title", "de", "Spezieller Titel", "default title"),
		},
	)

	return &d
}

func createScreenParamsDict() *Dict {

	d := createFloatingWindowsParamsDict()

	mhbe := Dict(
		map[string]Object{
			"Type": Name("MediaScreenParams"),
			"W":    Integer(0),
			"B":    NewNumberArray(1.0, 0.0, 0.0),
			"O":    Float(1.0),
			"M":    Integer(0),
			"F":    *d,
		},
	)

	d1 := Dict(
		map[string]Object{
			"Type": Name("MediaScreenParams"),
			"MH":   mhbe,
			"BE":   mhbe,
		},
	)

	return &d1
}

func createMediaRendition(xRefTable *XRefTable, mediaClipDataDict *IndirectRef) *Dict {

	mhbe := createMHBEDict()

	d1 := createMediaPlayParamsDict()
	d2 := createScreenParamsDict()

	d3 := Dict(
		map[string]Object{
			"Type": Name("Rendition"),
			"S":    Name("MR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"C":    *mediaClipDataDict,
			"P":    *d1,
			"SP":   *d2,
		},
	)

	return &d3
}

func createSectionMediaRendition(mediaClipDataDict *IndirectRef) *Dict {

	mhbe := createSectionMHBEDict()

	mediaClipSectionDict := Dict(
		map[string]Object{
			"Type": Name("MediaClip"),
			"S":    Name("MCS"), // media clip section
			"N":    StringLiteral("Sample movie"),
			"D":    *mediaClipDataDict,
			"Alt":  NewStringArray("en-US", "My vacation", "de", "Mein Urlaub", "", "default vacation"),
			"MH":   *mhbe,
			"BE":   *mhbe,
		},
	)

	mhbe = createMHBEDict()

	d := Dict(
		map[string]Object{
			"Type": Name("Rendition"),
			"S":    Name("MR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"C":    mediaClipSectionDict,
		},
	)

	return &d
}

func createSelectorRendition(mediaClipDataDict *IndirectRef) *Dict {

	mhbe := createMHBEDict()

	r := createSectionMediaRendition(mediaClipDataDict)

	d := Dict(
		map[string]Object{
			"Type": Name("Rendition"),
			"S":    Name("SR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"R":    Array{*r},
		},
	)

	return &d
}
