package create

import "github.com/hhrutter/pdfcpu/types"

func createMHBEDict() *types.PDFDict {

	softwareIdentDict := types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("SoftwareIdentifier"),
			"U":    types.PDFStringLiteral("vnd.adobe.swname:ADBE_Acrobat"),
			"L":    types.NewIntegerArray(0),
			"H":    types.NewIntegerArray(),
			"OS":   types.NewStringArray(),
		},
	}

	mediaCriteriaDict := types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("MediaCriteria"),
			"A":    types.PDFBoolean(false),
			"C":    types.PDFBoolean(false),
			"O":    types.PDFBoolean(false),
			"S":    types.PDFBoolean(false),
			"R":    types.PDFInteger(0),
			"D": types.PDFDict{
				Dict: map[string]types.PDFObject{
					"Type": types.PDFName("MinBitDepth"),
					"V":    types.PDFInteger(0),
					"M":    types.PDFInteger(0),
				},
			},
			"V": types.PDFArray{softwareIdentDict},
			"Z": types.PDFDict{
				Dict: map[string]types.PDFObject{
					"Type": types.PDFName("MinScreenSize"),
					"V":    types.NewIntegerArray(640, 480),
					"M":    types.PDFInteger(0),
				},
			},
			"P": types.NewNameArray("1.3"),
			"L": types.NewStringArray("en-US"),
		},
	}

	mhbe := types.NewPDFDict()
	mhbe.Insert("C", mediaCriteriaDict)

	return &mhbe
}

func createMediaPlayersDict() *types.PDFDict {

	softwareIdentDict := types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("SoftwareIdentifier"),
			"U":    types.PDFStringLiteral("vnd.adobe.swname:ADBE_Acrobat"),
			"L":    types.NewIntegerArray(0),
			"H":    types.NewIntegerArray(),
			"OS":   types.NewStringArray(),
		},
	}

	mediaPlayerInfoDict := types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("MediaPlayerInfo"),
			"PID":  softwareIdentDict,
		},
	}

	return &types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("MediaPlayers"),
			"MU":   types.PDFArray{mediaPlayerInfoDict},
		},
	}

}

func createMediaOffsetDict() *types.PDFDict {

	timeSpanDict := types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("Timespan"),
			"S":    types.PDFName("S"),
			"V":    types.PDFInteger(1),
		},
	}

	return &types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("MediaOffset"),
			"S":    types.PDFName("T"),
			"T":    timeSpanDict,
		},
	}

}

func createSectionMHBEDict() *types.PDFDict {

	d := createMediaOffsetDict()

	return &types.PDFDict{
		Dict: map[string]types.PDFObject{
			"B": *d,
			"E": *d,
		},
	}
}

func createMediaClipDataDict(xRefTable *types.XRefTable) (*types.PDFIndirectRef, error) {

	// not supported: mp3,mp4,m4a

	fileSpecDict, err := createFileSpecDict(xRefTable, testAudioFileWAV)
	if err != nil {
		return nil, err
	}

	mediaPermissionsDict := types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("MediaPermissions"),
			"TF":   types.PDFStringLiteral("TEMPNEVER"), //TEMPALWAYS
		},
	}

	mediaPlayersDict := createMediaPlayersDict()

	mhbe := types.PDFDict{Dict: map[string]types.PDFObject{"BU": nil}}

	d := types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("MediaClip"),
			"S":    types.PDFName("MCD"), // media clip data
			"N":    types.PDFStringLiteral("Sample Audio"),
			"D":    *fileSpecDict,
			"CT":   types.PDFStringLiteral("audio/x-wav"),
			//"CT": types.PDFStringLiteral("audio/mp4"),
			//"CT":   types.PDFStringLiteral("video/mp4"),
			"P":   mediaPermissionsDict,
			"Alt": types.NewStringArray("en-US", "My vacation", "de", "Mein Urlaub", "", "My vacation"),
			"PL":  *mediaPlayersDict,
			"MH":  mhbe,
			"BE":  mhbe,
		},
	}

	return xRefTable.IndRefForNewObject(d)
}

func createMediaPlayParamsMHBE() *types.PDFDict {

	timeSpanDict := types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("Timespan"),
			"S":    types.PDFName("S"),
			"V":    types.PDFFloat(10.0),
		},
	}

	mediaDurationDict := types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("MediaDuration"),
			"S":    types.PDFName("T"),
			"T":    timeSpanDict,
		},
	}

	return &types.PDFDict{
		Dict: map[string]types.PDFObject{
			"V":  types.PDFInteger(100),
			"C":  types.PDFBoolean(false),
			"F":  types.PDFInteger(5),
			"D":  mediaDurationDict,
			"A":  types.PDFBoolean(true),
			"RC": types.PDFFloat(1.0),
		},
	}

}

func createMediaPlayParamsDict() *types.PDFDict {

	d := createMediaPlayersDict()
	mhbe := createMediaPlayParamsMHBE()

	return &types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("MediaPlayParams"),
			"PL":   *d,
			"MH":   *mhbe,
			"BE":   *mhbe,
		},
	}

}

func createFloatingWindowsParamsDict() *types.PDFDict {

	return &types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("FWParams"),
			"D":    types.NewIntegerArray(200, 200),
			"RT":   types.PDFInteger(0),
			"P":    types.PDFInteger(4),
			"O":    types.PDFInteger(1),
			"T":    types.PDFBoolean(true),
			"UC":   types.PDFBoolean(true),
			"R":    types.PDFInteger(0),
			"TT":   types.NewStringArray("en-US", "Special title", "de", "Spezieller Titel", "default title"),
		},
	}
}

func createScreenParamsDict() *types.PDFDict {

	d := createFloatingWindowsParamsDict()

	mhbe := types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("MediaScreenParams"),
			"W":    types.PDFInteger(0),
			"B":    types.NewNumberArray(1.0, 0.0, 0.0),
			"O":    types.PDFFloat(1.0),
			"M":    types.PDFInteger(0),
			"F":    *d,
		},
	}

	return &types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("MediaScreenParams"),
			"MH":   mhbe,
			"BE":   mhbe,
		},
	}
}

func createMediaRendition(xRefTable *types.XRefTable, mediaClipDataDict *types.PDFIndirectRef) *types.PDFDict {

	mhbe := createMHBEDict()

	d1 := createMediaPlayParamsDict()
	d2 := createScreenParamsDict()

	return &types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("Rendition"),
			"S":    types.PDFName("MR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"C":    *mediaClipDataDict,
			"P":    *d1,
			"SP":   *d2,
		},
	}

}

func createSectionMediaRendition(mediaClipDataDict *types.PDFIndirectRef) *types.PDFDict {

	mhbe := createSectionMHBEDict()

	mediaClipSectionDict := types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("MediaClip"),
			"S":    types.PDFName("MCS"), // media clip section
			"N":    types.PDFStringLiteral("Sample movie"),
			"D":    *mediaClipDataDict,
			"Alt":  types.NewStringArray("en-US", "My vacation", "de", "Mein Urlaub", "", "default vacation"),
			"MH":   *mhbe,
			"BE":   *mhbe,
		},
	}

	mhbe = createMHBEDict()

	return &types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("Rendition"),
			"S":    types.PDFName("MR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"C":    mediaClipSectionDict,
		},
	}

}

func createSelectorRendition(mediaClipDataDict *types.PDFIndirectRef) *types.PDFDict {

	mhbe := createMHBEDict()

	r := createSectionMediaRendition(mediaClipDataDict)

	return &types.PDFDict{
		Dict: map[string]types.PDFObject{
			"Type": types.PDFName("Rendition"),
			"S":    types.PDFName("SR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"R":    types.PDFArray{*r},
		},
	}

}
