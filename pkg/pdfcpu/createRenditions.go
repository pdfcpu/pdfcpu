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

func createMHBEDict() *PDFDict {

	softwareIdentDict := PDFDict{
		Dict: map[string]Object{
			"Type": Name("SoftwareIdentifier"),
			"U":    StringLiteral("vnd.adobe.swname:ADBE_Acrobat"),
			"L":    NewIntegerArray(0),
			"H":    NewIntegerArray(),
			"OS":   NewStringArray(),
		},
	}

	mediaCriteriaDict := PDFDict{
		Dict: map[string]Object{
			"Type": Name("MediaCriteria"),
			"A":    Boolean(false),
			"C":    Boolean(false),
			"O":    Boolean(false),
			"S":    Boolean(false),
			"R":    Integer(0),
			"D": PDFDict{
				Dict: map[string]Object{
					"Type": Name("MinBitDepth"),
					"V":    Integer(0),
					"M":    Integer(0),
				},
			},
			"V": Array{softwareIdentDict},
			"Z": PDFDict{
				Dict: map[string]Object{
					"Type": Name("MinScreenSize"),
					"V":    NewIntegerArray(640, 480),
					"M":    Integer(0),
				},
			},
			"P": NewNameArray("1.3"),
			"L": NewStringArray("en-US"),
		},
	}

	mhbe := NewPDFDict()
	mhbe.Insert("C", mediaCriteriaDict)

	return &mhbe
}

func createMediaPlayersDict() *PDFDict {

	softwareIdentDict := PDFDict{
		Dict: map[string]Object{
			"Type": Name("SoftwareIdentifier"),
			"U":    StringLiteral("vnd.adobe.swname:ADBE_Acrobat"),
			"L":    NewIntegerArray(0),
			"H":    NewIntegerArray(),
			"OS":   NewStringArray(),
		},
	}

	mediaPlayerInfoDict := PDFDict{
		Dict: map[string]Object{
			"Type": Name("MediaPlayerInfo"),
			"PID":  softwareIdentDict,
		},
	}

	return &PDFDict{
		Dict: map[string]Object{
			"Type": Name("MediaPlayers"),
			"MU":   Array{mediaPlayerInfoDict},
		},
	}

}

func createMediaOffsetDict() *PDFDict {

	timeSpanDict := PDFDict{
		Dict: map[string]Object{
			"Type": Name("Timespan"),
			"S":    Name("S"),
			"V":    Integer(1),
		},
	}

	return &PDFDict{
		Dict: map[string]Object{
			"Type": Name("MediaOffset"),
			"S":    Name("T"),
			"T":    timeSpanDict,
		},
	}

}

func createSectionMHBEDict() *PDFDict {

	d := createMediaOffsetDict()

	return &PDFDict{
		Dict: map[string]Object{
			"B": *d,
			"E": *d,
		},
	}
}

func createMediaClipDataDict(xRefTable *XRefTable) (*IndirectRef, error) {

	// not supported: mp3,mp4,m4a

	fileSpecDict, err := createFileSpecDict(xRefTable, testAudioFileWAV)
	if err != nil {
		return nil, err
	}

	mediaPermissionsDict := PDFDict{
		Dict: map[string]Object{
			"Type": Name("MediaPermissions"),
			"TF":   StringLiteral("TEMPNEVER"), //TEMPALWAYS
		},
	}

	mediaPlayersDict := createMediaPlayersDict()

	mhbe := PDFDict{Dict: map[string]Object{"BU": nil}}

	d := PDFDict{
		Dict: map[string]Object{
			"Type": Name("MediaClip"),
			"S":    Name("MCD"), // media clip data
			"N":    StringLiteral("Sample Audio"),
			"D":    *fileSpecDict,
			"CT":   StringLiteral("audio/x-wav"),
			//"CT": StringLiteral("audio/mp4"),
			//"CT":   StringLiteral("video/mp4"),
			"P":   mediaPermissionsDict,
			"Alt": NewStringArray("en-US", "My vacation", "de", "Mein Urlaub", "", "My vacation"),
			"PL":  *mediaPlayersDict,
			"MH":  mhbe,
			"BE":  mhbe,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createMediaPlayParamsMHBE() *PDFDict {

	timeSpanDict := PDFDict{
		Dict: map[string]Object{
			"Type": Name("Timespan"),
			"S":    Name("S"),
			"V":    Float(10.0),
		},
	}

	mediaDurationDict := PDFDict{
		Dict: map[string]Object{
			"Type": Name("MediaDuration"),
			"S":    Name("T"),
			"T":    timeSpanDict,
		},
	}

	return &PDFDict{
		Dict: map[string]Object{
			"V":  Integer(100),
			"C":  Boolean(false),
			"F":  Integer(5),
			"D":  mediaDurationDict,
			"A":  Boolean(true),
			"RC": Float(1.0),
		},
	}

}

func createMediaPlayParamsDict() *PDFDict {

	d := createMediaPlayersDict()
	mhbe := createMediaPlayParamsMHBE()

	return &PDFDict{
		Dict: map[string]Object{
			"Type": Name("MediaPlayParams"),
			"PL":   *d,
			"MH":   *mhbe,
			"BE":   *mhbe,
		},
	}

}

func createFloatingWindowsParamsDict() *PDFDict {

	return &PDFDict{
		Dict: map[string]Object{
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
	}
}

func createScreenParamsDict() *PDFDict {

	d := createFloatingWindowsParamsDict()

	mhbe := PDFDict{
		Dict: map[string]Object{
			"Type": Name("MediaScreenParams"),
			"W":    Integer(0),
			"B":    NewNumberArray(1.0, 0.0, 0.0),
			"O":    Float(1.0),
			"M":    Integer(0),
			"F":    *d,
		},
	}

	return &PDFDict{
		Dict: map[string]Object{
			"Type": Name("MediaScreenParams"),
			"MH":   mhbe,
			"BE":   mhbe,
		},
	}
}

func createMediaRendition(xRefTable *XRefTable, mediaClipDataDict *IndirectRef) *PDFDict {

	mhbe := createMHBEDict()

	d1 := createMediaPlayParamsDict()
	d2 := createScreenParamsDict()

	return &PDFDict{
		Dict: map[string]Object{
			"Type": Name("Rendition"),
			"S":    Name("MR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"C":    *mediaClipDataDict,
			"P":    *d1,
			"SP":   *d2,
		},
	}

}

func createSectionMediaRendition(mediaClipDataDict *IndirectRef) *PDFDict {

	mhbe := createSectionMHBEDict()

	mediaClipSectionDict := PDFDict{
		Dict: map[string]Object{
			"Type": Name("MediaClip"),
			"S":    Name("MCS"), // media clip section
			"N":    StringLiteral("Sample movie"),
			"D":    *mediaClipDataDict,
			"Alt":  NewStringArray("en-US", "My vacation", "de", "Mein Urlaub", "", "default vacation"),
			"MH":   *mhbe,
			"BE":   *mhbe,
		},
	}

	mhbe = createMHBEDict()

	return &PDFDict{
		Dict: map[string]Object{
			"Type": Name("Rendition"),
			"S":    Name("MR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"C":    mediaClipSectionDict,
		},
	}

}

func createSelectorRendition(mediaClipDataDict *IndirectRef) *PDFDict {

	mhbe := createMHBEDict()

	r := createSectionMediaRendition(mediaClipDataDict)

	return &PDFDict{
		Dict: map[string]Object{
			"Type": Name("Rendition"),
			"S":    Name("SR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"R":    Array{*r},
		},
	}

}
