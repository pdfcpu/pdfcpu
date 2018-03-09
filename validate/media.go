package validate

import (
	"github.com/hhrutter/pdfcpu/types"
)

func validateMinimumBitDepthDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// see table 269

	dictName := "minBitDepthDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MinBitDepth" })
	if err != nil {
		return err
	}

	// V, required, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "V", REQUIRED, sinceVersion, func(i int) bool { return i >= 0 })
	if err != nil {
		return err
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "M", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMinimumScreenSizeDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// see table 269

	dictName := "minBitDepthDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MinScreenSize" })
	if err != nil {
		return err
	}

	// V, required, integer array, length 2
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "V", REQUIRED, sinceVersion, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "M", OPTIONAL, sinceVersion, nil)

	return err
}

func validateSoftwareIdentifierDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// see table 292

	dictName := "swIdDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "SoftwareIdentifier" })
	if err != nil {
		return err
	}

	// U, required, ASCII string
	_, err = validateStringEntry(xRefTable, dict, dictName, "U", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// L, optional, array
	_, err = validateArrayEntry(xRefTable, dict, dictName, "L", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// LI, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "LI", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// H, optional, array
	_, err = validateArrayEntry(xRefTable, dict, dictName, "H", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// HI, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "HI", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// OS, optional, array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "OS", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMediaCriteriaDictEntryD(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) error {

	d, err := validateDictEntry(xRefTable, dict, dictName, "D", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateMinimumBitDepthDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateMediaCriteriaDictEntryZ(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) error {

	d, err := validateDictEntry(xRefTable, dict, dictName, "Z", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateMinimumScreenSizeDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateMediaCriteriaDictEntryV(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) error {

	a, err := validateArrayEntry(xRefTable, dict, dictName, "V", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if a != nil {

		for _, v := range *a {

			if v == nil {
				continue
			}

			d, err := xRefTable.DereferenceDict(v)
			if err != nil {
				return err
			}

			if d != nil {
				err = validateSoftwareIdentifierDict(xRefTable, d, sinceVersion)
				if err != nil {
					return err
				}
			}

		}

	}

	return nil
}

func validateMediaCriteriaDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// see table 268

	dictName := "mediaCritDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaCriteria" })
	if err != nil {
		return err
	}

	// A, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "A", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// C, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "C", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// O, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "O", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// S, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "S", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// R, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "R", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// D, optional, dict
	err = validateMediaCriteriaDictEntryD(xRefTable, dict, dictName, OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// Z, optional, dict
	err = validateMediaCriteriaDictEntryZ(xRefTable, dict, dictName, OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// V, optional, array
	err = validateMediaCriteriaDictEntryV(xRefTable, dict, dictName, OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// P, optional, array
	_, err = validateNameArrayEntry(xRefTable, dict, dictName, "P", OPTIONAL, sinceVersion, func(a types.PDFArray) bool { return len(a) == 1 || len(a) == 2 })
	if err != nil {
		return err
	}

	// L, optional, array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "L", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMediaPermissionsDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see table 275
	d, err := validateDictEntry(xRefTable, dict, dictName, "P", OPTIONAL, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	dictName = "mediaPermissionDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaPermissions" })
	if err != nil {
		return err
	}

	// TF, optional, ASCII string
	validateTempFilePolicy := func(s string) bool {
		return memberOf(s, []string{"TEMPNEVER", "TEMPEXTRACT", "TEMPACCESS", "TEMPALWAYS"})
	}
	_, err = validateStringEntry(xRefTable, d, dictName, "TF", OPTIONAL, sinceVersion, validateTempFilePolicy)

	return err
}

func validateMediaPlayerInfoDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// see table 291

	dictName := "mediaPlayerInfoDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaPlayerInfo" })
	if err != nil {
		return err
	}

	// PID, required, software identifier dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "PID", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}
	err = validateSoftwareIdentifierDict(xRefTable, d, sinceVersion)
	if err != nil {
		return err
	}

	// MH, optional, dict
	_, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// BE, optional, dict
	_, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMediaPlayersDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// see 13.2.7.2

	dictName := "mediaPlayersDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaPlayers" })
	if err != nil {
		return err
	}

	// MU, optional, array of media player info dicts
	a, err := validateArrayEntry(xRefTable, dict, dictName, "MU", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if a != nil {

		for _, v := range *a {

			if v == nil {
				continue
			}

			d, err := xRefTable.DereferenceDict(v)
			if err != nil {
				return err
			}

			if d == nil {
				continue
			}

			err = validateMediaPlayerInfoDict(xRefTable, d, sinceVersion)
			if err != nil {
				return err
			}

		}

	}

	return nil

}

func validateFileSpecOrFormXObjectEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) error {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	return validateFileSpecificationOrFormObject(xRefTable, obj)
}

func validateMediaClipDataDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// see 13.2.4.2

	dictName := "mediaClipDataDict"

	// D, required, file specification or stream
	err := validateFileSpecOrFormXObjectEntry(xRefTable, dict, dictName, "D", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// CT, optional, ASCII string
	_, err = validateStringEntry(xRefTable, dict, dictName, "CT", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// P, optional, media permissions dict
	err = validateMediaPermissionsDict(xRefTable, dict, dictName, sinceVersion)
	if err != nil {
		return err
	}

	// Alt, optional, string array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "Alt", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// PL, optional, media players dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "PL", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaPlayersDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	// MH, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		// BU, optional, ASCII string
		_, err = validateStringEntry(xRefTable, d, "", "BU", OPTIONAL, sinceVersion, nil)
		if err != nil {
			return err
		}
	}

	// BE. optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		// BU, optional, ASCII string
		_, err = validateStringEntry(xRefTable, d, "", "BU", OPTIONAL, sinceVersion, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateTimespanDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	dictName := "timespanDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Timespan" })
	if err != nil {
		return err
	}

	// S, required, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return s == "S" })
	if err != nil {
		return err
	}

	// V, required, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "V", REQUIRED, sinceVersion, nil)

	return err
}

func validateMediaOffsetDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// see 13.2.6.2

	dictName := "mediaOffsetDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaOffset" })
	if err != nil {
		return err
	}

	// S, required, name
	subType, err := validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return memberOf(s, []string{"T", "F", "M"}) })
	if err != nil {
		return err
	}

	switch *subType {

	case "T":
		d, err := validateDictEntry(xRefTable, dict, dictName, "T", REQUIRED, sinceVersion, nil)
		if err != nil {
			return err
		}
		err = validateTimespanDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}

	case "F":
		_, err = validateIntegerEntry(xRefTable, dict, dictName, "F", REQUIRED, sinceVersion, func(i int) bool { return i >= 0 })
		if err != nil {
			return err
		}

	case "M":
		_, err = validateStringEntry(xRefTable, dict, dictName, "M", REQUIRED, sinceVersion, nil)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateMediaClipSectionDictMHBE(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	dictName := "mediaClipSectionMHBE"

	d, err := validateDictEntry(xRefTable, dict, dictName, "B", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaOffsetDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	d, err = validateDictEntry(xRefTable, dict, dictName, "E", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaOffsetDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateMediaClipSectionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// see 13.2.4.3

	dictName := "mediaClipSectionDict"

	// D, required, media clip dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "D", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}
	err = validateMediaClipDict(xRefTable, d, sinceVersion)
	if err != nil {
		return err
	}

	// Alt, optional, string array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "Alt", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// MH, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaClipSectionDictMHBE(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	// BE, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaClipSectionDictMHBE(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateMediaClipDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// see 13.2.4

	dictName := "mediaClipDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaClip" })
	if err != nil {
		return err
	}

	// S, required, name
	subType, err := validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return s == "MCD" || s == "MCS" })
	if err != nil {
		return err
	}

	// N, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "N", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if *subType == "MCD" {
		err = validateMediaClipDataDict(xRefTable, dict, sinceVersion)
		if err != nil {
			return err
		}
	}

	if *subType == "MCS" {
		err = validateMediaClipSectionDict(xRefTable, dict, sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateMediaDurationDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	dictName := "mediaDurationDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaDuration" })
	if err != nil {
		return err
	}

	// S, required, name
	validate := func(s string) bool { return memberOf(s, []string{"I", "F", "T"}) }
	s, err := validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, sinceVersion, validate)
	if err != nil {
		return err
	}

	// T, required if S == "T", timespann dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "T", *s == "T", sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateTimespanDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateMediaPlayParamsMHBEDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	dictName := "mediaPlayParamsMHBEDict"

	// V, optional, integer
	_, err := validateIntegerEntry(xRefTable, dict, dictName, "V", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// C, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "C", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// F, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "RT", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// D, optional, media duration dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "D", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaDurationDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	// A, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "A", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// RC, optional, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "RC", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateMediaPlayParamsDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// see 13.2.5

	dictName := "mediaPlayParamsDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaPlayParams" })
	if err != nil {
		return err
	}

	// PL, optional, media players dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "PL", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaPlayersDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	// MH, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaPlayParamsMHBEDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	// BE, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaPlayParamsMHBEDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateFloatingWindowsParameterDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// see table 284

	dictName := "floatWinParamsDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "FWParams" })
	if err != nil {
		return err
	}

	// D, required, array of integers
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "D", REQUIRED, sinceVersion, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// RT, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "RT", OPTIONAL, sinceVersion, func(i int) bool { return intMemberOf(i, []int{0, 1, 2, 3}) })
	if err != nil {
		return err
	}

	// P, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "P", OPTIONAL, sinceVersion, func(i int) bool { return intMemberOf(i, []int{0, 1, 2, 3, 4, 5, 6, 7, 8}) })
	if err != nil {
		return err
	}

	// O, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "O", OPTIONAL, sinceVersion, func(i int) bool { return intMemberOf(i, []int{0, 1, 2}) })
	if err != nil {
		return err
	}

	// T, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "T", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// UC, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "UC", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// R, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "R", OPTIONAL, sinceVersion, func(i int) bool { return intMemberOf(i, []int{0, 1, 2}) })
	if err != nil {
		return err
	}

	// TT, optional, string array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "TT", OPTIONAL, sinceVersion, nil)

	return err
}

func validateScreenParametersMHBEDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	dictName := "screenParmsMHBEDict"

	w := 3

	// W, optional, integer
	i, err := validateIntegerEntry(xRefTable, dict, dictName, "W", OPTIONAL, sinceVersion, func(i int) bool { return intMemberOf(i, []int{0, 1, 2, 3}) })
	if err != nil {
		return err
	}
	if i != nil {
		w = (*i).Value()
	}

	// B, optional, array of 3 numbers
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "B", OPTIONAL, sinceVersion, func(a types.PDFArray) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	// O, optional, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "O", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "M", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// F, required if W == 0, floating windows parameter dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "F", w == 0, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateFloatingWindowsParameterDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateScreenParametersDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// see 13.2.

	dictName := "screenParmsDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaScreenParams" })
	if err != nil {
		return err
	}

	// MH, optional, dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateScreenParametersMHBEDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	// BE. optional. dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateScreenParametersMHBEDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateMediaRenditionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// table 271

	dictName := "mediaRendDict"

	var d *types.PDFDict

	// C, optional, dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "C", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaClipDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	// P, required if C not present, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "P", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaPlayParamsDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	// SP, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "SP", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateScreenParametersDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateSelectorRenditionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	// table 272

	dictName := "selectorRendDict"

	a, err := validateArrayEntry(xRefTable, dict, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	for _, v := range *a {

		if v == nil {
			continue
		}

		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			return err
		}

		if d == nil {
			continue
		}

		err = validateRenditionDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateRenditionDictEntryMH(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	d, err := validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d != nil {

		d, err = validateDictEntry(xRefTable, d, "MHDict", "C", OPTIONAL, sinceVersion, nil)
		if err != nil {
			return err
		}

		if d != nil {
			err = validateMediaCriteriaDict(xRefTable, d, sinceVersion)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func validateRenditionDictEntryBE(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) (err error) {

	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	if d != nil {

		d, err = validateDictEntry(xRefTable, d, "BEDict", "C", OPTIONAL, sinceVersion, nil)
		if err != nil {
			return
		}

		err = validateMediaCriteriaDict(xRefTable, d, sinceVersion)
		if err != nil {
			return
		}

	}

	return
}

func validateRenditionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	dictName := "renditionDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Rendition" })
	if err != nil {
		return
	}

	// S, required, name
	renditionType, err := validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return s == "MR" || s == "SR" })
	if err != nil {
		return
	}

	// N, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "N", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	// MH, optional, dict
	err = validateRenditionDictEntryMH(xRefTable, dict, dictName, sinceVersion)
	if err != nil {
		return
	}

	// BE, optional, dict
	err = validateRenditionDictEntryBE(xRefTable, dict, dictName, sinceVersion)
	if err != nil {
		return
	}

	if *renditionType == "MR" {
		err = validateMediaRenditionDict(xRefTable, dict, sinceVersion)
		if err != nil {
			return
		}
	}

	if *renditionType == "SR" {
		err = validateSelectorRenditionDict(xRefTable, dict, sinceVersion)
		if err != nil {
			return
		}
	}

	return
}
