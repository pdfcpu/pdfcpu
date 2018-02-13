package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateMinimumBitDepthDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see table 269

	logInfoValidate.Println("*** validateMinimumBitDepthDict: begin ***")

	dictName := "minBitDepthDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MinBitDepth" })
	if err != nil {
		return err
	}

	// V, required, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "V", REQUIRED, types.V10, func(i int) bool { return i >= 0 })
	if err != nil {
		return err
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "M", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateMinimumBitDepthDict: end ***")

	return nil
}

func validateMinimumScreenSizeDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see table 269

	logInfoValidate.Println("*** validateMinimumScreenSizeDict: begin ***")

	dictName := "minBitDepthDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MinScreenSize" })
	if err != nil {
		return err
	}

	// V, required, integer array, length 2
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "V", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "M", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateMinimumScreenSizeDict: end ***")

	return nil
}

func validateSoftwareIdentifierDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see table 292

	logInfoValidate.Println("*** validateSoftwareIdentifierDict: begin ***")

	dictName := "swIdDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "SoftwareIdentifier" })
	if err != nil {
		return err
	}

	// U, required, ASCII string
	_, err = validateStringEntry(xRefTable, dict, dictName, "U", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// L, optional, array
	_, err = validateArrayEntry(xRefTable, dict, dictName, "L", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// LI, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "LI", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// H, optional, array
	_, err = validateArrayEntry(xRefTable, dict, dictName, "H", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// HI, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "HI", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// OS, optional, array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "OS", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateSoftwareIdentifierDict: end ***")

	return nil
}

func validateMediaCriteriaDictEntryD(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) error {

	var d *types.PDFDict

	d, err := validateDictEntry(xRefTable, dict, dictName, "D", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateMinimumBitDepthDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateMediaCriteriaDictEntryZ(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) error {

	var d *types.PDFDict

	d, err := validateDictEntry(xRefTable, dict, dictName, "Z", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateMinimumScreenSizeDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateMediaCriteriaDictEntryV(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) error {

	var a *types.PDFArray

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
				err = validateSoftwareIdentifierDict(xRefTable, d)
				if err != nil {
					return err
				}
			}

		}

	}

	return nil
}

func validateMediaCriteriaDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see table 268

	logInfoValidate.Println("*** validateMediaCriteriaDict: begin ***")

	dictName := "mediaCritDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaCriteria" })
	if err != nil {
		return err
	}

	// A, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "A", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// C, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// O, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "O", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// S, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "S", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// R, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "R", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// D, optional, dict
	err = validateMediaCriteriaDictEntryD(xRefTable, dict, dictName, OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// Z, optional, dict
	err = validateMediaCriteriaDictEntryZ(xRefTable, dict, dictName, OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// V, optional, array
	err = validateMediaCriteriaDictEntryV(xRefTable, dict, dictName, OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// P, optional, array
	_, err = validateNameArrayEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// L, optional, array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "L", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateMediaCriteriaDict: end ***")

	return nil
}

func validateMediaPermissionsDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string) error {

	// see table 275
	d, err := validateDictEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10, nil)
	if err != nil || d == nil {
		return err
	}

	logInfoValidate.Println("*** validateMediaPermissionsDict: begin ***")

	dictName = "mediaPermissionDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaPermissions" })
	if err != nil {
		return err
	}

	// TF, optional, ASCII string
	validateTempFilePolicy := func(s string) bool {
		return memberOf(s, []string{"TEMPNEVER", "TEMPEXTRACT", "TEMPACCESS", "TEMPALWAYS"})
	}
	_, err = validateStringEntry(xRefTable, d, dictName, "TF", OPTIONAL, types.V10, validateTempFilePolicy)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateMediaPermissionsDict: end ***")

	return nil
}

func validateMediaPlayerInfoDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see table 291

	logInfoValidate.Println("*** validateMediaPlayerInfoDict: begin ***")

	dictName := "mediaPlayerInfoDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaPlayerInfo" })
	if err != nil {
		return err
	}

	var d *types.PDFDict

	// PID, required, software identifier dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "PID", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}
	err = validateSoftwareIdentifierDict(xRefTable, d)
	if err != nil {
		return err
	}

	// MH, optional, dict
	_, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// BE, optional, dict
	_, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateMediaPlayerInfoDict: end ***")

	return nil
}

func validateMediaPlayersDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 13.2.7.2

	logInfoValidate.Println("*** validateMediaPlayersDict: begin ***")

	dictName := "mediaPlayersDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaPlayers" })
	if err != nil {
		return err
	}

	// MU, optional, array of media player info dicts
	a, err := validateArrayEntry(xRefTable, dict, dictName, "MU", OPTIONAL, types.V10, nil)
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

			err = validateMediaPlayerInfoDict(xRefTable, d)
			if err != nil {
				return err
			}

		}

	}

	logInfoValidate.Println("*** validateMediaPlayersDict: end ***")

	return nil

}

func validateFileSpecOrFormXObjectEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) error {

	logInfoValidate.Println("*** validateFileSpecOrFormXObjectEntry: begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateFileSpecOrFormXObjectEntry: missing entry \"%s\"", entryName)
		}
		return nil
	}

	_, err := validateFileSpecificationOrFormObject(xRefTable, obj)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateFileSpecOrFormXObjectEntry: begin ***")

	return nil
}

func validateMediaClipDataDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 13.2.4.2

	logInfoValidate.Println("*** validateMediaClipDataDict: begin ***")

	dictName := "mediaClipDataDict"

	// D, required, file specification or stream
	err := validateFileSpecOrFormXObjectEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10)
	if err != nil {
		return err
	}

	// CT, optional, ASCII string
	_, err = validateNameEntry(xRefTable, dict, dictName, "CT", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// P, optional, media permissions dict
	err = validateMediaPermissionsDict(xRefTable, dict, dictName)
	if err != nil {
		return err
	}

	// Alt, optional, string array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "Alt", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// PL, optional, media players dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "PL", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaPlayersDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	// MH, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		// BU, optional, ASCII string
		_, err = validateStringEntry(xRefTable, d, "", "BU", OPTIONAL, types.V10, nil)
		if err != nil {
			return err
		}
	}

	// BE. optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		// BU, optional, ASCII string
		_, err = validateStringEntry(xRefTable, d, "", "BU", OPTIONAL, types.V10, nil)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateMediaClipDataDict: end ***")

	return nil
}

func validateTimespanDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateTimespanDict: begin ***")

	dictName := "timespanDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Timespan" })
	if err != nil {
		return err
	}

	// S, required, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return s == "S" })
	if err != nil {
		return err
	}

	// V, required, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "V", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateTimespanDict: end ***")

	return nil
}

func validateMediaOffsetDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 13.2.6.2

	logInfoValidate.Println("*** validateMediaOffsetDict: begin ***")

	dictName := "mediaOffsetDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaOffset" })
	if err != nil {
		return err
	}

	// S, required, name
	subType, err := validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return memberOf(s, []string{"T", "F", "M"}) })
	if err != nil {
		return err
	}

	switch *subType {

	case "T":
		d, err := validateDictEntry(xRefTable, dict, dictName, "T", REQUIRED, types.V10, nil)
		if err != nil {
			return err
		}
		err = validateTimespanDict(xRefTable, d)
		if err != nil {
			return err
		}

	case "F":
		_, err = validateIntegerEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V10, func(i int) bool { return i >= 0 })
		if err != nil {
			return err
		}

	case "M":
		_, err = validateStringEntry(xRefTable, dict, dictName, "M", REQUIRED, types.V10, nil)
		if err != nil {
			return err
		}

	}

	logInfoValidate.Println("*** validateMediaOffsetDict: end ***")

	return nil
}

func validateMediaClipSectionDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 13.2.4.3

	logInfoValidate.Println("*** validateMediaClipSectionDict: begin ***")

	dictName := "mediaClipSectionDict"

	// D, required, media clip dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}
	err = validateMediaClipDict(xRefTable, d)
	if err != nil {
		return err
	}

	// Alt, optional, string array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "Alt", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// MH, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaOffsetDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	// BE, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaOffsetDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateMediaClipSectionDict: end ***")

	return nil
}

func validateMediaClipDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 13.2.4

	logInfoValidate.Println("*** validateMediaClipDict: begin ***")

	dictName := "mediaClipDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaClip" })
	if err != nil {
		return err
	}

	// S, required, name
	subType, err := validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return s == "MCD" || s == "MCS" })
	if err != nil {
		return err
	}

	// N, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "N", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	if *subType == "MCD" {
		err = validateMediaClipDataDict(xRefTable, dict)
		if err != nil {
			return err
		}
	}

	if *subType == "MCS" {
		err = validateMediaClipSectionDict(xRefTable, dict)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateMediaClipDict: end ***")

	return nil
}

func validateMediaPlayParametersDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 13.2.5

	logInfoValidate.Println("*** validateMediaPlayParametersDict: begin ***")

	dictName := "mediaPlayParmsDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaPlayParams" })
	if err != nil {
		return err
	}

	// PL, optional, media players dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "PL", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaPlayersDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	// MH, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaOffsetDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	// BE, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaOffsetDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateMediaPlayParametersDict: end ***")

	return nil
}

func validateFloatingWindowsParameterDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see table 284

	logInfoValidate.Println("*** validateFloatingWindowsParameterDict: begin ***")

	dictName := "floatWinParamsDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "FWParams" })
	if err != nil {
		return err
	}

	// D, required, array of integers
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// RT, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "RT", OPTIONAL, types.V10, func(i int) bool { return intMemberOf(i, []int{0, 1, 2, 3}) })
	if err != nil {
		return err
	}

	// P, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10, func(i int) bool { return intMemberOf(i, []int{0, 1, 2, 3, 4, 5, 6, 7, 8}) })
	if err != nil {
		return err
	}

	// O, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "O", OPTIONAL, types.V10, func(i int) bool { return intMemberOf(i, []int{0, 1, 2}) })
	if err != nil {
		return err
	}

	// T, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "T", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// UC, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "UC", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// R, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "R", OPTIONAL, types.V10, func(i int) bool { return intMemberOf(i, []int{0, 1, 2}) })
	if err != nil {
		return err
	}

	// TT, optional, string array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "TT", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateFloatingWindowsParameterDict: end ***")

	return nil
}

func validateScreenParametersMHBEDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateScreenParametersMHBEDict: begin ***")

	dictName := "screenParmsMHBEDict"

	w := 3

	// W, optional, integer
	i, err := validateIntegerEntry(xRefTable, dict, dictName, "W", OPTIONAL, types.V10, func(i int) bool { return intMemberOf(i, []int{0, 1, 2, 3}) })
	if err != nil {
		return err
	}
	if i != nil {
		w = (*i).Value()
	}

	// B, optional, array of 3 numbers
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "B", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	// O, optional, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "O", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "M", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// F, required if W == 0, floating windows parameter dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "F", w == 0, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateFloatingWindowsParameterDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateScreenParametersMHBEDict: end ***")

	return nil
}

func validateScreenParametersDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 13.2.

	logInfoValidate.Println("*** validateScreenParametersDict: begin ***")

	dictName := "screenParmsDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaScreenParams" })
	if err != nil {
		return err
	}

	// MH, optional, dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateScreenParametersMHBEDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	// BE. optional. dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateScreenParametersMHBEDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateScreenParametersDict: end ***")

	return nil
}

func validateMediaRenditionDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// table 271

	logInfoValidate.Println("*** validateMediaRenditionDict: begin ***")

	dictName := "mediaRendDict"

	var d *types.PDFDict

	// C, optional, dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaClipDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	// P, required if C not present, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateMediaPlayParametersDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	// SP, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "SP", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateScreenParametersDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateMediaRenditionDict: end ***")

	return nil
}

func validateSelectorRenditionDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// table 272

	logInfoValidate.Println("*** validateSelectorRenditionDict: begin ***")

	dictName := "selectorRendDict"

	a, err := validateArrayEntry(xRefTable, dict, dictName, "R", REQUIRED, types.V10, nil)
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

		err = validateRenditionDict(xRefTable, d)
		if err != nil {
			return err
		}

	}

	logInfoValidate.Println("*** validateSelectorRenditionDict: end ***")

	return nil
}

func validateRenditionDictEntryMH(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string) error {

	d, err := validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	if d != nil {

		d, err = validateDictEntry(xRefTable, d, "MHDict", "C", OPTIONAL, types.V10, nil)
		if err != nil {
			return err
		}

		if d != nil {
			err = validateMediaCriteriaDict(xRefTable, d)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func validateRenditionDictEntryBE(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string) (err error) {

	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	if d != nil {

		d, err = validateDictEntry(xRefTable, d, "BEDict", "C", OPTIONAL, types.V10, nil)
		if err != nil {
			return
		}

		err = validateMediaCriteriaDict(xRefTable, d)
		if err != nil {
			return
		}

	}

	return
}

func validateRenditionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Printf("*** validateRenditionDict: begin ***")

	dictName := "renditionDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Rendition" })
	if err != nil {
		return
	}

	// S, required, name
	renditionType, err := validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return s == "MR" || s == "SR" })
	if err != nil {
		return
	}

	// N, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "N", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// MH, optional, dict
	err = validateRenditionDictEntryMH(xRefTable, dict, dictName)
	if err != nil {
		return
	}

	// BE, optional, dict
	err = validateRenditionDictEntryBE(xRefTable, dict, dictName)
	if err != nil {
		return
	}

	if *renditionType == "MR" {
		err = validateMediaRenditionDict(xRefTable, dict)
		if err != nil {
			return
		}
	}

	if *renditionType == "SR" {
		err = validateSelectorRenditionDict(xRefTable, dict)
		if err != nil {
			return
		}
	}

	logInfoValidate.Printf("*** validateRenditionDict: end ***")

	return
}
