package pdfcpu

// Functions needed to create a test.pdf that gets used for validation testing (see process_test.go)

func createMHBEDict() *PDFDict {

	softwareIdentDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("SoftwareIdentifier"),
			"U":    PDFStringLiteral("vnd.adobe.swname:ADBE_Acrobat"),
			"L":    NewIntegerArray(0),
			"H":    NewIntegerArray(),
			"OS":   NewStringArray(),
		},
	}

	mediaCriteriaDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("MediaCriteria"),
			"A":    PDFBoolean(false),
			"C":    PDFBoolean(false),
			"O":    PDFBoolean(false),
			"S":    PDFBoolean(false),
			"R":    PDFInteger(0),
			"D": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("MinBitDepth"),
					"V":    PDFInteger(0),
					"M":    PDFInteger(0),
				},
			},
			"V": PDFArray{softwareIdentDict},
			"Z": PDFDict{
				Dict: map[string]PDFObject{
					"Type": PDFName("MinScreenSize"),
					"V":    NewIntegerArray(640, 480),
					"M":    PDFInteger(0),
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
		Dict: map[string]PDFObject{
			"Type": PDFName("SoftwareIdentifier"),
			"U":    PDFStringLiteral("vnd.adobe.swname:ADBE_Acrobat"),
			"L":    NewIntegerArray(0),
			"H":    NewIntegerArray(),
			"OS":   NewStringArray(),
		},
	}

	mediaPlayerInfoDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("MediaPlayerInfo"),
			"PID":  softwareIdentDict,
		},
	}

	return &PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("MediaPlayers"),
			"MU":   PDFArray{mediaPlayerInfoDict},
		},
	}

}

func createMediaOffsetDict() *PDFDict {

	timeSpanDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Timespan"),
			"S":    PDFName("S"),
			"V":    PDFInteger(1),
		},
	}

	return &PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("MediaOffset"),
			"S":    PDFName("T"),
			"T":    timeSpanDict,
		},
	}

}

func createSectionMHBEDict() *PDFDict {

	d := createMediaOffsetDict()

	return &PDFDict{
		Dict: map[string]PDFObject{
			"B": *d,
			"E": *d,
		},
	}
}

func createMediaClipDataDict(xRefTable *XRefTable) (*PDFIndirectRef, error) {

	// not supported: mp3,mp4,m4a

	fileSpecDict, err := createFileSpecDict(xRefTable, testAudioFileWAV)
	if err != nil {
		return nil, err
	}

	mediaPermissionsDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("MediaPermissions"),
			"TF":   PDFStringLiteral("TEMPNEVER"), //TEMPALWAYS
		},
	}

	mediaPlayersDict := createMediaPlayersDict()

	mhbe := PDFDict{Dict: map[string]PDFObject{"BU": nil}}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("MediaClip"),
			"S":    PDFName("MCD"), // media clip data
			"N":    PDFStringLiteral("Sample Audio"),
			"D":    *fileSpecDict,
			"CT":   PDFStringLiteral("audio/x-wav"),
			//"CT": PDFStringLiteral("audio/mp4"),
			//"CT":   PDFStringLiteral("video/mp4"),
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
		Dict: map[string]PDFObject{
			"Type": PDFName("Timespan"),
			"S":    PDFName("S"),
			"V":    PDFFloat(10.0),
		},
	}

	mediaDurationDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("MediaDuration"),
			"S":    PDFName("T"),
			"T":    timeSpanDict,
		},
	}

	return &PDFDict{
		Dict: map[string]PDFObject{
			"V":  PDFInteger(100),
			"C":  PDFBoolean(false),
			"F":  PDFInteger(5),
			"D":  mediaDurationDict,
			"A":  PDFBoolean(true),
			"RC": PDFFloat(1.0),
		},
	}

}

func createMediaPlayParamsDict() *PDFDict {

	d := createMediaPlayersDict()
	mhbe := createMediaPlayParamsMHBE()

	return &PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("MediaPlayParams"),
			"PL":   *d,
			"MH":   *mhbe,
			"BE":   *mhbe,
		},
	}

}

func createFloatingWindowsParamsDict() *PDFDict {

	return &PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("FWParams"),
			"D":    NewIntegerArray(200, 200),
			"RT":   PDFInteger(0),
			"P":    PDFInteger(4),
			"O":    PDFInteger(1),
			"T":    PDFBoolean(true),
			"UC":   PDFBoolean(true),
			"R":    PDFInteger(0),
			"TT":   NewStringArray("en-US", "Special title", "de", "Spezieller Titel", "default title"),
		},
	}
}

func createScreenParamsDict() *PDFDict {

	d := createFloatingWindowsParamsDict()

	mhbe := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("MediaScreenParams"),
			"W":    PDFInteger(0),
			"B":    NewNumberArray(1.0, 0.0, 0.0),
			"O":    PDFFloat(1.0),
			"M":    PDFInteger(0),
			"F":    *d,
		},
	}

	return &PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("MediaScreenParams"),
			"MH":   mhbe,
			"BE":   mhbe,
		},
	}
}

func createMediaRendition(xRefTable *XRefTable, mediaClipDataDict *PDFIndirectRef) *PDFDict {

	mhbe := createMHBEDict()

	d1 := createMediaPlayParamsDict()
	d2 := createScreenParamsDict()

	return &PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Rendition"),
			"S":    PDFName("MR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"C":    *mediaClipDataDict,
			"P":    *d1,
			"SP":   *d2,
		},
	}

}

func createSectionMediaRendition(mediaClipDataDict *PDFIndirectRef) *PDFDict {

	mhbe := createSectionMHBEDict()

	mediaClipSectionDict := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("MediaClip"),
			"S":    PDFName("MCS"), // media clip section
			"N":    PDFStringLiteral("Sample movie"),
			"D":    *mediaClipDataDict,
			"Alt":  NewStringArray("en-US", "My vacation", "de", "Mein Urlaub", "", "default vacation"),
			"MH":   *mhbe,
			"BE":   *mhbe,
		},
	}

	mhbe = createMHBEDict()

	return &PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Rendition"),
			"S":    PDFName("MR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"C":    mediaClipSectionDict,
		},
	}

}

func createSelectorRendition(mediaClipDataDict *PDFIndirectRef) *PDFDict {

	mhbe := createMHBEDict()

	r := createSectionMediaRendition(mediaClipDataDict)

	return &PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("Rendition"),
			"S":    PDFName("SR"),
			"MH":   *mhbe,
			"BE":   *mhbe,
			"R":    PDFArray{*r},
		},
	}

}
